package driver_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/realization"
	"tei/internal/security/legacyauth"
	"tei/internal/security/legacyauthorization"
	"tei/internal/semantic"
	"tei/internal/semantic/capabilityinvoke"
)

// syntheticOrderStateStoreは、CommandとObservationを別境界として実証するためだけのtest fixtureです。
// 実外部状態のEvidenceではありません。
type syntheticOrderStateStore struct {
	states map[string]string
}

type syntheticOrderStateReader struct {
	store *syntheticOrderStateStore
	err   error
	calls int
}

func (r *syntheticOrderStateReader) Identity() core.CanonicalID {
	return "synthetic.order_state.read"
}

func (r *syntheticOrderStateReader) Read(_ context.Context, orderID string) (string, error) {
	r.calls++
	if r.err != nil {
		return "", r.err
	}
	state, ok := r.store.states[orderID]
	if !ok {
		return "", nil
	}
	return state, nil
}

func TestOrderEntrySubmitScenarioObservesStateThroughSeparateExplicitRead(t *testing.T) {
	world := referenceWorld(t)
	resolver := driver.NewResolver(world)
	store, err := legacyauth.LoadFixtureStore(referenceLegacyAuthFixture(t))
	if err != nil {
		t.Fatalf("legacy authentication fixtureを読み込めません: %v", err)
	}
	authorizationRule, err := legacyauthorization.LoadFixtureRule(filepath.Join(filepath.Dir(referenceLegacyAuthFixture(t)), "authorization.json"))
	if err != nil {
		t.Fatalf("legacy authorization fixtureを読み込めません: %v", err)
	}

	now := time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC)
	authentication := legacyauth.Adapter{
		Store: store,
		Verifier: driverCredentialVerifierFunc(func(_ context.Context, ref, presented string) (bool, error) {
			if ref != "credential-ref://legacy/order-editor" {
				return false, fmt.Errorf("未登録のcredential参照です: %s", ref)
			}
			return presented == "editor-secret", nil
		}),
		Now: func() time.Time { return now },
	}
	authorization := legacyauthorization.Adapter{Store: store, Rule: authorizationRule}

	quantityPositive, lineNumberUnique := orderEntryValidationPair(t, world)
	validSource := submitOrderValidationSource{
		quantity:    mustRat(t, "2.5"),
		lineNumbers: []string{"1", "2"},
	}

	t.Run("Command acknowledgementだけではObserved stateを成立させない", func(t *testing.T) {
		state := &syntheticOrderStateStore{states: map[string]string{"order-1001": "pending"}}
		reader := &syntheticOrderStateReader{store: state}
		adapters := syntheticObservationScenarioAdapters(authentication, authorization, state)

		result, err := runSubmitOrderGateScenario(
			context.Background(),
			resolver,
			adapters,
			"legacy-order-editor",
			"editor-secret",
			validSource,
			[]semantic.Validation{quantityPositive, lineNumberUnique},
		)
		if err != nil {
			t.Fatalf("submit Scenarioを完了できません: %v", err)
		}
		if !result.CommandWasInvoked {
			t.Fatal("Commandが実行されていません")
		}
		if reader.calls != 0 {
			t.Fatalf("Command実行だけでObservation readが暗黙実行されています: calls=%d", reader.calls)
		}
		if result.Command.Capability != "sales.order.submit" || result.Command.Mode != capabilityinvoke.ResultOutput {
			t.Fatalf("Command Business Result=%#v", result.Command)
		}
		ack, ok := result.Command.Payload.(map[string]any)
		if !ok || ack["accepted"] != true {
			t.Fatalf("Command acknowledgement=%#v", result.Command.Payload)
		}
		if _, exists := ack["observedState"]; exists {
			t.Fatalf("Command payloadへObserved stateを混入させています: %#v", ack)
		}

		observation, err := driver.ExecuteObservationRead(
			context.Background(),
			string(reader.Identity()),
			driver.ObservationEvidenceSynthetic,
			func(ctx context.Context) (any, error) {
				return reader.Read(ctx, "order-1001")
			},
		)
		if err != nil {
			t.Fatalf("明示Observation readに失敗しました: %v", err)
		}
		if reader.calls != 1 {
			t.Fatalf("明示Observation readの呼出回数=%d", reader.calls)
		}
		if observation.ReadIdentity != "synthetic.order_state.read" || observation.Nature != driver.ObservationEvidenceSynthetic {
			t.Fatalf("Observation read execution=%#v", observation)
		}
		observed, ok := observation.Result.(string)
		if !ok || observed != "submitted" {
			t.Fatalf("Observed state=%#v, want submitted", observation.Result)
		}
	})

	t.Run("Observation Execution ErrorをCommand Business Resultへ畳まない", func(t *testing.T) {
		state := &syntheticOrderStateStore{states: map[string]string{"order-1001": "pending"}}
		reader := &syntheticOrderStateReader{
			store: state,
			err:   fmt.Errorf("synthetic read unavailable"),
		}
		adapters := syntheticObservationScenarioAdapters(authentication, authorization, state)

		result, err := runSubmitOrderGateScenario(
			context.Background(),
			resolver,
			adapters,
			"legacy-order-editor",
			"editor-secret",
			validSource,
			[]semantic.Validation{quantityPositive, lineNumberUnique},
		)
		if err != nil {
			t.Fatalf("CommandまでのScenarioにObservation errorが逆流しています: %v", err)
		}
		if !result.CommandWasInvoked || result.Command.Mode != capabilityinvoke.ResultOutput {
			t.Fatalf("Command Business Result=%#v", result.Command)
		}

		observation, observationErr := driver.ExecuteObservationRead(
			context.Background(),
			string(reader.Identity()),
			driver.ObservationEvidenceSynthetic,
			func(ctx context.Context) (any, error) {
				return reader.Read(ctx, "order-1001")
			},
		)
		if observationErr == nil {
			t.Fatal("Observation Execution Errorが失われています")
		}
		if observation.ReadIdentity != "synthetic.order_state.read" || observation.Nature != driver.ObservationEvidenceSynthetic {
			t.Fatalf("Observation失敗時のread identityが失われています: %#v", observation)
		}
		if observation.Result != nil {
			t.Fatalf("失敗したObservationに正常resultを保持しています: %#v", observation.Result)
		}
		if result.Command.Mode != capabilityinvoke.ResultOutput {
			t.Fatalf("Observation失敗後にCommand Business Resultを書き換えています: %#v", result.Command)
		}
	})
}

