package main

import "sort"

type CatalogResolvedDependency struct {
	Requirement ExtensionRequirement `json:"requirement"`
	Extension   ExtensionIdentity    `json:"extension"`
}

type CatalogDependencyPath struct {
	Root ExtensionRequirement `json:"root"`
	Path []ExtensionIdentity  `json:"path"`
}

type CatalogDependencyExplanation struct {
	Extension  ExtensionIdentity           `json:"extension"`
	Requires   []CatalogResolvedDependency `json:"requires,omitempty"`
	RequiredBy []ExtensionIdentity         `json:"requiredBy,omitempty"`
	RootPaths  []CatalogDependencyPath     `json:"rootPaths"`
}

func buildDependencyExplanations(
	selected map[string]ExtensionDescriptor,
	roots []ExtensionRequirement,
) []CatalogDependencyExplanation {
	ids := make([]string, 0, len(selected))
	for id := range selected {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	adjacency := map[string][]CatalogResolvedDependency{}
	reverse := map[string][]ExtensionIdentity{}
	for _, id := range ids {
		descriptor := selected[id]
		requirements := append([]ExtensionRequirement(nil), descriptor.Requires...)
		sort.Slice(requirements, func(i, j int) bool {
			if requirements[i].Extension != requirements[j].Extension {
				return requirements[i].Extension < requirements[j].Extension
			}
			return requirements[i].CompatibleMajor < requirements[j].CompatibleMajor
		})
		for _, requirement := range requirements {
			target, ok := selected[requirement.Extension]
			if !ok {
				continue
			}
			targetIdentity := extensionIdentity(target)
			adjacency[id] = append(adjacency[id], CatalogResolvedDependency{
				Requirement: requirement,
				Extension:   targetIdentity,
			})
			reverse[target.ID] = append(reverse[target.ID], extensionIdentity(descriptor))
		}
	}
	for id := range reverse {
		sort.Slice(reverse[id], func(i, j int) bool {
			return compareExtensionIdentity(reverse[id][i], reverse[id][j]) < 0
		})
	}

	pathsByTarget := map[string][]CatalogDependencyPath{}
	normalizedRoots := append([]ExtensionRequirement(nil), roots...)
	sort.Slice(normalizedRoots, func(i, j int) bool {
		if normalizedRoots[i].Extension != normalizedRoots[j].Extension {
			return normalizedRoots[i].Extension < normalizedRoots[j].Extension
		}
		return normalizedRoots[i].CompatibleMajor < normalizedRoots[j].CompatibleMajor
	})
	for _, root := range normalizedRoots {
		if _, ok := selected[root.Extension]; !ok {
			continue
		}
		for targetID, path := range canonicalShortestPaths(root.Extension, selected, adjacency) {
			pathsByTarget[targetID] = append(pathsByTarget[targetID], CatalogDependencyPath{
				Root: root,
				Path: path,
			})
		}
	}
	for id := range pathsByTarget {
		sort.Slice(pathsByTarget[id], func(i, j int) bool {
			left, right := pathsByTarget[id][i], pathsByTarget[id][j]
			if left.Root.Extension != right.Root.Extension {
				return left.Root.Extension < right.Root.Extension
			}
			if left.Root.CompatibleMajor != right.Root.CompatibleMajor {
				return left.Root.CompatibleMajor < right.Root.CompatibleMajor
			}
			return compareIdentityPath(left.Path, right.Path) < 0
		})
	}

	out := make([]CatalogDependencyExplanation, 0, len(ids))
	for _, id := range ids {
		out = append(out, CatalogDependencyExplanation{
			Extension:  extensionIdentity(selected[id]),
			Requires:   append([]CatalogResolvedDependency(nil), adjacency[id]...),
			RequiredBy: append([]ExtensionIdentity(nil), reverse[id]...),
			RootPaths:  append([]CatalogDependencyPath(nil), pathsByTarget[id]...),
		})
	}
	return out
}

func canonicalShortestPaths(
	rootID string,
	selected map[string]ExtensionDescriptor,
	adjacency map[string][]CatalogResolvedDependency,
) map[string][]ExtensionIdentity {
	previous := map[string]string{rootID: ""}
	queue := []string{rootID}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		neighbors := append([]CatalogResolvedDependency(nil), adjacency[current]...)
		sort.Slice(neighbors, func(i, j int) bool {
			return compareExtensionIdentity(neighbors[i].Extension, neighbors[j].Extension) < 0
		})
		for _, edge := range neighbors {
			next := edge.Extension.ID
			if _, seen := previous[next]; seen {
				continue
			}
			previous[next] = current
			queue = append(queue, next)
		}
	}

	out := map[string][]ExtensionIdentity{}
	for targetID := range previous {
		var reversed []ExtensionIdentity
		for id := targetID; id != ""; id = previous[id] {
			descriptor, ok := selected[id]
			if !ok {
				break
			}
			reversed = append(reversed, extensionIdentity(descriptor))
		}
		path := make([]ExtensionIdentity, len(reversed))
		for i := range reversed {
			path[len(reversed)-1-i] = reversed[i]
		}
		out[targetID] = path
	}
	return out
}

func extensionIdentity(descriptor ExtensionDescriptor) ExtensionIdentity {
	return ExtensionIdentity{ID: descriptor.ID, Version: descriptor.Version}
}

func compareExtensionIdentity(left, right ExtensionIdentity) int {
	if left.ID < right.ID {
		return -1
	}
	if left.ID > right.ID {
		return 1
	}
	return compareVersion(left.Version, right.Version)
}

func compareIdentityPath(left, right []ExtensionIdentity) int {
	limit := len(left)
	if len(right) < limit {
		limit = len(right)
	}
	for i := 0; i < limit; i++ {
		if cmp := compareExtensionIdentity(left[i], right[i]); cmp != 0 {
			return cmp
		}
	}
	switch {
	case len(left) < len(right):
		return -1
	case len(left) > len(right):
		return 1
	default:
		return 0
	}
}
