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
