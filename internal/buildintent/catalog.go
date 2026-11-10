package buildintent

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"reflect"
	"sort"
	"strings"

	"tei/internal/core"
	"tei/internal/plugin"
)

// This file carries the machinery a caller needs to turn a declaration into an
// exact BuildPlan: a source reader, a build catalog, and the three catalog
// lookups the resolver's ports ask for.
//
// SB-C0 §4.22 decided that the catalog is a data artifact the caller names and
// digests, not a table built into the binary. That decision is why these types
// carry product names rather than the "fixture" names they were introduced
// under: the mechanism is the product's, and which catalog a run used is
// recorded from the caller's input.

// DirSourceResolver reads a source reference from a directory root.
//
// It is not a substitute for language-specific closure resolution (#1096). It
// only fixes the contents of the named file, or of every regular file under the
// named directory, to a digest. It rejects:
//   - a symlink anywhere in the reference, including at the end
//   - a reference whose spelling does not match the real name exactly (case
//     differences are rejected)
//   - a path that leaves the root
//   - a member that is not a regular file
//   - a directory that contains no files
type DirSourceResolver struct {
	Root string
}

func (r DirSourceResolver) open() (*os.Root, error) {
	return os.OpenRoot(r.Root)
}

// exactEntry matches each component of the reference against the parent
// directory listing, spelling included, and confirms no symlink is traversed.
func exactEntry(root *os.Root, ref string) (fs.DirEntry, error) {
	dir := "."
	var entry fs.DirEntry
	for _, part := range strings.Split(ref, "/") {
		entries, err := fs.ReadDir(root.FS(), dir)
		if err != nil {
			return nil, err
		}
		entry = nil
		for _, e := range entries {
			if e.Name() == part {
				entry = e
				break
			}
		}
		if entry == nil {
			return nil, fmt.Errorf("source path component %q does not exist with exactly this spelling (case differences are rejected)", part)
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return nil, fmt.Errorf("source path %q passes through a symlink at %q", ref, part)
		}
		dir = path.Join(dir, part)
	}
	return entry, nil
}

// Resolve derives an exact BuildSource from the referenced content. The digest
// depends only on the relative member paths and the bytes, never on where the
// root sits on the host.
func (r DirSourceResolver) Resolve(src IntentSource) (ResolvedSource, error) {
	if err := ValidateSourceRef(src.Ref); err != nil {
		return ResolvedSource{}, err
	}
	root, err := r.open()
	if err != nil {
		return ResolvedSource{}, err
	}
	defer root.Close()
	entry, err := exactEntry(root, src.Ref)
	if err != nil {
		return ResolvedSource{}, err
	}
	type member struct {
		rel  string
		full string
	}
	var members []member
	switch {
	case entry.Type().IsRegular():
		members = append(members, member{rel: ".", full: src.Ref})
	case entry.IsDir():
		walkErr := fs.WalkDir(root.FS(), src.Ref, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if !d.Type().IsRegular() {
				return fmt.Errorf("source member %q is not a regular file (symlink and special files are rejected)", p)
			}
			members = append(members, member{rel: strings.TrimPrefix(p, src.Ref+"/"), full: p})
			return nil
		})
		if walkErr != nil {
			return ResolvedSource{}, walkErr
		}
		if len(members) == 0 {
			return ResolvedSource{}, fmt.Errorf("source %q contains no files", src.Ref)
		}
	default:
		return ResolvedSource{}, fmt.Errorf("source %q is neither a regular file nor a directory", src.Ref)
	}
	sort.Slice(members, func(i, j int) bool { return members[i].rel < members[j].rel })
	h := sha256.New()
	files := make(map[string][]byte, len(members))
	for _, m := range members {
		f, err := root.Open(m.full)
		if err != nil {
			return ResolvedSource{}, err
		}
		data, readErr := io.ReadAll(f)
		f.Close()
		if readErr != nil {
			return ResolvedSource{}, readErr
		}
		files[m.rel] = data
		fmt.Fprintf(h, "%d:%s\n%d\n", len(m.rel), m.rel, len(data))
		h.Write(data)
	}
	return ResolvedSource{
		Source: plugin.BuildSource{
			Kind:    src.Kind,
			ID:      src.ID,
			Locator: src.Ref,
			Digest:  "sha256:" + hex.EncodeToString(h.Sum(nil)),
		},
		Files: files,
	}, nil
}

