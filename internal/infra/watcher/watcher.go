package watcher

import (
	"errors"
	"hwp-searcher/internal/domain"
	"log"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

type FileHandler interface {
	IndexFile(path string)
	RemoveFile(path string)
}

type Registry struct {
	Watcher       *Watcher
	StartIndexing func(path string)
}

var (
	ErrNotStarted = errors.New("watcher not started")
	ErrClosed     = errors.New("watcher closed")
)

var defaultWatcher = New(noopFileHandler{})

type Watcher struct {
	watcher     *fsnotify.Watcher
	fileHandler FileHandler
	closed      bool
}

type noopFileHandler struct{}

func (noopFileHandler) IndexFile(path string) {}

func (noopFileHandler) RemoveFile(path string) {}

func New(handler FileHandler) *Watcher {
	if handler == nil {
		handler = noopFileHandler{}
	}
	return &Watcher{fileHandler: handler}
}

func SetFileHandler(handler FileHandler) {
	defaultWatcher.fileHandler = handler
	if defaultWatcher.fileHandler == nil {
		defaultWatcher.fileHandler = noopFileHandler{}
	}
}

func Start() error {
	return defaultWatcher.Start()
}

func (w *Watcher) Start() error {
	if w.closed {
		return ErrClosed
	}
	if w.watcher != nil {
		return nil
	}
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	w.watcher = fsWatcher

	go func() {
		for {
			select {
			case event, ok := <-fsWatcher.Events:
				if !ok {
					return
				}
				w.handleEvent(event)
			case err, ok := <-fsWatcher.Errors:
				if !ok {
					return
				}
				log.Println("Watcher error:", err)
			}
		}
	}()

	return nil
}

func (w *Watcher) Close() error {
	if w.watcher == nil {
		return nil
	}
	err := w.watcher.Close()
	w.watcher = nil
	w.closed = true
	return err
}

func (w *Watcher) addPath(root string) error {
	if w.closed {
		return ErrClosed
	}
	if w.watcher == nil {
		return ErrNotStarted
	}
	// Recursive add
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			err = w.watcher.Add(path)
			if err != nil {
				log.Println("Failed to watch:", path, err)
			} else {
				// log.Println("Watching:", path)
			}
		}
		return nil
	})
	if err != nil {
		log.Println("Walk error for watcher:", err)
		return err
	}
	return nil
}

func AddPath(root string) {
	_ = (Registry{}).AddPath(domain.WatchedPath(root))
}

func RemovePath(root string) {
	_ = defaultWatcher.RemovePath(root)
}

func (w *Watcher) AddPath(root string) error {
	return w.addPath(root)
}

func (w *Watcher) RemovePath(root string) error {
	if w.closed {
		return ErrClosed
	}
	if w.watcher == nil {
		return ErrNotStarted
	}
	return w.watcher.Remove(root)
}

func (r Registry) AddPath(path domain.WatchedPath) error {
	w := r.Watcher
	if w == nil {
		w = defaultWatcher
	}
	if err := w.AddPath(string(path)); err != nil {
		return err
	}
	if r.StartIndexing != nil {
		r.StartIndexing(string(path))
	}
	return nil
}

func (r Registry) RemovePath(path domain.WatchedPath) error {
	w := r.Watcher
	if w == nil {
		w = defaultWatcher
	}
	return w.RemovePath(string(path))
}

func handleEvent(event fsnotify.Event) {
	defaultWatcher.handleEvent(event)
}

func (w *Watcher) handleEvent(event fsnotify.Event) {
	if event.Op&fsnotify.Create == fsnotify.Create {
		info, err := os.Stat(event.Name)
		if err == nil && info.IsDir() {
			// If directory created, watch it
			_ = w.addPath(event.Name)
			return
		}
	}

	// Ignore temporary files
	if !isSupportedDocumentEvent(event.Name) {
		return
	}

	if event.Op&fsnotify.Create == fsnotify.Create {
		log.Println("File created:", event.Name)
		w.fileHandler.IndexFile(event.Name)
	}

	if event.Op&fsnotify.Write == fsnotify.Write {
		log.Println("File modified:", event.Name)
		w.fileHandler.IndexFile(event.Name)
	}

	if event.Op&fsnotify.Remove == fsnotify.Remove || event.Op&fsnotify.Rename == fsnotify.Rename {
		// If it was a file, remove from index
		// We can't easily check if it was a dir or file since it's gone
		// But we can try to remove from index anyway
		log.Println("File removed:", event.Name)
		w.fileHandler.RemoveFile(event.Name)
		// If it was a dir, fsnotify removes the watch automatically
	}
}

func isSupportedDocumentEvent(path string) bool {
	return domain.IsSupportedDocumentPath(path)
}
