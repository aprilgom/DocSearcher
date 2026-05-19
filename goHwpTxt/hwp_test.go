package goHwpTxt

import (
	"archive/zip"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractTextExtractsTextFromSyntheticHWPX(t *testing.T) {
	// Given
	path := writeSyntheticHWPX(t, t.TempDir(), "fixture.hwpx", []hwpxSection{
		{name: "Contents/section0.xml", text: "첫 문단"},
		{name: "Contents/header0.xml", text: "머리말"},
		{name: "Preview/ignored.xml", text: "무시할 문장"},
	})

	// When
	text, err := ExtractText(path)

	// Then
	if err != nil {
		t.Fatalf("ExtractText(%q) returned error for synthetic HWPX fixture: %v", path, err)
	}
	for _, want := range []string{"첫 문단", "머리말"} {
		if !strings.Contains(text, want) {
			t.Fatalf("ExtractText(%q) text = %q, want it to contain %q", path, text, want)
		}
	}
	if strings.Contains(text, "무시할 문장") {
		t.Fatalf("ExtractText(%q) text = %q, want Preview XML to be ignored", path, text)
	}
}

func TestExtractTextPreservesEscapedXMLTextFromSyntheticHWPX(t *testing.T) {
	// Given
	want := `A&B <테스트> "따옴표" '작은따옴표'`
	path := writeSyntheticHWPX(t, t.TempDir(), "escaped.hwpx", []hwpxSection{
		{name: "Contents/section0.xml", text: want},
	})

	// When
	text, err := ExtractText(path)

	// Then
	if err != nil {
		t.Fatalf("ExtractText(%q) returned error for escaped XML text fixture: %v", path, err)
	}
	if !strings.Contains(text, want) {
		t.Fatalf("ExtractText(%q) text = %q, want it to contain %q", path, text, want)
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
		content := `<?xml version="1.0" encoding="UTF-8"?><hp:section xmlns:hp="http://www.hancom.co.kr/hwpml/2011/paragraph"><hp:p><hp:t>` + escapeXMLText(t, section.text) + `</hp:t></hp:p></hp:section>`
		if _, err := file.Write([]byte(content)); err != nil {
			t.Fatalf("Write section %q in synthetic HWPX returned error: %v", section.name, err)
		}
	}

	return path
}

func escapeXMLText(t *testing.T, text string) string {
	t.Helper()
	var builder strings.Builder
	if err := xml.EscapeText(&builder, []byte(text)); err != nil {
		t.Fatalf("EscapeText(%q) returned error: %v", text, err)
	}
	return builder.String()
}
