package appgen

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// BuildResult holds build artifact metadata including performance timing.
type BuildResult struct {
	Digest        string
	BinaryPath    string
	SourceDir     string
	Platform      string
	BuildDuration time.Duration
	BinarySize    int64
}

// Build compiles the generated source tree into a single binary.
func Build(workspaceRoot, appID, sourceDir, outputPath, platform string) (*BuildResult, error) {
	if err := writeGoMod(sourceDir, appID); err != nil {
		return nil, fmt.Errorf("write go.mod: %w", err)
	}

	buildStart := time.Now()

	goCache := filepath.Join(sourceDir, ".gocache")
	goTmp := filepath.Join(sourceDir, ".gotmp")
	for _, dir := range []string{goCache, goTmp} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}

	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = sourceDir
	tidy.Env = append(minimalEnv(), "GOCACHE="+goCache, "GOTMPDIR="+goTmp)
	if out, err := tidy.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("go mod tidy failed: %v\n%s", err, out)
	}

	env := append(minimalEnv(), "GOCACHE="+goCache, "GOTMPDIR="+goTmp)
	if platform != "" {
		parts := strings.Split(platform, "/")
		if len(parts) == 2 {
			env = append(env, "GOOS="+parts[0], "GOARCH="+parts[1])
		}
	}

	cmd := exec.Command("go", "build", "-trimpath", "-buildvcs=false", "-ldflags=-s -w", "-o", outputPath, ".")
	cmd.Dir = sourceDir
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("go build failed: %v\n%s", err, out)
	}
	buildDuration := time.Since(buildStart)

	info, err := os.Stat(outputPath)
	if err != nil {
		return nil, fmt.Errorf("stat binary: %w", err)
	}

	digest, err := fileDigest(outputPath)
	if err != nil {
		return nil, fmt.Errorf("artifact digest: %w", err)
	}
	return &BuildResult{
		Digest:        "sha256:" + digest,
		BinaryPath:    outputPath,
		SourceDir:     sourceDir,
		Platform:      platform,
		BuildDuration: buildDuration,
		BinarySize:    info.Size(),
	}, nil
}

func writeGoMod(moduleDir, appID string) error {
	path := filepath.Join(moduleDir, "go.mod")
	body := "module app/" + appID + "\n\ngo 1.26.10\n\nrequire (\n\tmodernc.org/sqlite v1.34.5\n)\n"
	return os.WriteFile(path, []byte(body), 0o644)
}

func minimalEnv() []string {
	env := os.Environ()
	filtered := make([]string, 0, len(env))
	for _, kv := range env {
		switch {
		case kv == "":
		case len(kv) >= 8 && (kv[:8] == "TEI_DSN=" || kv[:8] == "TEI_APP=" || (len(kv) >= 11 && kv[:11] == "TEI_ENVIRON")):
			filtered = append(filtered, kv)
		case len(kv) >= 4 && (kv[:4] == "GOOS=" || kv[:4] == "GOAR" || kv[:7] == "GOCACHE"):
			filtered = append(filtered, kv)
		case len(kv) >= 3 && kv[:3] == "CC=":
			filtered = append(filtered, kv)
		case len(kv) >= 4 && (kv[:4] == "CC_" || kv[:5] == "CGO_"):
			filtered = append(filtered, kv)
		default:
			filtered = append(filtered, kv)
		}
	}
	return append(filtered, "GO111MODULE=on")
}

func fileDigest(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
