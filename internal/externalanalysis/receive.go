package externalanalysis

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"tei/internal/sourceindex"
)

// Receive parses the saved output bytes of one external analysis command into a
// receipt.
//
// The bytes are measured before they are interpreted: Raw.Fingerprint covers
// exactly what arrived, so a later reader can tell whether the file on disk is
// still the file that was received. The producer's Evidence record is then
// separated from the CLI response wrapper that may have carried it, because the
// two are different things: the wrapper is the producer's own execution record,
// and the record is the Evidence a later request must embed.
//
// Receive never fails. Every outcome — including "these bytes are not what the
// contract declares" — is a receipt with a state, because a caller that cannot
// tell a missing Evidence from a malformed one will eventually treat one as the
// other.
func Receive(raw []byte, tool Tool, at time.Time) Receipt {
	receipt := Receipt{
		SchemaVersion: receiptSchemaVersion,
		Raw:           Raw{Bytes: len(raw), Fingerprint: fingerprint(raw)},
		Tool:          tool,
		AcquiredAt:    at.UTC().Format("2006-01-02T15:04:05Z07:00"),
		Promotion:     Promotion{},
	}
	if len(raw) == 0 {
		receipt.State = StateMissing
		receipt.Reason = "受領bytesが空です: external analysisの出力が取得されていません"
		return receipt
	}
	encoding, text, err := decodeText(raw)
	if err != nil {
		receipt.State = StateMismatch
		receipt.Reason = err.Error()
		return receipt
	}
	receipt.Raw.Encoding = encoding

	document, err := parseDocument(text)
	if err != nil {
		receipt.State = StateMismatch
		receipt.Reason = fmt.Sprintf("受領bytesはJSONではありません: %v", err)
		return receipt
	}
	if failure, ok := document.failure(); ok {
		receipt.State = StateFailed
		receipt.Failure = &failure
		receipt.Reason = fmt.Sprintf("external toolが失敗を報告しました: %s %s", failure.Code, failure.Message)
		return receipt
	}
	body, wrapped, err := document.evidenceBody()
	if err != nil {
		receipt.State = StateMismatch
		receipt.Reason = err.Error()
		return receipt
	}
	record, problems := parseEvidenceRecord(body)
	if len(problems.missing) > 0 {
		receipt.State = StateMissing
		receipt.Reason = "Evidence recordに必須の内容がありません: " + strings.Join(problems.missing, ", ")
		return receipt
	}
	if len(problems.invalid) > 0 {
		receipt.State = StateMismatch
		receipt.Reason = "Evidence recordの形式が契約と異なります: " + strings.Join(problems.invalid, ", ")
		return receipt
	}
	receipt.Evidence = record.evidence(wrapped)
	receipt.Source = record.source()
	receipt.body = body
	if wrapped {
		receipt.Envelope = document.envelope()
	}
	receipt.State = StateComplete
	if !tool.Present && receipt.State == StateComplete {
		// The bytes are what they are, but this environment could not have
		// produced them: the receipt says so instead of letting a reader take
		// a saved fixture for a fresh run. The caveat applies to a whole
		// receipt only — a delivery with a problem of its own keeps that
		// problem as its state.
		receipt.State = StateToolAbsent
		receipt.Reason = fmt.Sprintf("受領したbytesは完全ですが、external tool %s がこの環境にありません: 新規の取得はできておらず、保存済みbytesの検査結果です（%s）", tool.Name, tool.AbsentReason)
	}
	return receipt
}

// parseDocument decodes one saved document. It is deliberately a raw-message
// struct: the point of this pass is to tell the wrapper apart from the record
// without assuming either shape. The original text is kept, because a bare
// Evidence record must be carried forward as the producer wrote it.
type document struct {
	OK     *bool           `json:"ok"`
	Data   json.RawMessage `json:"data"`
	Meta   json.RawMessage `json:"meta"`
	Error  json.RawMessage `json:"error"`
	Result json.RawMessage `json:"result"`
	Rea    string          `json:"$rea"`
	Type   string          `json:"type"`
	// The diagnostic's own fields, for the flat shape the tool also writes:
	// {"error":"…","code":"…","category":"…","message":"…"} with the code
	// beside the message rather than inside it.
	Code     string `json:"code"`
	Category string `json:"category"`
	Message  string `json:"message"`

	text string
}

func parseDocument(text string) (*document, error) {
	var parsed document
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return nil, err
	}
	parsed.text = text
	return &parsed, nil
}

