// Package gobuilder contains Go-specific reference build inputs shared by
// independent TEI build subjects. It does not determine Plugin/Runtime Engine
// identity, grant build authority, or attest OS-level isolation.
package gobuilder

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FixedSettings returns the exact Go settings used for each reference build.
// It is also used to record stable build policy independently of state paths.
func FixedSettings(goos, goarch string) []string {
	out := []string{
		"CGO_ENABLED=0",
		"GOARCH=" + goarch,
		"GOENV=off",
		"GOEXPERIMENT=",
		"GOFLAGS=",
		"GOOS=" + goos,
		"GOPROXY=off",
		"GOSUMDB=off",
		"GOTOOLCHAIN=local",
		"GOVCS=*:off",
		"GOWORK=off",
	}
	sort.Strings(out)
	return out
}

// AllowedHostEnvironment is deny-by-default: only the platform-required
// Windows variables may be inherited. Duplicate case-insensitive keys with
// different values are ambiguous and rejected.
func AllowedHostEnvironment(base []string, goos string) ([]string, error) {
	allowed := map[string]struct{}{}
	if goos == "windows" {
		for _, key := range []string{"COMSPEC", "PATHEXT", "SYSTEMROOT", "WINDIR"} {
			allowed[key] = struct{}{}
		}
	}
	values := map[string]string{}
	for _, entry := range base {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		upper := strings.ToUpper(key)
		if _, ok := allowed[upper]; !ok {
			continue
		}
		normalized := upper + "=" + value
		if old, exists := values[upper]; exists && old != normalized {
			return nil, fmt.Errorf("conflicting inherited Host environment for %s", upper)
		}
		values[upper] = normalized
	}
	out := make([]string, 0, len(values))
	for _, entry := range values {
		out = append(out, entry)
	}
	sort.Strings(out)
	return out, nil
}

// Environment constructs the subprocess environment without inheriting
// PATH, credentials, proxy settings or locale. Each nonempty stateDir is
// supplied by the caller and must be distinct for independent build attempts.
func Environment(base []string, stateDir, goos, goarch string) ([]string, error) {
	host, err := AllowedHostEnvironment(base, goos)
	if err != nil {
		return nil, err
	}
	out := append([]string(nil), host...)
	out = append(out, FixedSettings(goos, goarch)...)
	if stateDir != "" {
		if err := os.MkdirAll(stateDir, 0o755); err != nil {
			return nil, fmt.Errorf("create isolated build state: %w", err)
		}
		home := filepath.Join(stateDir, "home")
		temp := filepath.Join(stateDir, "tmp")
		for _, dir := range []string{home, temp} {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("create isolated build directory: %w", err)
			}
		}
		out = append(out,
			"GOCACHE="+filepath.Join(stateDir, "cache"),
			"GOMODCACHE="+filepath.Join(stateDir, "modcache"),
			"GOPATH="+filepath.Join(stateDir, "gopath"),
			"HOME="+home,
			"USERPROFILE="+home,
			"TMPDIR="+temp,
			"TMP="+temp,
			"TEMP="+temp,
		)
	}
	sort.Strings(out)
	return out, nil
}
