# reference-external-role-dispatcher（#4218）

ISSUE-000087 / GitHub #4218のPoC実装です。`AGENTS.md`の「モデル → ロール → 作業」分離規則を、GitHub Issue/PRを用いた最小のDispatcherとして動作させます。TEI Coreに新しいAgent Runtimeを追加せず、既存の`tools/`参照ツールとして分離します。

## 確認済みの範囲

- ロール別の作業取得、モデル選択、排他的lease、失敗時のエスカレーション、PRレビュー依頼を一つのCLIで確認。
- 同一ホストでの並行claimは単一成功、期限切れleaseは`sweep`で回復可能。
- DSH adapterは`dry-run`既定で、実際の自動起動は行いません。`gh` CLIも`dry-run`で変更を回避します。

## 構成

```text
tools/reference-external-role-dispatcher/
  internal/
    config.go        # 設定スキーマ、ロール→モデル解決
    lease.go         # ローカルSQLiteによる排他claim/lease/heartbeat/sweep
    github.go        # gh CLI薄いadapter
    dsh.go           # DSH headless/CLI薄いadapter
    dispatch.go      # 取得・実行・エスカレーション・レビュー依頼の orchestration
    *_test.go        # 合成試験
  cmd/dispatcher/
    main.go          # CLI入口
    dispatch_test.go # 結合試験
  config.example.json
```

## Work Packet契約

Issue/PRにはモデル名を書きません。作業は`route:<role>`ラベルと`state:ready`/`state:working`/`state:review`/`state:blocked`で示し、実行時にモデルを解決します。実行記録には`要求ロール／実モデルID／Harness／Session／選択理由`を残します。

## 使い方

```bash
# 設定をコピー
cp tools/reference-external-role-dispatcher/config.example.json /tmp/dispatcher.json

# ビルド
go run ./tools/reference-external-role-dispatcher/cmd/dispatcher model -role implementer -config /tmp/dispatcher.json

# dry-runでclaim
go run ./tools/reference-external-role-dispatcher/cmd/dispatcher claim -role implementer -config /tmp/dispatcher.json

# 実行（失敗時はdiagnosticianへエスカレーション）
go run ./tools/reference-external-role-dispatcher/cmd/dispatcher run -role implementer -issue 1 -task "..." -config /tmp/dispatcher.json

# 期限切れleaseの掃除
go run ./tools/reference-external-role-dispatcher/cmd/dispatcher sweep -config /tmp/dispatcher.json
```

## 検証

```bash
go test ./tools/reference-external-role-dispatcher/...
```

## 制約と未検証

- 複数ホスト横断の排他保証は**主張しません**。SQLite leaseは単一ホストでのみ成立します。
- GitHubラベルのread→writeは原子的ロックとみなしません。
- DSHの実際の自動起動・通知・結果回収は**未検証**です。dry-run/手動起動の運用手順を優先します。
- Codex/Claude Code/GPT Sol/TerraがDSHから直接使用可能とは**仮定しません**。
- A4効果（秘密送信、本番配備、対外公開等）はモデル判断で承認しません。
