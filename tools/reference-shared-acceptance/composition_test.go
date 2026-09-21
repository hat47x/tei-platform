package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"tei/internal/syntax"
	"tei/tools/internal/changeanalysis"
)

// 改訂版S4 Proposal（plan/SECOND_APPLICATION_S4_FULL_VALIDATION_PREFLIGHT_2026-09.md）の3 source。
// 各componentは末尾に改行を含めない。
const (
	s4Semantic = `namespace repair_intake

data Intake {
    attribute customerId type customer.customer_id
    relation customer to customer.customer one
        key repair_intake.intake.customer_id -> customer.customer.customer_id
}`

	s4Application = `application repair_intake {
    uses semantic repair_intake
    uses semantic customer

    database customer_master {
        target master_data
        access read
        requires {
            contract master_data.customer ^2
        }
    }
}`

	s4Pattern = `pattern repair_intake.customer_reference reference {
    relation repair_intake.intake.customer
    datasource customer_master
    contract-query {
        contract master_data.customer
        query find_by_id
        input customer.customer.customer_id -> id
    }
}`

	s4SemanticSHA256    = "1005c6deb670c277c8b80fb0b353f1eecea690594f2ef4b80df28cc638736e63"
	s4ApplicationSHA256 = "f96fdb46f9385c205d360f56d0f91a6e841ff31b348ebf1db2cf03381751772f"
	s4PatternSHA256     = "41c8f81ebab384486d4df6a7cdaade07df8d07c89d22c25c2291d8b71a583ab1"
	s4BundleSHA256      = "c18ae0fdef8e01aa965e2ad84c86ecfc21ca0608c07b1808e73368a5f434458d"

	s4SemanticPath    = "semantic/repair-intake/intake.tei"
	s4ApplicationPath = "applications/repair-intake/application.tei"
	s4PatternPath     = "applications/repair-intake/patterns/customer-reference.tei"
)

func sha256Hex(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

func s4Bundle(semantic, application, pattern string) string {
	return semantic + "\n---\n" + application + "\n---\n" + pattern
}

func secondApplicationCandidate(revision string) Candidate {
	return Candidate{
		Schema:            candidateSchema,
		ID:                "cand-second-application-repair-intake",
		BaseModelRevision: revision,
		Origin:            Origin{Kind: "cli", Actor: "test"},
		Intent:            "Second Application repair_intake（改訂版S4 composition）をSemantic / Application / Patternの3 sourceとして追加する",
		Evidence: []Evidence{
			{Subject: "order_entry.customer_reference", Note: "origin。provenanceだけで、意味はcopyしない"},
			{Subject: "customer.customer"},
		},
		Delta: []Operation{
			{
				Op: opAddData, Placement: s4SemanticPath,
				Data: &DataSpec{
					Namespace: "repair_intake", ID: "repair_intake.intake", Symbol: "Intake",
					Attributes: []AttributeSpec{{ID: "repair_intake.intake.customer_id", Symbol: "customerId", Type: "customer.customer_id"}},
					Relations: []RelationSpec{{
						ID: "repair_intake.intake.customer", Symbol: "customer", Target: "customer.customer", Cardinality: "one",
						Keys: []RelationKeySpec{{SourceAttribute: "repair_intake.intake.customer_id", TargetAttribute: "customer.customer.customer_id"}},
					}},
				},
			},
			{
				Op: opAddApplication, Placement: s4ApplicationPath,
				Application: &ApplicationSpec{
					ID: "repair_intake", UsesSemantic: []string{"repair_intake", "customer"},
					Databases: []DatabaseSpec{{
						Alias: "customer_master", Target: "master_data", Access: "read",
						Contracts: []ContractRequirementSpec{{Contract: "master_data.customer", Major: 2}},
					}},
				},
			},
			{
				Op: opAddReferencePattern, Placement: s4PatternPath,
				Pattern: &PatternSpec{
					ID: "repair_intake.customer_reference", Application: "repair_intake",
					Relation: "repair_intake.intake.customer", Datasource: "customer_master",
					ContractQuery: ContractQuerySpec{
						Contract: "master_data.customer", Query: "find_by_id",
						Inputs: []QueryInputSpec{{Attribute: "customer.customer.customer_id", Input: "id"}},
					},
				},
			},
		},
	}
}

func previewComposition(t *testing.T, mutate func(c *Candidate)) (string, evaluation) {
	t.Helper()
	dir := scratchWorkspaceBeforeSecondApplication(t)
	c := secondApplicationCandidate(revisionOfDir(t, dir))
	if mutate != nil {
		mutate(&c)
	}
	ev, err := evaluate(dir, mustJSON(t, c))
	if err != nil {
		t.Fatal(err)
	}
	return dir, ev
}

func keysOf(changes []changeanalysis.SubjectChange) []string {
	var out []string
	for _, c := range changes {
		out = append(out, c.Change+" "+c.Key())
	}
	return out
}

func TestSecondApplicationCompositionPreviewUsesSharedAcceptanceBoundary(t *testing.T) {
	_, ev := previewComposition(t, nil)
	p := ev.preview
	if !p.Acceptable {
		t.Fatalf("Previewが受理可能になりません: %+v", allDiagnostics(p))
	}
	var names []string
	for _, s := range p.Stages {
		names = append(names, s.Name)
		if s.Status != "pass" {
			t.Fatalf("stage %s=%s", s.Name, s.Status)
		}
	}
	if !reflect.DeepEqual(names, wantStages) {
		t.Fatalf("add-validationと同じstage順序である必要があります: %v want %v", names, wantStages)
	}
	if len(p.SourceEdits) != 3 {
		t.Fatalf("source edit=%+v", p.SourceEdits)
	}
	var paths []string
	for _, e := range p.SourceEdits {
		paths = append(paths, e.Path)
		if !e.Created {
			t.Fatalf("destination fileは新しく作る必要があります: %+v", e)
		}
	}
	want := []string{s4ApplicationPath, s4PatternPath, s4SemanticPath}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("editのpath=%v want %v", paths, want)
	}
	if !reflect.DeepEqual(p.ChangedCollections, []string{"applications", "semantics"}) {
		t.Fatalf("changed collections=%v", p.ChangedCollections)
	}
	for _, unchanged := range []string{"databases", "persistenceMappings", "migrations", "environments", "queries", "customGoBindings", "applicationReleases", "databaseReleases", "solutionReleases"} {
		if !hasCode(p.UnchangedCollections, unchanged) {
			t.Fatalf("collection %s が不変として示されていません: %v", unchanged, p.UnchangedCollections)
		}
	}
}

