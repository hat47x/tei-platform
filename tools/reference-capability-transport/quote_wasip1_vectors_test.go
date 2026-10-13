package capabilitytransport

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

// The same Go provider source is built for WASI Preview 1 and exercised by a
// second execution realization. Its outer carrier is still process-stdio-v1;
// this is NOT proof of a second transport or a Component Model/WIT ABI.
func TestIndependentQuoteWASIP1AgainstCandidateSemanticVectors(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("reference process launcher and Node WASI path handling are tested on POSIX hosts")
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node.js WASI host is not installed; no WASI evidence from this run")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	sourceDir := "../../plugins/reference-order-quote"
	work := t.TempDir()
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		t.Fatal(err)
	}
	copied := 0
	for _, entry := range entries {
		if entry.Name() == "README.md" {
			continue
		}
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 ||
			!strings.HasSuffix(entry.Name(), ".go") {
			t.Fatalf("unexpected independent quote provider source: %s", entry.Name())
		}
		raw, err := os.ReadFile(filepath.Join(sourceDir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(work, entry.Name()), raw, 0600); err != nil {
			t.Fatal(err)
		}
		copied++
	}
	if copied == 0 {
		t.Fatal("standalone quote provider has no Go source")
	}
	if err := os.WriteFile(filepath.Join(work, "go.mod"),
		[]byte("module example.invalid/quote-wasip1-reference\n\ngo 1.23\n"), 0600); err != nil {
		t.Fatal(err)
	}

	wasm := filepath.Join(t.TempDir(), "quote.wasm")
	build := exec.CommandContext(ctx, "go", "build", "-o", wasm, ".")
	build.Dir = work
	build.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm",
		"GOENV=off", "GOWORK=off", "GOFLAGS=", "GOTOOLCHAIN=local",
		"GOPROXY=off", "GOSUMDB=off")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("independent WASI quote build failed: %v\n%s", err, output)
	}
	bytes, err := os.ReadFile(wasm)
	if err != nil {
		t.Fatal(err)
	}
	wasmDigest := digestBytes(bytes)
	runner, err := filepath.Abs("quote_wasip1_runner.mjs")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(runner); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile("../../reference/candidates/order-quote/semantic-vectors-v0.json")
	if err != nil {
		t.Fatal(err)
	}
	var corpus quoteSemanticVectorFile
	if err := json.Unmarshal(raw, &corpus); err != nil {
		t.Fatal(err)
	}
	if corpus.Schema != "tei.reference.quote-semantic-vectors/candidate-v0" ||
		corpus.Status != "unaccepted" || corpus.Contract != "reference.order.quote-lines/v1" ||
		len(corpus.Vectors) != 23 {
		t.Fatalf("quote candidate oracle identity or count drifted: %+v", corpus)
	}

	provider := GenericProviderIdentity{
		PluginID: "tei.reference.order_quote", PluginVersion: "1.0.0",
		ImplementationID: "order-quote.v1", ArtifactDigest: wasmDigest,
	}
	dispatch := GenericDispatch{
		Schema: dispatchSchema, ContractID: corpus.Contract,
		ContractFingerprint: digestBytes([]byte("reference.order.quote-lines.contract/v1")),
		Target: "reference/order_entry", Provider: provider,
		RequestShapeFingerprint: digestBytes([]byte("reference.order.quote-lines.request/v1")),
		ResultShapeFingerprint: digestBytes([]byte("reference.order.quote-lines.result/v1")),
		PayloadCodec: "application/json",
	}
	plan, err := buildTransportDispatchPlan(genericDispatchIdentity(dispatch))
	if err != nil {
		t.Fatal(err)
	}
	carrier := LegacyCarrier{
		Plugin: plugin.Identity{
			ID: core.MustCanonicalID(provider.PluginID), Version: provider.PluginVersion,
		},
		CapabilityKind: plugin.CapabilityContractQueryExecutor,
		CapabilityID: core.MustCanonicalID("transport.order_quote_carrier"),
		DescriptorFingerprint: digestBytes([]byte("reference.order-quote.transport-carrier/v1")),
		Operation: "generic-dispatch-v0",
	}
	// Node is the host runtime executable, not the plugin artifact. The digest
	// identifies the Wasm guest; neither identity authenticates the host or
	// closes the path-to-execution TOCTOU window in this reference.
	command := pruntime.ProcessCommand{
		Executable: node,
		Args:       []string{runner, wasm},
	}
	launcher := &businessProcessLauncher{}
	positive, negative := 0, 0
	seen := map[string]bool{}
	var baseDigest, reorderDigest, splitDigest string
	for i, vector := range corpus.Vectors {
		if vector.Name == "" || seen[vector.Name] ||
			vector.Reject == (vector.Expect != nil) || !json.Valid([]byte(vector.Request)) {
			t.Fatalf("malformed or repeated quote vector %q", vector.Name)
		}
		seen[vector.Name] = true
		t.Run(vector.Name, func(t *testing.T) {
			// Refuse a guest that changed after the reference build. This is a
			// test-only comparison, not trusted runtime artifact attestation.
			now, err := os.ReadFile(wasm)
			if err != nil || digestBytes(now) != wasmDigest {
				t.Fatal("Wasm reference artifact no longer matches the build")
			}
			request := dispatch
			request.Payload = []byte(vector.Request)
			before := launcher.calls
			execution, err := invokePlannedOverProcessTransport(
				ctx, launcher, command, plan, carrier, "quote-wasip1-"+strconv.Itoa(i), request,
			)
			if launcher.calls != before+1 {
				t.Fatalf("actual WASI invocation count changed by %d", launcher.calls-before)
			}
			if vector.Reject {
				negative++
				if err == nil {
					t.Fatalf("WASI business provider accepted rejected vector: %+v", execution)
				}
				return
			}
			positive++
			if err != nil {
				t.Fatal(err)
			}
			var result quoteSemanticResult
			if err := decodeJSONExact(execution.ResultPayload, &result); err != nil {
				t.Fatalf("WASI business result shape invalid: %v", err)
			}
			if result != *vector.Expect {
				t.Fatalf("WASI result=%+v independent expected=%+v", result, *vector.Expect)
			}
			if execution.Receipt.DispatchPlanFingerprint != plan.PlanFingerprint ||
				execution.Receipt.RequestDigest != digestBytes(request.Payload) ||
				execution.Receipt.ResultDigest != digestBytes(execution.ResultPayload) ||
				execution.Receipt.Transport != "process-stdio-v1" {
				t.Fatalf("WASI process carrier receipt mismatch: %+v", execution.Receipt)
			}
			switch vector.Name {
			case "two item baseline":
				baseDigest = execution.Receipt.ResultDigest
			case "order permutation":
				reorderDigest = execution.Receipt.ResultDigest
			case "split equivalent":
				splitDigest = execution.Receipt.ResultDigest
			}
		})
	}
	if positive != 8 || negative != 15 || baseDigest == "" ||
		baseDigest != reorderDigest || baseDigest == splitDigest {
		t.Fatalf("candidate corpus coverage/invariants changed: positive=%d negative=%d", positive, negative)
	}

	// Pre-dispatch rejection is owned by the existing transport plan, not the
	// Wasm guest. A forged result shape must not cause another WASI invocation.
	invalid := dispatch
	invalid.ResultShapeFingerprint = digestBytes([]byte("unaccepted-wasip1-shape"))
	before := launcher.calls
	if _, err := invokePlannedOverProcessTransport(
		ctx, launcher, command, plan, carrier, "quote-wasip1-invalid-shape", invalid,
	); err == nil || launcher.calls != before {
		t.Fatalf("unaccepted shape reached WASI process: err=%v calls=%d", err, launcher.calls-before)
	}
}