// failure reports the producer's own failure record, in either shape it saves
// one: a bare {"error": …} document, or a jsonl/v1 control record carrying
// {"type": "error", "result": …}. The diagnostic itself is either nested under
// error/result or written flat beside it; both were observed on 2026-11-11.
func (d *document) failure() (Failure, bool) {
	if d.Rea == "jsonl/v1" && d.Type == "error" {
		if failure, ok := decodeFailure(d.Result); ok {
			return failure, true
		}
		if failure, ok := d.flatFailure(); ok {
			return failure, true
		}
		return Failure{Code: "error-record", Message: "external toolがerror control recordを出しました"}, true
	}
	if len(d.Error) > 0 && string(d.Error) != "null" {
		if failure, ok := decodeFailure(d.Error); ok {
			return failure, true
		}
		if failure, ok := d.flatFailure(); ok {
			return failure, true
		}
		return Failure{Code: "error", Message: "external toolがerror documentを出しました"}, true
	}
	return Failure{}, false
}

// flatFailure reads a diagnostic whose fields sit beside the error marker
// instead of inside it.
func (d *document) flatFailure() (Failure, bool) {
	if d.Code == "" && d.Message == "" {
		return Failure{}, false
	}
	return Failure{Code: d.Code, Category: d.Category, Message: d.Message}, true
}

