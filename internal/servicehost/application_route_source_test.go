package servicehost

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"tei/internal/composition"
	"tei/internal/core"
	pruntime "tei/internal/plugin/runtime"
)

func TestHostBoundApplicationEnvironmentReadsCurrentRouteWithoutBorrowing(t *testing.T) {
	digest := "sha256:" + strings.Repeat("a", 64)
	orderBinding := composition.Binding{
		ContractID: "sales.quote-lines", ContractVersion: "1.0.0",
		ProviderID: "logical-order", ArtifactDigest: digest, Generation: 1,
	}
	repairBinding := orderBinding
	repairBinding.ProviderID = "logical-repair"
	environment := map[core.CanonicalID]core.CanonicalID{
		"order_entry": "development",
		"repair_intake": "staging",
	}
	expected := map[core.CanonicalID]composition.Binding{
		"order_entry": orderBinding,
		"repair_intake": repairBinding,
	}
	routes := map[core.CanonicalID]SelectedServiceCarrierRoute{
		"order_entry": generationCarrierRoute(orderBinding, "tei.quote.shared"),
		"repair_intake": generationCarrierRoute(repairBinding, "tei.quote.shared"),
	}
	repairRoute := routes["repair_intake"]
	repairRoute.Plan.Environment = "staging"
	repairRoute.Plan.RuntimePolicyFingerprint = pruntime.PolicyFingerprint(repairRoute.Plan)
	routes["repair_intake"] = repairRoute

	process := &generationCarrierProcess{}
	verifier := &generationCarrierVerifier{}
	accepted := &generationCarrierAcceptance{}
	runtimeHost := pruntime.Registry{
		Processes: process, Verifier: verifier, ServiceAcceptance: accepted,
	}
	reads := map[core.CanonicalID]int{}
	var cancelAfterLookup context.CancelFunc
	current := func(
		_ context.Context, app, env core.CanonicalID, pinned composition.Binding,
	) (SelectedServiceCarrierRoute, error) {
		reads[app]++
		if environment[app] != env || expected[app] != pinned {
			return SelectedServiceCarrierRoute{}, errors.New("private Host application route mismatch")
		}
		route, ok := routes[app]
		if !ok {
			return SelectedServiceCarrierRoute{}, errors.New("private Host route source unavailable")
		}
		if cancelAfterLookup != nil {
			cancelAfterLookup()
		}
		return route, nil
	}
	makeInvoker := func(app core.CanonicalID, binding composition.Binding) *SelectedServiceCarrierInvoker {
		t.Helper()
		source, err := BindHostApplicationEnvironmentRoute(app, environment[app], current)
		if err != nil {
			t.Fatal(err)
		}
		router, err := composition.NewRouter(binding, composition.NewScope())
		if err != nil {
			t.Fatal(err)
		}
		coordinator, err := composition.NewCoordinator(router, func(actual composition.Binding) error {
			if actual != binding {
				return errors.New("private Host generation guard denied")
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		invoker, err := NewSelectedServiceCarrierInvoker(coordinator, runtimeHost, source)
		if err != nil {
			t.Fatal(err)
		}
		return invoker
	}
	order := makeInvoker("order_entry", orderBinding)
	repair := makeInvoker("repair_intake", repairBinding)
	payload := json.RawMessage(`{"quantity":2}`)
	invoke := func(ctx context.Context, invoker *SelectedServiceCarrierInvoker, id string) error {
		t.Helper()
		_, err := invoker.Invoke(ctx, id, payload)
		return err
	}
	check := func(orderReads, repairReads, acceptance, verification, children int) {
		t.Helper()
		if reads["order_entry"] != orderReads || reads["repair_intake"] != repairReads ||
			accepted.reads != acceptance || verifier.calls != verification || process.count() != children {
			t.Fatalf("source/Runtime calls: routes=%v accepted=%d verified=%d child=%d; want %d/%d/%d/%d/%d",
				reads, accepted.reads, verifier.calls, process.count(),
				orderReads, repairReads, acceptance, verification, children)
		}
	}

	// The caller's Context can contain arbitrary untrusted labels, but the
	// Host-owned Application/Environment pair was captured at composition time.
	type spoofKey struct{}
	for _, tc := range []struct {
		invoker *SelectedServiceCarrierInvoker
		id      string
	}{
		{order, "order-first"},
		{repair, "repair-first"},
	} {
		ctx := context.WithValue(context.Background(), spoofKey{}, "production/other_application")
		if err := invoke(ctx, tc.invoker, tc.id); err != nil {
			t.Fatal(err)
		}
	}
	check(1, 1, 2, 2, 2)

	// Removing the order Application from the CURRENT source fails closed
	// without revoking the physical Provider used by repair.
	saved := routes["order_entry"]
	delete(routes, "order_entry")
	if err := invoke(context.Background(), order, "order-withdrawn"); !errors.Is(err, ErrServiceCarrierGenerationUnavailable) {
		t.Fatalf("withdrawn Host Application accessed shared Provider: %v", err)
	}
	check(2, 1, 2, 2, 2)
	if err := invoke(context.Background(), repair, "repair-still-bound"); err != nil {
		t.Fatalf("unrelated Host lost a valid current binding: %v", err)
	}
	check(2, 2, 3, 3, 3)
	routes["order_entry"] = saved

	// Even a self-consistent plan fingerprint cannot authorize a plan from
	// another Environment: the trusted Host scope must match the current plan.
	wrongEnvironment := saved
	wrongEnvironment.Plan.Environment = "production"
	wrongEnvironment.Plan.RuntimePolicyFingerprint = pruntime.PolicyFingerprint(wrongEnvironment.Plan)
	routes["order_entry"] = wrongEnvironment
	if err := invoke(context.Background(), order, "wrong-environment"); !errors.Is(err, ErrServiceCarrierGenerationUnavailable) {
		t.Fatalf("wrong Environment Plan was admitted: %v", err)
	}
	check(3, 2, 3, 3, 3)
	routes["order_entry"] = saved

	// Source lookup can race cancellation. Even if it returns a valid route
	// after cancellation, acceptance/verification/process must not run.
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancelAfterLookup = cancel
	if err := invoke(cancelCtx, order, "cancel-after-lookup"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Host did not fail closed on canceled current read: %v", err)
	}
	cancelAfterLookup = nil
	check(4, 2, 3, 3, 3)

	if err := invoke(context.Background(), order, "order-restored"); err != nil {
		t.Fatalf("valid current source could not recover: %v", err)
	}
	check(5, 2, 4, 4, 4)
}

func TestHostBoundApplicationEnvironmentRejectsMissingScope(t *testing.T) {
	current := func(context.Context, core.CanonicalID, core.CanonicalID, composition.Binding) (SelectedServiceCarrierRoute, error) {
		return SelectedServiceCarrierRoute{}, nil
	}
	for _, tc := range []struct {
		name   string
		app    core.CanonicalID
		env    core.CanonicalID
		reader CurrentApplicationServiceRoute
	}{
		{"missing application", "", "development", current},
		{"missing environment", "order_entry", "", current},
		{"missing source", "order_entry", "development", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := BindHostApplicationEnvironmentRoute(tc.app, tc.env, tc.reader); !errors.Is(err, ErrServiceCarrierGenerationInvalid) {
				t.Fatalf("unbound Host route accepted: %v", err)
			}
		})
	}
}
