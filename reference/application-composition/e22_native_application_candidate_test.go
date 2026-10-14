// E22 checks whether a native application's minimal action shell can use
// existing TEI Workspace/Driver machinery without copying the native domain.
// Everything is written to t.TempDir; SUI Stage 0 remains unchanged.
package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/application"
	"tei/internal/driver"
	"tei/internal/workspace"
	"tei/tools/reference-sui-action-boundary/actionboundary"
)

const e22Capability = "namespace sui\ncapability sui.move {\n}\n"
const e22Application = "application sui {\n    uses semantic sui\n}\n"
const e22Pattern = "pattern sui.move command {\n    capability sui.move\n}\n"

// The candidate is not a modification to reference/sui-adoption/workspace.
// It describes the minimum *TEI-facing* shell and intentionally defines no
// Card, Island movement semantics, payload decoder, implementation, or grant.
func e22Workspace(t *testing.T, capability, duplicate bool) string {
	t.Helper()
	root := t.TempDir()
	existing := filepath.Join("..", "..", "reference", "sui-adoption", "workspace", "semantic", "sui", "island.tei")
	island, err := os.ReadFile(existing)
	if err != nil {
		t.Fatalf("read existing SUI semantic definition: %v", err)
	}
	files := map[string]string{
		"tei.workspace.yaml":                 "workspace: sui_composition_e22_candidate\nsources:\n  - semantic\n  - applications\n",
		"semantic/sui/island.tei":            string(island),
		"applications/sui/application.tei":   e22Application,
		"applications/sui/patterns/move.tei": e22Pattern,
	}
	if capability {
		files["semantic/sui/native-action.tei"] = e22Capability
	}
	if duplicate {
		files["applications/sui/patterns/duplicate.tei"] = e22Pattern
	}
	for name, content := range files {
		dst := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

type e22CountingStore struct{ writes int }

func (s *e22CountingStore) Commit(context.Context, actionboundary.Intent) (string, error) {
	s.writes++
	return "v2", nil
}

func TestE22NativeApplicationCandidateResolvesWithoutDomainCopy(t *testing.T) {
	stage0, err := workspace.Load(filepath.Join("..", "..", "reference", "sui-adoption", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	if len(stage0.World.Applications) != 0 {
		t.Fatalf("SUI Stage 0 must not be mistaken for an accepted Application: %d", len(stage0.World.Applications))
	}
	if _, err := driver.NewResolver(stage0.World).ResolvePattern("sui", "sui.move"); err == nil {
		t.Fatal("Stage 0 unexpectedly resolves a non-existent SUI Action")
	}

	candidate, err := workspace.Load(e22Workspace(t, true, false))
	if err != nil {
		t.Fatalf("isolated application overlay cannot load: %v", err)
	}
	resolved, err := driver.NewResolver(candidate.World).ResolvePattern("sui", "sui.move")
	if err != nil {
		t.Fatalf("existing Driver rejected the isolated candidate: %v", err)
	}
	if resolved.Application.ID != "sui" || resolved.Pattern.ID != "sui.move" ||
		resolved.Pattern.Kind != application.PatternCommand ||
		resolved.Capability == nil || resolved.Capability.ID != "sui.move" {
		t.Fatalf("unexpected action resolution: %+v", resolved)
	}
	if len(resolved.Capability.Inputs) != 0 || resolved.Capability.Output != "" ||
		resolved.Capability.Effect != nil {
		t.Fatal("the candidate must not invent SUI Card payload or persistence semantics")
	}
	if _, ok := candidate.Sources["application-pattern:sui.move"]; !ok {
		t.Fatal("action Source provenance not retained by the existing Loader")
	}
	if _, ok := candidate.Sources["semantic-capability:sui.move"]; !ok {
		t.Fatal("capability Source provenance not retained by the existing Loader")
	}
	order, err := workspace.Load(filepath.Join("..", "..", "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	actualOrder, err := driver.NewResolver(order.World).ResolvePattern("order_entry", "order_entry.submit_order")
	if err != nil || actualOrder.Capability == nil || actualOrder.Pattern.Kind != application.PatternCommand {
		t.Fatalf("existing order_entry consumer no longer resolves: %v", err)
	}
	if _, err := driver.NewResolver(candidate.World).ResolvePattern("order_entry", "order_entry.submit_order"); err == nil {
		t.Fatal("the isolated SUI candidate borrowed an unrelated Application")
	}
}

func TestE22NativeCandidateMissingOrDuplicateDeclarationsFailClosed(t *testing.T) {
	missing, err := workspace.Load(e22Workspace(t, false, false))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := driver.NewResolver(missing.World).ResolvePattern("sui", "sui.move"); err == nil {
		t.Fatal("missing SUI capability was silently inferred from Action name")
	}

	duplicate, err := workspace.Load(e22Workspace(t, true, true))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := driver.NewResolver(duplicate.World).ResolvePattern("sui", "sui.move"); err == nil {
		t.Fatal("duplicate SUI action declarations were silently selected by file order")
	}
}

func TestE22ResolvedCandidateCannotAuthorizeNativeCommit(t *testing.T) {
	candidate, err := workspace.Load(e22Workspace(t, true, false))
	if err != nil {
		t.Fatal(err)
	}
	store := &e22CountingStore{}
	host := actionboundary.Host{
		CommandGate: &actionboundary.CanonicalCommandGate{Resolver: driver.NewResolver(candidate.World)},
		Bindings: []actionboundary.Binding{{
			ApplicationID: "sui", ActionID: "sui.move", Store: store,
		}},
		// Deliberately no runtime Authorizer or verified SUI session.
	}
	_, err = host.Commit(context.Background(), actionboundary.Intent{
		ApplicationID: "sui", ResourceID: "doc-1", ActionID: "sui.move",
		ExpectedRevision: "v1",
		Payload:          actionboundary.MovePayload{CardID: "card-1", X: 3, Y: 4},
	})
	if !errors.Is(err, actionboundary.ErrDenied) || store.writes != 0 {
		t.Fatalf("Candidate Source granted unauthorized write: %v (writes=%d)", err, store.writes)
	}
}
