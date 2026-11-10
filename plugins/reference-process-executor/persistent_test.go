package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestPersistentReferenceRetainsOneShotAcknowledgmentsAndBoundsFrames(t *testing.T) {
	request := processRequest{ProtocolVersion: 1, RequestID: "one", Plugin: json.RawMessage(`{"id":"tei.process.reference","version":"1.0.0"}`), CapabilityID: "process.reference", DescriptorFingerprint: "reference", Operation: "execute", Payload: json.RawMessage(`{"id":"intent","actionId":"reference/apply","idempotencyKey":"reference-key"}`)}
	raw, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	var input bytes.Buffer
	for _, hold := range []bool{false, true} {
		_ = json.NewEncoder(&input).Encode(persistentFrame{Kind: "invoke", RequestID: "one", Request: raw, Hold: hold})
		if hold {
			_ = json.NewEncoder(&input).Encode(persistentFrame{Kind: "continue", RequestID: "one"})
		}
	}
	var output bytes.Buffer
	if err := servePersistent(&input, &output, []string{referenceArgvMarker}); err != nil {
		t.Fatal(err)
	}
	decode := json.NewDecoder(&output)
	var ready persistentFrame
	if err := decode.Decode(&ready); err != nil || ready.Kind != "ready" || ready.PID <= 0 {
		t.Fatal("missing readiness")
	}
	var one, started, two persistentFrame
	if decode.Decode(&one) != nil || decode.Decode(&started) != nil || decode.Decode(&two) != nil || one.Kind != "result" || started.Kind != "started" || two.Kind != "result" {
		t.Fatal("unexpected persistent sequence")
	}
	if !bytes.Equal(one.Response, two.Response) {
		t.Fatal("holding the request changed its acknowledgment")
	}
	if err := servePersistent(strings.NewReader(strings.Repeat("x", 65<<10)), &bytes.Buffer{}, nil); err == nil {
		t.Fatal("oversized frame accepted")
	}
	if err := servePersistent(strings.NewReader(""), &bytes.Buffer{}, []string{"--tei-reference-readiness-fail"}); err == nil {
		t.Fatal("readiness refusal ignored")
	}
}
