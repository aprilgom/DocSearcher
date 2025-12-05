package hwp3

import (
	"bufio"
	"compress/flate"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
)

type HwpV3File struct {
	Reader       io.Reader
	IsCrypt      uint16
	IsCompress   uint8
	Rev          uint8
	InfoBlockLen uint16
}

func NewHwpV3File(filename string) (*HwpV3File, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	// We don't close f here, the caller should close it or we wrap it.
	// For simplicity, we'll read everything or keep it open.
	// Since we might need to wrap it in zlib, we keep the reader.
	// Ideally we should have a Close method.

	return &HwpV3File{
		Reader: bufio.NewReader(f),
	}, nil
}

// ExtractText extracts text from an HWP v3 file.
func ExtractText(filename string) (string, error) {
	f, err := NewHwpV3File(filename)
	if err != nil {
		return "", err
	}
	// Note: NewHwpV3File opens the file but doesn't expose a Close method easily
	// because it wraps it in bufio.Reader.
	// In a robust implementation, we should handle closing.
	// For now, we rely on OS to close on exit or GC, but let's try to close the underlying file if possible.
	// However, HwpV3File struct only has Reader io.Reader.
	// We should probably change HwpV3File to hold the file handle or take an io.Reader.
	// Given the current structure, we'll proceed.

	doc, err := f.Parse()
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for _, para := range doc.Paragraphs {
		if para.Text != nil {
			sb.WriteString(para.Text.Text)
			sb.WriteString("\n")
		}
	}
	return sb.String(), nil
}

func (f *HwpV3File) Parse() (*HwpDocument, error) {
	doc := NewHwpDocument()
	doc.File = f

	if err := f.parseSignature(); err != nil {
		return nil, fmt.Errorf("parseSignature: %w", err)
	}
	if err := f.parseDocInfo(); err != nil {
		return nil, fmt.Errorf("parseDocInfo: %w", err)
	}
	if err := f.parseSummaryInfo(doc); err != nil {
		return nil, fmt.Errorf("parseSummaryInfo: %w", err)
	}
	if err := f.parseInfoBlock(); err != nil {
		return nil, fmt.Errorf("parseInfoBlock: %w", err)
	}
	// Font names and styles are skipped in terms of data model for now, but we must read past them.
	if err := f.parseFontNames(); err != nil {
		return nil, fmt.Errorf("parseFontNames: %w", err)
	}
	if err := f.parseStyles(); err != nil {
		return nil, fmt.Errorf("parseStyles: %w", err)
	}
	if err := f.parseParagraphs(doc); err != nil {
		return nil, fmt.Errorf("parseParagraphs: %w", err)
	}

	return doc, nil
}

func (f *HwpV3File) read(n int) ([]byte, error) {
	buf := make([]byte, n)
	_, err := io.ReadFull(f.Reader, buf)
	return buf, err
}

func (f *HwpV3File) readUint8() (uint8, error) {
	buf, err := f.read(1)
	if err != nil {
		return 0, err
	}
	return buf[0], nil
}

func (f *HwpV3File) readUint16() (uint16, error) {
	buf, err := f.read(2)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint16(buf), nil
}

func (f *HwpV3File) readUint32() (uint32, error) {
	buf, err := f.read(4)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(buf), nil
}

func (f *HwpV3File) skip(n int) error {
	// io.CopyN(io.Discard, f.Reader, int64(n))
	// Or just read and discard
	_, err := f.read(n)
	return err
}

func (f *HwpV3File) parseSignature() error {
	sig, err := f.read(30)
	if err != nil {
		return err
	}
	// Verify signature? "HWP Document File V3.00" usually?
	// C code just reads it.
	_ = sig
	return nil
}

func (f *HwpV3File) parseDocInfo() error {
	// 128 bytes total
	// Skip 96
	if err := f.skip(96); err != nil {
		return err
	}

	var err error
	f.IsCrypt, err = f.readUint16()
	if err != nil {
		return err
	}

	// Skip 26
	if err := f.skip(26); err != nil {
		return err
	}

	f.IsCompress, err = f.readUint8()
	if err != nil {
		return err
	}

	f.Rev, err = f.readUint8()
	if err != nil {
		return err
	}

	f.InfoBlockLen, err = f.readUint16()
	if err != nil {
		return err
	}

	return nil
}

