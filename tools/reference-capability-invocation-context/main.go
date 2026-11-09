package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	contextReceiptSchema = "tei.reference.capability-invocation-context-receipt/v0"
)

type ContextSlotContract struct {
	ID                           string
	Version                      string
	ForbidDirectResultReflection bool
}

type ContextSlotRequirement struct {
	ContractID      string
	ContractVersion string
}

type ContextSlotBinding struct {
	ContractID        string
	ContractVersion   string
	BindingReceiptRef string
	Value             []byte
}

type HostExecutionContext struct {
	Bindings          []ContextSlotBinding
	ReusePartitionRef string
}

type ContextBindingReceipt struct {
	ContractID        string `json:"contractId"`
	ContractVersion   string `json:"contractVersion"`
	BindingReceiptRef string `json:"bindingReceiptRef"`
}

type InvocationContextReceipt struct {
	Schema                string                  `json:"schema"`
	SemanticRequestDigest string                  `json:"semanticRequestDigest"`
	ResultDigest          string                  `json:"resultDigest"`
	ReusePartitionRef     string                  `json:"reusePartitionRef,omitempty"`
	Bindings              []ContextBindingReceipt `json:"bindings,omitempty"`
	AuthorityLimits       []string                `json:"authorityLimits"`
}

type InvocationContextExecution struct {
	ResultPayload []byte                   `json:"resultPayload"`
	Receipt       InvocationContextReceipt `json:"receipt"`
}

type ContextSlotView struct {
	values map[string]ContextSlotBinding
}

func (v ContextSlotView) Lookup(contractID string) ([]byte, bool) {
	binding, ok := v.values[contractID]
	if !ok {
		return nil, false
	}
	return append([]byte(nil), binding.Value...), true
}

func (v ContextSlotView) ContractVersion(contractID string) (string, bool) {
	binding, ok := v.values[contractID]
	if !ok {
		return "", false
	}
	return binding.ContractVersion, true
}

type NextInvocation func(
	context.Context,
	ContextSlotView,
	[]byte,
) ([]byte, error)

