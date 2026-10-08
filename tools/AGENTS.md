# tools/ — 検証・参照実行・開発補助

このルールは`tools/`以下のコマンド、Go package、shell scriptに適用する。ルート`../AGENTS.md`と`../CONTRIBUTING.md`を先に読む。

## 階層別の責務

- `repository-health-check.sh`は、コード変更を`main`へ統合するためのtrusted local全体検証の入口。`--fast`や一部の検査だけをfull PASSと表現しない。
- `repository-architecture-check/`、`repository-dependency-report/`、`repository-plan-check/`等はRepositoryの特定の制約や依存関係を確認する。解析結果は検査対象・限界のあるEvidenceであり、Canonicalモデルそのものではない。
- `reference-*/`、`sei-resume-reference/`などは指定シナリオの再現・Evidence取得・検査手順。検証用コードを`internal/`のproduction依存へ流し込まない。
- `internal/`（tools配下）はtool内部の共有補助。製品用の`../internal/`とは別の責務であり、共通化という理由だけで製品の型・権限をtoolへ移さない。

## 変更時の規則

1. 入力・出力・副作用・停止条件・対象revisionを明示する。read-onlyのつもりのcheckerがWorkspaceやgoldenを変更しないか確認する。
2. PASS / FAIL、未実行、環境不足、fixtureの制限を区別する。actual Provider / 外部資産に触れる検証は資格情報と副作用、隔離環境を確認する。
3. Gate用の検査を弱める変更やスキップを導入する場合は既存Issue・ADR・正本Gateとの整合を先に確認する。GitHub Actionsが利用できないことをskipの根拠にしない。
4. Go・shell等のcode変更にはルートのexact-head Repository Health Gateを適用する。CI設定の状態、検証可能な手元環境、必要な操作権限を混同しない。
