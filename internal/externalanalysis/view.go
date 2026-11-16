package externalanalysis

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf16"
)

// FramingFor returns the view framing contract for an observed tool version.
func FramingFor(version string) (Framing, bool) {
	switch version {
	case "6.7.0":
		return Framing{Records: 1, Control: "none", Kind: "jsonl"}, true
	case "6.8.0":
		return Framing{Records: 7, Control: "metadata", Kind: "jsonl", RequiresTerminal: true}, true
	}
	return Framing{}, false
}

// Framing is the expected view delivery shape for one tool version.
type Framing struct {
	Records          int
	Control          string
	Kind             string
	RequiresTerminal bool
}

// WithView inspects one saved view delivery against the framing the recorded
// producer version produces.
func (r Receipt) WithView(raw []byte, framing Framing) Receipt {
	if len(raw) == 0 {
		return r.withVerdict(StateMissing, "view deliveryのbytesが空です")
	}
	text := decodeTextLossy(raw)
	records := strings.Split(text, "\n")
	var trimmed []string
	for _, record := range records {
		if strings.TrimSpace(record) != "" {
			trimmed = append(trimmed, record)
		}
	}
	if len(trimmed) == 0 {
		return r.withVerdict(StateMissing, "view deliveryにrecordがありません")
	}

	view := View{
		Encoding:   encodingName(raw),
		Framing:    framing.Kind,
		Records:    len(trimmed),
		Collection: []string{"normalized_result,rows"},
	}

	last := trimmed[len(trimmed)-1]
	if strings.HasPrefix(last, "$rea=") {
		view.Control = "metadata"
		trimmed = trimmed[:len(trimmed)-1]
		view.Records = len(trimmed)
		if framing.RequiresTerminal {
			if view.Records != framing.Records {
				view.EvidenceID = firstEvidenceID(trimmed)
				r.View = &view
				return r.withVerdict(StatePartial, fmt.Sprintf("終端recordがありますが、data record数=%dで期待=%dと異なります", view.Records, framing.Records))
			}
			meta, err := parseTerminal(last)
			if err != nil {
				view.EvidenceID = firstEvidenceID(trimmed)
				r.View = &view
				return r.withVerdict(StateMismatch, "終端recordを解析できません: "+err.Error())
			}
			view.Emitted = meta.Emitted
			view.SourceCount = meta.SourceCount
		}
		if view.Emitted == 0 && view.SourceCount == 0 {
			view.Emitted = view.Records
		}
	} else if strings.Contains(last, `"$rea"`) {
		// Current REA JSONL uses a structured metadata object instead of the
		// legacy "$rea=..." line. Counts belong to the actual delivery, not
		// to the historical fixture's fixed number of records.
		view.Control = "metadata"
		trimmed = trimmed[:len(trimmed)-1]
		view.Records = len(trimmed)
		meta, err := parseJSONObjectTerminal(last)
		if err != nil {
			r.View = &view
			return r.withVerdict(StateMismatch, "JSONL終端metadataの形式が不正です: "+err.Error())
		}
		view.Collection = meta.Records.Path
		view.Emitted = *meta.Records.Emitted
		if meta.Records.Source != nil {
			view.SourceCount = *meta.Records.Source
		}
		view.EvidenceID = meta.Context.EvidenceID
		if !framing.RequiresTerminal {
			r.View = &view
			return r.withVerdict(StateMismatch, "このProvider版はJSONL終端metadataを宣言していません")
		}
		for _, line := range trimmed {
			if !json.Valid([]byte(line)) {
				r.View = &view
				return r.withVerdict(StateMismatch, "JSONLのdata recordがJSONとして不正です")
			}
		}
		if view.Records != view.Emitted {
			r.View = &view
			return r.withVerdict(StatePartial, fmt.Sprintf("JSONL data record数=%dがmetadata emitted=%dと一致しません", view.Records, view.Emitted))
		}
	} else {
		view.Control = "none"
		if framing.RequiresTerminal {
			view.EvidenceID = firstEvidenceID(trimmed)
			r.View = &view
			return r.withVerdict(StatePartial, "終端recordがありません: 受領が途中で切れている可能性があります")
		}
		if view.Records != framing.Records {
			view.EvidenceID = firstEvidenceID(trimmed)
			r.View = &view
			return r.withVerdict(StatePartial, fmt.Sprintf("data record数=%dで期待=%dと異なります", view.Records, framing.Records))
		}
	}

	if view.EvidenceID == "" {
		view.EvidenceID = firstEvidenceID(trimmed)
	}
	duplicates := findDuplicates(trimmed)
	if len(duplicates) > 0 {
		return r.withVerdict(StateMismatch, "重複したEvidence IDがあります: "+strings.Join(duplicates, ", "))
	}
	view.Duplicates = duplicates

	r.View = &view
	return r
}

