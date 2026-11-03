package capabilitytransport

import (
	"bytes"
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

// quoteRealizationStatusLauncher records the actual process response before
// transport validation. A rejected business vector must return Status=error,
// not merely fail because the host could not start a guest.
type quoteRealizationStatusLauncher struct {
	calls       int
	status      pruntime.ResponseStatus
	launchError error
}

func (l *quoteRealizationStatusLauncher) Invoke(ctx context.Context, command pruntime.ProcessCommand, request pruntime.Request) (pruntime.Response, error) {
	l.calls++
	response, err := (pruntime.StdioProcessLauncher{}).Invoke(ctx, command, request)
	l.status = response.Status
	l.launchError = err
	return response, err
}

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
	// #2057: go buildはキャッシュなし・CPU競合のCIで数十秒かかる一方、実プロセスの起動は
	// 1回あたり数十ms以内である。buildは独立した長い期限で待ち、実プロセスの期限は
	// build後に始めて、build時間が起動側の予算を食わないようにする。
	buildCtx, cancelBuild := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancelBuild()

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
	build := exec.CommandContext(buildCtx, "go", "build", "-o", wasm, ".")
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
	// Both builds consume precisely the same copied Go source. Each result is
	// an independent physical artifact with a different exact digest.
	native := filepath.Join(t.TempDir(), "order-quote-native")
	nativeBuild := exec.CommandContext(buildCtx, "go", "build", "-o", native, ".")
	nativeBuild.Dir = work
	nativeBuild.Env = append(os.Environ(), "GOOS="+runtime.GOOS, "GOARCH="+runtime.GOARCH, "GOENV=off", "GOWORK=off", "GOFLAGS=", "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off")
	if output, err := nativeBuild.CombinedOutput(); err != nil {
		t.Fatalf("independent Native build: %v\n%s", err, output)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	nativeBytes, err := os.ReadFile(native)
	if err != nil {
		t.Fatal(err)
	}
	nativeDigest := digestBytes(nativeBytes)
	if nativeDigest == artifactDigest {
		t.Fatal("different target artifacts unexpectedly have the same digest")
	}
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
	nativeProvider := provider
	nativeProvider.ArtifactDigest = nativeDigest
	nativeDispatch := dispatch
	nativeDispatch.Provider = nativeProvider
	nativePlan, err := buildTransportDispatchPlan(genericDispatchIdentity(nativeDispatch))
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
	launcher := &quoteRealizationStatusLauncher{}
	nativeLauncher := &quoteRealizationStatusLauncher{}
	nativeCommand := pruntime.ProcessCommand{Executable: native}
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
			nativeRequest := nativeDispatch
			nativeRequest.Payload = request.Payload
			currentNative, err := os.ReadFile(native)
			if err != nil || digestBytes(currentNative) != nativeDigest {
				t.Fatal("Native artifact drifted")
			}
			before := launcher.calls
			result, err := invokePlannedOverProcessTransport(ctx, launcher, command, plan, carrier, "quote-wasip1-"+strconv.Itoa(i), request)
			if launcher.calls != before+1 {
				t.Fatal("WASI Guest was not invoked exactly once")
			}
			nativeBefore := nativeLauncher.calls
			nativeResult, nativeErr := invokePlannedOverProcessTransport(ctx, nativeLauncher, nativeCommand, nativePlan, carrier, "quote-native-"+strconv.Itoa(i), nativeRequest)
			if nativeLauncher.calls != nativeBefore+1 {
				t.Fatal("Native Provider was not invoked exactly once")
			}
			if vector.Reject {
				negative++
				if err == nil || nativeErr == nil {
					t.Fatalf("invalid business input accepted: WASI=%v Native=%v", err, nativeErr)
				}
				if launcher.launchError != nil || nativeLauncher.launchError != nil ||
					launcher.status != pruntime.ResponseError || nativeLauncher.status != pruntime.ResponseError {
					t.Fatalf("expected business-level rejection, not host failure: WASI=(%s,%v), Native=(%s,%v)", launcher.status, launcher.launchError, nativeLauncher.status, nativeLauncher.launchError)
				}
				return
			}
			positive++
			if err != nil || nativeErr != nil {
				t.Fatalf("valid quote rejected: WASI=%v Native=%v", err, nativeErr)
			}
			if launcher.launchError != nil || nativeLauncher.launchError != nil ||
				launcher.status != pruntime.ResponseOK || nativeLauncher.status != pruntime.ResponseOK {
				t.Fatalf("unexpected host or business status: WASI=(%s,%v), Native=(%s,%v)", launcher.status, launcher.launchError, nativeLauncher.status, nativeLauncher.launchError)
			}
			if !bytes.Equal(result.ResultPayload, nativeResult.ResultPayload) ||
				result.Receipt.RequestDigest != nativeResult.Receipt.RequestDigest ||
				result.Receipt.ResultDigest != nativeResult.Receipt.ResultDigest {
				t.Fatalf("Native and WASI differ: Native=%s WASI=%s", nativeResult.ResultPayload, result.ResultPayload)
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
