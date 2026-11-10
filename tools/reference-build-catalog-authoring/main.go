// Command reference-build-catalog-authoring turns a capability declaration into
// the build catalog that `tei self-build -build-catalog` consumes.
//
// SB-C0 §4.32 decided the catalog is data outside the product: no Workspace key
// and no plugin syntax is added, because both parsers refuse unknown input and so
// a new key would make older binaries reject the file. That decision leaves one
// gap — something has to author the file. This command is that something, and it
// stays outside the product's contracts.
//
// The declaration says what a build tool can do; the artifact digests are not
// typed by hand. The tool reads each named artifact and computes its digest, so a
// catalog cannot claim a builder the repository does not have.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"tei/internal/buildintent"
	"tei/internal/core"
)

// Declaration is the maintainer-facing input. It is deliberately not a product
// format: it has no formatVersion promise and no compatibility surface. Only the
// catalog it produces is data a caller names and digests.
type Declaration struct {
	Providers    []DeclaredProvider   `json:"providers"`
	Dependencies []DeclaredDependency `json:"dependencies"`
	Targets      []DeclaredTarget     `json:"targets"`
}

type DeclaredProvider struct {
	Plugin         string   `json:"plugin"`
	Version        string   `json:"version"`
	Implementation string   `json:"implementation"`
	Contract       string   `json:"contract"`
	ContractMajor  int      `json:"contractMajor"`
	RecipeKinds    []string `json:"recipeKinds"`
	Targets        []string `json:"targets"`
	Profiles       []string `json:"profiles"`
	// Artifact is a path, relative to -root, to the builder executable this
	// provider names. The digest in the catalog is computed from this file.
	Artifact string `json:"artifact"`
	// ArtifactDigest is refused: a hand-typed digest would let the catalog claim a
	// builder whose bytes nobody read. It exists only so a caller who wrote one by
	// habit gets a named error instead of silence.
	ArtifactDigest string `json:"artifactDigest,omitempty"`
}

type DeclaredDependency struct {
	Kind     string `json:"kind"`
	ID       string `json:"id"`
	Version  string `json:"version"`
	Artifact string `json:"artifact"`
}

type DeclaredTarget struct {
	Target   string   `json:"target"`
	Profiles []string `json:"profiles"`
}

// Result is what the command prints. The digest is over the exact bytes written,
// which is the value the caller passes as -build-catalog-digest.
type Result struct {
	CatalogPath   string `json:"catalogPath"`
	CatalogDigest string `json:"catalogDigest"`
	Providers     int    `json:"providers"`
	Dependencies  int    `json:"dependencies"`
	Targets       int    `json:"targets"`
}

