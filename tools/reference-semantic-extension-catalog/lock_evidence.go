package main

import "fmt"

// verifySemanticLockAcceptedEvidenceBindings checks that the caller supplied
// currently accepted Evidence for every Conformance and namespace-authority
// obligation recorded in the semantic lock.
//
// This is deliberately separate from verifySemanticLockReplay. Replay proves
// that the locked meaning can be reconstructed from exact declarations.
// Evidence binding proves only that accepted Evidence inputs are present and
// exactly bound to the locked contract/profile or namespace/owner. It does not
// prove publisher trust or the authority that issued that Evidence.
func verifySemanticLockAcceptedEvidenceBindings(
	lock SemanticLock,
	conformances []SemanticInterfaceConformance,
	namespaceAuthorities []SemanticNamespaceAuthority,
) error {
	normalizeSemanticLock(&lock)

	acceptedConformance, err := normalizeSemanticInterfaceConformance(conformances)
	if err != nil {
		return err
	}
	acceptedNamespaceAuthority, err := normalizeSemanticNamespaceAuthorities(namespaceAuthorities)
	if err != nil {
		return err
	}

	for _, wiring := range lock.InterfaceWirings {
		key := semanticInterfaceConformanceKey(
			wiring.ProviderExtension,
			wiring.Interface,
			wiring.ContractFingerprint,
			wiring.ProviderConformanceProfile,
		)
		if _, ok := acceptedConformance[key]; !ok {
			return fmt.Errorf(
				"semantic lock requires accepted conformance Evidence for provider %s interface %s contract %s profile %s",
				wiring.ProviderExtension,
				wiring.Interface,
				wiring.ContractFingerprint,
				wiring.ProviderConformanceProfile,
			)
		}
	}

	for _, authority := range lock.NamespaceAuthorities {
		accepted, ok := acceptedNamespaceAuthority[authority.Namespace]
		if !ok {
			return fmt.Errorf(
				"semantic lock requires accepted namespace Authority Evidence for %s -> %s",
				authority.Namespace,
				authority.Extension,
			)
		}
		if accepted.Extension != authority.Extension {
			return fmt.Errorf(
				"semantic lock namespace %s is owned by %s, but accepted Authority Evidence names %s",
				authority.Namespace,
				authority.Extension,
				accepted.Extension,
			)
		}
	}

	return nil
}
