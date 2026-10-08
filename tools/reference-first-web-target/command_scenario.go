package main

import (
	"context"
	"fmt"
	"math/big"
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

type commandScenarioInput struct {
	LoginID    string
	Credential string
	From       string
	RowIndex   int
}

type commandScenarioResult struct {
	ScenarioID      string                       `json:"scenarioId"`
	ApplicationID   string                       `json:"applicationId"`
	Stage           string                       `json:"stage"`
	Authentication  *businessResultProjection    `json:"authentication,omitempty"`
	Read            *readScenarioResult          `json:"read,omitempty"`
	Validations     []validationResultProjection `json:"validations,omitempty"`
	Authorization   *businessResultProjection    `json:"authorization,omitempty"`
	Command         *businessResultProjection    `json:"command,omitempty"`
	Observation     *observationProjection       `json:"observation,omitempty"`
	AuthorityLimits []string                     `json:"authorityLimits"`
	Unresolved      []string                     `json:"unresolved"`
}

type validationResultProjection struct {
	Validation string `json:"validation"`
	Result     string `json:"result"`
}

type businessResultProjection struct {
	Capability string         `json:"capability"`
	Mode       string         `json:"mode"`
	Outcome    string         `json:"outcome,omitempty"`
	Payload    map[string]any `json:"payload,omitempty"`
}

type observationProjection struct {
	ReadIdentity string `json:"readIdentity"`
	Nature       string `json:"nature"`
	Result       any    `json:"result"`
}

type commandScenarioRunner interface {
	RunCommand(context.Context, commandScenarioInput) (commandScenarioResult, error)
}

type referenceCommandScenarioRunner struct {
	resolver      driver.Resolver
	read          readScenarioRunner
	store         *legacyauth.FixtureStore
	authorization *legacyauthorization.FixtureRule
	now           time.Time
}

func newReferenceCommandScenarioRunner(
	world ir.World,
	usersFixturePath string,
	authorizationFixturePath string,
) (*referenceCommandScenarioRunner, error) {
	store, err := legacyauth.LoadFixtureStore(usersFixturePath)
	if err != nil {
		return nil, err
	}
	authorization, err := legacyauthorization.LoadFixtureRule(authorizationFixturePath)
	if err != nil {
		return nil, err
	}

	return &referenceCommandScenarioRunner{
		resolver:      driver.NewResolver(world),
		read:          newReferenceReadScenarioRunner(world),
		store:         store,
		authorization: authorization,
		now:           time.Date(2026, 9, 19, 0, 0, 0, 0, time.UTC),
	}, nil
}

// ApplicationID returns the Application this runner executes (see referenceReadScenarioRunner).
func (r *referenceCommandScenarioRunner) ApplicationID() string {
	return "order_entry"
}

func (r *referenceCommandScenarioRunner) RunCommand(
	ctx context.Context,
	input commandScenarioInput,
) (commandScenarioResult, error) {
	result := commandScenarioResult{
		ScenarioID:    "order-entry.submit-order-web-slice",
		ApplicationID: "order_entry",
		Stage:         "authentication",
		AuthorityLimits: []string{
			"credential-is-runtime-input-and-is-not-retained-in-result",
			"legacy-auth-fixtures-remain-adapter-owned",
			"validation-rules-come-from-canonical-semantic-world",
			"authorization-denial-is-business-outcome-not-execution-error",
			"command-business-result-does-not-imply-observed-state",
			"observation-is-a-separate-explicit-read",
			"synthetic-command-state-is-not-external-production-state",
		},
		Unresolved: []string{
			"actual-functional-evidence-not-yet-acquired",
			"visual-evidence-not-yet-acquired",
			"actual-external-command-observation-not-acquired",
		},
	}
	if input.LoginID == "" {
		return result, fmt.Errorf("loginId is required")
	}
	if input.Credential == "" {
		return result, fmt.Errorf("credential is required")
	}
	if input.From == "" {
		return result, fmt.Errorf("from is required")
	}
	if input.RowIndex < 0 {
		return result, fmt.Errorf("row index must be non-negative")
	}

	authenticationAdapter := legacyauth.Adapter{
		Store:    r.store,
		Verifier: referenceCredentialVerifier{},
		Now:      func() time.Time { return r.now },
	}
	authentication, err := r.resolver.InvokeCapability(
		ctx,
		core.MustCanonicalID("order_entry"),
		core.MustCanonicalID("security.authenticate"),
		bindingAdapterResolverFunc(func(_ context.Context, binding realization.CustomGoBinding) (capabilityinvoke.Adapter, error) {
			if binding.Identity() != "order_entry/security.authenticate" {
				return nil, fmt.Errorf("unexpected Authentication Binding %s", binding.Identity())
			}
			return authenticationAdapter, nil
		}),
		legacyauth.Input{
			LoginID:    input.LoginID,
			Credential: input.Credential,
		},
	)
	if err != nil {
		return result, err
	}
	projectedAuthentication, err := projectBusinessResult(authentication.Result)
	if err != nil {
		return result, err
	}
	result.Authentication = &projectedAuthentication
	if authentication.Result.Outcome != "authenticated" {
		result.Stage = "authentication-denied"
		return result, nil
	}

	subject, ok := authentication.Result.Payload.(legacyauth.AuthenticatedSubject)
	if !ok || subject.SubjectID == "" {
		return result, fmt.Errorf("authenticated result does not contain subject identity")
	}

	result.Stage = "read"
	readResult, err := r.read.Run(ctx, readScenarioInput{
		From:     input.From,
		RowIndex: input.RowIndex,
	})
	if err != nil {
		return result, err
	}
	result.Read = &readResult

	result.Stage = "validation"
	// どのValidationを評価するかは、ApplicationがCanonicalに宣言した選択から得る（ADR-0029）。
	// Target側に一覧を持たない。宣言が無ければ、空を成功とみなさず、Execution Errorとして返す。
	rules, err := r.resolver.ResolveApplicationValidations(core.MustCanonicalID("order_entry"))
	if err != nil {
		return result, err
	}
	for _, rule := range rules {
		id := rule.ID
		evaluated, err := evaluateCommandValidation(rule, readResult.OrderLineRows)
		if err != nil {
			return result, err
		}
		result.Validations = append(result.Validations, validationResultProjection{
			Validation: string(id),
			Result:     string(evaluated),
		})
		if evaluated != validationeval.ResultSatisfied {
			result.Stage = "validation-not-satisfied"
			return result, nil
		}
	}

	result.Stage = "authorization"
	authorizationAdapter := legacyauthorization.Adapter{
		Store: r.store,
		Rule:  r.authorization,
	}
	authorization, err := r.resolver.InvokeCapability(
		ctx,
		core.MustCanonicalID("order_entry"),
		core.MustCanonicalID("security.authorize"),
		bindingAdapterResolverFunc(func(_ context.Context, binding realization.CustomGoBinding) (capabilityinvoke.Adapter, error) {
			if binding.Identity() != "order_entry/security.authorize" {
				return nil, fmt.Errorf("unexpected Authorization Binding %s", binding.Identity())
			}
			return authorizationAdapter, nil
		}),
		legacyauthorization.Input{
			SubjectID: subject.SubjectID,
			Action:    "execute",
			Resource:  "order_entry.submit_order",
		},
	)
	if err != nil {
		return result, err
	}
	projectedAuthorization, err := projectBusinessResult(authorization.Result)
	if err != nil {
		return result, err
	}
	result.Authorization = &projectedAuthorization
	if authorization.Result.Outcome != "allowed" {
		result.Stage = "authorization-denied"
		return result, nil
	}

	orderIDValue, ok := semanticRowValueByAttribute(
		readResult.SelectedOrder,
		"sales.order.order_id",
	)
	if !ok {
		return result, fmt.Errorf("selected order does not contain sales.order.order_id")
	}
	orderID, ok := orderIDValue.(int64)
	if !ok {
		return result, fmt.Errorf("sales.order.order_id has unexpected type %T", orderIDValue)
	}

	state := map[int64]string{orderID: "pending"}
	commandAdapter := capabilityAdapterFunc(func(
		_ context.Context,
		capability core.CanonicalID,
		raw any,
	) (capabilityinvoke.BusinessResult, error) {
		values, ok := raw.(map[string]any)
		if !ok {
			return capabilityinvoke.BusinessResult{}, fmt.Errorf("submit input has unexpected type %T", raw)
		}
		actualOrderID, ok := values["orderId"].(int64)
		if !ok || actualOrderID != orderID {
			return capabilityinvoke.BusinessResult{}, fmt.Errorf("submit orderId does not match selected order")
		}
		if actualOrderID == 1003 {
			return capabilityinvoke.BusinessResult{}, fmt.Errorf("synthetic submit adapter unavailable for order %d", actualOrderID)
		}
		state[actualOrderID] = "submitted"
		return capabilityinvoke.BusinessResult{
			Capability: capability,
			Mode:       capabilityinvoke.ResultOutput,
			Payload: map[string]any{
				"accepted": true,
				"orderId":  actualOrderID,
			},
		}, nil
	})

	result.Stage = "command"
	commandResolution, err := r.resolver.ResolvePattern(
		core.MustCanonicalID("order_entry"),
		core.MustCanonicalID("order_entry.submit_order"),
	)
	if err != nil {
		return result, err
	}
	if commandResolution.Capability == nil {
		return result, fmt.Errorf("submit-order Pattern did not resolve a Capability")
	}
	command, err := r.resolver.InvokeCapability(
		ctx,
		core.MustCanonicalID("order_entry"),
		commandResolution.Capability.ID,
		bindingAdapterResolverFunc(func(_ context.Context, binding realization.CustomGoBinding) (capabilityinvoke.Adapter, error) {
			if binding.Identity() != "order_entry/sales.order.submit" {
				return nil, fmt.Errorf("unexpected submit Binding %s", binding.Identity())
			}
			return commandAdapter, nil
		}),
		map[string]any{"orderId": orderID},
	)
	if err != nil {
		return result, err
	}
	projectedCommand, err := projectBusinessResult(command.Result)
	if err != nil {
		return result, err
	}
	result.Command = &projectedCommand

	result.Stage = "observation"
	observation, err := driver.ExecuteObservationRead(
		ctx,
		"synthetic.order_state.read",
		driver.ObservationEvidenceSynthetic,
		func(context.Context) (any, error) {
			value, exists := state[orderID]
			if !exists {
				return nil, fmt.Errorf("order state not found")
			}
			return value, nil
		},
	)
	if err != nil {
		return result, err
	}
	result.Observation = &observationProjection{
		ReadIdentity: observation.ReadIdentity,
		Nature:       string(observation.Nature),
		Result:       observation.Result,
	}
	result.Stage = "completed"
	return result, nil
}

type referenceCredentialVerifier struct{}

func (referenceCredentialVerifier) Verify(_ context.Context, ref, presented string) (bool, error) {
	expected := map[string]string{
		"credential-ref://legacy/order-editor":   "editor-secret",
		"credential-ref://legacy/order-viewer":   "viewer-secret",
		"credential-ref://legacy/order-locked":   "locked-secret",
		"credential-ref://legacy/order-disabled": "disabled-secret",
	}
	value, exists := expected[ref]
	if !exists {
		return false, fmt.Errorf("unknown credential reference %s", ref)
	}
	return presented == value, nil
}

type bindingAdapterResolverFunc func(
	context.Context,
	realization.CustomGoBinding,
) (capabilityinvoke.Adapter, error)

func (f bindingAdapterResolverFunc) ResolveAdapter(
	ctx context.Context,
	binding realization.CustomGoBinding,
) (capabilityinvoke.Adapter, error) {
	return f(ctx, binding)
}

type capabilityAdapterFunc func(
	context.Context,
	core.CanonicalID,
	any,
) (capabilityinvoke.BusinessResult, error)

func (f capabilityAdapterFunc) Invoke(
	ctx context.Context,
	capability core.CanonicalID,
	input any,
) (capabilityinvoke.BusinessResult, error) {
	return f(ctx, capability, input)
}

// evaluateCommandValidation applies a record-level positive rule to every line in
// the selected order. A single aggregate result is returned per Canonical rule.
// Relation-wide uniqueness is evaluated once over the complete collection.
// Missing values or an empty collection must not silently count as success.
func evaluateCommandValidation(
	rule semantic.Validation,
	orderLines []semanticRowProjection,
) (validationeval.Result, error) {
	if rule.Kind != semantic.ValidationPositive {
		return validationeval.Evaluate(rule, commandValidationSource{orderLines: orderLines})
	}
	if len(orderLines) == 0 {
		return "", fmt.Errorf("cannot evaluate positive validation %s over an empty order-line collection", rule.ID)
	}
	for index, line := range orderLines {
		result, err := validationeval.Evaluate(rule, commandValidationSource{
			orderLines: []semanticRowProjection{line},
		})
		if err != nil {
			return "", fmt.Errorf("validation %s on order line %d: %w", rule.ID, index, err)
		}
		if result != validationeval.ResultSatisfied {
			return result, nil
		}
	}
	return validationeval.ResultSatisfied, nil
}

type commandValidationSource struct {
	orderLines []semanticRowProjection
}

func (s commandValidationSource) Number(attribute core.CanonicalID) (*big.Rat, error) {
	if len(s.orderLines) == 0 {
		return nil, fmt.Errorf("order-line collection is empty")
	}
	// Number is a scalar read. A collection of multiple lines must never be
	// silently reduced to its first element; the caller must evaluate each
	// line explicitly (evaluateCommandValidation).
	if len(s.orderLines) != 1 {
		return nil, fmt.Errorf("numeric validation source requires exactly one order line, got %d", len(s.orderLines))
	}
	value, ok := semanticRowValueByAttribute(s.orderLines[0], string(attribute))
	if !ok {
		return nil, fmt.Errorf("attribute %s is not present", attribute)
	}
	switch typed := value.(type) {
	case *big.Rat:
		return new(big.Rat).Set(typed), nil
	case string:
		number, ok := new(big.Rat).SetString(typed)
		if !ok {
			return nil, fmt.Errorf("attribute %s value %q is not numeric", attribute, typed)
		}
		return number, nil
	case int:
		return new(big.Rat).SetInt64(int64(typed)), nil
	case int64:
		return new(big.Rat).SetInt64(typed), nil
	default:
		return nil, fmt.Errorf("attribute %s has unsupported numeric type %T", attribute, value)
	}
}

func (s commandValidationSource) RelationKeys(
	relation core.CanonicalID,
	attribute core.CanonicalID,
) ([]string, error) {
	if relation != core.MustCanonicalID("sales.order.lines") {
		return nil, fmt.Errorf("unsupported relation %s", relation)
	}
	result := make([]string, 0, len(s.orderLines))
	for rowIndex, row := range s.orderLines {
		value, ok := semanticRowValueByAttribute(row, string(attribute))
		if !ok {
			return nil, fmt.Errorf("row %d does not contain %s", rowIndex, attribute)
		}
		if value == nil {
			return nil, fmt.Errorf("row %d contains null key %s for relation %s", rowIndex, attribute, relation)
		}
		result = append(result, fmt.Sprint(value))
	}
	return result, nil
}

func semanticRowValueByAttribute(row semanticRowProjection, attribute string) (any, bool) {
	for _, value := range row.Attributes {
		if value.Attribute == attribute {
			return value.Value, true
		}
	}
	return nil, false
}

func projectBusinessResult(source capabilityinvoke.BusinessResult) (businessResultProjection, error) {
	result := businessResultProjection{
		Capability: string(source.Capability),
		Mode:       string(source.Mode),
		Outcome:    string(source.Outcome),
	}
	switch payload := source.Payload.(type) {
	case legacyauth.AuthenticatedSubject:
		result.Payload = map[string]any{"subjectId": payload.SubjectID}
	case legacyauth.AuthenticationDenial:
		result.Payload = map[string]any{"reason": payload.Reason}
	case legacyauthorization.AuthorizationGrant:
		result.Payload = map[string]any{
			"subjectId": payload.SubjectID,
			"action":    payload.Action,
			"resource":  payload.Resource,
		}
	case legacyauthorization.AuthorizationDenial:
		result.Payload = map[string]any{"reason": payload.Reason}
	case map[string]any:
		result.Payload = payload
	case nil:
	default:
		return businessResultProjection{}, fmt.Errorf(
			"unsupported Business Result payload type %T for %s",
			source.Payload,
			source.Capability,
		)
	}
	return result, nil
}
