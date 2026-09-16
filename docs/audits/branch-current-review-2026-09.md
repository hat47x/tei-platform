# Current Branch Review — 2026-09

この文書は、remote branchをcurrent `main`に対して再分類し、`UNMERGED_REVIEW` / `PATCH_EQUIVALENT_REVIEW` / ref用途保護対象を少数へ絞り込むための継続監査記録です。

恒久的なBranch / PR運用ルールは[`../../CONTRIBUTING.md`](../../CONTRIBUTING.md)と[`../../AGENTS.md`](../../AGENTS.md)、分類基準は[`branch-hygiene-2026-09.md`](branch-hygiene-2026-09.md)、実行済みnormalizationの履歴は[`branch-normalization-2026-09-14.md`](branch-normalization-2026-09-14.md)を正とします。

この文書自体はIssue state、Gate state、Architecture、Evidenceの正本ではありません。

## 1. 目的

過去のremote branchには、すでに`main`へ含まれたstale refと、固有patchを持つbranchと、ref名そのものにEvidence / provenance上の意味があり得るbranchが混在しています。

したがって、branch名や更新日時だけで一括reset / deletionせず、次の順に確認します。

```text
open PR headか
  ↓ no
current mainに対してahead_by > 0か
  ↓ no
ref自体にEvidence / bootstrap / external integration用途があり得るか
  ↓ no
normalization候補
```

`ahead_by > 0`の場合は、その時点でresetせず、`PATCH_EQUIVALENT_REVIEW`と`UNMERGED_REVIEW`の追加判定へ進めます。

## 2. 2026-09-16 current-state pass 1

開始時のcurrent `main`は途中で並行workstreamにより前進し、最終確認時点では次でした。

```text
321d7d916a98320efd7c430c9d82aa0885b4e135
```

open PRは#606のみで、そのhead `work/postgres-partition-drop-readiness-533`は作業中branchとして保護し、今回の対象から除外しました。

### 2.1 transient / temporary ref

次の5本をcurrent `main`と比較し、すべて`ahead_by = 0`を確認しました。

- `tmp/noop-check`
- `tmp-query-result-world`
- `tmp-validation-world-check`
- `noop`
- `noop-should-not-create`

これらは現在のbranch naming policyでは新規作成しない種類のrefであり、open PRもありません。固有commitを持たないことを確認したため、forceを使わずcurrent `main`へfast-forward normalizationしました。

### 2.2 historical integration / audit ref

次の3本も`ahead_by = 0`を確認しました。

- `integrate/open-stack-history-20260903`
- `integrate/scattered-work-20260903`
- `audit/issue-38-postgres-catalog-normalization`

いずれもcurrent `main`のancestorであり、open PRはありません。branch名を正本として扱わない現行ruleに従い、forceを使わずcurrent `main`へfast-forward normalizationしました。

### 2.3 ref identityを保持すると確定したbootstrap / Evidence branch

次の3本も`ahead_by = 0`ですが、**normalization / deletion対象にはしません**。

- `bootstrap-source`
- `evidence/soza-e3b-p2-2026-09-02`
- `evidence/soza-e3b-p2-version-exchange-2026-09-05`

参照元を再確認した結果、これらは通常のmerged work branchではなく、branch ref自体がpush-triggered workflowのhistorical execution contextに含まれています。

#### `bootstrap-source`

current tip:

```text
95f2f0d077c0e9251e12352e4e54b9b76f368e25
Push normalized source without workflow mutation
```

`.github/workflows/normalize-source.yml`は現在も:

```yaml
on:
  push:
    branches:
      - bootstrap-source
```

を持ちます。

このrefでは少なくともrun `33290738775`が:

```text
workflow: Normalize TEI M0.22.1 source
event: push
head_branch: bootstrap-source
head_sha: 95f2f0d077c0e9251e12352e4e54b9b76f368e25
conclusion: success
```

として保存されています。

したがって、current `main`へrefを動かすことは単なるancestry整理ではなく、新しい`push` event / normalization executionを発生させ得ます。過去runの`head_branch` / `head_sha`と現在refの対応関係も変えるため、通常のstale work branchと同じnormalization規則を適用しません。

