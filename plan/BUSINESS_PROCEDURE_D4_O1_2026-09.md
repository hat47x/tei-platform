# Business Procedure D4 (Step body minimality) + O1 (protected-target role) — 2026-09

Status: **Lane C / test-only architecture pressure / D4 + narrow O1 current minimum candidate / no production Core adoption**

## この文書について

本書は、Business Procedure Step bodyの最小比較候補D4と、Application Pattern参照の役割分離(protected target vs direct realization)を扱うO1を、時系列で調査した次の7文書を1本へ統合したものである。

1. `BUSINESS_PROCEDURE_D4_REF_KIND_PRESSURE_2026-09.md`（元・D4 Step body単体のref kind pressure）
2. `BUSINESS_PROCEDURE_APPLICATION_PATTERN_TARGET_PRESSURE_2026-09.md`（元・O1概念そのものの最初の導入、target≠realizationのtest-only shape）
3. `BUSINESS_PROCEDURE_D4_APPLICATION_PATTERN_TARGET_ROLE_PRESSURE_2026-09.md`（元・D4×O1 overlayの導入）
4. `BUSINESS_PROCEDURE_O1_AUTHORITY_DIRECTION_PRESSURE_2026-09.md`（元・O1の authority方向pressure、X-TEA E-015境界）
5. `BUSINESS_PROCEDURE_D4_O1_MINIMUM_CONTRACT_AUDIT_2026-09.md`（元・D4単独の損失性監査、D4+narrow O1への収束）
6. `BUSINESS_PROCEDURE_O1_CARDINALITY_PRESSURE_2026-09.md`（元・O1のscalar/collection cardinality監査）
7. `BUSINESS_PROCEDURE_O1_FILE_NATIVE_ACCEPTANCE_PRESSURE_2026-09.md`（元・O1のfile-native representation / Shared Acceptance pressure）

いずれも同一の"D4 Step body minimality + O1 protected-target role"という一続きの調査であり、後続文書は前段の結論を明示的に引用・縮小・拡張している。個別ファイルの履歴はGit historyから追跡できる。

外部から旧ファイル名を参照していた文書（`plan/issue/000042-business-procedure-cross-view-reference.md`、`plan/BUSINESS_PROCEDURE_EXPLICIT_REF_AUTHORITY_PRESSURE_2026-09.md`、`plan/BUSINESS_PROCEDURE_MAIN_BASELINE_REAUDIT_2026-09.md`、`plan/DESIGN_SYNTHESIS_ORDER_ENTRY_PROOF_REFERENCE_2026-09.md`、`plan/LANE_B_A70_A72_SECOND_CORPUS_OUTLINE_AUDIT_2026-09.md`）は本統合と同じPRで本ファイル名へ参照を更新した。

---

# Part 1 — D4 Evidence-Backed Ref Kind Pressure（D4単体の最小化）

## 1. 問い

D3 candidateはStep bodyにtyped explicit refsを残している。

```text
ApplicationPatterns
Data
Relations
Validations
Capabilities
```

しかしcurrent `order_entry` Referenceで、Business Action自身がPatternから独立してdirect Data / Relation identityを保持しなければ意味を失う例はまだ存在しない。

一方、Application Pattern / Validation / Capabilityについてはcurrent Referenceで必要性が実証されている。

そこでD3より一段小さい**Step body比較候補**D4を置く。

```text
D4 = D3 body
     - unproven direct Data ref slot
     - unproven direct Relation ref slot
     + fail-closed reopen rule
```

D4はproduction採用案ではなく、current Evidenceより強いsurfaceを先取りしていないか確認するtest-only Pressureである。

Step addressabilityはD4自身へ入れず、S4-Aとして独立に合成する。

## 2. current Evidenceで実証済みのref kind

### Application Pattern

```text
BP-00 -> order_entry.recent_orders
BP-01 -> order_entry.order_record
BP-02 -> order_entry.customer_reference
BP-03 -> order_entry.order_lines
BP-07 -> order_entry.submit_order
```

Pattern-owned Data / Relation / Query / Capability / Datasource / owner Applicationはnormalized Worldからderiveする。

### Validation

```text
BP-05
  -> sales.order_line.quantity_positive
  -> sales.order.lines.line_number_unique
```

BP-03 line reviewへ同じValidation authorityを重複しない。

### Capability

```text
BP-06 -> security.authorize
```

BP-07は`order_entry.submit_order` Patternから`sales.order.submit`をderiveできるためCapabilityを重複保持しない。

## 3. D4 test-only body shape

```text
Procedure body
  id
  purpose
  trigger
  actor
  ordered step bodies[]
    businessAction
    applicationPatterns[]
    validations[]
    capabilities[]
```

D4 bodyはStep local keyを所有しない。

```text
D4 body meaning
  !=
S4-A Step address metadata
```

current testではsource-local Step keyを使ってD4 bodyを検査対象へ対応付けるが、これはD4 bodyのfieldではない。

## 4. authority kindを失わない

D4ではrefをgeneric `[]CanonicalID`やkind-less `semanticRefs[]`へ畳まない。

```text
Application Pattern ref
Validation ref
Capability ref
```

は異なるauthority kindである。literal ID文字列が偶然同じでも同一refとして扱わない。

RQ-026のcross-view correctionでは、どのauthorityへ戻るべきかを失わないことが重要である。

## 5. Data / Relation direct refsの扱い

current Referenceからはminimum必要性が未証明である。

したがってD4 bodyはcurrent ReferenceではData / Relation slotを持たない。

ただし将来sourceにdirect Data / Relation refが現れた場合、D4はそれを黙って捨てない。

```text
future source contains direct Data / Relation ref
  -> D4 derive fails closed
  -> explicit-ref designをreopen
  -> realistic Referenceで必要性を再評価
```

