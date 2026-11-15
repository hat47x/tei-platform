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

// Receive reads one saved external analysis output and returns a Receipt that
// records what TEI observed about the bytes. It does not invoke the external
// tool; the caller records the environment separately.
func Receive(raw []byte, tool Tool, acquiredAt time.Time) Receipt {
	receipt := Receipt{
		SchemaVersion: receiptSchemaVersion,
		AcquiredAt:    acquiredAt.UTC().Format(time.RFC3339),
		Tool:          tool,
		Promotion:     Promotion{},
	}
	if len(raw) == 0 {
		receipt.State = StateMissing
		receipt.Reason = "受領したbytesが空です"
		return receipt
	}
	encoding, text, err := decodeText(raw)
	if err != nil {
		receipt.State = StateMismatch
		receipt.Reason = err.Error()
		return receipt
	}
	receipt.Raw = Raw{
		Bytes:       len(raw),
		Encoding:    encoding,
		Fingerprint: fingerprint(raw),
		LineEnding:  lineEnding(raw),
	}

	document, problems, err := parseDocument([]byte(text))
	if err != nil {
		receipt.State = StateMismatch
		receipt.Reason = err.Error()
		return receipt
	}
	if len(problems.missing) > 0 {
		receipt.State = StateMissing
		receipt.Reason = "Evidenceの必須フィールドが欠けています: " + strings.Join(problems.missing, ", ")
		return receipt
	}
	if len(problems.invalid) > 0 {
		receipt.State = StateMismatch
		receipt.Reason = "Evidenceの形式が契約と異なります: " + strings.Join(problems.invalid, "; ")
		return receipt
	}

	// A producer failure is a typed negative receipt, not a partial
	// application graph. It has no Evidence body and grants no promotion.
	if document.Failure != nil {
		receipt.Failure = document.Failure
		receipt.State = StateFailed
		receipt.Reason = failureReason(document.Failure)
		return receipt
	}
	receipt.body = document.Raw
	receipt.Evidence = document.Evidence.evidence(document.Envelope)
	receipt.Source = document.Evidence.source()
	receipt.Envelope = document.envelope()

	switch {
	case len(document.Evidence.Confidence) == 0 && len(document.Evidence.Authority) == 0:
		receipt.State = StatePartial
		receipt.Reason = "Evidenceにconfidence/authorityがなく、解析の状態を確認できません"
	default:
		receipt.State = StateComplete
		receipt.Reason = "bytesを復号し、Evidenceの形式を確認しました"
	}

	if receipt.Evidence != nil {
		receipt.Reason = fmt.Sprintf("%s (evidence_id=%s)", receipt.Reason, receipt.Evidence.ID)
	}
	return receipt
}

type document struct {
	Meta     json.RawMessage `json:"meta"`
	Evidence *evidenceRecord
	Failure  *Failure        `json:"failure"`
	Envelope bool
	Raw      []byte
}

