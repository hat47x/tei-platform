package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"tei/internal/operations"
	"tei/internal/plugin/transform"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// authoringBinding describes the exact Source-backed Data that contributed
// required response fields. It is provenance, not acceptance or permission.
type authoringBinding struct {
	DataID               string   `json:"dataId"`
	SourcePath           string   `json:"sourcePath"`
	SourceFingerprint    string   `json:"sourceFingerprint"`
	ModelRevision        string   `json:"modelRevision"`
	RequiredAttributeIDs []string `json:"requiredAttributeIds"`
	InputContractDigest  string   `json:"inputContractDigest"`
}

type requiredResponseField struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// inspectorFieldType deliberately accepts only wire types already verified
// for this particular Reader. Domain aliases and decimals must not be
// guessed from their type names.
func inspectorFieldType(typeID string) (string, bool) {
	switch typeID {
	case "std.text", "std.identifier":
		return "string", true
	default:
		return "", false
	}
}

// inspectSubjectStringField checks the actual Operation's JSON response shape,
// rather than treating any Data attribute name as a real InspectSubject field.
// Only non-optional top-level strings are admitted by this first binding.
// This is a structural check, not proof that a specific Host invocation ran.
func inspectSubjectStringField(name string) bool {
	t := reflect.TypeOf(operations.SubjectInspection{})
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonName, options, _ := strings.Cut(field.Tag.Get("json"), ",")
		if jsonName == name && !strings.Contains(options, "omitempty") &&
			field.Type.Kind() == reflect.String {
			return true
		}
	}
	return false
}

