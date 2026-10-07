package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"tei/internal/driver"
	"tei/internal/workspace"
)

// The second realistic Reference (make-to-order production and stocking) is a TEI-original
// seed whose provenance, epistemic ledger and Handoff are checked here against RRB-P1 / RRB-P2
// of LANE_B_SECOND_REALISTIC_REFERENCE_ACCEPTANCE_CONTRACT_2026-09.md, using the same loaders,
// normalized model, D4 derivation and standalone Handoff source form as the first Reference.

const (
	seedLedgerSchemaVersion = "business-seed-ledger-v0-pressure"
	seedLedgerArtifactRole  = "reference-seed-ledger-pressure"
)

type seedLedger struct {
	SchemaVersion        string            `json:"schemaVersion"`
	ArtifactRole         string            `json:"artifactRole"`
	Canonical            bool              `json:"canonical"`
	Seed                 string            `json:"seed"`
	IndependentNeed      string            `json:"independentNeed"`
	Scope                seedScope         `json:"scope"`
	Sources              []seedSource      `json:"sources"`
	CopiedArtifacts      []string          `json:"copiedArtifacts"`
	LicenseAnalogyMade   bool              `json:"licenseAnalogyMade"`
	SourceBackedFacts    []seedFact        `json:"sourceBackedFacts"`
	TEIOriginalDecisions []seedDecision    `json:"teiOriginalDecisions"`
	Derived              []json.RawMessage `json:"derived"`
	Proposals            []json.RawMessage `json:"proposals"`
	Unresolved           []seedUnresolved  `json:"unresolved"`
	CrossViewQuestions   []seedQuestion    `json:"crossViewQuestions"`
	Handoff              seedHandoffRecord `json:"handoff"`
	Review               seedReviewRecord  `json:"review"`
}

type seedScope struct {
	In  []string `json:"in"`
	Out []string `json:"out"`
}

type seedSource struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	URL         string `json:"url,omitempty"`
	Publication string `json:"publication,omitempty"`
	Use         string `json:"use"`
	Copied      bool   `json:"copied"`
	LicenseNote string `json:"licenseNote,omitempty"`
}

type seedFact struct {
	ID          string   `json:"id"`
	Text        string   `json:"text"`
	Sources     []string `json:"sources"`
	ConfirmedBy string   `json:"confirmedBy"`
}

type seedDecision struct {
	ID   string   `json:"id"`
	Text string   `json:"text"`
	Refs []string `json:"refs"`
}

type seedUnresolved struct {
	ID      string `json:"id"`
	Subject string `json:"subject"`
	Status  string `json:"status"`
	Note    string `json:"note"`
}

type seedQuestion struct {
	ID                     string   `json:"id"`
	Question               string   `json:"question"`
	Views                  []string `json:"views"`
	DataIdentity           string   `json:"dataIdentity,omitempty"`
	ApplicationRealization string   `json:"applicationRealization,omitempty"`
}

type seedHandoffRecord struct {
	Candidate       seedHandoffCandidate `json:"candidate"`
	ArisesNaturally bool                 `json:"arisesNaturally"`
	Basis           string               `json:"basis"`
	NotDerivedFrom  []string             `json:"notDerivedFrom"`
	ConditionIsNot  []string             `json:"conditionIsNot"`
}

type seedHandoffCandidate struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type seedReviewRecord struct {
	Checklist         string `json:"checklist"`
	Author            string `json:"author"`
	IndependentReview string `json:"independentReview"`
}

func secondCorpusRoot() string {
	return filepath.Join(repoPath(), "reference", "business-system-design", "second-corpus")
}

func loadSecondCorpus(t *testing.T) (referenceSet, map[string]fixture) {
	t.Helper()
	set, err := readReferenceSet(filepath.Join(secondCorpusRoot(), "production-and-stocking-reference-set.json"))
	if err != nil {
		t.Fatal(err)
	}
	fixtures, err := resolveReferenceSet(secondCorpusRoot(), set)
	if err != nil {
		t.Fatal(err)
	}
	return set, fixtures
}

func loadSeedLedger(t *testing.T) seedLedger {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(secondCorpusRoot(), "seed-ledger.json"))
	if err != nil {
		t.Fatal(err)
	}
	ledger, err := decodeSeedLedger(content)
	if err != nil {
		t.Fatal(err)
	}
	return ledger
}

func decodeSeedLedger(content []byte) (seedLedger, error) {
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	var ledger seedLedger
	if err := decoder.Decode(&ledger); err != nil {
		return seedLedger{}, err
	}
	if err := requireEOF(decoder); err != nil {
		return seedLedger{}, err
	}
	if ledger.SchemaVersion != seedLedgerSchemaVersion || ledger.ArtifactRole != seedLedgerArtifactRole {
		return seedLedger{}, fmt.Errorf("unsupported seed ledger %q / %q", ledger.SchemaVersion, ledger.ArtifactRole)
	}
	return ledger, nil
}

