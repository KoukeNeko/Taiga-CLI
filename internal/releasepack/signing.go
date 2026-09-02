package releasepack

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// Signing describes how macOS binaries are signed and notarized. A zero value
// disables both, which is what keeps unsigned archives byte-for-byte
// reproducible: a Developer ID signature embeds a secure timestamp from Apple's
// authority and can never reproduce.
type Signing struct {
	Identity     string
	Keychain     string
	NotaryKey    string
	NotaryKeyID  string
	NotaryIssuer string
}

func (s Signing) Enabled() bool { return strings.TrimSpace(s.Identity) != "" }

// NotarizationEnabled reports whether the App Store Connect credentials are
// present. The issuer is deliberately not part of this: notarytool requires it
// for a Team API key and rejects it for an Individual API key, so it is
// optional and validated only for shape.
func (s Signing) NotarizationEnabled() bool {
	return strings.TrimSpace(s.NotaryKey) != "" && strings.TrimSpace(s.NotaryKeyID) != ""
}

func (s Signing) partiallyConfigured() bool {
	filled := 0
	for _, value := range []string{s.NotaryKey, s.NotaryKeyID} {
		if strings.TrimSpace(value) != "" {
			filled++
		}
	}
	return filled == 1
}

func (s Signing) validate() error {
	if !s.Enabled() {
		if s.partiallyConfigured() || s.NotarizationEnabled() {
			return errors.New("notarization credentials require a signing identity")
		}
		return nil
	}
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("signing identity %q requires a macOS host; codesign and notarytool do not exist elsewhere", s.Identity)
	}
	if s.partiallyConfigured() {
		return errors.New("notarization needs both --notary-key and --notary-key-id")
	}
	if issuer := strings.TrimSpace(s.NotaryIssuer); issuer != "" && !issuerPattern.MatchString(issuer) {
		return fmt.Errorf("notary issuer %q must be a UUID; leave it empty for an Individual API key", issuer)
	}
	if s.NotarizationEnabled() {
		if _, err := os.Stat(s.NotaryKey); err != nil {
			return fmt.Errorf("read App Store Connect key: %w", err)
		}
	}
	return nil
}

// issuerPattern matches the App Store Connect issuer UUID. notarytool rejects
// anything else outright, so catching the shape here keeps the failure next to
// the configuration rather than deep inside a release build.
var issuerPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// signBinary applies a Developer ID signature with the hardened runtime and a
// secure timestamp. Notarization rejects binaries missing either.
func signBinary(ctx context.Context, signing Signing, binaryPath string) error {
	arguments := []string{"--force", "--timestamp", "--options", "runtime", "--sign", signing.Identity}
	if strings.TrimSpace(signing.Keychain) != "" {
		arguments = append(arguments, "--keychain", signing.Keychain)
	}
	arguments = append(arguments, binaryPath)
	if output, err := exec.CommandContext(ctx, "codesign", arguments...).CombinedOutput(); err != nil {
		return fmt.Errorf("codesign %s: %w: %s", filepath.Base(binaryPath), err, strings.TrimSpace(string(output)))
	}
	verify := exec.CommandContext(ctx, "codesign", "--verify", "--strict", "--verbose=2", binaryPath)
	if output, err := verify.CombinedOutput(); err != nil {
		return fmt.Errorf("verify signature of %s: %w: %s", filepath.Base(binaryPath), err, strings.TrimSpace(string(output)))
	}
	return nil
}

// notarizeBinary submits a signed binary to Apple and waits for the verdict.
//
// notarytool accepts only zip, pkg and dmg, so the binary is zipped purely for
// submission. The ticket cannot be stapled afterwards because stapler does not
// support a bare Mach-O executable; Gatekeeper resolves it online instead.
func notarizeBinary(ctx context.Context, signing Signing, binaryPath, workDir string) error {
	submission := filepath.Join(workDir, filepath.Base(binaryPath)+"-notarize.zip")
	defer func() { _ = os.Remove(submission) }()
	ditto := exec.CommandContext(ctx, "ditto", "-c", "-k", "--keepParent", binaryPath, submission)
	if output, err := ditto.CombinedOutput(); err != nil {
		return fmt.Errorf("package %s for notarization: %w: %s", filepath.Base(binaryPath), err, strings.TrimSpace(string(output)))
	}
	arguments := []string{"notarytool", "submit", submission, "--key", signing.NotaryKey, "--key-id", signing.NotaryKeyID, "--wait"}
	// notarytool requires the issuer for a Team API key and refuses it for an
	// Individual API key, so it is passed only when configured.
	if issuer := strings.TrimSpace(signing.NotaryIssuer); issuer != "" {
		arguments = append(arguments, "--issuer", issuer)
	}
	notarize := exec.CommandContext(ctx, "xcrun", arguments...)
	if output, err := notarize.CombinedOutput(); err != nil {
		return fmt.Errorf("notarize %s: %w: %s", filepath.Base(binaryPath), err, strings.TrimSpace(string(output)))
	}
	return nil
}

// applySigning signs and notarizes a macOS binary in place. Other targets are
// left untouched so their archives stay reproducible.
func applySigning(ctx context.Context, config Config, target Target, binaryPath, workDir string) error {
	if target.OS != "darwin" || !config.Signing.Enabled() {
		return nil
	}
	if err := signBinary(ctx, config.Signing, binaryPath); err != nil {
		return err
	}
	if !config.Signing.NotarizationEnabled() {
		return nil
	}
	return notarizeBinary(ctx, config.Signing, binaryPath, workDir)
}
