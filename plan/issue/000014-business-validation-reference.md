# ISSUE-000014: 業務Validationの最小Referenceと所有境界を実証する

- 状態: In Progress
- GitHub Issue: #125
- 上位Issue: `ISSUE-000013` / GitHub Issue #124
- 後段: `ISSUE-000006` / GitHub Issue #69（Application Driver）
- Canonical契約: PR #136

## 背景

Gate A監査では、PatternやCanonical Worldの構文・参照整合性を確認するValidationと、業務上の条件を評価するValidationを分けています。

Driver Scenarioが扱う「数量が正である」「一つの受注に属する明細番号が重複しない」といった条件は、ParserやWorkspace自体の整合性とは別の意味です。

この業務ruleをDriver専用Goコード、Web入力制約、Database constraintのいずれかへ閉じ込めると、同じ意味をTargetごとに作り直すことになります。

そのため、まず業務ruleの正本とidentityをCanonicalに置き、その後にReferenceとEvaluatorを接続します。

## 現在地

PR #136で、最初の二例に必要なSemantic業務Validationの最小Canonical契約まで実装しました。

### `positive`

単一Attributeが正数であるというDomain上の不変条件を表します。

```tei
validation sales.order_line.quantity_positive positive {
    attribute sales.order_line.quantity
}
```

### `relation-unique`

一つの`many` Relationに属するtarget Data群の中で、指定Attributeが一意であることを表します。

```tei
validation sales.order.lines.line_number_unique relation-unique {
    relation sales.order.lines
    attribute sales.order_line.line_number
}
```

Canonical model、Parser、Loader、Normalization、Source Index、共通Validationまで接続済みです。

このため、業務Validationは「正本をどこに置くか未定」の段階を通過しました。ただし、Reference Workspace本体への実rule追加と評価実行はまだ成立していません。

## 所有境界について得た判断

今回の二つのruleは、特定画面の入力都合ではなく、受注明細という業務意味そのものに属すると判断しました。

そのため、第一段階ではApplication PatternやWeb Targetではなく`sales` Semantic Namespaceを正本とします。

```text
sales Semantic
  ├─ sales.order_line.quantity_positive
  └─ sales.order.lines.line_number_unique
          ↓
      Driver / Targetが参照
```

この判断を、すべての業務Validationへ一般化したわけではありません。

Application固有の入力文脈、外部状態を伴う判断、独立したPolicy / Capabilityとして扱う方が自然なruleが現れた場合は、実例から別の所有境界を検討します。

## PR #136で成立した境界

### 型付きの最小ruleだけを扱う

今回必要な`positive`と`relation-unique`だけを扱います。

任意の式を記述できるValidation DSLや汎用rule engineは導入していません。

### 既存Semanticとの参照整合性を検査する

`positive`では、対象Attributeが数値型であることを求めます。

`relation-unique`では、次を求めます。

- 対象Relationが存在する
- Relationが`many`である
- 対象Attributeが存在する
- 対象AttributeがRelation target Dataに属する

これにより、「一つのOrderに属するOrderLine集合の中でlineNumberが一意」というRelation scopeを、物理DatabaseのUNIQUEやForeign Keyから逆算せず表せます。

### authoring sourceへ戻れる

Validation identityはSource Indexへ接続され、authoring sourceへ追跡できる構造になりました。

file pathそのものをValidation identityにはしません。

## 次に実証すること

Canonical契約が成立しただけではGate AでReadyとは判定しません。

次の二段階を実Referenceで確認します。

### 1. Reference Workspaceへ二つのruleを追加する

`sales` Semantic Namespaceへ次を実sourceとして追加します。

- `sales.order_line.quantity_positive`
- `sales.order.lines.line_number_unique`

確認すること:

- Reference Workspaceから読み込める
- Canonical Worldへ同じidentityで現れる
- Source Indexからauthoring sourceへ戻れる
- Reference WorldをRepository自身から再生成できる
- source fileの配置を変えてもCanonical meaningが変わらない

### 2. 最小Evaluatorを追加する

Driver専用のrule再実装ではなく、Canonical Validationを入力として評価する小さな入口を用意します。

結果は少なくとも次のように分けます。

```text
Validation Result
├─ Satisfied
└─ NotSatisfied

Evaluation Error
└─ 評価そのものを完了できなかった状態
```

`quantity <= 0`や同一Order内の`lineNumber`重複は`NotSatisfied`です。

一方、rule identityを解決できない、必要な入力が欠けていて評価不能である等の状態を`NotSatisfied`へ畳みません。

## Driverへ渡す情報

Gate Aでは、Driver #69が次をRepositoryから受け取れることを確認します。

- Validation identity
- rule kind
- 対象Attribute / Relation
- authoring source / provenance
- `Satisfied` / `NotSatisfied`という正常な評価結果
- 評価を完了できなかった状態との区別

Driverが`quantity > 0`等のrule本文を専用Goコードへもう一度書く状態は認めません。

## 守る境界

- HTML `required`やWeb form validatorをCanonicalな業務ruleの唯一の正本にしない
- Database CHECK / UNIQUE constraintだけを業務Validationの正本とみなさない
- Driver専用Goコードにだけruleを書かない
- Pattern KindのfieldへValidation式を押し込まない
- 同じruleをSemantic / Application / Database / Webへ手書きで複製しない
- Validation不成立を評価不能やExecution Errorと混同しない
- Authorization denialとValidation不成立を混同しない
- `lineNumber`のRelation scopeを物理Foreign Key設計から逆算しない
- 今回の二例だけを理由に汎用式言語やrule engineを追加しない

## Conformance

### 既に成立しているもの

- `positive` / `relation-unique`をParseできる
- 未検証のrule kindを拒否する
- kindごとの必須・禁止fieldを検査する
- `positive`を非数値Attributeへ適用できない
- `relation-unique`で`many`以外のRelationを拒否する
- Relation target外のAttributeを拒否する
- Validation identityをSource Indexへ接続できる

### 次に追加するもの

- 二つの実ruleをReference Workspaceから読み込める
- Validation identityからauthoring sourceへ戻れる
- `quantity > 0`をSatisfied / NotSatisfiedの両方で評価できる
- Relation scope内の`lineNumber`一意性をSatisfied / NotSatisfiedの両方で評価できる
- 評価不能をNotSatisfiedへ変換しない
- 同じCanonical ruleを後続Driverから利用できる

## 非目標

- 汎用rule engineの完成
- 汎用式言語の確定
- BPMN / workflow全般
- Web form framework
- Database constraintの自動生成
- AIによるrule自動生成
- 税・値引等を含む販売管理全体の業務rule

## Gate Aでの完了条件

1. `quantity > 0`のruleを実Referenceから読み込める
2. `sales.order.lines`内の`lineNumber`重複禁止ruleを実Referenceから読み込める
3. Validation identityからauthoring sourceへ戻れる
4. 同じCanonical ruleをEvaluatorが利用できる
5. Validation不成立を正常な`NotSatisfied`として扱える
6. 評価不能を`NotSatisfied`へ畳まず別経路で扱える
7. Driver #69がrule本文を独自実装せずValidation identityと結果契約を受け取れる
8. Reference / Conformanceから同じruleと評価結果を再現できる

## 関連

- GitHub Issue #124 / #125 / #69 / #77
- PR #136
- `ISSUE-000013`: Gate A到達性監査
- `ISSUE-000006`: Application Driver
- `ISSUE-000009`: Capability Outcome / Execution Error
- RQ-001, RQ-003, RQ-012, RQ-014, RQ-017, RQ-018, RQ-021