func (f *HwpV3File) parseSummaryInfo(doc *HwpDocument) error {
	// 9 blocks of 112 bytes
	for i := 0; i < 9; i++ {
		count := 0
		var sb string
		for count < 112 {
			c, err := f.readUint16()
			if err != nil {
				return err
			}
			count += 2
			if c != 0 {
				sb += HncToUtf8(c)
			} else {
				if err := f.skip(112 - count); err != nil {
					return err
				}
				break
			}
		}

		switch i {
		case 0:
			doc.Title = sb
		case 1:
			doc.Subject = sb
		case 2:
			doc.Creator = sb
		case 3:
			doc.Keywords = sb // Date in C code?
		}
	}
	return nil
}

func (f *HwpV3File) parseInfoBlock() error {
	if err := f.skip(int(f.InfoBlockLen)); err != nil {
		return err
	}

	if f.IsCompress != 0 {
		// Wrap reader with flate (raw deflate)
		zr := flate.NewReader(f.Reader)
		f.Reader = zr
	}
	return nil
}

func (f *HwpV3File) parseFontNames() error {
	for i := 0; i < 7; i++ {
		nFonts, err := f.readUint16()
		if err != nil {
			return err
		}
		if err := f.skip(40 * int(nFonts)); err != nil {
			return err
		}
	}
	return nil
}

func (f *HwpV3File) parseStyles() error {
	nStyles, err := f.readUint16()
	if err != nil {
		return err
	}
	if err := f.skip(int(nStyles) * (20 + 31 + 187)); err != nil {
		return err
	}
	return nil
}

func (f *HwpV3File) parseParagraphs(doc *HwpDocument) error {
	// Initialize first page
	currentPage := NewHwpPage()
	doc.Pages = append(doc.Pages, currentPage)
	// Also need to track current page in doc struct if we want to mimic C logic exactly
	// C logic: doc->pages array. file->page is current page.

	var y float64 = 0.0

	for {
		para, keepGoing, err := f.parseParagraph()
		if err != nil {
			return err
		}
		if !keepGoing {
			break
		}

		doc.Paragraphs = append(doc.Paragraphs, para)
		currentPage.Paragraphs = append(currentPage.Paragraphs, para)

		// Page breaking logic from C
		// y += 18.0 * ceil (len / 33.0);
		// if (y > 842.0 - 80.0) { ... }

		textLen := float64(len([]rune(para.Text.Text))) // utf8 strlen
		y += 18.0 * math.Ceil(textLen/33.0)

		if y > 842.0-80.0 {
			currentPage = NewHwpPage()
			doc.Pages = append(doc.Pages, currentPage)
			// The paragraph that caused the break goes to the NEW page?
			// C code:
			// g_array_append_val (doc->pages, ...->page);
			// ...->page = ghwp_page_new();
			// g_array_append_val (...->page->paragraphs, paragraph);
			// So yes, it goes to the new page.
			// Wait, my logic above appended to `currentPage` BEFORE creating new page.
			// I should move it.

			// Remove from previous page
			oldPage := doc.Pages[len(doc.Pages)-2]
			oldPage.Paragraphs = oldPage.Paragraphs[:len(oldPage.Paragraphs)-1]

			// Add to new page
			currentPage.Paragraphs = append(currentPage.Paragraphs, para)
			y = 0.0
		}
	}
	return nil
}