// jsonlObjectTerminal is the REA 6.8 JSONL control record. The records
// collection is a bounded projection; its emitted count is not a global
// completeness claim about the analyzed application.
type jsonlObjectTerminal struct {
	Protocol string `json:"$rea"`
	Type     string `json:"type"`
	Records  struct {
		Path    []string `json:"path"`
		Emitted *int     `json:"emitted"`
		Source  *int     `json:"source"`
	} `json:"records"`
	Context struct {
		EvidenceID string `json:"evidence_id"`
	} `json:"context"`
}

func parseJSONObjectTerminal(line string) (jsonlObjectTerminal, error) {
	var meta jsonlObjectTerminal
	if err := json.Unmarshal([]byte(line), &meta); err != nil {
		return meta, err
	}
	if meta.Protocol != "jsonl/v1" || meta.Type != "metadata" {
		return meta, fmt.Errorf("protocol=%q type=%q", meta.Protocol, meta.Type)
	}
	if len(meta.Records.Path) == 0 {
		return meta, fmt.Errorf("records.pathがありません")
	}
	if meta.Records.Emitted == nil || *meta.Records.Emitted < 0 {
		return meta, fmt.Errorf("records.emittedがありません、または負数です")
	}
	if meta.Records.Source != nil &&
		(*meta.Records.Source < 0 || *meta.Records.Source < *meta.Records.Emitted) {
		return meta, fmt.Errorf("records.sourceはemitted以上の非負整数である必要があります")
	}
	return meta, nil
}

type terminal struct {
	Emitted     int `json:"emitted"`
	SourceCount int `json:"source_count"`
}

func parseTerminal(line string) (terminal, error) {
	fields := strings.Split(line, ",")
	var t terminal
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if strings.HasPrefix(field, "emitted=") {
			fmt.Sscanf(field, "emitted=%d", &t.Emitted)
		}
		if strings.HasPrefix(field, "source_count=") {
			fmt.Sscanf(field, "source_count=%d", &t.SourceCount)
		}
	}
	if t.Emitted == 0 && t.SourceCount == 0 {
		return t, fmt.Errorf("emitted/source_countを読み取れません")
	}
	return t, nil
}

func firstEvidenceID(records []string) string {
	for _, record := range records {
		if id := evidenceID(record); id != "" {
			return id
		}
	}
	return ""
}

func findDuplicates(records []string) []string {
	seen := map[string]int{}
	var duplicates []string
	for _, record := range records {
		key := strings.TrimSpace(record)
		if key == "" {
			continue
		}
		seen[key]++
		if seen[key] == 2 {
			duplicates = append(duplicates, key)
		}
	}
	return duplicates
}

func evidenceID(record string) string {
	var value map[string]json.RawMessage
	if err := json.Unmarshal([]byte(record), &value); err != nil {
		return ""
	}
	raw, ok := value["evidence_id"]
	if !ok {
		return ""
	}
	var id string
	if err := json.Unmarshal(raw, &id); err != nil {
		return ""
	}
	return id
}

func encodingName(raw []byte) string {
	if len(raw) >= 3 && raw[0] == 0xEF && raw[1] == 0xBB && raw[2] == 0xBF {
		return "utf-8"
	}
	if len(raw) >= 2 && raw[0] == 0xFF && raw[1] == 0xFE {
		return "utf-16le"
	}
	if len(raw) >= 2 && raw[0] == 0xFE && raw[1] == 0xFF {
		return "utf-16be"
	}
	return "utf-8"
}

func decodeTextLossy(raw []byte) string {
	if len(raw) >= 3 && raw[0] == 0xEF && raw[1] == 0xBB && raw[2] == 0xBF {
		return string(raw[3:])
	}
	if len(raw) >= 2 && raw[0] == 0xFF && raw[1] == 0xFE {
		units := make([]uint16, 0, (len(raw)-2)/2)
		for i := 2; i+1 < len(raw); i += 2 {
			units = append(units, uint16(raw[i])|uint16(raw[i+1])<<8)
		}
		return string(utf16.Decode(units))
	}
	if len(raw) >= 2 && raw[0] == 0xFE && raw[1] == 0xFF {
		units := make([]uint16, 0, (len(raw)-2)/2)
		for i := 2; i+1 < len(raw); i += 2 {
			units = append(units, uint16(raw[i])<<8|uint16(raw[i+1]))
		}
		return string(utf16.Decode(units))
	}
	return string(raw)
}