func stepByID(f fixture, id string) (step, bool) {
	for _, s := range f.Steps {
		if s.ID == id {
			return s, true
		}
	}
	return step{}, false
}

// resolveLedgerRef resolves "procedure" or "procedure/step" against the loaded fixtures.
func resolveLedgerRef(fixtures map[string]fixture, ref string) bool {
	procedure, stepID, hasStep := strings.Cut(ref, "/")
	f, ok := fixtures[procedure]
	if !ok {
		return false
	}
	if !hasStep {
		return true
	}
	_, found := stepByID(f, stepID)
	return found
}

var secondCorpusAllowedRealizations = map[string]bool{
	"manual-unresolved":          true,
	"manual-external-unresolved": true,
	"observation-boundary":       true,
}

// secondCorpusFakeRealizationViolations lists every place where the fixture would make the Reference
// look executable: an Application context, or a step that names an Application Pattern, Data,
// Relation, Validation or Capability, or a realization that promises more than the seed evidences.
func secondCorpusFakeRealizationViolations(f fixture) []string {
	var out []string
	if f.Application != "" {
		out = append(out, "application context "+f.Application)
	}
	for _, s := range f.Steps {
		if len(s.ApplicationPatterns) > 0 || len(s.Data) > 0 || len(s.Relations) > 0 || len(s.Validations) > 0 || len(s.Capabilities) > 0 {
			out = append(out, "step "+s.ID+" carries an Application / Semantic reference")
		}
		if !secondCorpusAllowedRealizations[s.Realization] {
			out = append(out, "step "+s.ID+" has realization "+s.Realization)
		}
	}
	return out
}

// secondCorpusActualityViolations checks that instruction, physical work, physical confirmation and the
// completion record stay four distinct steps and that no step both confirms and records.
func secondCorpusActualityViolations(f fixture, instruction, physical, confirmation, record string) []string {
	var out []string
	ids := []string{instruction, physical, confirmation, record}
	seen := map[string]bool{}
	steps := make([]step, 0, len(ids))
	for _, id := range ids {
		s, ok := stepByID(f, id)
		if !ok {
			return append(out, "step "+id+" does not exist")
		}
		if seen[id] {
			out = append(out, "step "+id+" is used for two roles")
		}
		seen[id] = true
		steps = append(steps, s)
	}
	if steps[0].Realization != "manual-unresolved" || steps[3].Realization != "manual-unresolved" {
		out = append(out, "instruction / record must not be realized as physical work or observation")
	}
	if steps[1].Realization != "manual-external-unresolved" {
		out = append(out, "physical work must be manual / external")
	}
	if steps[2].Realization != "observation-boundary" {
		out = append(out, "physical confirmation must be an observation boundary")
	}
	if steps[2].Realization == steps[3].Realization {
		out = append(out, "confirmation and record share a realization")
	}
	for _, s := range f.Steps {
		if hasAnyWord(s.BusinessAction, "confirm", "confirms", "establish", "check") && hasAnyWord(s.BusinessAction, "record", "records", "enter") {
			out = append(out, "step "+s.ID+" both confirms and records")
		}
	}
	return out
}

// hasAnyWord reports whether the text contains one of the words as a whole word (so that the noun
// "confirmation" is not mistaken for the act of confirming).
func hasAnyWord(text string, words ...string) bool {
	fields := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return r < 'a' || r > 'z' })
	for _, field := range fields {
		for _, word := range words {
			if field == word {
				return true
			}
		}
	}
	return false
}

