package providerroutes

import (
	"reflect"
	"testing"
)

func order(alias, provider, endpoint, mapping, purpose string, read, write bool) Route {
	return Route{Alias: alias, SemanticData: "sales.order", Scope: "tenant-A", Provider: provider,
		Endpoint: endpoint, Mapping: mapping, MappingRevision: "v1", Purpose: purpose, CanRead: read, CanWrite: write}
}

func TestPR01SamePhysicalPathMultipleAliases(t *testing.T) {
	a := order("sales", "pg", "db-primary", "order-mapping", "current", true, true)
	b := a
	b.Alias = "sales_ro"
	b.CanWrite = false
	r := Explain([]Route{a, b}, "sales.order", "tenant-A", "read")
	if r.CandidateCount != 1 || !reflect.DeepEqual(r.Paths[0].Aliases, []string{"sales", "sales_ro"}) {
		t.Fatalf("same exact route should have both aliases: %#v", r)
	}
}
func TestPR02SameSemanticDataTwoDifferentProviders(t *testing.T) {
	a := order("archive", "pg", "db-archive", "order-archive", "archive", true, false)
	b := order("primary", "pg", "db-primary", "order-live", "current", true, true)
	r := Explain([]Route{a, b}, "sales.order", "tenant-A", "read")
	if r.CandidateCount != 2 || r.Paths[0].Endpoint == r.Paths[1].Endpoint {
		t.Fatalf("two endpoints must remain visible: %#v", r)
	}
}
func TestPR03ReplicaAndPrimaryAreNotIdentical(t *testing.T) {
	a := order("replica", "pg", "db-replica", "orders", "replica", true, false)
	b := order("primary", "pg", "db-primary", "orders", "current", true, true)
	rr := Explain([]Route{a, b}, "sales.order", "tenant-A", "read")
	wr := Explain([]Route{a, b}, "sales.order", "tenant-A", "write")
	if rr.CandidateCount != 2 || wr.CandidateCount != 1 || wr.Paths[0].Aliases[0] != "primary" {
		t.Fatalf("read and write qualification differ: rr=%#v wr=%#v", rr, wr)
	}
}
func TestPR04AliasRenameAndOrderingCannotChoosePath(t *testing.T) {
	a := order("archive", "pg", "db-archive", "old", "archive", true, false)
	b := order("primary", "pg", "db-primary", "new", "current", true, true)
	first := Explain([]Route{a, b}, "sales.order", "tenant-A", "read")
	a.Alias = "z_archive"
	b.Alias = "a_primary"
	second := Explain([]Route{b, a}, "sales.order", "tenant-A", "read")
	if first.CandidateCount != 2 || second.CandidateCount != 2 || first.Paths[0].Endpoint != second.Paths[0].Endpoint || first.Paths[1].Endpoint != second.Paths[1].Endpoint {
		t.Fatalf("alias order changed distinct physical candidates: %#v %#v", first, second)
	}
}
func TestPR05MappingRevisionKeepsTwoCandidates(t *testing.T) {
	a := order("m1", "pg", "db-primary", "orders", "current", true, true)
	b := a
	b.Alias = "m2"
	b.MappingRevision = "v2"
	r := Explain([]Route{a, b}, "sales.order", "tenant-A", "read")
	if r.CandidateCount != 2 {
		t.Fatalf("mapping revisions must not be silently coalesced: %#v", r)
	}
}
func TestPR06ScopeQualification(t *testing.T) {
	a := order("tenant-a", "pg", "db-shared", "orders", "current", true, true)
	b := a
	b.Alias = "tenant-b"
	b.Scope = "tenant-B"
	r := Explain([]Route{a, b}, "sales.order", "tenant-A", "read")
	if r.CandidateCount != 1 || r.Paths[0].Aliases[0] != "tenant-a" {
		t.Fatalf("scope must not be erased: %#v", r)
	}
}
func TestPR07LocalReadRestrictionDoesNotSuppressAnotherRoute(t *testing.T) {
	a := order("writeonly", "pg", "db-primary", "orders", "current", false, true)
	b := a
	b.Alias = "report"
	b.CanRead = true
	b.CanWrite = false
	r := Explain([]Route{a, b}, "sales.order", "tenant-A", "read")
	if r.CandidateCount != 1 || !reflect.DeepEqual(r.ExcludedAlias, []string{"writeonly"}) {
		t.Fatalf("local gaps and aggregate candidates coexist: %#v", r)
	}
}
func TestPR08UnknownMappingNotAQualifiedPath(t *testing.T) {
	a := order("unresolved", "pg", "db-primary", "", "current", true, true)
	r := Explain([]Route{a}, "sales.order", "tenant-A", "read")
	if r.CandidateCount != 0 || !reflect.DeepEqual(r.InvalidAliases, []string{"unresolved"}) {
		t.Fatalf("missing metadata must remain unresolved: %#v", r)
	}
}
func TestPR09DataProviderCanBeAnAPI(t *testing.T) {
	a := order("odata", "odata", "service/orders", "odata-orders", "current", true, true)
	b := order("pg", "pg", "db-primary", "orders", "current", true, true)
	r := Explain([]Route{a, b}, "sales.order", "tenant-A", "read")
	if r.CandidateCount != 2 {
		t.Fatalf("API and DB providers must stay distinct: %#v", r)
	}
}
func TestPR10SameProviderEndpointDifferentPurpose(t *testing.T) {
	a := order("historical", "pg", "db-primary", "orders", "history", true, false)
	b := a
	b.Alias = "current"
	b.Purpose = "current"
	r := Explain([]Route{a, b}, "sales.order", "tenant-A", "read")
	if r.CandidateCount != 2 {
		t.Fatalf("purpose is not a proof of logical equivalence: %#v", r)
	}
}
func TestPR11NoRouteIsNotDataNonexistence(t *testing.T) {
	r := Explain([]Route{}, "sales.order", "tenant-A", "read")
	if r.CandidateCount != 0 || len(r.ExcludedAlias) != 0 {
		t.Fatalf("no declaration must stay a diagnostic fact: %#v", r)
	}
}
func TestPR12DeterministicWithDifferentInputOrder(t *testing.T) {
	a := order("archive", "pg", "db-archive", "old", "archive", true, false)
	b := order("primary", "pg", "db-primary", "new", "current", true, true)
	x := Explain([]Route{a, b}, "sales.order", "tenant-A", "read")
	y := Explain([]Route{b, a}, "sales.order", "tenant-A", "read")
	if !reflect.DeepEqual(x, y) {
		t.Fatalf("candidate listing must not depend on input order: %#v %#v", x, y)
	}
}
