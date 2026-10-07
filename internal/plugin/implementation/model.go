package implementation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"tei/internal/core"
	"tei/internal/deployment/executor"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

const (
	StaticResponseProfileV1 = "static-response-v1"
	RealizationFormatVersion = 1
)

// Operation is one deterministic response selected by Request.Operation.
// Payload is normalized JSON and cannot access ambient Host state.
type Operation struct {
	Name    string          `json:"name"`
	Payload json.RawMessage `json:"payload"`
}

// Source is the first constrained TEI-native Plugin implementation profile.
// It intentionally implements exactly one existing Capability Contract.
type Source struct {
	Plugin         plugin.Identity       `json:"plugin"`
	Profile        string                `json:"profile"`
	CapabilityKind plugin.CapabilityKind `json:"capabilityKind"`
	CapabilityID   core.CanonicalID      `json:"capabilityId"`
	Operations     []Operation           `json:"operations"`
}

func (s Source) Identity() string {
	return string(s.Plugin.ID) + "@" + s.Plugin.Version + "/" + string(s.CapabilityKind) + "/" + string(s.CapabilityID)
}

func (s Source) Validate() error {
	if _, err := core.ParseCanonicalID(string(s.Plugin.ID)); err != nil {
		return fmt.Errorf("implementation plugin id: %w", err)
	}
	if !core.ValidNumericVersion(s.Plugin.Version) {
		return fmt.Errorf("implementation plugin version %q must be major.minor.patch", s.Plugin.Version)
	}
	if s.Profile != StaticResponseProfileV1 {
		return fmt.Errorf("unsupported plugin implementation profile %q", s.Profile)
	}
	if !s.CapabilityKind.Valid() {
		return fmt.Errorf("unsupported plugin implementation capability kind %q", s.CapabilityKind)
	}
	if s.CapabilityKind != plugin.CapabilityDeploymentExecutor {
		return fmt.Errorf(
			"%s profile currently supports only %s capability, got %s",
			StaticResponseProfileV1,
			plugin.CapabilityDeploymentExecutor,
			s.CapabilityKind,
		)
	}
	if _, err := core.ParseCanonicalID(string(s.CapabilityID)); err != nil {
		return fmt.Errorf("implementation capability id: %w", err)
	}
	if len(s.Operations) != 1 {
		return fmt.Errorf("%s profile requires exactly one execute operation", StaticResponseProfileV1)
	}
	seen := map[string]struct{}{}
	for _, operation := range s.Operations {
		name := strings.TrimSpace(operation.Name)
		if name == "" {
			return fmt.Errorf("plugin implementation operation name is required")
		}
		if name != "execute" {
			return fmt.Errorf("%s profile supports only execute operation, got %q", StaticResponseProfileV1, name)
		}
		if _, exists := seen[name]; exists {
			return fmt.Errorf("plugin implementation operation %q is declared more than once", name)
		}
		seen[name] = struct{}{}
		canonical, err := canonicalJSON(operation.Payload)
		if err != nil {
			return fmt.Errorf("plugin implementation operation %q payload: %w", name, err)
		}
		if err := validateDeploymentExecutorResponse(canonical); err != nil {
			return fmt.Errorf("plugin implementation operation %q payload: %w", name, err)
		}
	}
	return nil
}

func Normalize(s Source) (Source, error) {
	if err := s.Validate(); err != nil {
		return Source{}, err
	}
	out := s
	out.Operations = make([]Operation, len(s.Operations))
	for i, operation := range s.Operations {
		payload, err := canonicalJSON(operation.Payload)
		if err != nil {
			return Source{}, err
		}
		out.Operations[i] = Operation{Name: strings.TrimSpace(operation.Name), Payload: payload}
	}
	return out, nil
}

