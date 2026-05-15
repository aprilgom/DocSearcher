package config

import (
	"encoding/json"
	"fmt"
	"hwp-searcher/internal/domain"
	"os"
	"path/filepath"
	"sync"
)

const ConfigFile = "config.json"

type Config struct {
	WatchedPaths  []string             `json:"watched_paths"`
	DocumentRoots []documentRootConfig `json:"document_roots,omitempty"`
	mu            sync.Mutex
}

type documentRootConfig struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	ServerPath string           `json:"server_path"`
	SMBHost    string           `json:"smb_host"`
	SMBShare   string           `json:"smb_share"`
	SMBAliases []smbAliasConfig `json:"smb_aliases,omitempty"`
}

type smbAliasConfig struct {
	Host  string `json:"host"`
	Share string `json:"share"`
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
	loaded := Config{
		WatchedPaths: []string{},
	}
	if err := json.Unmarshal(data, &loaded); err != nil {
		return err
	}
	if err := loaded.validateDocumentRoots(); err != nil {
		return err
	}
	current.WatchedPaths = loaded.WatchedPaths
	current.DocumentRoots = loaded.DocumentRoots
	return nil
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

func (s Store) DocumentRoots() []domain.DocumentRoot {
	current := s.config()
	current.mu.Lock()
	defer current.mu.Unlock()

	roots := make([]domain.DocumentRoot, 0, len(current.DocumentRoots))
	for _, root := range current.DocumentRoots {
		roots = append(roots, root.toDomain())
	}
	return roots
}

func (s Store) AddPath(path domain.WatchedPath) error {
	return s.addPath(string(path))
}

func (s Store) RemovePath(path domain.WatchedPath) error {
	return s.removePath(string(path))
}

func (c *Config) validateDocumentRoots() error {
	seenIDs := map[domain.RootID]struct{}{}
	seenPaths := map[string]struct{}{}
	for i := range c.DocumentRoots {
		root := &c.DocumentRoots[i]
		if err := domain.ValidateRootID(domain.RootID(root.ID)); err != nil {
			return fmt.Errorf("document_roots[%d].id: %w", i, err)
		}
		rootID := domain.RootID(root.ID)
		if _, exists := seenIDs[rootID]; exists {
			return fmt.Errorf("document_roots[%d].id: duplicate root id %q", i, root.ID)
		}
		seenIDs[rootID] = struct{}{}

		cleanPath := filepath.Clean(root.ServerPath)
		if !filepath.IsAbs(cleanPath) {
			return fmt.Errorf("document_roots[%d].server_path must be absolute", i)
		}
		canonicalPath, err := filepath.EvalSymlinks(cleanPath)
		if err != nil {
			return fmt.Errorf("document_roots[%d].server_path: resolve symlinks: %w", i, err)
		}
		root.ServerPath = filepath.Clean(canonicalPath)
		if _, exists := seenPaths[root.ServerPath]; exists {
			return fmt.Errorf("document_roots[%d].server_path: duplicate server path %q", i, root.ServerPath)
		}
		seenPaths[root.ServerPath] = struct{}{}

		smb, err := domain.NewSMBShare(root.SMBHost, root.SMBShare)
		if err != nil {
			return fmt.Errorf("document_roots[%d].smb: %w", i, err)
		}
		root.SMBHost = smb.Host
		root.SMBShare = smb.Share
		for aliasIndex := range root.SMBAliases {
			alias, err := domain.NewSMBShare(root.SMBAliases[aliasIndex].Host, root.SMBAliases[aliasIndex].Share)
			if err != nil {
				return fmt.Errorf("document_roots[%d].smb_aliases[%d]: %w", i, aliasIndex, err)
			}
			root.SMBAliases[aliasIndex].Host = alias.Host
			root.SMBAliases[aliasIndex].Share = alias.Share
		}
	}
	return nil
}

func (root documentRootConfig) toDomain() domain.DocumentRoot {
	aliases := make([]domain.SMBAlias, 0, len(root.SMBAliases))
	for _, alias := range root.SMBAliases {
		aliases = append(aliases, domain.SMBAlias{Host: alias.Host, Share: alias.Share})
	}
	return domain.DocumentRoot{
		ID:         domain.RootID(root.ID),
		Name:       root.Name,
		ServerPath: root.ServerPath,
		SMB:        domain.SMBShare{Host: root.SMBHost, Share: root.SMBShare},
		SMBAliases: aliases,
	}
}