これはData / Relation direct refsの永久禁止ではない。

## 6. なぜfail closedにするか

D4が単にData / Relationをdropすると、future Referenceが新しいaccepted meaningを要求した時にsemantic lossを見逃す。

P11とRQ-003/RQ-017に従い、未対応の意味を「不要だった」と扱わない。

D4はcurrent minimumを狭くする一方、future extensionを黙殺しない。

## 7. D3との比較

```text
D3 body
  businessAction
  ApplicationPatterns
  Data
  Relations
  Validations
  Capabilities

D4 body
  businessAction
  ApplicationPatterns
  Validations
  Capabilities

future direct Data / Relation appears
  -> fail closed / reopen
```

D4がcurrent Referenceに対して意味保存できるなら、production ADRでD3のData / Relation slotを先行採用する根拠は弱くなる。

別のrealistic Referenceでdirect Data / Relationが必要ならD4は棄却または拡張する。

## 8. `realization`との関係

D4でも`realization`をproduction meaningへ戻さない。

ref kindはauthoring labelから推測しない。

- `validation-support`からValidationをguessしない
- `security-boundary`からCapabilityをguessしない
- `application-use`からPatternをguessしない

explicit refsはsourceに明示され、normalized authorityへexact resolutionされる。

## 9. S4-A / H1 / R3との合成

D4はStep bodyのminimum比較だけを担当する。

別pressureで次を合成する。

```text
D4 body
  + S4-A Procedure-scoped Step local address
  + H1/R3 handoff-only normalized relation
```

`d4_normalized_composition_pressure_test.go`でsource-level compositionを実装済み。

合成後も:

- D4 bodyへStep local addressを混ぜない
- stale handoff endpointをfail closed
- address + endpoint協調renameを許容
- Handoff condition changeをbodyから独立したBusiness meaning差として保持
- future direct Data / Relation refをaddress/handoff layerで隠さずD4 derivationでfail closed
- normalize後のmodelへduplicate Handoff relationが混入してもnormalized validator自身がfail closed

をPressureする。

最後のduplicate検査はsource-side `deriveCanonicalBusinessHandoffProbes()`の事前検査だけに依存しない。normalized modelだけでH1/R3 integrityを閉じるという契約を維持するためである。

D4からgeneric Business Flow / Edge / Runtime Workflowを導かない。

## 10. Repository Pressure（D4単体）

`tools/reference-business-procedure-check/d4_ref_kind_pressure_test.go`でD4 body単体を、`d4_normalized_composition_pressure_test.go`でS4-A/H1/R3との合成をPressureする。

D4 body単体:

1. D4 bodyがBusiness Action / Pattern / Validation / Capabilityだけを持つ
2. D4 bodyはData / Relation / Realization / Role / Kindを持たない
3. current ReferenceのBP-03/BP-05 authority分離を保持
4. BP-06 `security.authorize`を保持
5. BP-07へPattern-owned Capabilityをコピーしない
6. BP-04/BP-08/BP-09へfake refを付けない
7. direct Data refが現れたらfail closed
8. direct Relation refが現れたらfail closed
9. literal identity textが同じでもauthority kindを区別

Composition:

10. D4 bodyとS4-A address layerを分離
11. stale / missing Handoff endpointをfail closed
12. address + Handoff endpoint協調renameを許容
13. condition changeをHandoff meaning差として保持
14. duplicate Handoff relationをnormalized layerでfail closed
15. new ref kindをcomposition layerで隠さない

actual Repository runnerでgreenになるまではexecution Evidenceとは数えない。

## Part 1 pre-ADR判断

```text
D3
  broader Step body comparison candidate

D4
  smaller evidence-backed Step body candidate
  current Referenceで比較する価値あり
  not adopted

S4-A
  addressability concern
  D4とは独立
```

production contractでref kindをどこまで持つかは、current `order_entry`だけでなく次のrealistic Referenceから決める。

current Evidenceだけを理由にgeneric `semanticRefs[]`やdirect Data / Relation slotを先取りしない。

---

# Part 2 — Application Pattern Target Pressure（O1概念の最初の導入）

## 1. 問い

Lane B B-F23 / B-F24から、BP-06 Authorization concernには次の三つが同時に存在することが分かった。

```text
supporting semantic authority
  security.authorize

protected Application target
  order_entry.submit_order

direct Application Function realization
  none
```

current Reference checkerでは`applicationPatterns[]`がdirect Application Function realizationを表している。

したがってBP-06へ単純に:

```text
applicationPatterns:
  - order_entry.submit_order
```

を追加すると:

```text
BP-06 concerns/protects submit action
  -> BP-06 is realized by submit action
```

へ意味を誤変換する。

本Pressureでは、Application Patternという同じauthority kindへのrefでも、**target / concernとrealizationを混同しない最小表現が可能か**をtest-onlyに確認する。

## 2. D4との関係

D4はcurrent Evidenceから未証明のdirect Data / Relation slotをminimum candidateからsubtractし、Application Pattern / Validation / Capabilityというauthority kindを保持する比較候補である（Part 1参照）。

しかし:

```text
ref kind separation
  !=
ref role separation
```

である。

BP-06とBP-07はいずれもApplication Pattern identityへ関係するが、意味は異なる。

```text
BP-07 -> order_entry.submit_order
  direct realization

BP-06 -> order_entry.submit_order
  protected target / concern
  not realization
```

したがってO1はD4を置き換えるものではなく、D4だけでは解けない別pressureである。

## 3. O1 test-only shape

Production fieldを先に固定せず、test-only candidateとして次を比較する。

```text
TargetAwareStepProbe
  body
    businessAction
    applicationPatterns[]  // current D3/D4 body。direct realization only
    validations[]
    capabilities[]
  applicationPatternTargets[]  // test-only protected target / concern
```

