# ADR-0029: Applicationが評価するValidationの選択を、Canonicalな`uses validation`宣言から導く

- 状態: 提案中
- 日付: 2026-10-01
- 関連Issue: [ISSUE-000073](../issue/000073-derive-validation-selection-from-canonical.md)（作業範囲）、[ISSUE-000072](../issue/000072-rule-change-hand-edit-count.md)（この欠落を計測する）、[ISSUE-000027](../issue/000027-ai-proposal-acceptance-boundary.md)（`APPLICATION_VALIDATION_SELECTION_NOT_CANONICAL`を最初に記録）
- 関連ADR: [ADR-0024](0024-business-lifecycle-ownership.md) / [ADR-0025](0025-transition-evaluation-and-conflict-result.md)（遷移ごとの`requires`。いずれも提案中）
- 主担当要求: RQ-003 / RQ-013 / RQ-017 / RQ-021

この判断は評価待ちです。採用が決まるまで、実装やReference変更の根拠にしません。

## 背景

業務Validation（たとえば「受注明細の数量は正である」）の**内容**は、`reference/workspace/semantic/sales/validation.tei`という一つの正本にあります。Driverの確認用Scenarioと第一Web Targetは、同じ定義を読んで評価します。

しかし、**どのValidationを評価するか**という選択は、正本のどこにも宣言されていません。2026-10-01時点（revision `cb30a279`）の読解で、次のことを確認しました。

- TEIの構文には、Applicationが使うValidationを宣言する`uses validation <id>`があります。parserは`Application.Validations`へ読み込み（`internal/syntax/parser.go`）、共有Validationは参照先の存在と、そのnamespaceをApplicationが`uses semantic`していることを確かめます（`UNKNOWN_APPLICATION_VALIDATION`、`APPLICATION_VALIDATION_SEMANTIC_NOT_USED`）。Shared Acceptance Previewは、この宣言を差分とimpactへ含めます。
- しかし`order_entry`も`repair_intake`も、この宣言を持っていません。
- Driverの本体（`Resolver`）には、Validationの選択を扱う機能がありません。確認用Scenarioは、評価するValidationの一覧を呼び出し側から引数で受け取ります（`runSubmitOrderGateScenario`の`validations []semantic.Validation`）。
- 第一Web Targetは、`tools/reference-first-web-target/command_scenario.go`の二か所に、Canonical IDの一覧を直接書いています（`sales.order_line.quantity_positive`、`sales.order.lines.line_number_unique`）。

つまり、選択は呼び出し側ごとに手で書かれています。正本へ新しいValidationを足してShared Acceptanceで受理し、共有Validationがgreenでも、呼び出し側の一覧を直さない限り実行時には評価されません。画面側が規則の**内容**を書き写していないことは事実ですが、「どの規則が適用されるか」という意味の一部は、正本の外に二つ目の置き場を持っています。

## 防ぎたい失敗

- **追加した規則が効かない。** 正本へ規則を足し、受理し、検査がgreenなのに、実行すると評価されず、単価0の明細を含む受注が確定する。誰も気づかない。
- **経路ごとの食い違い。** 第二Target、API、AIエージェントがそれぞれ別の一覧を持ち、同じ受注に異なる規則が適用される。
- **一覧の黙った推測。** 宣言が無いときに、呼び出し側が「namespaceの全Validation」を評価するなどの暗黙の規則を作り、意図しない規則まで適用する。
- **空を成功と読む。** 宣言が無いApplicationで、評価対象が空であることを「満たされた」と扱い、検査なしに通す。

## 守る原則

- 評価対象の選択はCanonicalな意味であり、呼び出し側（Driver、Target、API、AIエージェントの経路）は選択を持たない（原則P1、P3、P9）。
- 宣言が無い、または解決できないときは、推測で補わず、成功にも丸めない（原則P6、P11）。
- 新しいCore primitiveを足す前に、既存の宣言で足りるかを確かめる（原則P16）。

## 判断

### 選択の正本を、Applicationの`uses validation`にする

Applicationが評価するValidationは、そのApplicationの`uses validation`宣言の集合とします。新しい構文もCore型も追加しません。

```text
application order_entry {
    uses semantic sales
    uses validation sales.order_line.quantity_positive
    uses validation sales.order.lines.line_number_unique
    ...
}
```

### 評価の順序は宣言の順序とする

現在の確認用Scenarioと第一Web Targetは、`quantity_positive`、`line_number_unique`の順に評価し、最初の不成立で止まります。`order_entry`は同じ順序で宣言し、既存のScenarioの結果を変えません。順序をsourceの宣言から導けるため、順序もCanonicalな情報になります。ただし、Validationを評価する段が、Authentication / Authorizationに対してどこに位置するかは、本ADRでは変えません（現行の順序のまま）。

### 呼び出し側へ一覧を渡さず、Resolverから得る

Driverの`Resolver`に、Application identityからそのApplicationが宣言したValidationを解決する入口を一つ置きます（名前は実装時に決めます。たとえば`ResolveApplicationValidations`）。Target、確認用Scenario、将来のAPIやAIエージェントの経路は、この入口だけを使い、独自の一覧を持ちません。

解決の失敗（宣言したIDがWorldに無い）は、既存の共有Validationが静的に検出するため、実行時には通常起きません。起きた場合は、評価を省略せずExecution Errorとして返します。

### 宣言が無いApplicationのCommandは、評価対象を推測せず、成功にしない

Commandを実行する経路が、宣言の無いApplicationのValidationを解決しようとしたときは、空の集合を「全て満たされた」として通しません。`APPLICATION_VALIDATION_SELECTION_NOT_DECLARED`という判定不能を返します。業務上の拒否（Validation不成立）とは区別し、Execution Error側の分類とします。

