# Branch Normalization Audit — 2026-09-14

この文書は、2026-09-13〜14に行ったremote branch正規化の**実行snapshot**です。

恒久的なBranch / PR運用ルールは[`../../CONTRIBUTING.md`](../../CONTRIBUTING.md)、分類・削除判断の基準は[`branch-hygiene-2026-09.md`](branch-hygiene-2026-09.md)を正とします。

このsnapshotは「どのbranchを、どの根拠で安全にmain系へ戻したか」を記録するもので、Product Requirement、Issue state、Gate state、Architectureの正本ではありません。

## 1. 今回行ったこと

Repositoryには、内容がmainへmergeされた後も古いtipを保持するremote branchが多数残っています。

今回の作業ではbranchを一括削除せず、**そのbranchにmain未包含の固有作業が残っていないことを確認できたものだけ**を、その時点のcurrent `main`へresetしました。

これはbranch refの削除ではありません。

```text
before

old merged work ───── branch
          \
           └───────── main ── newer work

normalization

old merged work ───────────── main ── newer work
                                  ↑
                                branch
```

目的は、過去のmerge済みsnapshotを「まだ作業中の差分」に見せないことです。

## 2. 安全判定

今回は次のいずれかを満たす場合だけresetしました。

### A. merged PR headと現在branch tipが一致する

```text
PR state: merged
PR head SHA == current branch tip SHA
open PR: none
```

PR merge後にそのbranchへ新しい作業が積まれていないことを直接確認できる場合です。

### B. branchがcurrent mainのancestorである

`compare main...branch`相当の確認で、

```text
ahead_by = 0
behind_by > 0
```

となる場合です。

この場合、branch tipはすでにmain履歴へ完全に含まれており、branch側だけに残るcommitはありません。

### resetしなかった条件

次の場合は、名前が古く見えてもresetしません。

- open PRがある
- branch側が`ahead_by > 0`
- merged PR headとbranch tipが異なり、ancestor確認も取れていない
- branch ref自体がEvidence / release / external integrationに必要か不明

## 3. 正規化したbranch

### Repository hygiene /重複PR整理

- `claude/gracious-bell-mjpby4`
- `fix/lane-c-exact-datasource-authority-probes`
- `work/repository-hygiene`

PR #578へ実差分を統合し、重複draft PR #569をcloseした後、関連branchをmain系へ戻しました。

### 文書品質rollout

- `docs/document-quality-rollout`
- `docs/documentation-quality-guidance`
- `docs/improve-core-reading-path`
- `docs/improve-business-ai-reading-path`
- `docs/improve-plan-issue-reading`

PR #574〜#582でmerge済みの文書品質改善について、完了済みbranchの古いtipをmain系へ戻しました。

`docs/document-quality-rollout`は同じworkstreamを複数PRで再利用した実例でもあります。

### Design Synthesis / Definition-centered feedback

- `docs/design-synthesis-local-cognition`
- `work/definition-centered-feedback-loop`
- `work/definition-derived-inventory`
- `work/definition-feedback-routing`

`docs/design-synthesis-local-cognition`はPR #573後もPR #585で再利用されていたため、古い#573 headを根拠にresetせず、最新の利用状態を確認してから扱いました。

`work/definition-feedback-routing`も、PR #589がopenの間は触れず、merge後にcurrent tipとmerge内容を確認してから正規化しました。

### Derived Cognition / research

- `docs/cognitive-requirement-draft`
- `docs/cognitive-top-down-structure`
- `research/cognitive-feature-corpus`
- `research/cognitive-benchmark-cases`
- `research/cognitive-feature-extraction-contract`
- `research/derived-cognitive-layer`

これらはPR #560〜#564、#572等でmerge済みのheadとの一致を確認したうえでmain系へ戻しました。

### Product / Business design quality

- `work/product-quality-requirements-architecture-565`
- `work/lane-c-business-design-quality-contract`
- `work/tei-tech-research-integration-2026-09-11`
- `work/application-driver-gate-b-execution-status-69`
- `work/lane-c-pre-adr-handoff-order-invariants`

`work/application-driver-gate-b-execution-status-69`は、PR #559の記録headと現在branch tipが一致しませんでした。

