// Package pluginbuildsupply runs the technical validation sample
// plugin-build-supply (sample 7): build intent and builder resolution
// (stage 1), build attempts and the content-addressed store (stage 2), the
// existing self-build runner (stage 3) and the call of the built process
// plugin over the stdio protocol version 1 (stage 4), joined by one run record.
//
// Everything here is specific to the sample and stays outside internal/.
// Product code (internal/plugin, internal/plugin/runtime, internal/syntax,
// internal/workspace) and tools/internal/gobuilder are called as they are.
package pluginbuildsupply

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

	"tei/internal/core"
	"tei/internal/plugin"
	"tei/internal/syntax"
	"tei/tools/internal/gobuilder"
)

// Intent is the author's build declaration. Its JSON form is the format of
// tools/reference-build-authoring-intent, so the same file is also given to
// that CLI. It holds no digest, locator or plan.
type Intent struct {
	FormatVersion int                     `json:"formatVersion"`
	Subject       IntentSubject           `json:"subject"`
	Sources       []IntentSource          `json:"sources"`
	Builder       BuilderRequirement      `json:"builder"`
	Recipe        plugin.BuildRecipe      `json:"recipe"`
	OutputKind    plugin.BuildOutputKind  `json:"outputKind"`
	Target        string                  `json:"target"`
	Profile       string                  `json:"profile,omitempty"`
	Parameters    []plugin.BuildParameter `json:"parameters,omitempty"`
}

type IntentSubject struct {
	Kind             string           `json:"kind"`
	ID               core.CanonicalID `json:"id"`
	Version          string           `json:"version"`
	DefinitionSource string           `json:"definitionSource"`
}

type IntentSource struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
	Ref  string `json:"ref"`
}

// BuilderRequirement is a selection policy, never an exact artifact.
type BuilderRequirement struct {
	Contract        string `json:"contract"`
	CompatibleMajor int    `json:"compatibleMajor"`
}

const (
	sourceKindDefinition = "plugin-definition"
	sourceKindGoClosure  = "go-source-closure"
	authorityBuildTool   = "build-tool"
)

// ParseIntent reads the declaration strictly: unknown fields and trailing
// data are rejected. The authoring-intent CLI applies its own stricter checks
// to the same bytes.
func ParseIntent(raw []byte) (Intent, error) {
	var in Intent
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		return Intent{}, fmt.Errorf("build intent: %w", err)
	}
	if _, err := dec.Token(); err != io.EOF {
		return Intent{}, fmt.Errorf("build intent has trailing data")
	}
	if in.FormatVersion != 1 || in.Subject.Kind != "plugin" || in.Subject.ID == "" || in.Subject.Version == "" {
		return Intent{}, fmt.Errorf("build intent subject is incomplete")
	}
	if in.Builder.Contract == "" || in.Builder.CompatibleMajor <= 0 {
		return Intent{}, fmt.Errorf("build intent builder requirement is incomplete")
	}
	return in, nil
}

// IntentFingerprint covers only what the author wrote, in a fixed order.
func IntentFingerprint(in Intent) (string, error) {
	n := in
	n.Sources = append([]IntentSource(nil), in.Sources...)
	n.Parameters = append([]plugin.BuildParameter(nil), in.Parameters...)
	sort.Slice(n.Sources, func(a, b int) bool { return n.Sources[a].ID < n.Sources[b].ID })
	sort.Slice(n.Parameters, func(a, b int) bool { return n.Parameters[a].Name < n.Parameters[b].Name })
	return fingerprintJSON(n)
}

// GoToolchain is the observed identity of the Go seed builder. The formula is
// the one tools/reference-plugin-self-build records as goToolchainDigest:
// sha256(go version, binary digest, GOROOT closure digest).
type GoToolchain struct {
	Executable      string `json:"-"`
	Version         string `json:"goVersion"`
	BinaryDigest    string `json:"goBinaryDigest"`
	RootDigest      string `json:"goRootDigest"`
	ToolchainDigest string `json:"goToolchainDigest"`
}

