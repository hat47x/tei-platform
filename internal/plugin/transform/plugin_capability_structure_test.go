package transform_test

import (
	"bytes"
	"strings"
	"testing"

	"tei/internal/deployment"
	"tei/internal/environment"
	"tei/internal/permission"
	"tei/internal/plugin"
	"tei/internal/plugin/transform"
)

func TestCapabilityValidationParityWithTEISource(t *testing.T) {
	validPermission := permission.Spec{Kind: permission.KindFilesystem, Access: permission.AccessRead, Scope: "files"}
	cases := []struct {
		name   string
		change func(*plugin.Definition)
		reject string
	}{
		{name: "valid scope and reordered permissions", change: func(d *plugin.Definition) {
			d.Provides[1].Permissions[0], d.Provides[1].Permissions[2] = d.Provides[1].Permissions[2], d.Provides[1].Permissions[0]
		}},
		{name: "valid additional permission", change: func(d *plugin.Definition) {
			d.Provides[1].Permissions = append(d.Provides[1].Permissions, validPermission)
		}},
		{name: "invalid capability kind", change: func(d *plugin.Definition) {
			d.Provides[0].Kind = plugin.CapabilityKind("other")
		}, reject: "assert-enum failed"},
		{name: "invalid action", change: func(d *plugin.Definition) {
			d.Provides[1].Actions = []deployment.ActionKind{"other"}
		}, reject: "assert-enum failed"},
		{name: "invalid observation", change: func(d *plugin.Definition) {
			d.Provides[0].Observations = []deployment.ObservationKind{"other"}
		}, reject: "assert-enum failed"},
		{name: "invalid evidence", change: func(d *plugin.Definition) {
			d.Provides[0].Evidence = []deployment.EvidenceKind{"other"}
		}, reject: "assert-enum failed"},
		{name: "generic database provider", change: func(d *plugin.Definition) {
			d.Provides[0].DatabaseProviders = []environment.DatabaseProvider{environment.ProviderGeneric}
		}, reject: "assert-enum failed"},
		{name: "invalid permission kind", change: func(d *plugin.Definition) {
			d.Provides[1].Permissions[0].Kind = permission.Kind("other")
		}, reject: "assert-enum failed"},
		{name: "invalid permission access", change: func(d *plugin.Definition) {
			d.Provides[1].Permissions[0].Access = permission.AccessConnect
		}, reject: "assert-enum failed"},
		{name: "permission scope absent", change: func(d *plugin.Definition) {
			d.Provides[1].Permissions[0].Scope = ""
		}, reject: "assert-nonblank failed"},
		{name: "permission scope blank", change: func(d *plugin.Definition) {
			d.Provides[1].Permissions[0].Scope = "  \t"
		}, reject: "assert-nonblank failed"},
		{name: "duplicate permission", change: func(d *plugin.Definition) {
			d.Provides[1].Permissions = append(d.Provides[1].Permissions, d.Provides[1].Permissions[0])
		}, reject: "duplicate key"},
		{name: "nil permissions", change: func(d *plugin.Definition) {
			d.Provides[1].Permissions = nil
		}},
		{name: "empty permissions", change: func(d *plugin.Definition) {
			d.Provides[1].Permissions = []permission.Spec{}
		}},
	}
	source := referencePluginPackageTransformSource(t)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			definition, plan := transformPackageFixture()
			tc.change(&definition)
			plan.PluginDefinitionFingerprint = plugin.BuildDefinitionFingerprint(definition)
			want, goErr := plugin.MaterializePluginPackage(definition, plan)
			got, teiErr := transform.Evaluate(source, map[string]transform.Value{
				"buildPlan":        valueFromGoJSON(t, plan),
				"pluginDefinition": valueFromGoJSON(t, definition),
			})
			if tc.reject != "" {
				if goErr == nil || teiErr == nil || !strings.Contains(teiErr.Error(), tc.reject) {
					t.Fatalf("invalid definition accepted: Go=%v TEI=%v", goErr, teiErr)
				}
				return
			}
			if goErr != nil || teiErr != nil {
				t.Fatalf("valid definition rejected: Go=%v TEI=%v", goErr, teiErr)
			}
			if got.Output.Kind != transform.ValueBytes || !bytes.Equal(got.Output.Bytes, want) {
				t.Fatal("valid definition changed Go/TEI package equivalence")
			}
		})
	}
}

func TestAssertEachValidatesMembersWithoutChangingSource(t *testing.T) {
	source := uniqueByTestSource()
	arr := transform.Expr{Op: transform.OpInput, Name: "dependencies"}
	item := transform.Expr{Op: transform.OpInput, Name: "dependency"}
	source.Program.Assertions = []transform.Expr{{
		Op: transform.OpAssertEach,
		From: &arr,
		Item: "dependency",
		Body: &transform.Expr{Op: transform.OpAssertEnum, Actual: &item, Allowed: [][]byte{[]byte("\"a\""), []byte("\"b\"")}},
	}}
	source.Program.Output = arr
	for _, tc := range []struct {
		values []any
		reject bool
	}{
		{values: []any{"a", "b"}},
		{values: []any{}},
		{values: []any{"a", "invalid"}, reject: true},
	} {
		input := valueFromGoJSON(t, tc.values)
		got, err := transform.Evaluate(source, map[string]transform.Value{"dependencies": input})
		if tc.reject {
			if err == nil {
				t.Fatal("assert-each accepted invalid member")
			}
		} else {
			if err != nil || !transform.Equal(got.Output, input) {
				t.Fatalf("assert-each altered valid input: %v", err)
			}
		}
	}
	invalid := source
	invalid.Program.Assertions[0].Item = "dependencies"
	if err := invalid.Validate(); err == nil {
		t.Fatal("assert-each shadowing must be rejected")
	}
}
