package externalanalysis

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"tei/internal/jsonl"
)

// Framing is the JSON Lines shape the recorded producer version emits.
type Framing struct {
	Version string
	// Control is true when object collections finish with a $rea=jsonl/v1
	// metadata record, and false when the version writes one document per
	// delivery with no control record at all.
	Control bool
	// Known reports whether this expectation was declared at all. A zero
	// Framing does not mean "no control records"; it means "nobody said", and
	// the inspection refuses to turn that into a verdict.
	Known bool
}

// observedFraming records the framing each tool version was observed to emit.
// Both entries were read off real runs on 2026-11-11 against the same saved
// Evidence: 6.7.0 wrote one Evidence document with no terminal record, 6.8.0
// wrote the rows plus a $rea=jsonl/v1 metadata terminal (#3394).
var observedFraming = map[string]bool{
	"6.7.0": false,
	"6.8.0": true,
}

// FramingFor returns the framing the recorded producer version emits, and
// whether that version is one this package has observed.
//
// An unobserved version returns a Framing nobody declared, and inspecting a
// delivery against it says so instead of guessing: a missing terminal record
// means "interrupted" for one version and "this version has no terminal" for
// another, and the difference is not visible in the bytes.
func FramingFor(version string) (Framing, bool) {
	control, ok := observedFraming[version]
	if !ok {
		return Framing{Version: version}, false
	}
	return Framing{Version: version, Control: control, Known: true}, true
}

// WithView classifies one saved view delivery and attaches it to the receipt.
//
// The delivery is checked against the framing the caller declared for the
// producer version, and the worse of the two verdicts wins, so a complete
// Evidence followed by an interrupted view is still a partial receipt.
func (r Receipt) WithView(raw []byte, expected Framing) Receipt {
	view, state, reason := inspectView(raw, expected, r.Evidence)
	if view.Failure != nil && r.Failure == nil {
		r.Failure = view.Failure
	}
	r.View = &view
	r.State, r.Reason = merge(r.State, r.Reason, state, reason)
	return r
}

// viewRecord is one line of a view delivery. The control fields are the
// protocol's own; the rest are read so a delivery that is not a view at all —
// a response envelope, an error document — is recognized as such instead of
// being counted as rows.
type viewRecord struct {
	Control       string          `json:"$rea"`
	Type          string          `json:"type"`
	OK            *bool           `json:"ok"`
	Result        json.RawMessage `json:"result"`
	Error         json.RawMessage `json:"error"`
	Data          json.RawMessage `json:"data"`
	Counts        *viewCounts     `json:"records"`
	Context       *viewContext    `json:"context"`
	ObservationID string          `json:"observation_id"`
	EvidenceID    string          `json:"evidence_id"`
	Operation     string          `json:"operation"`
	Normalized    json.RawMessage `json:"normalized_result"`
	// The diagnostic's own fields, for the flat shape the tool also writes.
	Code     string `json:"code"`
	Category string `json:"category"`
	Message  string `json:"message"`
}

type viewCounts struct {
	Path    []string `json:"path"`
	Emitted int      `json:"emitted"`
	Source  int      `json:"source"`
}

type viewContext struct {
	EvidenceID string          `json:"evidence_id"`
	Normalized json.RawMessage `json:"normalized_result"`
}

// viewResult is the projection the delivery carries about its parent. It is
// copied, not interpreted: the intake reads what the producer declared about
// coverage and unknowns so a later reader sees the same boundaries.
type viewResult struct {
	ParentEvidenceID string          `json:"parent_evidence_id"`
	Coverage         json.RawMessage `json:"coverage"`
	Limitations      []string        `json:"limitations"`
	Unknowns         []string        `json:"unknowns"`
	Rows             []struct {
		ObservationID string `json:"observation_id"`
	} `json:"rows"`
}

func inspectView(raw []byte, expected Framing, parent *Evidence) (View, State, string) {
	view := View{}
	if len(raw) == 0 {
		return view, StateMissing, "view配送のbytesが空です: 投影結果が取得されていません"
	}
	encoding, text, err := decodeText(raw)
	if err != nil {
		return view, StateMismatch, err.Error()
	}
	view.Encoding = encoding
	if !expected.Known {
		return view, StatePartial, fmt.Sprintf("tool version %q のJSONL framingが宣言されていないため、配送の完全性を判断できません", expected.Version)
	}
	// The line rules are the shared ones: a damaged line is an error and the
	// earlier lines come back, so "how far can this be trusted" is answerable
	// without a second implementation of the reader.
	records, readErr := jsonl.ReadBytes([]byte(text), "external analysis view", jsonl.DecodeJSON[viewRecord])
	if readErr != nil {
		if errors.Is(readErr, bufio.ErrTooLong) {
			return view, StatePartial, "一行が共有JSONL readerの上限を超えています: 行を分割して読み直せる形ではありません"
		}
		view.Records = countData(records)
		return view, StatePartial, fmt.Sprintf("受領した行の途中でJSONが壊れています（途中切断の可能性）: %v", readErr)
	}
	if len(records) == 0 {
		return view, StateMissing, "view配送にrecordが一行もありません"
	}
	for _, record := range records {
		if isEnvelopeRecord(record) {
			return view, StateMismatch, "CLI応答の包装({ok,data,meta})をview配送として受領しました: Evidenceの投影結果ではありません"
		}
	}
	for _, record := range records {
		if failure, ok := record.failure(); ok {
			view.Failure = &failure
			return view, StateFailed, fmt.Sprintf("external toolが失敗を報告しました: %s %s", failure.Code, failure.Message)
		}
	}
	if expected.Control {
		return inspectControlFraming(view, records, parent)
	}
	return inspectLegacyFraming(view, records, parent)
}

