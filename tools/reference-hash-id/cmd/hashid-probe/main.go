// hashid-probe is a read-only consumer of the experimental hashid Reference.
// It deliberately has no allocation, authorization, Source Acceptance, or
// canonical identity write operation.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	hashid "tei/tools/reference-hash-id"
)

const (
	maxInputBytes  = 4 << 20
	maxTargetBytes = 512 << 10
	targetWireV1   = "tei.hashid.target-probe/v1"
)

type targetProfileWire struct {
	Namespace       string   `json:"namespace"`
	ProfileVersion  string   `json:"profileVersion"`
	AllowedAlphabet string   `json:"allowedAlphabet"`
	MinLength       int      `json:"minLength"`
	MaxLength       int      `json:"maxLength"`
	ExactLength     int      `json:"exactLength"`
	CaseFoldASCII   bool     `json:"caseFoldASCII"`
	Reserved        []string `json:"reserved"`
}

type bindingWire struct {
	Namespace      string `json:"namespace"`
	ProfileVersion string `json:"profileVersion"`
	Subject        string `json:"subject"`
	SchemeJSON     string `json:"schemeJSON"`
	Text           string `json:"text"`
}

type targetProbeWire struct {
	Format   string            `json:"format"`
	Profile  targetProfileWire `json:"profile"`
	Bindings []bindingWire     `json:"bindings"`
}

// collisionBudgetEvidence is a caller-declared birthday-risk estimate,
// NOT a target policy acceptance, cryptographic strength certificate, or
// authority to allocate a physical identifier.
type collisionBudgetEvidence struct {
	ExpectedPopulation            uint64  `json:"expectedPopulation"`
	MaximumCollisionProbability   float64 `json:"maximumCollisionProbability"`
	EstimatedCollisionProbability float64 `json:"estimatedCollisionProbability"`
	CollisionRiskUpperBound       float64 `json:"collisionRiskUpperBound"`
	PolicySource                  string  `json:"policySource"`
}

