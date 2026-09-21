package changeanalysis

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/core"
	"tei/internal/ir"
)

const (
	repairIntakeSemantic = `namespace repair_intake

data Intake {
    attribute customerId type customer.customer_id
    relation customer to customer.customer one
        key repair_intake.intake.customer_id -> customer.customer.customer_id
}
`
	repairIntakeApplication = `application repair_intake {
    uses semantic repair_intake
    uses semantic customer

    database customer_master {
        target master_data
        access read
        requires {
            contract master_data.customer ^2
        }
    }
}
`
	repairIntakePattern = `pattern repair_intake.customer_reference reference {
    relation repair_intake.intake.customer
    datasource customer_master
    contract-query {
        contract master_data.customer
        query find_by_id
        input customer.customer.customer_id -> id
    }
}
`
)

func writeSource(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func worldWithSecondApplication(t *testing.T) (before, after ir.World) {
	t.Helper()
	before = loadWorld(t, referenceWorkspaceCopy(t)).World
	dir := referenceWorkspaceCopy(t)
	writeSource(t, dir, "semantic/repair-intake/intake.tei", repairIntakeSemantic)
	writeSource(t, dir, "applications/repair-intake/application.tei", repairIntakeApplication)
	writeSource(t, dir, "applications/repair-intake/patterns/customer-reference.tei", repairIntakePattern)
	return before, loadWorld(t, dir).World
}

func TestDiffApplicationsReportsAnApplicationAndItsPatternAsSeparateSubjects(t *testing.T) {
	before, after := worldWithSecondApplication(t)
	diff, err := DiffApplications(before, after)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"added application-pattern:repair_intake.customer_reference",
		"added application:repair_intake",
	}
	if got := keys(diff); !reflect.DeepEqual(got, want) {
		t.Fatalf("diff=%v want %v", got, want)
	}
	if same, err := DiffApplications(before, before); err != nil || len(same) != 0 {
		t.Fatalf("同一Worldに差分が出ています: %v %v", same, err)
	}
}

func TestDiffApplicationsDoesNotCountAPatternChangeAsAnApplicationChange(t *testing.T) {
	before := loadWorld(t, referenceWorkspaceCopy(t)).World
	after := loadWorld(t, referenceWorkspaceCopy(t)).World
	for i := range after.Applications {
		if after.Applications[i].ID != "order_entry" {
			continue
		}
		for p := range after.Applications[i].Patterns {
			if after.Applications[i].Patterns[p].ID == "order_entry.customer_reference" {
				after.Applications[i].Patterns[p].Datasource = "master_alias_changed"
			}
		}
	}
	diff, err := DiffApplications(before, after)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"changed application-pattern:order_entry.customer_reference"}
	if got := keys(diff); !reflect.DeepEqual(got, want) {
		t.Fatalf("Patternだけの変更がApplication本体まで二重にchanged扱いされています: %v", got)
	}
}

func TestDiffApplicationsSeesEveryPartOfTheApplicationBody(t *testing.T) {
	before := loadWorld(t, referenceWorkspaceCopy(t)).World
	mutations := map[string]func(app *ir.World){
		"semantic use": func(w *ir.World) {
			w.Applications[0].SemanticUses = append(w.Applications[0].SemanticUses, "extra")
		},
		"database requirement": func(w *ir.World) {
			w.Applications[0].Databases[0].RequiredContracts[0].CompatibleMajor++
		},
		"uses validation": func(w *ir.World) {
			w.Applications[0].Validations = append(w.Applications[0].Validations, core.CanonicalID("sales.order_line.quantity_positive"))
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			after := loadWorld(t, referenceWorkspaceCopy(t)).World
			mutate(&after)
			diff, err := DiffApplications(before, after)
			if err != nil {
				t.Fatal(err)
			}
			id := string(after.Applications[0].ID)
			if got, want := keys(diff), []string{"changed application:" + id}; !reflect.DeepEqual(got, want) {
				t.Fatalf("diff=%v want %v", got, want)
			}
		})
	}
}

func TestDiffApplicationsReportsARemovedPatternAndRefusesADuplicatePattern(t *testing.T) {
	before := loadWorld(t, referenceWorkspaceCopy(t)).World
	after := loadWorld(t, referenceWorkspaceCopy(t)).World
	for i := range after.Applications {
		if after.Applications[i].ID == "order_entry" {
			after.Applications[i].Patterns = after.Applications[i].Patterns[:len(after.Applications[i].Patterns)-1]
		}
	}
	diff, err := DiffApplications(before, after)
	if err != nil || len(diff) != 1 || diff[0].Change != "removed" || diff[0].Kind != "application-pattern" {
		t.Fatalf("diff=%+v err=%v", diff, err)
	}

	dup := loadWorld(t, referenceWorkspaceCopy(t)).World
	dup.Applications = append(dup.Applications, dup.Applications[0])
	if _, err := DiffApplications(dup, dup); err == nil {
		t.Fatal("同じApplication / Patternが複数回宣言されたWorldを差分できてしまいます")
	}
}

