package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"tei/internal/ir"
	"tei/internal/sourceindex"
	"tei/internal/validation"
	"tei/tools/internal/changeanalysis"
)

const decisionSchema = "tei.reference.change-decision/v0"

type Check struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail,omitempty"`
}

// Decisionは、Accept / Reject / 受入拒否の結果を記録する。
// authoritative sourceの一部ではなく、消去してもaccepted meaningは変わらない。
type Decision struct {
	Schema               string                         `json:"schema"`
	Decision             string                         `json:"decision"`
	Reason               string                         `json:"reason,omitempty"`
	CandidateID          string                         `json:"candidateId,omitempty"`
	CandidateFingerprint string                         `json:"candidateFingerprint,omitempty"`
	Origin               *Origin                        `json:"origin,omitempty"`
	BaseModelRevision    string                         `json:"baseModelRevision,omitempty"`
	ModelRevisionBefore  string                         `json:"modelRevisionBefore,omitempty"`
	ModelRevisionAfter   string                         `json:"modelRevisionAfter,omitempty"`
	SourceChanged        bool                           `json:"sourceChanged"`
	Stale                *Stale                         `json:"stale,omitempty"`
	Refusals             []Diagnostic                   `json:"refusals,omitempty"`
	SourceEdits          []SourceEdit                   `json:"sourceEdits,omitempty"`
	SemanticDiff         []changeanalysis.SubjectChange `json:"semanticDiff,omitempty"`
	ApplicationDiff      []changeanalysis.SubjectChange `json:"applicationDiff,omitempty"`
	Conformance          []Check                        `json:"conformance,omitempty"`
	DerivedWorldSHA256   string                         `json:"derivedWorldSha256,omitempty"`
}

// beforeEachWriteHookはtestが「Previewの後にsourceが変わる」状況を再現するための差し込み口である。
var beforeEachWriteHook func(index int, edit SourceEdit)

// afterWritesHookは、全fileを書いた直後、再読込の前に呼ばれる（受理後Conformance失敗の再現用）。
var afterWritesHook func(root string)

type acceptOptions struct {
	// DerivedOutは、accepted sourceから再生成したCanonical World JSONの出力先。空なら書き出さない。
	DerivedOut string
}

func encodeJSON(value any) ([]byte, error) {
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func worldSHA256(w ir.World) (string, []byte, error) {
	encoded, err := encodeJSON(w)
	if err != nil {
		return "", nil, err
	}
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:]), encoded, nil
}

func decisionFrom(p Preview, kind string) Decision {
	return Decision{
		Schema:               decisionSchema,
		Decision:             kind,
		CandidateID:          p.CandidateID,
		CandidateFingerprint: p.CandidateFingerprint,
		Origin:               p.Origin,
		BaseModelRevision:    p.BaseModelRevision,
		ModelRevisionBefore:  p.CurrentModelRevision,
		Stale:                p.Stale,
	}
}

