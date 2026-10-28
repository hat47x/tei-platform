package externaladoptionauthority

import (
	"encoding/json"
	"fmt"
)

const completionReportSchema = "tei.reference.external-adoption-authority.completion-report/v0"

// Natures of the evidence a completion report lists.
const (
	NatureExistingAsset = "existing-asset-unchanged-on-fixed-input"
	NatureSynthetic     = "synthetic"
	NatureSimulated     = "simulated"
	NatureReal          = "real-environment"
)

// EvidenceItem says how one part of the run was confirmed.
type EvidenceItem struct {
	Item   string `json:"item"`
	Nature string `json:"nature"`
	Note   string `json:"note"`
}

// CompletionReport is derived from the verified chain only. It separates
// the claim on the pinned version, the loss, how each part was confirmed
// and what the real environment confirmed (nothing).
type CompletionReport struct {
	Schema       string `json:"schema"`
	ChainHead    string `json:"chainHead"`
	FixedVersion struct {
		Repository      string            `json:"repository"`
		Revision        string            `json:"revision"`
		Blobs           map[string]string `json:"blobs"`
		Claim           string            `json:"claim"`
		ComparedWith    string            `json:"comparedWith"`
		FixtureVerdicts map[string]bool   `json:"fixtureVerdicts"`
	} `json:"fixedVersionConformance"`
	CurrentExternalApplication struct {
		Conformance string `json:"conformance"`
		Reason      string `json:"reason"`
	} `json:"currentExternalApplication"`
	Loss struct {
		Lossless         bool     `json:"lossless"`
		ParentPresence   string   `json:"parentPresenceSemantics"`
		Unresolved       []string `json:"unresolved"`
		NotExtended      []string `json:"notExtendedByStage2"`
		LossRecordDigest string   `json:"lossRecordDigest"`
	} `json:"loss"`
	Identity        Carried        `json:"identity"`
	Evidence        []EvidenceItem `json:"evidence"`
	RealEnvironment struct {
		Confirmed bool     `json:"confirmed"`
		Items     []string `json:"items"`
		Note      string   `json:"note"`
	} `json:"realEnvironment"`
	AdoptionEstablished bool `json:"adoptionEstablished"`
}

func stageDetail(c Chain, stage int) map[string]json.RawMessage {
	for _, r := range c.Records {
		if r.Stage == stage {
			var m map[string]json.RawMessage
			_ = json.Unmarshal(r.Detail, &m)
			return m
		}
	}
	return nil
}

