package gobuilder

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// SourceClosureObservation is the read-only description of what a target
// actually compiles, including external module members.
//
// Qualified is always false: this observation does not apply the existing
// external-module policy, so it cannot grant the closure qualification that
// ResolveSourceClosure withholds. It exists so a caller can see the full member
// set and digests without changing that policy (#1090).
type SourceClosureObservation struct {
	SchemaVersion         string               `json:"schemaVersion"`
	Package               string               `json:"package"`
	RepositoryMembers     []SourceMember       `json:"repositoryMembers"`
	RepositoryDigest      string               `json:"repositoryDigest"`
	ExternalMembers       []ModuleSourceMember `json:"externalMembers"`
	ExternalModules       []string             `json:"externalModules"`
	ObservedDigest        string               `json:"observedDigest"`
	Qualified             bool                 `json:"qualified"`
	QualificationWithheld string               `json:"qualificationWithheldReason"`
	Boundary              string               `json:"boundary"`
}

// ModuleSourceMember is one selected compiler input from outside the target
// module. Name is a namespaced identifier (module:<path>@<version>/<file>) and
// a module-relative path, because the physical cache location is machine
// specific and is not part of the content identity.
type ModuleSourceMember struct {
	Name   string `json:"name"`
	Digest string `json:"digest"`
}

const (
	observationSchema = "tei.reference.go-source-closure-observation/v0"

	withheldReason = "external module members are observed but not accepted: the existing ResolveSourceClosure refuses every external module root (#1090 policy, unchanged by this observation)"

	observationBoundary = "read-only listing of selected compiler inputs including external module files; no policy change, no build, no acceptance, no authority"
)

// ObserveSourceClosure lists the compiler-selected inputs of a target, keeping
// repository members and external module members in separate collections.
//
// The repository digest uses the existing SourceClosureDigest encoding so it
// stays comparable with qualified closures. The overall observed digest
// additionally namespaces external members, so an observed digest can never be
// mistaken for a qualified closure digest.
func ObserveSourceClosure(repo, goPath, packagePath, supply, stateDir string) (SourceClosureObservation, error) {
	root, err := filepath.Abs(repo)
	if err != nil {
		return SourceClosureObservation{}, err
	}
	if evaluated, err := filepath.EvalSymlinks(root); err == nil {
		root = evaluated
	}

	cmd := exec.Command(goPath, "list", "-deps", "-json", "-mod=readonly", packagePath)
	cmd.Dir = root
	env, err := ObservationEnvironment(os.Environ(), stateDir, supply)
	if err != nil {
		return SourceClosureObservation{}, fmt.Errorf("prepare observation environment: %w", err)
	}
	cmd.Env = env
	raw, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return SourceClosureObservation{}, fmt.Errorf("observe Go source closure: %w: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return SourceClosureObservation{}, fmt.Errorf("observe Go source closure: %w", err)
	}
	return observeFromList(root, packagePath, raw)
}

