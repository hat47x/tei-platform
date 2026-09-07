# Business Action × Semantic Namespace Lens Pressure — 2026-09

Status: **Lane C / T-GAP-04b partial answer / derived comprehension lens / no SubjectArea production contract adoption**

## 1. 目的

T-GAP-04では、X-TEAの`Subsystem` / `SubjectArea`が持っていた業務システム理解の境界を、current TEIの既存authorityからどこまでderived viewとして回復できるかを確認している。

`BUSINESS_ACTION_OWNERSHIP_BOUNDARY_PROJECTION_PRESSURE_2026-09.md`では、Application / Database ownership/access + evidence-backed CRUDからoperation ownership boundaryを部分的にderiveできることを確認した。

本Pressureでは別の軸として、Business Action × Data traceを既存Semantic Namespaceへ投影し、**comprehension / navigation lens**として有用かを確認する。

`Namespace == SubjectArea`とは仮定しない。

## 2. Authority chain

Business Action sourceからNamespaceを直接authorしない。

Current chain:

```text
Business Action
  -> exact Application Pattern / Validation / Capability ref
     -> Business Action × Data projection
        -> exact Semantic Data owner Namespace
           -> derived Semantic Namespace lens
```

つまり:

```text
Business wording
  -X-> Namespace

Data ID prefix string parsing
  -X-> Namespace

file path
  -X-> Namespace
```

ではなく、normalized `ir.World.Semantics`でData identityをexactに所有するNamespaceを解決する。

## 3. Current Reference rows

`business_action_semantic_namespace_lens_pressure_test.go`ではcurrent `order-entry.order-handling`から次を期待する。

```text
BP-00 -> sales
BP-01 -> sales
BP-02 -> customer + sales
BP-03 -> sales
BP-05 -> sales
BP-06 -> security
BP-07 -> sales

BP-04 -> no current Namespace trace
BP-08 -> no current Namespace trace
BP-09 -> no current Namespace trace
```

`no current Namespace trace`は:

```text
no business area
```

を意味しない。

元になるData traceがcurrent refsからないため、Namespace lensもunknownを保持しているだけである。

## 4. Why the lens is useful

### 4.1 Cross-domain dependency visibility

BP-02は:

```text
sales.order
  -> relation sales.order.customer
  -> customer.customer
```

を通じて:

```text
sales + customer
```

へまたがる。

Business Procedureだけを見るより、同じActionがどのSemantic knowledge domainへ依存するかを局所的に確認しやすい。

### 4.2 Cross-cutting semantic concern visibility

BP-06は`security.authorize`のaccepted payloadからsecurity Data traceを持ち、Namespace lensでは:

```text
security
```

として現れる。

これはBusiness Procedureの主要業務がsalesであっても、securityというcross-cutting semantic concernが独立して見えることを示す。

### 4.3 No second authored classification

Business sourceへ:

```text
subjectArea: sales
```

等を追加しなくても、現在のsemantic authorityからlensを再計算できる。

Data / Capability / Relationのaccepted meaningが変わればlensも追随する。

## 5. Why Namespace is not SubjectArea

この結果から:

```text
Semantic Namespace == Business SubjectArea
```

とは結論しない。

### 5.1 One Business Action may span namespaces

BP-02は`sales + customer`へまたがる。

これをどちらか一方のSubjectAreaへ自動分類すると、cross-view pressureを失う。

### 5.2 Cross-cutting namespace is not necessarily a business area

`security`は明確なSemantic Namespaceとして有用だが、受注業務のBusiness SubjectArea taxonomyで独立SubjectAreaにすべきかは別問題である。

### 5.3 Namespace is an authority/package boundary

Current NamespaceはSemantic Data / Type / Capability / Validationのdefinition groupingを所有する。

Business SubjectAreaが必要になる場合、それは:

```text
business comprehension
review scope
business vocabulary grouping
```

等を目的とする可能性があり、Semantic definition packagingと一致する保証はない。

### 5.4 SubjectArea may aggregate or split namespaces

Future realistic Referenceでは:

```text
one SubjectArea spans multiple namespaces
or
one namespace contains several business comprehension areas
```

のどちらもあり得る。

current one-domain Referenceだけで固定しない。

## 6. T-GAP-04bをさらに分解する

### T-GAP-04b1 — navigational semantic-domain lens

```text
Can existing Semantic Namespace ownership provide a useful navigation /
comprehension lens for Business Actions without new SubjectArea identity?
```

Current answer: **partial yes for current Reference**。

- exact Data authorityからderive可能
- cross-namespace dependencyを残せる
- security等のcross-cutting semantic concernも見える
- Business sourceへ分類を複製しない

### T-GAP-04b2 — durable business SubjectArea identity

```text
Does realistic business review require a durable grouping identity that is not
reducible to Semantic Namespace / Application / Database ownership?
```

Current answer: **still open**。

この問いだけが将来の`SubjectArea` Canonical conceptを正当化し得る。

## 7. Relation to operation ownership view

二つのderived viewは同じものではない。

```text
Semantic Namespace lens
  answers: which semantic domains does this Business Action touch?

Ownership boundary lens
  answers: through which Database ownership/access boundary does an accepted
           operation occur?
```

例:

```text
BP-02
  semantic domains:
    sales + customer

  accepted operation boundary:
    R customer.customer
    -> master_data
    -> external-observed / read
```

この重ね合わせは、Subsystem / SubjectArea Canonical nounなしでも三要素cross-reviewに有用な情報を与える。

しかし:

```text
semantic domain
!= database ownership
!= business subject area
!= application ownership
```

を維持する。

## 8. X-TEA fallback

Historical X-TEAは`Subsystem`と`SubjectArea`を別conceptとして扱っていた。

この事実は、current TEIでも:

```text
Namespace
Database ownership
Application
Business comprehension grouping
```

を一つへ潰さないためのnegative design pressureとして使う。

Historical nounをそのまま復活させるのではなく、まず既存authorityから複数lensをderivedし、それでも失われるreview meaningだけをfuture concept candidateとする。

## 9. Current production decision

追加しない。

- `SubjectArea` Canonical kind
- Business Procedure `subjectArea` field
- Namespace -> SubjectArea alias
- Application -> SubjectArea alias
- Database -> Subsystem alias
- automatic primary Business Area assignment
- one-namespace-per-Business-Action restriction

CurrentはSemantic Namespace lensをderived review projectionとして保持する。

## 10. Current T-GAP-04 state

```text
T-GAP-04a
operation ownership boundary
  -> partial positive evidence

T-GAP-04b1
semantic-domain navigation / comprehension lens
  -> partial positive evidence

T-GAP-04b2
durable SubjectArea identity
  -> unresolved

T-GAP-05
effective CUD / call topology / subsystem cohesion at realistic scale
  -> unresolved
```

この結果により、current Referenceでは新しい`Subsystem` / `SubjectArea` nounを導入せずとも、X-TEAが要求したcross-view理解の一部を複数のderived lensとして回復できる。

Actual Repository runnerによる`gofmt / go vet / go test` greenは未取得であり、本結果はtest-only / pre-contract Pressureである。
