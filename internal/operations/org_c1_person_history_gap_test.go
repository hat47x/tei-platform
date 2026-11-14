package operations

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/database/persistence"
	"tei/internal/database/providers/postgres"
	"tei/internal/database/schemadiff"
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
			// Check incoming relations as well: a Person -> Membership
			// authoring would close the gap without adding personId to Membership.
			for _, rel := range data.Relations {
				if data.ID != "organization.membership" && rel.Target == "organization.membership" {
					t.Fatalf("incoming Membership relation is now present; qualify its business direction/key: source=%s relation=%s", data.ID, rel.ID)
				}
			}
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

// The candidate exists only in an isolated copy of the real org-staged-axis
// Workspace. This exercises the *actual .tei authoring* parser, pinned loader,
// source provenance, validator, and offline Postgres realization, not just
// a Go struct or JSON roundtrip. It is not an Owner-accepted model.
func TestOrgC1CandidatePersonRelationStorageLoadsFromTEIDSL(t *testing.T) {
	source := filepath.Join("..", "..", "reference", "org-staged-axis")
	root := t.TempDir()
	if err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		target := filepath.Join(root, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("non-regular workspace source %s", path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	}); err != nil {
		t.Fatalf("copy real organization Workspace to isolated test root: %v", err)
	}

	semanticPath := filepath.Join(root, "semantic", "organization-axes", "model.tei")
	original, err := os.ReadFile(semanticPath)
	if err != nil {
		t.Fatalf("load copied Person source: %v", err)
	}
	before := "data Person {\n    attribute personId type organization.person_id\n}\n"
	after := "data Person {\n    attribute personId type organization.person_id\n\n    relation membershipsCandidate to organization.membership many\n}\n"
	if strings.Count(string(original), before) != 1 {
		t.Fatal("Person source changed; review temporary candidate insertion against current Semantic")
	}
	if err := os.WriteFile(semanticPath, []byte(strings.Replace(string(original), before, after, 1)), 0o644); err != nil {
		t.Fatalf("write temporary Person relation authoring: %v", err)
	}
	relativeStorage := filepath.Join("databases", "org-store", "relation-storage", "person-membership.tei")
	storagePath := filepath.Join(root, relativeStorage)
	if err := os.MkdirAll(filepath.Dir(storagePath), 0o755); err != nil {
		t.Fatal(err)
	}
	// Nullable is a test hypothesis: unknown historical persons cannot be
	// silently backfilled or turned into required rows by adding this Source.
	storageSource := "relation-storage org_store.membership {\n" +
		"    relation organization.person.memberships_candidate\n" +
		"    component organization.person.person_id -> person_support_id storage varchar optional\n" +
		"}\n"
	if err := os.WriteFile(storagePath, []byte(storageSource), 0o644); err != nil {
		t.Fatalf("write temporary Relation Storage DSL: %v", err)
	}

	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("actual TEI DSL Parser/Workspace Loader rejected candidate authoring: %v", err)
	}
	const relationID = "organization.person.memberships_candidate"
	var membership *persistence.Mapping
	seenPersonRelation := false
	for _, ns := range loaded.World.Semantics {
		for _, data := range ns.Data {
			if data.ID == "organization.person" {
				for _, relation := range data.Relations {
					if relation.ID == relationID && relation.Target == "organization.membership" &&
						relation.Cardinality == semantic.CardinalityMany && len(relation.Keys) == 0 {
						seenPersonRelation = true
					}
				}
			}
			if data.ID == "organization.membership" {
				for _, attribute := range data.Attributes {
					if attribute.ID == "organization.membership.person_id" {
						t.Fatal("temporary relation changed Membership Semantic attributes")
					}
				}
			}
		}
	}
	for i := range loaded.World.Persistence {
		if loaded.World.Persistence[i].ID == "org_store.membership" {
			membership = &loaded.World.Persistence[i]
		}
	}
	if !seenPersonRelation || membership == nil {
		t.Fatalf("temporary DSL Relation/target mapping missing: relation=%v mapping=%v", seenPersonRelation, membership != nil)
	}
	relationStorage, ok := membership.RelationStorageByRelation(relationID)
	if !ok || len(relationStorage.Components) != 1 {
		t.Fatalf("temporary Relation Storage not loaded into explicit mapping: %+v", membership.RelationStorage)
	}
	component := relationStorage.Components[0]
	if component.SourceAttribute != "organization.person.person_id" ||
		component.Column != "person_support_id" ||
		component.Storage != persistence.StorageVarchar ||
		component.Nullability != persistence.Optional {
		t.Fatalf("source/physical contract drifted during real DSL load: %+v", component)
	}
	sourceKey := "persistence-relation-storage:org_store.membership/" + relationID
	if loc, ok := loaded.Sources[sourceKey]; !ok ||
		loc.File != filepath.ToSlash(relativeStorage) {
		t.Fatalf("actual Source Index lost Relation Storage provenance: key=%q location=%+v present=%v", sourceKey, loc, ok)
	}
	if diagnostics := validation.ValidatePersistenceRelationStorageMappings(loaded.World); len(diagnostics) != 0 {
		t.Fatalf("candidate Relation Storage is not world-valid: %+v", diagnostics)
	}
	desired, err := persistence.RealizeDesired(
		[]persistence.Mapping{*membership}, "org_store", postgres.RealizationPolicy{},
	)
	if err != nil {
		t.Fatalf("offline desired schema with parsed Relation Storage: %v", err)
	}
	table, ok := desired.Table("public.org_memberships")
	if !ok {
		t.Fatalf("parsed candidate lost org_memberships: %+v", desired.Tables)
	}
	col, ok := table.Column("person_support_id")
	if !ok || !col.Nullable || col.Ordinal != len(membership.Columns)+1 {
		t.Fatalf("parsed optional support column not realized independently: %+v", table.Columns)
	}
	if len(table.PrimaryKey) != 1 || table.PrimaryKey[0] != "membership_id" ||
		len(table.ForeignKeys) != 0 {
		t.Fatalf("candidate must not silently resolve historical PK or generate FK: %+v", table)
	}
	// Compare against the same Source-derived Mapping without the candidate
	// support column. This is an OFFLINE baseline, not an observed live DB.
	// A safe Expand candidate adds exactly one nullable column; neither the
	// unchanged occurrence-only PK nor old rows gain invented Person history.
	baselineMapping := *membership
	baselineMapping.RelationStorage = nil
	physicalBefore, err := persistence.RealizeDesired(
		[]persistence.Mapping{baselineMapping}, "org_store", postgres.RealizationPolicy{},
	)
	if err != nil {
		t.Fatalf("realize the existing Source-derived physical baseline: %v", err)
	}
	changeSet, err := schemadiff.Compare(desired, physicalBefore)
	if err != nil {
		t.Fatalf("diff candidate vs. previous offline physical shape: %v", err)
	}
	if len(changeSet.Changes) != 1 {
		t.Fatalf("optional Person link must only add one support column: %+v", changeSet.Changes)
	}
	change := changeSet.Changes[0]
	if change.Kind != schemadiff.AddColumn || change.Table != "public.org_memberships" ||
		change.Column == nil || change.Column.Name != "person_support_id" ||
		change.Column.PhysicalType != "pg_catalog.varchar" || !change.Column.Nullable {
		t.Fatalf("offline schema diff must add only a nullable varchar support column: %+v", change)
	}
	statements, err := postgres.RenderMigrationSQL(changeSet)
	if err != nil {
		t.Fatalf("render offline PostgreSQL Expand candidate: %v", err)
	}
	const wantExpand = `ALTER TABLE "public"."org_memberships" ADD COLUMN "person_support_id" pg_catalog.varchar;`
	if len(statements) != 1 || statements[0] != wantExpand {
		t.Fatalf("expected non-destructive nullable support-column Expand, got: %+v", statements)
	}
	// This generated DDL has NOT been run or compared to introspected Actual.
	// Known-time record versioning and safe backfill remain separate decisions.
	// The normal authoring/loader path must reject physically plausible but
	// semantically invalid variants. These are actual .tei sources in the
	// isolated copy, not in-memory-only mutations or fabricated parser outcomes.
	for _, tc := range []struct {
		name          string
		source        string
		wantCode      string
		wantLoadError string
	}{
		{
			name:     "source attribute belongs to Membership instead of Person",
			source:   strings.Replace(storageSource, "organization.person.person_id ->", "organization.membership.membership_id ->", 1),
			wantCode: "PERSISTENCE_RELATION_STORAGE_SOURCE_ATTRIBUTE_PARENT_MISMATCH",
		},
		{
			name:     "relation target is Person instead of Membership",
			source:   strings.Replace(storageSource, relationID, "organization.employment.person", 1),
			wantCode: "PERSISTENCE_RELATION_STORAGE_TARGET_MISMATCH",
		},
		{
			name:     "unknown semantic relation",
			source:   strings.Replace(storageSource, relationID, "organization.person.unknown", 1),
			wantCode: "UNKNOWN_PERSISTENCE_RELATION_STORAGE_RELATION",
		},
		{
			name:          "missing explicitly named persistence mapping",
			source:        strings.Replace(storageSource, "relation-storage org_store.membership", "relation-storage org_store.missing", 1),
			wantLoadError: "where Relation Storage is merged, is not found",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.WriteFile(storagePath, []byte(tc.source), 0o644); err != nil {
				t.Fatalf("write isolated invalid Relation Storage DSL: %v", err)
			}
			wrong, err := workspace.Load(root)
			if tc.wantLoadError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantLoadError) {
					t.Fatalf("invalid target Mapping must fail at Source Loader: expected %q got %v", tc.wantLoadError, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("world-level invalidity should produce typed diagnostics after successful load: %v", err)
			}
			count := 0
			for _, diagnostic := range validation.ValidatePersistenceRelationStorageMappings(wrong.World) {
				if diagnostic.Code == tc.wantCode {
					count++
				}
			}
			if count != 1 {
				t.Fatalf("expected exactly one typed diagnostic %s, got %d", tc.wantCode, count)
			}
		})
	}
	if err := os.WriteFile(storagePath, []byte(storageSource), 0o644); err != nil {
		t.Fatalf("restore isolated valid Relation Storage DSL before duplicate test: %v", err)
	}
	t.Run("duplicate source identity must not merge twice", func(t *testing.T) {
		secondPath := filepath.Join(filepath.Dir(storagePath), "duplicate-person-membership.tei")
		if err := os.WriteFile(secondPath, []byte(storageSource), 0o644); err != nil {
			t.Fatalf("create duplicate isolated authoring source: %v", err)
		}
		_, err := workspace.Load(root)
		if err == nil || !strings.Contains(err.Error(), "already defined") {
			t.Fatalf("duplicate Relation Storage Source identity must be rejected, got %v", err)
		}
	})
	// A Person -> Membership physical support link does not automatically
	// implement inverse Membership -> Person traversal or r7 temporal claims.
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
