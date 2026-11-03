package externaladoptionauthority

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ErrInputMismatch is how every stage stops when what it receives differs
// from what the previous stage fixed.
var ErrInputMismatch = errors.New("input mismatch")

const lossRecordSchema = "tei.reference.external-adoption-authority.loss-record/v0"

// The four SUI files whose blobs stage 1 pins, in the order of the report
// fields of tools/reference-sui-adoption.
var pinnedFiles = []string{
	"island_contracts.ts",
	"island_hierarchy_handoff.ts",
	"island_contracts.test.ts",
	"island_hierarchy_handoff.test.ts",
}

// Fixtures of reference/sui-adoption and the result its README expects.
var stage1Fixtures = []struct {
	File   string
	Expect string
}{
	{"M1-root.valid.json", "valid"},
	{"M2-three-level.valid.json", "valid"},
	{"M3-missing-parent.invalid.json", "invalid"},
	{"M4-cycle.invalid.json", "invalid"},
}

// Coverage is one constraintCoverage entry of the stage 1 report, verbatim.
type Coverage struct {
	ID                string `json:"id"`
	SourceConstraint  string `json:"sourceConstraint"`
	TEIRepresentation string `json:"teiRepresentation"`
	Status            string `json:"status"`
	ProbeEnforcement  string `json:"probeEnforcement"`
}

type suiReport struct {
	SchemaVersion          string `json:"schemaVersion"`
	SourceContract         string `json:"sourceContract"`
	SourceRepository       string `json:"sourceRepository"`
	SourceRevision         string `json:"sourceRevision"`
	SourceContractBlob     string `json:"sourceContractBlob"`
	SourceHandoffBlob      string `json:"sourceHandoffBlob"`
	SourceContractTestBlob string `json:"sourceContractTestBlob"`
	SourceHandoffTestBlob  string `json:"sourceHandoffTestBlob"`
	TEIAuthority           string `json:"teiAuthority"`
	Fixture                string `json:"fixture"`
	FixtureValid           bool   `json:"fixtureValid"`
	Diagnostics            []struct {
		Code   string `json:"code"`
		Island string `json:"island"`
	} `json:"diagnostics"`
	Mapping struct {
		DataDefinition          string `json:"dataDefinition"`
		ParentRelation          string `json:"parentRelation"`
		RelationTarget          string `json:"relationTarget"`
		RelationCardinality     string `json:"relationCardinality"`
		ParentPresenceSemantics string `json:"parentPresenceSemantics"`
		Lossless                *bool  `json:"lossless"`
	} `json:"mapping"`
	ConstraintCoverage []Coverage `json:"constraintCoverage"`
	WriteAttempted     bool       `json:"writeAttempted"`
	Canonicalized      bool       `json:"canonicalizedInstanceData"`
}

func (r suiReport) pinned() PinnedSource {
	return PinnedSource{
		Repository: r.SourceRepository,
		Revision:   r.SourceRevision,
		Blobs: map[string]string{
			pinnedFiles[0]: r.SourceContractBlob,
			pinnedFiles[1]: r.SourceHandoffBlob,
			pinnedFiles[2]: r.SourceContractTestBlob,
			pinnedFiles[3]: r.SourceHandoffTestBlob,
		},
	}
}

// LossRecord states, from the stage 1 report and without reinterpretation,
// what the mapping could not represent. Later stages carry its bytes and
// digest; they never rebuild it.
type LossRecord struct {
	Schema                  string       `json:"schema"`
	SourceContract          string       `json:"sourceContract"`
	Source                  PinnedSource `json:"source"`
	MappingReportDigest     string       `json:"mappingReportDigest"`
	DataDefinition          string       `json:"dataDefinition"`
	ParentRelation          string       `json:"parentRelation"`
	RelationCardinality     string       `json:"relationCardinality"`
	ParentPresenceSemantics string       `json:"parentPresenceSemantics"`
	Lossless                bool         `json:"lossless"`
	ConstraintCoverage      []Coverage   `json:"constraintCoverage"`
	Unresolved              []string     `json:"unresolved"`
	ShadowOnly              []string     `json:"shadowOnly"`
	ShapeOnly               []string     `json:"shapeOnly"`
}

// ReadPinnedTable reads the "Pinned value" table of
// reference/sui-adoption/README.md: the fixed copy of what the SUI side
// pinned. Nothing is fetched from the SUI repository.
func ReadPinnedTable(readme string) (PinnedSource, error) {
	f, err := os.Open(readme)
	if err != nil {
		return PinnedSource{}, err
	}
	defer f.Close()
	out := PinnedSource{Blobs: map[string]string{}}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "|") {
			continue
		}
		cells := strings.Split(strings.Trim(line, "|"), "|")
		if len(cells) != 2 {
			continue
		}
		label := strings.TrimSpace(cells[0])
		value := strings.Trim(strings.TrimSpace(cells[1]), "`")
		switch {
		case label == "repository":
			out.Repository = value
		case label == "navigation revision":
			out.Revision = value
		case strings.HasSuffix(label, " blob"):
			name := strings.Trim(strings.TrimSuffix(label, " blob"), "`")
			out.Blobs[name] = value
		}
	}
	if err := sc.Err(); err != nil {
		return PinnedSource{}, err
	}
	if out.Repository == "" || out.Revision == "" || len(out.Blobs) != len(pinnedFiles) {
		return PinnedSource{}, fmt.Errorf("pinned table of %s is incomplete: %+v", readme, out)
	}
	for _, name := range pinnedFiles {
		if out.Blobs[name] == "" {
			return PinnedSource{}, fmt.Errorf("pinned table of %s has no blob for %s", readme, name)
		}
	}
	return out, nil
}

