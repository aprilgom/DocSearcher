package goHwpTxt

import (
	"bytes"
	"io"
	"os"

	"goHwpTxt/pkg/hwp3"
	"goHwpTxt/pkg/hwp5"
	"goHwpTxt/pkg/hwpx"
)

// ExtractText extracts text from an HWP file (v3, v5, or HWPX).
// It automatically detects the file format based on the file signature.
func ExtractText(filename string) (string, error) {
	// Detect version by reading the first few bytes
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	// We only read the header here, so close it immediately after peeking
	defer f.Close()

	// Read first 8 bytes for OLE signature
	buf := make([]byte, 8)
	if _, err := io.ReadFull(f, buf); err != nil {
		return "", err
	}

	// OLE Signature: D0 CF 11 E0 A1 B1 1A E1
	oleSig := []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}
	if bytes.Equal(buf, oleSig) {
		// Try HWP v5
		return hwp5.ExtractText(filename)
	}

	// ZIP Signature: 50 4B 03 04 (PK..)
	zipSig := []byte{0x50, 0x4B, 0x03, 0x04}
	if bytes.Equal(buf[:4], zipSig) {
		// Try HWPX
		return hwpx.ExtractText(filename)
	}

	// If not OLE or ZIP, assume HWP v3 and try to parse
	return hwp3.ExtractText(filename)
}