func TestSecondApplicationCompositionPreviewLeavesSourceByteIdentical(t *testing.T) {
	dir := scratchWorkspaceBeforeSecondApplication(t)
	before := treeDigest(t, dir)
	ev, err := evaluate(dir, mustJSON(t, secondApplicationCandidate(revisionOfDir(t, dir))))
	if err != nil || !ev.preview.Acceptable {
		t.Fatalf("preview=%+v err=%v", ev.preview, err)
	}
	assertTreeUnchanged(t, before, treeDigest(t, dir))
	for _, created := range []string{"semantic/repair-intake", "applications/repair-intake"} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(created))); err == nil {
			t.Fatalf("Previewが作成先directory %s を作っています", created)
		}
	}
}

func TestSecondApplicationCompositionPreviewBindsExactS4Proposal(t *testing.T) {
	// 1. S4のthree-source payloadを再計算し、文書のidentityと一致させる。
	if sha256Hex(s4Semantic) != s4SemanticSHA256 || sha256Hex(s4Application) != s4ApplicationSHA256 || sha256Hex(s4Pattern) != s4PatternSHA256 {
		t.Fatal("S4のcomponent sourceがproposal recordと一致しません")
	}
	bundle := s4Bundle(s4Semantic, s4Application, s4Pattern)
	if len(bundle) != 741 || sha256Hex(bundle) != s4BundleSHA256 {
		t.Fatalf("S4 bundle length=%d sha256=%s", len(bundle), sha256Hex(bundle))
	}

	// 2. S4のsourceを既存parserで読んだ意味から、型付きoperationを生成する。
	namespace, err := syntax.NewParser(s4Semantic).ParseSemanticFile()
	if err != nil {
		t.Fatal(err)
	}
	app, err := syntax.NewParser(s4Application).ParseApplication()
	if err != nil {
		t.Fatal(err)
	}
	pattern, err := syntax.NewParser(s4Pattern).ParseApplicationPattern()
	if err != nil {
		t.Fatal(err)
	}
	if len(namespace.Data) != 1 {
		t.Fatalf("S4 semantic data=%d", len(namespace.Data))
	}
	d := namespace.Data[0]
	data := DataSpec{Namespace: string(namespace.ID), ID: string(d.ID), Symbol: d.Symbol}
	for _, a := range d.Attributes {
		data.Attributes = append(data.Attributes, AttributeSpec{ID: string(a.ID), Symbol: a.Symbol, Type: string(a.TypeRef)})
	}
	for _, r := range d.Relations {
		spec := RelationSpec{ID: string(r.ID), Symbol: r.Symbol, Target: string(r.Target), Cardinality: string(r.Cardinality)}
		for _, k := range r.Keys {
			spec.Keys = append(spec.Keys, RelationKeySpec{SourceAttribute: string(k.SourceAttribute), TargetAttribute: string(k.TargetAttribute)})
		}
		data.Relations = append(data.Relations, spec)
	}
	application := ApplicationSpec{ID: string(app.ID)}
	for _, u := range app.SemanticUses {
		application.UsesSemantic = append(application.UsesSemantic, string(u))
	}
	for _, db := range app.Databases {
		spec := DatabaseSpec{Alias: string(db.Alias), Target: string(db.TargetDatabase), Access: string(db.Access)}
		for _, c := range db.RequiredContracts {
			spec.Contracts = append(spec.Contracts, ContractRequirementSpec{Contract: string(c.ContractID), Major: c.CompatibleMajor})
		}
		application.Databases = append(application.Databases, spec)
	}
	cq := ContractQuerySpec{Contract: string(pattern.ContractQuery.Contract), Query: string(pattern.ContractQuery.Query)}
	for _, in := range pattern.ContractQuery.Inputs {
		cq.Inputs = append(cq.Inputs, QueryInputSpec{Attribute: string(in.TargetAttribute), Input: string(in.Input)})
	}
	parsedOps := []Operation{
		{Op: opAddData, Placement: s4SemanticPath, Data: &data},
		{Op: opAddApplication, Placement: s4ApplicationPath, Application: &application},
		{Op: opAddReferencePattern, Placement: s4PatternPath, Pattern: &PatternSpec{
			ID: string(pattern.ID), Application: string(app.ID), Relation: string(pattern.Relation),
			Datasource: string(pattern.Datasource), ContractQuery: cq,
		}},
	}

	dir := scratchWorkspaceBeforeSecondApplication(t)
	revision := revisionOfDir(t, dir)
	fromParsed := secondApplicationCandidate(revision)
	fromParsed.Delta = parsedOps
	built := secondApplicationCandidate(revision)
	parsedNormalized, parsedFingerprint, err := normalizeCandidate(fromParsed)
	if err != nil {
		t.Fatal(err)
	}
	builtNormalized, builtFingerprint, err := normalizeCandidate(built)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(parsedNormalized.Delta, builtNormalized.Delta) || parsedFingerprint != builtFingerprint {
		t.Fatalf("S4をparseした意味から作った型付きoperationが、candidateのoperationと一致しません:\n%+v\n%+v", parsedNormalized.Delta, builtNormalized.Delta)
	}

	// 3. 型付きoperationから決定論的にsourceを再生成し、4. payloadがS4 identityと一致することを確認する。
	regenerated := map[string]string{}
	for _, op := range builtNormalized.Delta {
		text, ok := renderNewFile(op)
		if !ok {
			t.Fatalf("新しいfileを作らないoperation: %+v", op)
		}
		regenerated[op.Placement] = text
	}
	payload := s4Bundle(regenerated[s4SemanticPath], regenerated[s4ApplicationPath], regenerated[s4PatternPath])
	if sha256Hex(payload) != s4BundleSHA256 {
		t.Fatalf("再生成したthree-source payloadがS4 Proposal identityと一致しません: %s", sha256Hex(payload))
	}

	// Previewが示すsourceも、同じS4のcomponent（末尾の改行一つを除く）である。
	ev, err := evaluate(dir, mustJSON(t, built))
	if err != nil || !ev.preview.Acceptable {
		t.Fatalf("preview=%+v err=%v", ev.preview, err)
	}
	inserted := map[string]string{}
	for _, e := range ev.preview.SourceEdits {
		inserted[e.Path] = strings.TrimSuffix(e.InsertedText, "\n")
	}
	if s4Bundle(inserted[s4SemanticPath], inserted[s4ApplicationPath], inserted[s4PatternPath]) != s4Bundle(s4Semantic, s4Application, s4Pattern) {
		t.Fatal("Previewが示すsourceがS4 Proposalと一致しません")
	}

	// 5. 現在のbaseModelRevisionを加えたAcceptance candidate identityは、S4 identityとは別物として取得できる。
	if ev.preview.CandidateFingerprint != builtFingerprint || strings.Contains(ev.preview.CandidateFingerprint, s4BundleSHA256) {
		t.Fatalf("candidate fingerprint=%s", ev.preview.CandidateFingerprint)
	}
	other := secondApplicationCandidate("sha256:" + strings.Repeat("1", 64))
	_, otherFingerprint, err := normalizeCandidate(other)
	if err != nil || otherFingerprint == builtFingerprint {
		t.Fatalf("baseModelRevisionがAcceptance candidate identityへ反映されていません: %s %v", otherFingerprint, err)
	}
}

