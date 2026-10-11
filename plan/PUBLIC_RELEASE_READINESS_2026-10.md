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
| README | 生成AIによる開発・体系的な人的レビュー未実施の警告はmainの冒頭に掲載済み |
| `SECURITY.md` | mainに存在しない。連絡窓口の所有者判断が必要 |
| 第三者モデル | #941の安全側公開境界を維持。原モデル/手引の直接再配布は許諾確認まで禁止 |
| Git履歴の公開適合性 | mainの一部コミットの参照関係を確認。全refsのmetadata/contentの網羅検査は未了 |
| サーバー保護 | 過去の#513/#972でmain無保護が記録された。Private rulesets APIは現プランで403 |
| 全履歴・非Git面 | **未監査**。この一覧から公開可とは推論しない |

## 3. P0-A 機密性・権利・公開面

- [ ] 信頼済みHostでprivate mirror cloneを取得し、全refs（active branches、tags、PR refsを別途含む）と `git rev-list --objects --all` を列挙。正規Git/Goの動作確認と混同しない。
- [ ] バージョン固定済みのGitleaks等を使い、Git履歴と作業ツリーで秘密情報を検査。生成報告書・検出値・未公開pathはRepository外へ保存し、Issueへ秘密そのものを書かない。
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

## 5. 現行Sourceの個別評価と開発運用の継続

### 5.1 個別のファイル・資料・ツールの評価

公開候補の許可リストによる抽出・生成方式は採用しない。PR #1195および関連PR #1201は未マージでクローズし、公開の作業手順に含めない。

現行mainのファイルを、利用者・開発参加者にとっての必要性、コード・テストの依存、出典・再配布条件、秘密情報、現在の説明と旧資料の重複から個別に判断する。まとめて扱う場合も、対象パス・ファイルの内容と役割を確認する。

- 維持するもの：CLIとGo Runtime、意味モデル、`go test ./...`等の必須テストと入力データ、現行の製品価値・要求・Architecture・ADR正本、実行可能なReference/Evidence。
- 整理候補：過去のSource再構成断片、一時的な運用補助、現在の正本と重複する作業資料、TEIの機能・開発検証に不要な第三者のAI向けSkill。
- 単にファイル数を減らすためにGoテストや必要なEvidenceを失わない。テストが失われた状態の`go test`成功を、削除前と同じ回帰検証と扱わない。
- 削除候補はそれぞれ独立PRに分け、差分と参照の確認、正規Repository Healthを実施してから統合する。PR #1202は旧bootstrap断片、PR #1203は用途が異なる第三者Skillの整理。
- 旧配置の互換案内は、内部と外部からの参照を確認できるまで維持する。正本文書への導線の改修はPR #1091、#1198、#1199、#1200を参照。

### 5.2 公開するRepositoryとGit履歴

主な想定は既存`hat47x/tei-platform`をPrivateからPublicへ移行すること。現行HEADからの削除だけでは過去のblobやcommit metadata、PR refs等は消えないため、公開前に**全履歴・全公開面の監査**を行う。

必要なら別Public Repositoryという選択肢も比較するが、生成ツールによるSource抽出を前提としない。別Repositoryを選ぶ場合は、Issue/PR正本の切替えとGit SHA・検証資格の非移譲を明示的に扱う。

### 5.3 公開する前の必須確認

1. 現行mainを含む公開され得る全Git refs、Issue・PR・comments・Release・Actionsログ等を調べ、秘密情報・個人情報・機密資料を検査する。
2. Reference Modelや外部コンテンツの著作権・ライセンス・再配布条件を確認する。未検証の資料や古いGit blobの扱いも対象とする。
3. 各整理PRについて、実ファイルの正本性・参照・テスト依存・変更多発箇所を検査し、現在のブランチHEADで正規Repository Healthと前後のclean状態を確認する。
4. READMEの生成AI開発・体系的な人的レビュー未実施表示を維持する。古いGateのReady判定を最新mainの品質保証に置き換えない。
5. Public化前後にGitHubのbranch protection、required checks、Secrets不渡しとread-only Actionsを実機で確認し、GitHub-hosted runnerでのfull Repository Healthを観測する。
6. Ownerによる公開可否の最終判断を得る。未完了の監査項目がある場合はPrivateを維持する。

### 公開ドキュメントの整理

現在のProduct/Requirements/Architecture、使用・参加ガイド、ADR正本、実証に必要なReference/Evidenceを入口に据える。歴史的監査記録や旧計画は、単に古いという理由で消さず、個別に必要性と参照を確認する（#1197）。互換用の移動案内は必要な参照を確認するまで維持する。

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
