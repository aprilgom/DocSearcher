package app

import "hwp-searcher/internal/domain"

type ConfigStore interface {
	WatchedPaths() []domain.WatchedPath
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

func (w WatchPaths) Add(path string) error {
	watchedPath := domain.WatchedPath(path)
	if err := w.configStore.AddPath(watchedPath); err != nil {
		return err
	}
	return w.watchRegistry.AddPath(watchedPath)
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
	for _, path := range w.configStore.WatchedPaths() {
		if err := w.watchRegistry.AddPath(path); err != nil {
			return err
		}
	}
	return nil
}
