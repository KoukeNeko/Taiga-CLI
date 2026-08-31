package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const defaultProfile = "default"

type Profile struct {
	APIURL  string `toml:"api_url" json:"api_url"`
	Project string `toml:"project,omitempty" json:"project,omitempty"`
}

type File struct {
	CurrentProfile string             `toml:"current_profile" json:"current_profile"`
	Profiles       map[string]Profile `toml:"profiles" json:"profiles"`
}

type Store struct {
	path string
}

func DefaultPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate user config directory: %w", err)
	}
	return filepath.Join(base, "taiga-cli", "config.toml"), nil
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func (s *Store) Path() string { return s.path }

func (s *Store) Load() (File, error) {
	cfg := File{CurrentProfile: defaultProfile, Profiles: map[string]Profile{}}
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return File{}, fmt.Errorf("read config %q: %w", s.path, err)
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return File{}, fmt.Errorf("parse config %q: %w", s.path, err)
	}
	if cfg.CurrentProfile == "" {
		cfg.CurrentProfile = defaultProfile
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	return cfg, nil
}

func (s *Store) Save(cfg File) error {
	if cfg.CurrentProfile == "" {
		cfg.CurrentProfile = defaultProfile
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	data, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary config: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("protect temporary config: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temporary config: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync temporary config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary config: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		if runtime.GOOS != "windows" {
			return fmt.Errorf("replace config: %w", err)
		}
		if removeErr := os.Remove(s.path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return fmt.Errorf("remove previous config: %w", removeErr)
		}
		if renameErr := os.Rename(tmpName, s.path); renameErr != nil {
			return fmt.Errorf("replace config: %w", renameErr)
		}
	}
	return nil
}

func NormalizeProfileName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("profile name cannot be empty")
	}
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			continue
		}
		return "", fmt.Errorf("profile name %q contains unsupported character %q", name, r)
	}
	return name, nil
}

func DefaultProfileName() string { return defaultProfile }
