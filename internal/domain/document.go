package domain

import "strings"

type DocumentID string

type WatchedPath string

type Document struct {
	ID      DocumentID
	Path    string
	Content string
}

type IndexedDocument struct {
	ID             DocumentID
	Content        string
	ContentNoSpace string
	Path           string
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
