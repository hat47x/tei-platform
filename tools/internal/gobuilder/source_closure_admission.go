package gobuilder

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// This file implements the staged opt-in described in
// plan/design/EXTERNAL_GO_MODULE_ADMISSION_CONTRACT_2026-11.md (option C).
//
// The default policy does not change: ResolveSourceClosure still refuses every
// external module root. A caller that explicitly asks for external members
// gets them through ResolveQualifiedSourceClosure below, and the returned
// admission record states which policy produced the closure and on what basis
// the module bytes were accepted.

const (
	// ClosurePolicyRepositoryOnly is the existing behaviour: external module
	// roots are refused, so the closure covers the target module only.
	ClosurePolicyRepositoryOnly = "repository-only"

	// ClosurePolicyExternalModuleOptIn admits selected external module members
	// only when the caller names a supply directory. The Go toolchain itself
	// refuses module bytes that disagree with the committed go.sum, so the
	// admission relies on that check and records the reliance instead of
	// re-verifying the same bytes.
	ClosurePolicyExternalModuleOptIn = "external-module-opt-in"

	// externalMemberPrefix namespaces external members inside the closure.
	// Repository members are repository-relative and must never use it.
	externalMemberPrefix = "module:"

	closureAdmissionSchema = "tei.reference.go-closure-admission/v0"
)

// ClosureAdmission records how one closure was produced. It is evidence about
// the closure decision, not an acceptance and not an authority grant.
type ClosureAdmission struct {
	SchemaVersion      string   `json:"schemaVersion"`
	Policy             string   `json:"policy"`
	Package            string   `json:"package"`
	SupplyRoot         string   `json:"supplyRoot,omitempty"`
	RepositoryMembers  int      `json:"repositoryMembers"`
	ExternalMembers    int      `json:"externalMembers"`
	ExternalModules    []string `json:"externalModules,omitempty"`
	GoSumBinding       string   `json:"goSumBinding"`
	Qualified          bool     `json:"qualified"`
	QualificationBasis string   `json:"qualificationBasis"`
	RefusedReason      string   `json:"refusedReason,omitempty"`
	ClosureDigest      string   `json:"closureDigest,omitempty"`
	Boundary           string   `json:"boundary"`
}

const admissionBoundary = "closure membership and digest only; no supply origin, publisher, signature, build authority or runtime acceptance"

// goSumBindingText states exactly what binds the admitted module bytes. It
// names the Go toolchain's own check rather than claiming that this resolver
// verified the bytes again.
const goSumBindingText = "the Go toolchain refuses module bytes that disagree with the committed go.sum; this resolver records that reliance and does not re-verify the same bytes (see tools/reference-go-module-supply-verify for the independent check)"

