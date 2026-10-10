// Command tei-example-slide renders a checked illustrative case without
// modifying the authoritative TEI workspace.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"tei/internal/ir"
	example "tei/tools/reference-example-slide"
)

func main() {
	inputFile := flag.String("input", "", "Case/diagram/presentation JSON input")
	worldFile := flag.String("world", "", "optional normalized TEI World JSON")
	catalogOnly := flag.Bool("catalog", false, "print the read-only Catalog derived from -world")
	outDir := flag.String("out", "", "optional destination for slides as SVG and projection.json")
	flag.Parse()
	if err := run(*inputFile, *worldFile, *catalogOnly, *outDir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(inputPath, worldPath string, catalogOnly bool, outputPath string) error {
	var input example.Input
	if catalogOnly && (worldPath == "" || inputPath != "" || outputPath != "") {
		return fmt.Errorf("-catalog requires -world and excludes -input/-out")
	}
	if !catalogOnly {
		if inputPath == "" {
			return fmt.Errorf("-input is required unless -catalog is used")
		}
		if err := readJSON(inputPath, &input); err != nil {
			return fmt.Errorf("read input: %w", err)
		}
	}
	if worldPath != "" {
		var world ir.World
		if err := readJSON(worldPath, &world); err != nil {
			return fmt.Errorf("read World: %w", err)
		}
		derived, err := example.CatalogFromWorld(world)
		if err != nil {
			return err
		}
		if catalogOnly {
			return printJSON(derived)
		}
		// Do not silently rewrite a stale Example's accepted revision.
		input.Catalog = derived
	}
	projection, err := example.Build(input)
	if err != nil {
		return err
	}
	// A Review Packet is derived from the same immutable source Case.
	// It contains no accepted decisions or Semantic Patch.
	review, err := example.Review(input)
	if err != nil {
		return fmt.Errorf("derive review: %w", err)
	}
	if outputPath == "" {
		return printJSON(projection)
	}
	// Local reference output is explicit. Restrictive permissions protect
	// against accidental casual exposure of illustrative record values.
	if err := os.MkdirAll(outputPath, 0700); err != nil {
		return err
	}
	content, err := json.MarshalIndent(projection, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')
	if err := os.WriteFile(filepath.Join(outputPath, "projection.json"), content, 0600); err != nil {
		return err
	}
	reviewData, err := json.MarshalIndent(review, "", "  ")
	if err != nil {
		return err
	}
	reviewData = append(reviewData, '\n')
	if err := os.WriteFile(filepath.Join(outputPath, "review.json"), reviewData, 0600); err != nil {
		return err
	}
	for i, slide := range projection.Slides {
		file := filepath.Join(outputPath, fmt.Sprintf("slide-%03d.svg", i+1))
		if err := os.WriteFile(file, []byte(example.SVG(slide)), 0600); err != nil {
			return err
		}
	}
	return nil
}

func readJSON(path string, value any) error {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, value)
}

func printJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
