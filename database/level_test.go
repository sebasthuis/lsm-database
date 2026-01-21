package database

import (
	"fmt"
	"sort"
	"sync"
	"testing"
)

func TestLevel_NewLevel(t *testing.T) {
	tempDir := t.TempDir()
	tables := NewLevel(tempDir)

	if tables == nil {
		t.Fatal("NewLevel returned nil")
	}

	if tables.directory != tempDir {
		t.Errorf("directory = %q, want %q", tables.directory, tempDir)
	}

	if len(tables.tables) != 0 {
		t.Errorf("tables length = %d, want 0", len(tables.tables))
	}
}

func TestLevel_GetFromEmpty(t *testing.T) {
	tempDir := t.TempDir()
	tables := NewLevel(tempDir)

	result, err := tables.Get("anykey")
	if err != nil {
		t.Fatalf("Get returned unexpected error: %v", err)
	}

	if result.Status != NotFound {
		t.Errorf("Status = %v, want NotFound", result.Status)
	}

	if result.Value != "" {
		t.Errorf("Value = %q, want empty string", result.Value)
	}
}

func TestLevel_WriteAndGet(t *testing.T) {
	tempDir := t.TempDir()
	tables := NewLevel(tempDir)

	entries := []Entry{
		{Key: "apple", Value: "red"},
		{Key: "banana", Value: "yellow"},
		{Key: "cherry", Value: "red"},
	}

	err := tables.Write(entries)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	scenarios := []struct {
		key        string
		wantValue  string
		wantStatus LookupStatus
	}{
		{"apple", "red", Found},
		{"banana", "yellow", Found},
		{"cherry", "red", Found},
		{"grape", "", NotFound},
	}

	for _, tc := range scenarios {
		t.Run(tc.key, func(t *testing.T) {
			result, err := tables.Get(tc.key)
			if err != nil {
				t.Fatalf("Get(%q) error: %v", tc.key, err)
			}

			if result.Status != tc.wantStatus {
				t.Errorf("Get(%q) status = %v, want %v", tc.key, result.Status, tc.wantStatus)
			}

			if result.Value != tc.wantValue {
				t.Errorf("Get(%q) value = %q, want %q", tc.key, result.Value, tc.wantValue)
			}
		})
	}
}

func TestLevel_NewerTableTakesPrecedence(t *testing.T) {
	tempDir := t.TempDir()
	tables := NewLevel(tempDir)

	// Write first table with initial values
	err := tables.Write([]Entry{
		{Key: "apple", Value: "green"},
		{Key: "banana", Value: "green"},
	})
	if err != nil {
		t.Fatalf("First Write failed: %v", err)
	}

	// Write second table with updated value for apple
	err = tables.Write([]Entry{
		{Key: "apple", Value: "red"},
	})
	if err != nil {
		t.Fatalf("Second Write failed: %v", err)
	}

	// apple should return the newer value
	result, err := tables.Get("apple")
	if err != nil {
		t.Fatalf("Get(apple) error: %v", err)
	}
	if result.Status != Found {
		t.Errorf("Get(apple) status = %v, want Found", result.Status)
	}
	if result.Value != "red" {
		t.Errorf("Get(apple) value = %q, want %q", result.Value, "red")
	}

	// banana should still return from older table
	result, err = tables.Get("banana")
	if err != nil {
		t.Fatalf("Get(banana) error: %v", err)
	}
	if result.Status != Found {
		t.Errorf("Get(banana) status = %v, want Found", result.Status)
	}
	if result.Value != "green" {
		t.Errorf("Get(banana) value = %q, want %q", result.Value, "green")
	}
}

func TestLevel_TombstoneShadowsOlderValue(t *testing.T) {
	tempDir := t.TempDir()
	tables := NewLevel(tempDir)

	// Write first table with value
	err := tables.Write([]Entry{
		{Key: "apple", Value: "red"},
	})
	if err != nil {
		t.Fatalf("First Write failed: %v", err)
	}

	// Verify key exists
	result, err := tables.Get("apple")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if result.Status != Found {
		t.Errorf("Expected apple to be Found before deletion")
	}

	// Write tombstone in newer table
	err = tables.Write([]Entry{
		{Key: "apple", Value: "", Tombstone: true},
	})
	if err != nil {
		t.Fatalf("Second Write failed: %v", err)
	}

	// Should return Deleted, not Found
	result, err = tables.Get("apple")
	if err != nil {
		t.Fatalf("Get error after tombstone: %v", err)
	}
	if result.Status != Deleted {
		t.Errorf("Get(apple) status = %v, want Deleted", result.Status)
	}
}

