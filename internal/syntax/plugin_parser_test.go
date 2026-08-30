package syntax_test

import (
	"testing"

	"tei/internal/plugin"
	"tei/internal/syntax"
)

func TestParsePluginAndBundle(t *testing.T) {
	p, err := syntax.NewParser(`plugin tei.migration.atlas v1.2.0 {
        runtime builtin
        provides deployment-executor migration.atlas { action database-release }
        provides migration-provider migration.atlas { tool atlas }
        external migration tool atlas
    }`).ParsePlugin()
	if err != nil {
		t.Fatal(err)
	}
	if p.ID != "tei.migration.atlas" || p.Version != "1.2.0" || p.Runtime.Kind != plugin.RuntimeBuiltin || len(p.Provides) != 2 {
		t.Fatalf("plugin: %#v", p)
	}
	b, err := syntax.NewParser(`bundle tei.standard v1.0.0 { plugin tei.migration.atlas ^1 }`).ParseBundle()
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Plugins) != 1 || b.Plugins[0].CompatibleMajor != 1 {
		t.Fatalf("bundle: %#v", b)
	}
}

func TestParsePluginSandboxArtifactPermissionAndEnvironmentGrant(t *testing.T) {
	p, err := syntax.NewParser(`plugin tei.editor.secure v1.0.0 {
        runtime process "python3 -m worker" sandbox required
        artifact {
            digest "sha256:5af655bd19fffee7c3564b060f92056e1a6bd521c0b15bd1155dd244cd77de43"
            signature "sigstore://test"
            provenance "source://test"
        }
        provides editor-intelligence editor.secure {
            feature editor.sql_analysis
            permission filesystem read workspace-source
        }
    }`).ParsePlugin()
	if err != nil {
		t.Fatal(err)
	}
	if p.Runtime.Sandbox != plugin.SandboxRequired || p.Artifact.Digest == "" || len(p.Provides[0].Permissions) != 1 {
		t.Fatalf("plugin: %#v", p)
	}

	e, err := syntax.NewParser(`environment dev {
        plugin tei.editor.secure ^1
        feature editor.sql_analysis
        resource filesystem workspace-source -> dev.workspace_source
        grant editor-intelligence editor.secure permission filesystem read workspace-source
    }`).ParseEnvironment()
	if err != nil {
		t.Fatal(err)
	}
	if len(e.PermissionGrants) != 1 || e.PermissionGrants[0].CapabilityID != "editor.secure" || len(e.ResourceBindings) != 1 || e.ResourceBindings[0].ResourceID != "dev.workspace_source" {
		t.Fatalf("environment: %#v", e)
	}
}

func TestParseSecretInjectionBinding(t *testing.T) {
	e, err := syntax.NewParser(`environment prod {
        resource secret database-credential -> prod.db.secret inject environment "TEI_DATABASE_CREDENTIAL"
    }`).ParseEnvironment()
	if err != nil {
		t.Fatal(err)
	}
	if len(e.ResourceBindings) != 1 || e.ResourceBindings[0].SecretDelivery != "environment" || e.ResourceBindings[0].SecretTarget != "TEI_DATABASE_CREDENTIAL" {
		t.Fatalf("binding: %#v", e.ResourceBindings)
	}
}

func TestParseRefreshBindingAndLeasePolicy(t *testing.T) {
	e, err := syntax.NewParser(`environment production {
        resource secret database-credential -> production.db.secret inject environment "TEI_DATABASE_CREDENTIAL" refresh live
        lease-policy deployment-executor migration.atlas max-lifetime 45m renew-before 5m revoke-timeout 15s renewal-failure fail-closed
    }`).ParseEnvironment()
	if err != nil {
		t.Fatal(err)
	}
	if len(e.ResourceBindings) != 1 || e.ResourceBindings[0].Refresh != "live" {
		t.Fatalf("binding: %#v", e.ResourceBindings)
	}
	if len(e.LeasePolicies) != 1 {
		t.Fatalf("lease policies: %#v", e.LeasePolicies)
	}
	p := e.LeasePolicies[0].Policy
	if p.MaxLifetime != "45m" || p.RenewBefore != "5m" || p.RevokeTimeout != "15s" || p.RenewalFailure != "fail-closed" {
		t.Fatalf("policy: %#v", p)
	}
}
