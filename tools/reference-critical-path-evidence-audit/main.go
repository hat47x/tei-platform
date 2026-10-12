// Command reference-critical-path-evidence-audit checks the completeness of
// cross-lane evidence packets. It cannot grant a TEI Product Gate or verify
// a referenced artifact's authenticity.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

const schema = "tei.critical-path-evidence/v0"

var sha40 = regexp.MustCompile(`^[0-9a-f]{40}$`)
var revision = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

type Contract struct {
	ProducerPath  string `json:"producerPath"`
	ConsumerPath  string `json:"consumerPath"`
	PositiveTest  string `json:"positiveTest"`
	NegativeTest  string `json:"negativeTest"`
	EvidenceClass string `json:"evidenceClass"` // source-only, synthetic-tested, actual-observed
}

type Qualification struct {
	Head        string `json:"head"`
	Main        string `json:"main"`
	FullHealth  string `json:"fullHealth"` // PASS, FAIL, NOT_RUN
	CleanBefore bool   `json:"cleanBefore"`
	CleanAfter  bool   `json:"cleanAfter"`
	Receipt     string `json:"receipt"`
}

type Acceptance struct {
	R0                         string   `json:"r0"`
	R1                         string   `json:"r1"`
	CandidateFingerprint       string   `json:"candidateFingerprint"`
	ReviewFingerprint          string   `json:"reviewFingerprint"`
	ReviewCandidateFingerprint string   `json:"reviewCandidateFingerprint"` // ReviewProjection.candidateFingerprint; identity link, not proof
	DecisionReceipt            string   `json:"decisionReceipt"`
	ReloadReceipt              string   `json:"reloadReceipt"`
	Delta                      []string `json:"delta"`
}

type Web struct {
	AcceptedRevision         string `json:"acceptedRevision"`
	Application              string `json:"application"`
	Target                   string `json:"target"`
	SelectedValidation       string `json:"selectedValidation"`
	NegativeValidation       string `json:"negativeValidation"`       // asserted failing rule, not independently verified
	NegativeEvidenceMode     string `json:"negativeEvidenceMode"`     // running-server, handler-only
	ProviderKind             string `json:"providerKind"`             // synthetic, actual-db
	RunMode                  string `json:"runMode"`                  // running-server, handler-only
	WorkspaceRevisionReceipt string `json:"workspaceRevisionReceipt"` // independent revision observation of Web workspace
	TargetLaunchReceipt      string `json:"targetLaunchReceipt"`      // workspace, target build, start/stop, no credentials
	PositiveReceipt          string `json:"positiveReceipt"`
	NegativeReceipt          string `json:"negativeReceipt"`
	UnobservedControlReceipt string `json:"unobservedControlReceipt"` // N11
}

type Physical struct {
	AcceptedRevision       string `json:"acceptedRevision"`
	Datasource             string `json:"datasource"`
	SchemaCompatibility    string `json:"schemaCompatibility"` // compatible, incompatible, unresolved
	SchemaEvidence         string `json:"schemaEvidence"`
	HostAuthorityEvidence  string `json:"hostAuthorityEvidence"`
	CommandReceipt         string `json:"commandReceipt"`
	IndependentReadReceipt string `json:"independentReadReceipt"`
	RecordID               string `json:"recordId"`
	SchemaFailureControl   string `json:"schemaFailureControl"` // S02
	HostFailureControl     string `json:"hostFailureControl"`   // S05
}

type Packet struct {
	Schema        string        `json:"schema"`
	Claim         string        `json:"claim"` // E0, E1, E2: evidence tier requested, not a Gate PASS
	Contract      Contract      `json:"contract"`
	Qualification Qualification `json:"qualification"`
	Acceptance    Acceptance    `json:"acceptance"`
	Web           Web           `json:"web"`
	Physical      Physical      `json:"physical"`
}

