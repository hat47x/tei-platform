# Branch Delete Candidates — 2026-09

この文書は、remote branchのうち、current `main`へ成果が完全に包含され、open PRやEvidence / bootstrap等のref固有用途もなく、**remote ref自体を削除してよい候補**を明示的に管理する監査記録です。

恒久的なBranch / PR運用ルールは[`../../CONTRIBUTING.md`](../../CONTRIBUTING.md)、分類基準は[`branch-hygiene-2026-09.md`](branch-hygiene-2026-09.md)、current-state監査の経緯は[`branch-current-review-2026-09.md`](branch-current-review-2026-09.md)を正とします。

この文書はProduct / Issue / Gate / Architecture / Evidenceの正本ではありません。

## 1. 削除候補の判定

`DELETE_CANDIDATE`へ入れる前に、少なくとも次を確認します。

```text
open PR headではない
        +
current mainに対して ahead_by = 0
        +
compare files = []
        +
release / Evidence / bootstrap / external integration等の
ref固有用途を持たない
        ↓
DELETE_CANDIDATE
```

`PATCH_EQUIVALENT_REVIEW`、`UNMERGED_REVIEW`、Evidence / bootstrap ref、現在進行中workstreamはここへ入れません。

一度`DELETE_CANDIDATE`として確定したbranchは、mainが前進しても新しいmainへ再同期しません。branch数を減らさないref更新を繰り返さず、削除可能になった時点でこの明示listから一度だけ削除します。

## 2. 2026-09-16 candidate pass 1

確認時点のcurrent `main`:

```text
e197e997b5e6c7116a85ff64e946e78b6a58054f
```

### 2.1 Database Contract Query signature

次の4本はすべて `ahead_by = 0 / files = []` を確認しました。

- `work/database-contract-query-signature-authoring`
- `work/database-contract-query-signature-model`
- `work/database-contract-query-signature-syntax`
- `work/database-contract-query-signature-validation`

### 2.2 Driver

次の7本もすべて `ahead_by = 0 / files = []` を確認しました。

- `work/driver-authentication-authorization-scenario`
- `work/driver-browse-query-execution`
- `work/driver-capability-binding`
- `work/driver-capability-binding-resolution`
- `work/driver-capability-invocation`
- `work/driver-issue-four-evidence-lanes-sync`
- `work/driver-pattern-resolution`

### 2.3 Query Result Mapping

次の5本もすべて `ahead_by = 0 / files = []` を確認しました。

- `work/query-result-mapping-authoring`
- `work/query-result-mapping-runtime`
- `work/query-result-mapping-runtime-current`
- `work/query-result-mapping-runtime-v2`
- `work/query-result-mapping-validation`

`-current` / `-v2`を含むbranchも、名前だけで判定せずcurrent mainとの比較を改めて行っています。

### candidate pass 1結果

```text
reviewed refs:        16
DELETE_CANDIDATE:     16
ahead_by > 0 found:    0
ref updates:            0
new cleanup branches:   0
```

## 3. 2026-09-16 candidate pass 2 — Documentation quality refs

`branch-current-review-2026-09.md`のpass 8で、文書品質・読み順改善に関する次の5本もmain包含済みと確認しました。

- `docs/document-quality-rollout`
- `docs/documentation-quality-guidance`
- `docs/improve-business-ai-reading-path`
- `docs/improve-core-reading-path`
- `docs/improve-plan-issue-reading`

5本は同じhistorical tip `b7de002a02eac2979c3fbf0fd8e5e2e05890e4c9` を指しており、確認時点のcurrent `main` `e197e997b5e6c7116a85ff64e946e78b6a58054f` に完全包含されていました。open PR headでもなく、Evidence / bootstrap等のref固有用途もありません。

このため、この5本も`DELETE_CANDIDATE`へ移します。mainが前進しても再同期しません。

### candidate pass 2結果

```text
reviewed refs:         5
DELETE_CANDIDATE:      5
ahead_by > 0 found:    0
ref updates:            0
new cleanup branches:   0
```

## 4. 2026-09-16 candidate pass 3 — Semantic Relation refs

Semantic Relation key / traversalに関する次の6本をcurrent `main`と比較しました。

- `work/semantic-relation-key-authoring`
- `work/semantic-relation-key-parser`
- `work/semantic-relation-key-syntax`
- `work/semantic-relation-key-target-uniqueness`
- `work/semantic-relation-key-validation`
- `work/semantic-relation-traversal-keys`

確認時点のcurrent `main`:

```text
201cc06859418e06c51688ee77f68da6dcf578af
```

6本はいずれも:

```text
ahead_by = 0
files = []
```

でした。open PR headではなく、Evidence / bootstrap等のref固有用途もありません。

このため6本を`DELETE_CANDIDATE`へ追加します。ref自体は動かしていません。

### candidate pass 3結果

