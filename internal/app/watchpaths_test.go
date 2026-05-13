package app

import (
	"hwp-searcher/internal/domain"
	"testing"
)

func TestWatchPathsAddsPathToStoreAndRegistry(t *testing.T) {
	store := &fakeConfigStore{}
	registry := &fakeWatchRegistry{}
	watchPaths := NewWatchPaths(store, registry)

	err := watchPaths.Add("/docs")

	if err != nil {
		t.Fatalf("Add returned error: %v", err)
	}
	if store.added != "/docs" {
		t.Fatalf("stored path = %q, want %q", store.added, "/docs")
	}
	if len(registry.added) != 1 || registry.added[0] != "/docs" {
		t.Fatalf("registered paths = %v, want [/docs]", registry.added)
	}
}

func TestWatchPathsResetReindexesWatchedPaths(t *testing.T) {
	index := &fakeIndex{}
	store := &fakeConfigStore{paths: []domain.WatchedPath{"/a", "/b"}}
	registry := &fakeWatchRegistry{}
	watchPaths := NewWatchPaths(store, registry)

	err := watchPaths.ResetIndex(index)

	if err != nil {
		t.Fatalf("ResetIndex returned error: %v", err)
	}
	if !index.reset {
		t.Fatal("Reset was not called")
	}
	if len(registry.added) != 2 || registry.added[0] != "/a" || registry.added[1] != "/b" {
		t.Fatalf("registered paths = %v, want [/a /b]", registry.added)
	}
}

func TestWatchPathsStatsUsesSmallPorts(t *testing.T) {
	index := &fakeIndex{count: 3}
	store := &fakeConfigStore{paths: []domain.WatchedPath{"/a", "/b"}}
	stats := NewStats(index, store, fakeIndexingStatus{indexing: true})

	result, err := stats.Current()

	if err != nil {
		t.Fatalf("Current returned error: %v", err)
	}
	if result.DocumentCount != 3 || result.WatchedPathCount != 2 || !result.Indexing {
		t.Fatalf("stats = %+v, want count=3 watched=2 indexing=true", result)
	}
}
