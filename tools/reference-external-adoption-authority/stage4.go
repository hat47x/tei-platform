package externaladoptionauthority

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"tei/internal/modelref"
	"tei/internal/sourceacceptance"
	"tei/internal/sourceindex"
	"tei/internal/subject"
	"tei/internal/workspace"
	semanticcompositionpreflight "tei/reference/semantic-composition-preflight"
	semanticdistribution "tei/reference/semantic-distribution"
)

// ApplicationID is the consuming project of the sample.
const (
	ApplicationID = "board"
	ImportAlias   = "sui"
	ResourceScope = "sui-sensemaking:island-hierarchy"
)

// ExternalAcceptance is what the Host records next to the source and
// selection of the same decision: the identities and digests carried from
// stages 1 to 3, copied, never recomputed. It records the trust point of the
// decision; that point is history, not a current qualification.
type ExternalAcceptance struct {
	Alias                string `json:"alias"`
	Namespace            string `json:"namespace"`
	SemanticDigest       string `json:"semanticDigest"`
	ArtifactDigest       string `json:"artifactDigest"`
	PlanFingerprint      string `json:"planFingerprint"`
	Package              string `json:"package"`
	SignerRef            string `json:"signerRef"`
	SignerKeyFingerprint string `json:"signerKeyFingerprint"`
	NamespaceOwner       string `json:"namespaceOwner"`
	OwnerIssuer          string `json:"ownerIssuer"`
	LossRecordDigest     string `json:"lossRecordDigest"`
	Lossless             bool   `json:"lossless"`
	ParentPresence       string `json:"parentPresenceSemantics"`
	DecidedAtTrustPoint  int    `json:"decidedAtTrustPoint"`
}

// Decision is one Host decision: the source acceptance, the selected
// external namespace and the external acceptance, recorded together.
type Decision struct {
	Meaning  semanticcompositionpreflight.CurrentProjectMeaning `json:"meaning"`
	External ExternalAcceptance                                 `json:"external"`
}

// Host is a synthetic Host in the test process. It is the
// CurrentProjectMeaningReader the existing preflight reads; it is not
// product acceptance.
type Host struct {
	decisions map[string]Decision
}

func NewHost() *Host { return &Host{decisions: map[string]Decision{}} }

func (h *Host) ReadCurrentAcceptedProjectMeaning(_ context.Context, app string) (semanticcompositionpreflight.CurrentProjectMeaning, error) {
	d, ok := h.decisions[app]
	if !ok {
		return semanticcompositionpreflight.CurrentProjectMeaning{}, errors.New("no current decision")
	}
	return d.Meaning, nil
}

func requestedSelection() []semanticcompositionpreflight.ProjectMeaningRequest {
	return []semanticcompositionpreflight.ProjectMeaningRequest{{
		Alias: ImportAlias,
		Resource: modelref.Ref{Scope: ResourceScope,
			Subject: subject.Subject{Kind: "semantic-namespace", ID: "sui"}},
	}}
}

// Accept records one decision for the consumer as it is now. The source
// record and the selection come from the same Load; the selection has
// exactly one entry per import.
func (h *Host) Accept(root string, ext ExternalAcceptance, ref string) (Decision, error) {
	loaded, err := workspace.Load(root)
	if err != nil {
		return Decision{}, err
	}
	index, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		return Decision{}, err
	}
	if len(loaded.Imports) != 1 || loaded.Imports[0].Alias != ext.Alias {
		return Decision{}, fmt.Errorf("consumer must import exactly %s", ext.Alias)
	}
	if loaded.Imports[0].SemanticDigest != ext.SemanticDigest {
		return Decision{}, fmt.Errorf("%w: consumer imports %s, the distribution carries %s",
			ErrInputMismatch, loaded.Imports[0].SemanticDigest, ext.SemanticDigest)
	}
	req := requestedSelection()[0]
	d := Decision{
		Meaning: semanticcompositionpreflight.CurrentProjectMeaning{
			Source: sourceacceptance.Record{
				ApplicationID: ApplicationID, DecisionRef: ref, AuthorityRevision: "synthetic-host-1",
				ModelRevision: sourceindex.ModelRevision(index),
				Artifacts:     append([]sourceindex.Artifact(nil), index.Artifacts...),
			},
			Selected: []semanticcompositionpreflight.ProjectMeaningSelection{{
				Alias: req.Alias, Resource: req.Resource, SemanticDigest: loaded.Imports[0].SemanticDigest,
			}},
		},
		External: ext,
	}
	h.decisions[ApplicationID] = d
	return d, nil
}

