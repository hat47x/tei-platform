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
	"sort"
	"strings"

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

type ReviewBinding struct {
	Fingerprint string                `json:"fingerprint"`
	Coverage    ReviewCoverageSummary `json:"coverage"`
	Projection  *ReviewProjection     `json:"projection,omitempty"`
}

// Decisionは、Accept / Reject / 受入拒否の結果を記録する。
// authoritative sourceの一部ではなく、消去してもaccepted meaningは変わらない。
type Decision struct {
	Schema               string                         `json:"schema"`
	Decision             string                         `json:"decision"`
	Reason               string                         `json:"reason,omitempty"`
	CandidateID          string                         `json:"candidateId,omitempty"`
	CandidateFingerprint string                         `json:"candidateFingerprint,omitempty"`
	EvidenceFingerprint  string                         `json:"evidenceFingerprint,omitempty"`
	ReviewBinding        *ReviewBinding                 `json:"reviewBinding,omitempty"`
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
	EnvironmentDiff      []changeanalysis.SubjectChange `json:"environmentDiff,omitempty"`
	Conformance          []Check                        `json:"conformance,omitempty"`
	DerivedWorldSHA256   string                         `json:"derivedWorldSha256,omitempty"`

	// 以下は、後から入るAI / 人が「なぜこのsourceがあるのか」を読むための情報である。
	// fingerprintには入らないため、受入判定にもaccepted meaningにも影響しない。
	Intent     string                      `json:"intent,omitempty"`
	Evidence   []Evidence                  `json:"evidence,omitempty"`
	Unresolved []changeanalysis.Unresolved `json:"unresolved,omitempty"`
	RecordPath string                      `json:"recordPath,omitempty"`
}

// beforeEachWriteHookはtestが「Previewの後にsourceが変わる」状況を再現するための差し込み口である。
var beforeEachWriteHook func(index int, edit SourceEdit)

// afterWritesHookは、全fileを書いた直後、再読込の前に呼ばれる（受理後Conformance失敗の再現用）。
var afterWritesHook func(root string)

type acceptOptions struct {
	// DerivedOutは、accepted sourceから再生成したCanonical World JSONの出力先。空なら書き出さない。
	DerivedOut string
	// ExpectedReviewFingerprintは、callerが直前に確認したReview Projectionとの一致を要求する任意guardである。
	// 一致してもhuman approvalやauthorityを証明しない。
	ExpectedReviewFingerprint string
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
		EvidenceFingerprint:  p.EvidenceFingerprint,
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
	if opts.ExpectedReviewFingerprint != "" {
		review := buildReviewProjection(ev)
		if review.ReviewFingerprint != opts.ExpectedReviewFingerprint {
			decision.Decision = "refused"
			decision.Refusals = append(decision.Refusals, Diagnostic{
				Code: "REVIEW_FINGERPRINT_MISMATCH",
				Message: fmt.Sprintf(
					"Accept時のReview Projectionが期待値と一致しません: expected=%s current=%s",
					opts.ExpectedReviewFingerprint,
					review.ReviewFingerprint,
				),
			})
			return decision, nil
		}
		decision.ReviewBinding = &ReviewBinding{
			Fingerprint: review.ReviewFingerprint,
			Coverage:    review.Coverage,
			Projection:  &review,
		}
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
	postEnvironment, err := changeanalysis.DiffEnvironments(ev.before.loaded.World, post.loaded.World)
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
		{Name: "environment-diff-matches-preview", Passed: reflect.DeepEqual(postEnvironment, ev.preview.EnvironmentDiff)},
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
		// 派生物も原子的に書く。書けない場合は、既にそこにあるfileを壊さずrefuseする
		// （`reference/world.json`のようなgoldenを指されても、失敗が途中まで書いた内容を残さない）。
		if err := writeFileAtomic(opts.DerivedOut, derived); err != nil {
			return refuse("DERIVED_WRITE_FAILED", err.Error()), nil
		}
	}

	decision.SourceChanged = true
	decision.ModelRevisionAfter = post.revision
	decision.SourceEdits = ev.edits
	decision.SemanticDiff = postSemantic
	decision.ApplicationDiff = postApplication
	decision.EnvironmentDiff = postEnvironment
	decision.Conformance = checks
	decision.DerivedWorldSHA256 = derivedSHA
	decision.Intent = ev.candidate.Intent
	decision.Evidence = ev.candidate.Evidence
	decision.Unresolved = ev.preview.Unresolved
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
	var stale *Stale
	if normalized.BaseModelRevision != s.revision {
		stale = &Stale{
			BaseModelRevision:    normalized.BaseModelRevision,
			CurrentModelRevision: s.revision,
			Options:              staleOptions(),
		}
	}
	origin := normalized.Origin
	return Decision{
		Schema:               decisionSchema,
		Decision:             "rejected",
		Reason:               reason,
		CandidateID:          normalized.ID,
		CandidateFingerprint: fingerprint,
		EvidenceFingerprint:  evidenceFingerprintOf(normalized.Evidence),
		Origin:               &origin,
		BaseModelRevision:    normalized.BaseModelRevision,
		ModelRevisionBefore:  s.revision,
		ModelRevisionAfter:   s.revision,
		SourceChanged:        false,
		Stale:                stale,
		Intent:               normalized.Intent,
		Evidence:             normalized.Evidence,
	}, nil
}

