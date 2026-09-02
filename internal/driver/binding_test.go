package driver_test

import (
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/ir"
	"tei/internal/realization"
	"tei/internal/semantic"
)

func TestReferenceOrderEntryCapabilityBindingsUseOneApplicationScopedResolver(t *testing.T) {
	world := referenceWorld(t)
	resolver := driver.NewResolver(world)

	for _, capabilityID := range []core.CanonicalID{
		"sales.order.submit",
		"security.authenticate",
		"security.authorize",
	} {
		t.Run(string(capabilityID), func(t *testing.T) {
			resolved, err := resolver.ResolveCapabilityBinding("order_entry", capabilityID)
			if err != nil {
				t.Fatalf("Capability Bindingを解決できません: %v", err)
			}
			if resolved.Capability.ID != capabilityID {
				t.Fatalf("capability=%s want=%s", resolved.Capability.ID, capabilityID)
			}
			wantIdentity := realization.BindingIdentity("order_entry", resolved.Capability.ID)
			if resolved.Binding.Identity() != wantIdentity {
				t.Fatalf("binding identity=%s want=%s", resolved.Binding.Identity(), wantIdentity)
			}
		})
	}
}

func TestCommandPatternAndBindingResolutionRemainSeparate(t *testing.T) {
	world := referenceWorld(t)
	resolver := driver.NewResolver(world)

	pattern, err := resolver.ResolvePattern("order_entry", "order_entry.submit_order")
	if err != nil {
		t.Fatalf("command Patternを解決できません: %v", err)
	}
	if pattern.Capability == nil || pattern.Capability.ID != "sales.order.submit" {
		t.Fatalf("command Capability=%#v", pattern.Capability)
	}

	binding, err := resolver.ResolveCapabilityBinding("order_entry", pattern.Capability.ID)
	if err != nil {
		t.Fatalf("command Capability Bindingを解決できません: %v", err)
	}
	if binding.Binding.Identity() != "order_entry/sales.order.submit" {
		t.Fatalf("binding identity=%s", binding.Binding.Identity())
	}
}

func TestCapabilityBindingResolutionDoesNotSilentlyChooseMissingOrAmbiguousIdentity(t *testing.T) {
	baseApplication := application.Application{ID: "sample"}
	baseCapability := semantic.Capability{ID: "security.authenticate"}
	baseBinding := realization.CustomGoBinding{Application: "sample", Capability: "security.authenticate"}

	tests := []struct {
		name      string
		world     ir.World
		wantError string
	}{
		{
			name: "Applicationが存在しない",
			world: ir.World{
				Semantics: []semantic.Namespace{{ID: "security", Capabilities: []semantic.Capability{baseCapability}}},
				Bindings:  []realization.CustomGoBinding{baseBinding},
			},
			wantError: "Application sampleが見つかりません",
		},
		{
			name: "Capabilityが存在しない",
			world: ir.World{
				Applications: []application.Application{baseApplication},
				Bindings:     []realization.CustomGoBinding{baseBinding},
			},
			wantError: "Semantic Capability security.authenticateが見つかりません",
		},
		{
			name: "Capabilityが重複している",
			world: ir.World{
				Applications: []application.Application{baseApplication},
				Semantics: []semantic.Namespace{
					{ID: "security_a", Capabilities: []semantic.Capability{baseCapability}},
					{ID: "security_b", Capabilities: []semantic.Capability{baseCapability}},
				},
				Bindings: []realization.CustomGoBinding{baseBinding},
			},
			wantError: "Semantic Capability security.authenticateが複数定義されています",
		},
		{
			name: "Bindingが存在しない",
			world: ir.World{
				Applications: []application.Application{baseApplication},
				Semantics:    []semantic.Namespace{{ID: "security", Capabilities: []semantic.Capability{baseCapability}}},
			},
			wantError: "custom Go binding sample/security.authenticate is not defined",
		},
		{
			name: "Bindingが重複している",
			world: ir.World{
				Applications: []application.Application{baseApplication},
				Semantics:    []semantic.Namespace{{ID: "security", Capabilities: []semantic.Capability{baseCapability}}},
				Bindings:     []realization.CustomGoBinding{baseBinding, baseBinding},
			},
			wantError: "custom Go binding sample/security.authenticate is ambiguous",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := driver.NewResolver(tt.world)
			_, err := resolver.ResolveCapabilityBinding("sample", "security.authenticate")
			if err == nil {
				t.Fatal("errorを期待しました")
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("error=%q want substring=%q", err, tt.wantError)
			}
		})
	}
}
