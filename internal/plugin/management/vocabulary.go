package management

// SchemaVersion identifies the inventory report.
const SchemaVersion = "tei.plugin-management.inventory/v0"

// Scan states. A scan that did not finish is never reported as complete.
const (
	ScanComplete = "complete"
	ScanPartial  = "partial"
	ScanNotRun   = "not-run"
)

// notModelled names one fact TEI has no source for today. A row carries the
// facts it cannot answer instead of encoding them as false.
const (
	NotModelledInstall            = "install"
	NotModelledEnable             = "enable"
	NotModelledInitFailure        = "init-failure"
	NotModelledOptionalDependency = "optional-dependency"
	NotModelledRetainedData       = "retained-data"
	NotModelledInvocation         = "invocation"
	NotModelledEffect             = "effect"
	NotModelledSettingsVersion    = "settings-version"
)

// allNotModelled is the closed set every report names, so a consumer learns the
// gaps once rather than per row.
var allNotModelled = []string{
	NotModelledInstall,
	NotModelledEnable,
	NotModelledInitFailure,
	NotModelledOptionalDependency,
	NotModelledRetainedData,
	NotModelledInvocation,
	NotModelledEffect,
	NotModelledSettingsVersion,
}

// Reason codes of the management vocabulary. They are distinct from the
// PLUGIN_RESOLVE_* codes the resolver emits: reasons.go maps one to the other,
// and both codes are reported, so two vocabularies never both claim to be the
// reason.
const (
	ReasonPluginNotDeclared         = "plugin-not-declared"
	ReasonEnvNotSelected            = "env-not-selected"
	ReasonDeclaredButUnresolvable   = "declared-but-unresolvable"
	ReasonVersionUnsupported        = "version-unsupported"
	ReasonDependencyCycle           = "dependency-cycle"
	ReasonDependencyUnresolved      = "dependency-unresolved"
	ReasonContractIncompatible      = "contract-incompatible"
	ReasonResolverRefused           = "resolver-refused"
	ReasonPermissionMissing         = "permission-missing"
	ReasonPermissionUnknown         = "permission-unknown"
	ReasonGrantUnverified           = "grant-unverified"
	ReasonUnsupportedTarget         = "unsupported-target"
	ReasonRuntimeUnverified         = "runtime-unverified"
	ReasonNotObserved               = "not-observed"
	ReasonScanIncomplete            = "scan-incomplete"
	ReasonScanNotRun                = "scan-not-run"
	ReasonLockNotProvided           = "lock-not-provided"
	ReasonSourceFingerprintMismatch = "source-fingerprint-mismatch"
)

// messages maps each stable reason code to display text. Changing the text
// never changes the code.
func messages() map[string]string {
	return map[string]string{
		ReasonPluginNotDeclared:         "プラグインが宣言されていません。",
		ReasonEnvNotSelected:            "この環境で選択されていません。",
		ReasonDeclaredButUnresolvable:   "宣言はありますが解決できません。",
		ReasonVersionUnsupported:        "要求された互換版に合う宣言がありません。",
		ReasonDependencyCycle:           "依存関係に循環があります。",
		ReasonDependencyUnresolved:      "依存先が宣言されていません。",
		ReasonContractIncompatible:      "依存の互換版が解決済みの版と衝突します。",
		ReasonResolverRefused:           "解決が拒否されました。",
		ReasonPermissionMissing:         "必要な権限が許可されていません。",
		ReasonPermissionUnknown:         "権限の許可状態を確認できていません。",
		ReasonGrantUnverified:           "権限の不足は lock がある場合だけ確定します。",
		ReasonUnsupportedTarget:         "実行環境の種別に対応していません。",
		ReasonRuntimeUnverified:         "実行境界は lock がある場合だけ確定します。",
		ReasonNotObserved:               "実行の観測はこの一覧の範囲外です。",
		ReasonScanIncomplete:            "走査が完了していません。",
		ReasonScanNotRun:                "走査はまだ実行されていません。",
		ReasonLockNotProvided:           "Plugin Lock が与えられていません。",
		ReasonSourceFingerprintMismatch: "Source の指紋が lock と一致しません。",
	}
}
