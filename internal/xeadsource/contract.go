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

type TranscriptionContract struct {
	ConstructMappings map[string]map[string]TranscriptionContractRule `json:"constructMappings"`
}

func ClassificationPolicyFromContract(format Format, contractBytes []byte) (ClassificationPolicy, error) {
	var contract TranscriptionContract
	if err := json.Unmarshal(contractBytes, &contract); err != nil {
		return nil, fmt.Errorf("parse transcription contract: %w", err)
	}
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

func InspectWithContract(format Format, source []byte, contractBytes []byte) (Inspection, error) {
	policy, err := ClassificationPolicyFromContract(format, contractBytes)
	if err != nil {
		return Inspection{}, err
	}
	inspection, err := Inspect(format, source, policy)
	if err != nil {
		return Inspection{}, err
	}
	sum := sha256.Sum256(contractBytes)
	inspection.TranscriptionContractFingerprint = hex.EncodeToString(sum[:])
	return inspection, nil
}
