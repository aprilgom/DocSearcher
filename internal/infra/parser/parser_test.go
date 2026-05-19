package parser

import (
	"archive/zip"
	"encoding/xml"
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
		{name: "Contents/section2.xml", text: "A&B <테스트>"},
	})

	// When
	text, err := Parse(path)

	// Then
	if err != nil {
		t.Fatalf("Parse(%q) returned error for synthetic HWPX fixture: %v", path, err)
	}
	for _, want := range []string{"첫 문단", "둘째 문단", "A&B <테스트>"} {
		if !strings.Contains(text, want) {
			t.Fatalf("Parse(%q) text = %q, want it to contain %q", path, text, want)
		}
	}
}

func TestParseDispatchesUppercaseHWPXExtension(t *testing.T) {
	path := writeSyntheticHWPX(t, t.TempDir(), "REPORT.HWPX", []hwpxSection{
		{name: "Contents/section0.xml", text: "대문자 확장자"},
	})

	text, err := Parse(path)

	if err != nil {
		t.Fatalf("Parse(%q) returned error: %v", path, err)
	}
	if !strings.Contains(text, "대문자 확장자") {
		t.Fatalf("Parse(%q) text = %q, want it to contain uppercase-extension fixture text", path, text)
	}
}

func TestParseErrorCases(t *testing.T) {
	tests := []struct {
		name          string
		path          func(t *testing.T) string
		wantErrSubstr []string
	}{
		{
			name: "malformed_hwpx_zip",
			path: func(t *testing.T) string {
				path := filepath.Join(t.TempDir(), "broken.hwpx")
				if err := os.WriteFile(path, []byte("PK\x03\x04not a valid zip archive"), 0o600); err != nil {
					t.Fatalf("Write malformed HWPX zip %q returned error: %v", path, err)
				}
				return path
			},
			wantErrSubstr: []string{"zip"},
		},
		{
			name: "malformed_hwpx_xml",
			path: func(t *testing.T) string {
				return writeSyntheticHWPXRawXML(t, t.TempDir(), "broken-xml.hwpx", "Contents/section0.xml", `<hp:section><hp:p><hp:t>닫히지 않은 텍스트`)
			},
			wantErrSubstr: []string{"failed to parse Contents/section0.xml", "unexpected EOF"},
		},
		{
			name: "malformed_pdf",
			path: func(t *testing.T) string {
				path := filepath.Join(t.TempDir(), "broken.pdf")
				if err := os.WriteFile(path, []byte("%PDF-1.7\nnot a complete pdf\n"), 0o600); err != nil {
					t.Fatalf("Write malformed PDF %q returned error: %v", path, err)
				}
				return path
			},
		},
		{
			name: "unsupported_file_type",
			path: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "notes.txt")
			},
			wantErrSubstr: []string{"unsupported file type"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := test.path(t)
			_, err := Parse(path)

			if err == nil {
				t.Fatalf("Parse(%q) returned nil error, want error", path)
			}
			for _, want := range test.wantErrSubstr {
				if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(want)) {
					t.Errorf("Parse(%q) error = %q, want it to contain %q", path, err, want)
				}
			}
		})
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

func writeSyntheticHWPXRawXML(t *testing.T, dir, name, sectionName, content string) string {
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

	file, err := writer.Create(sectionName)
	if err != nil {
		t.Fatalf("Create section %q in synthetic HWPX returned error: %v", sectionName, err)
	}
	if _, err := file.Write([]byte(content)); err != nil {
		t.Fatalf("Write section %q in synthetic HWPX returned error: %v", sectionName, err)
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