func TestLevel_MultipleTablesSearchOrder(t *testing.T) {
	tempDir := t.TempDir()
	tables := NewLevel(tempDir)

	// Create 5 tables, each with a unique key and one shared key
	for i := 0; i < 5; i++ {
		entries := []Entry{
			{Key: fmt.Sprintf("unique-%d", i), Value: fmt.Sprintf("value-%d", i)},
			{Key: "shared", Value: fmt.Sprintf("version-%d", i)},
		}
		// Sort entries - SSTable requires sorted input
		sort.Slice(entries, func(a, b int) bool {
			return entries[a].Key < entries[b].Key
		})
		err := tables.Write(entries)
		if err != nil {
			t.Fatalf("Write %d failed: %v", i, err)
		}
	}

	// All unique keys should be found
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("unique-%d", i)
		result, err := tables.Get(key)
		if err != nil {
			t.Errorf("Get(%q) error: %v", key, err)
			continue
		}
		if result.Status != Found {
			t.Errorf("Get(%q) status = %v, want Found", key, result.Status)
		}
		wantValue := fmt.Sprintf("value-%d", i)
		if result.Value != wantValue {
			t.Errorf("Get(%q) value = %q, want %q", key, result.Value, wantValue)
		}
	}

	// Shared key should return most recent value (from table 4)
	result, err := tables.Get("shared")
	if err != nil {
		t.Fatalf("Get(shared) error: %v", err)
	}
	if result.Status != Found {
		t.Errorf("Get(shared) status = %v, want Found", result.Status)
	}
	if result.Value != "version-4" {
		t.Errorf("Get(shared) value = %q, want %q", result.Value, "version-4")
	}
}

func TestLevel_ConcurrentReads(t *testing.T) {
	tempDir := t.TempDir()
	tables := NewLevel(tempDir)

	// Write some data
	entries := make([]Entry, 100)
	for i := 0; i < 100; i++ {
		entries[i] = Entry{
			Key:   fmt.Sprintf("key-%03d", i),
			Value: fmt.Sprintf("value-%d", i),
		}
	}
	err := tables.Write(entries)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Perform concurrent reads
	var wg sync.WaitGroup
	errors := make(chan error, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%03d", idx)
			wantValue := fmt.Sprintf("value-%d", idx)

			result, err := tables.Get(key)
			if err != nil {
				errors <- fmt.Errorf("Get(%q) error: %v", key, err)
				return
			}
			if result.Status != Found {
				errors <- fmt.Errorf("Get(%q) not found", key)
				return
			}
			if result.Value != wantValue {
				errors <- fmt.Errorf("Get(%q) = %q, want %q", key, result.Value, wantValue)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}

func TestLevel_ConcurrentReadsAndWrites(t *testing.T) {
	tempDir := t.TempDir()
	tables := NewLevel(tempDir)

	// Pre-populate with initial data
	initialEntries := []Entry{
		{Key: "stable", Value: "initial"},
	}
	err := tables.Write(initialEntries)
	if err != nil {
		t.Fatalf("Initial Write failed: %v", err)
	}

	var wg sync.WaitGroup
	errors := make(chan error, 200)

	// Concurrent readers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := tables.Get("stable")
			if err != nil {
				errors <- fmt.Errorf("Get error: %v", err)
				return
			}
			if result.Status != Found {
				errors <- fmt.Errorf("Get(stable) not found")
			}
		}()
	}

	// Concurrent writers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			entries := []Entry{
				{Key: fmt.Sprintf("concurrent-%d", idx), Value: fmt.Sprintf("value-%d", idx)},
			}
			if err := tables.Write(entries); err != nil {
				errors <- fmt.Errorf("Write error: %v", err)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}

func TestLevel_WriteCreatesFile(t *testing.T) {
	tempDir := t.TempDir()
	tables := NewLevel(tempDir)

	entries := []Entry{
		{Key: "key", Value: "value"},
	}

	err := tables.Write(entries)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if len(tables.tables) != 1 {
		t.Errorf("tables count = %d, want 1", len(tables.tables))
	}

	// Verify the table has a valid path
	if tables.tables[0].path == "" {
		t.Error("SSTable path is empty")
	}
}