// BuildCompletionReport refuses an incomplete chain.
func BuildCompletionReport(c Chain) (CompletionReport, error) {
	var rep CompletionReport
	th := VerifyThrough(c)
	if !th.Complete {
		return rep, fmt.Errorf("chain is not complete: missing %v, mismatches %v", th.MissingStages, th.Mismatches)
	}
	last := c.Records[len(c.Records)-1].Carried
	d1, d2 := stageDetail(c, 1), stageDetail(c, 2)
	rep.Schema = completionReportSchema
	rep.ChainHead = th.Head
	rep.FixedVersion.Repository = last.Source.Repository
	rep.FixedVersion.Revision = last.Source.Revision
	rep.FixedVersion.Blobs = last.Source.Blobs
	rep.FixedVersion.Claim = "pinned-version-and-dependency-blobs-only"
	rep.FixedVersion.ComparedWith = "fixed copy of the pinned table in reference/sui-adoption/README.md; not the SUI repository"
	_ = json.Unmarshal(d1["fixtureValid"], &rep.FixedVersion.FixtureVerdicts)
	rep.CurrentExternalApplication.Conformance = "not-claimed"
	rep.CurrentExternalApplication.Reason = "the run never read the current SUI repository; only the pinned blobs were compared with a fixed copy"
	rep.Loss.Lossless = *last.Lossless
	rep.Loss.ParentPresence = last.ParentPresence
	_ = json.Unmarshal(d1["unresolved"], &rep.Loss.Unresolved)
	_ = json.Unmarshal(d2["coverageNotExtended"], &rep.Loss.NotExtended)
	rep.Loss.LossRecordDigest = last.LossRecordDigest
	rep.Identity = last
	rep.Evidence = []EvidenceItem{
		{"stage 1 shadow probe on the four fixtures", NatureExistingAsset, "tools/reference-sui-adoption built and run unchanged on a copy of reference/sui-adoption"},
		{"external dependency blobs", NatureSimulated, "the external side is a copy of the pinned table; a change is simulated by editing the copy"},
		{"stage 2 semantic lock, validation plan and owner judgment", NatureExistingAsset, "tools/reference-semantic-extension-catalog built and run unchanged on its own testdata descriptors"},
		{"conformance evidence, namespace authority statement, trust grants, trust points", NatureSynthetic, "written by the sample; no real publisher or Host trust store"},
		{"stage 3 distribution, build lock and signed preflight", NatureExistingAsset, "reference/semantic-distribution, semantic-build-bridge and semantic-composition-preflight called unchanged"},
		{"publication signature and signer keys", NatureSynthetic, "reference ed25519 format with keys derived from fixed seeds in the sample"},
		{"stage 4 acceptance decision", NatureSynthetic, "a Host in the test process; not product acceptance"},
		{"logical time", NatureSimulated, "trust points and expiry use fixed logical times; the wall clock is not read"},
	}
	rep.RealEnvironment.Confirmed = false
	rep.RealEnvironment.Items = []string{}
	rep.RealEnvironment.Note = "no real SUI repository, publisher, Host trust store or product acceptance was used"
	rep.AdoptionEstablished = false
	return rep, nil
}

// VerifyCompletionReport compares a report with the chain. It returns the
// mismatches; none means the report says what the chain shows and claims
// nothing more.
func VerifyCompletionReport(c Chain, rep CompletionReport) []string {
	var out []string
	add := func(format string, args ...any) { out = append(out, fmt.Sprintf(format, args...)) }
	th := VerifyThrough(c)
	if !th.Complete {
		add("chain is not complete")
		return out
	}
	last := c.Records[len(c.Records)-1].Carried
	if rep.Schema != completionReportSchema {
		add("schema %q", rep.Schema)
	}
	if rep.ChainHead != th.Head {
		add("report names chain head %s, chain head is %s", rep.ChainHead, th.Head)
	}
	if !(PinnedSource{Repository: rep.FixedVersion.Repository, Revision: rep.FixedVersion.Revision, Blobs: rep.FixedVersion.Blobs}).equal(last.Source) {
		add("fixed version differs from the chain")
	}
	if rep.FixedVersion.Claim != "pinned-version-and-dependency-blobs-only" {
		add("fixed version claim %q", rep.FixedVersion.Claim)
	}
	if rep.CurrentExternalApplication.Conformance != "not-claimed" {
		add("report claims %q for the current external application", rep.CurrentExternalApplication.Conformance)
	}
	if rep.Loss.Lossless {
		add("report reads lossless as true")
	}
	if rep.Loss.Lossless != *last.Lossless || rep.Loss.ParentPresence != last.ParentPresence || rep.Loss.LossRecordDigest != last.LossRecordDigest {
		add("loss differs from the chain")
	}
	if !contains(rep.Loss.Unresolved, "parent-presence") {
		add("report does not list parent-presence as unresolved")
	}
	a, _ := digestJSON(rep.Identity)
	b, _ := digestJSON(last)
	if a != b {
		add("identity differs from stage 4 of the chain")
	}
	natures := map[string]bool{}
	for _, e := range rep.Evidence {
		natures[e.Nature] = true
	}
	for _, n := range []string{NatureExistingAsset, NatureSynthetic, NatureSimulated} {
		if !natures[n] {
			add("evidence has no %s item", n)
		}
	}
	if natures[NatureReal] || rep.RealEnvironment.Confirmed || len(rep.RealEnvironment.Items) > 0 {
		add("report claims real-environment confirmation")
	}
	if rep.AdoptionEstablished {
		add("report claims adoption is established")
	}
	return out
}

