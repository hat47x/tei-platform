package driver_test

import (
	"testing"

	"tei/internal/application"
	"tei/internal/driver"
	"tei/internal/ir"
	"tei/internal/realization"
	"tei/internal/semantic"
)

func TestReferenceOrderEntryCapabilityBindingsResolveByCanonicalIdentity(t *testing.T) {
	world := referenceWorld(t)
	resolver := driver.NewResolver(world)

	t.Run("command Patternからsales.order.submit Bindingを解決する", func(t *testing.T) {
		resolved, err := resolver.ResolveCommandBinding("order_entry", "order_entry.submit_order")
		if err != nil {
			t.Fatalf("command Bindingを解決できません: %v", err)
		}
		if resolved.Pattern.Pattern.ID != "order_entry.submit_order" ||
			resolved.Binding.Capability.ID != "sales.order.submit" ||
			resolved.Binding.Binding.Identity() != "order_entry/sales.order.submit" {
			t.Fatalf("command binding=%#v", resolved)
		}
		if resolved.Binding.Binding.ExpectedAdapterFilename() != "sales__order__submit.bind.go" {
			t.Fatalf("adapter filename=%q", resolved.Binding.Binding.ExpectedAdapterFilename())
		}
		if resolved.Binding.Binding.ExpectedHookName() != "TeiBindSalesOrderSubmit" {
			t.Fatalf("hook name=%q", resolved.Binding.Binding.ExpectedHookName())
		}
	})

	for _, tt := range []struct {
		name       string
		capability string
		identity   string
		filename   string
		hook       string
	}{
		{
			name:       "Authentication",
			capability: "security.authenticate",
			identity:   "order_entry/security.authenticate",
			filename:   "security__authenticate.bind.go",
			hook:       "TeiBindSecurityAuthenticate",
		},
		{
			name:       "Authorization",
			capability: "security.authorize",
			identity:   "order_entry/security.authorize",
			filename:   "security__authorize.bind.go",
			hook:       "TeiBindSecurityAuthorize",
		},
	} {
		t.Run(tt.name+"も同じApplication-scoped入口から解決する", func(t *testing.T) {
			resolved, err := resolver.ResolveCapabilityBinding("order_entry", driverCanonicalID(tt.capability))
			if err != nil {
				t.Fatalf("Capability Bindingを解決できません: %v", err)
			}
			if resolved.Capability.ID != driverCanonicalID(tt.capability) || resolved.Binding.Identity() != tt.identity {
				t.Fatalf("binding=%#v", resolved)
			}
			if resolved.Binding.ExpectedAdapterFilename() != tt.filename {
				t.Fatalf("adapter filename=%q want=%q", resolved.Binding.ExpectedAdapterFilename(), tt.filename)
			}
			if resolved.Binding.ExpectedHookName() != tt.hook {
				t.Fatalf("hook name=%q want=%q", resolved.Binding.ExpectedHookName(), tt.hook)
			}
		})
	}
}

func TestCapabilityBindingResolutionDoesNotSelectImplicitCandidates(t *testing.T) {
	baseWorld := ir.World{
		Applications: []application.Application{
		{
			ID: "app_a",
			Patterns: []application.Pattern{
				{ID: "app_a.command", Kind: application.PatternCommand, Capability: "sales.order.submit"},
				{ID: "app_a.record", Kind: application.PatternRecord, Data: "sales.order"},
			},
		},
		Semantics: []semantic.Namespace{
			{
				ID: "sales",
				Data: []semantic.Data{{ID: "sales.order"}},
				Capabilities: []semantic.Capability{
					{ID: "sales.order.submit", Output: "sales.order"},
				},
			},
		},
	}

	t.Run("Bindingがなければ失敗する", func(t *testing.T) {
		resolver := driver.NewResolver(baseWorld)
		if _, err := resolver.ResolveCapabilityBinding("app_a", "sales.order.submit"); err == nil {
			t.Fatal("存在しないBindingを暗黙に補っています")
		}
		if _, err := resolver.ResolveCommandBinding("app_a", "app_a.command"); err == nil {
			t.Fatal("command Patternから存在しないBindingを暗黙に補っています")
		}
	})

	t.Run("同じApplicationとCapabilityに複数Bindingがあれば曖昧として失敗する", func(t *testing.T) {
		world := baseWorld
		world.Bindings = []realization.CustomGoBinding{
			{Application: "app_a", Capability: "sales.order.submit"},
			{Application: "app_a", Capability: "sales.order.submit"},
		}
		resolver := driver.NewResolver(world)
		if _, err := resolver.ResolveCapabilityBinding("app_a", "sales.order.submit"); err == nil {
			t.Fatal("複数Bindingから一件を暗黙に選択しています")
		}
	})

	t.Run("command以外のPatternをBinding実行対象として扱わない", func(t *testing.T) {
		world := baseWorld
		world.Bindings = []realization.CustomGoBinding{{Application: "app_a", Capability: "sales.order.submit"}}
		resolver := driver.NewResolver(world)
		if _, err := resolver.ResolveCommandBinding("app_a", "app_a.record"); err == nil {
			t.Fatal("record Patternをcommandとして扱っています")
		}
	})
}

func driverCanonicalID(value string) core.CanonicalID {
	return core.CanonicalID(value)
}