ここで`applicationPatternTargets`は**比較用語彙**である。

- current `step` fixture schemaへ追加しない
- `business-procedure-reference-v0`を変更しない
- production Canonical fieldとして採用しない
- generic ref `role` enumを導入しない

まず意味の必要性だけを検証する。

## 4. source-level test-only Pressure

`tools/reference-business-procedure-check/application_pattern_target_pressure_test.go`で次を実装した。

### O1-1 target != realization

BP-06について:

```text
ApplicationPatterns = []
ApplicationPatternTargets = [order_entry.submit_order]
Capabilities = [security.authorize]
NoApplicationFunction = true
```

を同時に保持できること。

### O1-2 exact target resolution

`order_entry.submit_order`をnormalized `ir.World`でexactに1件解決する。

名称類似や文字列prefixから補完しない。

### O1-3 missing / ambiguous fail closed

missing targetは:

```text
BUSINESS_APPLICATION_PATTERN_TARGET_REF_NOT_FOUND
```

ambiguous targetは:

```text
BUSINESS_APPLICATION_PATTERN_TARGET_REF_AMBIGUOUS
```

としてblocking diagnosticへできることをPressureする。

このdiagnostic kind自体もtest-only comparison vocabularyであり、production Conformance code採用ではない。

### O1-4 target-only change

```text
order_entry.submit_order
  -> order_entry.order_record
```

のようにtargetだけを別のaccepted Patternへ変更した場合:

- BP-06 Business bodyは不変
- direct Application Function realizationはnoneのまま
- target-aware cross-view meaning差は観測できる

ことを確認する。

### O1-5 adjacencyからderiveしない

BP-07のdirect realization Patternを変更しても、BP-06 targetを自動変更しない。

```text
ordered adjacency
  != Authorization protected-target relation
```

を固定する。

### O1-6 Legacy policy shapeを逆流させない

Test-only target candidateへ次を入れない。

- organization code
- role code
- allow / deny effect
- Principal
- Permission
- provider policy

Legacy Auth Referenceに`execute + order_entry.submit_order` ruleが存在することはtarget identityのcross-view corroborationには使えるが、Business Procedureのauthorityにはしない。

## 5. なぜauthoring fixtureへまだ追加しないか

O1がsource-level testとして説明できても、ただちにPrimary fixtureへ新fieldを追加すると:

```text
representation pressure found
  -> authoring schema adopted
```

という先行固定になる。

現在必要なのはまず:

```text
target / concern relation carries accepted meaning
  AND
it cannot be represented as realization
```

をactual runnerでPressureすることである。

actual green後に、次を改めて比較する。

1. current Business wordingだけで十分か
2. O1のfield-specific authoring refが必要か
3. D4 bodyへどの形で統合するか
4. 複数realistic Referenceで同じrole差が反復するか

## 6. generic role-qualified refをまだ作らない

次はdeferする。

```text
ExplicitSemanticRef {
  Kind
  Role
  Target
}
```

理由:

- current強いrole pressureはBP-06一例
- BP-03/BP-05 Validation重複はref削減で解けた
- Observationは独立Semantic authority自体が未証明
- `supports / concerns / protects / checks / realizes` taxonomyは未証明

したがって:

```text
O1 field-specific pressure
  before
O2 generic role taxonomy
```

を維持する。

## 7. RQ / PASへの対応

新しいProduct Requirement / PAS / E7 / Acceptance IDは追加しない。

RQ-025は既に:

```text
Business Action != Application Function
どのApplication Functionをどう利用するかを設計できる
```

を要求している。

RQ-026はBusiness ProcedureからApplication Pattern / Action / Capabilityへのtraceabilityを要求している。

O1はこの既存RequirementをReference representationへPressureする。

PAS-07では既存E7-01 / E7-02 / E7-04 / E7-09で扱える。

## Part 2 現時点の判定

```text
protected target exact identity need
  Confirmed as representation pressure

applicationPatterns realization field reuse
  Reject

adjacency / wording / Legacy policy derivation
  Reject

O1 field-specific target-aware test shape
  Source-level implemented
  actual execution PASS（Gate A、R = e2856e6、2026-09-20）

authoring fixture field adoption
  Not decided

generic role-qualified ref
  Deferred

production schema change
  Not proven
```

## Part 2 次

1. actual Repository environmentでO1 testを実行する。
2. O1がgreenでも即authoring schemaへ昇格しない。
3. D4 ref-kind pressureとO1 role pressureを組み合わせたとき、current meaningを最小に保持できるか比較する（→ Part 3）。
4. 複数realistic Referenceでtarget/concern roleが反復するまでgeneric role taxonomyを作らない。
5. Business ProcedureからLegacy Authorization policyを参照・複製しない。
6. actual greenまでproduction Canonical ADRへ進まない。

---

# Part 3 — D4 × Application Pattern Target Role Pressure（O1の導入）

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

## 10. Repository Pressure（D4×O1導入）

`d4_application_pattern_target_composition_pressure_test.go`で次を固定する。

1. BP-06 D4 bodyの`ApplicationPatterns`は空のまま
2. `security.authorize` Capability refを保持
3. `order_entry.submit_order`をprotected target overlayとして保持
4. targetをexact resolution
5. target-only changeはD4 bodyを変えずoverlay meaningを変える
6. BP-07隣接realizationからtargetをderiveしない
7. generic `Kind / Role / Policy / Principal / Permission` surfaceを導入しない

actual Repository runnerでgreenになるまではexecution Evidenceとは数えない。

## 11. authoritative source boundary — deferred

Current O1 / D4+O1 testsではprotected targetをtest helper引数として与えている。