// Rejudgment is the judgment of an existing decision at one trust point.
// It never reads the decision's own trust point as a qualification.
type Rejudgment struct {
	Point            int            `json:"trustPoint"`
	Label            string         `json:"trustPointLabel"`
	Meaning          string         `json:"meaning"`
	SourceOnly       string         `json:"sourceOnlyAcceptance"`
	Owner            OwnerJudgment  `json:"owner"`
	Signer           SignerJudgment `json:"signer"`
	SignerKeyCurrent bool           `json:"signerKeyIsCurrent"`
	Loss             string         `json:"loss"`
	Established      bool           `json:"established"`
}

type sourceRecordReader struct{ r sourceacceptance.Record }

func (s sourceRecordReader) ReadCurrentAcceptedSource(context.Context, string) (sourceacceptance.Record, error) {
	return s.r, nil
}

type project struct {
	Root     string
	Provider string
	Consumer string
}

// setupProject copies the provider namespace and the sample consumer.
func (s *Sample) setupProject(providerSrc, dir string) (project, error) {
	p := project{Root: dir, Provider: filepath.Join(dir, "provider"), Consumer: filepath.Join(dir, "consumer")}
	if err := copyTree(providerSrc, p.Provider); err != nil {
		return p, err
	}
	if err := copyTree(filepath.Join(s.Repo, "tools", "reference-external-adoption-authority", "testdata", "consumer"), p.Consumer); err != nil {
		return p, err
	}
	return p, nil
}

// rejudge checks the decision at trust point p: the meaning through the
// existing InspectCurrentProjectMeaning, the owner through the catalog CLI,
// the signer through InspectPublished, and the carried loss record.
func (s *Sample) rejudge(ctx context.Context, h *Host, proj project, s2 stage2Out, s3 stage3Out, lossPath string, p TrustPoint, dir string) (Rejudgment, error) {
	r := Rejudgment{Point: p.Seq, Label: p.Label}
	d := h.decisions[ApplicationID]
	verified, err := semanticcompositionpreflight.InspectCurrentProjectMeaning(ctx, h, ApplicationID, proj.Consumer, requestedSelection())
	switch {
	case err != nil:
		r.Meaning = err.Error()
	case len(verified.Acceptance.Selected) != 1 || verified.Acceptance.Selected[0].SemanticDigest != d.External.SemanticDigest:
		r.Meaning = "selection does not match the external acceptance"
	default:
		r.Meaning = "ok"
	}
	// The source-only check is recorded to show it is not the external
	// acceptance: it can pass while the provider's meaning changed.
	if loaded, err := workspace.Load(proj.Consumer); err != nil {
		r.SourceOnly = err.Error()
	} else if index, err := workspace.BuildSourceIndex(proj.Consumer, loaded); err != nil {
		r.SourceOnly = err.Error()
	} else if _, err := sourceacceptance.VerifyCurrent(ctx, sourceRecordReader{d.Meaning.Source}, ApplicationID, index); err != nil {
		r.SourceOnly = err.Error()
	} else {
		r.SourceOnly = "ok"
	}
	r.Owner, err = s.judgeOwner(ctx, s2.Inputs, s2.LockPath, p, dir)
	if err != nil {
		return r, err
	}
	r.Signer = judgeSigner(s3.Publication, p, nil)
	if key, ok := p.Key(d.External.SignerRef); ok {
		r.SignerKeyCurrent = key.Fingerprint() == d.External.SignerKeyFingerprint
	}
	if loss, _, err := readLossRecord(lossPath, d.External.LossRecordDigest); err != nil {
		r.Loss = err.Error()
	} else if loss.Lossless || d.External.Lossless || loss.ParentPresenceSemantics != d.External.ParentPresence {
		r.Loss = "loss record and acceptance disagree"
	} else {
		r.Loss = "ok"
	}
	r.Established = r.Meaning == "ok" && r.Owner.Qualified && r.Signer.Qualified && r.SignerKeyCurrent && r.Loss == "ok"
	return r, nil
}

