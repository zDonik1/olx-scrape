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

func initCache() {
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

	if err := os.MkdirAll(getPagesDir(), 0o755); err != nil {
		slog.Error("could not create pages cache dir", "path", getPagesDir(), "error", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(getAdsDir(), 0o755); err != nil {
		slog.Error("could not create ads cache dir", "path", getPagesDir(), "error", err)
		os.Exit(1)
	}
}

func loadAiCache() map[uint]AdData {
	aiCache := map[uint]AdData{}
	data, err := os.ReadFile(getAiCachePath())
	if err == nil {
		slog.Info("using AI cache")
	} else if !os.IsNotExist(err) {
		slog.Error("failed to open file", "path", getAiCachePath(), "error", err)
		os.Exit(1)
	} else {
		return aiCache
	}

	if err := json.Unmarshal(data, &aiCache); err != nil {
		slog.Error("failed to unmarshal AI cache into json", "error", err)
		os.Exit(1)
	}
	return aiCache
}

func saveAiCache(cache map[uint]AdData) error {
	data, err := json.Marshal(cache)
	if err != nil {
		return fmt.Errorf("failed to marshal to json: %w\n%v", err, cache)
	}

	tempFile, err := os.CreateTemp("", "ai_cache_*.json")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	if _, err := tempFile.Write(data); err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to write to backup: %w", err)
	}
	tempFile.Close()

	return os.Rename(tempFile.Name(), getAiCachePath())
}

func getPagesDir() string {
	return path.Join(cfg.CacheDir, savePagesDir)
}

func getAdsDir() string {
	return path.Join(cfg.CacheDir, saveAdsDir)
}

func getAiCachePath() string {
	return path.Join(cfg.CacheDir, "ai_cache.json")
}
