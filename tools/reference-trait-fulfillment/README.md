# 原子的Traitの実装義務と履行判定 Reference（#3279 P0-F）

本ディレクトリは、**Traitが明示した実装義務が満たされているかを、既存のContract・Provider・観測から読取り専用で判定する小さな参照**です。正式なTrait Schema、Resolver、Registry、Validator基盤、Core状態enum、Host、受入済みAPIではありません。実装は`traitfulfillment.go`、受入は`traitfulfillment_test.go`です。Sourceや設定を書き換える経路はありません。

設計の出典は[原子的Traitの実装義務・合成・履行検証](../../plan/design/TRAIT_ATOMIC_COMPOSITION_AND_FULFILLMENT_2026-11.md)の§18.2・§21.4（P0-F）です。

## 何を原子的Traitとして扱うか

新しい型は作らず、**既存の`composition.Contract`一つ**を、一つのSubjectと一つのScopeへ付けたものをTrait要求として扱います。最初の一件は著作用InspectorのContract `tei.operations.inspect`（v0）で、不変条件は`operations.ValidateInspectionProjection`です。

これを原子的とみなす理由は次の三つです。

- Operationが一つで、要求・結果・不変条件がContractの`Shape`一つに固定されている。
- 不変条件を一回の呼出しごとに機械で検査できる。
- 既存のProvider選択と実呼出しの観測がすでにあり、宣言と実行を別々に確かめられる。

## 再利用するものと、足した入力

| 判定材料 | 出所 |
| --- | --- |
| Providerの有無・競合・依存不足 | `composition.Registry.InspectWiringSet` |
| 選択されたProviderの実コードへ入ったこと、Workspaceの出所 | `inspectoradapter.LocalInspectorHost`の`Invocation` |
| 不変条件の検査 | `inspectoradapter.Selected.Inspect`が適用する`ValidateInspectionProjection` |
| **読込みの完全性（本参照で足した唯一の入力）** | 呼び手が渡す`Closure` |

Registryは「見つからない」を常に`unknown`と答えます。未読込みと不在を区別できないためです。本参照は、呼び手が**Source・Required集合・Providerカタログをどこまで読み込んだか**を`Closure`で受け取り、その範囲でだけ不在を結論します。`Closure`の真偽は検証しません。

## 判定する意味（採用前）

| 条件 | 結果 |
| --- | --- |
| Sourceとカタログが完全で、`manual-required`の実装だけがない | `unfulfilled`。同じContractの別Providerは`NotChosen`へ並べ、代用しない |
| Sourceが途中、カタログ未読込み、カタログの版が古い、Registryがない | `unknown` |
| Required集合が一部しか取得できていない | 確認済みの欠落は`unfulfilled`のまま報告。残りが宣言済みでも全体は`partial` |
| ProviderがContractを宣言しているだけ | `declared-unverified` |
| 署名は同じだが別Subjectの答えを返す | `declared-unverified`（不変条件が拒否） |
| 手動実装のIDに別のコードが入っている | `declared-unverified`（入ったコードが期待と違う） |
| 呼出しはあったが不変条件を検査していない | `declared-unverified` |
| 選択した実装のコードへ入り、不変条件を通った | `invariant-observed`。`Observed`に見えた事実だけを並べる |
| `provided`で同じContractのProviderが二つ | `conflict`。どちらも選ばない |
| `advisory`の実装がない | `not-provided`。全体の結果に影響しない |
| 同じ要求の重複、逆順、`required`と`advisory`の併記 | 結果は同じ。`required`は弱まらない |
| 同じSubjectの同じTraitへ異なる実装方式 | 順序によらず`conflict` |

状態の語は本参照の説明用の文字列です。

## 主張しないこと

`invariant-observed`は「その一回の呼出しで、選択した実装のコードへ入り、不変条件を通った」ことだけを示します。次は別の資格であり、本参照の結果から導きません。`Report.NotQualified`にも同じ内容を返します。

- 正式ProviderのArtifactを製品のConsumerが実際に呼んだこと（#1251）
- Provider・Consumer・Targetを固定したBOMの検証（#3267）
- 現在のHost Grant、開示、Effect（#1174）
- Ownerによる意味の受理

## 現行の契約に足りないもの

本参照が呼び手の申告や手書きの期待値で補っている部分です。正式化する場合の不足事項として残します。

1. **読込みの完全性を示す既存の出力がない。** `ValidationReport.Incomplete`と未読Sourceの警告はSource側の部分的な信号で、Required集合とProviderカタログの完全性を示すものはありません。`Closure`は申告のままです。
2. **TraitをSubjectへ付ける宣言がSourceにない。** 要求（`Obligation`）は呼び手が組み立てます。`manual-required`と`advisory`を書く場所は未定です。
3. **Inspector HostのRegistryは内部にあり、外から読めない。** 判定に使うRegistryとHostの選択は、Provider IDとContractが一致することでしか結び付きません。
4. **期待するコードの名前は手書き。** 「手動実装として入るべき関数」を独立に示す正式な出所がありません。
5. **`generated`は扱わない。** Generatorの版と生成物の失効（TR-03）は#1121の範囲です。
6. **観測は一回・一Subject・同一プロセス内。** Target差、複数Subject、外部Providerは対象外です。

## 実行

```bash
go test ./tools/reference-trait-fulfillment -count=1
```
