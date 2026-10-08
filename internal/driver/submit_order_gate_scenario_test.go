package driver_test

import (
	"context"
	"fmt"
	"math/big"
	"path/filepath"
	"testing"
	"time"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/ir"
	"tei/internal/realization"
	"tei/internal/security/legacyauth"
	"tei/internal/security/legacyauthorization"
	"tei/internal/semantic"
	"tei/internal/semantic/capabilityinvoke"
	"tei/internal/semantic/validationeval"
)

type submitOrderValidationSource struct {
	quantity    *big.Rat
	lineNumber  *big.Rat
	lineNumbers []string
}

func (s submitOrderValidationSource) Number(attribute core.CanonicalID) (*big.Rat, error) {
	switch attribute {
	case "sales.order_line.quantity":
		if s.quantity == nil {
			return nil, fmt.Errorf("sales.order_line.quantityの値がありません")
		}
		return new(big.Rat).Set(s.quantity), nil
	case "sales.order_line.line_number":
		if s.lineNumber == nil {
			return nil, fmt.Errorf("sales.order_line.line_numberの値がありません")
		}
		return new(big.Rat).Set(s.lineNumber), nil
	default:
		return nil, fmt.Errorf("未登録の数値Attributeです: %s", attribute)
	}
}

func (s submitOrderValidationSource) RelationKeys(relation, attribute core.CanonicalID) ([]string, error) {
	if relation != "sales.order.lines" || attribute != "sales.order_line.line_number" {
		return nil, fmt.Errorf("未登録のRelation Validation入力です: %s / %s", relation, attribute)
	}
	return append([]string(nil), s.lineNumbers...), nil
}

type submitOrderGateResult struct {
	Validation        map[core.CanonicalID]validationeval.Result
	Authorization     capabilityinvoke.BusinessResult
	Command           capabilityinvoke.BusinessResult
	CommandWasInvoked bool
}

