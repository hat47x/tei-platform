package webapicontract

import (
	"errors"
	"testing"

	"tei/internal/core"
	"tei/internal/workspace"
)

func TestExistingOrderCommandIsOnlyResolvedNotReimplemented(t *testing.T) {
	root, _ := orderWorld(t)
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	app := core.MustCanonicalID("order_entry")
	action := core.MustCanonicalID("order_entry.submit_order")
	confirmed, err := ResolvePublishedCommand(loaded.World, app, action, true)
	if err != nil {
		t.Fatal(err)
	}
	if confirmed.PatternID != action ||
		confirmed.CapabilityID != core.MustCanonicalID("sales.order.submit") {
		t.Fatalf("command must preserve existing meaning: %+v", confirmed)
	}
	if len(confirmed.Inputs) != 1 ||
		string(confirmed.Inputs[0].Name) != "order" ||
		confirmed.Inputs[0].Type != core.MustCanonicalID("sales.order") {
		t.Fatalf("published action is not the typed source contract: %+v", confirmed.Inputs)
	}
	cases := []struct {
		name string
		app core.CanonicalID
		pattern core.CanonicalID
		allowed bool
	}{
		{"denied", app, action, false},
		{"wrong app", core.MustCanonicalID("repair_intake"), action, true},
		{"read pattern not command", app, core.MustCanonicalID("order_entry.recent_orders"), true},
		{"missing action", app, core.MustCanonicalID("order_entry.undefined"), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ResolvePublishedCommand(loaded.World, c.app, c.pattern, c.allowed)
			if !errors.Is(err, ErrRejected) {
				t.Fatalf("invalid write API exposure accepted: %v", err)
			}
		})
	}
}
