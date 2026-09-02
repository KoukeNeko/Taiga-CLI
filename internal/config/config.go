package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/KoukeNeko/aihki/internal/atomicfile"
	"github.com/pelletier/go-toml/v2"
)

const defaultProfile = "default"

const (
	configDirectory = "aihki"
	// legacyConfigDirectory is where this tool kept its config before it was
	// renamed. Falling back to it means an existing profile survives the
	// rename; the next save rewrites it at the current location.
	legacyConfigDirectory = "taiga-cli"
)

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
	return filepath.Join(base, configDirectory, "config.toml"), nil
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func (s *Store) Path() string { return s.path }

// legacyPath is the pre-rename sibling of path, used only when path is absent.
func legacyPath(path string) string {
	directory := filepath.Dir(path)
	if filepath.Base(directory) != configDirectory {
		return ""
	}
	return filepath.Join(filepath.Dir(directory), legacyConfigDirectory, filepath.Base(path))
}

func (s *Store) Load() (File, error) {
	cfg := File{CurrentProfile: defaultProfile, Profiles: map[string]Profile{}}
	// source is the file the bytes came from, so that a parse failure names
	// the file to fix rather than a current path that may not exist yet.
	source := s.path
	data, err := os.ReadFile(source)
	if errors.Is(err, os.ErrNotExist) {
		if legacy := legacyPath(s.path); legacy != "" {
			if inherited, legacyErr := os.ReadFile(legacy); legacyErr == nil { // nosemgrep -- the path is built from the config location, never from input
				data, err, source = inherited, nil, legacy
			}
		}
	}
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return File{}, fmt.Errorf("read config %q: %w", source, err)
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return File{}, fmt.Errorf("parse config %q: %w", source, err)
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
	if err := atomicfile.Write(s.path, data); err != nil {
		return fmt.Errorf("save config %q: %w", s.path, err)
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
