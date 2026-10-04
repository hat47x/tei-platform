package main

import (
	"fmt"
	"sort"
	"strings"
)

type SemanticNamespaceAuthority struct {
	Namespace           string `json:"namespace"`
	Extension           string `json:"extension"`
	EvidenceFingerprint string `json:"evidenceFingerprint"`
}

type CatalogNamespaceAuthority struct {
	Namespace           string `json:"namespace"`
	Extension           string `json:"extension"`
	EvidenceFingerprint string `json:"evidenceFingerprint,omitempty"`
}

func normalizeSemanticNamespaceAuthorities(
	values []SemanticNamespaceAuthority,
) (map[string]SemanticNamespaceAuthority, error) {
	out := map[string]SemanticNamespaceAuthority{}
	for _, value := range values {
		value.Namespace = strings.TrimSpace(value.Namespace)
		value.Extension = strings.TrimSpace(value.Extension)
		value.EvidenceFingerprint = strings.TrimSpace(value.EvidenceFingerprint)
		if value.Namespace == "" || value.Extension == "" ||
			!validSHA256Fingerprint(value.EvidenceFingerprint) {
			return nil, fmt.Errorf(
				"semantic namespace authority requires namespace, extension, and evidence fingerprint",
			)
		}
		if existing, ok := out[value.Namespace]; ok && existing != value {
			return nil, fmt.Errorf(
				"semantic namespace %s has conflicting authority grants for %s and %s",
				value.Namespace,
				existing.Extension,
				value.Extension,
			)
		}
		out[value.Namespace] = value
	}
	return out, nil
}

func isReservedStandardNamespace(namespace string) bool {
	namespace = strings.TrimSpace(namespace)
	return namespace == "std" || strings.HasPrefix(namespace, "std.")
}

func resolveNamespaceAuthorities(
	selected map[string]ExtensionDescriptor,
	grants []SemanticNamespaceAuthority,
) (
	map[string]bool,
	[]CatalogNamespaceAuthority,
	[]CatalogUnresolved,
	error,
) {
	accepted, err := normalizeSemanticNamespaceAuthorities(grants)
	if err != nil {
		return nil, nil, nil, err
	}

	owners := map[string][]string{}
	for id, descriptor := range selected {
		owners[descriptor.Namespace] = append(owners[descriptor.Namespace], id)
	}
	for namespace := range owners {
		sort.Strings(owners[namespace])
	}

	blocked := map[string]bool{}
	var resolved []CatalogNamespaceAuthority
	var unresolved []CatalogUnresolved
	usedGrants := map[string]bool{}

	var namespaces []string
	for namespace := range owners {
		namespaces = append(namespaces, namespace)
	}
	sort.Strings(namespaces)

	for _, namespace := range namespaces {
		namespaceOwners := owners[namespace]
		grant, hasGrant := accepted[namespace]

		if isReservedStandardNamespace(namespace) && !hasGrant {
			for _, owner := range namespaceOwners {
				blocked[owner] = true
			}
			unresolved = append(unresolved, CatalogUnresolved{
				Code: "namespace-authority-missing",
				Detail: fmt.Sprintf(
					"reserved namespace %q requires accepted authority; claimants: %s",
					namespace,
					strings.Join(namespaceOwners, ", "),
				),
			})
			continue
		}

		if hasGrant {
			usedGrants[namespace] = true
			if _, selectedOwner := selected[grant.Extension]; !selectedOwner {
				for _, owner := range namespaceOwners {
					blocked[owner] = true
				}
				unresolved = append(unresolved, CatalogUnresolved{
					Code: "namespace-authority-owner-unavailable",
					Detail: fmt.Sprintf(
						"namespace %q authority selects %s, which is not in the resolved Semantic Set",
						namespace,
						grant.Extension,
					),
				})
				continue
			}
			if selected[grant.Extension].Namespace != namespace {
				for _, owner := range namespaceOwners {
					blocked[owner] = true
				}
				unresolved = append(unresolved, CatalogUnresolved{
					Code: "namespace-authority-mismatch",
					Detail: fmt.Sprintf(
						"namespace %q authority selects %s, whose descriptor namespace is %q",
						namespace,
						grant.Extension,
						selected[grant.Extension].Namespace,
					),
				})
				continue
			}

			resolved = append(resolved, CatalogNamespaceAuthority{
				Namespace:           namespace,
				Extension:           grant.Extension,
				EvidenceFingerprint: grant.EvidenceFingerprint,
			})
			for _, owner := range namespaceOwners {
				if owner == grant.Extension {
					continue
				}
				blocked[owner] = true
				unresolved = append(unresolved, CatalogUnresolved{
					Code: "namespace-authority-mismatch",
					Detail: fmt.Sprintf(
						"namespace %q is owned by %s; unauthorized claimant %s",
						namespace,
						grant.Extension,
						owner,
					),
				})
			}
			continue
		}

		if len(namespaceOwners) > 1 {
			for _, owner := range namespaceOwners {
				blocked[owner] = true
			}
			unresolved = append(unresolved, CatalogUnresolved{
				Code: "namespace-ownership-conflict",
				Detail: fmt.Sprintf(
					"namespace %q has multiple owners: %s",
					namespace,
					strings.Join(namespaceOwners, ", "),
				),
			})
		}
	}

	for namespace, grant := range accepted {
		if usedGrants[namespace] {
			continue
		}
		unresolved = append(unresolved, CatalogUnresolved{
			Code: "namespace-authority-unrequired",
			Detail: fmt.Sprintf(
				"namespace authority %q -> %s does not match a selected namespace",
				namespace,
				grant.Extension,
			),
		})
	}

	sort.Slice(resolved, func(i, j int) bool {
		if resolved[i].Namespace != resolved[j].Namespace {
			return resolved[i].Namespace < resolved[j].Namespace
		}
		return resolved[i].Extension < resolved[j].Extension
	})
	sort.Slice(unresolved, func(i, j int) bool {
		if unresolved[i].Code != unresolved[j].Code {
			return unresolved[i].Code < unresolved[j].Code
		}
		return unresolved[i].Detail < unresolved[j].Detail
	})
	return blocked, resolved, unresolved, nil
}
