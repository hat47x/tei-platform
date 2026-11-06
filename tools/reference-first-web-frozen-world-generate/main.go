// Diagnostic build-only exporter for a normalized, parser-independent
// World. The output is derived from the existing TEI Source loader, not a
// second Canonical authority. The builder pins its exact digest separately.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"tei/internal/workspace"
)

func main() {
	source := flag.String("workspace", "reference/workspace", "TEI source Workspace")
	application := flag.String("application", "", "exact Application")
	baseline := flag.String("baseline-id", "", "Handover B")
	revision := flag.String("baseline-revision", "", "Handover R")
	target := flag.String("target-revision", "", "build T")
	lineageOut := flag.String("lineage-out", "", "optional build-only derived World/digest observation path")
	grantsPath := flag.String("grants", "", "optional JSON grant file; when set, the observation path must be covered by it")
	declarationPath := flag.String("declaration", "", "optional grants file the environment declared; when set, the grants must stay inside it")
	accessLog := flag.String("access-log", "", "optional JSON Lines file recording whether the write was allowed")
	flag.Parse()
	if flag.NArg() != 0 || *application != "order_entry" || *baseline == "" ||
		*revision == "" || *target == "" {
		fmt.Fprintln(os.Stderr, "explicit order_entry, B, R and T are required")
		os.Exit(2)
	}
	loaded, err := workspace.Load(*source)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	snapshot := struct {
		Schema                     string `json:"schema"`
		ApplicationID              string `json:"applicationId"`
		HandoverBaselineID         string `json:"handoverBaselineId"`
		BaselineRepositoryRevision string `json:"baselineRepositoryRevision"`
		TargetBuildRevision        string `json:"targetBuildRevision"`
		World                      any    `json:"world"`
	}{
		Schema:                     "tei.reference.first-web-frozen-world/v1",
		ApplicationID:              *application,
		HandoverBaselineID:         *baseline,
		BaselineRepositoryRevision: *revision,
		TargetBuildRevision:        *target,
		World:                      loaded.World,
	}
	var frozen bytes.Buffer
	encoder := json.NewEncoder(&frozen)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(snapshot); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *lineageOut != "" {
		// Shared Acceptance's worldSHA256 uses encoding/json with two-space
		// indentation and the default HTML escaping. Match that exact byte
		// convention, without redefining its Source/Decision authority.
		derived, err := json.MarshalIndent(loaded.World, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		derived = append(derived, '\n')
		worldSum := sha256.Sum256(derived)
		frozenSum := sha256.Sum256(frozen.Bytes())
		lineage := struct {
			DerivedWorldSHA256 string `json:"derivedWorldSha256"`
			FrozenWorldSHA256  string `json:"frozenWorldSha256"`
		}{
			DerivedWorldSHA256: "sha256:" + hex.EncodeToString(worldSum[:]),
			FrozenWorldSHA256:  hex.EncodeToString(frozenSum[:]),
		}
		output, err := json.Marshal(lineage)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if *grantsPath != "" {
			grants, err := loadGrants(*grantsPath, *declarationPath)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			if err := writeLineageWithGrants(*lineageOut, append(output, '\n'), grants, *accessLog); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		} else {
			fmt.Fprintln(os.Stderr, "観測を書くには -grants で付与を渡してください（付与の無い書出しは行いません）")
			os.Exit(1)
		}
	}
	if _, err := os.Stdout.Write(frozen.Bytes()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
