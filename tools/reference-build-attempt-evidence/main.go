package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"tei/internal/core"
	"tei/internal/plugin"
)

const (
	attemptEvidenceSchema = "tei.reference.build-attempt-evidence/v0"
	attemptHistorySchema  = "tei.reference.build-attempt-history/v0"
	referenceSchema       = "tei.reference.build-attempt-evidence-run/v0"

	attemptOutcomeFailed = "failed"

	attemptPhaseCompile    = "compile"
	attemptPhaseLink       = "link"
	attemptPhaseBuilderRun = "builder-run"

	nodeAttemptFailed  = "failed"
	nodeAttemptBlocked = "blocked"
)

var attemptAuthorityLimits = []string{
	"attempt-evidence-is-not-build-success",
	"attempt-evidence-does-not-produce-artifact-lock",
	"failure-evidence-is-not-reproducibility-result",
	"diagnostic-digest-is-not-build-identity",
	"diagnostic-retention-is-not-publication-authority",
	"ai-repair-input-is-not-acceptance-authority",
}

type diagnosticCaptureEvidence struct {
	Digest        string `json:"digest"`
	Bytes         int64  `json:"bytes"`
	RetainedBytes int    `json:"retainedBytes"`
	Truncated     bool   `json:"truncated"`
	AccessClass   string `json:"accessClass"`
}

type buildAttemptEvidence struct {
	Schema               string                     `json:"schema"`
	AttemptOrdinal       int                        `json:"attemptOrdinal"`
	BuildPlanFingerprint string                     `json:"buildPlanFingerprint"`
	Builder              plugin.BuilderIdentity     `json:"builder"`
	Target               string                     `json:"target"`
	Profile              string                     `json:"profile,omitempty"`
	Phase                string                     `json:"phase"`
	Outcome              string                     `json:"outcome"`
	FailureClass         string                     `json:"failureClass"`
	ExitCode             *int                       `json:"exitCode,omitempty"`
	Signal               string                     `json:"signal,omitempty"`
	DiagnosticSummary    string                     `json:"diagnosticSummary"`
	Stdout               diagnosticCaptureEvidence  `json:"stdout"`
	Stderr               diagnosticCaptureEvidence  `json:"stderr"`
	AuthorityLimits      []string                   `json:"authorityLimits"`
	EvidenceFingerprint  string                     `json:"evidenceFingerprint"`
}

type buildAttemptHistory struct {
	Schema                    string                     `json:"schema"`
	BuildPlanFingerprint      string                     `json:"buildPlanFingerprint"`
	FailedAttemptFingerprints []string                   `json:"failedAttemptFingerprints"`
	SuccessfulResult          *successfulResultReference `json:"successfulResult,omitempty"`
}

type successfulResultReference struct {
	BuildPlanFingerprint string `json:"buildPlanFingerprint"`
	ResultFingerprint    string `json:"resultFingerprint"`
}

type buildGraphAttemptState struct {
	NodeID                     string   `json:"nodeId"`
	State                      string   `json:"state"`
	AttemptEvidenceFingerprint string   `json:"attemptEvidenceFingerprint,omitempty"`
	BlockedBy                  []string `json:"blockedBy,omitempty"`
}

type fakeFailure struct {
	Phase             string
	FailureClass      string
	ExitCode          *int
	Signal            string
	DiagnosticSummary string
	Stdout            []byte
	Stderr            []byte
}

type retainedDiagnostics struct {
	Stdout []byte
	Stderr []byte
}

type referenceEvidence struct {
	Schema          string                   `json:"schema"`
	Attempt         buildAttemptEvidence     `json:"attempt"`
	History         buildAttemptHistory      `json:"history"`
	GraphStates     []buildGraphAttemptState `json:"graphStates"`
	AuthorityLimits []string                 `json:"authorityLimits"`
}

func main() {
	evidence, err := buildReferenceEvidence()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(evidence); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	fmt.Fprintln(os.Stderr, "reference fake builder attempt failed as intended")
	os.Exit(1)
}

