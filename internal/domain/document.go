package domain

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

type DocumentID string

type WatchedPath string

type Document struct {
	ID      DocumentID
	Path    string
	Content string
}

type IndexedDocument struct {
	ID             DocumentID
	RootID         RootID
	RelativePath   RelativePath
	Content        string
	ContentNoSpace string
	Path           string
	ServerPath     string
}

type IndexSchema struct {
	ContentField        string
	ContentNoSpaceField string
	PathField           string
	RootIDField         string
	RelativePathField   string
	ServerPathField     string
}

type SearchPolicy struct {
	MinQueryLength      int
	PartialMatchMinGram int
	PartialMatchMaxGram int
	IgnoreWhitespace    bool
}

type SearchMode int

const (
	SearchModeQuery SearchMode = iota
	SearchModeExact
	SearchModeIgnoreSpaces
)

type SearchRequest struct {
	Query string
	Mode  SearchMode
}

type SearchResult struct {
	Total uint64
	Hits  []SearchHit
}

type SearchHit struct {
	ID           DocumentID
	RootID       RootID
	RelativePath RelativePath
	Path         string
	Fragment     string
}

type Stats struct {
	DocumentCount    uint64
	WatchedPathCount int
	Indexing         bool
}

func NewDocument(path string, content string) Document {
	return Document{
		ID:      DocumentID(path),
		Path:    path,
		Content: content,
	}
}

func NewIndexedDocument(doc Document) IndexedDocument {
	return IndexedDocument{
		ID:             doc.ID,
		Content:        doc.Content,
		ContentNoSpace: NormalizeNoSpaceContent(doc.Content),
		Path:           doc.Path,
		ServerPath:     doc.Path,
	}
}

func DefaultIndexSchema() IndexSchema {
	return IndexSchema{
		ContentField:        "content",
		ContentNoSpaceField: "content_nospace",
		PathField:           "path",
		RootIDField:         "root_id",
		RelativePathField:   "relative_path",
		ServerPathField:     "server_path",
	}
}

func PersonNameSearchPolicy() SearchPolicy {
	return SearchPolicy{
		MinQueryLength:      2,
		PartialMatchMinGram: 2,
		PartialMatchMaxGram: 5,
		IgnoreWhitespace:    true,
	}
}

func NormalizeNoSpaceContent(content string) string {
	contentNoSpace := strings.ReplaceAll(content, " ", "")
	contentNoSpace = strings.ReplaceAll(contentNoSpace, "\n", "")
	contentNoSpace = strings.ReplaceAll(contentNoSpace, "\t", "")
	contentNoSpace = strings.ReplaceAll(contentNoSpace, "\r", "")
	return contentNoSpace
}

func SearchModeFromFlags(exact bool, ignoreSpaces bool) SearchMode {
	if ignoreSpaces {
		return SearchModeIgnoreSpaces
	}
	if exact {
		return SearchModeExact
	}
	return SearchModeQuery
}

func (req SearchRequest) Validate(policy SearchPolicy) error {
	if utf8.RuneCountInString(strings.TrimSpace(req.Query)) < policy.MinQueryLength {
		return fmt.Errorf("query must be at least %d characters", policy.MinQueryLength)
	}
	return nil
}