func invokeWithExecutionContext(
	parent context.Context,
	slotContracts []ContextSlotContract,
	requirements []ContextSlotRequirement,
	host HostExecutionContext,
	semanticRequest []byte,
	next NextInvocation,
) (InvocationContextExecution, error) {
	if parent == nil {
		return InvocationContextExecution{}, fmt.Errorf("invocation context requires a control context")
	}
	if next == nil {
		return InvocationContextExecution{}, fmt.Errorf("invocation context requires a downstream invoker")
	}

	// Keep the request identity tied to the bytes handed to the downstream
	// invoker, even if the caller changes its own slice during invocation.
	semanticRequestSnapshot := append([]byte(nil), semanticRequest...)

	contracts, err := normalizeContextSlotContracts(slotContracts)
	if err != nil {
		return InvocationContextExecution{}, err
	}
	required, err := normalizeContextSlotRequirements(requirements, contracts)
	if err != nil {
		return InvocationContextExecution{}, err
	}
	bindings, err := normalizeRequiredHostBindings(host.Bindings, required)
	if err != nil {
		return InvocationContextExecution{}, err
	}

	if len(required) > 0 && strings.TrimSpace(host.ReusePartitionRef) == "" {
		return InvocationContextExecution{}, fmt.Errorf(
			"execution context with required slots needs a Host-issued reuse partition ref",
		)
	}

	view := ContextSlotView{values: map[string]ContextSlotBinding{}}
	receipts := make([]ContextBindingReceipt, 0, len(required))
	for _, requirement := range required {
		binding, ok := bindings[requirement.ContractID]
		if !ok {
			return InvocationContextExecution{}, fmt.Errorf(
				"required execution context slot %s@%s is missing",
				requirement.ContractID,
				requirement.ContractVersion,
			)
		}
		if binding.ContractVersion != requirement.ContractVersion {
			return InvocationContextExecution{}, fmt.Errorf(
				"execution context slot %s version mismatch: required=%s current=%s",
				requirement.ContractID,
				requirement.ContractVersion,
				binding.ContractVersion,
			)
		}
		view.values[requirement.ContractID] = binding
		receipts = append(receipts, ContextBindingReceipt{
			ContractID:        binding.ContractID,
			ContractVersion:   binding.ContractVersion,
			BindingReceiptRef: binding.BindingReceiptRef,
		})
	}

	// Host-issued receipt identifiers are declared non-secret, but prevent the
	// simplest misbinding where a required confidential slot's raw value is
	// embedded verbatim into any field that will appear in the receipt.
	// This is a defense-in-depth literal check, not proof that the reference
	// is non-secret (encoded, hashed, or transformed leaks remain possible).
	for _, requirement := range required {
		if !contracts[requirement.ContractID].ForbidDirectResultReflection {
			continue
		}
		secret := view.values[requirement.ContractID].Value
		if bytes.Contains([]byte(host.ReusePartitionRef), secret) {
			return InvocationContextExecution{}, fmt.Errorf(
				"execution context slot %s directly exposes a confidential value in reuse partition ref",
				requirement.ContractID,
			)
		}
		for _, receipt := range receipts {
			if bytes.Contains([]byte(receipt.BindingReceiptRef), secret) {
				return InvocationContextExecution{}, fmt.Errorf(
					"execution context slot %s directly exposes a confidential value in binding receipt ref",
					requirement.ContractID,
				)
			}
		}
	}

	// Do not enter the downstream invoker when cancellation is already known.
	// Cancellation after this check is still conveyed by the control context.
	if err := parent.Err(); err != nil {
		return InvocationContextExecution{}, fmt.Errorf("invocation control context ended before dispatch: %w", err)
	}
	// Snapshot only control signals, not the original parent Context object.
	// A Provider callback may share this Go package and inspect concrete fields;
	// storing parent here would let it bypass Value() and read ambient secrets.
	deadline, hasDeadline := parent.Deadline()
	control := controlOnlyContext{
		deadline:    deadline,
		hasDeadline: hasDeadline,
		done:        parent.Done(),
		errFn:       parent.Err,
	}
	// Keep the Host snapshot used for leak detection inaccessible from a
	// potentially mutable same-package callback. Copy both map and bytes;
	// a shallow copy lets the Provider erase the original comparison value.
	providerBindings := make(map[string]ContextSlotBinding, len(view.values))
	for id, binding := range view.values {
		binding.Value = append([]byte(nil), binding.Value...)
		providerBindings[id] = binding
	}
	providerView := ContextSlotView{values: providerBindings}
	// Provider errors and panic values may execute untrusted formatting code or
	// contain raw secrets. For confidential slots, neither may cross the gate.
	confidential := false
	for _, requirement := range required {
		if contracts[requirement.ContractID].ForbidDirectResultReflection {
			confidential = true
			break
		}
	}
	requestForProvider := append([]byte(nil), semanticRequestSnapshot...)
	var resultPayload []byte
	if confidential {
		resultPayload, err = invokeConfidentialProvider(next, control, providerView, requestForProvider)
	} else {
		resultPayload, err = next(control, providerView, requestForProvider)
	}
	// Do not return a result, error object, or receipt after trusted control
	// cancellation. Check before inspecting a potentially untrusted provider
	// error; a provider cannot forge this state by wrapping context.Canceled.
	if controlErr := control.Err(); controlErr != nil {
		return InvocationContextExecution{}, controlErr
	}
	if err != nil {
		if confidential {
			// Never call Error(), Unwrap(), or custom formatting on an untrusted
			// provider error: those methods themselves can panic or reveal secrets.
			return InvocationContextExecution{}, fmt.Errorf(
				"downstream invocation failed (confidential error detail withheld)",
			)
		}
		return InvocationContextExecution{}, err
	}

	if leak := directContextReflection(required, contracts, view, resultPayload, "result"); leak != nil {
		return InvocationContextExecution{}, leak
	}

	requestSum := sha256.Sum256(semanticRequestSnapshot)
	resultSum := sha256.Sum256(resultPayload)
	sort.Slice(receipts, func(i, j int) bool {
		return receipts[i].ContractID < receipts[j].ContractID
	})

	return InvocationContextExecution{
		ResultPayload: append([]byte(nil), resultPayload...),
		Receipt: InvocationContextReceipt{
			Schema:                contextReceiptSchema,
			SemanticRequestDigest: "sha256:" + hex.EncodeToString(requestSum[:]),
			ResultDigest:          "sha256:" + hex.EncodeToString(resultSum[:]),
			ReusePartitionRef:     strings.TrimSpace(host.ReusePartitionRef),
			Bindings:              receipts,
			AuthorityLimits: []string{
				"context-gate-does-not-select-capability-provider-or-scope",
				"context-gate-does-not-prove-downstream-invocation-is-read-only",
				"context-gate-relies-on-capability-request-shape-to-exclude-execution-context-fields",
				"context-requirements-are-trusted-contract-input-not-consumer-payload",
				"context-slot-binding-is-not-canonical-acceptance",
				"context-slot-binding-is-not-objective-truth",
				"context-slot-binding-does-not-grant-additional-authority",
				"control-context-does-not-expose-caller-values",
				"binding-receipt-does-not-contain-slot-value",
				"binding-and-reuse-refs-are-host-declared-non-secret-not-validated",
				"reuse-partition-is-host-input-not-authorization-proof",
				"direct-reflection-check-does-not-prove-general-non-disclosure",
			},
		},
	}, nil
}

