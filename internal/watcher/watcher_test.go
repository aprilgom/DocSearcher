package watcher

import (
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
	previous := fileHandler
	fileHandler = handler
	defer func() { fileHandler = previous }()

	handleEvent(fsnotify.Event{Name: "report.hwpx", Op: fsnotify.Write})
	handleEvent(fsnotify.Event{Name: "report.pdf", Op: fsnotify.Remove})

	if len(handler.indexed) != 1 || handler.indexed[0] != "report.hwpx" {
		t.Fatalf("indexed = %v, want [report.hwpx]", handler.indexed)
	}
	if len(handler.removed) != 1 || handler.removed[0] != "report.pdf" {
		t.Fatalf("removed = %v, want [report.pdf]", handler.removed)
	}
}