func TestSecondRealisticReferenceLedgerSeparatesFactsDecisionsAndUnresolved(t *testing.T) {
	set, fixtures := loadSecondCorpus(t)
	ledger := loadSeedLedger(t)

	if ledger.Canonical {
		t.Fatal("seed ledger claims Canonical authority")
	}
	if strings.TrimSpace(ledger.IndependentNeed) == "" || len(ledger.Scope.In) == 0 || len(ledger.Scope.Out) == 0 {
		t.Fatalf("independent need and a bounded scope (in / out) are required: %+v", ledger.Scope)
	}
	for _, text := range append([]string{ledger.IndependentNeed}, purposes(fixtures)...) {
		lowered := strings.ToLower(text)
		if strings.Contains(lowered, "handoff") || strings.Contains(lowered, "hand-off") || strings.Contains(lowered, "hand over") {
			t.Fatalf("the independent need must stand without the Handoff, but mentions it: %q", text)
		}
	}

	if len(ledger.CopiedArtifacts) != 0 || ledger.LicenseAnalogyMade {
		t.Fatalf("external artifacts were copied or a license was assumed by analogy: %+v", ledger)
	}
	sources := map[string]bool{}
	for _, s := range ledger.Sources {
		if sources[s.ID] || s.ID == "" {
			t.Fatalf("source id %q is empty or duplicated", s.ID)
		}
		sources[s.ID] = true
		if s.Copied || s.Use != "design-pressure-only" || strings.TrimSpace(s.Title) == "" {
			t.Fatalf("source %q is not design-pressure-only: %+v", s.ID, s)
		}
	}
	if len(sources) == 0 {
		t.Fatal("no provenance source is recorded")
	}

	var stepAddress []string
	var actions = map[string]bool{}
	for id, f := range fixtures {
		stepAddress = append(stepAddress, id)
		for _, s := range f.Steps {
			stepAddress = append(stepAddress, s.ID)
			actions[strings.ToLower(s.BusinessAction)] = true
		}
	}
	seenFacts := map[string]bool{}
	for _, fact := range ledger.SourceBackedFacts {
		if seenFacts[fact.ID] || len(fact.Sources) == 0 || strings.TrimSpace(fact.ConfirmedBy) == "" {
			t.Fatalf("source-backed fact %q is duplicated or has no source / confirmation: %+v", fact.ID, fact)
		}
		seenFacts[fact.ID] = true
		for _, source := range fact.Sources {
			if !sources[source] {
				t.Fatalf("fact %q cites unknown source %q", fact.ID, source)
			}
		}
		for _, address := range stepAddress {
			if strings.Contains(fact.Text, address) {
				t.Fatalf("source-backed fact %q refers to TEI-original address %q; facts and design decisions must stay apart", fact.ID, address)
			}
		}
		if actions[strings.ToLower(fact.Text)] {
			t.Fatalf("fact %q is a copy of a TEI step", fact.ID)
		}
	}
	if len(ledger.SourceBackedFacts) == 0 {
		t.Fatal("no source-backed fact is recorded")
	}

	seenDecisions := map[string]bool{}
	for _, decision := range ledger.TEIOriginalDecisions {
		if seenDecisions[decision.ID] || seenFacts[decision.ID] || len(decision.Refs) == 0 {
			t.Fatalf("TEI-original decision %q is duplicated or has no refs", decision.ID)
		}
		seenDecisions[decision.ID] = true
		for _, ref := range decision.Refs {
			if !resolveLedgerRef(fixtures, ref) {
				t.Fatalf("decision %q refers to %q, which does not exist in the fixtures", decision.ID, ref)
			}
		}
	}

	if len(ledger.Derived) != 0 || len(ledger.Proposals) != 0 {
		t.Fatalf("the reviewed seed must not carry unreviewed derived / proposal entries: derived=%d proposals=%d", len(ledger.Derived), len(ledger.Proposals))
	}

	required := []string{
		"actor-identity", "trigger-exact", "step-sequence-authority", "responsibility-boundary-detail",
		"handoff-condition-evidence", "data-identity", "application-realization", "actuality-evidence",
		"acceptance-criteria",
	}
	got := map[string]bool{}
	for _, u := range ledger.Unresolved {
		if u.Status != "unresolved" || strings.TrimSpace(u.Note) == "" {
			t.Fatalf("unresolved item %q was promoted or left unexplained: %+v", u.ID, u)
		}
		got[u.Subject] = true
	}
	for _, subject := range required {
		if !got[subject] {
			t.Fatalf("unresolved subject %q is not listed", subject)
		}
	}

	if len(ledger.CrossViewQuestions) == 0 {
		t.Fatal("a meaningful cross-view question is required")
	}
	for _, q := range ledger.CrossViewQuestions {
		distinct := map[string]bool{}
		for _, v := range q.Views {
			distinct[v] = true
		}
		if len(distinct) < 2 || strings.TrimSpace(q.Question) == "" {
			t.Fatalf("cross-view question %q needs at least two views: %+v", q.ID, q)
		}
	}

	candidate := ledger.Handoff.Candidate
	handoff := set.Handoffs[0]
	if candidate.From != handoff.From.Procedure+"/"+handoff.From.Step || candidate.To != handoff.To.Procedure+"/"+handoff.To.Step {
		t.Fatalf("the ledger's Handoff candidate differs from the reference set: %+v vs %+v", candidate, handoff)
	}
	if !ledger.Handoff.ArisesNaturally || strings.TrimSpace(ledger.Handoff.Basis) == "" {
		t.Fatalf("the Handoff needs a stated basis for arising naturally: %+v", ledger.Handoff)
	}
	for _, inference := range []string{"step-adjacency", "actor-difference", "function-sequence", "positive-observation", "lifecycle-transition", "command-success"} {
		if !contains(ledger.Handoff.NotDerivedFrom, inference) {
			t.Fatalf("the Handoff does not state that it is not derived from %q", inference)
		}
	}
	if ledger.Review.Author != "remediation-authored-after-independent-review" || ledger.Review.IndependentReview != "re-review-required" {
		t.Fatalf("review status must preserve the blocked independent review and require a new independent review: %+v", ledger.Review)
	}
}

