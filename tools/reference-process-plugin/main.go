package main

import (
	"encoding/json"
	"fmt"
	"os"

	"tei/internal/deployment/executor"
	pruntime "tei/internal/plugin/runtime"
)

func main() {
	var req pruntime.Request
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		fmt.Fprintln(os.Stderr, "invalid TEI process request")
		os.Exit(2)
	}
	if req.ProtocolVersion != pruntime.ProtocolVersion {
		writeError(req, "unsupported protocol version")
		return
	}
	if req.Operation != "execute" {
		writeError(req, "unsupported operation")
		return
	}

	var command executor.Command
	if err := json.Unmarshal(req.Payload, &command); err != nil {
		writeError(req, "invalid deployment command payload")
		return
	}
	result := executor.Result{
		AttemptID:      "reference-process-" + req.RequestID,
		ActionID:       command.ActionID,
		IdempotencyKey: command.IdempotencyKey,
		Outcome:        executor.OutcomeAccepted,
		Message:        "reference process accepted deployment intent",
		ExternalRef:    "reference-process://" + command.ID,
	}
	payload, err := json.Marshal(result)
	if err != nil {
		fmt.Fprintln(os.Stderr, "encode reference executor result")
		os.Exit(3)
	}
	writeResponse(pruntime.Response{
		ProtocolVersion:       req.ProtocolVersion,
		RequestID:             req.RequestID,
		Plugin:                req.Plugin,
		CapabilityID:          req.CapabilityID,
		DescriptorFingerprint: req.DescriptorFingerprint,
		Status:                pruntime.ResponseOK,
		Payload:               payload,
	})
}

func writeError(req pruntime.Request, message string) {
	writeResponse(pruntime.Response{
		ProtocolVersion:       req.ProtocolVersion,
		RequestID:             req.RequestID,
		Plugin:                req.Plugin,
		CapabilityID:          req.CapabilityID,
		DescriptorFingerprint: req.DescriptorFingerprint,
		Status:                pruntime.ResponseError,
		Error:                 message,
	})
}

func writeResponse(resp pruntime.Response) {
	if err := json.NewEncoder(os.Stdout).Encode(resp); err != nil {
		fmt.Fprintln(os.Stderr, "encode TEI process response")
		os.Exit(4)
	}
}
