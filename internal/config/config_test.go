package config

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestStoreRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.toml")
	store := NewStore(path)
	want := File{CurrentProfile: "local", Profiles: map[string]Profile{"local": {APIURL: "http://localhost/api/v1/", Project: "demo"}}}
	if err := store.Save(want); err != nil {
		t.Fatal(err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.CurrentProfile != "local" || got.Profiles["local"].Project != "demo" {
		t.Fatalf("config = %#v", got)
	}
	want.Profiles["local"] = Profile{APIURL: "http://localhost/api/v1/", Project: "changed"}
	if err := store.Save(want); err != nil {
		t.Fatalf("replace config: %v", err)
	}
	got, err = store.Load()
	if err != nil || got.Profiles["local"].Project != "changed" {
		t.Fatalf("reloaded config = %#v, error = %v", got, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}
}

func TestGitLocal(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command("git", "init", "-q", dir)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	local := NewGitLocal(dir)
	ctx := context.Background()
	if err := local.Set(ctx, "profile", "local"); err != nil {
		t.Fatal(err)
	}
	if err := local.Set(ctx, "project", "demo"); err != nil {
		t.Fatal(err)
	}
	values, err := local.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if values.Profile != "local" || values.Project != "demo" {
		t.Fatalf("values = %#v", values)
	}
}

// A profile written before the rename must survive it, or every user silently
// loses their API URL and default project.
func TestLoadFallsBackToTheLegacyDirectory(t *testing.T) {
	base := t.TempDir()
	legacy := filepath.Join(base, legacyConfigDirectory)
	if err := os.MkdirAll(legacy, 0o700); err != nil {
		t.Fatal(err)
	}
	contents := "current_profile = \"company\"\n\n[profiles.company]\napi_url = \"https://example.test/api/v1/\"\nproject = \"demo\"\n"
	if err := os.WriteFile(filepath.Join(legacy, "config.toml"), []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewStore(filepath.Join(base, configDirectory, "config.toml"))
	cfg, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CurrentProfile != "company" || cfg.Profiles["company"].Project != "demo" {
		t.Fatalf("config = %#v, want the legacy profile", cfg)
	}
	// Saving moves it to the current location without touching the old file.
	if err := store.Save(cfg); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(base, configDirectory, "config.toml")); err != nil {
		t.Fatalf("save did not write the current location: %v", err)
	}
}

func TestLoadPrefersTheCurrentDirectory(t *testing.T) {
	base := t.TempDir()
	for directory, profile := range map[string]string{configDirectory: "current", legacyConfigDirectory: "stale"} {
		if err := os.MkdirAll(filepath.Join(base, directory), 0o700); err != nil {
			t.Fatal(err)
		}
		body := "current_profile = \"" + profile + "\"\n"
		if err := os.WriteFile(filepath.Join(base, directory, "config.toml"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	cfg, err := NewStore(filepath.Join(base, configDirectory, "config.toml")).Load()
	if err != nil || cfg.CurrentProfile != "current" {
		t.Fatalf("config = %#v, error = %v", cfg, err)
	}
}

func TestGitLocalReadsTheLegacySection(t *testing.T) {
	dir := t.TempDir()
	if output, err := exec.Command("git", "init", "-q", dir).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	set := func(key, value string) {
		command := exec.Command("git", "config", "--local", key, value)
		command.Dir = dir
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git config %s: %v: %s", key, err, output)
		}
	}
	set(legacyConfigSection+".project", "pinned-before-the-rename")
	values, err := NewGitLocal(dir).Load(context.Background())
	if err != nil || values.Project != "pinned-before-the-rename" {
		t.Fatalf("values = %#v, error = %v", values, err)
	}
	set(configSection+".project", "pinned-after")
	values, err = NewGitLocal(dir).Load(context.Background())
	if err != nil || values.Project != "pinned-after" {
		t.Fatalf("values = %#v, error = %v", values, err)
	}
}
