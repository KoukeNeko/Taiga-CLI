// Package atomicfile replaces file contents so that a concurrent reader
// observes either the previous file or the complete new one, never a partial
// write, and never a file that is briefly world-readable.
package atomicfile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const (
	// DirectoryMode and FileMode keep CLI state private to the current user.
	DirectoryMode = 0o700
	FileMode      = 0o600
)

// Write replaces path with data, creating the parent directory when absent.
func Write(path string, data []byte) error {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, DirectoryMode); err != nil {
		return fmt.Errorf("create directory %q: %w", directory, err)
	}
	temporary, err := os.CreateTemp(directory, "."+filepath.Base(path)+"-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary file in %q: %w", directory, err)
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if err := writeAndClose(temporary, data); err != nil {
		return err
	}
	return replace(temporaryPath, path)
}

func writeAndClose(temporary *os.File, data []byte) error {
	if err := temporary.Chmod(FileMode); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("protect temporary file: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write temporary file: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("sync temporary file: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary file: %w", err)
	}
	return nil
}

// replace renames temporary over path. Windows refuses to rename onto an
// existing file, so the previous file is removed first on that platform only.
func replace(temporaryPath, path string) error {
	err := os.Rename(temporaryPath, path)
	if err == nil {
		return nil
	}
	if runtime.GOOS != "windows" {
		return fmt.Errorf("replace %q: %w", path, err)
	}
	if removeErr := os.Remove(path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
		return fmt.Errorf("remove previous %q: %w", path, removeErr)
	}
	if renameErr := os.Rename(temporaryPath, path); renameErr != nil {
		return fmt.Errorf("replace %q: %w", path, renameErr)
	}
	return nil
}
