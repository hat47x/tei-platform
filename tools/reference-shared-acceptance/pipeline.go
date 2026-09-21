package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"tei/internal/subject"
	"tei/internal/validation"
	"tei/tools/internal/changeanalysis"
)

const (
	previewSchema     = "tei.reference.change-preview/v0"
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
	ApplicationDiff      []changeanalysis.SubjectChange `json:"applicationDiff,omitempty"`
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
			target := filepath.Join(dir, filepath.FromSlash(edit.Path))
			if edit.Created {
				if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
					infraErr = err
					return []Diagnostic{{Code: "OVERLAY_WRITE", Message: err.Error()}}
				}
			}
			if err := os.WriteFile(target, edit.after, 0o644); err != nil {
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
		semanticChanges, err := changeanalysis.Diff(before.loaded.World, after.loaded.World)
		if err != nil {
			return []Diagnostic{{Code: "SEMANTIC_DIFF_FAILED", Message: err.Error()}}
		}
		applicationChanges, err := changeanalysis.DiffApplications(before.loaded.World, after.loaded.World)
		if err != nil {
			return []Diagnostic{{Code: "APPLICATION_DIFF_FAILED", Message: err.Error()}}
		}
		ev.preview.SemanticDiff = semanticChanges
		ev.preview.ApplicationDiff = applicationChanges
		changed, unchanged, err := changeanalysis.CollectionDiff(before.loaded.World, after.loaded.World)
		if err != nil {
			return []Diagnostic{{Code: "COLLECTION_DIFF_FAILED", Message: err.Error()}}
		}
		ev.preview.ChangedCollections = changed
		ev.preview.UnchangedCollections = unchanged

		return checkExpectedDelta(allChanges(ev.preview), expectedChanges(candidate.Delta), changed, expectedCollections(candidate.Delta))
	})

	run("impact", func() []Diagnostic {
		ev.preview.Impact, ev.preview.Unresolved = changeanalysis.Impact(after.loaded.World, allChanges(ev.preview))
		return nil
	})

	run("acceptance-policy", func() []Diagnostic {
		if n := len(allChanges(ev.preview)); n > maxSubjectChanges {
			return []Diagnostic{{
				Code:    "DELTA_TOO_LARGE",
				Message: fmt.Sprintf("変更されるsubjectが%d件で、上限%d件を超えています。whole-model再生成ではなく、最小のsemantic deltaへ分けてください", n, maxSubjectChanges),
			}}
		}
		return nil
	})

	ev.preview.Acceptable = !failed
	return ev, infraErr
}

// checkExpectedDeltaは、実際の差分がcandidateの宣言したdeltaと一致し、変わったWorld collectionが
// deltaから導かれる集合とexactに一致することを確認する。「semantics以外も許す」へ緩めない。
func checkExpectedDelta(actual, expected []changeanalysis.SubjectChange, changedCollections, expectedCollections []string) []Diagnostic {
	var out []Diagnostic
	if !changesEqual(actual, expected) {
		out = append(out, Diagnostic{
			Code:    "SEMANTIC_DIFF_MISMATCH",
			Message: fmt.Sprintf("実際の差分が、candidateが宣言したdeltaと一致しません: actual=%s expected=%s", describeChanges(actual), describeChanges(expected)),
		})
	}
	if !reflect.DeepEqual(nonNil(changedCollections), nonNil(expectedCollections)) {
		out = append(out, Diagnostic{
			Code:    "UNEXPECTED_COLLECTION_CHANGE",
			Message: fmt.Sprintf("変わったWorld collection %v が、deltaから導かれる集合 %v と一致しません", changedCollections, expectedCollections),
		})
	}
	return out
}

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

// checkContainmentは、変更がdeltaのために作られたsource fileだけに閉じ、各fileがそのeditの所有層にあることを確認する。
func checkContainment(before, after snapshot, edits []SourceEdit) []Diagnostic {
	declared := map[string]SourceEdit{}
	allowedPrefixes := map[string]bool{}
	for _, e := range edits {
		declared[e.Path] = e
		allowedPrefixes[layerPrefix(e.Layer)] = true
	}
	changed := changedFiles(before.fingerprints(), after.fingerprints())
	var out []Diagnostic
	changedSet := map[string]bool{}
	for _, path := range changed {
		changedSet[path] = true
		edit, ok := declared[path]
		if !ok {
			out = append(out, Diagnostic{Code: "UNEXPECTED_SOURCE_CHANGE", Message: fmt.Sprintf("deltaに対応しないsource fileが変わっています: %s", path), Subject: "file:" + path})
			inAllowedLayer := false
			for prefix := range allowedPrefixes {
				if strings.HasPrefix(path, prefix) {
					inAllowedLayer = true
				}
			}
			if !inAllowedLayer {
				out = append(out, Diagnostic{Code: "LAYER_VIOLATION", Message: fmt.Sprintf("deltaが扱う層の外のsource fileが変わっています: %s", path), Subject: "file:" + path})
			}
			continue
		}
		if !strings.HasPrefix(path, layerPrefix(edit.Layer)) {
			out = append(out, Diagnostic{Code: "LAYER_VIOLATION", Message: fmt.Sprintf("editの所有層（%s）の外のsource fileが変わっています: %s", layerPrefix(edit.Layer), path), Subject: "file:" + path})
		}
	}
	for path := range declared {
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

// expectedChangesは、型付きdeltaから、Canonical Worldに現れるはずのsubject差分を導く。
func expectedChanges(ops []Operation) []changeanalysis.SubjectChange {
	var out []changeanalysis.SubjectChange
	add := func(kind, id string) {
		out = append(out, changeanalysis.SubjectChange{Kind: kind, ID: id, Change: "added"})
	}
	for _, op := range ops {
		switch op.Op {
		case opAddValidation:
			add("validation", op.ID)
		case opAddData:
			add("namespace", op.Data.Namespace)
			add("data", op.Data.ID)
			for _, a := range op.Data.Attributes {
				add("attribute", a.ID)
			}
			for _, r := range op.Data.Relations {
				add("relation", r.ID)
			}
		case opAddApplication:
			add("application", op.Application.ID)
		case opAddReferencePattern:
			add("application-pattern", op.Pattern.ID)
		}
	}
	sortChanges(out)
	return out
}

// expectedCollectionsは、型付きdeltaから、変わるはずのWorld直下のcollectionを導く。
func expectedCollections(ops []Operation) []string {
	set := map[string]bool{}
	for _, op := range ops {
		switch op.Op {
		case opAddValidation, opAddData:
			set["semantics"] = true
		case opAddApplication, opAddReferencePattern:
			set["applications"] = true
		}
	}
	out := make([]string, 0, len(set))
	for name := range set {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func sortChanges(changes []changeanalysis.SubjectChange) {
	sort.Slice(changes, func(i, j int) bool {
		if changes[i].Kind != changes[j].Kind {
			return changes[i].Kind < changes[j].Kind
		}
		return changes[i].ID < changes[j].ID
	})
}

func allChanges(p Preview) []changeanalysis.SubjectChange {
	out := append(append([]changeanalysis.SubjectChange{}, p.SemanticDiff...), p.ApplicationDiff...)
	sortChanges(out)
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