func TestSecondApplicationCompositionPreviewReportsSemanticAndApplicationDiff(t *testing.T) {
	_, ev := previewComposition(t, nil)
	wantSemantic := []string{
		"added attribute:repair_intake.intake.customer_id",
		"added data:repair_intake.intake",
		"added namespace:repair_intake",
		"added relation:repair_intake.intake.customer",
	}
	wantApplication := []string{
		"added application:repair_intake",
		"added application-pattern:repair_intake.customer_reference",
	}
	if got := keysOf(ev.preview.SemanticDiff); !reflect.DeepEqual(got, wantSemantic) {
		t.Fatalf("semantic diff=%v want %v", got, wantSemantic)
	}
	if got := keysOf(ev.preview.ApplicationDiff); !reflect.DeepEqual(got, wantApplication) {
		t.Fatalf("application diff=%v want %v", got, wantApplication)
	}
	if n := len(ev.preview.SemanticDiff) + len(ev.preview.ApplicationDiff); n != 6 || n > maxSubjectChanges {
		t.Fatalf("変更subject数=%d", n)
	}

	edges := map[string]bool{}
	for _, e := range ev.preview.Impact {
		edges[e.From+" -> "+e.To+" via "+e.Via] = true
	}
	for _, want := range []string{
		"application-pattern:repair_intake.customer_reference -> application:repair_intake via application.pattern.owner",
		"application-pattern:repair_intake.customer_reference -> relation:repair_intake.intake.customer via application.pattern.relation",
		"application-pattern:repair_intake.customer_reference -> contract-query:master_data.customer/find_by_id via application.pattern.contract-query",
		"application:repair_intake -> namespace:repair_intake via application.uses-semantic-namespace",
		"application:repair_intake -> contract:master_data.customer via application.database.requires-contract",
		"relation:repair_intake.intake.customer -> data:customer.customer via semantic.relation.target",
	} {
		if !edges[want] {
			t.Fatalf("impact edge %q がありません: %v", want, ev.preview.Impact)
		}
	}
	var codes []string
	for _, u := range ev.preview.Unresolved {
		codes = append(codes, u.Code)
	}
	for _, want := range []string{"ENVIRONMENT_BINDING_NOT_AUTHORED", "DRIVER_SCENARIO_NOT_RUN", "TARGET_PROJECTION_NOT_EVALUATED"} {
		if !hasCode(codes, want) {
			t.Fatalf("unresolved %s が残っていません: %v", want, codes)
		}
	}
}