func purposes(fixtures map[string]fixture) []string {
	var out []string
	for _, f := range fixtures {
		out = append(out, f.Procedure.Purpose)
	}
	sort.Strings(out)
	return out
}

func TestSecondRealisticReferenceResolvesThroughTheSameLoaderAsTheFirstReference(t *testing.T) {
	set, fixtures := loadSecondCorpus(t)
	if len(set.Procedures) != 2 || len(fixtures) != 2 || len(set.Handoffs) != 1 {
		t.Fatalf("unexpected second Reference shape: procedures=%d fixtures=%d handoffs=%d", len(set.Procedures), len(fixtures), len(set.Handoffs))
	}
	if err := validateReferenceSetHandoffs(set, fixtures); err != nil {
		t.Fatal(err)
	}
	for id, f := range fixtures {
		if f.Canonical || f.Procedure.IdentityStatus != referenceLocalStable {
			t.Fatalf("procedure %q claims Canonical identity: %+v", id, f.Procedure)
		}
		if f.Procedure.Trigger.Status != "candidate" || f.Procedure.Actor.Status != "candidate" {
			t.Fatalf("procedure %q promotes an unresolved trigger / actor to a known fact: %+v", id, f.Procedure)
		}
	}
}

func TestSecondRealisticReferenceIsIndependentOfOrderEntry(t *testing.T) {
	_, second := loadSecondCorpus(t)
	firstRoot := filepath.Join(repoPath(), "reference", "business-system-design")
	firstSet, err := readReferenceSet(filepath.Join(firstRoot, "order-entry-procedure-reference-set.json"))
	if err != nil {
		t.Fatal(err)
	}
	first, err := resolveReferenceSet(firstRoot, firstSet)
	if err != nil {
		t.Fatal(err)
	}
	for id := range second {
		if _, clash := first[id]; clash {
			t.Fatalf("second Reference procedure %q shares an id with the first Reference", id)
		}
	}
	if len(first) != 2 || len(firstSet.Handoffs) != 1 {
		t.Fatalf("the first Reference changed: procedures=%d handoffs=%d", len(first), len(firstSet.Handoffs))
	}
	for id, f := range second {
		if strings.Contains(strings.ToLower(f.Application+f.Procedure.Purpose), "order_entry") {
			t.Fatalf("procedure %q depends on order_entry", id)
		}
	}
}

func TestSecondRealisticReferenceProcedureBodiesStandAloneWithoutTheHandoff(t *testing.T) {
	set, fixtures := loadSecondCorpus(t)
	withoutHandoff := set
	withoutHandoff.Handoffs = nil
	model, err := deriveNormalizedBusinessModelProbe(withoutHandoff, fixtures)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateNormalizedBusinessModelProbe(model); err != nil {
		t.Fatal(err)
	}
	if len(model.Procedures) != 2 || len(model.Handoffs) != 0 {
		t.Fatalf("removing the Handoff must leave two valid procedures and no relation: %+v", model)
	}
	for _, p := range model.Procedures {
		if len(p.Steps) < 3 {
			t.Fatalf("procedure %q has no independent body once the Handoff is removed", p.ID)
		}
	}
}

func TestSecondRealisticReferenceProductionReleaseHasIndependentBusinessMeaning(t *testing.T) {
	_, fixtures := loadSecondCorpus(t)
	ledger := loadSeedLedger(t)
	production := fixtures["make-to-order.production-run"]
	release, ok := stepByID(production, "PR-06")
	if !ok {
		t.Fatal("production terminal release PR-06 is missing")
	}
	for label, text := range map[string]string{
		"production purpose": production.Procedure.Purpose,
		"independent need":   ledger.IndependentNeed,
		"release action":     release.BusinessAction,
	} {
		// "releasing"のような活用形も、同じ業務上のreleaseとして受ける。
		lowered := strings.ToLower(text)
		if !strings.Contains(lowered, "production control") || !strings.Contains(lowered, "releas") {
			t.Fatalf("%s does not explain the terminal production release independently: %q", label, text)
		}
	}
	loweredAction := strings.ToLower(release.BusinessAction)
	for _, receivingSpecific := range []string{"stocking", "warehouse", "handoff"} {
		if strings.Contains(loweredAction, receivingSpecific) {
			t.Fatalf("PR-06 depends on the receiving Handoff instead of the production boundary (%q): %q", receivingSpecific, release.BusinessAction)
		}
	}
	foundDecision := false
	for _, decision := range ledger.TEIOriginalDecisions {
		if decision.ID != "TD-1" {
			continue
		}
		foundDecision = true
		if !contains(decision.Refs, "make-to-order.production-run/PR-06") ||
			!strings.Contains(strings.ToLower(decision.Text), "production control") {
			t.Fatalf("TD-1 does not bind PR-06 to the independent production need: %+v", decision)
		}
	}
	if !foundDecision {
		t.Fatal("TD-1 is missing")
	}
}

