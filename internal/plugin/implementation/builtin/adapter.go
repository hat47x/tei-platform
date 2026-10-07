package builtin

import (
	"context"
	"fmt"

	"tei/internal/plugin"
	"tei/internal/plugin/implementation"
	pruntime "tei/internal/plugin/runtime"
)

type staticResponseBuiltin struct {
	source                implementation.Source
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
		payload := append([]byte(nil), operation.Payload...)
		return pruntime.Response{Status: pruntime.ResponseOK, Payload: payload}, nil
	}
	return pruntime.Response{}, fmt.Errorf("TEI-native implementation does not support operation %q", req.Operation)
}

// New adapts a validated TEI-native implementation Source to the existing
// runtime.Builtin ABI. Host acceptance and build provenance remain outside this
// adapter and are supplied through runtime.BuiltinRegistration.
func New(source implementation.Source, definition plugin.Definition) (pruntime.Builtin, error) {
	if err := implementation.ValidateAgainstDefinition(source, definition); err != nil {
		return nil, err
	}
	normalized, err := implementation.Normalize(source)
	if err != nil {
		return nil, err
	}
	return staticResponseBuiltin{
		source:                normalized,
		descriptorFingerprint: plugin.DefinitionFingerprint(definition),
	}, nil
}