// PinCheck is the comparison of the reported pin with the external side.
type PinCheck struct {
	BlobMismatches   []string `json:"blobMismatches,omitempty"`
	RevisionAdvanced bool     `json:"revisionAdvanced"`
	RepositoryDiffer bool     `json:"repositoryDiffers,omitempty"`
}

// CheckExternalPin compares the pin stage 1 reported with the external
// side observed now. A changed dependency blob or another repository stops
// the stage; a revision that moved while the four blobs are unchanged does
// not (the README binds freshness to the blobs), but it is recorded.
func CheckExternalPin(reported, observed PinnedSource) (PinCheck, error) {
	var pc PinCheck
	pc.RepositoryDiffer = reported.Repository != observed.Repository
	pc.RevisionAdvanced = reported.Revision != observed.Revision
	for _, name := range pinnedFiles {
		if reported.Blobs[name] != observed.Blobs[name] {
			pc.BlobMismatches = append(pc.BlobMismatches,
				fmt.Sprintf("%s: pinned %s, external %s", name, reported.Blobs[name], observed.Blobs[name]))
		}
	}
	if len(reported.Blobs) != len(pinnedFiles) || len(observed.Blobs) != len(pinnedFiles) {
		pc.BlobMismatches = append(pc.BlobMismatches, "the pin does not have exactly four dependency blobs")
	}
	if pc.RepositoryDiffer || len(pc.BlobMismatches) > 0 {
		return pc, fmt.Errorf("%w: SUI dependency blobs differ from the pinned version: %s",
			ErrInputMismatch, strings.Join(pc.BlobMismatches, "; "))
	}
	return pc, nil
}

func readPinnedSource(path string) (PinnedSource, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return PinnedSource{}, err
	}
	var p PinnedSource
	if err := json.Unmarshal(raw, &p); err != nil {
		return PinnedSource{}, err
	}
	return p, nil
}

// WriteExternalCopy writes the simulated external side: a copy of the
// pinned table. Fault injections change this copy, never the repository.
func WriteExternalCopy(p PinnedSource, dir string) (string, error) {
	raw, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "pinned-source.json")
	return path, writeFile(path, append(raw, '\n'))
}

type stage1Out struct {
	InputDir      string
	ReportsDir    string
	LossPath      string
	Loss          LossRecord
	Reports       map[string]suiReport
	Pin           PinCheck
	Carried       Carried
	FixtureValid  map[string]bool
	ReportDigests map[string]string
}

func mappingDigestFromDir(dir string) (string, map[string]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", nil, err
	}
	type item struct {
		Fixture string `json:"fixture"`
		Digest  string `json:"digest"`
	}
	var items []item
	digests := map[string]string{}
	for _, e := range entries {
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return "", nil, err
		}
		d := digestBytes(raw)
		items = append(items, item{Fixture: e.Name(), Digest: d})
		digests[e.Name()] = d
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Fixture < items[j].Fixture })
	if len(items) != len(stage1Fixtures) {
		return "", nil, fmt.Errorf("%w: %d mapping reports, want %d", ErrInputMismatch, len(items), len(stage1Fixtures))
	}
	d, err := digestJSON(items)
	return d, digests, err
}