func observeFromList(root, packagePath string, raw []byte) (SourceClosureObservation, error) {
	// Normalize the repository root the same way module directories are
	// normalized. On Windows a temporary directory can be reported as a short
	// (8.3) path while the resolver reports the long path, and comparing those
	// two unequal strings would misclassify every repository member as an
	// external module member.
	if evaluated, err := filepath.EvalSymlinks(root); err == nil {
		root = evaluated
	} else {
		root = filepath.Clean(root)
	}
	tracked, err := trackedSourcePaths(root)
	if err != nil {
		return SourceClosureObservation{}, err
	}
	observation := SourceClosureObservation{
		SchemaVersion:         observationSchema,
		Package:               packagePath,
		Qualified:             false,
		QualificationWithheld: withheldReason,
		Boundary:              observationBoundary,
	}
	repository := map[string]struct{}{"go.mod": {}}
	externalPath := map[string]string{}
	modules := map[string]struct{}{}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	for {
		var pkg GoListPackage
		if err := decoder.Decode(&pkg); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return SourceClosureObservation{}, fmt.Errorf("decode Go package list: %w", err)
		}
		if pkg.Standard {
			continue
		}
		if pkg.Error != nil {
			return SourceClosureObservation{}, fmt.Errorf("selected package %s failed to resolve: %s", pkg.ImportPath, pkg.Error.Err)
		}
		if pkg.Module == nil {
			return SourceClosureObservation{}, fmt.Errorf("Go dependency %s has no module identity", pkg.ImportPath)
		}
		moduleDir := pkg.Module.Dir
		if evaluated, err := filepath.EvalSymlinks(moduleDir); err == nil {
			moduleDir = evaluated
		} else {
			moduleDir = filepath.Clean(moduleDir)
		}
		for _, name := range PackageBuildFiles(pkg) {
			full := filepath.Join(pkg.Dir, name)
			if moduleDir == root {
				rel, err := filepath.Rel(root, full)
				if err != nil {
					return SourceClosureObservation{}, err
				}
				rel = filepath.ToSlash(rel)
				if rel == ".." || strings.HasPrefix(rel, "../") {
					return SourceClosureObservation{}, fmt.Errorf("Go package source escapes repository: %s", full)
				}
				repository[rel] = struct{}{}
				continue
			}
			rel, err := filepath.Rel(moduleDir, full)
			if err != nil {
				return SourceClosureObservation{}, err
			}
			module := pkg.Module.Path + "@" + pkg.Module.Version
			modules[module] = struct{}{}
			externalPath[module+"/"+filepath.ToSlash(rel)] = full
		}
	}

	orderedRepository := sortedKeys(repository)
	members := make([]SourceMember, 0, len(orderedRepository))
	for _, rel := range orderedRepository {
		if _, ok := tracked[rel]; !ok {
			// Untracked repository members are reported, not silently dropped:
			// the caller decides whether that invalidates its own claim.
			return SourceClosureObservation{}, fmt.Errorf("repository member is not Git-tracked: %s", rel)
		}
		digest, err := digestSourceFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return SourceClosureObservation{}, err
		}
		members = append(members, SourceMember{Path: rel, Digest: digest})
	}
	observation.RepositoryMembers = members
	observation.RepositoryDigest = SourceClosureDigest(members)

	observation.ExternalModules = sortedKeys(modules)
	for _, name := range sortedKeysOf(externalPath) {
		if _, _, ok := splitModuleMember(name); !ok {
			return SourceClosureObservation{}, fmt.Errorf("external member name is malformed: %s", name)
		}
		digest, err := digestSourceFile(externalPath[name])
		if err != nil {
			return SourceClosureObservation{}, err
		}
		observation.ExternalMembers = append(observation.ExternalMembers, ModuleSourceMember{Name: "module:" + name, Digest: digest})
	}
	observation.ObservedDigest = observedDigest(observation.RepositoryMembers, observation.ExternalMembers)
	return observation, nil
}

func sortedKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedKeysOf(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// splitModuleMember splits `<path>@<version>/<file>`. Module paths contain
// slashes, so the separator is the right-most '@' whose following segment is a
// version followed by a slash. A file whose name contains '@' therefore cannot
// be misread as the separator.
func splitModuleMember(name string) (string, string, bool) {
	rest := strings.TrimPrefix(name, "module:")
	for at := strings.LastIndex(rest, "@"); at > 0; at = strings.LastIndex(rest[:at], "@") {
		segment := rest[at+1:]
		slash := strings.Index(segment, "/")
		if slash <= 0 {
			continue
		}
		if !looksLikeVersion(segment[:slash]) {
			continue
		}
		return rest[:at], segment[slash+1:], true
	}
	return "", "", false
}

// looksLikeVersion accepts Go module versions (v1.2.3, v0.0.0-2024..., v2.9.0+incompatible)
// and rejects everything else, so a stray '@' cannot move the split point.
func looksLikeVersion(value string) bool {
	if len(value) < 2 || value[0] != 'v' {
		return false
	}
	return value[1] >= '0' && value[1] <= '9'
}

// observedDigest keeps the repository encoding and appends namespaced external
// members under a distinct prefix line.
func observedDigest(repository []SourceMember, external []ModuleSourceMember) string {
	combined := make([]SourceMember, 0, len(repository)+len(external))
	combined = append(combined, repository...)
	for _, member := range external {
		combined = append(combined, SourceMember{Path: member.Name, Digest: member.Digest})
	}
	sort.Slice(combined, func(i, j int) bool { return combined[i].Path < combined[j].Path })
	var builder strings.Builder
	builder.WriteString("observed\n")
	for _, member := range combined {
		builder.WriteString(member.Path)
		builder.WriteByte(0)
		builder.WriteString(member.Digest)
		builder.WriteByte('\n')
	}
	sum := sha256.Sum256([]byte(builder.String()))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// ObservationEnvironment reuses the adapter's deny-by-default construction with
// an optional module supply. With no supply the observation behaves like the
// existing resolver (offline, isolated), so a caller can contrast both.
func ObservationEnvironment(base []string, stateDir, supply string) ([]string, error) {
	env, err := Environment(base, stateDir, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(supply) == "" {
		return env, nil
	}
	absolute, err := filepath.Abs(supply)
	if err != nil {
		return nil, err
	}
	clean := filepath.ToSlash(filepath.Clean(absolute))
	url := "file://" + clean
	if runtime.GOOS == "windows" {
		url = "file:///" + strings.TrimPrefix(clean, "/")
	}
	out := make([]string, 0, len(env)+1)
	for _, entry := range env {
		if strings.HasPrefix(entry, "GOPROXY=") {
			continue
		}
		out = append(out, entry)
	}
	out = append(out, "GOPROXY="+url)
	sort.Strings(out)
	return out, nil
}
