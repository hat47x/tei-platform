package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"tei/internal/sourceindex"
)

type SemanticInterfaceContract struct {
	ID                    string   `json:"id"`
	Version               string   `json:"version"`
	InputSemantics        []string `json:"inputSemantics"`
	OutputSemantics       []string `json:"outputSemantics"`
	UnknownSemantics      string   `json:"unknownSemantics"`
	FailureSemantics      string   `json:"failureSemantics"`
	ProviderConformanceRef string   `json:"providerConformanceRef"`
}

type SemanticInterfaceRequirement struct {
	Interface       string `json:"interface"`
	CompatibleMajor int    `json:"compatibleMajor"`
}

type SemanticInterfaceProvider struct {
	Interface           string `json:"interface"`
	ContractVersion     string `json:"contractVersion"`
	ContractFingerprint string `json:"contractFingerprint"`
	ConformanceRef      string `json:"conformanceRef"`
}

type SemanticInterfaceWiring struct {
	ConsumerExtension string `json:"consumerExtension"`
	Interface         string `json:"interface"`
	ProviderExtension string `json:"providerExtension"`
}

type CatalogSemanticInterface struct {
	ID                  string            `json:"id"`
	Version             string            `json:"version"`
	ContractFingerprint string            `json:"contractFingerprint"`
	InputSemantics      []string          `json:"inputSemantics"`
	OutputSemantics     []string          `json:"outputSemantics"`
	UnknownSemantics    string            `json:"unknownSemantics"`
	FailureSemantics    string            `json:"failureSemantics"`
	ProviderConformanceRef string         `json:"providerConformanceRef"`
	Extension           ExtensionIdentity `json:"extension"`
	Source              string            `json:"source,omitempty"`
	SourceFingerprint   string            `json:"sourceFingerprint,omitempty"`
}

type CatalogInterfaceWiring struct {
	ConsumerExtension   string `json:"consumerExtension"`
	Interface           string `json:"interface"`
	ContractVersion     string `json:"contractVersion"`
	ContractFingerprint string `json:"contractFingerprint"`
	ProviderExtension   string `json:"providerExtension"`
	ProviderConformance string `json:"providerConformance"`
	Selection           string `json:"selection"`
}

