package orderdatarealization

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	exampleslide "tei/tools/reference-example-slide"
	ercrossview "tei/tools/reference-er-table-crossview"
)

// Step1AddAttribute copies the workspace twice (the chain copy and the variant
// without decisions, plus an untouched R0 copy) and adds the attribute by
// direct edits. The shared acceptance tool has no typed operation for adding
// an attribute to existing Data, so acceptance is not connected here.
func (c *Chain) Step1AddAttribute(ctx context.Context) error {
	for _, dst := range []string{c.R0Dir, c.WorkDir, c.VariantDir} {
		if err := CopyWorkspace(c.Source, dst); err != nil {
			return fmt.Errorf("step 1: %w", err)
		}
	}
	r0, err := Revision(c.WorkDir)
	if err != nil {
		return fmt.Errorf("step 1: %w", err)
	}
	world0, errs, err := LoadValidated(c.R0Dir)
	if err != nil || len(errs) > 0 {
		return fmt.Errorf("step 1: R0 copy invalid: %v %v", err, errs)
	}
	c.worldR0 = world0
	var edits []EditRecord
	for _, e := range attributeEdits(c.Decl) {
		rec, err := applyLineEdit(c.WorkDir, e)
		if err != nil {
			return fmt.Errorf("step 1: %w", err)
		}
		if _, err := applyLineEdit(c.VariantDir, e); err != nil {
			return fmt.Errorf("step 1 variant: %w", err)
		}
		edits = append(edits, rec)
	}
	r1, err := Revision(c.WorkDir)
	if err != nil {
		return fmt.Errorf("step 1: %w", err)
	}
	rv, err := Revision(c.VariantDir)
	if err != nil {
		return fmt.Errorf("step 1: %w", err)
	}
	world1, errs, err := LoadValidated(c.WorkDir)
	if err != nil || len(errs) > 0 {
		return fmt.Errorf("step 1: copy with the attribute is invalid: %v %v", err, errs)
	}
	if !hasAttribute(world1, c.Decl.Attribute.Attribute) {
		return fmt.Errorf("step 1: attribute %s is not in the loaded World", c.Decl.Attribute.Attribute)
	}
	if r1 == r0 || rv != r1 {
		return fmt.Errorf("step 1: revisions do not move as expected (R0 %s, R1 %s, variant %s)", r0, r1, rv)
	}
	c.Rec.Revisions.R0, c.Rec.Revisions.R1, c.Rec.Revisions.VariantNoDecisions = r0, r1, rv
	lines := 0
	for _, e := range edits {
		lines += e.LinesInserted
	}
	_, dotTEI := os.Stat(filepath.Join(c.WorkDir, ".tei"))
	c.addStep(StepRecord{
		Step: 1, Name: "属性の追加と版の記録", ModelRevision: r0, RevisionAfter: r1,
		Assets: []AssetUse{
			{Asset: "tools/reference-shared-acceptance", Called: false, Reason: "共通受理の型付き操作(13種)に既存のDataへの属性追加がない。複製への直接編集として記録し、受理は未接続"},
		},
		Output: map[string]any{
			"source":               "reference/workspace(読むだけ。一時ディレクトリへ複製)",
			"directEdits":          edits,
			"filesEdited":          len(edits),
			"linesInserted":        lines,
			"acceptance":           "not-connected",
			"attribute":            c.Decl.Attribute.Attribute,
			"validationErrors":     0,
			"dotTeiCopied":         dotTEI == nil,
			"dotTeiAffectsLoad":    "複製した.tei/(決定の記録とplugin-lock)を含めたまま読込みと検証が通った",
			"variantSameRevision":  rv == r1,
		},
	})
	c.gap("コア", "意味の変更(既存のDataへの属性追加)を受理版として同定する型付き操作がない", "工程1の複製への直接編集。編集箇所と件数を記録")
	return nil
}

