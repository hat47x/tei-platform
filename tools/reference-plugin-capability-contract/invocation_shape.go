package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

const (
	invocationShapeArtifactSchema = "tei.reference.invocation-shape/v0"
	invocationJSONShapeFormat      = "tei.reference.json-shape/v0"
	invocationOpaqueShapeFormat    = "tei.reference.opaque-shape/v0"
)

//go:embed shapes/*.json
var invocationShapeFS embed.FS

type InvocationShapeIssuer struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type InvocationShapeArtifact struct {
	Schema       string                `json:"schema"`
	ID           string                `json:"id"`
	Issuer       InvocationShapeIssuer `json:"issuer"`
	PayloadCodec string                `json:"payloadCodec"`
	Format       string                `json:"format"`
	Root         InvocationShapeNode   `json:"root"`
}

type InvocationShapeNode struct {
	Kind                 string                         `json:"kind"`
	Required             []string                       `json:"required,omitempty"`
	Properties           map[string]InvocationShapeNode `json:"properties,omitempty"`
	Items                *InvocationShapeNode           `json:"items,omitempty"`
	AdditionalProperties *InvocationShapeNode           `json:"additionalProperties,omitempty"`
}

func referenceInvocationShapeIssuer() InvocationShapeIssuer {
	return mustReferenceInvocationShape(
		"shapes/source-adapter-request-v0.json",
	).Issuer
}

func referenceInvocationShapeIssuerPointer() *InvocationShapeIssuer {
	issuer := referenceInvocationShapeIssuer()
	return &issuer
}

func invocationShapeFingerprint(artifact InvocationShapeArtifact) string {
	normalized := normalizeInvocationShapeArtifact(artifact)
	return fingerprint(normalized)
}

func normalizeInvocationShapeArtifact(
	artifact InvocationShapeArtifact,
) InvocationShapeArtifact {
	out := artifact
	out.Root = normalizeInvocationShapeNode(artifact.Root)
	return out
}

func normalizeInvocationShapeNode(node InvocationShapeNode) InvocationShapeNode {
	out := node
	out.Required = append([]string(nil), node.Required...)
	sort.Strings(out.Required)
	if node.Properties != nil {
		out.Properties = make(map[string]InvocationShapeNode, len(node.Properties))
		for name, child := range node.Properties {
			out.Properties[name] = normalizeInvocationShapeNode(child)
		}
	}
	if node.Items != nil {
		item := normalizeInvocationShapeNode(*node.Items)
		out.Items = &item
	}
	if node.AdditionalProperties != nil {
		additional := normalizeInvocationShapeNode(*node.AdditionalProperties)
		out.AdditionalProperties = &additional
	}
	return out
}

func validateInvocationShapeArtifact(artifact InvocationShapeArtifact) error {
	if artifact.Schema != invocationShapeArtifactSchema {
		return fmt.Errorf("invocation shape %s has unsupported schema %q", artifact.ID, artifact.Schema)
	}
	if strings.TrimSpace(artifact.ID) == "" || artifact.ID != strings.TrimSpace(artifact.ID) {
		return fmt.Errorf("invocation shape id %q is invalid", artifact.ID)
	}
	if artifact.Issuer.ID == "" || artifact.Issuer.Version == "" {
		return fmt.Errorf("invocation shape %s has incomplete issuer", artifact.ID)
	}
	if _, err := parseVersion(artifact.Issuer.Version); err != nil {
		return fmt.Errorf("invocation shape %s issuer version: %w", artifact.ID, err)
	}
	if artifact.PayloadCodec == "" {
		return fmt.Errorf("invocation shape %s has no payload codec", artifact.ID)
	}
	switch artifact.Format {
	case invocationJSONShapeFormat:
		if !strings.HasPrefix(artifact.PayloadCodec, "application/json") {
			return fmt.Errorf(
				"JSON invocation shape %s cannot use codec %q",
				artifact.ID,
				artifact.PayloadCodec,
			)
		}
	case invocationOpaqueShapeFormat:
		if artifact.Root.Kind != "bytes" {
			return fmt.Errorf("opaque invocation shape %s must use bytes root", artifact.ID)
		}
	default:
		return fmt.Errorf(
			"invocation shape %s has unsupported format %q",
			artifact.ID,
			artifact.Format,
		)
	}
	if err := validateInvocationShapeNode(artifact.Root, "$"); err != nil {
		return fmt.Errorf("invocation shape %s: %w", artifact.ID, err)
	}
	return nil
}

