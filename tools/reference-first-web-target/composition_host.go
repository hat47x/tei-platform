package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"

	"tei/internal/composition"
)

// Reference contract fingerprints identify typed APIs, not semantic conformity.
func scenarioContract(id, shape string) composition.Contract {
	return composition.Contract{ID: id, Version: "1", Shape: fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(shape)))}
}

func projectCompositionBinding(binding composition.Binding) *compositionTrace {
	return &compositionTrace{ContractID: binding.ContractID, ContractVersion: binding.ContractVersion, ProviderID: binding.ProviderID, ArtifactDigest: binding.ArtifactDigest, Generation: binding.Generation}
}

// Hash the actual linked executable. This is identity, not publisher trust or
// business authority. The helper has no Application-specific dependency.
func runningExecutableDigest() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", err
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("sha256:%x", h.Sum(nil)), nil
}

func referenceInstanceGuard(digest string, offers map[string]composition.Contract) func(composition.Binding) error {
	// Copy host configuration so a caller cannot mutate permissions after setup.
	selected := make(map[string]composition.Contract, len(offers))
	for id, c := range offers {
		selected[id] = c
	}
	return func(b composition.Binding) error {
		c, ok := selected[b.ProviderID]
		if !ok || b.ContractID != c.ID || b.ContractVersion != c.Version || b.ArtifactDigest != digest || b.Generation != 1 {
			return fmt.Errorf("reference host rejects instance binding")
		}
		return nil
	}
}