// checkStage3Handoff is stage 4's entry check: the distribution file must
// be the locked artifact and hold the carried meaning.
func checkStage3Handoff(s3 stage3Out, c Carried) error {
	raw, err := os.ReadFile(s3.PayloadPath)
	if err != nil {
		return err
	}
	if d := sourceindex.Fingerprint(raw); d != c.ArtifactDigest {
		return fmt.Errorf("%w: distribution digest %s, carried %s", ErrInputMismatch, d, c.ArtifactDigest)
	}
	exp := s3.Expected
	if exp.ArtifactDigest != c.ArtifactDigest || exp.SemanticDigest != c.SemanticDigest ||
		exp.Package.PackageID+"@"+exp.Package.Version != c.Package || s3.Locked.PlanFingerprint != c.PlanFingerprint {
		return fmt.Errorf("%w: stage 3 expectation differs from the carried values", ErrInputMismatch)
	}
	if _, err := semanticdistribution.InspectSelectedReference(raw, exp); err != nil {
		return fmt.Errorf("%w: %v", ErrInputMismatch, err)
	}
	return nil
}

func externalFrom(c Carried, p TrustPoint) ExternalAcceptance {
	return ExternalAcceptance{
		Alias: ImportAlias, Namespace: c.Namespace, SemanticDigest: c.SemanticDigest,
		ArtifactDigest: c.ArtifactDigest, PlanFingerprint: c.PlanFingerprint, Package: c.Package,
		SignerRef: c.SignerRef, SignerKeyFingerprint: c.SignerKeyFingerprint,
		NamespaceOwner: c.NamespaceOwner, OwnerIssuer: c.OwnerIssuer,
		LossRecordDigest: c.LossRecordDigest, Lossless: *c.Lossless, ParentPresence: c.ParentPresence,
		DecidedAtTrustPoint: p.Seq,
	}
}

// runStage4 records the decision and judges it at the same trust point.
func (s *Sample) runStage4(ctx context.Context, s1 stage1Out, s2 stage2Out, s3 stage3Out, c Carried, p TrustPoint, dir string) (*Host, project, map[string]any, error) {
	if err := checkStage3Handoff(s3, c); err != nil {
		return nil, project{}, nil, err
	}
	loss, _, err := readLossRecord(s1.LossPath, c.LossRecordDigest)
	if err != nil {
		return nil, project{}, nil, err
	}
	proj, err := s.setupProject(filepath.Join(s1.InputDir, "workspace"), filepath.Join(dir, "project"))
	if err != nil {
		return nil, proj, nil, err
	}
	h := NewHost()
	ext := externalFrom(c, p)
	ext.Lossless = loss.Lossless
	decision, err := h.Accept(proj.Consumer, ext, "synthetic://external-adoption-authority/decision/1")
	if err != nil {
		return nil, proj, nil, err
	}
	r, err := s.rejudge(ctx, h, proj, s2, s3, s1.LossPath, p, dir)
	if err != nil {
		return nil, proj, nil, err
	}
	if !r.Established {
		return h, proj, map[string]any{"rejudgment": r}, fmt.Errorf("acceptance is not established at trust point %d: %+v", p.Seq, r)
	}
	fp, err := digestJSON(decision)
	if err != nil {
		return nil, proj, nil, err
	}
	detail := map[string]any{
		"trustPoint":           p.Seq,
		"decisionRef":          decision.Meaning.Source.DecisionRef,
		"decisionFingerprint":  fp,
		"selectedEntries":      len(decision.Meaning.Selected),
		"importsInConsumer":    1,
		"externalAcceptance":   decision.External,
		"rejudgment":           r,
		"hostIsSynthetic":      true,
		"acceptsDataOnlyAsset": false,
	}
	return h, proj, detail, nil
}
