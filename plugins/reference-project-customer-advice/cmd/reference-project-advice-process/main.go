// A standalone project-owned process adapter. The actual rule remains in the
// project package; this program only maps the existing process wire shape.
// This is a reference transport and does not mint any execution authority.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	customeradvice "tei/plugins/reference-project-customer-advice"
)

type request struct {
	ProtocolVersion       int             `json:"protocolVersion"`
	RequestID             string          `json:"requestId"`
	Plugin                json.RawMessage `json:"plugin"`
	CapabilityKind        string          `json:"capabilityKind"`
	CapabilityID          string          `json:"capabilityId"`
	DescriptorFingerprint string          `json:"descriptorFingerprint"`
	Operation             string          `json:"operation"`
	Payload               json.RawMessage `json:"payload"`
}

type response struct {
	ProtocolVersion       int             `json:"protocolVersion"`
	RequestID             string          `json:"requestId"`
	Plugin                json.RawMessage `json:"plugin"`
	CapabilityID          string          `json:"capabilityId"`
	DescriptorFingerprint string          `json:"descriptorFingerprint"`
	Status                string          `json:"status"`
	Payload               json.RawMessage `json:"payload,omitempty"`
	Error                 string          `json:"error,omitempty"`
}

func main() {
	if err := serve(os.Stdin, os.Stdout, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "invalid project advisory process request")
		os.Exit(2)
	}
}

func serve(in io.Reader, out io.Writer, argv []string) error {
	// The exact dispatch ID is supplied by the Host's candidate binding as a
	// launch argument, not by the user request. This does not establish that
	// the candidate itself was Accepted by a trusted issuer.
	if len(argv) != 1 || !strings.HasPrefix(argv[0], "--tei-expected-dispatch=") {
		return errors.New("missing reference dispatch binding")
	}
	expected := strings.TrimPrefix(argv[0], "--tei-expected-dispatch=")
	if expected == "" {
		return errors.New("empty reference dispatch binding")
	}
	decoder := json.NewDecoder(io.LimitReader(in, 64<<10))
	var req request
	if err := decoder.Decode(&req); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("multiple or malformed process requests")
	}
	resp := response{
		ProtocolVersion:       req.ProtocolVersion,
		RequestID:             req.RequestID,
		Plugin:                req.Plugin,
		CapabilityID:          req.CapabilityID,
		DescriptorFingerprint: req.DescriptorFingerprint,
		Status:                "error",
	}
	reject := func(message string) error {
		resp.Error = message
		return json.NewEncoder(out).Encode(resp)
	}
	if req.ProtocolVersion != 1 || req.RequestID == "" ||
		len(req.Plugin) == 0 || req.CapabilityID != expected ||
		req.CapabilityKind != "generic-runtime-dispatch" ||
		req.Operation != "evaluate-advice" {
		return reject("project advisory process dispatch mismatch")
	}
	var input customeradvice.Input
	if err := json.Unmarshal(req.Payload, &input); err != nil {
		return reject("invalid project advisory JSON")
	}
	result, err := customeradvice.Evaluate(input)
	if err != nil {
		return reject("invalid project advisory input")
	}
	payload, err := json.Marshal(result)
	if err != nil {
		return err
	}
	resp.Status = "ok"
	resp.Payload = payload
	return json.NewEncoder(out).Encode(resp)
}
