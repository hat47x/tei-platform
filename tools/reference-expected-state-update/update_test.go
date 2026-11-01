package expectedstateupdate

import (
	"reflect"
	"testing"
)

func baseState() State {
	return State{Records: map[string]string{"order:1": "old"}}
}

func request(newValue, idempotencyKey string) UpdateRequest {
	return UpdateRequest{
		Key: "order:1", ExpectedPresent: true, Expected: "old", NewValue: newValue,
		IdempotencyKey: idempotencyKey, Authorized: true, Valid: true,
	}
}

func TestSecondConcurrentUpdateIsRejectedAndFirstRemains(t *testing.T) {
	run, err := Run(baseState(), []UpdateRequest{request("A", "update-a"), request("B", "update-b")})
	if err != nil {
		t.Fatal(err)
	}
	if len(run.Results) != 2 {
		t.Fatalf("results=%+v", run.Results)
	}
	if run.Results[0].Outcome != OutcomeApplied {
		t.Fatalf("first outcome=%+v", run.Results[0])
	}
	second := run.Results[1]
	if second.Outcome != OutcomeConflict || second.Reason != ReasonExpectedMismatch {
		t.Fatalf("second outcome=%+v", second)
	}
	if second.ExecutionError {
		t.Fatal("a conflict was reported as an execution error")
	}
	if run.State.Records["order:1"] != "A" {
		t.Fatalf("the first update did not remain: %+v", run.State.Records)
	}
	if run.Observation.ConflictRejections != 1 || run.Observation.LostUpdatesPrevented != 1 || run.Observation.Applied != 1 {
		t.Fatalf("observation=%+v", run.Observation)
	}
}

func TestPermissionAndValidationAreDistinctBusinessRejections(t *testing.T) {
	denied := request("A", "a")
	denied.Authorized = false
	run, err := Run(baseState(), []UpdateRequest{denied})
	if err != nil {
		t.Fatal(err)
	}
	if run.Results[0].Outcome != OutcomeDenied || run.Results[0].Reason != ReasonPermission || run.Results[0].ExecutionError {
		t.Fatalf("denied=%+v", run.Results[0])
	}
	if run.State.Records["order:1"] != "old" {
		t.Fatal("a denied update changed the record")
	}

	invalid := request("A", "a")
	invalid.Valid = false
	run, err = Run(baseState(), []UpdateRequest{invalid})
	if err != nil {
		t.Fatal(err)
	}
	if run.Results[0].Outcome != OutcomeInvalid || run.Results[0].Reason != ReasonValidation || run.Results[0].ExecutionError {
		t.Fatalf("invalid=%+v", run.Results[0])
	}
}

func TestIdempotentReplayDoesNotApplyTwice(t *testing.T) {
	run, err := Run(baseState(), []UpdateRequest{request("A", "same-key"), request("A", "same-key")})
	if err != nil {
		t.Fatal(err)
	}
	if run.Results[0].Outcome != OutcomeApplied || run.Results[1].Outcome != OutcomeIdempotentReplay {
		t.Fatalf("results=%+v", run.Results)
	}
	if run.Results[1].Before != "old" || run.Results[1].NewValue != "A" {
		t.Fatalf("replay lost the original result: %+v", run.Results[1])
	}
	if run.Observation.Applied != 1 || run.Observation.IdempotentReplays != 1 {
		t.Fatalf("observation=%+v", run.Observation)
	}
	if run.State.Records["order:1"] != "A" {
		t.Fatalf("records=%+v", run.State.Records)
	}
}

func TestIdempotencyKeyReuseWithAnotherUpdateIsRejected(t *testing.T) {
	run, err := Run(baseState(), []UpdateRequest{request("A", "same-key"), request("B", "same-key")})
	if err != nil {
		t.Fatal(err)
	}
	second := run.Results[1]
	if second.Outcome != OutcomeRejected || second.Reason != ReasonIdempotencyKeyReused || second.ExecutionError {
		t.Fatalf("second=%+v", second)
	}
	if run.State.Records["order:1"] != "A" {
		t.Fatalf("records=%+v", run.State.Records)
	}
	if run.Observation.IdempotencyKeyRejections != 1 {
		t.Fatalf("observation=%+v", run.Observation)
	}
}

