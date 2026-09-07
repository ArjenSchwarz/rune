package task

import (
	"fmt"
	"strings"
	"unicode"
)

// NormalizePhaseName trims surrounding whitespace from a phase name and
// validates the result, returning the canonical name to use from that point on.
//
// Normalisation and validation are deliberately a single operation. Phase names
// are rendered verbatim into a markdown header ("## {name}") and are matched by
// exact string equality against existing phase markers, so every entry point
// must agree on both the trimming and the rejection rules. Splitting them into a
// separate trim step and a separate validator let the two drift apart: some call
// sites validated the raw argument while others validated the trimmed one, which
// made the same phase name succeed on one path and fail on another.
//
// A name is rejected if it is empty after trimming, or if the trimmed name still
// contains a control character. Trailing or leading newlines are therefore
// trimmed and accepted; a newline in the middle of a name is rejected, because
// that is what lets a phase name inject extra markdown/task lines into the file.
func NormalizePhaseName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", fmt.Errorf("phase name cannot be empty")
	}
	if strings.ContainsFunc(trimmed, isInvalidPhaseNameRune) {
		return "", fmt.Errorf("phase name cannot contain control characters (e.g. newlines)")
	}
	return trimmed, nil
}

// isInvalidPhaseNameRune reports whether r may not appear inside a phase name.
//
// Tab is deliberately allowed, matching containsNullByte (used for task titles,
// details and references) and the parser, which happily reads back a
// "## Design<TAB>Phase" header. Rejecting tab here would have made an existing
// phase with a tab in its name unusable: it would still parse and list, but no
// path could add a task to it. Every other control character is rejected —
// the line terminators because they are the actual injection vector, and the
// remainder (NUL, DEL, C1) because they have no meaning in a markdown heading.
func isInvalidPhaseNameRune(r rune) bool {
	return r != '\t' && unicode.IsControl(r)
}