func TestSecondApplicationCompositionPreviewRejectsStaleBase(t *testing.T) {
	dir := scratchWorkspaceBeforeSecondApplication(t)
	staleRevision := revisionOfDir(t, dir)
	semanticFile := filepath.Join(dir, "semantic", "sales", "order.tei")
	content, err := os.ReadFile(semanticFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(semanticFile, append(content, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	before := treeDigest(t, dir)

	data := mustJSON(t, secondApplicationCandidate(staleRevision))
	ev, err := evaluate(dir, data)
	if err != nil {
		t.Fatal(err)
	}
	if ev.preview.Acceptable || ev.preview.Stale == nil || !hasCode(diagnosticCodes(ev.preview), "STALE_BASE_MODEL_REVISION") {
		t.Fatalf("staleなbaseが検出されていません: %+v", ev.preview.Stages)
	}
	decision, err := accept(dir, data, acceptOptions{})
	if err != nil || decision.Decision != "refused" || decision.SourceChanged {
		t.Fatalf("decision=%+v err=%v", decision, err)
	}
	assertTreeUnchanged(t, before, treeDigest(t, dir))
}

func TestSecondApplicationCompositionPreviewRejectsExistingDestinationCollision(t *testing.T) {
	for _, tc := range []struct {
		name string
		path string
		body string
	}{
		{"semantic", s4SemanticPath, "namespace other_space\n"},
		{"application", s4ApplicationPath, "application other_app {\n    uses semantic customer\n}\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := scratchWorkspaceBeforeSecondApplication(t)
			writeFile(t, filepath.Join(dir, filepath.FromSlash(tc.path)), tc.body)
			before := treeDigest(t, dir)
			data := mustJSON(t, secondApplicationCandidate(revisionOfDir(t, dir)))

			ev, err := evaluate(dir, data)
			if err != nil {
				t.Fatal(err)
			}
			if ev.preview.Acceptable || !hasCode(diagnosticCodes(ev.preview), "DESTINATION_ALREADY_EXISTS") {
				t.Fatalf("既存のdestinationとの衝突が検出されていません: %v", diagnosticCodes(ev.preview))
			}
			if got := stageStatus(t, ev.preview, "resolve").Status; got != "fail" {
				t.Fatalf("resolve=%s", got)
			}
			decision, err := accept(dir, data, acceptOptions{})
			if err != nil || decision.Decision != "refused" {
				t.Fatalf("decision=%+v err=%v", decision, err)
			}
			assertTreeUnchanged(t, before, treeDigest(t, dir))
		})
	}
}

func TestSecondApplicationCompositionPreviewRejectsUndeclaredFourthEdit(t *testing.T) {
	dir := scratchWorkspaceBeforeSecondApplication(t)
	ev, err := evaluate(dir, mustJSON(t, secondApplicationCandidate(revisionOfDir(t, dir))))
	if err != nil || !ev.preview.Acceptable {
		t.Fatalf("preview=%+v err=%v", ev.preview, err)
	}
	for _, tc := range []struct {
		name     string
		touch    string
		want     []string
		wantNone []string
	}{
		{"shared Customer source", "semantic/customer/customer.tei", []string{"UNEXPECTED_SOURCE_CHANGE"}, []string{"LAYER_VIOLATION"}},
		{"origin order_entry", "applications/order-entry/application.tei", []string{"UNEXPECTED_SOURCE_CHANGE"}, []string{"LAYER_VIOLATION"}},
		{"Environment", "environments/development/environment.tei", []string{"UNEXPECTED_SOURCE_CHANGE", "LAYER_VIOLATION"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// 宣言済みの3 editを適用し、さらに宣言していない4つ目のfileを書き換えたworkspaceを作る。
			overlay := filepath.Join(t.TempDir(), "overlay")
			if err := copyTree(dir, overlay); err != nil {
				t.Fatal(err)
			}
			for _, e := range ev.edits {
				writeFile(t, filepath.Join(overlay, filepath.FromSlash(e.Path)), string(e.after))
			}
			fourth := filepath.Join(overlay, filepath.FromSlash(tc.touch))
			content, err := os.ReadFile(fourth)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(fourth, append(content, '\n'), 0o644); err != nil {
				t.Fatal(err)
			}
			overlaySnapshot, err := takeSnapshot(overlay)
			if err != nil {
				t.Fatal(err)
			}
			got := checkContainment(ev.before, overlaySnapshot, ev.edits)
			var codes []string
			for _, d := range got {
				codes = append(codes, d.Code)
			}
			for _, want := range tc.want {
				if !hasCode(codes, want) {
					t.Fatalf("%s が検出されていません: %+v", want, got)
				}
			}
			for _, none := range tc.wantNone {
				if hasCode(codes, none) {
					t.Fatalf("%s は出ないはずです: %+v", none, got)
				}
			}
		})
	}
}

func TestSecondApplicationCompositionPreviewRejectsMissingPattern(t *testing.T) {
	t.Run("pattern without its owner application", func(t *testing.T) {
		dir, ev := previewComposition(t, func(c *Candidate) {
			c.Delta = []Operation{c.Delta[0], c.Delta[2]}
		})
		if ev.preview.Acceptable || !hasCode(diagnosticCodes(ev.preview), "PATTERN_OWNER_APPLICATION_NOT_FOUND") {
			t.Fatalf("owner applicationの無いPatternが受理可能です: %v", diagnosticCodes(ev.preview))
		}
		if _, err := os.Stat(filepath.Join(dir, "applications", "repair-intake")); err == nil {
			t.Fatal("拒否したのにdirectoryが作られています")
		}
	})
	t.Run("pattern outside its owner directory", func(t *testing.T) {
		_, ev := previewComposition(t, func(c *Candidate) {
			c.Delta[2].Placement = "applications/other-dir/patterns/customer-reference.tei"
		})
		if ev.preview.Acceptable || !hasCode(diagnosticCodes(ev.preview), "PATTERN_OWNER_DIRECTORY_MISMATCH") {
			t.Fatalf("owner directoryの外のPatternが受理可能です: %v", diagnosticCodes(ev.preview))
		}
	})
	t.Run("placement that the loader would not read", func(t *testing.T) {
		_, ev := previewComposition(t, func(c *Candidate) {
			c.Delta[2].Placement = "applications/repair-intake/customer-reference.tei"
		})
		if ev.preview.Acceptable || !hasCode(diagnosticCodes(ev.preview), "PLACEMENT_SHAPE_INVALID") {
			t.Fatalf("loaderが読まないplacementが受理可能です: %v", diagnosticCodes(ev.preview))
		}
	})
}

func TestSecondApplicationCompositionPreviewRejectsUnknownRelation(t *testing.T) {
	dir, ev := previewComposition(t, func(c *Candidate) {
		c.Delta[2].Pattern.Relation = "repair_intake.intake.unknown"
	})
	if ev.preview.Acceptable {
		t.Fatal("存在しないRelationを指すPatternが受理可能です")
	}
	validate := stageStatus(t, ev.preview, "validate")
	if validate.Status != "fail" {
		t.Fatalf("validate=%s: %+v", validate.Status, allDiagnostics(ev.preview))
	}
	var codes []string
	for _, d := range validate.Diagnostics {
		codes = append(codes, d.Code)
	}
	if !hasCode(codes, "UNKNOWN_APPLICATION_PATTERN_RELATION") {
		t.Fatalf("通常のValidation diagnosticへ到達していません: %v", codes)
	}
	if _, err := os.Stat(filepath.Join(dir, "applications", "repair-intake")); err == nil {
		t.Fatal("拒否したのにdirectoryが作られています")
	}
}

func TestSecondApplicationCompositionPreviewRejectsCardinalityMismatch(t *testing.T) {
	_, ev := previewComposition(t, func(c *Candidate) {
		c.Delta[0].Data.Relations[0].Cardinality = "many"
	})
	if ev.preview.Acceptable {
		t.Fatal("many RelationとzeroOrOne Query resultの不一致が受理可能です")
	}
	var codes []string
	for _, d := range stageStatus(t, ev.preview, "validate").Diagnostics {
		codes = append(codes, d.Code)
	}
	if !hasCode(codes, "REFERENCE_CONTRACT_QUERY_RESULT_CARDINALITY_MISMATCH") {
		t.Fatalf("通常のValidation diagnosticへ到達していません: %v", codes)
	}
}

func TestSecondApplicationCompositionPreviewRejectsUndeclaredCollectionChange(t *testing.T) {
	normalized, _, err := normalizeCandidate(secondApplicationCandidate("sha256:" + strings.Repeat("0", 64)))
	if err != nil {
		t.Fatal(err)
	}
	wantCollections := expectedCollections(normalized.Delta)
	if !reflect.DeepEqual(wantCollections, []string{"applications", "semantics"}) {
		t.Fatalf("deltaから導かれるcollection=%v", wantCollections)
	}
	if got := expectedCollections([]Operation{{Op: opAddValidation}}); !reflect.DeepEqual(got, []string{"semantics"}) {
		t.Fatalf("add-validationのcollection=%v", got)
	}

	expected := expectedChanges(normalized.Delta)
	for name, changed := range map[string][]string{
		"an undeclared collection":       {"applications", "environments", "semantics"},
		"persistence":                    {"applications", "persistenceMappings", "semantics"},
		"fewer than the declared layers": {"semantics"},
		"application layer only":         {"applications"},
	} {
		got := checkExpectedDelta(expected, expected, changed, wantCollections)
		if len(got) != 1 || got[0].Code != "UNEXPECTED_COLLECTION_CHANGE" {
			t.Fatalf("%s: %+v", name, got)
		}
	}
	if got := checkExpectedDelta(expected, expected, []string{"applications", "semantics"}, wantCollections); len(got) != 0 {
		t.Fatalf("宣言どおりのcollection変更が拒否されました: %+v", got)
	}
}

func TestSecondApplicationCompositionOriginDoesNotChangeTheDecision(t *testing.T) {
	origins := []Origin{
		{Kind: "file"},
		{Kind: "cli", Actor: "developer"},
		{Kind: "studio", Actor: "studio-session-7"},
		{Kind: "ai", Actor: "assistant-x", Notes: map[string]string{"model": "model-x-2026", "promptDigest": "sha256:feedface"}},
	}
	type outcome struct {
		fingerprint string
		after       string
		derivedSHA  string
		inserted    map[string]string
		tree        map[string]string
	}
	var outcomes []outcome
	for _, origin := range origins {
		dir := scratchWorkspaceBeforeSecondApplication(t)
		c := secondApplicationCandidate(revisionOfDir(t, dir))
		c.ID = "cand-" + origin.Kind
		c.Origin = origin
		decision, err := accept(dir, mustJSON(t, c), acceptOptions{})
		if err != nil || decision.Decision != "accepted" {
			t.Fatalf("origin=%s decision=%+v err=%v", origin.Kind, decision, err)
		}
		inserted := map[string]string{}
		for _, e := range decision.SourceEdits {
			inserted[e.Path] = e.InsertedText
		}
		for path, content := range readTree(t, dir) {
			for _, private := range []string{"model-x-2026", "sha256:feedface", "studio-session-7", "assistant-x"} {
				if strings.Contains(content, private) {
					t.Fatalf("origin固有のEvidence %q がsource %s へ入っています", private, path)
				}
			}
		}
		outcomes = append(outcomes, outcome{decision.CandidateFingerprint, decision.ModelRevisionAfter, decision.DerivedWorldSHA256, inserted, treeDigest(t, dir)})
	}
	for i := 1; i < len(outcomes); i++ {
		if !reflect.DeepEqual(outcomes[0], outcomes[i]) {
			t.Fatalf("入口が%sと%sで、受入結果が異なります", origins[0].Kind, origins[i].Kind)
		}
	}
}

func TestSecondApplicationCompositionAcceptWritesExactlyTheThreeDeclaredFiles(t *testing.T) {
	dir := scratchWorkspaceBeforeSecondApplication(t)
	before := treeDigest(t, dir)
	beforeEnvironments := map[string]string{}
	for path, digest := range before {
		if strings.HasPrefix(path, "environments/") {
			beforeEnvironments[path] = digest
		}
	}
	revision := revisionOfDir(t, dir)
	data := mustJSON(t, secondApplicationCandidate(revision))

	ev, err := evaluate(dir, data)
	if err != nil || !ev.preview.Acceptable {
		t.Fatalf("preview=%+v err=%v", ev.preview, err)
	}
	decision, err := accept(dir, data, acceptOptions{})
	if err != nil || decision.Decision != "accepted" || !decision.SourceChanged {
		t.Fatalf("decision=%+v err=%v", decision, err)
	}

	after := treeDigest(t, dir)
	var added []string
	for path, digest := range after {
		old, existed := before[path]
		switch {
		case !existed:
			added = append(added, path)
		case old != digest:
			t.Fatalf("既存のfile %s が変更されています", path)
		}
	}
	if len(after) != len(before)+3 {
		t.Fatalf("file数 before=%d after=%d", len(before), len(after))
	}
	wantAdded := map[string]bool{s4SemanticPath: true, s4ApplicationPath: true, s4PatternPath: true}
	for _, path := range added {
		if !wantAdded[path] {
			t.Fatalf("宣言していないfileが作られています: %s", path)
		}
	}
	for path, digest := range beforeEnvironments {
		if after[path] != digest {
			t.Fatalf("Environment file %s が変更されています", path)
		}
	}
	for _, c := range decision.Conformance {
		if !c.Passed {
			t.Fatalf("受理後のConformance %s が通っていません", c.Name)
		}
	}
	if !reflect.DeepEqual(decision.SemanticDiff, ev.preview.SemanticDiff) || !reflect.DeepEqual(decision.ApplicationDiff, ev.preview.ApplicationDiff) {
		t.Fatal("受理した差分がPreviewの差分と一致しません")
	}
	if decision.ModelRevisionAfter != ev.preview.ModelRevisionAfter {
		t.Fatalf("受理後のrevision=%s Preview=%s", decision.ModelRevisionAfter, ev.preview.ModelRevisionAfter)
	}

	// 書かれたsourceは、S4 Proposalそのもの（末尾の改行一つを除く）である。
	read := func(path string) string {
		content, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		return strings.TrimSuffix(string(content), "\n")
	}
	if sha256Hex(s4Bundle(read(s4SemanticPath), read(s4ApplicationPath), read(s4PatternPath))) != s4BundleSHA256 {
		t.Fatal("受理されたsourceがS4 Proposalと一致しません")
	}
	uuid := regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	for _, path := range []string{s4SemanticPath, s4ApplicationPath, s4PatternPath} {
		if uuid.MatchString(read(path)) {
			t.Fatalf("authoredされたsourceへUUIDが入っています: %s", path)
		}
	}
}

func TestSecondApplicationCompositionAcceptRollsAllThreeFilesBackWhenConformanceFails(t *testing.T) {
	dir := scratchWorkspaceBeforeSecondApplication(t)
	before := treeDigest(t, dir)
	data := mustJSON(t, secondApplicationCandidate(revisionOfDir(t, dir)))

	afterWritesHook = func(root string) {
		path := filepath.Join(root, filepath.FromSlash(s4PatternPath))
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(content, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	defer func() { afterWritesHook = nil }()

	decision, err := accept(dir, data, acceptOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Decision != "refused" || decision.SourceChanged || !hasCode(refusalCodes(decision), "POST_ACCEPT_CONFORMANCE_FAILED") {
		t.Fatalf("decision=%+v", decision)
	}
	assertTreeUnchanged(t, before, treeDigest(t, dir))
	for _, created := range []string{"semantic/repair-intake", "applications/repair-intake", "applications/repair-intake/patterns"} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(created))); err == nil {
			t.Fatalf("rollback後に作成したdirectory %s が残っています", created)
		}
	}
	for _, kept := range []string{"semantic", "applications"} {
		if _, err := os.Stat(filepath.Join(dir, kept)); err != nil {
			t.Fatalf("元からあるdirectory %s が消えています", kept)
		}
	}
}

func TestSecondApplicationCompositionAcceptRefusesAConcurrentCreateWithoutPartialWrite(t *testing.T) {
	dir := scratchWorkspaceBeforeSecondApplication(t)
	data := mustJSON(t, secondApplicationCandidate(revisionOfDir(t, dir)))
	concurrent := filepath.Join(dir, filepath.FromSlash(s4PatternPath))

	// edit順はpath順: [0] application.tei, [1] patterns/customer-reference.tei, [2] intake.tei
	beforeEachWriteHook = func(index int, edit SourceEdit) {
		if index == 1 {
			writeFile(t, concurrent, "pattern concurrently.created reference {\n}\n")
		}
	}
	defer func() { beforeEachWriteHook = nil }()

	decision, err := accept(dir, data, acceptOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Decision != "refused" || decision.SourceChanged || !hasCode(refusalCodes(decision), "SOURCE_CHANGED_DURING_ACCEPT") {
		t.Fatalf("decision=%+v", decision)
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(s4ApplicationPath))); err == nil {
		t.Fatal("先に書いたapplication.teiが元へ戻されていません（partial writeが残っています）")
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(s4SemanticPath))); err == nil {
		t.Fatal("後続のfileが書かれています")
	}
	content, err := os.ReadFile(concurrent)
	if err != nil || !strings.Contains(string(content), "concurrently.created") {
		t.Fatalf("並行して作られたfileを消して/上書きしてしまいました: %v", err)
	}
}

func TestMalformedCompositionOperationsAreRefusedByNormalization(t *testing.T) {
	revision := "sha256:" + strings.Repeat("a", 64)
	cases := map[string]func(c *Candidate){
		"add-data without placement":       func(c *Candidate) { c.Delta[0].Placement = "" },
		"add-data id that does not match":  func(c *Candidate) { c.Delta[0].Data.ID = "repair_intake.other" },
		"add-data symbol with a dash":      func(c *Candidate) { c.Delta[0].Data.Symbol = "In-take" },
		"attribute id that does not match": func(c *Candidate) { c.Delta[0].Data.Attributes[0].ID = "repair_intake.intake.customer" },
		"relation cardinality":             func(c *Candidate) { c.Delta[0].Data.Relations[0].Cardinality = "few" },
		"duplicate attribute": func(c *Candidate) {
			c.Delta[0].Data.Attributes = append(c.Delta[0].Data.Attributes, c.Delta[0].Data.Attributes[0])
		},
		"application without usesSemantic":      func(c *Candidate) { c.Delta[1].Application.UsesSemantic = nil },
		"duplicate usesSemantic":                func(c *Candidate) { c.Delta[1].Application.UsesSemantic = []string{"customer", "customer"} },
		"database access":                       func(c *Candidate) { c.Delta[1].Application.Databases[0].Access = "sudo" },
		"database alias not a local name":       func(c *Candidate) { c.Delta[1].Application.Databases[0].Alias = "Customer Master" },
		"contract major zero":                   func(c *Candidate) { c.Delta[1].Application.Databases[0].Contracts[0].Major = 0 },
		"pattern without datasource":            func(c *Candidate) { c.Delta[2].Pattern.Datasource = "" },
		"pattern input not a local name":        func(c *Candidate) { c.Delta[2].Pattern.ContractQuery.Inputs[0].Input = "The ID" },
		"add-data with a validation field":      func(c *Candidate) { c.Delta[0].Attribute = "sales.order.total" },
		"add-application with a second payload": func(c *Candidate) { c.Delta[1].Data = c.Delta[0].Data },
		"duplicate application in the delta": func(c *Candidate) {
			c.Delta = append(c.Delta, c.Delta[1])
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			c := secondApplicationCandidate(revision)
			mutate(&c)
			if _, _, err := normalizeCandidate(c); err == nil {
				t.Fatal("不正なoperationがnormalizeを通りました")
			}
		})
	}
}

