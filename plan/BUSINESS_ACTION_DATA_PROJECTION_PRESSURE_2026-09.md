# Business Action × Data Projection Pressure — 2026-09

Status: **Lane C / T-GAP-03 test-only pressure / derived projection / no production schema adoption**

## 1. 目的

Business Procedureの各Business Actionについて、既存Canonical / Application meaningからData / Relation traceをどこまで機械的に導出できるかを確認する。

狙いはCRUD matrix等のcross-view reviewに使えるprojectionを得ることであり、Business Procedure sourceへData依存を複製することではない。

また:

```text
traceがない
  != Dataと無関係
```

を保持する。

## 2. Authority direction

Projectionは次のaccepted identityだけを辿る。

```text
Business Step
  -> Application Pattern
     -> Data
     -> Relation
        -> source Data
        -> target Data
     -> Capability
        -> input/output/outcome Data payload

Business Step
  -> Validation
     -> Attribute owner Data
     -> Relation
        -> source/target Data

Business Step
  -> explicit Data / Relation
     only when source actually contains them

Business Step
  -> explicit Capability
     -> Data payload contract
```

Capabilityのinput/output/outcome payload IDはSemantic DataだけでなくSemantic Typeも指し得るため、Data projectionではkindを推測しない。

```text
payload ID -> exact Data 1件 / Type 0件
  -> Data traceとして採用

payload ID -> DataとTypeの両方に存在
  -> kind ambiguityとしてfail closed

payload ID -> Data 0件
  -> Type等のnon-Data payloadであり得るためDataへ変換しない
```

したがって「Dataとして1件見つかった」という理由だけで、同じCanonicalIDの別kindを無視してData authorityを確定しない。

使わない:

- Business Action wordingからのentity name抽出
- Step adjacency
- Pattern ID prefix
- O1 test helper targetの暗黙利用
- datasource名からDataの推測
- Database ownershipからBusiness dependencyを生成
- Query実装内部の推測

## 3. Projection shape

Test-only row:

```text
BusinessActionDataProjectionRow
  StepID
  Data[]
  Relations[]
  EvidencePaths[]
  NoDataTraceFromCurrentRefs
```

`NoDataTraceFromCurrentRefs=true`は:

> current explicit refsからData / Relation traceを一本も構成できない

ことだけを意味する。

falseでも「依存Dataを完全に列挙した」とは主張しない。

したがってprojectionに`Complete` / `Authoritative` / `Ownership` / `Access`等のfieldを置かない。

## 4. Current order_entry projection

Test-only projectionから期待する主要行:

| Step | Derived Data | Derived Relation | 備考 |
| --- | --- | --- | --- |
| BP-00 | `sales.order` | - | browse PatternのData |
| BP-01 | `sales.order` | - | record PatternのData |
| BP-02 | `sales.order`, `customer.customer` | `sales.order.customer` | reference Pattern Relationの両端 |
| BP-03 | `sales.order`, `sales.order_line` | `sales.order.lines` | header-detail Relationの両端 |
| BP-04 | traceなし | traceなし | manual judgment。No Dataとは言わない |
| BP-05 | `sales.order`, `sales.order_line` | `sales.order.lines` | Validation Attribute/Relationからderive |
| BP-06 | `security.authenticated_subject`, `security.authorization_grant`, `security.authorization_denial` | - | explicit `security.authorize` contractからderive |
| BP-07 | `sales.order` | - | Pattern-owned `sales.order.submit` input/outputからderive |
| BP-08 | traceなし | traceなし | Observation intentionだけ。runtime ReadIdentityからDataを作らない |
| BP-09 | traceなし | traceなし | manual/external continuation。No Dataとは言わない |

## 5. BP-05 — Validationを名前ではなく構造から辿る

Semantic Validation contractは:

```text
positive
  -> Attribute

relation-unique
  -> Relation
  -> Attribute
```

を持つ。

Current Referenceでは:

```text
sales.order_line.quantity_positive
  -> sales.order_line.quantity
  -> owner Data sales.order_line

sales.order.lines.line_number_unique
  -> sales.order.lines
  -> source Data sales.order
  -> target Data sales.order_line
  -> sales.order_line.line_number
```

とexactに辿れる。

Validation ID文字列から`order_line`等を切り出していない。

## 6. BP-06 — partial traceをcompleteと呼ばない

BP-06のauthoritative Business sourceには:

