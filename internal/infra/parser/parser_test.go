package parser

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseExtractsTextFromSyntheticHWPX(t *testing.T) {
	// Given
	path := writeSyntheticHWPX(t, t.TempDir(), "report.hwpx", []hwpxSection{
		{name: "Contents/section0.xml", text: "첫 문단"},
		{name: "Contents/section1.xml", text: "둘째 문단"},
	})

	// When
	text, err := Parse(path)

	// Then
	if err != nil {
		t.Fatalf("Parse(%q) returned error for synthetic HWPX fixture: %v", path, err)
	}
	for _, want := range []string{"첫 문단", "둘째 문단"} {
		if !strings.Contains(text, want) {
			t.Fatalf("Parse(%q) text = %q, want it to contain %q", path, text, want)
		}
	}
}

func TestParseRejectsUnsupportedFileType(t *testing.T) {
	// Given
	path := filepath.Join(t.TempDir(), "notes.txt")

	// When
	_, err := Parse(path)

	// Then
	if err == nil {
		t.Fatalf("Parse(%q) returned nil error, want unsupported file type error", path)
	}
	if !strings.Contains(err.Error(), "unsupported file type") {
		t.Fatalf("Parse(%q) error = %q, want unsupported file type diagnostic", path, err)
	}
}

type hwpxSection struct {
	name string
	text string
}

func writeSyntheticHWPX(t *testing.T, dir, name string, sections []hwpxSection) string {
	t.Helper()

	path := filepath.Join(dir, name)
	out, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create synthetic HWPX %q returned error: %v", path, err)
	}
	defer func() {
		if err := out.Close(); err != nil {
			t.Fatalf("Close synthetic HWPX file %q returned error: %v", path, err)
		}
	}()

	writer := zip.NewWriter(out)
	defer func() {
		if err := writer.Close(); err != nil {
			t.Fatalf("Close synthetic HWPX %q returned error: %v", path, err)
		}
	}()

	for _, section := range sections {
		file, err := writer.Create(section.name)
		if err != nil {
			t.Fatalf("Create section %q in synthetic HWPX returned error: %v", section.name, err)
		}
		xml := `<?xml version="1.0" encoding="UTF-8"?><hp:section xmlns:hp="http://www.hancom.co.kr/hwpml/2011/paragraph"><hp:p><hp:t>` + section.text + `</hp:t></hp:p></hp:section>`
		if _, err := file.Write([]byte(xml)); err != nil {
			t.Fatalf("Write section %q in synthetic HWPX returned error: %v", section.name, err)
		}
	}

	return path
}
