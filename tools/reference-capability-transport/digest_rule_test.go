package capabilitytransport

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
)

func independentSHA256(value []byte) string {
	sum := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func TestTransportReceiptDeclaresRawPayloadDigestRule(t *testing.T) {
	dispatch := genericDispatch([]byte{0x00, 0x01, 0xff, 'T', 'E', 'I'})
	result, err := echoHTTPTransportImplementation(nil, dispatch)
	if err != nil {
		t.Fatal(err)
	}
	receipt := newTransportReceipt(
		"reference-test",
		TransportCarrierReceipt{Kind: "test"},
		"digest-rule",
		dispatch,
		result,
		nil,
	)
	if receipt.PayloadDigestRule != rawPayloadDigestRule {
		t.Fatalf("payload digest rule=%q", receipt.PayloadDigestRule)
	}
	if receipt.RequestDigest != independentSHA256(dispatch.Payload) ||
		receipt.ResultDigest != independentSHA256(result.Payload) {
		t.Fatalf("payload digest was not calculated from raw bytes: %+v", receipt)
	}
}

func TestTransportDispatchPlanDeclaresJSONMarshalFingerprintRule(t *testing.T) {
	plan, err := buildTransportDispatchPlan(
		genericDispatchIdentity(genericDispatch(nil)),
	)
	if err != nil {
		t.Fatal(err)
	}
	if plan.FingerprintRule != dispatchPlanFingerprintRule {
		t.Fatalf("plan fingerprint rule=%q", plan.FingerprintRule)
	}

	unsigned := plan
	unsigned.PlanFingerprint = ""
	payload, err := json.Marshal(unsigned)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlanFingerprint != independentSHA256(payload) {
		t.Fatalf(
			"plan fingerprint=%q independently recalculated=%q",
			plan.PlanFingerprint,
			independentSHA256(payload),
		)
	}
}

func TestTransportDispatchPlanRejectsFingerprintRuleDrift(t *testing.T) {
	plan, err := buildTransportDispatchPlan(
		genericDispatchIdentity(genericDispatch(nil)),
	)
	if err != nil {
		t.Fatal(err)
	}
	plan.FingerprintRule = "tei.reference.other-rule/v0"
	plan.PlanFingerprint = transportDispatchPlanFingerprint(plan)

	if err := validateTransportDispatchPlan(plan); err == nil {
		t.Fatal("dispatch plan with unknown fingerprint byte rule was accepted")
	}
}
