package validation

import (
	"testing"

	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/ir"
)

// The resolver reports UNKNOWN_* when a reference does not exist and
// AMBIGUOUS_* when it matches more than one declaration. Both families are part
// of the contract a reader of the diagnostics relies on, so each code is
// produced here by breaking one reference in an otherwise accepted world
// (#4032).
func TestHeaderDetailBindingResolutionCodesAreProduced(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*ir.World)
		want   string
	}{
		{
			name: "unknown application",
			mutate: func(w *ir.World) {
				w.HeaderDetailContractQueryBindings[0].Application = "missing_app"
			},
			want: "UNKNOWN_HEADER_DETAIL_BINDING_APPLICATION",
		},
		{
			name: "ambiguous application",
			mutate: func(w *ir.World) {
				w.Applications = append(w.Applications, w.Applications[0])
			},
			want: "AMBIGUOUS_HEADER_DETAIL_BINDING_APPLICATION",
		},
		{
			name: "unknown pattern",
			mutate: func(w *ir.World) {
				w.HeaderDetailContractQueryBindings[0].Pattern = "order_entry.missing"
			},
			want: "UNKNOWN_HEADER_DETAIL_BINDING_PATTERN",
		},
		{
			name: "ambiguous pattern",
			mutate: func(w *ir.World) {
				w.Applications[0].Patterns = append(w.Applications[0].Patterns, w.Applications[0].Patterns[0])
			},
			want: "AMBIGUOUS_HEADER_DETAIL_BINDING_PATTERN",
		},
		{
			name: "unknown datasource",
			mutate: func(w *ir.World) {
				w.HeaderDetailContractQueryBindings[0].Datasource = "missing"
			},
			want: "UNKNOWN_HEADER_DETAIL_BINDING_DATASOURCE",
		},
		{
			name: "ambiguous datasource",
			mutate: func(w *ir.World) {
				w.Applications[0].Databases = append(w.Applications[0].Databases, w.Applications[0].Databases[0])
			},
			want: "AMBIGUOUS_HEADER_DETAIL_BINDING_DATASOURCE",
		},
		{
			name: "ambiguous contract requirement",
			mutate: func(w *ir.World) {
				requirement := w.Applications[0].Databases[0].RequiredContracts[0]
				w.Applications[0].Databases[0].RequiredContracts = append(
					w.Applications[0].Databases[0].RequiredContracts, requirement)
			},
			want: "AMBIGUOUS_HEADER_DETAIL_BINDING_CONTRACT_REQUIREMENT",
		},
		{
			name: "unknown database",
			mutate: func(w *ir.World) {
				w.HeaderDetailContractQueryBindings[0].Target.Database = "missing_db"
				w.Applications[0].Databases[0].TargetDatabase = "missing_db"
			},
			want: "UNKNOWN_HEADER_DETAIL_BINDING_DATABASE",
		},
		{
			name: "ambiguous database",
			mutate: func(w *ir.World) {
				w.Databases = append(w.Databases, w.Databases[0])
			},
			want: "AMBIGUOUS_HEADER_DETAIL_BINDING_DATABASE",
		},
		{
			name: "unknown contract",
			mutate: func(w *ir.World) {
				w.HeaderDetailContractQueryBindings[0].Target.Contract = "sales_core.missing"
				w.Applications[0].Databases[0].RequiredContracts = append(
					w.Applications[0].Databases[0].RequiredContracts,
					database.ContractRequirement{ContractID: "sales_core.missing", CompatibleMajor: 1})
			},
			want: "UNKNOWN_HEADER_DETAIL_BINDING_CONTRACT",
		},
		{
			name: "ambiguous contract",
			mutate: func(w *ir.World) {
				w.Databases[0].Contracts = append(w.Databases[0].Contracts, w.Databases[0].Contracts[0])
			},
			want: "AMBIGUOUS_HEADER_DETAIL_BINDING_CONTRACT",
		},
		{
			name: "unknown query",
			mutate: func(w *ir.World) {
				w.HeaderDetailContractQueryBindings[0].Target.Query = core.LocalName("missing")
			},
			want: "UNKNOWN_HEADER_DETAIL_BINDING_QUERY",
		},
		{
			name: "ambiguous query",
			mutate: func(w *ir.World) {
				query := w.Databases[0].Contracts[0].Queries[0]
				w.Databases[0].Contracts[0].Queries = append(w.Databases[0].Contracts[0].Queries, query)
			},
			want: "AMBIGUOUS_HEADER_DETAIL_BINDING_QUERY",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			world := headerDetailContractQueryBindingWorld()
			testCase.mutate(&world)
			assertHeaderDetailBindingDiagnostic(t, ValidateHeaderDetailContractQueryBindings(world), testCase.want)
		})
	}
}

// The baseline world must stay accepted: every case above depends on it, and a
// resolver change that rejects it would make the table prove nothing.
func TestHeaderDetailBindingBaselineWorldIsAccepted(t *testing.T) {
	world := headerDetailContractQueryBindingWorld()
	if diagnostics := ValidateHeaderDetailContractQueryBindings(world); len(diagnostics) != 0 {
		t.Fatalf("baseline world is rejected: %+v", diagnostics)
	}
}
