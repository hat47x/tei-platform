# Business Procedure Canonical Contract Pre-ADR Audit — 2026-09

Status: **Lane C / sole Pre-ADR readiness authority / Not Ready / no production Core adoption**

## 1. この文書の責務

この文書は、Business Procedure / Handoffのproduction contractについて、**ADRを起こしてよいかだけを判定する正本**です。

詳細な実験履歴・Pressure・Evidenceは次へ委譲します。

- `plan/issue/000042-business-procedure-cross-view-reference.md` — A-series Evidence / Pressure inventory
- `plan/BUSINESS_MODEL_CURRENT_MINIMUM_CANDIDATE_2026-09.md` — current minimum candidateの簡潔な比較基線
- 個別`BUSINESS_PROCEDURE_*_PRESSURE_*` / `BUSINESS_ACTION_*_PRESSURE_*`文書 — 各論のEvidence
- `plan/XTEA_DESIGN_EVIDENCE_REGISTER_2026-09.md`およびE-015 / E-016 — X-TEA fallback provenance

この文書へ過去Pressureの全文を重複転記しません。

## 2. Current minimum semantic candidate

現時点の最小候補は次です。

```text
BusinessModel
  Procedures[]
    Procedure
      id
      purpose
      trigger { epistemicStatus, meaning }
      actor   { epistemicStatus, businessLabel }
      Steps[]
        localKey                         // S4-A: Procedure-scoped address
        Body                             // D4
          businessAction
          applicationPatterns[]          // direct realization only
          validations[]
          capabilities[]
        protectedApplicationPattern?     // narrow O1, zero-or-one

  Handoffs[]                              // H1/R3, set-level relation
    from { procedure, step }
    to   { procedure, step }
    condition
```

解釈:

```text
D4 body
+ singular narrow O1
+ S4-A Procedure-scoped Step address
+ H1/R3 thin set-level Handoff
```

### D3の扱い

D3はsubtraction / historical comparison Evidenceとして残しますが、current minimum candidateではありません。

- direct Data / Relation slotsはcurrent Referenceでminimum必要性が未証明。
- Data / Relation review traceはexisting Pattern / Validation / Capability refsからかなりderiveできる。
- D3へ戻してもO1 protected-target roleは復元できない。

### D4単独の扱い

D4単独もcurrent accepted meaningに対してlosslessではありません。

BP-06はBusiness wordingとしてsubmit application actionを保護対象として扱う意味を持つ一方、D4 direct-realization refsだけではexact target `order_entry.submit_order`を保持できません。

したがってcurrent minimum comparisonではnarrow O1を含めます。

## 3. Current authoring candidates

### 3.1 Procedure / O1

Current strongest authoring candidateは同じBusiness Procedure source内のsingular optional fieldです。

```text
protectedApplicationPattern?
```

Test-only inline candidateで次をPressure済みです。

- strict decode
- round-trip lossless
- zero-or-one cardinality
- multi-targetをfail closed
- generic role fieldをfail closed
- previous sidecar pressureと同じD4+O1 normalized meaningへ変換

したがって「narrow O1をfile-nativeに表現できるか」は大きく前進しました。

Production採用はまだ行いません。残る論点はsource migration / versioning / Shared Acceptanceへの正式接続です。

Production sidecarは、単一Step-local relationに対してcomposite source fingerprint / Acceptance machineryを要求するため第一候補から下げます。

### 3.2 Handoff

Current strongest H1/R3 authoring candidateはProcedure artifact locatorを持たないstandalone set-level relation sourceです。

```text
Handoffs[]
  from { procedure, step }
  to   { procedure, step }
  condition
```

Test-only standalone candidateで次をPressure済みです。

- current Reference Setと同じnormalized Handoff meaning
- Procedure artifact locatorを保持しない
- stale Procedure / Step endpointをfail closed
- `retry`等のWorkflow fieldをstrict decodeで拒否

Reference Set manifest内Handoffはfixture assemblyには便利ですが、source locator metadataとBusiness meaningを混在させるためproduction authoring候補としては弱いです。