func TestSecondRealisticReferenceCarriesNoFakeRealization(t *testing.T) {
	_, fixtures := loadSecondCorpus(t)
	for id, f := range fixtures {
		if violations := secondCorpusFakeRealizationViolations(f); len(violations) != 0 {
			t.Fatalf("procedure %q makes the Reference look executable: %v", id, violations)
		}
	}

	// The check must bite: each way of faking a realization is detected.
	broken := map[string]func(f *fixture){
		"application context": func(f *fixture) { f.Application = "order_entry" },
		"application pattern": func(f *fixture) { f.Steps[0].ApplicationPatterns = []string{"order_entry.order_record"} },
		"data":                func(f *fixture) { f.Steps[0].Data = []string{"sales.order"} },
		"capability":          func(f *fixture) { f.Steps[0].Capabilities = []string{"security.authorize"} },
		"application use":     func(f *fixture) { f.Steps[0].Realization = "application-use" },
	}
	for name, mutate := range broken {
		clone := fixtures["make-to-order.production-run"]
		clone.Steps = append([]step(nil), clone.Steps...)
		mutate(&clone)
		if len(secondCorpusFakeRealizationViolations(clone)) == 0 {
			t.Fatalf("a fake realization (%s) was not detected", name)
		}
	}
}

func TestSecondRealisticReferenceKeepsInstructionEntryAndPhysicalCompletionApart(t *testing.T) {
	_, fixtures := loadSecondCorpus(t)
	ledger := loadSeedLedger(t)
	var refs []string
	for _, decision := range ledger.TEIOriginalDecisions {
		if decision.ID == "TD-2" {
			refs = decision.Refs
		}
	}
	if len(refs) != 4 {
		t.Fatalf("the ledger must name the four separated steps (instruction, work, confirmation, record): %v", refs)
	}
	roles := make([]string, 0, 4)
	for _, ref := range refs {
		_, stepID, _ := strings.Cut(ref, "/")
		roles = append(roles, stepID)
	}
	production := fixtures["make-to-order.production-run"]
	if violations := secondCorpusActualityViolations(production, roles[0], roles[1], roles[2], roles[3]); len(violations) != 0 {
		t.Fatalf("instruction, physical work, confirmation and record are not kept apart: %v", violations)
	}

	// The check must bite.
	merged := production
	merged.Steps = append([]step(nil), production.Steps...)
	for i := range merged.Steps {
		if merged.Steps[i].ID == roles[2] {
			merged.Steps[i].BusinessAction = "confirm the completed quantity and record it"
			merged.Steps[i].Realization = "manual-unresolved"
		}
	}
	if len(secondCorpusActualityViolations(merged, roles[0], roles[1], roles[2], roles[3])) == 0 {
		t.Fatal("a step that both confirms and records was not detected")
	}
	if len(secondCorpusActualityViolations(production, roles[0], roles[1], roles[2], roles[2])) == 0 {
		t.Fatal("using one step for two roles was not detected")
	}
}

