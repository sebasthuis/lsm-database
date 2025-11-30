package database

import (
	"fmt"
	"sync"
	"testing"
)

func TestMemTable_PutAndGet(t *testing.T) {
	m := NewMemTable()
	m.Put("apple", "red")

	val, found := m.Get("apple")
	if !found {
		t.Error("Expected key 'apple' to be found")
	}
	if val != "red" {
		t.Errorf("Expected value 'red', got %q", val)
	}
}

func TestMemTable_Update(t *testing.T) {
	m := NewMemTable()
	m.Put("apple", "red")
	m.Put("apple", "green")

	val, found := m.Get("apple")
	if !found {
		t.Error("Expected key 'apple' to be found")
	}
	if val != "green" {
		t.Errorf("Expected value 'green', got %q", val)
	}
}

func TestMemTable_GetNonExistent(t *testing.T) {
	m := NewMemTable()
	_, found := m.Get("banana")
	if found {
		t.Error("Expected key 'banana' not to be found")
	}
}

func TestMemTable_Delete(t *testing.T) {
	m := NewMemTable()
	m.Put("apple", "red")
	m.Delete("apple")

	_, found := m.Get("apple")
	if found {
		t.Error("Expected key 'apple' to be deleted")
	}
}

func TestMemTable_DeleteNonExistent(t *testing.T) {
	m := NewMemTable()
	m.Delete("ghost") // Should not panic

	_, found := m.Get("ghost")
	if found {
		t.Error("Expected key 'ghost' to remain not found")
	}
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

		gotVal, found := m.Get(key)
		if !found {
			t.Fatalf("Get(%q) not found", key)
		}
		if gotVal != wantVal {
			t.Fatalf("Get(%q) = %q, want %q", key, gotVal, wantVal)
		}
	}

	// Verify non-existence
	_, found := m.Get("key-99999")
	if found {
		t.Error("Found non-existent key")
	}
}

func TestMemTable_Concurrency(t *testing.T) {
	// Stresstest RWMutex locks
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
				got, found := m.Get(key)
				if !found || got != val {
					t.Errorf("Concurrent consistency failure: key=%s", key)
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestMemTable_ConcurrentSharedKey(t *testing.T) {
	// This tests multiple workers updating the SAME key to ensure
	// we don't corrupt the specific node or the list structure.
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
	val, found := m.Get("shared-key")
	if !found {
		t.Error("Expected 'shared-key' to exist after concurrent writes")
	}
	if len(val) == 0 {
		t.Error("Expected 'shared-key' to have a value")
	}
}

func TestMemTable_ConcurrentMixedWorkload(t *testing.T) {
	// Separate readers and writers to simulate a real database workload
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

	// 5 Readers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				key := fmt.Sprintf("key-%d", j)
				_, found := m.Get(key)
				if !found {
					// In this specific test setup, keys are never deleted,
					// so a 'not found' is a bug (race condition or logic error).
					t.Errorf("Reader failed to find key %s", key)
				}
			}
		}()
	}

	wg.Wait()
}
