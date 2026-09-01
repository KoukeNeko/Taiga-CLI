package releasepack

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestDeterministicArchives(t *testing.T) {
	stamp := time.Unix(1788250086, 0).UTC()
	entries := []archiveEntry{{Name: "taiga_0.1.0_linux_amd64/README.md", Mode: 0o644, Data: []byte("readme\n")}, {Name: "taiga_0.1.0_linux_amd64/taiga", Mode: 0o755, Data: []byte("binary")}}
	for _, test := range []struct {
		name  string
		write func(string, []archiveEntry, time.Time) error
	}{
		{name: "tar.gz", write: writeTarGzip},
		{name: "zip", write: writeZip},
	} {
		t.Run(test.name, func(t *testing.T) {
			first, second := filepath.Join(t.TempDir(), "first"), filepath.Join(t.TempDir(), "second")
			if err := test.write(first, entries, stamp); err != nil {
				t.Fatal(err)
			}
			if err := test.write(second, entries, stamp); err != nil {
				t.Fatal(err)
			}
			firstData, _ := os.ReadFile(first)
			secondData, _ := os.ReadFile(second)
			if !bytes.Equal(firstData, secondData) {
				t.Fatal("archives differ for identical inputs")
			}
		})
	}
}

func TestArchiveMetadata(t *testing.T) {
	stamp := time.Unix(1788250086, 0).UTC()
	entries := []archiveEntry{{Name: "root/taiga", Mode: 0o755, Data: []byte("binary")}}
	tarPath := filepath.Join(t.TempDir(), "release.tar.gz")
	if err := writeTarGzip(tarPath, entries, stamp); err != nil {
		t.Fatal(err)
	}
	file, _ := os.Open(tarPath)
	gzipReader, _ := gzip.NewReader(file)
	header, err := tar.NewReader(gzipReader).Next()
	if err != nil {
		t.Fatal(err)
	}
	if header.Name != "root/taiga" || header.FileInfo().Mode().Perm() != 0o755 || !header.ModTime.Equal(stamp) {
		t.Fatalf("tar header = %#v", header)
	}
	_ = gzipReader.Close()
	_ = file.Close()

	zipPath := filepath.Join(t.TempDir(), "release.zip")
	if err := writeZip(zipPath, entries, stamp); err != nil {
		t.Fatal(err)
	}
	zipReader, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = zipReader.Close() }()
	if len(zipReader.File) != 1 || zipReader.File[0].Name != "root/taiga" || zipReader.File[0].Mode().Perm() != 0o755 || !zipReader.File[0].Modified.Equal(stamp) {
		t.Fatalf("zip entries = %#v", zipReader.File)
	}
}

func TestGenerateSBOMIsStableSPDX(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	config := Config{RepoRoot: root, Version: "v0.1.0", Commit: "0123456789abcdef0123456789abcdef01234567", Epoch: 1788250086}
	stamp := time.Unix(config.Epoch, 0).UTC()
	first, err := generateSBOM(context.Background(), config, stamp)
	if err != nil {
		t.Fatal(err)
	}
	second, err := generateSBOM(context.Background(), config, stamp)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("SBOM is not deterministic")
	}
	var document spdxDocument
	if err := json.Unmarshal(first, &document); err != nil {
		t.Fatal(err)
	}
	if document.SPDXVersion != "SPDX-2.3" || document.SPDXID != "SPDXRef-DOCUMENT" || len(document.Packages) < 2 {
		t.Fatalf("document = %#v", document)
	}
	describes := false
	for _, relationship := range document.Relationships {
		if relationship.Element == "SPDXRef-DOCUMENT" && relationship.Type == "DESCRIBES" {
			describes = true
		}
	}
	if !describes {
		t.Fatal("SBOM does not describe the main package")
	}
}

func TestCollectEntriesSortsPaths(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(root, "z"), []byte("z"), 0o644)
	_ = os.WriteFile(filepath.Join(root, "nested", "a"), []byte("a"), 0o644)
	entries, err := collectEntries(root, "release")
	if err != nil {
		t.Fatal(err)
	}
	names := []string{entries[0].Name, entries[1].Name}
	if !reflect.DeepEqual(names, []string{"release/nested/a", "release/z"}) {
		t.Fatalf("names = %#v", names)
	}
}

func TestValidateConfig(t *testing.T) {
	config := Config{RepoRoot: t.TempDir(), Version: "dev", Commit: "bad", Epoch: 0}
	if err := validateConfig(&config); err == nil {
		t.Fatal("expected invalid release configuration")
	}
	config = Config{RepoRoot: t.TempDir(), Version: "v0.1.0", Commit: "0123456789abcdef0123456789abcdef01234567", Epoch: 1788250086, Targets: []Target{{OS: "../linux", Arch: "amd64"}}}
	if err := validateConfig(&config); err == nil {
		t.Fatal("expected invalid target error")
	}
}

func TestRunRefusesNonEmptyOutput(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	output := t.TempDir()
	if err := os.WriteFile(filepath.Join(output, "existing"), []byte("do not overwrite"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = Run(context.Background(), Config{RepoRoot: root, Output: output, Version: "v0.1.0", Commit: "0123456789abcdef0123456789abcdef01234567", Epoch: 1788250086, Targets: []Target{{OS: "linux", Arch: "amd64"}}})
	if err == nil {
		t.Fatal("expected non-empty output error")
	}
	data, readErr := os.ReadFile(filepath.Join(output, "existing"))
	if readErr != nil || string(data) != "do not overwrite" {
		t.Fatalf("existing output changed: data=%q err=%v", data, readErr)
	}
}
