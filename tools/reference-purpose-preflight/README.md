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

この第2段階の入口が扱うのは `application-pattern` であり、後続の第3段階以降に `semantic-validation` とDecision履歴の読取り専用照合を追加した。ただし業務手順の正式な受理、適用時点、認可、実際の外部処理までは結合していない。こうした要素は、それぞれの既存機構と後続の接続実証で確認する。

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

ただし、この段階で確認対象として表示することは、そのすべてを案件に必須と確定することではない。対象がValidation宣言だけの場合、有効期間の選択が必要か、外部Providerを使うか、書込み・外部副作用を伴うかは**実際のOperationと契約を確認してから**決まる。これらは `conditional` と表示し、全案件共通の必須Gateへ昇格させない。

`-purpose` を省略した場合は従来の平坦な作業領域調査結果を返すので、既存の利用先の入力形式を壊さない。合成JSONだけの `-input` と実作業領域への `-purpose` を同時指定することはできない。必要条件の実評価、実Providerとの接続、状態変更はこの段階の対象外である。

## Operationが宣言した書込意図による条件付き要件の絞込み（第6段階）

「書込の有無」がCanonicalのApplication PatternからCapabilityへ追える場合だけ、条件付きだった副作用フェンスの必要性を少し具体化する。

```sh
go run ./tools/reference-purpose-preflight \
  -workspace reference/workspace \
  -application order_entry \
  -subject-kind application-pattern \
  -subject-id order_entry.submit_order \
  -purpose execute-preflight
```

この経路は既存Canonical Worldの `Pattern(kind=command, capability=...)` から唯一の `Capability` を解決し、`Capability.effect(create|update, data)` の**明示された宣言**を読み取る。実TargetのProviderが何を実行したかの証明ではない。

`operationWriteIntent.status` が `declared-write` の場合に限り、`readiness.obligations[effect-fence]` を「必要な実行時チェックだが未評価」の `not-evaluated` とする。権限を与えず、更新確定・重複防止は既存#1174へ委ねる。

一方、Referenceの `order_entry.submit_order` は `sales.order.submit` を参照するが、**現在のCapabilityには `effect` が明記されていない**。そこで `effect-not-declared` と表示し、読み取り専用であることも、副作用が不要であることも主張しない。対象のCapabilityが見つからない・重複する場合も実行可能扱いにしない。実際のOperationの仕様に応じて後続の検証が必要となる。

固定のCore Capability列挙、独立した実行フェンス、すべてのプラグインへの厳格な必須メタデータは新設していない。この分類は**既に明示された書込意図があるときに役立つ補助表示**であり、Providerの実際のEffectや外部副作用を網羅した保証ではない。


## Contract Queryの明示参照とProvider未検証の分離（第7段階）

実Reference `order_entry.customer_reference` が指定する `master_data.customer/find_by_id` を、既存のCanonical `Pattern` と `Application.database.master.requires` を使って突き合わせる。今回確認するのは、**ApplicationがContract Queryを明示し、同じContractへの要求を宣言しているか**だけである。

```sh
go run ./tools/reference-purpose-preflight \
  -workspace reference/workspace \
  -application order_entry \
  -subject-kind application-pattern \
  -subject-id order_entry.customer_reference \
  -purpose execute-preflight
```

目的指定した結果の `assessment.contractQueryUse` に、`datasourceAlias`、`targetDatabase`、`contract`、`query`、`compatibleMajor` を表示する。これは既存の型付き識別子から派生する局所的な読取り専用表示であり、別のContract正本を生成しない。Datasource aliasは既存 `DatabaseByAliasExact` で重複を許さず照合する。

`declared-requirement` であっても `readiness` の `provider-contract` は `conditional` のまま、理由を「既知のQuery契約があるがProvider適合・接続は未検証」と具体化する。契約版の一致、Query実装、実TargetのResource binding、Hostの実権限、外部副作用の有無、受理済みBusiness Ruleは証明しない。要求契約が欠ける、aliasが重複する、Queryが未指定の場合、推測で代替せず未解決とする。

`-purpose` を指定しない従来の平坦JSONからはこの補助フィールドを除く。作用・実行権限の評価を行わず、`executionAllowed=false` を維持する。

## Environment別の明示BindingとProvider適合を区別する（第8段階）

`-purpose execute-preflight` では、任意の `-environment <ID>` で**指定した環境だけ**の宣言を確認できる。環境未指定時は、従来どおり環境を推定しない。

```sh
go run ./tools/reference-purpose-preflight \
  -workspace reference/workspace \
  -application order_entry \
  -subject-kind application-pattern \
  -subject-id order_entry.customer_reference \
  -purpose execute-preflight \
  -environment development
```

この調査は、**PatternとEnvironmentを一回のWorkspace読込みで取得した同じCanonical World**を対象にする。前節の `contractQueryUse.status=declared-requirement` が成立した場合にのみ、既存Canonical `Environment.bindings` の `(application, alias)` を**一意に**解決し、その `DatabaseInstance.definition` がApplicationの要求対象DBと一致するかを確認する。正本の `development` では `order_entry.master -> master_data.dev`、`production` では `order_entry.master -> master_data.prod` を追跡できる。両Instanceが宣言しているDB種別は `oracle`。

