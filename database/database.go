package database

import (
	"errors"
	"sync"
)

// 4MB is a common default (used by LevelDB).
const FlushThreshold = 4 * 1024 * 1024

var ErrKeyNotFound = errors.New("key not found")

type Database struct {
	mutable *MemTable
	// Snapshot before table gets flushed to disk
	immutable *MemTable
	disk      *SSTables
	// lock protects the memTable pointer for concurrent access and swapping
	lock sync.RWMutex
}

func Create(directory string) (*Database, error) {
	return &Database{
		mutable: NewMemTable(),
		disk:    NewSSTables(directory),
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
	db.lock.RUnlock() // Only needs to guard

	value, found, err := db.disk.Get(key)
	if err != nil {
		return "", err
	} else if found {
		return value, nil
	} else {
		return "", ErrKeyNotFound
	}
}

func (db *Database) Delete(key string) error {
	db.lock.RLock()
	defer db.lock.RUnlock()

	db.mutable.Delete(key)
	return nil
}

func (db *Database) flush() error {
	// db.lock.Lock()

	// // Double-check: Another thread might have flushed while we were waiting for the lock
	// if db.mutable.size < FlushThreshold {
	// 	db.lock.Unlock()
	// 	return nil
	// }

	// db.immutable = db.mutable
	// db.mutable = NewMemTable()

	// db.lock.Unlock()

	// entries := db.immutable.All()
	// // TODO: Needs to be moved into separate function inside the SSTables
	// filename := fmt.Sprintf("sstable-%d.csv", time.Now().UnixNano())
	// path := filepath.Join(db.directory, filename)

	// sstable, err := Write(path, entries)
	// if err != nil {
	// 	return fmt.Errorf("failed to write SSTable: %w", err)
	// }

	// db.lock.Lock()
	// db.sstables = append(db.sstables, sstable)
	// db.immutable = nil
	// db.lock.Unlock()

	return nil
}
