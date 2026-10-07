package pylearn

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"
)

func readOneRecord(t *testing.T, record map[string]any) string {
	t.Helper()
	record["item"] = "go-alp-a1-context"
	record["event"] = map[string]any{"id": "row:go-alp-a1-context", "revision": "sha256:" + strings.Repeat("a", 64), "synthetic": true}
	record["observedAt"] = "2026-10-07T08:10:00Z"
	export, err := json.Marshal(map[string]any{
		"schemaVersion": 2, "platform": "pylearn", "instance": "pylearn-local", "exportedAt": "2026-10-07T09:00:00Z",
		"user":    map[string]any{"id": "usr_7f3a"},
		"cursor":  map[string]any{"since": nil, "next": "cursor-0001"},
		"targets": []any{map[string]any{"id": "go-alp", "records": []any{record}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	batch, err := NewAdapter().ReadActivity(export, "go-alp", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(batch.Records) != 1 {
		t.Fatalf("records = %d", len(batch.Records))
	}
	return batch.Records[0].Signal.Observation
}

func TestFailedAttemptKeepsOnlyABoundedSanitizedFailureSummary(t *testing.T) {
	failure := "\n  --- FAIL: TestCancel (0.01s)\x1b[31m\n" +
		"    /Users/dana/src/go-alp/context_test.go:14: ctx.Err() = <nil>, want context.Canceled\n" +
		"FAIL\nexit status 1\nGOPATH=/home/dana/go SECRET_TOKEN=abc123"
	observation := readOneRecord(t, map[string]any{"kind": "attempt", "variant": "v1", "status": "failed", "failure": failure})

	want := "PyLearn exercise variant v1 failed its checks. First failure: --- FAIL: TestCancel (0.01s)"
	if observation != want {
		t.Fatalf("observation = %q\nwant          %q", observation, want)
	}
}

func TestFailedAttemptFailureSummaryDropsPathsAndIsBounded(t *testing.T) {
	failure := "context_test.go:14 in /Users/dana/src/go-alp/context_test.go: " + strings.Repeat("ctx.Err() = <nil> ", 40)
	observation := readOneRecord(t, map[string]any{"kind": "attempt", "variant": "v1", "status": "error", "failure": failure})

	if strings.Contains(observation, "/Users/dana") || strings.Contains(observation, "dana") {
		t.Fatalf("observation keeps a local filesystem path: %q", observation)
	}
	if !strings.Contains(observation, "in context_test.go: ctx.Err()") {
		t.Fatalf("observation lost the failing file name: %q", observation)
	}
	summary := strings.TrimPrefix(observation, "PyLearn exercise variant v1 raised an error. First failure: ")
	if utf8.RuneCountInString(summary) > 200 || !strings.HasSuffix(summary, "…") {
		t.Fatalf("failure summary is not bounded to %d runes: %d %q", 200, utf8.RuneCountInString(summary), summary)
	}
}

func TestReflectionKeepsTheLearnersReflectionAsLearningEvidence(t *testing.T) {
	text := "Cancellation flows down the tree:\na child context never outlives its parent."
	observation := readOneRecord(t, map[string]any{"kind": "reflection", "text": "  " + text + "\n"})

	if observation != "PyLearn learner reflection: "+text {
		t.Fatalf("observation = %q", observation)
	}
}
