//go:build !tei_repair_only

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"

	"tei/internal/security/effectcommit/submit"
)

// errOrderSubmitIntentMismatch is the only error checkOrderSubmitIntent returns.
// It deliberately carries no payload or operation text.
var errOrderSubmitIntentMismatch = errors.New("order submit intent does not match its operation")

// checkOrderSubmitIntent is the single, shared check that the Target (the Web
// store and the CLI driver) installs as submit.Spec.ValidateIntent (Owner
// provisional decision D11). It requires that
//
//   - the OperationID ends with "/<n>", n a canonical positive int64,
//   - the payload is exactly {"orderId": <n>, "to": "submitted"} in any key
//     order, with no other, duplicate or missing key and no trailing data,
//   - orderId equals n.
//
// Anything else (overflow, zero, negative, fraction, a different order, a
// different target state) is refused before execution. The effect routine takes
// the order id from the payload, so this is what keeps the receipt's
// OperationID and the row that changes the same order.
func checkOrderSubmitIntent(in submit.Intent) error {
	idx := strings.LastIndexByte(in.OperationID, '/')
	if idx < 0 {
		return errOrderSubmitIntentMismatch
	}
	opOrder, ok := canonicalPositiveInt64(in.OperationID[idx+1:])
	if !ok {
		return errOrderSubmitIntentMismatch
	}

	dec := json.NewDecoder(bytes.NewReader(in.Payload))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return errOrderSubmitIntentMismatch
	}
	seen := map[string]json.RawMessage{}
	for dec.More() {
		keyTok, err := dec.Token()
		key, isString := keyTok.(string)
		if err != nil || !isString {
			return errOrderSubmitIntentMismatch
		}
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return errOrderSubmitIntentMismatch
		}
		if _, dup := seen[key]; dup || (key != "orderId" && key != "to") {
			return errOrderSubmitIntentMismatch
		}
		seen[key] = raw
	}
	if tok, err := dec.Token(); err != nil || tok != json.Delim('}') {
		return errOrderSubmitIntentMismatch
	}
	if _, err := dec.Token(); err != io.EOF {
		return errOrderSubmitIntentMismatch
	}
	rawID, okID := seen["orderId"]
	rawTo, okTo := seen["to"]
	if !okID || !okTo || len(seen) != 2 {
		return errOrderSubmitIntentMismatch
	}
	payloadOrder, ok := canonicalPositiveInt64(string(rawID))
	if !ok || payloadOrder != opOrder {
		return errOrderSubmitIntentMismatch
	}
	var to string
	if err := json.Unmarshal(rawTo, &to); err != nil || to != "submitted" {
		return errOrderSubmitIntentMismatch
	}
	return nil
}

// canonicalPositiveInt64 parses s as a base-10 integer > 0 that fits int64,
// with no sign, leading zero, fraction or exponent.
func canonicalPositiveInt64(s string) (int64, bool) {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n <= 0 || strconv.FormatInt(n, 10) != s {
		return 0, false
	}
	return n, true
}
