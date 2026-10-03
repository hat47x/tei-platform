package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"tei/internal/workspace"
)

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
	flags.Var(&roots, "require", "root extension requirement EXTENSION@MAJOR; repeatable")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*workspacePath) == "" || strings.TrimSpace(*extensionsPath) == "" || len(roots) == 0 {
		fmt.Fprintln(os.Stderr, "usage: reference-semantic-extension-catalog -workspace DIR -extensions DIR -require EXTENSION@MAJOR [-require ...] [-subject-kind KIND] [-subject-id ID]")
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
	catalog, err := resolveSemanticCatalog(coreCatalogFromSourceIndex(index), descriptors, roots)
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
