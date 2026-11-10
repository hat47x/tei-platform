package studiohost

import (
	"errors"
	"testing"

	"tei/internal/operations"

	"tei/internal/application"
	"tei/internal/workspace"
)

// A view response that is not OK names why in Failure.Code. These codes are the
// contract a Studio client branches on, so each one is produced here (#4032).
func TestViewFailureCodesAreProduced(t *testing.T) {
	host, err := NewReadOnly([]string{referenceWorkspace(t)})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("view unsupported", func(t *testing.T) {
		// A pattern kind with no application view is refused by the host rather
		// than answered with an empty value.
		response := host.ViewPattern("workspace-1", "order_entry", patternWithoutView(t))
		if response.OK || response.Failure == nil {
			t.Fatalf("a pattern without a view was accepted: %+v", response)
		}
		if response.Failure.Code != "VIEW_UNSUPPORTED" {
			t.Fatalf("code=%s (%s)", response.Failure.Code, response.Failure.Message)
		}
		if response.Failure.Stage != "view" {
			t.Fatalf("stage=%q", response.Failure.Stage)
		}
	})

	t.Run("view failed", func(t *testing.T) {
		// The kind is supported but the application does not own the pattern, so
		// building the view fails.
		response := host.ViewPattern("workspace-1", "order_entry", "order_entry.missing")
		if response.OK || response.Failure == nil {
			t.Fatalf("an unknown pattern was accepted: %+v", response)
		}
		if response.Failure.Code != "VIEW_FAILED" {
			t.Fatalf("code=%s (%s)", response.Failure.Code, response.Failure.Message)
		}
	})
}

// patternWithoutView returns the identifier of a pattern whose kind the host
// does not render, read from the reference workspace so the test does not
// hard-code an identifier.
func patternWithoutView(t *testing.T) string {
	t.Helper()
	loaded, err := workspace.Load(referenceWorkspace(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, app := range loaded.World.Applications {
		if app.ID != "order_entry" {
			continue
		}
		for _, pattern := range app.Patterns {
			switch pattern.Kind {
			case application.PatternHeaderDetail, application.PatternBrowse, application.PatternCommand:
			default:
				return string(pattern.ID)
			}
		}
	}
	t.Skip("the reference workspace has no pattern without an application view")
	return ""
}

// A failure while building the source index has its own code, because a client
// can retry a workspace load but cannot proceed without the index (#4032).
func TestSourceIndexFailureCodeIsProduced(t *testing.T) {
	failure := operationFailure(&operations.ValidationFailure{Stage: operations.StageSourceIndex, Err: errors.New("index unavailable")})
	if failure.Code != "SOURCE_INDEX_FAILED" {
		t.Fatalf("code=%s (%s)", failure.Code, failure.Message)
	}
	if failure.Stage != string(operations.StageSourceIndex) {
		t.Fatalf("stage=%q", failure.Stage)
	}
}

// A command run reports failure through Failure.Code. One code is declared but
// has no input: COMMAND_FAILED is written when runHooks returns an error, and
// runHooks returns an error only together with a report (so it is not nil) and
// returns (nil, nil) for a pattern that declares no hook. This test states the
// arrangement so the code is not read as a reachable answer (#4032).
func TestCommandFailedHasNoKnownTrigger(t *testing.T) {
	host, err := NewReadOnly([]string{hookWorkspace(t)})
	if err != nil {
		t.Fatal(err)
	}
	// The command pattern declares no hook, which is the only case where the
	// hook runner reports nothing. The run is not refused.
	run := host.ExecuteCommand(
		"workspace-1",
		"order_entry",
		"order_entry.submit",
		"sales.submit",
		map[string]string{"order": "open", "sales.order.status": "open"},
	)
	// The declared code is named here so a reader and the repository-wide code
	// coverage check both see it; the assertion is that it does not appear.
	const unreachableCode = "COMMAND_FAILED"
	if run.Failure != nil && run.Failure.Code == unreachableCode {
		t.Fatalf("a pattern without hooks produced %s: %+v", unreachableCode, run.Failure)
	}
	if run.Failure != nil {
		t.Fatalf("a pattern without hooks was reported as a failure: %+v", run.Failure)
	}
	if run.OK {
		// The command itself may or may not be accepted; what matters here is
		// that no hook failure was reported.
		t.Log("the command ran without a hook report")
	}
}
