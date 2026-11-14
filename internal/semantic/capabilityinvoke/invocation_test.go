package capabilityinvoke_test

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/capabilityinvoke"
	"tei/internal/workspace"
)

type adapterFunc func(context.Context, core.CanonicalID, any) (capabilityinvoke.BusinessResult, error)

func (f adapterFunc) Invoke(ctx context.Context, capability core.CanonicalID, input any) (capabilityinvoke.BusinessResult, error) {
	return f(ctx, capability, input)
}

func TestReferenceAuthenticationReturnsNamedBusinessOutcomes(t *testing.T) {
	capability := referenceCapability(t, "security.authenticate")
	adapter := adapterFunc(func(_ context.Context, id core.CanonicalID, input any) (capabilityinvoke.BusinessResult, error) {
		switch input {
		case "correct":
			return capabilityinvoke.BusinessResult{
				Capability: id,
				Mode:       capabilityinvoke.ResultOutcome,
				Outcome:    "authenticated",
				Payload:    map[string]string{"subjectId": "subject-001"},
			}, nil
		case "mismatch":
			return capabilityinvoke.BusinessResult{
				Capability: id,
				Mode:       capabilityinvoke.ResultOutcome,
				Outcome:    "denied",
				Payload:    map[string]string{"reason": "credential_mismatch"},
			}, nil
		case "locked":
			return capabilityinvoke.BusinessResult{
				Capability: id,
				Mode:       capabilityinvoke.ResultOutcome,
				Outcome:    "denied",
				Payload:    map[string]string{"reason": "account_locked"},
			}, nil
		default:
			return capabilityinvoke.BusinessResult{}, errors.New("authentication database unavailable")
		}
	})

	for _, tt := range []struct {
		name        string
		input       string
		wantOutcome core.LocalName
	}{
		{name: "認証成功", input: "correct", wantOutcome: "authenticated"},
		{name: "credential不一致", input: "mismatch", wantOutcome: "denied"},
		{name: "account lock", input: "locked", wantOutcome: "denied"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result, err := capabilityinvoke.Invoke(context.Background(), capability, adapter, tt.input)
			if err != nil {
				t.Fatalf("業務OutcomeをExecution Errorとして扱っています: %v", err)
			}
			if result.Mode != capabilityinvoke.ResultOutcome || result.Outcome != tt.wantOutcome {
				t.Fatalf("result=%#v want outcome=%q", result, tt.wantOutcome)
			}
		})
	}
}

func TestReferenceAuthenticationExecutionFailureDoesNotBecomeDenied(t *testing.T) {
	capability := referenceCapability(t, "security.authenticate")
	adapter := adapterFunc(func(context.Context, core.CanonicalID, any) (capabilityinvoke.BusinessResult, error) {
		return capabilityinvoke.BusinessResult{}, errors.New("database connection failed")
	})

	result, err := capabilityinvoke.Invoke(context.Background(), capability, adapter, "correct")
	if err == nil {
		t.Fatalf("実行障害がerrorになりませんでした: result=%#v", result)
	}
	if result.Outcome == "denied" {
		t.Fatalf("実行障害がdeniedへ変換されています: result=%#v err=%v", result, err)
	}
}

func TestUndefinedOutcomeIsExecutionContractError(t *testing.T) {
	capability := referenceCapability(t, "security.authenticate")
	adapter := adapterFunc(func(_ context.Context, id core.CanonicalID, _ any) (capabilityinvoke.BusinessResult, error) {
		return capabilityinvoke.BusinessResult{
			Capability: id,
			Mode:       capabilityinvoke.ResultOutcome,
			Outcome:    "unknown",
			Payload:    map[string]string{"reason": "unexpected"},
		}, nil
	})

	_, err := capabilityinvoke.Invoke(context.Background(), capability, adapter, nil)
	if err == nil || !strings.Contains(err.Error(), "does not define outcome") {
		t.Fatalf("an undefined Outcome was accepted as a normal Business Result: %v", err)
	}
}

func TestExistingSingleOutputCapabilityUsesSameInvocationBoundary(t *testing.T) {
	capability := referenceCapability(t, "sales.order.submit")
	adapter := adapterFunc(func(_ context.Context, id core.CanonicalID, _ any) (capabilityinvoke.BusinessResult, error) {
		return capabilityinvoke.BusinessResult{
			Capability: id,
			Mode:       capabilityinvoke.ResultOutput,
			Payload:    map[string]string{"orderId": "order-001"},
		}, nil
	})

	result, err := capabilityinvoke.Invoke(context.Background(), capability, adapter, nil)
	if err != nil {
		t.Fatalf("既存output Capabilityを同じInvocation境界で扱えません: %v", err)
	}
	if result.Mode != capabilityinvoke.ResultOutput || result.Outcome != "" {
		t.Fatalf("既存output Capabilityの結果境界が変わっています: %#v", result)
	}
}