このため即resetせず、mainとの比較を追加しました。結果は、

```text
ahead_by = 0
behind_by = 81
merge_base = branch tip
```

であり、現在tipそのものがmainのancestorと確認できたため、固有未統合作業なしとして正規化しました。

これは「PR headが違うから危険」と機械的に止めるのではなく、**より強い現在のGit ancestry Evidenceで安全性を再確認する**例です。

## 4. reset先について

Repositoryでは並行workstreamからmainへ継続的にmergeされています。

そのため全branchを一つの固定SHAへ揃えること自体は目的にしません。

今回も作業途中でmainが進み、複数のcurrent main SHAへ段階的にresetしています。

重要なのは、

```text
branchにmain未包含の固有commitを残さない
```

ことであって、

```text
全branch tip == 今この瞬間のmain tip
```

を常時保証することではありません。

後からmainが進めば、正規化済みbranchが再び`behind`になることは正常です。

## 5. branch数は減っていない

この作業で行ったのはrefの**reset / normalization**であり、remote ref deletionではありません。

現在利用しているGitHub接続にはbranch refを削除するactionがないため、branch数そのものはこの作業では減りません。

したがって、

```text
normalized branch count increase
        !=
remote branch count decrease
```

です。

branch数を減らす作業は、削除可能なrefを明示的に確定した後、branch deletionを実行できる認可済み手段で別途行う必要があります。

## 6. GitHub Actions guardの扱い

Branch fragmentation防止のため、命名policyとread-only inventory workflowも追加しました。

ただし2026-09-14時点ではGitHub-hosted jobがstep開始前にfailureとなり、次が観測されています。

```text
runner_id: 0
runner_name: ""
steps: []
```

checkoutを除いたjobでも同じ状態だったため、workflow logicの成功・失敗としては扱っていません。

全PRへ意味のないfailureを付け続けないよう、自動triggerは一時停止し、`workflow_dispatch`だけをrunner復旧確認用に残しています。

現在有効な運用は次です。

```text
CONTRIBUTING / AGENTS rules
        +
tools/repository-branch-policy.sh
        +
tools/repository-branch-audit.sh
```

GitHub Actionsの自動guardは、runner上で実際にstepが開始・完了するEvidenceを得てから再有効化します。

## 7. 次の監査対象

今後は、既に正規化したbranchを繰り返し調べるのではなく、次を優先します。

1. `ahead_by > 0`となるbranchの抽出
2. PR #578で名前が保存されなかった「conflict 8本」の再特定
3. `PATCH_EQUIVALENT_REVIEW`の確認
4. open PR branchの保護
5. 実際にdelete可能と判断したrefの明示list作成

`ahead_by > 0`のbranchについては、内容を読まずにresetしません。

## 8. 現時点の結論

今回の正規化により、直近の主要workstreamについては、merge済み内容を保持した古いbranch tipを「未統合作業」と誤認する余地を大きく減らしました。

一方で、Repository全体のremote branch数を減らしたわけではなく、古いbranchすべてを安全確認したわけでもありません。

次段階では、**branch数を減らすこと自体ではなく、main未包含の固有作業を持つbranchを少数へ絞り込むこと**を先に完了させます。

## 9. 2026-09-16 follow-up

2026-09-16は、新しいcleanup用branchを作らず、既存の`work/repository-hygiene`を安全確認後にcurrent `main`へ戻して再利用しました。監査開始時と記録直前にopen PRを確認し、作業中のPR #605が使用する`work/second-realistic-handoff-corpus-selection-050`はcleanup対象から除外しています。

このfollow-upでも、branch名やclose理由だけではresetしていません。closed-unmerged PRのheadをcurrent `main`と比較し、**確認時点で`ahead_by = 0`となったbranchだけ**をmain系へ戻しました。`ahead_by > 0`や`PATCH_EQUIVALENT_REVIEW`に相当するbranchを、この手順だけでresetする扱いには変更していません。

### M0.27 / Application Driver / Contract Query

