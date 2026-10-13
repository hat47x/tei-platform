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
	"path"
	"sort"
	"strings"

	"tei/internal/core"
	"tei/internal/plugin"
)

// DirSourceResolver は実ファイルを読むReference用のソース解決である。
// 言語固有のclosure解決（#1096）の代替ではなく、参照先のファイル、または
// ディレクトリ配下の全通常ファイルの内容をdigestへ固定するだけである。
// 次を拒否する。
//   - 参照の途中・末尾にあるsymlink
//   - 綴りが実在の名前と大文字小文字まで一致しない参照
//   - root外へ出る経路
//   - 通常ファイル以外のメンバー
//   - ファイルを一つも含まないディレクトリ
type DirSourceResolver struct {
	Root string
}

func (r DirSourceResolver) open() (*os.Root, error) {
	return os.OpenRoot(r.Root)
}

// exactEntry は参照の各要素を親ディレクトリの一覧と綴りまで照合し、
// symlinkを経由しないことを確かめる。
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

// Resolve は参照先の内容からexactなBuildSourceを導出する。
// digestは相対member pathと内容だけから決まり、Host上のrootの場所に依存しない。
func (r DirSourceResolver) Resolve(src IntentSource) (ResolvedSource, error) {
	if err := validateSourceRef(src.Ref); err != nil {
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
	var c FixtureCatalog
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return FixtureCatalog{}, fmt.Errorf("fixture catalog: %w", err)
	}
	if c.FormatVersion != 1 {
		return FixtureCatalog{}, fmt.Errorf("unsupported fixture catalog format %d", c.FormatVersion)
	}
	if err := c.rejectDuplicates(); err != nil {
		return FixtureCatalog{}, err
	}
	return c, nil
}

// rejectDuplicates は同じ対象の重複宣言を拒否する。重複を許すと、
// 列挙順に依存して意味が変わる余地が生じる。
func (c FixtureCatalog) rejectDuplicates() error {
	seen := map[string]struct{}{}
	add := func(kind, key string) error {
		k := kind + "|" + key
		if _, dup := seen[k]; dup {
			return fmt.Errorf("fixture catalog declares %s more than once: %s", kind, key)
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

// 以下は列挙順を除いたカタログの照合値である。

func (c FixtureCatalog) builderFingerprint() (string, error) {
	providers := append([]FixtureProvider(nil), c.Providers...)
	for i := range providers {
		providers[i].RecipeKinds = sortedCopy(providers[i].RecipeKinds)
		providers[i].Targets = sortedCopy(providers[i].Targets)
		providers[i].Profiles = sortedCopy(providers[i].Profiles)
	}
	return fingerprintJSON(sortByRecord(providers))
}

// sortByRecord は各要素の完全なJSON表現の辞書順で整列する。同順位が生じない。
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

func (c FixtureCatalog) dependencyFingerprint() (string, error) {
	deps := append([]FixtureDependency(nil), c.Dependencies...)
	return fingerprintJSON(sortByRecord(deps))
}

func (c FixtureCatalog) targetFingerprint() (string, error) {
	targets := append([]FixtureTargetEntry(nil), c.Targets...)
	for i := range targets {
		targets[i].Profiles = sortedCopy(targets[i].Profiles)
	}
	return fingerprintJSON(sortByRecord(targets))
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