type Report struct {
	Schema         string   `json:"schema"`
	Requested      string   `json:"requested"`
	Status         string   `json:"status"`      // INCOMPLETE or REVIEW_REQUIRED; never PASS
	ProductGate    string   `json:"productGate"` // UNQUALIFIED; caller must independently verify
	Diagnostics    []string `json:"diagnostics"`
	AuthorityLimit string   `json:"authorityLimit"`
}

// Audit checks declarations and cross-stage consistency only. Evidence locators
// are not dereferenced or trusted; neither this tool nor its exit code approves
// a merge, deployment, user acceptance, or external database mutation.
func Audit(p Packet, expectedHead, expectedMain string) Report {
	r := Report{
		Schema: schema, Requested: p.Claim, Status: "INCOMPLETE", ProductGate: "UNQUALIFIED",
		Diagnostics: []string{}, AuthorityLimit: "caller-supplied locators and assertions; review actual bytes, runtime, permission, and exact trusted checkout independently",
	}
	add := func(code string) { r.Diagnostics = append(r.Diagnostics, code) }
	if p.Schema != schema {
		add("UNSUPPORTED_PACKET_SCHEMA")
	}
	if p.Contract.ProducerPath == "" || p.Contract.ConsumerPath == "" ||
		p.Contract.PositiveTest == "" || p.Contract.NegativeTest == "" ||
		(p.Contract.EvidenceClass != "source-only" && p.Contract.EvidenceClass != "synthetic-tested" && p.Contract.EvidenceClass != "actual-observed") {
		add("E0_CONTRACT_OR_POSITIVE_NEGATIVE_TEST_MISSING")
	}
	if p.Claim != "E0" && p.Claim != "E1" && p.Claim != "E2" {
		add("UNSUPPORTED_EVIDENCE_TIER")
	}
	if !sha40.MatchString(expectedHead) || !sha40.MatchString(expectedMain) {
		add("EXPECTED_HEAD_OR_MAIN_REQUIRED")
	}
	if !sha40.MatchString(p.Qualification.Head) || !sha40.MatchString(p.Qualification.Main) ||
		p.Qualification.Head != expectedHead || p.Qualification.Main != expectedMain {
		add("REVISION_QUALIFICATION_STALE_OR_MISSING")
	}
	if p.Qualification.FullHealth != "PASS" || !p.Qualification.CleanBefore ||
		!p.Qualification.CleanAfter || strings.TrimSpace(p.Qualification.Receipt) == "" {
		add("TRUSTED_FULL_HEALTH_NOT_RECORDED")
	}
	if p.Claim == "E1" || p.Claim == "E2" {
		if p.Contract.EvidenceClass != "actual-observed" {
			add("E1_E2_ACTUAL_EXECUTION_CLASS_REQUIRED")
		}
		a, w := p.Acceptance, p.Web
		if !revision.MatchString(a.R0) || !revision.MatchString(a.R1) || a.R0 == a.R1 {
			add("ACCEPTED_R0_R1_LINEAGE_MISSING")
		}
		if !revision.MatchString(a.CandidateFingerprint) ||
			!revision.MatchString(a.ReviewFingerprint) ||
			strings.TrimSpace(a.DecisionReceipt) == "" || strings.TrimSpace(a.ReloadReceipt) == "" {
			add("ACCEPTANCE_REVIEW_AND_RELOAD_EVIDENCE_MISSING")
		}
		if !exactSet(a.Delta, "add-validation", "add-application-validation-use") {
			add("VALIDATION_AND_APPLICATION_USE_NOT_ATOMIC")
		}
		if w.AcceptedRevision != a.R1 || w.Application != "order_entry" ||
			w.Target != "web" || w.SelectedValidation != "sales.order_line.unit_price_positive" {
			add("WEB_NOT_BOUND_TO_ACCEPTED_R1_RULE_SELECTION")
		}
		if w.ProviderKind != "synthetic" && w.ProviderKind != "actual-db" {
			add("WEB_PROVIDER_KIND_UNRESOLVED")
		}
		if w.RunMode != "running-server" {
			add("E1_RUNNING_WEB_TARGET_RECEIPT_REQUIRED")
		}
		// The old B/R/T Web envelope exposes repository baseline and Target
		// build identities, but not the managed-source R1 actually loaded.
		// An E1 reporter must supply independent provenance for both the
		// workspace revision and the exact launched Target instance.
		if strings.TrimSpace(w.WorkspaceRevisionReceipt) == "" ||
			strings.TrimSpace(w.TargetLaunchReceipt) == "" ||
			w.WorkspaceRevisionReceipt == w.TargetLaunchReceipt ||
			w.WorkspaceRevisionReceipt == w.PositiveReceipt ||
			w.WorkspaceRevisionReceipt == w.NegativeReceipt ||
			w.TargetLaunchReceipt == w.PositiveReceipt ||
			w.TargetLaunchReceipt == w.NegativeReceipt {
			add("E1_WEB_WORKSPACE_R1_AND_TARGET_LAUNCH_PROVENANCE_MISSING")
		}
		if w.PositiveReceipt == "" || w.NegativeReceipt == "" ||
			w.PositiveReceipt == w.NegativeReceipt {
			add("ACTUAL_WEB_POSITIVE_AND_NEGATIVE_RECEIPTS_REQUIRED")
		}
		if w.UnobservedControlReceipt == "" || w.UnobservedControlReceipt == w.PositiveReceipt ||
			w.UnobservedControlReceipt == w.NegativeReceipt {
			add("N11_UNOBSERVED_CONTROL_MISSING")
		}
		// Shared Acceptance fingerprints the candidate's operation delta and
		// the entire Review Projection separately. Those digests need NOT match.
		// The Review Projection instead embeds the candidate fingerprint.
		// This is only a caller-reported join, not verification of the receipt.
		if !revision.MatchString(a.ReviewCandidateFingerprint) ||
			a.ReviewCandidateFingerprint != a.CandidateFingerprint {
			add("ACCEPTANCE_REVIEW_CANDIDATE_BINDING_MISSING_OR_MISMATCH")
		}
		// The existing synthetic normal fixture rejects an invalid quantity, not
		// the newly selected unit-price rule. A handler-only negative test is
		// valuable E0 evidence but cannot prove a running Web Target refused it.
		if w.NegativeValidation != w.SelectedValidation ||
			w.NegativeEvidenceMode != "running-server" {
			add("E1_NEGATIVE_SELECTED_RULE_RUNNING_WEB_EVIDENCE_REQUIRED")
		}
	}
	if p.Claim == "E2" {
		a, w, ph := p.Acceptance, p.Web, p.Physical
		if w.ProviderKind != "actual-db" {
			add("SYNTHETIC_WEB_PROVIDER_CANNOT_CLAIM_E2")
		}
		if ph.AcceptedRevision != a.R1 || ph.Datasource == "" ||
			ph.SchemaCompatibility != "compatible" || ph.SchemaEvidence == "" {
			add("S02_REAL_DATASOURCE_SCHEMA_READINESS_MISSING")
		}
		if ph.HostAuthorityEvidence == "" {
			add("S05_CURRENT_HOST_AUTHORITY_MISSING")
		}
		if ph.CommandReceipt == "" || ph.IndependentReadReceipt == "" ||
			ph.CommandReceipt == ph.IndependentReadReceipt || ph.RecordID == "" {
			add("E2_COMMAND_AND_INDEPENDENT_READ_NOT_CORRELATED")
		}
		if ph.SchemaFailureControl == "" {
			add("S02_NEGATIVE_CONTROL_MISSING")
		}
		if ph.HostFailureControl == "" {
			add("S05_NEGATIVE_CONTROL_MISSING")
		}
	}
	sort.Strings(r.Diagnostics)
	if len(r.Diagnostics) == 0 {
		r.Status = "REVIEW_REQUIRED"
	}
	return r
}

