package database

import (
	"fmt"
	"sync"
	"testing"
)

func TestMemTable_BasicOperations(t *testing.T) {
	// Shared state for the lifecycle test
	m := NewMemTable()

	t.Run("1. Put and Get new key", func(t *testing.T) {
		m.Put("apple", "red")

		result := m.Get("apple")
		if result.Status != Found {
			t.Error("Expected key 'apple' to be found")
		}
		if result.Value != "red" {
			t.Errorf("Expected value 'red', got %q", result.Value)
		}
	})

	t.Run("2. Update existing key", func(t *testing.T) {
		m.Put("apple", "green")

		result := m.Get("apple")
		if result.Status != Found {
			t.Error("Expected key 'apple' to be found")
		}
		if result.Value != "green" {
			t.Errorf("Expected value 'green', got %q", result.Value)
		}
	})

	t.Run("3. Get non-existent key", func(t *testing.T) {
		result := m.Get("banana")
		if result.Status != NotFound {
			t.Error("Expected key 'banana' not to be found")
		}
	})

	t.Run("4. Delete key", func(t *testing.T) {
		m.Delete("apple")

		result := m.Get("apple")
		if result.Status != Deleted {
			t.Errorf("Expected key 'apple' to be Deleted, got %v", result.Status)
		}
	})

	t.Run("5. Delete non-existent key", func(t *testing.T) {
		m.Delete("ghost") // Should not panic

		result := m.Get("ghost")
		if result.Status != NotFound {
			t.Errorf("Expected key 'ghost' to remain NotFound, got %v", result.Status)
		}
	})
}

func TestMemTable_LargeDataset(t *testing.T) {
	// This tests the Skip List structure's ability to handle many items
	// and ensures the probabilistic balancing doesn't break links.
	m := NewMemTable()
	count := 10000

	// Insert
	for i := 0; i < count; i++ {
		key := fmt.Sprintf("key-%05d", i)
		val := fmt.Sprintf("val-%05d", i)
		m.Put(key, val)
	}

	// Verify existence
	for i := 0; i < count; i++ {
		key := fmt.Sprintf("key-%05d", i)
		wantVal := fmt.Sprintf("val-%05d", i)

		result := m.Get(key)
		if result.Status != Found {
			t.Fatalf("Get(%q) not found", key)
		}
		if result.Value != wantVal {
			t.Fatalf("Get(%q) = %q, want %q", key, result.Value, wantVal)
		}
	}

	// Verify non-existence
	result := m.Get("key-99999")
	if result.Status != NotFound {
		t.Error("Found non-existent key")
	}
}

func TestMemTable_Concurrency(t *testing.T) {
	t.Run("ReadWrite", func(t *testing.T) {
		m := NewMemTable()
		var wg sync.WaitGroup
		workers := 10
		opsPerWorker := 100

		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				for j := 0; j < opsPerWorker; j++ {
					key := fmt.Sprintf("k-%d-%d", workerID, j)
					val := fmt.Sprintf("v-%d-%d", workerID, j)

					m.Put(key, val)

					// Immediate read-back check
					result := m.Get(key)
					if result.Status != Found || result.Value != val {
						t.Errorf("Concurrent consistency failure: key=%s", key)
					}
				}
			}(i)
		}
		wg.Wait()
	})

	t.Run("SharedKey", func(t *testing.T) {
		m := NewMemTable()
		var wg sync.WaitGroup
		workers := 10
		opsPerWorker := 100

		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < opsPerWorker; j++ {
					// Everyone writes to "shared-key"
					val := fmt.Sprintf("val-%d-%d", id, j)
					m.Put("shared-key", val)
				}
			}(i)
		}
		wg.Wait()

		// Final check: The key should exist and have a valid value
		result := m.Get("shared-key")
		if result.Status != Found {
			t.Error("Expected 'shared-key' to exist after concurrent writes")
		}
		if len(result.Value) == 0 {
			t.Error("Expected 'shared-key' to have a value")
		}
	})

	t.Run("MixedWorkload", func(t *testing.T) {
		m := NewMemTable()
		var wg sync.WaitGroup

		// Pre-populate
		for i := 0; i < 100; i++ {
			m.Put(fmt.Sprintf("key-%d", i), "initial")
		}

		// 5 Writers
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					key := fmt.Sprintf("key-%d", j) // Overwrite existing keys
					m.Put(key, fmt.Sprintf("writer-%d-val-%d", id, j))
				}
			}(i)
		}

		// 2 Deleters
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					if j%2 == 0 { // Delete even keys
						key := fmt.Sprintf("key-%d", j)
						m.Delete(key)
					}
				}
			}(i)
		}

		// 5 Readers
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					key := fmt.Sprintf("key-%d", j)
					result := m.Get(key)

					if result.Status == NotFound {
						t.Errorf("Reader failed to find key %s (got NotFound, expected Found or Deleted)", key)
					}
				}
			}()
		}
		wg.Wait()
	})
}
