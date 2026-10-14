package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"sort"
	"strings"

	"tei/internal/core"
	"tei/internal/plugin"
)

// DirSourceResolver は実ファイルを読むReference用のSource解決である。
// 言語固有のclosure解決（#1096）の代替ではなく、refが指すファイルまたは
// ディレクトリ配下の全通常ファイルの内容をdigestへ固定するだけである。
// symlink、root外へ出る経路、特殊ファイルは拒否する。
type DirSourceResolver struct {
	Root string
}

func (r DirSourceResolver) open() (*os.Root, error) {
	return os.OpenRoot(r.Root)
}

// ReadFile はroot内の通常ファイルを読む。
func (r DirSourceResolver) ReadFile(ref string) ([]byte, error) {
	if err := validateSourceRef(ref); err != nil {
		return nil, err
	}
	root, err := r.open()
	if err != nil {
		return nil, err
	}
	defer root.Close()
	info, err := root.Lstat(ref)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("source %q is not a regular file", ref)
	}
	f, err := root.Open(ref)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

// Resolve はrefが指す内容からexactなBuildSourceを導出する。
// digestは相対member pathと内容だけから決まり、Host上のrootの場所に依存しない。
func (r DirSourceResolver) Resolve(src IntentSource) (plugin.BuildSource, error) {
	if err := validateSourceRef(src.Ref); err != nil {
		return plugin.BuildSource{}, err
	}
	root, err := r.open()
	if err != nil {
		return plugin.BuildSource{}, err
	}
	defer root.Close()
	info, err := root.Lstat(src.Ref)
	if err != nil {
		return plugin.BuildSource{}, err
	}
	type member struct {
		rel  string
		full string
	}
	var members []member
	switch {
	case info.Mode().IsRegular():
		members = append(members, member{rel: ".", full: src.Ref})
	case info.IsDir():
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
			return plugin.BuildSource{}, walkErr
		}
	default:
		return plugin.BuildSource{}, fmt.Errorf("source %q is neither a regular file nor a directory", src.Ref)
	}
	sort.Slice(members, func(i, j int) bool { return members[i].rel < members[j].rel })
	h := sha256.New()
	for _, m := range members {
		f, err := root.Open(m.full)
		if err != nil {
			return plugin.BuildSource{}, err
		}
		data, readErr := io.ReadAll(f)
		f.Close()
		if readErr != nil {
			return plugin.BuildSource{}, readErr
		}
		fmt.Fprintf(h, "%d:%s\n%d\n", len(m.rel), m.rel, len(data))
		h.Write(data)
	}
	return plugin.BuildSource{
		Kind:    src.Kind,
		ID:      src.ID,
		Locator: src.Ref,
		Digest:  "sha256:" + hex.EncodeToString(h.Sum(nil)),
	}, nil
}

// FixtureCatalog はBuilder Provider、Build依存、Target / ProfileのReference用Catalogである。
// production Resolverではなく、解決の入口（BuilderResolver / DependencyResolver /
// TargetCatalog）へ接続できることを確かめるための固定入力である。
type FixtureCatalog struct {
	FormatVersion int                  `json:"formatVersion"`
	Providers     []FixtureProvider    `json:"providers"`
	Dependencies  []FixtureDependency  `json:"dependencies"`
	Targets       []FixtureTargetEntry `json:"targets"`
}

type FixtureProvider struct {
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

type FixtureDependency struct {
	Kind           string `json:"kind"`
	ID             string `json:"id"`
	Version        string `json:"version"`
	ArtifactDigest string `json:"artifactDigest"`
}

type FixtureTargetEntry struct {
	Target   string   `json:"target"`
	Profiles []string `json:"profiles"`
}

// ParseFixtureCatalog はCatalog JSONを厳密に読む。
func ParseFixtureCatalog(raw []byte) (FixtureCatalog, error) {
	if err := validateInputJSON(raw); err != nil {
		return FixtureCatalog{}, fmt.Errorf("fixture catalog: %w", err)
	}
	var c FixtureCatalog
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return FixtureCatalog{}, fmt.Errorf("fixture catalog: %w", err)
	}
	if c.FormatVersion != 1 {
		return FixtureCatalog{}, fmt.Errorf("unsupported fixture catalog format %d", c.FormatVersion)
	}
	return c, nil
}

