package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func assertNoError(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func assertEqual(t *testing.T, expected, got interface{}, name string) {
	if expected != got {
		t.Errorf("%s: expected %v, got %v", name, expected, got)
	}
}

func TestDefaultConfig(t *testing.T) {
	expectedEditor := "nvim"
	expectedSearchPaths := 4
	expectedHomeDir, err := os.UserHomeDir()

	if err != nil {
		t.Fatalf("failed to get user home directory: %v", err)
	}

	cfg, err := DefaultConfig()
	if err != nil {
		t.Fatalf("failed to create default config: %v", err)
	}
	if cfg == nil {
		t.Fatal("default config is nil")
	}

	if cfg.Editor != expectedEditor {
		t.Errorf("expected editor %q, got %q", expectedEditor, cfg.Editor)
	}

	if len(cfg.SearchPaths) != expectedSearchPaths {
		t.Errorf("expected %d search paths, got %d", expectedSearchPaths, len(cfg.SearchPaths))
	}

	for _, path := range cfg.SearchPaths {
		if !strings.Contains(path, expectedHomeDir) {
			t.Errorf("expected search path %q to contain %q", path, expectedHomeDir)
		}
	}
}

func TestConfigPath(t *testing.T) {
	cfgPath, err := ConfigPath()
	expectedCfg := "gf/config.json"

	if err != nil {
		t.Fatalf("failed to get config path: %v", err)
	}

	if cfgPath == "" {
		t.Fatal("config path is empty")
	}

	if !strings.HasSuffix(cfgPath, expectedCfg) {
		t.Errorf("expected config path %q to end with %q", cfgPath, expectedCfg)
	}
}

func TestLoad_FileNotExists(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")

	cfg, err := load(configFile)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if cfg != nil {
		t.Errorf("expected nil config, got %v", cfg)
	}

	_, err = os.Stat(configFile)
	if err == nil {
		t.Fatal("expected file not to exist")
	}
}

func TestSave_CreatesDirectory(t *testing.T) {
	cfg := &Config{
		Editor: "vim",
		SearchPaths: []string{
			"/test-dir",
		},
	}

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, ".config", "gf", "config.json")

	err := save(configFile, cfg)

	assertNoError(t, err)

	if _, err := os.Stat(configFile); err != nil {
		t.Fatalf("failed to stat config file: %v", err)
	}

	_, err = os.Stat(filepath.Dir(configFile))
	if err != nil {
		t.Fatalf("failed to stat config directory: %v", err)
	}

	loadedCfg, err := load(configFile)
	assertNoError(t, err)

	if loadedCfg.Editor != cfg.Editor {
		t.Errorf("expected editor %q, got %q", cfg.Editor, loadedCfg.Editor)
	}

}

func TestLoadSave_RoundTrip(t *testing.T) {
	originalCfg := &Config{
		Editor: "code",
		SearchPaths: []string{
			"/home/user/dev",
			"/home/user/projects",
			"/home/user/workspace",
		},
	}
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")

	err := save(configFile, originalCfg)
	assertNoError(t, err)

	loadedCfg, err := load(configFile)
	assertNoError(t, err)

	if loadedCfg.Editor != originalCfg.Editor {
		t.Errorf("editor mismatch: expected %q, got %q", originalCfg.Editor, loadedCfg.Editor)
	}

	if len(loadedCfg.SearchPaths) != len(originalCfg.SearchPaths) {
		t.Fatalf("search paths mismatch: expected %d, got %d", len(originalCfg.SearchPaths), len(loadedCfg.SearchPaths))
	}

	for i, path := range originalCfg.SearchPaths {
		if loadedCfg.SearchPaths[i] != path {
			t.Errorf("search path mismatch at index %d: expected %q, got %q", i, path, loadedCfg.SearchPaths[i])
		}
	}

}

func TestLoad_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")

	invalidJSON := []byte(`{ "editor": "vim", invalid }`)
	err := os.WriteFile(configFile, invalidJSON, 0644)
	assertNoError(t, err)

	cfg, err := load(configFile)

	if err == nil {
		t.Fatal("expected error when loading invalid JSON, got nil")
	}

	if cfg != nil {
		t.Errorf("expected nil config, got %v", cfg)
	}

	if !strings.Contains(err.Error(), "unmarshal") {
		t.Errorf("expected error message to contain 'unmarshal', got: %v", err)
	}
}
