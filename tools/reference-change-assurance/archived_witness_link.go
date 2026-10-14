package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"tei/internal/modelref"
	"tei/internal/sourceindex"
)

// These are the existing eight bounded, synthetic-only #1163 E1 smoke
// artifacts. This tool does not create another Evidence Packet schema or
// interpret the official #1119 preflight's Product Gate.
var witnessNatures = map[string]string{
	"workspace-r1.json":            "test-only-managed-source-lineage",
	"target-launch.json":           "running-first-web-target-with-synthetic-provider",
	"web-positive.json":            "running-web-http-positive-synthetic-provider",
	"target-negative-launch.json":  "test-only-Go-overlay-Web-target",
	"web-negative-unit-price.json": "running-web-http-negative-synthetic-fixture-overlay",
	"target-n11-launch.json":       "test-only-Go-overlay-Web-target-N11",
	"n11-unobserved-control.json":  "running-web-http-synthetic-Observation-read-failure",
	"stale-source-control.json":    "running-web-http-stale-source-R0-label-spoof-control",
}

var gitRevision40 = regexp.MustCompile("^[0-9a-f]{40}$")

type WitnessFile struct {
	Name        string `json:"name"`
	Fingerprint string `json:"fingerprint"`
}

// ArchivedWitnessLinkReport only checks the *claims inside* pre-existing
// archived receipts against independently indexed before/after Workspace
// Source bytes. It is never an attestation of execution or receipt origin.
type ArchivedWitnessLinkReport struct {
	SchemaVersion    string              `json:"schemaVersion"`
	Authority        string              `json:"authority"`
	Diff             WorkspaceDiffReport `json:"diff"`
	IntegrationHead  string              `json:"integrationHead"`
	Files            []WitnessFile       `json:"files"`
	LinkStatus       string              `json:"linkStatus"`
	Diagnostics      []string            `json:"diagnostics"`
	Qualification    string              `json:"qualification"`
	ExecutionAllowed bool                `json:"executionAllowed"`
	AuthorityLimits  []string            `json:"authorityLimits"`
}

// Only small, regular, rooted files may be considered. A Source or receipt
// symlink outside the receipt root cannot be followed through os.Root.
func readArchivedWitness(root *os.Root, name string) ([]byte, error) {
	f, err := root.Open(filepath.FromSlash(name))
	if err != nil {
		return nil, fmt.Errorf("open archived witness %s: %w", name, err)
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() || st.Size() > 1<<20 {
		return nil, fmt.Errorf("archived witness %s is not a bounded regular file", name)
	}
	data, err := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > 1<<20 {
		return nil, fmt.Errorf("archived witness %s exceeds 1 MiB", name)
	}
	return data, nil
}

// Top-level receipt fields are intentionally sparse: #1163 retains ownership
// of its receipt bodies. Reject duplicate (including case-folded) top-level
// fields rather than letting a later JSON value shadow an earlier claim.
func decodeArchivedWitness(data []byte) (map[string]json.RawMessage, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil || tok != json.Delim('{') {
		return nil, fmt.Errorf("archived witness must be one JSON object")
	}
	result := make(map[string]json.RawMessage)
	seen := make(map[string]bool)
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := keyTok.(string)
		if !ok {
			return nil, fmt.Errorf("non-string archived witness key")
		}
		// Only a bounded set of top-level receipt claims is expected.
		// Avoid an unbounded quadratic case-fold scan on untrusted JSON.
		if len(seen) >= 128 {
			return nil, fmt.Errorf("archived witness has too many top-level fields")
		}
		for prev := range seen {
			if strings.EqualFold(prev, key) {
				return nil, fmt.Errorf("duplicate archived witness key %q", key)
			}
		}
		seen[key] = true
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			return nil, err
		}
		result[key] = value
	}
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("archived witness has trailing content")
	}
	return result, nil
}

func claimedString(w map[string]json.RawMessage, key string) string {
	var value string
	if err := json.Unmarshal(w[key], &value); err != nil {
		return ""
	}
	return value
}

func claimedBool(w map[string]json.RawMessage, key string, expected bool) bool {
	var value bool
	return len(w[key]) > 0 && json.Unmarshal(w[key], &value) == nil && value == expected
}

