package actionboundary

import (
	"bytes"
	"encoding/json"
	"errors"
)

// decodeStrictHTTPCommitIntent applies exact v1 envelope member names and
// rejects duplicates. encoding/json's default struct decoding would otherwise
// accept case-insensitive aliases and silently overwrite earlier members.
// Payload semantics remain the responsibility of the selected Application.
func decodeStrictHTTPCommitIntent(decoder *json.Decoder) (httpCommitIntent, error) {
	invalid := errors.New("invalid action envelope")
	var result httpCommitIntent
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('{') {
		return result, invalid
	}
	members := make(map[string]json.RawMessage, 6)
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return result, invalid
		}
		key, ok := token.(string)
		if !ok {
			return result, invalid
		}
		switch key {
		case "protocolVersion", "applicationID", "resourceID", "actionID", "expectedRevision", "payload":
		default:
			return result, invalid
		}
		if _, duplicate := members[key]; duplicate {
			return result, invalid
		}
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return result, invalid
		}
		members[key] = raw
	}
	closing, err := decoder.Token()
	if err != nil || closing != json.Delim('}') {
		return result, invalid
	}
	stringMember := func(key string) (string, error) {
		raw, present := members[key]
		if !present {
			return "", invalid
		}
		var value string
		if err := json.Unmarshal(raw, &value); err != nil || value == "" {
			return "", invalid
		}
		return value, nil
	}
	if result.ProtocolVersion, err = stringMember("protocolVersion"); err != nil {
		return httpCommitIntent{}, err
	}
	if result.ApplicationID, err = stringMember("applicationID"); err != nil {
		return httpCommitIntent{}, err
	}
	if result.ResourceID, err = stringMember("resourceID"); err != nil {
		return httpCommitIntent{}, err
	}
	if result.ActionID, err = stringMember("actionID"); err != nil {
		return httpCommitIntent{}, err
	}
	if result.ExpectedRevision, err = stringMember("expectedRevision"); err != nil {
		return httpCommitIntent{}, err
	}
	payload, present := members["payload"]
	if !present || bytes.Equal(bytes.TrimSpace(payload), []byte("null")) {
		return httpCommitIntent{}, invalid
	}
	result.Payload = payload
	return result, nil
}
