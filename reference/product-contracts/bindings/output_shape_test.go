package bindings

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"testing"
)

// declaredAttributes reads the attribute names of one `data` type from
// semantic/contracts.tei. It is a textual reading of the declaration, not the
// TEI loader, because the question here is only which keys a binding returns
// compared with the keys the declaration lists (#3657).
func declaredAttributes(t *testing.T, typeName string) []string {
	t.Helper()
	source, err := os.ReadFile("../semantic/contracts.tei")
	if err != nil {
		t.Fatal(err)
	}
	block := regexp.MustCompile(`(?s)\ndata ` + typeName + ` \{(.*?)\n\}`).FindSubmatch(source)
	if block == nil {
		t.Fatalf("data %s is not declared in semantic/contracts.tei", typeName)
	}
	var names []string
	for _, m := range regexp.MustCompile(`attribute (\w+) type`).FindAllSubmatch(block[1], -1) {
		names = append(names, string(m[1]))
	}
	sort.Strings(names)
	if len(names) == 0 {
		t.Fatalf("data %s has no attribute", typeName)
	}
	return names
}

func payloadKeys(t *testing.T, payload any) []string {
	t.Helper()
	fields, ok := payload.(map[string]any)
	if !ok {
		t.Fatalf("payload is %T, want an object", payload)
	}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// difference returns the elements of a that are not in b, in the order of a.
func difference(a, b []string) []string {
	in := map[string]bool{}
	for _, v := range b {
		in[v] = true
	}
	var out []string
	for _, v := range a {
		if !in[v] {
			out = append(out, v)
		}
	}
	return out
}

func contains(list []string, want string) bool {
	return len(difference([]string{want}, list)) == 0
}

// TestSyntheticEKIBindingOutputsAgainstDeclaredTypes records, per binding, the
// keys the synthetic output lacks and the keys it adds beyond the declared
// output type. The recorded gaps are an explicit allowance, not an approval:
// whether to fill them in, drop them from the declaration, or map them is an
// Owner decision (#3657). The test fails when a binding or the declaration
// moves away from the record in either direction, so a gap can neither grow nor
// close unnoticed.
func TestSyntheticEKIBindingOutputsAgainstDeclaredTypes(t *testing.T) {
	ctx := context.Background()
	submit, err := EKITaskSubmit{}.Invoke(ctx, "product_contracts.eki_task_submit", map[string]any{"taskId": "tsk_reference", "executorId": "dev.example.transform"})
	if err != nil {
		t.Fatal(err)
	}
	execute, err := EKIAttemptExecute{}.Invoke(ctx, "product_contracts.eki_attempt_execute", map[string]any{"attemptId": "atm_reference"})
	if err != nil {
		t.Fatal(err)
	}
	accept, err := EKIResultAccept{}.Invoke(ctx, "product_contracts.eki_result_accept", map[string]any{"claimId": "clm_reference"})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name         string
		declaredType string
		payload      any
		wantMissing  []string
		wantExtra    []string
	}{
		{"EKITaskSubmit", "Attempt", submit.Payload, []string{"fenceEpoch", "leaseId", "workerIdentity"}, []string{"executor"}},
		{"EKIAttemptExecute", "ResultClaim", execute.Payload, []string{"claimedAt", "exitCode", "outputBytes", "outputDigest"}, nil},
		{"EKIResultAccept", "AcceptedResult", accept.Payload, []string{"acceptedAt", "level", "policyDigest"}, []string{"outcome"}},
	}
	for _, tc := range cases {
		declared := declaredAttributes(t, tc.declaredType)
		actual := payloadKeys(t, tc.payload)
		if got := difference(declared, actual); !reflect.DeepEqual(got, tc.wantMissing) {
			t.Errorf("%s: declared %s keys missing from the output = %v, recorded %v", tc.name, tc.declaredType, got, tc.wantMissing)
		}
		if got := difference(actual, declared); !reflect.DeepEqual(got, tc.wantExtra) {
			t.Errorf("%s: output keys not declared on %s = %v, recorded %v", tc.name, tc.declaredType, got, tc.wantExtra)
		}
	}
}