- `claude/merge-unmerged-branches-main-po0ixx`
- `work/semantic-relation-traversal-keys`
- `work/semantic-relation-key-authoring`
- `work/database-contract-query-signature-model`
- `work/database-contract-query-signature-validation`
- `work/database-contract-query-signature-syntax`
- `work/reference-contract-query-invocation-model`
- `work/reference-contract-query-invocation-validation`
- `work/reference-contract-query-invocation-authoring`
- `work/reference-contract-query-resolution`
- `work/reference-contract-query-target-validation`
- `work/semantic-relation-key-target-uniqueness`
- `plan/driver-relation-data-access-gaps`
- `plan/reference-contract-query-invocation`
- `plan/application-driver-reference-progress`
- `work/remove-duplicate-applicationdriver-current`

Relation traversal key、Contract Query signature、Reference invocation、Driver resolution、旧`internal/applicationdriver`整理等を含むため、文書branchより先に固有実装の残存有無を確認しました。上記はすべて確認時点で`ahead_by = 0`でした。

### Evidence / Gate B

- `work/postgres-evidence-source-revision-doc-sync`
- `work/gate-b-final-scenario-audit-sync`

いずれも後続のEvidence正本・監査文書へ収束済みで、確認時点ではmain未包含commitを持っていませんでした。

### Runtime / Authenticationの差し替え・重複PR

- `work/native-query-postgres-reference`
- `work/native-query-postgres-execution-reference`
- `work/driver-pattern-resolution`
- `work/legacy-auth-function-adapter`
- `work/legacy-auth-compatibility-reference`
- `work/security-authentication-reference-v2`

これらはPR本文で後継PRまたは正本PRへの差し替え・重複整理が明示されていたbranchですが、close理由だけに依存せず、現在のGit ancestryでも`ahead_by = 0`を確認してから正規化しました。

### Application Pattern / v1 planning

- `plan/application-pattern-stage-complete`
- `work/header-detail-pattern-contract`
- `work/header-detail-semantic-substrate`
- `work/application-pattern-doc-sync`
- `work/application-pattern-diagnostic-projection`
- `plan/v1-current-state-sync`
- `plan/application-pattern-progress-sync`

一部先行反映、重複PR、GitHub側のPR head追跡不整合による載せ替え等、close理由は異なりますが、上記もすべて確認時点で`ahead_by = 0`でした。

### Branch ruleとの整合

このfollow-upでは、`-sync`等を含む古いbranch名を新しく作成していません。既存のlegacy refを監査してmain系へ戻しただけです。今後のworkstream名としてこれらを再採用する意図もありません。

また、`work/repository-hygiene`自体も、PR #592がmerge済みで未merge変更を持たないことと`ahead_by = 0`を確認してからcurrent `main`へ戻し、この記録更新に再利用しました。

### 残件

今回確認した集合では、新たな`ahead_by > 0` branchは見つかりませんでした。ただし、これはRepository全remote branchの再分類完了を意味しません。

特に次は引き続き未解決です。

- PR #578で名前が保存されなかった「conflict 8本」のidentity再確定
- `PATCH_EQUIVALENT_REVIEW`のmerge-only固有変更確認
- closed PRを持たないremote branchを含むcurrent全branch report
- delete可能refの明示list作成

したがって、次もbranch数そのものを減らす一括操作ではなく、**current mainにない固有patchを持つbranchを先に特定して保護する**ことを優先します。

## 10. 2026-09-16 legacy suffix / snapshot normalization

このpassでも新しいcleanup branchは作成せず、既存の`work/repository-hygiene`を再利用しました。開始時点のcurrent `main`は`34e75e8de0e853f88a4dda41a77c82a241a8bea3`で、`work/repository-hygiene`も同じSHAに揃っていることを確認しています。

open PRは#606の1本だけで、そのhead `work/postgres-partition-drop-readiness-533`は作業中branchとして保護し、今回のnormalization対象から除外しました。

今回の対象は、過去の作業過程で残った`-final`、`-current`、`-v2`、`-baseline`、`-snapshot`、`-scratch`、`-unused`等のgeneration / state suffixを持つlegacy refを中心としています。名前だけでは判断せず、各tipをcurrent `main`と比較し、**`ahead_by = 0`を確認できた52本だけ**をmain系へ正規化しました。この集合では`ahead_by > 0`のbranchは見つかっていません。

