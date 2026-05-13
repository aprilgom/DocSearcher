package usecase

import "hwp-searcher/internal/domain"

type TextExtractor interface {
	ExtractText(path string) (string, error)
}

type DocumentWriter interface {
	IndexDocument(doc domain.IndexedDocument) error
	DeleteDocument(id domain.DocumentID) error
}

type Indexer struct {
	textExtractor TextExtractor
	documentIndex DocumentWriter
}

func NewIndexer(textExtractor TextExtractor, documentIndex DocumentWriter) Indexer {
	return Indexer{
		textExtractor: textExtractor,
		documentIndex: documentIndex,
	}
}

func (i Indexer) IndexFile(path string) error {
	content, err := i.textExtractor.ExtractText(path)
	if err != nil {
		return err
	}

	doc := domain.NewDocument(path, content)
	return i.documentIndex.IndexDocument(domain.NewIndexedDocument(doc))
}

func (i Indexer) RemoveFile(path string) error {
	return i.documentIndex.DeleteDocument(domain.DocumentID(path))
}
