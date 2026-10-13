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

// This tests a second physical realization (WASI Guest), not a second wire
// transport: the outer test still uses the existing process-stdio carrier.
func TestIndependentQuoteWASIP1AgainstCandidateSemanticVectors(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("reference Node WASI path handling is currently tested on POSIX")
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node.js is unavailable; this is not evidence of WASI parity")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	work := t.TempDir()
	files, err := os.ReadDir("../../plugins/reference-order-quote")
	if err != nil {
		t.Fatal(err)
	}
	copied := 0
	for _, entry := range files {
		if entry.Name() == "README.md" {
			continue
		}
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || !strings.HasSuffix(entry.Name(), ".go") {
			t.Fatalf("unexpected standalone source: %s", entry.Name())
		}
		raw, err := os.ReadFile(filepath.Join("../../plugins/reference-order-quote", entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(work, entry.Name()), raw, 0600); err != nil {
			t.Fatal(err)
		}
		copied++
	}
	if copied == 0 {
		t.Fatal("no independent provider Go source")
	}
	if err := os.WriteFile(filepath.Join(work, "go.mod"), []byte("module example.invalid/quote-wasip1-reference\n\ngo 1.23\n"), 0600); err != nil {
		t.Fatal(err)
	}
	wasm := filepath.Join(t.TempDir(), "order-quote.wasm")
	build := exec.CommandContext(ctx, "go", "build", "-o", wasm, ".")
	build.Dir = work
	build.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm", "GOENV=off", "GOWORK=off", "GOFLAGS=", "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("independent WASI build: %v\n%s", err, output)
	}
	rawWasm, err := os.ReadFile(wasm)
	if err != nil {
		t.Fatal(err)
	}
	artifactDigest := digestBytes(rawWasm)
	runner, err := filepath.Abs("quote_wasip1_runner.mjs")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(runner); err != nil {
		t.Fatal(err)
	}

	rawVectors, err := os.ReadFile("../../reference/candidates/order-quote/semantic-vectors-v0.json")
	if err != nil {
		t.Fatal(err)
	}
	var corpus quoteSemanticVectorFile
	if err := json.Unmarshal(rawVectors, &corpus); err != nil {
		t.Fatal(err)
	}
	if corpus.Schema != "tei.reference.quote-semantic-vectors/candidate-v0" || corpus.Status != "unaccepted" || corpus.Contract != "reference.order.quote-lines/v1" || len(corpus.Vectors) != 23 {
		t.Fatalf("reference candidate corpus identity drifted: %+v", corpus)
	}
	provider := GenericProviderIdentity{
		PluginID: "tei.reference.order_quote", PluginVersion: "1.0.0",
		ImplementationID: "order-quote.v1", ArtifactDigest: artifactDigest,
	}
	dispatch := GenericDispatch{
		Schema: dispatchSchema, ContractID: corpus.Contract,
		ContractFingerprint: digestBytes([]byte("reference.order.quote-lines.contract/v1")),
		Target:              "reference/order_entry", Provider: provider,
		RequestShapeFingerprint: digestBytes([]byte("reference.order.quote-lines.request/v1")),
		ResultShapeFingerprint:  digestBytes([]byte("reference.order.quote-lines.result/v1")),
		PayloadCodec:            "application/json",
	}
	plan, err := buildTransportDispatchPlan(genericDispatchIdentity(dispatch))
	if err != nil {
		t.Fatal(err)
	}
	carrier := LegacyCarrier{
		Plugin:                plugin.Identity{ID: core.MustCanonicalID(provider.PluginID), Version: provider.PluginVersion},
		CapabilityKind:        plugin.CapabilityContractQueryExecutor,
		CapabilityID:          core.MustCanonicalID("transport.order_quote_carrier"),
		DescriptorFingerprint: digestBytes([]byte("reference.order-quote.transport-carrier/v1")),
		Operation:             "generic-dispatch-v0",
	}
	// The fixed artifact digest names the Guest, not the Node runtime/script.
	// This local read check does not close the executable path TOCTOU window.
	command := pruntime.ProcessCommand{Executable: node, Args: []string{runner, wasm}}
	launcher := &businessProcessLauncher{}
	seen := map[string]bool{}
	positive, negative := 0, 0
	var base, reordered, split string
	for i, vector := range corpus.Vectors {
		if vector.Name == "" || seen[vector.Name] || vector.Reject == (vector.Expect != nil) || !json.Valid([]byte(vector.Request)) {
			t.Fatalf("invalid or repeated vector %q", vector.Name)
		}
		seen[vector.Name] = true
		t.Run(vector.Name, func(t *testing.T) {
			current, err := os.ReadFile(wasm)
			if err != nil || digestBytes(current) != artifactDigest {
				t.Fatal("WASI Guest artifact drifted")
			}
			request := dispatch
			request.Payload = []byte(vector.Request)
			before := launcher.calls
			result, err := invokePlannedOverProcessTransport(ctx, launcher, command, plan, carrier, "quote-wasip1-"+strconv.Itoa(i), request)
			if launcher.calls != before+1 {
				t.Fatal("WASI Guest was not invoked exactly once")
			}
			if vector.Reject {
				negative++
				if err == nil {
					t.Fatal("invalid business request accepted by WASI Guest")
				}
				return
			}
			positive++
			if err != nil {
				t.Fatal(err)
			}
			var actual quoteSemanticResult
			if err := decodeJSONExact(result.ResultPayload, &actual); err != nil {
				t.Fatal(err)
			}
			if actual != *vector.Expect {
				t.Fatalf("WASI result=%+v expected=%+v", actual, *vector.Expect)
			}
			if result.Receipt.DispatchPlanFingerprint != plan.PlanFingerprint || result.Receipt.RequestDigest != digestBytes(request.Payload) || result.Receipt.ResultDigest != digestBytes(result.ResultPayload) || result.Receipt.Transport != "process-stdio-v1" {
				t.Fatalf("WASI carrier receipt failed: %+v", result.Receipt)
			}
			switch vector.Name {
			case "two item baseline":
				base = result.Receipt.ResultDigest
			case "order permutation":
				reordered = result.Receipt.ResultDigest
			case "split equivalent":
				split = result.Receipt.ResultDigest
			}
		})
	}
	if positive != 8 || negative != 15 || base == "" || base != reordered || base == split {
		t.Fatalf("semantic vector coverage or invariants failed: %d successful, %d rejected", positive, negative)
	}
	invalid := dispatch
	invalid.ResultShapeFingerprint = digestBytes([]byte("unaccepted-wasip1-shape"))
	before := launcher.calls
	if _, err := invokePlannedOverProcessTransport(ctx, launcher, command, plan, carrier, "quote-wasip1-invalid-shape", invalid); err == nil || launcher.calls != before {
		t.Fatalf("unaccepted shape reached the WASI guest: err=%v calls=%d", err, launcher.calls-before)
	}
}
