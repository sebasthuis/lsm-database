// Goal: Implement meta abstraction that
// - Schedules compaction
// - (future) can reconstruct after a restart based on dictionary

package database

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"
)

type SSTables struct {
	tables    []*SSTable
	directory string
	lock      sync.RWMutex
}

func NewSSTables(directory string) *SSTables {
	return &SSTables{
		directory: directory,
	}
}

func (t *SSTables) Flush(sorted []Entry) error {
	path := filepath.Join(t.directory, generateFileName())
	table, err := Write(path, sorted)
	if err != nil {
		return err
	}
	t.lock.Lock()
	defer t.lock.Unlock()
	t.tables = append(t.tables, table)
	return nil
	// TODO: Schedule compaction, but lock should be released as they
	// are taking a new lock.
}

func (t *SSTables) Lookup(key string) (string, bool, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	for i := len(t.tables) - 1; i >= 0; i-- {
		result, err := t.tables[i].Lookup(key)
		if err != nil {
			return "", false, fmt.Errorf("error reading sstable: %w", err)
		}

		switch result.Status {
		case Found:
			return result.Value, true, nil
		case Deleted:
			return "", false, nil
		}
	}
	return "", false, nil
}

// TODO: Implement more robust order than time based. In the future implement
// consensus across nodes.
func generateFileName() string {
	return fmt.Sprintf("sstable-%d.csv", time.Now().UnixNano())
}
