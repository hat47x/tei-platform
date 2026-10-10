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

	control := controlOnlyContext{parent: parent}
	resultPayload, err := next(
		control,
		view,
		append([]byte(nil), semanticRequest...),
	)
	if err != nil {
		return InvocationContextExecution{}, err
	}

	for _, requirement := range required {
		contract := contracts[requirement.ContractID]
		if !contract.ForbidDirectResultReflection {
			continue
		}
		value := view.values[requirement.ContractID].Value
		if len(value) > 0 && bytes.Contains(resultPayload, value) {
			return InvocationContextExecution{}, fmt.Errorf(
				"execution context slot %s was directly reflected into result",
				requirement.ContractID,
			)
		}
	}

	requestSum := sha256.Sum256(semanticRequest)
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
				"context-slot-binding-is-not-canonical-acceptance",
				"context-slot-binding-is-not-objective-truth",
				"context-slot-binding-does-not-grant-additional-authority",
				"control-context-does-not-expose-caller-values",
				"binding-receipt-does-not-contain-slot-value",
				"reuse-partition-is-host-input-not-authorization-proof",
				"direct-reflection-check-does-not-prove-general-non-disclosure",
			},
		},
	}, nil
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
	parent context.Context
}

func (c controlOnlyContext) Deadline() (time.Time, bool) {
	return c.parent.Deadline()
}

func (c controlOnlyContext) Done() <-chan struct{} {
	return c.parent.Done()
}

func (c controlOnlyContext) Err() error {
	return c.parent.Err()
}

func (c controlOnlyContext) Value(any) any {
	return nil
}

func main() {}