// decisionDirは、決定記録を置くworkspace内のdirectoryである。managed sourceではないため
// model revisionには入らず、plugin-lockと同じ派生物の置き場に並ぶ。
const decisionDir = ".tei/decisions"

const decisionIndexSchema = "tei.reference.decision-index/v0"

// DecisionRecordは、保存された決定一件と、それが現在のworkspaceをまだ説明できるかを表す。
type DecisionRecord struct {
	Path string `json:"path"`
	// Statusは、記録した時点のrevisionが現在のrevisionと一致するか（current / superseded）。
	Status string `json:"status"`
	// SourceEditsは、その決定が書いたfileが今も同じ内容か（intact / diverged / none）。
	SourceEdits                 string              `json:"sourceEdits"`
	Diverged                    []string            `json:"divergedPaths,omitempty"`
	EvidenceStatus              string              `json:"evidenceStatus"`
	SourceEvidenceStatus        string              `json:"sourceEvidenceStatus"`
	ArtifactEvidenceStatus      string              `json:"artifactEvidenceStatus"`
	EvidenceDiagnostics         []Diagnostic        `json:"evidenceDiagnostics,omitempty"`
	ArtifactEvidenceDiagnostics []Diagnostic        `json:"artifactEvidenceDiagnostics,omitempty"`
	ReviewBindingStatus         string              `json:"reviewBindingStatus"`
	ReviewCoverageStatus        string              `json:"reviewCoverageStatus"`
	ReviewIncomplete            []ReviewCoverageGap `json:"reviewIncomplete,omitempty"`
	ReviewDiagnostics           []Diagnostic        `json:"reviewDiagnostics,omitempty"`
	DecisionActorStatus         string              `json:"decisionActorStatus"`
	DecisionAuthorityStatus     string              `json:"decisionAuthorityStatus"`
	Decision                    Decision            `json:"decision"`
}

type DecisionIndex struct {
	Schema        string           `json:"schema"`
	ModelRevision string           `json:"modelRevision"`
	Records       []DecisionRecord `json:"records"`
}

// recordDecisionは、決定を派生物として保存し、保存先のworkspace相対pathを返す。
// 内容でaddressingするため、同じ決定を二度保存しても増えない。
// この記録はauthorityではない。消してもaccepted meaningもmodel revisionも変わらない。
func recordDecision(root string, d Decision) (string, error) {
	d.RecordPath = ""
	encoded, err := encodeJSON(d)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	rel := decisionDir + "/" + hex.EncodeToString(sum[:]) + ".json"
	target := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", err
	}
	if err := writeFileAtomic(target, encoded); err != nil {
		return "", err
	}
	return rel, nil
}

