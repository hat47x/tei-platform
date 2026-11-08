// Package harnesswire summarizes HTTP request bodies captured by a mock
// provider endpoint standing in for an external AI harness.
//
// It reports what a request carried, not what any of it means for a Product
// Owner: a captured field does not prove that the disclosure was permitted,
// and a missing field does not prove that no other path sends it. The caller
// records the harness, profile and fixtures separately.
package harnesswire

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// SessionLogField is the harness-side request extension that carries Session
// events outside the model input. Its name is fixed by the harness, so it is
// recorded here as an observed field name rather than as a TEI contract.
const SessionLogField = "dsh_session_log"

// SessionLog summarizes the observed session-log extension.
type SessionLog struct {
	FieldBytes      int            `json:"fieldBytes"`
	Keys            []string       `json:"keys"`
	AfterSeq        *int64         `json:"afterSeq"`
	ThroughSeq      *int64         `json:"throughSeq"`
	EventCount      int            `json:"eventCount"`
	EventTypes      map[string]int `json:"eventTypes"`
	MarkerHits      map[string]int `json:"markerHits,omitempty"`
	Unserializable  bool           `json:"unserializable,omitempty"`
	UnparseableBody bool           `json:"unparseableBody,omitempty"`
}

// Summary is one captured request body.
type Summary struct {
	Bytes           int            `json:"bytes"`
	SHA256          string         `json:"sha256"`
	TopLevelKeys    []string       `json:"topLevelKeys"`
	Model           string         `json:"model"`
	Stream          bool           `json:"stream"`
	MessageCount    int            `json:"messageCount"`
	ToolCount       int            `json:"toolCount"`
	ModelInputBytes int            `json:"modelInputBytes"`
	SessionLog      *SessionLog    `json:"sessionLog"`
	ModelInputHits  map[string]int `json:"modelInputHits,omitempty"`
}

// Summarize reads one request body. The marker list is searched separately in
// the model-input fields and in the session-log extension, so a marker found
// only in one of them stays distinguishable.
func Summarize(body []byte, markers []string) (Summary, error) {
	sum := Summary{
		Bytes:  len(body),
		SHA256: hash(body),
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(body, &top); err != nil {
		// A body that cannot be parsed is still a captured body; report its
		// size and digest instead of dropping it.
		sum.TopLevelKeys = nil
		return sum, fmt.Errorf("request body is not a JSON object: %w", err)
	}

	sum.TopLevelKeys = sortedKeys(top)
	sum.Model = stringField(top, "model")
	sum.Stream = boolField(top, "stream")

	var messages []json.RawMessage
	if raw, ok := top["messages"]; ok {
		if err := json.Unmarshal(raw, &messages); err == nil {
			sum.MessageCount = len(messages)
		}
	}
	var tools []json.RawMessage
	if raw, ok := top["tools"]; ok {
		if err := json.Unmarshal(raw, &tools); err == nil {
			sum.ToolCount = len(tools)
		}
	}

	modelInput := modelInputBytes(top)
	sum.ModelInputBytes = len(modelInput)
	if hits := countAll(string(modelInput), markers); len(hits) > 0 {
		sum.ModelInputHits = hits
	}

	if raw, ok := top[SessionLogField]; ok && len(raw) > 0 && string(raw) != "null" {
		sum.SessionLog = summarizeSessionLog(raw, markers)
	}
	return sum, nil
}

// modelInputBytes renders the fields that reach the model, so their byte size
// can be compared with the session-log field size without mixing the two.
func modelInputBytes(top map[string]json.RawMessage) []byte {
	picked := make(map[string]json.RawMessage, 3)
	for _, key := range []string{"messages", "system", "tools"} {
		if raw, ok := top[key]; ok {
			picked[key] = raw
		}
	}
	out, err := json.Marshal(picked)
	if err != nil {
		return nil
	}
	return out
}

func summarizeSessionLog(raw json.RawMessage, markers []string) *SessionLog {
	log := &SessionLog{FieldBytes: len(raw), MarkerHits: map[string]int{}}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		log.UnparseableBody = true
		log.MarkerHits = countAll(string(raw), markers)
		return log
	}
	log.Keys = sortedKeys(obj)
	log.AfterSeq = intField(obj, "afterSeq")
	log.ThroughSeq = intField(obj, "throughSeq")

	var events []json.RawMessage
	if evRaw, ok := obj["events"]; ok {
		if err := json.Unmarshal(evRaw, &events); err == nil {
			log.EventCount = len(events)
			log.EventTypes = eventTypes(events)
		}
	}
	if hits := countAll(string(raw), markers); len(hits) > 0 {
		log.MarkerHits = hits
	} else {
		log.MarkerHits = nil
	}
	return log
}

func eventTypes(events []json.RawMessage) map[string]int {
	types := map[string]int{}
	for _, raw := range events {
		var event map[string]json.RawMessage
		if err := json.Unmarshal(raw, &event); err != nil {
			types["<unparseable>"]++
			continue
		}
		name := stringField(event, "type")
		if name == "" {
			name = "<missing>"
		}
		types[name]++
	}
	return types
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func stringField(m map[string]json.RawMessage, key string) string {
	raw, ok := m[key]
	if !ok {
		return ""
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return value
}

func boolField(m map[string]json.RawMessage, key string) bool {
	raw, ok := m[key]
	if !ok {
		return false
	}
	var value bool
	if err := json.Unmarshal(raw, &value); err != nil {
		return false
	}
	return value
}

func intField(m map[string]json.RawMessage, key string) *int64 {
	raw, ok := m[key]
	if !ok {
		return nil
	}
	var value int64
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil
	}
	return &value
}

func countAll(text string, markers []string) map[string]int {
	hits := map[string]int{}
	for _, marker := range markers {
		if marker == "" {
			continue
		}
		if n := strings.Count(text, marker); n > 0 {
			hits[marker] = n
		}
	}
	return hits
}

func hash(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}