// bindAuthoringData augments the existing Reader contract with a Data table
// selected by its exact canonical ID. The table remains the sole authoring
// Source; the JSON contract is an input to the existing Generator, not an
// independently edited model of the Data. Baseline response requirements
// remain enforced.
func bindAuthoringData(root, dataID string, contractRaw []byte) ([]byte, *authoringBinding, error) {
	if root == "" || dataID == "" || strings.TrimSpace(dataID) != dataID {
		return nil, nil, fmt.Errorf("authoring requires a root and an exact Data ID")
	}
	// Unlike encoding/json's map decoder, the existing transform parser
	// rejects duplicate JSON keys rather than silently choosing the last.
	if _, err := transform.ParseJSON(contractRaw); err != nil {
		return nil, nil, fmt.Errorf("Reader contract: %w", err)
	}
	loaded, err := workspace.Load(root)
	if err != nil {
		return nil, nil, fmt.Errorf("load authoring Workspace: %w", err)
	}
	index, err := workspace.BuildSourceIndexFromLoad(root, loaded)
	if err != nil {
		return nil, nil, fmt.Errorf("index authoring Source from Load: %w", err)
	}
	var dataCount int
	var attributes []struct {
		ID       string
		Symbol   string
		TypeID   string
		Required bool
	}
	for _, ns := range loaded.World.Semantics {
		for _, data := range ns.Data {
			if string(data.ID) != dataID {
				continue
			}
			dataCount++
			for _, a := range data.Attributes {
				attributes = append(attributes, struct {
					ID       string
					Symbol   string
					TypeID   string
					Required bool
				}{string(a.ID), a.Symbol, string(a.TypeRef), a.Required})
			}
		}
	}
	if dataCount != 1 {
		return nil, nil, fmt.Errorf("authoring Data %q must resolve to exactly one canonical definition; found %d", dataID, dataCount)
	}
	refs := index.ReferencesFor(sourceindex.Subject{Kind: "semantic-data", ID: dataID})
	if len(refs) != 1 || refs[0].Role != sourceindex.RoleDefinition ||
		!strings.HasSuffix(refs[0].ArtifactPath, ".tei.md") {
		return nil, nil, fmt.Errorf("authoring Data %q needs one indexed tei.authoring/v1 Markdown definition", dataID)
	}
	sourcePath := refs[0].ArtifactPath
	var selected bool
	for _, read := range loaded.AuthoringSources {
		if read.Path == sourcePath && read.Language.LanguageID == "tei.authoring" &&
			read.Language.ContractVersion == "v1" && read.Definitions > 0 {
			selected = true
		}
	}
	if !selected {
		return nil, nil, fmt.Errorf("Data %q was not read by selected tei.authoring/v1", dataID)
	}
	var sourceFingerprint string
	for _, a := range index.Artifacts {
		if a.Path == sourcePath {
			sourceFingerprint = a.Fingerprint
			break
		}
	}
	if sourceFingerprint == "" {
		return nil, nil, fmt.Errorf("indexed authoring Source %q has no byte fingerprint", sourcePath)
	}

	var sections map[string]json.RawMessage
	if err := json.Unmarshal(contractRaw, &sections); err != nil {
		return nil, nil, err
	}
	// The response shape checked below belongs only to this exact Operation.
	// An arbitrary bridge must not inherit its field qualifications.
	var op struct {
		Bridge string `json:"bridge"`
		Effect string `json:"effect"`
	}
	if err := json.Unmarshal(sections["operation"], &op); err != nil {
		return nil, nil, fmt.Errorf("Reader Operation: %w", err)
	}
	if op.Bridge != "main.StudioBridge.InspectSubject" || op.Effect != "read-only" {
		return nil, nil, fmt.Errorf("authoring Reader binding requires the read-only InspectSubject Operation")
	}
	rawRequired, ok := sections[requiredResponseFieldsSection]
	if !ok {
		return nil, nil, fmt.Errorf("Reader contract has no %s baseline", requiredResponseFieldsSection)
	}
	var fields []requiredResponseField
	if err := json.Unmarshal(rawRequired, &fields); err != nil {
		return nil, nil, fmt.Errorf("Reader baseline fields: %w", err)
	}
	seen := make(map[string]string, len(fields))
	for _, field := range fields {
		if !codeSafeScalar.MatchString(field.Name) || field.Type == "" {
			return nil, nil, fmt.Errorf("invalid baseline required response field %q", field.Name)
		}
		if _, exists := seen[field.Name]; exists {
			return nil, nil, fmt.Errorf("duplicate baseline required response field %q", field.Name)
		}
		seen[field.Name] = field.Type
	}
	var requiredIDs []string
	for _, attr := range attributes {
		if !attr.Required {
			continue
		}
		if !codeSafeScalar.MatchString(attr.Symbol) {
			return nil, nil, fmt.Errorf("required Data attribute %q is not a safe response property name", attr.ID)
		}
		attrRefs := index.ReferencesFor(sourceindex.Subject{Kind: "semantic-attribute", ID: attr.ID})
		if len(attrRefs) != 1 || attrRefs[0].Role != sourceindex.RoleDefinition ||
			attrRefs[0].ArtifactPath != sourcePath {
			return nil, nil, fmt.Errorf("required attribute %q has no unique Source occurrence in %s", attr.ID, sourcePath)
		}
		fieldType, supported := inspectorFieldType(attr.TypeID)
		if !supported {
			return nil, nil, fmt.Errorf("required attribute %q has unsupported Reader wire type %q", attr.ID, attr.TypeID)
		}
		if fieldType != "string" || !inspectSubjectStringField(attr.Symbol) {
			return nil, nil, fmt.Errorf("required attribute %q does not match a non-optional string field of InspectSubject response", attr.ID)
		}
		if current, exists := seen[attr.Symbol]; exists {
			if current != fieldType {
				return nil, nil, fmt.Errorf("required attribute %q conflicts with baseline field %q type %q", attr.ID, attr.Symbol, current)
			}
		} else {
			seen[attr.Symbol] = fieldType
			fields = append(fields, requiredResponseField{Name: attr.Symbol, Type: fieldType})
		}
		requiredIDs = append(requiredIDs, attr.ID)
	}
	if len(requiredIDs) == 0 {
		return nil, nil, fmt.Errorf("authoring Data %q declares no required attributes", dataID)
	}
	encoded, err := json.Marshal(fields)
	if err != nil {
		return nil, nil, err
	}
	sections[requiredResponseFieldsSection] = encoded
	effective, err := json.Marshal(sections)
	if err != nil {
		return nil, nil, err
	}
	return effective, &authoringBinding{
		DataID: dataID, SourcePath: sourcePath, SourceFingerprint: sourceFingerprint,
		ModelRevision: sourceindex.ModelRevision(index), RequiredAttributeIDs: requiredIDs,
		InputContractDigest: digest(contractRaw),
	}, nil
}
