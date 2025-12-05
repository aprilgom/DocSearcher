package config

import (
	"encoding/json"
	"os"
	"sync"
)

const ConfigFile = "config.json"

type Config struct {
	WatchedPaths []string `json:"watched_paths"`
	mu           sync.Mutex
}

var Current *Config

func init() {
	Current = &Config{
		WatchedPaths: []string{},
	}
}

func Load() error {
	Current.mu.Lock()
	defer Current.mu.Unlock()

	data, err := os.ReadFile(ConfigFile)
	if os.IsNotExist(err) {
		return nil // No config file yet, use defaults
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, Current)
}

func Save() error {
	Current.mu.Lock()
	defer Current.mu.Unlock()

	data, err := json.MarshalIndent(Current, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigFile, data, 0644)
}

func AddPath(path string) error {
	Current.mu.Lock()
	// Check if already exists
	for _, p := range Current.WatchedPaths {
		if p == path {
			Current.mu.Unlock()
			return nil
		}
	}
	Current.WatchedPaths = append(Current.WatchedPaths, path)
	Current.mu.Unlock()
	return Save()
}

func RemovePath(path string) error {
	Current.mu.Lock()
	newPaths := []string{}
	for _, p := range Current.WatchedPaths {
		if p != path {
			newPaths = append(newPaths, p)
		}
	}
	Current.WatchedPaths = newPaths
	Current.mu.Unlock()
	return Save()
}
