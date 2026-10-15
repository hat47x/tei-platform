package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// HostProcessEnvironmentProfile is a Host-owned, explicit description of the
// environment supplied to a Process Plugin. It must be assembled from trusted
// Host configuration, never from a plugin request or untrusted descriptor.
//
// No field reads the Host process environment. The profile controls only child
// environment variables; it does not enforce filesystem or network isolation,
// validate the executable's provenance, or deliver/revoke secrets.
type HostProcessEnvironmentProfile struct {
	// SearchDirectories form the child's PATH in the given order. Every entry
	// must be absolute; empty, relative and path-list-containing entries fail.
	// Omit this field when only explicit executable paths are permitted.
	SearchDirectories []string
	// HomeDirectory sets HOME on Unix and USERPROFILE on Windows, if provided.
	HomeDirectory string
	// TemporaryDirectory sets TMPDIR on Unix and TEMP/TMP on Windows, if given.
	TemporaryDirectory string
	// Additional holds Host-approved, non-standard variables, not inherited
	// Host os.Environ values. PATH/HOME/TMPDIR/TEMP/TMP/USERPROFILE cannot be
	// overridden through this field. Secret delivery requires separate Host
	// authority and must not be configured from plugin-supplied values.
	Additional map[string]string
}

// Environment returns a deterministic complete child environment. An empty
// profile returns a non-nil empty slice so exec.Cmd.Env never inherits Host
// variables. The Host remains responsible for approving each supplied value.
func (p HostProcessEnvironmentProfile) Environment() ([]string, error) {
	env := make([]string, 0, len(p.Additional)+5)
	paths := make([]string, 0, len(p.SearchDirectories))
	for _, directory := range p.SearchDirectories {
		if err := validateHostProcessEnvironmentDirectory(directory); err != nil {
			return nil, err
		}
		paths = append(paths, filepath.Clean(directory))
	}
	if len(paths) != 0 {
		env = append(env, "PATH="+strings.Join(paths, string(os.PathListSeparator)))
	}
	if p.HomeDirectory != "" {
		if err := validateHostProcessEnvironmentDirectory(p.HomeDirectory); err != nil {
			return nil, err
		}
		if runtime.GOOS == "windows" {
			env = append(env, "USERPROFILE="+filepath.Clean(p.HomeDirectory))
		} else {
			env = append(env, "HOME="+filepath.Clean(p.HomeDirectory))
		}
	}
	if p.TemporaryDirectory != "" {
		if err := validateHostProcessEnvironmentDirectory(p.TemporaryDirectory); err != nil {
			return nil, err
		}
		if runtime.GOOS == "windows" {
			env = append(env, "TEMP="+filepath.Clean(p.TemporaryDirectory), "TMP="+filepath.Clean(p.TemporaryDirectory))
		} else {
			env = append(env, "TMPDIR="+filepath.Clean(p.TemporaryDirectory))
		}
	}
	keys := make([]string, 0, len(p.Additional))
	for key := range p.Additional {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		// A key containing '=' would be interpreted as a different variable
		// by exec.Cmd even though the map key is not a reserved name.
		if key == "" || strings.ContainsRune(key, '=') {
			return nil, fmt.Errorf("process environment profile contains an invalid variable name")
		}
		if isReservedHostProcessEnvironmentKey(key) {
			return nil, fmt.Errorf("process environment profile cannot override a managed directory variable")
		}
		env = append(env, key+"="+p.Additional[key])
	}
	if err := validateProcessEnvironment(env); err != nil {
		return nil, err
	}
	return env, nil
}

// NewStdioProcessLauncherForHostProfile binds a validated Host profile to the
// real process transport, without changing its existing direct-Env API.
func NewStdioProcessLauncherForHostProfile(profile HostProcessEnvironmentProfile) (StdioProcessLauncher, error) {
	env, err := profile.Environment()
	if err != nil {
		return StdioProcessLauncher{}, err
	}
	return StdioProcessLauncher{Env: env}, nil
}

func validateHostProcessEnvironmentDirectory(directory string) error {
	if !filepath.IsAbs(directory) || strings.ContainsRune(directory, os.PathListSeparator) ||
		strings.IndexByte(directory, 0) >= 0 {
		return fmt.Errorf("process environment profile requires an absolute, single directory")
	}
	return nil
}

func isReservedHostProcessEnvironmentKey(key string) bool {
	for _, reserved := range []string{"PATH", "HOME", "TMPDIR", "TEMP", "TMP", "USERPROFILE"} {
		if runtime.GOOS == "windows" {
			if strings.EqualFold(key, reserved) {
				return true
			}
		} else if key == reserved {
			return true
		}
	}
	return false
}
