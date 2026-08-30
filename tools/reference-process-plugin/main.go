package main

import (
	"encoding/json"
	"fmt"
	"os"

	"tei/internal/deployment/executor"
	pruntime "tei/internal/plugin/runtime"
)

const referenceArgvMarker = "--tei-reference-marker=argv value preserved"

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
	if !hasExactArg(referenceArgvMarker) {
		writeError(req, "reference argv marker missing")
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
		Message:        "reference process accepted deployment intent; argv=" + referenceArgvMarker,
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

func hasExactArg(want string) bool {
	for _, arg := range os.Args[1:] {
		if arg == want {
			return true
		}
	}
	return false
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
