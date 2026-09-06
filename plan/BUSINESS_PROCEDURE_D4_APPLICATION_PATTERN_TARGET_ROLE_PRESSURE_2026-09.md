# Business Procedure D4 × Application Pattern Target Role Pressure — 2026-09

Status: **Lane C / test-only composition pressure / O1 overlay / no production schema adoption**

## 1. 問い

D4はcurrent EvidenceからStep bodyのexplicit ref kindを次へ絞っている。

```text
Application Pattern
Validation
Capability
```

ただし、同じApplication Pattern authorityへの参照でも意味上の役割は一つではない。

BP-07では:

```text
Business Action
  -> order_entry.submit_order
  direct Application Function realization
```

である。

一方BP-06では:

```text
Business Action
  -> security.authorize
  supporting Semantic authority

Business Action
  -> order_entry.submit_order
  protected target / concern
  NOT direct realization
```

である。

したがって:

```text
ref kind separation
  !=
ref role separation
```

を追加Pressureとして扱う。

## 2. D4をgeneric role-qualified refへ拡張しない

一例から直ちに次をproduction化しない。

```text
ExplicitSemanticRef {
  Kind
  Role
  Target
}
```

Current Evidenceで強く確認できているrole差は、Application Patternについての:

```text
direct realization
protected target / concern
```

である。

まずO1 field-specific overlayで意味をPressureする。

## 3. O1 overlay candidate

```text
D4TargetAwareStep
  body
    D4 Step body
      businessAction
      applicationPatterns[]  // direct realization only
      validations[]
      capabilities[]

  applicationPatternTargets[]  // protected target / concern
```

`applicationPatternTargets`はtest-only vocabularyであり、current authoring fixtureやproduction Canonical fieldではない。

## 4. BP-06で保持したい非対称性

```text
BP-06
  D4 body
    businessAction = authorization decision
    applicationPatterns = []
    capabilities = [security.authorize]

  target overlay
    applicationPatternTargets = [order_entry.submit_order]
```

これにより:

- BP-06はApplication Functionを持たないまま
- Authorization Semantic authorityへtraceでき
- 何をprotect/concernしているかをexact Application Pattern identityで表せる

という三つを同時に保持する。

## 5. target-only change

```text
protected target
  order_entry.submit_order
    -> order_entry.order_record
```

だけが変わった場合:

- D4 Step bodyは不変
- direct Application Function realizationはnoneのまま
- target-aware cross-view meaningは変化

する必要がある。

これはtarget relationが単なる説明文や隣接Stepからのderived decorationではないことをPressureする。

## 6. adjacencyからderiveしない

BP-07のApplication Patternが変わっても、BP-06 protected targetを自動変更しない。

```text
ordered adjacency
  !=
protected-target relation
```

である。

Business Procedureの順序からsecurity concernを推測してaccepted meaningへ昇格しない。

## 7. exact resolution

Target identityはnormalized `ir.World`でApplication Patternとしてexact resolutionする。

- missing -> fail closed
- ambiguous -> fail closed
- prefix / similarity -> 使用しない

Existing `application_pattern_target_pressure_test.go`がO1単体をPressureし、新しい`d4_application_pattern_target_composition_pressure_test.go`がD4との合成をPressureする。

## 8. Authorization policyを逆流させない

Protected Application targetを表すために、Business Procedureへ次を入れない。

- Principal
- Role
- Organization
- Permission
- allow / deny Effect
- provider policy

`security.authorize`がSemantic authority、Legacy policy / provider realizationはそれぞれの既存boundaryに残る。

## 9. D4への影響

O1はD4 Step bodyそのものへ即時統合しない。

```text
D4
  current minimum Step body

O1
  target-role overlay pressure
```

として比較する。

理由:

1. current strong exampleはBP-06中心
2. generic role taxonomyは未証明
3. current authoring fixtureへtarget fieldを追加するactual-green Evidenceがない
4. `targets / concerns / protects / supports / checks / realizes`全体のtaxonomyを固定するには早い

## 10. Repository Pressure

`d4_application_pattern_target_composition_pressure_test.go`で次を固定する。

1. BP-06 D4 bodyの`ApplicationPatterns`は空のまま
2. `security.authorize` Capability refを保持
3. `order_entry.submit_order`をprotected target overlayとして保持
4. targetをexact resolution
5. target-only changeはD4 bodyを変えずoverlay meaningを変える
6. BP-07隣接realizationからtargetをderiveしない
7. generic `Kind / Role / Policy / Principal / Permission` surfaceを導入しない

actual Repository runnerでgreenになるまではexecution Evidenceとは数えない。

## 11. pre-ADR判断

```text
D4 ref kind separation
  useful / stronger than kind-less bag

O1 Application Pattern target role
  separate representation pressure
  source-level test-only

Generic role-qualified ref model
  defer
```

D4 + O1をactual greenと次のrealistic Referenceへ当て、field-specific target relationで十分か、role-qualified ref abstractionが本当に必要かを決める。

## 12. 非対象

- production `applicationPatternTargets` field採用
- generic explicit-ref Role enum
- Authorization policy schemaのBusiness Procedureへの移動
- Function realization定義の変更
- runtime access-control enforcement
- new Product Requirement / PAS追加
