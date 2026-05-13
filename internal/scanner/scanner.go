package scanner

import (
	"os"
	"path/filepath"
	"strings"
)

func Walk(root string, visit func(path string) error) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && IsSupportedDocumentFile(path) {
			return visit(path)
		}
		return nil
	})
}

func IsSupportedDocumentFile(path string) bool {
	name := filepath.Base(path)
	if strings.Contains(name, "~$") || strings.HasSuffix(strings.ToLower(name), ".tmp") {
		return false
	}

	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".hwp" || ext == ".hwpx" || ext == ".pdf"
}
