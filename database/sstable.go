package database

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
)

type SSTable struct {
	path  string
	index map[string]int64 // key -> byte offset in file
}

type Entry struct {
	Key   string
	Value string
}

func Write(path string, sorted []Entry) (*SSTable, error) {
	file, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSTable file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	index := make(map[string]int64)
	offset := int64(0)

	for _, entry := range sorted {
		// TODO: Change to sparse index
		index[entry.Key] = offset

		if err := writer.Write([]string{entry.Key, entry.Value}); err != nil {
			return nil, fmt.Errorf("failed to write entry: %w", err)
		}

		// Flush to get accurate offset
		writer.Flush()

		newOffset, err := file.Seek(0, io.SeekCurrent)
		if err != nil {
			return nil, err
		}

		offset = newOffset
	}

	return &SSTable{
		path:  path,
		index: index,
	}, nil
}

func (sst *SSTable) Get(key string) (string, bool, error) {
	// TODO: Implement Sparse index
	offset, exists := sst.index[key]
	if !exists {
		return "", false, nil
	}

	file, err := os.Open(sst.path)
	if err != nil {
		return "", false, fmt.Errorf("failed to open SSTable: %w", err)
	}
	defer file.Close()

	if _, err := file.Seek(offset, 0); err != nil {
		return "", false, fmt.Errorf("failed to seek SSTable: %w", err)
	}

	reader := csv.NewReader(file)
	record, err := reader.Read()
	if err != nil {
		return "", false, fmt.Errorf("failed to read record: %w", err)
	}

	if len(record) != 2 {
		return "", false, fmt.Errorf("CSV record malformed")
	}
	return record[1], true, nil
}