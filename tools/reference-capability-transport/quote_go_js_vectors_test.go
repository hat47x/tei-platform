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

// A separate JavaScript implementation must preserve the semantic service
// contract, not necessarily the source language or the carrier's raw bytes.
// Both physical providers use the existing process-stdio reference transport.
func TestIndependentGoAndJavaScriptQuoteSemanticParity(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Node process parity reference currently runs on POSIX")
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node.js unavailable: not evidence of cross-language parity")
	}

	// #2057: go buildはキャッシュなし・CPU競合のCIで数十秒かかる一方、実プロセスの起動は
	// 1回あたり数十ms以内である。buildは独立した長い期限で待ち、実プロセスの期限は
	// build後に始めて、build時間が起動側の予算を食わないようにする。
	buildCtx, cancelBuild := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancelBuild()

	// Build the reference Go provider from a self-contained, offline module.
	work := t.TempDir()
	files, err := os.ReadDir("../../plugins/reference-order-quote")
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, entry := range files {
		if entry.Name() == "README.md" {
			continue
		}
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || !strings.HasSuffix(entry.Name(), ".go") {
			t.Fatalf("unexpected Go provider source: %s", entry.Name())
		}
		raw, err := os.ReadFile(filepath.Join("../../plugins/reference-order-quote", entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(work, entry.Name()), raw, 0600); err != nil {
			t.Fatal(err)
		}
		n++
	}
	if n == 0 {
		t.Fatal("no independent Go provider source")
	}
	if err := os.WriteFile(filepath.Join(work, "go.mod"), []byte("module example.invalid/quote-go-js-comparison\n\ngo 1.23\n"), 0600); err != nil {
		t.Fatal(err)
	}
	goBinary := filepath.Join(t.TempDir(), "order-quote-go")
	build := exec.CommandContext(buildCtx, "go", "build", "-o", goBinary, ".")
	build.Dir = work
	build.Env = append(os.Environ(), "GOOS="+runtime.GOOS, "GOARCH="+runtime.GOARCH,
		"GOENV=off", "GOWORK=off", "GOFLAGS=", "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("standalone Go build failed: %v\n%s", err, output)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	goBytes, err := os.ReadFile(goBinary)
	if err != nil {
		t.Fatal(err)
	}
	goDigest := digestBytes(goBytes)

	jsScript, err := filepath.Abs("../../plugins/reference-order-quote-js/main.mjs")
	if err != nil {
		t.Fatal(err)
	}
	jsBytes, err := os.ReadFile(jsScript)
	if err != nil {
		t.Fatal(err)
	}
	jsDigest := digestBytes(jsBytes)
	if jsDigest == goDigest {
		t.Fatal("distinct implementations must have distinct artifact digests")
	}
	rawVectors, err := os.ReadFile("../../reference/candidates/order-quote/semantic-vectors-v0.json")
	if err != nil {
		t.Fatal(err)
	}
	var corpus quoteSemanticVectorFile
	if err := json.Unmarshal(rawVectors, &corpus); err != nil {
		t.Fatal(err)
	}
	if corpus.Schema != "tei.reference.quote-semantic-vectors/candidate-v0" ||
		corpus.Status != "unaccepted" || corpus.Contract != "reference.order.quote-lines/v1" ||
		len(corpus.Vectors) != 23 {
		t.Fatalf("unaccepted candidate corpus drifted: %+v", corpus)
	}

	goProvider := GenericProviderIdentity{
		PluginID: "tei.reference.order_quote", PluginVersion: "1.0.0",
		ImplementationID: "order-quote.v1", ArtifactDigest: goDigest,
	}
	goDispatch := GenericDispatch{
		Schema: dispatchSchema, ContractID: corpus.Contract,
		ContractFingerprint: digestBytes([]byte("reference.order.quote-lines.contract/v1")),
		Target:              "reference/order_entry", Provider: goProvider,
		RequestShapeFingerprint: digestBytes([]byte("reference.order.quote-lines.request/v1")),
		ResultShapeFingerprint:  digestBytes([]byte("reference.order.quote-lines.result/v1")),
		PayloadCodec:            "application/json",
	}
	goPlan, err := buildTransportDispatchPlan(genericDispatchIdentity(goDispatch))
	if err != nil {
		t.Fatal(err)
	}
	jsDispatch := goDispatch
	jsProvider := goProvider
	jsProvider.ArtifactDigest = jsDigest
	jsDispatch.Provider = jsProvider
	jsPlan, err := buildTransportDispatchPlan(genericDispatchIdentity(jsDispatch))
	if err != nil {
		t.Fatal(err)
	}
	carrier := LegacyCarrier{
		Plugin:                plugin.Identity{ID: core.MustCanonicalID(goProvider.PluginID), Version: goProvider.PluginVersion},
		CapabilityKind:        plugin.CapabilityContractQueryExecutor,
		CapabilityID:          core.MustCanonicalID("transport.order_quote_carrier"),
		DescriptorFingerprint: digestBytes([]byte("reference.order-quote.transport-carrier/v1")),
		Operation:             "generic-dispatch-v0",
	}
	goLauncher := &quoteRealizationStatusLauncher{}
	jsLauncher := &quoteRealizationStatusLauncher{}
	goCommand := pruntime.ProcessCommand{Executable: goBinary}
	jsCommand := pruntime.ProcessCommand{Executable: node, Args: []string{jsScript}}

	// This assertion deliberately checks the outcome class, not error prose.
	// A failed Host launch is not a successful business rejection.
	check := func(t *testing.T, name, body string, reject bool, expected *quoteSemanticResult) {
		t.Helper()
		currentGo, e1 := os.ReadFile(goBinary)
		currentJS, e2 := os.ReadFile(jsScript)
		if e1 != nil || e2 != nil || digestBytes(currentGo) != goDigest || digestBytes(currentJS) != jsDigest {
			t.Fatal("provider artifact changed before invocation")
		}
		g := goDispatch
		g.Payload = []byte(body)
		j := jsDispatch
		j.Payload = []byte(body)
		beforeGo, beforeJS := goLauncher.calls, jsLauncher.calls
		goResult, goErr := invokePlannedOverProcessTransport(ctx, goLauncher, goCommand, goPlan, carrier, name+"-go", g)
		jsResult, jsErr := invokePlannedOverProcessTransport(ctx, jsLauncher, jsCommand, jsPlan, carrier, name+"-js", j)
		if goLauncher.calls != beforeGo+1 || jsLauncher.calls != beforeJS+1 {
			t.Fatal("each provider must be invoked once")
		}
		if goLauncher.launchError != nil || jsLauncher.launchError != nil {
			t.Fatalf("host failed, not a valid business result: Go=%v JS=%v", goLauncher.launchError, jsLauncher.launchError)
		}
		if reject {
			if goErr == nil || jsErr == nil ||
				goLauncher.status != pruntime.ResponseError || jsLauncher.status != pruntime.ResponseError {
				t.Fatalf("business rejection differs: Go=(%s,%v) JS=(%s,%v)", goLauncher.status, goErr, jsLauncher.status, jsErr)
			}
			return
		}
		if goErr != nil || jsErr != nil ||
			goLauncher.status != pruntime.ResponseOK || jsLauncher.status != pruntime.ResponseOK {
			t.Fatalf("business acceptance differs: Go=(%s,%v) JS=(%s,%v)", goLauncher.status, goErr, jsLauncher.status, jsErr)
		}
		var goQuote, jsQuote quoteSemanticResult
		if err := decodeJSONExact(goResult.ResultPayload, &goQuote); err != nil {
			t.Fatal(err)
		}
		if err := decodeJSONExact(jsResult.ResultPayload, &jsQuote); err != nil {
			t.Fatal(err)
		}
		if expected == nil || goQuote != *expected || jsQuote != *expected {
			t.Fatalf("semantic mismatch: Go=%+v JS=%+v expected=%+v", goQuote, jsQuote, expected)
		}
		// Strict serialized-byte equality is a *local regression guard* for
		// this pair of specimens, not a new requirement on all providers.
		if !bytes.Equal(goResult.ResultPayload, jsResult.ResultPayload) ||
			goResult.Receipt.ResultDigest != jsResult.Receipt.ResultDigest ||
			goResult.Receipt.RequestDigest != jsResult.Receipt.RequestDigest {
			t.Fatalf("serialized outcomes differ: Go=%s JS=%s", goResult.ResultPayload, jsResult.ResultPayload)
		}
		if goResult.Receipt.DispatchPlanFingerprint != goPlan.PlanFingerprint ||
			jsResult.Receipt.DispatchPlanFingerprint != jsPlan.PlanFingerprint ||
			goResult.Receipt.Transport != "process-stdio-v1" || jsResult.Receipt.Transport != "process-stdio-v1" {
			t.Fatal("unexpected transport receipt identity")
		}
	}
	seen := make(map[string]bool)
	accepted, denied := 0, 0
	for i, v := range corpus.Vectors {
		if v.Name == "" || seen[v.Name] || v.Reject == (v.Expect != nil) || !json.Valid([]byte(v.Request)) {
			t.Fatalf("bad candidate vector %q", v.Name)
		}
		seen[v.Name] = true
		t.Run(v.Name, func(t *testing.T) {
			check(t, "quote-"+strconv.Itoa(i), v.Request, v.Reject, v.Expect)
		})
		if v.Reject {
			denied++
		} else {
			accepted++
		}
	}
	if accepted != 8 || denied != 15 {
		t.Fatalf("candidate corpus coverage: accepted=%d denied=%d", accepted, denied)
	}

	// Independent adversarial probes: these are test assertions, NOT an
	// amendment of the unchanged business owner candidate oracle.
	probes := []struct{ name, raw string }{
		{"unsafe JavaScript integer", `{"currency":"JPY","lines":[{"sku":"P","quantity":9007199254740993,"unitPriceMinor":1}]}`},
		{"exponent integer", `{"currency":"JPY","lines":[{"sku":"P","quantity":1e0,"unitPriceMinor":1}]}`},
		{"top-level duplicate", `{"currency":"JPY","currency":"JPY","lines":[{"sku":"P","quantity":1,"unitPriceMinor":1}]}`},
		{"null price", `{"currency":"JPY","lines":[{"sku":"P","quantity":1,"unitPriceMinor":null}]}`},
		{"unknown nested field", `{"currency":"JPY","lines":[{"sku":"P","quantity":1,"unitPriceMinor":1,"ignored":0}]}`},
	}
	for i, probe := range probes {
		t.Run("adversarial/"+probe.name, func(t *testing.T) {
			check(t, "probe-"+strconv.Itoa(i), probe.raw, true, nil)
		})
	}
	invalid := jsDispatch
	invalid.ResultShapeFingerprint = digestBytes([]byte("unaccepted-js-result-shape"))
	before := jsLauncher.calls
	if _, err := invokePlannedOverProcessTransport(ctx, jsLauncher, jsCommand, jsPlan, carrier, "js-invalid-shape", invalid); err == nil ||
		jsLauncher.calls != before {
		t.Fatalf("invalid shape must reject before starting JavaScript: %v", err)
	}
}
