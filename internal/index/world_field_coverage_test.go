package index

import (
	"reflect"
	"testing"

	"tei/internal/ir"
)

// Tests in this file make the gap between "the World carries it" and "the
// reference graph shows it" visible. Four rounds of dogfooding found the same
// defect class in different layers (#3884 outcome, #3983 realize bindings, #4001
// persistence columns): a field existed in ir.World or in an index entry, but no
// dependency reached the graph, so trace answered "nothing" instead of "there is
// an edge you cannot see". Each was fixed one at a time, and the next field
// repeated the pattern.

// worldFieldCoverage records, per ir.World field, whether index.Build reads it.
// A field that is not read contributes nothing to the graph. That may be
// deliberate (plugins, releases and environments are not graph subjects), but it
// must be stated here rather than discovered later as a surprise.
var worldFieldCoverage = map[string]struct {
	read      bool
	whyUnread string
}{
	"Plugins":             {read: true},
	"Bundles":             {read: true},
	"Semantics":           {read: true},
	"Applications":        {read: true},
	"Databases":           {read: true},
	"Persistence":         {read: true},
	"Migrations":          {read: true},
	"MigrationProviders":  {read: true},
	"ApplicationReleases": {read: true},
	"DatabaseReleases":    {read: true},
	"SolutionReleases":    {read: true},
	"Environments":        {read: true},
	"Queries":             {read: true},
	"Bindings":            {read: true},
	"HeaderDetailContractQueryBindings": {
		whyUnread: "indexed nowhere; #3983 records this as a known gap",
	},
	"ContractQueryExecutorBindings": {
		whyUnread: "indexed nowhere; #3983 records this as a known gap",
	},
	"ContractQuerySQLRealizations": {
		whyUnread: "indexed nowhere; #3983 records this as a known gap",
	},
	"RelationQueries": {
		whyUnread: "indexed in relation query resolver; graph coverage tracked separately",
	},
}

// TestEveryWorldFieldIsAccountedFor fails when ir.World gains a field that this
// table does not mention. A new field silently absent from the graph is exactly
// the defect four rounds kept rediscovering.
func TestEveryWorldFieldIsAccountedFor(t *testing.T) {
	worldType := reflect.TypeOf(ir.World{})
	for i := 0; i < worldType.NumField(); i++ {
		name := worldType.Field(i).Name
		if _, ok := worldFieldCoverage[name]; !ok {
			t.Errorf("ir.World.%s is not in worldFieldCoverage; decide whether index.Build reads it and say so here", name)
		}
	}
	for name := range worldFieldCoverage {
		if _, ok := worldType.FieldByName(name); !ok {
			t.Errorf("worldFieldCoverage names ir.World.%s, which no longer exists; remove the row", name)
		}
	}
}

// TestUnreadWorldFieldsAreExplained keeps the "not read" rows honest: a gap
// without a reason is an unexplained hole, not a decision.
func TestUnreadWorldFieldsAreExplained(t *testing.T) {
	for name, row := range worldFieldCoverage {
		if row.read && row.whyUnread != "" {
			t.Errorf("%s is marked read but also carries a whyUnread reason", name)
		}
		if !row.read && row.whyUnread == "" {
			t.Errorf("%s is marked unread without a reason", name)
		}
	}
}