Production採用はまだ行いません。残る論点はmigration / source discovery / Shared Acceptanceへの正式接続と、second realistic Handoffで`from / to / condition`だけで十分かの確認です。

## 4. O1 authority boundary

Current strong O1 case:

```text
BP-06
  supporting semantic authority -> security.authorize
  protected Application target  -> order_entry.submit_order
  direct Function realization   -> none
```

採らない表現:

- wordingだけにexact identityを埋め込む
- `applicationPatterns[]`へtargetを混ぜてrealization扱いする
- generic `ExplicitSemanticRef { Kind, Role, Target }`
- generic role taxonomy
- Authorization policyからBusiness targetをbackfillする

Current minimum cardinalityはzero-or-oneです。複数targetが必要なrealistic caseが現れた場合はfail closedしてreopenします。

Authorization policyはcross-view corroborationにだけ使い、Business targetのauthor / backfill sourceにはしません。

## 5. H1/R3 authority boundary

Handoff condition / endpointはProcedure bodyだけから復元できず、accepted Business meaningとして必要です。

Authority direction:

```text
explicit accepted Handoff source
  -> normalized H1/R3 relation

Step order / adjacency
  -X-> Handoff

physical Observation
  -X-> Handoff

Application Function sequence
  -X-> Handoff
```

Handoffは複数Procedureを結ぶためProcedure bodyへinlineしません。

Current H1/R3は**handoff-only**です。Business Flow / DFD、generic Business Edge、Workflow runtimeへ拡張しません。

Production top-level `BusinessHandoff` kindの採用はまだ決定しません。

## 6. Step identity / address boundary

Business Stepのbody meaningと外部relationからのaddressabilityを分離します。

```text
Step body meaning
  != Step address metadata
```

Current candidateはglobal Canonical Step IDではなく、Procedure-scoped `localKey`です。

- stale endpointはfail closed
- Procedure + Step addressの協調renameは明示change
- file pathはStep identityではない
- Source IndexだけにStep addressを置いてCanonical validationをSource infrastructureへ依存させない

## 7. Placement decision pressure

Business Modelをfirst-classな設計意味として扱うことと、`ir.World`へ埋め込むことを分離します。

Current strongest placement candidate:

```text
Business authoring sources
  -> normalized Business model

TEI workspace sources
  -> ir.World

normalized Business model
  <-> exact cross-view validation <-> ir.World
```

Current Evidenceでは、Business Modelを`ir.World.Business`へ入れなくても:

- exact Application / Validation / Capability / Pattern ref validation
- protected target validation
- Data / CRUD / ownership / Namespace derived lens
- Handoff referential integrity
- Business-only semantic revision

を保持できます。

Business wording change、protected-target-only change、Handoff condition-only changeはBusiness semantic revisionを変えながら`ir.World`を変更せずcross-view validateできます。Application targetがWorldから消えればseparate planeでもfail closedします。

したがって現時点では:

```text
Business Model first-class meaning     -> yes, design pressureとして
separate normalized Business plane     -> strongest placement candidate
ir.World.Business                      -> not required by current Evidence
production internal/business package   -> deferred
```

### `ir.World` integration reopen conditions

次の具体的必要性が出た場合だけ再検討します。

1. compiler / DriverがBusiness Procedure meaningを直接必要とする。
2. release / compatibility / deployment判定がBusiness modelを同一compiled unitとして必要とする。
3. Application/Data/Businessを一つのatomic accepted revisionにしないと整合性が保てない実例が出る。
4. separate Business planeではcross-view Conformanceをlosslessに実装できない。
5. Editor / Target projection / reloadでseparate planeが実際に二重SOTを生む。
6. Business semantic revisionとruntime/application revisionを分離するとactual operational inconsistencyが発生する。

## 8. Derived views — authored meaningへ戻さない

### Business Action × Data / Relation

Pattern / Validation / Capability refsからexact identityでderiveします。