// TestSyntheticEKIChainPassesValuesByKeyName feeds each synthetic output into
// the next binding as its input. It checks that the key the next stage requires
// is a key of the producing stage's declared type, and that the value arrives
// unchanged, so a renamed key breaks the chain here instead of at a real caller.
func TestSyntheticEKIChainPassesValuesByKeyName(t *testing.T) {
	ctx := context.Background()
	submit, err := EKITaskSubmit{}.Invoke(ctx, "product_contracts.eki_task_submit", map[string]any{"taskId": "tsk_chain", "executorId": "dev.example.transform"})
	if err != nil {
		t.Fatal(err)
	}
	submitted := submit.Payload.(map[string]any)

	if !contains(declaredAttributes(t, "Attempt"), "attemptId") {
		t.Fatal("eki_attempt_execute requires attemptId, but Attempt does not declare it")
	}
	execute, err := EKIAttemptExecute{}.Invoke(ctx, "product_contracts.eki_attempt_execute", submitted)
	if err != nil {
		t.Fatalf("submit output was not accepted as the execute input: %v", err)
	}
	claimed := execute.Payload.(map[string]any)
	if claimed["attemptId"] != submitted["attemptId"] {
		t.Fatalf("attemptId changed between stages: %v -> %v", submitted["attemptId"], claimed["attemptId"])
	}

	if !contains(declaredAttributes(t, "ResultClaim"), "claimId") {
		t.Fatal("eki_result_accept requires claimId, but ResultClaim does not declare it")
	}
	accept, err := EKIResultAccept{}.Invoke(ctx, "product_contracts.eki_result_accept", claimed)
	if err != nil {
		t.Fatalf("execute output was not accepted as the accept input: %v", err)
	}
	if got := accept.Payload.(map[string]any)["claimId"]; got != claimed["claimId"] {
		t.Fatalf("claimId changed between stages: %v -> %v", claimed["claimId"], got)
	}
}

// TestEKITaskSubmitHTTPOutputIsNotAnAttempt uses the 202 body shape from the
// EKI external contract (TaskAccepted in eki-fabric spec/external/openapi.json,
// observed at eki-fabric f8047c1). That body names the Task (task_id) and its
// status; it has no Attempt, because an Attempt does not exist while a Task is
// only queued. The binding returns the body unchanged, so none of the declared
// Attempt keys is present and the next stage cannot take it as input. This
// records the gap as "not connected" (#3657); it does not map the shapes.
//
// The body lives in testdata/task_accepted.json rather than inline so that the
// pin in ../external-contract-pin.json can be checked against the same bytes:
// a contract change has to move the fixture and the pin together (#3836).
func TestEKITaskSubmitHTTPOutputIsNotAnAttempt(t *testing.T) {
	accepted, err := os.ReadFile(filepath.Join("testdata", "task_accepted.json"))
	if err != nil {
		t.Fatalf("read the accepted body fixture: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write(accepted)
	}))
	defer server.Close()

	result, err := EKITaskSubmitHTTP{BaseURL: server.URL, Client: server.Client()}.Invoke(context.Background(), "product_contracts.eki_task_submit", map[string]any{
		"idempotencyKey": "idem-shape-1", "request": map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	actual := payloadKeys(t, result.Payload)
	declared := declaredAttributes(t, "Attempt")
	if shared := difference(actual, difference(actual, declared)); len(shared) != 0 {
		t.Fatalf("HTTP output now shares keys with Attempt %v; update the record and the README", shared)
	}
	if _, err := (EKIAttemptExecute{}).Invoke(context.Background(), "product_contracts.eki_attempt_execute", result.Payload); err == nil {
		t.Fatal("the HTTP acceptance body was accepted as eki_attempt_execute input; the gap record is stale")
	}
}

// TestEKITaskSubmitInputShapesDiffer records that the two bindings of the same
// capability take different inputs, and that only the synthetic one lies inside
// the declared TaskRequest attributes.
func TestEKITaskSubmitInputShapesDiffer(t *testing.T) {
	declared := declaredAttributes(t, "TaskRequest")
	synthetic := []string{"executorId", "taskId"}
	httpInput := []string{"idempotencyKey", "request"}
	if extra := difference(synthetic, declared); len(extra) != 0 {
		t.Errorf("synthetic input keys outside TaskRequest: %v", extra)
	}
	if extra := difference(httpInput, declared); !reflect.DeepEqual(extra, httpInput) {
		t.Errorf("HTTP input keys outside TaskRequest = %v, recorded %v", extra, httpInput)
	}
}
