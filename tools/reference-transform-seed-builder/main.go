package main

import (
	"flag"
	"fmt"
	"os"

	"tei/internal/plugin/transform"
)

func main() {
	sourcePath := flag.String("source", "", "path to deterministic transform Source JSON")
	outputPath := flag.String("out", "", "path for canonical transform builder realization")
	flag.Parse()

	if err := run(*sourcePath, *outputPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(sourcePath, outputPath string) error {
	if sourcePath == "" || outputPath == "" {
		return fmt.Errorf("source and out are required")
	}
	raw, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("read transform Source: %w", err)
	}
	source, err := transform.ParseSourceJSON(raw)
	if err != nil {
		return err
	}
	artifact, _, err := transform.MaterializeSource(source)
	if err != nil {
		return err
	}
	if err := os.WriteFile(outputPath, artifact, 0o644); err != nil {
		return fmt.Errorf("write transform builder realization: %w", err)
	}
	return nil
}
