package cmd

import (
	"encoding/json"
	"strings"
	"testing"
)

// T-905: --phase --claim must claim a ready child of a completed parent, as
// the non-phase --claim does. The phase is already selected because of that
// child, so returning an empty claim set was inconsistent.
func TestNextPhaseClaimReadyChildOfCompletedParent(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, `# Project Tasks

## Phase 1
- [x] 1. Completed parent
  - [ ] 1.1. Pending child

## Phase 2
- [ ] 2. Later task
`)

	out, _ := runRuneCLI(t, dir, "next", "tasks.md", "--phase", "--claim", "agent-1", "--format", "json")
	var got ClaimResponse
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("parse claim JSON: %v\n%s", err, out)
	}
	if len(got.Claimed) != 1 || got.Claimed[0].ID != "1.1" {
		t.Fatalf("want task 1.1 claimed, got: %s", out)
	}
}

// T-905: a pending parent keeps the existing behaviour: the parent is claimed,
// not its children, and a claimed (in-progress) parent hides its children.
func TestNextPhaseClaimOnlyDescendsThroughCompletedParents(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, `# Project Tasks

## Phase 1
- [ ] 1. Pending parent
  - [ ] 1.1. Pending child
- [-] 2. Claimed parent
  - Owner: agent-x
  - [ ] 2.1. Pending child
`)

	out, _ := runRuneCLI(t, dir, "next", "tasks.md", "--phase", "--claim", "agent-1", "--format", "json", "--dry-run")
	var got ClaimResponse
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("parse claim JSON: %v\n%s", err, out)
	}
	if len(got.Claimed) != 1 || got.Claimed[0].ID != "1" {
		t.Fatalf("want only task 1 claimed, got: %s", out)
	}
}

const unphasedStreamFile = `# Repro

- [ ] 1. Ready default stream

- [ ] 2. Ready stream two
  - Stream: 2
`

// T-1319: --phase --stream on a file without phase headers must fall back to
// the whole file, as --phase does, instead of reporting an empty stream.
func TestNextPhaseStreamUnphasedFile(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, unphasedStreamFile)

	out, _ := runRuneCLI(t, dir, "next", "tasks.md", "--phase", "--stream", "1", "--format", "json")
	var got PhaseOutputJSONWithStreams
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("parse phase JSON: %v\n%s", err, out)
	}
	if len(got.Tasks) != 1 || got.Tasks[0].ID != "1" {
		t.Fatalf("want task 1 for stream 1, got: %s", out)
	}
}

// T-1319: the same fallback applies to --phase --stream --claim.
func TestNextPhaseStreamClaimUnphasedFile(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, unphasedStreamFile)

	out, _ := runRuneCLI(t, dir, "next", "tasks.md", "--phase", "--stream", "2", "--claim", "agent-1", "--format", "json", "--dry-run")
	var got ClaimResponse
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("parse claim JSON: %v\n%s", err, out)
	}
	if len(got.Claimed) != 1 || got.Claimed[0].ID != "2" {
		t.Fatalf("want task 2 claimed, got: %s", out)
	}
}

// T-1433: next --stream N must not show descendants whose effective stream is
// not N, in any output format.
func TestNextStreamExcludesOtherStreamChildren(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, `# Project Tasks

- [ ] 1. Parent in stream 2 <!-- id:abc1234 -->
  - Stream: 2
  - [ ] 1.1. Child in stream 1 <!-- id:def5678 -->
    - Stream: 1
  - [ ] 1.2. Child in stream 2 <!-- id:fed8765 -->
    - Stream: 2
`)

	for _, f := range []string{"json", "table", "markdown"} {
		out, _ := runRuneCLI(t, dir, "next", "tasks.md", "--stream", "2", "--format", f)
		if strings.Contains(out, "Child in stream 1") {
			t.Errorf("%s output includes stream 1 child:\n%s", f, out)
		}
		for _, shown := range []string{"Parent in stream 2", "Child in stream 2"} {
			if !strings.Contains(out, shown) {
				t.Errorf("%s output is missing %q:\n%s", f, shown, out)
			}
		}
	}
}

// T-1705: next --phase --stream N must not show children from other streams,
// in any output format.
func TestNextPhaseStreamExcludesOtherStreamChildren(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, `# Project

## Build
- [ ] 1. Parent stream 2 <!-- id:aaa0001 -->
  - Stream: 2
  - [ ] 1.1. Child stream 1 <!-- id:bbb0002 -->
    - Stream: 1
`)

	for _, f := range []string{"json", "table", "markdown"} {
		out, _ := runRuneCLI(t, dir, "next", "tasks.md", "--phase", "--stream", "2", "--format", f)
		if strings.Contains(out, "Child stream 1") {
			t.Errorf("%s output includes stream 1 child:\n%s", f, out)
		}
		if !strings.Contains(out, "Parent stream 2") {
			t.Errorf("%s output is missing the stream 2 parent:\n%s", f, out)
		}
	}
}

// T-2000: in phase mode the streams summary describes the selected phase only.
func TestNextPhaseStreamsSummaryScopedToPhase(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, `# Project

## Phase One
- [ ] 1. One
- [-] 2. Active one
  - Owner: agent-x

## Phase Two
- [ ] 3. Two
  - Stream: 2
`)

	out, _ := runRuneCLI(t, dir, "next", "tasks.md", "--phase", "--format", "json")
	var got PhaseOutputJSONWithStreams
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("parse phase JSON: %v\n%s", err, out)
	}
	if got.PhaseName != "Phase One" {
		t.Fatalf("want Phase One selected, got: %s", out)
	}
	if len(got.StreamsSummary) != 1 {
		t.Fatalf("want only stream 1 in summary, got: %s", out)
	}
	s := got.StreamsSummary[0]
	if s.ID != 1 || strings.Join(s.Ready, ",") != "1" || strings.Join(s.Active, ",") != "2" {
		t.Errorf("unexpected stream 1 summary: %+v", s)
	}
}

// T-2000 with --stream: the summary covers the selected phase's stream tasks
// and lists nested matches once.
func TestNextPhaseStreamSummaryScopedToPhase(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, `# Project

## Phase One
- [ ] 1. Parent <!-- id:aaa0001 -->
  - Stream: 2
  - [ ] 1.1. Child <!-- id:bbb0002 -->
    - Stream: 2

## Phase Two
- [ ] 2. Later <!-- id:ccc0003 -->
  - Stream: 2
`)

	out, _ := runRuneCLI(t, dir, "next", "tasks.md", "--phase", "--stream", "2", "--format", "json")
	var got PhaseOutputJSONWithStreams
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("parse phase JSON: %v\n%s", err, out)
	}
	if len(got.StreamsSummary) != 1 || strings.Join(got.StreamsSummary[0].Ready, ",") != "1,1.1" {
		t.Errorf("want stream 2 ready [1 1.1] only, got: %s", out)
	}
}
