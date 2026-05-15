package usecase

import "hwp-searcher/internal/domain"

type WatchPathReader interface {
	WatchedPaths() []domain.WatchedPath
}

type DocumentRootReader interface {
	DocumentRoots() []domain.DocumentRoot
}

type ConfigStore interface {
	WatchPathReader
	DocumentRootReader
	Load() error
	AddPath(path domain.WatchedPath) error
	RemovePath(path domain.WatchedPath) error
}

type WatchRegistry interface {
	AddPath(path domain.WatchedPath) error
	RemovePath(path domain.WatchedPath) error
}

type IndexResetter interface {
	Reset() error
}

type WatchPaths struct {
	configStore   ConfigStore
	watchRegistry WatchRegistry
}

func NewWatchPaths(configStore ConfigStore, watchRegistry WatchRegistry) WatchPaths {
	return WatchPaths{
		configStore:   configStore,
		watchRegistry: watchRegistry,
	}
}

func (w WatchPaths) List() []domain.WatchedPath {
	return w.configStore.WatchedPaths()
}

func (w WatchPaths) Start() error {
	if err := w.configStore.Load(); err != nil {
		return err
	}
	var firstErr error
	for _, path := range w.rootPaths() {
		if err := w.watchRegistry.AddPath(path); err != nil {
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

func (w WatchPaths) Add(path string) error {
	watchedPath := domain.WatchedPath(path)
	if err := w.watchRegistry.AddPath(watchedPath); err != nil {
		return err
	}
	return w.configStore.AddPath(watchedPath)
}

func (w WatchPaths) Remove(path string) error {
	watchedPath := domain.WatchedPath(path)
	if err := w.configStore.RemovePath(watchedPath); err != nil {
		return err
	}
	return w.watchRegistry.RemovePath(watchedPath)
}

func (w WatchPaths) ResetIndex(index IndexResetter) error {
	if err := index.Reset(); err != nil {
		return err
	}
	for _, path := range w.rootPaths() {
		if err := w.watchRegistry.AddPath(path); err != nil {
			return err
		}
	}
	return nil
}

func (w WatchPaths) rootPaths() []domain.WatchedPath {
	roots := w.configStore.DocumentRoots()
	if len(roots) == 0 {
		return w.configStore.WatchedPaths()
	}
	paths := make([]domain.WatchedPath, 0, len(roots))
	for _, root := range roots {
		paths = append(paths, domain.WatchedPath(root.ServerPath))
	}
	return paths
}
