package operationpatterncrosswalk

import (
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/database/persistence"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// world builds a synthetic World. It is deliberately small: the point of the
// reference is the shape of the link, not a business model.
func world() ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{{ID: "sales",
			Data: []semantic.Data{
				{ID: "sales.order", Symbol: "Order", Relations: []semantic.Relation{
					{ID: "sales.order.lines", Symbol: "lines", Target: "sales.order_line", Cardinality: semantic.CardinalityMany},
					{ID: "sales.order.customer", Symbol: "customer", Target: "customer.customer", Cardinality: semantic.CardinalityOne},
				}},
				{ID: "sales.order_line", Symbol: "OrderLine"},
				{ID: "customer.customer", Symbol: "Customer"},
			},
			Capabilities: []semantic.Capability{
				{ID: "sales.order.submit", Symbol: "submit", Effect: &semantic.DataEffect{Kind: semantic.DataEffectUpdate, Data: "sales.order"}},
				{ID: "sales.order.quote_lines", Symbol: "quote", Output: "sales.order"},
			},
		}},
		// The operation surface only derives standard operations for a Data that
		// an Application aliases and a Persistence mapping names, so the fixture
		// declares both. Nothing else in the fixture is a product claim.
		Databases: []database.Database{
			{ID: "sales_core"},
			{ID: "master_data"},
		},
		Persistence: []persistence.Mapping{
			{ID: "sales_core.order", Database: "sales_core", SemanticData: "sales.order", Table: "order",
				PrimaryKey: []core.CanonicalID{"sales.order.order_id"}},
			{ID: "sales_core.order_line", Database: "sales_core", SemanticData: "sales.order_line", Table: "order_line",
				PrimaryKey: []core.CanonicalID{"sales.order_line.order_line_id"}},
			// The reference Pattern names the master datasource, so the read
			// operation it can link to exists only when that alias has a
			// mapping for the Relation's target Data.
			{ID: "master_data.customer", Database: "master_data", SemanticData: "customer.customer", Table: "customer",
				PrimaryKey: []core.CanonicalID{"customer.customer.customer_id"}},
		},
		Applications: []application.Application{{ID: "order_entry",
			Databases: []application.DatabaseRequirement{
				{Alias: "sales", TargetDatabase: "sales_core", Access: database.AccessReadWrite},
				{Alias: "master", TargetDatabase: "master_data", Access: database.AccessRead},
			},
			Patterns: []application.Pattern{
				{ID: "order_entry.recent_orders", Kind: application.PatternBrowse, Data: "sales.order", Query: "order_entry.recent_orders"},
				{ID: "order_entry.order_record", Kind: application.PatternRecord, Data: "sales.order"},
				{ID: "order_entry.submit_order", Kind: application.PatternCommand, Capability: "sales.order.submit"},
				{ID: "order_entry.preview_order_quote", Kind: application.PatternCommand, Capability: "sales.order.quote_lines"},
				{ID: "order_entry.customer_reference", Kind: application.PatternReference, Relation: "sales.order.customer", Datasource: "master"},
				{ID: "order_entry.order_lines", Kind: application.PatternHeaderDetail, Relation: "sales.order.lines"},
				{ID: "order_entry.no_data", Kind: application.PatternRecord},
				{ID: "order_entry.gone_capability", Kind: application.PatternCommand, Capability: "sales.gone"},
				{ID: "order_entry.gone_relation", Kind: application.PatternReference, Relation: "sales.gone"},
			}}},
	}
}

func linkOf(r Report, operation string) *Link {
	for i := range r.Applications[0].Links {
		if r.Applications[0].Links[i].Operation == operation {
			return &r.Applications[0].Links[i]
		}
	}
	return nil
}

func unlinkedOf(r Report, pattern string) *Unlinked {
	for i := range r.Applications[0].Unlinked {
		if r.Applications[0].Unlinked[i].Pattern == pattern {
			return &r.Applications[0].Unlinked[i]
		}
	}
	return nil
}

func TestBrowseAndRecordLinkOnDataOnly(t *testing.T) {
	r := Derive(world())
	browse := linkOf(r, "order_entry.list.sales.order")
	if browse == nil || browse.Pattern != "order_entry.recent_orders" || browse.Via != ViaData {
		t.Fatalf("browse link = %+v", browse)
	}
	record := linkOf(r, "order_entry.read.sales.order")
	if record == nil || record.Pattern != "order_entry.order_record" || record.Via != ViaData {
		t.Fatalf("record link = %+v", record)
	}
}

func TestCommandLinksOnlyWhenTheCapabilityDeclaresAnEffect(t *testing.T) {
	r := Derive(world())
	update := linkOf(r, "order_entry.update.sales.order")
	if update == nil || update.Pattern != "order_entry.submit_order" || update.Via != ViaCapability {
		t.Fatalf("command link = %+v", update)
	}
	// The quote capability has no effect. The rule must not invent one to make
	// the link appear.
	if got := unlinkedOf(r, "order_entry.preview_order_quote"); got == nil || got.Code != "out-of-scope" {
		t.Fatalf("read-only command = %+v", got)
	}
	if link := linkOf(r, "order_entry.create.sales.order"); link != nil && link.Pattern == "order_entry.preview_order_quote" {
		t.Fatal("read-only command linked to a write operation")
	}
}

func TestRelationKindsReadThroughToTheTargetData(t *testing.T) {
	r := Derive(world())
	read := linkOf(r, "order_entry.read.customer.customer")
	if read == nil || read.Pattern != "order_entry.customer_reference" || read.Via != ViaRelation {
		t.Fatalf("reference link = %+v", read)
	}
	// header-detail can match list and read on the same Data; both links are
	// reported and the Pattern is marked ambiguous instead of being chosen.
	if len(r.Applications[0].Ambiguous) != 1 || r.Applications[0].Ambiguous[0] != "order_entry.order_lines" {
		t.Fatalf("ambiguous = %v", r.Applications[0].Ambiguous)
	}
}

func TestDeclarationsThatCannotBeReadAreReportedNotGuessed(t *testing.T) {
	r := Derive(world())
	for pattern, want := range map[string]string{
		"order_entry.no_data":         "missing-semantics",
		"order_entry.gone_capability": "missing-input",
		"order_entry.gone_relation":   "missing-input",
	} {
		got := unlinkedOf(r, pattern)
		if got == nil || got.Code != want {
			t.Fatalf("%s = %+v, want code %s", pattern, got, want)
		}
	}
}

func TestTwoDeclarationsOnOneOperationConflictInsteadOfChoosing(t *testing.T) {
	w := world()
	w.Applications[0].Patterns = append(w.Applications[0].Patterns,
		application.Pattern{ID: "order_entry.recent_orders2", Kind: application.PatternBrowse, Data: "sales.order"})
	r := Derive(w)
	if len(r.Applications[0].Conflicts) != 1 {
		t.Fatalf("conflicts = %+v", r.Applications[0].Conflicts)
	}
	if link := linkOf(r, "order_entry.list.sales.order"); link != nil {
		t.Fatalf("conflict was resolved to %+v", link)
	}
}

func TestReportStatesItsAuthorityAndIsDeterministic(t *testing.T) {
	a := Derive(world())
	b := Derive(world())
	ja, err := a.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	jb, err := b.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if string(ja) != string(jb) {
		t.Fatal("report is not repeatable")
	}
	if a.Authority != Authority || a.Schema != Schema {
		t.Fatalf("authority = %q schema = %q", a.Authority, a.Schema)
	}
	if a.RuleFingerprint == "" {
		t.Fatal("rule fingerprint missing")
	}
}