func buildReferenceEvidence() (referenceEvidence, error) {
	plan := referenceBuildPlan()
	exitCode := 2
	attempt, retained, err := newFailedAttempt(
		plan,
		1,
		fakeFailure{
			Phase:             attemptPhaseCompile,
			FailureClass:      "compile-type-error",
			ExitCode:          &exitCode,
			DiagnosticSummary: "compile failed with a structured type error",
			Stdout:            []byte("compile-progress\n"),
			Stderr: []byte(
				"private-path=/workspace/private/source.go " +
					"token=secret-never-inline " +
					"type mismatch in generated expression\n",
			),
		},
		32,
	)
	if err != nil {
		return referenceEvidence{}, err
	}
	_ = retained

	history, err := newAttemptHistory(plan)
	if err != nil {
		return referenceEvidence{}, err
	}
	if err := history.AppendFailure(plan, attempt); err != nil {
		return referenceEvidence{}, err
	}
	history.SuccessfulResult = &successfulResultReference{
		BuildPlanFingerprint: attempt.BuildPlanFingerprint,
		ResultFingerprint:    strings.Repeat("a", 64),
	}
	if err := history.Validate(); err != nil {
		return referenceEvidence{}, err
	}

	failedState := buildGraphAttemptState{
		NodeID:                     "generator",
		State:                      nodeAttemptFailed,
		AttemptEvidenceFingerprint: attempt.EvidenceFingerprint,
	}
	blockedState := buildGraphAttemptState{
		NodeID:    "consumer",
		State:     nodeAttemptBlocked,
		BlockedBy: []string{"generator"},
	}
	for _, state := range []buildGraphAttemptState{failedState, blockedState} {
		if err := state.Validate(); err != nil {
			return referenceEvidence{}, err
		}
	}

	return referenceEvidence{
		Schema:      referenceSchema,
		Attempt:     attempt,
		History:     history,
		GraphStates: []buildGraphAttemptState{failedState, blockedState},
		AuthorityLimits: append(
			[]string(nil),
			attemptAuthorityLimits...,
		),
	}, nil
}

func referenceBuildPlan() plugin.BuildPlan {
	return plugin.BuildPlan{
		FormatVersion: plugin.BuildPlanFormatVersion,
		Plugin: plugin.Identity{
			ID:      core.MustCanonicalID("tei.reference.failed_build"),
			Version: "1.0.0",
		},
		PluginDefinitionFingerprint: strings.Repeat("d", 64),
		Sources: []plugin.BuildSource{{
			Kind:    "reference-source",
			ID:      "failed-build-source",
			Locator: "reference://failed-build-source",
			Digest:  "sha256:" + strings.Repeat("a", 64),
		}},
		Builder: plugin.BuilderIdentity{
			ID:              core.MustCanonicalID("tei.builder.reference.failure"),
			Version:         "1.0.0",
			ContractVersion: "tei.reference.builder-failure/v1",
			ArtifactDigest:  "sha256:" + strings.Repeat("b", 64),
		},
		Recipe: plugin.BuildRecipe{
			Kind:       "deterministic-failure",
			Entrypoint: "failed-build-source",
		},
		OutputKind: plugin.BuildOutputRuntimeArtifact,
		Target:     "host/reference",
		Profile:    "failure-reference",
	}
}

