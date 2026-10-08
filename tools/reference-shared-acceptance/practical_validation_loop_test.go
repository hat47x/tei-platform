package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/workspace"
)

// A new rule and its Application selection form one business change.
// Splitting them would temporarily make the new rule invisible to a Command.
func unitPriceWithSelectionCandidate(revision string) Candidate {
	candidate := unitPriceCandidate(revision)
	candidate.ID = "cand-order-entry-unit-price-loop"
	candidate.Intent = "受注明細の単価の正値制約と、確定前の評価対象を一度に変更する"
	candidate.Delta = append(candidate.Delta,
		validationUseOp("order_entry", "sales.order_line.unit_price_positive"),
	)
	candidate.Evidence = append(candidate.Evidence,
		Evidence{Source: orderEntryApplicationSource, Note: "評価対象を宣言するApplication source"},
	)
	return candidate
}

func TestPracticalRuleChangePreviewsAndAcceptsBothMeaningAndApplicationSelection(t *testing.T) {
	dir := scratchWorkspace(t)
	before := treeDigest(t, dir)
	candidate := unitPriceWithSelectionCandidate(revisionOfDir(t, dir))
	payload := mustJSON(t, candidate)

	ev, err := evaluate(dir, payload)
	if err != nil {
		t.Fatal(err)
	}
	if !ev.preview.Acceptable {
		t.Fatalf("ruleと選択の同時変更がPreviewで拒否されました: %+v", allDiagnostics(ev.preview))
	}
	assertTreeUnchanged(t, before, treeDigest(t, dir))
	if len(ev.preview.SemanticDiff) != 1 ||
		ev.preview.SemanticDiff[0].Kind != "validation" ||
		ev.preview.SemanticDiff[0].ID != "sales.order_line.unit_price_positive" ||
		ev.preview.SemanticDiff[0].Change != "added" {
		t.Fatalf("semantic diff=%+v", ev.preview.SemanticDiff)
	}
	if len(ev.preview.ApplicationDiff) != 1 ||
		ev.preview.ApplicationDiff[0].ID != "order_entry" ||
		ev.preview.ApplicationDiff[0].Change != "changed" {
		t.Fatalf("application diff=%+v", ev.preview.ApplicationDiff)
	}
	if len(ev.preview.SourceEdits) != 2 {
		t.Fatalf("source edits=%+v", ev.preview.SourceEdits)
	}
	if got := ev.preview.ChangedCollections; len(got) != 2 ||
		!hasCode(got, "semantics") || !hasCode(got, "applications") {
		t.Fatalf("changed collections=%v", got)
	}

	decision, err := accept(dir, payload, acceptOptions{})
	if err != nil || decision.Decision != "accepted" || !decision.SourceChanged {
		t.Fatalf("accept=%+v error=%v", decision, err)
	}
	loaded, err := workspace.Load(dir)
	if err != nil {
		t.Fatalf("受理後のSourceを再読込できません: %v", err)
	}
	rules, err := driver.NewResolver(loaded.World).ResolveApplicationValidations(
		core.MustCanonicalID("order_entry"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 3 ||
		rules[0].ID != "sales.order_line.quantity_positive" ||
		rules[1].ID != "sales.order.lines.line_number_unique" ||
		rules[2].ID != "sales.order_line.unit_price_positive" {
		t.Fatalf("Driverが解決したValidationの順序=%+v", rules)
	}
}

func TestPracticalRuleChangeRejectsBrokenOrRepeatedDeltaWithoutChangingSource(t *testing.T) {
	cases := []struct {
		name string
		edit func(*Candidate)
		code string
	}{
		{
			name: "unknown rule",
			edit: func(c *Candidate) {
				c.Delta[1].ApplicationValidationUse.Validation = "sales.order_line.unknown"
			},
			code: "VALIDATION_NOT_FOUND",
		},
		{
			name: "same rule declared twice",
			edit: func(c *Candidate) {
				c.Delta = append(c.Delta, c.Delta[0])
			},
			code: "CANDIDATE_NORMALIZE",
		},
		{
			name: "same use declared twice",
			edit: func(c *Candidate) {
				c.Delta = append(c.Delta, c.Delta[1])
			},
			code: "CANDIDATE_NORMALIZE",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := scratchWorkspace(t)
			before := treeDigest(t, dir)
			candidate := unitPriceWithSelectionCandidate(revisionOfDir(t, dir))
			tc.edit(&candidate)
			ev, err := evaluate(dir, mustJSON(t, candidate))
			if err != nil {
				t.Fatal(err)
			}
			if ev.preview.Acceptable || !hasCode(diagnosticCodes(ev.preview), tc.code) {
				t.Fatalf("acceptable=%t diagnostics=%v want %s",
					ev.preview.Acceptable, diagnosticCodes(ev.preview), tc.code)
			}
			assertTreeUnchanged(t, before, treeDigest(t, dir))
		})
	}
}

func TestPracticalRuleChangeKeepsAcceptedMeaningInTwoSourceFiles(t *testing.T) {
	dir := scratchWorkspace(t)
	candidate := unitPriceWithSelectionCandidate(revisionOfDir(t, dir))
	decision, err := accept(dir, mustJSON(t, candidate), acceptOptions{})
	if err != nil || decision.Decision != "accepted" {
		t.Fatalf("accept=%+v error=%v", decision, err)
	}
	for _, tc := range []struct {
		path string
		want string
	}{
		{
			path: "semantic/sales/validation.tei",
			want: "validation sales.order_line.unit_price_positive positive",
		},
		{
			path: orderEntryApplicationSource,
			want: "    uses validation sales.order_line.unit_price_positive",
		},
	} {
		content, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(tc.path)))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(content), tc.want) {
			t.Fatalf("%s に %q がありません", tc.path, tc.want)
		}
	}
}
