# E12：E7実診断とE11再レビュー候補を同じSource報告へ結ぶ

- **状態**：読み取り専用の局所接続Reference。実Go modelref統合、アプリケーション実行、採用済み業務意味、Runtime Authority、全依存閉包を示すものではない。
- **管理**：[Issue #1016](https://github.com/hat47x/tei-platform/issues/1016)／[Draft PR #1051](https://github.com/hat47x/tei-platform/pull/1051)。
- **前段**：[E7のSource Index対比較](E7_PAIRED_OBSERVATION.md)と[E11のImpactレビュー](E11_TYPED_IMPACT_REVIEW_GATE.md)。関連する型付き参照・Impactの正本候補は[#1120 / PR #1139](https://github.com/hat47x/tei-platform/pull/1139)。

## 実装した接続

[E12の接着関数](e12-bound-review.mjs)は、変更前後のE7 Envelope（各々E5 ReportとSource Indexの組）、選択Application／scope、別の型付きImpact候補を受け取り、以下を行う。

1. 変更前後のE5 Reportと、明示されたApplicationの同一性を確認。別ApplicationのReportを選択結果へ流用しない。
2. **E7の実関数 `assessPairedEvidence(baseline,current)` を呼ぶ**。Source ReportとIndexの定義出典・指紋・対象集合の照合はE7→E6→E4に任せる。E12はここに独自のIndex比較器を持たない。
3. **E11に渡すApplication→Pattern所有情報は、E7で検査対象とした `current.report` から直接取得**する。別々に渡された`sourceDrift`や独立した`sourceReport`を利用しない。以前のE11単体のCallerが異なる時点のReportを取り違える余地を、少なくともこの接続APIでは閉じる。
4. E11に既存の型付きImpact候補を渡す。関係するPatternとApplicationの候補だけを返す。E7のSource変更・定義不一致があれば、型付きImpactが空でも`unresolved`を保持する。
5. どの状態でも`authority:none`／`adoption:not-decided`／`fullClosure:false`を維持し、`evidenceBoundary:unverified-caller-supplied-snapshots`と`modelrefSnapshotBound:false`を返す。

**重要**：E7のJSONは同一Workspaceから生成されたと**形式上称している**だけであり、形式検証だけでは改ざんや別時点の混入を検出できない。E12の「束ねた」という語は、同じ**入力Envelope**の中のSource Reportを再利用するという意味に限定する。#1120の`typedProjection`がそのSourceと同じWorkspace revisionから生成された証拠も、現段階では無い。

## 実装した負例

| 操作 | 判定 |
| --- | --- |
| E5 ReportとIndex／Application／Pattern所有に矛盾なし | `limited-unchanged`（ただし限定Source集合のみ） |
| 既知のRelationの型付きImpactがApplication所有Patternに到達 | `review-candidate`（適用許可ではない） |
| PatternのSource指紋や依存集合が変わった | `unresolved` |
| Source Reportが別Applicationを名乗る、Formatを偽装する、定義SourceとIndexが食い違う | `unresolved` |
| E7が認識しないRelationをReportに挿入する | `unresolved` |
| Callerが偽`sourceDrift`や別の`sourceReport`を同時に渡す | E12は**それらの外部入力を受け取らず**E7実結果を使用する |
| #1120候補に未解決の型付き参照がある | `unresolved` |
| 無関係なSource Index Artifactだけが変わる | 列挙済みの直接依存について`limited-unchanged`のまま（完全性は非主張） |

## 実際の局所実行と出典

```sh
node --test reference/application-composition/e12-bound-review.test.mjs
```

2026-10-10、Node.js **22.16.0の標準`node:test`**で**15件成功、失敗0件**。受け渡し先のGitHub原文を一切編集せず、同一の隔離ディレクトリへ再現して実行した。実行した6ファイルとGitHubのGit blob SHAはそれぞれ一致した。

| ファイル | Git blob SHA |
| --- | --- |
| `e4-index-scope.mjs` | `d2a61dac60b1641e453513e3f72b7c6c2ef43501` |
| `e6-typed-index-bridge.mjs` | `9ca7f39a91a89503a7955940b254754b02581666` |
| `e7-paired-evidence.mjs` | `910ef058ed96d5830f026c0552196682580a20eb` |
| `e11-review-gate.mjs` | `afe66251af87a9cfe587aa99030539ad4453c17f` |
| `e12-bound-review.mjs` | `3e53b85f970900279ede3f8e47d04a79075aabd0` |
| `e12-bound-review.test.mjs` | `7274756d7be9f02b59c4fb14fc38126c5ccb3241` |

これは**実E7/E11/E6/E4モジュールのJavaScript結合テスト**ではあるが、**E5 Goの実Workspace読取り、#1120 Go Impact出力、実SUI、実Runtimeの結合テストではない**。テスト対象は合成JSONである。クリーンなfull checkoutのexact-head `tools/repository-health-check.sh`、Go test、go vet、gofmt、実Workspace CLI、実行前後cleanの証拠は未取得。Draftを保持する。

## 次のGate：Go側の同じsnapshotから証跡を供給する

E12はScope/所有者の推測を増やさず、**どこで実証が不足しているかを公開すること**を目的とする。Product用に進めるには、次の三つが必要となる。

1. `workspace.Load`＋Source Index生成と`index.Build(loaded.World)`＋`derivedbridge.FromIndex`＋`Validate`＋`Impact`を、同じ解析済みWorld／同じ固定revisionから実行する**信頼されたGoの呼出し元**。既存#1017のLoader解析bytes対策も確認し、別のResolverを増やさない。
2. 対象Applicationとtyped scope、業務/Nativeの所有関係を、既存の型付きWorld／Declared Edgeから照合する。E11の`sourceReport`と#1120候補の`nodes`が、同じ時点・完全な所有関係であることを実データで確認する。
3. 実`order_entry`でRelation変更によりPattern・Applicationの再レビュー候補が出ることと、実SUIではまだ完全なTEI Applicationになっていない不足を、同じ基準で明示する。

Goの`#1120`は別Draftであり、承認されていない製品契約をこのブランチから直接importする新しいGo product実装はまだ作らない。これにより、並行ブランチのCoreへの依存や二重Graphを増やさず、次の資格確認に必要な**結合点**を限定する。
