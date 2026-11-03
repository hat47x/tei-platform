package expectedstateupdate

import (
	"bytes"
	"strings"
	"testing"
)

func recordFixture(t *testing.T) ([]UpdateRequest, []Result, Record) {
	t.Helper()
	requests := []UpdateRequest{
		{Key: "order:1", ExpectedPresent: true, Expected: "secret-old", NewValue: "secret-A", IdempotencyKey: "update-a", Authorized: true, Valid: true},
		{Key: "order:1", ExpectedPresent: true, Expected: "secret-old", NewValue: "secret-B", IdempotencyKey: "update-b", Authorized: true, Valid: true},
		{Key: "order:9", ExpectedPresent: true, Expected: "x", NewValue: "y", IdempotencyKey: "update-c", Authorized: true, Valid: true},
	}
	state := State{Records: map[string]string{"order:1": "secret-old"}}
	run, err := Run(state, requests)
	if err != nil {
		t.Fatal(err)
	}
	record, err := RecordRun(requests, run.Results)
	if err != nil {
		t.Fatal(err)
	}
	return requests, run.Results, record
}

func TestRecordRoundTripReproducesTheRun(t *testing.T) {
	requests, results, record := recordFixture(t)
	var encoded bytes.Buffer
	if err := EncodeRecord(&encoded, record); err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeRecord(&encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Digest != record.Digest {
		t.Fatalf("digest changed across the round trip")
	}
	if err := VerifyRecord(requests, results, decoded); err != nil {
		t.Fatalf("recorded run was not reproducible: %v", err)
	}
	// Recording the same run twice yields the same digest.
	again, err := RecordRun(requests, results)
	if err != nil {
		t.Fatal(err)
	}
	if again.Digest != record.Digest {
		t.Fatalf("same run produced different digests: %s != %s", again.Digest, record.Digest)
	}
}

func TestRecordDoesNotStoreRawValues(t *testing.T) {
	_, _, record := recordFixture(t)
	var encoded bytes.Buffer
	if err := EncodeRecord(&encoded, record); err != nil {
		t.Fatal(err)
	}
	text := encoded.String()
	for _, secret := range []string{"secret-old", "secret-A", "secret-B"} {
		if strings.Contains(text, secret) {
			t.Fatalf("record stored the raw value %q:\n%s", secret, text)
		}
	}
	if !strings.Contains(text, "sha256:") {
		t.Fatalf("record has no value digests:\n%s", text)
	}
}

func TestDecodeRecordRejectsTamperedEntry(t *testing.T) {
	_, _, record := recordFixture(t)
	var encoded bytes.Buffer
	if err := EncodeRecord(&encoded, record); err != nil {
		t.Fatal(err)
	}
	tampered := strings.Replace(encoded.String(), `"outcome": "APPLIED"`, `"outcome": "CONFLICT"`, 1)
	if tampered == encoded.String() {
		t.Fatal("test fixture did not contain an APPLIED outcome")
	}
	if _, err := DecodeRecord(strings.NewReader(tampered)); err == nil {
		t.Fatal("tampered record was accepted")
	}
}

func TestVerifyRecordDetectsDivergentRerun(t *testing.T) {
	requests, results, record := recordFixture(t)
	diverged := append([]Result(nil), results...)
	diverged[0].Outcome = OutcomeConflict
	if err := VerifyRecord(requests, diverged, record); err == nil {
		t.Fatal("divergent rerun was accepted")
	}
	if err := VerifyRecord(requests[:2], results[:2], record); err == nil {
		t.Fatal("shorter rerun was accepted")
	}
}

func TestVerifyRecordDetectsComparisonByteDrift(t *testing.T) {
	requests, results, record := recordFixture(t)
	diverged := append([]Result(nil), results...)
	diverged[0].Observation.ComparisonInputBytes++
	if err := VerifyRecord(requests, diverged, record); err == nil {
		t.Fatal("comparison byte drift was accepted")
	}
}

func TestRetainRunsKeepsTheNewest(t *testing.T) {
	records := []Record{{Digest: "1"}, {Digest: "2"}, {Digest: "3"}}
	kept, err := RetainRuns(records, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(kept) != 2 || kept[0].Digest != "2" || kept[1].Digest != "3" {
		t.Fatalf("kept=%+v", kept)
	}
	if _, err := RetainRuns(records, 0); err == nil {
		t.Fatal("zero retention was accepted")
	}
}
