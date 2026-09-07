package task

import "testing"

// TestValidatePhaseName covers the shared phase-name validation used by
// add-phase, add --phase, and batch phase operations.
//
// Regression test for T-1603: a phase name containing a newline (or other
// control character) was accepted and rendered verbatim into the markdown
// output as "## {name}", letting the extra line(s) inject arbitrary
// markdown/task lines into the file. ValidatePhaseName must reject such
// names before they ever reach rendering.
func TestValidatePhaseName(t *testing.T) {
	tests := map[string]struct {
		name    string
		wantErr bool
	}{
		"valid simple name": {
			name:    "Planning",
			wantErr: false,
		},
		"valid name with spaces and punctuation": {
			name:    "Q&A / Testing",
			wantErr: false,
		},
		"empty name": {
			name:    "",
			wantErr: true,
		},
		"whitespace-only name": {
			name:    "   ",
			wantErr: true,
		},
		"name containing newline": {
			// This is the T-1603 reproduction: a newline lets the phase name
			// inject an extra markdown line (e.g. a fake task) into the file.
			name:    "Bad\n- [ ] 999. Injected",
			wantErr: true,
		},
		"name containing carriage return": {
			name:    "Bad\rInjected",
			wantErr: true,
		},
		"name containing CRLF": {
			name:    "Bad\r\nInjected",
			wantErr: true,
		},
		"name containing tab": {
			name:    "Bad\tName",
			wantErr: true,
		},
		"name containing null byte": {
			name:    "Bad\x00Name",
			wantErr: true,
		},
		"name with trailing newline only": {
			name:    "Trailing\n",
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := ValidatePhaseName(tc.name)
			if tc.wantErr && err == nil {
				t.Errorf("ValidatePhaseName(%q): expected error, got nil", tc.name)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("ValidatePhaseName(%q): expected no error, got %v", tc.name, err)
			}
		})
	}
}
