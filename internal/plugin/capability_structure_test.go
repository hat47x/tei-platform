package plugin_test

import (
	"strings"
	"testing"

	"tei/internal/deployment"
	"tei/internal/environment"
	"tei/internal/permission"
	"tei/internal/plugin"
)

func TestDirectPackageRejectsInvalidCapabilityDeclarations(t *testing.T) {
	validPermission := permission.Spec{Kind: permission.KindFilesystem, Access: permission.AccessRead, Scope: "data"}
	tests := []struct {
		name   string
		change func(*plugin.Definition)
		reject string
	}{
		{name: "valid permissions", change: func(d *plugin.Definition) {
			d.Provides[0].Permissions = []permission.Spec{validPermission, {Kind: permission.KindNetwork, Access: permission.AccessConnect, Scope: "service"}}
		}},
		{name: "invalid capability kind", change: func(d *plugin.Definition) {
			d.Provides[0].Kind = plugin.CapabilityKind("missing")
		}, reject: "invalid kind"},
		{name: "invalid action", change: func(d *plugin.Definition) {
			d.Provides[0].Actions = []deployment.ActionKind{"missing"}
		}, reject: "invalid action"},
		{name: "invalid observation", change: func(d *plugin.Definition) {
			d.Provides[0].Observations = []deployment.ObservationKind{"missing"}
		}, reject: "invalid observation"},
		{name: "invalid evidence", change: func(d *plugin.Definition) {
			d.Provides[0].Evidence = []deployment.EvidenceKind{"missing"}
		}, reject: "invalid evidence"},
		{name: "generic database provider", change: func(d *plugin.Definition) {
			d.Provides[0].DatabaseProviders = []environment.DatabaseProvider{environment.ProviderGeneric}
		}, reject: "invalid database provider"},
		{name: "unknown permission kind", change: func(d *plugin.Definition) {
			d.Provides[0].Permissions = []permission.Spec{{Kind: permission.Kind("unknown"), Access: permission.AccessRead, Scope: "data"}}
		}, reject: "unknown permission kind"},
		{name: "invalid permission access", change: func(d *plugin.Definition) {
			d.Provides[0].Permissions = []permission.Spec{{Kind: permission.KindNetwork, Access: permission.AccessRead, Scope: "data"}}
		}, reject: "network permission access"},
		{name: "empty permission scope", change: func(d *plugin.Definition) {
			d.Provides[0].Permissions = []permission.Spec{{Kind: permission.KindNetwork, Access: permission.AccessConnect}}
		}, reject: "scope is required"},
		{name: "whitespace permission scope", change: func(d *plugin.Definition) {
			d.Provides[0].Permissions = []permission.Spec{{Kind: permission.KindNetwork, Access: permission.AccessConnect, Scope: " \t "}}
		}, reject: "scope is required"},
		{name: "duplicate permission identity", change: func(d *plugin.Definition) {
			d.Provides[0].Permissions = []permission.Spec{validPermission, validPermission}
		}, reject: "more than once"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			source := artifactFreePackageSource()
			tc.change(&source)
			_, err := plugin.MaterializePluginPackage(source, pluginPackagePlan(source))
			if tc.reject == "" && err != nil {
				t.Fatalf("valid definition was rejected: %v", err)
			}
			if tc.reject != "" && (err == nil || !strings.Contains(err.Error(), tc.reject)) {
				t.Fatalf("want error containing %q; got %v", tc.reject, err)
			}
		})
	}
}

func TestPermissionDuplicateDiagnosticIgnoresOrder(t *testing.T) {
	a := permission.Spec{Kind: permission.KindFilesystem, Access: permission.AccessRead, Scope: "a"}
	z := permission.Spec{Kind: permission.KindNetwork, Access: permission.AccessConnect, Scope: "z"}
	err1 := plugin.ValidateUniqueCapabilityPermissions([]permission.Spec{z, a, z, a})
	err2 := plugin.ValidateUniqueCapabilityPermissions([]permission.Spec{a, z, a, z})
	if err1 == nil || err2 == nil || err1.Error() != err2.Error() || !strings.Contains(err1.Error(), "a") {
		t.Fatalf("diagnostics vary by permutation: %v vs %v", err1, err2)
	}
}
