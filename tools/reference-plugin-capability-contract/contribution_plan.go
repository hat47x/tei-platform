package main

import (
	"fmt"
	"sort"
)

// MeaningPin identifies shared design semantics, not a phase-specific API shape.
// These caller-supplied pins are reference inputs, not publisher authentication.
type MeaningPin struct {
	ID          string
	Version     string
	Fingerprint string
}

type ContributionRequirement struct {
	Slot                string
	Meaning             MeaningPin
	Target              string
	Scope               AuthorityScope
	ContractFingerprint string
	Requirement         Requirement
	Wirings             []Wiring
}

// A package can supply several contributions; different packages may realize
// the same meaning in different phases. Package versions need not be equal.
type ContributionOffer struct {
	Meaning  MeaningPin
	Target   string
	Provider Provider
}

type ContributionBinding struct {
	Slot       string
	Meaning    MeaningPin
	Target     string
	Resolution Resolution
}

// planContributions checks an already pinned composition without loading code
// or granting authority. Provider choice delegates to the existing resolver.
// Target is an exact reference label, not a platform compatibility algorithm.
func planContributions(requirements []ContributionRequirement, offers []ContributionOffer,
	contracts []Contract, evidence []ConformanceEvidence) ([]ContributionBinding, error) {
	bindings := make([]ContributionBinding, 0, len(requirements))
	seen := map[string]bool{}
	for _, requirement := range requirements {
		if requirement.Slot == "" || seen[requirement.Slot] || requirement.Target == "" ||
			requirement.Meaning.ID == "" || requirement.Meaning.Fingerprint == "" ||
			requirement.ContractFingerprint == "" {
			return nil, fmt.Errorf("incomplete or duplicate contribution slot %q", requirement.Slot)
		}
		if _, err := parseVersion(requirement.Meaning.Version); err != nil {
			return nil, fmt.Errorf("slot %s meaning version: %w", requirement.Slot, err)
		}
		seen[requirement.Slot] = true
		var providers []Provider
		for _, offer := range offers {
			if offer.Meaning == requirement.Meaning && offer.Target == requirement.Target &&
				offer.Provider.ContractFingerprint == requirement.ContractFingerprint {
				providers = append(providers, offer.Provider)
			}
		}
		resolution, err := resolveCapability(ResolveInput{
			Scope: requirement.Scope, Requirement: requirement.Requirement,
			Contracts: contracts, Providers: providers, Evidence: evidence, Wirings: requirement.Wirings,
		})
		if err != nil {
			return nil, fmt.Errorf("slot %s: %w", requirement.Slot, err)
		}
		bindings = append(bindings, ContributionBinding{
			Slot: requirement.Slot, Meaning: requirement.Meaning, Target: requirement.Target, Resolution: resolution,
		})
	}
	sort.Slice(bindings, func(i, j int) bool { return bindings[i].Slot < bindings[j].Slot })
	return bindings, nil
}
