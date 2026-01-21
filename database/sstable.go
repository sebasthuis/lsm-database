package database

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"sort"
)

type IndexEntry struct {
	Key    string
	Offset int64
}

type SSTable struct {
	path        string
	sparseIndex []IndexEntry
}

type Entry struct {
	Key       string
	Value     string
	Tombstone bool
}

var IndexInterval = 128 // Taken from Cassandra's default sparse index interval

// TODO: Split up into packages, the name space is getting messy with SSTable and
// SSTables. Maybe "WriteTable"
func Write(path string, sorted []Entry) (*SSTable, error) {
	file, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSTable file: %w", err)
	}

	cleanup := func() {
		file.Close()
		os.Remove(path)
	}

	writer := csv.NewWriter(file)
	sparseIndex := make([]IndexEntry, 0)
	offset := int64(0)

	for i, entry := range sorted {
		if i%IndexInterval == 0 {
			sparseIndex = append(sparseIndex, IndexEntry{
				Key:    entry.Key,
				Offset: offset,
			})
		}

		tombstone := "0"
		if entry.Tombstone {
			tombstone = "1"
		}

		if err := writer.Write([]string{entry.Key, entry.Value, tombstone}); err != nil {
			cleanup()
			return nil, fmt.Errorf("failed to write entry: %w", err)
		}

		// Flush to get accurate offset
		writer.Flush()

		newOffset, err := file.Seek(0, io.SeekCurrent)
		if err != nil {
			cleanup()
			return nil, err
		}

		offset = newOffset
	}

	writer.Flush()
	if err := file.Close(); err != nil {
		os.Remove(path)
		return nil, fmt.Errorf("failed to close SSTable file: %w", err)
	}

	return &SSTable{
		path:        path,
		sparseIndex: sparseIndex,
	}, nil
}

func (sst *SSTable) Get(key string) (*LookupResult, error) {
	offset := sst.findOffset(key)

	file, err := os.Open(sst.path)
	if err != nil {
		return nil, fmt.Errorf("failed to open SSTable: %w", err)
	}
	defer file.Close()

	if _, err := file.Seek(offset, 0); err != nil {
		return nil, fmt.Errorf("failed to seek SSTable: %w", err)
	}

	reader := csv.NewReader(file)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			return &LookupResult{Value: "", Status: NotFound}, nil
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read record: %w", err)
		}

		if len(record) != 3 {
			return nil, fmt.Errorf("malformed CSV record")
		}

		entryKey := record[0]
		entryValue := record[1]
		entryTombstone := record[2] == "1"

		if entryKey == key {
			if entryTombstone {
				return &LookupResult{Value: entryValue, Status: Deleted}, nil
			} else {
				return &LookupResult{Value: entryValue, Status: Found}, nil
			}
		}

		// Early termination: file is sorted, if we've passed the key, it doesn't exist
		if entryKey > key {
			return &LookupResult{Value: "", Status: NotFound}, nil
		}
	}
	// Unreachable?
}

func (sst *SSTable) findOffset(key string) int64 {
	position := sort.Search(len(sst.sparseIndex), func(i int) bool {
		// Easier to to identify unfound key by looking for keys greater than
		// the target key
		return sst.sparseIndex[i].Key > key
	})

	if position == 0 {
		return 0
	} else {
		return sst.sparseIndex[position-1].Offset
	}
}
