# ISSUE-000014: 業務Validationの最小Referenceと所有境界を実証する

- 状態: 着手可能
- GitHub Issue: #125
- 上位Issue: `ISSUE-000013` / GitHub Issue #124
- 後段: `ISSUE-000006` / GitHub Issue #69（Application Driver）

## 背景

Gate A監査で、Patternの構文・参照整合性を検査するValidationと、Driver Scenarioが実行する業務Validationは別の意味であることを確認しました。

現在のRepositoryにはApplication Patternの共通Validationがありますが、数量や明細集合に対する業務ruleの正本は、まだReferenceとして存在しません。

この状態でDriverへ進むと、業務ruleをDriver専用GoコードやWeb入力制約へ暗黙に埋め込むおそれがあります。

## 目的

`order_entry`で、性質の異なる次の二種類を一例ずつ扱います。

1. 一つの属性だけで判定できるrule
2. Relationで結ばれた複数Detailを見なければ判定できないrule

大きなValidation DSLやrule engineを先に作ることが目的ではありません。

業務ruleの正本、identity、評価結果の意味、DriverとTargetからの再利用境界を、具体例から確認します。

## 最初のReference候補

### 1. 単一属性 — `OrderLine.quantity`は正である

```text
sales.order_line.quantity > 0
```

既存の`quantity`だけで成立し、Webのinput属性やDatabase CHECKへ閉じずに「業務上有効な数量」という意味を扱えるかを確認します。

### 2. Relation単位 — 一つのOrder内で`lineNumber`が重複しない

```text
sales.order.lines
  └─ sales.order_line.line_number is unique within one order
```

この例を第一候補にする理由は次です。

- 既存`sales.order.lines`と`OrderLine.lineNumber`だけで表せる
- `lineTotal`、税、値引等の新しいSemanticをValidationのためだけに追加しなくてよい
- 一項目のHTML validationへ還元できない
- `OrderLine`全体で一意ではなく「一つのOrderに属する明細集合の中で一意」というRelation scopeを必要とする
- 物理Foreign KeyやDatabase UNIQUE constraintの実現方法を先にSemanticへ持ち込まずに済む

実Referenceでこの例が不自然だと分かった場合は変更できますが、金額計算等へ広げる前にこの小さい例から始めます。

## 所有境界として比較するもの

### Semantic側のrule

複数Applicationでも変わらないDomain上の不変条件として扱う場合です。

`quantity > 0`はこの候補になり得ます。

### Application-owned Validation

同じSemantic Dataでも、そのApplicationの入力・編集・操作文脈で成立する条件として扱う場合です。

### Capability / Policy等の独立した意味単位

外部状態や実行を伴う判断、または再利用可能なPolicyとして独立させる必要が実例から見えた場合に検討します。

ただし、このIssueだけを理由に新しい汎用`Policy` modelや式言語を先に追加しません。

## 表現方法を先に固定しない

最初に決めるのは「どのruleがどこに属するか」であり、「どのDSL構文で書くか」ではありません。

候補には、型付きの小さなCanonical rule、既存Capability境界の利用、native implementationへのBinding等があります。

実Referenceから必要性を確認できない抽象を増やさず、次の条件を満たす最小表現を選びます。

- rule identityを追跡できる
- authoring sourceへ戻れる
- Driverがrule本文を独自に再実装しない
- Targetが同じ結果を利用できる
- 正常な不成立とExecution Errorを区別できる

## 結果境界

Validationが正常に評価された結果と、Validation自体を実行できなかった状態を分けます。

```text
Validation Result
  ├─ satisfied
  └─ not satisfied

Execution Error
  └─ evaluation自体を完了できなかった状態
```

結果契約は#77 / ADR-0020と矛盾しないようにしますが、Validationを必ずSemantic Capabilityとして実装すると先に決めません。

Authorization denialとも別の意味です。

## 守る境界

- HTML `required`やWeb form validatorをCanonicalな業務ruleの唯一の正本にしない
- Database CHECK / UNIQUE constraintだけを業務Validationの正本とみなさない
- Driver専用Goコードにだけruleを書かない
- Pattern KindのfieldへValidation式を押し込まない
- 同じruleをSemantic / Application / Database / Webへ手書きで複製しない
- Validation不成立をExecution Errorと混同しない
- Authorization不成立とValidation不成立を混同しない
- `lineNumber`のRelation scopeを物理Foreign Key設計から逆算しない

## Gate Aへ渡す情報

- Validation identity
- ruleが所有される層
- 対象Data / Attribute / Relation等のidentity
- Validation Resultの意味
- implementationが必要な場合のBinding境界
- authoring source / provenance
- Driverがrule本文を独自実装せず評価できる入口

## Conformance

少なくとも次を確認します。

- `quantity > 0`相当の単一属性ruleをReferenceから読み込める
- `sales.order.lines`内の`lineNumber`重複禁止相当のRelation-scope ruleをReferenceから読み込める
- Validation identityからauthoring sourceへ戻れる
- Driver専用Goコードを正本にしない
- Web固有Validationを正本にしない
- Database constraintを唯一の正本にしない
- Validation不成立とExecution Errorを区別できる
- 同じValidation identityを後続Driverへ渡せる

## 非目標

- 汎用rule engineの完成
- 汎用式言語の確定
- BPMN / workflow全般
- Web form framework
- Database constraintの自動生成
- AIによるrule自動生成
- 税・値引等を含む販売管理全体の業務rule

## 完了条件

1. `quantity > 0`相当の単一属性ruleをReference化できる
2. `sales.order.lines`内の`lineNumber`重複禁止相当のRelation-scope ruleをReference化できる
3. Validationの正本がDriver / Web / Databaseのいずれか一つの実装へ閉じていない
4. Pattern構文・参照整合性Validationと業務Validationを明確に区別できる
5. Validation不成立を正常な業務結果として扱い、Execution Errorと区別できる
6. Driver #69がValidation identityと結果契約をGate Aから受け取れる
7. Reference / Conformanceから同じruleを再現できる

## 関連

- GitHub Issue #124 / #125 / #69 / #77
- `ISSUE-000013`: Gate A到達性監査
- `ISSUE-000006`: Application Driver
- `ISSUE-000009`: Capability Outcome / Execution Error
- RQ-001, RQ-003, RQ-012, RQ-014, RQ-017, RQ-018, RQ-021
