package operations

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/database/persistence"
	"tei/internal/database/providers/postgres"
	"tei/internal/semantic"
	"tei/internal/validation"
	"tei/internal/workspace"
)

// These tests characterize the current product gap, not the accepted shape of
// a future Organization profile. When a Person link or version storage is
// implemented, replace the corresponding gap assertion with its positive,
// independently reviewed conformance test instead of preserving the gap.
func TestOrgC1CurrentMembershipDoesNotHavePersonLink(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "org-staged-axis")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load real org-staged-axis workspace: %v", err)
	}

	var membership semantic.Data
	foundMembership := false
	personExists := false
	otherPersonLinks := map[string]bool{}
	for _, ns := range loaded.World.Semantics {
		for _, data := range ns.Data {
			switch string(data.ID) {
			case "organization.person":
				personExists = true
			case "organization.membership":
				membership = data
				foundMembership = true
			case "organization.employment", "organization.affiliation":
				for _, rel := range data.Relations {
					if rel.Target == "organization.person" {
						otherPersonLinks[string(data.ID)] = true
					}
				}
			}
		}
	}
	if !foundMembership || !personExists {
		t.Fatalf("expected Membership and Person in the loaded world: membership=%v person=%v", foundMembership, personExists)
	}
	if !otherPersonLinks["organization.employment"] || !otherPersonLinks["organization.affiliation"] {
		t.Fatalf("positive control: Employment and Affiliation must explicitly link Person: %+v", otherPersonLinks)
	}
	for _, attr := range membership.Attributes {
		if attr.ID == "organization.membership.person_id" {
			t.Fatal("the direct Person attribute is now present; replace this gap characterization with positive conformance")
		}
	}
	for _, rel := range membership.Relations {
		if rel.Target == "organization.person" || rel.Target == "organization.employment" || rel.Target == "organization.affiliation" {
			t.Fatalf("Membership now has a potential Person path; qualify its keys before claiming the gap: %+v", rel)
		}
	}
	// A Person link on another Data does not imply an executable Membership
	// relation traversal, much less a current Principal or Host Grant.
	t.Log("GAP CONFIRMED: real typed Membership has no Person path; other Data's links cannot be inferred")
}

func TestOrgC1CurrentPersistenceCannotCoStoreTwoVersionsByMembershipPK(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "org-staged-axis")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load real org-staged-axis workspace: %v", err)
	}
	found := false
	for _, mapping := range loaded.World.Persistence {
		if mapping.SemanticData != "organization.membership" {
			continue
		}
		found = true
		if mapping.Table != "public.org_memberships" {
			t.Fatalf("unexpected Membership table; review version storage evidence: %q", mapping.Table)
		}
		if len(mapping.PrimaryKey) != 1 || mapping.PrimaryKey[0] != "organization.membership.membership_id" {
			t.Fatalf("version/physical key scheme changed; replace this gap test with version co-storage conformance: %+v", mapping.PrimaryKey)
		}
		for _, column := range mapping.Columns {
			if column.Attribute == "organization.membership.person_id" {
				t.Fatal("Person column now exists; update the Product Read qualification test")
			}
		}
		// RelationStorage can carry database-owned support columns without a
		// Person attribute on Membership. Do not assume only Columns matter.
		if len(mapping.RelationStorage) != 0 {
			t.Fatalf("relation storage is now present; qualify its source relation and physical binding: %+v", mapping.RelationStorage)
		}
	}
	if !found {
		t.Fatal("actual Membership persistence mapping was not loaded")
	}
	// With membership_id alone as PK, A2-old and A2-new from r7 cannot
	// simultaneously occupy distinct rows in this one physical table.
	t.Log("GAP CONFIRMED: Membership is stored under an occurrence-only physical PK")
}

