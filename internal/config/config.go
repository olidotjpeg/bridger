package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type Config struct {
	ScanDirs   []string `json:"scan_dirs"`
	DBPath     string   `json:"db_path"`
	ThumbsPath string   `json:"thumbs_path"`
}

func defaultPaths() (dbPath, thumbsPath string, err error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return
	}
	base := filepath.Join(dir, "bridger")
	dbPath = filepath.Join(base, "bridger.db")
	thumbsPath = filepath.Join(base, "thumbs")
	return
}

func filePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "bridger", "config.json"), nil
}

func Load() (*Config, error) {
	path, err := filePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		dbPath, thumbsPath, err := defaultPaths()
		if err != nil {
			return nil, err
		}
		return &Config{DBPath: dbPath, ThumbsPath: thumbsPath}, nil
	}
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func Save(cfg *Config) error {
	path, err := filePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func NeedsSetup(cfg *Config) bool {
	return len(cfg.ScanDirs) == 0
}