// reviewArchivedWitnessLink is a diagnostics-only connection to the eight
// *existing* #1163 test-harness files, not an E1/E2 preflight replacement.
// A consistent claim can only become "declared-links-consistent"; the
// qualification remains unresolved for both matching and mismatching files.
func reviewArchivedWitnessLink(beforeRoot, afterRoot, scope string, consumer modelref.Key, evidenceDir string) (ArchivedWitnessLinkReport, error) {
	if evidenceDir == "" || consumer.Subject.Kind != "application" {
		return ArchivedWitnessLinkReport{}, fmt.Errorf("archived witness requires a directory and application consumer")
	}
	diff, err := reviewWorkspaceDiff(beforeRoot, afterRoot, scope, consumer)
	if err != nil {
		return ArchivedWitnessLinkReport{}, err
	}
	root, err := os.OpenRoot(evidenceDir)
	if err != nil {
		return ArchivedWitnessLinkReport{}, fmt.Errorf("open archived witness directory: %w", err)
	}
	defer root.Close()
	names := make([]string, 0, len(witnessNatures))
	for name := range witnessNatures {
		names = append(names, name)
	}
	sort.Strings(names)
	claims := make(map[string]map[string]json.RawMessage)
	report := ArchivedWitnessLinkReport{
		SchemaVersion:    "tei.reference.archived-witness-link/v0",
		Authority:        "read-only-existing-synthetic-1163-witness-claims",
		Diff:             diff,
		Files:            []WitnessFile{},
		Diagnostics:      []string{},
		LinkStatus:       "unresolved",
		Qualification:    "unresolved",
		ExecutionAllowed: false,
		AuthorityLimits: []string{
			"archived-receipts-are-test-harness-claims-without-authenticated-provenance",
			"no-independent-database-read-or-current-runtime-authority",
			"scenario-1180-is-not-the-same-execution-as-the-1163-web-smoke",
			"no-product-gate-or-evidence-packet-preflight-approval",
		},
	}
	add := func(code string) { report.Diagnostics = append(report.Diagnostics, code) }
	for _, name := range names {
		body, err := readArchivedWitness(root, name)
		if err != nil {
			return ArchivedWitnessLinkReport{}, err
		}
		claim, err := decodeArchivedWitness(body)
		if err != nil {
			return ArchivedWitnessLinkReport{}, fmt.Errorf("%s: %w", name, err)
		}
		claims[name] = claim
		report.Files = append(report.Files, WitnessFile{Name: name, Fingerprint: sourceindex.Fingerprint(body)})
		if claimedString(claim, "evidenceNature") != witnessNatures[name] {
			add("UNEXPECTED_WITNESS_NATURE:" + name)
		}
	}
	lineage := claims["workspace-r1.json"]
	launch := claims["target-launch.json"]
	positive := claims["web-positive.json"]
	negative := claims["web-negative-unit-price.json"]
	n11 := claims["n11-unobserved-control.json"]
	stale := claims["stale-source-control.json"]
	head := claimedString(lineage, "integrationHead")
	report.IntegrationHead = head
	if !gitRevision40.MatchString(head) {
		add("INTEGRATION_HEAD_MISSING_OR_INVALID")
	}
	for _, name := range names {
		if claimedString(claims[name], "integrationHead") != head || head == "" {
			add("INTEGRATION_HEAD_DIFFERS:" + name)
		}
	}
	before, after := diff.BeforeSourceRevision, diff.AfterSourceRevision
	if before == after || len(diff.SourceChanges) == 0 {
		add("NO_OBSERVED_R0_R1_SOURCE_CHANGE")
	}
	if claimedString(lineage, "sourceBefore") != before ||
		claimedString(lineage, "sourceAfter") != after ||
		claimedString(lineage, "decisionModelRevisionAfter") != after {
		add("LINEAGE_SOURCE_REVISION_MISMATCH")
	}
	if claimedString(launch, "workspaceSourceRevisionBeforeLaunch") != after ||
		claimedString(launch, "application") != consumer.Subject.ID ||
		claimedString(launch, "targetBuildRevision") != head {
		add("TARGET_LAUNCH_SOURCE_OR_BUILD_MISMATCH")
	}
	if claimedString(positive, "acceptedSourceRevisionClaim") != after ||
		claimedString(negative, "acceptedSourceRevisionClaim") != after ||
		claimedString(n11, "acceptedSourceRevisionClaim") != after {
		add("WEB_AND_OBSERVATION_R1_CLAIMS_MISMATCH")
	}
	if claimedString(stale, "acceptedSourceRevisionClaim") != after ||
		claimedString(stale, "actuallyLoadedWorkspaceSourceRevision") != before ||
		!claimedBool(stale, "mismatchDetectedByExternalTestHarness", true) ||
		!claimedBool(stale, "webServerRejectsMismatchedLabels", false) {
		add("STALE_SOURCE_CONTROL_NOT_SEPARATELY_OBSERVED")
	}
	if claimedString(n11, "executionErrorNature") != "synthetic-read-unavailable" ||
		!claimedBool(n11, "commandCompleted", true) ||
		!claimedBool(n11, "observationReadAttempted", true) ||
		!claimedBool(n11, "observationResultRecorded", false) {
		add("N11_UNOBSERVED_CONTROL_INCOMPLETE")
	}
	if claimedString(positive, "observationNature") == "" ||
		!claimedBool(positive, "independentDatabaseRead", false) ||
		!claimedBool(n11, "independentDatabaseRead", false) ||
		!claimedBool(launch, "runtimeAuthorityVerified", false) ||
		!claimedBool(lineage, "humanApproval", false) {
		add("SYNTHETIC_OR_AUTHORITY_LIMITS_NOT_EXPLICIT")
	}
	// A typed impact that was not fully resolvable remains so. Matching
	// receipt *claims* cannot override this separate change review.
	if !diff.Impact.GraphValid || diff.Decision.Action == "recheck" {
		add("SOURCE_IMPACT_REQUIRES_SEPARATE_REVIEW")
	}
	sort.Strings(report.Diagnostics)
	if len(report.Diagnostics) == 0 {
		report.LinkStatus = "declared-links-consistent"
	}
	return report, nil
}