// This is an isolated IR capability check, not an accepted Person-Membership
// business link, DSL parsing proof, actual SQL mapping, or Provider call.
// A typed Relation may use a database-owned support column without introducing
// organization.membership.person_id as a Semantic attribute.
func TestOrgC1RelationStorageIRCanRepresentSupportColumnWithoutPersonAttribute(t *testing.T) {
	candidate := persistence.Mapping{
		ID:           "org_store.membership_person_candidate",
		Database:     "org_store",
		SemanticData: "organization.membership",
		Table:        "public.org_memberships_candidate",
		Columns: []persistence.ColumnMapping{
			{Attribute: "organization.membership.membership_id", Column: "membership_id"},
		},
		RelationStorage: []persistence.RelationStorageMapping{
			{
				Relation: "organization.person.memberships_candidate",
				Components: []persistence.RelationStorageComponentMapping{
					{
						SourceAttribute: "organization.person.person_id",
						Column:          "person_support_id",
					},
				},
			},
		},
	}
	raw, err := json.Marshal(candidate)
	if err != nil {
		t.Fatalf("marshal candidate relation storage: %v", err)
	}
	var restored persistence.Mapping
	if err := json.Unmarshal(raw, &restored); err != nil {
		t.Fatalf("unmarshal candidate relation storage: %v", err)
	}
	if len(restored.Columns) != 1 ||
		restored.Columns[0].Attribute != "organization.membership.membership_id" {
		t.Fatalf("candidate must not add an implicit Membership Person attribute: %+v", restored.Columns)
	}
	if len(restored.RelationStorage) != 1 ||
		restored.RelationStorage[0].Relation != "organization.person.memberships_candidate" ||
		len(restored.RelationStorage[0].Components) != 1 ||
		restored.RelationStorage[0].Components[0].SourceAttribute != "organization.person.person_id" ||
		restored.RelationStorage[0].Components[0].Column != "person_support_id" {
		t.Fatalf("candidate relation support identity was not preserved: %+v", restored.RelationStorage)
	}
}

// This pressure test uses the REAL organization Workspace and the existing
// production validation/desired-schema code. It mutates only a local loaded
// World, never the accepted .tei Source, database, or Owner meaning.
// The new Person -> Membership keyless relation is just a candidate, not a
// commitment to Person as the only possible member type.
func TestOrgC1CandidatePersonRelationStorageValidatesAndRealizesOffline(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "org-staged-axis")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load org-staged-axis: %v", err)
	}
	world := loaded.World
	personNamespace, personData, mappingIndex := -1, -1, -1
	for i := range world.Semantics {
		for j := range world.Semantics[i].Data {
			if world.Semantics[i].Data[j].ID == "organization.person" {
				personNamespace, personData = i, j
			}
		}
	}
	for i := range world.Persistence {
		if world.Persistence[i].SemanticData == "organization.membership" {
			mappingIndex = i
		}
	}
	if personNamespace < 0 || mappingIndex < 0 {
		t.Fatalf("person/membership mapping missing: namespace=%d mapping=%d", personNamespace, mappingIndex)
	}

	relation := semantic.Relation{
		ID:          "organization.person.memberships_candidate",
		Symbol:      "membershipsCandidate",
		Target:      "organization.membership",
		Cardinality: semantic.CardinalityMany,
	}
	person := &world.Semantics[personNamespace].Data[personData]
	person.Relations = append(person.Relations, relation)
	mapping := &world.Persistence[mappingIndex]
	mapping.RelationStorage = append(mapping.RelationStorage, persistence.RelationStorageMapping{
		Relation: relation.ID,
		Components: []persistence.RelationStorageComponentMapping{{
			SourceAttribute: "organization.person.person_id",
			Column:          "person_support_id",
			Storage:         persistence.StorageVarchar,
			Nullability:     persistence.Required,
		}},
	})

	if diagnostics := validation.ValidatePersistenceRelationStorageMappings(world); len(diagnostics) != 0 {
		t.Fatalf("candidate source->target relation storage rejected by real validator: %+v", diagnostics)
	}
	desired, err := persistence.RealizeDesired(
		[]persistence.Mapping{*mapping}, "org_store", postgres.RealizationPolicy{},
	)
	if err != nil {
		t.Fatalf("real offline PostgreSQL schema realization rejected the candidate: %v", err)
	}
	table, ok := desired.Table("public.org_memberships")
	if !ok {
		t.Fatalf("candidate mapping did not realize org_memberships table: %+v", desired.Tables)
	}
	support, ok := table.Column("person_support_id")
	if !ok || support.Nullable || support.Ordinal != len(mapping.Columns)+1 {
		t.Fatalf("Relation Storage support column not realized as a required physical column: %+v", table.Columns)
	}
	if len(table.PrimaryKey) != 1 || table.PrimaryKey[0] != "membership_id" {
		t.Fatalf("candidate must not silently change history PK: %+v", table.PrimaryKey)
	}
	if len(table.ForeignKeys) != 0 {
		t.Fatalf("candidate must not infer a physical FK from business Relation: %+v", table.ForeignKeys)
	}
	for _, column := range mapping.Columns {
		if column.Attribute == "organization.membership.person_id" {
			t.Fatal("candidate incorrectly added a Person attribute to Membership")
		}
	}

	// Counterexample: source component actually belongs to Membership, not
	// the Person data that declares this Relation. The existing validator
	// must reject the misdirected physical join.
	mapping.RelationStorage[0].Components[0].SourceAttribute = "organization.membership.membership_id"
	diagnostics := validation.ValidatePersistenceRelationStorageMappings(world)
	found := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "PERSISTENCE_RELATION_STORAGE_SOURCE_ATTRIBUTE_PARENT_MISMATCH" {
			found = true
		}
	}
	if !found {
		t.Fatalf("source/target attribute confusion was not rejected: %+v", diagnostics)
	}

	// This validates only the *hypothetical* IR and desired schema. It does
	// not load a Person-Membership .tei relation-storage authoring file, read
	// actual PostgreSQL rows, ensure bitemporal co-storage, or accept meaning.
}