```text
capability = security.authorize
```

があるため、そのCapability contractからsecurity Dataをderiveできる。

一方、O1でPressureしている:

```text
protected Application target = order_entry.submit_order
```

は現在test helper inputであり、authoritative Business sourceへpersistしていない。

したがってcurrent Business Action × Data projectionはBP-06へ`sales.order`を追加しない。

```text
security Data trace exists
  != BP-06 Data dependency is complete
```

である。

BP-07隣接realizationやBusiness Action wordingから`order_entry.submit_order` / `sales.order`を補完しない。

この不足はO1 source persistenceを直ちに採用する根拠ではないが、protected-target relationをaccepted meaningとして残す場合にprojection / impact analysisへ効く具体例になる。

## 7. BP-04 / BP-08 / BP-09 — unknownをunknownのまま残す

Current refsからData traceを作れないStepは:

```text
NoDataTraceFromCurrentRefs = true
```

とする。

表示上:

```text
unknown / no current trace
```

と扱うべきであり:

```text
no data dependency
```

とは扱わない。

これはP11 `unresolved remains unresolved`と一致する。

## 8. Data authorityとの分離

本projectionはData identityを表示するが、次は所有しない。

- Database ownership
- datasource access
- write authority
- managed-here / external-observed policy
- physical realization

それらは既存Application / Database / Provider authorityから別projectionとしてcross-checkする。

`BUSINESS_PROCEDURE_SECOND_CORPUS_PRESSURE_2026-09.md`（Part 2、旧`SECOND_CORPUS_DATA_AUTHORITY_DEFERRAL`）で保留したmanufacturing / inventory ownershipを、本projectionを理由にsynthetic化しない。

## 9. CRUD matrixへの含意

Watanabe / X-TEA pressureと同様、matrixはstructured designからderiveする。

Current pressureでまず安全に作れるのは:

```text
Business Action × Data/Relation trace matrix
```

である。

ただしC/R/U/Dを付与するには、Pattern / Query / Capability / datasource contractからoperation semanticsをさらにEvidence-backedに導出する必要がある。

Data identityが見えたことだけでCRUD operationを推測しない。

## 10. Derived-only change boundary

Projectionが第二正本ではなくderived viewであることを次でPressureする。

```text
Business wording-only change
explicit refs unchanged
  -> Business accepted meaning may change
  -> Data projection unchanged

Application Pattern / Validation / Capability authority change
Business source unchanged
  -> Data projection changes when referenced meaning changes
```

したがって依存方向は:

```text
Business source + normalized referenced authorities
  -> derived Data projection

Data projection
  -X-> author Business source
  -X-> author Application / Semantic meaning
```

とする。

またmissing / ambiguous referenced identityは名称推測で補わずfail closedする。Capability payloadのように複数semantic kindを取り得る参照では、Data/Type kind ambiguityもfail closed対象とする。

このboundaryは「Data projectionを豊かにしたい」という理由だけでBusiness sourceへdirect Data refやO1 targetを追加することも禁止する。

## 11. T-GAP-03の現在回答

問い:

> Can a Business Action × Data matrix be derived far enough from existing Pattern/Query/Capability references to be useful while keeping unknown cells genuinely unknown?

Test-only回答:

```text
yes, useful trace rows can be derived
and
no-trace rows can remain explicitly unresolved
without authored Data duplication
```

ただし:

```text
projection completeness
CRUD operation completeness
O1 protected-target Data coverage
actual runner green
```

は未証明である。

したがってT-GAP-03は「構造上の回答を得た / actual Evidence待ち」とする。

## 12. Production contractへの影響

現時点で不要:

- Business Procedureへのmandatory Data refs
- D4へのData / Relation slot再追加
- generic dependency Edge
- Data projectionのCanonical化
- matrixの第二SOT化

Current D4のままでも、Pattern / Validation / CapabilityからかなりのData traceをderiveできる。

将来current refsでは失われるaccepted Business dependencyが具体的に出た場合だけ、direct Data / Relation refをreopenする。

## 13. Verification status

Repository上へ`business_action_data_projection_pressure_test.go`を追加した。

Derived-only change boundaryに加え、Capability payloadのData/Type kind ambiguityをDataへ暗黙解決しないtest-only Pressureを追加した。

actual Repository runnerでの:

```text
gofmt
go vet
go test
checker
```

は未実行のためPASSを主張しない。
