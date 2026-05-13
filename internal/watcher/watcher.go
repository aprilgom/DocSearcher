package watcher

import (
	"hwp-searcher/internal/config"
	"hwp-searcher/internal/domain"
	"hwp-searcher/internal/indexer"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

type Registry struct{}

var (
	watcher *fsnotify.Watcher
	done    chan bool
)

func Start() {
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
		AddPath(path)
	}
}

func AddPath(root string) {
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
	}

	// Also trigger an initial scan
	indexer.Start(root)
}

func RemovePath(root string) {
	// We can't easily remove recursive watchers without walking again
	// But simply removing the root from config prevents it from being watched next time
	// For now, we just remove the root watcher
	watcher.Remove(root)
}

func (Registry) AddPath(path domain.WatchedPath) error {
	AddPath(string(path))
	return nil
}

func (Registry) RemovePath(path domain.WatchedPath) error {
	RemovePath(string(path))
	return nil
}

func handleEvent(event fsnotify.Event) {
	// Ignore temporary files
	if strings.Contains(event.Name, "~$") || strings.HasSuffix(event.Name, ".tmp") {
		return
	}

	ext := strings.ToLower(filepath.Ext(event.Name))
	isTarget := ext == ".hwp" || ext == ".pdf"

	if event.Op&fsnotify.Create == fsnotify.Create {
		info, err := os.Stat(event.Name)
		if err == nil && info.IsDir() {
			// If directory created, watch it
			AddPath(event.Name)
		} else if isTarget {
			log.Println("File created:", event.Name)
			indexer.IndexFile(event.Name)
		}
	}

	if event.Op&fsnotify.Write == fsnotify.Write {
		if isTarget {
			log.Println("File modified:", event.Name)
			indexer.IndexFile(event.Name)
		}
	}

	if event.Op&fsnotify.Remove == fsnotify.Remove || event.Op&fsnotify.Rename == fsnotify.Rename {
		// If it was a file, remove from index
		// We can't easily check if it was a dir or file since it's gone
		// But we can try to remove from index anyway
		if isTarget {
			log.Println("File removed:", event.Name)
			indexer.RemoveFile(event.Name)
		}
		// If it was a dir, fsnotify removes the watch automatically
	}
}
