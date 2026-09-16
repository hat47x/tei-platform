# TEIへのContribute

TEIは、現在もArchitectureとReferenceを相互にpressureしながら育てている実験的なプロジェクトです。

Code contributionだけでなく、次のような参加も重要です。

- Reference WorkspaceやCLIを実際に試した結果
- 既存の業務システムが現在のmodelへ素直に当てはまらない例
- migration / database / deploymentのedge case
- 既存のlow-code、model-driven、workflow、schema toolとの比較
- 設計上の調査結果や運用上の知見
- Architecture上の不整合を小さく再現できる例

大きなCore primitiveを追加する前に、まず「どのProduct Requirement上の不確実性を減らしたいか」「どの利用例で必要か」をIssueや設計文書へ残すことを優先します。

## 設計上の基本姿勢

TEIでは、概ね次を優先します。

- runtimeの推測より明示された意味
- special kindを増やすより小さいkernel
- 既に強いnative formatは無理に置き換えない
- AIが不要な箇所ではdeterministicな生成・検証
- command acknowledgementを事実とみなさずObservationを確認する
- 特定技術をSemantic Definitionへ埋め込むよりProvider境界を置く

## Branch / PR運用

Branchは作業履歴の保管場所ではありません。**履歴はmainとPRへ残し、remote branchは現在進行中の責務だけを表す**ものとします。

### 1. 一つのworkstreamにremote branchを一つだけ持つ

同じ目的の作業で、次のような派生branchを増やしません。

```text
feature-x-final
feature-x-sync
feature-x-current
feature-x-rebased
feature-x-v2
feature-x-20260913
```

同じworkstreamを続ける場合は、既存branchが安全に再利用できるかを確認し、再利用します。

例:

```text
docs/document-quality-rollout
work/issue-000038-business-lifecycle
research/derived-cognitive-retrieval
```

branch名は「今回の一回の作業」ではなく、継続して同じ責務を持つworkstreamを表します。

### 2. 新しいbranchを作る前に既存branchを確認する

新規branch作成前に、同じIssue / lane / workstreamのbranchが既にないか確認します。

既存branchがある場合は、次を確認します。

1. open PRや未merge commitを持っていないか
2. 以前の作業がすでにmainへmerge済みか
3. branch上の変更がPR / mainへすべて取り込まれているか

安全に再利用できる場合、新しいsuffix付きbranchを作らず、既存branchをcurrent mainへ戻してから次の変更を始めます。

未mergeの変更が残るbranchをforce resetしてはいけません。

### 3. 一つのbranchで同時に一つのPRだけ扱う

同じbranchから複数のopen PRを並行させません。

```text
branch
  ↓
小さく整合した変更
  ↓
PR
  ↓
review / check
  ↓
merge
  ↓
branchをcurrent mainへ戻す
  ↓
次の変更
```

関連作業を一つの巨大PRへ積み上げる必要もありません。**branchは再利用してよいが、PRは判断・reviewしやすい粒度へ分ける**ことを基本とします。

### 4. merge後のbranchを放置しない

PRをmergeした後は、次のどちらかにします。

- 一回限りのbranchならremote branchを削除する
- 継続workstream branchなら、次の作業を始める前にcurrent `main`と同じcommitへ戻して再利用する

通常のmerge / rebase mergeでfast-forward可能ならfast-forwardします。**squash mergeでは元branchのcommit列がmainの祖先にならないため、PRがmerge済みで未merge変更がないことを確認した場合に限り、継続branchをcurrent mainへresetしてよい**ものとします。

このresetは履歴破壊を目的とするforce pushではなく、merge済みworkstream branchを再利用可能な空の作業面へ戻すための操作です。open PRや未merge commitが一つでも残る場合は行いません。

merge済みbranchを過去のsnapshotとして残す必要はありません。必要な履歴はPRとGit historyから辿れます。

接続tool等の制約でremote branchを削除できない場合も、同じ目的の新branchを増やさず、上記条件を満たす継続branchをmainへ戻して再利用します。

### 5. temporary branchをremoteへ残さない

`tmp-*`、`noop-*`、単発の確認専用branchは原則としてlocalだけで使います。

remoteで共有する必要がある実験は、何を検証するかが分かる`research/<workstream>`として扱い、対応するIssue / planning documentへ目的と終了条件を残します。

### 6. branch種別を増やしすぎない

基本形は次です。

```text
work/<issue-or-workstream>      実装・Reference・横断作業
docs/<workstream>               文書品質・説明・文書構造
research/<workstream>           未採用方式の比較・architecture probe
evidence/<issue-or-gate>        actual Evidence取得が主目的の作業
fix/<workstream>                 狭い修正
```

