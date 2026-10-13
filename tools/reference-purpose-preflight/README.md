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

この作業領域で `order_entry.customer_reference` が定義されていても、「50万円以上は承認必須」という規則が受理された根拠にはならない。 同じく `order_entry.submit_order` は実際に存在する操作機能だが、その機能の存在から出荷承認規則を推定しない。追加の負例では、この実操作の入力元が追跡できても `ruleMeaning=not-established` を維持する。そのため `ruleMeaning=not-established`、`executionAllowed=false` を維持する。未定義の機能を指定すると `sourceStatus=unresolved` とし、代わりの機能や意味を推測しない。

検査できる対象は、現時点では `application-pattern` のみであり、業務手順の正式な正本、受理した変更記録、適用時点、認可、実際の外部処理まで結合したものではない。こうした要素は、それぞれの既存機構と後続の接続実証で確認する。

```sh
go test ./tools/reference-purpose-preflight -count=1
```

## 正本にあるValidationを確認する（第3段階）

`order_entry.submit_order` の定義元を追跡できても、「出荷時の50万円承認規則」は証明されない。一方、既存の正本には `sales.order_line.line_number_positive`（受注明細の行番号に対する正数検査）が実際に記述され、`order_entry` がそれを利用する宣言も存在する。

その違いを、**新しい規則や受理台帳を作らず**、既存の知識の適用条件検査と入力元索引で確かめる。

```sh
go run ./tools/reference-purpose-preflight \
  -workspace reference/workspace \
  -application order_entry \
  -subject-kind semantic-validation \
  -subject-id sales.order_line.line_number_positive
```

この入口では、正本のValidationが一意であること、指定したApplicationがそのValidationと名前空間を明示的に利用していること、型付き定義元と現在の照合値を検査する。満たす場合の `ruleMeaning=canonical-validation-declared` は、**正本に機械可読な意味宣言があること**を表す。人間の承認履歴を確認したという意味ではない。人の受理資格と実行時検証はそれぞれ `human-acceptance-not-verified`、`runtime-validation-not-evaluated` として残し、`executionAllowed=false` を維持する。

実際の決定記録は既存のShared Acceptance側で照合できる。

```sh
go run ./tools/reference-shared-acceptance decisions -workspace reference/workspace
```

この決定記録は、保存された過去の判断と現在のソースがどう対応するかを確かめるための**既存の検査入口**であり、本ツール内に第二のDecision Indexは作らない。決定記録が `accepted` でも、現在の実行権限や、当該規則が人によって正式に確認されたことまで自動的に証明するわけではない。過去の変更の受理と、利用目的に対して必要な人間の確認範囲を区別する。

この第3段階は業務規則の存在・出所を検査する参照実装であって、**50万円閾値を正本から導出したものではない**。適用期間、実行時のデータ、実Providerの能力、実行許可は後続の検証対象である。

## Shared Acceptanceの既存決定索引との照合（第4段階）

正本上のValidationの存在と、過去の受理Decisionを結ぶ場合は、**まず既存のShared Acceptanceが決定索引を生成**する。参照実装自身がDecision台帳を読取り・検証し直すのではなく、その報告を限定された履歴情報として消費する。

```sh
go run ./tools/reference-shared-acceptance decisions \
  -workspace reference/workspace > /tmp/tei-shared-decisions.json

go run ./tools/reference-purpose-preflight \
  -workspace reference/workspace \
  -application order_entry \
  -subject-kind semantic-validation \
  -subject-id sales.order_line.line_number_positive \
  -decision-index /tmp/tei-shared-decisions.json
```

ここでは正確に一致する種類付きValidation ID（`semanticDiff.kind=validation` と `id`）、Decisionの `accepted` と `sourceChanged`、Shared Acceptance索引が報告する `sourceEdits=intact`、現在の入力元索引で確認した定義元パスと照合値を組み合わせる。該当記録があれば `decisionHistoryStatus=reported-source-matched` として記録への参照を示す。該当IDの記録が無ければ `not-matched`、記録はあるが定義元の照合値が異なるなどの場合は `recorded-subject-source-not-current` とする。

**これは現在のSourceに一致する過去の変更決定が報告された、という限定的な照合であり、人間の受理を証明したものではない。** 決定索引の内容ハッシュ・検証根拠・Review Projectionを検証する責任は既存Shared Acceptanceに残る。この参照実装は外部JSONの真正性や、生成された索引が本当に指定作業領域と同じ版から取得されたことを独立には検証しない。そのため、手で書き換えたJSONを渡しても出力は`報告値`のままであり、実権限や真正な履歴の証明として使用できない。

さらにReviewの照合値が `verified` でも、レビュー内容の提示と判断権限を持つ人の本人確認は別である。決定索引に含まれる `decisionActorStatus` と `decisionAuthorityStatus` をそのまま表示するが、実行許可や確認済み業務規則としては扱わない。`human-acceptance-not-verified` と `runtime-validation-not-evaluated` は引き続き未解決として保持し、`executionAllowed=false` を維持する。

本段階で追跡する受注明細行番号の正数規則と、合成fixtureの50万円承認規則は別物である。他の規則の受理記録を転用して合成閾値がAcceptedとなったと主張しない。業務上の適用期間、複数業務ルールの合成、実Provider・外部効果については後続の検証が必要である。

## 利用目的に応じた必要条件の表示（第5段階）

同じ正本や決定索引を利用しても、文章の説明、試作の検証、実行に向けた事前確認では、確認すべき条件が異なる。そこで、元の正本と過去の受理記録を変更せず、対象操作と**利用目的に限った必要条件**を派生表示する。

```sh
go run ./tools/reference-purpose-preflight \
  -workspace reference/workspace \
  -application order_entry \
  -subject-kind semantic-validation \
  -subject-id sales.order_line.line_number_positive \
  -purpose execute-preflight
```

`-purpose` に指定できる値は、当参照実装で実証する範囲の `describe`（説明）、`simulate`（合成値での検証）、`execute-preflight`（実行前の不足確認）の三種類。**TEI Coreの新しい固定列挙ではない。** 説明には定義元と意味宣言の存在、模擬検証にはさらに入力fixtureと独立した期待値、実行準備には人の確認、業務上の有効範囲、現在の業務データ、契約・提供元、認可、副作用確定、Driver Scenarioの観測が関わる。

結果は `assessment`（既存の定義元・決定履歴の調査結果）と `readiness`（目的別の必要条件）の二領域に分ける。`readiness.obligations[*].status` は、この参照実装が確認した範囲を `observed`／`declared`／`unresolved`／`unverified`／`not-evaluated` などで示す。特に、`canonical-validation-declared` の正本宣言や `reported-source-matched` のDecision履歴があっても、**実際に人が必要な内容を承認したこと、実行時の権限、処理成功にはならない**。両領域の `executionAllowed` は常に `false`。

`-purpose` を省略した場合は従来の平坦な作業領域調査結果を返すので、既存の利用先の入力形式を壊さない。合成JSONだけの `-input` と実作業領域への `-purpose` を同時指定することはできない。必要条件の実評価、実Providerとの接続、状態変更はこの段階の対象外である。

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
