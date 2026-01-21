package database

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"
)

type Level struct {
	directory string
	tables    []*SSTable   // Stack; Most recent appended at the end
	lock      sync.RWMutex // Lock for `tables` pointer
}

func NewLevel(directory string) *Level {
	return &Level{directory: directory, tables: make([]*SSTable, 0)}
}

func (s *Level) Get(key string) (*LookupResult, error) {
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

func (s *Level) Write(snapshot []Entry) error {
	path := s.createFilePath()
	table, err := Write(path, snapshot)
	if err != nil {
		return err
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	s.tables = append(s.tables, table)
	return nil
}

func (s *Level) createFilePath() string {
	return filepath.Join(s.directory, createFileName())
}

func createFileName() string {
	// TODO: Change to an increasing number instead of time
	return fmt.Sprintf("sstable-%d.csv", time.Now().UnixNano())
}
