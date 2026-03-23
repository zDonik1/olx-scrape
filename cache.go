package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path"
)

const (
	savePagesDir = "pages"
	saveAdsDir   = "ads"
)

func InitCache() {
	if cfg.RefreshCache {
		if err := os.RemoveAll(cfg.CacheDir); err != nil {
			slog.Error("could not remove cache", "path", cfg.CacheDir, "error", err)
			os.Exit(1)
		}
	}
	if cfg.RefreshPagesCache {
		if err := os.RemoveAll(getPagesDir()); err != nil {
			slog.Error("could not remove pages cache", "path", getPagesDir(), "error", err)
			os.Exit(1)
		}
	}
	if cfg.RefreshDataCache {
		if err := os.RemoveAll(getDataCachePath()); err != nil {
			slog.Error("could not remove pages cache", "path", getPagesDir(), "error", err)
			os.Exit(1)
		}
	}

	if err := os.MkdirAll(getPagesDir(), 0o755); err != nil {
		slog.Error("could not create pages cache dir", "path", getPagesDir(), "error", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(getAdsDir(), 0o755); err != nil {
		slog.Error("could not create ads cache dir", "path", getPagesDir(), "error", err)
		os.Exit(1)
	}
}

type Cache map[string]AdData

func NewCache() Cache {
	return map[string]AdData{}
}

func (c *Cache) Load() {
	data, err := os.ReadFile(getDataCachePath())
	if err == nil {
		slog.Debug("cache found", "path", getDataCachePath())
	} else if !os.IsNotExist(err) {
		slog.Error("failed to open file", "path", getDataCachePath(), "error", err)
		os.Exit(1)
	} else {
		return
	}

	if err := json.Unmarshal(data, c); err != nil {
		slog.Error("failed to unmarshal cache into json", "error", err)
		os.Exit(1)
	}
}

func (c Cache) Save() error {
	data, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal to json: %w\n%v", err, map[string]AdData(c))
	}

	tempFile, err := os.CreateTemp("", "cache_*.json")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	if _, err := tempFile.Write(data); err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to write to backup: %w", err)
	}
	tempFile.Close()

	return os.Rename(tempFile.Name(), getDataCachePath())
}

func getPagesDir() string {
	return path.Join(cfg.CacheDir, savePagesDir)
}

func getAdsDir() string {
	return path.Join(cfg.CacheDir, saveAdsDir)
}

func getDataCachePath() string {
	return path.Join(cfg.CacheDir, "cache.json")
}
