package main

import (
	"flag"
	"fmt"
	"os"

	"tei/internal/plugin/implementation"
	"tei/internal/syntax"
)

func main() {
	implementationPath := flag.String("implementation", "", "path to plugin implementation.tei")
	pluginPath := flag.String("plugin", "", "path to artifact-free plugin.tei")
	outputPath := flag.String("out", "", "path for deterministic realization artifact")
	flag.Parse()

	if err := run(*implementationPath, *pluginPath, *outputPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(implementationPath, pluginPath, outputPath string) error {
	if implementationPath == "" || pluginPath == "" || outputPath == "" {
		return fmt.Errorf("implementation, plugin, and out are required")
	}

	implementationSource, err := os.ReadFile(implementationPath)
	if err != nil {
		return fmt.Errorf("read implementation source: %w", err)
	}
	source, err := syntax.NewParser(string(implementationSource)).ParsePluginImplementation()
	if err != nil {
		return fmt.Errorf("parse implementation source: %w", err)
	}

	pluginSource, err := os.ReadFile(pluginPath)
	if err != nil {
		return fmt.Errorf("read Plugin definition: %w", err)
	}
	definition, err := syntax.NewParser(string(pluginSource)).ParsePlugin()
	if err != nil {
		return fmt.Errorf("parse Plugin definition: %w", err)
	}
	if definition.Artifact.Digest != "" || definition.Artifact.SignatureRef != "" || definition.Artifact.Provenance != "" {
		return fmt.Errorf("authoritative Plugin definition must be artifact-free")
	}
	if err := implementation.ValidateAgainstDefinition(source, definition); err != nil {
		return err
	}

	artifact, _, err := implementation.Materialize(source)
	if err != nil {
		return err
	}
	if err := os.WriteFile(outputPath, artifact, 0o644); err != nil {
		return fmt.Errorf("write realization artifact: %w", err)
	}
	return nil
}
