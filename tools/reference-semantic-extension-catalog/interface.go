package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"tei/internal/sourceindex"
)

type SemanticInterfaceContract struct {
	ID                     string   `json:"id"`
	Version                string   `json:"version"`
	InputSemantics         []string `json:"inputSemantics"`
	OutputSemantics        []string `json:"outputSemantics"`
	UnknownSemantics       string   `json:"unknownSemantics"`
	FailureSemantics       string   `json:"failureSemantics"`
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
}

type SemanticInterfaceConformance struct {
	ProviderExtension   string `json:"providerExtension"`
	Interface           string `json:"interface"`
	ContractFingerprint string `json:"contractFingerprint"`
	Profile             string `json:"profile"`
	EvidenceFingerprint string `json:"evidenceFingerprint"`
}

type SemanticInterfaceWiring struct {
	ConsumerExtension string `json:"consumerExtension"`
	Interface         string `json:"interface"`
	ProviderExtension string `json:"providerExtension"`
}

type CatalogSemanticInterface struct {
	ID                     string            `json:"id"`
	Version                string            `json:"version"`
	ContractFingerprint    string            `json:"contractFingerprint"`
	InputSemantics         []string          `json:"inputSemantics"`
	OutputSemantics        []string          `json:"outputSemantics"`
	UnknownSemantics       string            `json:"unknownSemantics"`
	FailureSemantics       string            `json:"failureSemantics"`
	ProviderConformanceRef string            `json:"providerConformanceRef"`
	Extension              ExtensionIdentity `json:"extension"`
	Source                 string            `json:"source,omitempty"`
	SourceFingerprint      string            `json:"sourceFingerprint,omitempty"`
}

