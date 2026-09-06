# Business Procedure Canonical Contract Pre-ADR Audit — 2026-09

Status: **Lane C / Pre-ADR audit / no Core adoption**

## 1. 目的

Business ProcedureのReference pressureが増えた現在、production Canonical contractのADRを起こす前に、次を明確に分ける。

1. accepted Business meaningとしてCoreに残る必要があるもの
2. 既存Canonical meaningからderivedできるもの
3. authoring / source / review infrastructureへ残すもの
4. diagnostic / Projectionだけに置くもの
5. Runtime / Authorization等、Business Procedure Coreへ入れてはいけないもの
6. Reference Evidence不足のためまだ決めないもの

本監査はADRではない。

Production `ir.World`変更、Canonical schema採用、Workflow Engine導入を決定しない。

## 2. 現在までの反証

### 2.1 identity-onlyは最小十分ではない

Procedure IDだけをCanonical subjectにしても、同じIDのままpurpose / actor / trigger / manual Business Actionが変更された場合、Business-only accepted semantic changeをCanonical diffから観測できない。

したがって、Canonical contractが必要ならidentityだけでなくBusiness residual meaningが必要な可能性が高い。

### 2.2 full Workflow modelも不要

Referenceが必要としているのは業務上の意味であり、runtime orchestrationではない。

retry / timer / queue / scheduler / saga / runtime state等を入れなくても、current ReferenceのBusiness Procedure / manual work / handoffは表現できる。

### 2.3 単一Application ownerは不要

Business Procedureは複数Application Pattern、manual work、external workを跨ぎ得る。

したがってProcedureを一つのApplicationの子へ固定することは、current pressureより強すぎる。

### 2.4 uncertaintyはmeaningの一部だが、汎用frameworkは不要

trigger / actorの`candidate`を`accepted`へ丸めるとP11に反する。

一方、current Referenceだけからcertainty latticeやapproval state machineを設計する根拠はない。

## 3. Candidate Core meaning

現時点で最も小さいproduction候補は次である。

```text
BusinessProcedure
  id
  purpose
  trigger
    epistemicStatus
    meaning
  actor
    epistemicStatus
    businessLabel
  orderedSteps

BusinessProcedureStep
  id
  businessAction
  explicitSemanticRefs
```

ここで`explicitSemanticRefs`は、Business側が明示的に参照する既存Canonical identityを指す。

例:

- Application Pattern
- Validation
- Data / Relation（Patternから導けないBusiness上のexplicit refが本当に必要な場合だけ）
- Capability（同上）

### 3.1 なぜpurposeを残すか

purposeはApplication Patternから導出できず、Business Procedure全体の意味を識別・レビューするために必要である。

purpose wordingのaccepted changeはBusiness semantic changeである。

### 3.2 なぜtrigger / actorのstatusを残すか

current Referenceはtrigger / actorをcandidateとして扱っている。

statusをProjection decorationとして捨てると、candidate / unresolvedという意味差をCanonical側が失う。

ただしstatus vocabulary自体はまだ固定しない。

### 3.3 なぜordered stepを残すか

Business Procedureでは行為の順序そのものが意味を持つ。

set-likeなsemantic refsと異なり、step順をnormalizationでsortしてはならない。

### 3.4 なぜBusiness Actionを残すか

manual / external / judgment等、Application Functionから導けないBusiness actionが存在する。

Application Patternだけを残すとBP-04 / BP-09のようなBusiness meaningを失う。

## 4. Candidate Canonical relation — handoff

handoffはProcedure bodyへruntime control flowとして埋め込むより、独立Business semantic relationとして扱う方が現在のpressureに合う。

```text
BusinessHandoff
  from
    procedure
    step
  to
    procedure
    step
  condition
```

このrelationをproduction Coreへ入れるかはまだ決定しない。

採用条件は、Reference-local relationではcross-view diff / impact / reuseを安全に再構成できないことがEvidenceで示されることである。

## 5. Derivedであるべきもの

### 5.1 Pattern-owned meaning

Application Pattern refから解決できる次をProcedureへコピーしない。

```text
Data
Relation
Query
Capability
Datasource
```

これらをProcedureへ複製すると二重Authorityになる。

### 5.2 Application association

Procedureに必須`application` ownerを持たせない。

どのApplicationと関係するかはqualified Application Pattern refsからderivedできる。

一つのProcedureが複数Applicationへ関係してよい。

### 5.3 Function / Data projection rows

三視点レビュー用のtable / matrix / rowはProjectionであり、Canonical subjectではない。

### 5.4 cross-view diagnostic

`semantic-error` / `architecture-policy` / review diagnostic等のchecker出力は、Canonical Business Procedureそのものではない。

Diagnosticのseverity taxonomyもProcedure schemaへ埋め込まない。

## 6. Authoring / source infrastructureへ残すもの

次はaccepted Business meaningではなくSource / Review / Acceptance infrastructureである。

- file path / artifact locator
- source range
- source fingerprint
- schemaVersion
- artifactRole
- Git revision
- ETag
- approval snapshot metadata
- ManagedFiles entry

Canonical identityとSource locatorを分離する。

pure file moveでBusiness meaningが変化してはならない。

## 7. Coreへ入れないもの

### 7.1 Runtime Workflow semantics

- branch execution state
- retry
- timer
- queue
- scheduler
- saga
- lease
- runtime workflow instance
- execution ownership
- transport

Business Procedureを表現できることを理由にWorkflow Engineを導入しない。

### 7.2 Authorization semantics

- Authorization Role
- Principal / Subject
- Permission
- Credential
- Authority policy