// SingleFileSourceDigest returns the digest a plan records for a source that is
// one regular file (as opposed to a directory of members). The builder side needs
// this to check that the bytes it read are the bytes the plan names; without one
// implementation the two sides could frame the input differently and agree on
// nothing.
//
// The framing is part of the value, not an implementation detail: the resolver
// writes the member name and length before the bytes, so this reproduces it
// exactly for the single-file case, where the member name is ".".
func SingleFileSourceDigest(data []byte) string {
	h := sha256.New()
	fmt.Fprintf(h, "%d:%s\n%d\n", len("."), ".", len(data))
	h.Write(data)
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

// BuildCatalog is the data a caller supplies so a declaration can be resolved
// into an exact BuildPlan: which builder providers exist, which build
// dependencies are available, and which targets and profiles are defined.
//
// It is not a production catalog and this package ships none. §4.22 requires the
// caller to name the catalog and its digest, so a run can record which data it
// resolved against.
type BuildCatalog struct {
	FormatVersion int                   `json:"formatVersion"`
	Providers     []BuildProvider       `json:"providers"`
	Dependencies  []BuildDependencyItem `json:"dependencies"`
	Targets       []BuildTargetEntry    `json:"targets"`
}

// BuildProvider is one exact builder artifact a catalog offers.
type BuildProvider struct {
	Plugin         core.CanonicalID `json:"plugin"`
	Version        string           `json:"version"`
	Implementation string           `json:"implementation"`
	Contract       string           `json:"contract"`
	ContractMajor  int              `json:"contractMajor"`
	AuthorityScope string           `json:"authorityScope"`
	ArtifactDigest string           `json:"artifactDigest"`
	RecipeKinds    []string         `json:"recipeKinds"`
	Targets        []string         `json:"targets"`
	Profiles       []string         `json:"profiles"`
}

// BuildDependencyItem is one available build dependency.
type BuildDependencyItem struct {
	Kind           string `json:"kind"`
	ID             string `json:"id"`
	Version        string `json:"version"`
	ArtifactDigest string `json:"artifactDigest"`
}

// BuildTargetEntry is one target and the profiles defined for it.
type BuildTargetEntry struct {
	Target   string   `json:"target"`
	Profiles []string `json:"profiles"`
}

// ParseBuildCatalog reads a catalog strictly. Unknown fields, case variants and
// duplicate entries are refused: a catalog that can be read two ways cannot be
// recorded by digest and trusted.
func ParseBuildCatalog(raw []byte) (BuildCatalog, error) {
	if err := ValidateInputJSON(raw); err != nil {
		return BuildCatalog{}, fmt.Errorf("build catalog: %w", err)
	}
	var generic any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return BuildCatalog{}, fmt.Errorf("build catalog: %w", err)
	}
	if err := RejectKeyCaseVariants(generic, reflect.TypeOf(BuildCatalog{}), "$catalog"); err != nil {
		return BuildCatalog{}, err
	}
	var c BuildCatalog
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return BuildCatalog{}, fmt.Errorf("build catalog: %w", err)
	}
	if c.FormatVersion != 1 {
		return BuildCatalog{}, fmt.Errorf("unsupported build catalog format %d", c.FormatVersion)
	}
	if err := c.rejectDuplicates(); err != nil {
		return BuildCatalog{}, err
	}
	return c, nil
}