func validateInvocationShapeNode(node InvocationShapeNode, path string) error {
	switch node.Kind {
	case "any", "string", "number", "boolean", "null", "bytes":
		if len(node.Required) != 0 || len(node.Properties) != 0 ||
			node.Items != nil || node.AdditionalProperties != nil {
			return fmt.Errorf("%s %s node has object or array members", path, node.Kind)
		}
		return nil
	case "array":
		if node.Items == nil {
			return fmt.Errorf("%s array has no item shape", path)
		}
		if len(node.Required) != 0 || len(node.Properties) != 0 ||
			node.AdditionalProperties != nil {
			return fmt.Errorf("%s array has object members", path)
		}
		return validateInvocationShapeNode(*node.Items, path+"[]")
	case "object":
		if node.Items != nil {
			return fmt.Errorf("%s object has array item shape", path)
		}
		required := map[string]struct{}{}
		for _, name := range node.Required {
			if name == "" {
				return fmt.Errorf("%s object has empty required property", path)
			}
			if _, duplicate := required[name]; duplicate {
				return fmt.Errorf("%s object requires property %q more than once", path, name)
			}
			required[name] = struct{}{}
			if _, declared := node.Properties[name]; !declared {
				return fmt.Errorf(
					"%s object requires undeclared property %q",
					path,
					name,
				)
			}
		}
		for name, child := range node.Properties {
			if name == "" {
				return fmt.Errorf("%s object has empty property name", path)
			}
			if err := validateInvocationShapeNode(child, path+"."+name); err != nil {
				return err
			}
		}
		if node.AdditionalProperties != nil {
			if err := validateInvocationShapeNode(
				*node.AdditionalProperties,
				path+".*",
			); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("%s has unsupported kind %q", path, node.Kind)
	}
}

func validateInvocationPayload(
	artifact InvocationShapeArtifact,
	payload []byte,
) error {
	if err := validateInvocationShapeArtifact(artifact); err != nil {
		return err
	}
	switch artifact.Format {
	case invocationOpaqueShapeFormat:
		return nil
	case invocationJSONShapeFormat:
		decoder := json.NewDecoder(bytes.NewReader(payload))
		decoder.UseNumber()
		var value any
		if err := decoder.Decode(&value); err != nil {
			return fmt.Errorf("decode JSON payload: %w", err)
		}
		var trailing any
		if err := decoder.Decode(&trailing); err != io.EOF {
			if err == nil {
				return fmt.Errorf("trailing JSON value is not allowed")
			}
			return fmt.Errorf("decode trailing JSON value: %w", err)
		}
		return validateInvocationJSONValue(value, artifact.Root, "$")
	default:
		return fmt.Errorf("unsupported invocation shape format %q", artifact.Format)
	}
}

func validateInvocationJSONValue(
	value any,
	node InvocationShapeNode,
	path string,
) error {
	switch node.Kind {
	case "any":
		return nil
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("%s must be string", path)
		}
		return nil
	case "number":
		if _, ok := value.(json.Number); !ok {
			return fmt.Errorf("%s must be number", path)
		}
		return nil
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("%s must be boolean", path)
		}
		return nil
	case "null":
		if value != nil {
			return fmt.Errorf("%s must be null", path)
		}
		return nil
	case "array":
		values, ok := value.([]any)
		if !ok {
			return fmt.Errorf("%s must be array", path)
		}
		for i, item := range values {
			if err := validateInvocationJSONValue(
				item,
				*node.Items,
				fmt.Sprintf("%s[%d]", path, i),
			); err != nil {
				return err
			}
		}
		return nil
	case "object":
		values, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("%s must be object", path)
		}
		for _, name := range node.Required {
			if _, exists := values[name]; !exists {
				return fmt.Errorf("%s is missing required property %q", path, name)
			}
		}
		for name, item := range values {
			child, declared := node.Properties[name]
			if !declared {
				if node.AdditionalProperties == nil {
					return fmt.Errorf("%s has undeclared property %q", path, name)
				}
				child = *node.AdditionalProperties
			}
			if err := validateInvocationJSONValue(item, child, path+"."+name); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("%s cannot validate kind %q as JSON", path, node.Kind)
	}
}

func mustReferenceInvocationShape(path string) InvocationShapeArtifact {
	content, err := invocationShapeFS.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("read invocation shape %s: %v", path, err))
	}
	var artifact InvocationShapeArtifact
	if err := json.Unmarshal(content, &artifact); err != nil {
		panic(fmt.Sprintf("parse invocation shape %s: %v", path, err))
	}
	if err := validateInvocationShapeArtifact(artifact); err != nil {
		panic(fmt.Sprintf("validate invocation shape %s: %v", path, err))
	}
	return artifact
}

func sourceAdapterRequestShapeArtifact() InvocationShapeArtifact {
	return mustReferenceInvocationShape("shapes/source-adapter-request-v0.json")
}

func sourceAdapterResultShapeArtifact() InvocationShapeArtifact {
	return mustReferenceInvocationShape("shapes/source-adapter-result-v0.json")
}

func contractQueryRequestShapeArtifact() InvocationShapeArtifact {
	return mustReferenceInvocationShape("shapes/contract-query-request-v0.json")
}

func contractQueryResultShapeArtifact() InvocationShapeArtifact {
	return mustReferenceInvocationShape("shapes/contract-query-result-v0.json")
}

func opaqueInvocationShapeArtifact(
	id string,
	codec string,
) InvocationShapeArtifact {
	return InvocationShapeArtifact{
		Schema:       invocationShapeArtifactSchema,
		ID:           id,
		Issuer:       referenceInvocationShapeIssuer(),
		PayloadCodec: codec,
		Format:       invocationOpaqueShapeFormat,
		Root:         InvocationShapeNode{Kind: "bytes"},
	}
}
