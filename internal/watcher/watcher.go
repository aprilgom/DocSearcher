package watcher

import (
	"hwp-searcher/internal/config"
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
	StartIndexing func(path string)
}

var (
	watcher     *fsnotify.Watcher
	done        chan bool
	fileHandler FileHandler = noopFileHandler{}
)

type noopFileHandler struct{}

func (noopFileHandler) IndexFile(path string) {}

func (noopFileHandler) RemoveFile(path string) {}

func SetFileHandler(handler FileHandler) {
	if handler == nil {
		fileHandler = noopFileHandler{}
		return
	}
	fileHandler = handler
}

func Start(registry Registry) {
	var err error
	watcher, err = fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	done = make(chan bool)

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				handleEvent(event)
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("Watcher error:", err)
			}
		}
	}()

	// Watch paths from config
	config.Load()
	for _, path := range config.Current.WatchedPaths {
		registry.AddPath(domain.WatchedPath(path))
	}
}

func addPath(root string) error {
	// Recursive add
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			err = watcher.Add(path)
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
	// We can't easily remove recursive watchers without walking again
	// But simply removing the root from config prevents it from being watched next time
	// For now, we just remove the root watcher
	watcher.Remove(root)
}

func (r Registry) AddPath(path domain.WatchedPath) error {
	if err := addPath(string(path)); err != nil {
		return err
	}
	if r.StartIndexing != nil {
		r.StartIndexing(string(path))
	}
	return nil
}

func (Registry) RemovePath(path domain.WatchedPath) error {
	RemovePath(string(path))
	return nil
}

func handleEvent(event fsnotify.Event) {
	if event.Op&fsnotify.Create == fsnotify.Create {
		info, err := os.Stat(event.Name)
		if err == nil && info.IsDir() {
			// If directory created, watch it
			_ = addPath(event.Name)
			return
		}
	}

	// Ignore temporary files
	if !isSupportedDocumentEvent(event.Name) {
		return
	}

	if event.Op&fsnotify.Create == fsnotify.Create {
		log.Println("File created:", event.Name)
		fileHandler.IndexFile(event.Name)
	}

	if event.Op&fsnotify.Write == fsnotify.Write {
		log.Println("File modified:", event.Name)
		fileHandler.IndexFile(event.Name)
	}

	if event.Op&fsnotify.Remove == fsnotify.Remove || event.Op&fsnotify.Rename == fsnotify.Rename {
		// If it was a file, remove from index
		// We can't easily check if it was a dir or file since it's gone
		// But we can try to remove from index anyway
		log.Println("File removed:", event.Name)
		fileHandler.RemoveFile(event.Name)
		// If it was a dir, fsnotify removes the watch automatically
	}
}

func isSupportedDocumentEvent(path string) bool {
	return domain.IsSupportedDocumentPath(path)
}
