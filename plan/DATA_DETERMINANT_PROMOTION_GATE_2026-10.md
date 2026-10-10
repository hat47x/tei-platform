# Data Determinant Promotion Gate — 2026-10

Status: **Reference pressure / Core contract promotion not ready**

## 1. 目的

TEIのData Design Contextは、現行Canonical modelにfirst-classなbusiness occurrence identifier、candidate key、functional dependencyが無い状態でも、確定済みのSource / Constraintと未解決の意味を分けて扱えるところまで進んだ。

一方、derived candidateが存在することと、Core contractへ新しいprimitiveを追加すべきことは同じではない。

```text
derived design candidate
  != accepted business meaning
  != reusable Core primitive
```

本Gateは、business occurrence identifier、candidate key、functional dependency等をCoreへ昇格する前に必要なEvidenceを固定する。

## 2. 現在確認できている三類型

### A. 親occurrence内部の反復

Reference: `sales.order.lines -> sales.order_line`

- Relation cardinalityは`many`
- `sales.order.lines.line_number_unique`がaccepted Constraintとして存在する
- 各Relation source occurrenceの範囲で`line_number`が一意であることまでは確定できる
- Data Design Contextは、`relation-source-occurrence + local attribute`をbusiness identity / FDの**candidate**として提示できる
- 親occurrenceのbusiness identifier、Relationのbusiness meaning、target membershipの排他性、時間意味は未確定

この類型だけから「子tableの識別は親キー + 連番」と一般化しない。

### B. 変更可能な参照

Reference shape: `repair.intake -> customer.customer`

- Relationは`one`
- Relation shapeだけでは、参照が固定か変更可能か、identityを構成するかを確定できない
- Relation keyや参照先IDが存在しても、Intakeのbusiness occurrence identifierには昇格しない
- Data Design ContextはRelation topologyを返すが、business meaningは`unresolved`に残す

### C. 独立した識別対象同士の組合せ

Reference pressure shape: association Dataが二つの独立Dataへ`one` Relationを持つ構造。

actual Source-backed negative caseとして、`security.AuthorizationGrant`も確認する。`subjectId`、`action`、`resource`という複数の識別軸を持つが、それらの組合せがbusiness occurrence identityまたはcandidate keyであるというaccepted Constraint / Contractは存在しない。このためData Design Contextは3 AttributeとSource Evidenceを提示しつつ、business identity / FD candidateを生成しない。

- 二つのRelationが存在することだけでは、その組合せがbusiness occurrenceを識別するとは言えない
- association Dataが独自identityを持つ場合、組合せそのものがidentityである場合、履歴versionを持つ場合を区別する必要がある
- 現行Canonical modelには、この違いをaccepted meaningとして表す十分なprimitiveが無い
- したがってbusiness identity / candidate key / FD candidateも自動生成しない

このnegative pressureは、contained repeatの規則をassociationへ一般化しないために必要である。

## 3. Promotion Gate

Core contractへbusiness determinantを追加するには、少なくとも次を満たす。

### G1 — business needがimplementation shapeから独立している

必要性をtable、ORM、UI record、runtime object、physical primary keyだけで説明しない。

### G2 — occurrence identityとhistory / version identityを区別できる

同じbusiness occurrenceの履歴行なのか、別occurrenceなのかをSource / Contractから判断できる。

### G3 — uniqueness scopeが明示される

global、親occurrence内、tenant内、期間内等のscopeを区別できる。

### G4 — Relation semanticsが明示される

少なくとも、変更可能な参照、contained repeat、独立associationを同じRelation shapeとして扱わない。

### G5 — determinant directionを明示できる

functional dependencyは方向を持つ。reverse dependencyを自動で仮定しない。

### G6 — Source / Contractへ追跡できる

accepted business claimは、Canonical subjectとauthoritative Source / Contractへ戻れる必要がある。

### G7 — Persistence差に耐える

同じSemantic Dataが複数Databaseで異なるprimary key / unique constraintを持っても、business meaningが変わらない。

### G8 — Reference diversityがある

少なくとも次をpressureする。

- contained repeat
- changeable reference
- independent association

さらに時間・履歴を持つcaseを別軸として確認する。

一つのReference shapeだけでCore primitiveを決めない。

### G9 — Shared Acceptanceへ接続できる

derived candidateをaccepted meaningへ昇格する場合、Proposal / Review / Accept境界を通る。candidate fingerprint自体をauthorityにしない。

### G10 — primitiveを追加しないnegative caseを説明できる

既存Constraint / Relation / Contractで十分なcase、新primitiveが過剰なcaseを同時に示す。

## 4. current verdict

```text
contained repeat pressure
  -> candidate derivation exists

changeable reference pressure
  -> business meaning unresolved

independent association pressure
  -> business identity unresolved

temporal/history boundary
  -> unresolved / negative pressure exists

cross-database realization pressure
  -> physical keys do not change logical authority

first-class business determinant Core contract
  -> Not Ready
```

Not Readyの主因は、実装不足ではなく**accepted business semanticsのReference diversity不足**である。

現在の`businessIdentityCandidates` / `functionalDependencyCandidates`は、この不足を隠さず次に確認するEvidenceを示すためのderived projectionとして維持する。

## 5. 次のReference pressure

次段では新しいCore syntaxを先に作らない。

association Dataのpositive pressureとして、`reference/business-system-design/data-determinant-association-pressure.md`を追加した。CoverageAssignmentを用い、ServiceContract occurrence + Asset occurrence + effectiveFromがbusiness occurrence identityを構成し得る一方、effectiveToやhistory revisionは別責務であること、surrogate primary keyの有無でbusiness meaningを変えないことをReference business factとして固定した。

このReferenceにより、現行Core contractではSource-backedなbusiness occurrence identity / scoped FDをaccepted meaningとして直接authorできないrepresentational gapが具体化した。ただし、syntax採用はまだ行わない。

次に優先するのは、association Dataについて次をSource-backedに説明できるReferenceをさらに検証することである。

1. 何を一つのbusiness occurrenceと数えるか
2. 構成Dataの組合せがidentityなのか
3. association自体に独立identityがあるか
4. Relation先の差し替えが同一occurrenceの変更か、別occurrenceか
5. 同じ組合せを時間を変えて再作成できるか
6. Persistence primary keyを変えてもmeaningが維持されるか

このReferenceが成立した後、contained repeatとの共通部分だけをCore contract候補へ抽出する。

## 6. Product Value

このGateにより、生成AIが「とりあえず単独ID」「親ID + 連番」「二つのFKだから複合キー」といった一般的schema patternを、業務意味の確認なしに確定することを防ぐ。

同時に、すべてを`unresolved`のまま放置するのではなく、accepted Constraintから安全に導けるcandidateと、追加確認が必要なEvidenceを機械可読に保つ。

結果として、AI支援Data設計の速度を保ちながら、physical schema都合でbusiness meaningを失うリスクを下げる。
