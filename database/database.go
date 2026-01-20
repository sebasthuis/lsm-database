package database

import (
	"errors"
	"fmt"
	"sync"
)

// 4MB is a common default (used by LevelDB).
const FlushThreshold = 4 * 1024 * 1024

var ErrKeyNotFound = errors.New("key not found")

type Database struct {
	// TODO: Change naming for mutable and immutable. I think it should be clearer
	// that one of them is the active memtable and the other is a temporary
	// placeholder.
	mutable *MemTable
	// Snapshot before table gets flushed to disk
	immutable *MemTable
	sstables  *SSTables
	// lock protects the memTable pointer for concurrent access and swapping
	// TODO: Add here that it is just used for swapping the memtables.
	lock sync.RWMutex
}

func Create(directory string) (*Database, error) {
	return &Database{
		mutable:  NewMemTable(),
		sstables: NewSSTables(directory),
	}, nil
}

func (db *Database) Put(key, value string) error {
	db.lock.RLock()
	currentSize := db.mutable.Put(key, value)
	db.lock.RUnlock()

	if currentSize >= FlushThreshold {
		return db.flush()
	}

	return nil
}

func (db *Database) Get(key string) (string, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	result := db.mutable.Get(key)
	if result.Status == Found {
		return result.Value, nil
	}

	if db.immutable != nil {
		result := db.immutable.Get(key)
		if result.Status == Found {
			return result.Value, nil
		}
	}

	// TODO: If I would like the error to bubble up, how would I do that here?
	tableResult, err := db.sstables.Get(key)
	if err != nil {
		return "", err
	}

	if tableResult.Status == Found {
		return tableResult.Value, nil
	}

	return "", ErrKeyNotFound
}

func (db *Database) Delete(key string) error {
	db.lock.RLock()
	defer db.lock.RUnlock()

	db.mutable.Delete(key)
	return nil
}

func (db *Database) flush() error {
	db.lock.Lock()

	// Double-check: Another thread might have flushed while we were waiting for the lock
	if db.mutable.size < FlushThreshold {
		db.lock.Unlock()
		return nil
	}

	db.immutable = db.mutable
	db.mutable = NewMemTable()

	db.lock.Unlock()

	entries := db.immutable.All()

	err := db.sstables.Write(entries)
	if err != nil {
		return fmt.Errorf("failed to write SSTable: %w", err)
	}

	db.lock.Lock()
	db.immutable = nil
	db.lock.Unlock()

	return nil
}