func TestSourceGenerationIsDeterministicAndKeepsAuthoredOrder(t *testing.T) {
	revision := "sha256:" + strings.Repeat("a", 64)
	a, fpA, err := normalizeCandidate(secondApplicationCandidate(revision))
	if err != nil {
		t.Fatal(err)
	}
	reversed := secondApplicationCandidate(revision)
	reversed.Delta[0], reversed.Delta[2] = reversed.Delta[2], reversed.Delta[0]
	b, fpB, err := normalizeCandidate(reversed)
	if err != nil {
		t.Fatal(err)
	}
	if fpA != fpB || !reflect.DeepEqual(a.Delta, b.Delta) {
		t.Fatal("operationの並び順の違いがnormalize後に残っています")
	}

	swapped := secondApplicationCandidate(revision)
	swapped.Delta[1].Application.UsesSemantic = []string{"customer", "repair_intake"}
	c, fpC, err := normalizeCandidate(swapped)
	if err != nil {
		t.Fatal(err)
	}
	textA, _ := renderNewFile(a.Delta[0])
	textC, _ := renderNewFile(c.Delta[0])
	if fpC == fpA || textA == textC || !strings.Contains(textA, "uses semantic repair_intake\n    uses semantic customer") {
		t.Fatal("usesSemanticはauthoredされた順序を保ち、順序の違いはsource / fingerprintに現れる必要があります")
	}
}

