package search

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/blevesearch/bleve/v2"
)

type indexStore struct {
	indexPath string
	index     bleve.Index
	mu        sync.RWMutex
}

func newIndexStore(indexPath string) *indexStore {
	return &indexStore{indexPath: indexPath}
}

func (s *indexStore) open() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("Initializing index at %s", s.indexPath)
	index, err := s.openOrCreate()
	if err != nil {
		return fmt.Errorf("failed to open/create index: %w", err)
	}
	if index == nil {
		return fmt.Errorf("bleve returned nil index with no error")
	}

	s.index = index
	return nil
}

func (s *indexStore) indexDocument(id string, document any) error {
	if s == nil {
		return fmt.Errorf("index is closed")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.index == nil {
		return fmt.Errorf("index is closed")
	}

	return s.index.Index(id, document)
}

func (s *indexStore) search(req *bleve.SearchRequest) (*bleve.SearchResult, error) {
	if s == nil {
		return nil, fmt.Errorf("index is closed")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.index == nil {
		return nil, fmt.Errorf("index is closed")
	}

	return s.index.Search(req)
}

func (s *indexStore) count() (uint64, error) {
	if s == nil {
		return 0, fmt.Errorf("index is closed")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.index == nil {
		return 0, fmt.Errorf("index is closed")
	}

	return s.index.DocCount()
}

func (s *indexStore) deleteDocument(id string) error {
	if s == nil {
		return fmt.Errorf("index is closed")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.index == nil {
		return fmt.Errorf("index is closed")
	}

	return s.index.Delete(id)
}

func (s *indexStore) reset() error {
	if s == nil {
		return fmt.Errorf("index is closed")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.index == nil {
		return fmt.Errorf("index is closed")
	}

	if err := s.index.Close(); err != nil {
		return err
	}
	s.index = nil

	newIndex, err := s.recreate()
	if err != nil {
		return err
	}
	s.index = newIndex
	return nil
}

func (s *indexStore) openOrCreate() (bleve.Index, error) {
	if _, statErr := os.Stat(s.indexPath); os.IsNotExist(statErr) {
		log.Println("Index does not exist, creating new...")
		return s.create()
	}

	log.Println("Index exists, opening...")
	index, err := bleve.Open(s.indexPath)
	if err == nil {
		return index, nil
	}

	return nil, fmt.Errorf("open existing index %s: %w", s.indexPath, err)
}

func (s *indexStore) recreate() (bleve.Index, error) {
	if _, err := normalizeIndexPath(s.indexPath); err != nil {
		return nil, err
	}
	if err := os.RemoveAll(s.indexPath); err != nil {
		return nil, fmt.Errorf("failed to remove corrupted index: %w", err)
	}
	return s.create()
}

func (s *indexStore) create() (bleve.Index, error) {
	indexMapping, err := buildIndexMapping()
	if err != nil {
		return nil, err
	}
	return bleve.New(s.indexPath, indexMapping)
}

func (s *indexStore) close() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.index == nil {
		return nil
	}

	err := s.index.Close()
	s.index = nil
	return err
}