func newFailedAttempt(
	plan plugin.BuildPlan,
	ordinal int,
	failure fakeFailure,
	maxRetainedBytes int,
) (buildAttemptEvidence, retainedDiagnostics, error) {
	if err := plan.Validate(); err != nil {
		return buildAttemptEvidence{}, retainedDiagnostics{}, fmt.Errorf("exact BuildPlan is required before Build Attempt Evidence: %w", err)
	}
	planFingerprint, err := plugin.BuildPlanFingerprint(plan)
	if err != nil {
		return buildAttemptEvidence{}, retainedDiagnostics{}, err
	}
	if ordinal <= 0 {
		return buildAttemptEvidence{}, retainedDiagnostics{}, fmt.Errorf("attempt ordinal must be positive")
	}
	if maxRetainedBytes < 0 {
		return buildAttemptEvidence{}, retainedDiagnostics{}, fmt.Errorf("max retained diagnostic bytes must not be negative")
	}

	stdoutEvidence, stdoutRetained, err := captureDiagnostic(
		strings.NewReader(string(failure.Stdout)),
		maxRetainedBytes,
	)
	if err != nil {
		return buildAttemptEvidence{}, retainedDiagnostics{}, err
	}
	stderrEvidence, stderrRetained, err := captureDiagnostic(
		strings.NewReader(string(failure.Stderr)),
		maxRetainedBytes,
	)
	if err != nil {
		return buildAttemptEvidence{}, retainedDiagnostics{}, err
	}

	evidence := buildAttemptEvidence{
		Schema:               attemptEvidenceSchema,
		AttemptOrdinal:       ordinal,
		BuildPlanFingerprint: planFingerprint,
		Builder:              plan.Builder,
		Target:               plan.Target,
		Profile:              plan.Profile,
		Phase:                failure.Phase,
		Outcome:              attemptOutcomeFailed,
		FailureClass:         failure.FailureClass,
		ExitCode:             failure.ExitCode,
		Signal:               failure.Signal,
		DiagnosticSummary:    failure.DiagnosticSummary,
		Stdout:               stdoutEvidence,
		Stderr:               stderrEvidence,
		AuthorityLimits:      append([]string(nil), attemptAuthorityLimits...),
	}
	if err := evidence.ValidateAgainst(plan); err != nil {
		return buildAttemptEvidence{}, retainedDiagnostics{}, err
	}
	evidence.EvidenceFingerprint, err = buildAttemptEvidenceFingerprint(evidence)
	if err != nil {
		return buildAttemptEvidence{}, retainedDiagnostics{}, err
	}
	if err := evidence.ValidateAgainst(plan); err != nil {
		return buildAttemptEvidence{}, retainedDiagnostics{}, err
	}
	return evidence, retainedDiagnostics{
		Stdout: stdoutRetained,
		Stderr: stderrRetained,
	}, nil
}

func (e buildAttemptEvidence) ValidateAgainst(plan plugin.BuildPlan) error {
	if e.Schema != attemptEvidenceSchema {
		return fmt.Errorf("unsupported Build Attempt Evidence schema %q", e.Schema)
	}
	if e.AttemptOrdinal <= 0 {
		return fmt.Errorf("attempt ordinal must be positive")
	}
	if err := plan.Validate(); err != nil {
		return err
	}
	planFingerprint, err := plugin.BuildPlanFingerprint(plan)
	if err != nil {
		return err
	}
	if normalizeFingerprint(e.BuildPlanFingerprint) != normalizeFingerprint(planFingerprint) {
		return fmt.Errorf("Build Attempt Evidence does not match exact BuildPlan")
	}
	if !sameBuilderIdentity(e.Builder, plan.Builder) {
		return fmt.Errorf("Build Attempt Evidence does not match exact Builder identity")
	}
	if e.Target != plan.Target || e.Profile != plan.Profile {
		return fmt.Errorf("Build Attempt Evidence target/profile do not match BuildPlan")
	}
	if !validAttemptPhase(e.Phase) {
		return fmt.Errorf("unsupported Build Attempt phase %q", e.Phase)
	}
	if e.Outcome != attemptOutcomeFailed {
		return fmt.Errorf("Build Attempt Evidence outcome must be failed")
	}
	if strings.TrimSpace(e.FailureClass) == "" {
		return fmt.Errorf("Build Attempt Evidence failure class is required")
	}
	if e.ExitCode == nil && strings.TrimSpace(e.Signal) == "" {
		return fmt.Errorf("Build Attempt Evidence requires exit code or signal")
	}
	if strings.TrimSpace(e.DiagnosticSummary) == "" {
		return fmt.Errorf("Build Attempt Evidence diagnostic summary is required")
	}
	if err := e.Stdout.Validate(); err != nil {
		return fmt.Errorf("Build Attempt stdout: %w", err)
	}
	if err := e.Stderr.Validate(); err != nil {
		return fmt.Errorf("Build Attempt stderr: %w", err)
	}
	if err := requireAuthorityLimits(e.AuthorityLimits, attemptAuthorityLimits); err != nil {
		return err
	}
	if e.EvidenceFingerprint != "" {
		if err := validateFingerprint(e.EvidenceFingerprint); err != nil {
			return fmt.Errorf("Build Attempt Evidence fingerprint: %w", err)
		}
		expected, err := buildAttemptEvidenceFingerprint(e)
		if err != nil {
			return err
		}
		if normalizeFingerprint(e.EvidenceFingerprint) != expected {
			return fmt.Errorf("Build Attempt Evidence fingerprint does not match its content")
		}
	}
	return nil
}