// isEnvelopeRecord recognizes the CLI response wrapper standing where a view
// record should be. This is the same confusion the request derivation refuses,
// caught on the way back in.
func isEnvelopeRecord(record viewRecord) bool {
	return record.OK != nil && len(record.Data) > 0 && string(record.Data) != "null"
}

func (r viewRecord) failure() (Failure, bool) {
	if r.Control == "jsonl/v1" && r.Type == "error" {
		if failure, ok := decodeFailure(r.Result); ok {
			return failure, true
		}
		if failure, ok := r.flatFailure(); ok {
			return failure, true
		}
		return Failure{Code: "error-record", Message: "external toolがerror control recordを出しました"}, true
	}
	if len(r.Error) > 0 && string(r.Error) != "null" {
		if failure, ok := decodeFailure(r.Error); ok {
			return failure, true
		}
		if failure, ok := r.flatFailure(); ok {
			return failure, true
		}
		return Failure{Code: "error", Message: "external toolがerror documentを出しました"}, true
	}
	return Failure{}, false
}

// flatFailure reads a diagnostic whose fields sit beside the error marker
// instead of inside it.
func (r viewRecord) flatFailure() (Failure, bool) {
	if r.Code == "" && r.Message == "" {
		return Failure{}, false
	}
	return Failure{Code: r.Code, Category: r.Category, Message: r.Message}, true
}

func countData(records []viewRecord) int {
	count := 0
	for _, record := range records {
		if record.Control == "" {
			count++
		}
	}
	return count
}

// inspectControlFraming checks a delivery whose declared version finishes
// object collections with a $rea=jsonl/v1 metadata record.
func inspectControlFraming(view View, records []viewRecord, parent *Evidence) (View, State, string) {
	view.Framing = "jsonl/v1-rows-and-terminal"
	controls := 0
	terminalIndex := -1
	for index, record := range records {
		switch {
		case record.Control == "":
			// a data record
		case record.Control == "jsonl/v1":
			controls++
			terminalIndex = index
		default:
			return view, StateMismatch, fmt.Sprintf("未知の制御record $rea=%qです: このprotocolのrecordとして扱えません", record.Control)
		}
	}
	view.Records = countData(records)
	if controls == 0 {
		return view, StatePartial, "終端metadata recordがありません: 配送が途中で切れたか、宣言されたversionと違うprotocolです。届いている行だけで全体であるかのように扱わないでください"
	}
	if controls > 1 {
		return view, StateMismatch, fmt.Sprintf("制御recordが%d件あります: 終端は一つであるべきです", controls)
	}
	if terminalIndex != len(records)-1 {
		return view, StateMismatch, "終端metadata recordが最後にありません: 配送の順序がprotocolと異なります"
	}
	terminal := records[terminalIndex]
	if terminal.Type != "metadata" {
		return view, StateMismatch, fmt.Sprintf("終端の制御recordのtype=%qはmetadataではありません", terminal.Type)
	}
	view.Control = "metadata"
	if terminal.Counts != nil {
		view.Emitted = terminal.Counts.Emitted
		view.SourceCount = terminal.Counts.Source
		view.Collection = terminal.Counts.Path
		if terminal.Counts.Emitted != view.Records {
			return view, StateMismatch, fmt.Sprintf("終端の宣言emitted=%dが届いたdata record数%dと異なります", terminal.Counts.Emitted, view.Records)
		}
		if terminal.Counts.Source < terminal.Counts.Emitted {
			return view, StateMismatch, fmt.Sprintf("終端の宣言source=%dがemitted=%dより小さいです", terminal.Counts.Source, terminal.Counts.Emitted)
		}
	}
	if terminal.Context == nil {
		return view, StateMismatch, "終端metadata recordにcontextがありません: 親Evidence・coverage・unknownsを読み取れません"
	}
	view.EvidenceID = terminal.Context.EvidenceID
	if duplicates := duplicateIdentities(records); len(duplicates) > 0 {
		view.Duplicates = duplicates
		return view, StateMismatch, "同じidentityのrecordが複数届いています: " + strings.Join(duplicates, ", ")
	}
	result, ok := decodeViewResult(terminal.Context.Normalized)
	if !ok {
		return view, StateMismatch, "終端metadata recordのcontextにnormalized_resultがありません"
	}
	view.ParentEvidenceID = result.ParentEvidenceID
	view.Coverage = decodeCoverage(result.Coverage)
	view.Limitations = result.Limitations
	view.Unknowns = result.Unknowns
	if parent != nil && view.ParentEvidenceID != "" {
		if view.ParentEvidenceID != parent.ID {
			return view, StateMismatch, fmt.Sprintf("配送が宣言する親Evidence %sは受領したEvidence %sと異なります", view.ParentEvidenceID, parent.ID)
		}
		view.ParentLinked = true
	}
	state, reason := coverageVerdict(view.Coverage)
	return view, state, reason
}