#### `evidence/soza-e3b-p2-2026-09-02`

branch上のEvidence workflow自身が:

```yaml
on:
  push:
    branches:
      - evidence/soza-e3b-p2-2026-09-02
```

を持ちます。

Actions履歴には同branch名を`head_branch`として持つSOZA E3b-P2 Evidence runが保存されており、run `33589084290`はsuccessです。後続runも同じbranch identityで保存されています。

Evidenceの正本はbranch名だけではなくexact run / `head_sha` / artifact / outputですが、branch refを後からcurrent mainへ移動する積極的理由もありません。移動自体が新しいpush-triggered Evidence executionを作り得るため、historical Evidence refとしてfreezeします。

#### `evidence/soza-e3b-p2-version-exchange-2026-09-05`

current mainの`.github/workflows/soza-e3b-p2-evidence.yml`は現在も:

```yaml
on:
  push:
    branches:
      - evidence/soza-e3b-p2-version-exchange-2026-09-05
  workflow_dispatch:
```

を持ちます。

Actions履歴にはrun `33913500170`、`34112936976`等がこのbranch identityで保存されています。これらはrunner allocation問題を含むversion-exchange / runner-image再観測の履歴でもあります。

したがって、このrefもhistorical Evidence execution contextとしてfreezeします。

#### 判定

```text
commit ancestry safe
  !=
ref identity disposable

push-triggered bootstrap / Evidence ref
  -> KEEP / FROZEN
  -> no normalization
  -> no deletion
```

この3本は今後の通常branch cleanup対象から除外し、変更が必要な場合はbootstrap / Evidence experimentとして明示的に扱います。

## 3. PR #578 conflict 8本のidentity再調査

PR #578の保存Evidenceを再確認しました。

確認対象:

- PR本文
- Issue / PR conversation comment
- PR commit list
- 現在のbranch hygiene audit

PR本文には、当時528 remote branchを調査し、15本にmain未包含commitがあり、7本をclean integrationし、8本はconflictのため別reviewとした事実が残っています。一方、8本のbranch名は記録されていません。

Issue / PR conversation commentにもidentityを補う記録はなく、PR commit listはclean integrationした7本側の履歴を追えるだけで、mergeに失敗した8本のbranch名を復元する根拠にはなりませんでした。

したがって、現時点では「おそらくこの8本」という推定を監査Evidenceへ昇格させません。

## 4. pass 1の結果

```text
reviewed refs:       11
normalized:           8
KEEP / FROZEN:        3
ahead_by > 0 found:   0
force updates:        0
new cleanup branches: 0
```

これはRepository全branchのcurrent再分類完了を意味しません。

pass 1の価値は、一時・integration・audit系の明確な集合で固有patch残存がないことを確認し、同時にbootstrap / Evidence系を「ancestryだけでは動かさない」から、**push-triggered historical execution contextなので保持する**という確定分類へ進めたことです。

## 5. 2026-09-16 current-state pass 2 — Lane C stale sync refs

既存`work/repository-hygiene`をcurrent `main`へ戻したうえで、Lane Cの古い同期branchから、同一目的の名前違い・世代違いが目立つ小集合を確認しました。

対象:

- `work/lane-c-issue-inventory-sync`
- `work/lane-c-issue-inventory-sync-2`
- `work/lane-c-issue-inventory-sync-final`
- `work/lane-c-two-blocker-ownership-sync`
- `work/lane-c-two-blocker-ownership-sync-v2`

確認時点のcurrent `main`:

```text
7ae67a9d0bc77220bb5f3fa57984695b37aed8f4
```

5本はいずれもcurrent `main`との比較で:

```text
ahead_by = 0
behind_by = 11
files = []
```

でした。open PR headは#606のみであり、この5本はopen PR headではありません。Evidence / bootstrap用refでもありません。

したがって、固有未統合作業を持たないstale work refとして、**forceを使わず**current `main`へfast-forward normalizationしました。

この操作で、特に次のような名称世代だけが残る状態を解消しています。

