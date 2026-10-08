# internal/ — 製品Go実装

このルールは`internal/`以下のすべてのGo packageに適用する。着手前にルート`../AGENTS.md`、対象packageの既存実装・テスト、対応するIssue・ADRを読む。

## package階層の扱い

- `core/`、`subject/`等は基礎の識別・表現層。`semantic/`は業務上の意味、`syntax/`・`workspace/`は入力構文と読み込み、`ir/`は統合表現を担当する。これらをProvider固有の実行処理と無条件に混ぜない。
- `plugin/`はPluginの宣言・解決・構成、`plugin/runtime/`は実行時の接続を扱う。Provider、Capability、権限、Artifact、Lockの所有境界が曖昧な場合は既存契約を調べてから変更する。標準機能を組み込む場合も、Plugin化の方針と実際の導入段階を区別する。
- `database/`、`deployment/`、`driver/`、`realization/`等は具体的な実現・実行と外部資産との境界。Database製品に固有の挙動、観測、破壊的処理の安全条件をCanonicalな意味へ取り込まない。
- その他のpackageも名前だけで責務を確定せず、既存の呼び出し元・テスト・採用済みADRから境界を確認する。階層を動かすためだけに新しいinterfaceやpackageを作らない。

## 変更時の規則

1. 修正対象の呼出元、下流の依存先、境界テストを確認する。必要な変更を機能の所有層へ置き、`cmd/`・`tools/`・`reference/`・`examples/`のpackageを製品実装からimportしない。
2. Repositoryには`tools/repository-architecture-check/`の依存検査があり、現状`core/`・`subject/`・`permission/`・`semantic/`・`stdlib/`などに限定した規則がある。**存在しない全面的な層分離を保証済みと主張しない**。新しい禁止規則は測定と実例に基づき検討する。
3. 変更した責務のunit testと、必要に応じてReference / CLI / Providerの回帰検証を加える。mockやsynthetic fixtureの成功を外部Providerの実証へ読み替えない。
4. `main`統合前に対象exact-headのtrusted checkoutで`tools/repository-health-check.sh`を通し、clean pre/post stateとPASSを記録する。部分テストやGitHubのmergeable判定は代替不可。未実施ならcode-changing PRをDraftで保持する。

参照: `../plan/design/REPOSITORY_ARCHITECTURE_DEPENDENCY_GUARD_2026-10.md`、`../CONTRIBUTING.md`。検証の現状は該当Issue・PRのEvidenceに従う。