// rejectDuplicates refuses two declarations of the same thing. Allowing them
// would let enumeration order change the meaning.
func (c BuildCatalog) rejectDuplicates() error {
	seen := map[string]struct{}{}
	add := func(kind, key string) error {
		k := kind + "|" + key
		if _, dup := seen[k]; dup {
			return fmt.Errorf("build catalog declares %s more than once: %s", kind, key)
		}
		seen[k] = struct{}{}
		return nil
	}
	for _, p := range c.Providers {
		if err := add("provider", strings.Join([]string{string(p.Plugin), p.Version, p.Implementation, p.ArtifactDigest}, "|")); err != nil {
			return err
		}
	}
	for _, d := range c.Dependencies {
		if err := add("dependency", strings.Join([]string{d.Kind, d.ID, d.Version}, "|")); err != nil {
			return err
		}
	}
	for _, t := range c.Targets {
		if err := add("target", t.Target); err != nil {
			return err
		}
	}
	return nil
}

func contains(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

// ResolveBuilder answers which exact builder artifact satisfies the request: the
// contract and major it declares, the recipe kind, the target, and the profile
// when one is named. An explicit provider or pin narrows the choice further.
func (c BuildCatalog) ResolveBuilder(req BuilderRequest) (BuilderResolution, error) {
	var candidates []BuildProvider
	for _, p := range c.Providers {
		if p.AuthorityScope != builderAuthorityScope ||
			p.Contract != req.Contract || p.ContractMajor != req.CompatibleMajor ||
			!contains(p.RecipeKinds, req.RecipeKind) || !contains(p.Targets, req.Target) {
			continue
		}
		if req.Profile != "" && !contains(p.Profiles, req.Profile) {
			continue
		}
		if w := req.Provider; w != nil {
			if p.Plugin != w.Plugin || (w.Implementation != "" && p.Implementation != w.Implementation) {
				continue
			}
		}
		if req.Pin != nil && p.ArtifactDigest != req.Pin.ArtifactDigest {
			continue
		}
		candidates = append(candidates, p)
	}
	switch len(candidates) {
	case 0:
		if req.Pin != nil {
			return BuilderResolution{}, fmt.Errorf("no compatible builder provider matches the pinned artifact %s", req.Pin.ArtifactDigest)
		}
		return BuilderResolution{}, fmt.Errorf("no compatible builder provider for contract %s/v%d recipe %s target %s", req.Contract, req.CompatibleMajor, req.RecipeKind, req.Target)
	case 1:
		p := candidates[0]
		return BuilderResolution{
			Identity: plugin.BuilderIdentity{
				ID:              p.Plugin,
				Version:         p.Version,
				ContractVersion: BuilderContractVersion(p.Contract, p.ContractMajor),
				ArtifactDigest:  p.ArtifactDigest,
			},
			Implementation: p.Implementation,
			Pinned:         req.Pin != nil,
		}, nil
	}
	names := make([]string, 0, len(candidates))
	for _, p := range candidates {
		names = append(names, fmt.Sprintf("%s@%s/%s(%s)", p.Plugin, p.Version, p.Implementation, p.ArtifactDigest))
	}
	sort.Strings(names)
	return BuilderResolution{}, fmt.Errorf("builder provider is ambiguous; choose explicitly in builder.provider or builder.pin: %s", strings.Join(names, ", "))
}

// ResolveDependency answers which exact build dependency satisfies the request.
func (c BuildCatalog) ResolveDependency(req DependencyRequirement) (plugin.BuildDependency, error) {
	var candidates []BuildDependencyItem
	for _, d := range c.Dependencies {
		if d.Kind != req.Kind || d.ID != req.ID {
			continue
		}
		if !plugin.CompatibleMajor(d.Version, req.CompatibleMajor) {
			continue
		}
		candidates = append(candidates, d)
	}
	switch len(candidates) {
	case 0:
		return plugin.BuildDependency{}, fmt.Errorf("no producer output satisfies %s/%s major %d", req.Kind, req.ID, req.CompatibleMajor)
	case 1:
		d := candidates[0]
		return plugin.BuildDependency{Kind: d.Kind, ID: d.ID, Version: d.Version, ArtifactDigest: d.ArtifactDigest}, nil
	}
	versions := make([]string, 0, len(candidates))
	for _, d := range candidates {
		versions = append(versions, d.Version+"("+d.ArtifactDigest+")")
	}
	sort.Strings(versions)
	return plugin.BuildDependency{}, fmt.Errorf("build dependency %s/%s is ambiguous: %s", req.Kind, req.ID, strings.Join(versions, ", "))
}

// CheckTarget confirms the target is defined and, when a profile is named, that
// the profile is defined for it.
func (c BuildCatalog) CheckTarget(target, profile string) error {
	for _, t := range c.Targets {
		if t.Target != target {
			continue
		}
		if profile == "" || contains(t.Profiles, profile) {
			return nil
		}
		return fmt.Errorf("profile %q is not defined for target %q", profile, target)
	}
	return fmt.Errorf("target %q is not defined", target)
}

// The following are the catalog's fingerprints without enumeration order.

func (c BuildCatalog) builderFingerprint() (string, error) {
	providers := append([]BuildProvider(nil), c.Providers...)
	for i := range providers {
		providers[i].RecipeKinds = sortedCopy(providers[i].RecipeKinds)
		providers[i].Targets = sortedCopy(providers[i].Targets)
		providers[i].Profiles = sortedCopy(providers[i].Profiles)
	}
	return FingerprintJSON(sortByRecord(providers))
}

// sortByRecord orders elements by their full JSON rendering, so no two elements
// can compare equal.
func sortByRecord[T any](in []T) []T {
	type keyed struct {
		key string
		v   T
	}
	items := make([]keyed, len(in))
	for i, v := range in {
		raw, _ := json.Marshal(v)
		items[i] = keyed{key: string(raw), v: v}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].key < items[j].key })
	out := make([]T, len(in))
	for i, it := range items {
		out[i] = it.v
	}
	return out
}

