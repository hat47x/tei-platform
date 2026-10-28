package pluginbuildsupply

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"

	"tei/internal/plugin"
)

const (
	attemptSchema         = "tei.reference.plugin-build-supply.attempt/v0"
	reproducibilitySchema = "tei.reference.plugin-build-supply.reproducibility/v0"

	OutcomeSucceeded = "succeeded"
	OutcomeFailed    = "failed"

	// Environment variables of the builder child (the test binary).
	EnvRole          = "TEI_PBS_ROLE"
	RoleBuilder      = "builder-attempt"
	envOccurrence    = "TEI_PBS_OCCURRENCE"
	builderStartLine = "started"
)

// Builder modes. Only ModeBuild runs the real Go build; the others are
// simulated builders used for failure injection and say so in the record.
const (
	ModeBuild            = "build"
	ModePartial          = "simulated-partial-output"
	ModeNondeterministic = "simulated-nondeterministic-output"
)

var (
	ErrNotIssued               = errors.New("occurrence was not issued by this host")
	ErrOccurrenceCounted       = errors.New("occurrence is already counted in the history")
	ErrInsufficientOccurrences = errors.New("reproducibility needs two successful attempts with distinct occurrences")
	ErrNotReproducible         = errors.New("two successful attempts produced different artifacts")
	ErrPlanMismatch            = errors.New("attempt belongs to another build plan")
	ErrRecordAltered           = errors.New("attempt record does not match its fingerprint")
)

// Termination is the typed end of a builder process as the host saw it from
// the operating system, not as the builder reported it.
type Termination struct {
	Exited   bool   `json:"exited"`
	ExitCode int    `json:"exitCode"`
	Signal   string `json:"signal,omitempty"`
}

// PartialOutput is what a failed attempt left at its output path. It is
// moved aside and can never be stored or counted.
type PartialOutput struct {
	Digest      string `json:"digest"`
	Bytes       int64  `json:"bytes"`
	Quarantined bool   `json:"quarantined"`
}

// AttemptRecord is the evidence of one builder launch.
type AttemptRecord struct {
	Schema          string                 `json:"schema"`
	PlanFingerprint string                 `json:"planFingerprint"`
	Builder         plugin.BuilderIdentity `json:"builder"`
	OccurrenceID    string                 `json:"occurrenceId"`
	Ordinal         int                    `json:"ordinal"`
	Mode            string                 `json:"mode"`
	Outcome         string                 `json:"outcome"`
	Termination     Termination            `json:"termination"`
	ArtifactDigest  string                 `json:"artifactDigest,omitempty"`
	PartialOutput   *PartialOutput         `json:"partialOutput,omitempty"`
	Fingerprint     string                 `json:"fingerprint"`

	outputPath string
}

func attemptFingerprint(r AttemptRecord) (string, error) {
	r.Fingerprint = ""
	return fingerprintJSON(r)
}

type occurrence struct {
	planFingerprint string
	ordinal         int
	terminated      bool
	outcome         string
}

// Host launches builder processes and issues one occurrence ID per launch.
// The ID comes from the host before the process starts; a builder cannot
// choose or repeat it.
type Host struct {
	// Executable is the builder child (the test binary in this sample).
	Executable string
	// WorkDir holds one directory per attempt and the quarantine.
	WorkDir string

	mu      sync.Mutex
	issued  map[string]*occurrence
	ordinal int
}

func NewHost(executable, workDir string) *Host {
	return &Host{Executable: executable, WorkDir: workDir, issued: map[string]*occurrence{}}
}

func (h *Host) issue(planFingerprint string) (string, int, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", 0, err
	}
	id := hex.EncodeToString(raw[:])
	h.mu.Lock()
	defer h.mu.Unlock()
	h.ordinal++
	h.issued[id] = &occurrence{planFingerprint: planFingerprint, ordinal: h.ordinal}
	return id, h.ordinal, nil
}

func (h *Host) finish(id, outcome string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if o := h.issued[id]; o != nil {
		o.terminated = true
		o.outcome = outcome
	}
}

// IssuedCount is the number of launches the host performed.
func (h *Host) IssuedCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.issued)
}

