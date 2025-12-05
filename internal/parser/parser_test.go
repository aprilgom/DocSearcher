package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseHWP(t *testing.T) {
	// Path to the sample HWP file in goHwpTxt/testdata
	// We are running tests from internal/parser, so we need to go up to root then to goHwpTxt
	samplePath := filepath.Join("..", "..", "goHwpTxt", "testdata", "2019가단3702.hwp")
	absPath, err := filepath.Abs(samplePath)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skipf("Sample HWP file not found: %s", absPath)
	}

	text, err := Parse(absPath)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if text == "" {
		t.Error("Extracted text is empty")
	}

	// Check for some expected content (this depends on the file content)
	// For now, just checking it's not empty is a good start.
	// If we knew the content, we could check for specific strings.
	if !strings.Contains(text, "원") { // Assuming the file contains "원" (Won) or similar common Korean char
		t.Logf("Warning: '원' not found in extracted text. Text length: %d", len(text))
	}
}
