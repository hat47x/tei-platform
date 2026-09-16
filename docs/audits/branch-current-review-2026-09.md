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

### 2.3 ancestryは安全でもref用途を未確認のため保護したbranch

次の3本も`ahead_by = 0`でしたが、今回はrefを動かしていません。

- `bootstrap-source`
- `evidence/soza-e3b-p2-2026-09-02`
- `evidence/soza-e3b-p2-version-exchange-2026-09-05`

理由は、`ahead_by = 0`が示すのは「固有commitがない」ことまでであり、ref名そのものをbootstrap provenanceやEvidence参照に利用していないことまでは証明しないためです。

```text
commit ancestry safe
  !=
ref identity disposable
```

この3本は、参照元を確認するまで保護対象とします。

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

## 4. このpassの結果

```text
reviewed refs:       11
normalized:           8
protected by ref use: 3
ahead_by > 0 found:   0
force updates:        0
new cleanup branches: 0
```

これはRepository全branchのcurrent再分類完了を意味しません。

今回の価値は、一時・integration・audit系の明確な集合で固有patch残存がないことを確認し、同時にEvidence / bootstrap系を「ancestryだけでは動かさない」保護境界として明示したことです。

## 5. 次の監査順序

次は、新しいhygiene branchを作らず、`work/repository-hygiene`を再利用して次の順に進めます。

1. closed PRを持たない通常`work/*` branchを小集合ずつcurrent `main`と比較する
2. `ahead_by > 0`を検出した時点でresetせず、patch-equivalenceとmerge-only changeを確認する
3. Evidence / bootstrap refの参照元を確認し、ref identity保持要否を判定する
4. PR #578 conflict 8本は履歴Evidenceが得られた場合だけidentityを更新する
5. deletion可能refは、normalizationとは別の明示listとして確定する

branch数を減らすことより、**mainにない固有作業と、ref名に残る必要なprovenanceを失わないこと**を優先します。
