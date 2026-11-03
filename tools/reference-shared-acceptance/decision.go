package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
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
	// Warningsは、判定自体は成立したが、付随する記録・出力を残せなかったことを示す。
	// Refusalsと違い、Sourceの受理状態を覆さない。現在はDecision ledgerの書込み失敗だけが入る。
	Warnings []Diagnostic `json:"warnings,omitempty"`
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
	// Pin the named Workspace before any read; concurrent Shared Acceptance
	// operations must use the SAME create-only lock, including Preview,
	// mutation, conformance reload, and rollback. Uncooperative external
	// writers are NOT serialized by this Reference.
	files, err := os.OpenRoot(root)
	if err != nil {
		return Decision{}, fmt.Errorf("open Acceptance Workspace root: %w", err)
	}
	defer files.Close()

	guard, err := acquireSourceAcceptLock(files)
	if errors.Is(err, errSourceAcceptBusy) {
		return Decision{
			Schema:        decisionSchema,
			Decision:      "refused",
			SourceChanged: false,
			Refusals: []Diagnostic{{
				Code:    "SOURCE_ACCEPT_BUSY",
				Message: "別の受理操作または未確認の古いlockが存在します。処理を再送せず、現在のSourceとlockを確認してください",
			}},
		}, nil
	}
	if err != nil {
		return Decision{}, fmt.Errorf("acquire Workspace Acceptance lock: %w", err)
	}
	decision, acceptErr := acceptWithPinnedRoot(root, files, data, opts)
	releaseErr := guard.release()
	if releaseErr != nil {
		return decision, errors.Join(acceptErr, fmt.Errorf(
			"Acceptance outcome may have changed Source, but lock cleanup could not be confirmed: %w", releaseErr))
	}
	return decision, acceptErr
}

