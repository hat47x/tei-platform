package capabilitytransport

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

type quoteSemanticVectorFile struct {
	Schema   string                `json:"schema"`
	Status   string                `json:"status"`
	Contract string                `json:"contract"`
	Vectors  []quoteSemanticVector `json:"vectors"`
}

type quoteSemanticVector struct {
	Name    string               `json:"name"`
	Request string               `json:"request"`
	Reject  bool                 `json:"reject"`
	Expect  *quoteSemanticResult `json:"expect"`
}

type quoteSemanticResult struct {
	Currency      string `json:"currency"`
	LineCount     int    `json:"lineCount"`
	SubtotalMinor int64  `json:"subtotalMinor"`
}

// A fixed corpus is a REFERENCE semantic regression oracle, not a signed
// Conformance Evidence. Expected values are independently written into the
// corpus and never obtained from the provider under test.
func TestIndependentQuoteProcessAgainstCandidateSemanticVectors(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
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
		len(corpus.Vectors) < 20 {
		t.Fatalf("reference candidate vector identity drifted: %+v", corpus)
	}
	// Offline: the full separate provider module must not depend on TEI internals.
	src := "../../plugins/reference-order-quote"
	work := t.TempDir()
	entries, err := os.ReadDir(src)
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
			t.Fatalf("unexpected standalone provider source: %s", entry.Name())
		}
		code, err := os.ReadFile(filepath.Join(src, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(work, entry.Name()), code, 0600); err != nil {
			t.Fatal(err)
		}
		copied++
	}
	if copied == 0 {
		t.Fatal("no independent provider Go source")
	}
	if err := os.WriteFile(filepath.Join(work, "go.mod"),
		[]byte("module example.invalid/quote-semantic-vectors\n\ngo 1.23\n"), 0600); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "quote-provider")
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, ".")
	build.Dir = work
	build.Env = append(os.Environ(), "GOENV=off", "GOWORK=off", "GOFLAGS=",
		"GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("independent quote build: %v\n%s", err, output)
	}
	image, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	provider := GenericProviderIdentity{
		PluginID:         "tei.reference.order_quote",
		PluginVersion:    "1.0.0",
		ImplementationID: "order-quote.v1",
		ArtifactDigest:   digestBytes(image),
	}
	dispatch := GenericDispatch{
		Schema:                  dispatchSchema,
		ContractID:              corpus.Contract,
		ContractFingerprint:     digestBytes([]byte("reference.order.quote-lines.contract/v1")),
		Target:                  "reference/order_entry",
		Provider:                provider,
		RequestShapeFingerprint: digestBytes([]byte("reference.order.quote-lines.request/v1")),
		ResultShapeFingerprint:  digestBytes([]byte("reference.order.quote-lines.result/v1")),
		PayloadCodec:            "application/json",
	}
	plan, err := buildTransportDispatchPlan(genericDispatchIdentity(dispatch))
	if err != nil {
		t.Fatal(err)
	}
	carrier := LegacyCarrier{
		Plugin:                plugin.Identity{
			ID: core.MustCanonicalID(provider.PluginID), Version: provider.PluginVersion,
		},
		CapabilityKind:        plugin.CapabilityContractQueryExecutor,
		CapabilityID:          core.MustCanonicalID("transport.order_quote_carrier"),
		DescriptorFingerprint: digestBytes([]byte("reference.order-quote.transport-carrier/v1")),
		Operation:             "generic-dispatch-v0",
	}
	launch := &businessProcessLauncher{}
	seen := make(map[string]bool)
	positive, negative := 0, 0
	var baselineDigest, permutationDigest, splitDigest string
	for i, vector := range corpus.Vectors {
		if vector.Name == "" || seen[vector.Name] {
			t.Fatalf("empty or duplicate vector name: %q", vector.Name)
		}
		seen[vector.Name] = true
		if !json.Valid([]byte(vector.Request)) || vector.Reject == (vector.Expect != nil) {
			t.Fatalf("malformed vector expectation %q", vector.Name)
		}
		t.Run(vector.Name, func(t *testing.T) {
			d := dispatch
			d.Payload = []byte(vector.Request)
			before := launch.calls
			result, err := invokePlannedOverProcessTransport(
				ctx, launch, pruntime.ProcessCommand{Executable: binary},
				plan, carrier, "quote-vector-"+strconv.Itoa(i), d,
			)
			if launch.calls != before+1 {
				t.Fatalf("vector did not invoke actual process exactly once: %d", launch.calls-before)
			}
			if vector.Reject {
				negative++
				if err == nil {
					t.Fatalf("invalid business request was accepted: %+v", result)
				}
				return
			}
			positive++
			if err != nil {
				t.Fatal(err)
			}
			var actual quoteSemanticResult
			if err := decodeJSONExact(result.ResultPayload, &actual); err != nil {
				t.Fatalf("result is not exact quote JSON: %v", err)
			}
			if actual != *vector.Expect {
				t.Fatalf("semantic conformance: got=%+v independent expected=%+v", actual, *vector.Expect)
			}
			if result.Receipt.ResultDigest != digestBytes(result.ResultPayload) ||
				result.Receipt.RequestDigest != digestBytes(d.Payload) {
				t.Fatal("generic transport receipt did not bind actual input and result")
			}
			switch vector.Name {
			case "two item baseline":
				baselineDigest = result.Receipt.ResultDigest
			case "order permutation":
				permutationDigest = result.Receipt.ResultDigest
			case "split equivalent":
				splitDigest = result.Receipt.ResultDigest
			}
		})
	}
	if positive < 7 || negative < 10 ||
		baselineDigest == "" || baselineDigest != permutationDigest ||
		baselineDigest == splitDigest {
		t.Fatalf("semantic vector coverage or permutation/split invariants drifted: positive=%d negative=%d", positive, negative)
	}
	// Never describe these reference vectors as an accepted Provider conformance
	// profile, a signed publisher trust grant or a Runtime resource lease.
}