```text
reviewed refs:         6
DELETE_CANDIDATE:      6
ahead_by > 0 found:    0
ref updates:            0
new cleanup branches:   0
```

## 5. 2026-09-16 candidate pass 4 — Application Pattern planning refs

Application Patternの計画・進捗同期・stage completionに関する次の10本をcurrent `main`と比較しました。

- `plan/application-pattern-command-current`
- `plan/application-pattern-command-sync`
- `plan/application-pattern-next-evidence`
- `plan/application-pattern-next-evidence-final`
- `plan/application-pattern-progress-final`
- `plan/application-pattern-progress-sync`
- `plan/application-pattern-stage-complete`
- `plan/application-pattern-stage-complete-followup`
- `plan/application-pattern-v1-design`
- `plan/application-pattern-v1-design-rebased`

確認時点のcurrent `main`:

```text
2c6ca29ad780705fe9026287d7d28d5a53126d8f
```

10本はいずれも:

```text
ahead_by = 0
files = []
```

でした。`-final`、`-followup`、`-rebased`等の世代名を根拠にせず、各refをcurrent mainと個別比較しています。いずれもopen PR headではなく、Evidence / bootstrap等のref固有用途もありません。

このため10本を`DELETE_CANDIDATE`へ追加します。新方針に従い、candidate確定後のref自体はcurrent mainへ再同期しません。

### candidate pass 4結果

```text
reviewed refs:         10
DELETE_CANDIDATE:      10
ahead_by > 0 found:     0
ref updates:             0
new cleanup branches:    0
```

## 6. 2026-09-16 candidate pass 5 — Gate A planning refs

Application Pattern Gate Aの到達過程・readiness・validation同期に関する次の13本をcurrent `main`と個別比較しました。

- `plan/gate-a-after-binding`
- `plan/gate-a-after-outcome`
- `plan/gate-a-after-validation`
- `plan/gate-a-completion`
- `plan/gate-a-outcome-progress`
- `plan/gate-a-readiness-audit`
- `plan/gate-a-readiness-current`
- `plan/gate-a-readiness-final`
- `plan/gate-a-readiness-sync`
- `plan/gate-a-validation-binding-refinement`
- `plan/gate-a-validation-contract-sync`
- `plan/gate-a-validation-partial`
- `plan/gate-a-validation-ready-sync`

確認時点のcurrent `main`:

```text
666efeb2c9ed1bfd884aed71729efaf4c711959f
```

13本はいずれも:

```text
ahead_by = 0
files = []
```

でした。`current` / `final` / `sync` / `partial`等の名称を現在状態の根拠にはせず、各refの固有patch有無をcurrent mainとの差分で確認しています。いずれもopen PR headではなく、Evidence / bootstrap等のref固有用途もありません。

このため13本を`DELETE_CANDIDATE`へ追加します。ref自体は動かさず、今後の通常監査から除外します。

### candidate pass 5結果

```text
reviewed refs:         13
DELETE_CANDIDATE:      13
ahead_by > 0 found:     0
ref updates:             0
new cleanup branches:    0
```

### 累計

```text
DELETE_CANDIDATE: 50
```

## 7. 現在の削除実行状態

上記50本は削除条件を満たしていますが、現在この作業で利用しているGitHub接続面にはremote branch refを削除する操作が公開されていません。

この制約を理由に、候補branchをcurrent mainへ何度もfast-forwardして代替しません。50本はこの文書上で`DELETE_CANDIDATE`としてfreezeし、以後の通常current-state監査から除外します。

```text
DELETE_CANDIDATE確定
        ↓
refを再同期しない
        ↓
通常監査から除外
        ↓
remote ref削除が実行可能になった時点で一度だけ削除
```

削除実行後は、branchが実際に存在しないことを確認したうえで、この文書に実行結果を追記します。

## 8. KEEP / FROZENとの分離

少なくとも次は、この削除候補一覧へ入れません。

- `bootstrap-source`
- `evidence/soza-e3b-p2-2026-09-02`
- `evidence/soza-e3b-p2-version-exchange-2026-09-05`
- open PR #606 head `work/postgres-partition-drop-readiness-533`
- 継続監査branch `work/repository-hygiene`

前3本はpush-triggered workflowのhistorical execution contextとしてKEEP / FROZEN、#606 headは現在進行中作業、`work/repository-hygiene`は継続workstreamです。

## 9. 次の追加方針

今後のbranch監査では、未監査の通常`work/*` / `docs/*` / `plan/*`を小集合ずつ比較し、次のどちらかへ送ります。

```text
ahead_by = 0 + ref用途なし
  -> この文書のDELETE_CANDIDATEへ追記

ahead_by > 0
  -> resetせずPATCH_EQUIVALENT_REVIEW / UNMERGED_REVIEWへ
```

すでにこの文書へ入ったbranchを、mainの前進だけを理由に再監査しません。
