package xeadsource

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

type TranscriptionContractRule struct {
	Classification  []string `json:"classification"`
	Extension       string   `json:"extension,omitempty"`
	AuthorityLimits []string `json:"authorityLimits,omitempty"`
}

type SourceProfile struct {
	RootKind                string   `json:"rootKind"`
	SupportedFormatVersions []string `json:"supportedFormatVersions"`
}

type TranscriptionContract struct {
	ConstructMappings map[string]map[string]TranscriptionContractRule `json:"constructMappings"`
	SourceProfiles    map[string]SourceProfile                        `json:"sourceProfiles"`
}

func ClassificationPolicyFromContract(format Format, contractBytes []byte) (ClassificationPolicy, error) {
	contract, err := parseTranscriptionContract(contractBytes)
	if err != nil {
		return nil, err
	}
	return classificationPolicyFromContract(format, contract)
}

func InspectWithContract(format Format, source []byte, contractBytes []byte) (Inspection, error) {
	contract, err := parseTranscriptionContract(contractBytes)
	if err != nil {
		return Inspection{}, err
	}
	policy, err := classificationPolicyFromContract(format, contract)
	if err != nil {
		return Inspection{}, err
	}
	document, err := Parse(format, source)
	if err != nil {
		return Inspection{}, err
	}
	if err := validateSourceProfile(format, document, contract); err != nil {
		return Inspection{}, err
	}
	inspection := inspectDocument(document, policy)
	sum := sha256.Sum256(contractBytes)
	inspection.TranscriptionContractFingerprint = hex.EncodeToString(sum[:])
	return inspection, nil
}

func parseTranscriptionContract(contractBytes []byte) (TranscriptionContract, error) {
	var contract TranscriptionContract
	if err := json.Unmarshal(contractBytes, &contract); err != nil {
		return TranscriptionContract{}, fmt.Errorf("parse transcription contract: %w", err)
	}
	return contract, nil
}

func classificationPolicyFromContract(
	format Format,
	contract TranscriptionContract,
) (ClassificationPolicy, error) {
	mappings, ok := contract.ConstructMappings[string(format)]
	if !ok {
		return nil, fmt.Errorf("transcription contract has no mapping for format %q", format)
	}
	policy := make(ClassificationPolicy, len(mappings))
	for sourceName, mapping := range mappings {
		policy[sourceName] = CandidateRule{
			Classifications: append([]string(nil), mapping.Classification...),
			Extension:       mapping.Extension,
			AuthorityLimits: append([]string(nil), mapping.AuthorityLimits...),
		}
	}
	return policy, nil
}

func validateSourceProfile(
	format Format,
	document Document,
	contract TranscriptionContract,
) error {
	profile, ok := contract.SourceProfiles[string(format)]
	if !ok {
		return fmt.Errorf(
			"transcription contract has no source profile for format %q",
			format,
		)
	}
	if profile.RootKind == "" || len(profile.SupportedFormatVersions) == 0 {
		return fmt.Errorf(
			"transcription contract source profile for %q is incomplete",
			format,
		)
	}
	if document.Root.Name != profile.RootKind {
		return fmt.Errorf(
			"source root kind %q is not supported for format %q; want %q",
			document.Root.Name,
			format,
			profile.RootKind,
		)
	}

	var formatVersions []string
	for _, attribute := range document.Root.Attributes {
		if attribute.Name == "FormatVersion" {
			formatVersions = append(formatVersions, attribute.Value)
		}
	}
	if len(formatVersions) != 1 {
		return fmt.Errorf(
			"source format %q requires exactly one root FormatVersion; found %d",
			format,
			len(formatVersions),
		)
	}
	for _, supported := range profile.SupportedFormatVersions {
		if formatVersions[0] == supported {
			return nil
		}
	}
	return fmt.Errorf(
		"source format %q version %q is not supported by transcription contract",
		format,
		formatVersions[0],
	)
}
