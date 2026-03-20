package config

import (
	"os"
	"path/filepath"
	"testing"
)

// usesTempDir redirects the config directory to an isolated temp dir for the
// duration of a test and restores the original env/home afterwards.
func usesTempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	return filepath.Join(dir, configDirName, configFileName)
}

// TestEnsureCreatedMakesFile verifies that EnsureCreated writes the default
// config when the file does not exist yet.
func TestEnsureCreatedMakesFile(t *testing.T) {
	path := usesTempDir(t)

	created, err := EnsureCreated()
	if err != nil {
		t.Fatalf("EnsureCreated: %v", err)
	}
	if !created {
		t.Fatal("expected created=true on first call")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("config file not readable: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("config file is empty")
	}
}

// TestEnsureCreatedIdempotent verifies calling EnsureCreated twice does not
// overwrite an already existing file.
func TestEnsureCreatedIdempotent(t *testing.T) {
	path := usesTempDir(t)

	if _, err := EnsureCreated(); err != nil {
		t.Fatalf("first EnsureCreated: %v", err)
	}

	// Stamp the file with known content
	if err := os.WriteFile(path, []byte("# custom"), 0o600); err != nil {
		t.Fatalf("write custom file: %v", err)
	}

	created, err := EnsureCreated()
	if err != nil {
		t.Fatalf("second EnsureCreated: %v", err)
	}
	if created {
		t.Fatal("expected created=false when file already exists")
	}

	data, _ := os.ReadFile(path)
	if string(data) != "# custom" {
		t.Errorf("file was overwritten; got %q", string(data))
	}
}

// TestLoadEmpty verifies Load on a missing file returns a zero Config, not an error.
func TestLoadEmpty(t *testing.T) {
	usesTempDir(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load on missing file: %v", err)
	}
	if cfg.Server != "" {
		t.Errorf("expected empty server, got %q", cfg.Server)
	}
}

// TestSaveAndLoad round-trips a server value through Save/Load.
func TestSaveAndLoad(t *testing.T) {
	usesTempDir(t)

	if _, err := EnsureCreated(); err != nil {
		t.Fatalf("EnsureCreated: %v", err)
	}

	want := "https://paste.example.com"
	if err := Save(&Config{Server: want}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server != want {
		t.Errorf("server: got %q, want %q", cfg.Server, want)
	}
}

// TestSaveUncommentsExistingKey verifies that Save activates a commented-out
// line rather than appending a duplicate.
func TestSaveUncommentsExistingKey(t *testing.T) {
	path := usesTempDir(t)

	// Write a file where the key is commented out (the default template)
	initial := "# krypt config\n#server = \"https://old.example.com\"\n"
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := Save(&Config{Server: "https://new.example.com"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, _ := os.ReadFile(path)
	content := string(data)

	// Must contain the new active line
	if !containsLine(content, `server = "https://new.example.com"`) {
		t.Errorf("active server line not found in:\n%s", content)
	}
	// Must NOT have two server = lines
	count := countOccurrences(content, "server =")
	if count != 1 {
		t.Errorf("expected exactly 1 server= line, got %d:\n%s", count, content)
	}
}

// TestServerDefaultPriority verifies the priority chain.
func TestServerDefaultPriority(t *testing.T) {
	usesTempDir(t)

	// No env, no config → fallback
	t.Setenv("KRYPT_SERVER", "")
	got := ServerDefault()
	if got != FallbackServer {
		t.Errorf("fallback: got %q, want %q", got, FallbackServer)
	}

	// Config file wins over fallback
	if _, err := EnsureCreated(); err != nil {
		t.Fatal(err)
	}
	if err := Save(&Config{Server: "https://from-config.example.com"}); err != nil {
		t.Fatal(err)
	}
	got = ServerDefault()
	if got != "https://from-config.example.com" {
		t.Errorf("config: got %q", got)
	}

	// Env wins over config
	t.Setenv("KRYPT_SERVER", "https://from-env.example.com")
	got = ServerDefault()
	if got != "https://from-env.example.com" {
		t.Errorf("env: got %q", got)
	}
}

// TestFilePermissions verifies the config file is created with 0600.
func TestFilePermissions(t *testing.T) {
	path := usesTempDir(t)
	if _, err := EnsureCreated(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("expected 0600 permissions, got %04o", perm)
	}
}

// ---- helpers ----------------------------------------------------------------

func containsLine(content, substr string) bool {
	for _, line := range splitLines(content) {
		if line == substr {
			return true
		}
	}
	return false
}

func countOccurrences(s, substr string) int {
	n := 0
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			n++
		}
	}
	return n
}

func splitLines(s string) []string {
	var lines []string
	cur := ""
	for _, ch := range s {
		if ch == '\n' {
			lines = append(lines, cur)
			cur = ""
		} else {
			cur += string(ch)
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}
