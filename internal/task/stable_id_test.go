package task

import (
	"strconv"
	"strings"
	"testing"
)

func TestNewStableIDGenerator(t *testing.T) {
	tests := map[string]struct {
		existingIDs []string
		description string
	}{
		"empty_existing_ids": {
			existingIDs: []string{},
			description: "should create generator with empty existing IDs",
		},
		"with_existing_ids": {
			existingIDs: []string{"0000001", "0000002", "0000003"},
			description: "should create generator with existing IDs",
		},
		"with_high_value_ids": {
			existingIDs: []string{"000000z", "00000zz", "0000zzz"},
			description: "should handle high base36 values",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			gen := NewStableIDGenerator(tc.existingIDs)
			if gen == nil {
				t.Fatal("NewStableIDGenerator returned nil")
			}

			// Verify all existing IDs are marked as used
			for _, id := range tc.existingIDs {
				if !gen.IsUsed(id) {
					t.Errorf("existing ID %q should be marked as used", id)
				}
			}
		})
	}
}

func TestStableIDGenerator_Generate(t *testing.T) {
	tests := map[string]struct {
		existingIDs []string
		description string
	}{
		"first_id_generation": {
			existingIDs: []string{},
			description: "generates valid ID from empty state",
		},
		"continues_from_existing": {
			existingIDs: []string{"0000001", "0000002"},
			description: "continues counter from existing IDs",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			gen := NewStableIDGenerator(tc.existingIDs)
			id, err := gen.Generate()
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			// Verify ID format: 7 lowercase alphanumeric characters
			if len(id) != 7 {
				t.Errorf("Generate() = %q, want 7 characters, got %d", id, len(id))
			}

			// Verify lowercase alphanumeric only (base36)
			for _, c := range id {
				if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'z')) {
					t.Errorf("Generate() = %q contains invalid character %q", id, c)
				}
			}

			// Verify the ID is marked as used
			if !gen.IsUsed(id) {
				t.Errorf("generated ID %q should be marked as used", id)
			}
		})
	}
}

func TestStableIDGenerator_SevenCharLength(t *testing.T) {
	gen := NewStableIDGenerator([]string{})

	// Generate multiple IDs and verify all are 7 characters
	for i := 0; i < 100; i++ {
		id, err := gen.Generate()
		if err != nil {
			t.Fatalf("Generate() iteration %d error = %v", i, err)
		}
		if len(id) != 7 {
			t.Errorf("Generate() iteration %d = %q, want 7 characters", i, id)
		}
	}
}

func TestStableIDGenerator_Base36Encoding(t *testing.T) {
	tests := map[string]struct {
		existingIDs []string
		wantNext    string
		description string
	}{
		"increments_from_0000001": {
			existingIDs: []string{"0000001"},
			wantNext:    "0000002",
			description: "should increment to 2",
		},
		"increments_from_0000009": {
			existingIDs: []string{"0000009"},
			wantNext:    "000000a",
			description: "should increment to a (base36)",
		},
		"increments_from_000000z": {
			existingIDs: []string{"000000z"},
			wantNext:    "0000010",
			description: "should roll over to 10 (base36)",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			gen := NewStableIDGenerator(tc.existingIDs)
			id, err := gen.Generate()
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			if id != tc.wantNext {
				t.Errorf("Generate() = %q, want %q", id, tc.wantNext)
			}
		})
	}
}

func TestStableIDGenerator_CollisionDetection(t *testing.T) {
	// Create generator with some existing IDs
	existingIDs := []string{"0000001", "0000002", "0000003"}
	gen := NewStableIDGenerator(existingIDs)

	// Generate new IDs and verify no collisions
	generatedIDs := make(map[string]bool)
	for _, id := range existingIDs {
		generatedIDs[id] = true
	}

	for i := 0; i < 100; i++ {
		id, err := gen.Generate()
		if err != nil {
			t.Fatalf("Generate() iteration %d error = %v", i, err)
		}
		if generatedIDs[id] {
			t.Errorf("Generate() produced duplicate ID %q at iteration %d", id, i)
		}
		generatedIDs[id] = true
	}
}

