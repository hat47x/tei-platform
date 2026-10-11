# TEI Public化準備・公開判定Runbook（候補）

- 日付: 2026-10-11
- 判定: **NOT READY — 未監査のままPrivate→Publicを実施しない**
- Parent: [#1187](https://github.com/hat47x/tei-platform/issues/1187)
- 機密性・非Git公開面: [#1188](https://github.com/hat47x/tei-platform/issues/1188)
- CI・保護設定: [#1189](https://github.com/hat47x/tei-platform/issues/1189)
- 文書・書誌情報・公開用Git構成: [#986](https://github.com/hat47x/tei-platform/issues/986)、[Draft PR #1164](https://github.com/hat47x/tei-platform/pull/1164)、[#1190](https://github.com/hat47x/tei-platform/issues/1190)
- 外部参照モデルの公開境界: [#941](https://github.com/hat47x/tei-platform/issues/941)、[既存方針](REFERENCE_MODEL_PUBLICATION_BOUNDARY_2026-10.md)

## 1. 目的と境界

Public化は標準GitHub-hosted Actionsによる正規checkoutのCIを可能にし、Private認証がなくてもSourceを取得できるようにするための候補。ChatGPTの実行制限まで解消するわけではない。MCPトンネルを利用する実機検証は別途必要となる。

Private→Publicによって過去のGit historyだけでなくActions実行履歴・ログ、Issue/PR、添付等が公開範囲になる。Public→Privateに戻しても既に取得されたコピーを回収できない。公開操作自体はこのRunbookやIssue起票によって承認されたものではない。

## 2. 2026-10-11に観測した事実

| 対象 | 状態 |
| --- | --- |
| `hat47x/tei-platform` | Private、default `main` |
| main SHA | `53ce39c35851b1ec731eb6cd3066d9a708f3620d`（観測時点のsnapshot） |
| main Git tree | recursive 1,928 entries / truncated=false（**file path inventoryのみ**） |
| root LICENSE | MIT |
| `skills/ponytail/LICENSE` | 独立した第三者のMIT著作権表示あり |
| main Actions workflow | `.github/workflows/knowledge-portability-reference-evidence.yml`: manual-only、job `if: false` |
| README | 生成AIを用いて開発中・体系的な人的レビュー未実施の警告を冒頭へ追加する候補 |
| `SECURITY.md` | mainに存在しない。連絡窓口の所有者判断が必要 |
| 第三者モデル | #941の安全側公開境界を維持。原モデル/手引の直接再配布は許諾確認まで禁止 |
| Git履歴の公開適合性 | mainの一部コミットの参照関係を確認。全refsのmetadata/contentの網羅検査は未了 |
| サーバー保護 | 過去の#513/#972でmain無保護が記録された。Private rulesets APIは現プランで403 |
| 全履歴・非Git面 | **未監査**。この一覧から公開可とは推論しない |

## 3. P0-A 機密性・権利・公開面

- 2026-10-11にGitHubの全1,081 refsを一時mirrorし、Gitleaks v8.29.1でGit本文を走査した記録: [Git履歴機密情報スキャン](../docs/audits/PUBLIC_GIT_HISTORY_SECRET_SCAN_2026-10-11.md)。検出2件はGo enum識別子の誤検知と初期判定した。独立した人間reviewerの確認は未完了であり、Issue #1188のGateは閉じない。
- [x] 信頼済みHostでprivate mirror cloneを取得し、全refs（active branches、tags、PR refsを含む）と `git rev-list --objects --all` を列挙。2026-10-11時点のmirrorは1,081 refs / 46,016 objects。commit author/committer metadataとメッセージの公開適合性は未確認。
- [x] バージョン固定済みのGitleaksでGit履歴を走査。2検出をGo enum識別子の誤検知と初期判定したが、独立した人間reviewerの承認待ち。報告書・検出値はRepository外に保管し、Issueへ候補値を記録していない。Evidence: [Git履歴機密情報スキャン](../docs/audits/PUBLIC_GIT_HISTORY_SECRET_SCAN_2026-10-11.md)。作業ツリー自体の未追跡・一時ファイル検査は別途未実施。
- [ ] 個人情報、社内・顧客情報、credential、endpoint、private URL、外部文献の出典・利用条件等の公開適合性を確認する。
- [ ] Issue/PR/reviews/commentsと添付、全過去Actions logs/artifacts、Release/assets、Wiki/Pages等も対象にする。API取得不可なら `NOT_INSPECTED`。
- [ ] `LICENSE` とThird-party source/Skill/assets/資料のcopyright・再配布条件を照合。#941の参照モデル原物・抜粋・変換物等の再配布範囲は独立審査。
- [ ] 対応が必要な項目が判明した場合は、適切な権限管理・情報の取扱い・必要な資格情報の更新手順を定め、共有refsへの変更を一括して計画する。
- [ ] 安全なscope・scanner version・refs数・objects数・未解決件数・人間reviewerだけをpublic Evidenceへ出す。

## 4. P0-B 書誌情報と公開用Git構成

原則: 文献の著者・書誌情報・必要な著作権表示を維持し、外部資料の記載内容とTEI独自の設計判断を区別する。文書の見出し、ファイル名、リンク、Git metadataは読み手にとって明確で追跡可能な構成とする。

1. 文書・ファイル名・内部リンクの整合性確認は**既存Draft PR #1164**で進行する。別PRに重複実装しない。
2. trusted full historyから `git log --all` のsubject/body/trailer、著作者・コミッター情報、GitHub commit commentsを取得し、公開対象と出典・ライセンス要件への適合性を確認する。
3. Git履歴の構成を変更する場合は、多数の子コミット・branch・PR・Release・exact-head Evidenceへの参照に影響する。範囲と移行計画を確定するまで、部分的なforce pushは実施しない。
4. #1190で **(A) 既存Repositoryの公開** と **(B) 新しいPublic Repositoryに監査済みSourceを導入** を比較する。GitHub管理のPR refs、キャッシュ、外部cloneの可視性と、公開URL・Issue番号・PRの継続性も評価する。
5. Aで共有Git履歴の移行が必要な場合は開発laneのfreeze、Private mirror backup、再現可能な移行手順、旧→新SHA対応、PR/refs確認、資格Evidence再取得、復旧計画、最終Human reviewを先に整える。既存receiptを異なるheadへ機械的に転用しない。
6. Bを採用する場合はPrivate/公開RepositoryのSource同期方式、Issue運用の正本、GitHub Actionsの資格の対応を明示する。新RepositoryへのSource導入では既存Git lineageが自動継承されない。

### 2026-10-11 全Git refsの追加観測

GitHub API `git/matching-refs/` で観測されたGit refは計 **1,078件**（heads **64** / tags **4** / pull **1,010**）。ref名称の確認だけでは参照先commitの本文やmetadataの公開適合性を判定できない。

**PR参照はGitHub管理下にあり、通常のforce/mirror pushでは任意に更新できない。** 参照先SHA・PR ref・キャッシュ・外部cloneの継続性を考慮して、公開先Repositoryに含めるGit historyの範囲を決める。

**Git履歴とIssue/PRの継続性を優先する構成A**と、**公開用Sourceと開発記録を分離する構成B**の双方について、保守負担、外部からの参加、公開範囲、資格Evidenceの扱いを比較して決定する。

## 5. 公開候補の事前生成と開発運用の移管

### 5.1 固定コミットからの出力（Draft PR #1195）

`tools/public-release/prepare-candidate.sh`（[Draft PR #1195](https://github.com/hat47x/tei-platform/pull/1195)）は、信頼済みPrivate checkout内の固定Git commitと、レビューした個別ファイルパス一覧から、**Git履歴を含まないローカル公開候補**を作る。外部ネットワークへの送信、Public Repositoryの作成、Git ref更新、`git push` は行わない。

- pathの明示allowlist方式とし、source tree全体を無条件でコピーしない。選別済みファイルが十分な依存閉包を持つかは、別途Build/Testで検証する。
- 候補はREADME/Licenceと、冒頭の生成AI開発・人的レビュー状況の表示がなければ拒否する。
- Git blob SHA照合、directory traversal・duplicate・symlink等の拒否は補助検査。機密性・権利・内容の安全性を判定するものではない。
- 候補に対するsecret/license/PII scan、全ファイル内容の人的確認、Source全体のテスト、公開用CI、保護設定は独立Gate。
- 2026-10-11時点の合成Repositoryでの隔離テストはPASS。ただし正規TEI checkoutのfull health、公開前監査、別Public Repository作成は未実施。

### 5.2 公開後の正本とIssue/CIの接続

公開用Sourceだけを新Repositoryへ配っても、Private RepositoryのPRのHEADでPublic Actionsを動かしたことにはならない。Private側とPublic側ではGit commit SHAもGit historyも異なる。

そのため次の方式を比較して運用を選定する。

| 方式 | CI・Issueへの効果 | 留意点 |
| --- | --- | --- |
| 既存RepositoryをPublic化 | 既存Issue/PRの正本とPublic CIが同一Repository | 開発履歴、Issue、PR、Actions logs等の公開範囲を十分に審査 |
| 新Public Repositoryを開発正本にする | Cutover後のIssue/PRは最初からPublic CIで資格確認できる | Privateにある大量のIssue/PR、進行中ブランチ、引用SHAを計画的に引き継ぐ |
| Private開発＋Public配布 | Private作業・資料を維持し、承認済みSourceのみ公開可能 | Private PRの統合資格をPublic CIから自動判定できない。公開配布は遅延しやすい |

いずれも、公開用リポジトリの名前、既存URL・Issue番号の継続要件、保護する研究データ、公開する開発履歴、公開後のIssue SSOTをownerが確定する。既存Private Repositoryを安易にrenameしてURL redirect/remoteへのpush先を変えない。

### 5.3 公開する前の必須確認

1. 全非公開Source・GitHub面を監査し、公開可能な正本集合を固定する。
2. Reference Modelや外部コンテンツの著作権・再配布条件、READMEのレビュー状況表示、機密情報Scanを確認する。
3. Candidateの内容を独立にレビューし、必要なGo/CLI/Webのテストを**Candidateの実ファイル**に対して実行する。
4. 新Repositoryの場合はSourceの旧→新revisionとfile digestsの照合をPrivate記録に保持する。Privateの過去Evidenceを新公開SHAのPASSに読み替えない。
5. 公開後はGitHub-hosted runnerで実際のcheckoutとfull Repository Healthを行い、必要なbranch protection/required statusを確認する。

本節およびPR #1195は公開操作そのものを認可しない。

### 公開ドキュメントの構成・整理（#1197）

Public向けには現行Product/Requirements/Architecture、使用・参加ガイド、正本ADR、再現可能なReference/Evidenceを優先する。内部の検討経緯や古い監査記録は、日付だけで機械的に削除せず、必要性・権利条件・参照関係を確認して掲載を判断する。

- 2026-10-11のmainにはMarkdown 482件（`plan/` 352件、`docs/` 73件）。
- 内容が別ファイルへ移った互換案内38件（`plan/` 22件と`docs/adr/` 16件）の参照元・正本先を[Draft PR #1195](https://github.com/hat47x/tei-platform/pull/1195)の選別表に記録した。
- 新Public Repositoryへ選択済みファイルのみを導入する場合に限り、**正本が同梱され、残存リンクが検証された後で**これらの互換案内を公開候補から除外する。
- Private開発Repositoryの旧URL案内を削除するかは別の判断。既存[Draft PR #1091](https://github.com/hat47x/tei-platform/pull/1091)の内部参照監査と外部互換性を踏まえ、現時点では維持する。
- 選別器・テストスクリプトの追加だけで実TEI全件に対する安全な削除を証明したとは扱わない。公開前にtrusted checkoutで検査する。

## 6. P0-C Public向けGitHub Actionsと保護

- [ ] Public化に伴うpush ruleset無効化など、Settingsの変更前後の差分を記録。
- [ ] ActionsはGitHub-hosted standard runnerで `permissions: contents: read`、`pull_request`を基本、Forkからの未信頼SourceへSecretsやwrite tokenを渡さない。`pull_request_target`で未信頼コードを実行しない。
- [ ] `bash tools/repository-health-check.sh` をfull modeで起動し、`--fast`やisolated Go testを正規資格に格上げしない。
- [ ] PRの自動test merge commitとexact candidate HEADを区別。#972のpre/post clean、main ancestry、runner blob、Receipt等を保持し、独立レビューと現在のPolicyによるmerge許可を分ける。
- [ ] mainのrequired checks、branch protection / rulesets、Draft PRの統合禁止、並行PRによるmain/head driftをfail closedにする。
- [ ] #513の過去runner起動前エラーと区別し、実際のcheckout・test step・full PASSを観測するまではCI復旧未達。
- [ ] Workflow/runner・外部Action/依存物・fork時の秘密保護をレビューしてから停止中のjobを有効化。Privateで利用枠を消費するような再起動はしない。

## 7. 公開へのGo / No-Go

**公開しない**条件:
- full-history / non-Git公開面の監査が欠けている
- 資格情報・第三者著作物・個人情報の重大な未解決問題がある
- #986の文書・リンク品質と#1190の公開構成が未判定
- GitHubの保護・CI再開の見通しと必要権限が確認できない
- Repo ownerによる公開可否の最終判断がない

公開に進む場合は、直前に同じGitHub stateでこれらを再確認し、公開後のbranch protection・read-only CI・actual runner testを再監査する。作業の承認と実施は別の操作として記録。

## 8. 参照

- [GitHub: Setting repository visibility](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/managing-repository-settings/setting-repository-visibility)
- [GitHub: Removing sensitive data from a repository](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/removing-sensitive-data-from-a-repository)
- [GitHub: Actions billing](https://docs.github.com/en/billing/concepts/product-billing/github-actions)
