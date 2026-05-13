package domain

import (
	"path/filepath"
	"strings"
)

func IsSupportedDocumentPath(path string) bool {
	name := filepath.Base(path)
	if strings.Contains(name, "~$") || strings.HasSuffix(strings.ToLower(name), ".tmp") {
		return false
	}

	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".hwp" || ext == ".hwpx" || ext == ".pdf"
}