// ObserveGoToolchain hashes the exact executable and its GOROOT closure.
func ObserveGoToolchain(goExecutable string) (GoToolchain, error) {
	path, err := gobuilder.SelectExecutable(goExecutable, "", runtime.GOOS)
	if err != nil {
		return GoToolchain{}, err
	}
	binary, err := hashFile(path)
	if err != nil {
		return GoToolchain{}, err
	}
	env, err := gobuilder.Environment(os.Environ(), "", runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return GoToolchain{}, err
	}
	version, err := commandOutput(env, "", path, "version")
	if err != nil {
		return GoToolchain{}, fmt.Errorf("read Go version: %w", err)
	}
	root, err := commandOutput(env, "", path, "env", "GOROOT")
	if err != nil {
		return GoToolchain{}, fmt.Errorf("resolve GOROOT: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		root = resolved
	}
	rootDigest, err := gobuilder.HashToolchain(root, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return GoToolchain{}, err
	}
	return GoToolchain{
		Executable:      path,
		Version:         version,
		BinaryDigest:    binary,
		RootDigest:      rootDigest,
		ToolchainDigest: sha256Digest([]byte(version + "\n" + binary + "\n" + rootDigest + "\n")),
	}, nil
}

// Provider is one entry of the builder provider catalog.
type Provider struct {
	ID             core.CanonicalID `json:"id"`
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

// GoSeedProvider turns an observed toolchain into a catalog entry. The
// artifact digest is observed, never written by hand.
func GoSeedProvider(tc GoToolchain, target, profile string) Provider {
	return Provider{
		ID: core.MustCanonicalID("tei.builder.go.seed"), Version: tc.Version, Implementation: "go.seed",
		Contract: "tei.plugin-build", ContractMajor: 1, AuthorityScope: authorityBuildTool,
		ArtifactDigest: tc.ToolchainDigest, RecipeKinds: []string{"go-package"},
		Targets: []string{target}, Profiles: []string{profile},
	}
}

func (p Provider) builderIdentity() plugin.BuilderIdentity {
	return plugin.BuilderIdentity{
		ID: p.ID, Version: p.Version,
		ContractVersion: fmt.Sprintf("%s/v%d", p.Contract, p.ContractMajor),
		ArtifactDigest:  p.ArtifactDigest,
	}
}

func (p Provider) label() string {
	return fmt.Sprintf("%s@%s/%s(%s)", p.ID, p.Version, p.Implementation, p.ArtifactDigest)
}

// BuilderRequest is what the resolver must satisfy.
type BuilderRequest struct {
	Contract        string
	CompatibleMajor int
	RecipeKind      string
	Target          string
	Profile         string
	// ExplicitProvider, when set, narrows the candidates by logical ID.
	ExplicitProvider core.CanonicalID
}

var (
	ErrAmbiguousProvider = errors.New("builder provider is ambiguous")
	ErrNoProvider        = errors.New("no compatible builder provider")
	ErrLockDrift         = errors.New("locked builder provider is no longer in the catalog as locked")
	ErrBuilderRevoked    = errors.New("builder trust is revoked")
)

func eligible(p Provider, req BuilderRequest) bool {
	return p.AuthorityScope == authorityBuildTool && p.Contract == req.Contract &&
		p.ContractMajor == req.CompatibleMajor && contains(p.RecipeKinds, req.RecipeKind) &&
		contains(p.Targets, req.Target) && (req.Profile == "" || contains(p.Profiles, req.Profile))
}

// ResolveBuilder selects exactly one eligible provider. Several eligible
// providers stop the resolution; catalog order never decides.
func ResolveBuilder(req BuilderRequest, catalog []Provider) (Provider, error) {
	var candidates []Provider
	for _, p := range catalog {
		if !eligible(p, req) {
			continue
		}
		if req.ExplicitProvider != "" && p.ID != req.ExplicitProvider {
			continue
		}
		candidates = append(candidates, p)
	}
	switch len(candidates) {
	case 0:
		return Provider{}, fmt.Errorf("%w for %s/v%d recipe %s target %s", ErrNoProvider, req.Contract, req.CompatibleMajor, req.RecipeKind, req.Target)
	case 1:
		return candidates[0], nil
	}
	names := make([]string, 0, len(candidates))
	for _, p := range candidates {
		names = append(names, p.label())
	}
	sort.Strings(names)
	return Provider{}, fmt.Errorf("%w; choose explicitly: %s", ErrAmbiguousProvider, strings.Join(names, ", "))
}

// ResolutionLock is the fixed record of one resolution. Replaying it never
// consults the rest of the catalog, so added providers cannot change it.
type ResolutionLock struct {
	FormatVersion     int                    `json:"formatVersion"`
	IntentFingerprint string                 `json:"intentFingerprint"`
	Provider          Provider               `json:"provider"`
	Builder           plugin.BuilderIdentity `json:"builder"`
	Fingerprint       string                 `json:"fingerprint"`
}

func NewResolutionLock(intentFingerprint string, p Provider) (ResolutionLock, error) {
	lock := ResolutionLock{FormatVersion: 1, IntentFingerprint: intentFingerprint, Provider: p, Builder: p.builderIdentity()}
	fp, err := lockFingerprint(lock)
	if err != nil {
		return ResolutionLock{}, err
	}
	lock.Fingerprint = fp
	return lock, nil
}

func lockFingerprint(lock ResolutionLock) (string, error) {
	lock.Fingerprint = ""
	return fingerprintJSON(lock)
}

// Validate recomputes the lock's own consistency.
func (l ResolutionLock) Validate() error {
	fp, err := lockFingerprint(l)
	if err != nil {
		return err
	}
	if fp != l.Fingerprint || l.Builder != l.Provider.builderIdentity() {
		return fmt.Errorf("resolution lock is not self-consistent")
	}
	return nil
}

// ReplayLock reproduces the locked resolution against the current catalog.
// The exact locked provider must still be present and eligible; other
// providers are ignored. Trust is deliberately not consulted: reproduction
// stays possible after revocation, use does not (see Admit).
func ReplayLock(lock ResolutionLock, req BuilderRequest, catalog []Provider) (ResolutionLock, error) {
	if err := lock.Validate(); err != nil {
		return ResolutionLock{}, err
	}
	want, err := fingerprintJSON(lock.Provider)
	if err != nil {
		return ResolutionLock{}, err
	}
	matches := 0
	for _, p := range catalog {
		got, err := fingerprintJSON(p)
		if err != nil {
			return ResolutionLock{}, err
		}
		if got == want && eligible(p, req) {
			matches++
		}
	}
	if matches != 1 {
		return ResolutionLock{}, fmt.Errorf("%w: %s (matches=%d)", ErrLockDrift, lock.Provider.label(), matches)
	}
	return lock, nil
}

// Revocation withdraws trust from one exact builder artifact.
type Revocation struct {
	ProviderID     core.CanonicalID `json:"providerId"`
	ArtifactDigest string           `json:"artifactDigest"`
	Reason         string           `json:"reason"`
}

// TrustStore holds the current revocations. Its absence of an entry is not
// a grant of trust beyond this sample: it only means "not revoked here".
type TrustStore struct {
	Revoked []Revocation `json:"revoked"`
}

// Admit decides whether a locked builder may be used now.
func (t TrustStore) Admit(lock ResolutionLock) error {
	for _, r := range t.Revoked {
		if r.ProviderID == lock.Builder.ID && r.ArtifactDigest == lock.Builder.ArtifactDigest {
			return fmt.Errorf("%w: %s (%s)", ErrBuilderRevoked, lock.Provider.label(), r.Reason)
		}
	}
	return nil
}

// Stage1 is the confirmed plan with the observations it was derived from.
type Stage1 struct {
	Revision          string                   `json:"repositoryRevision"`
	IntentFingerprint string                   `json:"intentFingerprint"`
	Toolchain         GoToolchain              `json:"toolchain"`
	SourceClosure     []gobuilder.SourceMember `json:"-"`
	Lock              ResolutionLock           `json:"resolutionLock"`
	Plan              plugin.BuildPlan         `json:"plan"`
	PlanFingerprint   string                   `json:"planFingerprint"`
}

// Request returns the builder request the intent implies.
func (in Intent) Request() BuilderRequest {
	return BuilderRequest{
		Contract: in.Builder.Contract, CompatibleMajor: in.Builder.CompatibleMajor,
		RecipeKind: in.Recipe.Kind, Target: in.Target, Profile: in.Profile,
	}
}

// DeriveOptions carries what stage 1 observes besides the repository.
type DeriveOptions struct {
	Toolchain GoToolchain
	// Catalog is the provider catalog at derivation time.
	Catalog []Provider
	// Lock, when set, is replayed instead of resolving fresh.
	Lock *ResolutionLock
	// StateDir is an empty directory for the isolated go list.
	StateDir string
}

// Derive confirms the build plan from the intent and real files. It stops on
// a dirty working tree, an external module, an ambiguous provider or a
// drifted lock. Nothing the author wrote is a digest.
func Derive(repo string, in Intent, opt DeriveOptions) (Stage1, error) {
	if err := RequireCleanRepository(repo); err != nil {
		return Stage1{}, err
	}
	revision, err := gitOutput(repo, "rev-parse", "HEAD")
	if err != nil {
		return Stage1{}, err
	}
	intentFP, err := IntentFingerprint(in)
	if err != nil {
		return Stage1{}, err
	}
	var (
		sources    []plugin.BuildSource
		definition *plugin.Definition
		closure    []gobuilder.SourceMember
	)
	for _, src := range in.Sources {
		switch src.Kind {
		case sourceKindDefinition:
			raw, err := readInside(repo, src.Ref)
			if err != nil {
				return Stage1{}, err
			}
			parsed, err := syntax.NewParser(string(raw)).ParsePlugin()
			if err != nil {
				return Stage1{}, fmt.Errorf("parse plugin definition %s: %w", src.Ref, err)
			}
			if src.ID == in.Subject.DefinitionSource {
				definition = &parsed
			}
			sources = append(sources, plugin.BuildSource{Kind: src.Kind, ID: src.ID, Locator: src.Ref, Digest: sha256Digest(raw)})
		case sourceKindGoClosure:
			pkg := "./" + src.Ref
			members, digest, err := gobuilder.ResolveSourceClosure(repo, opt.Toolchain.Executable, pkg, opt.StateDir)
			if err != nil {
				return Stage1{}, fmt.Errorf("source closure cannot be fixed: %w", err)
			}
			closure = members
			sources = append(sources, plugin.BuildSource{Kind: src.Kind, ID: src.ID, Locator: pkg, Digest: digest})
		default:
			return Stage1{}, fmt.Errorf("source kind %q has no resolver in this sample", src.Kind)
		}
	}
	if definition == nil {
		return Stage1{}, fmt.Errorf("intent names no plugin definition source %q", in.Subject.DefinitionSource)
	}
	if definition.ID != in.Subject.ID || definition.Version != in.Subject.Version {
		return Stage1{}, fmt.Errorf("plugin definition %s@%s does not match the intent subject", definition.ID, definition.Version)
	}
	if definition.Artifact.Digest != "" {
		return Stage1{}, fmt.Errorf("plugin definition predeclares an artifact")
	}
	req := in.Request()
	var lock ResolutionLock
	if opt.Lock != nil {
		if opt.Lock.IntentFingerprint != intentFP {
			return Stage1{}, fmt.Errorf("resolution lock was made for another intent")
		}
		if lock, err = ReplayLock(*opt.Lock, req, opt.Catalog); err != nil {
			return Stage1{}, err
		}
	} else {
		provider, err := ResolveBuilder(req, opt.Catalog)
		if err != nil {
			return Stage1{}, err
		}
		if lock, err = NewResolutionLock(intentFP, provider); err != nil {
			return Stage1{}, err
		}
	}
	plan := plugin.BuildPlan{
		FormatVersion:               plugin.BuildPlanFormatVersion,
		Plugin:                      plugin.Identity{ID: in.Subject.ID, Version: in.Subject.Version},
		PluginDefinitionFingerprint: plugin.BuildDefinitionFingerprint(*definition),
		Sources:                     sources,
		Builder:                     lock.Builder,
		Recipe:                      in.Recipe,
		OutputKind:                  in.OutputKind,
		Target:                      in.Target,
		Profile:                     in.Profile,
		Parameters:                  append([]plugin.BuildParameter(nil), in.Parameters...),
	}
	planFP, err := plugin.BuildPlanFingerprint(plan)
	if err != nil {
		return Stage1{}, fmt.Errorf("derived build plan is invalid: %w", err)
	}
	return Stage1{
		Revision: revision, IntentFingerprint: intentFP, Toolchain: opt.Toolchain, SourceClosure: closure,
		Lock: lock, Plan: plan, PlanFingerprint: planFP,
	}, nil
}

// RequireCleanRepository applies the runner's own rule: no change and no
// untracked file.
func RequireCleanRepository(repo string) error {
	status, err := gitOutput(repo, "status", "--porcelain=v1", "--untracked-files=all")
	if err != nil {
		return err
	}
	if status != "" {
		return fmt.Errorf("working tree is not clean; the source closure cannot be fixed")
	}
	return nil
}

func readInside(repo, ref string) ([]byte, error) {
	if ref == "" || filepath.IsAbs(ref) || strings.Contains(ref, `\`) || strings.HasPrefix(ref, "../") || filepath.ToSlash(filepath.Clean(ref)) != ref {
		return nil, fmt.Errorf("source ref %q must be a clean relative path", ref)
	}
	root, err := os.OpenRoot(repo)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	f, err := root.Open(ref)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

func contains(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

func fingerprintJSON(v any) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func sha256Digest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

func commandOutput(env []string, dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Env = env
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func gitOutput(repo string, args ...string) (string, error) {
	out, err := commandOutput(os.Environ(), "", "git", append([]string{"-C", repo}, args...)...)
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return out, nil
}