// acceptは、candidateを受入境界の全検査へ通し、通った場合だけauthoritative sourceを更新する。
// 更新後はtoolを介さない共有loaderで再読込し、Previewが示した差分どおりか・Validationが通るか・
// 派生物を決定論的に再生成できるかを確認し、一つでも満たさなければ元のsourceへ戻す
// （追記したfileは元の内容へ、作成したfileと、そのために作ったdirectoryは削除する）。
func accept(root string, data []byte, opts acceptOptions) (Decision, error) {
	ev, err := evaluate(root, data)
	if err != nil {
		return Decision{}, err
	}
	decision := decisionFrom(ev.preview, "accepted")
	if !ev.preview.Acceptable {
		decision.Decision = "refused"
		decision.Refusals = allDiagnostics(ev.preview)
		return decision, nil
	}

	var written []SourceEdit
	var createdDirs []string
	rollback := func() {
		for _, e := range written {
			target := filepath.Join(root, filepath.FromSlash(e.Path))
			if e.Created {
				_ = os.Remove(target)
			} else {
				_ = writeFileAtomic(target, e.before)
			}
		}
		for i := len(createdDirs) - 1; i >= 0; i-- {
			_ = os.Remove(createdDirs[i])
		}
	}
	refuse := func(code, message string) Decision {
		rollback()
		decision.Decision = "refused"
		decision.SourceChanged = false
		decision.Refusals = append(decision.Refusals, Diagnostic{Code: code, Message: message})
		return decision
	}

	for i, e := range ev.edits {
		if beforeEachWriteHook != nil {
			beforeEachWriteHook(i, e)
		}
		target := filepath.Join(root, filepath.FromSlash(e.Path))
		if e.Created {
			if _, err := os.Stat(target); err == nil {
				return refuse("SOURCE_CHANGED_DURING_ACCEPT", fmt.Sprintf("作成先 %s がPreviewの後に作られました", e.Path)), nil
			}
			dirs, err := makeDirs(filepath.Dir(target))
			createdDirs = append(createdDirs, dirs...)
			if err != nil {
				return refuse("SOURCE_WRITE_FAILED", err.Error()), nil
			}
		} else {
			current, err := os.ReadFile(target)
			if err != nil {
				return refuse("SOURCE_UNREADABLE", err.Error()), nil
			}
			if sourceindex.Fingerprint(current) != e.BeforeFingerprint {
				return refuse("SOURCE_CHANGED_DURING_ACCEPT", fmt.Sprintf("%s がPreviewの後に変更されました", e.Path)), nil
			}
		}
		if err := writeFileAtomic(target, e.after); err != nil {
			return refuse("SOURCE_WRITE_FAILED", err.Error()), nil
		}
		written = append(written, e)
	}

	if afterWritesHook != nil {
		afterWritesHook(root)
	}
	post, err := takeSnapshot(root)
	if err != nil {
		return refuse("POST_ACCEPT_RELOAD_FAILED", err.Error()), nil
	}
	postSemantic, err := changeanalysis.Diff(ev.before.loaded.World, post.loaded.World)
	if err != nil {
		return refuse("POST_ACCEPT_DIFF_FAILED", err.Error()), nil
	}
	postApplication, err := changeanalysis.DiffApplications(ev.before.loaded.World, post.loaded.World)
	if err != nil {
		return refuse("POST_ACCEPT_DIFF_FAILED", err.Error()), nil
	}
	derivedSHA, derived, err := worldSHA256(post.loaded.World)
	if err != nil {
		return refuse("POST_ACCEPT_DERIVE_FAILED", err.Error()), nil
	}
	again, _, err := worldSHA256(post.loaded.World)
	if err != nil {
		return refuse("POST_ACCEPT_DERIVE_FAILED", err.Error()), nil
	}

	var validationErrors int
	for _, d := range validation.Validate(post.loaded.World) {
		if d.Severity == validation.Error {
			validationErrors++
		}
	}
	checks := []Check{
		{Name: "reloaded-with-shared-loader", Passed: true},
		{Name: "model-revision-matches-preview", Passed: post.revision == ev.preview.ModelRevisionAfter, Detail: post.revision},
		{Name: "semantic-diff-matches-preview", Passed: reflect.DeepEqual(postSemantic, ev.preview.SemanticDiff)},
		{Name: "application-diff-matches-preview", Passed: reflect.DeepEqual(postApplication, ev.preview.ApplicationDiff)},
		{Name: "validation-clean", Passed: validationErrors == 0},
		{Name: "derived-world-reproducible", Passed: derivedSHA == again, Detail: derivedSHA},
	}
	for _, c := range checks {
		if !c.Passed {
			decision.Conformance = checks
			return refuse("POST_ACCEPT_CONFORMANCE_FAILED", "受理後の再読込・Conformanceが通らなかったため、sourceを元へ戻しました: "+c.Name), nil
		}
	}

	if opts.DerivedOut != "" {
		if err := os.MkdirAll(filepath.Dir(opts.DerivedOut), 0o755); err != nil {
			return refuse("DERIVED_WRITE_FAILED", err.Error()), nil
		}
		if err := os.WriteFile(opts.DerivedOut, derived, 0o644); err != nil {
			return refuse("DERIVED_WRITE_FAILED", err.Error()), nil
		}
	}

	decision.SourceChanged = true
	decision.ModelRevisionAfter = post.revision
	decision.SourceEdits = ev.edits
	decision.SemanticDiff = postSemantic
	decision.ApplicationDiff = postApplication
	decision.Conformance = checks
	decision.DerivedWorldSHA256 = derivedSHA
	return decision, nil
}

// makeDirsは、directoryを作り、そのために新しく作ったdirectory（外側から内側の順）を返す。
func makeDirs(dir string) ([]string, error) {
	var missing []string
	for d := dir; ; d = filepath.Dir(d) {
		if _, err := os.Stat(d); err == nil {
			break
		}
		missing = append([]string{d}, missing...)
		if filepath.Dir(d) == d {
			break
		}
	}
	for _, d := range missing {
		if err := os.Mkdir(d, 0o755); err != nil {
			return missing, err
		}
	}
	return missing, nil
}

// rejectは、candidateを退け、authoritative sourceを変更しないことを記録する。
// staleなcandidateも退けられるため、基準revisionの一致は要求しない。
func reject(root string, data []byte, reason string) (Decision, error) {
	candidate, err := decodeCandidate(data)
	if err != nil {
		return Decision{Schema: decisionSchema, Decision: "refused", Refusals: []Diagnostic{{Code: "CANDIDATE_DECODE", Message: err.Error()}}}, nil
	}
	normalized, fingerprint, err := normalizeCandidate(candidate)
	if err != nil {
		return Decision{Schema: decisionSchema, Decision: "refused", Refusals: []Diagnostic{{Code: "CANDIDATE_NORMALIZE", Message: err.Error()}}}, nil
	}
	s, err := takeSnapshot(root)
	if err != nil {
		return Decision{}, err
	}
	origin := normalized.Origin
	return Decision{
		Schema:               decisionSchema,
		Decision:             "rejected",
		Reason:               reason,
		CandidateID:          normalized.ID,
		CandidateFingerprint: fingerprint,
		Origin:               &origin,
		BaseModelRevision:    normalized.BaseModelRevision,
		ModelRevisionBefore:  s.revision,
		ModelRevisionAfter:   s.revision,
		SourceChanged:        false,
	}, nil
}
