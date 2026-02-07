package task

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// maxBase36_7Chars is the maximum value representable in 7 base36 chars: 36^7 - 1
const maxBase36_7Chars int64 = 78364164095 // "zzzzzzz" in base36

// stableIDPattern validates stable ID format: exactly 7 lowercase alphanumeric characters
var stableIDPattern = regexp.MustCompile(`^[a-z0-9]{7}$`)

// StableIDGenerator generates unique stable IDs for tasks
type StableIDGenerator struct {
	usedIDs map[string]bool
	counter int64
}

// NewStableIDGenerator creates a generator pre-populated with existing IDs.
// Seeds counter from the highest existing ID value to avoid collisions.
func NewStableIDGenerator(existingIDs []string) *StableIDGenerator {
	g := &StableIDGenerator{
		usedIDs: make(map[string]bool, len(existingIDs)),
	}

	var maxValue int64
	for _, id := range existingIDs {
		g.usedIDs[id] = true
		if val, err := strconv.ParseInt(id, 36, 64); err == nil && val > maxValue {
			maxValue = val
		}
	}

	if maxValue > 0 {
		g.counter = maxValue
	} else {
		// Seed from crypto/rand for first ID in file
		var buf [8]byte
		if _, err := rand.Read(buf[:]); err != nil {
			// Fallback to time-based if crypto/rand fails (shouldn't happen)
			g.counter = time.Now().UnixNano() % maxBase36_7Chars
		} else {
			// Use unsigned to avoid negative values, then take modulo
			g.counter = int64(binary.BigEndian.Uint64(buf[:]) % uint64(maxBase36_7Chars))
		}
	}

	return g
}

// Generate creates a new unique 7-character base36 ID
func (g *StableIDGenerator) Generate() (string, error) {
	for range 1000 {
		g.counter++

		// Check for exhaustion (practically impossible with 78 billion IDs)
		if g.counter > maxBase36_7Chars {
			return "", errors.New("stable ID space exhausted (78 billion IDs used)")
		}

		// Convert to base36 and zero-pad to 7 characters
		base36 := strconv.FormatInt(g.counter, 36)
		id := strings.Repeat("0", 7-len(base36)) + base36

		if !g.usedIDs[id] {
			g.usedIDs[id] = true
			return id, nil
		}
	}
	return "", errors.New("failed to generate unique stable ID after 1000 attempts")
}

// IsUsed checks if an ID is already in use
func (g *StableIDGenerator) IsUsed(id string) bool {
	return g.usedIDs[id]
}

// IsValidStableID checks if a string is a valid stable ID format
// (exactly 7 lowercase alphanumeric characters)
func IsValidStableID(id string) bool {
	return stableIDPattern.MatchString(id)
}
