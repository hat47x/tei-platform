// Package generatorcontext builds a bounded, digest-bound context for an
// external generator model. It selects only the accepted contract, fixed
// fixtures, and workspace revision; it does not call a model.
package generatorcontext

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

const Schema = "tei.reference.generator-context/v0"

type Input struct {
	Attribute string `json:"attribute"`
	Primitive string `json:"primitive"`
}

type Rounding struct {
	Precision int    `json:"precision"`
	Scale     int    `json:"scale"`
	Mode      string `json:"mode"`
}

type ContractInfo struct {
	ID              string   `json:"id"`
	Version         string   `json:"version"`
	Status          string   `json:"status"`
	Digest          string   `json:"digest"`
	Inputs          []Input  `json:"inputs"`
	OutputPrimitive string   `json:"outputPrimitive"`
	Rounding        Rounding `json:"rounding"`
}

type FixtureInfo struct {
	ContractID      string            `json:"contractId"`
	ContractVersion string            `json:"contractVersion"`
	Digest          string            `json:"digest"`
	Cases           []json.RawMessage `json:"cases"`
}

type WorkspaceInfo struct {
	Revision string `json:"revision"`
}

type Context struct {
	Schema      string        `json:"schema"`
	Contract    ContractInfo  `json:"contract"`
	Fixtures    FixtureInfo   `json:"fixtures"`
	Workspace   WorkspaceInfo `json:"workspace"`
	Constraints []string      `json:"constraints"`
	MaxBytes    int           `json:"maxBytes"`
	ActualBytes int           `json:"actualBytes"`
}

type contractFile struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Status  string `json:"status"`
	Inputs  []struct {
		Attribute string `json:"attribute"`
		Primitive string `json:"primitive"`
	} `json:"inputs"`
	Output struct {
		Primitive string `json:"primitive"`
	} `json:"output"`
	Rounding Rounding `json:"rounding"`
}

type fixtureFile struct {
	Schema          string            `json:"schema"`
	ContractID      string            `json:"contractId"`
	ContractVersion string            `json:"contractVersion"`
	Cases           []json.RawMessage `json:"cases"`
}

func digestBytes(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func Build(workspaceDir, contractPath, fixturePath string, maxBytes int) (Context, []byte, error) {
	if maxBytes <= 0 {
		maxBytes = 32768
	}
	contractRaw, err := os.ReadFile(contractPath)
	if err != nil {
		return Context{}, nil, err
	}
	var contract contractFile
	if err := json.Unmarshal(contractRaw, &contract); err != nil {
		return Context{}, nil, fmt.Errorf("%s: %w", contractPath, err)
	}
	if strings.TrimSpace(contract.ID) == "" || strings.TrimSpace(contract.Version) == "" {
		return Context{}, nil, fmt.Errorf("contract id and version are required")
	}
	if len(contract.Inputs) == 0 || strings.TrimSpace(contract.Output.Primitive) == "" {
		return Context{}, nil, fmt.Errorf("contract inputs and output are required")
	}
	fixtureRaw, err := os.ReadFile(fixturePath)
	if err != nil {
		return Context{}, nil, err
	}
	var fixtures fixtureFile
	if err := json.Unmarshal(fixtureRaw, &fixtures); err != nil {
		return Context{}, nil, fmt.Errorf("%s: %w", fixturePath, err)
	}
	if fixtures.ContractID != contract.ID || fixtures.ContractVersion != contract.Version {
		return Context{}, nil, fmt.Errorf("fixtures target %s@%s, contract is %s@%s",
			fixtures.ContractID, fixtures.ContractVersion, contract.ID, contract.Version)
	}
	loaded, err := workspace.Load(workspaceDir)
	if err != nil {
		return Context{}, nil, fmt.Errorf("load Workspace: %w", err)
	}
	index, err := workspace.BuildSourceIndex(workspaceDir, loaded)
	if err != nil {
		return Context{}, nil, fmt.Errorf("build Source Index: %w", err)
	}
	inputs := make([]Input, len(contract.Inputs))
	for i, input := range contract.Inputs {
		inputs[i] = Input{Attribute: input.Attribute, Primitive: input.Primitive}
	}
	context := Context{
		Schema: Schema,
		Contract: ContractInfo{
			ID: contract.ID, Version: contract.Version, Status: contract.Status,
			Digest: digestBytes(contractRaw), Inputs: inputs,
			OutputPrimitive: contract.Output.Primitive, Rounding: contract.Rounding,
		},
		Fixtures: FixtureInfo{
			ContractID: fixtures.ContractID, ContractVersion: fixtures.ContractVersion,
			Digest: digestBytes(fixtureRaw), Cases: fixtures.Cases,
		},
		Workspace: WorkspaceInfo{Revision: sourceindex.ModelRevision(index)},
		Constraints: []string{
			"Use only the listed accepted attributes and primitives.",
			"Do not invent attributes, operations, or output fields.",
			"Match every fixed fixture case exactly.",
			"Do not include secrets or customer data.",
			"Return source for the external generator adapter; acceptance and deployment remain outside this context.",
		},
		MaxBytes: maxBytes,
	}
	var raw []byte
	for i := 0; i < 2; i++ {
		raw, err = json.Marshal(context)
		if err != nil {
			return Context{}, nil, err
		}
		context.ActualBytes = len(raw)
	}
	if len(raw) > maxBytes {
		return Context{}, nil, fmt.Errorf("context is %d bytes, exceeding maxBytes %d", len(raw), maxBytes)
	}
	return context, raw, nil
}
