package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.ScanDirs) != 0 {
		t.Errorf("expected empty ScanDirs, got %v", cfg.ScanDirs)
	}
	if cfg.DBPath == "" {
		t.Error("expected non-empty DBPath")
	}
}

func TestNeedsSetup(t *testing.T) {
	if !NeedsSetup(&Config{}) {
		t.Error("expected NeedsSetup true for empty config")
	}
	if NeedsSetup(&Config{ScanDirs: []string{"/photos"}}) {
		t.Error("expected NeedsSetup false when dirs are set")
	}
}

func TestSaveRoundtrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	// Also set XDG_CONFIG_HOME so os.UserConfigDir() uses our temp dir on Linux
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, ".config"))

	original := &Config{
		ScanDirs:   []string{"/photos", "/more"},
		DBPath:     "/tmp/bridger.db",
		ThumbsPath: "/tmp/thumbs",
	}
	if err := Save(original); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(loaded.ScanDirs) != 2 {
		t.Errorf("expected 2 scan dirs, got %d", len(loaded.ScanDirs))
	}
	if loaded.DBPath != original.DBPath {
		t.Errorf("DBPath mismatch: got %s", loaded.DBPath)
	}
}

func TestSaveCreatesDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, ".config"))

	cfg := &Config{ScanDirs: []string{"/photos"}, DBPath: "/db", ThumbsPath: "/thumbs"}
	if err := Save(cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	path, _ := filePath()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("config file not found: %v", err)
	}
}
