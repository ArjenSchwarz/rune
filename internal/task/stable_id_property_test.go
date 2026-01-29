package task

import (
	"strconv"
	"testing"

	"pgregory.net/rapid"
)

// TestProperty_StableIDUniqueness tests that all generated stable IDs in a list are unique
func TestProperty_StableIDUniqueness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random number of IDs to request (1-1000)
		numIDs := rapid.IntRange(1, 1000).Draw(t, "numIDs")

		// Optionally start with some existing IDs
		numExisting := rapid.IntRange(0, 100).Draw(t, "numExisting")
		existingIDs := make([]string, 0, numExisting)
		for i := range numExisting {
			// Generate valid base36 IDs for existing set
			val := int64(i + 1)
			base36 := strconv.FormatInt(val, 36)
			id := "0000000"[:7-len(base36)] + base36
			existingIDs = append(existingIDs, id)
		}

		gen := NewStableIDGenerator(existingIDs)
		generated := make(map[string]bool)

		// Mark existing as generated
		for _, id := range existingIDs {
			generated[id] = true
		}

		// Generate new IDs
		for range numIDs {
			id, err := gen.Generate()
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			// Property: each generated ID must be unique
			if generated[id] {
				t.Fatalf("duplicate ID generated: %s", id)
			}
			generated[id] = true
		}
	})
}

// TestProperty_StableIDFormat tests that all generated IDs have valid format
func TestProperty_StableIDFormat(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		numIDs := rapid.IntRange(1, 500).Draw(t, "numIDs")

		gen := NewStableIDGenerator([]string{})

		for range numIDs {
			id, err := gen.Generate()
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			// Property: all IDs must be valid stable IDs
			if !IsValidStableID(id) {
				t.Fatalf("generated invalid ID: %s", id)
			}

			// Property: all IDs must be exactly 7 characters
			if len(id) != 7 {
				t.Fatalf("generated ID with wrong length: %s (len=%d)", id, len(id))
			}
		}
	})
}

// TestProperty_StableIDMonotonic tests that IDs increase monotonically when starting from existing IDs
func TestProperty_StableIDMonotonic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Start with a random existing ID
		startValue := rapid.Int64Range(1, 1000000).Draw(t, "startValue")
		base36 := strconv.FormatInt(startValue, 36)
		startID := "0000000"[:7-len(base36)] + base36

		gen := NewStableIDGenerator([]string{startID})
		numIDs := rapid.IntRange(1, 100).Draw(t, "numIDs")

		var lastValue int64 = startValue

		for range numIDs {
			id, err := gen.Generate()
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			// Parse the generated ID
			currentValue, err := strconv.ParseInt(id, 36, 64)
			if err != nil {
				t.Fatalf("failed to parse generated ID %s: %v", id, err)
			}

			// Property: each ID should be greater than the last
			if currentValue <= lastValue {
				t.Fatalf("ID not monotonically increasing: %d <= %d", currentValue, lastValue)
			}
			lastValue = currentValue
		}
	})
}

// TestProperty_StableIDSurvivesMultipleCycles tests that IDs remain unique across generator instances
func TestProperty_StableIDSurvivesMultipleCycles(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		numCycles := rapid.IntRange(2, 5).Draw(t, "numCycles")
		idsPerCycle := rapid.IntRange(10, 100).Draw(t, "idsPerCycle")

		allIDs := make([]string, 0)

		for range numCycles {
			// Create a new generator with all previously generated IDs
			gen := NewStableIDGenerator(allIDs)

			for range idsPerCycle {
				id, err := gen.Generate()
				if err != nil {
					t.Fatalf("Generate() error = %v", err)
				}
				allIDs = append(allIDs, id)
			}
		}

		// Property: all IDs across all cycles must be unique
		seen := make(map[string]bool)
		for _, id := range allIDs {
			if seen[id] {
				t.Fatalf("duplicate ID across cycles: %s", id)
			}
			seen[id] = true
		}
	})
}