// Step2AcceptBusinessKey asks the design question through the example slide
// review, binds the answer to the review, and accepts the candidate key through
// the shared acceptance tool on the copy.
func (c *Chain) Step2AcceptBusinessKey(ctx context.Context) error {
	r1 := c.Rec.Revisions.R1
	if got, err := Revision(c.WorkDir); err != nil || got != r1 {
		return &RevisionMismatch{Step: 2, Workspace: c.WorkDir, Want: r1, Got: got}
	}
	world, errs, err := LoadValidated(c.WorkDir)
	if err != nil || len(errs) > 0 {
		return fmt.Errorf("step 2: %v %v", err, errs)
	}
	catalog, err := exampleslide.CatalogFromWorld(world)
	if err != nil {
		return fmt.Errorf("step 2: %w", err)
	}
	keyField := c.Decl.BusinessKey.Attributes[0]
	rec := func(id, ext string, note any) exampleslide.Record {
		v := map[string]json.RawMessage{keyField: mustJSON(ext), c.Decl.Attribute.Attribute: mustJSON(note)}
		return exampleslide.Record{ID: id, Table: c.Decl.BusinessKey.Data, Values: v}
	}
	questionID := "external-number-business-key"
	input := exampleslide.Input{
		Catalog: catalog,
		Example: exampleslide.Example{
			ID: "order-data-realization", ModelRevision: catalog.Revision,
			Base: []exampleslide.Record{
				rec("order-1001", "0070001", "午前中に配達"),
				rec("order-1002", "0070002", nil),
				rec("order-1003", "000123", ""),
			},
			Questions: []exampleslide.Question{{ID: questionID, Text: c.Decl.BusinessKey.Question}},
		},
		Diagram:      exampleslide.Diagram{ID: "order", Placements: []exampleslide.Placement{{ID: "order", Table: c.Decl.BusinessKey.Data}}},
		Presentation: exampleslide.Presentation{Frames: []exampleslide.Frame{{ID: "base", Title: "受注", Stage: "base"}}},
	}
	review, err := exampleslide.Review(input)
	if err != nil {
		return fmt.Errorf("step 2 review: %w", err)
	}
	findings, err := exampleslide.ProbeBaseCandidateKeys(catalog, input.Example, []exampleslide.CandidateKeyHypothesis{
		{ID: "external-number", Table: c.Decl.BusinessKey.Data, Fields: c.Decl.BusinessKey.Attributes},
	})
	if err != nil {
		return fmt.Errorf("step 2 key probe: %w", err)
	}
	handoff, err := exampleslide.BindAnswers(input, exampleslide.AnswerInput{
		Schema: "tei.reference.example-review-answer/v0", CaseID: review.CaseID,
		ModelRevision: review.ModelRevision, ReviewFingerprint: review.Fingerprint,
		Answers: []exampleslide.AnswerEntry{{
			Question: exampleslide.QuestionRef{Kind: "authored", ID: questionID},
			Response: c.Decl.BusinessKey.Answer, Disposition: "propose-model-change",
			Context:   "order-data-realization 工程2",
			ClaimedBy: exampleslide.AnswerActor{Kind: "human", ID: "synthetic-data-owner"},
		}},
	})
	if err != nil {
		return fmt.Errorf("step 2 answer: %w", err)
	}

	// The shared acceptance tool computes the same revision.
	raw, err := runTool(ctx, c.Tools.SharedAcceptance, "revision", "-workspace", c.WorkDir)
	if err != nil {
		return fmt.Errorf("step 2: %w", err)
	}
	var revOut struct {
		ModelRevision string `json:"modelRevision"`
	}
	if err := json.Unmarshal(raw, &revOut); err != nil || revOut.ModelRevision != r1 {
		return fmt.Errorf("step 2: shared acceptance revision %q differs from R1 %s (%v)", revOut.ModelRevision, r1, err)
	}
	candidate := map[string]any{
		"schema": "tei.reference.change-candidate/v0", "id": "odr-external-number-key",
		"baseModelRevision": r1,
		"origin": map[string]any{"kind": "cli", "actor": "order-data-realization",
			"notes": map[string]string{"answerHandoff": handoff.Fingerprint, "reviewFingerprint": review.Fingerprint}},
		"intent": c.Decl.BusinessKey.Answer,
		"delta": []any{map[string]any{"op": "add-candidate-key", "addCandidateKey": map[string]any{
			"data": c.Decl.BusinessKey.Data, "id": c.Decl.BusinessKey.ID,
			"symbol": c.Decl.BusinessKey.Symbol, "attributes": c.Decl.BusinessKey.Attributes,
		}}},
	}
	candPath := filepath.Join(c.Dir, "candidate-business-key.json")
	if err := writeJSON(candPath, candidate); err != nil {
		return err
	}
	if _, err := runTool(ctx, c.Tools.SharedAcceptance, "preview", "-workspace", c.WorkDir, "-candidate", candPath); err != nil {
		return fmt.Errorf("step 2: %w", err)
	}
	raw, err = runTool(ctx, c.Tools.SharedAcceptance, "review", "-workspace", c.WorkDir, "-candidate", candPath, "-format", "json")
	if err != nil {
		return fmt.Errorf("step 2: %w", err)
	}
	var rv struct {
		ReviewFingerprint string `json:"reviewFingerprint"`
		PreviewAcceptable bool   `json:"previewAcceptable"`
	}
	if err := json.Unmarshal(raw, &rv); err != nil || !rv.PreviewAcceptable || rv.ReviewFingerprint == "" {
		return fmt.Errorf("step 2: candidate key is not acceptable: %v %s", err, raw)
	}
	raw, err = runTool(ctx, c.Tools.SharedAcceptance, "accept", "-workspace", c.WorkDir, "-candidate", candPath, "-expected-review-fingerprint", rv.ReviewFingerprint)
	if err != nil {
		return fmt.Errorf("step 2: %w", err)
	}
	var acc struct {
		Decision            string `json:"decision"`
		ModelRevisionBefore string `json:"modelRevisionBefore"`
		ModelRevisionAfter  string `json:"modelRevisionAfter"`
		RecordPath          string `json:"recordPath"`
		CandidateFP         string `json:"candidateFingerprint"`
	}
	if err := json.Unmarshal(raw, &acc); err != nil || acc.Decision != "accepted" || acc.ModelRevisionBefore != r1 {
		return fmt.Errorf("step 2: acceptance did not accept from R1: %v %s", err, raw)
	}
	r2, err := Revision(c.WorkDir)
	if err != nil {
		return err
	}
	if r2 != acc.ModelRevisionAfter {
		return fmt.Errorf("step 2: copy revision %s differs from accepted revision %s", r2, acc.ModelRevisionAfter)
	}
	c.Rec.Revisions.R2 = r2
	c.addStep(StepRecord{
		Step: 2, Name: "設計質問への回答と業務キーの受理", ModelRevision: r1, RevisionAfter: r2,
		Assets: []AssetUse{
			{Asset: "tools/reference-example-slide", Called: true, How: "Goの取込み: CatalogFromWorld、Review、ProbeBaseCandidateKeys、BindAnswers"},
			{Asset: "tools/reference-shared-acceptance", Called: true, How: "go buildした実行ファイル: revision、preview、review -format json、accept(add-candidate-key)"},
		},
		Output: map[string]any{
			"reviewFingerprint":        review.Fingerprint,
			"reviewModelRevision":      review.ModelRevision,
			"authoredQuestions":        review.Authored,
			"keyProbe":                 findings,
			"answerHandoffFingerprint": handoff.Fingerprint,
			"answerStatus":             handoff.Answers[0].Status,
			"acceptanceReview":         rv.ReviewFingerprint,
			"decision":                 acc.Decision,
			"decisionRecord":           acc.RecordPath,
			"candidateFingerprint":     acc.CandidateFP,
			"sharedAcceptanceRevisionEqualsR1": true,
			"note": "例の記録に衝突がないことは業務キーの根拠にならない(probeはunresolvedを返す)。受理は回答の転記であり、回答者の認証はない",
		},
	})
	return nil
}