func (d diagnosticCaptureEvidence) Validate() error {
	if _, err := normalizeDigest(d.Digest); err != nil {
		return err
	}
	if d.Bytes < 0 || d.RetainedBytes < 0 || int64(d.RetainedBytes) > d.Bytes {
		return fmt.Errorf("invalid diagnostic byte counts")
	}
	if d.Truncated != (int64(d.RetainedBytes) < d.Bytes) {
		return fmt.Errorf("diagnostic truncation flag does not match retained bytes")
	}
	if d.AccessClass != "restricted-build-diagnostic" {
		return fmt.Errorf("diagnostic access class must remain restricted")
	}
	return nil
}

func captureDiagnostic(
	reader io.Reader,
	maxRetainedBytes int,
) (diagnosticCaptureEvidence, []byte, error) {
	if maxRetainedBytes < 0 {
		return diagnosticCaptureEvidence{}, nil, fmt.Errorf("max retained diagnostic bytes must not be negative")
	}
	hasher := sha256.New()
	retained := make([]byte, 0, maxRetainedBytes)
	buffer := make([]byte, 4096)
	var observed int64
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			chunk := buffer[:n]
			if _, writeErr := hasher.Write(chunk); writeErr != nil {
				return diagnosticCaptureEvidence{}, nil, writeErr
			}
			observed += int64(n)
			if len(retained) < maxRetainedBytes {
				remaining := maxRetainedBytes - len(retained)
				if remaining > n {
					remaining = n
				}
				retained = append(retained, chunk[:remaining]...)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return diagnosticCaptureEvidence{}, nil, err
		}
	}
	return diagnosticCaptureEvidence{
		Digest:        "sha256:" + hex.EncodeToString(hasher.Sum(nil)),
		Bytes:         observed,
		RetainedBytes: len(retained),
		Truncated:     int64(len(retained)) < observed,
		AccessClass:   "restricted-build-diagnostic",
	}, retained, nil
}