func (h *Host) lookup(id string) (occurrence, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	o, ok := h.issued[id]
	if !ok {
		return occurrence{}, false
	}
	return *o, true
}

// AttemptSpec is one launch of the builder for a confirmed plan.
type AttemptSpec struct {
	Plan   plugin.BuildPlan
	Lock   ResolutionLock
	Trust  TrustStore
	Repo   string
	GoPath string
	Mode   string
	// OnStarted runs after the builder reported that it started. Calling
	// kill stops the whole process group of the attempt.
	OnStarted func(occurrenceID string, kill func() error)
}

// Run launches one attempt. A revoked builder is refused before any
// occurrence is issued.
func (h *Host) Run(ctx context.Context, spec AttemptSpec) (AttemptRecord, error) {
	if err := spec.Trust.Admit(spec.Lock); err != nil {
		return AttemptRecord{}, err
	}
	if spec.Plan.Builder != spec.Lock.Builder {
		return AttemptRecord{}, fmt.Errorf("plan builder differs from the resolution lock")
	}
	planFP, err := plugin.BuildPlanFingerprint(spec.Plan)
	if err != nil {
		return AttemptRecord{}, err
	}
	id, ordinal, err := h.issue(planFP)
	if err != nil {
		return AttemptRecord{}, err
	}
	dir := filepath.Join(h.WorkDir, "attempt-"+id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return AttemptRecord{}, err
	}
	out := filepath.Join(dir, "output")
	cmd := exec.CommandContext(ctx, h.Executable,
		"-repo", spec.Repo, "-go", spec.GoPath, "-package", spec.Plan.Recipe.Entrypoint,
		"-out", out, "-state", filepath.Join(dir, "state"), "-mode", spec.Mode)
	cmd.Env = append(os.Environ(), EnvRole+"="+RoleBuilder, envOccurrence+"="+id)
	cmd.Stderr = io.Discard
	setProcessGroup(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return AttemptRecord{}, err
	}
	if err := cmd.Start(); err != nil {
		h.finish(id, OutcomeFailed)
		return AttemptRecord{}, err
	}
	reader := bufio.NewReader(stdout)
	line, _ := reader.ReadString('\n')
	if line == builderStartLine+"\n" && spec.OnStarted != nil {
		spec.OnStarted(id, func() error { return killProcessGroup(cmd) })
	}
	_, _ = io.Copy(io.Discard, reader)
	waitErr := cmd.Wait()
	var exitErr *exec.ExitError
	if waitErr != nil && !errors.As(waitErr, &exitErr) {
		h.finish(id, OutcomeFailed)
		return AttemptRecord{}, waitErr
	}
	record := AttemptRecord{
		Schema: attemptSchema, PlanFingerprint: planFP, Builder: spec.Plan.Builder,
		OccurrenceID: id, Ordinal: ordinal, Mode: spec.Mode, Termination: termination(cmd.ProcessState),
		outputPath: out,
	}
	info, statErr := os.Lstat(out)
	hasOutput := statErr == nil && info.Mode().IsRegular()
	if record.Termination.Exited && record.Termination.ExitCode == 0 && hasOutput && info.Size() > 0 {
		record.Outcome = OutcomeSucceeded
		if record.ArtifactDigest, err = hashFile(out); err != nil {
			return AttemptRecord{}, err
		}
	} else {
		record.Outcome = OutcomeFailed
		if hasOutput {
			digest, err := hashFile(out)
			if err != nil {
				return AttemptRecord{}, err
			}
			quarantine := filepath.Join(h.WorkDir, "quarantine")
			if err := os.MkdirAll(quarantine, 0o755); err != nil {
				return AttemptRecord{}, err
			}
			moved := os.Rename(out, filepath.Join(quarantine, id)) == nil
			record.PartialOutput = &PartialOutput{Digest: digest, Bytes: info.Size(), Quarantined: moved}
			record.outputPath = ""
		}
	}
	if record.Fingerprint, err = attemptFingerprint(record); err != nil {
		return AttemptRecord{}, err
	}
	h.finish(id, record.Outcome)
	return record, nil
}

