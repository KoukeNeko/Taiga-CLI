package cli

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/KoukeNeko/taiga-cli/internal/config"
	"github.com/KoukeNeko/taiga-cli/internal/credential"
)

func TestDoctorBundleDoesNotLeakRuntimeIdentifiersOrSecrets(t *testing.T) {
	const profile = "customer-secret-profile"
	const project = "private-project-slug"
	const authToken = "auth-token-must-not-leak"
	const refreshToken = "refresh-token-must-not-leak"
	const username = "private-username"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+authToken {
			t.Fatalf("Authorization = %q", r.Header.Get("Authorization"))
		}
		switch r.URL.Path {
		case "/api/v1/locales":
			_, _ = io.WriteString(w, `[{"code":"en"}]`)
		case "/api/v1/users/me":
			_, _ = io.WriteString(w, `{"id":1,"username":"`+username+`"}`)
		case "/api/v1/projects/by_slug":
			_, _ = io.WriteString(w, `{"id":7,"name":"Private Project","slug":"`+project+`"}`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	app, out, stderr, credentials := testApp(t, server)
	apiURL := server.URL + "/api/v1/"
	if err := app.Config.Save(config.File{CurrentProfile: profile, Profiles: map[string]config.Profile{profile: {APIURL: apiURL, Project: project}}}); err != nil {
		t.Fatal(err)
	}
	credentials.values = map[string]credential.Tokens{credential.Account(profile, apiURL): {AuthToken: authToken, RefreshToken: refreshToken}}
	gitDir := t.TempDir()
	if output, err := runGit(gitDir, "init", "-q"); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	app.GitLocal = config.NewGitLocal(gitDir)
	if err := app.GitLocal.Set(context.Background(), "profile", profile); err != nil {
		t.Fatal(err)
	}
	if err := app.GitLocal.Set(context.Background(), "project", project); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "diagnostics.zip")
	if code := app.Execute(context.Background(), []string{"--json", "doctor", "bundle", path}); code != ExitSuccess {
		t.Fatalf("exit=%d stderr=%s", code, stderr.String())
	}
	contents, names := readDiagnosticArchive(t, path)
	if !sort.StringsAreSorted(names) || !bytes.Contains(contents, []byte(`"credential_present": true`)) || !bytes.Contains(contents, []byte(`"status": "ok"`)) {
		t.Fatalf("names=%#v contents=%s stdout=%s", names, contents, out.String())
	}
	for _, forbidden := range []string{profile, project, authToken, refreshToken, username, server.URL, gitDir} {
		if bytes.Contains(contents, []byte(forbidden)) {
			t.Fatalf("diagnostic archive leaked %q: %s", forbidden, contents)
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode=%v", info.Mode().Perm())
	}
}

func TestDoctorBundleRefusesOverwriteWithoutForce(t *testing.T) {
	app, out, stderr, credentials := testApp(t, nil)
	credentials.values = map[string]credential.Tokens{}
	if err := app.Config.Save(config.File{CurrentProfile: "default", Profiles: map[string]config.Profile{}}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "diagnostics.zip")
	if err := os.WriteFile(path, []byte("keep-me"), 0o600); err != nil {
		t.Fatal(err)
	}
	code := app.Execute(context.Background(), []string{"--json", "doctor", "bundle", path})
	if code != ExitValidation || out.Len() != 0 || !strings.Contains(stderr.String(), "output_exists") {
		t.Fatalf("exit=%d stdout=%s stderr=%s", code, out.String(), stderr.String())
	}
	data, _ := os.ReadFile(path)
	if string(data) != "keep-me" {
		t.Fatalf("existing output changed: %q", data)
	}
	app, _, stderr, credentials = testApp(t, nil)
	credentials.values = map[string]credential.Tokens{}
	if err := app.Config.Save(config.File{CurrentProfile: "default", Profiles: map[string]config.Profile{}}); err != nil {
		t.Fatal(err)
	}
	if code := app.Execute(context.Background(), []string{"--json", "doctor", "bundle", path, "--force"}); code != ExitSuccess {
		t.Fatalf("force exit=%d stderr=%s", code, stderr.String())
	}
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("replacement is not zip: %v", err)
	}
	_ = reader.Close()
}

func TestDiagnosticFailureOmitsMessageAndDetails(t *testing.T) {
	failure := diagnosticFailure("api", &contractError{Code: "secret-code", Message: "contains private URL", ExitCode: ExitTransport, Retryable: true, Details: map[string]any{"token": "secret"}})
	data, err := json.Marshal(failure)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte("private URL")) || bytes.Contains(data, []byte("token")) || !bytes.Contains(data, []byte("secret-code")) {
		t.Fatalf("failure = %s", data)
	}
}

func readDiagnosticArchive(t *testing.T, path string) ([]byte, []string) {
	t.Helper()
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()
	var contents bytes.Buffer
	names := make([]string, 0, len(reader.File))
	for _, entry := range reader.File {
		names = append(names, entry.Name)
		part, err := entry.Open()
		if err != nil {
			t.Fatal(err)
		}
		_, copyErr := io.Copy(&contents, part)
		closeErr := part.Close()
		if copyErr != nil || closeErr != nil {
			t.Fatalf("read %s: copy=%v close=%v", entry.Name, copyErr, closeErr)
		}
	}
	return contents.Bytes(), names
}

func runGit(directory string, args ...string) ([]byte, error) {
	command := exec.Command("git", args...)
	command.Dir = directory
	return command.CombinedOutput()
}
