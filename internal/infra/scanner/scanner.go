package scanner

import (
	"hwp-searcher/internal/domain"
	"os"
	"path/filepath"
)

type WalkOptions struct {
	SkipDir func(path string) bool
}

func Walk(root string, visit func(path string) error) error {
	return WalkWithOptions(root, WalkOptions{}, visit)
}

func WalkWithOptions(root string, options WalkOptions, visit func(path string) error) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && options.SkipDir != nil && options.SkipDir(path) {
			return filepath.SkipDir
		}
		if info.Mode()&os.ModeSymlink != 0 {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !info.IsDir() && domain.IsSupportedDocumentPath(path) {
			return visit(path)
		}
		return nil
	})
}
