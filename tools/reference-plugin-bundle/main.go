// A reference single executable that explicitly bundles two logical plugins.
// This uses the current legacy process wire only for an integration pressure test.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"tei/plugins/reference-bundle-alpha"
	"tei/plugins/reference-bundle-beta"
)

const referenceMarker = "--tei-reference-marker=argv value preserved"

type request struct {
	ProtocolVersion       int             `json:"protocolVersion"`
	RequestID             string          `json:"requestId"`
	Plugin                identity        `json:"plugin"`
	CapabilityID          string          `json:"capabilityId"`
	DescriptorFingerprint string          `json:"descriptorFingerprint"`
	Operation             string          `json:"operation"`
	Payload               json.RawMessage `json:"payload"`
}

type identity struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type command struct {
	ID             string `json:"id"`
	ActionID       string `json:"actionId"`
	IdempotencyKey string `json:"idempotencyKey"`
}

type response struct {
	ProtocolVersion       int             `json:"protocolVersion"`
	RequestID             string          `json:"requestId"`
	Plugin                identity        `json:"plugin"`
	CapabilityID          string          `json:"capabilityId"`
	DescriptorFingerprint string          `json:"descriptorFingerprint"`
	Status                string          `json:"status"`
	Payload               json.RawMessage `json:"payload,omitempty"`
	Error                 string          `json:"error,omitempty"`
}

type acknowledgment struct {
	AttemptID      string `json:"attemptId"`
	ActionID       string `json:"actionId"`
	IdempotencyKey string `json:"idempotencyKey"`
	Outcome        string `json:"outcome"`
	Message        string `json:"message"`
	ExternalRef    string `json:"externalRef"`
}

func main() {
	if err := serve(os.Stdin, os.Stdout, os.Args[1:]); err != nil {
		// Never copy payloads from an untrusted request into stderr.
		fmt.Fprintln(os.Stderr, "invalid bundled reference request")
		os.Exit(2)
	}
}

func serve(input io.Reader, output io.Writer, args []string) error {
	dec := json.NewDecoder(input)
	var req request
	if err := dec.Decode(&req); err != nil {
		return fmt.Errorf("decode request: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("exactly one JSON request is required")
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
		return json.NewEncoder(output).Encode(resp)
	}
	if req.ProtocolVersion != 1 || req.RequestID == "" || req.DescriptorFingerprint == "" ||
		req.Operation != "execute" || len(args) != 1 || args[0] != referenceMarker {
		return reject("invalid reference request envelope")
	}
	var accept func(string) (string, string)
	switch {
	case req.Plugin == (identity{ID: alpha.PluginID, Version: alpha.Version}) &&
		req.CapabilityID == alpha.CapabilityID:
		accept = alpha.Accept
	case req.Plugin == (identity{ID: beta.PluginID, Version: beta.Version}) &&
		req.CapabilityID == beta.CapabilityID:
		accept = beta.Accept
	default:
		return reject("unknown plugin identity or capability")
	}
	var cmd command
	if len(req.Payload) == 0 || json.Unmarshal(req.Payload, &cmd) != nil ||
		cmd.ID == "" || cmd.ActionID == "" || cmd.IdempotencyKey == "" {
		return reject("invalid command payload")
	}
	message, externalRef := accept(cmd.ID)
	raw, err := json.Marshal(acknowledgment{
		AttemptID: "reference-bundle-" + req.RequestID,
		ActionID:  cmd.ActionID, IdempotencyKey: cmd.IdempotencyKey,
		Outcome: "accepted", Message: message, ExternalRef: externalRef,
	})
	if err != nil {
		return fmt.Errorf("encode response: %w", err)
	}
	resp.Status = "ok"
	resp.Payload = raw
	return json.NewEncoder(output).Encode(resp)
}
