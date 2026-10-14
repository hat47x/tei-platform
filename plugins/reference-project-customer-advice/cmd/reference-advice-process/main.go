// Command reference-advice-process is a separately buildable, project-owned
// reference process. Its transport identity is pinned to a SYNTHETIC contract
// fixture in #976; it is not an accepted Provider or a signed plugin artifact.
// Do not reuse this constant to publish a production Service Contract.
package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"

	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	customeradvice "tei/plugins/reference-project-customer-advice"
)

const referenceDispatchID = "runtime_dispatch.d_ce048e07732d7e7aa5d21453c66a94d1d5fadd94e033e3fcd8a1eaec8438af56"

func main() {
	if err := serve(os.Stdin, os.Stdout); err != nil {
		os.Exit(2)
	}
}

func serve(reader io.Reader, writer io.Writer) error {
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	var req pruntime.Request
	if err := decoder.Decode(&req); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return io.ErrUnexpectedEOF
		}
		return err
	}

	response := pruntime.Response{
		ProtocolVersion:       req.ProtocolVersion,
		RequestID:             req.RequestID,
		Plugin:                req.Plugin,
		CapabilityID:          req.CapabilityID,
		DescriptorFingerprint: req.DescriptorFingerprint,
		Status:                pruntime.ResponseError,
		Error:                 "project advice request rejected",
	}
	if req.ProtocolVersion != pruntime.ProtocolVersion ||
		req.RequestID == "" ||
		req.Plugin != (plugin.Identity{
			ID:      core.MustCanonicalID("tei.reference.project_customer_advice"),
			Version: "1.0.0",
		}) ||
		req.CapabilityKind != plugin.CapabilityKind("generic-runtime-dispatch") ||
		req.CapabilityID != core.CanonicalID(referenceDispatchID) ||
		req.Operation != "evaluate-advice" ||
		len(req.Permissions) != 0 || req.ResourceLease != nil {
		return json.NewEncoder(writer).Encode(response)
	}
	inputDecoder := json.NewDecoder(bytes.NewReader(req.Payload))
	inputDecoder.DisallowUnknownFields()
	var input customeradvice.Input
	if err := inputDecoder.Decode(&input); err != nil {
		return json.NewEncoder(writer).Encode(response)
	}
	if err := inputDecoder.Decode(&trailing); err != io.EOF {
		return json.NewEncoder(writer).Encode(response)
	}
	result, err := customeradvice.Evaluate(input)
	if err != nil {
		return json.NewEncoder(writer).Encode(response)
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return err
	}
	response.Status = pruntime.ResponseOK
	response.Payload = raw
	response.Error = ""
	return json.NewEncoder(writer).Encode(response)
}
