package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"tei/tools/internal/gobuilder"
)

// supplyEnvironment derives the resolver environment from the same
// deny-by-default construction the Go build adapter uses, then points GOPROXY
// at the named supply. It deliberately does not inherit the Host module cache:
// a verification that could fall back to the Host cache would report success
// for a supply that was never used.
func supplyEnvironment(base []string, stateDir, supply string) ([]string, error) {
	host, err := gobuilder.AllowedHostEnvironment(base, runtime.GOOS)
	if err != nil {
		return nil, err
	}
	settings := gobuilder.FixedSettings(runtime.GOOS, runtime.GOARCH)
	out := append([]string(nil), host...)
	for _, setting := range settings {
		key, _, _ := strings.Cut(setting, "=")
		switch key {
		case "GOPROXY":
			// replaced below with the named supply
			continue
		case "GOFLAGS":
			// go list must be able to read the repository's -mod=readonly
			// intent from the invocation, not from ambient Host settings.
			continue
		}
		out = append(out, setting)
	}
	supplyURL := fileURL(supply)
	out = append(out, "GOPROXY="+supplyURL, "GOFLAGS=")
	if stateDir != "" {
		if err := os.MkdirAll(stateDir, 0o755); err != nil {
			return nil, fmt.Errorf("create isolated verification state: %w", err)
		}
		for _, dir := range []string{
			filepath.Join(stateDir, "home"),
			filepath.Join(stateDir, "tmp"),
		} {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("create isolated verification directory: %w", err)
			}
		}
		out = append(out,
			"GOCACHE="+filepath.Join(stateDir, "cache"),
			"GOMODCACHE="+filepath.Join(stateDir, "modcache"),
			"GOPATH="+filepath.Join(stateDir, "gopath"),
			"HOME="+filepath.Join(stateDir, "home"),
			"USERPROFILE="+filepath.Join(stateDir, "home"),
			"TMPDIR="+filepath.Join(stateDir, "tmp"),
			"TMP="+filepath.Join(stateDir, "tmp"),
			"TEMP="+filepath.Join(stateDir, "tmp"),
		)
	}
	sort.Strings(out)
	return out, nil
}

// fileURL renders one local path as an absolute file:// proxy URL. Windows
// needs the leading slash before the volume so the URL keeps the volume in the
// path component.
func fileURL(path string) string {
	clean := filepath.ToSlash(filepath.Clean(path))
	if runtime.GOOS == "windows" {
		return "file:///" + strings.TrimPrefix(clean, "/")
	}
	return "file://" + clean
}