したがって現在証明しているのは:

```text
protected target role separation is representable
```

までであり、次とは同一ではない。

```text
protected target is persisted in authoritative Business source
protected target survives decode / reload as the same accepted meaning
protected target participates in Shared Acceptance / stale detection
```

Primary `business-procedure-reference-v0` fixtureへ`applicationPatternTargets`等を追加することは、actual green前には行わない。

O1 actual green後、protected-target relationそのものをproduction候補のaccepted Business meaningとして残す場合だけauthoritative sourceをreopenする。その場合は少なくとも:

1. file-native authoring sourceからtarget roleを明示できる
2. decode / normalize / reload後も同じApplication Pattern identityとroleを保持する
3. source revision / fingerprint変更時にShared Acceptanceを暗黙carry-forwardしない
4. missing / ambiguous targetをfail closedする
5. pure source locator moveだけではtarget meaningを変えない
6. RQ-002 / RQ-003 / Shared Acceptanceのauthority境界と矛盾しない

ことをPressureする。

逆にactual greenと次のrealistic ReferenceからO1をproduction accepted meaningとして保持する必要性が出なければ、fixture fieldを追加する理由にはしない。

したがって:

```text
O1 representation feasibility
  !=
O1 authoritative-source adoption
```

を固定する。

## Part 3 pre-ADR判断

```text
D4 ref kind separation
  useful / stronger than kind-less bag

O1 Application Pattern target role
  separate representation pressure
  source-level test-only

O1 authoritative source
  intentionally deferred until actual green + necessity decision

Generic role-qualified ref model
  defer
```

D4 + O1をactual greenと次のrealistic Referenceへ当て、field-specific target relationで十分か、role-qualified ref abstractionが本当に必要かを決める。

O1をproduction accepted meaningとして残す場合は、authoritative source / reload / Shared Acceptance boundaryを閉じるまでproduction Canonical contract ADRへ進めない。

## Part 3 非対象

- production `applicationPatternTargets` field採用
- generic explicit-ref Role enum
- Authorization policy schemaのBusiness Procedureへの移動
- Function realization定義の変更
- runtime access-control enforcement
- new Product Requirement / PAS追加

---

# Part 4 — O1 Authority Direction Pressure（authority方向とX-TEA境界）

## 1. 目的

O1の`protected Application target`について、Authorization側とBusiness Procedure側のどちらがaccepted meaningを所有するかを分離する。

Current case:

```text
BP-06 Business Action
  determine whether the authenticated subject may execute the submit application action

supporting Semantic authority
  security.authorize

protected Application target
  order_entry.submit_order

direct Application Function realization
  none
```

ここでAuthorization policyに`order_entry.submit_order`が存在することを理由に、Business protected targetをpolicyから逆算しない。

## 2. X-TEA fallback — E-015

### Evidence

XeadModelerでは`TaskAction`と`TaskFunctionIO`が分離され、`TaskFunctionIO`が`FunctionID` / `IOID`を保持する。

これは少なくとも次を支持する。

```text
Business Action
  != Application Function

Business Action
  may use / refer to Function or IO
```

### Evidenceが証明しないこと

このsource evidenceから、TEIのgenericなref role taxonomy:

```text
realizes
protects
supports
concerns
checks
...
```

を導いてはならない。

特にX-TEA sourceで確認したTask/Function利用関係だけでは、BP-06のような「Functionをprotected targetとして指すが、そのFunctionでBusiness Actionを実現しない」というroleを一般概念として証明しない。

したがってO1はX-TEA概念の移植ではなく、TEI current Authorization cross-view pressureとして限定する。

## 3. TEI current authorities

### 3.1 Semantic Authorization contract

`reference/workspace/semantic/security/authorization.tei`の`security.authorize`は:

```text
subject
action
resource
```

を入力とする汎用Authorization Capabilityである。

このcontract自身は:

```text
BP-06
order_entry.submit_order
Application Pattern protected-target role
```

を所有しない。

したがって`security.authorize` refだけからO1 targetをderiveできない。

### 3.2 Legacy Authorization policy

`reference/legacy-auth/authorization.json`にはcurrent Referenceとして:

```text
organization = SALES
role         = ORDER_EDITOR
action       = execute
resource     = order_entry.submit_order
effect       = allow
```

が存在する。

これはO1 targetと一致するためcross-view corroborationには使える。

しかしLegacy Authorization policyはAuthorization Role / Action / Resource / Effectのsecurity-side authorityであり、Business Procedure sourceではない。

## 4. Authority direction

Current pressureでは次を固定する。

```text
Business protected target
  if accepted/persisted in the future
  -> Business-side authoritative meaning

Authorization policy
  -> security-side authoritative meaning
  -> may corroborate or contradict Business target
  -> must not author/backfill Business target
```

禁止する方向:

```text
Legacy Authorization policy
  -> infer BP-06 exists
  -> infer BP-06 protected target
  -> populate O1 automatically
```

許可する方向:

```text
explicit O1 target
  + Authorization policy
  -> cross-view consistency diagnostic
```

Policy changeはBusiness targetそのものを書き換えず、不一致をdiagnostic / review pressureとして露出させる。

## 5. Test-only Pressure（Authority Direction）

`tools/reference-business-procedure-check/authorization_target_authority_direction_pressure_test.go`を追加した。

### TestO1ProtectedTargetCanBeCorroboratedByLegacyPolicyWithoutMakingPolicyAuthoritative

1. O1 targetを`order_entry.submit_order`として明示する。
2. Legacy policyに`execute / order_entry.submit_order / allow`があることを確認する。
3. test memory上でpolicy resourceだけを別Patternへ変更する。
4. corroborationは失われるが、O1 targetがpolicy変更で書き換わらないことを確認する。