// Step3CrossviewAndUnique compares the accepted key with the physical keys and
// records the unique-constraint decision as a direct edit of the storage
// mapping (no typed operation exists for it).
func (c *Chain) Step3CrossviewAndUnique(ctx context.Context) error {
	r2 := c.Rec.Revisions.R2
	if got, err := Revision(c.WorkDir); err != nil || got != r2 {
		return &RevisionMismatch{Step: 3, Workspace: c.WorkDir, Want: r2, Got: got}
	}
	world, errs, err := LoadValidated(c.WorkDir)
	if err != nil || len(errs) > 0 {
		return fmt.Errorf("step 3: %v %v", err, errs)
	}
	before, err := ercrossview.Project(world)
	if err != nil {
		return fmt.Errorf("step 3: %w", err)
	}
	keyID := c.Decl.BusinessKey.ID
	cmpBefore := keyComparisons(before, keyID)
	if len(cmpBefore) == 0 {
		return fmt.Errorf("step 3: no key comparison for %s", keyID)
	}
	for _, k := range cmpBefore {
		if k.ConstraintKind == "primary-key" && k.Status == "same-attribute-set" {
			return fmt.Errorf("step 3: the business key unexpectedly equals the primary key")
		}
		if k.ConstraintKind != "primary-key" {
			return fmt.Errorf("step 3: a physical %s already exists before the decision", k.ConstraintKind)
		}
	}
	var edit *EditRecord
	if c.Decl.UniqueDecision.Declare {
		rec, err := applyLineEdit(c.WorkDir, uniqueEdit(c.Decl))
		if err != nil {
			return fmt.Errorf("step 3: %w", err)
		}
		edit = &rec
	}
	r3, err := Revision(c.WorkDir)
	if err != nil {
		return err
	}
	final, errs, err := LoadValidated(c.WorkDir)
	if err != nil || len(errs) > 0 {
		return fmt.Errorf("step 3: final version invalid: %v %v", err, errs)
	}
	after, err := ercrossview.Project(final)
	if err != nil {
		return err
	}
	cmpAfter := keyComparisons(after, keyID)
	uniqueSame := slices.ContainsFunc(cmpAfter, func(k ercrossview.KeyComparison) bool {
		return k.ConstraintKind == "unique" && k.ConstraintName == c.Decl.UniqueDecision.Name && k.Status == "same-attribute-set"
	})
	if c.Decl.UniqueDecision.Declare && !uniqueSame {
		return fmt.Errorf("step 3: declared unique constraint does not match the business key: %+v", cmpAfter)
	}
	variant, errs, err := LoadValidated(c.VariantDir)
	if err != nil || len(errs) > 0 {
		return fmt.Errorf("step 3 variant: %v %v", err, errs)
	}
	variantView, err := ercrossview.Project(variant)
	if err != nil {
		return err
	}
	c.worldFinal, c.worldVariant = final, variant
	c.final = r3
	c.Rec.Revisions.R3 = r3
	c.addStep(StepRecord{
		Step: 3, Name: "論理ERと物理表の対応、一意制約の判断", ModelRevision: r2, RevisionAfter: r3,
		Assets: []AssetUse{
			{Asset: "tools/reference-er-table-crossview", Called: true, How: "Goの取込み: Project(判断の前後と、判断なしの版)"},
			{Asset: "tools/reference-shared-acceptance", Called: false, Reason: "保存写像へ一意制約を加える型付き操作がない。保存写像への直接編集として記録し、受理は未接続"},
		},
		Output: map[string]any{
			"keyComparisonsBeforeDecision": cmpBefore,
			"keyComparisonsAfterDecision":  cmpAfter,
			"primaryKeyNotTakenAsBusinessKey": true,
			"uniqueDecision":               c.Decl.UniqueDecision,
			"directEdit":                   edit,
			"unresolvedPassedThrough":      after.Unresolved,
			"variantKeyComparisons":        keyComparisons(variantView, keyID),
			"structureRevision":            after.StructureRevision,
		},
	})
	c.gap("コア", "保存写像へ一意制約を宣言する受理の操作がない(候補キーは共通受理で受理できる)", "工程3の保存写像への直接編集")
	return nil
}