結果は `assessment.environmentBinding` に追加し、定義一致時の `status=declared-environment-binding` は**EnvironmentがこのDB接続先を宣言したという事実**だけを表す。実Providerの通信・Query実現・Contract互換性・Plugin Conformance・リソース/権限・認可の有効性を証明しない。そのため目的別 `provider-contract` は `not-evaluated`（確認事項が具体化されたが評価は未実施）であり、`executionAllowed=false` を維持する。

対象Environment、alias/Binding、Instanceの不存在・重複は `unresolved`、Binding先が要求DBと違う場合は `incompatible` として返し、候補から適当な一件を選ばない。`-environment` は `application-pattern` と `-purpose execute-preflight` の組合せでのみ使える。従来の `-purpose` 未指定時の出力形式は変えない。

ここでも独立したProvider Resolverや権限台帳を増設しない。実Providerへの接続・Conformanceと副作用の実行境界は、既存Owner機構が提示する検証根拠を別途消費する必要がある。

## Database Contract Queryの実装宣言とProvider Conformanceを分離する（第9段階）

`-purpose execute-preflight -environment development` まで指定すると、同じロード済Canonical World内で、前段のContract Query要求・Environment bindingから指定された **Database / Contract / Query / Provider** を正確に照合する。

既存のDatabase正本に存在するContract署名（メジャー・マイナーバージョンとQuery名）を確認したあと、`ContractQuerySQLRealizations` に同じTargetおよびDB種別向けの明示SQL実装がちょうど一件あるかを調べる。見つかる場合は既存 `ContractQuerySQLRealization.ValidateContract` を使用して入力・結果mappingの構造を確認する。SQL本文の実行や実Providerの署名・Conformanceは調べない。

実Referenceの `master_data.customer v2.4 / find_by_id` は `development` / `production` ともに `oracle` ラベルまで追跡できるが、Reference WorkspaceにはそのTargetとProviderのSQL realizationがない。そのため `assessment.queryImplementation.status=contract-signature-declared` とし、`query-implementation` は `conditional`（別Provider実装や外部Evidenceも許容）と表示する。これは**Oracleへ接続できない、実装が存在しない、または導入を拒否すべきだという判定ではない**。

一方、合成FixtureでSQL realizationが明示され、既存Query署名へのmapping検査が通った場合でも `sql-realization-declared` までしか表示しない。DBとの実通信、結果の正しさ、既存Capability Contract ReferenceのPublisher/Provider Conformance、現在のHost権限を満たしたとは扱わない。

契約メジャーバージョンの不一致は `incompatible`、対象署名の不存在・重複、SQL realizationの重複、mapping不適合は `unresolved` とする。対象の入力元の欠落を推測・自動生成せず、必要な追跡先だけを提示する。旧CLIの平坦JSONは維持し、`executionAllowed=false` は変わらない。

## Contract Query Executorの既存Resolverとの接続（第10段階）

前段では、Database Contractの署名・SQL realization・Providerの構成を分けて調べた。この段階では、さらに `World.ContractQueryExecutorBindings` の明示宣言を、**既存の** `plugin.Catalog.ResolveEnvironment` と `realization.ResolveContractQueryExecutor` で解決する。新しいProvider Resolverや第二のConformance台帳は作らない。

```sh
go run ./tools/reference-purpose-preflight \
  -workspace reference/workspace \
  -application order_entry \
  -subject-kind application-pattern \
  -subject-id order_entry.customer_reference \
  -purpose execute-preflight \
  -environment development
```

Reference Workspaceの `development` には `master_data.dev` と `master_data.customer/find_by_id` を `database.query.oracle` に結ぶ明示Bindingがあり、既存Pluginカタログから `tei.database.oracle@1.0.0` へ解決できる。これは **静的な定義を解決できたという限定された結果**（`assessment.executorHandoff.status=resolved-declaration`）に過ぎない。Hostによる現在のContract受理、Provider Conformance Evidence、Issuer Trust、Resource/Permission、実Oracle接続は独立したままで、目的別の `query-executor` は `not-evaluated`、`executionAllowed=false` を維持する。

一方、現在の `production` は `master_data.prod` の環境Bindingは存在するが、そのTargetとInstanceを示す `ContractQueryExecutorBinding` はReference正本内に存在しない。 `not-declared` と表示し、外部実装や別経路が存在しないとは断定せず `query-executor=conditional` とする。明示Bindingの重複・Plugin未解決・DB provider不適合・Capability不一致は `unresolved`。正本にないCapabilityをDB種別から自動選択しない。

この検査は指定した `-environment` と `-purpose execute-preflight` に限る。従来の平坦JSONに新項目は出さない。既存の `internal/plugin/runtime.ServiceAcceptanceReader` が持つHost現在受理と、`tools/reference-plugin-capability-contract` のIssuer/Provider Conformance Referenceは**別の検証責務**であり、ここで合成Evidenceを真正なHost受理に読み替えない。

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

開発工程として使う場合も、最初からすべての設計が厳密に揃うことを要求しない。説明・試作・実行準備の各目的で、どの条件が不十分かを導く設計を別途検証する。現在の `purpose` は参照実装の局所的な実験区分であり、目的別の未充足事項の**表示**までは実装した。一方、実Providerや現在の人間受理・Runtime Contextから証明義務が充足されたと判定する閉包は**未実装**である。
