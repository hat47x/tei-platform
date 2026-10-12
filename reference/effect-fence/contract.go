package effectfence

// BoundaryKind は副作用と権限の版を確定する物理的な境界を表す。
// 宣言だけで実際の構成が保証されるわけではない。
type BoundaryKind string

const (
	BoundaryMemory      BoundaryKind = "single-process"
	BoundaryTransaction BoundaryKind = "same-transaction"
	BoundaryConditional BoundaryKind = "provider-conditional"
	BoundaryIndependent BoundaryKind = "independent-provider"
)

// ProofKind は合成試験と実際の保存先を用いた証拠を区別する。
// VerifiedProvider は第三者による認証を意味しない。証拠の妥当性は別途審査が必要。
type ProofKind string

const (
	ProofUnverified ProofKind = "unverified"
	ProofSynthetic  ProofKind = "synthetic"
	ProofProvider   ProofKind = "provider-tested"
)

// Guarantee は要求が満たせる保証の上限である。
// DurableAtomic は権限版、更新、確定記録が一体で永続化される契約に限る。
type Guarantee string

const (
	GuaranteeNone    Guarantee = "unavailable"
	GuaranteeProcess Guarantee = "process-atomic"
	GuaranteeDurable Guarantee = "durable-atomic"
)

// BoundaryContract は、保存先Adapterが提示する保証可能性の申告である。
// 次の真偽値はAdapter内部の具体的な確定処理をレビューして確かめる。
// RemoteEffect はトランザクション外の送信やキュー発行を含む場合に真。
type BoundaryContract struct {
	Boundary                BoundaryKind
	AuthorityCheckedAtWrite bool
	ResourceCheckedAtWrite  bool
	ReceiptStoredWithWrite  bool
	SubjectLinksCovered     bool
	RemoteEffect            bool
	DurableReceipt          bool
	RecoveryLookup          bool
	Proof                   ProofKind
	ProofReference          string
}

// Assessment は業務結果ではない。呼出前にAdapter選択を制限するための評価結果。
type Assessment struct {
	Allowed   bool
	Available Guarantee
	Reason    string
}

// AssessBoundary は根拠不明・外部副作用・未証明の構成を安全側へ倒す。
// require=Durable は実Provider証拠を必要とする。値が未知なら許可しない。
// 申告内容自体の実地検証や偽装の排除は、この関数の責務外である。
func AssessBoundary(c BoundaryContract, require Guarantee) Assessment {
	if require != GuaranteeProcess && require != GuaranteeDurable {
		return Assessment{Reason: "invalid-requirement", Available: GuaranteeNone}
	}
	if !c.AuthorityCheckedAtWrite || !c.ResourceCheckedAtWrite || !c.ReceiptStoredWithWrite || !c.SubjectLinksCovered {
		return Assessment{Reason: "incomplete-commit-boundary", Available: GuaranteeNone}
	}
	if c.RemoteEffect {
		return Assessment{Reason: "effect-outside-boundary", Available: GuaranteeNone}
	}
	if c.ProofReference == "" || c.Proof == ProofUnverified || (c.Proof != ProofSynthetic && c.Proof != ProofProvider) {
		return Assessment{Reason: "unverified-contract", Available: GuaranteeNone}
	}
	switch c.Boundary {
	case BoundaryMemory:
		if require == GuaranteeProcess {
			return Assessment{Allowed: true, Available: GuaranteeProcess}
		}
		return Assessment{Reason: "memory-is-not-durable", Available: GuaranteeProcess}
	case BoundaryTransaction, BoundaryConditional:
		// 合成DBや単体試験だけでは、本番の永続的な原子性は確認できない。
		if c.Proof == ProofProvider && c.DurableReceipt && c.RecoveryLookup {
			return Assessment{Allowed: true, Available: GuaranteeDurable}
		}
		return Assessment{Reason: "durable-provider-proof-missing", Available: GuaranteeNone}
	default:
		return Assessment{Reason: "authority-and-effect-not-atomic", Available: GuaranteeNone}
	}
}

// MemoryStoreの契約は同一プロセス内だけの証拠であり、永続Provider保証ではない。
func (m *MemoryStore) BoundaryContract() BoundaryContract {
	return BoundaryContract{
		Boundary: BoundaryMemory, AuthorityCheckedAtWrite: true,
		ResourceCheckedAtWrite: true, ReceiptStoredWithWrite: true,
		SubjectLinksCovered: true, RemoteEffect: false,
		DurableReceipt: false, RecoveryLookup: true,
		Proof: ProofSynthetic, ProofReference: "reference/effect-fence/fence_test.go",
	}
}