func Fingerprint(s Source) (string, error) {
	normalized, err := Normalize(s)
	if err != nil {
		return "", err
	}
	raw, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

type RealizationManifest struct {
	FormatVersion     int                   `json:"formatVersion"`
	SourceFingerprint string                `json:"sourceFingerprint"`
	Source            Source                `json:"source"`
}

// Materialize emits deterministic bytes that identify the in-process
// realization. The digest can be bound to BuildArtifactLock and Runtime Plugin
// Lock without claiming that it is a standalone executable file.
func Materialize(s Source) ([]byte, string, error) {
	normalized, err := Normalize(s)
	if err != nil {
		return nil, "", err
	}
	sourceFingerprint, err := Fingerprint(normalized)
	if err != nil {
		return nil, "", err
	}
	manifest := RealizationManifest{
		FormatVersion:     RealizationFormatVersion,
		SourceFingerprint: sourceFingerprint,
		Source:            normalized,
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		return nil, "", err
	}
	raw = append(raw, '\n')
	sum := sha256.Sum256(raw)
	return raw, "sha256:" + hex.EncodeToString(sum[:]), nil
}

type staticResponseBuiltin struct {
	source                Source
	descriptorFingerprint string
}

func (b staticResponseBuiltin) Identity() plugin.Identity {
	return b.source.Plugin
}

func (b staticResponseBuiltin) DescriptorFingerprint() string {
	return b.descriptorFingerprint
}

func (b staticResponseBuiltin) Invoke(_ context.Context, req pruntime.Request) (pruntime.Response, error) {
	if req.Plugin != b.source.Plugin {
		return pruntime.Response{}, fmt.Errorf("TEI-native implementation plugin identity mismatch")
	}
	if req.CapabilityKind != b.source.CapabilityKind || req.CapabilityID != b.source.CapabilityID {
		return pruntime.Response{}, fmt.Errorf(
			"TEI-native implementation does not provide capability %s %s",
			req.CapabilityKind, req.CapabilityID,
		)
	}
	for _, operation := range b.source.Operations {
		if operation.Name != req.Operation {
			continue
		}
		payload := append(json.RawMessage(nil), operation.Payload...)
		return pruntime.Response{Status: pruntime.ResponseOK, Payload: payload}, nil
	}
	return pruntime.Response{}, fmt.Errorf("TEI-native implementation does not support operation %q", req.Operation)
}



func ValidateAgainstDefinition(source Source, definition plugin.Definition) error {
	normalized, err := Normalize(source)
	if err != nil {
		return err
	}
	if definition.ID != normalized.Plugin.ID || definition.Version != normalized.Plugin.Version {
		return fmt.Errorf(
			"implementation source %s does not match plugin definition %s@%s",
			normalized.Identity(), definition.ID, definition.Version,
		)
	}
	if definition.Runtime.Kind != plugin.RuntimeBuiltin {
		return fmt.Errorf("TEI-native static response Reference requires builtin runtime")
	}
	var matches int
	var matched plugin.Capability
	for _, capability := range definition.Provides {
		if capability.Kind == normalized.CapabilityKind && capability.ID == normalized.CapabilityID {
			matches++
			matched = capability
		}
	}
	if matches != 1 {
		return fmt.Errorf(
			"plugin definition must provide capability %s %s exactly once",
			normalized.CapabilityKind, normalized.CapabilityID,
		)
	}
	if len(matched.Actions) != 1 || !matched.Actions[0].Valid() {
		return fmt.Errorf("%s profile requires exactly one valid deployment action", StaticResponseProfileV1)
	}
	if len(matched.Permissions) != 0 {
		return fmt.Errorf("%s profile does not accept capability permissions", StaticResponseProfileV1)
	}
	return nil
}

// NewBuiltin adapts the constrained implementation Source to the existing
// runtime.Builtin ABI. Registration acceptance remains a Host responsibility.
func NewBuiltin(source Source, definition plugin.Definition) (pruntime.Builtin, error) {
	if err := ValidateAgainstDefinition(source, definition); err != nil {
		return nil, err
	}
	normalized, err := Normalize(source)
	if err != nil {
		return nil, err
	}
	return staticResponseBuiltin{
		source:                normalized,
		descriptorFingerprint: plugin.DefinitionFingerprint(definition),
	}, nil
}



func validateDeploymentExecutorResponse(raw json.RawMessage) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var result executor.Result
	if err := decoder.Decode(&result); err != nil {
		return fmt.Errorf("deployment executor response schema: %w", err)
	}
	if result.ExecutorID != "" || result.ActionID != "" || result.IdempotencyKey != "" || result.PluginInvocation != nil {
		return fmt.Errorf("deployment executor response must not set Host-owned invocation identity or provenance")
	}
	switch result.Outcome {
	case executor.OutcomeAccepted, executor.OutcomeRejected, executor.OutcomeUnknown:
	default:
		return fmt.Errorf("deployment executor response outcome %q is invalid", result.Outcome)
	}
	if strings.TrimSpace(result.AttemptID) == "" {
		return fmt.Errorf("deployment executor response attemptId is required")
	}
	return nil
}

func canonicalJSON(raw json.RawMessage) (json.RawMessage, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("JSON payload is empty")
	}
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("JSON payload contains multiple values")
		}
		return nil, err
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return canonical, nil
}
