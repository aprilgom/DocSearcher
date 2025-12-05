package hwp3

import (
	"fmt"
)

// HwpDocument represents the parsed HWP 3.0 document.
type HwpDocument struct {
	File       *HwpV3File
	Title      string
	Subject    string
	Creator    string
	Keywords   string
	Pages      []*HwpPage
	Paragraphs []*HwpParagraph // Flat list of all paragraphs? Or just root? C code seems to append to doc->paragraphs
}

// HwpPage represents a page in the document.
type HwpPage struct {
	Paragraphs []*HwpParagraph
}

// HwpParagraph represents a paragraph.
type HwpParagraph struct {
	Text *HwpText
}

// HwpText represents the text content of a paragraph.
type HwpText struct {
	Text string
}

func NewHwpDocument() *HwpDocument {
	return &HwpDocument{
		Pages:      make([]*HwpPage, 0),
		Paragraphs: make([]*HwpParagraph, 0),
	}
}

func NewHwpPage() *HwpPage {
	return &HwpPage{
		Paragraphs: make([]*HwpParagraph, 0),
	}
}

func NewHwpParagraph() *HwpParagraph {
	return &HwpParagraph{}
}

func NewHwpText(text string) *HwpText {
	return &HwpText{Text: text}
}

func (p *HwpParagraph) SetText(text *HwpText) {
	p.Text = text
}

func (d *HwpDocument) String() string {
	return fmt.Sprintf("Title: %s\nSubject: %s\nCreator: %s\nKeywords: %s\nPages: %d", d.Title, d.Subject, d.Creator, d.Keywords, len(d.Pages))
}
