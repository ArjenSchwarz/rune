package task

import "testing"

// TestNormalizePhaseName covers the shared phase-name normalisation and
// validation used by add-phase, add --phase, AddTaskToPhase, and the batch
// phase operations.
//
// Regression test for T-1603: a phase name containing a newline (or other
// control character) was accepted and rendered verbatim into the markdown
// output as "## {name}", letting the extra line(s) inject arbitrary
// markdown/task lines into the file. NormalizePhaseName must reject such names
// before they ever reach rendering, while still trimming (and accepting) names
// whose only offence is surrounding whitespace.
func TestNormalizePhaseName(t *testing.T) {
	tests := map[string]struct {
		name    string
		want    string
		wantErr bool
	}{
		"valid simple name": {
			name: "Planning",
			want: "Planning",
		},
		"valid name with spaces and punctuation": {
			name: "Q&A / Testing",
			want: "Q&A / Testing",
		},
		"surrounding whitespace is trimmed": {
			name: "  Planning  ",
			want: "Planning",
		},
		"name containing tab is accepted": {
			// Tab is permitted for consistency with task titles
			// (containsNullByte) and the parser, which reads back a
			// "## Design<TAB>Phase" header without complaint.
			name: "Design\tPhase",
			want: "Design\tPhase",
		},
		"surrounding tabs are trimmed": {
			name: "\tPlanning\t",
			want: "Planning",
		},
		"trailing newline is trimmed and accepted": {
			// This worked before T-1603 and must keep working: the newline is
			// not an injection vector once it has been trimmed off the end.
			name: "Trailing\n",
			want: "Trailing",
		},
		"leading and trailing CRLF is trimmed and accepted": {
			name: "\r\nPlanning\r\n",
			want: "Planning",
		},
		"empty name": {
			name:    "",
			wantErr: true,
		},
		"whitespace-only name": {
			name:    "   ",
			wantErr: true,
		},
		"newline-only name": {
			name:    "\n",
			wantErr: true,
		},
		"embedded newline": {
			// This is the T-1603 reproduction: a newline lets the phase name
			// inject an extra markdown line (e.g. a fake task) into the file.
			name:    "Bad\n- [ ] 999. Injected",
			wantErr: true,
		},
		"embedded carriage return": {
			name:    "Bad\rInjected",
			wantErr: true,
		},
		"embedded CRLF": {
			name:    "Bad\r\nInjected",
			wantErr: true,
		},
		"embedded null byte": {
			name:    "Bad\x00Name",
			wantErr: true,
		},
		"embedded DEL": {
			name:    "Bad\x7fName",
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := NormalizePhaseName(tc.name)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("NormalizePhaseName(%q): expected error, got nil (result %q)", tc.name, got)
				}
				if got != "" {
					t.Errorf("NormalizePhaseName(%q): expected empty name on error, got %q", tc.name, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizePhaseName(%q): unexpected error: %v", tc.name, err)
			}
			if got != tc.want {
				t.Errorf("NormalizePhaseName(%q) = %q, want %q", tc.name, got, tc.want)
			}
		})
	}
}
