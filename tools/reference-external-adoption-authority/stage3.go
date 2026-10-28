package externaladoptionauthority

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"tei/internal/composition"
	"tei/internal/core"
	"tei/internal/plugin"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
	semanticbuildbridge "tei/reference/semantic-build-bridge"
	semanticcompositionpreflight "tei/reference/semantic-composition-preflight"
	semanticdistribution "tei/reference/semantic-distribution"
)

// Identities of the sample's distribution. They are synthetic.
const (
	PackageID        = "sui_sensemaking.island_models"
	PackageVersion   = "1.0.0"
	ProviderID       = "read-sui"
	CompositionScope = "project-board"
	PublisherID      = "sui_sensemaking.publisher"
	SignerRef        = "sui_sensemaking.semantic_signer"
)

var semanticRead = composition.Contract{
	ID: "reference.semantic.asset_read", Version: "1.0.0", Shape: "data-only/select-and-fetch/v0",
}

type stage3Out struct {
	Dir         string
	PayloadPath string
	Plan        plugin.BuildPlan
	Lock        plugin.BuildArtifactLock
	Locked      plugin.LockedBuild
	Expected    semanticdistribution.Expectation
	Publication semanticcompositionpreflight.PublishedSelection
	Carried     Carried
}

func loadNamespace(root, namespace string) (semantic.Namespace, error) {
	loaded, err := workspace.Load(root)
	if err != nil {
		return semantic.Namespace{}, err
	}
	for _, ns := range loaded.World.Semantics {
		if string(ns.ID) == namespace {
			return ns, nil
		}
	}
	return semantic.Namespace{}, fmt.Errorf("namespace %s is not in %s", namespace, root)
}

// distributionPlan binds the distributed namespace to what stages 1 and 2
// fixed: the semantic digest, the semantic lock, the mapping report, the
// loss record and the pinned SUI source. The plan fingerprint covers them,
// and the publication signature covers the plan fingerprint.
func distributionPlan(c Carried) (plugin.BuildPlan, error) {
	pinned, err := digestJSON(c.Source)
	if err != nil {
		return plugin.BuildPlan{}, err
	}
	def := digestBytes([]byte("synthetic definition of " + PackageID))
	return plugin.BuildPlan{
		FormatVersion:               plugin.BuildPlanFormatVersion,
		Plugin:                      plugin.Identity{ID: core.MustCanonicalID(PackageID), Version: PackageVersion},
		PluginDefinitionFingerprint: def[len("sha256:"):],
		Sources: []plugin.BuildSource{
			{Kind: "semantic-namespace", ID: c.Namespace, Locator: "reference://external-adoption-authority/semantic/" + c.Namespace, Digest: c.SemanticDigest},
			{Kind: "semantic-lock", ID: "semantic-lock", Locator: "reference://external-adoption-authority/stage2/semantic-lock.json", Digest: c.SemanticLockDigest},
			{Kind: "mapping-report", ID: "mapping-report", Locator: "reference://external-adoption-authority/stage1/reports", Digest: c.MappingReportDigest},
			{Kind: "mapping-loss-record", ID: "loss-record", Locator: "reference://external-adoption-authority/stage1/loss-record.json", Digest: c.LossRecordDigest},
			{Kind: "pinned-external-source", ID: "sui-source", Locator: "reference://external-adoption-authority/stage1/pinned-source", Digest: pinned},
		},
		Builder: plugin.BuilderIdentity{
			ID: "tei.builder.semantic_export_reference", Version: "1.0.0",
			ContractVersion: "tei.semantic-export-reference/v0",
			ArtifactDigest:  digestBytes([]byte("synthetic builder tei.builder.semantic_export_reference")),
		},
		Recipe:     plugin.BuildRecipe{Kind: "deterministic-semantic-reference", Entrypoint: c.Namespace},
		OutputKind: semanticdistribution.SemanticReferenceOutputKind,
		Target:     semanticbuildbridge.ReferenceTarget,
	}, nil
}

func signPublication(asset semanticcompositionpreflight.SelectedAsset, key SignerKey) (semanticcompositionpreflight.PublicationProof, error) {
	proof := semanticcompositionpreflight.PublicationProof{
		PublisherID: PublisherID, SignerRef: key.Ref, ValidUntil: PublicationValidUntil,
	}
	msg, err := semanticcompositionpreflight.SemanticPublicationMessage(CompositionScope, asset, proof)
	if err != nil {
		return proof, err
	}
	proof.Signature = ed25519.Sign(key.private, msg)
	return proof, nil
}