「このApplicationのCommandには、評価すべきValidationが無い」ことを意図して表す方法は、現時点では決めません（下記「未解決」）。

### Shared Acceptance Previewのunresolvedを見直す

Previewは、Validationを追加する変更について、そのnamespaceを使うApplicationが`uses validation`で宣言していない場合に`APPLICATION_VALIDATION_SELECTION_NOT_CANONICAL`を返します（`tools/internal/changeanalysis/impact.go`）。このunresolvedは「宣言しても実行時に選択へ使われない」ことも含意しているため、本ADRの採用後は、宣言が無いことだけを示す`APPLICATION_VALIDATION_SELECTION_NOT_DECLARED`へ改め、実行時の選択が宣言から導かれるようになった後は「宣言すれば評価される」ことを前提にしたメッセージにします。

## 採用しない案

### A′: 遷移（Lifecycle）の`requires`だけに選択を置く

[ADR-0025](0025-transition-evaluation-and-conflict-result.md)の設計は、遷移ごとに必要なValidationを`requires`へ書きます。操作ごとの違いを表せますが、ADR-0024 / 0025は提案中で、Lifecycleを持たないApplicationや、遷移を伴わないCommandには使えません。本ADRの問題（宣言が無い）は、Lifecycleの採否と独立に起きています。

本ADRは、Applicationの宣言を**そのApplicationが評価してよいValidationの全体集合**とし、ADR-0025が採用された場合の`requires`は、この集合の中から操作ごとに選ぶ層として重ねる関係を想定します。重ね方（`requires`が全体集合の部分集合であることを検査するか等）は、ADR-0025の採否の時点で決めます。

### B: Capabilityまたは Business Actionへ、実行前に満たすValidationを宣言する

操作ごとの違いを表せます。しかし新しい宣言が要り、ADR-0025と重なります。現在の`order_entry`の受注確定は唯一のCommandで、Aで意味を失いません。操作ごとの違いが必要になる具体的なReferenceが現れてから判断します。

### C: 使っているsemantic namespaceの全Validationを評価する

宣言が不要です。しかし、namespaceへ追加したValidationが、意図していないCommandにも暗黙に適用されます。「規則を足す」ことが「全ての操作の条件を変える」ことになり、変更の影響を宣言から読めなくなります。「一覧の黙った推測」そのものであり、採りません。

### D: 呼び出し側の一覧を残し、正本との一致をtestで検査する

食い違いは検出できますが、一覧の置き場が二つあることは変わりません。変更のたびに人が二か所を直し、testがその二重管理を強制します。RQ-017（人の二重管理を最小化する）に反します。

## 影響

### 良くなる点

- Validationを足すとき、人が直すのは正本（Validation定義とApplicationの宣言）だけになる。Target側の一覧の手直しが要らない
- 第二Target、API、AIエージェントの経路が、同じ選択を同じ入口から得られる
- 受理前のPreviewで、評価対象の変化が差分とimpactに現れる（既に`Application.Validations`を差分へ含めている）
- 宣言が無いときの振る舞いが推測ではなく、明示された判定不能になる

### 増える負担・制約

- 既存の`order_entry`のApplication sourceが変わります。`reference/world.json`も再生成され、Shared Acceptance Boundaryを通した受理（explicit human Acceptance decision）が要ります。
- Gate B / Gate CのEvidenceは、それぞれのexact revision（`R = e5231839…` / `T = 0ba87a4e…`）へ紐づく歴史的Evidenceです。この変更後のworkspaceを、旧baselineと同じものとして読み替えません。
- 確認用Scenarioのtest（`submit_order_gate_scenario_test.go`、`gate_b_*_scenario_test.go`、`final_gate_b_scenario_test.go`）と第一Web Targetの評価対象は、一覧を引数・直書きから、Resolverの入口へ移す変更が入ります。Evidence auditの期待値（Validation 2件、最初の不成立で停止）は変えません。
- `repair_intake`はCommandを持たないため、宣言が無くても、このADRの判定不能に当たりません。

### 残るリスク

- 「全Applicationに宣言が要る」と読まれると、Commandを持たないApplicationへ空の宣言を強いる運用になる。判定不能は、Commandを実行する経路が選択を要求したときだけ返します。
- 宣言の順序に意味を持たせるため、順序を入れ替える変更は意味の変更になります。Shared Acceptance Previewは、`Application.Validations`の順序の変更を差分として扱う必要があります（現在の差分は`Validations`のslice順をそのままJSON化してbyteで比較するため、順序の入れ替えも変更として現れます）。

## 未解決

- 「評価すべきValidationが無い」ことを意図して表す方法（空の宣言を許すのか、明示的な宣言を足すのか）。現時点で、そのReference上の必要は見つかっていません。
- 操作（Command）ごとに評価対象が異なる必要が生じたときの層の重ね方（上記A′とB）。
- `APPLICATION_VALIDATION_SELECTION_NOT_DECLARED`を、Business ResultとExecution Errorのどちらの分類へ置くか。本ADRはExecution Error側を提案しますが、[ADR-0020](0020-capability-business-outcome-boundary.md)の分類との整合を実装前に確認します。

## 確かめ方

採用後は、次をReferenceで確かめます。

1. 正本へValidationを一件足し、Applicationへ宣言を一行足すだけで、Target側のコードを変えずに、画面からの確定でその規則が評価される。
2. 宣言を外すと、評価されないまま成功するのではなく、判定不能として返る。
3. 既存のCommand scenario（成功、Authorization拒否、Validation不成立、Execution Error、Observation）の結果と、Evidence auditの期待値が変わらない。
4. [ISSUE-000072](../issue/000072-rule-change-hand-edit-count.md)の最初の計測ケースを再計測し、「正本の外にある意味の手直し」（C3）が減る。
