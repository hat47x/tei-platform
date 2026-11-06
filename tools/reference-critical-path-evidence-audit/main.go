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
	R0                         string `json:"r0"`
	R1                         string `json:"r1"`
	CandidateFingerprint       string `json:"candidateFingerprint"`
	ReviewFingerprint          string `json:"reviewFingerprint"`
	ReviewCandidateFingerprint string `json:"reviewCandidateFingerprint"` // ReviewProjection.candidateFingerprint; identity link, not proof
	// AcceptanceKind is the optional kind of the Host Acceptance behind
	// DecisionReceipt (D9). "provisional" and "unspecified" (a Decision signed
	// before the marker existed) are never formal human Acceptance, and
	// always leave the packet INCOMPLETE. Self-declared, like the Provenance.
	AcceptanceKind  string   `json:"acceptanceKind,omitempty"`
	DecisionReceipt string   `json:"decisionReceipt"`
	ReloadReceipt   string   `json:"reloadReceipt"`
	Delta           []string `json:"delta"`
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

// Guarantee mirrors effectcommit.Evidence Guarantee (Level) and GuaranteeReason.
type Guarantee struct {
	Level  string `json:"level"`
	Reason string `json:"reason"`
}

type Physical struct {
	AcceptedRevision      string `json:"acceptedRevision"`
	Datasource            string `json:"datasource"`
	SchemaCompatibility   string `json:"schemaCompatibility"` // compatible, incompatible, unresolved
	SchemaEvidence        string `json:"schemaEvidence"`
	HostAuthorityEvidence string `json:"hostAuthorityEvidence"`
	// HostAuthorityProvenance is the optional Provenance label of the effect
	// Grant behind HostAuthorityEvidence. "provisional-grant" (D8, development
	// only) is never G2 evidence and always leaves the packet INCOMPLETE.
	HostAuthorityProvenance string `json:"hostAuthorityProvenance,omitempty"`
	// Guarantee is the optional atomicity declaration of the Provider that wrote
	// the effect (effectcommit.Evidence Guarantee / GuaranteeReason), copied from
	// the Evidence a Runner produced. It is a claim, not proof: the tool does not
	// verify it. E2 requires it to be "atomic-with-commit"; a declaration whose
	// level or reason says "provisional" (D10) is never G2 evidence.
	Guarantee              *Guarantee `json:"guarantee,omitempty"`
	CommandReceipt         string     `json:"commandReceipt"`
	IndependentReadReceipt string     `json:"independentReadReceipt"`
	RecordID               string     `json:"recordId"`
	SchemaFailureControl   string     `json:"schemaFailureControl"` // S02
	HostFailureControl     string     `json:"hostFailureControl"`   // S05
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
	// Any claim: a packet that rests on a provisional effect Grant is not
	// qualifying evidence, whatever else it contains. The label is normalized
	// (format/space characters removed, case and "_" folded) so that formatting
	// or an alias cannot hide it. Limit: a look-alike in another script (a
	// homoglyph such as Cyrillic o) is not folded. Only the E2 allow-list below
	// refuses it (as an unknown label); an E0/E1 packet carries no Physical
	// evidence, so such a label is not detected there.
	if normalizedProvenance(p.Physical.HostAuthorityProvenance) == "provisional-grant" {
		add("PROVISIONAL_GRANT_NOT_G2_EVIDENCE")
	} else if p.Claim == "E2" && !allowedHostAuthorityProvenance[p.Physical.HostAuthorityProvenance] {
		// An E2 claim must NAME where its effect Grant came from, using one of
		// the known labels exactly. Omitting the field, an unknown alias or a
		// look-alike must not pass as an ordinary Grant. The label stays a
		// self-declaration of the packet's author.
		add("E2_HOST_AUTHORITY_PROVENANCE_REQUIRED")
	}
	// Any claim: a Provider atomicity declaration that is provisional (D10) is not
	// G2 evidence. Level and reason are normalized like the Provenance so case,
	// space, "_" and zero-width characters cannot hide the word. Limit: as with
	// the Provenance, a look-alike in another script is not folded.
	if g := p.Physical.Guarantee; g != nil &&
		(strings.Contains(normalizedProvenance(g.Level), "provisional") ||
			strings.Contains(normalizedProvenance(g.Reason), "provisional")) {
		add("PROVISIONAL_GUARANTEE_NOT_G2_EVIDENCE")
	}
	// Any claim (D9): an Acceptance recorded as provisional, or whose kind is
	// unspecified, is not a formal human Acceptance. Normalized like the
	// Provenance so formatting or an alias cannot hide it. A packet that omits
	// the field is unchanged (legacy shape). Any other label, including a
	// self-declared "formal", is unrecognized: the Store refuses "formal" too and
	// no path issues a formal Acceptance yet. An E2 claim must NAME the kind
	// (like the Provenance and the Guarantee): omitting it, or leaving only
	// blank/zero-width characters, would otherwise dodge the provisional
	// diagnosis. E0/E1 keep the legacy shape. The kind stays a self-declaration.
	if p.Claim == "E2" && normalizedProvenance(p.Acceptance.AcceptanceKind) == "" {
		add("E2_ACCEPTANCE_KIND_REQUIRED")
	}
	switch kind := normalizedProvenance(p.Acceptance.AcceptanceKind); {
	case kind == "":
		// Omitted or blank-only: the E2 "required" diagnosis above already
		// covers it. Reporting it as unrecognized as well would be a second
		// diagnosis for the same defect.
	case kind == "provisional" || kind == "unspecified":
		add("PROVISIONAL_ACCEPTANCE_NOT_FORMAL_EVIDENCE")
	default:
		add("ACCEPTANCE_KIND_UNRECOGNIZED")
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
		if strings.TrimSpace(w.PositiveReceipt) == "" || strings.TrimSpace(w.NegativeReceipt) == "" ||
			w.PositiveReceipt == w.NegativeReceipt {
			add("ACTUAL_WEB_POSITIVE_AND_NEGATIVE_RECEIPTS_REQUIRED")
		}
		// N11 must be a separately located no-read control, not a recycled
		// success/denial receipt or the unrelated Source/Target provenance.
		// Locator separation remains a claim, not inspection of receipt bytes.
		missingN11 := strings.TrimSpace(w.UnobservedControlReceipt) == "" ||
			w.UnobservedControlReceipt == w.PositiveReceipt ||
			w.UnobservedControlReceipt == w.NegativeReceipt ||
			w.UnobservedControlReceipt == w.WorkspaceRevisionReceipt ||
			w.UnobservedControlReceipt == w.TargetLaunchReceipt ||
			w.UnobservedControlReceipt == a.DecisionReceipt ||
			w.UnobservedControlReceipt == a.ReloadReceipt ||
			w.UnobservedControlReceipt == p.Qualification.Receipt
		if p.Claim == "E2" {
			missingN11 = missingN11 ||
				w.UnobservedControlReceipt == p.Physical.CommandReceipt ||
				w.UnobservedControlReceipt == p.Physical.IndependentReadReceipt
		}
		if missingN11 {
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
		// Only a declared synthetic Provider gets the synthetic-specific
		// diagnosis. An empty or unknown kind is already reported as
		// WEB_PROVIDER_KIND_UNRESOLVED above; labeling it "synthetic" as well
		// would be false and a second report of the same defect.
		if w.ProviderKind == "synthetic" {
			add("SYNTHETIC_WEB_PROVIDER_CANNOT_CLAIM_E2")
		}
		if ph.AcceptedRevision != a.R1 || ph.Datasource == "" ||
			ph.SchemaCompatibility != "compatible" || ph.SchemaEvidence == "" {
			add("S02_REAL_DATASOURCE_SCHEMA_READINESS_MISSING")
		}
		if ph.HostAuthorityEvidence == "" {
			add("S05_CURRENT_HOST_AUTHORITY_MISSING")
		}
		if strings.TrimSpace(ph.CommandReceipt) == "" ||
			strings.TrimSpace(ph.IndependentReadReceipt) == "" ||
			ph.CommandReceipt == ph.IndependentReadReceipt || ph.RecordID == "" {
			add("E2_COMMAND_AND_INDEPENDENT_READ_NOT_CORRELATED")
		}
		// The Provider must DECLARE atomic-with-commit (exact label). A missing
		// declaration (e.g. a path that emits no Evidence), "undeclared",
		// "not-atomic" or an unknown value is incomplete. Declaring it only
		// makes the claim visible; the audit does not verify it.
		if ph.Guarantee == nil || ph.Guarantee.Level != "atomic-with-commit" {
			add("E2_PROVIDER_GUARANTEE_NOT_ATOMIC")
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

// allowedHostAuthorityProvenance are the effectcommit.Provenance values an E2
// packet may declare for its effect Grant. "provisional-grant" is deliberately
// absent (it is diagnosed separately), as are "unspecified" and every new value:
// adding one is a deliberate change to this list.
var allowedHostAuthorityProvenance = map[string]bool{"real-pg": true, "synthetic": true}

// normalizedProvenance lower-cases a label, drops control and format characters
// (zero-width, bidi), folds "_" and whitespace to "-" and trims "-" at the ends.
func normalizedProvenance(s string) string {
	var out strings.Builder
	for _, r := range s {
		switch {
		case unicode.IsControl(r) && !unicode.IsSpace(r), unicode.Is(unicode.Cf, r):
			continue
		case r == '_', unicode.IsSpace(r):
			out.WriteRune('-')
		default:
			out.WriteRune(unicode.ToLower(r))
		}
	}
	return strings.Trim(out.String(), "-")
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