func TestOutcomeCannotBeReturnedForSingleOutputCapability(t *testing.T) {
	capability := referenceCapability(t, "sales.order.submit")
	adapter := adapterFunc(func(_ context.Context, id core.CanonicalID, _ any) (capabilityinvoke.BusinessResult, error) {
		return capabilityinvoke.BusinessResult{
			Capability: id,
			Mode:       capabilityinvoke.ResultOutcome,
			Outcome:    "submitted",
			Payload:    map[string]string{"orderId": "order-001"},
		}, nil
	})

	_, err := capabilityinvoke.Invoke(context.Background(), capability, adapter, nil)
	if err == nil {
		t.Fatal("既存output Capabilityへ暗黙のOutcomeを追加できてしまいました")
	}
}

// Post-dispatch Result contract failures are not proof of non-execution.
//
// An effectful Provider could have performed work before returning a malformed
// result. This boundary can reject the result but must not claim rollback,
// effect absence, or eligibility for automatic retry (#4495, EL-12).
func TestPostDispatchContractErrorPreservesAdapterCallEvidence(t *testing.T) {
	capability := referenceCapability(t, "security.authenticate")
	tests := []struct {
		name   string
		result capabilityinvoke.BusinessResult
		want   string
	}{
		{
			name: "undeclared outcome",
			result: capabilityinvoke.BusinessResult{
				Capability: capability.ID,
				Mode:       capabilityinvoke.ResultOutcome,
				Outcome:    "not_declared",
				Payload:    "unexpected",
			},
			want: "does not define outcome",
		},
		{
			name: "different capability",
			result: capabilityinvoke.BusinessResult{
				Capability: "wrong.capability",
				Mode:       capabilityinvoke.ResultOutcome,
				Outcome:    "denied",
				Payload:    "unexpected",
			},
			want: "different Capability",
		},
		{
			name: "wrong result mode",
			result: capabilityinvoke.BusinessResult{
				Capability: capability.ID,
				Mode:       capabilityinvoke.ResultOutput,
				Payload:    "unexpected",
			},
			want: "named-outcome contract",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			adapter := adapterFunc(func(context.Context, core.CanonicalID, any) (capabilityinvoke.BusinessResult, error) {
				calls++
				return tt.result, nil
			})
			result, err := capabilityinvoke.Invoke(context.Background(), capability, adapter, nil)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("post-dispatch contract violation was not rejected: result=%#v err=%v", result, err)
			}
			if calls != 1 {
				t.Fatalf("post-dispatch result violation must remain a call=1 case, got %d", calls)
			}
			if result.Capability != "" || result.Mode != "" || result.Outcome != "" || result.Payload != nil {
				t.Fatalf("invalid result was returned as accepted business output: %#v", result)
			}
		})
	}
}

// Cancellation returned by an Adapter occurs after dispatch. The invocation
// must preserve error identity without converting the result into a business
// Outcome or misclassifying this as the pre-dispatch call=0 case.
func TestPostDispatchCancellationKeepsAdapterCallEvidence(t *testing.T) {
	capability := referenceCapability(t, "sales.order.submit")
	calls := 0
	adapter := adapterFunc(func(context.Context, core.CanonicalID, any) (capabilityinvoke.BusinessResult, error) {
		calls++
		return capabilityinvoke.BusinessResult{}, context.Canceled
	})
	result, err := capabilityinvoke.Invoke(context.Background(), capability, adapter, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Adapter cancellation identity was lost: result=%#v err=%v", result, err)
	}
	if calls != 1 {
		t.Fatalf("Adapter cancellation after dispatch must remain a call=1 case, got %d", calls)
	}
	if result.Capability != "" || result.Mode != "" || result.Outcome != "" || result.Payload != nil {
		t.Fatalf("post-dispatch cancellation produced a business outcome: %#v", result)
	}
}

// Cancellation known before dispatch is a non-execution result, even for
// effectful single-output Capabilities. No Adapter invocation is permitted.
func TestCancelledContextDoesNotDispatchCapabilityAdapter(t *testing.T) {
	capability := referenceCapability(t, "sales.order.submit")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	adapter := adapterFunc(func(context.Context, core.CanonicalID, any) (capabilityinvoke.BusinessResult, error) {
		calls++
		return capabilityinvoke.BusinessResult{
			Capability: capability.ID,
			Mode:       capabilityinvoke.ResultOutput,
			Payload:    "unexpected",
		}, nil
	})

	result, err := capabilityinvoke.Invoke(ctx, capability, adapter, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-dispatch cancellation must remain an execution error: result=%#v err=%v", result, err)
	}
	if calls != 0 {
		t.Fatalf("cancelled request dispatched adapter %d times", calls)
	}
	if result.Capability != "" || result.Mode != "" || result.Outcome != "" || result.Payload != nil {
		t.Fatalf("cancelled request produced an invented business result: %#v", result)
	}
}

