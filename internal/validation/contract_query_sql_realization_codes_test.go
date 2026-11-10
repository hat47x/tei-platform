package validation

import (
	"testing"

	"tei/internal/core"
	"tei/internal/ir"
)

// A Contract Query SQL realization names a Database, a Contract and a Query.
// The validator reports a distinct code for "not there" and "declared twice" at
// each step, so each of the six codes is produced here by breaking one step of
// an accepted world (#4032).
func TestContractQuerySQLRealizationCodesAreProduced(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*ir.World)
		want   string
	}{
		{
			name: "unknown database",
			mutate: func(w *ir.World) {
				w.ContractQuerySQLRealizations[0].Target.Database = "missing_db"
			},
			want: "UNKNOWN_CONTRACT_QUERY_SQL_REALIZATION_DATABASE",
		},
		{
			name: "ambiguous database",
			mutate: func(w *ir.World) {
				w.Databases = append(w.Databases, w.Databases[0])
			},
			want: "AMBIGUOUS_CONTRACT_QUERY_SQL_REALIZATION_DATABASE",
		},
		{
			name: "unknown contract",
			mutate: func(w *ir.World) {
				w.ContractQuerySQLRealizations[0].Target.Contract = "master_data.missing"
			},
			want: "UNKNOWN_CONTRACT_QUERY_SQL_REALIZATION_CONTRACT",
		},
		{
			name: "ambiguous contract",
			mutate: func(w *ir.World) {
				w.Databases[0].Contracts = append(w.Databases[0].Contracts, w.Databases[0].Contracts[0])
			},
			want: "AMBIGUOUS_CONTRACT_QUERY_SQL_REALIZATION_CONTRACT",
		},
		{
			name: "unknown query",
			mutate: func(w *ir.World) {
				w.ContractQuerySQLRealizations[0].Target.Query = core.LocalName("missing")
			},
			want: "UNKNOWN_CONTRACT_QUERY_SQL_REALIZATION_QUERY",
		},
		{
			name: "ambiguous query",
			mutate: func(w *ir.World) {
				query := w.Databases[0].Contracts[0].Queries[0]
				w.Databases[0].Contracts[0].Queries = append(w.Databases[0].Contracts[0].Queries, query)
			},
			want: "AMBIGUOUS_CONTRACT_QUERY_SQL_REALIZATION_QUERY",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			world := contractQuerySQLValidationWorld()
			testCase.mutate(&world)
			assertHasDiagnostic(t, ValidateContractQuerySQLRealizations(world), testCase.want)
		})
	}
}
