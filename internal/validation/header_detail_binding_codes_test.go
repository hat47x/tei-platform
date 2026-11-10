package validation

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/realization"
)

// The header-detail Contract Query binding validator reports one code per
// broken field. Those codes read as a contract, so each of them has to be
// produced by something: a code that no input can raise is a claim the
// validator cannot keep. This test breaks one field at a time and checks the
// code that comes back.
func TestHeaderDetailBindingFieldCodesAreProduced(t *testing.T) {
	valid := func() realization.HeaderDetailContractQueryBinding {
		return realization.HeaderDetailContractQueryBinding{
			Application: "order_entry",
			Pattern:     "order_entry.header_detail",
			Datasource:  "orders",
			Target: realization.ContractQueryTarget{
				Database: "sales_db",
				Contract: "sales.order_list",
				Query:    "list",
			},
		}
	}

	cases := []struct {
		name   string
		mutate func(*realization.HeaderDetailContractQueryBinding)
		want   string
	}{
		{
			name:   "application missing",
			mutate: func(b *realization.HeaderDetailContractQueryBinding) { b.Application = "" },
			want:   "HEADER_DETAIL_BINDING_APPLICATION_REQUIRED",
		},
		{
			name:   "application malformed",
			mutate: func(b *realization.HeaderDetailContractQueryBinding) { b.Application = core.CanonicalID("Not A Name") },
			want:   "INVALID_HEADER_DETAIL_BINDING_APPLICATION",
		},
		{
			name:   "pattern missing",
			mutate: func(b *realization.HeaderDetailContractQueryBinding) { b.Pattern = "" },
			want:   "HEADER_DETAIL_BINDING_PATTERN_REQUIRED",
		},
		{
			name:   "pattern malformed",
			mutate: func(b *realization.HeaderDetailContractQueryBinding) { b.Pattern = core.CanonicalID("bad pattern") },
			want:   "INVALID_HEADER_DETAIL_BINDING_PATTERN",
		},
		{
			name:   "datasource missing",
			mutate: func(b *realization.HeaderDetailContractQueryBinding) { b.Datasource = "" },
			want:   "HEADER_DETAIL_BINDING_DATASOURCE_REQUIRED",
		},
		{
			name:   "datasource malformed",
			mutate: func(b *realization.HeaderDetailContractQueryBinding) { b.Datasource = core.LocalName("bad name") },
			want:   "INVALID_HEADER_DETAIL_BINDING_DATASOURCE",
		},
		{
			name:   "database missing",
			mutate: func(b *realization.HeaderDetailContractQueryBinding) { b.Target.Database = "" },
			want:   "HEADER_DETAIL_BINDING_DATABASE_REQUIRED",
		},
		{
			name: "database malformed",
			mutate: func(b *realization.HeaderDetailContractQueryBinding) {
				b.Target.Database = core.CanonicalID("bad database")
			},
			want: "INVALID_HEADER_DETAIL_BINDING_DATABASE",
		},
		{
			name:   "contract missing",
			mutate: func(b *realization.HeaderDetailContractQueryBinding) { b.Target.Contract = "" },
			want:   "HEADER_DETAIL_BINDING_CONTRACT_REQUIRED",
		},
		{
			name: "contract malformed",
			mutate: func(b *realization.HeaderDetailContractQueryBinding) {
				b.Target.Contract = core.CanonicalID("bad contract")
			},
			want: "INVALID_HEADER_DETAIL_BINDING_CONTRACT",
		},
		{
			name:   "query missing",
			mutate: func(b *realization.HeaderDetailContractQueryBinding) { b.Target.Query = "" },
			want:   "HEADER_DETAIL_BINDING_QUERY_REQUIRED",
		},
		{
			name:   "query malformed",
			mutate: func(b *realization.HeaderDetailContractQueryBinding) { b.Target.Query = core.LocalName("bad query") },
			want:   "INVALID_HEADER_DETAIL_BINDING_QUERY",
		},
		{
			name: "input source attribute missing",
			mutate: func(b *realization.HeaderDetailContractQueryBinding) {
				b.Inputs = []realization.HeaderDetailContractQueryInputBinding{{Input: "order_id"}}
			},
			want: "HEADER_DETAIL_BINDING_SOURCE_ATTRIBUTE_REQUIRED",
		},
		{
			name: "input source attribute malformed",
			mutate: func(b *realization.HeaderDetailContractQueryBinding) {
				b.Inputs = []realization.HeaderDetailContractQueryInputBinding{{SourceAttribute: "bad attribute", Input: "order_id"}}
			},
			want: "INVALID_HEADER_DETAIL_BINDING_SOURCE_ATTRIBUTE",
		},
		{
			name: "input missing",
			mutate: func(b *realization.HeaderDetailContractQueryBinding) {
				b.Inputs = []realization.HeaderDetailContractQueryInputBinding{{SourceAttribute: "order_entry.order_id"}}
			},
			want: "HEADER_DETAIL_BINDING_INPUT_REQUIRED",
		},
		{
			name: "input malformed",
			mutate: func(b *realization.HeaderDetailContractQueryBinding) {
				b.Inputs = []realization.HeaderDetailContractQueryInputBinding{{SourceAttribute: "order_entry.order_id", Input: "bad input"}}
			},
			want: "INVALID_HEADER_DETAIL_BINDING_INPUT",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			binding := valid()
			testCase.mutate(&binding)
			world := ir.World{HeaderDetailContractQueryBindings: []realization.HeaderDetailContractQueryBinding{binding}}
			for _, diagnostic := range ValidateHeaderDetailContractQueryBindings(world) {
				if diagnostic.Code == testCase.want {
					return
				}
			}
			var codes []string
			for _, diagnostic := range ValidateHeaderDetailContractQueryBindings(world) {
				codes = append(codes, diagnostic.Code)
			}
			t.Fatalf("expected %s, got %s", testCase.want, strings.Join(codes, ", "))
		})
	}
}