### TestLegacyAuthorizationPolicyCannotBackfillMissingO1ProtectedTarget

1. O1 targetを明示しないcandidateを作る。
2. Legacy policyには`order_entry.submit_order` ruleが存在することを確認する。
3. policyの存在だけではO1 targetが補完されないことを確認する。

## 6. 設計上の含意

### 固定する

```text
Authorization policy may corroborate Business target
Authorization policy must not author Business target
protected target != direct Function realization
Business actor != Authorization Role
```

### まだ固定しない

```text
production applicationPatternTargets field
generic ExplicitSemanticRef { Kind, Role, Target }
generic role enum
Business Procedure -> Authorization policy binding
automatic policy generation from Business Procedure
```

## 7. Production source判断

本Pressureは「O1 roleを表現できる」「security-side policyと独立authorityにできる」ことだけを確認する。

まだ次は確認していない。

```text
file-native Business source persists O1 target
decode / reload preserves O1 target
Shared Acceptance preserves O1 target
stale detection includes O1 target
second realistic corpus also requires protected-target role
```

したがってPrimary `business-procedure-reference-v0`へtarget fieldを追加しない。

O1をproduction accepted Business meaningとして残す場合のみ、authoritative source pressureへ進む。

## 8. E-015 conclusion

E-015の結論を次とする。

```text
X-TEA evidence
  supports Business Action / Function separation
  does not prove generic protected-target role taxonomy

TEI local evidence
  proves current BP-06 needs target-vs-realization distinction for cross-view review
  proves Legacy Authorization policy can corroborate but must not author that target

therefore
  keep O1 narrow and test-only
  do not infer target from security policy
  do not generalize role taxonomy yet
```

---

# Part 5 — D4 + O1 Minimum Contract Audit（D4単独の損失性監査、現在の中心結論）

## 1. 問い

Current candidate stackではBusiness Procedure Step bodyをD3からD4へ縮小し、protected Application targetをO1 test-only overlayとして別pressureにしてきた。

本監査では次を判定する。

```text
Is D4 alone lossless for current accepted Business meaning?

If not,
what is the narrowest addition that preserves the missing meaning without
re-expanding the Step into D3 or introducing a generic ref-role model?
```

## 2. Current accepted Business meaning

Primary ReferenceのBP-06は:

```text
businessAction:
  determine whether the authenticated subject may execute the submit application action

realization:
  security-boundary

capabilities:
  security.authorize
```

をauthorしている。

したがってBusiness meaningとして少なくとも:

```text
there is a protected Application action
```

が存在する。

Cross-view pressureではそのexact targetを:

```text
order_entry.submit_order
```

として解決してきた。

同じBP-06はそのFunctionでrealizeされるわけではない。

```text
supporting Semantic authority
  security.authorize

protected Application target
  order_entry.submit_order

direct Application Function realization
  none
```

この三者分離はcurrent Referenceで意味を持つ。

## 3. D4 alone

D4 Step bodyは:

```text
businessAction
applicationPatterns[]  // direct realization
validations[]
capabilities[]
```

だけを持つ。

BP-06では:

```text
businessAction = authorization Business wording
capabilities   = security.authorize
applicationPatterns = []
```

を保持できる。

しかしprotected target exact identityは保持できない。

Target-only change:

```text
order_entry.submit_order
  -> order_entry.order_record
```

が起きてもD4 bodyは不変である。

したがって、protected target exact traceをaccepted Business meaningとして保持するなら:

> **D4 alone is lossy.**

これはD4全体をrejectする意味ではない。D4 bodyの縮小判断は維持し、missing role relationだけを別axisで補う。

## 4. D3へ戻っても解決しない

D3 / earlier thin StepにはD4より広い:

```text
realization
Data[]
Relations[]
```

等がある。

しかしD3の`applicationPatterns[]`もdirect Application Function realizationとして扱っている。

したがってBP-06へ:

```text
applicationPatterns:
  order_entry.submit_order
```

を入れると:

```text
protected target
  -> direct realization
```

へ意味を誤変換する。

つまりO1 pressureはD4縮小によって生じた欠損ではない。

```text
D3
  broader than D4
  AND
  still lacks target-vs-realization role separation
```

である。

このためO1解決のためにD3へ戻る理由はない。

## 5. Narrow minimum candidate

Current Evidenceからの最小比較候補を:

```text
D4 body
  businessAction
  applicationPatterns[]
  validations[]
  capabilities[]

+

optional narrow protected-target relation
  protectedApplicationPatternTargets[]
```

とする。

名称は最終production field名ではなくcomparison vocabularyである。

重要なのはshapeより意味境界:

```text
Application Pattern realization ref
  !=
protected Application Pattern target ref
```

である。

## 6. Why narrow field before generic role model

Current Evidenceが必要としているrole差は具体的に:

```text
realizes
vs
is protected target of authorization Business Action
```

である。

まだ次のtaxonomyは証明されていない。

```text
supports
concerns
checks
protects
observes
precedes
owns
...
```

したがって:

```text
GenericSemanticRef {
  Kind
  Role
  Target
}
```

を導入しない。

O1はApplication Patternという既存authority kindに対する一つのnarrow Business relationとしてPressureする。

## 7. Machine pressure

`d4_o1_minimum_losslessness_pressure_test.go`で次を追加した。

### M1 — D4 alone loses target-only meaning

同じD4 bodyに対し:

```text
protected target A = order_entry.submit_order
protected target B = order_entry.order_record
```

を比較するとD4 aloneは差を表現できない。

D4 + O1 narrow candidateは差を保持する。

### M2 — target does not become realization

BP-06で:

```text
ApplicationPatterns = []
Capabilities = [security.authorize]
ProtectedApplicationPatternTargets = [order_entry.submit_order]
```

