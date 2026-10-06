package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"tei/internal/sourceindex"
)

const (
	semanticLockSchema                 = "tei.reference.semantic-lock/v0"
	semanticLockCanonicalizationPolicy = "semantic-catalog-json/v0"
)

type SemanticLockExtension struct {
	ID                    string `json:"id"`
	Namespace             string `json:"namespace"`
	Version               string `json:"version"`
	DescriptorFingerprint string `json:"descriptorFingerprint"`
}

type SemanticLockInterface struct {
	ID                  string            `json:"id"`
	Version             string            `json:"version"`
	ContractFingerprint string            `json:"contractFingerprint"`
	Extension           ExtensionIdentity `json:"extension"`
}

type SemanticLockInterfaceWiring struct {
	ConsumerExtension          string `json:"consumerExtension"`
	Interface                  string `json:"interface"`
	ContractVersion            string `json:"contractVersion"`
	ContractFingerprint        string `json:"contractFingerprint"`
	ProviderExtension          string `json:"providerExtension"`
	ProviderConformanceProfile string `json:"providerConformanceProfile"`
}

type SemanticLockNamespaceAuthority struct {
	Namespace string `json:"namespace"`
	Extension string `json:"extension"`
}

type SemanticLock struct {
	Schema                  string                           `json:"schema"`
	CanonicalizationPolicy  string                           `json:"canonicalizationPolicy"`
	ResolutionPolicy        string                           `json:"resolutionPolicy"`
	Requirements            []ExtensionRequirement           `json:"requirements"`
	RequirementsFingerprint string                           `json:"requirementsFingerprint"`
	CoreSubjectFingerprint  string                           `json:"coreSubjectFingerprint,omitempty"`
	Extensions              []SemanticLockExtension          `json:"extensions"`
	NamespaceAuthorities    []SemanticLockNamespaceAuthority `json:"namespaceAuthorities,omitempty"`
	Interfaces              []SemanticLockInterface          `json:"interfaces,omitempty"`
	InterfaceWirings        []SemanticLockInterfaceWiring    `json:"interfaceWirings,omitempty"`
	SemanticSetFingerprint  string                           `json:"semanticSetFingerprint"`
}

func buildSemanticLock(catalog SemanticCatalog) (SemanticLock, error) {
	if len(catalog.Unresolved) != 0 {
		return SemanticLock{}, fmt.Errorf("semantic lock requires a fully resolved catalog")
	}
	if catalog.ResolutionFingerprint == "" || !validSHA256Fingerprint(catalog.ResolutionFingerprint) {
		return SemanticLock{}, fmt.Errorf("semantic lock requires an exact semantic resolution fingerprint")
	}
	if catalog.ResolutionPolicy == "" {
		return SemanticLock{}, fmt.Errorf("semantic lock requires resolution policy identity")
	}
	requirements, err := normalizeRootRequirements(catalog.Requirements)
	if err != nil {
		return SemanticLock{}, err
	}

	lock := SemanticLock{
		Schema:                  semanticLockSchema,
		CanonicalizationPolicy:  semanticLockCanonicalizationPolicy,
		ResolutionPolicy:        catalog.ResolutionPolicy,
		Requirements:            requirements,
		RequirementsFingerprint: extensionRequirementsFingerprint(requirements),
		CoreSubjectFingerprint:  catalog.CoreSubjectFingerprint,
		SemanticSetFingerprint:  catalog.ResolutionFingerprint,
	}
	for _, extension := range catalog.Extensions {
		lock.Extensions = append(lock.Extensions, SemanticLockExtension{
			ID:                    extension.ID,
			Namespace:             extension.Namespace,
			Version:               extension.Version,
			DescriptorFingerprint: extension.DescriptorFingerprint,
		})
	}
	for _, authority := range catalog.NamespaceAuthorities {
		lock.NamespaceAuthorities = append(lock.NamespaceAuthorities, SemanticLockNamespaceAuthority{
			Namespace: authority.Namespace,
			Extension: authority.Extension,
		})
	}
	for _, semanticInterface := range catalog.Interfaces {
		lock.Interfaces = append(lock.Interfaces, SemanticLockInterface{
			ID:                  semanticInterface.ID,
			Version:             semanticInterface.Version,
			ContractFingerprint: semanticInterface.ContractFingerprint,
			Extension:           semanticInterface.Extension,
		})
	}
	for _, wiring := range catalog.InterfaceWirings {
		lock.InterfaceWirings = append(lock.InterfaceWirings, SemanticLockInterfaceWiring{
			ConsumerExtension:          wiring.ConsumerExtension,
			Interface:                  wiring.Interface,
			ContractVersion:            wiring.ContractVersion,
			ContractFingerprint:        wiring.ContractFingerprint,
			ProviderExtension:          wiring.ProviderExtension,
			ProviderConformanceProfile: wiring.ProviderConformanceProfile,
		})
	}
	normalizeSemanticLock(&lock)
	return lock, nil
}

