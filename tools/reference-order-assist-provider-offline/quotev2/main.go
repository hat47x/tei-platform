// Generation 2 of the read-only quote realization for the order-assist-provider
// offline extension (steps 7 and 8). It keeps the generation-1 contract
// behaviour of plugins/reference-order-quote for every request that does not
// carry the new optional member, and adds one revision of the quote rule:
// an optional discount rate in basis points (0..3000) whose discount is
// floor(subtotal * rate / 10000).
//
// Like generation 1 it uses only the Go standard library and the existing
// process carrier, so the same source builds a native executable and a WASI
// Preview 1 module. It is a sample-local candidate, not an accepted contract.
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
	contractID       = "reference.order.quote-lines/v2"
	implementationID = "order-quote.v2"
	ruleVersion      = "reference.order.quote-rule/v2"
	pluginID         = "tei.reference.order_quote"
	pluginVersion    = "2.0.0"
	carrierID        = "transport.order_quote_carrier"
	contractRule     = "reference.order.quote-lines.contract/v2"
	requestRule      = "reference.order.quote-lines.request/v2"
	resultRule       = "reference.order.quote-lines.result/v2"
	maxBodyBytes     = 1 << 20
	maxDiscountBp    = 3000
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
	SKU            string
	Quantity       int64
	UnitPriceMinor int64
}

// quoteResult's member order is part of the byte-level comparison with the
// JavaScript realization of the same generation.
type quoteResult struct {
	Currency       string `json:"currency"`
	LineCount      int    `json:"lineCount"`
	SubtotalMinor  int64  `json:"subtotalMinor"`
	DiscountRateBp int64  `json:"discountRateBp"`
	DiscountMinor  int64  `json:"discountMinor"`
	NetMinor       int64  `json:"netMinor"`
	RuleVersion    string `json:"ruleVersion"`
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

// strictFields checks exact spelling, uniqueness and presence of members.
// Optional members may be absent but, when present, follow the same rules.
func strictFields(raw []byte, required, optional []string) (map[string]json.RawMessage, error) {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	first, err := dec.Token()
	if err != nil || first != json.Delim('{') {
		return nil, errors.New("quote value must be an object")
	}
	allowed := make(map[string]bool, len(required)+len(optional))
	for _, key := range required {
		allowed[key] = true
	}
	for _, key := range optional {
		allowed[key] = true
	}
	found := make(map[string]json.RawMessage, len(allowed))
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
		if err := dec.Decode(&value); err != nil || strings.TrimSpace(string(value)) == "null" {
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
	for _, key := range required {
		if _, ok := found[key]; !ok {
			return nil, errors.New("missing required quote member")
		}
	}
	return found, nil
}

// integer accepts only a plain JSON integer token (no fraction or exponent).
func integer(raw json.RawMessage) (int64, error) {
	text := strings.TrimSpace(string(raw))
	if text == "" || strings.ContainsAny(text, ".eE") {
		return 0, errors.New("not an integer")
	}
	var v int64
	if err := json.Unmarshal([]byte(text), &v); err != nil {
		return 0, err
	}
	return v, nil
}

func calculate(raw []byte) (quoteResult, error) {
	top, err := strictFields(raw, []string{"currency", "lines"}, []string{"discountRateBp"})
	if err != nil {
		return quoteResult{}, err
	}
	var currency string
	if err := json.Unmarshal(top["currency"], &currency); err != nil {
		return quoteResult{}, errors.New("quote currency must be a string")
	}
	var rawLines []json.RawMessage
	if err := json.Unmarshal(top["lines"], &rawLines); err != nil || rawLines == nil {
		return quoteResult{}, errors.New("quote lines must be an array")
	}
	var rate int64
	if value, ok := top["discountRateBp"]; ok {
		if rate, err = integer(value); err != nil {
			return quoteResult{}, errors.New("discount rate must be an integer")
		}
	}
	if currency != "JPY" || len(rawLines) == 0 || len(rawLines) > 50 || rate < 0 || rate > maxDiscountBp {
		return quoteResult{}, errors.New("unsupported quote shape")
	}
	var total int64
	for _, rawLine := range rawLines {
		fields, err := strictFields(rawLine, []string{"sku", "quantity", "unitPriceMinor"}, nil)
		if err != nil {
			return quoteResult{}, err
		}
		var line quoteLine
		if err := json.Unmarshal(fields["sku"], &line.SKU); err != nil {
			return quoteResult{}, errors.New("quote SKU must be a string")
		}
		if line.Quantity, err = integer(fields["quantity"]); err != nil {
			return quoteResult{}, errors.New("quote quantity must be an integer")
		}
		if line.UnitPriceMinor, err = integer(fields["unitPriceMinor"]); err != nil {
			return quoteResult{}, errors.New("quote unit price must be an integer")
		}
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
	// total <= 1e12 and rate <= 3000, so the product stays below 2^63.
	discount := total * rate / 10000
	return quoteResult{
		Currency: "JPY", LineCount: len(rawLines), SubtotalMinor: total,
		DiscountRateBp: rate, DiscountMinor: discount, NetMinor: total - discount,
		RuleVersion: ruleVersion,
	}, nil
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
		req.Plugin != (identity{ID: pluginID, Version: pluginVersion}) ||
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
		dispatch.Provider.PluginVersion != pluginVersion ||
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