// Gap is one platform gap the run observed.
type Gap struct {
	Requirement string `json:"requirement"`
	Area        string `json:"area"`
	Gap         string `json:"gap"`
	SampleSide  string `json:"sampleSide"`
}

// Gaps are the platform gaps the run observed.
func Gaps() []Gap {
	return []Gap{
		{"P-GOV-01", "ツール", "tools/reference-sui-adoptionは、SUIの版と依存blob4件を定数として報告へ書くだけで、外部の実物とも固定コピーとも照合しない",
			"サンプル側(CheckExternalPin)で、報告の固定値をreference/sui-adoption/README.mdの固定表の複製と照合し、blobが1件でも違えば工程1を止める。SUIの実物は取得していない"},
		{"固有", "コア", "TEIの関係は一対一と一対多だけで、親の不在の意味を表せない。損失を後続へ運ぶ型がない。配布物の封筒(Envelope)は未知の項目を拒否するため、損失の記録を載せられない",
			"工程1の報告から損失の記録(JSON)を作り、そのダイジェストを各工程の記録と構築計画の入力に載せる。損失の記録は作り直さず、バイト列とダイジェストを照合する"},
		{"固有", "プラグイン", "拡張カタログの意味の固定記録(semantic lock)と、配布側の成果物の固定記録(BuildArtifactLock)を結ぶ形がない",
			"構築計画の入力(BuildSource)に、意味の固定記録・写像報告・損失の記録・固定版のダイジェストを並べ、計画の照合値を署名の対象に含める。正式な接続ではない"},
		{"固有", "実行時", "InspectCurrentProjectMeaningはパス取込みだけを受理照合し、データのみの配布物を受理しない。外部意味の受理を製品のホストが記録する手段がない",
			"利用側はパス取込みで提供元の複製を読み、その意味の照合値が配布物の意味の照合値と同じことをサンプルが照合する。受理は試験プロセス内の合成ホスト"},
		{"P-OPS-02", "プラグイン", "署名つき配布の前検査はPublisherTrustLookupという抽象の入口だけで、名前空間所有者の判定(authoritytrust.Verify)とは別の判定。信頼付与の運用・失効の伝播がない",
			"サンプルのLookupが、署名者の鍵の世代ごとにauthoritytrust.Verifyで判定する(認可の種別だけ別)。信頼の状態は論理的な時点の列で、時点ごとに判定し直す"},
		{"固有", "コア", "拡張カタログのCLIは評価(evaluateSemanticConstraintWithPlan)を公開しない。評価器の成果物のダイジェストに当たる実物がない",
			"工程2は共有パッケージ(hierarchy、acyclicgraph)を直接呼び、工程1の判定と一致することを確かめる。評価器の束縛のダイジェストはacyclicgraph/graph.goのダイジェストで代用する"},
		{"固有", "検証根拠", "第二の外部アプリで同じ欠落が繰り返されるかを比べる入力がない",
			"この通しの検証では比べない(範囲外)"},
	}
}

// Limits states what the run is not evidence for.
func Limits() []string {
	return []string{
		"conformance to the current SUI application: only the pinned revision and blobs were compared, with a fixed copy, never with the SUI repository",
		"a real publisher, signer or Host trust store: grants, keys and trust points are synthetic and in the test process",
		"product acceptance: the decision is a synthetic Host; InspectCurrentProjectMeaning only covers path imports",
		"revocation propagation across servers or processes: trust points are one in-memory list judged at explicit logical points",
		"a production distribution format, signature format or BuildOutputKind: all are the reference formats of the existing assets",
		"lossless mapping: the parent presence semantics stays unresolved and lossless=false throughout",
	}
}
