// Reference implementation of an external process plugin.
// It intentionally imports only the Go standard library.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	protocolVersion     = 1
	referenceArgvMarker = "--tei-reference-marker=argv value preserved"
)

// A link-time marker for reference-only binary generation tests; not protocol data.
var referenceRevision = "v1"

// These private structures select only fields needed from the current process
// wire format. They are not a second normative definition of the TEI contract.
type processRequest struct {
	ProtocolVersion       int             `json:"protocolVersion"`
	RequestID             string          `json:"requestId"`
	Plugin                json.RawMessage `json:"plugin"`
	CapabilityID          string          `json:"capabilityId"`
	DescriptorFingerprint string          `json:"descriptorFingerprint"`
	Operation             string          `json:"operation"`
	Payload               json.RawMessage `json:"payload"`
}

type processResponse struct {
	ProtocolVersion       int             `json:"protocolVersion"`
	RequestID             string          `json:"requestId"`
	Plugin                json.RawMessage `json:"plugin"`
	CapabilityID          string          `json:"capabilityId"`
	DescriptorFingerprint string          `json:"descriptorFingerprint"`
	Status                string          `json:"status"`
	Payload               json.RawMessage `json:"payload,omitempty"`
	Error                 string          `json:"error,omitempty"`
}

type deploymentCommand struct {
	ID             string `json:"id"`
	ActionID       string `json:"actionId"`
	IdempotencyKey string `json:"idempotencyKey"`
}

type deploymentAcknowledgment struct {
	AttemptID      string `json:"attemptId"`
	ActionID       string `json:"actionId"`
	IdempotencyKey string `json:"idempotencyKey"`
	Outcome        string `json:"outcome"`
	Message        string `json:"message"`
	ExternalRef    string `json:"externalRef"`
}

func main() {
	if err := serve(os.Stdin, os.Stdout, os.Args[1:]); err != nil {
		// Never echo request data: payloads may contain secrets.
		fmt.Fprintln(os.Stderr, "invalid process plugin request")
		os.Exit(2)
	}
}

func serve(input io.Reader, output io.Writer, args []string) error {
	decoder := json.NewDecoder(input)
	var req processRequest
	if err := decoder.Decode(&req); err != nil {
		return fmt.Errorf("decode request: %w", err)
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("process protocol expects exactly one JSON request")
	}

	response := processResponse{
		ProtocolVersion:       req.ProtocolVersion,
		RequestID:             req.RequestID,
		Plugin:                req.Plugin,
		CapabilityID:          req.CapabilityID,
		DescriptorFingerprint: req.DescriptorFingerprint,
		Status:                "error",
	}
	writeError := func(message string) error {
		response.Error = message
		return json.NewEncoder(output).Encode(response)
	}

	switch {
	case req.RequestID == "" || len(req.Plugin) == 0 || req.CapabilityID == "" || req.DescriptorFingerprint == "":
		return writeError("incomplete process request identity")
	case req.ProtocolVersion != protocolVersion:
		return writeError("unsupported protocol version")
	case req.Operation != "execute":
		return writeError("unsupported operation")
	case !hasExactArg(args, referenceArgvMarker):
		return writeError("reference argv marker missing")
	}

	var command deploymentCommand
	if err := json.Unmarshal(req.Payload, &command); err != nil {
		return writeError("invalid deployment command payload")
	}
	if command.ID == "" || command.ActionID == "" || command.IdempotencyKey == "" {
		return writeError("incomplete deployment command")
	}

	ack := deploymentAcknowledgment{
		AttemptID:      "reference-process-" + req.RequestID,
		ActionID:       command.ActionID,
		IdempotencyKey: command.IdempotencyKey,
		Outcome:        "accepted",
		Message:        "reference process accepted deployment intent; argv=" + referenceArgvMarker + "; revision=" + referenceRevision,
		ExternalRef:    "reference-process://" + command.ID,
	}
	payload, err := json.Marshal(ack)
	if err != nil {
		return fmt.Errorf("encode deployment acknowledgment: %w", err)
	}
	response.Status = "ok"
	response.Payload = payload
	return json.NewEncoder(output).Encode(response)
}

func hasExactArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}
