package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"tei/tools/internal/gobuilder"
)

// preloadEnvironment and supplyEnvironment point GOPROXY at the named supply
// and keep the module cache isolated. Neither falls back to the Host cache: an
// attempt that could silently read the Host cache would not show that the supply
// alone is sufficient.
func supplyEnvironment(base []string, stateDir, supply string) ([]string, error) {
	return preloadEnvironment(base, stateDir, supply)
}

func preloadEnvironment(base []string, stateDir, supply string) ([]string, error) {
	host, err := gobuilder.AllowedHostEnvironment(base, runtime.GOOS)
	if err != nil {
		return nil, err
	}
	out := append([]string(nil), host...)
	for _, setting := range gobuilder.FixedSettings(runtime.GOOS, runtime.GOARCH) {
		key, _, _ := strings.Cut(setting, "=")
		if key == "GOPROXY" || key == "GOFLAGS" {
			continue
		}
		out = append(out, setting)
	}
	out = append(out, "GOPROXY="+fileURL(supply), "GOFLAGS=")
	if stateDir != "" {
		if err := os.MkdirAll(stateDir, 0o755); err != nil {
			return nil, fmt.Errorf("create isolated build state: %w", err)
		}
		for _, dir := range []string{filepath.Join(stateDir, "home"), filepath.Join(stateDir, "tmp")} {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("create isolated build directory: %w", err)
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

func fileURL(path string) string {
	clean := filepath.ToSlash(filepath.Clean(path))
	if runtime.GOOS == "windows" {
		return "file:///" + strings.TrimPrefix(clean, "/")
	}
	return "file://" + clean
}

func fileDigest(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// resolveGo turns a toolchain argument into one exact executable. A bare name is
// a PATH lookup; anything with a path separator is taken verbatim and validated,
// so an explicitly pinned toolchain is never substituted.
func resolveGo(argument string) (string, error) {
	trimmed := strings.TrimSpace(argument)
	if trimmed == "" {
		trimmed = "go"
	}
	if !strings.Contains(trimmed, "/") && !strings.Contains(trimmed, string(filepath.Separator)) {
		path, err := exec.LookPath(trimmed)
		if err != nil {
			return "", fmt.Errorf("resolve Go executable: %w", err)
		}
		trimmed = path
	}
	absolute, err := filepath.Abs(trimmed)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return "", fmt.Errorf("resolve Go executable: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("Go executable is not a regular file: %s", absolute)
	}
	return absolute, nil
}