func newRegistry() (*composition.Registry, error) {
	r := composition.NewRegistry()
	_, err := r.Apply(0, []composition.Provider{{ID: ProviderID, Scope: CompositionScope, Offers: semanticRead}}, nil)
	return r, err
}

// SignerJudgment is one run of the existing signed preflight at one trust
// point and logical time.
type SignerJudgment struct {
	Point        int    `json:"trustPoint"`
	Label        string `json:"trustPointLabel"`
	Now          string `json:"logicalNow"`
	Qualified    bool   `json:"qualified"`
	Error        string `json:"error,omitempty"`
	Dependencies int    `json:"semanticDependencies"`
	Diagnostics  int    `json:"diagnostics"`
}

// judgeSigner runs InspectPublished with the lookup of one trust point.
func judgeSigner(pub semanticcompositionpreflight.PublishedSelection, p TrustPoint, lookup semanticcompositionpreflight.PublisherTrustLookup) SignerJudgment {
	j := SignerJudgment{Point: p.Seq, Label: p.Label, Now: p.Now.Format("2006-01-02T15:04:05Z")}
	registry, err := newRegistry()
	if err != nil {
		j.Error = err.Error()
		return j
	}
	if lookup == nil {
		lookup = p.SignerLookup()
	}
	report, err := semanticcompositionpreflight.InspectPublished(registry, CompositionScope, nil,
		[]semanticcompositionpreflight.PublishedSelection{pub}, lookup, p.Now)
	j.Dependencies = len(report.References.Dependencies)
	j.Diagnostics = len(report.Diagnostics)
	if err != nil {
		j.Error = err.Error()
		return j
	}
	j.Qualified = true
	return j
}

func fileFetcher(path string) func(string) ([]byte, error) {
	return func(digest string) ([]byte, error) {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if sourceindex.Fingerprint(raw) != digest {
			return nil, errors.New("distribution file does not hold the requested artifact")
		}
		return raw, nil
	}
}

// checkStage2Handoff is stage 3's entry check.
func checkStage2Handoff(s2 stage2Out, providerRoot string, c Carried) error {
	lockRaw, err := os.ReadFile(s2.LockPath)
	if err != nil {
		return err
	}
	if d := digestBytes(lockRaw); d != c.SemanticLockDigest {
		return fmt.Errorf("%w: semantic lock digest %s, carried %s", ErrInputMismatch, d, c.SemanticLockDigest)
	}
	var lock struct {
		SemanticSetFingerprint string `json:"semanticSetFingerprint"`
	}
	if err := json.Unmarshal(lockRaw, &lock); err != nil {
		return err
	}
	if lock.SemanticSetFingerprint != c.SemanticSetFingerprint {
		return fmt.Errorf("%w: semantic set %s, carried %s", ErrInputMismatch, lock.SemanticSetFingerprint, c.SemanticSetFingerprint)
	}
	d, err := semanticDigestOf(providerRoot, c.Namespace)
	if err != nil {
		return err
	}
	if d != c.SemanticDigest {
		return fmt.Errorf("%w: namespace %s semantic digest %s, carried %s", ErrInputMismatch, c.Namespace, d, c.SemanticDigest)
	}
	return nil
}