// acceptWithPinnedRoot runs only while the caller owns the cooperative lock.
// This is still the SAME shared Acceptance implementation, not a new entry
// point for Studio or AI. Call accept, never this helper, from adapters.
func acceptWithPinnedRoot(root string, files *os.Root, data []byte, opts acceptOptions) (Decision, error) {
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
	// Rollback must not blindly overwrite a file another process modified
	// after our write. This is a best-effort byte guard, NOT an atomic
	// compare-and-swap: path replacement between read and restore remains
	// unresolved until the common Source Authority owns the write fence.
	rollback := func() []string {
		var incomplete []string
		for i := len(written) - 1; i >= 0; i-- {
			e := written[i]
			current, err := sourceReadWithinRoot(files, e.Path)
			if err != nil || sourceindex.Fingerprint(current) != e.AfterFingerprint {
				incomplete = append(incomplete, e.Path)
				continue
			}
			if e.Created {
				err = sourceRemoveWithinRoot(files, e.Path)
			} else {
				err = sourceWriteWithinRoot(files, e.Path, e.before, false)
			}
			if err != nil {
				incomplete = append(incomplete, e.Path)
			}
		}
		incomplete = append(incomplete, sourceRemoveDirsIfEmpty(files, createdDirs)...)
		return incomplete
	}
	refuse := func(code, message string) Decision {
		incomplete := rollback()
		decision.Decision = "refused"
		decision.SourceChanged = len(incomplete) > 0
		decision.Refusals = append(decision.Refusals, Diagnostic{Code: code, Message: message})
		if len(incomplete) > 0 {
			sort.Strings(incomplete)
			decision.Refusals = append(decision.Refusals, Diagnostic{
				Code:    "ROLLBACK_INCOMPLETE",
				Message: fmt.Sprintf("並行変更または書込み失敗により元のSourceへ完全に戻せません: %s。再読込と明示的な競合解消が必要です", strings.Join(incomplete, ", ")),
			})
		}
		return decision
	}

	if err := checkAcceptanceManifest(files, ev.before.manifest); err != nil {
		return refuse("SOURCE_MANIFEST_CHANGED_DURING_ACCEPT", err.Error()), nil
	}
	for i, e := range ev.edits {
		if beforeEachWriteHook != nil {
			beforeEachWriteHook(i, e)
		}
		if err := checkAcceptedWorkspaceIdentity(root, files); err != nil {
			return refuse("SOURCE_WORKSPACE_REPLACED", err.Error()), nil
		}
		if err := checkAcceptanceManifest(files, ev.before.manifest); err != nil {
			return refuse("SOURCE_MANIFEST_CHANGED_DURING_ACCEPT", err.Error()), nil
		}
		if e.Created {
			name, err := acceptedSourcePath(e.Path)
			if err != nil {
				return refuse("SOURCE_WRITE_FAILED", err.Error()), nil
			}
			_, err = files.Lstat(name)
			if err == nil {
				return refuse("SOURCE_CHANGED_DURING_ACCEPT", fmt.Sprintf("作成先 %s がPreviewの後に作られました", e.Path)), nil
			}
			if !os.IsNotExist(err) {
				return refuse("SOURCE_UNREADABLE", err.Error()), nil
			}
			dirs, err := sourceMakeDirs(files, filepath.Dir(name))
			createdDirs = append(createdDirs, dirs...)
			if err != nil {
				return refuse("SOURCE_WRITE_FAILED", err.Error()), nil
			}
		} else {
			current, err := sourceReadWithinRoot(files, e.Path)
			if err != nil {
				return refuse("SOURCE_UNREADABLE", err.Error()), nil
			}
			if sourceindex.Fingerprint(current) != e.BeforeFingerprint {
				return refuse("SOURCE_CHANGED_DURING_ACCEPT", fmt.Sprintf("%s がPreviewの後に変更されました", e.Path)), nil
			}
		}
		if err := sourceWriteWithinRoot(files, e.Path, e.after, e.Created); err != nil {
			if e.Created && os.IsExist(err) {
				return refuse("SOURCE_CHANGED_DURING_ACCEPT", fmt.Sprintf("作成先 %s が書込み前に作られました", e.Path)), nil
			}
			return refuse("SOURCE_WRITE_FAILED", err.Error()), nil
		}
		written = append(written, e)
	}

	if afterWritesHook != nil {
		afterWritesHook(root)
	}
	if err := checkAcceptanceManifest(files, ev.before.manifest); err != nil {
		return refuse("SOURCE_MANIFEST_CHANGED_DURING_ACCEPT", err.Error()), nil
	}
	post, err := takeSnapshot(root)
	if err != nil {
		return refuse("POST_ACCEPT_RELOAD_FAILED", err.Error()), nil
	}
	if post.manifest != ev.before.manifest {
		return refuse("SOURCE_MANIFEST_CHANGED_DURING_ACCEPT", "Workspace manifest changed during post-Accept reload"), nil
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

	if err := checkAcceptanceManifest(files, ev.before.manifest); err != nil {
		return refuse("SOURCE_MANIFEST_CHANGED_DURING_ACCEPT", err.Error()), nil
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
	// DecisionWitnessStatusは、OpenSSH署名witness（.tei/decision-witnesses）とこの
	// recordのexact binding状態を示す。none / recorded / verified / invalid /
	// conflicted。verifiedでも受理状態やaccepted meaningは変わらない。
	DecisionWitnessStatus string       `json:"decisionWitnessStatus"`
	WitnessDiagnostics    []Diagnostic `json:"witnessDiagnostics,omitempty"`
	Decision              Decision     `json:"decision"`
}

type DecisionIndex struct {
	Schema        string           `json:"schema"`
	ModelRevision string           `json:"modelRevision"`
	Records       []DecisionRecord `json:"records"`
}

// recordDecisionは、決定を派生物として保存し、保存先のworkspace相対pathを返す。
// 内容でaddressingするため、同じ決定を二度保存しても増えない。
// この記録はauthorityではない。消してもaccepted meaningもmodel revisionも変わらない。
// 書き込みはWorkspace rootに固定したos.Rootの相対操作で行い、ledger directoryの
// 外向きシンボリックリンクを辿らない。
func recordDecision(root string, d Decision) (string, error) {
	d.RecordPath = ""
	encoded, err := encodeJSON(d)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	rel := decisionDir + "/" + hex.EncodeToString(sum[:]) + ".json"
	files, err := os.OpenRoot(root)
	if err != nil {
		return "", fmt.Errorf("open Workspace root for Decision ledger: %w", err)
	}
	defer files.Close()
	if err := ledgerWriteFile(files, rel, encoded); err != nil {
		return "", err
	}
	return rel, nil
}

// loadDecisionRecordsは、保存された決定を読み、現在のworkspaceに対して検証する。
// 記録の内容を信用してmeaningを再構成せず、現在のsourceと照合した結果だけを付ける。
// witnessがnilでない場合、.tei/decision-witnessesのOpenSSH署名witnessをrecord本文に
// exactにbindingして検証する。nilなら検証せず、witnessがあってもstatusを解決しない。
func loadDecisionRecords(root string, witness *WitnessVerifyOptions) (DecisionIndex, error) {
	index := DecisionIndex{Schema: decisionIndexSchema, Records: []DecisionRecord{}}
	s, err := takeSnapshot(root)
	if err != nil {
		return index, err
	}
	index.ModelRevision = s.revision

	files, err := os.OpenRoot(root)
	if err != nil {
		return index, fmt.Errorf("open Workspace root for Decision ledger: %w", err)
	}
	defer files.Close()
	entries, err := ledgerReadDir(files, decisionDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
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

	// witness判定にはrecord本文のdigestと、mutate可能なrecordの参照が必要である。
	// 最後に値へ戻してindexへ入れる。
	witnessRecords := make([]*DecisionRecord, 0, len(names))
	witnessDigests := make(map[string]string, len(names))

	for _, name := range names {
		rel := decisionDir + "/" + name
		content, err := ledgerReadFile(files, rel)
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
			DecisionActorStatus:     actorStatusUnresolved,
			DecisionAuthorityStatus: authorityStatusUnresolved,
			DecisionWitnessStatus:   witnessStatusNone,
		}
		if d.ModelRevisionAfter == s.revision {
			record.Status = "current"
		}
		if len(d.SourceEdits) > 0 {
			record.SourceEdits = "intact"
			for _, e := range d.SourceEdits {
				current, err := sourceReadWithinRoot(files, e.Path)
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
		witnessRecords = append(witnessRecords, &record)
		witnessDigests[rel] = "sha256:" + hex.EncodeToString(sum[:])
	}

	witnesses, err := loadDecisionWitnesses(files)
	if err != nil {
		return index, err
	}
	verifier := (*witnessVerifyInternal)(nil)
	if witness != nil {
		// loader層ではprovider binaryをPATHから選ばせる（flagsの明示指定は
		// Reference / test用に限定する）。
		verifier = &witnessVerifyInternal{WitnessVerifyOptions: *witness}
	}
	if err := bindDecisionWitnesses(witnessRecords, witnessDigests, witnesses, verifier); err != nil {
		return index, err
	}
	for _, record := range witnessRecords {
		index.Records = append(index.Records, *record)
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
