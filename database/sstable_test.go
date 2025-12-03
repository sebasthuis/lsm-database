package database

import (
	"fmt"
	"path/filepath"
	"sort"
	"testing"
)

func TestSSTable_EmptyDatabase(t *testing.T) {
	tempDir := t.TempDir()
	sstPath := filepath.Join(tempDir, "empty.sst")
	sst, err := Write(sstPath, []Entry{})
	if err != nil {
		t.Fatalf("Failed to write empty SSTable: %v", err)
	}

	result, err := sst.Get("doesnotexist")
	if err != nil {
		t.Fatalf("Get returned unexpected error: %v", err)
	}

	if result.Status == Found {
		t.Errorf("Expected found=false, got true")
	}

	if result.Value != "" {
		t.Errorf("Expected empty value, got %q", result.Value)
	}
}

func TestSSTable_LargeDataset(t *testing.T) {
	tempDir := t.TempDir()
	sstPath := filepath.Join(tempDir, "large.sst")

	// Generate 1280 entries (10 blocks of 128)
	count := 1280
	inputEntries := make([]Entry, count)
	for i := 0; i < count; i++ {
		// Use padded numbers to ensure sorted order matches string order
		// "key-0000", "key-0001", ...
		key := fmt.Sprintf("key-%04d", i)
		value := fmt.Sprintf("value-%d", i)
		inputEntries[i] = Entry{Key: key, Value: value}
	}

	sst, err := Write(sstPath, inputEntries)
	if err != nil {
		t.Fatalf("Failed to write large SSTable: %v", err)
	}

	// Verify random access
	checkIndices := []int{0, 52, 127, 128, 129, 160, 280, 1279}
	for _, i := range checkIndices {
		key := inputEntries[i].Key
		wantValue := inputEntries[i].Value

		result, err := sst.Get(key)
		if err != nil {
			t.Errorf("Get(%q) failed: %v", key, err)
			continue
		}
		if result.Status == NotFound {
			t.Errorf("Get(%q) not found", key)
			continue
		}
		if result.Value != wantValue {
			t.Errorf("Get(%q) = %q, want %q", key, result.Value, wantValue)
		}
	}

	// Verify non-existent key
	result, err := sst.Get("key-9999")
	if err != nil {
		t.Errorf("Get(non-existent) failed: %v", err)
	}
	if result.Status == Found {
		t.Errorf("Get(non-existent) should return false")
	}
}

func TestSSTable_FormatValidation(t *testing.T) {
	tempDir := t.TempDir()
	sstPath := filepath.Join(tempDir, "validation.sst")

	// Assumes sorted entries
	inputEntries := []Entry{
		{Key: "apple", Value: "red"},
		{Key: "banana", Value: "yellow"},
		{Key: "emoji", Value: "🚀"},    // Unicode check
		{Key: "empty_val", Value: ""}, // Empty value check
		{Key: "grape", Value: "purple"},
		{Key: "key with spaces", Value: "value with spaces"},
		{Key: "key,comma", Value: "val,comma"}, // CSV special char check
		{Key: "key@#$%", Value: "value@#$%"},   // Special symbols
		{Key: "long_key_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Value: "long_value"},
		{Key: "12345", Value: "numeric_key"},
		{Key: "1", Value: "", Tombstone: true},
	}

	// Ensure sorted input for Write()
	sort.Slice(inputEntries, func(i, j int) bool {
		return inputEntries[i].Key < inputEntries[j].Key
	})

	sst, err := Write(sstPath, inputEntries)
	if err != nil {
		t.Fatalf("Failed to write SSTable: %v", err)
	}

	scenarios := []struct {
		name       string
		key        string
		wantValue  string
		wantStatus LookupStatus
	}{
		{
			name:       "Get existing key (first)",
			key:        "apple",
			wantValue:  "red",
			wantStatus: Found,
		},
		{
			name:       "Get existing key (middle)",
			key:        "banana",
			wantValue:  "yellow",
			wantStatus: Found,
		},
		{
			name:       "Get existing key (last)",
			key:        "grape",
			wantValue:  "purple",
			wantStatus: Found,
		},
		{
			name:       "Get unicode value",
			key:        "emoji",
			wantValue:  "🚀",
			wantStatus: Found,
		},
		{
			name:       "Get empty value",
			key:        "empty_val",
			wantValue:  "",
			wantStatus: Found,
		},
		{
			name:       "Get CSV special chars",
			key:        "key,comma",
			wantValue:  "val,comma",
			wantStatus: Found,
		},
		{
			name:       "Get non-existent key",
			key:        "orange",
			wantValue:  "",
			wantStatus: NotFound,
		},
		{
			name:       "Get deleted record (tombstone)",
			key:        "1",
			wantValue:  "",
			wantStatus: Deleted,
		},
	}

	for _, tc := range scenarios {
		t.Run(tc.name, func(t *testing.T) {
			result, err := sst.Get(tc.key)
			if err != nil {
				t.Fatalf("Get(%q) returned unexpected error: %v", tc.key, err)
			}

			if result.Status != tc.wantStatus {
				t.Errorf("Get(%q) found = %v, want %v", tc.key, result.Status, tc.wantStatus)
			}

			if result.Value != tc.wantValue {
				t.Errorf("Get(%q) value = %q, want %q", tc.key, result.Value, tc.wantValue)
			}
		})
	}
}
