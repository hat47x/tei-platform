package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tei/internal/core"
	"tei/internal/subject"
	"tei/internal/syntax"
	"tei/internal/validation"
	"tei/tools/internal/changeanalysis"
)

const (
	previewSchema     = "tei.reference.change-preview/v0"
	semanticRoot      = "semantic/"
	maxSubjectChanges = 8
)

type Diagnostic struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Subject string `json:"subject,omitempty"`
}

type Stage struct {
	Name        string       `json:"name"`
	Status      string       `json:"status"`
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`
}

// Staleは、candidateの基準revisionが現在のsourceと一致しないことと、その後の選択肢を示す。
// staleなcandidateは黙って最新版へ適用しない。
type Stale struct {
	BaseModelRevision    string   `json:"baseModelRevision"`
	CurrentModelRevision string   `json:"currentModelRevision"`
	Options              []string `json:"options"`
}

// Previewは、authoritative sourceを変更せずに、受入境界の各検査結果と派生する影響を示す。
type Preview struct {
	Schema               string                         `json:"schema"`
	CandidateID          string                         `json:"candidateId,omitempty"`
	CandidateFingerprint string                         `json:"candidateFingerprint,omitempty"`
	Origin               *Origin                        `json:"origin,omitempty"`
	BaseModelRevision    string                         `json:"baseModelRevision,omitempty"`
	CurrentModelRevision string                         `json:"currentModelRevision,omitempty"`
	ModelRevisionAfter   string                         `json:"modelRevisionAfter,omitempty"`
	Stale                *Stale                         `json:"stale,omitempty"`
	Stages               []Stage                        `json:"stages"`
	Acceptable           bool                           `json:"acceptable"`
	SourceEdits          []SourceEdit                   `json:"sourceEdits,omitempty"`
	SemanticDiff         []changeanalysis.SubjectChange `json:"semanticDiff,omitempty"`
	ChangedCollections   []string                       `json:"changedCollections,omitempty"`
	UnchangedCollections []string                       `json:"unchangedCollections,omitempty"`
	Impact               []changeanalysis.ImpactEdge    `json:"impact,omitempty"`
	Unresolved           []changeanalysis.Unresolved    `json:"unresolved,omitempty"`
}

type evaluation struct {
	preview Preview
	before  snapshot
	edits   []SourceEdit
}

func staleOptions() []string {
	return []string{
		"reject",
		"re-read-current-revision-and-regenerate-candidate",
		"explicit-rebase-candidate",
		"manual-conflict-resolution",
	}
}

// evaluateは、candidateを受入境界の各検査へ通す。authoritative sourceは書き換えない。
// 検査はどの入口（file / CLI / Studio / AI）のcandidateでも同じ順序・同じ規則で行う。
func evaluate(root string, data []byte) (evaluation, error) {
	ev := evaluation{preview: Preview{Schema: previewSchema}}
	var (
		candidate Candidate
		before    snapshot
		after     snapshot
		overlay   string
		failed    bool
		infraErr  error
	)
	defer func() {
		if overlay != "" {
			_ = os.RemoveAll(overlay)
		}
	}()

	run := func(name string, fn func() []Diagnostic) {
		if failed {
			ev.preview.Stages = append(ev.preview.Stages, Stage{Name: name, Status: "skipped"})
			return
		}
		diagnostics := fn()
		status := "pass"
		if len(diagnostics) > 0 {
			status = "fail"
			failed = true
		}
		ev.preview.Stages = append(ev.preview.Stages, Stage{Name: name, Status: status, Diagnostics: diagnostics})
	}

	run("decode", func() []Diagnostic {
		c, err := decodeCandidate(data)
		if err != nil {
			return []Diagnostic{{Code: "CANDIDATE_DECODE", Message: err.Error()}}
		}
		candidate = c
		return nil
	})

	run("normalize", func() []Diagnostic {
		normalized, fingerprint, err := normalizeCandidate(candidate)
		if err != nil {
			return []Diagnostic{{Code: "CANDIDATE_NORMALIZE", Message: err.Error()}}
		}
		candidate = normalized
		ev.preview.CandidateID = candidate.ID
		ev.preview.CandidateFingerprint = fingerprint
		origin := candidate.Origin
		ev.preview.Origin = &origin
		ev.preview.BaseModelRevision = candidate.BaseModelRevision
		return nil
	})

	run("base-revision", func() []Diagnostic {
		s, err := takeSnapshot(root)
		if err != nil {
			return []Diagnostic{{Code: "WORKSPACE_UNREADABLE", Message: err.Error()}}
		}
		before = s
		ev.before = s
		ev.preview.CurrentModelRevision = s.revision
		if candidate.BaseModelRevision == s.revision {
			return nil
		}
		ev.preview.Stale = &Stale{
			BaseModelRevision:    candidate.BaseModelRevision,
			CurrentModelRevision: s.revision,
			Options:              staleOptions(),
		}
		return []Diagnostic{{
			Code:    "STALE_BASE_MODEL_REVISION",
			Message: "candidateの基準revisionが現在のsourceと一致しません。黙って最新版へ適用せず、reject / 再読込して再生成 / 明示的なrebase / 手動conflict解消のいずれかへ進みます",
		}}
	})

	run("resolve", func() []Diagnostic { return resolveCandidate(before, candidate) })

	run("apply", func() []Diagnostic {
		edits, diagnostics := planEdits(before, candidate.Delta)
		if len(diagnostics) > 0 {
			return diagnostics
		}
		dir, err := os.MkdirTemp("", "tei-acceptance-overlay-*")
		if err != nil {
			infraErr = err
			return []Diagnostic{{Code: "OVERLAY_CREATE", Message: err.Error()}}
		}
		overlay = dir
		if err := copyTree(before.root, dir); err != nil {
			infraErr = err
			return []Diagnostic{{Code: "OVERLAY_COPY", Message: err.Error()}}
		}
		for _, edit := range edits {
			if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(edit.Path)), edit.after, 0o644); err != nil {
				infraErr = err
				return []Diagnostic{{Code: "OVERLAY_WRITE", Message: err.Error()}}
			}
		}
		ev.edits = edits
		ev.preview.SourceEdits = edits
		return nil
	})

	run("reload", func() []Diagnostic {
		s, err := takeSnapshot(overlay)
		if err != nil {
			return []Diagnostic{{Code: "RELOAD_FAILED", Message: err.Error()}}
		}
		after = s
		ev.preview.ModelRevisionAfter = s.revision
		return nil
	})

	run("validate", func() []Diagnostic {
		known := map[string]bool{}
		for _, d := range before.diagnostics() {
			known[d.Code+"\x00"+d.Message] = true
		}
		var out []Diagnostic
		for _, d := range after.diagnostics() {
			if d.Severity != validation.Error {
				continue
			}
			message := d.Message
			if known[d.Code+"\x00"+d.Message] {
				message += "（変更前から存在するerror）"
			}
			out = append(out, Diagnostic{Code: d.Code, Message: message, Subject: formatSubject(d.Subject)})
		}
		return out
	})

	run("architecture-policy", func() []Diagnostic { return checkContainment(before, after, ev.edits) })

	run("semantic-diff", func() []Diagnostic {
		actual, err := changeanalysis.Diff(before.loaded.World, after.loaded.World)
		if err != nil {
			return []Diagnostic{{Code: "SEMANTIC_DIFF_FAILED", Message: err.Error()}}
		}
		ev.preview.SemanticDiff = actual
		changed, unchanged, err := changeanalysis.CollectionDiff(before.loaded.World, after.loaded.World)
		if err != nil {
			return []Diagnostic{{Code: "COLLECTION_DIFF_FAILED", Message: err.Error()}}
		}
		ev.preview.ChangedCollections = changed
		ev.preview.UnchangedCollections = unchanged

		return checkExpectedDelta(actual, expectedChanges(candidate.Delta), changed)
	})

	run("impact", func() []Diagnostic {
		ev.preview.Impact, ev.preview.Unresolved = changeanalysis.Impact(after.loaded.World, ev.preview.SemanticDiff)
		return nil
	})

	run("acceptance-policy", func() []Diagnostic {
		if len(ev.preview.SemanticDiff) > maxSubjectChanges {
			return []Diagnostic{{
				Code:    "DELTA_TOO_LARGE",
				Message: fmt.Sprintf("変更されるsubjectが%d件で、上限%d件を超えています。whole-model再生成ではなく、最小のsemantic deltaへ分けてください", len(ev.preview.SemanticDiff), maxSubjectChanges),
			}}
		}
		return nil
	})

	ev.preview.Acceptable = !failed
	return ev, infraErr
}

// checkExpectedDeltaは、実際の差分がcandidateの宣言したdeltaと一致し、
// semantic層以外のWorld collectionへ副作用が及んでいないことを確認する。
func checkExpectedDelta(actual, expected []changeanalysis.SubjectChange, changedCollections []string) []Diagnostic {
	var out []Diagnostic
	if !changesEqual(actual, expected) {
		out = append(out, Diagnostic{
			Code:    "SEMANTIC_DIFF_MISMATCH",
			Message: fmt.Sprintf("実際のsemantic差分が、candidateが宣言したdeltaと一致しません: actual=%s expected=%s", describeChanges(actual), describeChanges(expected)),
		})
	}
	if len(changedCollections) != 1 || changedCollections[0] != "semantics" {
		out = append(out, Diagnostic{
			Code:    "UNEXPECTED_COLLECTION_CHANGE",
			Message: fmt.Sprintf("Validationの追加でsemantic層以外のWorld collectionが変わっています: %v", changedCollections),
		})
	}
	return out
}

func resolveCandidate(s snapshot, c Candidate) []Diagnostic {
	var out []Diagnostic
	world := s.loaded.World
	namespaces := map[core.CanonicalID]bool{}
	validations := map[core.CanonicalID]bool{}
	attributes := map[core.CanonicalID]bool{}
	relations := map[core.CanonicalID]bool{}
	subjects := map[string]bool{}
	for _, ns := range world.Semantics {
		namespaces[ns.ID] = true
		subjects[string(ns.ID)] = true
		for _, t := range ns.Types {
			subjects[string(t.ID)] = true
		}
		for _, v := range ns.Validations {
			validations[v.ID] = true
			subjects[string(v.ID)] = true
		}
		for _, cap := range ns.Capabilities {
			subjects[string(cap.ID)] = true
		}
		for _, d := range ns.Data {
			subjects[string(d.ID)] = true
			for _, a := range d.Attributes {
				attributes[a.ID] = true
				subjects[string(a.ID)] = true
			}
			for _, r := range d.Relations {
				relations[r.ID] = true
				subjects[string(r.ID)] = true
			}
		}
	}
	for _, app := range world.Applications {
		subjects[string(app.ID)] = true
		for _, p := range app.Patterns {
			subjects[string(p.ID)] = true
		}
	}

	for _, op := range c.Delta {
		key := "validation:" + op.ID
		if !namespaces[core.CanonicalID(op.Namespace)] {
			out = append(out, Diagnostic{Code: "NAMESPACE_NOT_FOUND", Message: fmt.Sprintf("semantic namespace %s は存在しません", op.Namespace), Subject: key})
		}
		if validations[core.CanonicalID(op.ID)] {
			out = append(out, Diagnostic{Code: "VALIDATION_ALREADY_EXISTS", Message: fmt.Sprintf("validation %s は既に存在します", op.ID), Subject: key})
		}
		if !attributes[core.CanonicalID(op.Attribute)] {
			out = append(out, Diagnostic{Code: "ATTRIBUTE_NOT_FOUND", Message: fmt.Sprintf("attribute %s は存在しません", op.Attribute), Subject: key})
		}
		if op.Relation != "" && !relations[core.CanonicalID(op.Relation)] {
			out = append(out, Diagnostic{Code: "RELATION_NOT_FOUND", Message: fmt.Sprintf("relation %s は存在しません", op.Relation), Subject: key})
		}
		out = append(out, checkPlacementOwnership(s, op)...)
	}

	fingerprints := s.fingerprints()
	for i, e := range c.Evidence {
		if e.Subject != "" && !subjects[e.Subject] {
			out = append(out, Diagnostic{Code: "EVIDENCE_SUBJECT_NOT_FOUND", Message: fmt.Sprintf("evidence[%d].subject %s はCanonical Worldに存在しません", i, e.Subject)})
		}
		if e.Source != "" {
			if _, ok := fingerprints[e.Source]; !ok {
				out = append(out, Diagnostic{Code: "EVIDENCE_SOURCE_NOT_FOUND", Message: fmt.Sprintf("evidence[%d].source %s は管理対象sourceではありません", i, e.Source)})
			}
		}
	}
	return out
}

// checkPlacementOwnershipは、明示placementがValidationを所有するsemantic層のfileであること、
// そのfileが同じnamespaceを宣言していることを、sourceを書き換える前に確認する。
func checkPlacementOwnership(s snapshot, op Operation) []Diagnostic {
	if op.Placement == "" {
		return nil
	}
	key := "validation:" + op.ID
	if !strings.HasPrefix(op.Placement, semanticRoot) {
		return []Diagnostic{{
			Code:    "LAYER_VIOLATION",
			Message: fmt.Sprintf("Validationは意味を所有するsemantic層（%s）でauthorします。placement %s は別の層です", semanticRoot, op.Placement),
			Subject: key,
		}}
	}
	content, err := os.ReadFile(filepath.Join(s.root, filepath.FromSlash(op.Placement)))
	if err != nil {
		return []Diagnostic{{Code: "PLACEMENT_NOT_MANAGED_SOURCE", Message: fmt.Sprintf("placement %s を読めません: %v", op.Placement, err), Subject: key}}
	}
	ns, err := syntax.NewParser(string(content)).ParseSemanticFile()
	if err != nil {
		return []Diagnostic{{Code: "PLACEMENT_NOT_SEMANTIC_FILE", Message: fmt.Sprintf("placement %s はsemantic fileとして読めません: %v", op.Placement, err), Subject: key}}
	}
	if string(ns.ID) != op.Namespace {
		return []Diagnostic{{
			Code:    "PLACEMENT_NAMESPACE_MISMATCH",
			Message: fmt.Sprintf("placement %s はnamespace %s を宣言しており、%s ではありません", op.Placement, ns.ID, op.Namespace),
			Subject: key,
		}}
	}
	return nil
}

// checkContainmentは、変更がdeltaのために作られたsource fileだけに閉じていることを確認する。
func checkContainment(before, after snapshot, edits []SourceEdit) []Diagnostic {
	expected := map[string]bool{}
	for _, e := range edits {
		expected[e.Path] = true
	}
	changed := changedFiles(before.fingerprints(), after.fingerprints())
	var out []Diagnostic
	for _, path := range changed {
		if !expected[path] {
			out = append(out, Diagnostic{Code: "UNEXPECTED_SOURCE_CHANGE", Message: fmt.Sprintf("deltaに対応しないsource fileが変わっています: %s", path), Subject: "file:" + path})
		}
		if !strings.HasPrefix(path, semanticRoot) {
			out = append(out, Diagnostic{Code: "LAYER_VIOLATION", Message: fmt.Sprintf("semantic層の外のsource fileが変わっています: %s", path), Subject: "file:" + path})
		}
	}
	changedSet := map[string]bool{}
	for _, path := range changed {
		changedSet[path] = true
	}
	for path := range expected {
		if !changedSet[path] {
			out = append(out, Diagnostic{Code: "EXPECTED_SOURCE_CHANGE_MISSING", Message: fmt.Sprintf("deltaが変更するはずのsource fileが変わっていません: %s", path), Subject: "file:" + path})
		}
	}
	return out
}

func changedFiles(before, after map[string]string) []string {
	var out []string
	for path, fingerprint := range before {
		if after[path] != fingerprint {
			out = append(out, path)
		}
	}
	for path := range after {
		if _, ok := before[path]; !ok {
			out = append(out, path)
		}
	}
	sort.Strings(out)
	return out
}

func expectedChanges(ops []Operation) []changeanalysis.SubjectChange {
	out := make([]changeanalysis.SubjectChange, 0, len(ops))
	for _, op := range ops {
		out = append(out, changeanalysis.SubjectChange{Kind: "validation", ID: op.ID, Change: "added"})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key() < out[j].Key() })
	return out
}

func describeChanges(changes []changeanalysis.SubjectChange) string {
	parts := make([]string, 0, len(changes))
	for _, c := range changes {
		parts = append(parts, c.Change+" "+c.Key())
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func formatSubject(s *subject.Subject) string {
	if s == nil {
		return ""
	}
	return s.Kind + ":" + s.ID
}

func allDiagnostics(p Preview) []Diagnostic {
	var out []Diagnostic
	for _, stage := range p.Stages {
		out = append(out, stage.Diagnostics...)
	}
	return out
}

func changesEqual(x, y []changeanalysis.SubjectChange) bool {
	if len(x) != len(y) {
		return false
	}
	for i := range x {
		if x[i].Key() != y[i].Key() || x[i].Change != y[i].Change {
			return false
		}
	}
	return true
}
