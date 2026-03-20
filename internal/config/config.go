// Package config manages the krypt CLI configuration file.
//
// Config file location: $XDG_CONFIG_HOME/krypt/config.toml
// Falls back to:        ~/.config/krypt/config.toml
//
// On first run (file does not exist) the file is created automatically with
// commented-out defaults so the user can see what options are available.
//
// Priority order for the server URL (highest to lowest):
//  1. --server flag  (handled in cmd layer)
//  2. KRYPT_SERVER environment variable
//  3. server value in config.toml
//  4. built-in default (http://localhost:3000)
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	FallbackServer = "http://localhost:3000"
	configDirName  = "krypt"
	configFileName = "config.toml"
)

// Config holds all persisted settings.
type Config struct {
	Server string // Base URL of the krypt server
}

// defaultTOML is written on first run. Keys are commented out so the file
// is self-documenting but does not override the built-in default until edited.
const defaultTOML = `# krypt CLI configuration
#
# Base URL of the krypt server.
# Uncomment and edit to set your default server.
#server = "https://paste.example.com"
`

// configDir returns the platform-appropriate config directory.
// Respects XDG_CONFIG_HOME on Linux/macOS.
func configDir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, configDirName), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".config", configDirName), nil
}

// Path returns the full path to the config file.
func Path() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFileName), nil
}

// EnsureCreated creates the config file with defaults if it does not yet exist.
// It is called on every invocation so first-run creation is transparent.
// Returns (true, nil) when the file was newly created, (false, nil) otherwise.
func EnsureCreated() (created bool, err error) {
	path, err := Path()
	if err != nil {
		return false, err
	}

	if _, statErr := os.Stat(path); statErr == nil {
		return false, nil // already exists
	}

	// Create the parent directory if needed
	if mkErr := os.MkdirAll(filepath.Dir(path), 0o700); mkErr != nil {
		return false, fmt.Errorf("create config directory: %w", mkErr)
	}

	if writeErr := os.WriteFile(path, []byte(defaultTOML), 0o600); writeErr != nil {
		return false, fmt.Errorf("write default config: %w", writeErr)
	}
	return true, nil
}

// Load reads the config file and returns the parsed Config.
// Missing or unreadable files are silently ignored (returns empty Config).
func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		// If we can't find the path, return empty config — not fatal.
		return &Config{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	return parse(string(data)), nil
}

// Save writes the config to disk, preserving any comments in an existing file
// by rewriting only the keys that are present in cfg.
func Save(cfg *Config) error {
	path, err := Path()
	if err != nil {
		return err
	}

	if mkErr := os.MkdirAll(filepath.Dir(path), 0o700); mkErr != nil {
		return fmt.Errorf("create config directory: %w", mkErr)
	}

	// Read existing content so we can do a surgical in-place update.
	existing, readErr := os.ReadFile(path)
	var content string
	if readErr == nil {
		content = string(existing)
	} else {
		content = defaultTOML
	}

	content = setKey(content, "server", cfg.Server)

	if writeErr := os.WriteFile(path, []byte(content), 0o600); writeErr != nil {
		return fmt.Errorf("write config %s: %w", path, writeErr)
	}
	return nil
}

// ServerDefault returns the effective default server URL by consulting, in order:
//  1. KRYPT_SERVER environment variable
//  2. server key in config file
//  3. built-in fallback
//
// The --server flag is applied on top of this in the cmd layer.
func ServerDefault() string {
	if s := os.Getenv("KRYPT_SERVER"); s != "" {
		return s
	}
	cfg, err := Load()
	if err == nil && cfg.Server != "" {
		return cfg.Server
	}
	return FallbackServer
}

// ---- minimal hand-rolled TOML parser (subset: key = "value") ---------------
// We deliberately avoid external dependencies for a single-file config.

// parse reads key = "value" pairs, ignoring comment lines and blank lines.
func parse(content string) *Config {
	cfg := &Config{}
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := parseLine(line)
		if !ok {
			continue
		}
		switch key {
		case "server":
			cfg.Server = val
		}
	}
	return cfg
}

// parseLine splits `key = "value"` into (key, value, true).
func parseLine(line string) (key, value string, ok bool) {
	idx := strings.IndexByte(line, '=')
	if idx < 0 {
		return "", "", false
	}
	key = strings.TrimSpace(line[:idx])
	raw := strings.TrimSpace(line[idx+1:])
	// Strip surrounding quotes
	if len(raw) >= 2 && raw[0] == '"' && raw[len(raw)-1] == '"' {
		value = raw[1 : len(raw)-1]
	} else {
		value = raw
	}
	return key, value, key != ""
}

// setKey replaces or appends a key = "value" line in a TOML-ish content string.
// It un-comments an existing commented-out line for the key if present.
func setKey(content, key, value string) string {
	target := fmt.Sprintf("%s = %q", key, value)
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		stripped := strings.TrimSpace(strings.TrimLeft(line, "#"))
		k, _, ok := parseLine(stripped)
		if ok && k == key {
			lines[i] = target
			return strings.Join(lines, "\n")
		}
	}

	// Key not found — append it
	return strings.TrimRight(content, "\n") + "\n" + target + "\n"
}
