// Independent, read-only business plug-in reference. This intentionally uses
// only the Go standard library and the EXISTING generic transport carrier.
// The legacy carrier is not a proposed universal plug-in ABI.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	protocolVersion  = 1
	contractID       = "reference.order.quote-lines/v1"
	implementationID = "order-quote.v1"
	pluginID         = "tei.reference.order-quote"
	carrierID        = "transport.order_quote_carrier"
	contractRule     = "reference.order.quote-lines.contract/v1"
	requestRule      = "reference.order.quote-lines.request/v1"
	resultRule       = "reference.order.quote-lines.result/v1"
	maxBodyBytes     = 1 << 20
)

type identity struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}
type carrierRequest struct {
	ProtocolVersion       int             `json:"protocolVersion"`
	RequestID             string          `json:"requestId"`
	Plugin                identity        `json:"plugin"`
	CapabilityKind        string          `json:"capabilityKind"`
	CapabilityID          string          `json:"capabilityId"`
	DescriptorFingerprint string          `json:"descriptorFingerprint"`
	Operation             string          `json:"operation"`
	Payload               json.RawMessage `json:"payload"`
}
type provider struct {
	PluginID         string `json:"pluginId"`
	PluginVersion    string `json:"pluginVersion"`
	ImplementationID string `json:"implementationId"`
	ArtifactDigest   string `json:"artifactDigest"`
}
type genericDispatch struct {
	Schema                  string   `json:"schema"`
	ContractID              string   `json:"contractId"`
	ContractFingerprint     string   `json:"contractFingerprint"`
	Target                  string   `json:"target"`
	Provider                provider `json:"provider"`
	RequestShapeFingerprint string   `json:"requestShapeFingerprint"`
	ResultShapeFingerprint  string   `json:"resultShapeFingerprint"`
	PayloadCodec            string   `json:"payloadCodec"`
	Payload                 []byte   `json:"payload"`
}
type genericResult struct {
	Schema                  string   `json:"schema"`
	ContractID              string   `json:"contractId"`
	ContractFingerprint     string   `json:"contractFingerprint"`
	Target                  string   `json:"target"`
	Provider                provider `json:"provider"`
	RequestShapeFingerprint string   `json:"requestShapeFingerprint"`
	ResultShapeFingerprint  string   `json:"resultShapeFingerprint"`
	PayloadCodec            string   `json:"payloadCodec"`
	Payload                 []byte   `json:"payload"`
}
type carrierResponse struct {
	ProtocolVersion       int             `json:"protocolVersion"`
	RequestID             string          `json:"requestId"`
	Plugin                identity        `json:"plugin"`
	CapabilityID          string          `json:"capabilityId"`
	DescriptorFingerprint string          `json:"descriptorFingerprint"`
	Status                string          `json:"status"`
	Payload               json.RawMessage `json:"payload,omitempty"`
	Error                 string          `json:"error,omitempty"`
}
type quoteLine struct {
	SKU            string `json:"sku"`
	Quantity       int64  `json:"quantity"`
	UnitPriceMinor int64  `json:"unitPriceMinor"`
}
type quoteRequest struct {
	Currency string      `json:"currency"`
	Lines    []quoteLine `json:"lines"`
}
type quoteResult struct {
	Currency      string `json:"currency"`
	LineCount     int    `json:"lineCount"`
	SubtotalMinor int64  `json:"subtotalMinor"`
}
func fingerprint(s string) string {
	sum := sha256.Sum256([]byte(s))
	return "sha256:" + hex.EncodeToString(sum[:])
}
func decodeExact(raw []byte, dst any) error {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("trailing JSON value")
	}
	return nil
}
func calculate(raw []byte) (quoteResult, error) {
	var request quoteRequest
	if err := decodeExact(raw, &request); err != nil {
		return quoteResult{}, errors.New("invalid quote request")
	}
	if request.Currency != "JPY" || len(request.Lines) == 0 || len(request.Lines) > 50 {
		return quoteResult{}, errors.New("unsupported quote shape")
	}
	var total int64
	for _, line := range request.Lines {
		if strings.TrimSpace(line.SKU) == "" || len(line.SKU) > 64 ||
			line.Quantity < 1 || line.Quantity > 100 ||
			line.UnitPriceMinor < 0 || line.UnitPriceMinor > 1_000_000_000 {
			return quoteResult{}, errors.New("invalid quote line")
		}
		lineTotal := line.Quantity * line.UnitPriceMinor
		if lineTotal > 1_000_000_000_000-total {
			return quoteResult{}, errors.New("quote total exceeds limit")
		}
		total += lineTotal
	}
	return quoteResult{Currency: "JPY", LineCount: len(request.Lines), SubtotalMinor: total}, nil
}
func serve(in io.Reader, out io.Writer) error {
	body, err := io.ReadAll(io.LimitReader(in, maxBodyBytes+1))
	if err != nil || len(body) > maxBodyBytes {
		return errors.New("invalid carrier body")
	}
	var req carrierRequest
	if err := decodeExact(body, &req); err != nil {
		return errors.New("invalid carrier request")
	}
	if req.ProtocolVersion != protocolVersion || req.RequestID == "" ||
		req.Plugin != (identity{ID: pluginID, Version: "1.0.0"}) ||
		req.CapabilityKind != "contract-query-executor" ||
		req.CapabilityID != carrierID || req.DescriptorFingerprint == "" ||
		req.Operation != "generic-dispatch-v0" {
		return errors.New("carrier does not match reference")
	}
	resp := carrierResponse{
		ProtocolVersion: req.ProtocolVersion,
		RequestID: req.RequestID,
		Plugin: req.Plugin,
		CapabilityID: req.CapabilityID,
		DescriptorFingerprint: req.DescriptorFingerprint,
		Status: "error",
	}
	deny := func() error {
		resp.Error = "invalid or unsupported read-only quote"
		return json.NewEncoder(out).Encode(resp)
	}
	var dispatch genericDispatch
	if err := decodeExact(req.Payload, &dispatch); err != nil {
		return deny()
	}
	if dispatch.Schema != "tei.reference.capability-transport-dispatch/v0" ||
		dispatch.ContractID != contractID ||
		dispatch.ContractFingerprint != fingerprint(contractRule) ||
		dispatch.RequestShapeFingerprint != fingerprint(requestRule) ||
		dispatch.ResultShapeFingerprint != fingerprint(resultRule) ||
		dispatch.Target != "reference/order_entry" ||
		dispatch.PayloadCodec != "application/json" ||
		dispatch.Provider.PluginID != pluginID ||
		dispatch.Provider.PluginVersion != "1.0.0" ||
		dispatch.Provider.ImplementationID != implementationID ||
		!strings.HasPrefix(dispatch.Provider.ArtifactDigest, "sha256:") {
		return deny()
	}
	value, err := calculate(dispatch.Payload)
	if err != nil {
		return deny()
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return deny()
	}
	result, err := json.Marshal(genericResult{
		Schema: "tei.reference.capability-transport-result/v0",
		ContractID: dispatch.ContractID,
		ContractFingerprint: dispatch.ContractFingerprint,
		Target: dispatch.Target,
		Provider: dispatch.Provider,
		RequestShapeFingerprint: dispatch.RequestShapeFingerprint,
		ResultShapeFingerprint: dispatch.ResultShapeFingerprint,
		PayloadCodec: dispatch.PayloadCodec,
		Payload: payload,
	})
	if err != nil {
		return deny()
	}
	resp.Status, resp.Payload = "ok", result
	return json.NewEncoder(out).Encode(resp)
}
func main() {
	if err := serve(os.Stdin, os.Stdout); err != nil {
		// Never put request payload, principal, credentials or secret text on stderr.
		_, _ = fmt.Fprintln(os.Stderr, "invalid read-only quote carrier")
		os.Exit(2)
	}
}
