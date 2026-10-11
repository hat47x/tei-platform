# 公開前Git履歴の機密情報スキャン

- 実施日時: 2026-10-11 16:14 JST
- 対象: `hat47x/tei-platform` のGitHub remote refs全件を取得した一時mirror
- mirror時点の`main`: `e9589c8814077c96d20469e352b31c9dc604c92b`
- 判定: **自動スキャンと初期トリアージ完了。独立した人間レビュー待ちのため、公開Gateは未完了。**

## 対象と完全性

- `git ls-remote --refs origin`とmirrorのref名・object IDを照合し、一致を確認した。
- 1,081 refs: heads 71、tags 4、その他1,006（主にGitHub管理のPR refs）。
- `git rev-list --all --count`: 11,880 reachable commits。
- `git rev-list --objects --all`: 46,016 reachable objects。
- `git fsck --full --no-reflogs --unreachable`: exit 0、診断出力なし。
- 実行後にもremote ref一覧を再取得し、mirrorと同じ1,081件・同じobject IDであることを確認した。

## スキャナーと実行

- Gitleaks v8.29.1。GitHub公式releaseのLinux x64 archiveを取得し、同梱checksumを照合した。archive SHA-256: `e4eb209d04e20339d77122a3bdf9cd41351255cfb27ebcb75e85325e04f88924`。
- スキャン前に隔離した合成RSA秘密鍵で検知動作を確認し、1件を検出した。合成値と報告ファイルはRepository外に置いた。
- 全refsとmerge commitの各親との差分を対象に `gitleaks git --log-opts='--all --full-history -m' --ignore-gitleaks-allow --redact` を実行した。
- Gitleaksログ: 11,507 commits scanned、約916 MB、2件の検出。Exit 1は検出があったため。
- 候補値を含む機械出力はRepository外の制限付き一時領域に保存し、本記録には値・内容断片・候補値が推測できる情報を記録していない。

## 初期トリアージ

2件は同じGoテストの同じ比較式に対する重複検出で、merge差分にも現れたもの。該当箇所はテスト中の `database.OwnershipExternalObserved` 等の型付きenum識別子であり、API keyや資格情報ではないことをソース上で確認した。資格情報の失効・ローテーションは不要と判断する。将来の検出を抑止するallowlistは追加していない。

このトリアージは自動スキャナーとCodexによるソース確認であり、独立した人間レビューではない。公開前に人間reviewerが当該判断を確認し、Issue #1188へ安全な要約と担当者・判定を記録する必要がある。

## 未検査範囲

この走査はGit object本文の秘密らしい文字列を探すもの。commit author/committer metadata・commit messageの個人情報評価、Issue/PR/comment/review/添付、Actions logs/artifacts、Release assets、Wiki/Pages、第三者ライセンス・著作権・個人情報の人間確認は含まない。GitHub上のRefと履歴が取得時点以降に変わっていないことも、公開直前に再確認する。

したがって、この記録はGitHub全公開面の監査完了やPublic化承認を意味しない。関連: [Issue #1187](https://github.com/hat47x/tei-platform/issues/1187)、[Issue #1188](https://github.com/hat47x/tei-platform/issues/1188)。

## Latest refresh — 2026-10-11 18:45 JST

- Mirror source: `hat47x/tei-platform`; current `main` `6af87d84041e731d67634d887b62ceccd219723f`.
- `git ls-remote --refs` matched the mirror exactly: 1,053 refs (74 heads, 4 tags, 975 PR-head refs). `git rev-list --all --count`: 11,903 reachable commits; `git rev-list --objects --all`: 46,203 objects. After pruning obsolete PR merge refs and garbage-collecting only this disposable mirror, `git fsck --full --no-reflogs --unreachable` returned no diagnostics.
- Gitleaks v8.29.1 scanned 11,526 commits / about 1.03 GB across all refs with full history and merge-parent diffs. It reported 12 finding rows at 2 source-line sites in the same test file (`business_action_ownership_boundary_projection_pressure_test.go`, lines 86–87); no new rule or source site appeared. Values and report JSON remain outside the repository. These source sites still need independent human review.
- The earlier phrase “2件” described the 2 distinct source-line sites, not the 12 machine-report rows. It should not be interpreted as the total scan finding count.
- Candidate history at `c0a1b556659e1214b38cdb186e5b29437fd16d00` was scanned separately: 8,375 commits / about 754 MB, with 6 finding rows at the same 2 source-line sites. The directory scan of the candidate working tree reported 1 row at line 86, the same known site. The directory scan included the pending documentation dispositions but not later Git metadata; the exact machine reports remain outside the repository.
- Neither scan covers Git metadata privacy, Issues/PR discussions, workflow logs/artifacts, releases, Wiki/Pages, or human rights review.