func decodeFailure(raw json.RawMessage) (Failure, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return Failure{}, false
	}
	var parsed struct {
		Code     string `json:"code"`
		Category string `json:"category"`
		Message  string `json:"message"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return Failure{}, false
	}
	if parsed.Code == "" && parsed.Message == "" {
		return Failure{}, false
	}
	return Failure{Code: parsed.Code, Category: parsed.Category, Message: parsed.Message}, true
}

// evidenceBody separates the Evidence record from the CLI response wrapper.
//
// The wrapper is {ok, data, meta}; the record is data. A caller that hands the
// wrapper to the tool as the Evidence body gets every required field reported
// missing and ok/data/meta reported unknown — that failure was observed against
// the real CLI on 2026-11-11, which is why the separation is a rule here and
// not a convention.
func (d *document) evidenceBody() (body []byte, wrapped bool, err error) {
	if d.OK == nil {
		return []byte(d.text), false, nil
	}
	if !*d.OK {
		return nil, false, fmt.Errorf("CLI応答のokがfalseです: %s", summarize(d.Data))
	}
	if len(d.Data) == 0 || string(d.Data) == "null" {
		return nil, false, fmt.Errorf("CLI応答にdataがありません: Evidence本体ではなく応答包装のみを受領した可能性があります")
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(d.Data, &object); err != nil {
		return nil, false, fmt.Errorf("CLI応答のdataがobjectではありません: %v", err)
	}
	return d.Data, true, nil
}

func summarize(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "(なし)"
	}
	text := string(raw)
	if len(text) > 200 {
		return text[:200] + "…"
	}
	return text
}

// evidenceRecord is the producer's Evidence record, read as raw messages so a
// missing field is distinguishable from a null one. The required set mirrors
// what the real CLI admits: a record omitting any of these is refused before a
// view is projected, and the refusal lists them by name.
type evidenceRecord struct {
	EvidenceID           json.RawMessage `json:"evidence_id"`
	Subject              json.RawMessage `json:"subject"`
	Provider             json.RawMessage `json:"provider"`
	AnalysisProfile      json.RawMessage `json:"analysis_profile"`
	PredicateType        json.RawMessage `json:"predicate_type"`
	Operation            json.RawMessage `json:"operation"`
	Parameters           json.RawMessage `json:"parameters"`
	RawResult            json.RawMessage `json:"raw_result"`
	NormalizedResult     json.RawMessage `json:"normalized_result"`
	Confidence           json.RawMessage `json:"confidence"`
	Authority            json.RawMessage `json:"authority"`
	Environment          json.RawMessage `json:"environment"`
	Limitations          json.RawMessage `json:"limitations"`
	RelationshipCoverage json.RawMessage `json:"relationship_coverage"`
	Locations            json.RawMessage `json:"locations"`
	EvidenceLinks        json.RawMessage `json:"evidence_links"`
}

// evidenceProblems separates "the content is not there" from "the content has
// a shape the contract does not declare". A caller acts on the two differently:
// the first says go and get it, the second says the bytes are not what they
// claim to be.
type evidenceProblems struct {
	missing []string
	invalid []string
}

var (
	confidenceValues = []string{"observed", "derived", "inferred"}
	authorityValues  = []string{"shipped-artifact", "controlled-replay", "historical-reference", "external-service", "analyst-inference"}
)

// evidenceMarkers are the fields that make a document recognizable as an
// Evidence record at all. A document carrying none of them is not a damaged
// record; it is a different document.
var evidenceMarkers = []string{"evidence_id", "operation", "normalized_result", "subject", "provider"}

func parseEvidenceRecord(body []byte) (*evidenceRecord, evidenceProblems) {
	var problems evidenceProblems
	var record evidenceRecord
	if err := json.Unmarshal(body, &record); err != nil {
		problems.invalid = append(problems.invalid, fmt.Sprintf("JSONとして読めません: %v", err))
		return &record, problems
	}
	fields := map[string]json.RawMessage{
		"evidence_id":           record.EvidenceID,
		"subject":               record.Subject,
		"provider":              record.Provider,
		"analysis_profile":      record.AnalysisProfile,
		"predicate_type":        record.PredicateType,
		"operation":             record.Operation,
		"parameters":            record.Parameters,
		"raw_result":            record.RawResult,
		"normalized_result":     record.NormalizedResult,
		"confidence":            record.Confidence,
		"authority":             record.Authority,
		"environment":           record.Environment,
		"limitations":           record.Limitations,
		"relationship_coverage": record.RelationshipCoverage,
		"locations":             record.Locations,
		"evidence_links":        record.EvidenceLinks,
	}
	markers := 0
	for _, name := range evidenceMarkers {
		if len(fields[name]) > 0 {
			markers++
		}
	}
	if markers == 0 {
		problems.invalid = append(problems.invalid, "Evidence recordの標識（evidence_id/operation/normalized_result/subject/provider）が一つもありません")
		return &record, problems
	}
	for name, value := range fields {
		if len(value) == 0 {
			problems.missing = append(problems.missing, name)
		}
	}
	if len(problems.missing) > 0 {
		return &record, problems
	}
	problems.invalid = append(problems.invalid, checkString("evidence_id", record.EvidenceID, true)...)
	problems.invalid = append(problems.invalid, checkString("predicate_type", record.PredicateType, false)...)
	problems.invalid = append(problems.invalid, checkString("operation", record.Operation, false)...)
	problems.invalid = append(problems.invalid, checkObject("subject", record.Subject)...)
	problems.invalid = append(problems.invalid, checkObject("provider", record.Provider)...)
	problems.invalid = append(problems.invalid, checkArray("limitations", record.Limitations)...)
	problems.invalid = append(problems.invalid, checkArray("locations", record.Locations)...)
	problems.invalid = append(problems.invalid, checkArray("evidence_links", record.EvidenceLinks)...)
	problems.invalid = append(problems.invalid, checkObjectOrNull("analysis_profile", record.AnalysisProfile)...)
	problems.invalid = append(problems.invalid, checkObjectOrNull("environment", record.Environment)...)
	problems.invalid = append(problems.invalid, checkObjectOrNull("relationship_coverage", record.RelationshipCoverage)...)
	problems.invalid = append(problems.invalid, checkEnum("confidence", record.Confidence, confidenceValues)...)
	problems.invalid = append(problems.invalid, checkEnum("authority", record.Authority, authorityValues)...)
	return &record, problems
}

func checkString(name string, raw json.RawMessage, nonEmpty bool) []string {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return []string{fmt.Sprintf("%sがstringではありません", name)}
	}
	if nonEmpty && strings.TrimSpace(value) == "" {
		return []string{fmt.Sprintf("%sが空です", name)}
	}
	return nil
}

func checkObject(name string, raw json.RawMessage) []string {
	var value map[string]json.RawMessage
	if err := json.Unmarshal(raw, &value); err != nil {
		return []string{fmt.Sprintf("%sがobjectではありません", name)}
	}
	return nil
}

func checkObjectOrNull(name string, raw json.RawMessage) []string {
	if string(raw) == "null" {
		return nil
	}
	return checkObject(name, raw)
}

func checkArray(name string, raw json.RawMessage) []string {
	var value []json.RawMessage
	if err := json.Unmarshal(raw, &value); err != nil {
		return []string{fmt.Sprintf("%sがarrayではありません", name)}
	}
	return nil
}

func checkEnum(name string, raw json.RawMessage, allowed []string) []string {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return []string{fmt.Sprintf("%sがstringではありません", name)}
	}
	for _, candidate := range allowed {
		if value == candidate {
			return nil
		}
	}
	return []string{fmt.Sprintf("%s=%qは許容値%sのいずれでもありません", name, value, strings.Join(allowed, "|"))}
}

func (r *evidenceRecord) evidence(wrapped bool) *Evidence {
	evidence := &Evidence{
		ID:         rawString(r.EvidenceID),
		Operation:  rawString(r.Operation),
		Predicate:  rawString(r.PredicateType),
		Confidence: rawString(r.Confidence),
		Authority:  rawString(r.Authority),
		Envelope:   wrapped,
	}
	var provider struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	_ = json.Unmarshal(r.Provider, &provider)
	evidence.Provider = Provider{ID: provider.ID, Name: provider.Name, Version: provider.Version}
	var links []string
	_ = json.Unmarshal(r.EvidenceLinks, &links)
	evidence.Links = links
	return evidence
}

func (r *evidenceRecord) source() *Source {
	source := &Source{}
	var subject struct {
		Name   string `json:"name"`
		Format string `json:"format"`
		Path   string `json:"local_path"`
		Digest struct {
			SHA256 string `json:"sha256"`
		} `json:"digest"`
	}
	if err := json.Unmarshal(r.Subject, &subject); err == nil {
		source.Subject = subject.Name
		source.Format = subject.Format
		source.Path = subject.Path
		if subject.Digest.SHA256 != "" {
			source.Revision = "sha256:" + subject.Digest.SHA256
		}
	}
	var locations []struct {
		Kind        string `json:"kind"`
		Path        string `json:"path"`
		StartOffset *int   `json:"start_offset"`
		EndOffset   *int   `json:"end_offset"`
	}
	if err := json.Unmarshal(r.Locations, &locations); err == nil {
		for _, location := range locations {
			position := Position{Kind: location.Kind, Path: location.Path}
			if location.StartOffset != nil && location.EndOffset != nil {
				position.Range = &Range{
					Start: sourceindex.Position{Offset: *location.StartOffset},
					End:   sourceindex.Position{Offset: *location.EndOffset},
				}
			}
			source.Positions = append(source.Positions, position)
		}
	}
	return source
}

func (d *document) envelope() *Envelope {
	if len(d.Meta) == 0 || string(d.Meta) == "null" {
		return nil
	}
	var meta struct {
		Command  string `json:"command"`
		Duration string `json:"duration"`
	}
	if err := json.Unmarshal(d.Meta, &meta); err != nil {
		return nil
	}
	if meta.Command == "" && meta.Duration == "" {
		return nil
	}
	return &Envelope{Command: meta.Command, Duration: meta.Duration}
}

func rawString(raw json.RawMessage) string {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return value
}

// decodeText decodes the received bytes into text and names the encoding it
// used. The name is part of the receipt because the same analysis saved on two
// hosts does not produce the same bytes: the real CLI wrote UTF-16LE with a BOM
// on Windows and UTF-8 on Linux for the same command (observed 2026-11-11).
//
// An encoding without a BOM is refused rather than guessed. A UTF-16 stream
// without its BOM is indistinguishable from damaged UTF-8 by anything cheaper
// than a guess, and a receipt built on a guess is not a receipt.
func decodeText(raw []byte) (encoding, text string, err error) {
	switch {
	case len(raw) >= 3 && raw[0] == 0xEF && raw[1] == 0xBB && raw[2] == 0xBF:
		return "utf-8", string(raw[3:]), nil
	case len(raw) >= 2 && raw[0] == 0xFF && raw[1] == 0xFE:
		if len(raw)%2 != 0 {
			return "", "", fmt.Errorf("UTF-16LEの宣言(BOM)はありますが、bytes数が奇数です: 受領が途中で切れています")
		}
		return "utf-16le", decodeUTF16(raw[2:], true), nil
	case len(raw) >= 2 && raw[0] == 0xFE && raw[1] == 0xFF:
		if len(raw)%2 != 0 {
			return "", "", fmt.Errorf("UTF-16BEの宣言(BOM)はありますが、bytes数が奇数です: 受領が途中で切れています")
		}
		return "utf-16be", decodeUTF16(raw[2:], false), nil
	}
	if !utf8.Valid(raw) {
		if hasNullEarly(raw) {
			return "", "", fmt.Errorf("BOMのないUTF-16と思われるbytesを推測して解釈しません: 受領したままのbytesを再取得してください")
		}
		return "", "", fmt.Errorf("受領bytesは有効なUTF-8ではなく、BOMもありません")
	}
	return "utf-8", string(raw), nil
}

func hasNullEarly(raw []byte) bool {
	limit := len(raw)
	if limit > 64 {
		limit = 64
	}
	for _, b := range raw[:limit] {
		if b == 0 {
			return true
		}
	}
	return false
}

func decodeUTF16(raw []byte, littleEndian bool) string {
	units := make([]uint16, 0, len(raw)/2)
	for i := 0; i+1 < len(raw); i += 2 {
		if littleEndian {
			units = append(units, uint16(raw[i])|uint16(raw[i+1])<<8)
		} else {
			units = append(units, uint16(raw[i])<<8|uint16(raw[i+1]))
		}
	}
	return string(utf16.Decode(units))
}
