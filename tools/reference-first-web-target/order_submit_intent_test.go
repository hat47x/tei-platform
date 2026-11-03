//go:build !tei_repair_only

package main

import (
	"strings"
	"testing"

	"tei/internal/security/effectcommit/submit"
)

func TestCheckOrderSubmitIntent(t *testing.T) {
	const maxInt = "9223372036854775807"
	for _, tc := range []struct {
		name, op, payload string
		ok                bool
	}{
		{"web-match", "order-submit/1001", `{"orderId":1001,"to":"submitted"}`, true},
		{"cli-match-key-order", "cli-order-submit/7", `{"to":"submitted","orderId":7}`, true},
		{"whitespace-ok", "order-submit/7", ` { "orderId" : 7 , "to" : "submitted" } `, true},
		{"max-int64", "order-submit/" + maxInt, `{"orderId":` + maxInt + `,"to":"submitted"}`, true},
		{"other-order", "order-submit/1001", `{"orderId":1002,"to":"submitted"}`, false},
		{"other-state", "order-submit/1001", `{"orderId":1001,"to":"draft"}`, false},
		{"overflow-payload", "order-submit/1", `{"orderId":9223372036854775808,"to":"submitted"}`, false},
		{"overflow-op", "order-submit/9223372036854775808", `{"orderId":9223372036854775808,"to":"submitted"}`, false},
		{"negative", "order-submit/-5", `{"orderId":-5,"to":"submitted"}`, false},
		{"zero", "order-submit/0", `{"orderId":0,"to":"submitted"}`, false},
		{"leading-zero-op", "order-submit/01", `{"orderId":1,"to":"submitted"}`, false},
		{"plus-op", "order-submit/+1", `{"orderId":1,"to":"submitted"}`, false},
		{"fraction", "order-submit/1", `{"orderId":1.0,"to":"submitted"}`, false},
		{"exponent", "order-submit/1000", `{"orderId":1e3,"to":"submitted"}`, false},
		{"string-id", "order-submit/1", `{"orderId":"1","to":"submitted"}`, false},
		{"missing-orderId", "order-submit/1", `{"to":"submitted"}`, false},
		{"missing-to", "order-submit/1", `{"orderId":1}`, false},
		{"extra-key", "order-submit/1", `{"orderId":1,"to":"submitted","x":true}`, false},
		{"duplicate-key", "order-submit/1", `{"orderId":2,"orderId":1,"to":"submitted"}`, false},
		{"trailing-data", "order-submit/1", `{"orderId":1,"to":"submitted"} {}`, false},
		{"not-object", "order-submit/1", `[1]`, false},
		{"null", "order-submit/1", `null`, false},
		{"empty-payload", "order-submit/1", ``, false},
		{"no-slash", "order-submit-1", `{"orderId":1,"to":"submitted"}`, false},
		{"empty-suffix", "order-submit/", `{"orderId":1,"to":"submitted"}`, false},
		{"empty-op", "", `{"orderId":1,"to":"submitted"}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := checkOrderSubmitIntent(submit.Intent{OperationID: tc.op, Payload: []byte(tc.payload)})
			if (err == nil) != tc.ok {
				t.Fatalf("err=%v want ok=%v", err, tc.ok)
			}
			if err != nil && (strings.Contains(err.Error(), tc.payload) && tc.payload != "" || strings.Contains(err.Error(), tc.op) && tc.op != "") {
				t.Fatalf("error leaks input: %v", err)
			}
		})
	}
}

func TestOrderSubmitSpecsInstallSharedCheck(t *testing.T) {
	if runnerSubmitSpec.ValidateIntent == nil || cliSpec.ValidateIntent == nil {
		t.Fatal("a Target spec has no intent check")
	}
	in, err := runnerSubmitIntent(1001)
	if err != nil || checkOrderSubmitIntent(in) != nil || checkOrderSubmitIntent(cliIntent(1001)) != nil {
		t.Fatalf("callers' own intents must pass: %v", err)
	}
}