// ResolveQualifiedSourceClosure resolves a source closure and admits selected
// external module members when supply is non-empty. An empty supply keeps the
// existing repository-only policy and returns a refusal for any target that
// selects an external module, rather than a partial closure.
func ResolveQualifiedSourceClosure(repo, goPath, packagePath, supply, stateDir string) ([]SourceMember, string, ClosureAdmission, error) {
	root, err := filepath.Abs(repo)
	if err != nil {
		return nil, "", ClosureAdmission{}, err
	}
	if evaluated, err := filepath.EvalSymlinks(root); err == nil {
		root = evaluated
	}
	admission := ClosureAdmission{
		SchemaVersion: closureAdmissionSchema,
		Package:       packagePath,
		Policy:        ClosurePolicyRepositoryOnly,
		Boundary:      admissionBoundary,
	}

	cmd := exec.Command(goPath, "list", "-deps", "-json", "-mod=readonly", packagePath)
	cmd.Dir = root
	env, err := ObservationEnvironment(os.Environ(), stateDir, supply)
	if err != nil {
		return nil, "", admission, fmt.Errorf("prepare closure environment: %w", err)
	}
	cmd.Env = env
	raw, err := cmd.Output()
	if err != nil {
		// The resolver exits before reporting modules when the target needs a
		// supply it does not have. The refusal must still state its policy and
		// reason, so a caller cannot mistake "no closure" for "everything fine".
		admission.RefusedReason = "the resolver did not produce a package list: " + closedReason(err)
		if strings.TrimSpace(supply) == "" {
			admission.RefusedReason += "; the repository-only policy needs no supply, but a target with external modules cannot be resolved offline"
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, "", admission, fmt.Errorf("resolve Go package source closure: %w: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, "", admission, fmt.Errorf("resolve Go package source closure: %w", err)
	}

	members, modules, err := selectedMembers(root, raw)
	if err != nil {
		return nil, "", admission, err
	}
	if len(modules) > 0 && strings.TrimSpace(supply) == "" {
		admission.RefusedReason = "the target selects external modules and the repository-only policy refuses them; name an explicit supply to opt in"
		return nil, "", admission, errors.New(admission.RefusedReason)
	}

	repositoryMembers := 0
	for _, member := range members {
		if member.Path == externalMemberPrefix || strings.HasPrefix(member.Path, externalMemberPrefix) {
			continue
		}
		repositoryMembers++
	}
	externalMembers := len(members) - repositoryMembers
	if externalMembers > 0 {
		admission.Policy = ClosurePolicyExternalModuleOptIn
		absolute, err := filepath.Abs(supply)
		if err != nil {
			return nil, "", admission, err
		}
		admission.SupplyRoot = filepath.Clean(absolute)
	}
	admission.RepositoryMembers = repositoryMembers
	admission.ExternalMembers = externalMembers
	admission.ExternalModules = modules
	admission.GoSumBinding = goSumBindingText
	admission.Qualified = true
	admission.QualificationBasis = "selected compiler inputs, with repository members Git-tracked and external members named by module identity"
	admission.ClosureDigest = SourceClosureDigest(members)
	return members, admission.ClosureDigest, admission, nil
}

// selectedMembers decodes the resolver output into closure members. Repository
// members keep their repository-relative path; external members are prefixed so
// a repository file can never collide with a module file, and the prefix is
// reserved (a repository path using it is rejected).
func closedReason(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func selectedMembers(root string, raw []byte) ([]SourceMember, []string, error) {
	tracked, err := trackedSourcePaths(root)
	if err != nil {
		return nil, nil, err
	}
	paths := map[string]struct{}{"go.mod": {}}
	modules := map[string]struct{}{}
	externalPath := map[string]string{}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	for {
		var pkg GoListPackage
		if err := decoder.Decode(&pkg); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, nil, fmt.Errorf("decode Go package list: %w", err)
		}
		if pkg.Standard {
			continue
		}
		if pkg.Error != nil {
			return nil, nil, fmt.Errorf("selected package %s failed to resolve: %s", pkg.ImportPath, pkg.Error.Err)
		}
		if pkg.Module == nil {
			return nil, nil, fmt.Errorf("Go dependency %s has no module identity", pkg.ImportPath)
		}
		moduleDir := cleanDir(pkg.Module.Dir)
		packageDir := cleanDir(pkg.Dir)
		for _, name := range PackageBuildFiles(pkg) {
			full := filepath.Join(packageDir, name)
			if moduleDir == root {
				rel, err := filepath.Rel(root, full)
				if err != nil {
					return nil, nil, err
				}
				rel = filepath.ToSlash(rel)
				if rel == ".." || strings.HasPrefix(rel, "../") {
					return nil, nil, fmt.Errorf("Go package source escapes repository: %s", full)
				}
				if strings.HasPrefix(rel, externalMemberPrefix) {
					return nil, nil, fmt.Errorf("repository path uses the reserved external member namespace: %s", rel)
				}
				paths[rel] = struct{}{}
				continue
			}
			rel, err := filepath.Rel(moduleDir, full)
			if err != nil {
				return nil, nil, err
			}
			if rel == ".." || strings.HasPrefix(rel, "../") {
				return nil, nil, fmt.Errorf("external member escapes its module directory: %s", full)
			}
			module := pkg.Module.Path + "@" + pkg.Module.Version
			if pkg.Module.Version == "" {
				return nil, nil, fmt.Errorf("external module %s has no version", pkg.Module.Path)
			}
			modules[module] = struct{}{}
			externalPath[externalMemberPrefix+module+"/"+filepath.ToSlash(rel)] = full
		}
	}

	ordered := make([]string, 0, len(paths))
	for path := range paths {
		ordered = append(ordered, path)
	}
	sort.Strings(ordered)
	members := make([]SourceMember, 0, len(ordered)+len(externalPath))
	for _, rel := range ordered {
		if _, ok := tracked[rel]; !ok {
			return nil, nil, fmt.Errorf("Go package source closure member is not Git-tracked: %s", rel)
		}
		path := filepath.Join(root, filepath.FromSlash(rel))
		info, err := os.Lstat(path)
		if err != nil {
			return nil, nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return nil, nil, fmt.Errorf("Go package source closure member is not a regular file: %s", rel)
		}
		digest, err := digestSourceFile(path)
		if err != nil {
			return nil, nil, err
		}
		members = append(members, SourceMember{Path: rel, Digest: digest})
	}

	externalNames := make([]string, 0, len(externalPath))
	for name := range externalPath {
		externalNames = append(externalNames, name)
	}
	sort.Strings(externalNames)
	for _, name := range externalNames {
		path := externalPath[name]
		info, err := os.Lstat(path)
		if err != nil {
			return nil, nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return nil, nil, fmt.Errorf("external member is not a regular file: %s", name)
		}
		digest, err := digestSourceFile(path)
		if err != nil {
			return nil, nil, err
		}
		members = append(members, SourceMember{Path: name, Digest: digest})
	}

	moduleList := make([]string, 0, len(modules))
	for module := range modules {
		moduleList = append(moduleList, module)
	}
	sort.Strings(moduleList)
	return members, moduleList, nil
}

// cleanDir normalizes a directory reported by the resolver. Windows can report
// short 8.3 paths for one of the two directories, which would otherwise make a
// repository member look external.
func cleanDir(dir string) string {
	if evaluated, err := filepath.EvalSymlinks(dir); err == nil {
		return evaluated
	}
	return filepath.Clean(dir)
}