func TestSecondApplicationCompositionRejectsPlacementInTheWrongLayerOrOnACollision(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(c *Candidate)
		want   string
	}{
		{"data in the application layer", func(c *Candidate) { c.Delta[0].Placement = "applications/repair-intake/intake.tei" }, "LAYER_VIOLATION"},
		{"application in the semantic layer", func(c *Candidate) { c.Delta[1].Placement = "semantic/repair-intake/application.tei" }, "LAYER_VIOLATION"},
		{"pattern in the semantic layer", func(c *Candidate) { c.Delta[2].Placement = "semantic/repair-intake/patterns/customer-reference.tei" }, "LAYER_VIOLATION"},
		{"application that the loader would not read", func(c *Candidate) { c.Delta[1].Placement = "applications/repair-intake/app.tei" }, "PLACEMENT_SHAPE_INVALID"},
		{"data outside a tei file", func(c *Candidate) { c.Delta[0].Placement = "semantic/repair-intake/intake.txt" }, "PLACEMENT_SHAPE_INVALID"},
		{"two operations on one file", func(c *Candidate) {
			c.Delta = append(c.Delta, Operation{
				Op: opAddData, Placement: s4SemanticPath,
				Data: &DataSpec{Namespace: "other_space", ID: "other_space.thing", Symbol: "Thing"},
			})
		}, "PLACEMENT_COLLISION"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir, ev := previewComposition(t, tc.mutate)
			if ev.preview.Acceptable || !hasCode(diagnosticCodes(ev.preview), tc.want) {
				t.Fatalf("%s が検出されていません: %v", tc.want, diagnosticCodes(ev.preview))
			}
			if got := stageStatus(t, ev.preview, "resolve").Status; got != "fail" {
				t.Fatalf("resolve=%s", got)
			}
			if _, err := os.Stat(filepath.Join(dir, "applications", "repair-intake")); err == nil {
				t.Fatal("拒否したのにdirectoryが作られています")
			}
		})
	}
}