func TestStableIDGenerator_Uniqueness10000(t *testing.T) {
	gen := NewStableIDGenerator([]string{})
	seen := make(map[string]bool)

	for i := 0; i < 10000; i++ {
		id, err := gen.Generate()
		if err != nil {
			t.Fatalf("Generate() iteration %d error = %v", i, err)
		}
		if seen[id] {
			t.Fatalf("duplicate ID %q at iteration %d", id, i)
		}
		seen[id] = true
	}
}

func TestStableIDGenerator_CounterContinuation(t *testing.T) {
	// When existing IDs are provided, the counter should continue from the highest value
	existingIDs := []string{"0000100", "0000050", "0000200"}
	gen := NewStableIDGenerator(existingIDs)

	// The next ID should be greater than 0000200 (the highest)
	id, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Parse the generated ID as base36
	generatedVal, err := strconv.ParseInt(id, 36, 64)
	if err != nil {
		t.Fatalf("failed to parse generated ID %q: %v", id, err)
	}

	// Parse the highest existing ID as base36
	highestVal, err := strconv.ParseInt("0000200", 36, 64)
	if err != nil {
		t.Fatalf("failed to parse highest ID: %v", err)
	}

	if generatedVal <= highestVal {
		t.Errorf("Generate() = %q (value %d), should be > %q (value %d)",
			id, generatedVal, "0000200", highestVal)
	}
}

func TestStableIDGenerator_CryptoRandSeeding(t *testing.T) {
	// Create two generators with empty existing IDs
	// They should produce different starting values due to crypto/rand seeding
	gen1 := NewStableIDGenerator([]string{})
	gen2 := NewStableIDGenerator([]string{})

	id1, err := gen1.Generate()
	if err != nil {
		t.Fatalf("gen1.Generate() error = %v", err)
	}

	id2, err := gen2.Generate()
	if err != nil {
		t.Fatalf("gen2.Generate() error = %v", err)
	}

	// With crypto/rand seeding, they should almost certainly be different
	// (probability of collision is 1 in 78 billion)
	if id1 == id2 {
		t.Logf("Warning: two generators produced same first ID %q (possible but unlikely)", id1)
	}
}

func TestStableIDGenerator_IsUsed(t *testing.T) {
	existingIDs := []string{"abc1234", "def5678"}
	gen := NewStableIDGenerator(existingIDs)

	tests := map[string]struct {
		id   string
		want bool
	}{
		"existing_id_1": {
			id:   "abc1234",
			want: true,
		},
		"existing_id_2": {
			id:   "def5678",
			want: true,
		},
		"non_existing_id": {
			id:   "xyz9999",
			want: false,
		},
		"empty_string": {
			id:   "",
			want: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := gen.IsUsed(tc.id)
			if got != tc.want {
				t.Errorf("IsUsed(%q) = %v, want %v", tc.id, got, tc.want)
			}
		})
	}
}

func TestStableIDGenerator_LowercaseOnly(t *testing.T) {
	gen := NewStableIDGenerator([]string{})

	for i := 0; i < 100; i++ {
		id, err := gen.Generate()
		if err != nil {
			t.Fatalf("Generate() iteration %d error = %v", i, err)
		}

		if strings.ToLower(id) != id {
			t.Errorf("Generate() = %q contains uppercase characters", id)
		}
	}
}

func TestIsValidStableID(t *testing.T) {
	tests := map[string]struct {
		id   string
		want bool
	}{
		"valid_7_chars_alphanumeric": {
			id:   "abc1234",
			want: true,
		},
		"valid_all_numbers": {
			id:   "0000001",
			want: true,
		},
		"valid_all_letters": {
			id:   "abcdefg",
			want: true,
		},
		"invalid_too_short": {
			id:   "abc123",
			want: false,
		},
		"invalid_too_long": {
			id:   "abc12345",
			want: false,
		},
		"invalid_uppercase": {
			id:   "ABC1234",
			want: false,
		},
		"invalid_special_chars": {
			id:   "abc-123",
			want: false,
		},
		"invalid_empty": {
			id:   "",
			want: false,
		},
		"invalid_mixed_case": {
			id:   "aBc1234",
			want: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := IsValidStableID(tc.id)
			if got != tc.want {
				t.Errorf("IsValidStableID(%q) = %v, want %v", tc.id, got, tc.want)
			}
		})
	}
}
