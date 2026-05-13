package watcher

import (
	"errors"
	"strings"
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

func TestHandleEventUsesInjectedHandler(t *testing.T) {
	handler := &recordingHandler{}
	previous := defaultWatcher
	defaultWatcher = New(handler)
	defer func() { defaultWatcher = previous }()

	handleEvent(fsnotify.Event{Name: "report.hwpx", Op: fsnotify.Write})
	handleEvent(fsnotify.Event{Name: "report.pdf", Op: fsnotify.Remove})

	if len(handler.indexed) != 1 || handler.indexed[0] != "report.hwpx" {
		t.Fatalf("indexed = %v, want [report.hwpx]", handler.indexed)
	}
	if len(handler.removed) != 1 || handler.removed[0] != "report.pdf" {
		t.Fatalf("removed = %v, want [report.pdf]", handler.removed)
	}
}

func TestInstanceAddPathBeforeStartReturnsError(t *testing.T) {
	w := New(noopFileHandler{})

	err := w.AddPath(t.TempDir())

	if !errors.Is(err, ErrNotStarted) {
		t.Fatalf("AddPath before Start error = %v, want ErrNotStarted", err)
	}
}

func TestInstanceRemovePathBeforeStartReturnsError(t *testing.T) {
	w := New(noopFileHandler{})

	err := w.RemovePath(t.TempDir())

	if !errors.Is(err, ErrNotStarted) {
		t.Fatalf("RemovePath before Start error = %v, want ErrNotStarted", err)
	}
}

func TestInstanceCloseBeforeStartReturnsNil(t *testing.T) {
	w := New(noopFileHandler{})

	if err := w.Close(); err != nil {
		t.Fatalf("Close before Start error = %v, want nil", err)
	}
}

func TestInstanceCloseMakesAddPathErrorVisible(t *testing.T) {
	w := New(noopFileHandler{})
	if err := w.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	err := w.AddPath(t.TempDir())

	if err == nil {
		t.Fatal("AddPath after Close error = nil, want error")
	}
	if !strings.Contains(err.Error(), "closed") {
		t.Fatalf("AddPath after Close error = %v, want closed watcher error", err)
	}
}
