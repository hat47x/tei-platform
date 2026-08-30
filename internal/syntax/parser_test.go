package syntax_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/syntax"
)

func TestSQLRawBlockPreservesNestedBracesAndComments(t *testing.T) {
	src := `query app.q {
      datasource sales
      sql postgres {
        SELECT '{"a": 1}'::jsonb -- } ignored
        /* { ignored } */
      }
    }`
	q, err := syntax.NewParser(src).ParseQuery(core.MustCanonicalID("app"))
	if err != nil {
		t.Fatal(err)
	}
	if q.Dialect != "postgres" {
		t.Fatalf("dialect=%s", q.Dialect)
	}
	if q.Source == "" {
		t.Fatal("empty SQL source")
	}
}

func TestApplicationParsesMultipleDatabaseRequirements(t *testing.T) {
	src := `application app {
      database a { target db_a access read requires { contract db_a.c ^1 } }
      database b { target db_b access read-write requires { contract db_b.c ^2 } }
    }`
	a, err := syntax.NewParser(src).ParseApplication()
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Databases) != 2 {
		t.Fatalf("got %d database requirements", len(a.Databases))
	}
}

func TestSemanticFileDerivesStableCanonicalShape(t *testing.T) {
	src := `namespace sales

    data Order {
      attribute orderId type std.identifier
      attribute status type std.code
      relation customer to customer.customer one
    }

    capability sales.order.submit {
      input order sales.order
      output sales.order
    }`
	ns, err := syntax.NewParser(src).ParseSemanticFile()
	if err != nil {
		t.Fatal(err)
	}
	if ns.ID != "sales" || len(ns.Data) != 1 || ns.Data[0].ID != "sales.order" {
		t.Fatalf("unexpected namespace/data: %#v", ns)
	}
	if got := ns.Data[0].Attributes[1].ID; got != "sales.order.status" {
		t.Fatalf("attribute id=%s", got)
	}
	if got := ns.Data[0].Relations[0].ID; got != "sales.order.customer" {
		t.Fatalf("relation id=%s", got)
	}
	if got := ns.Capabilities[0].ID; got != "sales.order.submit" {
		t.Fatalf("capability id=%s", got)
	}
}

func TestSemanticTypeAliasParses(t *testing.T) {
	src := `namespace sales

    type OrderId from std.identifier
    data Order { attribute orderId type sales.order_id }`
	ns, err := syntax.NewParser(src).ParseSemanticFile()
	if err != nil {
		t.Fatal(err)
	}
	if len(ns.Types) != 1 || ns.Types[0].ID != "sales.order_id" || ns.Types[0].Underlying != "std.identifier" {
		t.Fatalf("unexpected type: %#v", ns.Types)
	}
}

func TestEnvironmentFeaturesParse(t *testing.T) {
	src := `environment development {
      feature editor.sql_analysis
      database-instance db.dev { definition db provider postgres }
      deployment-executor database-release for database db -> migration.atlas
      observation-provider database-state -> database.observer
      evidence-provider rollout-health -> health.reference
    }`
	e, err := syntax.NewParser(src).ParseEnvironment()
	if err != nil {
		t.Fatal(err)
	}
	if len(e.Features) != 1 || e.Features[0] != "editor.sql_analysis" {
		t.Fatalf("features: %#v", e.Features)
	}
	if len(e.DeploymentExecutors) != 1 || e.DeploymentExecutors[0].Executor != "migration.atlas" {
		t.Fatalf("deployment executors: %#v", e.DeploymentExecutors)
	}
	if len(e.ObservationProviders) != 1 || e.ObservationProviders[0].Provider != "database.observer" {
		t.Fatalf("observation providers: %#v", e.ObservationProviders)
	}
	if len(e.EvidenceProviders) != 1 || e.EvidenceProviders[0].Provider != "health.reference" {
		t.Fatalf("evidence providers: %#v", e.EvidenceProviders)
	}
}

func TestParseDatabaseRelease(t *testing.T) {
	p := syntax.NewParser(`database-release sales_core 4.2.0 {
        schema-fingerprint "fp-42"
        includes migration 20260826_001
        provides contract sales_core.order v4.2
        data-readiness ready
    }`)
	r, err := p.ParseDatabaseRelease()
	if err != nil {
		t.Fatal(err)
	}
	if r.Database != core.MustCanonicalID("sales_core") || r.Version != "4.2.0" || r.SchemaFingerprint != "fp-42" {
		t.Fatalf("release=%#v", r)
	}
	if len(r.Migrations) != 1 || len(r.Provides) != 1 || r.Provides[0].Version.Major != 4 || r.Provides[0].Version.Minor != 2 {
		t.Fatalf("release=%#v", r)
	}
}