を同時に保持する。

### M3 — no generic role taxonomy

Candidate Stepは:

```text
Body
ProtectedApplicationPatternTargets
```

だけで比較し、Role / Kind / Policy / Principal / Permission等を持たない。

### M4 — fail closed

Unknown Step / duplicate protected targetを黙ってnormalizeしない。

### M5 — no symmetry pressure

BP-06以外のStepへprotected target fieldを埋めることを要求しない。

Optional relationとして扱う。

## 8. Source placement comparison

### S0 — wording only

```text
BusinessAction text says "submit application action"
```

Pros:
- source最小

Cons:
- exact identity traceがない
- rename / impact / stale ref detectionができない
- wording解析へauthorityを移す危険

Current: **insufficient if exact cross-view trace remains accepted**。

### S1 — separate production sidecar

Current test-only overlayはmeaningの独立性をPressureする装置として有用。

しかし別sourceにするとProcedure-only source fingerprintはtarget-only changeを検出できない。

Production採用には:

```text
Procedure source + O1 source
  -> composite Shared Acceptance target
```

が必要になる。

これはone narrow relationのためにauthoring / acceptance modelを複雑化する。

Current: **useful pressure apparatus / weak production candidate**。

### S2 — same Business source, narrow optional field

Conceptual source:

```text
BP-06
  businessAction ...
  capability security.authorize
  protectedApplicationPattern order_entry.submit_order
```

Pros:
- protected target changeがProcedure source fingerprintへ自然に含まれる
- current Shared Acceptance stale ruleを再利用しやすい
- targetとBusiness Actionが同じreview unitにある
- sidecar locator / composite acceptanceが不要

Risks:
- current one strong caseだけでfieldをproduction固定する危険
- naming / cardinalityはまだ未確定

Current: **preferred minimum source candidate if O1 survives actual pressure**。

### S3 — generic role-qualified refs

Current: **defer / over-generalized**。

## 9. Candidate ranking update

Before:

```text
D3 broader candidate
D4 smaller candidate
O1 separate target overlay pressure
```

After this audit:

```text
D3
  not preferred as minimum
  broader fields do not solve O1 role loss

D4 alone
  minimal body
  but lossy for accepted protected-target exact trace

D4 + narrow O1
  current minimum semantic candidate

O1 sidecar
  pressure apparatus, not preferred production source shape

generic ref-role model
  deferred
```

D3はhistorical comparison Evidenceとして残すが、current minimum candidate競争の先頭から外す。

## 10. What is now proven vs not proven

### Proven as design pressure

```text
BP-06 has protected Application action meaning
protected target != direct realization
D4 alone cannot preserve exact target-only change
D3 breadth does not solve role distinction
narrow field can preserve current difference without generic role taxonomy
separate sidecar requires broader Acceptance machinery
```

### Not proven

```text
production field name
single vs multiple target cardinality
production Canonical BusinessProcedure kind
production World placement
second independent protected-target example
generic role taxonomy
actual runner green
```

## 11. Part 5 現在の決定

Minimum semantic candidate for current Reference becomes:

```text
D4 + narrow O1
```

rather than:

```text
D4 alone
```

This is a **candidate refinement**, not production adoption.

Production source change remains deferred until:

1. actual Repository tests are green;
2. Pre-ADR Gate is otherwise ready enough to justify source shape freezing;
3. narrow target field still proves smaller than keeping an external source-set Acceptance model;
4. exact target meaning remains necessary after another Business-design review / realistic pressure.

A second protected-target case is required before generalizing a role taxonomy, but not used to erase the already-observed BP-06 exact-trace loss.

## 12. Consequence for Pre-ADR

Pre-ADR comparison should now ask:

```text
Can D4 + narrow O1 + S4-A + H1/R3 preserve all current accepted Business meaning?
```

not:

```text
D3 vs D4 alone
```

D3 remains useful as subtraction history; D4 alone remains useful as proof of why O1 is a separate axis.

Neither remains the strongest standalone production candidate.

---

# Part 6 — O1 Cardinality Pressure（scalarかcollectionか）

## 1. 問い

`D4 + narrow O1`をcurrent minimum semantic candidateとした後、protected Application targetを最初からcollectionとして固定すべきかを確認する。

Current strong caseはBP-06だけであり、accepted targetは一つである。

```text
BP-06
  supporting capability = security.authorize
  protected Application Pattern = order_entry.submit_order
  direct Application Function = none
```

## 2. Current evidence

Current Referenceには、一つのBusiness Actionが複数のApplication Patternを同時にprotected targetとして持つaccepted caseはない。

またO1の目的はgeneric relation modelを作ることではなく、BP-06で失われるexact target traceを最小限に保存することである。

したがってcollection cardinalityを先回りして固定する根拠はない。

## 3. Minimum candidate

Current comparison vocabularyは:

```text
D4 Step
  Body
  protectedApplicationPattern?  // zero or one
```

とする。

ここでzeroは:

```text
no protected target authored for this Business Action
```

であり、

```text
there can never be a protected target
```

を意味しない。

oneはcurrent BP-06を表す。

## 4. Multiple target future case

将来realistic Business sourceで:

```text
one Business Action
  -> protects Application Pattern A
  -> protects Application Pattern B
```

がacceptedされた場合、current minimum projectionは黙って:

- 最初の一件だけ残す
- 重複除去してcollectionへ暗黙拡張する
- Business wordingへ押し戻す

ことをしない。

Fail closedしてO1 cardinalityをreopenする。

比較候補はその時点で:

```text
zero-or-many explicit targets
one Business Action split into narrower Business Actions
a named protected-target relation
another domain-specific boundary
```

をEvidenceから選ぶ。

## 5. Why singular before plural

### Singular

