package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"tei/internal/core"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/plugin/transform"
)

// exactEngineOptions selects an independently built, verified process engine.
// An omitted option retains the existing in-process reference path.
type exactEngineOptions struct {
	Executable    string
	BuildEvidence string
}

type exactEngineSourceMember struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
}

type exactEngineGoBuilder struct {
	Version         string `json:"version"`
	BinaryDigest    string `json:"binaryDigest"`
	ToolchainDigest string `json:"toolchainDigest"`
}

// Keep the field order and JSON representation aligned with the build
// reference. Its fingerprint covers the complete typed build receipt.
type exactEngineBuildEvidence struct {
	Schema               string                          `json:"schema"`
	RepositoryRevision   string                          `json:"repositoryRevision"`
	EngineID             string                          `json:"engineId"`
	EngineVersion        string                          `json:"engineVersion"`
	EngineContract       string                          `json:"engineContract"`
	Target               string                          `json:"target"`
	RuntimeEngine        *pruntime.RuntimeEngineIdentity `json:"runtimeEngine,omitempty"`
	Entrypoint           string                          `json:"entrypoint"`
	SourceClosure        []exactEngineSourceMember       `json:"sourceClosure"`
	SourceClosureDigest  string                          `json:"sourceClosureDigest"`
	Builder              exactEngineGoBuilder            `json:"builder"`
	Recipe               []string                        `json:"recipe"`
	BuildSettings        []string                        `json:"buildSettings"`
	HostEnvironment      []string                        `json:"hostEnvironment,omitempty"`
	FirstArtifactDigest  string                          `json:"firstArtifactDigest"`
	SecondArtifactDigest string                          `json:"secondArtifactDigest"`
	Reproducible         bool                            `json:"reproducible"`
	ArtifactDigest       string                          `json:"artifactDigest"`
	EvidenceFingerprint  string                          `json:"evidenceFingerprint"`
}

