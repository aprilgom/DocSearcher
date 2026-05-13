package parser

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"goHwpTxt"

	"github.com/ledongthuc/pdf"
)

// Parse extracts text from a file based on its extension
func Parse(path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".hwp", ".hwpx":
		return parseHWP(path)
	case ".pdf":
		return parsePDF(path)
	default:
		return "", fmt.Errorf("unsupported file type: %s", ext)
	}
}

type TextExtractor struct{}

func (TextExtractor) ExtractText(path string) (string, error) {
	return Parse(path)
}

func parseHWP(path string) (string, error) {
	return goHwpTxt.ExtractText(path)
}

func parsePDF(path string) (string, error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var buf bytes.Buffer
	b, err := r.GetPlainText()
	if err != nil {
		return "", err
	}
	buf.ReadFrom(b)
	return buf.String(), nil
}
