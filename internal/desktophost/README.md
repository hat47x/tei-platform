# Desktop Host Draft Marker Store（#1529 / Reference）

このpackageは、Desktop Studioの再起動前後に**未受理の状態を識別し直す**ためのHost側Referenceである。Go標準ライブラリの`os.Root`を使う。**Accepted Source、Source Authority、Shared Acceptanceの実装ではない**。

## 状態と操作

1. HostがユーザーとOSの許可を確認し、Workspaceとは独立した**privateな既存の絶対パス**（Unixでは0700相当、WindowsではHost自身がACLを確認）を用意する。
2. Hostは`OpenDraftMarkerStore(privateDir)`で専用ディレクトリを開く。WebViewやCandidateの指定したパスから開いてはならない。
3. `ObserveSource(workspace)`で管理対象SourceとManifestの観測を作り、`DraftMarker`に`CandidateRef`・`BaseModelRevision`・観測を保持する。
4. 保存が許されるときだけ`Save(marker)`する。保存対象は**Draft参照情報のみ**であり、候補本文やSecret、Permission、Resource Lease、外部Command、Previewで確定した権限は含まない。
5. 新しいHost processでprivate directoryを再度開き、`Recover(key, authorizedWorkspaceRoot, ObserveSource)`する。結果は`needs-review`、`source-changed`、`source-unavailable`のいずれか。**自動Acceptしない**。
6. 明示的なReview・Authorization・Acceptanceは別の共通Operationで行う。前回のReviewや指紋だけから受理権限を推測しない。

## 安全性上の境界と未実証

- 保存領域は、呼出し側が用意する必要がある。Hostが`privateDir`を正当化できなければこのAPIを呼び出さない。OS別のACL／Keychain、アプリ署名、ディスク暗号化、完全な機微Draft保存は未実装。
- `Save`は同じWorkspace・CandidateRefについて**create-only**。二つのHostウィンドウが同じMarkerを保存しても、既存の内容を上書きしない。更新したい場合は新しいCandidateRefを使う。保持期限・削除・ゴミ収集・暗号化は後続Host責務。
- `os.Root`はNative Hostのパス境界に利用できるが、JS/WASMのTOCTOU保証は異なる。Unixの権限ビット確認だけで所有者・同一UIDプロセスからの攻撃・WindowsのACLを証明できない。
- 作成中の一時ファイルを`Sync`して、同一ディレクトリ内のHard Linkで不可変ファイルとして公開する。Hard Link不可のファイルシステムでは安全に失敗させる。完全な停電時耐久性、ディレクトリエントリの永続化、複数OSでのatomic publishは未確認。
- `Baseline.WorkspaceRoot`、`Artifacts[*].Path`、fingerprintは機密情報になり得る。Host側の保存許可と外部送出の許可を別々に判断し、AI/WebViewテレメトリへ無条件送出しない（#1178）。
- Readerの`EvalSymlinks`→`Open`はレースフリーではない（#1017）。Hostの保管領域を限定しても既存Workspace Loaderの安全性を代替しない。
- `SourceObservation`は**複数時点の観測**であり、共有Acceptanceのモデルrevision／権限／Sourceファイル更新を原子的に拘束するものではない（#267 / #1158）。
- Desktop Studio成功を生成業務Desktop Target（#259）やWindows/macOS/Linux Wails実機検証の代わりに数えない。

## テスト

`go test ./internal/desktophost` は、本packageのread-only監視と保管のReferenceを検査する。`tools/reference-shared-acceptance/desktop_host_handoff_test.go` は別途Shared Acceptance本体との一巡をscratch Workspace上で検査する。**テストハーネス上のAcceptanceを人間の承認と同一視しない**。

設計候補: PR #1528 `plan/design/DESKTOP_STUDIO_HOST_INTEGRATION_DESIGN.md`。