func TestSecondRealisticReferenceHandoffKeepsFromToAndConditionLosslessly(t *testing.T) {
	set, fixtures := loadSecondCorpus(t)
	model, err := deriveNormalizedBusinessModelProbe(set, fixtures)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateNormalizedBusinessModelProbe(model); err != nil {
		t.Fatal(err)
	}
	if len(model.Handoffs) != 1 {
		t.Fatalf("handoffs=%d", len(model.Handoffs))
	}
	source := set.Handoffs[0]
	got := model.Handoffs[0]
	want := canonicalBusinessHandoffProbe{
		From:      canonicalBusinessHandoffEndpointProbe{Procedure: source.From.Procedure, Step: source.From.Step},
		To:        canonicalBusinessHandoffEndpointProbe{Procedure: source.To.Procedure, Step: source.To.Step},
		Condition: source.Condition,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("from / to / condition were not kept losslessly: got=%+v want=%+v", got, want)
	}
	if got.From.Procedure != "make-to-order.production-run" || got.From.Step != "PR-06" || got.To.Procedure != "finished-goods.stocking" || got.To.Step != "ST-00" {
		t.Fatalf("the endpoints drifted from the reviewed candidate: %+v", got)
	}

	// The condition is Business meaning: it is none of the things the quality model forbids it to be.
	to := fixtures[source.To.Procedure]
	from := fixtures[source.From.Procedure]
	if strings.EqualFold(got.Condition, to.Procedure.Trigger.Text) {
		t.Fatal("the condition restates the target procedure trigger")
	}
	loweredTrigger := strings.ToLower(to.Procedure.Trigger.Text)
	if strings.Contains(loweredTrigger, "production control") {
		t.Fatalf("the target trigger absorbed the production-side release authority: %q", to.Procedure.Trigger.Text)
	}
	if !strings.Contains(strings.ToLower(got.Condition), "production control") {
		t.Fatalf("the Handoff condition lost the production-side release authority: %q", got.Condition)
	}
	for _, f := range []fixture{from, to} {
		for _, s := range f.Steps {
			if strings.EqualFold(got.Condition, s.BusinessAction) || strings.Contains(got.Condition, s.ID) {
				t.Fatalf("the condition restates step %s or names its identity", s.ID)
			}
		}
	}
	for _, forbidden := range []string{"Observation", "observation", "event id", "Command", "queue"} {
		if strings.Contains(got.Condition, forbidden) {
			t.Fatalf("the condition names runtime / observation vocabulary %q", forbidden)
		}
	}
	loweredCondition := strings.ToLower(got.Condition)
	for _, forbidden := range []string{"taken in charge", "count moves", "count accountability has transferred", "reconciliation already succeeded"} {
		if strings.Contains(loweredCondition, forbidden) {
			t.Fatalf("the condition pre-completes the target intake or count reconciliation (%q): %q", forbidden, got.Condition)
		}
	}
	if !strings.Contains(loweredCondition, "reconciliation") || !strings.Contains(loweredCondition, "pending") {
		t.Fatalf("the condition must leave received-count reconciliation pending: %q", got.Condition)
	}
	if strings.Count(got.Condition, " ") < 10 {
		t.Fatalf("the condition is too thin to carry Business meaning: %q", got.Condition)
	}
}

func TestSecondRealisticReferenceKeepsCountAccountabilityUnresolvedUntilReconciliation(t *testing.T) {
	set, fixtures := loadSecondCorpus(t)
	ledger := loadSeedLedger(t)
	condition := strings.ToLower(set.Handoffs[0].Condition)
	if !strings.Contains(condition, "reconciliation") || !strings.Contains(condition, "pending") {
		t.Fatalf("Handoff condition does not preserve pending count reconciliation: %q", set.Handoffs[0].Condition)
	}
	for _, premature := range []string{"count moves", "count accountability has transferred", "reconciliation succeeded"} {
		if strings.Contains(condition, premature) {
			t.Fatalf("Handoff condition prematurely resolves count accountability (%q): %q", premature, set.Handoffs[0].Condition)
		}
	}
	stocking := fixtures["finished-goods.stocking"]
	reconcile, ok := stepByID(stocking, "ST-01")
	if !ok || reconcile.Realization != "observation-boundary" || !hasAnyWord(reconcile.BusinessAction, "check") {
		t.Fatalf("ST-01 must remain the separate received-count reconciliation boundary: %+v", reconcile)
	}
	requiredUnresolved := map[string]bool{
		"responsibility-boundary-detail": false,
		"data-identity":                  false,
		"acceptance-criteria":            false,
	}
	for _, u := range ledger.Unresolved {
		if _, ok := requiredUnresolved[u.Subject]; ok && u.Status == "unresolved" {
			requiredUnresolved[u.Subject] = true
		}
	}
	for subject, found := range requiredUnresolved {
		if !found {
			t.Fatalf("count-accountability remediation must retain unresolved subject %q", subject)
		}
	}
	for _, q := range ledger.CrossViewQuestions {
		if q.ID == "CQ-1" && q.DataIdentity != "unresolved" {
			t.Fatalf("CQ-1 must not fabricate Data ownership while P2 item 4 is unresolved: %+v", q)
		}
	}
}