### Gate B generation / readiness

- `work/gate-b-v1-driver-scope-sync`
- `work/gate-b-v1-driver-scope-sync-final`
- `work/gate-b-v1-driver-scope-sync-mergecheck`
- `work/gate-b-v1-driver-scope-sync-r2`
- `work/gate-b-generated-artifact-clean-revision`
- `work/gate-b-generated-artifact-clean-revision-v2`
- `work/gate-b-post-300-plan-sync`
- `work/gate-b-post-300-plan-sync-v2`
- `work/gate-b-readiness-main-sync`
- `work/gate-b-readiness-main-sync-2`

### Lane C generation

- `work/lane-c-issue-inventory-sync`
- `work/lane-c-issue-inventory-sync-2`
- `work/lane-c-issue-inventory-sync-final`
- `work/lane-c-two-blocker-ownership-sync`
- `work/lane-c-two-blocker-ownership-sync-v2`
- `work/lane-c-pre-adr-current-state-sync`

### Business / Reference

- `work/business-validation-reference`
- `work/business-validation-reference-v2`
- `work/business-validation-reference-v3`
- `work/business-validation-reference-v4`
- `work/business-procedure-physical-observation-pressure`
- `work/business-procedure-physical-observation-pressure-actual`
- `work/business-procedure-physical-observation-pressure-final`
- `work/business-procedure-physical-observation-pressure-unused`

### Application Pattern / planning / current-state

- `work/application-pattern-header-detail-kind`
- `work/application-pattern-header-detail-kind-current`
- `work/application-datasource-alias-exact-resolution-scratch`
- `docs/readme-gate-a-current`
- `docs/status-gate-a-current`
- `plan/gate-a-readiness-current`
- `plan/application-pattern-command-current`
- `plan/header-detail-stage-current`
- `plan/v1-current-state-final`
- `work/application-pattern-docs-current`
- `work/architecture-current-boundaries`
- `work/business-procedure-cross-view-reference-current`

### Gate B current / baseline / snapshot

- `work/application-driver-gate-b-current-sync`
- `work/gate-b-audit-issue-current-sync`
- `work/gate-b-evidence-acquisition-current-baseline`
- `work/gate-b-evidence-acquisition-current-contract-sync`
- `work/gate-b-final-issue-current-baseline`
- `work/gate-b-final-issue-current-snapshot`
- `work/gate-b-readiness-audit-current-baseline`
- `work/gate-b-readiness-current-evidence-sync`
- `work/gate-b-readiness-issue-current-snapshot`
- `work/current-entrypoint-baseline`
- `work/gate-b-runbook-current-baseline`
- `work/gate-b-runbook-secret-read-current-sync`

### その他のcurrent / baseline

- `work/master-v1-plan-current-baseline`
- `work/product-gap-audit-current-baseline`
- `work/query-result-mapping-runtime-current`
- `work/upstream-trace-current-baseline`

### Branch fragmentation ruleとの整合

このpassで`-final`、`-current`、`-v2`、`-baseline`、`-snapshot`、`-scratch`、`-unused`を含むbranchを**新規作成していません**。既存legacy refの安全性を確認し、main系へ正規化しただけです。

また、この作業はref deletionではありません。したがって52本を正規化してもremote branch countそのものは減少していません。削除候補の確定と実際のref deletionは、固有patch・Evidence用途・外部参照の有無を別途確認した後の作業です。

PR #578で名前が保存されなかった「conflict 8本」については、今回も記録からidentityを確定できていません。候補名を推測してresetすることはせず、**未特定のまま保護対象として扱います**。

### 次の監査対象

次の優先順位は変えません。

1. current全branchから`ahead_by > 0`を抽出し、固有patchを持つbranchを保護する
2. PR #578の「conflict 8本」を履歴Evidenceから再特定する
3. `PATCH_EQUIVALENT_REVIEW`のmerge-only固有変更を確認する
4. open PR branchを継続して保護する
5. ancestry / Evidence確認済みのdelete可能refを明示listへ落とす

branch数を機械的に減らすことよりも、**mainにない固有作業を失わないことを先に保証する**方針を維持します。