```text
issue-inventory-sync
  -> issue-inventory-sync-2
  -> issue-inventory-sync-final

two-blocker-ownership-sync
  -> two-blocker-ownership-sync-v2
```

branch名自体を削除したわけではありませんが、5本とも作業固有tipを持たずcurrent main同値となるため、過去の同期branchを「未統合作業候補」と誤認しにくくなります。

### pass 2結果

```text
reviewed refs:       5
normalized:          5
ahead_by > 0 found:  0
force updates:       0
new cleanup branches: 0
```

## 6. 2026-09-16 current-state pass 3 — Lane C Gate A / runbook refs

次に、Lane CのGate A実行準備・runbook・verification stateに関する旧work refを小集合で確認しました。

対象:

- `work/lane-c-gate-a-focused-test-manifest`
- `work/lane-c-gate-a-semantic-guard-manifest`
- `work/lane-c-gate-a-authority-transition-manifest`
- `work/lane-c-gate-a-post-merge-runbook`
- `work/lane-c-verification-state-depin`

確認時点のcurrent `main`:

```text
83f24fe563aac79c497d7a1260ba28e2e543e815
```

比較結果はすべて:

```text
ahead_by = 0
files = []
```

で、behindは142〜211 commitでした。いずれも現在のopen PR headではなく、bootstrap / Evidence refでもありません。

このため、5本すべてを**forceなし**でcurrent `main`へfast-forward normalizationしました。

これらは過去にGate A exact manifest、semantic guard、authority transition、post-merge runbook、verification-state depinをそれぞれ完結させたbranchですが、その成果はすでにmainへ含まれており、旧tipを作業branchとして保持する必要はありません。

### pass 3結果

```text
reviewed refs:       5
normalized:          5
ahead_by > 0 found:  0
force updates:       0
new cleanup branches: 0
```

## 7. 2026-09-16 current-state pass 4 — Lane C Handoff identity / order refs

Lane CでHandoff relationのidentity・condition distinction・normalized order invariance・Pre-ADR反映・Gate A triageを順に完結させた旧work refを確認しました。

対象:

- `work/lane-c-handoff-endpoint-condition-order-pressure`
- `work/lane-c-handoff-normalized-condition-distinction-pressure`
- `work/lane-c-normalized-handoff-order-invariance-pressure`
- `work/lane-c-pre-adr-handoff-order-invariants`
- `work/lane-c-gate-a-normalized-handoff-identity-manifest`

確認時点のcurrent `main`:

```text
894417a52485d758e7516ea8b9f2eaaf5f8516e6
```

5本はいずれも比較時点で:

```text
ahead_by = 0
files = []
```

で、42〜142 commit behindでした。いずれもopen PR headではなく、bootstrap / Evidence用途のrefでもありません。

これらの成果はすでにmainへ取り込まれており、旧tipを独立した作業候補として残す意味はありません。このため5本とも**forceを使わず**current `main`へfast-forward normalizationしました。

今回のnormalizationはHandoff semanticsを変更するものではありません。既存の:

```text
exact same from/to/condition
  -> duplicate authorityとしてreject

same endpoints + different condition
  -> distinct H1/R3 meaningとして保持

same meaning set
  -> normalized resultはinput order invariant
```

というmain上の設計・Pressureを保持したまま、過去work refのtipだけをcurrent mainへ揃えています。

### pass 4結果

```text
reviewed refs:       5
normalized:          5
ahead_by > 0 found:  0
force updates:       0
new cleanup branches: 0
```

## 8. 2026-09-16 current-state pass 5 — Application Driver / Application Pattern refs

通常の`work/*` branchから、Application DriverとApplication Patternの旧work refを小集合で確認しました。open PR #606のhead `work/postgres-partition-drop-readiness-533`は引き続き保護し、今回の対象から除外しています。

### 8.1 Application Driver

対象:

- `work/application-driver-evidence-stage-latest-sync`
- `work/application-driver-final-revision-sync`
- `work/application-driver-gate-b-current-sync`
- `work/application-driver-gate-b-execution-status-69`
- `work/application-driver-pattern-scenario`
- `work/application-driver-plan-sync-after-reference-data-access`
- `work/application-driver-resource-resolution`