func normalizeSemanticInterfaceDeclarations(d *ExtensionDescriptor) error {
	seenContracts := map[string]struct{}{}
	for i := range d.Interfaces {
		contract := &d.Interfaces[i]
		contract.ID = strings.TrimSpace(contract.ID)
		contract.Version = strings.TrimSpace(contract.Version)
		contract.UnknownSemantics = strings.TrimSpace(contract.UnknownSemantics)
		contract.FailureSemantics = strings.TrimSpace(contract.FailureSemantics)
		contract.ProviderConformanceRef = strings.TrimSpace(contract.ProviderConformanceRef)
		if contract.ID == "" || contract.Version == "" {
			return fmt.Errorf("extension %s interface[%d] requires id and version", d.ID, i)
		}
		if !interfaceBelongsToNamespace(contract.ID, d.Namespace) {
			return fmt.Errorf("extension %s interface %s is outside namespace %s", d.ID, contract.ID, d.Namespace)
		}
		if _, err := parseVersion(contract.Version); err != nil {
			return fmt.Errorf("extension %s interface %s: %w", d.ID, contract.ID, err)
		}
		contract.InputSemantics = normalizeSemanticLabels(contract.InputSemantics)
		contract.OutputSemantics = normalizeSemanticLabels(contract.OutputSemantics)
		if len(contract.InputSemantics) == 0 || len(contract.OutputSemantics) == 0 ||
			contract.UnknownSemantics == "" || contract.FailureSemantics == "" || contract.ProviderConformanceRef == "" {
			return fmt.Errorf("extension %s interface %s has incomplete semantic contract", d.ID, contract.ID)
		}
		key := contract.ID + "@" + contract.Version
		if _, ok := seenContracts[key]; ok {
			return fmt.Errorf("extension %s duplicates interface contract %s", d.ID, key)
		}
		seenContracts[key] = struct{}{}
	}
	sort.Slice(d.Interfaces, func(i, j int) bool {
		if d.Interfaces[i].ID != d.Interfaces[j].ID {
			return d.Interfaces[i].ID < d.Interfaces[j].ID
		}
		return compareVersion(d.Interfaces[i].Version, d.Interfaces[j].Version) < 0
	})

	seenRequirements := map[string]struct{}{}
	for i := range d.RequiresInterfaces {
		requirement := &d.RequiresInterfaces[i]
		requirement.Interface = strings.TrimSpace(requirement.Interface)
		if requirement.Interface == "" || requirement.CompatibleMajor < 0 {
			return fmt.Errorf("extension %s has invalid semantic interface requirement", d.ID)
		}
		key := fmt.Sprintf("%s@%d", requirement.Interface, requirement.CompatibleMajor)
		if _, ok := seenRequirements[key]; ok {
			return fmt.Errorf("extension %s duplicates semantic interface requirement %s", d.ID, key)
		}
		seenRequirements[key] = struct{}{}
	}
	sort.Slice(d.RequiresInterfaces, func(i, j int) bool {
		if d.RequiresInterfaces[i].Interface != d.RequiresInterfaces[j].Interface {
			return d.RequiresInterfaces[i].Interface < d.RequiresInterfaces[j].Interface
		}
		return d.RequiresInterfaces[i].CompatibleMajor < d.RequiresInterfaces[j].CompatibleMajor
	})

	seenProviders := map[string]struct{}{}
	for i := range d.ProvidesInterfaces {
		provider := &d.ProvidesInterfaces[i]
		provider.Interface = strings.TrimSpace(provider.Interface)
		provider.ContractVersion = strings.TrimSpace(provider.ContractVersion)
		provider.ContractFingerprint = strings.TrimSpace(provider.ContractFingerprint)
		provider.ConformanceRef = strings.TrimSpace(provider.ConformanceRef)
		if provider.Interface == "" || provider.ContractVersion == "" ||
			!validSHA256Fingerprint(provider.ContractFingerprint) || provider.ConformanceRef == "" {
			return fmt.Errorf("extension %s has incomplete semantic interface provider[%d]", d.ID, i)
		}
		if _, err := parseVersion(provider.ContractVersion); err != nil {
			return fmt.Errorf("extension %s provider %s: %w", d.ID, provider.Interface, err)
		}
		key := provider.Interface + "@" + provider.ContractVersion
		if _, ok := seenProviders[key]; ok {
			return fmt.Errorf("extension %s duplicates semantic interface provider %s", d.ID, key)
		}
		seenProviders[key] = struct{}{}
	}
	sort.Slice(d.ProvidesInterfaces, func(i, j int) bool {
		if d.ProvidesInterfaces[i].Interface != d.ProvidesInterfaces[j].Interface {
			return d.ProvidesInterfaces[i].Interface < d.ProvidesInterfaces[j].Interface
		}
		return compareVersion(d.ProvidesInterfaces[i].ContractVersion, d.ProvidesInterfaces[j].ContractVersion) < 0
	})
	return nil
}

func semanticInterfaceFingerprint(contract SemanticInterfaceContract) string {
	encoded, _ := jsonMarshalStable(contract)
	return fingerprintBytes(encoded)
}

func normalizeSemanticLabels(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return uniqueSortedStrings(out)
}

func interfaceBelongsToNamespace(id, namespace string) bool {
	return id == namespace || strings.HasPrefix(id, namespace+".")
}

func validSHA256Fingerprint(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+64 {
		return false
	}
	for _, r := range strings.TrimPrefix(value, "sha256:") {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

func jsonMarshalStable(value any) ([]byte, error) {
	return json.Marshal(value)
}

func fingerprintBytes(value []byte) string {
	return sourceindex.Fingerprint(value)
}
