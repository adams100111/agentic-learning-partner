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
	if err != nil { t.Fatal(err) }
	local, err := storepkg.OpenLocal(root, validator)
	if err != nil { t.Fatal(err) }
	before, err := local.Revision(context.Background())
	if err != nil { t.Fatal(err) }

	input := filepath.Join(t.TempDir(), "evidence.yaml")
	evidence := []byte(`schemaVersion: 1
id: ev_local_acceptance
recordedAt: 2026-10-06T12:00:00Z
domain: go
competencies:
  - go.runtime.context
type: diagnostic
source:
  kind: diagnostic
  ref: acceptance-local
observation: Correctly explained request cancellation ownership.
result: pass
strength: moderate
`)
	if err := os.WriteFile(input, evidence, 0o644); err != nil { t.Fatal(err) }

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
	if err != nil { t.Fatal(err) }
	if after == before {
		t.Fatal("Local Store evidence checkpoint must advance workspace revision")
	}
	if _, err := os.Stat(filepath.Join(root, "evidence", "ev_local_acceptance.yaml")); err != nil {
		t.Fatalf("evidence not published through Store transaction: %v", err)
	}
}