// runStage1 copies reference/sui-adoption, runs the unchanged shadow probe
// on the four fixtures, compares the reported pin with the external side and
// writes the loss record. It returns ErrInputMismatch without any output
// when the external side differs.
func runStage1(ctx context.Context, tools Tools, repo, dir, externalPath string) (stage1Out, error) {
	out := stage1Out{
		InputDir:      filepath.Join(dir, "input"),
		ReportsDir:    filepath.Join(dir, "reports"),
		Reports:       map[string]suiReport{},
		FixtureValid:  map[string]bool{},
		ReportDigests: map[string]string{},
	}
	if err := copyTree(filepath.Join(repo, "reference", "sui-adoption"), out.InputDir); err != nil {
		return out, err
	}
	var first *suiReport
	for _, fx := range stage1Fixtures {
		res, err := runCLI(ctx, tools.SUIAdoption, out.InputDir,
			"-workspace", "workspace", "-fixture", fx.File, "-expect", fx.Expect)
		if err != nil {
			return out, err
		}
		if res.Exit != 0 {
			return out, fmt.Errorf("shadow probe %s exit %d: %s", fx.File, res.Exit, res.Stderr)
		}
		var rep suiReport
		if err := json.Unmarshal(res.Stdout, &rep); err != nil {
			return out, fmt.Errorf("shadow probe %s report: %w", fx.File, err)
		}
		if rep.Mapping.Lossless == nil {
			return out, fmt.Errorf("shadow probe %s report has no lossless field", fx.File)
		}
		if first == nil {
			first = &rep
		} else if !rep.pinned().equal(first.pinned()) || !sameMapping(rep, *first) ||
			!sameCoverage(rep.ConstraintCoverage, first.ConstraintCoverage) {
			return out, fmt.Errorf("shadow probe reports disagree on pin, mapping or coverage (%s)", fx.File)
		}
		out.Reports[fx.File] = rep
		out.FixtureValid[fx.File] = rep.FixtureValid
		if err := writeFile(filepath.Join(out.ReportsDir, fx.File), res.Stdout); err != nil {
			return out, err
		}
	}

	observed, err := readPinnedSource(externalPath)
	if err != nil {
		return out, err
	}
	out.Pin, err = CheckExternalPin(first.pinned(), observed)
	if err != nil {
		// Stop before anything is handed on: no loss record, no carried values.
		_ = os.RemoveAll(out.ReportsDir)
		return out, err
	}

	mapping, digests, err := mappingDigestFromDir(out.ReportsDir)
	if err != nil {
		return out, err
	}
	out.ReportDigests = digests
	input, err := treeDigest(out.InputDir)
	if err != nil {
		return out, err
	}
	loss := LossRecord{
		Schema:                  lossRecordSchema,
		SourceContract:          first.SourceContract,
		Source:                  first.pinned(),
		MappingReportDigest:     mapping,
		DataDefinition:          first.Mapping.DataDefinition,
		ParentRelation:          first.Mapping.ParentRelation,
		RelationCardinality:     first.Mapping.RelationCardinality,
		ParentPresenceSemantics: first.Mapping.ParentPresenceSemantics,
		Lossless:                *first.Mapping.Lossless,
		ConstraintCoverage:      first.ConstraintCoverage,
	}
	for _, c := range first.ConstraintCoverage {
		switch c.Status {
		case "unresolved":
			loss.Unresolved = append(loss.Unresolved, c.ID)
		case "shadow-only":
			loss.ShadowOnly = append(loss.ShadowOnly, c.ID)
		case "shape-only":
			loss.ShapeOnly = append(loss.ShapeOnly, c.ID)
		}
	}
	raw, err := json.MarshalIndent(loss, "", "  ")
	if err != nil {
		return out, err
	}
	raw = append(raw, '\n')
	out.LossPath = filepath.Join(dir, "loss-record.json")
	if err := writeFile(out.LossPath, raw); err != nil {
		return out, err
	}
	out.Loss = loss
	lossless := loss.Lossless
	out.Carried = Carried{
		Source:              first.pinned(),
		InputDigest:         input,
		MappingReportDigest: mapping,
		Namespace:           namespaceOf(first.Mapping.DataDefinition),
		Subject:             "semantic-data:" + first.Mapping.DataDefinition,
		Lossless:            &lossless,
		ParentPresence:      loss.ParentPresenceSemantics,
		LossRecordDigest:    digestBytes(raw),
	}
	return out, nil
}

func namespaceOf(id string) string {
	if i := strings.Index(id, "."); i > 0 {
		return id[:i]
	}
	return id
}

func sameMapping(a, b suiReport) bool {
	x, y := a.Mapping, b.Mapping
	x.Lossless, y.Lossless = nil, nil
	return x == y && *a.Mapping.Lossless == *b.Mapping.Lossless &&
		a.SchemaVersion == b.SchemaVersion && a.SourceContract == b.SourceContract && a.TEIAuthority == b.TEIAuthority
}

func sameCoverage(a, b []Coverage) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// readLossRecord re-reads the carried loss record and checks it against
// the carried digest. A stage that needs the loss record uses only this.
func readLossRecord(path, wantDigest string) (LossRecord, []byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return LossRecord{}, nil, err
	}
	if d := digestBytes(raw); d != wantDigest {
		return LossRecord{}, nil, fmt.Errorf("%w: loss record digest %s, carried %s", ErrInputMismatch, d, wantDigest)
	}
	var loss LossRecord
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&loss); err != nil {
		return LossRecord{}, nil, err
	}
	if loss.Lossless {
		return LossRecord{}, nil, fmt.Errorf("%w: loss record reads lossless=true", ErrInputMismatch)
	}
	return loss, raw, nil
}

func diagnosticCodes(r suiReport) string {
	var codes []string
	for _, d := range r.Diagnostics {
		codes = append(codes, d.Code)
	}
	sort.Strings(codes)
	return strings.Join(codes, ",")
}

func diagnosticIslands(r suiReport) string {
	var islands []string
	for _, d := range r.Diagnostics {
		islands = append(islands, d.Island)
	}
	return strings.Join(islands, ",")
}
