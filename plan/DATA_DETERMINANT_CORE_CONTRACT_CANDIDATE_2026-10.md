# Data Determinant Core Contract Candidate — 2026-10

Status: **Design candidate / not accepted Core syntax**

## 1. 結論

contained repeat、changeable reference、independent associationのReference pressureを比較すると、business determinantを一つの万能`key` primitiveへまとめるのは適切ではない。

共通化すべきなのは、determinantを構成するtermの表現までである。

~~~text
shared determinant term vocabulary
        ↓
occurrence identity assertion
candidate-key assertion
functional-dependency assertion

は別のsemantic assertionとして扱う
~~~

physical primary key、Relation key、UI / runtime identityは、これらのauthorityにはしない。

## 2. Reference比較

| 類型 | business meaning | identityで必要なterm | 自動昇格しないもの |
| --- | --- | --- | --- |
| contained repeat | 親occurrence内部の反復 | relation source occurrence + local attribute | child table PK、line number単独 |
| changeable reference | 独立occurrenceへの参照 | 通常はidentity componentではない | FK、Relation key、参照先ID |
| independent association | 独立occurrence同士を結ぶassociation | relation target occurrences + 必要なlocal/temporal attribute | 二つのFKだけ、surrogate IDだけ |
| history/version | 同じoccurrenceの履歴 | occurrence identityとは別のversion term | version番号、latest flag、effective dateの名称だけ |

この差から、Relationが存在すること自体をidentity termへ昇格する設計は採らない。

## 3. Shared determinant term vocabulary

Core contract候補は、少なくとも次のtermを表現できる必要がある。

### Attribute term

~~~text
attribute occurrence value
~~~

例:

- `sales.order_line.line_number`
- `coverage.assignment.effective_from`

### Related occurrence term

~~~text
occurrence reached through an explicit Relation
~~~

termはRelation identityと、どちら側のoccurrenceを使うかを持つ。

例:

- contained repeatで`order_line`を識別するときの`order` occurrence
- CoverageAssignmentを識別するときのServiceContract occurrence
- CoverageAssignmentを識別するときのAsset occurrence

Data IDだけではなくRelation identityを持つ。これにより、同じDataへ複数の業務Relationがある場合を区別する。

### 将来候補

外部制度・Contractが直接identity componentを与えるcaseはあり得るが、Referenceなしに追加しない。

## 4. Occurrence identity assertion

business occurrence identityは、「何を同じ一件として扱うか」を表す。

概念shape:

~~~json
{
  "data": "coverage.assignment",
  "role": "business-occurrence",
  "components": [
    {"kind":"related-occurrence","relation":"coverage.assignment.contract","end":"target"},
    {"kind":"related-occurrence","relation":"coverage.assignment.asset","end":"target"},
    {"kind":"attribute","attribute":"coverage.assignment.effective_from"}
  ],
  "scope": "...",
  "temporalMeaning": "occurrence"
}
~~~

これはsyntax案ではなく、必要情報のshapeを示す。

### Candidate key

同じterm shapeを使えるが、business occurrence identifierとして選択済みか、代替candidate keyかは区別する。

~~~text
candidate key
  != selected business occurrence identifier
~~~

候補を複数保持できる必要がある。

## 5. History / version identity

history / version identityはoccurrence identityへ混ぜない。

CoverageAssignmentでrevisionを保存する場合:

~~~text
(contract occurrence, asset occurrence, effectiveFrom)
  = business occurrence identity

+ revision
  = history/version identity
~~~

適用開始日が常にversion termになるわけでも、常にoccurrence termになるわけでもない。どちらかはSource-backed business semanticsで決める。

## 6. Functional dependency assertion

functional dependencyはidentityとは別assertionとする。

理由:

1. dependencyには方向がある。
2. dependentはoccurrence全体とは限らず、一部Attributeの場合がある。
3. identityでなくても成立するFDがある。
4. identity candidateが存在しても、business ruleとしてFDがacceptedとは限らない。
5. reverse dependencyは自動で成立しない。

概念shape:

~~~json
{
  "scopeData": "coverage.assignment",
  "determinants": [
    {"kind":"related-occurrence","relation":"coverage.assignment.contract","end":"target"},
    {"kind":"related-occurrence","relation":"coverage.assignment.asset","end":"target"},
    {"kind":"attribute","attribute":"coverage.assignment.effective_from"}
  ],
  "dependents": [
    {"kind":"attribute","attribute":"coverage.assignment.effective_to"}
  ],
  "scope": "...",
  "temporalScope": "..."
}
~~~

これもsyntax決定ではない。

## 7. Relation semanticsは別責務

Relationがidentity componentになるかどうかを決めるには、Relationのbusiness meaningが必要である。

~~~text
relation topology
  != relation business semantics
~~~

したがって、次をRelation shapeだけから決めない。

- contained repeatか
- changeable referenceか
- independent associationか
- Relation先の差し替えが同一occurrenceの更新か
- Relation membershipが時間で変わるか

Relation semanticsをfirst-classにする必要があるかは別Evidence questionとして残す。

## 8. Source / Evidence trace

identity / FD assertionへSource pathを直接埋め込んで第二正本にしない。

Canonical subject identityをSource Index / provenanceへ戻せる既存境界を利用する。

accepted assertionは、少なくとも次へtraceできる必要がある。

- assertion subject
- component / determinant subject
- Relation subject
- Constraint / Contract
- authoritative Source / fingerprint
- Shared Acceptance Decision

## 9. Persistenceとの境界

次の二つは同じbusiness identityを実現できる。

~~~text
surrogate PK + business UNIQUE

composite physical PK
~~~

Core identity assertionはどちらにも依存しない。

Persistence Mappingは、Core assertionのcomponentsをどのcolumn / constraintへ実現したかを追跡する責務を持ち得るが、physical schemaからCore identityを逆生成しない。

## 10. Alternative案の評価

### A. Validationへすべて入れる

**保留。** uniqueness constraintはValidationで表現できるが、business occurrence / candidate key / history versionというsemantic roleを表しにくい。

### B. Persistence keyをbusiness keyへ昇格する

**不採用。** Target / Provider差でmeaningが変わる。

### C. Relation keyから自動導出する

**不採用。** changeable referenceとの区別ができない。

### D. 一つの`Key` primitiveへidentityとFDを統合する

**不採用。** dependency方向、dependent、history identityの区別を失う。

### E. determinant term語彙だけ共通化しassertionを分離する

**現時点の第一候補。** contained repeat / associationの共通構造を再利用しつつ、意味の違いを維持できる。

## 11. 実装開始Gate

次を満たすまでParser / Canonical IRへ新primitiveを追加しない。

1. CoverageAssignment Referenceでbusiness factsがレビュー可能になっている。
2. contained repeatとassociationで同じterm vocabularyが使える。
3. changeable referenceへ誤適用しないnegative testがある。
4. occurrence identityとhistory/version identityを別assertionとして説明できる。
5. surrogate / composite physical keyの双方でmeaningが変わらない。
6. Source / Shared Acceptance traceの置き場所が決まる。
7. first-class Relation semanticsが必要か、既存Source Evidenceで十分かを再評価する。

## 12. current decision

~~~text
shared determinant term vocabulary
  -> Candidate

occurrence identity assertion
  -> Candidate

candidate-key assertion
  -> Candidate

functional-dependency assertion
  -> Candidate

new Core syntax / parser implementation
  -> Not Ready
~~~

次段はsyntax実装ではなく、CoverageAssignment Referenceを既存Business System Design review flowへ通し、どのassertionが本当にdurable meaningとして必要かを確認する。
