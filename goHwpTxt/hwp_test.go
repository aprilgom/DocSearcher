package goHwpTxt

import (
	"path/filepath"
	"testing"
)

func TestExtractText(t *testing.T) {
	// Find all files in testdata
	files, err := filepath.Glob("testdata/*")
	if err != nil {
		t.Fatal(err)
	}

	if len(files) == 0 {
		t.Skip("No test files found in testdata/")
	}

	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			t.Logf("Testing file: %s", file)
			text, err := ExtractText(file)
			if err != nil {
				t.Errorf("Failed to extract text from %s: %v", file, err)
				return
			}
			if len(text) == 0 {
				t.Errorf("Extracted text is empty for %s", file)
			}
			// Optional: print length for debug
			t.Logf("Extracted %d characters from %s", len(text), filepath.Base(file))
		})
	}
}