func parseDocument(text []byte) (*document, evidenceProblems, error) {
	// REA 6.8 JSONL errors are one "$rea":"jsonl/v1" control
	// document. Recognize only an explicit failure marker and a typed
	// error code; never reinterpret arbitrary JSON or a failed Evidence
	// projection as a successful analysis.
	var control struct {
		Protocol string          `json:"$rea"`
		Type     string          `json:"type"`
		OK       *bool           `json:"ok"`
		Result   json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(text, &control); err == nil && control.Protocol != "" {
		if control.Protocol != "jsonl/v1" || control.Type != "error" ||
			control.OK == nil || *control.OK {
			return &document{}, evidenceProblems{invalid: []string{"REA JSONL controlは明示された失敗応答ではありません"}}, nil
		}
		failure, err := decodeProducerFailure(control.Result)
		if err != nil {
			return &document{}, evidenceProblems{invalid: []string{"REA JSONL失敗応答: " + err.Error()}}, nil
		}
		return &document{Failure: &failure, Raw: text}, evidenceProblems{}, nil
	}

	var envelope struct {
		OK    *bool           `json:"ok"`
		Data  json.RawMessage `json:"data"`
		Meta  json.RawMessage `json:"meta"`
		Error json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(text, &envelope); err == nil && envelope.OK != nil && !*envelope.OK && len(envelope.Error) > 0 {
		failure, err := decodeProducerFailure(envelope.Error)
		if err != nil {
			return &document{}, evidenceProblems{invalid: []string{"REA JSON失敗応答: " + err.Error()}}, nil
		}
		return &document{Failure: &failure, Raw: text}, evidenceProblems{}, nil
	}
	if err := json.Unmarshal(text, &envelope); err == nil && envelope.OK != nil && *envelope.OK && len(envelope.Data) > 0 {
		record, problems := parseEvidenceRecord(envelope.Data)
		return &document{
			Meta:     envelope.Meta,
			Evidence: record,
			Raw:      envelope.Data,
			Envelope: true,
		}, problems, nil
	}
	record, problems := parseEvidenceRecord(text)
	return &document{Evidence: record, Raw: text}, problems, nil
}

// decodeProducerFailure handles a direct typed error and the full-output
// {ok:false,error:{...}} shape wrapped by REA JSONL's result. Missing or
// non-string codes never become a StateFailed receipt.
func decodeProducerFailure(raw json.RawMessage) (Failure, error) {
	var payload struct {
		Code     string          `json:"code"`
		Category string          `json:"category"`
		Message  string          `json:"message"`
		Error    json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return Failure{}, err
	}
	if len(payload.Error) > 0 {
		return decodeProducerFailure(payload.Error)
	}
	if strings.TrimSpace(payload.Code) == "" {
		return Failure{}, fmt.Errorf("型付きerror.codeがありません")
	}
	return Failure{Code: payload.Code, Category: payload.Category, Message: payload.Message}, nil
}

type evidenceRecord struct {
	EvidenceID           json.RawMessage `json:"evidence_id"`
	AnalysisProfile      json.RawMessage `json:"analysis_profile"`
	PredicateType        json.RawMessage `json:"predicate_type"`
	Operation            json.RawMessage `json:"operation"`
	Parameters           json.RawMessage `json:"parameters"`
	RawResult            json.RawMessage `json:"raw_result"`
	NormalizedResult     json.RawMessage `json:"normalized_result"`
	Confidence           json.RawMessage `json:"confidence"`
	Authority            json.RawMessage `json:"authority"`
	Environment          json.RawMessage `json:"environment"`
	Subject              json.RawMessage `json:"subject"`
	Provider             json.RawMessage `json:"provider"`
	Limitations          json.RawMessage `json:"limitations"`
	RelationshipCoverage json.RawMessage `json:"relationship_coverage"`
	Locations            json.RawMessage `json:"locations"`
	EvidenceLinks        json.RawMessage `json:"evidence_links"`
}

type evidenceProblems struct {
	missing []string
	invalid []string
}

var evidenceMarkers = []string{"evidence_id", "operation", "normalized_result", "subject", "provider"}

func parseEvidenceRecord(body []byte) (*evidenceRecord, evidenceProblems) {
	var problems evidenceProblems
	var record evidenceRecord
	if err := json.Unmarshal(body, &record); err != nil {
		problems.invalid = append(problems.invalid, fmt.Sprintf("JSONとして読めません: %v", err))
		return &record, problems
	}
	fields := map[string]json.RawMessage{
		"evidence_id":           json.RawMessage(record.EvidenceID),
		"subject":               record.Subject,
		"provider":              record.Provider,
		"analysis_profile":      record.AnalysisProfile,
		"predicate_type":        json.RawMessage(record.PredicateType),
		"operation":             json.RawMessage(record.Operation),
		"parameters":            record.Parameters,
		"raw_result":            record.RawResult,
		"normalized_result":     record.NormalizedResult,
		"confidence":            json.RawMessage(record.Confidence),
		"authority":             json.RawMessage(record.Authority),
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
	problems.invalid = append(problems.invalid, checkString("evidence_id", json.RawMessage(record.EvidenceID), true)...)
	problems.invalid = append(problems.invalid, checkString("predicate_type", json.RawMessage(record.PredicateType), false)...)
	problems.invalid = append(problems.invalid, checkString("operation", json.RawMessage(record.Operation), false)...)
	problems.invalid = append(problems.invalid, checkObject("subject", record.Subject)...)
	problems.invalid = append(problems.invalid, checkObject("provider", record.Provider)...)
	problems.invalid = append(problems.invalid, checkArray("limitations", record.Limitations)...)
	problems.invalid = append(problems.invalid, checkArray("locations", record.Locations)...)
	problems.invalid = append(problems.invalid, checkArray("evidence_links", record.EvidenceLinks)...)
	problems.invalid = append(problems.invalid, checkObjectOrNull("analysis_profile", record.AnalysisProfile)...)
	problems.invalid = append(problems.invalid, checkObjectOrNull("environment", record.Environment)...)
	problems.invalid = append(problems.invalid, checkObjectOrNull("relationship_coverage", record.RelationshipCoverage)...)
	problems.invalid = append(problems.invalid, checkEnum("confidence", json.RawMessage(record.Confidence), []string{"observed", "derived", "inferred"})...)
	problems.invalid = append(problems.invalid, checkEnum("authority", json.RawMessage(record.Authority), []string{"shipped-artifact", "controlled-replay", "historical-reference", "external-service", "analyst-inference"})...)
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

func failureReason(failure *Failure) string {
	if failure == nil {
		return ""
	}
	parts := []string{failure.Code}
	if failure.Category != "" {
		parts = append(parts, failure.Category)
	}
	if failure.Message != "" {
		parts = append(parts, failure.Message)
	}
	return strings.Join(parts, ": ")
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

func lineEnding(raw []byte) string {
	if strings.Contains(string(raw), "\r\n") {
		return "CRLF"
	}
	if strings.Contains(string(raw), "\n") {
		return "LF"
	}
	return "none"
}