// runStage3 fixes the namespace as a data-only distribution, binds it with
// the existing BuildPlan / BuildArtifactLock, signs it, and runs the
// existing build-bridge reader and the signed composition preflight.
func (s *Sample) runStage3(ctx context.Context, s1 stage1Out, s2 stage2Out, c Carried, p TrustPoint, dir string) (stage3Out, map[string]any, error) {
	out := stage3Out{Dir: dir}
	providerRoot := filepath.Join(s1.InputDir, "workspace")
	if err := checkStage2Handoff(s2, providerRoot, c); err != nil {
		return out, nil, err
	}
	if _, _, err := readLossRecord(s1.LossPath, c.LossRecordDigest); err != nil {
		return out, nil, err
	}
	owner, err := s.judgeOwner(ctx, s2.Inputs, s2.LockPath, p, dir)
	if err != nil {
		return out, nil, err
	}
	if !owner.Qualified {
		return out, map[string]any{"ownerJudgment": owner},
			fmt.Errorf("namespace owner is not qualified at trust point %d: %s", p.Seq, owner.Error)
	}
	ns, err := loadNamespace(providerRoot, c.Namespace)
	if err != nil {
		return out, nil, err
	}
	identity := semanticdistribution.Identity{PackageID: PackageID, Version: PackageVersion}
	payload, err := semanticdistribution.MaterializeReference(identity, ns)
	if err != nil {
		return out, nil, err
	}
	out.PayloadPath = filepath.Join(dir, "distribution.json")
	if err := writeFile(out.PayloadPath, payload); err != nil {
		return out, nil, err
	}
	out.Plan, err = distributionPlan(c)
	if err != nil {
		return out, nil, err
	}
	artifact := plugin.BuildOutputDigest(payload)
	out.Locked, err = plugin.LockBuildOutput(out.Plan, artifact)
	if err != nil {
		return out, nil, err
	}
	out.Lock = plugin.BuildArtifactLock{FormatVersion: plugin.BuildArtifactLockFormatVersion, Builds: []plugin.LockedBuild{out.Locked}}
	out.Expected = semanticdistribution.Expectation{
		Package: identity, NamespaceID: c.Namespace, ArtifactDigest: artifact, SemanticDigest: c.SemanticDigest,
	}
	read, err := semanticbuildbridge.InspectLockedExport(out.Plan, out.Lock, out.Locked, payload, out.Expected)
	if err != nil {
		return out, nil, fmt.Errorf("build bridge: %w", err)
	}
	raw, err := json.Marshal(read)
	if err != nil {
		return out, nil, err
	}
	if sourceindex.Fingerprint(raw) != c.SemanticDigest {
		return out, nil, fmt.Errorf("namespace read back from the distribution has another semantic digest")
	}
	key, ok := p.Key(SignerRef)
	if !ok {
		return out, nil, fmt.Errorf("trust point %d has no key for %s", p.Seq, SignerRef)
	}
	asset := semanticcompositionpreflight.SelectedAsset{
		ProviderID: ProviderID, Contract: semanticRead, Lock: out.Lock, Build: out.Locked,
		Expected: out.Expected, Fetch: fileFetcher(out.PayloadPath),
	}
	proof, err := signPublication(asset, key)
	if err != nil {
		return out, nil, err
	}
	out.Publication = semanticcompositionpreflight.PublishedSelection{Asset: asset, Proof: proof}
	signer := judgeSigner(out.Publication, p, nil)
	if !signer.Qualified {
		return out, map[string]any{"signerJudgment": signer}, fmt.Errorf("signed preflight refused: %s", signer.Error)
	}
	for name, v := range map[string]any{"build-plan.json": out.Plan, "build-artifact-lock.json": out.Lock,
		"publication-proof.json": map[string]string{
			"publisherId": proof.PublisherID, "signerRef": proof.SignerRef,
			"validUntil": proof.ValidUntil.Format("2006-01-02T15:04:05Z"), "signature": hex.EncodeToString(proof.Signature),
		}} {
		if _, err := s.writeJSON(filepath.Join(dir, name), v); err != nil {
			return out, nil, err
		}
	}
	out.Carried = c
	out.Carried.ArtifactDigest = artifact
	out.Carried.PlanFingerprint = out.Locked.PlanFingerprint
	out.Carried.Package = PackageID + "@" + PackageVersion
	out.Carried.SignerRef = key.Ref
	out.Carried.SignerKeyFingerprint = key.Fingerprint()
	var kinds []string
	for _, src := range out.Plan.Sources {
		kinds = append(kinds, src.Kind)
	}
	detail := map[string]any{
		"trustPoint":                     p.Seq,
		"ownerJudgment":                  owner,
		"signerJudgment":                 signer,
		"artifactDigestDiffersFromMeans": artifact != c.SemanticDigest,
		"planSources":                    kinds,
		"buildBridgeRead":                true,
		"signatureFormat":                "reference ed25519 (semantic-composition-preflight), sample-derived key",
		"signerKeyEpoch":                 key.Epoch,
	}
	return out, detail, nil
}