func TestOrgC1CurrentAsOfQueryDoesNotYieldPersonIdentity(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "org-staged-axis")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load real org-staged-axis workspace: %v", err)
	}
	found := false
	for _, query := range loaded.World.Queries {
		if query.ID != "org_axes_view.as_of_membership" {
			continue
		}
		found = true
		if len(query.ResultColumns) != 3 {
			t.Fatalf("query result contract changed; review Person/version provenance: %+v", query.ResultColumns)
		}
		for _, result := range query.ResultColumns {
			if result.Attribute == "organization.membership.person_id" {
				t.Fatal("Person projection now exists; replace this negative test with full read provenance conformance")
			}
		}
	}
	if !found {
		t.Fatal("real org_axes_view.as_of_membership query not loaded")
	}
	// Membership row count alone cannot prove distinct Person count or
	// which known-time record version produced a row.
	t.Log("GAP CONFIRMED: actual as-of result does not identify a Person or historical record version")
}

func TestOrgC1SyntheticOracleHasTwoA2VersionsAndSharedP2(t *testing.T) {
	path := filepath.Join("..", "..", "reference", "org-semantic-views", "source-fixture.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read existing independent r7 fixture: %v", err)
	}
	var fixture struct {
		SourceRevision string   `json:"sourceRevision"`
		Persons        []string `json:"persons"`
		Versions       []struct {
			ID        string  `json:"id"`
			Person    string  `json:"person"`
			KnownFrom string  `json:"knownFrom"`
			KnownTo   *string `json:"knownTo"`
		} `json:"membershipVersions"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decode existing r7 source: %v", err)
	}
	if fixture.SourceRevision != "org-synthetic-r7" || len(fixture.Persons) != 3 || len(fixture.Versions) != 5 {
		t.Fatalf("fixture changed; qualify against the independent r7 Expected: revision=%q persons=%d versions=%d",
			fixture.SourceRevision, len(fixture.Persons), len(fixture.Versions))
	}
	var a2Old, a2New, a3 bool
	for _, v := range fixture.Versions {
		switch {
		case v.ID == "A2" && v.Person == "P2" && v.KnownFrom == "2027-04-01" && v.KnownTo != nil && *v.KnownTo == "2027-07-02":
			a2Old = true
		case v.ID == "A2" && v.Person == "P2" && v.KnownFrom == "2027-07-02" && v.KnownTo == nil:
			a2New = true
		case v.ID == "A3" && v.Person == "P2":
			a3 = true
		}
	}
	if !a2Old || !a2New || !a3 {
		t.Fatalf("independent r7 counterexample no longer has A2 two known versions and A3 sharing P2: old=%v new=%v a3=%v",
			a2Old, a2New, a3)
	}
	// This is Source/Expected evidence, not an actual TEI Query, DB version
	// co-storage, an observed complete Person lookup, or Owner acceptance.
}
