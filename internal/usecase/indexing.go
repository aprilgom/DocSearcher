package usecase

import (
	"fmt"
	"hwp-searcher/internal/domain"
	"os"
	"path/filepath"
	"strings"
)

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
	documentRoots []domain.DocumentRoot
}

func NewIndexer(textExtractor TextExtractor, documentIndex DocumentWriter, documentRoots ...[]domain.DocumentRoot) Indexer {
	var roots []domain.DocumentRoot
	if len(documentRoots) > 0 {
		roots = append([]domain.DocumentRoot(nil), documentRoots[0]...)
	}
	return Indexer{
		textExtractor: textExtractor,
		documentIndex: documentIndex,
		documentRoots: roots,
	}
}

func (i Indexer) IndexFile(path string) error {
	if isSymlink(path) {
		return nil
	}
	content, err := i.textExtractor.ExtractText(path)
	if err != nil {
		return err
	}

	doc, err := i.newIndexedDocument(path, content)
	if err != nil {
		return err
	}
	return i.documentIndex.IndexDocument(doc)
}

func isSymlink(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSymlink != 0
}

func (i Indexer) RemoveFile(path string) error {
	id, err := i.documentIDForPath(path)
	if err != nil {
		return err
	}
	return i.documentIndex.DeleteDocument(id)
}

func (i Indexer) newIndexedDocument(serverPath string, content string) (domain.IndexedDocument, error) {
	if len(i.documentRoots) == 0 {
		return domain.NewIndexedDocument(domain.NewDocument(serverPath, content)), nil
	}

	root, relativePath, err := i.logicalPath(serverPath)
	if err != nil {
		return domain.IndexedDocument{}, err
	}
	logicalID, err := domain.NewLogicalDocumentID(root.ID, relativePath)
	if err != nil {
		return domain.IndexedDocument{}, err
	}

	return domain.IndexedDocument{
		ID:             domain.DocumentID(logicalID),
		RootID:         root.ID,
		RelativePath:   relativePath,
		Content:        content,
		ContentNoSpace: domain.NormalizeNoSpaceContent(content),
		Path:           serverPath,
		ServerPath:     serverPath,
	}, nil
}

func (i Indexer) documentIDForPath(serverPath string) (domain.DocumentID, error) {
	if len(i.documentRoots) == 0 {
		return domain.DocumentID(serverPath), nil
	}
	root, relativePath, err := i.logicalPath(serverPath)
	if err != nil {
		return "", err
	}
	logicalID, err := domain.NewLogicalDocumentID(root.ID, relativePath)
	if err != nil {
		return "", err
	}
	return domain.DocumentID(logicalID), nil
}

func (i Indexer) logicalPath(serverPath string) (domain.DocumentRoot, domain.RelativePath, error) {
	root, rel, err := i.matchRoot(serverPath)
	if err != nil {
		return domain.DocumentRoot{}, "", err
	}
	relativePath := domain.RelativePath(filepath.ToSlash(rel))
	if err := domain.ValidateRelativePath(relativePath); err != nil {
		return domain.DocumentRoot{}, "", err
	}
	return root, relativePath, nil
}

func (i Indexer) matchRoot(serverPath string) (domain.DocumentRoot, string, error) {
	cleanPath := filepath.Clean(serverPath)
	var bestRoot domain.DocumentRoot
	var bestRel string
	bestLen := -1

	for _, root := range i.documentRoots {
		rootPath := filepath.Clean(root.ServerPath)
		rel, err := filepath.Rel(rootPath, cleanPath)
		if err != nil || rel == "." || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		if len(rootPath) > bestLen {
			bestRoot = root
			bestRel = rel
			bestLen = len(rootPath)
		}
	}
	if bestLen == -1 {
		return domain.DocumentRoot{}, "", fmt.Errorf("path %q is not under a configured document root", serverPath)
	}
	return bestRoot, bestRel, nil
}
