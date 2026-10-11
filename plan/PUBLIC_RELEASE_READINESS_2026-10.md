# TEI Public化準備・公開判定Runbook（候補）

- 日付: 2026-10-11
- 判定: **NOT READY — 未監査のままPrivate→Publicを実施しない**
- Parent: [#1187](https://github.com/hat47x/tei-platform/issues/1187)
- 機密性・非Git公開面: [#1188](https://github.com/hat47x/tei-platform/issues/1188)
- CI・保護設定: [#1189](https://github.com/hat47x/tei-platform/issues/1189)
- 人物帰属・履歴移行: [#986](https://github.com/hat47x/tei-platform/issues/986)、[Draft PR #1164](https://github.com/hat47x/tei-platform/pull/1164)、[#1190](https://github.com/hat47x/tei-platform/issues/1190)
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
| 履歴の人物帰属 | #986の調査でmain祖先19コミットの名前入りmessageが報告済み。全refsの検索は未了 |
| サーバー保護 | 過去の#513/#972でmain無保護が記録された。Private rulesets APIは現プランで403 |
| 全履歴・非Git面 | **未監査**。この一覧から公開可とは推論しない |

## 3. P0-A 機密性・権利・公開面

- [ ] 信頼済みHostでprivate mirror cloneを取得し、全refs（active branches、tags、PR refsを別途含む）と `git rev-list --objects --all` を列挙。正規Git/Goの動作確認と混同しない。
- [ ] バージョン固定済みのGitleaks等を使い、Git履歴と作業ツリーで秘密情報を検査。生成報告書・検出値・未公開pathはRepository外へ保存し、Issueへ秘密そのものを書かない。
- [ ] 個人情報、社内/顧客情報、credential、endpoint、private URL、実名を含む不用意な帰属・評価をhuman triageする。
- [ ] Issue/PR/reviews/commentsと添付、全過去Actions logs/artifacts、Release/assets、Wiki/Pages等も対象にする。API取得不可なら `NOT_INSPECTED`。
- [ ] `LICENSE` とThird-party source/Skill/assets/資料のcopyright・再配布条件を照合。#941の参照モデル原物・抜粋・変換物等の再配布範囲は独立審査。
- [ ] 問題検出時はキー無効化/ローテーションや情報源除去を先に計画し、安易なshared history rewriteを実施しない。
- [ ] 安全なscope・scanner version・refs数・objects数・未解決件数・人間reviewerだけをpublic Evidenceへ出す。

## 4. P0-B 外部設計者への言及とコミット履歴

原則: 出典の著者表示・原著作権のNoticeは保持。TEI固有の技術的評価を出典著者の個人的意図やお墨付きと混同しない。特定個人名をタイトル/ファイル名/仕様の主語に不必要に置かず、一次資料の確認できる事実とTEIの推論を区別する。

1. 現行treeの表現・ファイル名の修正は**既存Draft PR #1164**で進行する。別PRに重複実装しない。
2. 過去commitの名称を含む件数はGitHub検索で把握したが、必ずtrusted full historyで `git log --all` のsubject/body/trailerを再監査し、GitHub commit comments（別リソース）も確認。
3. ユーザーは一定のリスクを伴っても履歴表記を是正する方向を希望。ただし共有mainに19件以上の対象があり、大量の子コミット/branch/PR/Releaseやexact-head Evidenceへの参照が変わるため、即時の部分force-pushは認めない。
4. #1190で **(A) 全refsの履歴再構築** と **(B) Private履歴を保持し新たなClean Public Repositoryへ既存ソースを初回import** を比較し、公開URL/Issue番号/PR継続の要件も評価。GitHubではrewrite後も古いPR refs / cached SHAや外部cloneに過去commitが残り得る。GitHub Supportによるnon-sensitive content削除は期待できない。
5. Aを採用する場合は開発laneのfreeze、Private mirror backup、reproducible filter、全コミットの旧→新SHA対応、PR/refs再作成、資格Evidence再取得、復旧計画、最終Human reviewが先行条件。過去の検証receiptを変更後headへ機械転用しない。
6. Bを採用する場合はPrivate/公開RepositoryのSource同期方式、Issue運用の正本、GitHub Actionsの資格の対応を明示する。Clean Public exportは元RepositoryのGit lineageを自動保持しない。

## 5. P0-C Public向けGitHub Actionsと保護

- [ ] Public化に伴うpush ruleset無効化など、Settingsの変更前後の差分を記録。
- [ ] ActionsはGitHub-hosted standard runnerで `permissions: contents: read`、`pull_request`を基本、Forkからの未信頼SourceへSecretsやwrite tokenを渡さない。`pull_request_target`で未信頼コードを実行しない。
- [ ] `bash tools/repository-health-check.sh` をfull modeで起動し、`--fast`やisolated Go testを正規資格に格上げしない。
- [ ] PRの自動test merge commitとexact candidate HEADを区別。#972のpre/post clean、main ancestry、runner blob、Receipt等を保持し、独立レビューと現在のPolicyによるmerge許可を分ける。
- [ ] mainのrequired checks、branch protection / rulesets、Draft mergeの失敗例、並行PRによるmain/head driftをfail closedにする。
- [ ] #513の過去runner起動前エラーと区別し、実際のcheckout・test step・full PASSを観測するまではCI復旧未達。
- [ ] Workflow/runner・外部Action/依存物・fork時の秘密保護をレビューしてから停止中のjobを有効化。Privateで利用枠を消費するような再起動はしない。

## 6. 公開へのGo / No-Go

**公開しない**条件:
- full-history / non-Git公開面の監査が欠けている
- 資格情報・第三者著作物・個人情報の重大な未解決問題がある
- #986の前方変更と#1190の履歴方針が未判定
- GitHubの保護・CI再開の見通しと必要権限が確認できない
- Repo ownerによる公開可否の最終判断がない

公開に進む場合は、直前に同じGitHub stateでこれらを再確認し、公開後のbranch protection・read-only CI・actual runner testを再監査する。作業の承認と実施は別の操作として記録。

## 7. 参照

- [GitHub: Setting repository visibility](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/managing-repository-settings/setting-repository-visibility)
- [GitHub: Removing sensitive data from a repository](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/removing-sensitive-data-from-a-repository)
- [GitHub: Actions billing](https://docs.github.com/en/billing/concepts/product-billing/github-actions)
