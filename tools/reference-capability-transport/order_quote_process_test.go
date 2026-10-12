package capabilitytransport

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"tei/internal/composition"
	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

type businessProcessLauncher struct {
	calls int
}

func (l *businessProcessLauncher) Invoke(ctx context.Context, command pruntime.ProcessCommand, request pruntime.Request) (pruntime.Response, error) {
	l.calls++
	return (pruntime.StdioProcessLauncher{}).Invoke(ctx, command, request)
}

// The standalone provider performs a REAL deterministic read-only calculation.
// It is NOT an order submission, database write, transaction, or replacement
// for Host-side business authorization and effect-time Permission checks.
func TestIndependentBusinessQuoteOverExistingGenericProcessTransport(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Build in a separate module: no TEI internal package, private go.mod,
	// package-level generated protocol or shared Go type dependency is needed.
	binary := filepath.Join(t.TempDir(), "order-quote")
	work := t.TempDir()
	source, err := os.ReadFile("../../plugins/reference-order-quote/main.go")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "main.go"), source, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "go.mod"), []byte("module example.invalid/reference-order-quote\n\ngo 1.23\n"), 0600); err != nil {
		t.Fatal(err)
	}
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, ".")
	build.Dir = work
	build.Env = append(os.Environ(), "GOWORK=off", "GOTOOLCHAIN=local")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("standalone business plugin build failed: %v\n%s", err, output)
	}
	image, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	sha := sha256.Sum256(image)
	imageDigest := "sha256:" + hex.EncodeToString(sha[:])

	provider := GenericProviderIdentity{
		PluginID:         "tei.reference.order_quote",
		PluginVersion:    "1.0.0",
		ImplementationID: "order-quote.v1",
		ArtifactDigest:   imageDigest,
	}
	payload := []byte(`{"currency":"JPY","lines":[{"sku":"PEN-01","quantity":3,"unitPriceMinor":250},{"sku":"BOOK-02","quantity":2,"unitPriceMinor":400}]}`)
	dispatch := GenericDispatch{
		Schema:                  dispatchSchema,
		ContractID:              "reference.order.quote-lines/v1",
		ContractFingerprint:     digestBytes([]byte("reference.order.quote-lines.contract/v1")),
		Target:                  "reference/order_entry",
		Provider:                provider,
		RequestShapeFingerprint: digestBytes([]byte("reference.order.quote-lines.request/v1")),
		ResultShapeFingerprint:  digestBytes([]byte("reference.order.quote-lines.result/v1")),
		PayloadCodec:            "application/json",
		Payload:                 payload,
	}
	plan, err := buildTransportDispatchPlan(genericDispatchIdentity(dispatch))
	if err != nil {
		t.Fatal(err)
	}
	carrier := LegacyCarrier{
		Plugin: plugin.Identity{
			ID: core.MustCanonicalID(provider.PluginID), Version: provider.PluginVersion,
		},
		CapabilityKind:        plugin.CapabilityContractQueryExecutor,
		CapabilityID:          core.MustCanonicalID("transport.order_quote_carrier"),
		DescriptorFingerprint: digestBytes([]byte("reference.order-quote.transport-carrier/v1")),
		Operation:             "generic-dispatch-v0",
	}

	contract := composition.Contract{
		ID: dispatch.ContractID, Version: "1.0.0", Shape: dispatch.RequestShapeFingerprint + "/" + dispatch.ResultShapeFingerprint,
	}
	registry := composition.NewRegistry()
	snapshot, err := registry.Apply(0, []composition.Provider{{
		ID: "order-quote-provider", Scope: dispatch.Target, Offers: contract,
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := registry.Select("order-quote-provider", dispatch.Target)
	if err != nil {
		t.Fatal(err)
	}
	binding := composition.Binding{
		ContractID: contract.ID, ContractVersion: contract.Version,
		ProviderID: selected.ProviderID, ArtifactDigest: imageDigest, Generation: 1,
	}
	hostAllowed := true
	guardChecks := 0
	coordinator, err := composition.NewSelectedCoordinator(registry, selected, binding, composition.NewScope(),
		func(actual composition.Binding) error {
			guardChecks++
			if !hostAllowed || actual != binding {
				return errors.New("Host approval unavailable")
			}
			return nil
		})
	if err != nil {
		t.Fatal(err)
	}

	launcher := &businessProcessLauncher{}
	invoke := func(d GenericDispatch) (TransportExecution, error) {
		lease, err := coordinator.Acquire()
		if err != nil {
			return TransportExecution{}, err
		}
		defer lease.Release()
		if lease.Binding.ContractID != d.ContractID ||
			lease.Binding.ArtifactDigest != d.Provider.ArtifactDigest ||
			plan.InvocationIdentity != genericDispatchIdentity(d) {
			return TransportExecution{}, errors.New("selected service does not match fixed dispatch identity")
		}
		// This is test-only host verification of the exact executable bytes.
		// The generic transport does not verify production artifact Authority.
		raw, err := os.ReadFile(binary)
		if err != nil || digestBytes(raw) != lease.Binding.ArtifactDigest {
			return TransportExecution{}, errors.New("local executable bytes drifted")
		}
		return invokePlannedOverProcessTransport(
			ctx, launcher, pruntime.ProcessCommand{Executable: binary}, plan, carrier,
			fmt.Sprintf("reference-order-quote-%d", launcher.calls+1), d,
		)
	}
	execution, err := invoke(dispatch)
	if err != nil {
		t.Fatal(err)
	}
	var quote struct {
		Currency      string `json:"currency"`
		LineCount     int    `json:"lineCount"`
		SubtotalMinor int64  `json:"subtotalMinor"`
	}
	if err := decodeJSONExact(execution.ResultPayload, &quote); err != nil {
		t.Fatalf("business result shape invalid: %v", err)
	}
	if quote.Currency != "JPY" || quote.LineCount != 2 || quote.SubtotalMinor != 1550 {
		t.Fatalf("independent business calculation differs: %+v", quote)
	}
	if execution.Receipt.DispatchPlanFingerprint != plan.PlanFingerprint ||
		execution.Receipt.RequestDigest != digestBytes(payload) ||
		execution.Receipt.ResultDigest != digestBytes(execution.ResultPayload) ||
		execution.Receipt.InvocationIdentity != plan.InvocationIdentity ||
		execution.Receipt.Transport != "process-stdio-v1" {
		t.Fatalf("process receipt lost semantic transport identity: %+v", execution.Receipt)
	}
	if launcher.calls != 1 || guardChecks != 1 {
		t.Fatalf("missing real process or Host check: calls=%d guards=%d", launcher.calls, guardChecks)
	}
	t.Run("different business request, same accepted semantic plan", func(t *testing.T) {
		next := dispatch
		next.Payload = []byte(`{"currency":"JPY","lines":[{"sku":"BOOK-02","quantity":2,"unitPriceMinor":900}]}`)
		got, err := invoke(next)
		if err != nil {
			t.Fatal(err)
		}
		var result struct {
			Currency      string `json:"currency"`
			LineCount     int    `json:"lineCount"`
			SubtotalMinor int64  `json:"subtotalMinor"`
		}
		if err := decodeJSONExact(got.ResultPayload, &result); err != nil {
			t.Fatal(err)
		}
		if result.SubtotalMinor != 1800 || result.LineCount != 1 ||
			got.Receipt.DispatchPlanFingerprint != plan.PlanFingerprint ||
			got.Receipt.RequestDigest == execution.Receipt.RequestDigest ||
			got.Receipt.ResultDigest == execution.Receipt.ResultDigest {
			t.Fatalf("request payload or quote result was not independently calculated: %+v %+v", result, got.Receipt)
		}
	})
	t.Run("invalid business request rejected by actual plugin", func(t *testing.T) {
		invalid := dispatch
		invalid.Payload = []byte(`{"currency":"JPY","lines":[{"sku":"PEN","quantity":-1,"unitPriceMinor":250}]}`)
		before := launcher.calls
		if _, err := invoke(invalid); err == nil || launcher.calls != before+1 {
			t.Fatalf("malformed business input was accepted: err=%v calls=%d", err, launcher.calls-before)
		}
	})
	t.Run("unaccepted shape stops before process", func(t *testing.T) {
		changed := dispatch
		changed.ResultShapeFingerprint = digestBytes([]byte("unaccepted-result-shape"))
		before := launcher.calls
		if _, err := invoke(changed); err == nil || launcher.calls != before {
			t.Fatalf("unaccepted shape reached external process: err=%v calls=%d", err, launcher.calls-before)
		}
	})
	t.Run("different artifact identity stops before process", func(t *testing.T) {
		changed := dispatch
		changed.Provider.ArtifactDigest = digestBytes([]byte("unauthorized-binary"))
		before := launcher.calls
		if _, err := invoke(changed); err == nil || launcher.calls != before {
			t.Fatalf("unbound artifact reached process: err=%v calls=%d", err, launcher.calls-before)
		}
	})
	t.Run("current Host authority revoked stops before process", func(t *testing.T) {
		hostAllowed = false
		defer func() { hostAllowed = true }()
		before := launcher.calls
		if _, err := invoke(dispatch); err == nil || launcher.calls != before {
			t.Fatalf("revoked Host Authority dispatched: err=%v calls=%d", err, launcher.calls-before)
		}
	})
	t.Run("provider removed stops before process", func(t *testing.T) {
		if _, err := registry.Apply(snapshot.Revision, nil, []string{"order-quote-provider"}); err != nil {
			t.Fatal(err)
		}
		before := launcher.calls
		if _, err := invoke(dispatch); err == nil || launcher.calls != before {
			t.Fatalf("withdrawn provider reached process: err=%v calls=%d", err, launcher.calls-before)
		}
	})
}

func TestReferenceBusinessQuoteTransportCannotProveSemanticResultShape(t *testing.T) {
	// The generic transport compares declared identity and schema echo, but
	// does not itself validate domain-shaped result bytes. Consumer-specific
	// result decoding / conformance remains a separate mandatory boundary.
	identity := genericDispatch([]byte("original"))
	response := GenericTransportResult{
		Schema:                  resultSchema,
		ContractID:              identity.ContractID,
		ContractFingerprint:     identity.ContractFingerprint,
		Target:                  identity.Target,
		Provider:                identity.Provider,
		RequestShapeFingerprint: identity.RequestShapeFingerprint,
		ResultShapeFingerprint:  identity.ResultShapeFingerprint,
		PayloadCodec:            identity.PayloadCodec,
		Payload:                 []byte(`{"currency":"JPY","subtotalMinor":"wrong-type"}`),
	}
	if err := validateGenericTransportResult(identity, response); err != nil {
		t.Fatalf("generic carrier unexpectedly pretended to know business schema: %v", err)
	}
	var typed struct {
		Currency      string `json:"currency"`
		SubtotalMinor int64  `json:"subtotalMinor"`
	}
	if err := decodeJSONExact(response.Payload, &typed); err == nil {
		t.Fatal("consumer-side shape validation accepted a spoofed business result")
	}
}