確認時点のcurrent `main`:

```text
5b6f9fb7b3618d9f3196013674fc172b68ec6b9b
```

7本はいずれも:

```text
ahead_by = 0
files = []
```

でした。固有未統合作業を持たないことを確認したため、7本とも**forceなし**でcurrent `main`へfast-forward normalizationしました。

### 8.2 Application Pattern

次に、Application Patternの基礎work refを確認しました。

対象:

- `work/application-pattern-browse`
- `work/application-pattern-command-kind`
- `work/application-pattern-command-reference`
- `work/application-pattern-header-detail-kind`
- `work/application-pattern-header-detail-reference`
- `work/application-pattern-loader`
- `work/application-pattern-record-kind`
- `work/application-pattern-record-reference`

並行workstreamによりcurrent `main`が次へ進んだ後に比較しました。

```text
894417a52485d758e7516ea8b9f2eaaf5f8516e6
```

8本はいずれも:

```text
ahead_by = 0
files = []
```

でした。このため8本とも**forceなし**でcurrent `main`へfast-forward normalizationしました。

### pass 5結果

```text
reviewed refs:        15
normalized:           15
ahead_by > 0 found:    0
force updates:         0
new cleanup branches:  0
```

Application Driver / Patternの過去成果はすでにmainへ包含されており、今回確認した集合では未救済の固有patchは見つかりませんでした。

## 9. 2026-09-16 current-state pass 6 — Application Pattern remaining refs

Application Patternのうち、pass 5でまだ確認していなかったReference・Source Index・stage completion・Validation系の旧work refを確認しました。

対象:

- `work/application-pattern-reference`
- `work/application-pattern-reference-kind`
- `work/application-pattern-reference-reference`
- `work/application-pattern-reference-rename-invariance`
- `work/application-pattern-source-index`
- `work/application-pattern-stage-complete-docs`
- `work/application-pattern-validation`
- `work/application-pattern-validation-integration`

確認時点のcurrent `main`:

```text
f267d735ff3767fde1f9ace4196ce4fa99ce4311
```

8本はいずれも:

```text
ahead_by = 0
files = []
```

でした。open PR headではなく、bootstrap / Evidence用途のrefでもないため、8本とも**forceを使わず**current `main`へfast-forward normalizationしました。

### pass 6結果

```text
reviewed refs:        8
normalized:           8
ahead_by > 0 found:   0
force updates:        0
new cleanup branches: 0
```

これでApplication Patternの主要な旧work refについて、少なくとも今回監査した範囲ではmain未包含の固有patchを持たないことを確認できました。

## 10. 2026-09-16 current-state pass 7 — Capability / Business Validation / Contract Query refs

次に、Semantic Capability・Business Validation・Contract Queryに関する通常の旧`work/*` refを確認しました。open PR #606のheadは引き続き対象外です。

### 10.1 Capability / Business Validation

対象:

- `work/capability-business-result-boundary`
- `work/capability-named-outcome`
- `work/capability-outcome-core`
- `work/capability-outcome-semantic-anchor`
- `work/business-validation-evaluator`
- `work/business-validation-reference`
- `work/business-validation-semantic-contract`

### 10.2 Contract Query

対象:

- `work/contract-query-ci-status-sync`
- `work/contract-query-diagnostic-source`
- `work/contract-query-executor-binding-authoring`
- `work/contract-query-executor-capability`
- `work/contract-query-no-realization-fallback-conformance`
- `work/contract-query-plan-sync`
- `work/contract-query-realization-identity`
- `work/contract-query-runtime-instance-key`
- `work/contract-query-sql-executor-adapter`
- `work/contract-query-sql-loader-integration`
- `work/contract-query-sql-realization`

確認時点のcurrent `main`:

```text
37a1e672ace1838354af592f8aebfde62596ea51
```

18本はいずれも:

```text
ahead_by = 0
files = []
```

でした。open PR headではなく、bootstrap / Evidence用途のrefでもありません。したがって18本とも**forceを使わず**current `main`へfast-forward normalizationしました。