func TestImpactOfASecondApplicationTracesEveryDeclaredReference(t *testing.T) {
	before, after := worldWithSecondApplication(t)
	semanticChanges, err := Diff(before, after)
	if err != nil {
		t.Fatal(err)
	}
	applicationChanges, err := DiffApplications(before, after)
	if err != nil {
		t.Fatal(err)
	}
	edges, unresolved := Impact(after, append(semanticChanges, applicationChanges...))

	got := map[string]bool{}
	for _, e := range edges {
		if e.Reason == "" {
			t.Fatalf("辺に導出理由がありません: %+v", e)
		}
		got[e.From+" -> "+e.To+" via "+e.Via] = true
	}
	for _, want := range []string{
		"application-pattern:repair_intake.customer_reference -> application:repair_intake via application.pattern.owner",
		"application-pattern:repair_intake.customer_reference -> relation:repair_intake.intake.customer via application.pattern.relation",
		"application-pattern:repair_intake.customer_reference -> contract-query:master_data.customer/find_by_id via application.pattern.contract-query",
		"application-pattern:repair_intake.customer_reference -> attribute:customer.customer.customer_id via application.pattern.contract-query.input",
		"application:repair_intake -> namespace:repair_intake via application.uses-semantic-namespace",
		"application:repair_intake -> namespace:customer via application.uses-semantic-namespace",
		"application:repair_intake -> database:master_data via application.database.target",
		"application:repair_intake -> contract:master_data.customer via application.database.requires-contract",
		"namespace:repair_intake -> application:repair_intake via application.uses-semantic-namespace",
		"attribute:repair_intake.intake.customer_id -> data:repair_intake.intake via semantic.data.attribute",
		"relation:repair_intake.intake.customer -> data:customer.customer via semantic.relation.target",
		"relation:repair_intake.intake.customer -> attribute:repair_intake.intake.customer_id via semantic.relation.key.source",
		"relation:repair_intake.intake.customer -> attribute:customer.customer.customer_id via semantic.relation.key.target",
		"relation:repair_intake.intake.customer -> application-pattern:repair_intake.customer_reference via application.pattern.relation",
	} {
		if !got[want] {
			t.Fatalf("辺 %q がありません: %v", want, edges)
		}
	}
	if len(edges) != 14 {
		t.Fatalf("宣言された参照以外の辺が混じっています（%d本）: %v", len(edges), edges)
	}
	var codes []string
	for _, u := range unresolved {
		codes = append(codes, u.Code)
	}
	want := []string{"ENVIRONMENT_BINDING_NOT_AUTHORED", "DRIVER_SCENARIO_NOT_RUN", "TARGET_PROJECTION_NOT_EVALUATED"}
	if !reflect.DeepEqual(codes, want) {
		t.Fatalf("unresolved=%v want %v", codes, want)
	}
}

func TestEnvironmentBindingIsOnlyFlaggedForANewApplication(t *testing.T) {
	before := loadWorld(t, referenceWorkspaceCopy(t)).World
	after := loadWorld(t, referenceWorkspaceCopy(t)).World
	after.Applications[0].SemanticUses = append(after.Applications[0].SemanticUses, "extra")
	diff, err := DiffApplications(before, after)
	if err != nil {
		t.Fatal(err)
	}
	_, unresolved := Impact(after, diff)
	for _, u := range unresolved {
		if u.Code == "ENVIRONMENT_BINDING_NOT_AUTHORED" {
			t.Fatalf("既存Applicationの変更に、新規Applicationのunresolvedが付いています: %+v", u)
		}
	}
}

func TestRemovingANamespaceProducesNoEdgesEvenWhenAnApplicationStillUsesIt(t *testing.T) {
	before := loadWorld(t, referenceWorkspaceCopy(t)).World
	after := loadWorld(t, referenceWorkspaceCopy(t)).World
	kept := after.Semantics[:0]
	for _, ns := range after.Semantics {
		if ns.ID != "customer" {
			kept = append(kept, ns)
		}
	}
	after.Semantics = kept

	diff, err := Diff(before, after)
	if err != nil {
		t.Fatal(err)
	}
	removedNamespace := false
	for _, c := range diff {
		if c.Kind == "namespace" && c.ID == "customer" && c.Change == "removed" {
			removedNamespace = true
		}
	}
	if !removedNamespace {
		t.Fatalf("前提: namespace customer のremovedが差分に無い: %v", keys(diff))
	}
	edges, _ := Impact(after, diff)
	if len(edges) != 0 {
		t.Fatalf("削除されたsubjectから辺を作っています（afterに存在しないsubjectは辿れない）: %v", edges)
	}
}
