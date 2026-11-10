package appgen

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Build compiles the generated source tree into a single binary.
func Build(workspaceRoot, appID, sourceDir, outputPath, platform string) (string, error) {
	if err := writeGoMod(sourceDir, appID); err != nil {
		return "", fmt.Errorf("write go.mod: %w", err)
	}
	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = sourceDir
	tidy.Env = minimalEnv()
	if out, err := tidy.CombinedOutput(); err != nil {
		return "", fmt.Errorf("go mod tidy failed: %v\n%s", err, out)
	}

	env := minimalEnv()
	if platform != "" {
		parts := strings.Split(platform, "/")
		if len(parts) == 2 {
			env = append(env, "GOOS="+parts[0], "GOARCH="+parts[1])
		}
	}

	cmd := exec.Command("go", "build", "-trimpath", "-buildvcs=false", "-o", outputPath, ".")
	cmd.Dir = sourceDir
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("go build failed: %v\n%s", err, out)
	}

	digest, err := fileDigest(outputPath)
	if err != nil {
		return "", fmt.Errorf("artifact digest: %w", err)
	}
	return "sha256:" + digest, nil
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