func (c BuildCatalog) dependencyFingerprint() (string, error) {
	deps := append([]BuildDependencyItem(nil), c.Dependencies...)
	return FingerprintJSON(sortByRecord(deps))
}

func (c BuildCatalog) targetFingerprint() (string, error) {
	targets := append([]BuildTargetEntry(nil), c.Targets...)
	for i := range targets {
		targets[i].Profiles = sortedCopy(targets[i].Profiles)
	}
	return FingerprintJSON(sortByRecord(targets))
}

func sortedCopy(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

// The following types present one catalog as the three resolver ports.
type catalogBuilders struct{ c BuildCatalog }
type catalogDependencies struct{ c BuildCatalog }
type catalogTargets struct{ c BuildCatalog }

func (f catalogBuilders) Resolve(r BuilderRequest) (BuilderResolution, error) {
	return f.c.ResolveBuilder(r)
}
func (f catalogBuilders) CatalogFingerprint() (string, error) { return f.c.builderFingerprint() }
func (f catalogDependencies) Resolve(r DependencyRequirement) (plugin.BuildDependency, error) {
	return f.c.ResolveDependency(r)
}
func (f catalogDependencies) CatalogFingerprint() (string, error) {
	return f.c.dependencyFingerprint()
}
func (f catalogTargets) Check(target, profile string) error  { return f.c.CheckTarget(target, profile) }
func (f catalogTargets) CatalogFingerprint() (string, error) { return f.c.targetFingerprint() }

// NewBuildContext assembles a ResolutionContext from a source root and a catalog.
func NewBuildContext(sourceRoot string, catalog BuildCatalog) ResolutionContext {
	return ResolutionContext{
		Sources:      DirSourceResolver{Root: sourceRoot},
		Builders:     catalogBuilders{catalog},
		Dependencies: catalogDependencies{catalog},
		Targets:      catalogTargets{catalog},
	}
}