// Adapter errors retain identity through invocation; they are never interpreted
// as an accepted business denial or proof that an effect was rolled back.
func TestAdapterCancellationErrorDoesNotBecomeBusinessOutcome(t *testing.T) {
	capability := referenceCapability(t, "sales.order.submit")
	adapter := adapterFunc(func(context.Context, core.CanonicalID, any) (capabilityinvoke.BusinessResult, error) {
		return capabilityinvoke.BusinessResult{}, context.Canceled
	})
	result, err := capabilityinvoke.Invoke(context.Background(), capability, adapter, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("adapter cancellation error identity was lost: result=%#v err=%v", result, err)
	}
	if result.Outcome != "" || result.Mode != "" || result.Payload != nil {
		t.Fatalf("adapter failure became business result: %#v", result)
	}
}

// A malformed Capability must not reach an Adapter: checking only the returned
// BusinessResult would be too late for a potentially effectful implementation.
func TestMalformedResultContractRefusedBeforeAdapterDispatch(t *testing.T) {
	base := referenceCapability(t, "security.authenticate")
	cases := []struct {
		name    string
		corrupt func(*semantic.Capability)
		want    string
	}{
		{
			name:    "missing identity",
			corrupt: func(c *semantic.Capability) { c.ID = "" },
			want:    "identity is missing",
		},
		{
			name: "no result",
			corrupt: func(c *semantic.Capability) {
				c.Outcomes = nil
			},
			want: "has no result contract",
		},
		{
			name: "mixed output and outcome",
			corrupt: func(c *semantic.Capability) {
				c.Output = core.MustCanonicalID("std.text")
			},
			want: "both a single output and named outcomes",
		},
		{
			name: "empty outcome name",
			corrupt: func(c *semantic.Capability) {
				c.Outcomes[0].Name = ""
			},
			want: "incomplete named outcome",
		},
		{
			name: "empty outcome payload",
			corrupt: func(c *semantic.Capability) {
				c.Outcomes[0].Payload = ""
			},
			want: "incomplete named outcome",
		},
		{
			name: "duplicate outcome name",
			corrupt: func(c *semantic.Capability) {
				c.Outcomes[1].Name = c.Outcomes[0].Name
			},
			want: "duplicate outcome",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			capability := base
			capability.Outcomes = append([]semantic.Outcome(nil), base.Outcomes...)
			tc.corrupt(&capability)
			calls := 0
			adapter := adapterFunc(func(context.Context, core.CanonicalID, any) (capabilityinvoke.BusinessResult, error) {
				calls++
				return capabilityinvoke.BusinessResult{
					Capability: capability.ID,
					Mode:       capabilityinvoke.ResultOutcome,
					Outcome:    "authenticated",
					Payload:    map[string]string{"subjectId": "not-reached"},
				}, nil
			})
			result, err := capabilityinvoke.Invoke(context.Background(), capability, adapter, nil)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("malformed result contract was not rejected: result=%#v err=%v", result, err)
			}
			if calls != 0 {
				t.Fatalf("malformed result contract dispatched Adapter %d times", calls)
			}
			if result.Capability != "" || result.Mode != "" || result.Outcome != "" || result.Payload != nil {
				t.Fatalf("malformed contract produced a business result: %#v", result)
			}
		})
	}
}

func TestDirectBusinessResultValidationRejectsMixedResultContract(t *testing.T) {
	capability := referenceCapability(t, "security.authenticate")
	capability.Output = core.MustCanonicalID("std.text")
	result := capabilityinvoke.BusinessResult{
		Capability: capability.ID,
		Mode:       capabilityinvoke.ResultOutput,
		Payload:    "appears valid in output mode",
	}
	if err := capabilityinvoke.ValidateBusinessResult(capability, result); err == nil {
		t.Fatal("direct validation accepted a Capability with contradictory result modes")
	}
}

// The generic invocation layer cannot infer whether a single-output type is
// nullable. Payload shape/presence is checked against accepted meaning by the
// product-specific Adapter, not by this structural result-mode checker.
func TestSingleOutputPayloadPresenceIsNotInferredFromResultMode(t *testing.T) {
	capability := referenceCapability(t, "sales.order.submit")
	result := capabilityinvoke.BusinessResult{
		Capability: capability.ID,
		Mode:       capabilityinvoke.ResultOutput,
	}
	if err := capabilityinvoke.ValidateBusinessResult(capability, result); err != nil {
		t.Fatalf("result-mode validation claimed to know the output's nullability: %v", err)
	}
}