func TestSecondRealisticReferenceHandoffIsNotSynthesizedFromAdjacencyActorChangeOrPositiveObservation(t *testing.T) {
	set, fixtures := loadSecondCorpus(t)
	handoff := set.Handoffs[0]
	from := fixtures[handoff.From.Procedure]
	to := fixtures[handoff.To.Procedure]

	// Adjacency, an actor change and a positive physical Observation are all present ...
	if last := from.Steps[len(from.Steps)-1].ID; last != handoff.From.Step {
		t.Fatalf("test setup: the source step must end its procedure, got %q", last)
	}
	if to.Steps[0].ID != handoff.To.Step {
		t.Fatalf("test setup: the target step must start its procedure, got %q", to.Steps[0].ID)
	}
	if from.Procedure.Actor.Label == to.Procedure.Actor.Label {
		t.Fatal("test setup: the two responsibility contexts must differ")
	}
	execution, err := driver.ExecuteObservationRead(
		context.Background(),
		"second-reference.physical-arrival.read",
		driver.ObservationEvidenceExternalObserved,
		func(context.Context) (any, error) { return true, nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	if observed, ok := execution.Result.(bool); !ok || !observed {
		t.Fatalf("test setup: the positive physical Observation was not established: %+v", execution)
	}

	// ... and still none of them creates the responsibility relation.
	withoutHandoff := set
	withoutHandoff.Handoffs = nil
	model, err := deriveNormalizedBusinessModelProbe(withoutHandoff, fixtures)
	if err != nil {
		t.Fatal(err)
	}
	if len(model.Handoffs) != 0 {
		t.Fatalf("adjacency / actor change / positive Observation synthesized a Handoff: %+v", model.Handoffs)
	}
	d4, err := deriveD4NormalizedBusinessModelProbe(withoutHandoff, fixtures)
	if err != nil {
		t.Fatal(err)
	}
	if len(d4.Handoffs) != 0 {
		t.Fatalf("the D4 derivation synthesized a Handoff: %+v", d4.Handoffs)
	}
}

func TestSecondRealisticReferenceExplicitHandoffDoesNotDependOnActorDifference(t *testing.T) {
	set, fixtures := loadSecondCorpus(t)
	from := fixtures[set.Handoffs[0].From.Procedure]
	to := fixtures[set.Handoffs[0].To.Procedure]
	to.Procedure.Actor.Label = from.Procedure.Actor.Label
	fixtures[set.Handoffs[0].To.Procedure] = to

	model, err := deriveNormalizedBusinessModelProbe(set, fixtures)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateNormalizedBusinessModelProbe(model); err != nil {
		t.Fatal(err)
	}
	if len(model.Handoffs) != 1 {
		t.Fatalf("explicit Handoff disappeared when actor labels became equal: %+v", model.Handoffs)
	}

	withoutHandoff := set
	withoutHandoff.Handoffs = nil
	model, err = deriveNormalizedBusinessModelProbe(withoutHandoff, fixtures)
	if err != nil {
		t.Fatal(err)
	}
	if len(model.Handoffs) != 0 {
		t.Fatalf("equal actor labels synthesized a Handoff without explicit source: %+v", model.Handoffs)
	}
}

func TestSecondRealisticReferenceHandoffFailsClosedOnStaleMissingOrEmptyEndpoints(t *testing.T) {
	set, fixtures := loadSecondCorpus(t)
	cases := map[string]func(s *referenceSet){
		"stale target step":  func(s *referenceSet) { s.Handoffs[0].To.Step = "ST-00-renamed" },
		"stale source step":  func(s *referenceSet) { s.Handoffs[0].From.Step = "PR-06-renamed" },
		"unknown source":     func(s *referenceSet) { s.Handoffs[0].From.Procedure = "make-to-order.unknown" },
		"unknown target":     func(s *referenceSet) { s.Handoffs[0].To.Procedure = "finished-goods.unknown" },
		"empty condition":    func(s *referenceSet) { s.Handoffs[0].Condition = "" },
		"duplicate relation": func(s *referenceSet) { s.Handoffs = append(s.Handoffs, s.Handoffs[0]) },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			changed := set
			changed.Handoffs = append([]referenceSetHandoff(nil), set.Handoffs...)
			mutate(&changed)
			if secondCorpusHandoffAccepted(changed, fixtures) {
				t.Fatal("an invalid Handoff was accepted silently")
			}
		})
	}
}