// Step4StandardFunctions derives the (non-executable) standard function
// candidates from the final version and keeps their missing items.
func (c *Chain) Step4StandardFunctions(ctx context.Context) error {
	if err := c.require(4, c.WorkDir); err != nil {
		return err
	}
	raw, err := runTool(ctx, c.Tools.StandardFunctions, "-workspace", c.WorkDir, "-application", "order_entry")
	if err != nil {
		return fmt.Errorf("step 4: %w", err)
	}
	var report struct {
		Schema     string `json:"schemaVersion"`
		Authority  string `json:"authority"`
		Candidates []struct {
			DerivedID  string `json:"derivedId"`
			Intent     string `json:"intent"`
			Data       string `json:"data"`
			Executable bool   `json:"executable"`
			Fields     []struct {
				ID string `json:"id"`
			} `json:"fields"`
			Missing []string `json:"missing"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		return fmt.Errorf("step 4: %w", err)
	}
	c.missing = map[string][]string{}
	var orderCandidates []map[string]any
	for _, cand := range report.Candidates {
		if cand.Data != c.Decl.Attribute.Data {
			continue
		}
		if cand.Executable {
			return fmt.Errorf("step 4: candidate %s claims to be executable", cand.DerivedID)
		}
		hasAttr := slices.ContainsFunc(cand.Fields, func(f struct {
			ID string `json:"id"`
		}) bool {
			return f.ID == c.Decl.Attribute.Attribute
		})
		if !hasAttr {
			return fmt.Errorf("step 4: candidate %s lacks %s", cand.DerivedID, c.Decl.Attribute.Attribute)
		}
		c.missing[cand.Intent] = cand.Missing
		orderCandidates = append(orderCandidates, map[string]any{"derivedId": cand.DerivedID, "intent": cand.Intent, "missing": cand.Missing, "hasAddedAttribute": true})
	}
	if len(c.missing["create"]) == 0 {
		return fmt.Errorf("step 4: no create candidate for %s", c.Decl.Attribute.Data)
	}
	c.addStep(StepRecord{
		Step: 4, Name: "標準機能候補の生成", ModelRevision: c.final,
		Assets: []AssetUse{{Asset: "tools/reference-standard-function-candidates", Called: true, How: "go buildした実行ファイル: -workspace <複製> -application order_entry"}},
		Output: map[string]any{
			"schema": report.Schema, "authority": report.Authority,
			"orderCandidates": orderCandidates,
			"note":            "候補は実行しない。登録(工程6・9)は不足項目(capability、binding、persistence、provider、authority、independent-read)を手書きのアダプターと独立の読取りで補う",
		},
	})
	return nil
}

func keyComparisons(p ercrossview.Projection, keyID string) []ercrossview.KeyComparison {
	var out []ercrossview.KeyComparison
	for _, k := range p.KeyComparisons {
		if k.LogicalKeyID == keyID {
			out = append(out, k)
		}
	}
	return out
}

func mustJSON(v any) json.RawMessage {
	raw, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return raw
}

func writeJSON(path string, v any) error {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}
