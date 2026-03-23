package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"
	"sync"
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

type Cache struct {
	m  map[string]AdData
	mu sync.RWMutex
}

func NewCache() *Cache {
	return &Cache{m: map[string]AdData{}, mu: sync.RWMutex{}}
}

func (c *Cache) Load(k string) (AdData, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	data, exists := c.m[k]
	return data, exists
}

func (c *Cache) Store(k string, v AdData) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[k] = v
}

func (c *Cache) LoadFromFile() {
	data, err := os.ReadFile(getDataCachePath())
	if err == nil {
		slog.Debug("cache found", "path", getDataCachePath())
	} else if !os.IsNotExist(err) {
		slog.Error("failed to open file", "path", getDataCachePath(), "error", err)
		os.Exit(1)
	} else {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if err := json.Unmarshal(data, &c.m); err != nil {
		slog.Error("failed to unmarshal cache into json", "error", err)
		os.Exit(1)
	}
}

func (c *Cache) SaveToFile() error {
	c.mu.RLock()
	data, err := json.Marshal(c.m)
	c.mu.RUnlock()
	if err != nil {
		return fmt.Errorf("failed to marshal to json: %w", err)
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
	return path.Join(cfg.CacheDir, getNormalizedCategory(), savePagesDir)
}

func getAdsDir() string {
	return path.Join(cfg.CacheDir, getNormalizedCategory(), saveAdsDir)
}

func getDataCachePath() string {
	return path.Join(cfg.CacheDir, getNormalizedCategory(), "cache.json")
}

func getNormalizedCategory() string {
	return strings.ReplaceAll(cfg.Category, "/", "-")
}
