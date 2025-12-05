package hwpx

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"sort"
	"strings"
)

// ExtractText extracts text from an HWPX file.
func ExtractText(filename string) (string, error) {
	r, err := zip.OpenReader(filename)
	if err != nil {
		return "", err
	}
	defer r.Close()

	// Find content XML files
	var sectionFiles []*zip.File
	for _, f := range r.File {
		// Check for various content files
		if strings.HasSuffix(f.Name, ".xml") && strings.HasPrefix(f.Name, "Contents/") {
			if strings.HasPrefix(f.Name, "Contents/section") ||
				strings.HasPrefix(f.Name, "Contents/header") ||
				strings.HasPrefix(f.Name, "Contents/footer") ||
				strings.HasPrefix(f.Name, "Contents/footnote") ||
				strings.HasPrefix(f.Name, "Contents/endnote") ||
				strings.HasPrefix(f.Name, "Contents/masterpage") {
				sectionFiles = append(sectionFiles, f)
			}
		}
	}

	// Sort by name to ensure correct order (section0.xml, section1.xml, ...)
	sort.Slice(sectionFiles, func(i, j int) bool {
		return sectionFiles[i].Name < sectionFiles[j].Name
	})

	var sb strings.Builder
	for _, f := range sectionFiles {
		text, err := extractTextFromXML(f)
		if err != nil {
			return "", fmt.Errorf("failed to parse %s: %w", f.Name, err)
		}
		sb.WriteString(text)
	}

	return sb.String(), nil
}

func extractTextFromXML(f *zip.File) (string, error) {
	rc, err := f.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	decoder := xml.NewDecoder(rc)
	var sb strings.Builder

	for {
		t, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		switch se := t.(type) {
		case xml.StartElement:
			// <hp:t> contains text
			if se.Name.Local == "t" {
				var text string
				if err := decoder.DecodeElement(&text, &se); err != nil {
					return "", err
				}
				sb.WriteString(text)
			}
			// <hp:p> implies a paragraph break (newline)
			// We might want to add newline at the end of <hp:p> or <hp:lineSeg>?
			// For simplicity, let's just append newline if we encounter a paragraph end,
			// but XML parsing is stream-based.
			// Let's check for EndElement.
		case xml.EndElement:
			if se.Name.Local == "p" {
				sb.WriteString("\n")
			}
		}
	}
	return sb.String(), nil
}