// loadDecisionRecordsは、保存された決定を読み、現在のworkspaceに対して検証する。
// 記録の内容を信用してmeaningを再構成せず、現在のsourceと照合した結果だけを付ける。
func loadDecisionRecords(root string) (DecisionIndex, error) {
	index := DecisionIndex{Schema: decisionIndexSchema, Records: []DecisionRecord{}}
	s, err := takeSnapshot(root)
	if err != nil {
		return index, err
	}
	index.ModelRevision = s.revision

	dir := filepath.Join(root, filepath.FromSlash(decisionDir))
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return index, nil
		}
		return index, err
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		rel := decisionDir + "/" + name
		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return index, err
		}
		sum := sha256.Sum256(content)
		expectedName := hex.EncodeToString(sum[:]) + ".json"
		if name != expectedName {
			return index, fmt.Errorf(
				"decision record %s: content hashがfile名と一致しません: expected=%s actual=%s",
				rel,
				expectedName,
				name,
			)
		}
		var d Decision
		decoder := json.NewDecoder(bytes.NewReader(content))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&d); err != nil {
			return index, fmt.Errorf("decision record %s: %w", rel, err)
		}
		record := DecisionRecord{
			Path:                    rel,
			Decision:                d,
			Status:                  "superseded",
			SourceEdits:             "none",
			EvidenceStatus:          "none",
			SourceEvidenceStatus:    "none",
			ArtifactEvidenceStatus:  "none",
			ReviewBindingStatus:     "none",
			ReviewCoverageStatus:    "none",
			DecisionActorStatus:     "unresolved",
			DecisionAuthorityStatus: "unresolved",
		}
		if d.ModelRevisionAfter == s.revision {
			record.Status = "current"
		}
		if len(d.SourceEdits) > 0 {
			record.SourceEdits = "intact"
			for _, e := range d.SourceEdits {
				current, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(e.Path)))
				if err != nil || sourceindex.Fingerprint(current) != e.AfterFingerprint {
					record.SourceEdits = "diverged"
					record.Diverged = append(record.Diverged, e.Path)
				}
			}
		}
		record.ReviewBindingStatus,
			record.ReviewCoverageStatus,
			record.ReviewDiagnostics = classifyReviewBinding(d.ReviewBinding)
		if d.ReviewBinding != nil && record.ReviewBindingStatus != "invalid" {
			record.ReviewIncomplete = append(
				record.ReviewIncomplete,
				d.ReviewBinding.Coverage.Incomplete...,
			)
		}

		if len(d.Evidence) > 0 || d.EvidenceFingerprint != "" {
			var sourceEvidence []Evidence
			var artifactEvidence []Evidence
			for _, evidence := range d.Evidence {
				if evidence.Artifact != nil {
					artifactEvidence = append(artifactEvidence, evidence)
				} else {
					sourceEvidence = append(sourceEvidence, evidence)
				}
			}

			if len(sourceEvidence) > 0 {
				record.SourceEvidenceStatus = "current"
				sourceDiagnostics := validateEvidence(s, sourceEvidence)
				record.EvidenceDiagnostics = append(record.EvidenceDiagnostics, sourceDiagnostics...)
				if len(sourceDiagnostics) > 0 {
					record.SourceEvidenceStatus = "diverged"
				}
			}
			if len(artifactEvidence) > 0 {
				record.ArtifactEvidenceStatus = "recorded"
				for _, evidence := range artifactEvidence {
					diagnostics := validateEvidenceArtifact(evidence.Artifact)
					record.ArtifactEvidenceDiagnostics = append(record.ArtifactEvidenceDiagnostics, diagnostics...)
					record.EvidenceDiagnostics = append(record.EvidenceDiagnostics, diagnostics...)
				}
				if len(record.ArtifactEvidenceDiagnostics) > 0 {
					record.ArtifactEvidenceStatus = "invalid"
				}
			}

			fingerprintMismatch := false
			if d.EvidenceFingerprint != "" {
				currentFingerprint := evidenceFingerprintOf(d.Evidence)
				if currentFingerprint != d.EvidenceFingerprint {
					fingerprintMismatch = true
					record.EvidenceDiagnostics = append(record.EvidenceDiagnostics, Diagnostic{
						Code: "EVIDENCE_FINGERPRINT_MISMATCH",
						Message: fmt.Sprintf(
							"decision recordのevidenceFingerprintがEvidence本文と一致しません: recorded=%s current=%s",
							d.EvidenceFingerprint,
							currentFingerprint,
						),
					})
				}
			}

			switch {
			case fingerprintMismatch ||
				record.SourceEvidenceStatus == "diverged" ||
				record.ArtifactEvidenceStatus == "invalid":
				record.EvidenceStatus = "diverged"
			case record.SourceEvidenceStatus == "current" && record.ArtifactEvidenceStatus == "recorded":
				record.EvidenceStatus = "current-with-recorded-artifacts"
			case record.SourceEvidenceStatus == "current":
				record.EvidenceStatus = "current"
			case record.ArtifactEvidenceStatus == "recorded":
				record.EvidenceStatus = "recorded"
			default:
				record.EvidenceStatus = "none"
			}
		}
		record.Decision.RecordPath = rel
		index.Records = append(index.Records, record)
	}

	return index, nil
}