type CatalogInterfaceWiring struct {
	ConsumerExtension          string `json:"consumerExtension"`
	Interface                  string `json:"interface"`
	ContractVersion            string `json:"contractVersion"`
	ContractFingerprint        string `json:"contractFingerprint"`
	ProviderExtension          string `json:"providerExtension"`
	ProviderConformanceProfile string `json:"providerConformanceProfile"`
	ConformanceEvidence        string `json:"conformanceEvidenceFingerprint"`
	Selection                  string `json:"selection"`
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
		if provider.Interface == "" || provider.ContractVersion == "" ||
			!validSHA256Fingerprint(provider.ContractFingerprint) {
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

type semanticInterfaceContractOwner struct {
	Contract  SemanticInterfaceContract
	Extension ExtensionDescriptor
}

type semanticInterfaceProviderCandidate struct {
	Extension ExtensionDescriptor
	Provider  SemanticInterfaceProvider
}

func resolveSemanticInterfaces(
	selected map[string]ExtensionDescriptor,
	blockedExtensions map[string]bool,
	wirings []SemanticInterfaceWiring,
	conformances []SemanticInterfaceConformance,
) ([]CatalogSemanticInterface, []CatalogInterfaceWiring, []CatalogUnresolved, error) {
	explicit, err := normalizeSemanticInterfaceWirings(wirings)
	if err != nil {
		return nil, nil, nil, err
	}
	verifiedConformance, err := normalizeSemanticInterfaceConformance(conformances)
	if err != nil {
		return nil, nil, nil, err
	}

	var ids []string
	for id := range selected {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	contractsByID := map[string][]semanticInterfaceContractOwner{}
	for _, id := range ids {
		if blockedExtensions[id] {
			continue
		}
		descriptor := selected[id]
		for _, contract := range descriptor.Interfaces {
			contractsByID[contract.ID] = append(contractsByID[contract.ID], semanticInterfaceContractOwner{
				Contract:  contract,
				Extension: descriptor,
			})
		}
	}

	var interfaces []CatalogSemanticInterface
	for _, owners := range contractsByID {
		for _, owner := range owners {
			interfaces = append(interfaces, catalogSemanticInterface(owner))
		}
	}
	sort.Slice(interfaces, func(i, j int) bool {
		if interfaces[i].ID != interfaces[j].ID {
			return interfaces[i].ID < interfaces[j].ID
		}
		if interfaces[i].Version != interfaces[j].Version {
			return compareVersion(interfaces[i].Version, interfaces[j].Version) < 0
		}
		return interfaces[i].Extension.ID < interfaces[j].Extension.ID
	})

	var unresolved []CatalogUnresolved
	var resolved []CatalogInterfaceWiring
	usedExplicit := map[string]bool{}

	for _, consumerID := range ids {
		if blockedExtensions[consumerID] {
			continue
		}
		consumer := selected[consumerID]
		for _, requirement := range consumer.RequiresInterfaces {
			matches := interfaceContractsCompatibleWithMajor(
				contractsByID[requirement.Interface],
				requirement.CompatibleMajor,
			)
			if len(matches) == 0 {
				identity := ExtensionIdentity{ID: consumer.ID, Version: consumer.Version}
				unresolved = append(unresolved, CatalogUnresolved{
					Code:      "interface-contract-unavailable",
					Extension: &identity,
					Detail: fmt.Sprintf(
						"%s requires semantic interface %s major %d",
						consumer.ID,
						requirement.Interface,
						requirement.CompatibleMajor,
					),
				})
				continue
			}
			if len(matches) != 1 {
				identity := ExtensionIdentity{ID: consumer.ID, Version: consumer.Version}
				unresolved = append(unresolved, CatalogUnresolved{
					Code:      "interface-contract-ambiguous",
					Extension: &identity,
					Detail: fmt.Sprintf(
						"%s sees %d contracts for semantic interface %s major %d",
						consumer.ID,
						len(matches),
						requirement.Interface,
						requirement.CompatibleMajor,
					),
				})
				continue
			}

			contractOwner := matches[0]
			contractFingerprint := semanticInterfaceFingerprint(contractOwner.Contract)
			validProviders, providerDiagnostics := interfaceProvidersForContract(
				selected,
				blockedExtensions,
				requirement.Interface,
				contractOwner.Contract.Version,
				contractFingerprint,
				contractOwner.Contract.ProviderConformanceRef,
				verifiedConformance,
			)
			unresolved = append(unresolved, providerDiagnostics...)

			wiringKey := semanticInterfaceWiringKey(consumer.ID, requirement.Interface)
			requested, hasRequested := explicit[wiringKey]
			if hasRequested {
				usedExplicit[wiringKey] = true
				provider, ok := semanticInterfaceProviderByExtension(validProviders, requested.ProviderExtension)
				if !ok {
					identity := ExtensionIdentity{ID: consumer.ID, Version: consumer.Version}
					unresolved = append(unresolved, CatalogUnresolved{
						Code:      "interface-wiring-invalid",
						Extension: &identity,
						Detail: fmt.Sprintf(
							"%s wires %s to %s, but that provider is not conformant to %s@%s",
							consumer.ID,
							requirement.Interface,
							requested.ProviderExtension,
							contractOwner.Contract.ID,
							contractOwner.Contract.Version,
						),
					})
					continue
				}
				resolved = append(resolved, catalogInterfaceWiring(
					consumer,
					contractOwner.Contract,
					contractFingerprint,
					provider.Extension,
					provider.Provider,
					verifiedConformance[semanticInterfaceConformanceKey(
						provider.Extension.ID,
						requirement.Interface,
						contractFingerprint,
						contractOwner.Contract.ProviderConformanceRef,
					)],
					"explicit",
				))
				continue
			}

			switch len(validProviders) {
			case 0:
				identity := ExtensionIdentity{ID: consumer.ID, Version: consumer.Version}
				unresolved = append(unresolved, CatalogUnresolved{
					Code:      "interface-provider-unavailable",
					Extension: &identity,
					Detail: fmt.Sprintf(
						"%s requires provider for %s@%s",
						consumer.ID,
						contractOwner.Contract.ID,
						contractOwner.Contract.Version,
					),
				})
			case 1:
				resolved = append(resolved, catalogInterfaceWiring(
					consumer,
					contractOwner.Contract,
					contractFingerprint,
					validProviders[0].Extension,
					validProviders[0].Provider,
					verifiedConformance[semanticInterfaceConformanceKey(
						validProviders[0].Extension.ID,
						requirement.Interface,
						contractFingerprint,
						contractOwner.Contract.ProviderConformanceRef,
					)],
					"unique-provider",
				))
			default:
				identity := ExtensionIdentity{ID: consumer.ID, Version: consumer.Version}
				var providerIDs []string
				for _, provider := range validProviders {
					providerIDs = append(providerIDs, provider.Extension.ID)
				}
				sort.Strings(providerIDs)
				unresolved = append(unresolved, CatalogUnresolved{
					Code:      "interface-provider-ambiguous",
					Extension: &identity,
					Detail: fmt.Sprintf(
						"%s has multiple conformant providers for %s@%s: %s",
						consumer.ID,
						contractOwner.Contract.ID,
						contractOwner.Contract.Version,
						strings.Join(providerIDs, ", "),
					),
				})
			}
		}
	}

	for key, wiring := range explicit {
		if usedExplicit[key] {
			continue
		}
		unresolved = append(unresolved, CatalogUnresolved{
			Code: "interface-wiring-unrequired",
			Detail: fmt.Sprintf(
				"explicit wiring %s -> %s via %s does not match a selected interface requirement",
				wiring.ConsumerExtension,
				wiring.Interface,
				wiring.ProviderExtension,
			),
		})
	}

	sort.Slice(resolved, func(i, j int) bool {
		if resolved[i].ConsumerExtension != resolved[j].ConsumerExtension {
			return resolved[i].ConsumerExtension < resolved[j].ConsumerExtension
		}
		if resolved[i].Interface != resolved[j].Interface {
			return resolved[i].Interface < resolved[j].Interface
		}
		return resolved[i].ProviderExtension < resolved[j].ProviderExtension
	})
	sort.Slice(unresolved, func(i, j int) bool {
		if unresolved[i].Code != unresolved[j].Code {
			return unresolved[i].Code < unresolved[j].Code
		}
		return unresolved[i].Detail < unresolved[j].Detail
	})
	return interfaces, resolved, unresolved, nil
}

func catalogSemanticInterface(owner semanticInterfaceContractOwner) CatalogSemanticInterface {
	identity := ExtensionIdentity{ID: owner.Extension.ID, Version: owner.Extension.Version}
	return CatalogSemanticInterface{
		ID:                     owner.Contract.ID,
		Version:                owner.Contract.Version,
		ContractFingerprint:    semanticInterfaceFingerprint(owner.Contract),
		InputSemantics:         append([]string(nil), owner.Contract.InputSemantics...),
		OutputSemantics:        append([]string(nil), owner.Contract.OutputSemantics...),
		UnknownSemantics:       owner.Contract.UnknownSemantics,
		FailureSemantics:       owner.Contract.FailureSemantics,
		ProviderConformanceRef: owner.Contract.ProviderConformanceRef,
		Extension:              identity,
		Source:                 owner.Extension.sourcePath,
		SourceFingerprint:      owner.Extension.sourceFingerprint,
	}
}

func interfaceProvidersForContract(
	selected map[string]ExtensionDescriptor,
	blockedExtensions map[string]bool,
	interfaceID string,
	contractVersion string,
	contractFingerprint string,
	conformanceProfile string,
	conformances map[string]SemanticInterfaceConformance,
) ([]semanticInterfaceProviderCandidate, []CatalogUnresolved) {
	var ids []string
	for id := range selected {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var valid []semanticInterfaceProviderCandidate
	var unresolved []CatalogUnresolved
	for _, id := range ids {
		if blockedExtensions[id] {
			continue
		}
		descriptor := selected[id]
		for _, provider := range descriptor.ProvidesInterfaces {
			if provider.Interface != interfaceID {
				continue
			}
			if provider.ContractVersion != contractVersion ||
				provider.ContractFingerprint != contractFingerprint {
				identity := ExtensionIdentity{ID: descriptor.ID, Version: descriptor.Version}
				unresolved = append(unresolved, CatalogUnresolved{
					Code:      "interface-provider-contract-mismatch",
					Extension: &identity,
					Detail: fmt.Sprintf(
						"%s claims %s@%s fingerprint %s; required contract is %s@%s fingerprint %s",
						descriptor.ID,
						provider.Interface,
						provider.ContractVersion,
						provider.ContractFingerprint,
						interfaceID,
						contractVersion,
						contractFingerprint,
					),
				})
				continue
			}
			conformanceKey := semanticInterfaceConformanceKey(
				descriptor.ID,
				interfaceID,
				contractFingerprint,
				conformanceProfile,
			)
			conformance, ok := conformances[conformanceKey]
			if !ok ||
				conformance.ContractFingerprint != contractFingerprint ||
				conformance.Profile != conformanceProfile {
				identity := ExtensionIdentity{ID: descriptor.ID, Version: descriptor.Version}
				unresolved = append(unresolved, CatalogUnresolved{
					Code:      "interface-provider-conformance-unverified",
					Extension: &identity,
					Detail: fmt.Sprintf(
						"%s has no accepted conformance for %s fingerprint %s profile %s",
						descriptor.ID,
						interfaceID,
						contractFingerprint,
						conformanceProfile,
					),
				})
				continue
			}
			valid = append(valid, semanticInterfaceProviderCandidate{
				Extension: descriptor,
				Provider:  provider,
			})
		}
	}
	return valid, unresolved
}

func interfaceContractsCompatibleWithMajor(
	owners []semanticInterfaceContractOwner,
	major int,
) []semanticInterfaceContractOwner {
	var out []semanticInterfaceContractOwner
	for _, owner := range owners {
		if compatibleMajor(owner.Contract.Version, major) {
			out = append(out, owner)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Contract.Version != out[j].Contract.Version {
			return compareVersion(out[i].Contract.Version, out[j].Contract.Version) > 0
		}
		return out[i].Extension.ID < out[j].Extension.ID
	})
	return out
}

func normalizeSemanticInterfaceWirings(
	values []SemanticInterfaceWiring,
) (map[string]SemanticInterfaceWiring, error) {
	out := map[string]SemanticInterfaceWiring{}
	for _, value := range values {
		value.ConsumerExtension = strings.TrimSpace(value.ConsumerExtension)
		value.Interface = strings.TrimSpace(value.Interface)
		value.ProviderExtension = strings.TrimSpace(value.ProviderExtension)
		if value.ConsumerExtension == "" || value.Interface == "" || value.ProviderExtension == "" {
			return nil, fmt.Errorf("semantic interface wiring requires consumerExtension, interface, and providerExtension")
		}
		key := semanticInterfaceWiringKey(value.ConsumerExtension, value.Interface)
		if existing, ok := out[key]; ok && existing.ProviderExtension != value.ProviderExtension {
			return nil, fmt.Errorf(
				"semantic interface wiring %s has multiple providers: %s and %s",
				key,
				existing.ProviderExtension,
				value.ProviderExtension,
			)
		}
		out[key] = value
	}
	return out, nil
}

func semanticInterfaceWiringKey(consumerExtension, interfaceID string) string {
	return fmt.Sprintf("%q|%q", consumerExtension, interfaceID)
}

func semanticInterfaceProviderByExtension(
	values []semanticInterfaceProviderCandidate,
	extensionID string,
) (semanticInterfaceProviderCandidate, bool) {
	for _, value := range values {
		if value.Extension.ID == extensionID {
			return value, true
		}
	}
	return semanticInterfaceProviderCandidate{}, false
}

func catalogInterfaceWiring(
	consumer ExtensionDescriptor,
	contract SemanticInterfaceContract,
	contractFingerprint string,
	providerExtension ExtensionDescriptor,
	provider SemanticInterfaceProvider,
	conformance SemanticInterfaceConformance,
	selection string,
) CatalogInterfaceWiring {
	return CatalogInterfaceWiring{
		ConsumerExtension:          consumer.ID,
		Interface:                  contract.ID,
		ContractVersion:            contract.Version,
		ContractFingerprint:        contractFingerprint,
		ProviderExtension:          providerExtension.ID,
		ProviderConformanceProfile: conformance.Profile,
		ConformanceEvidence:        conformance.EvidenceFingerprint,
		Selection:                  selection,
	}
}

func normalizeSemanticInterfaceConformance(
	values []SemanticInterfaceConformance,
) (map[string]SemanticInterfaceConformance, error) {
	out := map[string]SemanticInterfaceConformance{}
	for _, value := range values {
		value.ProviderExtension = strings.TrimSpace(value.ProviderExtension)
		value.Interface = strings.TrimSpace(value.Interface)
		value.ContractFingerprint = strings.TrimSpace(value.ContractFingerprint)
		value.Profile = strings.TrimSpace(value.Profile)
		value.EvidenceFingerprint = strings.TrimSpace(value.EvidenceFingerprint)
		if value.ProviderExtension == "" || value.Interface == "" ||
			!validSHA256Fingerprint(value.ContractFingerprint) ||
			value.Profile == "" ||
			!validSHA256Fingerprint(value.EvidenceFingerprint) {
			return nil, fmt.Errorf("semantic interface conformance requires exact provider, interface, contract fingerprint, profile, and evidence fingerprint")
		}
		key := semanticInterfaceConformanceKey(
			value.ProviderExtension,
			value.Interface,
			value.ContractFingerprint,
			value.Profile,
		)
		if existing, ok := out[key]; ok && existing != value {
			return nil, fmt.Errorf("semantic interface conformance %s is declared more than once with different evidence", key)
		}
		out[key] = value
	}
	return out, nil
}

func semanticInterfaceConformanceKey(
	providerExtension string,
	interfaceID string,
	contractFingerprint string,
	profile string,
) string {
	return fmt.Sprintf(
		"%q|%q|%q|%q",
		providerExtension,
		interfaceID,
		contractFingerprint,
		profile,
	)
}
