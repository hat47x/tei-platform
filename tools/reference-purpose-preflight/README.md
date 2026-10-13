# 開発目的別の業務条件を確認する小さな参照実装

対象：[開発工程プラグインの課題 #1445](https://github.com/hat47x/tei-platform/issues/1445)、[業務手順からの機能展開 #1444](https://github.com/hat47x/tei-platform/issues/1444)、[文章の定式化 #1440](https://github.com/hat47x/tei-platform/issues/1440)。

## なぜ必要か

業務規則の意味が決まっていない状態と、規則は確定していても個別の取引データをまだ読んでいない状態を区別する必要がある。「高額受注」という言葉の基準が不明なら承認条件は定まらない。一方、税込受注総額が50万円以上なら承認が必要という意味が確定していれば、各受注の金額は実行時に評価できる。

この参照実装は**合成データによる条件評価と、実際のTEI作業領域の読取り専用調査を分けて提供する検討用具**である。後者は既存の入力元索引と知識の適用条件検査を利用するが、Shared Acceptanceの業務規則受理、実際の契約解決、実行時の認可、副作用確定処理へは接続していない。入力JSONの `meaning=accepted` は**テスト用の自己申告**であり、人が受理した証拠ではない。出力の `executionAllowed` は、すべての場合に `false` とする。

## 実行と検証

リポジトリのルートから：

```sh
go run ./tools/reference-purpose-preflight -input tools/reference-purpose-preflight/order-approval.json
go test ./tools/reference-purpose-preflight -count=1
```

この実装が読む `order-approval.json` の金額は、円の整数として扱う。対象は税込受注総額のみで、契約上の通貨・算定基準が一致しない場合は結果を判断不能とする。実際の会計処理、消費税計算、複数通貨、制度上の取引規則を一般化したものではない。

## 実際のTEI作業領域を参照する（第2段階）

試験用フラグ `meaning=accepted` から独立した入口として、既存の作業領域・入力元索引・知識の適用条件検査を**読取り専用で再利用**する。

```sh
go run ./tools/reference-purpose-preflight \
  -workspace reference/workspace \
  -application order_entry \
  -subject-kind application-pattern \
  -subject-id order_entry.customer_reference
```

この入口では、正本の `Application Pattern` が一意に宣言されていること、入力元索引に照合値付きの定義元があることを確認する。実際の検査は既存の `workspace.Load`、`workspace.BuildSourceIndex`、`knowledgecontext.Build` と `knowledgecontext.QualifyWithSources` を利用する。結果の `sourceStatus=traceable-source-declaration` は、**入力元へ追跡できるという限定された主張**である。

この作業領域で `order_entry.customer_reference` が定義されていても、「50万円以上は承認必須」という規則が受理された根拠にはならない。そのため `ruleMeaning=not-established`、`executionAllowed=false` を維持する。未定義の機能を指定すると `sourceStatus=unresolved` とし、代わりの機能や意味を推測しない。

検査できる対象は、現時点では `application-pattern` のみであり、業務手順の正式な正本、受理した変更記録、適用時点、認可、実際の外部処理まで結合したものではない。こうした要素は、それぞれの既存機構と後続の接続実証で確認する。

```sh
go test ./tools/reference-purpose-preflight -count=1
```

## 結果の読み方

| `status` | この参照実装が示すもの |
| --- | --- |
| `semantic-unresolved` | 受理済み規則への参照や比較基準が不明 |
| `deferred-to-runtime` | 比較規則は仮に整っているが、取引金額が未取得 |
| `evaluated` | 合成入力による条件式の評価に成功。 `guard=true/false` は**承認が必要か**の判定値であって実行許可ではない |
| `indeterminate` | 通貨・算定基準・取引の版などが不明または不適合 |
| `blocked` | 対象の版不一致や利用目的の不適合を検出 |

この出力は業務操作の受理、Provider適合、実行権限、外部更新、出荷完了を一切証明しない。過去の成功を他の意味版・環境・Providerへ流用することもできない。

## 次の実装へ引き渡すもの

最初の負例では、未確定な業務規則を実行時に回すのではなく、**意味上の未解決事項**とする。規則の意味が受理されていて、必要な取引情報だけ未取得なら、**実行時に必要となる入力**として残す。通貨が欠落した場合は `false` とせず判断不能、対象版が変われば版不一致として扱う。また、金額が未取得でも、通貨・算定基準の既知の不一致や受注の既知の版不一致があれば、実行時への繰延べを優先せず、不適合・版不一致を報告する。これらを明示して、単純な値比較ではなく依存関係・条件付き分岐の評価へ拡張する。

製品へ組み込むときは、#1120の型付き参照、#267の実際の受理・出所、#1121の変換、#951の契約・提供元、#1174の副作用確定境界、#1180の正本となる検証シナリオを**消費**する。各機構を再実装したり、新たな実行許可を発行したりしない。業務上の適用期間、複数DB、外部通知の成否、権限失効はこの合成参照ではまだ実証していない。

開発工程として使う場合も、最初からすべての設計が厳密に揃うことを要求しない。説明・試作・実行準備の各目的で、どの条件が不十分かを導く設計を別途検証する。現在の `purpose` は入力検証に使うだけで、目的別の証明義務の導出や条件閉包を**未実装**とする。