// inspectLegacyFraming checks a delivery whose declared version writes one
// document with no control record.
func inspectLegacyFraming(view View, records []viewRecord, parent *Evidence) (View, State, string) {
	view.Framing = "legacy-single-document"
	for _, record := range records {
		if record.Control != "" {
			return view, StateMismatch, fmt.Sprintf("制御record($rea=%q)がありますが、宣言されたversionの配送は単一documentです: 版の取り違えです", record.Control)
		}
	}
	view.Records = len(records)
	if len(records) != 1 {
		return view, StateMismatch, fmt.Sprintf("単一documentの配送であるべきところが%d行です: 版の取り違えか、配送の分割です", len(records))
	}
	document := records[0]
	if document.EvidenceID == "" || document.Operation == "" {
		return view, StateMismatch, "documentがEvidence recordではありません: evidence_idとoperationがありません"
	}
	view.Control = "none"
	view.EvidenceID = document.EvidenceID
	result, ok := decodeViewResult(document.Normalized)
	if !ok {
		return view, StateMismatch, "documentにnormalized_resultがありません"
	}
	view.ParentEvidenceID = result.ParentEvidenceID
	view.Coverage = decodeCoverage(result.Coverage)
	view.Limitations = result.Limitations
	view.Unknowns = result.Unknowns
	if duplicates := duplicateRowIdentities(result.Rows); len(duplicates) > 0 {
		view.Duplicates = duplicates
		return view, StateMismatch, "同じidentityのrowが複数届いています: " + strings.Join(duplicates, ", ")
	}
	if parent != nil && view.ParentEvidenceID != "" {
		if view.ParentEvidenceID != parent.ID {
			return view, StateMismatch, fmt.Sprintf("配送が宣言する親Evidence %sは受領したEvidence %sと異なります", view.ParentEvidenceID, parent.ID)
		}
		view.ParentLinked = true
	}
	state, reason := coverageVerdict(view.Coverage)
	return view, state, reason
}

func decodeViewResult(raw json.RawMessage) (viewResult, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return viewResult{}, false
	}
	var result viewResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return viewResult{}, false
	}
	return result, true
}

func decodeCoverage(raw json.RawMessage) *Coverage {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var document struct {
		Status     string `json:"status"`
		Examined   int    `json:"examined"`
		Total      *int   `json:"total"`
		NextOffset *int   `json:"next_offset"`
		Exhausted  *bool  `json:"exhausted"`
	}
	if err := json.Unmarshal(raw, &document); err != nil {
		return nil
	}
	return &Coverage{
		Status:     document.Status,
		Examined:   document.Examined,
		Total:      document.Total,
		NextOffset: document.NextOffset,
		Exhausted:  document.Exhausted,
	}
}

// coverageVerdict reads the producer's own coverage declaration. A page that is
// not exhausted, an unavailable collection and an unknown coverage are all
// partial: the producer said it did not see everything, and a receipt that
// rounds that up to complete is lying about the delivery.
func coverageVerdict(coverage *Coverage) (State, string) {
	if coverage == nil {
		return StatePartial, "配送にcoverageの宣言がありません: どこまで届いたか判断できません"
	}
	switch coverage.Status {
	case "complete":
		return StateComplete, ""
	case "page":
		if coverage.Exhausted != nil && *coverage.Exhausted {
			return StateComplete, ""
		}
		return StatePartial, "page配送で、続きがあります（exhaustedではない）: 続きを取得するまで全体として扱わないでください"
	case "unavailable":
		return StatePartial, "coverage.status=unavailable: このcollectionは観測できません"
	case "unknown":
		return StatePartial, "coverage.status=unknown: 網羅できたか分かりません"
	}
	return StatePartial, fmt.Sprintf("coverage.status=%qは未知の値です: 既知の値ではないため完全だったと判断できません", coverage.Status)
}

func duplicateIdentities(records []viewRecord) []string {
	seen := map[string]int{}
	for _, record := range records {
		if record.Control != "" {
			continue
		}
		if identity := record.ObservationID; identity != "" {
			seen[identity]++
		}
	}
	return duplicated(seen)
}

func duplicateRowIdentities(rows []struct {
	ObservationID string `json:"observation_id"`
}) []string {
	seen := map[string]int{}
	for _, row := range rows {
		if row.ObservationID != "" {
			seen[row.ObservationID]++
		}
	}
	return duplicated(seen)
}

func duplicated(seen map[string]int) []string {
	var duplicates []string
	for identity, count := range seen {
		if count > 1 {
			duplicates = append(duplicates, identity)
		}
	}
	return duplicates
}
