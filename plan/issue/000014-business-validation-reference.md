# ISSUE-000014: 業務Validationの最小Referenceと所有境界を実証する

- 状態: 着手可能
- GitHub Issue: #125
- 上位Issue: `ISSUE-000013` / GitHub Issue #124
- 後段: `ISSUE-000006` / GitHub Issue #69（Application Driver）

## 背景

Gate A監査で、Patternの構文・参照整合性を検査するValidationと、Driver Scenarioが実行する業務Validationは別の意味であることを確認しました。

現在のRepositoryにはApplication Patternの共通Validationがありますが、「数量が業務上有効か」「HeaderとDetailの組み合わせが成立するか」といった業務ruleの正本は、まだReferenceとして存在しません。

この状態でDriverへ進むと、業務ruleをDriver専用GoコードやWeb入力制約へ暗黙に埋め込むおそれがあります。

## 目的

`order_entry`で、少なくとも次の二種類の業務Validationを小さなReferenceとして成立させます。

1. 一つのData / fieldだけで判定できるValidation
2. HeaderとDetail、または複数fieldをまたいで判定するValidation

目的は大きなrule engineを設計することではありません。

業務ruleの正本をどこに置き、DriverとTargetが同じ意味を再利用できるかを確認します。

## 最初のReference候補

既存`sales.order` / `sales.order_line`だけで説明できる小さいruleを優先します。

たとえば、次のような候補があります。

- `OrderLine.quantity`が業務上有効な正の数量である
- HeaderとDetailをまたぐ整合として、既存fieldだけで判定できる小さい条件を一つ選ぶ

Validationのためだけに、現行Semanticへ`lineTotal`、税、値引等を追加しません。

実例の選択自体もEvidenceの一部とし、既存モデルを必要以上に広げないことを優先します。

## 所有候補

最初から一つの抽象へ固定せず、少なくとも次を比較します。

### Semantic rule

業務Domainそのものの不変条件であり、複数Applicationから共有される性質が強いruleです。

### Application-owned Validation

同じSemantic Dataでも、Applicationの利用目的や操作によって異なる条件です。

### Capability

単純なpredicateではなく、外部状態や業務処理を伴う判断です。

二つの具体例から、どこに置くと意味の二重管理が最も少なく、DriverとTargetから同じidentityを利用できるかを確認します。

## Business Resultとの境界

Validation不成立は、Validationの実行そのものが正常に完了した結果です。

そのため、次を区別します。

```text
Validation Result
  ├─ satisfied
  └─ not satisfied

Execution Error
  └─ Validation自体を実行できなかった状態
```

名前やpayloadの最終形は#77のCapability Outcome境界と整合させますが、Validation不成立を例外やRuntime障害へ押し込みません。

Authorization denialとも別の意味として扱います。

## 守る境界

- HTML `required`やWeb form validatorを業務ruleの唯一の正本にしない
- Database CHECK constraintだけを業務Validationの正本とみなさない
- Driver専用Goコードにだけruleを書かない
- Pattern KindのfieldへValidation式を押し込まない
- 同じruleをSemantic / Application / DB / Webへ手書きで複製しない
- Validation failureをExecution Errorと混同しない
- Authorization failureとValidation failureを混同しない

## Gate Aへ渡す情報

少なくとも次をDriverから解決できる形にします。

- Validation identity
- ruleが所有される層
- 対象となるData / Relation / Capability等のidentity
- Validation Resultの意味
- implementationが必要な場合のBinding境界
- authoring source / provenance

Driverは、これらを専用設定から初めて知るのではなく、Gate Aで成立した定義から受け取ります。

## Conformance

少なくとも次を確認します。

- 一項目型のruleをReferenceとして読み込める
- Header / Detailまたは複数field型のruleをReferenceとして読み込める
- Validation identityからauthoring sourceへ戻れる
- Driver専用Goコードを正本にしない
- Web固有Validationを正本にしない
- Validation不成立とExecution Errorを区別できる
- 同じValidation identityを後続Driverへ渡せる

## 非目標

- 汎用rule engineの完成
- BPMN / workflow全般
- Web form framework
- Database constraintの自動生成
- AIによるrule自動生成
- 税・値引等を含む販売管理全体の業務rule

## 完了条件

1. 一項目型と複数Data / Header-Detail型のValidationを少なくとも一例ずつReference化できる
2. Validationの正本がDriver / Web / Databaseのいずれか一つの実装へ閉じていない
3. Pattern構文・参照整合性Validationと業務Validationを明確に区別できる
4. Validation不成立を正常な業務結果として扱い、Execution Errorと区別できる
5. Driver #69がValidation identityと結果契約をGate Aから受け取れる
6. Reference / Conformanceから同じruleを再現できる

## 関連

- GitHub Issue #124 / #125 / #69 / #77
- `ISSUE-000013`: Gate A到達性監査
- `ISSUE-000006`: Application Driver
- `ISSUE-000009`: Capability Outcome / Execution Error
- RQ-001, RQ-003, RQ-012, RQ-014, RQ-017, RQ-018, RQ-021
