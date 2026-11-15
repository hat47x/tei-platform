package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"tei/tools/reference-application-binary/internal/browsegen"
)

// runBrowse generates and builds one browse Function from the Source. It
// never falls back to the legacy fixed-model generator.
func runBrowse(workspace, application, function, target, out, binary string) error {
	plan, err := browsegen.NewPlan(workspace, application, function, target)
	if err != nil {
		return err
	}
	if out == "" {
		return fmt.Errorf("-out is required with -function")
	}
	if _, err := browsegen.Generate(plan, out); err != nil {
		return fmt.Errorf("generate: %w", err)
	}
	if binary == "" {
		binary = filepath.Join(out, "bin", "app")
		if runtime.GOOS == "windows" {
			binary += ".exe"
		}
	}
	rc, err := browsegen.Build(out, binary)
	if err != nil {
		return fmt.Errorf("build: %w", err)
	}
	b, _ := json.MarshalIndent(rc, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(b, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}