func main() {
	if code := run(os.Args[1:], os.Stdout, os.Stderr); code != 0 {
		os.Exit(code)
	}
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("reference-build-catalog-authoring", flag.ContinueOnError)
	fs.SetOutput(stderr)
	declarationPath := fs.String("declaration", "", "capability declaration JSON (required)")
	root := fs.String("root", ".", "root that relative artifact paths resolve from")
	outputPath := fs.String("output", "", "where to write the catalog JSON (required; never overwritten)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *declarationPath == "" || *outputPath == "" {
		fmt.Fprintln(stderr, "-declaration and -output are required")
		return 2
	}

	result, err := author(*declarationPath, *root, *outputPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func author(declarationPath, root, outputPath string) (Result, error) {
	raw, err := os.ReadFile(declarationPath)
	if err != nil {
		return Result{}, fmt.Errorf("declaration: %w", err)
	}
	var declaration Declaration
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&declaration); err != nil {
		return Result{}, fmt.Errorf("declaration: %w", err)
	}

	catalog := buildintent.BuildCatalog{FormatVersion: 1}
	for i, provider := range declaration.Providers {
		if provider.ArtifactDigest != "" {
			return Result{}, fmt.Errorf("provider %d (%s): artifactDigest must not be declared; the tool computes it from the artifact file so a catalog cannot claim bytes nobody read", i, provider.Plugin)
		}
		digest, err := digestArtifact(root, provider.Artifact)
		if err != nil {
			return Result{}, fmt.Errorf("provider %d (%s): %w", i, provider.Plugin, err)
		}
		id, err := canonicalID(provider.Plugin)
		if err != nil {
			return Result{}, fmt.Errorf("provider %d: %w", i, err)
		}
		catalog.Providers = append(catalog.Providers, buildintent.BuildProvider{
			Plugin:         id,
			Version:        provider.Version,
			Implementation: provider.Implementation,
			Contract:       provider.Contract,
			ContractMajor:  provider.ContractMajor,
			AuthorityScope: buildintent.BuilderAuthorityScope,
			ArtifactDigest: digest,
			RecipeKinds:    provider.RecipeKinds,
			Targets:        provider.Targets,
			Profiles:       provider.Profiles,
		})
	}
	for i, dependency := range declaration.Dependencies {
		digest, err := digestArtifact(root, dependency.Artifact)
		if err != nil {
			return Result{}, fmt.Errorf("dependency %d (%s): %w", i, dependency.ID, err)
		}
		catalog.Dependencies = append(catalog.Dependencies, buildintent.BuildDependencyItem{
			Kind:           dependency.Kind,
			ID:             dependency.ID,
			Version:        dependency.Version,
			ArtifactDigest: digest,
		})
	}
	for _, target := range declaration.Targets {
		catalog.Targets = append(catalog.Targets, buildintent.BuildTargetEntry{
			Target:   target.Target,
			Profiles: target.Profiles,
		})
	}

	// The catalog is written only if it reads back: the entry parses it strictly,
	// so a catalog this tool cannot re-read would be refused later, at the point
	// where the caller believes it has a catalog.
	encoded, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return Result{}, err
	}
	encoded = append(encoded, '\n')
	if _, err := buildintent.ParseBuildCatalog(encoded); err != nil {
		return Result{}, fmt.Errorf("the assembled catalog does not parse: %w", err)
	}

	if _, err := os.Stat(outputPath); err == nil {
		return Result{}, fmt.Errorf("output already exists; this tool does not overwrite: %s", outputPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Result{}, err
	}
	if err := os.WriteFile(outputPath, encoded, 0o600); err != nil {
		return Result{}, err
	}

	sum := sha256.Sum256(encoded)
	return Result{
		CatalogPath:   outputPath,
		CatalogDigest: "sha256:" + hex.EncodeToString(sum[:]),
		Providers:     len(catalog.Providers),
		Dependencies:  len(catalog.Dependencies),
		Targets:       len(catalog.Targets),
	}, nil
}

// digestArtifact reads the artifact and returns its digest. It refuses a path that
// leaves the root, because a catalog assembled from a file the root does not own
// would describe a builder the caller did not hand over.
func digestArtifact(root, artifact string) (string, error) {
	if strings.TrimSpace(artifact) == "" {
		return "", errors.New("artifact is required")
	}
	if filepath.IsAbs(artifact) {
		return "", fmt.Errorf("artifact %q must be relative to -root", artifact)
	}
	cleaned := filepath.Clean(artifact)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("artifact %q leaves -root", artifact)
	}
	info, err := os.Stat(filepath.Join(root, cleaned))
	if err != nil {
		return "", fmt.Errorf("artifact %q: %w", artifact, err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("artifact %q is not a regular file", artifact)
	}
	file, err := os.Open(filepath.Join(root, cleaned))
	if err != nil {
		return "", err
	}
	defer file.Close()
	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(sum.Sum(nil)), nil
}

func canonicalID(value string) (core.CanonicalID, error) {
	parsed, err := core.ParseCanonicalID(value)
	if err != nil {
		return "", err
	}
	return parsed, nil
}
