package externalanalysis

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ViewRequest is the view selector TEI asks the external tool to project from
// one Evidence.
//
// The intake does not interpret the selector beyond requiring a kind: the
// tool's view families belong to the tool. A selector TEI cannot read is still
// a selector the tool can refuse, and a refusal is a receipt, not a crash.
type ViewRequest struct {
	Kind     string
	Selector json.RawMessage
}

// observedSourceKinds records which source.kind values each tool version was
// observed to accept. The values come from the tool's own refusal, which names
// the accepted set in details.issues when a wrong kind is sent; they were read
// off real runs on 2026-11-11 (#3394). A version that is not in this table is
// unknown, and an unknown version is not guessed at: the caller declares the
// set or the request is not derived.
var observedSourceKinds = map[string][]string{
	"6.7.0": {"retained-evidence", "inline"},
	"6.8.0": {"retained-evidence", "inline", "evidence-file"},
}

// AcceptedSourceKinds returns the source.kind values the recorded tool version
// accepts, and whether that version is one this package has observed.
//
// The version difference is not cosmetic: 6.7.0 refuses evidence-file with
// invalid_input, and 6.8.0 accepts it. A coordination mechanism that hard-codes
// one set breaks the moment the tool is upgraded, which is why the set travels
// with the receipt instead of living in the caller.
func AcceptedSourceKinds(version string) ([]string, bool) {
	kinds, ok := observedSourceKinds[version]
	if !ok {
		return nil, false
	}
	return append([]string(nil), kinds...), true
}