// History is the attempt log of one confirmed plan.
type History struct {
	PlanFingerprint string          `json:"planFingerprint"`
	Attempts        []AttemptRecord `json:"attempts"`
}

// Append accepts only records this host issued, for this plan, once each.
func (h *History) Append(host *Host, r AttemptRecord) error {
	if r.PlanFingerprint != h.PlanFingerprint {
		return ErrPlanMismatch
	}
	fp, err := attemptFingerprint(r)
	if err != nil {
		return err
	}
	if fp != r.Fingerprint {
		return ErrRecordAltered
	}
	occ, ok := host.lookup(r.OccurrenceID)
	if !ok || !occ.terminated || occ.planFingerprint != r.PlanFingerprint || occ.ordinal != r.Ordinal || occ.outcome != r.Outcome {
		return ErrNotIssued
	}
	for _, existing := range h.Attempts {
		if existing.OccurrenceID == r.OccurrenceID {
			return ErrOccurrenceCounted
		}
		if existing.Ordinal == r.Ordinal {
			return fmt.Errorf("attempt ordinal %d is already used", r.Ordinal)
		}
	}
	h.Attempts = append(h.Attempts, r)
	return nil
}

// Reproducibility is derived only from successful attempts with distinct
// host-issued occurrences of the same plan.
type Reproducibility struct {
	Schema          string   `json:"schema"`
	PlanFingerprint string   `json:"planFingerprint"`
	Occurrences     []string `json:"occurrences"`
	Digests         []string `json:"digests"`
	Reproducible    bool     `json:"reproducible"`
	ArtifactDigest  string   `json:"artifactDigest,omitempty"`
	Fingerprint     string   `json:"fingerprint"`
}

// DeriveReproducibility re-checks every record against the host and refuses
// to derive anything from failures or from a single occurrence. A mismatch is
// returned as a record together with ErrNotReproducible.
func DeriveReproducibility(host *Host, h History) (Reproducibility, error) {
	var successes []AttemptRecord
	seen := map[string]bool{}
	for _, r := range h.Attempts {
		if r.PlanFingerprint != h.PlanFingerprint {
			return Reproducibility{}, ErrPlanMismatch
		}
		fp, err := attemptFingerprint(r)
		if err != nil {
			return Reproducibility{}, err
		}
		if fp != r.Fingerprint {
			return Reproducibility{}, ErrRecordAltered
		}
		occ, ok := host.lookup(r.OccurrenceID)
		if !ok || !occ.terminated || occ.outcome != r.Outcome || occ.ordinal != r.Ordinal {
			return Reproducibility{}, ErrNotIssued
		}
		if seen[r.OccurrenceID] {
			return Reproducibility{}, ErrOccurrenceCounted
		}
		seen[r.OccurrenceID] = true
		if r.Outcome == OutcomeSucceeded && r.ArtifactDigest != "" {
			successes = append(successes, r)
		}
	}
	if len(successes) < 2 {
		return Reproducibility{}, fmt.Errorf("%w (successful=%d)", ErrInsufficientOccurrences, len(successes))
	}
	sort.Slice(successes, func(a, b int) bool { return successes[a].Ordinal < successes[b].Ordinal })
	first, second := successes[0], successes[1]
	rep := Reproducibility{
		Schema: reproducibilitySchema, PlanFingerprint: h.PlanFingerprint,
		Occurrences:  []string{first.OccurrenceID, second.OccurrenceID},
		Digests:      []string{first.ArtifactDigest, second.ArtifactDigest},
		Reproducible: first.ArtifactDigest == second.ArtifactDigest,
	}
	if rep.Reproducible {
		rep.ArtifactDigest = first.ArtifactDigest
	}
	fp, err := fingerprintJSON(rep)
	if err != nil {
		return Reproducibility{}, err
	}
	rep.Fingerprint = fp
	if !rep.Reproducible {
		return rep, ErrNotReproducible
	}
	return rep, nil
}

// OutputOf returns the output path of a successful attempt.
func (r AttemptRecord) OutputOf() (string, error) {
	if r.Outcome != OutcomeSucceeded || r.outputPath == "" {
		return "", fmt.Errorf("attempt %s has no successful output", r.OccurrenceID)
	}
	return r.outputPath, nil
}