func syntheticObservationScenarioAdapters(
	authentication legacyauth.Adapter,
	authorization legacyauthorization.Adapter,
	state *syntheticOrderStateStore,
) driver.BindingAdapterResolver {
	return bindingAdapterResolverFunc(func(_ context.Context, binding realization.CustomGoBinding) (capabilityinvoke.Adapter, error) {
		switch binding.Identity() {
		case "order_entry/security.authenticate":
			return authentication, nil
		case "order_entry/security.authorize":
			return authorization, nil
		case "order_entry/sales.order.submit":
			return capabilityAdapterFunc(func(_ context.Context, capability core.CanonicalID, input any) (capabilityinvoke.BusinessResult, error) {
				values, ok := input.(map[string]any)
				if !ok {
					return capabilityinvoke.BusinessResult{}, fmt.Errorf("Command inputの型が不正です: %T", input)
				}
				orderID, ok := values["orderId"].(string)
				if !ok || orderID == "" {
					return capabilityinvoke.BusinessResult{}, fmt.Errorf("Command inputにorderIdがありません")
				}
				state.states[orderID] = "submitted"
				return capabilityinvoke.BusinessResult{
					Capability: capability,
					Mode:       capabilityinvoke.ResultOutput,
					Payload: map[string]any{
						"accepted": true,
					},
				}, nil
			}), nil
		default:
			return nil, fmt.Errorf("未登録のBindingです: %s", binding.Identity())
		}
	})
}
