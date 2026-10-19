package deliveryprofile

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type semver struct {
	major int
	minor int
	patch int
}

func parseSemver(value string) (semver, error) {
	value = strings.TrimSpace(value)
	parts := strings.Split(value, ".")
	if len(parts) < 1 || len(parts) > 3 {
		return semver{}, fmt.Errorf("version %q is not a supported numeric version", value)
	}
	numbers := [3]int{}
	for i, part := range parts {
		number, err := strconv.Atoi(part)
		if err != nil || number < 0 {
			return semver{}, fmt.Errorf("version %q has a non-numeric component", value)
		}
		numbers[i] = number
	}
	return semver{major: numbers[0], minor: numbers[1], patch: numbers[2]}, nil
}

func compareSemver(a, b semver) int {
	switch {
	case a.major != b.major:
		if a.major < b.major {
			return -1
		}
		return 1
	case a.minor != b.minor:
		if a.minor < b.minor {
			return -1
		}
		return 1
	case a.patch != b.patch:
		if a.patch < b.patch {
			return -1
		}
		return 1
	}
	return 0
}

type comparator struct {
	op      string
	version semver
}

func (c comparator) match(value semver) bool {
	switch c.op {
	case "=":
		return compareSemver(value, c.version) == 0
	case ">":
		return compareSemver(value, c.version) > 0
	case ">=":
		return compareSemver(value, c.version) >= 0
	case "<":
		return compareSemver(value, c.version) < 0
	case "<=":
		return compareSemver(value, c.version) <= 0
	}
	return false
}

func upperForCaret(v semver) semver {
	switch {
	case v.major > 0:
		return semver{major: v.major + 1}
	case v.minor > 0:
		return semver{minor: v.minor + 1}
	default:
		return semver{patch: v.patch + 1}
	}
}

func upperForTilde(v semver) semver {
	return semver{major: v.major, minor: v.minor + 1}
}

func parseConstraint(value string) ([]comparator, error) {
	value = strings.TrimSpace(strings.ReplaceAll(value, ",", " "))
	if value == "" {
		return nil, fmt.Errorf("version constraint is empty")
	}
	var comparators []comparator
	wildcard := false
	for _, token := range strings.Fields(value) {
		if token == "*" || token == "x" || token == "X" {
			wildcard = true
			continue
		}
		switch {
		case strings.HasPrefix(token, "^"):
			lower, err := parseSemver(strings.TrimPrefix(token, "^"))
			if err != nil {
				return nil, err
			}
			comparators = append(comparators,
				comparator{op: ">=", version: lower},
				comparator{op: "<", version: upperForCaret(lower)})
		case strings.HasPrefix(token, "~"):
			lower, err := parseSemver(strings.TrimPrefix(token, "~"))
			if err != nil {
				return nil, err
			}
			comparators = append(comparators,
				comparator{op: ">=", version: lower},
				comparator{op: "<", version: upperForTilde(lower)})
		case strings.HasPrefix(token, ">="), strings.HasPrefix(token, "<="),
			strings.HasPrefix(token, ">"), strings.HasPrefix(token, "<"), strings.HasPrefix(token, "="):
			op := ""
			for _, candidate := range []string{">=", "<=", ">", "<", "="} {
				if strings.HasPrefix(token, candidate) {
					op = candidate
					break
				}
			}
			version, err := parseSemver(strings.TrimPrefix(token, op))
			if err != nil {
				return nil, err
			}
			comparators = append(comparators, comparator{op: op, version: version})
		default:
			version, err := parseSemver(token)
			if err != nil {
				return nil, err
			}
			comparators = append(comparators, comparator{op: "=", version: version})
		}
	}
	if len(comparators) == 0 && !wildcard {
		return nil, fmt.Errorf("version constraint %q has no comparators", value)
	}
	return comparators, nil
}

func matchesConstraint(value semver, comparators []comparator) bool {
	for _, comparator := range comparators {
		if !comparator.match(value) {
			return false
		}
	}
	return true
}

// ResolveConstraint selects the highest catalog version that satisfies the
// constraint. A bare version such as "1.2" is an exact match.
func (c Catalog) ResolveConstraint(id, constraint string) (CatalogEntry, error) {
	versions, ok := c[id]
	if !ok {
		return CatalogEntry{}, fmt.Errorf("plugin %s is not in the catalog", id)
	}
	comparators, err := parseConstraint(constraint)
	if err != nil {
		return CatalogEntry{}, fmt.Errorf("plugin %s: %w", id, err)
	}
	type candidate struct {
		raw     string
		version semver
		entry   CatalogEntry
	}
	var matches []candidate
	for raw, entry := range versions {
		version, err := parseSemver(raw)
		if err != nil {
			return CatalogEntry{}, fmt.Errorf("plugin %s version %q: %w", id, raw, err)
		}
		if matchesConstraint(version, comparators) {
			matches = append(matches, candidate{raw: raw, version: version, entry: entry})
		}
	}
	if len(matches) == 0 {
		return CatalogEntry{}, fmt.Errorf("plugin %s has no version matching %q", id, constraint)
	}
	sort.Slice(matches, func(i, j int) bool {
		return compareSemver(matches[i].version, matches[j].version) > 0
	})
	if len(matches) > 1 && compareSemver(matches[0].version, matches[1].version) == 0 {
		return CatalogEntry{}, fmt.Errorf("plugin %s has ambiguous versions %q and %q", id, matches[0].raw, matches[1].raw)
	}
	return matches[0].entry, nil
}
