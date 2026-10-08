package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"tei/internal/plugin"
	"tei/internal/syntax"
)

func main() {
	pluginPath := flag.String("plugin", "", "path to artifact-free plugin.tei")
	planPath := flag.String("plan", "", "path to resolved BuildPlan JSON")
	outputPath := flag.String("out", "", "path for deterministic Plugin package")
	flag.Parse()

	if err := run(*pluginPath, *planPath, *outputPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(pluginPath, planPath, outputPath string) error {
	if pluginPath == "" || planPath == "" || outputPath == "" {
		return fmt.Errorf("plugin, plan, and out are required")
	}
	pluginSource, err := os.ReadFile(pluginPath)
	if err != nil {
		return fmt.Errorf("read Plugin definition: %w", err)
	}
	definition, err := syntax.NewParser(string(pluginSource)).ParsePlugin()
	if err != nil {
		return fmt.Errorf("parse Plugin definition: %w", err)
	}

	planSource, err := os.ReadFile(planPath)
	if err != nil {
		return fmt.Errorf("read Build Plan: %w", err)
	}
	var plan plugin.BuildPlan
	decoder := json.NewDecoder(bytes.NewReader(planSource))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&plan); err != nil {
		return fmt.Errorf("decode Build Plan: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("Build Plan contains multiple values")
		}
		return fmt.Errorf("decode trailing Build Plan content: %w", err)
	}

	artifact, err := plugin.MaterializePluginPackage(definition, plan)
	if err != nil {
		return err
	}
	if err := os.WriteFile(outputPath, artifact, 0o644); err != nil {
		return fmt.Errorf("write Plugin package: %w", err)
	}
	return nil
}