prefixを細分化すること自体を目的にしません。同じ責務なら同じbranchを使います。

### 7. branchはProduct stateを表さない

`ready`、`final`、`done`等をbranch名でProduct stateとして表現しません。

Product / Issue / Gateの状態は、それぞれの正本文書とEvidenceで管理します。

```text
branch name != Issue state != Gate state != Product claim
```

### 8. mainへ直接書き込まない

通常の変更は、

```text
current main
  ↓ branch
変更
  ↓ commit
PR
  ↓ check / review
merge
```

の順で行います。

緊急のRepository復旧等を除き、通常作業でmainへ直接file updateを行いません。

### 9. branch数が増えたらread-only監査から始める

remote branchを整理するときは、名前だけを見て削除しません。

```bash
tools/repository-branch-audit.sh
```

このtoolはrefを削除・変更せず、current `origin/main`に対してbranchを次のように分類します。

- `MERGED_ANCESTRY` — branch tipがmainのancestor
- `IDENTICAL_TREE` — branch tipとmainのtreeが同一
- `PATCH_EQUIVALENT_REVIEW` — 固有non-merge patchは見つからないが、merge topology等の確認が必要
- `UNMERGED_REVIEW` — mainにない固有patchを持つため個別reviewが必要

`MERGED_ANCESTRY`や`IDENTICAL_TREE`であっても、open PR、release、external integration等でbranch refを必要としていないことを削除前に確認します。

`PATCH_EQUIVALENT_REVIEW`を自動削除してはいけません。squash merge後のstale branchであることが多い一方、merge-onlyの固有変更を持つ可能性を安全側に残しています。

一度、`MERGED_ANCESTRY` / `IDENTICAL_TREE`相当でmain包含済み、open PRなし、ref固有用途なしと確認した一回限りのstale branchは、監査記録上の`DELETE_CANDIDATE`として固定します。**mainが前進するたびに、そのbranchを新しいmainへfast-forwardし直しません。**

繰り返しrefを更新してもbranch数は減らず、push-triggered workflowを不要に起動する可能性があります。remote ref削除の操作手段や権限がその場で利用できない場合は、候補refを動かさず監査済みとして次回passから除外し、削除可能になった時点で明示された候補リストから一度だけ削除します。

```text
main包含済み + ref用途なし
        ↓
DELETE_CANDIDATEとして記録
        ↓
再同期しない / 再監査しない
        ↓
削除可能になった時点で一度だけ削除
```

2026年9月の既存branch棚卸しについては[`docs/audits/branch-hygiene-2026-09.md`](docs/audits/branch-hygiene-2026-09.md)を参照してください。

### 10. 新しいbranchはpolicy scriptで確認する

新しいPRを作る前に、[`tools/repository-branch-policy.sh`](tools/repository-branch-policy.sh)でbranch名を確認します。

```bash
tools/repository-branch-policy.sh "$(git branch --show-current)"
```

次を新規branch名として使いません。

- `-final` / `-sync` / `-current` / `-rebased` / `-latest` / `-done` / `-ready`
- 日付suffix
- `-v2` / `-v3`等の世代suffix
- remote上の`tmp` / `noop` branch

目的は名前を形式的に揃えることではなく、**同じworkstreamに新しい世代branchを作る代わりに、既存branchを安全に再利用する習慣を機械的に支えること**です。

2026-09-14時点ではGitHub-hosted runnerがjob step開始前に割り当てられない状態を観測しているため、PRへの自動checkは一時停止しています。自動checkの有無にかかわらず、このRepositoryルールとpolicy scriptは有効です。

### 11. GitHub Actionsは現在、手動診断だけに使う

`.github/workflows/repository-branch-hygiene.yml`は、runner利用可能性を再確認するための`workflow_dispatch`だけを残しています。

自動PR check / weekly auditは、少なくとも一度manual runがrunner上で正常に開始・完了できることを確認してから再有効化します。

```text
runner unavailable
   ↓
automatic hygiene checks suspended
   ↓
local policy / read-only auditを継続
   ↓
manual workflowでrunner復旧を確認
   ↓
automatic triggerを再有効化
```

workflowが実行可能になっても、branchを自動削除・resetするauthorityは与えません。

## Development

```bash
go test ./...
go vet ./...
go run ./cmd/tei validate -workspace reference/workspace
```

Reference Workspaceには、過去contractとの互換性を確認するためのwarningが意図的に含まれる場合があります。

文書だけを変更した場合も、リンク、正本性、状態表現、Gate / Evidence claimを壊していないかを確認してください。文書品質の詳細は[`docs/documentation-quality.md`](docs/documentation-quality.md)を参照します。