func classifyReviewBinding(binding *ReviewBinding) (string, string, []Diagnostic) {
	if binding == nil {
		return "none", "none", nil
	}
	var diagnostics []Diagnostic
	switch fingerprint := strings.TrimSpace(binding.Fingerprint); {
	case fingerprint == "":
		diagnostics = append(diagnostics, Diagnostic{
			Code:    "REVIEW_BINDING_FINGERPRINT_EMPTY",
			Message: "decision recordのreviewBindingにreview fingerprintがありません",
		})
	case !revisionPattern.MatchString(fingerprint):
		diagnostics = append(diagnostics, Diagnostic{
			Code:    "REVIEW_BINDING_FINGERPRINT_INVALID",
			Message: "decision recordのreviewBinding fingerprintがsha256:<64桁の16進>形式ではありません",
		})
	}

	coverage := binding.Coverage
	if coverage.Available < 0 || coverage.Partial < 0 || coverage.NotEvaluated < 0 {
		diagnostics = append(diagnostics, Diagnostic{
			Code: "REVIEW_COVERAGE_COUNT_INVALID",
			Message: fmt.Sprintf(
				"review coverageの件数に負数があります: available=%d partial=%d notEvaluated=%d",
				coverage.Available,
				coverage.Partial,
				coverage.NotEvaluated,
			),
		})
	}
	total := coverage.Available + coverage.Partial + coverage.NotEvaluated
	if total != 8 {
		diagnostics = append(diagnostics, Diagnostic{
			Code: "REVIEW_COVERAGE_COUNT_MISMATCH",
			Message: fmt.Sprintf(
				"review coverageのsection数がA1〜A8と一致しません: available=%d partial=%d notEvaluated=%d",
				coverage.Available,
				coverage.Partial,
				coverage.NotEvaluated,
			),
		})
	}

	validCodes := map[string]bool{
		"A1": true, "A2": true, "A3": true, "A4": true,
		"A5": true, "A6": true, "A7": true, "A8": true,
	}
	seen := map[string]bool{}
	var partialGaps, notEvaluatedGaps int
	for _, gap := range coverage.Incomplete {
		if !validCodes[gap.Code] || seen[gap.Code] {
			diagnostics = append(diagnostics, Diagnostic{
				Code:    "REVIEW_COVERAGE_GAP_INVALID",
				Message: "review coverageに不正または重複したsection codeがあります: " + gap.Code,
			})
		}
		seen[gap.Code] = true
		switch gap.Coverage {
		case reviewCoveragePartial:
			partialGaps++
		case reviewCoverageNotEvaluated:
			notEvaluatedGaps++
		default:
			diagnostics = append(diagnostics, Diagnostic{
				Code:    "REVIEW_COVERAGE_GAP_INVALID",
				Message: "review coverageのincomplete sectionがpartial / not-evaluatedではありません: " + gap.Code,
			})
		}
	}
	if partialGaps != coverage.Partial || notEvaluatedGaps != coverage.NotEvaluated {
		diagnostics = append(diagnostics, Diagnostic{
			Code: "REVIEW_COVERAGE_INCOMPLETE_MISMATCH",
			Message: fmt.Sprintf(
				"review coverageのincomplete内訳がsummaryと一致しません: partial=%d/%d notEvaluated=%d/%d",
				partialGaps,
				coverage.Partial,
				notEvaluatedGaps,
				coverage.NotEvaluated,
			),
		})
	}
	if projection := binding.Projection; projection != nil {
		if projection.Schema != reviewSchema {
			diagnostics = append(diagnostics, Diagnostic{
				Code: "REVIEW_BINDING_PROJECTION_SCHEMA_MISMATCH",
				Message: fmt.Sprintf(
					"review bindingのProjection schemaが一致しません: recorded=%s expected=%s",
					projection.Schema,
					reviewSchema,
				),
			})
		}
		if projection.ReviewFingerprint != "" && projection.ReviewFingerprint != binding.Fingerprint {
			diagnostics = append(diagnostics, Diagnostic{
				Code: "REVIEW_BINDING_PROJECTION_ID_MISMATCH",
				Message: fmt.Sprintf(
					"review bindingのProjection内fingerprintがbindingと一致しません: projection=%s binding=%s",
					projection.ReviewFingerprint,
					binding.Fingerprint,
				),
			})
		}
		if calculated := fingerprintReviewProjection(*projection); calculated != binding.Fingerprint {
			diagnostics = append(diagnostics, Diagnostic{
				Code: "REVIEW_BINDING_PROJECTION_FINGERPRINT_MISMATCH",
				Message: fmt.Sprintf(
					"review bindingのProjection内容がfingerprintと一致しません: recorded=%s calculated=%s",
					binding.Fingerprint,
					calculated,
				),
			})
		}
		if !reflect.DeepEqual(projection.Coverage, coverage) {
			diagnostics = append(diagnostics, Diagnostic{
				Code:    "REVIEW_BINDING_COVERAGE_MISMATCH",
				Message: "review bindingのcoverageが保存済みProjectionのcoverageと一致しません",
			})
		}
	}

	if len(diagnostics) > 0 {
		return "invalid", "invalid", diagnostics
	}
	status := "recorded"
	if binding.Projection != nil {
		status = "verified"
	}
	if coverage.Partial == 0 && coverage.NotEvaluated == 0 {
		return status, "complete", nil
	}
	return status, "incomplete", nil
}
