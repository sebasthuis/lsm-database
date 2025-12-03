package database

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestDatabase_BasicOperations(t *testing.T) {
	dir, err := os.MkdirTemp("", "test-operations")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	db, err := Create(dir)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	t.Run("Put new key", func(t *testing.T) {
		if err := db.Put("key1", "value1"); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
	})

	t.Run("Get key", func(t *testing.T) {
		val, err := db.Get("key1")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if val != "value1" {
			t.Errorf("Expected 'value1', got '%s'", val)
		}
	})

	t.Run("Update key", func(t *testing.T) {
		if err := db.Put("key1", "value2"); err != nil {
			t.Fatalf("Put update failed: %v", err)
		}

		val, err := db.Get("key1")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if val != "value2" {
			t.Errorf("Expected 'value2', got '%s'", val)
		}
	})

	t.Run("Delete key", func(t *testing.T) {
		if err := db.Delete("key1"); err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
		_, err = db.Get("key1")
		if err != ErrKeyNotFound {
			t.Errorf("Expected ErrKeyNotFound, got %v", err)
		}
	})
}

// TODO: Investigate whether I could do some invariant testing here
func TestDatabase_LargeDataset(t *testing.T) {
	dir, err := os.MkdirTemp("", "test-flush")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	db, err := Create(dir)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	// The threshold is 4MB. We need to write enough data to trigger a flush.
	// Each entry: Key(10) + Value(100) + Overhead(56) + Pointers(avg 16?) ~= 180 bytes.
	// 4,194,304 / 180 ~= 23,000 entries.

	entryCount := 200000
	value := strings.Repeat("x", 100) // 100 bytes value

	t.Logf("Writing %d entries to trigger flush...", entryCount)
	for i := 0; i < entryCount; i++ {
		key := fmt.Sprintf("key-%06d", i)
		if err := db.Put(key, value); err != nil {
			t.Fatalf("Put failed at index %d: %v", i, err)
		}
	}

	earlyKey := "key-000000"
	val, err := db.Get(earlyKey)
	if err != nil {
		t.Errorf("Failed to get flushed key %s: %v", earlyKey, err)
	}
	if val != value {
		t.Errorf("Value mismatch for %s", earlyKey)
	}

	lateKey := fmt.Sprintf("key-%06d", entryCount-1)
	val, err = db.Get(lateKey)
	if err != nil {
		t.Errorf("Failed to get memtable key %s: %v", lateKey, err)
	}
	if val != value {
		t.Errorf("Value mismatch for %s", lateKey)
	}
}
