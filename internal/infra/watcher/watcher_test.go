package watcher

import (
	"errors"
	"slices"
	"testing"

	"github.com/fsnotify/fsnotify"
)

type recordingHandler struct {
	indexed []string
	removed []string
}

func (h *recordingHandler) IndexFile(path string) {
	h.indexed = append(h.indexed, path)
}

func (h *recordingHandler) RemoveFile(path string) {
	h.removed = append(h.removed, path)
}

func TestHandleEvent(t *testing.T) {
	tests := []struct {
		name        string
		events      []fsnotify.Event
		wantIndexed []string
		wantRemoved []string
	}{
		{
			name: "index_and_remove_supported_documents",
			events: []fsnotify.Event{
				{Name: "report.hwpx", Op: fsnotify.Write},
				{Name: "report.pdf", Op: fsnotify.Remove},
			},
			wantIndexed: []string{"report.hwpx"},
			wantRemoved: []string{"report.pdf"},
		},
		{
			name: "ignore_unsupported_document_events",
			events: []fsnotify.Event{
				{Name: "notes.txt", Op: fsnotify.Create},
				{Name: "~$lock.hwpx", Op: fsnotify.Write},
				{Name: "draft.hwp.tmp", Op: fsnotify.Remove},
			},
		},
		{
			name: "remove_for_rename_and_remove",
			events: []fsnotify.Event{
				{Name: "renamed.pdf", Op: fsnotify.Rename},
				{Name: "deleted.hwp", Op: fsnotify.Remove},
			},
			wantRemoved: []string{"renamed.pdf", "deleted.hwp"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &recordingHandler{}
			w := New(handler)

			for _, event := range tt.events {
				w.handleEvent(event)
			}

			if !slices.Equal(handler.indexed, tt.wantIndexed) {
				t.Errorf("indexed = %v, want %v", handler.indexed, tt.wantIndexed)
			}
			if !slices.Equal(handler.removed, tt.wantRemoved) {
				t.Errorf("removed = %v, want %v", handler.removed, tt.wantRemoved)
			}
		})
	}
}

func TestInstanceStateErrors(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, w *Watcher)
		run     func(t *testing.T, w *Watcher) error
		wantErr error
	}{
		{
			name: "add_before_start",
			run: func(t *testing.T, w *Watcher) error {
				return w.AddPath(t.TempDir())
			},
			wantErr: ErrNotStarted,
		},
		{
			name: "remove_before_start",
			run: func(t *testing.T, w *Watcher) error {
				return w.RemovePath(t.TempDir())
			},
			wantErr: ErrNotStarted,
		},
		{
			name: "close_before_start",
			run: func(t *testing.T, w *Watcher) error {
				return w.Close()
			},
		},
		{
			name: "add_after_close",
			setup: func(t *testing.T, w *Watcher) {
				t.Helper()
				if err := w.Start(); err != nil {
					t.Fatalf("Start: %v", err)
				}
				if err := w.Close(); err != nil {
					t.Fatalf("Close: %v", err)
				}
			},
			run: func(t *testing.T, w *Watcher) error {
				return w.AddPath(t.TempDir())
			},
			wantErr: ErrClosed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := New(noopFileHandler{})
			if tt.setup != nil {
				tt.setup(t, w)
			}

			err := tt.run(t, w)

			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("%s error = %v, want nil", tt.name, err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("%s error = %v, want %v", tt.name, err, tt.wantErr)
			}
		})
	}
}

func TestInstanceAddPathAfterStartAddsExistingDirectory(t *testing.T) {
	w := New(noopFileHandler{})
	if err := w.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		if err := w.Close(); err != nil && !errors.Is(err, ErrClosed) {
			t.Fatalf("Close: %v", err)
		}
	})

	if err := w.AddPath(t.TempDir()); err != nil {
		t.Fatalf("AddPath after Start returned error: %v", err)
	}
}

func TestInstanceRemovePathAfterStartRemovesWatchedDirectory(t *testing.T) {
	w := New(noopFileHandler{})
	if err := w.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		if err := w.Close(); err != nil && !errors.Is(err, ErrClosed) {
			t.Fatalf("Close: %v", err)
		}
	})
	root := t.TempDir()
	if err := w.AddPath(root); err != nil {
		t.Fatalf("AddPath: %v", err)
	}

	if err := w.RemovePath(root); err != nil {
		t.Fatalf("RemovePath after AddPath returned error: %v", err)
	}
}

func TestInstanceRemovePathAfterStartReturnsErrorForUnwatchedDirectory(t *testing.T) {
	w := New(noopFileHandler{})
	if err := w.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		if err := w.Close(); err != nil && !errors.Is(err, ErrClosed) {
			t.Fatalf("Close: %v", err)
		}
	})

	if err := w.RemovePath(t.TempDir()); err == nil {
		t.Fatal("RemovePath for unwatched directory returned nil, want error")
	}
}