Pros:
- current Evidenceと一致
- source / reviewが最小
- generic relation modelへ近づかない
- one Business Action / one targetというcurrent意味が明瞭

Risk:
- future multi-target caseでschema reopenが必要

これは意図的なfail-closed costとして受け入れる。

### Plural from the start

Pros:
- future multi-targetを受けられる

Risks:
- current未証明cardinalityを固定
- collection ordering / duplicate semantics / empty-vs-absent等を先回り
- generic target relationへの滑り込みを招く

Current: **not preferred**。

## 6. Test pressure

`d4_o1_minimum_losslessness_pressure_test.go`のminimum candidateをscalarへ縮小した。

Pressure:

1. BP-06のsingle exact targetを保持する。
2. target-only changeをmeaning差として保持する。
3. targetをdirect realizationへ変換しない。
4. unrelated Stepにsymmetry目的のfieldを要求しない。
5. unknown Stepはfail closedする。
6. multiple target inputはcurrent Evidence未証明としてfail closedする。

## 7. Test-only sidecarとの違い

`order-entry-protected-target-overlay.json`はO1 meaning / reload / AcceptanceをPressureするための実験artifactであり、現在は`applicationPatternTargets[]`というcollection shapeを使う。

これはproduction cardinality decisionではない。

```text
pressure apparatus shape
  != production minimum source shape
```

とする。

Sidecarは複数source Acceptanceのcostを露出させるため残し、minimum production comparisonはscalar narrow fieldで行う。

## 8. Part 6 現在の決定

```text
O1 semantic necessity
  current BP-06でconfirmed as pressure

minimum cardinality
  zero-or-one

minimum comparison field
  protectedApplicationPattern?

multiple target
  fail closed / reopen

production field adoption
  deferred

generic role relation
  deferred
```

この判断はPart 5（旧`BUSINESS_PROCEDURE_D4_O1_MINIMUM_CONTRACT_AUDIT_2026-09.md`）をさらに縮小するcardinality refinementである。

---

# Part 7 — O1 File-Native / Shared Acceptance Pressure（representation narrowing、最新の結論）

## 1. 目的

Current BP-06には、既存D4 bodyだけではlosslessに保持できないaccepted cross-view pressureがある。

```text
Business Action
  determine whether the authenticated subject may execute the submit application action

supporting Semantic authority
  security.authorize

protected Application target
  order_entry.submit_order

direct Application Function realization
  none
```

`applicationPatterns[]`はdirect realizationを意味するため、protected targetをそこへ入れると意味を誤変換する。

これまでO1 test-only candidateで:

```text
applicationPatternTargets[]
```

を別roleとして保持できることは確認した。

本Pressureでは次を追加確認する。

1. protected targetをfile-native sourceとしてdecode / reloadできるか。
2. stale Procedure / Step / targetをfail closedできるか。
3. protected-target-only changeがShared Acceptance approvalをinvalidにできるか。
4. sidecar source方式の追加コストをproduction representation判断へ反映できるか。

## 2. File-native test-only overlay

追加したReference pressure artifact:

```text
reference/business-system-design/order-entry-protected-target-overlay.json
```

内容は概念上:

```text
schemaVersion = business-procedure-pattern-target-overlay-v0
artifactRole  = reference-pressure-overlay
canonical     = false
procedure     = order-entry.order-handling

BP-06
  applicationPatternTargets = [order_entry.submit_order]
```

重要:

- primary `business-procedure-reference-v0` schemaは変更していない。
- production Canonical sourceではない。
- generic `role: protects`等は受け付けない。
- O1のsource mechanicsを検査するためだけのsidecarである。

## 3. File-native pressure result

`application_pattern_target_authoring_pressure_test.go`で次をPressureする。

### F1 — round-trip

JSON decode / encode / reload後も:

```text
BP-06 -> order_entry.submit_order
```

をlosslessに保持する。

### F2 — target != realization

File-native targetを読み込んでも:

```text
BP-06 ApplicationPatterns = []
BP-06 Capabilities        = [security.authorize]
BP-06 protected targets   = [order_entry.submit_order]
```

を維持し、BP-06をApplication Function realized stepへ変換しない。

### F3 — stale Procedure / Step fail closed

OverlayのProcedure identityまたはStep local addressがprimary Business sourceと一致しなければrejectする。

### F4 — duplicate source fail closed

同一Stepへ複数overlay entryを持たせて曖昧なtarget authorityを作らない。

### F5 — generic role vocabulary fail closed

Unknown field:

```text
role: protects
```

を受理しない。

O1を理由にgeneric role taxonomyをsidecarへ忍び込ませない。

## 4. Existing Procedure-only Acceptance blind spot

Current Business Procedure authoring pressureでは、Procedure source file自身のcontent fingerprintをAcceptance targetとしている。

`sourceindex.NewArtifact`のfingerprintはsource pathを含まず、content bytesだけのSHA-256である。

この性質は:

```text
pure file move
  -> same fingerprint
```

を可能にするため望ましい。

しかしO1を別sidecarへ置くと問題がある。

```text
Procedure source unchanged
O1 overlay target changed
```

の場合:

```text
Procedure fingerprint = unchanged
Business protected-target meaning = changed
```

となる。

`application_pattern_target_procedure_only_acceptance_negative_test.go`では、overlay targetを:

```text
order_entry.submit_order
  -> order_entry.order_record
```

へ変更した後も、current Procedure-only `acceptProcedureSource`が旧Procedure fingerprintで成功することを意図的に確認する。

これはcurrent Procedure acceptanceのbugではない。

> separate semantic sourceを追加するなら、Procedure-only fingerprintではそのsourceを保護できない

というsidecar designへのnegative Evidenceである。

## 5. Composite Shared Acceptance pressure

