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
	semanticLockOut := flags.String("semantic-lock-out", "", "write resolved semantic lock JSON")
	verifySemanticLockPath := flags.String("verify-semantic-lock", "", "replay an existing semantic lock against exact available declarations")
	verifySemanticLockEvidence := flags.Bool("verify-semantic-lock-evidence", false, "also require accepted Conformance and namespace Authority Evidence bindings")
	var roots requirementFlags
	var wiringPaths pathFlags
	var conformancePaths pathFlags
	var namespaceAuthorityPaths pathFlags
	flags.Var(&roots, "require", "root extension requirement EXTENSION@MAJOR; repeatable")
	flags.Var(&wiringPaths, "interface-wiring", "semantic interface wiring JSON file; repeatable")
	flags.Var(&conformancePaths, "interface-conformance", "accepted semantic interface conformance JSON file; repeatable")
	flags.Var(&namespaceAuthorityPaths, "namespace-authority", "accepted semantic namespace authority JSON file; repeatable")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *verifySemanticLockEvidence && strings.TrimSpace(*verifySemanticLockPath) == "" {
		fmt.Fprintln(os.Stderr, "-verify-semantic-lock-evidence requires -verify-semantic-lock FILE")
		return 2
	}
	if strings.TrimSpace(*workspacePath) == "" || strings.TrimSpace(*extensionsPath) == "" ||
		(strings.TrimSpace(*verifySemanticLockPath) == "" && len(roots) == 0) {
		fmt.Fprintln(os.Stderr, "usage: reference-semantic-extension-catalog -workspace DIR -extensions DIR (-require EXTENSION@MAJOR ... | -verify-semantic-lock FILE) [-interface-wiring FILE] [-interface-conformance FILE] [-namespace-authority FILE] [-semantic-lock-out FILE] [-verify-semantic-lock-evidence] [-subject-kind KIND] [-subject-id ID]")
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
	if strings.TrimSpace(*verifySemanticLockPath) != "" {
		var lock SemanticLock
		if err := decodeStrictJSONFile(*verifySemanticLockPath, &lock); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if err := verifySemanticLockReplay(lock, coreCatalogFromSourceIndex(index), descriptors); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		result := map[string]any{
			"semanticReplayValid":             true,
			"semanticSetFingerprint":          lock.SemanticSetFingerprint,
			"resolutionPolicy":                lock.ResolutionPolicy,
			"acceptedEvidenceBindingsChecked": false,
		}
		if *verifySemanticLockEvidence {
			conformances, err := loadSemanticInterfaceConformances(conformancePaths)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			namespaceAuthorities, err := loadSemanticNamespaceAuthorities(namespaceAuthorityPaths)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			if err := verifySemanticLockAcceptedEvidenceBindings(
				lock,
				conformances,
				namespaceAuthorities,
			); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			result["acceptedEvidenceBindingsChecked"] = true
			result["acceptedEvidenceBindingsValid"] = true
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
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
	namespaceAuthorities, err := loadSemanticNamespaceAuthorities(namespaceAuthorityPaths)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	catalog, err := resolveSemanticCatalogWithAllInputs(
		coreCatalogFromSourceIndex(index),
		descriptors,
		roots,
		wirings,
		conformances,
		namespaceAuthorities,
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if strings.TrimSpace(*semanticLockOut) != "" {
		lock, err := buildSemanticLock(catalog)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if err := writeJSONFile(*semanticLockOut, lock); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
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

func loadSemanticNamespaceAuthorities(paths []string) ([]SemanticNamespaceAuthority, error) {
	out := make([]SemanticNamespaceAuthority, 0, len(paths))
	for _, path := range paths {
		var value SemanticNamespaceAuthority
		if err := decodeStrictJSONFile(path, &value); err != nil {
			return nil, fmt.Errorf("namespace authority %s: %w", path, err)
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


func writeJSONFile(path string, value any) error {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	return os.WriteFile(path, encoded, 0o644)
}