func exactSet(actual []string, expected ...string) bool {
	if len(actual) != len(expected) {
		return false
	}
	want := make(map[string]int, len(expected))
	for _, s := range expected {
		want[s]++
	}
	for _, s := range actual {
		want[s]--
	}
	for _, count := range want {
		if count != 0 {
			return false
		}
	}
	return true
}

// foldedJSONKey uses Unicode simple folding (the equivalence used by
// strings.EqualFold) to produce a deterministic canonical key. Comparing every
// previously observed key would be quadratic for attacker-supplied JSON with
// thousands of distinct fields; a lookup per folded key remains linear.
func foldedJSONKey(key string) string {
	var out strings.Builder
	out.Grow(len(key))
	for _, r := range key {
		minimum := r
		for candidate := unicode.SimpleFold(r); candidate != r; candidate = unicode.SimpleFold(candidate) {
			if candidate < minimum {
				minimum = candidate
			}
		}
		out.WriteRune(minimum)
	}
	return out.String()
}

// Strict token scan rejects duplicate object keys, including nested keys. The
// normal decoder otherwise silently uses the last duplicate value, allowing
// caller-supplied Evidence fields to shadow an earlier contradictory claim.
// A depth bound also rejects pathological nesting before any typed decoding.
func rejectDuplicateJSONKeys(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	var walk func(int) error
	walk = func(depth int) error {
		if depth > 64 {
			return errors.New("packet JSON nesting exceeds 64")
		}
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		switch tok {
		case json.Delim('{'):
			keys := make(map[string]struct{})
			for dec.More() {
				keyToken, err := dec.Token()
				if err != nil {
					return err
				}
				key, ok := keyToken.(string)
				if !ok {
					return errors.New("packet has non-string JSON object key")
				}
				// Match encoding/json's case-insensitive struct-field behavior
				// without a quadratic scan of prior keys. Values remain untrusted.
				folded := foldedJSONKey(key)
				if _, duplicate := keys[folded]; duplicate {
					return fmt.Errorf("duplicate JSON object key %q (case-folded alias)", key)
				}
				keys[folded] = struct{}{}
				if err := walk(depth + 1); err != nil {
					return err
				}
			}
			_, err = dec.Token() // closing }
			return err
		case json.Delim('['):
			for dec.More() {
				if err := walk(depth + 1); err != nil {
					return err
				}
			}
			_, err = dec.Token() // closing ]
			return err
		case json.Delim('}'), json.Delim(']'):
			return errors.New("unexpected closing JSON delimiter")
		default:
			return nil
		}
	}
	if err := walk(0); err != nil {
		return err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("packet has trailing JSON")
		}
		return err
	}
	return nil
}