// invokeConfidentialProvider prevents a synchronous provider panic from
// escaping into caller logging or stack traces. Never format the panic value.
// Panics in other goroutines and fatal runtime errors are outside this guard.
func invokeConfidentialProvider(
	next NextInvocation,
	control context.Context,
	view ContextSlotView,
	request []byte,
) (payload []byte, err error) {
	// A bare recover() != nil check misses panic(nil) under the legacy
	// panicnil setting (including GO111MODULE=off builds). Record whether
	// the provider returned normally rather than trusting the panic value.
	completed := false
	defer func() {
		if !completed {
			_ = recover() // Do not inspect or format the panic payload.
			payload = nil
			err = fmt.Errorf("downstream invocation failed (confidential error detail withheld)")
		}
	}()
	payload, err = next(control, view, request)
	completed = true
	return payload, err
}

func directContextReflection(
	required []ContextSlotRequirement,
	contracts map[string]ContextSlotContract,
	view ContextSlotView,
	surface []byte,
	surfaceName string,
) error {
	for _, requirement := range required {
		contract := contracts[requirement.ContractID]
		if !contract.ForbidDirectResultReflection {
			continue
		}
		value := view.values[requirement.ContractID].Value
		if len(value) > 0 && bytes.Contains(surface, value) {
			return fmt.Errorf(
				"execution context slot %s was directly reflected into %s",
				requirement.ContractID,
				surfaceName,
			)
		}
	}
	return nil
}

func normalizeContextSlotContracts(
	values []ContextSlotContract,
) (map[string]ContextSlotContract, error) {
	out := make(map[string]ContextSlotContract, len(values))
	for i, value := range values {
		value.ID = strings.TrimSpace(value.ID)
		value.Version = strings.TrimSpace(value.Version)
		if value.ID == "" || value.Version == "" {
			return nil, fmt.Errorf("context slot contract[%d] requires id and version", i)
		}
		if _, duplicate := out[value.ID]; duplicate {
			return nil, fmt.Errorf("context slot contract %s is declared more than once", value.ID)
		}
		out[value.ID] = value
	}
	return out, nil
}

func normalizeContextSlotRequirements(
	values []ContextSlotRequirement,
	contracts map[string]ContextSlotContract,
) ([]ContextSlotRequirement, error) {
	out := make([]ContextSlotRequirement, 0, len(values))
	seen := map[string]struct{}{}
	for i, value := range values {
		value.ContractID = strings.TrimSpace(value.ContractID)
		value.ContractVersion = strings.TrimSpace(value.ContractVersion)
		if value.ContractID == "" || value.ContractVersion == "" {
			return nil, fmt.Errorf("context slot requirement[%d] requires contract id and version", i)
		}
		contract, ok := contracts[value.ContractID]
		if !ok {
			return nil, fmt.Errorf("context slot requirement %s has no contract", value.ContractID)
		}
		if contract.Version != value.ContractVersion {
			return nil, fmt.Errorf(
				"context slot requirement %s version %s has no exact contract",
				value.ContractID,
				value.ContractVersion,
			)
		}
		if _, duplicate := seen[value.ContractID]; duplicate {
			return nil, fmt.Errorf("context slot requirement %s is duplicated", value.ContractID)
		}
		seen[value.ContractID] = struct{}{}
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ContractID < out[j].ContractID
	})
	return out, nil
}

func normalizeRequiredHostBindings(
	values []ContextSlotBinding,
	required []ContextSlotRequirement,
) (map[string]ContextSlotBinding, error) {
	requiredIDs := make(map[string]struct{}, len(required))
	for _, requirement := range required {
		requiredIDs[requirement.ContractID] = struct{}{}
	}

	out := make(map[string]ContextSlotBinding, len(required))
	for i, value := range values {
		value.ContractID = strings.TrimSpace(value.ContractID)
		if _, needed := requiredIDs[value.ContractID]; !needed {
			continue
		}
		value.ContractVersion = strings.TrimSpace(value.ContractVersion)
		value.BindingReceiptRef = strings.TrimSpace(value.BindingReceiptRef)
		if value.ContractVersion == "" ||
			value.BindingReceiptRef == "" ||
			len(value.Value) == 0 {
			return nil, fmt.Errorf(
				"required execution context binding[%d] requires version, receipt, and value",
				i,
			)
		}
		if _, duplicate := out[value.ContractID]; duplicate {
			return nil, fmt.Errorf("execution context slot %s is bound more than once", value.ContractID)
		}
		value.Value = append([]byte(nil), value.Value...)
		out[value.ContractID] = value
	}
	return out, nil
}

type controlOnlyContext struct {
	deadline    time.Time
	hasDeadline bool
	done        <-chan struct{}
	errFn       func() error
}

func (c controlOnlyContext) Deadline() (time.Time, bool) {
	return c.deadline, c.hasDeadline
}

func (c controlOnlyContext) Done() <-chan struct{} {
	return c.done
}

func (c controlOnlyContext) Err() error {
	return c.errFn()
}

func (c controlOnlyContext) Value(any) any {
	return nil
}

func main() {}