func buildAttemptEvidenceFingerprint(e buildAttemptEvidence) (string, error) {
	e.EvidenceFingerprint = ""
	e.BuildPlanFingerprint = normalizeFingerprint(e.BuildPlanFingerprint)
	e.Builder.ArtifactDigest = normalizeDigestMust(e.Builder.ArtifactDigest)
	e.Stdout.Digest = normalizeDigestMust(e.Stdout.Digest)
	e.Stderr.Digest = normalizeDigestMust(e.Stderr.Digest)
	e.AuthorityLimits = append([]string(nil), e.AuthorityLimits...)
	sort.Strings(e.AuthorityLimits)
	raw, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func newAttemptHistory(plan plugin.BuildPlan) (buildAttemptHistory, error) {
	fingerprint, err := plugin.BuildPlanFingerprint(plan)
	if err != nil {
		return buildAttemptHistory{}, err
	}
	return buildAttemptHistory{
		Schema:               attemptHistorySchema,
		BuildPlanFingerprint: fingerprint,
	}, nil
}

func (h *buildAttemptHistory) AppendFailure(
	plan plugin.BuildPlan,
	e buildAttemptEvidence,
) error {
	if h == nil {
		return fmt.Errorf("Build Attempt History is nil")
	}
	planFingerprint, err := plugin.BuildPlanFingerprint(plan)
	if err != nil {
		return err
	}
	if normalizeFingerprint(h.BuildPlanFingerprint) != normalizeFingerprint(planFingerprint) {
		return fmt.Errorf("Build Attempt History does not match exact BuildPlan")
	}
	if err := e.ValidateAgainst(plan); err != nil {
		return fmt.Errorf("Build Attempt History failure Evidence: %w", err)
	}
	h.FailedAttemptFingerprints = append(
		h.FailedAttemptFingerprints,
		normalizeFingerprint(e.EvidenceFingerprint),
	)
	return nil
}

func (h buildAttemptHistory) Validate() error {
	if h.Schema != attemptHistorySchema {
		return fmt.Errorf("unsupported Build Attempt History schema %q", h.Schema)
	}
	if err := validateFingerprint(h.BuildPlanFingerprint); err != nil {
		return err
	}
	for _, fingerprint := range h.FailedAttemptFingerprints {
		if err := validateFingerprint(fingerprint); err != nil {
			return err
		}
	}
	if h.SuccessfulResult != nil {
		if err := validateFingerprint(h.SuccessfulResult.BuildPlanFingerprint); err != nil {
			return fmt.Errorf("successful result BuildPlan fingerprint: %w", err)
		}
		if normalizeFingerprint(h.SuccessfulResult.BuildPlanFingerprint) != normalizeFingerprint(h.BuildPlanFingerprint) {
			return fmt.Errorf("successful result belongs to a different BuildPlan")
		}
		if err := validateFingerprint(h.SuccessfulResult.ResultFingerprint); err != nil {
			return fmt.Errorf("successful result fingerprint: %w", err)
		}
	}
	return nil
}

func (s buildGraphAttemptState) Validate() error {
	if strings.TrimSpace(s.NodeID) == "" {
		return fmt.Errorf("Build graph attempt state node id is required")
	}
	switch s.State {
	case nodeAttemptFailed:
		if err := validateFingerprint(s.AttemptEvidenceFingerprint); err != nil {
			return fmt.Errorf("failed node requires Build Attempt Evidence fingerprint: %w", err)
		}
		if len(s.BlockedBy) != 0 {
			return fmt.Errorf("failed node must not also be blocked")
		}
	case nodeAttemptBlocked:
		if s.AttemptEvidenceFingerprint != "" {
			return fmt.Errorf("blocked node must not claim a Build Attempt Evidence fingerprint")
		}
		if len(s.BlockedBy) == 0 {
			return fmt.Errorf("blocked node requires at least one blocker")
		}
		seen := map[string]struct{}{}
		for _, blocker := range s.BlockedBy {
			if strings.TrimSpace(blocker) == "" {
				return fmt.Errorf("blocked node blocker id is required")
			}
			if _, duplicate := seen[blocker]; duplicate {
				return fmt.Errorf("blocked node blocker %s is declared more than once", blocker)
			}
			seen[blocker] = struct{}{}
		}
	default:
		return fmt.Errorf("unsupported Build graph attempt state %q", s.State)
	}
	return nil
}

func validAttemptPhase(phase string) bool {
	switch phase {
	case attemptPhaseCompile, attemptPhaseLink, attemptPhaseBuilderRun:
		return true
	default:
		return false
	}
}

func sameBuilderIdentity(left, right plugin.BuilderIdentity) bool {
	return left.ID == right.ID &&
		left.Version == right.Version &&
		left.ContractVersion == right.ContractVersion &&
		normalizeDigestMust(left.ArtifactDigest) == normalizeDigestMust(right.ArtifactDigest)
}

func requireAuthorityLimits(got, required []string) error {
	have := map[string]struct{}{}
	for _, value := range got {
		have[value] = struct{}{}
	}
	for _, value := range required {
		if _, ok := have[value]; !ok {
			return fmt.Errorf("Build Attempt Evidence authority limit %q is missing", value)
		}
	}
	return nil
}

func normalizeFingerprint(value string) string {
	return strings.ToLower(value)
}

func validateFingerprint(value string) error {
	if len(value) != 64 {
		return fmt.Errorf("fingerprint must contain 64 hex characters")
	}
	for _, r := range value {
		if !((r >= '0' && r <= '9') ||
			(r >= 'a' && r <= 'f') ||
			(r >= 'A' && r <= 'F')) {
			return fmt.Errorf("fingerprint contains non-hex character")
		}
	}
	return nil
}

func normalizeDigest(value string) (string, error) {
	if !strings.HasPrefix(value, "sha256:") {
		return "", fmt.Errorf("digest must use sha256:<64-hex>")
	}
	hexPart := strings.TrimPrefix(value, "sha256:")
	if err := validateFingerprint(hexPart); err != nil {
		return "", err
	}
	return "sha256:" + strings.ToLower(hexPart), nil
}

func normalizeDigestMust(value string) string {
	normalized, err := normalizeDigest(value)
	if err != nil {
		return value
	}
	return normalized
}
