package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	storepkg "github.com/adams100111/agentic-learning-partner/internal/store"
	"github.com/adams100111/agentic-learning-partner/internal/workspace"
)

func TestEvidenceAddCheckpointsThroughLocalStore(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "workspace.yaml"), []byte("schemaVersion: 2\nworkspaceId: ws_local_learning\nlearnerId: learner\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	validator, err := workspace.NewValidator()
	if err != nil {
		t.Fatal(err)
	}
	local, err := storepkg.OpenLocal(root, validator)
	if err != nil {
		t.Fatal(err)
	}
	before, err := local.Revision(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	input := filepath.Join(t.TempDir(), "evidence.yaml")
	evidence := []byte(`schemaVersion: 1
id: ev_local_acceptance
recordedAt: 2026-10-06T12:00:00Z
domain: go
competencies:
  - go.runtime.context
type: explanation
source:
  kind: diagnostic
  ref: acceptance-local
observation: Correctly explained request cancellation ownership.
result: pass
strength: moderate
`)
	if err := os.WriteFile(input, evidence, 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	app := App{Out: &out, ErrOut: &errOut, Getwd: func() (string, error) { return t.TempDir(), nil }}
	if code := app.Run([]string{"evidence", "add", "--workspace", root, "--file", input}); code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "ev_local_acceptance") {
		t.Fatalf("stdout = %q", out.String())
	}
	after, err := local.Revision(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if after == before {
		t.Fatal("Local Store evidence checkpoint must advance workspace revision")
	}
	if _, err := os.Stat(filepath.Join(root, "evidence", "ev_local_acceptance.yaml")); err != nil {
		t.Fatalf("evidence not published through Store transaction: %v", err)
	}
}

func TestSessionStagesEvidenceUntilSingleCloseCheckpoint(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "workspace.yaml"), []byte("schemaVersion: 2\nworkspaceId: ws_session_cli\nlearnerId: learner\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	validator, err := workspace.NewValidator()
	if err != nil {
		t.Fatal(err)
	}
	local, err := storepkg.OpenLocal(root, validator)
	if err != nil {
		t.Fatal(err)
	}
	before, err := local.Revision(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	var beginOut bytes.Buffer
	var beginErr bytes.Buffer
	app := App{Out: &beginOut, ErrOut: &beginErr, Getwd: func() (string, error) { return t.TempDir(), nil }}
	if code := app.Run([]string{"session", "begin", "--workspace", root, "--mode", "session", "--harness", "codex"}); code != 0 {
		t.Fatalf("begin exit=%d stderr=%s", code, beginErr.String())
	}

	input := filepath.Join(t.TempDir(), "session-evidence.yaml")
	evidence := []byte(`schemaVersion: 1
id: ev_session_acceptance
recordedAt: 2026-10-06T12:00:00Z
domain: go
competencies:
  - go.runtime.context
type: explanation
source:
  kind: diagnostic
  ref: acceptance-session
observation: Correctly explained cancellation ownership.
result: pass
strength: moderate
`)
	if err := os.WriteFile(input, evidence, 0o644); err != nil {
		t.Fatal(err)
	}

	var evidenceOut bytes.Buffer
	var evidenceErr bytes.Buffer
	app.Out, app.ErrOut = &evidenceOut, &evidenceErr
	if code := app.Run([]string{"evidence", "add", "--workspace", root, "--file", input}); code != 0 {
		t.Fatalf("evidence exit=%d stderr=%s", code, evidenceErr.String())
	}
	mid, err := local.Revision(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if mid != before {
		t.Fatalf("active-session evidence must remain staged; before=%s mid=%s", before, mid)
	}
	if _, err := os.Stat(filepath.Join(root, "evidence", "ev_session_acceptance.yaml")); !os.IsNotExist(err) {
		t.Fatalf("staged evidence leaked into canonical workspace: %v", err)
	}

	var closeOut bytes.Buffer
	var closeErr bytes.Buffer
	app.Out, app.ErrOut = &closeOut, &closeErr
	if code := app.Run([]string{"session", "close", "--workspace", root, "--summary", "Go context diagnostic"}); code != 0 {
		t.Fatalf("close exit=%d stderr=%s", code, closeErr.String())
	}
	after, err := local.Revision(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if after == before {
		t.Fatal("session close must publish one Store checkpoint")
	}
	if _, err := os.Stat(filepath.Join(root, "evidence", "ev_session_acceptance.yaml")); err != nil {
		t.Fatalf("evidence missing after session close: %v", err)
	}
	sessions, err := filepath.Glob(filepath.Join(root, "sessions", "sess_*.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Fatalf("compact session records = %v", sessions)
	}
	sessionData, err := os.ReadFile(sessions[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(sessionData), "ev_session_acceptance") || !strings.Contains(string(sessionData), "codex") {
		t.Fatalf("session record missing evidence/harness provenance:\n%s", sessionData)
	}
}