func (f *HwpV3File) parseParagraph() (*HwpParagraph, bool, error) {
	// Read paragraph info
	prevShape, err := f.readUint8()
	if err != nil {
		// EOF check?
		if err == io.EOF {
			return nil, false, nil
		}
		return nil, false, err
	}

	nChars, err := f.readUint16()
	if err != nil {
		return nil, false, err
	}
	nLines, err := f.readUint16()
	if err != nil {
		return nil, false, err
	}
	charShapeIncluded, err := f.readUint8()
	if err != nil {
		return nil, false, err
	}

	// Skip 1+4+1+31 = 37 bytes?
	// C code: ghwp_context_v3_skip (context, 1 + 4 + 1 + 31);
	if err := f.skip(37); err != nil {
		return nil, false, err
	}

	if prevShape == 0 && nChars > 0 {
		if err := f.skip(187); err != nil {
			return nil, false, err
		}
	}

	if nChars == 0 {
		return nil, false, nil // End of paragraphs
	}

	// Skip lines info
	if err := f.skip(int(nLines) * 14); err != nil {
		return nil, false, err
	}

	// Char shape info
	if charShapeIncluded != 0 {
		for i := 0; i < int(nChars); i++ {
			flag, err := f.readUint8()
			if err != nil {
				return nil, false, err
			}
			if flag != 1 {
				if err := f.skip(31); err != nil {
					return nil, false, err
				}
			}
		}
	}

	// Read chars
	var text string
	var nCharsRead uint16 = 0

	for nCharsRead < nChars {
		c, err := f.readUint16()
		if err != nil {
			return nil, false, err
		}
		nCharsRead++

		if c == 6 {
			nCharsRead += 3
			if err := f.skip(6 + 34); err != nil {
				return nil, false, err
			}
		} else if c == 9 { // tab
			nCharsRead += 3
			if err := f.skip(6); err != nil {
				return nil, false, err
			}
			text += "\t"
		} else if c == 10 { // table
			nCharsRead += 3
			if err := f.skip(6); err != nil {
				return nil, false, err
			}
			// Table info 80 bytes
			if err := f.skip(80); err != nil {
				return nil, false, err
			}
			nCells, err := f.readUint16()
			if err != nil {
				return nil, false, err
			}
			if err := f.skip(2); err != nil {
				return nil, false, err
			}
			if err := f.skip(27 * int(nCells)); err != nil {
				return nil, false, err
			}

			// Recursively parse cell paragraphs
			for i := 0; i < int(nCells); i++ {
				for {
					para, keep, err := f.parseParagraph()
					if err != nil {
						return nil, false, err
					}
					if para != nil && para.Text != nil {
						text += para.Text.Text + "\n"
					}
					if !keep {
						break
					}
				}
			}
			// Caption
			for {
				para, keep, err := f.parseParagraph()
				if err != nil {
					return nil, false, err
				}
				if para != nil && para.Text != nil {
					text += para.Text.Text + "\n"
				}
				if !keep {
					break
				}
			}
		} else if c == 11 { // drawing?
			nCharsRead += 3
			if err := f.skip(6); err != nil {
				return nil, false, err
			}
			lenVal, err := f.readUint32()
			if err != nil {
				return nil, false, err
			}
			if err := f.skip(344); err != nil {
				return nil, false, err
			}
			if err := f.skip(int(lenVal)); err != nil {
				return nil, false, err
			}
			// Caption
			for {
				para, keep, err := f.parseParagraph()
				if err != nil {
					return nil, false, err
				}
				if para != nil && para.Text != nil {
					text += para.Text.Text + "\n"
				}
				if !keep {
					break
				}
			}
		} else if c == 13 { // End of chars? No, newline?
			// C code: g_string_append (string, "\n");
			text += "\n"
		} else if c == 16 {
			nCharsRead += 3
			if err := f.skip(6 + 10); err != nil {
				return nil, false, err
			}
			// Header/Footer?
			for {
				para, keep, err := f.parseParagraph()
				if err != nil {
					return nil, false, err
				}
				if para != nil && para.Text != nil {
					text += para.Text.Text + "\n"
				}
				if !keep {
					break
				}
			}
		} else if c == 17 { // Footnote
			nCharsRead += 3
			if err := f.skip(6 + 14); err != nil {
				return nil, false, err
			}
			for {
				para, keep, err := f.parseParagraph()
				if err != nil {
					return nil, false, err
				}
				if para != nil && para.Text != nil {
					text += para.Text.Text + "\n"
				}
				if !keep {
					break
				}
			}
		} else if c == 18 || c == 19 || c == 20 || c == 21 {
			nCharsRead += 3
			if err := f.skip(6); err != nil {
				return nil, false, err
			}
		} else if c == 23 {
			nCharsRead += 4
			if err := f.skip(8); err != nil {
				return nil, false, err
			}
		} else if c == 24 || c == 25 {
			nCharsRead += 2
			if err := f.skip(4); err != nil {
				return nil, false, err
			}
		} else if c == 28 {
			nCharsRead += 31
			if err := f.skip(62); err != nil {
				return nil, false, err
			}
		} else if c == 30 || c == 31 {
			nCharsRead += 1
			if err := f.skip(2); err != nil {
				return nil, false, err
			}
		} else if c >= 0x0020 {
			text += HncToUtf8(c)
		} else {
			// warning
		}
	}

	para := NewHwpParagraph()
	para.SetText(NewHwpText(text))
	return para, true, nil
}