Business actorは業務上のjob / responsibility labelであり、security identityではない。

### 7.3 Provider / deployment Evidence

- Database readiness
- Provider binding details
- deployment status
- Gate B Evidence
- Target rendering state

上流Business meaningを実装・Evidence状態へ結合しない。

### 7.4 UI layout

Business Procedure Projectionを表・グラフ・UI・文章のどれで表示するかはTarget / Editor concernであり、Canonical meaningではない。

## 8. `realization`の再監査

current fixture / test-only probeには次のような`realization` labelがある。

```text
application-use
manual-unresolved
validation-support
security-boundary
observation-boundary
manual-external-unresolved
```

これは現段階では**便利なBusiness review label**であるが、production Core enumとして採用する根拠はまだ不足している。

理由:

1. checkerはFunction有無を`realization`文字列から推論せず、Application Pattern refs等の実体から判定している。
2. `manual-unresolved`はmanual性とepistemic uncertaintyを一語へ畳んでおり、将来の直交軸を先に固定する危険がある。
3. `security-boundary`はBusiness ProcedureがAuthorization contractを所有することを意味しない。
4. `observation-boundary`もCommand / Observation分離を示すreview上の意味であり、runtime implementation kindとは限らない。

したがってpre-ADRでは次とする。

```text
realization
  retain in Reference authoring / review pressure
  do not make authoritative Core enum yet
```

Production候補では、Business Action + explicit semantic refsだけで意味を保持できるかを先に検証する。

もしmanual / external / system-use等の区別がBusiness-only semantic diffに不可欠で、refsから安全に導出できないことが現実的Referenceで示された場合だけ、独立したBusiness realization kindを再検討する。

## 9. Step identityの再監査

current Referenceでは`BP-04`等のStep keyがhandoff / review endpointに使われる。

しかしproductionでStep IDをdurable Canonical identityにするかは未決定である。

必要性を示すpressure:

- handoff endpoint
- review comment target
- cross-revision diff continuity
- external trace / link

不要性を示すpressure:

- Procedure内部のordered positionだけで安全に再構成できる
- Step renameがBusiness identity changeではなくauthoring-only changeとして扱える

ADR前に、Step IDが単なるReference-local keyなのかdurable semantic identityなのかを分けるEvidenceが必要である。

## 10. Handoff conditionの再監査

current handoff `condition`は自然言語Business meaningとしてpressure testしている。

まだ次を固定しない。

- Boolean Expression化
- event model
- branch DSL
- state machine condition
- executable predicate

自然言語conditionのaccepted changeをsemantic diffとして保持できることを先に確認する。

## 11. Uncertainty vocabularyの再監査

current test-only modelはexplicit statusをlosslessに保持する。

これにより必要条件は確認できる。

一方、次はまだ決めない。

```text
allowed status enum
status transition rules
missing status severity
unknown status severity
approvalとの関係
trigger / actor以外への一般化
```

P11を守る最小条件は「明示された不確定性を勝手に既知へ丸めないこと」である。

## 12. Watanabe pressureとの整合

本候補は、Data / Function-Interaction / Businessを同一objectへ潰さない。

- Dataは既存Canonical semantic model
- Function-InteractionはApplication Pattern / View / Action等
- Business Procedureはpurpose / actor / trigger / manual action等

として独立した視点を保ち、explicit identityでcross-checkする。

これによりBusiness Modelを業務マニュアル / social systemとして扱う余地を残しつつ、Data-firstをData-onlyへしない。

またProcedureを実行runtimeへ直結させず、UI / simulator / Reference checker等を通じて設計を批判・修正できる形を優先する。

## 13. ADR作成前の採用Gate

次を満たすまでproduction Canonical contract ADRを作らない。

1. actual Repository environmentでLane Cのtest-only pressureがgreen
2. Business-only wording / uncertainty changeをsemantic diffとして観測できる
3. pure file moveがmeaning不変
4. logical identity renameがfail closed
5. manual / external Business Actionをfake Functionなしで保持
6. Pattern-owned meaningを重複しない
7. one Procedure -> multiple Applicationsを単一owner fieldなしで保持
8. actor uncertaintyをAuthorizationへ混ぜない
9. handoff relationをruntime Workflow化せず保持
10. `realization`を除いても必要Business meaningを保持できるか検証
11. Step durable identityの必要性をhandoff / review Evidenceで説明
12. 少なくともcurrent `order_entry`より一段現実的なmulti-Application / manual / external Reference pressureを得る

## 14. 現時点の判断

### productionへ今入れない

- Canonical `BusinessProcedure` schema
- Canonical `BusinessHandoff` schema
- `realization` enum
- generic Uncertainty framework
- Workflow Engine

### Candidateとして残す

```text
Procedure identity
purpose
trigger status + meaning
actor status + business label
ordered Business Action
explicit semantic refs
```

### Derived / externalへ残す

```text
Pattern-owned meaning
Application association
Projection matrix
Diagnostics
source locator / fingerprint / revision
runtime realization
Authorization realization
```

## 15. 次のLane C作業

ADRへ進む前に、次の二つをReference pressureで閉じる。

1. **realization subtraction pressure**
   - thin Canonical candidateから`realization`を外してもBusiness-only diff / cross-view reviewが成立するか
   - 不足する場合、必要なBusiness distinctionだけを抽出する

2. **durable Step identity pressure**
   - handoff / review / cross-revision continuityにStep durable identityが本当に必要か
   - position / content fingerprint / local keyとの差を比較する

この二つとactual test executionが揃うまで、PR #482はDraft、Issue 000042はIn Progressを維持する。