func verifySemanticLockReplay(
	lock SemanticLock,
	core CoreCatalogInput,
	available []ExtensionDescriptor,
) error {
	normalizeSemanticLock(&lock)
	if lock.Schema != semanticLockSchema {
		return fmt.Errorf("semantic lock schema %q is not supported", lock.Schema)
	}
	if lock.CanonicalizationPolicy != semanticLockCanonicalizationPolicy {
		return fmt.Errorf(
			"semantic lock canonicalization policy %q is not current %q",
			lock.CanonicalizationPolicy,
			semanticLockCanonicalizationPolicy,
		)
	}
	if lock.ResolutionPolicy != semanticResolutionPolicy {
		return fmt.Errorf(
			"semantic lock resolution policy %q is not current %q",
			lock.ResolutionPolicy,
			semanticResolutionPolicy,
		)
	}
	if lock.RequirementsFingerprint != extensionRequirementsFingerprint(lock.Requirements) {
		return fmt.Errorf("semantic lock root requirements fingerprint is stale")
	}
	if lock.CoreSubjectFingerprint != core.SubjectFingerprint {
		return fmt.Errorf(
			"semantic lock core subject fingerprint is stale: have %s want %s",
			lock.CoreSubjectFingerprint,
			core.SubjectFingerprint,
		)
	}
	if !validSHA256Fingerprint(lock.SemanticSetFingerprint) {
		return fmt.Errorf("semantic lock semantic set fingerprint is invalid")
	}

	exactDescriptors, err := exactLockedDescriptors(lock.Extensions, available)
	if err != nil {
		return err
	}
	interfaceWirings := make([]SemanticInterfaceWiring, 0, len(lock.InterfaceWirings))
	interfaceConformance := make([]SemanticInterfaceConformance, 0, len(lock.InterfaceWirings))
	for _, wiring := range lock.InterfaceWirings {
		interfaceWirings = append(interfaceWirings, SemanticInterfaceWiring{
			ConsumerExtension: wiring.ConsumerExtension,
			Interface:         wiring.Interface,
			ProviderExtension: wiring.ProviderExtension,
		})
		interfaceConformance = append(interfaceConformance, SemanticInterfaceConformance{
			ProviderExtension:   wiring.ProviderExtension,
			Interface:           wiring.Interface,
			ContractFingerprint: wiring.ContractFingerprint,
			Profile:             wiring.ProviderConformanceProfile,
			EvidenceFingerprint: syntheticLockVerificationEvidenceFingerprint(),
		})
	}
	namespaceAuthorities := make([]SemanticNamespaceAuthority, 0, len(lock.NamespaceAuthorities))
	for _, authority := range lock.NamespaceAuthorities {
		namespaceAuthorities = append(namespaceAuthorities, SemanticNamespaceAuthority{
			Namespace:           authority.Namespace,
			Extension:           authority.Extension,
			EvidenceFingerprint: syntheticLockVerificationEvidenceFingerprint(),
		})
	}

	reconstructed, err := resolveSemanticCatalogWithAllInputs(
		core,
		exactDescriptors,
		lock.Requirements,
		interfaceWirings,
		interfaceConformance,
		namespaceAuthorities,
	)
	if err != nil {
		return fmt.Errorf("semantic lock exact graph cannot be reconstructed: %w", err)
	}
	if len(reconstructed.Unresolved) != 0 {
		return fmt.Errorf("semantic lock exact graph is unresolved: %+v", reconstructed.Unresolved)
	}
	if reconstructed.ResolutionFingerprint != lock.SemanticSetFingerprint {
		return fmt.Errorf(
			"semantic lock semantic set fingerprint is stale: have %s want %s",
			lock.SemanticSetFingerprint,
			reconstructed.ResolutionFingerprint,
		)
	}
	reconstructedLock, err := buildSemanticLock(reconstructed)
	if err != nil {
		return err
	}
	if !sameSemanticLockProjection(lock, reconstructedLock) {
		return fmt.Errorf("semantic lock exact graph does not match reconstructed catalog")
	}
	return nil
}