func TestOrderEntrySubmitScenarioExecutesCommandOnlyAfterValidationAndAuthorization(t *testing.T) {
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
			switch ref {
			case "credential-ref://legacy/order-editor":
				return presented == "editor-secret", nil
			case "credential-ref://legacy/order-viewer":
				return presented == "viewer-secret", nil
			default:
				return false, fmt.Errorf("未登録のcredential参照です: %s", ref)
			}
		}),
		Now: func() time.Time { return now },
	}
	authorization := legacyauthorization.Adapter{Store: store, Rule: authorizationRule}

	quantityPositive, lineNumberUnique := orderEntryValidationPair(t, world)
	validSource := submitOrderValidationSource{
		quantity:    mustRat(t, "2.5"),
		lineNumbers: []string{"1", "2"},
	}

	t.Run("ValidationとAuthorizationを通過したORDER_EDITORだけがCommandを実行する", func(t *testing.T) {
		commandCalls := 0
		authorizationCalls := 0
		adapters := submitScenarioAdapters(authentication, authorization, &authorizationCalls, &commandCalls)

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
		if result.Validation[quantityPositive.ID] != validationeval.ResultSatisfied || result.Validation[lineNumberUnique.ID] != validationeval.ResultSatisfied {
			t.Fatalf("Validation result=%#v", result.Validation)
		}
		if result.Authorization.Outcome != "allowed" || authorizationCalls != 1 {
			t.Fatalf("Authorization result=%#v calls=%d", result.Authorization, authorizationCalls)
		}
		if !result.CommandWasInvoked || commandCalls != 1 {
			t.Fatalf("Command invocation=%v calls=%d", result.CommandWasInvoked, commandCalls)
		}
		if result.Command.Capability != "sales.order.submit" || result.Command.Mode != capabilityinvoke.ResultOutput {
			t.Fatalf("Command Business Result=%#v", result.Command)
		}
	})

	t.Run("業務Validation不成立ではAuthorizationにもCommandにも進まない", func(t *testing.T) {
		commandCalls := 0
		authorizationCalls := 0
		adapters := submitScenarioAdapters(authentication, authorization, &authorizationCalls, &commandCalls)
		invalidSource := submitOrderValidationSource{
			quantity:    mustRat(t, "0"),
			lineNumbers: []string{"1", "2"},
		}

		result, err := runSubmitOrderGateScenario(
			context.Background(),
			resolver,
			adapters,
			"legacy-order-editor",
			"editor-secret",
			invalidSource,
			[]semantic.Validation{quantityPositive, lineNumberUnique},
		)
		if err != nil {
			t.Fatalf("NotSatisfiedをExecution Errorへ変換しています: %v", err)
		}
		if result.Validation[quantityPositive.ID] != validationeval.ResultNotSatisfied {
			t.Fatalf("quantity_positive result=%q", result.Validation[quantityPositive.ID])
		}
		if authorizationCalls != 0 || commandCalls != 0 || result.CommandWasInvoked {
			t.Fatalf("Validation不成立後に後続処理へ進んでいます: authorization=%d command=%d result=%#v", authorizationCalls, commandCalls, result)
		}
	})

	t.Run("Validation評価不能はExecution ErrorとなりAuthorizationにもCommandにも進まない", func(t *testing.T) {
		commandCalls := 0
		authorizationCalls := 0
		adapters := submitScenarioAdapters(authentication, authorization, &authorizationCalls, &commandCalls)
		missingValueSource := submitOrderValidationSource{lineNumbers: []string{"1", "2"}}

		result, err := runSubmitOrderGateScenario(
			context.Background(),
			resolver,
			adapters,
			"legacy-order-editor",
			"editor-secret",
			missingValueSource,
			[]semantic.Validation{quantityPositive, lineNumberUnique},
		)
		if err == nil {
			t.Fatalf("Validation評価不能がExecution Errorになりませんでした: result=%#v", result)
		}
		if _, ok := result.Validation[quantityPositive.ID]; ok {
			t.Fatalf("評価不能なValidationを成立・不成立として記録しています: %#v", result.Validation)
		}
		if authorizationCalls != 0 || commandCalls != 0 || result.CommandWasInvoked {
			t.Fatalf("Validation評価不能後に後続処理へ進んでいます: authorization=%d command=%d result=%#v", authorizationCalls, commandCalls, result)
		}
	})

	t.Run("Authorization deniedはExecution ErrorではなくCommand未実行の業務結果になる", func(t *testing.T) {
		commandCalls := 0
		authorizationCalls := 0
		adapters := submitScenarioAdapters(authentication, authorization, &authorizationCalls, &commandCalls)

		result, err := runSubmitOrderGateScenario(
			context.Background(),
			resolver,
			adapters,
			"legacy-order-viewer",
			"viewer-secret",
			validSource,
			[]semantic.Validation{quantityPositive, lineNumberUnique},
		)
		if err != nil {
			t.Fatalf("Authorization deniedをExecution Errorへ変換しています: %v", err)
		}
		if result.Authorization.Outcome != "denied" || authorizationCalls != 1 {
			t.Fatalf("Authorization result=%#v calls=%d", result.Authorization, authorizationCalls)
		}
		if commandCalls != 0 || result.CommandWasInvoked {
			t.Fatalf("Authorization denied後にCommandを実行しています: calls=%d result=%#v", commandCalls, result)
		}
	})
}