func run(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("critical-path-evidence-audit", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("packet", "", "evidence packet JSON")
	head := fs.String("head", "", "exact candidate HEAD SHA")
	mainSHA := fs.String("main", "", "exact reviewed main SHA")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 || *path == "" {
		return errors.New("-packet, -head, -main are required")
	}
	f, err := os.Open(*path)
	if err != nil {
		return err
	}
	defer f.Close()
	// Reject over-limit inputs rather than treating the limit as an artificial
	// EOF: a valid first MiB followed by hidden trailing bytes must not pass.
	const maxPacketBytes = 1 << 20
	data, err := io.ReadAll(io.LimitReader(f, maxPacketBytes+1))
	if err != nil {
		return err
	}
	if len(data) > maxPacketBytes {
		return errors.New("packet exceeds 1 MiB")
	}
	if !utf8.Valid(data) {
		return errors.New("packet is not valid UTF-8")
	}
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var p Packet
	if err := dec.Decode(&p); err != nil {
		return fmt.Errorf("packet: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("packet has trailing JSON")
		}
		return fmt.Errorf("trailing JSON: %w", err)
	}
	report := Audit(p, *head, *mainSHA)
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		return err
	}
	if report.Status != "REVIEW_REQUIRED" {
		return errors.New("evidence packet incomplete")
	}
	return nil
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
