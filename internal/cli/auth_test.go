package cli

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KoukeNeko/aihki/internal/config"
	"github.com/KoukeNeko/aihki/internal/credential"
)

// A script that forgot the URL is told what to pass, not asked a question it
// cannot answer.
func TestLoginRequiresAURLOffATerminal(t *testing.T) {
	app, _, stderr, _ := testApp(t, nil)
	if err := app.Config.Save(config.File{CurrentProfile: "test", Profiles: map[string]config.Profile{"test": {}}}); err != nil {
		t.Fatal(err)
	}
	app.In = strings.NewReader("token\n")
	if code := app.Execute(context.Background(), []string{"--json", "auth", "login", "--with-token"}); code != ExitValidation {
		t.Fatalf("exit=%d stderr=%s", code, stderr.String())
	}
	for _, expected := range []string{`"code":"missing_api_url"`, "--url"} {
		if !strings.Contains(stderr.String(), expected) {
			t.Errorf("stderr = %s, want it to contain %q", stderr.String(), expected)
		}
	}
}

// The URL people have is the page in front of them, and the flag that takes
// it is --url; --host keeps working for the scripts that learned it.
func TestLoginDiscoversTheAPIFromAnyPageUnderEitherFlagName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/conf.json":
			_, _ = io.WriteString(w, `{"api":"http://`+r.Host+`/api/v1/","baseHref":"/"}`)
		case "/api/v1/locales":
			_, _ = io.WriteString(w, `[{"code":"en","name":"English"}]`)
		case "/api/v1/users/me":
			if r.Header.Get("Authorization") != "Bearer pasted-token" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_, _ = io.WriteString(w, `{"id":1,"username":"demo"}`)
		default:
			_, _ = io.WriteString(w, `<!doctype html><html><body>taiga</body></html>`)
		}
	}))
	defer server.Close()
	for _, flag := range []string{"--url", "--host"} {
		t.Run(flag, func(t *testing.T) {
			app, out, stderr, credentials := testApp(t, server)
			app.In = strings.NewReader("pasted-token\n")
			code := app.Execute(context.Background(), []string{"--profile", "site", "--json", "auth", "login", flag, server.URL + "/project/demo/backlog", "--with-token"})
			if code != ExitSuccess {
				t.Fatalf("exit=%d stderr=%s", code, stderr.String())
			}
			if !strings.Contains(out.String(), `"api_url":"`+server.URL+`/api/v1/"`) {
				t.Errorf("stdout = %s, want the discovered API", out.String())
			}
			if saved := credentials.values[credential.Account("site", server.URL+"/api/v1/")]; saved.AuthToken != "pasted-token" {
				t.Errorf("saved = %#v, want the pasted token under the discovered API", saved)
			}
		})
	}
}

func TestReadChoiceTakesANumberOrTheFirstOnEnter(t *testing.T) {
	for input, want := range map[string]int{"\n": 0, "2\n": 1, "1": 0} {
		app := &App{In: strings.NewReader(input), Err: &bytes.Buffer{}}
		got, err := app.readChoice("Pick", []string{"first", "second"})
		if err != nil || got != want {
			t.Errorf("input %q: got %d, %v; want %d", input, got, err, want)
		}
	}
	for _, input := range []string{"3\n", "0\n", "x\n"} {
		app := &App{In: strings.NewReader(input), Err: &bytes.Buffer{}}
		if _, err := app.readChoice("Pick", []string{"first", "second"}); err == nil {
			t.Errorf("input %q was accepted", input)
		}
	}
}
