package config

import (
	"encoding/json"
	"hwp-searcher/internal/domain"
	"os"
	"sync"
)

const ConfigFile = "config.json"

type Config struct {
	WatchedPaths []string `json:"watched_paths"`
	mu           sync.Mutex
}

var Current *Config

type Store struct {
	path    string
	current *Config
}

func NewStore(path string) Store {
	return Store{
		path: path,
		current: &Config{
			WatchedPaths: []string{},
		},
	}
}

func init() {
	Current = &Config{
		WatchedPaths: []string{},
	}
}

func Load() error {
	return Store{}.Load()
}

func Save() error {
	return Store{}.save()
}

func AddPath(path string) error {
	return Store{}.addPath(path)
}

func RemovePath(path string) error {
	return Store{}.removePath(path)
}

func (s Store) configPath() string {
	if s.path != "" {
		return s.path
	}
	return ConfigFile
}

func (s Store) config() *Config {
	if s.current != nil {
		return s.current
	}
	return Current
}

func (s Store) Load() error {
	current := s.config()
	current.mu.Lock()
	defer current.mu.Unlock()

	data, err := os.ReadFile(s.configPath())
	if os.IsNotExist(err) {
		return nil // No config file yet, use defaults
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, current)
}

func (s Store) save() error {
	current := s.config()
	current.mu.Lock()
	defer current.mu.Unlock()

	data, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.configPath(), data, 0644)
}

func (s Store) addPath(path string) error {
	current := s.config()
	current.mu.Lock()
	for _, p := range current.WatchedPaths {
		if p == path {
			current.mu.Unlock()
			return nil
		}
	}
	current.WatchedPaths = append(current.WatchedPaths, path)
	current.mu.Unlock()
	return s.save()
}

func (s Store) removePath(path string) error {
	current := s.config()
	current.mu.Lock()
	newPaths := []string{}
	for _, p := range current.WatchedPaths {
		if p != path {
			newPaths = append(newPaths, p)
		}
	}
	current.WatchedPaths = newPaths
	current.mu.Unlock()
	return s.save()
}

func (s Store) WatchedPaths() []domain.WatchedPath {
	current := s.config()
	current.mu.Lock()
	defer current.mu.Unlock()

	paths := make([]domain.WatchedPath, 0, len(current.WatchedPaths))
	for _, path := range current.WatchedPaths {
		paths = append(paths, domain.WatchedPath(path))
	}
	return paths
}

func (s Store) AddPath(path domain.WatchedPath) error {
	return s.addPath(string(path))
}

func (s Store) RemovePath(path domain.WatchedPath) error {
	return s.removePath(string(path))
}
