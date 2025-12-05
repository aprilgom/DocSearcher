package hwp5

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"unicode"
	"unicode/utf16"

	"github.com/richardlehane/mscfb"
)

var (
	Signature = []byte("HWP Document File")
)

// HwpParser handles parsing of HWP files.
type HwpParser struct {
	reader *mscfb.Reader
	// We might need to keep the file open if we were doing lazy loading,
	// but mscfb reads everything or relies on the underlying reader.
	// For this implementation, we'll read relevant streams into memory during Parse.
}

// NewParser creates a new HwpParser.
// Note: mscfb.New requires io.ReaderAt to support random access for OLE parsing.
func NewParser(r io.ReaderAt) (*HwpParser, error) {
	doc, err := mscfb.New(r)
	if err != nil {
		return nil, err
	}
	return &HwpParser{reader: doc}, nil
}

// Parse reads the HWP file structure and extracts text.
func (p *HwpParser) Parse() (string, error) {
	streams := make(map[string][]byte)
	var bodyTextSections []string

	// Iterate through all entries in the OLE container
	for entry, err := p.reader.Next(); err == nil; entry, err = p.reader.Next() {
		if entry.Name == "FileHeader" {
			buf := make([]byte, entry.Size)
			_, err := io.ReadFull(p.reader, buf)
			if err != nil {
				return "", err
			}
			streams["FileHeader"] = buf
		} else if strings.HasPrefix(entry.Name, "Section") {
			buf := make([]byte, entry.Size)
			_, err := io.ReadFull(p.reader, buf)
			if err != nil {
				return "", err
			}
			streams[entry.Name] = buf
			bodyTextSections = append(bodyTextSections, entry.Name)
		}
	}

	// Validate FileHeader
	headerData, ok := streams["FileHeader"]
	if !ok {
		return "", errors.New("FileHeader not found")
	}

	if !bytes.HasPrefix(headerData, Signature) {
		return "", errors.New("invalid HWP signature")
	}

	flags := binary.LittleEndian.Uint32(headerData[36:40])
	isCompressed := (flags & 1) != 0
	isEncrypted := (flags & 2) != 0

	if isEncrypted {
		return "", errors.New("encrypted files are not supported")
	}

	// Sort sections to ensure text order
	sort.Strings(bodyTextSections)

	var sb strings.Builder

	for _, sectionName := range bodyTextSections {
		data := streams[sectionName]
		var r io.Reader = bytes.NewReader(data)

		if isCompressed {
			zr := flate.NewReader(r)
			decompressed, err := io.ReadAll(zr)
			zr.Close()
			if err != nil {
				// Warning: decompression failed, trying raw data as fallback
				// In a real app, we might want to log this via a logger interface, but not direct to stderr.
				r = bytes.NewReader(data)
			} else {
				r = bytes.NewReader(decompressed)
			}
		}

		// Read records from the section stream
		for {
			record, err := ReadRecord(r)
			if err == io.EOF {
				break
			}
			if err != nil {
				return "", fmt.Errorf("failed to read record in %s: %v", sectionName, err)
			}

			if record.TagID == HWPTAG_PARA_TEXT {
				text := extractTextFromPayload(record.Payload)
				sb.WriteString(text)
				sb.WriteString("\n") // Paragraph break
			}
		}
	}

	return sb.String(), nil
}

// ExtractText is a helper function to extract text from a file path.
// It wraps the HwpParser for convenience.
func ExtractText(filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()

	parser, err := NewParser(f)
	if err != nil {
		return "", err
	}

	return parser.Parse()
}

// extractTextFromPayload extracts text from the payload, filtering control characters.
func extractTextFromPayload(payload []byte) string {
	if len(payload)%2 != 0 {
		if len(payload) > 0 {
			payload = payload[:len(payload)-1]
		}
	}

	u16s := make([]uint16, len(payload)/2)
	binary.Read(bytes.NewReader(payload), binary.LittleEndian, &u16s)

	var sb strings.Builder
	for _, r := range utf16.Decode(u16s) {
		// Filter control characters
		// Keep newlines, tabs, and normal text.
		// HWP uses some control codes for inline objects which we want to skip.
		if unicode.IsPrint(r) || unicode.IsSpace(r) {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}
