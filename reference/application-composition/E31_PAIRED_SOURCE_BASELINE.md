# E31：SUI利用先の構成作業を比較する際は、TEIとSUI双方の版を照合する

- **管理**：[Issue #1016](https://github.com/hat47x/tei-platform/issues/1016)。
- **前提**：[E25のGit改訂比較](E25_CROSS_REPOSITORY_REQUALIFICATION.md)・[E27の選択監査](E27_RESIDUAL_SELECTION_AUDIT.md)・[E28の作業申告計測](E28_WORKFLOW_MEASUREMENT.md)・E29の未完了比較拒否。
- **実装**：[E31の読取り比較](e31_paired_source_comparison.mjs)と[Node回帰試験](e31_paired_source_comparison.test.mjs)。新しい版固定台帳、正本Source、Plugin LockやResolverを作らない。
- **検証用TEI基点**：`d3035976455e165ff7fc566c9c12429070fcca2f`。実際のSUI改訂対はE25で観測した`72368e7c1822fa894e2323a38f207117a521d2ed` → `4bd493f9942890d52117461886207d628f3554e2`。改訂対の**比較条件**を試験に再利用するもので、現在のSUIの受理契約・配備版を示すものではない。

## 確認した不足

既存のE28の作業申告は`scenario.baseRevision`というGitコミットを一つ持つ。これはTEI側の同じSource改訂を比較するには役立つが、**SUI Native利用先は実際にはTEI側のStage 0参照と、別リポジトリのSUI Native実装を使う**。TEI側だけが同じ改訂であっても、SUI側が更新されていれば、二つの導入方式を「同じ出発点で比較した」とはいえない。

既存E28の`compareE28`は利用先、案件ID、開始Git SHA、Target、作業申告が完了しているかを検査するが、別リポジトリの開始改訂を比較する項目がない。この不足があるまま`reportedDelta`を業務導入の同条件比較に使うことを避ける。

## 最小の確認方法

E31は既存E28の報告と、比較時に**明示的に渡された版の対応**だけを照合する読み取り専用の関数である。`claimedRevisions.existing`と`claimedRevisions.candidate`には`tei`と、SUIが関わる場合`sui`のGit SHAを指定する。

| 検査条件 | 判定 |
| --- | --- |
| E28の受理済み比較条件が揃い、TEIの版とE28の`baseRevision`が一致 | TEI基点は申告上対応する |
| SUIの利用先で`sui`の版が欠落している | `not-comparable` |
| TEIの版は同じでも、SUIの版が異なる | `not-comparable` |
| Web受注明細でSUIへ依存していない | SUI版を必須にしない。明示された場合は一致を要求 |
| 両版が申告上一致しても、E28が部分記録・比較拒否 | `not-comparable` |
| 版とE28申告条件が揃う | `reported-paired-source-only`。実配備・適合・現在権限の証拠ではない |

SUIの二つのGit SHAは、単に**呼び出し側の申告**であり、E31がGitHubへ照会したりGitオブジェクトを検証したりするわけではない。比較結果には常に`sourceRevisionsVerified:false`、`runtimeAuthorityVerified:false`、`verifiedImprovement:null`を付ける。実際の版対応と依存範囲の完全性は、既存のSource Index、Git出典、Build/Release、Plugin Lock等の正本で別途確認する。

## 回帰試験

```sh
node --test reference/application-composition/e31_paired_source_comparison.test.mjs
```

E25で記録したSUIの旧→新改訂を、TEIの固定改訂と組み合わせて入力する。E28の1コミット指定が一致して見えても、SUI改訂が違えば拒否する負例が中核となる。ほかにSUI版欠落、TEI改訂の不整合、部分記録、異なるConsumerを拒否し、SUIに依存しない受注明細には不要な版を求めない。

## 設計判断

**新しいComposition Profileの製品採用は引き続き保留**。E31は別リポジトリを含む比較の必要条件を一つ明示しただけで、仮の構成宣言や第二の版固定帳票を新設していない。真のSUI Actionの入力意味・本人委譲と保存の成立、現在の権限、実Web／Native Target、人的工数とAI質問の実測は未解決。

並行する[E30 PR #1626](https://github.com/hat47x/tei-platform/pull/1626)は、E28既存計数器の出典粒度と項目別の未測定値を修正する別担当作業である。E31はそのファイルへ書き込まず、**既存の公開関数を呼び出すだけ**に留める。