func TestSecondRealisticReferenceStandaloneHandoffSourceCarriesTheSameMeaning(t *testing.T) {
	set, fixtures := loadSecondCorpus(t)
	want, err := deriveCanonicalBusinessHandoffProbes(set, fixtures)
	if err != nil {
		t.Fatal(err)
	}
	source := standaloneHandoffSourceProbe{
		SchemaVersion: standaloneHandoffSchemaVersion,
		ArtifactRole:  standaloneHandoffArtifactRole,
		Canonical:     false,
		Handoffs:      append([]referenceSetHandoff(nil), set.Handoffs...),
	}
	got, err := deriveStandaloneHandoffProbes(source, fixtures)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("the standalone Handoff source form differs from the reference set for the second Reference: got=%+v want=%+v", got, want)
	}
	encoded, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, []byte(`"artifact"`)) {
		t.Fatalf("a Procedure artifact locator leaked into the standalone Handoff source: %s", encoded)
	}
	reloaded, err := decodeStandaloneHandoffSourceProbe(bytes.NewReader(encoded))
	if err != nil || !reflect.DeepEqual(source, reloaded) {
		t.Fatalf("the standalone Handoff source did not round-trip: %v", err)
	}
}

func TestSecondRealisticReferenceD4ModelHoldsWithoutAProtectedTargetRole(t *testing.T) {
	set, fixtures := loadSecondCorpus(t)
	model, err := deriveD4NormalizedBusinessModelProbe(set, fixtures)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateD4NormalizedBusinessModelProbe(model); err != nil {
		t.Fatal(err)
	}
	if len(model.Procedures) != 2 || len(model.Handoffs) != 1 {
		t.Fatalf("unexpected D4 model: procedures=%d handoffs=%d", len(model.Procedures), len(model.Handoffs))
	}
	// O1 applies only where a protected-target role actually repeats. This seed references no
	// Application Pattern, so nothing is protected and no target-role taxonomy is introduced.
	_, fixtures = loadSecondCorpus(t)
	for id, f := range fixtures {
		for _, s := range f.Steps {
			if len(s.ApplicationPatterns) != 0 {
				t.Fatalf("procedure %q step %q references an Application Pattern; O1 would then have to be reviewed", id, s.ID)
			}
		}
	}
}

func TestSecondRealisticReferenceBaselineCheckerReportsNoDiagnostics(t *testing.T) {
	loaded, err := workspace.Load(filepath.Join(repoPath(), "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	_, fixtures := loadSecondCorpus(t)
	for id, f := range fixtures {
		result := evaluate(loaded.World, f)
		if len(result.Diagnostics) != 0 {
			t.Fatalf("baseline checker reports diagnostics for %q: %+v", id, result.Diagnostics)
		}
		if result.CanonicalProcedureIdentityUsed {
			t.Fatalf("procedure %q used Canonical identity", id)
		}
		// C1: the Procedure is explained without returning it to a single Application owner, and every
		// manual / physical / observation Action is held without a fake Application Function.
		if result.Application != "" {
			t.Fatalf("procedure %q was reported under an Application owner %q", id, result.Application)
		}
		for _, row := range result.BusinessActionFunctionRows {
			if !row.NoApplicationFunction || len(row.ApplicationPatterns) != 0 {
				t.Fatalf("procedure %q step %q was given an Application Function: %+v", id, row.StepID, row)
			}
		}
		if len(result.BusinessActionPatternMeaningRows) != 0 {
			t.Fatalf("procedure %q derived Application Pattern meaning: %+v", id, result.BusinessActionPatternMeaningRows)
		}
	}
}

func TestSecondRealisticReferenceDoesNotCopyExternalMaterial(t *testing.T) {
	_, fixtures := loadSecondCorpus(t)
	ledger := loadSeedLedger(t)
	externalNames := []string{"TALON", "talon", "CONCEPTWARE", "XEAD", "Xead", "X-TEA", "http://", "https://"}
	for id, f := range fixtures {
		texts := []string{f.Procedure.Purpose, f.Procedure.Trigger.Text, f.Procedure.Actor.Label}
		for _, s := range f.Steps {
			texts = append(texts, s.BusinessAction)
			if len(s.BusinessAction) > 160 {
				t.Fatalf("procedure %q step %q is long enough to be a transcription", id, s.ID)
			}
		}
		for _, text := range texts {
			for _, name := range externalNames {
				if strings.Contains(text, name) {
					t.Fatalf("procedure %q carries external material or naming (%q): %q", id, name, text)
				}
			}
		}
	}
	// Provenance lives in the ledger, not in the fixtures.
	if len(ledger.Sources) == 0 {
		t.Fatal("provenance is missing")
	}
}

// secondCorpusHandoffAccepted reports whether every check that the first Reference applies to its Handoff
// accepts the reference set: the canonical derivation, then the normalized model and its validation.
func secondCorpusHandoffAccepted(set referenceSet, fixtures map[string]fixture) bool {
	if _, err := deriveCanonicalBusinessHandoffProbes(set, fixtures); err != nil {
		return false
	}
	model, err := deriveNormalizedBusinessModelProbe(set, fixtures)
	if err != nil {
		return false
	}
	return validateNormalizedBusinessModelProbe(model) == nil
}