```text
no current trace != no Data dependency
trace exists      != complete dependency set
```

### Business Action × CRUD

Current structured meaningから安全に確定できる主要cellはRだけです。

```text
BP-00 -> R sales.order
BP-02 -> R customer.customer
BP-03 -> R sales.order_line
```

禁止:

- Data traceだけからCRUD letterを付ける
- `record`からR/Uを推測する
- `command`からC/U/Dを推測する
- Database Contract `Writes[]`をBusiness Action Uへ帰属する
- Capability input/outputをmutation contractへ読み替える
- Lifecycle transitionをCRUD Uへ短絡する
- Business wordingの`edit/update/create/delete`をoperation authorityへする

Current TEI-original Reference corpusにはqualifyingなaccepted C / D / non-state U exampleがありません。

C/U/D gapはBusiness Procedure schemaより、Function / realization / mutation-authority側のpressureとして扱います。

X-TEA E-016ではCRUD matrixが`TaskFunctionIO`のexplicit `OpC / OpR / OpU / OpD`からderiveされていたことを確認しています。将来C/U/D attributionが必要になった場合もBusiness Procedure CRUD annotationを第一候補にしません。

### Ownership / Namespace lens

Evidence-backed CRUD rowへDatabase ownership/accessを重ねるderived lensと、Data owner Semantic Namespace lensでcurrent Referenceではかなりのreview情報を得られます。

ただし:

```text
Namespace != SubjectArea
Database ownership/access lens != Subsystem
```

です。

Canonical Subsystem / SubjectAreaは、direct/effective CUD、explicit Function call、published cross-boundary Action encapsulation等でcurrent lensが具体的に破綻した場合だけreopenします。

## 9. Command / Observation / physical-world boundary

- Command acceptedだけでphysical completionを成立させない
- transport/read成功だけで`observed=true`へ丸めない
- runtime `ReadIdentity` / Observation Evidence natureをBusiness Canonical identityへ昇格しない
- physical Observationからresponsibility Handoffを生成しない

Manufacturing / inventory second-corpus outlineはこの非対称性を別domainでも補強しましたが、O1 generic role taxonomyやsynthetic Data ownershipを追加する根拠にはしていません。

## 10. Coreへ今入れないもの

- production `BusinessProcedure` / `BusinessHandoff` kind
- `ir.World.Business`
- production `internal/business` package
- Workflow state / retry / timer / queue / scheduler / saga
- full Business Flow / DFD
- generic Core / Business Edge
- generic `semanticRefs[]`
- generic explicit-ref Role enum
- generic Uncertainty framework
- `realization` enum
- global Canonical Step ID
- Observation専用Canonical identity / physical-world state engine
- Business Procedure CRUD field
- Application Pattern generic CRUD field
- Semantic Capability generic Effect
- Database Contract C/U/D enum
- mutation binding / MutationIntent / Function-Data-I/O Canonical kind
- synthetic second-corpus Data / Database / Pattern ownership

## 11. ADR readiness gates

### Gate A — actual execution Evidence

Actual Repository environmentで少なくとも:

```text
gofmt
go vet
go test
reference checker
```

が実行・greenであること。

過去GitHub-hosted Actionsの`runner_id=0 / steps=[]`やworkflow未起動はPASS/FAIL Evidenceに数えません。

### Gate B — minimum Business meaning

Actual testsで:

- Business-only wording / uncertainty changeをsemantic diffとして観測
- pure file move = meaning no-op
- manual / external Business Actionをfake Functionなしで保持
- one Procedure -> multiple Applicationsをsingle ownerなしで保持
- Pattern-owned meaning非重複
- actor uncertainty != Authorization identity
- D4 bodyのfuture direct Data / Relation ref fail-closed
- ref authority kindをgeneric bagへ失わない

### Gate C — address / Handoff