### pass 7結果

```text
reviewed refs:        18
normalized:           18
ahead_by > 0 found:    0
force updates:         0
new cleanup branches:  0
```

Capability outcome、Business Validation、Contract Query realization / executor / runtime identity等の過去成果は、今回確認した範囲ではすでにmainへ完全に包含されています。

## 11. 2026-09-16 current-state pass 8 — Documentation quality / reading-path refs

文書品質・読み順改善に関する旧`docs/*` refのうち、同じhistorical tipを共有していた小集合を確認しました。

対象:

- `docs/document-quality-rollout`
- `docs/documentation-quality-guidance`
- `docs/improve-business-ai-reading-path`
- `docs/improve-core-reading-path`
- `docs/improve-plan-issue-reading`

5本はいずれも監査開始時点で同じtip:

```text
b7de002a02eac2979c3fbf0fd8e5e2e05890e4c9
```

を指していました。このtipをcurrent `main`:

```text
e197e997b5e6c7116a85ff64e946e78b6a58054f
```

と比較したところ、`b7de002...`はcurrent mainのancestorで、比較結果は:

```text
current main ahead_by = 51
historical tip behind_by = 0
```

でした。つまり5本は固有未統合commitを持たず、open PR headでもなく、bootstrap / Evidence用途のrefでもありません。

このため5本とも**forceを使わず**current `main`へfast-forward normalizationしました。

### pass 8結果

```text
reviewed refs:        5
normalized:           5
ahead_by > 0 found:   0
force updates:        0
new cleanup branches: 0
```

文書品質・読み順改善の成果はすでにmainへ包含されており、今回の操作は旧tipを作業候補として誤認しにくくするためのref normalizationだけです。

## 12. 2026-09-16 current-state pass 9 — Lane C Business Procedure observation-pressure refs

Lane Cのsecond-corpus / actuality境界を検証した際に派生した、Business Procedureのphysical observation Pressure周辺の旧work refを確認しました。

対象:

- `work/business-procedure-cross-view-reference-current`
- `work/business-procedure-physical-observation-pressure`
- `work/business-procedure-physical-observation-pressure-actual`
- `work/business-procedure-physical-observation-pressure-final`
- `work/business-procedure-physical-observation-pressure-unused`

確認時点のcurrent `main`:

```text
05827eb3bdcbcacc48cf29539c69f1db056ef4e8
```

5本はいずれも比較時点で:

```text
ahead_by = 0
behind_by = 27
files = []
```

で、同じhistorical tip `34e75e8de0e853f88a4dda41a77c82a241a8bea3` を指していました。open PR headではなく、bootstrap / Evidence用途のrefでもありません。

これらの成果はすでにmainへ取り込まれており、branch名だけが`pressure` → `actual` → `final` → `unused`と世代化して残っていました。未統合作業候補と誤認しないよう、5本とも**forceを使わず**current `main`へfast-forward normalizationしました。

今回のnormalizationはBusiness Procedure semantics、second Handoffのpositive / negative Evidence、Pre-ADR readinessのいずれも変更しません。過去work refのtipだけをcurrent mainへ揃えるRepository hygieneです。

### pass 9結果

```text
reviewed refs:        5
normalized:           5
ahead_by > 0 found:   0
force updates:        0
new cleanup branches: 0
```

## 13. 次の監査順序

次も新しいhygiene branchを作らず、`work/repository-hygiene`を再利用します。

1. closed PRを持たない通常`work/*` / `docs/*` / `plan/*` branchを小集合ずつcurrent `main`と比較する
2. 同一責務の世代branchが複数ある場合は優先して比較する
3. `ahead_by > 0`を検出した時点でresetせず、patch-equivalenceとmerge-only changeを確認する
4. `bootstrap-source` / `evidence/*` 2本はKEEP / FROZENとして通常cleanupから除外する
5. PR #578 conflict 8本は履歴Evidenceが得られた場合だけidentityを更新する
6. deletion可能refは、normalizationとは別の明示listとして確定する

branch数を減らすことより、**mainにない固有作業と、ref名に残る必要なprovenanceを失わないこと**を優先します。