func TestDigestComparisonDoesNotCarryTheRawValue(t *testing.T) {
	digestRequest := request("A", "digest-key")
	digestRequest.Compare = CompareDigest
	digestRequest.ExpectedDigest = valueDigest("old")
	run, err := Run(baseState(), []UpdateRequest{digestRequest})
	if err != nil {
		t.Fatal(err)
	}
	if run.Results[0].Outcome != OutcomeApplied {
		t.Fatalf("result=%+v", run.Results[0])
	}
	if run.Results[0].CurrentValue != "" {
		t.Fatalf("digest mode exposed the current value: %+v", run.Results[0])
	}
	if run.Observation.ComparisonInputBytes != len(digestRequest.ExpectedDigest) {
		t.Fatalf("digest comparison input bytes=%d", run.Observation.ComparisonInputBytes)
	}

	mismatch := request("A", "digest-mismatch")
	mismatch.Compare = CompareDigest
	mismatch.ExpectedDigest = valueDigest("not-the-current-value")
	run, err = Run(baseState(), []UpdateRequest{mismatch})
	if err != nil {
		t.Fatal(err)
	}
	if run.Results[0].Outcome != OutcomeConflict || run.Results[0].CurrentValue != "" {
		t.Fatalf("mismatch=%+v", run.Results[0])
	}
}

func TestMissingRecordIsDistinct(t *testing.T) {
	missing := request("A", "a")
	missing.Key = "order:missing"
	run, err := Run(baseState(), []UpdateRequest{missing})
	if err != nil {
		t.Fatal(err)
	}
	if run.Results[0].Outcome != OutcomeMissing || run.Results[0].Reason != ReasonMissingRecord {
		t.Fatalf("result=%+v", run.Results[0])
	}
	if run.Observation.MissingRecords != 1 {
		t.Fatalf("observation=%+v", run.Observation)
	}
}

func TestExpectedEmptyValueIsDistinctFromNotSupplied(t *testing.T) {
	state := State{Records: map[string]string{"order:1": ""}}
	empty := request("A", "empty-key")
	empty.Expected = ""
	run, err := Run(state, []UpdateRequest{empty})
	if err != nil {
		t.Fatal(err)
	}
	if run.Results[0].Outcome != OutcomeApplied {
		t.Fatalf("an expected empty value was not matched: %+v", run.Results[0])
	}

	notSupplied := request("A", "missing-expected")
	notSupplied.ExpectedPresent = false
	run, err = Run(state, []UpdateRequest{notSupplied})
	if err != nil {
		t.Fatal(err)
	}
	if run.Results[0].Outcome != OutcomeInvalid || run.Results[0].Reason != ReasonRequestInvalid {
		t.Fatalf("an omitted expected state was accepted: %+v", run.Results[0])
	}
}

func TestRunIsDeterministicAndDoesNotMutateTheInput(t *testing.T) {
	state := baseState()
	requests := []UpdateRequest{request("A", "a"), request("B", "b")}
	first, err := Run(state, requests)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Run(state, requests)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("the same input produced different runs:\n%+v\n%+v", first, second)
	}
	if first.Digest == "" {
		t.Fatal("run digest is empty")
	}
	if !reflect.DeepEqual(state, baseState()) {
		t.Fatalf("Run mutated the input state: %+v", state)
	}
}

func TestObservationCountsAreReproducible(t *testing.T) {
	requests := []UpdateRequest{
		request("A", "a"),
		request("B", "b"), // conflict
		request("A", "a"), // replay
	}
	run, err := Run(baseState(), requests)
	if err != nil {
		t.Fatal(err)
	}
	want := Observation{
		Comparisons: 2, Applied: 1, ConflictRejections: 1, IdempotentReplays: 1,
		ComparisonInputBytes: 10, LostUpdatesPrevented: 1,
	}
	if !reflect.DeepEqual(run.Observation, want) {
		t.Fatalf("observation=%+v want %+v", run.Observation, want)
	}
}
