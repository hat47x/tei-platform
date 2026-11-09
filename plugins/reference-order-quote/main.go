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
	pluginID         = "tei.reference.order_quote"
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
	PluginLockFingerprint string          `json:"pluginLockFingerprint"`
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

// strictQuoteFields validates the exact spelling, uniqueness and presence of
// business JSON members. Go's DisallowUnknownFields still accepts case-folded
// names and silently replaces earlier duplicate keys; neither behavior is
// appropriate for a contract-bound business payload.
func strictQuoteFields(raw []byte, required ...string) (map[string]json.RawMessage, error) {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	first, err := dec.Token()
	if err != nil || first != json.Delim('{') {
		return nil, errors.New("quote value must be an object")
	}
	allowed := make(map[string]bool, len(required))
	for _, key := range required {
		allowed[key] = true
	}
	found := make(map[string]json.RawMessage, len(required))
	for dec.More() {
		keyToken, err := dec.Token()
		key, ok := keyToken.(string)
		if err != nil || !ok || !allowed[key] {
			return nil, errors.New("unknown quote member")
		}
		if _, duplicate := found[key]; duplicate {
			return nil, errors.New("duplicate quote member")
		}
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil ||
			strings.TrimSpace(string(value)) == "null" {
			return nil, errors.New("invalid quote member")
		}
		found[key] = value
	}
	last, err := dec.Token()
	if err != nil || last != json.Delim('}') {
		return nil, errors.New("invalid quote object")
	}
	var trailing any
	if err := dec.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, errors.New("trailing quote data")
	}
	if len(found) != len(allowed) {
		return nil, errors.New("missing required quote member")
	}
	return found, nil
}

func decodeQuoteRequest(raw []byte) (quoteRequest, error) {
	top, err := strictQuoteFields(raw, "currency", "lines")
	if err != nil {
		return quoteRequest{}, err
	}
	var request quoteRequest
	if err := json.Unmarshal(top["currency"], &request.Currency); err != nil {
		return quoteRequest{}, errors.New("quote currency must be a string")
	}
	var lines []json.RawMessage
	if err := json.Unmarshal(top["lines"], &lines); err != nil || lines == nil {
		return quoteRequest{}, errors.New("quote lines must be an array")
	}
	request.Lines = make([]quoteLine, 0, len(lines))
	for _, rawLine := range lines {
		fields, err := strictQuoteFields(rawLine, "sku", "quantity", "unitPriceMinor")
		if err != nil {
			return quoteRequest{}, err
		}
		var line quoteLine
		if err := json.Unmarshal(fields["sku"], &line.SKU); err != nil {
			return quoteRequest{}, errors.New("quote SKU must be a string")
		}
		if err := json.Unmarshal(fields["quantity"], &line.Quantity); err != nil {
			return quoteRequest{}, errors.New("quote quantity must be an integer")
		}
		if err := json.Unmarshal(fields["unitPriceMinor"], &line.UnitPriceMinor); err != nil {
			return quoteRequest{}, errors.New("quote unit price must be an integer")
		}
		request.Lines = append(request.Lines, line)
	}
	return request, nil
}

func calculate(raw []byte) (quoteResult, error) {
	request, err := decodeQuoteRequest(raw)
	if err != nil {
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
		ProtocolVersion:       req.ProtocolVersion,
		RequestID:             req.RequestID,
		Plugin:                req.Plugin,
		CapabilityID:          req.CapabilityID,
		DescriptorFingerprint: req.DescriptorFingerprint,
		Status:                "error",
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
		Schema:                  "tei.reference.capability-transport-result/v0",
		ContractID:              dispatch.ContractID,
		ContractFingerprint:     dispatch.ContractFingerprint,
		Target:                  dispatch.Target,
		Provider:                dispatch.Provider,
		RequestShapeFingerprint: dispatch.RequestShapeFingerprint,
		ResultShapeFingerprint:  dispatch.ResultShapeFingerprint,
		PayloadCodec:            dispatch.PayloadCodec,
		Payload:                 payload,
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