// Malformed declared inputs and effects are a pre-dispatch concern even when
// the result contract itself is valid. This test does not assert that the
// incoming input value is typed, authorized, or bound to a published Entry.
func TestMalformedInputOrEffectDeclarationNeverDispatchesAdapter(t *testing.T) {
	base := referenceCapability(t, "security.authenticate")
	cases := []struct {
		name   string
		mutate func(*semantic.Capability)
		want   string
	}{
		{
			name: "input name missing",
			mutate: func(c *semantic.Capability) {
				c.Inputs = []semantic.Parameter{{Name: "", Type: "std.text"}}
			},
			want: "incomplete input",
		},
		{
			name: "input type missing",
			mutate: func(c *semantic.Capability) {
				c.Inputs = []semantic.Parameter{{Name: "token", Type: ""}}
			},
			want: "incomplete input",
		},
		{
			name: "duplicate input names",
			mutate: func(c *semantic.Capability) {
				c.Inputs = []semantic.Parameter{
					{Name: "token", Type: "std.text"},
					{Name: "token", Type: "std.text"},
				}
			},
			want: "duplicate input",
		},
		{
			name: "unsupported data effect",
			mutate: func(c *semantic.Capability) {
				c.Effect = &semantic.DataEffect{Kind: "unknown", Data: "sales.order"}
			},
			want: "unsupported data effect",
		},
		{
			name: "effect data identity missing",
			mutate: func(c *semantic.Capability) {
				c.Effect = &semantic.DataEffect{Kind: semantic.DataEffectCreate}
			},
			want: "unsupported data effect",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			capability := base
			tc.mutate(&capability)
			calls := 0
			adapter := adapterFunc(func(context.Context, core.CanonicalID, any) (capabilityinvoke.BusinessResult, error) {
				calls++
				return capabilityinvoke.BusinessResult{
					Capability: capability.ID,
					Mode:       capabilityinvoke.ResultOutcome,
					Outcome:    "authenticated",
					Payload:    map[string]string{"subjectId": "unexpected"},
				}, nil
			})
			result, err := capabilityinvoke.Invoke(context.Background(), capability, adapter, nil)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("malformed declaration was admitted: result=%#v err=%v", result, err)
			}
			if calls != 0 {
				t.Fatalf("malformed declaration reached Go Adapter %d times", calls)
			}
			if result.Capability != "" || result.Mode != "" || result.Outcome != "" || result.Payload != nil {
				t.Fatalf("malformed declaration returned BusinessResult: %#v", result)
			}
		})
	}
}

func TestNilContextNeverDispatchesCapabilityAdapter(t *testing.T) {
	capability := referenceCapability(t, "sales.order.submit")
	calls := 0
	adapter := adapterFunc(func(context.Context, core.CanonicalID, any) (capabilityinvoke.BusinessResult, error) {
		calls++
		return capabilityinvoke.BusinessResult{
			Capability: capability.ID, Mode: capabilityinvoke.ResultOutput,
			Payload: "unexpected",
		}, nil
	})
	result, err := capabilityinvoke.Invoke(nil, capability, adapter, nil)
	if err == nil || !strings.Contains(err.Error(), "no context") {
		t.Fatalf("nil context was not rejected: result=%#v err=%v", result, err)
	}
	if calls != 0 {
		t.Fatalf("nil context reached Go Adapter %d times", calls)
	}
}

func TestResultOnlyValidatorDoesNotClaimInputOrEffectAcceptance(t *testing.T) {
	capability := referenceCapability(t, "sales.order.submit")
	capability.Inputs = []semantic.Parameter{
		{Name: "token", Type: "std.text"},
		{Name: "token", Type: "std.text"},
	}
	result := capabilityinvoke.BusinessResult{
		Capability: capability.ID,
		Mode:       capabilityinvoke.ResultOutput,
		Payload:    "a result mode, not input publication",
	}
	if err := capabilityinvoke.ValidateBusinessResult(capability, result); err != nil {
		t.Fatalf("result-only validation unexpectedly became input/Entry authority: %v", err)
	}
}

func referenceCapability(t *testing.T, id core.CanonicalID) semantic.Capability {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test fileの位置を取得できません")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatalf("Reference Workspaceの読み込みに失敗しました: %v", err)
	}
	for _, namespace := range loaded.World.Semantics {
		for _, capability := range namespace.Capabilities {
			if capability.ID == id {
				return capability
			}
		}
	}
	t.Fatalf("Semantic Capability %sがReference Workspaceに見つかりません", id)
	return semantic.Capability{}
}