Sidecar方式を成立させる最小条件として、test-only composite fingerprintをPressureする。

```text
business-procedure-source=<procedure content fingerprint>
protected-target-overlay=<overlay content fingerprint>
  -> sourceindex.Fingerprint(...)
```

Pathは含めず、semantic source roleとcontent fingerprintだけを合成する。

### A1 — target-only change invalidates approval

Overlay targetだけを変更すると:

```text
Procedure fingerprint = unchanged
Overlay fingerprint   = changed
Composite fingerprint = changed
```

となり、旧composite fingerprintによるacceptanceをrejectする。

### A2 — pure overlay file move is no-op

Overlay contentを変更せずfile pathだけを移動すると:

```text
Overlay fingerprint   = unchanged
Composite fingerprint = unchanged
```

となる。

これはexisting Procedure source move semanticsと一致する。

### A3 — fresh fingerprint is not enough

Composite fingerprintを更新しても、target identityがnormalized Worldでmissingならacceptしない。

```text
fresh source revision
  != semantically valid source
```

を維持する。

### A4 — source roles are explicit

Procedure source fingerprintとoverlay source fingerprintは単なるunordered setとして混ぜず、役割付きで合成する。

```text
Business Procedure source
Protected-target source
```

の区別を失わない。

## 6. Representation option audit after file-native pressure

### O1-R0 — wording only / structured targetを持たない

Current Business wordingにはsubmit actionへの言及があるため、人間には意味を読める。

しかし:

- exact target resolution不可
- target rename / stale detection不可
- Authorization policyとのdeterministic corroboration不可
- target-only semantic diffを構造化できない

Current: **insufficient if protected target is accepted review meaning**。

### O1-R1 — production sidecar overlay

Pros:

- primary Procedure schemaを膨らませない
- role-specific sourceを独立管理できる

Cons:

- Procedure-only acceptanceでは意味変更を見逃す
- composite source revision / source-set identityが必要
- source move / stale / duplicate / registry管理が二source化
- Business Procedureの一つのaccepted meaningが複数fileへ分散する
- one-maintainer operational burdenが増える

Current: **test apparatusとして有用 / production candidateとして弱い**。

### O1-R2 — same Business source内のnarrow field

Conceptual pressure only:

```text
step
  businessAction
  applicationPatterns[]        // direct realization
  applicationPatternTargets[]  // protected/concern target, narrow role
  validations[]
  capabilities[]
```

Pros:

- one Business source revisionでapprovalをbindできる
- Procedure/Step renameと同じsource transactionで変更できる
- O1 meaningをD4 bodyと同じauthoring lifecycleへ載せられる
- generic role taxonomyを導入しなくてよい

Risks:

- one exact BP-06 caseからproduction fieldを増やすことになる
- field名/role semanticsを将来一般化しにくい可能性
- Application Pattern以外のfuture protected targetを別modelで扱う必要が出る可能性

Current: **preferred narrow production candidate if O1 is retained; not adopted**。

### O1-R3 — generic role-qualified semantic reference

```text
ExplicitSemanticRef {
  Kind
  Role
  Target
}
```

Current: **defer / effectively reject for current contract**。

O1一例から`protects / concerns / supports / checks / realizes...` taxonomyを作るEvidenceはない。

## 7. Source-of-truth judgment

Sidecar pressureから得られる重要な結論は:

> fileを分ければCore schemaを小さく保てる、とは限らない。

Semantic meaningが一体なら、別file化は:

```text
source set
composite fingerprint
stale coordination
move coordination
approval coordination
```

という別の複雑さを生む。

P3 `no implicit multiple SOT`、RQ-017 dual-maintenance minimization、one-maintainer maintainabilityの観点では、O1をaccepted Business meaningとしてproductionへ残す場合、**同じBusiness source内のnarrow explicit field**の方が現在は自然である。

ただしこれはproduction schema adoption decisionではない。

## 8. What is now proven vs still open

### Stronger Evidence now available

```text
protected target != direct realization
protected target exact identity is useful for deterministic cross-view review
file-native narrow target can round-trip
stale Procedure/Step can fail closed
separate sidecar needs composite Acceptance fingerprint
target-only change must invalidate approval
pure source move should remain semantic no-op
```

### Still open

```text
actual gofmt / go vet / go test green
Shared Acceptance production API shape
whether O1 should survive into the minimum production contract
whether a second realistic domain repeats target-vs-realization role
future non-Application protected target needs
generic role taxonomy
```

## 9. Pre-ADR implication

Pre-ADR checklist item:

```text
O1をproduction accepted meaningとして残す場合、
authoritative source / decode / reload / Shared Acceptance境界を閉じる
```

について、test-only構造上はかなり前進した。

Current classification:

```text
authoritative production source
  not yet

decode / reload pressure
  implemented test-only

stale Procedure / Step
  implemented test-only

Shared Acceptance target-only invalidation
  implemented test-only composite pressure

production representation
  sidecar disfavored
  narrow inline field preferred candidate
  generic role model deferred
```

Actual runner greenとPre-ADR全体の整合が取れるまでReadyへ変更しない。

## 10. Part 7（全体の現時点）決定

今はprimary fixture schemaへfieldを追加しない。

```text
O1 meaning pressure
  -> retain

production sidecar
  -> disfavor

production narrow inline field
  -> preferred candidate, not adopted

generic role-qualified ref
  -> defer

Procedure-only Acceptance for separate O1 source
  -> insufficient by construction

composite Acceptance
  -> test-only proof of requirement if sidecar were used
```

この結果により、O1の未決は「target roleが必要か」から、より狭い:

> accepted O1 meaningをminimum production Business sourceへ残す価値が、一つのnarrow field追加に見合うか

へ絞られた。