func exactLockedDescriptors(
	locked []SemanticLockExtension,
	available []ExtensionDescriptor,
) ([]ExtensionDescriptor, error) {
	availableByKey := map[string][]ExtensionDescriptor{}
	for _, descriptor := range available {
		normalized, err := normalizeDescriptor(descriptor)
		if err != nil {
			return nil, err
		}
		key := semanticLockExtensionKey(normalized.ID, normalized.Version)
		availableByKey[key] = append(availableByKey[key], normalized)
	}

	out := make([]ExtensionDescriptor, 0, len(locked))
	for _, item := range locked {
		if item.ID == "" || item.Namespace == "" || item.Version == "" ||
			!validSHA256Fingerprint(item.DescriptorFingerprint) {
			return nil, fmt.Errorf("semantic lock has incomplete extension identity")
		}
		matches := availableByKey[semanticLockExtensionKey(item.ID, item.Version)]
		if len(matches) != 1 {
			return nil, fmt.Errorf(
				"semantic lock requires exact extension %s@%s exactly once; found %d",
				item.ID,
				item.Version,
				len(matches),
			)
		}
		descriptor := matches[0]
		if descriptor.Namespace != item.Namespace {
			return nil, fmt.Errorf(
				"semantic lock extension %s@%s namespace changed: have %s want %s",
				item.ID,
				item.Version,
				item.Namespace,
				descriptor.Namespace,
			)
		}
		if descriptorFingerprint(descriptor) != item.DescriptorFingerprint {
			return nil, fmt.Errorf(
				"semantic lock extension %s@%s declaration fingerprint is stale",
				item.ID,
				item.Version,
			)
		}
		out = append(out, descriptor)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ID != out[j].ID {
			return out[i].ID < out[j].ID
		}
		return compareVersion(out[i].Version, out[j].Version) < 0
	})
	return out, nil
}

func normalizeSemanticLock(lock *SemanticLock) {
	if lock == nil {
		return
	}
	lock.Schema = strings.TrimSpace(lock.Schema)
	lock.CanonicalizationPolicy = strings.TrimSpace(lock.CanonicalizationPolicy)
	lock.ResolutionPolicy = strings.TrimSpace(lock.ResolutionPolicy)
	lock.RequirementsFingerprint = strings.TrimSpace(lock.RequirementsFingerprint)
	lock.CoreSubjectFingerprint = strings.TrimSpace(lock.CoreSubjectFingerprint)
	lock.SemanticSetFingerprint = strings.TrimSpace(lock.SemanticSetFingerprint)
	requirements, err := normalizeRootRequirements(lock.Requirements)
	if err == nil {
		lock.Requirements = requirements
	}
	sort.Slice(lock.Extensions, func(i, j int) bool {
		if lock.Extensions[i].ID != lock.Extensions[j].ID {
			return lock.Extensions[i].ID < lock.Extensions[j].ID
		}
		return compareVersion(lock.Extensions[i].Version, lock.Extensions[j].Version) < 0
	})
	sort.Slice(lock.NamespaceAuthorities, func(i, j int) bool {
		if lock.NamespaceAuthorities[i].Namespace != lock.NamespaceAuthorities[j].Namespace {
			return lock.NamespaceAuthorities[i].Namespace < lock.NamespaceAuthorities[j].Namespace
		}
		return lock.NamespaceAuthorities[i].Extension < lock.NamespaceAuthorities[j].Extension
	})
	sort.Slice(lock.Interfaces, func(i, j int) bool {
		if lock.Interfaces[i].ID != lock.Interfaces[j].ID {
			return lock.Interfaces[i].ID < lock.Interfaces[j].ID
		}
		if lock.Interfaces[i].Version != lock.Interfaces[j].Version {
			return compareVersion(lock.Interfaces[i].Version, lock.Interfaces[j].Version) < 0
		}
		return lock.Interfaces[i].Extension.ID < lock.Interfaces[j].Extension.ID
	})
	sort.Slice(lock.InterfaceWirings, func(i, j int) bool {
		if lock.InterfaceWirings[i].ConsumerExtension != lock.InterfaceWirings[j].ConsumerExtension {
			return lock.InterfaceWirings[i].ConsumerExtension < lock.InterfaceWirings[j].ConsumerExtension
		}
		if lock.InterfaceWirings[i].Interface != lock.InterfaceWirings[j].Interface {
			return lock.InterfaceWirings[i].Interface < lock.InterfaceWirings[j].Interface
		}
		return lock.InterfaceWirings[i].ProviderExtension < lock.InterfaceWirings[j].ProviderExtension
	})
}

func sameSemanticLockProjection(left, right SemanticLock) bool {
	left.RequirementsFingerprint = ""
	right.RequirementsFingerprint = ""
	left.SemanticSetFingerprint = ""
	right.SemanticSetFingerprint = ""
	leftJSON, _ := json.Marshal(left)
	rightJSON, _ := json.Marshal(right)
	return string(leftJSON) == string(rightJSON)
}

func extensionRequirementsFingerprint(requirements []ExtensionRequirement) string {
	normalized := append([]ExtensionRequirement(nil), requirements...)
	sort.Slice(normalized, func(i, j int) bool {
		if normalized[i].Extension != normalized[j].Extension {
			return normalized[i].Extension < normalized[j].Extension
		}
		return normalized[i].CompatibleMajor < normalized[j].CompatibleMajor
	})
	encoded, _ := json.Marshal(normalized)
	return sourceindex.Fingerprint(encoded)
}

func semanticLockExtensionKey(id, version string) string {
	return fmt.Sprintf("%q|%q", id, version)
}

func syntheticLockVerificationEvidenceFingerprint() string {
	return sourceindex.Fingerprint([]byte("semantic-lock-verification"))
}
