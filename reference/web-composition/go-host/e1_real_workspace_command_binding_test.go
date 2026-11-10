package webcomposition

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/ir"
	"tei/internal/realization"
	"tei/internal/workspace"
)

// This exercises the existing Driver, reading the real reference Workspace.
// It proves a bounded Canonical Pattern -> Capability -> Custom Go Binding
// resolution, not adapter loading/invocation, Provider readiness, Host wiring,
// graph completeness, runtime permission, effect persistence or delivery.
func TestHDGRealWorkspaceCommandBindingWitness(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate Go reference test Source")
	}
	root := filepath.Join(filepath.Dir(currentFile), "..", "..", "workspace")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	pairBytes, err := os.ReadFile(filepath.Join(filepath.Dir(currentFile), "..", "e1-go-route-pair.fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pair routePairFixture
	if err := json.Unmarshal(pairBytes, &pair); err != nil {
		t.Fatal(err)
	}
	if !routePairMatches(pair) {
		t.Fatal("E1/Go test-only Action pair is not identical")
	}
	applicationID := core.MustCanonicalID("order_entry")
	patternID := core.MustCanonicalID("order_entry.submit_order")
	capabilityID := core.MustCanonicalID("sales.order.submit")
	const wantBinding = "order_entry/sales.order.submit"

	binding, err := driver.NewResolver(loaded.World).ResolveCommandBinding(applicationID, patternID)
	if err != nil {
		t.Fatalf("actual Driver rejected Canonical command Binding: %v", err)
	}
	if binding.Pattern.Application.ID != applicationID ||
		binding.Pattern.Pattern.ID != patternID ||
		binding.Pattern.Pattern.Kind != application.PatternCommand ||
		binding.Pattern.Pattern.Capability != capabilityID ||
		binding.Pattern.Capability == nil ||
		binding.Pattern.Capability.ID != capabilityID ||
		binding.Binding.Capability.ID != capabilityID ||
		binding.Binding.Binding.Identity() != wantBinding ||
		binding.Binding.Binding.Application != applicationID ||
		binding.Binding.Binding.Capability != capabilityID ||
		pair.E1.Route.ActionID != string(patternID) ||
		pair.GoHost.ActionID != string(patternID) {
		t.Fatalf("actual Canonical chain differs from independent expected: %+v", binding)
	}

	// The fixture must not select an implementation merely by its filename.
	// These negative controls mutate a private World copy, not Workspace
	// Sources or the accepted Canonical files on disk.
	cloneWorld := func() ir.World {
		other := loaded.World
		other.Bindings = append([]realization.CustomGoBinding(nil), loaded.World.Bindings...)
		return other
	}
	t.Run("HDG-R02-missing-explicit-Binding-must-fail", func(t *testing.T) {
		world := cloneWorld()
		keep := world.Bindings[:0]
		for _, candidate := range world.Bindings {
			if candidate.Identity() != wantBinding {
				keep = append(keep, candidate)
			}
		}
		world.Bindings = keep
		if _, err := driver.NewResolver(world).ResolveCommandBinding(applicationID, patternID); err == nil {
			t.Fatal("Driver filled in a missing realization by name or guessed path")
		}
	})
	t.Run("HDG-R03-duplicate-Binding-must-fail", func(t *testing.T) {
		world := cloneWorld()
		world.Bindings = append(world.Bindings, binding.Binding.Binding)
		if _, err := driver.NewResolver(world).ResolveCommandBinding(applicationID, patternID); err == nil {
			t.Fatal("Driver silently selected one of two same-identity realizations")
		}
	})
	t.Run("HDG-R04-different-valid-Action-Capability-is-not-the-expected-Binding", func(t *testing.T) {
		world := cloneWorld()
		world.Applications = append([]application.Application(nil), loaded.World.Applications...)
		var changed bool
		for i := range world.Applications {
			if world.Applications[i].ID != applicationID {
				continue
			}
			world.Applications[i].Patterns = append([]application.Pattern(nil), world.Applications[i].Patterns...)
			for j := range world.Applications[i].Patterns {
				if world.Applications[i].Patterns[j].ID == patternID {
					world.Applications[i].Patterns[j].Capability = core.MustCanonicalID("security.authorize")
					changed = true
				}
			}
		}
		if !changed {
			t.Fatal("expected Canonical command Pattern fixture was not present")
		}
		other, err := driver.NewResolver(world).ResolveCommandBinding(applicationID, patternID)
		if err != nil {
			t.Fatalf("negative case needs an independently valid alternative Binding: %v", err)
		}
		if other.Binding.Binding.Identity() == wantBinding || other.Binding.Capability.ID == capabilityID {
			t.Fatalf("valid but semantically wrong Capability passed independent expected: %+v", other.Binding)
		}
	})
}
