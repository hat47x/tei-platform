# ISSUE-000060: 過去の一回限りGitHub Actions workflow 16本を棚卸しする

- 状態: Done — 不要な17本を削除した（2026-09-20）
- 提案日: 2026-09-19
- 性格: Repository hygiene / CI safety
- 主担当要求: RQ-023
- 関連: ISSUE-000055 / CONTRIBUTING.md「GitHub Actionsは現在、手動診断だけに使う」

## 何に困っているのか

`.github/workflows/`には19本のworkflowがあります。このうち16本は、過去の作業branch（`work/issue-50-core-model`、`work/capability-named-outcome`、`work/semantic-relation-key-parser`、`work/legacy-authentication-binding-reference`、`work/security-authentication-outcome-reference`、`work/issue-50-advanced-index-actual-diff`、`bootstrap-source`、`evidence/soza-e3b-p2-version-exchange-2026-09-05`）へのpushだけで起動する、一回限りの作業用workflowです。ローカルにGoが使えない時期に、patchの適用・format・vet・compileをActions上で行い、結果をbranchへcommitし戻すために作られたものと見られます。

起票時に、意図して残すと分類していたのは次の3本です（`workflow_dispatch`等の手動・PR用）。うち2本は、下の「決定と実施」で削除に改めました。

- `repository-branch-hygiene.yml`（手動のrunner確認）
- `gate-b-baseline-source-qualification.yml`（手動・PR）
- `knowledge-portability-reference-evidence.yml`（手動）

## 減らしたい不確実性・リスク

現在、Actionsのquota停止でこれらは実行されません。しかし、Actionsを再開したとき（`ISSUE-000055`の案D・E）に、次が起き得ます。

- 16本のうち**12本が`contents: write`権限**を持ちます。過去と同じ名前のbranchを作る、または残っている場合、push一つでrepositoryへ書き込むworkflowが起動します。
- 名前が古い（`soza`は`sei-cognition`へ改名済み）。実体との対応が分からず、読む人の負担になります。
- 何が生きているかを見分けるために、19本を毎回読む必要があります。

repositoryはprivateで、現時点で実害は確認していません。あくまで、再開前に整理しておくべき衛生の課題です。

## 現在分かっていること

- 上記branch名のうち、2026-09-19時点でremoteに存在するものはありません。
- 各workflowが呼ぶscriptやpatch本文は、workflow内にinlineで含まれます（`issue50-core-model-apply*.yml`は220〜300行）。削除しても、内容はGit履歴に残ります。

## 決定と実施（2026-09-20）

利用者の方針は「不要なのであれば削除」です。棚卸しの結果、次のとおり削除しました（内容はGit履歴に残ります）。

- **削除した17本**
  - 過去の一回限りの作業用workflow 16本（`on: push`で起動し、うち12本が`contents: write`）: `apply-capability-outcome-patch` / `issue50-*`（9本） / `normalize-source` / `patch-semantic-relation-key-parser` / `soza-e3b-p2-evidence` / `update-legacy-auth-binding-world` / `update-security-authentication-world` / `update-semantic-anchor-outcome`
  - `repository-branch-hygiene`（branch運用の廃止に伴う）
  - `gate-b-baseline-source-qualification`（Gate B専用で、`tools/repository-health-check.sh`が包含する。`pull_request`起動もPR廃止と矛盾する）

  ※上のうち`repository-branch-hygiene`と`gate-b-baseline-source-qualification`は、起票時は「意図して残す」に分類していたものを、branch / PR運用の廃止（`ISSUE-000055`）に伴い削除に改めました。
- **残した1本**: `knowledge-portability-reference-evidence.yml`。手動起動のみで、同じ確認を行う`tools/run-knowledge-portability-reference-evidence.sh`の薄いラッパーです。研究laneの複数の文書が成果物として参照しています。

これで、GitHub Actionsを将来再開しても、書き込み権限を持つ過去のworkflowが起動することはありません。

## 決めること（以下は起票時の選択肢。上のとおり決定済み）

workflowごとに、次のいずれかを選びます。

1. **削除**: 目的を終えたものとして、`.github/workflows/`から取り除く（履歴には残る）。
2. **無効化して保管**: `on:`を外す、または`docs/`へ移すなどして、起動できない形で残す。
3. **維持**: 現在も使う。

削除・変更はCI/CDの変更にあたるため、利用者の承認のうえで行います。

## 完了条件

- [x] 19本の棚卸しを行う（用途、起動条件、書き込み権限、対応branch、判断は上記）
- [x] 16本の扱いを決める（すべて削除）
- [x] 決めた内容を反映する（mainへ直接commit）
- [x] 残るworkflowは1本のみであり、`CONTRIBUTING.md`の開発フローに、Actionsを使えないことと手動診断用だけを残す旨を記載した

## 非目標

- GitHub Actionsの再開そのもの（`ISSUE-000055`）
- 手動診断・source qualification用の3本の変更
- workflowの内容の書き換え

## 関連

- `.github/workflows/`
- `plan/issue/000055-verification-gate-while-ci-suspended.md`
