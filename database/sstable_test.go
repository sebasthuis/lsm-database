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

	value, found, err := sst.Get("doesnotexist")
	if err != nil {
		t.Fatalf("Get returned unexpected error: %v", err)
	}

	if found {
		t.Errorf("Expected found=false, got true")
	}

	if value != "" {
		t.Errorf("Expected empty value, got %q", value)
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

		gotValue, found, err := sst.Get(key)
		if err != nil {
			t.Errorf("Get(%q) failed: %v", key, err)
			continue
		}
		if !found {
			t.Errorf("Get(%q) not found", key)
			continue
		}
		if gotValue != wantValue {
			t.Errorf("Get(%q) = %q, want %q", key, gotValue, wantValue)
		}
	}

	// Verify non-existent key
	_, found, err := sst.Get("key-9999")
	if err != nil {
		t.Errorf("Get(non-existent) failed: %v", err)
	}
	if found {
		t.Errorf("Get(non-existent) should return false")
	}
}

func TestSSTable_KeyValidation(t *testing.T) {
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
		name      string
		key       string
		wantValue string
		wantFound bool
	}{
		{
			name:      "Get existing key (first)",
			key:       "apple",
			wantValue: "red",
			wantFound: true,
		},
		{
			name:      "Get existing key (middle)",
			key:       "banana",
			wantValue: "yellow",
			wantFound: true,
		},
		{
			name:      "Get existing key (last)",
			key:       "grape",
			wantValue: "purple",
			wantFound: true,
		},
		{
			name:      "Get unicode value",
			key:       "emoji",
			wantValue: "🚀",
			wantFound: true,
		},
		{
			name:      "Get empty value",
			key:       "empty_val",
			wantValue: "",
			wantFound: true,
		},
		{
			name:      "Get CSV special chars",
			key:       "key,comma",
			wantValue: "val,comma",
			wantFound: true,
		},
		{
			name:      "Get non-existent key",
			key:       "orange",
			wantValue: "",
			wantFound: false,
		},
	}

	for _, tc := range scenarios {
		t.Run(tc.name, func(t *testing.T) {
			gotValue, gotFound, err := sst.Get(tc.key)
			if err != nil {
				t.Fatalf("Get(%q) returned unexpected error: %v", tc.key, err)
			}

			if gotFound != tc.wantFound {
				t.Errorf("Get(%q) found = %v, want %v", tc.key, gotFound, tc.wantFound)
			}

			if gotValue != tc.wantValue {
				t.Errorf("Get(%q) value = %q, want %q", tc.key, gotValue, tc.wantValue)
			}
		})
	}
}