func exactEngineFingerprint(e exactEngineBuildEvidence) (string, error) {
	e.EvidenceFingerprint = ""
	raw, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func verifyExactEngine(root, revision string, options exactEngineOptions) (pruntime.RuntimeEngineIdentity, string, error) {
	if options.Executable == "" || options.BuildEvidence == "" {
		return pruntime.RuntimeEngineIdentity{}, "", fmt.Errorf("exact Runtime Engine requires both executable and build evidence")
	}
	executable, err := filepath.Abs(options.Executable)
	if err != nil {
		return pruntime.RuntimeEngineIdentity{}, "", err
	}
	info, err := os.Lstat(executable)
	if err != nil {
		return pruntime.RuntimeEngineIdentity{}, "", err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return pruntime.RuntimeEngineIdentity{}, "", fmt.Errorf("exact Runtime Engine must be a regular non-symlink file")
	}
	relative, err := filepath.Rel(root, executable)
	if err != nil {
		return pruntime.RuntimeEngineIdentity{}, "", err
	}
	if relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator))) {
		return pruntime.RuntimeEngineIdentity{}, "", fmt.Errorf("exact Runtime Engine executable must be outside the repository")
	}
	receipt, err := os.Open(options.BuildEvidence)
	if err != nil {
		return pruntime.RuntimeEngineIdentity{}, "", err
	}
	defer receipt.Close()
	raw, err := io.ReadAll(io.LimitReader(receipt, 2<<20+1))
	if err != nil {
		return pruntime.RuntimeEngineIdentity{}, "", err
	}
	if len(raw) > 2<<20 {
		return pruntime.RuntimeEngineIdentity{}, "", fmt.Errorf("exact Runtime Engine build evidence exceeds byte limit")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var evidence exactEngineBuildEvidence
	if err := decoder.Decode(&evidence); err != nil {
		return pruntime.RuntimeEngineIdentity{}, "", fmt.Errorf("decode exact Runtime Engine build evidence: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return pruntime.RuntimeEngineIdentity{}, "", fmt.Errorf("exact Runtime Engine build evidence contains trailing data")
	}
	if evidence.Schema != "tei.reference.transform-engine-build-evidence/v1" ||
		evidence.RepositoryRevision != revision ||
		evidence.EngineID != "tei.runtime.engine.deterministic_transform.reference" ||
		evidence.EngineVersion != transformRuntimeEngineVersion ||
		evidence.EngineContract != "tei.runtime-engine/deterministic-transform-v1" ||
		evidence.Entrypoint != "./tools/reference-transform-runtime-engine" ||
		evidence.Target != "host/"+runtime.GOOS+"/"+runtime.GOARCH {
		return pruntime.RuntimeEngineIdentity{}, "", fmt.Errorf("exact Runtime Engine build receipt does not match this checkout and execution contract")
	}
	if !evidence.Reproducible || evidence.RuntimeEngine == nil ||
		evidence.ArtifactDigest == "" ||
		evidence.FirstArtifactDigest != evidence.ArtifactDigest ||
		evidence.SecondArtifactDigest != evidence.ArtifactDigest ||
		len(evidence.SourceClosure) == 0 || evidence.SourceClosureDigest == "" ||
		evidence.Builder.ToolchainDigest == "" {
		return pruntime.RuntimeEngineIdentity{}, "", fmt.Errorf("exact Runtime Engine build is missing reproducible provenance")
	}
	engine := *evidence.RuntimeEngine
	if err := engine.Validate(); err != nil {
		return pruntime.RuntimeEngineIdentity{}, "", err
	}
	if engine.ID != core.MustCanonicalID(evidence.EngineID) ||
		engine.Version != evidence.EngineVersion ||
		engine.ContractVersion != evidence.EngineContract ||
		engine.Target != evidence.Target ||
		engine.ImplementationDigest != evidence.ArtifactDigest {
		return pruntime.RuntimeEngineIdentity{}, "", fmt.Errorf("exact Runtime Engine identity differs from build receipt")
	}
	fingerprint, err := exactEngineFingerprint(evidence)
	if err != nil {
		return pruntime.RuntimeEngineIdentity{}, "", err
	}
	if fingerprint != evidence.EvidenceFingerprint {
		return pruntime.RuntimeEngineIdentity{}, "", fmt.Errorf("exact Runtime Engine build evidence fingerprint mismatch")
	}
	digest, err := hashFile(executable)
	if err != nil {
		return pruntime.RuntimeEngineIdentity{}, "", err
	}
	if digest != evidence.ArtifactDigest {
		return pruntime.RuntimeEngineIdentity{}, "", fmt.Errorf("exact Runtime Engine executable digest differs from verified build receipt")
	}
	return engine, fingerprint, nil
}

type exactEngineWireField struct {
	Name  string               `json:"name"`
	Value exactEngineWireValue `json:"value"`
}

type exactEngineWireValue struct {
	Kind        transform.ValueKind     `json:"kind"`
	Bool        *bool                   `json:"bool,omitempty"`
	Number      *string                 `json:"number,omitempty"`
	String      *string                 `json:"string,omitempty"`
	Array       *[]exactEngineWireValue `json:"array,omitempty"`
	Object      *[]exactEngineWireField `json:"object,omitempty"`
	BytesBase64 *string                 `json:"bytesBase64,omitempty"`
}

func exactEngineWire(v transform.Value, depth int) (exactEngineWireValue, error) {
	if depth > 128 {
		return exactEngineWireValue{}, fmt.Errorf("exact Runtime Engine input nesting exceeds limit")
	}
	w := exactEngineWireValue{Kind: v.Kind}
	switch v.Kind {
	case transform.ValueNull:
	case transform.ValueBool:
		val := v.Bool
		w.Bool = &val
	case transform.ValueNumber:
		val := v.Number
		w.Number = &val
	case transform.ValueString:
		val := v.String
		w.String = &val
	case transform.ValueBytes:
		val := base64.StdEncoding.EncodeToString(v.Bytes)
		w.BytesBase64 = &val
	case transform.ValueArray:
		items := make([]exactEngineWireValue, len(v.Array))
		for i, child := range v.Array {
			item, err := exactEngineWire(child, depth+1)
			if err != nil {
				return exactEngineWireValue{}, err
			}
			items[i] = item
		}
		w.Array = &items
	case transform.ValueObject:
		fields := make([]exactEngineWireField, len(v.Object))
		for i, field := range v.Object {
			child, err := exactEngineWire(field.Value, depth+1)
			if err != nil {
				return exactEngineWireValue{}, err
			}
			fields[i] = exactEngineWireField{Name: field.Name, Value: child}
		}
		w.Object = &fields
	default:
		return exactEngineWireValue{}, fmt.Errorf("unsupported exact Runtime Engine input kind %q", v.Kind)
	}
	return w, nil
}

func evaluateWithExactEngine(executable, sourcePath string, source transform.Source, inputs map[string]transform.Value) (transform.EvaluationResult, error) {
	type input struct {
		Name  string               `json:"name"`
		Value exactEngineWireValue `json:"value"`
	}
	names := make([]string, 0, len(inputs))
	for name := range inputs {
		names = append(names, name)
	}
	sort.Strings(names)
	wireInputs := make([]input, 0, len(names))
	for _, name := range names {
		wire, err := exactEngineWire(inputs[name], 0)
		if err != nil {
			return transform.EvaluationResult{}, err
		}
		wireInputs = append(wireInputs, input{Name: name, Value: wire})
	}
	request, err := json.Marshal(struct {
		Schema string  `json:"schema"`
		Inputs []input `json:"inputs"`
	}{
		Schema: "tei.reference.transform-engine-request/v1",
		Inputs: wireInputs,
	})
	if err != nil {
		return transform.EvaluationResult{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, "-source", sourcePath)
	cmd.Stdin = bytes.NewReader(request)
	cmd.Env = []string{}
	stdout, err := cmd.Output()
	if err != nil {
		return transform.EvaluationResult{}, fmt.Errorf("exact Runtime Engine execution failed: %w", err)
	}
	if len(stdout) > source.Limits.MaxOutputBytes*4+(1<<20) {
		return transform.EvaluationResult{}, fmt.Errorf("exact Runtime Engine response exceeds byte limit")
	}
	decoder := json.NewDecoder(bytes.NewReader(stdout))
	decoder.DisallowUnknownFields()
	var response struct {
		Schema string               `json:"schema"`
		Output exactEngineWireValue `json:"output"`
		Steps  int                  `json:"steps"`
	}
	if err := decoder.Decode(&response); err != nil {
		return transform.EvaluationResult{}, fmt.Errorf("decode exact Runtime Engine response: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return transform.EvaluationResult{}, fmt.Errorf("exact Runtime Engine returned trailing data")
	}
	if response.Schema != "tei.reference.transform-engine-response/v1" ||
		response.Output.Kind != transform.ValueBytes ||
		response.Output.BytesBase64 == nil ||
		response.Steps < 1 || response.Steps > source.Limits.MaxSteps {
		return transform.EvaluationResult{}, fmt.Errorf("exact Runtime Engine returned an invalid transform response")
	}
	raw, err := base64.StdEncoding.Strict().DecodeString(*response.Output.BytesBase64)
	if err != nil {
		return transform.EvaluationResult{}, err
	}
	if len(raw) > source.Limits.MaxOutputBytes {
		return transform.EvaluationResult{}, fmt.Errorf("exact Runtime Engine output exceeded source limit")
	}
	return transform.EvaluationResult{Output: transform.Bytes(raw), Steps: response.Steps}, nil
}
