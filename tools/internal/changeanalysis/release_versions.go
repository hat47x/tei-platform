package changeanalysis

import (
	"strconv"
	"strings"
)

// compareReleaseVersion orders two numeric major.minor.patch versions. Callers
// pass only versions that parseReleaseVersion accepted.
func compareReleaseVersion(a, b string) int {
	av, _ := parseReleaseVersion(a)
	bv, _ := parseReleaseVersion(b)
	for i := 0; i < 3; i++ {
		if av[i] < bv[i] {
			return -1
		}
		if av[i] > bv[i] {
			return 1
		}
	}
	return 0
}

func parseReleaseVersion(v string) ([3]int, bool) {
	var out [3]int
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}
