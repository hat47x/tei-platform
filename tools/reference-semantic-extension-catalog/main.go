package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"tei/internal/workspace"
)

type pathFlags []string

func (p *pathFlags) String() string { return strings.Join(*p, ",") }
func (p *pathFlags) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("path must not be empty")
	}
	*p = append(*p, value)
	return nil
}

type requirementFlags []ExtensionRequirement

func (r *requirementFlags) String() string {
	var values []string
	for _, item := range *r {
		values = append(values, fmt.Sprintf("%s@%d", item.Extension, item.CompatibleMajor))
	}
	return strings.Join(values, ",")
}

func (r *requirementFlags) Set(value string) error {
	value = strings.TrimSpace(value)
	at := strings.LastIndex(value, "@")
	if at <= 0 || at == len(value)-1 {
		return fmt.Errorf("requirement must be EXTENSION@MAJOR")
	}
	major, err := strconv.Atoi(value[at+1:])
	if err != nil || major < 0 {
		return fmt.Errorf("requirement major must be a non-negative integer")
	}
	*r = append(*r, ExtensionRequirement{
		Extension:       strings.TrimSpace(value[:at]),
		CompatibleMajor: major,
	})
	return nil
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	flags := flag.NewFlagSet("reference-semantic-extension-catalog", flag.ContinueOnError)
	workspacePath := flags.String("workspace", "", "TEI workspace directory")
	extensionsPath := flags.String("extensions", "", "reference semantic extension descriptor directory")
	subjectKind := flags.String("subject-kind", "", "optional subject kind filter")
	subjectID := flags.String("subject-id", "", "optional subject ID filter")
	var roots requirementFlags
	var wiringPaths pathFlags
	var conformancePaths pathFlags
	flags.Var(&roots, "require", "root extension requirement EXTENSION@MAJOR; repeatable")
	flags.Var(&wiringPaths, "interface-wiring", "semantic interface wiring JSON file; repeatable")
	flags.Var(&conformancePaths, "interface-conformance", "accepted semantic interface conformance JSON file; repeatable")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*workspacePath) == "" || strings.TrimSpace(*extensionsPath) == "" || len(roots) == 0 {
		fmt.Fprintln(os.Stderr, "usage: reference-semantic-extension-catalog -workspace DIR -extensions DIR -require EXTENSION@MAJOR [-require ...] [-interface-wiring FILE] [-interface-conformance FILE] [-subject-kind KIND] [-subject-id ID]")
		return 2
	}

	loaded, err := workspace.Load(*workspacePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	index, err := workspace.BuildSourceIndex(*workspacePath, loaded)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	descriptors, err := loadExtensionDescriptors(*extensionsPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	wirings, err := loadSemanticInterfaceWirings(wiringPaths)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	conformances, err := loadSemanticInterfaceConformances(conformancePaths)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	catalog, err := resolveSemanticCatalogWithInterfaceInputs(
		coreCatalogFromSourceIndex(index),
		descriptors,
		roots,
		wirings,
		conformances,
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	catalog = filterCatalog(catalog, *subjectKind, *subjectID)

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(catalog); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if len(catalog.Unresolved) > 0 {
		return 1
	}
	return 0
}


func loadSemanticInterfaceWirings(paths []string) ([]SemanticInterfaceWiring, error) {
	out := make([]SemanticInterfaceWiring, 0, len(paths))
	for _, path := range paths {
		var value SemanticInterfaceWiring
		if err := decodeStrictJSONFile(path, &value); err != nil {
			return nil, fmt.Errorf("interface wiring %s: %w", path, err)
		}
		out = append(out, value)
	}
	return out, nil
}

func loadSemanticInterfaceConformances(paths []string) ([]SemanticInterfaceConformance, error) {
	out := make([]SemanticInterfaceConformance, 0, len(paths))
	for _, path := range paths {
		var value SemanticInterfaceConformance
		if err := decodeStrictJSONFile(path, &value); err != nil {
			return nil, fmt.Errorf("interface conformance %s: %w", path, err)
		}
		out = append(out, value)
	}
	return out, nil
}

func decodeStrictJSONFile(path string, dst any) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("JSONの後ろに余分な内容があります")
		}
		return fmt.Errorf("JSON末尾: %w", err)
	}
	return nil
}
