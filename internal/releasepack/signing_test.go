package releasepack

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSigningDisabledByDefault(t *testing.T) {
	var signing Signing
	if signing.Enabled() || signing.NotarizationEnabled() {
		t.Fatalf("zero value = %#v, want signing and notarization disabled", signing)
	}
	if err := signing.validate(); err != nil {
		t.Fatalf("validate() = %v, want an unsigned build to be valid", err)
	}
}

func TestSigningRejectsNotarizationWithoutIdentity(t *testing.T) {
	signing := Signing{NotaryKey: "key.p8", NotaryKeyID: "ABC123", NotaryIssuer: "issuer"}
	err := signing.validate()
	if err == nil || !strings.Contains(err.Error(), "signing identity") {
		t.Fatalf("validate() = %v, want a missing-identity error", err)
	}
}

func TestSigningRejectsPartialNotarizationCredentials(t *testing.T) {
	for _, incomplete := range []Signing{
		{Identity: "Developer ID Application: Example (TEAM)", NotaryKey: "key.p8"},
		{Identity: "Developer ID Application: Example (TEAM)", NotaryKeyID: "ABC123"},
		{Identity: "Developer ID Application: Example (TEAM)", NotaryKey: "key.p8", NotaryIssuer: "issuer"},
	} {
		if incomplete.NotarizationEnabled() {
			t.Fatalf("NotarizationEnabled() = true for %#v", incomplete)
		}
		err := incomplete.validate()
		if err == nil {
			t.Fatalf("validate() = nil for %#v, want an incomplete-credentials error", incomplete)
		}
		if runtime.GOOS == "darwin" && !strings.Contains(err.Error(), "notarization needs all of") {
			t.Fatalf("validate() = %v, want an incomplete-credentials error", err)
		}
	}
}

func TestSigningRequiresMacOSHost(t *testing.T) {
	signing := Signing{Identity: "Developer ID Application: Example (TEAM)"}
	err := signing.validate()
	if runtime.GOOS == "darwin" {
		if err != nil {
			t.Fatalf("validate() = %v, want signing to be allowed on macOS", err)
		}
		return
	}
	if err == nil || !strings.Contains(err.Error(), "macOS host") {
		t.Fatalf("validate() = %v, want a host error away from macOS", err)
	}
}

func TestSigningRequiresAReadableNotaryKey(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("signing validation reaches the key check only on macOS")
	}
	signing := Signing{
		Identity:     "Developer ID Application: Example (TEAM)",
		NotaryKey:    filepath.Join(t.TempDir(), "missing.p8"),
		NotaryKeyID:  "ABC123",
		NotaryIssuer: "issuer",
	}
	if err := signing.validate(); err == nil || !strings.Contains(err.Error(), "App Store Connect key") {
		t.Fatalf("validate() = %v, want a missing-key error", err)
	}
	present := filepath.Join(t.TempDir(), "key.p8")
	if err := os.WriteFile(present, []byte("key"), 0o600); err != nil {
		t.Fatal(err)
	}
	signing.NotaryKey = present
	if err := signing.validate(); err != nil {
		t.Fatalf("validate() = %v, want a complete configuration to be valid", err)
	}
}

func TestApplySigningSkipsNonDarwinTargets(t *testing.T) {
	// A bogus identity would make codesign fail, so reaching nil here proves
	// that non-darwin targets never invoke it and stay reproducible.
	config := Config{Signing: Signing{Identity: "Developer ID Application: Nonexistent (TEAM)"}}
	for _, target := range []Target{{OS: "linux", Arch: "amd64"}, {OS: "windows", Arch: "arm64"}} {
		if err := applySigning(t.Context(), config, target, "/nonexistent/taiga", t.TempDir()); err != nil {
			t.Fatalf("applySigning(%s/%s) = %v, want no signing", target.OS, target.Arch, err)
		}
	}
}