func TestSecondApplicationCompositionRejectsAnExistingIdentity(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(c *Candidate)
		want   string
	}{
		{"existing semantic namespace", func(c *Candidate) {
			c.Delta[0].Data = &DataSpec{Namespace: "sales", ID: "sales.intake", Symbol: "Intake"}
			c.Delta[0].Placement = "semantic/sales/intake.tei"
		}, "NAMESPACE_ALREADY_EXISTS"},
		{"two data operations in one new namespace", func(c *Candidate) {
			c.Delta = append(c.Delta, Operation{
				Op: opAddData, Placement: "semantic/repair-intake/second.tei",
				Data: &DataSpec{Namespace: "repair_intake", ID: "repair_intake.second", Symbol: "Second"},
			})
		}, "NAMESPACE_ALREADY_EXISTS"},
		{"existing application", func(c *Candidate) { c.Delta[1].Application.ID = "order_entry" }, "APPLICATION_ALREADY_EXISTS"},
		{"existing pattern", func(c *Candidate) { c.Delta[2].Pattern.ID = "order_entry.customer_reference" }, "PATTERN_ALREADY_EXISTS"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir, ev := previewComposition(t, tc.mutate)
			if ev.preview.Acceptable || !hasCode(diagnosticCodes(ev.preview), tc.want) {
				t.Fatalf("%s が検出されていません: %v", tc.want, diagnosticCodes(ev.preview))
			}
			if _, err := os.Stat(filepath.Join(dir, "applications", "repair-intake")); err == nil {
				t.Fatal("拒否したのにdirectoryが作られています")
			}
		})
	}
}
