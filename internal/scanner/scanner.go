package scanner

import (
	"hwp-searcher/internal/domain"
	"os"
	"path/filepath"
)

func Walk(root string, visit func(path string) error) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && domain.IsSupportedDocumentPath(path) {
			return visit(path)
		}
		return nil
	})
}