// report never claims target allocation or semantic Source Acceptance.
type report struct {
	Status               string                   `json:"status"`
	Authority            string                   `json:"authority"`
	StorageWrite         bool                     `json:"storageWrite"`
	Subject              string                   `json:"subject"`
	SchemeJSON           string                   `json:"schemeJSON"`
	SourceByteCount      int                      `json:"sourceByteCount"`
	SourceSHA256         string                   `json:"sourceSHA256"`
	HashIDText           string                   `json:"hashIdText"`
	DigestHex            string                   `json:"digestHex"`
	TargetNamespace      string                   `json:"targetNamespace,omitempty"`
	NormalizedTargetKey  string                   `json:"normalizedTargetKey,omitempty"`
	TargetSnapshotSHA256 string                   `json:"targetSnapshotSHA256,omitempty"`
	CallerPinsMatched    bool                     `json:"callerPinsMatched,omitempty"`
	CollisionBudget      *collisionBudgetEvidence `json:"collisionBudget,omitempty"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("hashid-probe", flag.ContinueOnError)
	flags.SetOutput(stderr)
	schemePath := flags.String("scheme", "", "path to canonical Scheme Wire JSON")
	inputPath := flags.String("input", "", "path to raw canonical input bytes")
	subject := flags.String("subject", "", "caller-supplied qualified subject identity")
	targetPath := flags.String("target-probe", "", "optional local target snapshot JSON; never an authoritative store")
	expectedSource := flags.String("expected-source-sha256", "", "caller-pinned SHA-256 of raw source bytes")
	expectedTarget := flags.String("expected-target-sha256", "", "caller-pinned SHA-256 of target snapshot bytes; required for target preflight")
	population := flags.Uint64("expected-population", 0, "caller-declared expected ID population for target namespace; required for target preflight")
	maxCollision := flags.String("max-collision-probability", "", "caller-declared maximum birthday collision probability; required for target preflight")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 || *schemePath == "" || *inputPath == "" ||
		*subject == "" || strings.TrimSpace(*subject) != *subject {
		fmt.Fprintln(stderr, "scheme, input and nonblank qualified subject are required; no positional arguments")
		return 2
	}
	if (*targetPath != "" && (*expectedSource == "" || *expectedTarget == "")) ||
		(*targetPath == "" && *expectedTarget != "") {
		fmt.Fprintln(stderr, "target preflight requires both source and target SHA-256 pins")
		return 2
	}
	if (*targetPath != "" && (*population < 2 || *maxCollision == "")) ||
		(*targetPath == "" && (*population != 0 || *maxCollision != "")) {
		fmt.Fprintln(stderr, "target preflight requires explicit expected population >= 2 and collision risk maximum")
		return 2
	}
	if err := execute(*schemePath, *inputPath, *subject, *targetPath, *expectedSource, *expectedTarget, *population, *maxCollision, stdout); err != nil {
		fmt.Fprintln(stderr, "reference preview rejected:", err)
		return 1
	}
	return 0
}

func execute(schemePath, inputPath, subject, targetPath, expectedSource, expectedTarget string, population uint64, maxCollision string, stdout io.Writer) error {
	schemeBytes, err := readBounded(schemePath, 4096)
	if err != nil {
		return fmt.Errorf("scheme: %w", err)
	}
	s, err := hashid.ParseSchemeJSON(schemeBytes)
	if err != nil {
		return err
	}
	// The collision budget is checked against the full generator address
	// space before emitting a candidate. It does not replace actual target
	// uniqueness, collation, source identity, or an Owner-approved policy.
	var budget *collisionBudgetEvidence
	if targetPath != "" {
		maximum, err := strconv.ParseFloat(maxCollision, 64)
		if err != nil {
			return fmt.Errorf("invalid caller collision budget: %w", err)
		}
		if err := hashid.CheckCollisionBudget(s, population, maximum); err != nil {
			return fmt.Errorf("caller collision budget: %w", err)
		}
		estimated, err := hashid.CollisionProbability(s, population)
		if err != nil {
			return err
		}
		upper, err := hashid.CollisionRiskUpperBound(s, population)
		if err != nil {
			return err
		}
		budget = &collisionBudgetEvidence{
			ExpectedPopulation:            population,
			MaximumCollisionProbability:   maximum,
			EstimatedCollisionProbability: estimated,
			CollisionRiskUpperBound:       upper,
			PolicySource:                  "caller-declared-unverified",
		}
	}
	input, err := readBounded(inputPath, maxInputBytes)
	if err != nil {
		return fmt.Errorf("input: %w", err)
	}
	if err := checkCallerPin(expectedSource, input); err != nil {
		return fmt.Errorf("source pin: %w", err)
	}
	result, err := hashid.Derive(s, input)
	if err != nil {
		return err
	}
	sourceHash := sha256.Sum256(input)
	out := report{
		Status:          "derived-only",
		Authority:       "none",
		StorageWrite:    false,
		Subject:         subject,
		SchemeJSON:      string(schemeBytes),
		SourceByteCount: len(input),
		SourceSHA256:    hex.EncodeToString(sourceHash[:]),
		HashIDText:      result.Text,
		DigestHex:       hex.EncodeToString(result.Digest),
		CollisionBudget: budget,
	}
	if targetPath != "" {
		targetBytes, err := readBounded(targetPath, maxTargetBytes)
		if err != nil {
			return fmt.Errorf("target probe: %w", err)
		}
		if err := checkCallerPin(expectedTarget, targetBytes); err != nil {
			return fmt.Errorf("target pin: %w", err)
		}
		var probe targetProbeWire
		if err := json.Unmarshal(targetBytes, &probe); err != nil {
			return fmt.Errorf("target probe JSON: %w", err)
		}
		// This file is a local *test snapshot*, not an authenticated store.
		// Require canonical bytes so duplicate/unknown/missing keys and
		// noncanonical Unicode escapes cannot silently change interpretation.
		canonical, err := json.Marshal(probe)
		if err != nil || !bytes.Equal(canonical, targetBytes) ||
			probe.Format != targetWireV1 || probe.Profile.Reserved == nil ||
			probe.Bindings == nil {
			return errors.New("noncanonical or unsupported target probe snapshot")
		}
		profile := hashid.TargetProfile{
			Namespace:       probe.Profile.Namespace,
			ProfileVersion:  probe.Profile.ProfileVersion,
			AllowedAlphabet: probe.Profile.AllowedAlphabet,
			MinLength:       probe.Profile.MinLength,
			MaxLength:       probe.Profile.MaxLength,
			ExactLength:     probe.Profile.ExactLength,
			CaseFoldASCII:   probe.Profile.CaseFoldASCII,
			Reserved:        probe.Profile.Reserved,
		}
		bindings := make([]hashid.TargetBinding, 0, len(probe.Bindings))
		knownSubjects := make(map[string]bool)
		for _, existing := range probe.Bindings {
			if existing.Subject == "" || knownSubjects[existing.Namespace+"\x00"+existing.Subject] {
				return errors.New("target snapshot contains an empty or duplicate qualified subject")
			}
			knownSubjects[existing.Namespace+"\x00"+existing.Subject] = true
			priorScheme, err := hashid.ParseSchemeJSON([]byte(existing.SchemeJSON))
			if err != nil {
				return fmt.Errorf("target snapshot scheme: %w", err)
			}
			bindings = append(bindings, hashid.TargetBinding{
				Namespace:      existing.Namespace,
				ProfileVersion: existing.ProfileVersion,
				Subject:        existing.Subject,
				Scheme:         priorScheme,
				Text:           existing.Text,
			})
		}
		key, err := hashid.CheckTargetBinding(result, subject, profile, bindings)
		if err != nil {
			return fmt.Errorf("target preflight: %w", err)
		}
		out.Status = "caller-pinned-budget-snapshot-preflight-only"
		out.TargetNamespace = profile.Namespace
		out.NormalizedTargetKey = key
		targetHash := sha256.Sum256(targetBytes)
		out.TargetSnapshotSHA256 = hex.EncodeToString(targetHash[:])
		out.CallerPinsMatched = true
		// Detect ordinary replacement between snapshot read and reporting.
		// This is NOT an atomic read transaction or a defense against ABA.
		if err := checkUnchangedFile(targetPath, maxTargetBytes, targetBytes); err != nil {
			return fmt.Errorf("target snapshot changed during preview: %w", err)
		}
	}
	// Reject observed local changes before emitting any report. A read-time
	// file comparison still cannot establish workspace/Source authority, nor
	// prevent symlink retargeting, ABA or a change immediately after return.
	if err := checkUnchangedFile(schemePath, 4096, schemeBytes); err != nil {
		return fmt.Errorf("scheme changed during preview: %w", err)
	}
	if err := checkUnchangedFile(inputPath, maxInputBytes, input); err != nil {
		return fmt.Errorf("source changed during preview: %w", err)
	}
	return json.NewEncoder(stdout).Encode(out)
}

func readBounded(path string, max int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, max+1))
	if err != nil {
		return nil, err
	}
	if len(data) > int(max) {
		return nil, fmt.Errorf("file exceeds %d bytes", max)
	}
	return data, nil
}

func checkCallerPin(expected string, content []byte) error {
	if expected == "" {
		return nil // permitted only in derive-only mode
	}
	if len(expected) != 64 || strings.ToLower(expected) != expected {
		return errors.New("expected lowercase SHA-256 hex (64 characters)")
	}
	decoded, err := hex.DecodeString(expected)
	if err != nil || len(decoded) != sha256.Size {
		return errors.New("malformed SHA-256 pin")
	}
	actual := sha256.Sum256(content)
	if !bytes.Equal(decoded, actual[:]) {
		return errors.New("caller pin does not match current file bytes")
	}
	return nil
}

func checkUnchangedFile(path string, limit int64, first []byte) error {
	next, err := readBounded(path, limit)
	if err != nil {
		return err
	}
	if !bytes.Equal(next, first) {
		return errors.New("input file bytes differ from first read")
	}
	return nil
}