func runSubmitOrderGateScenario(
	ctx context.Context,
	resolver driver.Resolver,
	adapters driver.BindingAdapterResolver,
	loginID, credential string,
	source validationeval.Source,
	validations []semantic.Validation,
) (submitOrderGateResult, error) {
	result := submitOrderGateResult{Validation: make(map[core.CanonicalID]validationeval.Result, len(validations))}

	authentication, err := resolver.InvokeCapability(
		ctx,
		"order_entry",
		"security.authenticate",
		adapters,
		legacyauth.Input{LoginID: loginID, Credential: credential},
	)
	if err != nil {
		return result, err
	}
	if authentication.Result.Outcome != "authenticated" {
		return result, nil
	}
	subject, ok := authentication.Result.Payload.(legacyauth.AuthenticatedSubject)
	if !ok || subject.SubjectID == "" {
		return result, fmt.Errorf("AuthenticationのBusiness Resultにsubjectがありません: %#v", authentication.Result)
	}

	for _, rule := range validations {
		evaluated, err := validationeval.Evaluate(rule, source)
		if err != nil {
			return result, err
		}
		result.Validation[rule.ID] = evaluated
		if evaluated == validationeval.ResultNotSatisfied {
			return result, nil
		}
	}

	authorization, err := resolver.InvokeCapability(
		ctx,
		"order_entry",
		"security.authorize",
		adapters,
		legacyauthorization.Input{
			SubjectID: subject.SubjectID,
			Action:    "execute",
			Resource:  "order_entry.submit_order",
		},
	)
	if err != nil {
		return result, err
	}
	result.Authorization = authorization.Result
	if authorization.Result.Outcome != "allowed" {
		return result, nil
	}

	command, err := resolver.InvokeCapability(
		ctx,
		"order_entry",
		"sales.order.submit",
		adapters,
		map[string]any{"orderId": "order-1001"},
	)
	if err != nil {
		return result, err
	}
	result.Command = command.Result
	result.CommandWasInvoked = true
	return result, nil
}

func submitScenarioAdapters(
	authentication legacyauth.Adapter,
	authorization legacyauthorization.Adapter,
	authorizationCalls, commandCalls *int,
) driver.BindingAdapterResolver {
	return bindingAdapterResolverFunc(func(_ context.Context, binding realization.CustomGoBinding) (capabilityinvoke.Adapter, error) {
		switch binding.Identity() {
		case "order_entry/security.authenticate":
			return authentication, nil
		case "order_entry/security.authorize":
			return capabilityAdapterFunc(func(ctx context.Context, capability core.CanonicalID, input any) (capabilityinvoke.BusinessResult, error) {
				(*authorizationCalls)++
				return authorization.Invoke(ctx, capability, input)
			}), nil
		case "order_entry/sales.order.submit":
			return capabilityAdapterFunc(func(_ context.Context, capability core.CanonicalID, input any) (capabilityinvoke.BusinessResult, error) {
				(*commandCalls)++
				return capabilityinvoke.BusinessResult{
					Capability: capability,
					Mode:       capabilityinvoke.ResultOutput,
					Payload:    input,
				}, nil
			}), nil
		default:
			return nil, fmt.Errorf("未登録のBindingです: %s", binding.Identity())
		}
	})
}

// orderEntryValidationsは、order_entryが`uses validation`で宣言したValidationを、宣言順に返す（ADR-0029）。
// Scenarioは評価するValidationの一覧を自分で持たない。ここで固定するIDは選択の入力ではなく、
// 現在のReferenceが宣言している内容に対する期待値である。
func orderEntryValidations(t *testing.T, world ir.World) []semantic.Validation {
	t.Helper()
	rules, err := driver.NewResolver(world).ResolveApplicationValidations("order_entry")
	if err != nil {
		t.Fatalf("order_entryのValidation選択を解決できません: %v", err)
	}
	if len(rules) != 3 ||
		rules[0].ID != "sales.order_line.quantity_positive" ||
		rules[1].ID != "sales.order.lines.line_number_unique" ||
		rules[2].ID != "sales.order_line.line_number_positive" {
		t.Fatalf("order_entryが宣言したValidationが想定と異なります: %+v", rules)
	}
	return rules
}

func orderEntryValidationPair(t *testing.T, world ir.World) (semantic.Validation, semantic.Validation) {
	t.Helper()
	rules := orderEntryValidations(t, world)
	return rules[0], rules[1]
}

func submitScenarioValidation(t *testing.T, world ir.World, id core.CanonicalID) semantic.Validation {
	t.Helper()
	for _, namespace := range world.Semantics {
		for _, rule := range namespace.Validations {
			if rule.ID == id {
				return rule
			}
		}
	}
	t.Fatalf("Semantic Validation %sがReference Worldにありません", id)
	return semantic.Validation{}
}

func mustRat(t *testing.T, text string) *big.Rat {
	t.Helper()
	value, ok := new(big.Rat).SetString(text)
	if !ok {
		t.Fatalf("test fixtureの数値%qを解釈できません", text)
	}
	return value
}
