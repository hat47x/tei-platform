# reference-durable-run-golden

Reference Durable Runを、保存済みfingerprintの手編集ではなく、`tei deployment-start`と同じinternal API経路から再生成・照合する専用toolです。

GitHub Issue #255のうち、次だけを担当します。

```text
Reference Workspace
  + fixed observed deployment state
  + Reference Plugin Lock
  + Solution Release / run identity
        ↓
releaseplan.CompileWithDeployment
        ↓
Plugin Lock / Runtime Policy fingerprint
        ↓
deploymentdurable.StartWithRuntimePolicy
        ↓
reference/order_rollout.durable-run.m0.21.start.json
```

## 役割分担

`tools/reference-golden`とは責務を分けます。

- `reference-golden`
  - `reference/world.json`
  - `reference/workspace/.tei/plugin-lock.json`
  - 保存Durable RunのPlugin Lock依存チェック
- `reference-durable-run-golden`
  - Durable Run全体の正規生成
  - 保存Durable Runとの完全一致check
  - 明示`-write`時の安全な全体置換

`reference/observed/production.deployment-state.json`はfixed observed fixtureです。このtoolから更新しません。

## 既定identity

第一Referenceの既定値は現在次です。

```text
workspace:   reference/workspace
solution:    order_rollout
version:     2026.8.27
deployment:  reference/observed/production.deployment-state.json
run-id:      order-rollout-prod-m0.21
plugin-lock: reference/workspace/.tei/plugin-lock.json
durable-run: reference/order_rollout.durable-run.m0.21.start.json
```

必要ならflagで明示できますが、物理情報やidentityをpath名から推測しません。

## check

Repositoryを変更せず、現在の保存Durable Runが正規生成結果と完全一致するか確認します。

```bash
go run ./tools/reference-durable-run-golden
```

内部では次を順に確認します。

1. Reference WorkspaceをLoad
2. Canonical Worldを通常Validation
3. Solution ReleaseをID + versionで完全一致解決
4. fixed observed deployment stateをLoad
5. Deployment Validation
6. `CompileWithDeployment`
7. Reference Plugin LockをLoad
8. observed EnvironmentをCanonical Worldから解決
9. `pluginruntime.BuildPlan`からPlugin Lock / Runtime Policy fingerprintを取得
10. `StartWithRuntimePolicy`でDurable Run全体を生成
11. 保存goldenとbyte単位で比較

Validation ErrorやPlugin Lock / Runtime Policy不整合があれば、保存fileへ触れず停止します。

## write

差分が現在のCanonical authoring / Plugin Lock / fixed observed fixtureから説明できると確認した場合だけ更新します。

```bash
go run ./tools/reference-durable-run-golden -write
```

保存先を直接truncateしてから生成することはありません。

同じdirectoryに一時fileを作り、

1. 全JSONを書き込む
2. `Sync`
3. `Close`
4. `Rename`

の順で置換します。生成途中の失敗で既存Durable Runを空fileにしません。

shellの`mktemp` / `mv`に依存しないため、Goが動作する環境ではOSごとに別手順を用意しません。

更新後は必ず差分を確認します。

```bash
git diff -- reference/order_rollout.durable-run.m0.21.start.json
```

## #255での推奨順序

まずWorld / Plugin Lock側を確認します。

```bash
go run ./tools/reference-golden
```

World / Plugin Lockがstaleなら、意味上の差分を確認したうえで次を実行します。

```bash
go run ./tools/reference-golden -write

git diff -- \
  reference/world.json \
  reference/workspace/.tei/plugin-lock.json
```

その後Durable Runをcheckします。

```bash
go run ./tools/reference-durable-run-golden
```

staleなら正規生成します。

```bash
go run ./tools/reference-durable-run-golden -write

git diff -- reference/order_rollout.durable-run.m0.21.start.json
```

最後に依存全体を再確認します。

```bash
go run ./tools/reference-golden

go run ./tools/reference-durable-run-golden
```

## 非目標

- observed deployment fixtureの自動更新
- Plugin Lock fingerprintだけの手編集
- Runtime Policy fingerprintだけの手編集
- `deployment-start`と異なる第二のDurable Run意味を作ること
- #308 real PostgreSQL engine Evidence
- #214 actual Oracle Evidence
- GitHub Actionsの再有効化

## 実行Evidence

このtoolがRepositoryに存在するだけでは#255完了にはなりません。

通常Go runner上でcheck / 必要なwrite / diff確認を実行し、最終的に保存artifactが正規生成結果と一致したEvidenceを取得して#255 / #273へ対応付けます。

## 関連

- GitHub Issue #255
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `tools/reference-golden/`
- `cmd/tei deployment-start`