// DeriveViewRequest builds the request bytes TEI hands to the tool for one
// Evidence, and returns them with nothing else.
//
// The Evidence is embedded verbatim: not re-serialized, and not compacted.
// Re-serializing it would produce a different byte sequence for the same record
// — key order is the producer's, not ours — and a request whose bytes cannot be
// traced back to the received ones cannot be re-read later.
//
// The response envelope is refused here as well as at the tool. Handing the
// tool {ok, data, meta} as the Evidence body was observed to fail with every
// required field reported missing and ok/data/meta reported unknown; deriving
// it that way is now impossible rather than merely discouraged.
func DeriveViewRequest(body []byte, view ViewRequest, acceptedKinds []string) ([]byte, error) {
	if len(body) == 0 {
		return nil, fmt.Errorf("Evidence本体のbytesが空です: 受領したEvidenceをそのまま渡してください")
	}
	// The producer may have written the Evidence in an encoding this host does
	// not use; the request is UTF-8 JSON, so the Evidence is embedded as its
	// decoded text. It is never re-serialized: key order belongs to the
	// producer, and a request that cannot be traced back to the received bytes
	// cannot be re-read later.
	_, text, err := decodeText(body)
	if err != nil {
		return nil, err
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal([]byte(text), &object); err != nil {
		return nil, fmt.Errorf("Evidence本体がJSON objectではありません: %v", err)
	}
	if _, hasOK := object["ok"]; hasOK {
		if _, hasData := object["data"]; hasData {
			return nil, fmt.Errorf("CLI応答の包装({ok,data,meta})をEvidence本体として渡そうとしています: dataの中身を渡してください")
		}
	}
	if err := view.validate(); err != nil {
		return nil, err
	}
	kind, err := derivableSourceKind(acceptedKinds)
	if err != nil {
		return nil, err
	}
	// The request is assembled by hand rather than marshaled, because
	// json.Marshal compacts a RawMessage: the embedded Evidence would no
	// longer be the bytes that were received whenever the producer wrote
	// insignificant whitespace.
	request := make([]byte, 0, len(text)+len(view.Selector)+64)
	request = append(request, `{"source":{"kind":"`...)
	request = append(request, kind...)
	request = append(request, `","evidence":`...)
	request = append(request, text...)
	request = append(request, `},"view":`...)
	request = append(request, view.Selector...)
	request = append(request, '}')
	return request, nil
}

// derivableSourceKind picks the one kind the intake derives: inline. The
// Evidence travels inside the request, so the request is self-contained and
// its digest covers the Evidence the tool will read. evidence-file would point
// at bytes outside the request's own digest, and a saved file is not always
// readable as the tool expects: a Windows-captured UTF-16LE file was refused
// with evidence_integrity_mismatch on 2026-11-11.
func derivableSourceKind(acceptedKinds []string) (string, error) {
	if len(acceptedKinds) == 0 {
		return "", fmt.Errorf("source.kindの許容値が宣言されていません: 実toolのversionを宣言してから派生入力を組み立ててください")
	}
	for _, kind := range acceptedKinds {
		if kind == sourceKindInline {
			return sourceKindInline, nil
		}
	}
	return "", fmt.Errorf("記録されたtoolのversionはsource.kind=%qを受け付けません: 受諾済みの値は%sです", sourceKindInline, strings.Join(acceptedKinds, "|"))
}

func (v ViewRequest) validate() error {
	if strings.TrimSpace(v.Kind) == "" {
		return fmt.Errorf("viewのkindが空です: 工具へ何を投影するかを宣言してください")
	}
	if len(v.Selector) == 0 {
		return fmt.Errorf("viewのselectorが空です")
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(v.Selector, &object); err != nil {
		return fmt.Errorf("viewのselectorがJSON objectではありません: %v", err)
	}
	kind := rawString(object["kind"])
	if kind == "" {
		return fmt.Errorf("viewのselectorにkindがありません")
	}
	if kind != v.Kind {
		return fmt.Errorf("viewのselectorのkind=%qが宣言%sと異なります", kind, v.Kind)
	}
	return nil
}

const (
	sourceKindInline           = "inline"
	sourceKindRetainedEvidence = "retained-evidence"
	sourceKindEvidenceFile     = "evidence-file"
)

// WithRequest validates one derived request against the Evidence this receipt
// holds, and records what the request would actually hand the tool.
//
// This is the check that keeps the coordination loop honest in both
// directions: a request that embeds a different Evidence than the one received,
// or that embeds the response envelope instead of the Evidence body, is a
// mismatch — not a request with a quirk.
func (r Receipt) WithRequest(raw []byte, acceptedKinds []string) Receipt {
	if len(raw) == 0 {
		return r.withVerdict(StateMissing, "派生入力のbytesが空です")
	}
	_, text, err := decodeText(raw)
	if err != nil {
		return r.withVerdict(StateMismatch, err.Error())
	}
	var request struct {
		Source json.RawMessage `json:"source"`
		View   json.RawMessage `json:"view"`
	}
	if err := json.Unmarshal([]byte(text), &request); err != nil {
		return r.withVerdict(StateMismatch, fmt.Sprintf("派生入力はJSONではありません: %v", err))
	}
	if len(request.Source) == 0 || len(request.View) == 0 {
		return r.withVerdict(StateMismatch, "派生入力にsourceまたはviewがありません")
	}
	var source struct {
		Kind       string          `json:"kind"`
		Evidence   json.RawMessage `json:"evidence"`
		EvidenceID string          `json:"evidence_id"`
		Path       string          `json:"path"`
	}
	if err := json.Unmarshal(request.Source, &source); err != nil {
		return r.withVerdict(StateMismatch, fmt.Sprintf("派生入力のsourceがobjectではありません: %v", err))
	}
	if strings.TrimSpace(source.Kind) == "" {
		return r.withVerdict(StateMismatch, "派生入力のsource.kindが空です")
	}
	if len(acceptedKinds) == 0 {
		return r.withVerdict(StateMismatch, "source.kindの許容値が宣言されていないため、派生入力のsource.kindを検証できません")
	}
	if !containsString(acceptedKinds, source.Kind) {
		return r.withVerdict(StateMismatch, fmt.Sprintf("派生入力のsource.kind=%qは記録されたtoolのversionが受け付けません: 受諾済みの値は%sです", source.Kind, strings.Join(acceptedKinds, "|")))
	}
	var viewObject map[string]json.RawMessage
	if err := json.Unmarshal(request.View, &viewObject); err != nil {
		return r.withVerdict(StateMismatch, fmt.Sprintf("派生入力のviewがobjectではありません: %v", err))
	}
	if rawString(viewObject["kind"]) == "" {
		return r.withVerdict(StateMismatch, "派生入力のviewにkindがありません")
	}
	derived := &Derived{SourceKind: source.Kind, AcceptedKinds: append([]string(nil), acceptedKinds...)}
	switch source.Kind {
	case sourceKindEvidenceFile:
		if strings.TrimSpace(source.Path) == "" {
			return r.withVerdict(StateMismatch, "source.kind=evidence-fileの派生入力にpathがありません")
		}
		derived.ByReference = source.Path
	case sourceKindRetainedEvidence:
		if strings.TrimSpace(source.EvidenceID) == "" {
			return r.withVerdict(StateMismatch, "source.kind=retained-evidenceの派生入力にevidence_idがありません")
		}
		if r.Evidence != nil && source.EvidenceID != r.Evidence.ID {
			return r.withVerdict(StateMismatch, fmt.Sprintf("派生入力が指すEvidence %sは受領したEvidence %sと異なります", source.EvidenceID, r.Evidence.ID))
		}
	case sourceKindInline:
		embedded, problems := parseEvidenceRecord(source.Evidence)
		if len(problems.missing) > 0 {
			return r.withVerdict(StateMissing, "派生入力が埋め込むEvidenceに必須の内容がありません: "+strings.Join(problems.missing, ", "))
		}
		if len(problems.invalid) > 0 {
			return r.withVerdict(StateMismatch, "派生入力が埋め込むEvidenceの形式が契約と異なります: "+strings.Join(problems.invalid, ", "))
		}
		if r.Evidence != nil && embedded.ID() != r.Evidence.ID {
			return r.withVerdict(StateMismatch, fmt.Sprintf("派生入力が埋め込むEvidence %sは受領したEvidence %sと異なります", embedded.ID(), r.Evidence.ID))
		}
	default:
		return r.withVerdict(StateMismatch, fmt.Sprintf("source.kind=%qはこの切片では扱いません", source.Kind))
	}
	derived.Bytes = len(raw)
	derived.Fingerprint = fingerprint(raw)
	r.Derived = derived
	return r
}

func (r *evidenceRecord) ID() string { return rawString(r.EvidenceID) }

// withVerdict records a verdict about a derived request without letting it
// launder an earlier, worse verdict.
func (r Receipt) withVerdict(state State, reason string) Receipt {
	r.State, r.Reason = merge(r.State, r.Reason, state, reason)
	return r
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
