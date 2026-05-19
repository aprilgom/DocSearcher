package usecase

import (
	"errors"
	"hwp-searcher/internal/domain"
	"testing"
)

func TestWatchPathsAddsPathToStoreAndRegistry(t *testing.T) {
	// Given
	store := &fakeConfigStore{}
	registry := &fakeWatchRegistry{}
	watchPaths := NewWatchPaths(store, registry)

	// When
	err := watchPaths.Add("/docs")

	// Then
	if err != nil {
		t.Fatalf("Add returned error: %v", err)
	}
	if !store.HasStoredPath("/docs") {
		t.Fatalf("Add should persist the watched path after registry accepts it: stored paths = %v, want /docs", store.StoredPaths())
	}
	assertWatchedPaths(t, registry.RegisteredPaths(), []domain.WatchedPath{"/docs"}, "Add should register the watched path")
}

func TestWatchPathsAddDoesNotStorePathWhenRegistryFails(t *testing.T) {
	// Given
	store := &fakeConfigStore{}
	registry := &fakeWatchRegistry{addErr: errors.New("watch failed")}
	watchPaths := NewWatchPaths(store, registry)

	// When
	err := watchPaths.Add("/docs")

	// Then
	if err == nil {
		t.Fatal("Add returned nil, want registry error")
	}
	if len(store.StoredPaths()) != 0 {
		t.Fatalf("Add should not persist path when registry rejects it: stored paths = %v, want empty", store.StoredPaths())
	}
	assertWatchedPaths(t, registry.RegisteredPaths(), []domain.WatchedPath{"/docs"}, "Add should attempt registry before persistence")
}

func TestWatchPathsResetReindexesWatchedPaths(t *testing.T) {
	// Given
	index := &fakeIndex{}
	store := &fakeConfigStore{paths: []domain.WatchedPath{"/a", "/b"}}
	registry := &fakeWatchRegistry{}
	watchPaths := NewWatchPaths(store, registry)

	// When
	err := watchPaths.ResetIndex(index)

	// Then
	if err != nil {
		t.Fatalf("ResetIndex returned error: %v", err)
	}
	if !index.WasReset() {
		t.Fatal("ResetIndex should clear the existing index before re-registering watched paths")
	}
	assertWatchedPaths(t, registry.RegisteredPaths(), []domain.WatchedPath{"/a", "/b"}, "ResetIndex should re-register configured paths")
}

func TestWatchPathsStartLoadsStoreAndRegistersWatchedPaths(t *testing.T) {
	// Given
	store := &fakeConfigStore{paths: []domain.WatchedPath{"/a", "/b"}}
	registry := &fakeWatchRegistry{}
	watchPaths := NewWatchPaths(store, registry)

	// When
	err := watchPaths.Start()

	// Then
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if !store.WasLoaded() {
		t.Fatal("Start should load persisted watched paths before registering them")
	}
	assertWatchedPaths(t, registry.RegisteredPaths(), []domain.WatchedPath{"/a", "/b"}, "Start should register persisted paths")
}

func TestWatchPathsStartKeepsRegisteringAfterRegistryError(t *testing.T) {
	// Given
	store := &fakeConfigStore{paths: []domain.WatchedPath{"/a", "/b"}}
	registry := &fakeWatchRegistry{addErr: errors.New("watch failed")}
	watchPaths := NewWatchPaths(store, registry)

	// When
	err := watchPaths.Start()

	// Then
	if err == nil {
		t.Fatal("Start returned nil, want error")
	}
	assertWatchedPaths(t, registry.RegisteredPaths(), []domain.WatchedPath{"/a", "/b"}, "Start should attempt every persisted path even after a registry error")
}

func TestWatchPathsRemoveRemovesPathFromStoreThenRegistry(t *testing.T) {
	// Given
	store := &fakeConfigStore{paths: []domain.WatchedPath{"/docs"}}
	registry := &fakeWatchRegistry{}
	watchPaths := NewWatchPaths(store, registry)

	// When
	err := watchPaths.Remove("/docs")

	// Then
	if err != nil {
		t.Fatalf("Remove returned error: %v", err)
	}
	if store.removed != "/docs" {
		t.Fatalf("Remove should remove the watched path from store: got %q, want %q", store.removed, "/docs")
	}
	if registry.removed != "/docs" {
		t.Fatalf("Remove should unregister the watched path after store removal: got %q, want %q", registry.removed, "/docs")
	}
}

func TestWatchPathsRemoveReturnsStoreErrorWithoutTouchingRegistry(t *testing.T) {
	// Given
	wantErr := errors.New("store remove failed")
	store := &fakeConfigStore{removeErr: wantErr}
	registry := &fakeWatchRegistry{}
	watchPaths := NewWatchPaths(store, registry)

	// When
	err := watchPaths.Remove("/docs")

	// Then
	if !errors.Is(err, wantErr) {
		t.Fatalf("Remove error = %v, want store error %v", err, wantErr)
	}
	if store.removed != "/docs" {
		t.Fatalf("Remove should attempt store removal for requested path: got %q, want %q", store.removed, "/docs")
	}
	if registry.removed != "" {
		t.Fatalf("Remove should not unregister path when store removal fails: got registry removal %q, want empty", registry.removed)
	}
}

func TestWatchPathsRemovePropagatesRegistryError(t *testing.T) {
	// Given
	wantErr := errors.New("watch remove failed")
	store := &fakeConfigStore{}
	registry := &fakeWatchRegistry{removeErr: wantErr}
	watchPaths := NewWatchPaths(store, registry)

	// When
	err := watchPaths.Remove("/docs")

	// Then
	if !errors.Is(err, wantErr) {
		t.Fatalf("Remove error = %v, want registry error %v", err, wantErr)
	}
	if store.removed != "/docs" {
		t.Fatalf("Remove should remove from store before unregistering: got %q, want %q", store.removed, "/docs")
	}
	if registry.removed != "/docs" {
		t.Fatalf("Remove should attempt registry removal before returning error: got %q, want %q", registry.removed, "/docs")
	}
}

func TestStatsReportsDocumentWatchPathAndIndexingCounts(t *testing.T) {
	// Given
	index := &fakeIndex{count: 3}
	store := &fakeConfigStore{paths: []domain.WatchedPath{"/a", "/b"}}
	stats := NewStats(index, store, fakeIndexingStatus{indexing: true})

	// When
	result, err := stats.Current()

	// Then
	if err != nil {
		t.Fatalf("Current returned error: %v", err)
	}
	if result.DocumentCount != 3 || result.WatchedPathCount != 2 || !result.Indexing {
		t.Fatalf("stats = %+v, want count=3 watched=2 indexing=true", result)
	}
}

func TestStatsCurrentPropagatesCountError(t *testing.T) {
	// Given
	wantErr := errors.New("count failed")
	index := &fakeIndex{countErr: wantErr}
	store := &fakeConfigStore{paths: []domain.WatchedPath{"/a"}}
	stats := NewStats(index, store, fakeIndexingStatus{indexing: true})

	// When
	result, err := stats.Current()

	// Then
	if !errors.Is(err, wantErr) {
		t.Fatalf("Current error = %v, want count error %v", err, wantErr)
	}
	if result != (domain.Stats{}) {
		t.Fatalf("Current result = %+v, want zero stats when count fails", result)
	}
}
