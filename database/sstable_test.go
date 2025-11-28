package database

import (
	"path/filepath"
	"testing"
)


func TestSSTable_RoundTrip(t *testing.T) {
	tempDir := t.TempDir()
	sstPath := filepath.Join(tempDir, "test.sst")

	// Assumes sorted entries
	inputEntries := []Entry{
		{Key: "apple", Value: "red"},
		{Key: "banana", Value: "yellow"},
		{Key: "emoji", Value: "🚀"},            // Unicode check
		{Key: "empty_val", Value: ""},          // Empty value check
		{Key: "grape", Value: "purple"},
		{Key: "key,comma", Value: "val,comma"}, // CSV special char check
	}

	sst, err := Write(sstPath, inputEntries)
	if err != nil {
		t.Fatalf("Failed to write SSTable: %v", err)
	}

	// 3. Verification: Define scenarios to test against the written file
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
