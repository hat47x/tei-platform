# GitHub上の公開面スナップショット（2026-10-11）

## 対象と結論

2026-10-11にPrivate repository `hat47x/tei-platform` のGitHub REST APIから、Issue/PR本文・コメントの一部とRepository/Actions設定を読み取り専用で確認した。これは公開前監査の一部であり、**公開可の判定ではない**。本文や添付ファイルを含む全公開面の人手確認、ReleaseとActionsログの内容確認、Ownerの最終判断は未完了。

## 観測した状態

| 対象 | 観測 |
| --- | --- |
| Repository | Private、default branch `main`、未アーカイブ、Issues有効、Wiki/Pages無効 |
| Issue API一覧 | 1,217件（PRを含む。IssueオブジェクトのPR識別子は986件） |
| Pull Request API / GraphQL | PR 989件。Issue一覧に対して3件の差があり、このsnapshotでは解消できていない |
| IssueコメントAPI | 1,265件を取得 |
| PR review comment API | 0件 |
| PR review本文 | GraphQLで989 PRすべてを走査し、16件のreview本文を取得。review page未取得なし |
| Releases | 4件。説明、添付assetの内容は未確認 |
| Actions runs | 776件（2026-08-30〜2026-09-22、すべてcompleted）。成功397、失敗378、skipped 1 |
| Actions artifacts | 現在のAPI一覧で0件 |
| Actions設定 | 有効、許可Actionはall、SHA pinning必須ではない |
| main branch protection API | 403。現Privateプランでは取得できず、保護設定を確認できない |
| Workflow | 手動起動のみ、job条件は常にfalse、`contents: read`。現在実行されるhosted jobはない |
| Git状態の再確認 | main `095e8c49b5a905f6c0e4ff15a709479efdeb3de1`、Open PR 0件、origin branch heads 77件の全commitをcandidate `a3068a194cf8dd331db3d0e546dbf6ee620c2004`が含む |

## 限定的な候補文字列走査

取得したIssue/PR本文1,217件とIssueコメント1,265件を対象に、メール形式、credential assignment、Bearer token、RFC1918 IPv4の簡易パターンを走査した。本文で2件、コメントで1件の候補が出た。文脈を確認したところ、いずれもスキーマの例・型識別子の一部であり、credentialや個人メールアドレスの露出とは判断しなかった。候補の実値は監査記録に複写していない。

この走査は汎用secret scannerや人手による全記録レビューではない。Release説明、Issue/PR添付、commit comment、外部連携先に複製された情報は未確認である。Issue一覧とPull Request API間の3件差も解消していない。

## Actions logとRelease assetの追加確認

- 776 runすべてのlog archive APIを照会。751件でZIP応答を取得し、うち512件にlog filesがあった。25件はAPIが404を返したため未取得として扱う。
- 取得できた1,738 log files（展開後合計約33 MB）を一時領域へ安全な相対pathで展開し、Gitleaks v8.29.1で走査。検出0件。
- 4 Releaseの各asset（計4件、gzip tar）を取得。全アーカイブのpathを検査し、不正な絶対path・親ディレクトリ参照は0件。展開物を同じGitleaksで走査し、検出0件。
- GraphQLでPR 989件のreview summaryを全件取得（本文16件、追加ページなし）。本文の同じ候補パターン走査では該当0件。PR review comment APIは0件。
- 走査はsecret検知に限る。ログ・Evidence・Releaseの意味内容、第三者権利、個人情報の全件人手レビューはしていない。404となった25 runのlogsも未確認。

取得したAPI応答、一時ファイル、検知報告はrepository外の `/tmp/tei-public-audit/` に保管している。認証情報や候補値は出力・commitしていない。

## 必要な継続作業

- Issue/PR本文・コメントを人が公開適合性確認し、添付の有無と内容も確認する。
- commit commentsとRelease説明を取得して確認する。Issue/PR添付もAPI外の参照先を含め確認する。
- 404の25 run logsを取得可能な経路で確認する。Actionsのartifactは現時点で0件だが、workflowごとの権限・fork動作も履歴上のworkflow定義と照合する。
- Issue/PR本文・コメント、取得済みActions log、Release説明・assetについて独立した人手レビューを行う。今回のsecret scan PASSを内容・権利審査の代用にしない。
- branch protection / rulesetsは取得可能な権限・プランで再確認し、mainの必須checkとDraft禁止方針を確認する。
- 独立した人間レビュー後にのみ、公開前判定を更新する。
