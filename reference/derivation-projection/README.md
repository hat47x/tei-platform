# D10 仮想項目のモデル表示・一時プレビュー・編集禁止・確定値の接続 Reference（#1707）

**状態: 合成Fixtureの読取り専用Reference。Editor、ER Editor、Reactive Engine、Command/Transactionを実装しない。**

このパッケージは、統合モデラー／標準業務画面へ導出項目を投影するときの最小契約を固定する。四つの状態（意味の宣言、未保存プレビュー、実行時導出値、確定保存値）を混同すると、モデルの正本や確定業務事実が壊れる。

## 区別する四つの状態

1. `declared`: Authoritative Sourceにある規則とBinding。実行結果ではない。
2. `preview`: 未保存入力から求める一時評価。計算に成功しても保存・確定ではない。
3. `evaluated`: 受理済み規則を現在の文脈・認可で評価した表示値。保存可能な業務属性と混同しない。
4. `confirmed`: 確定Actionが保存した事実。再描画・入力変更・規則更新で勝手に上書きしない。

これに `input-unavailable` / `evaluation-failed` / `authorization-denied` を加える。`Value.Amount` は `preview` / `evaluated` / `confirmed` のときだけ設定し、未取得・失敗・拒否のセルは値を持たない。古い600を新しい結果として見せない。

## 最小Projection契約

- `Field` は `physical-stored` / `derived` / `confirmed-stored` を区別し、`Writable()` は derived で常に false。Widgetの具体形は #1124 / #997 へ委譲する。
- `WriteTarget` はUIではなく**Action／保存境界**で導出セルへの直接書込を拒否する。エディタを隠すことだけを防壁にしない。
- `RuleEdit` は規則の意味の変更であり、値の書込とは別の型として扱う。Proposal→Acceptance→Source（#1124 / #267）の既存経路を消費し、第二Editorを作らない。
- `Reproject` は規則版変更後も `Field.ID` を保つ。生成物IDの変化を業務属性の別IDとして偽装しない。
- `Renderer` は `MeaningState` を受け取り、それぞれの規則を保持・実行しない。Web/CLIで同じ意味状態を渡せる。

## 合成受注明細Fixture

| 条件 | 期待 | 試験 |
| --- | --- | --- |
| 数量2・単価300→一時表示600、未保存で数量3→プレビュー900 | 保存事実は不変 | `TestD10PreviewDoesNotChangeConfirmedFact` |
| 導出金額セルへ直接999 | 編集禁止（UIでなく保存境界で拒否） | `TestD10DerivedDirectWriteRejectedAtBoundary` |
| 未取得単価で再評価不可 | 古い600を新結果として継続表示しない | `TestD10UnavailableDoesNotShowStaleValue` |
| 確定600の後にマスター単価300→350 | 保存済み600は不変、新規未確定の計算のみ700 | `TestD10ConfirmedSurvivesMasterPriceChange` |
| 認可拒否・計算エラー | `authorization-denied` / `evaluation-failed`、値は表示しない | `TestD10DeniedAndFailureStates` |
| Web Rendererを案件UIへ変更 | 意味・状態・編集不可の契約は同一 | `TestD10RendererPortability` |
| 規則版変更後の再投影 | `Field.ID` を保つ | `TestD10ReprojectionKeepsFieldIdentity` |
| 規則そのものの変更 | 値の書込とは別経路 | `TestD10RuleEditIsNotAValueWrite` |

## 焦点試験

```sh
gofmt -l reference/derivation-projection/*.go
go test -count=1 ./reference/derivation-projection
go vet ./reference/derivation-projection
```

## 未実装・依存

- 規則と入力は #1637、適用先解決は #1692、計算値・確定保存値の区別は #1641、項目権限は #1693、根拠の診断表示は #1696 が所有する。
- 統合モデラーの編集／Source反映は #1124、ERとFieldは #1126、標準CRUD・保守画面は #1127、UI Composition Kitは #997、AI提案／受理は #267 が所有する。
- 標準UI一式の開発、ER Editor新設、リアルタイム共同編集、Command/Transaction実装、汎用Reactive計算エンジンは行わない。
- 合成Fixtureのみ。実Runtimeの一画面接続（M2）、案件固有UIによるOverride（M3）は未検証。未検証を明示し、契約試験だけを実証済みと扱わない。

実行の有無、exact SHA、成功／失敗／未検証を Issue #1707 と対象PRへ記録する。