- Procedure-scoped Step local addressで必要addressabilityを保持
- global Step IDを不要と説明
- stale endpointをfail closed
- H1 semantic preservationをactual testsで確認
- explicit Handoffだけがrelation authority
- adjacency / Observation / Function sequenceからHandoffを生成しない
- Handoff-only boundaryを保ちBusiness Flowへ膨張しない
- D4 + S4-A + H1/R3 compositionをactual testsで確認
- standalone Handoff source candidateがReference Setと同じnormalized meaningを持つことをactual testsで確認
- production採用時のsource discovery / migrationを説明

### Gate D — O1

- protected target != realizationをactual testsで確認
- D4単独ではtarget-only changeを失うことを確認
- singular narrow O1でcurrent meaningを保持できることを確認
- inline O1 authoring candidateがstrict decode / round-trip / zero-or-one / sidecar-equivalent normalizationを満たすことをactual testsで確認
- target-only changeがBusiness semantic revisionを変えることを確認
- Authorization policyはcorroboration-onlyで、author/backfill sourceにしない
- production採用時のsource migration / versioningを説明
- multi-targetが必要なrealistic caseがない限り配列を先取りしない
- generic role taxonomyが複数realistic casesで本当に必要かを別途判断

### Gate E — placement / Shared Acceptance

- `D4 + narrow O1 + S4-A + H1/R3` normalized modelでcross-view validationがSource Index非依存に成立
- Business-only changeで`ir.World`を変更しなくてもsemantic revisionを観測可能
- pure artifact locator moveでnormalized Business meaningが不変
- source orderingだけでmeaningが変わらない
- protected-target change / Handoff condition changeでmeaningが変わる
- inline Procedure source(s) + standalone Handoff sourceを一つのBusiness semantic revision / Shared Acceptanceへ束ねる方法を説明
- separate Business planeでlossless cross-view Conformanceが成立することをactual testsで確認
- `ir.World` embeddingが必要になる具体的Evidenceがない限り配置Decisionを分離する

### Gate F — derived projections / mutation gap

- Business Action × Data / Relation projectionをactual testsで確認
- no-traceを`no dependency`へ丸めない
- projectionをOwnership / Access / Canonical authorityへ昇格しない
- partial CRUD R rowsをexact authorityからderive
- Data trace != CRUD operation traceを保持
- C/U/D不足をBusiness Procedure fieldで埋めない
- realistic C/U/D caseが現れるまでgeneric mutation/effect contractを作らない
- future operation modelではdirect / effective effect distinctionを失わない

### Gate G — adjacent domain pressure

- Gate 15a multi-Application structural pressureをactual testsで確認
- Gate 15b social-system realismをactual testsで確認
- Gate 15cは具体的contract不足が出た時だけ起動
- physical-world Command / Observation separationをactual testsで確認
- second-corpus Data authorityはexact TEI identitiesなしにsynthetic化しない
- Subsystem / SubjectAreaはderived lensが不足するconcrete caseまでreopenしない

## 12. Current readiness decision

```text
Current minimum semantic candidate
  = D4 + singular narrow O1 + S4-A + H1/R3

Current strongest Procedure authoring candidate
  = same-source singular protectedApplicationPattern?

Current strongest Handoff authoring candidate
  = standalone set-level Handoff source

Current strongest placement candidate
  = separate normalized Business semantic plane

D3
  = historical/subtraction Evidence, not preferred minimum

D4 alone
  = lossy if protected-target meaning is retained

ir.World.Business
  = not required by current Evidence

Production ADR
  = Not Ready
```

Not Readyの主要理由:

1. actual Repository runner greenが未取得。
2. inline O1 / standalone Handoffのproduction adoption・migration strategy未確定。
3. Procedure source(s) + Handoff sourceを束ねるproduction Shared Acceptance / Business semantic revision mechanism未確定。
4. second realistic Handoff caseでthin relation sufficiencyをまだ確認していない。
5. qualifying TEI-original C/U/D caseがなくmutation/effect boundaryは未決。
6. separate Business planeが不足するEvidenceはなく、`ir.World` integration判断はまだ不要。

したがって**Pre-ADR Status = Not Ready**を維持します。
