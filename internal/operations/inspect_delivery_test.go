package operations

import (
	"encoding/json"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// inspectReleaseFixture mirrors the shapes the delivery-layer kinds resolve. Each
// kind must satisfy ValidateInspectionProjection, which requires the Canonical
// object's own top-level id to equal the requested id (#4128).
func TestDeliveryLayerSubjectsAreInspectable(t *testing.T) {
	root := "../../reference/workspace"
	cases := []struct {
		kind string
		id   string
		// wantVersions is non-nil when the projection aggregates versions.
		wantVersions []string
	}{
		{kind: "migration", id: "20260826_001"},
		{kind: "application-release", id: "order_entry"},
		// The binding layer: a designer can declare a database and stand up an
		// environment, and both must be readable (#4128).
		{kind: "database", id: "sales_core"},
		{kind: "environment", id: "development"},
		// The persistence layer: a designer can declare that a semantic data
		// type lives in a table, and must be able to read the mapping back.
		{kind: "persistence-mapping", id: "sales_core.order"},
		{kind: "database-release", id: "sales_core", wantVersions: []string{"4.0.0", "4.1.0", "4.2.0"}},
		{kind: "solution-release", id: "order_rollout", wantVersions: []string{"2026.8.27"}},
	}
	for _, c := range cases {
		got, err := InspectSubject(root, sourceindex.Subject{Kind: c.kind, ID: c.id})
		if err != nil {
			t.Errorf("%s:%s did not resolve: %v", c.kind, c.id, err)
			continue
		}
		// The same contract the CLI and Studio apply before reporting success.
		if err := ValidateInspectionProjection(sourceindex.Subject{Kind: c.kind, ID: c.id}, got); err != nil {
			t.Errorf("%s:%s failed the projection check: %v", c.kind, c.id, err)
		}
		var canonical struct {
			ID       string `json:"id"`
			Releases []struct {
				Version string `json:"version"`
			} `json:"releases"`
		}
		if err := json.Unmarshal(got.Canonical, &canonical); err != nil {
			t.Errorf("%s:%s canonical is not an object: %v", c.kind, c.id, err)
			continue
		}
		if canonical.ID != c.id {
			t.Errorf("%s: canonical id is %q, want the requested %q", c.kind, canonical.ID, c.id)
		}
		if c.wantVersions == nil {
			continue
		}
		var versions []string
		for _, r := range canonical.Releases {
			versions = append(versions, r.Version)
		}
		if len(versions) != len(c.wantVersions) {
			t.Errorf("%s: got %d versions %v, want %v", c.kind, len(versions), versions, c.wantVersions)
			continue
		}
		for i := range versions {
			if versions[i] != c.wantVersions[i] {
				t.Errorf("%s: version %d is %q, want %q", c.kind, i, versions[i], c.wantVersions[i])
			}
		}
	}
}

// A release subject names its owner, so an owner with no release is not found
// rather than returning an empty set that reads like a success.
func TestUnreleasedOwnerIsNotFound(t *testing.T) {
	for _, kind := range []string{"database-release", "solution-release", "database", "environment", "persistence-mapping"} {
		_, err := InspectSubject("../../reference/workspace",
			sourceindex.Subject{Kind: kind, ID: "nope_owner"})
		inspection, ok := err.(*InspectionError)
		if !ok {
			t.Fatalf("%s: want an InspectionError, got %T (%v)", kind, err, err)
		}
		if inspection.Code != InspectionNotFound {
			t.Errorf("%s: code is %q, want %q", kind, inspection.Code, InspectionNotFound)
		}
	}
}

// Every advertised kind that the reference Workspace actually declares must
// resolve. An advertised kind with no case in the dispatch still answers
// "not found" forever, which reads like "you have not written one yet".
func TestEveryAdvertisedKindWithADeclarationResolves(t *testing.T) {
	loaded, err := workspace.Load("../../reference/workspace")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	declared := map[string][]string{}
	seen := map[string]bool{}
	for _, n := range loaded.World.Semantics {
		declared["semantic"] = append(declared["semantic"], string(n.ID))
		for _, x := range n.Types {
			declared["semantic-type"] = append(declared["semantic-type"], string(x.ID))
		}
		for _, x := range n.Data {
			declared["semantic-data"] = append(declared["semantic-data"], string(x.ID))
			for _, a := range x.Attributes {
				declared["semantic-attribute"] = append(declared["semantic-attribute"], string(a.ID))
			}
			for _, r := range x.Relations {
				declared["semantic-relation"] = append(declared["semantic-relation"], string(r.ID))
			}
		}
		for _, x := range n.Capabilities {
			declared["semantic-capability"] = append(declared["semantic-capability"], string(x.ID))
		}
		for _, x := range n.Validations {
			declared["semantic-validation"] = append(declared["semantic-validation"], string(x.ID))
		}
	}
	for _, a := range loaded.World.Applications {
		declared["application"] = append(declared["application"], string(a.ID))
		for _, x := range a.Patterns {
			declared["application-pattern"] = append(declared["application-pattern"], string(x.ID))
		}
	}
	for _, r := range loaded.World.ApplicationReleases {
		if id := string(r.Application); !seen["app:"+id] {
			declared["application-release"] = append(declared["application-release"], id)
			seen["app:"+id] = true
		}
	}
	for _, q := range loaded.World.Queries {
		declared["query"] = append(declared["query"], string(q.ID))
	}
	for _, m := range loaded.World.Migrations {
		declared["migration"] = append(declared["migration"], m.ID)
	}
	for _, r := range loaded.World.DatabaseReleases {
		if id := string(r.Database); !seen["db:"+id] {
			declared["database-release"] = append(declared["database-release"], id)
			seen["db:"+id] = true
		}
	}
	for _, r := range loaded.World.SolutionReleases {
		if id := string(r.ID); !seen["sol:"+id] {
			declared["solution-release"] = append(declared["solution-release"], id)
			seen["sol:"+id] = true
		}
	}

	for _, kind := range InspectionSubjectKinds {
		ids := declared[kind]
		if len(ids) == 0 {
			continue // a kind with no declaration in this fixture proves nothing
		}
		if _, err := InspectSubject("../../reference/workspace",
			sourceindex.Subject{Kind: kind, ID: ids[0]}); err != nil {
			t.Errorf("advertised kind %q has a declaration %q but does not resolve: %v",
				kind, ids[0], err)
		}
	}
}

// Every advertised kind must be accepted, so the help text and the behaviour
// cannot drift apart.
func TestEveryAdvertisedInspectionKindIsAccepted(t *testing.T) {
	for _, kind := range InspectionSubjectKinds {
		if !supportedInspectionKind(kind) {
			t.Errorf("InspectionSubjectKinds advertises %q but it is rejected", kind)
		}
	}
	for _, kind := range []string{"migration", "database-release", "solution-release"} {
		if !supportedInspectionKind(kind) {
			t.Errorf("%q is not accepted", kind)
		}
	}
}