func contains(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

// Resolve はContract / Authority scope / Recipe / Target / Profileに適合する
// Providerを列挙し、明示wiringとpinで絞る。複数残る場合は列挙順で選ばず停止する。
func (c FixtureCatalog) ResolveBuilder(req BuilderRequest) (BuilderResolution, error) {
	var candidates []FixtureProvider
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
				ContractVersion: fmt.Sprintf("%s/v%d", p.Contract, p.ContractMajor),
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

func (c FixtureCatalog) ResolveDependency(req DependencyRequirement) (plugin.BuildDependency, error) {
	var candidates []FixtureDependency
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

func (c FixtureCatalog) CheckTarget(target, profile string) error {
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

// 以下は列挙順を除いたCatalog fingerprintである。

func (c FixtureCatalog) builderFingerprint() (string, error) {
	providers := append([]FixtureProvider(nil), c.Providers...)
	for i := range providers {
		providers[i].RecipeKinds = sortedCopy(providers[i].RecipeKinds)
		providers[i].Targets = sortedCopy(providers[i].Targets)
		providers[i].Profiles = sortedCopy(providers[i].Profiles)
	}
	sort.Slice(providers, func(i, j int) bool {
		a, b := providers[i], providers[j]
		if a.Plugin != b.Plugin {
			return a.Plugin < b.Plugin
		}
		if a.Implementation != b.Implementation {
			return a.Implementation < b.Implementation
		}
		if a.Version != b.Version {
			return a.Version < b.Version
		}
		return a.ArtifactDigest < b.ArtifactDigest
	})
	return fingerprintJSON(providers)
}

func (c FixtureCatalog) dependencyFingerprint() (string, error) {
	deps := append([]FixtureDependency(nil), c.Dependencies...)
	sort.Slice(deps, func(i, j int) bool {
		a, b := deps[i], deps[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.ID != b.ID {
			return a.ID < b.ID
		}
		return a.Version < b.Version
	})
	return fingerprintJSON(deps)
}

func (c FixtureCatalog) targetFingerprint() (string, error) {
	targets := append([]FixtureTargetEntry(nil), c.Targets...)
	for i := range targets {
		targets[i].Profiles = sortedCopy(targets[i].Profiles)
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].Target < targets[j].Target })
	return fingerprintJSON(targets)
}

func sortedCopy(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

// FixtureBuilders などは、一つのCatalogを三つの解決入口として見せる薄い型である。
type fixtureBuilders struct{ c FixtureCatalog }
type fixtureDependencies struct{ c FixtureCatalog }
type fixtureTargets struct{ c FixtureCatalog }

func (f fixtureBuilders) Resolve(r BuilderRequest) (BuilderResolution, error) {
	return f.c.ResolveBuilder(r)
}
func (f fixtureBuilders) CatalogFingerprint() (string, error) { return f.c.builderFingerprint() }
func (f fixtureDependencies) Resolve(r DependencyRequirement) (plugin.BuildDependency, error) {
	return f.c.ResolveDependency(r)
}
func (f fixtureDependencies) CatalogFingerprint() (string, error) {
	return f.c.dependencyFingerprint()
}
func (f fixtureTargets) Check(target, profile string) error  { return f.c.CheckTarget(target, profile) }
func (f fixtureTargets) CatalogFingerprint() (string, error) { return f.c.targetFingerprint() }

// NewFixtureContext はDirSourceResolverとFixtureCatalogからResolutionContextを組む。
func NewFixtureContext(sourceRoot string, catalog FixtureCatalog) ResolutionContext {
	return ResolutionContext{
		Sources:      DirSourceResolver{Root: sourceRoot},
		Builders:     fixtureBuilders{catalog},
		Dependencies: fixtureDependencies{catalog},
		Targets:      fixtureTargets{catalog},
	}
}
