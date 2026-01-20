package database

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"
)

type SSTables struct {
	directory string
	tables    []*SSTable // Most recent at end
	lock      sync.RWMutex
}

func NewSSTables(directory string) *SSTables {
	return &SSTables{directory: directory, tables: make([]*SSTable, 0)}
}

func (s *SSTables) Get(key string) (*LookupResult, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	for i := len(s.tables) - 1; i >= 0; i-- {
		table := s.tables[i]
		result, err := table.Get(key)

		if err != nil {
			return nil, err
		}

		if result.Status != NotFound {
			// Either deleted (tombstone) or found
			return result, nil
		}
	}

	return &LookupResult{Status: NotFound, Value: ""}, nil
}

// TODO: Rename and package differently, so that namespace isn't too messy. This
// write is thankfully only called on the SSTables abstraction.
func (s *SSTables) Write(sorted []Entry) error {
	// 1. Write to disk without holding the lock
	// This ensures that reads (Get) are not blocked by slow I/O
	table, err := Write(s.createFilePath(), sorted)
	if err != nil {
		return err
	}

	// 2. Lock only for the pointer memory operation
	s.lock.Lock()
	defer s.lock.Unlock()

	s.tables = append(s.tables, table)
	return nil
}

func (s *SSTables) createFilePath() string {
	return filepath.Join(s.directory, createFileName())
}

func createFileName() string {
	// TODO: Change to an increasing number instead of time
	return fmt.Sprintf("sstable-%d.csv", time.Now().UnixNano())
}
