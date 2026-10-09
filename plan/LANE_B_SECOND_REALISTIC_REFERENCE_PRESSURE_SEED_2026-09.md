# Lane B — Second Realistic Business Reference Pressure Seed — 2026-09

Status: **Lane B / provenance-backed pressure seed / not a Business authoring source / not a Reference fixture / no production schema**

Related:

- `docs/audits/LANE_B_REALISTIC_BUSINESS_CORPUS_PUBLIC_SOURCE_AUDIT_2026-09.md`
- `plan/LANE_B_SECOND_REALISTIC_REFERENCE_ACCEPTANCE_CONTRACT_2026-09.md`
- Lane C PR #482 / ISSUE-000042

## 1. 目的

Lane Cが第二Business Referenceを作る際、公開Referenceをコピーせず、かつ「何となく製造業らしい」業務事実を創作しないための最小pressure seedを渡す。

本書はBusiness authoring sourceではない。

```text
public provenance observations
  -> design pressure seed
  -> Lane CがTEI-original Reference sourceを作る
```

という一方向のhandoffである。

## 2. Fact / Candidate / Unresolvedの定義

### Fact

公開資料から高水準に確認できる範囲だけを記録する。

### Candidate

Factを圧力試験へ変換するためにTEI側で置く仮説。accepted business truthではない。

### Unresolved

Referenceを作る際にも勝手に補完しないもの。

## 3. Provenance-backed Facts

公開source auditで確認した高水準factだけを使う。

### F-01 — 複数の業務領域が存在する

CONCEPTWARE／受注生産を再現した公開TALON資料では、少なくとも次の業務領域が説明されている。

```text
master management
purchasing
manufacturing
inventory
sales
accounting
statistics
```

TEI pressure:

```text
one Business Procedure / continuation may cross one software-owned context
```

ただしこれをTEI `Application` / `Subsystem` / namespaceへ一対一変換しない。

### F-02 — 物理世界の完了を伴う業務がある

公開資料から高水準に次の業務行為が確認できる。

```text
material receipt / issue
manufacturing completion
shipment
warehouse stock movement / stocktaking
```

TEI pressure:

```text
Application Command result
  != necessarily physical-world completion
```

### F-03 — responsibility contextが単一ではない

公開資料には、system administration、purchasing、accounting approval等のresponsibility contextが現れる。

TEI pressure:

```text
Business actor / responsibility
  != Authentication Principal / Authorization Role
```

exact actor taxonomyは導かない。

### F-04 — Function surfaceが画面操作だけではない

公開説明では画面・帳票・batch・common logicを含む実装規模が説明されている。

TEI pressure:

```text
Function-Interaction viewpoint
  != UI click sequence only
```

このfactだけからTEIにBatch Canonical kindを追加しない。

## 4. Candidate pressure scenario

以下は外部modelの再現ではなく、上記Factから作る**TEI独自candidate**である。

### Candidate Event

```text
C-E01
  manufacturing / fulfillment work becomes necessary
  status = candidate
```

公開資料からexact trigger条件までは固定しない。

### Candidate Action chain

```text
C-A01  required material / input needを確認する
C-A02  material receipt / availabilityを確認する
C-A03  materialをworkへissueする
C-A04  manufacturing workを行う / 完了を確認する
C-A05  finished itemをstockへ反映する
C-A06  shipment preparation / shipment continuationへ渡す
```

このsequence自体はaccepted factではない。Lane Cが最小pressureを構成するためのcandidateである。

### 最小化案

最初のTEI-native Referenceでは全chainを実装する必要はない。

例えば:

```text
C-A03 material issue
  -> C-A04 manufacturing completion
  -> C-A05 stock continuation
```

の3 actionだけでも、次をPressureできるなら十分である。

```text
manual / physical completion
state ownership boundary
Command result != Observation
Business handoff != Runtime workflow
```

## 5. Unresolved — 勝手に決めないもの

次は公開高水準資料だけからaccepted factへしない。

```text
U-01 exact trigger / event condition
U-02 exact actor names / organization structure
U-03 exact Action ordering / optional branches
U-04 exact handoff condition
U-05 exact Data entity names / table structure
U-06 which system/Application owns each action
U-07 which actions are manual vs software-assisted
U-08 exact authorization policy
U-09 exact accounting consequences
U-10 exact state machine / status fields
```

Lane C fixtureで必要なら:

```text
candidate
unresolved
```

として明示する。

## 6. 最小cross-view questions

第二Referenceで答えを固定するのではなく、少なくとも次の問いをchecker / reviewへ投げる。

### Q-01 — physical completion

```text
software command accepted
  -> physical work complete ?
```

自動的にyesにしない。

### Q-02 — Data/state authority

```text
material / work / stock state
  which accepted semantic source owns the fact?
```

名前prefixやBusiness wordingから推測しない。

### Q-03 — Business vs Function

```text
which Business Actions have actual Application realization?
which remain manual / external / unresolved?
```

未実装Actionにfake Capabilityを足さない。

### Q-04 — responsibility vs security

```text
responsibility context
  != Authorization Role
```

Business actorからPrincipal / Permissionを生成しない。

### Q-05 — handoff vs Business Flow

```text
Action / Procedure continuation relation
  can H1/R3 preserve the needed meaning?
```

H1/R3が足りてもfull Business Flow / DFDが解けたとはしない。

### Q-06 — D4 minimum

```text
Application Pattern / Validation / Capability refs only
  sufficient?
```

Business sourceに本当にdirect Data / Relation refが必要ならD4をfail closedしてreopenする。

### Q-07 — O1 recurrence

```text
protected target / concern != realization
```

が第二Referenceでも自然に出るか。

出なければO1を一般化するEvidenceにはしない。

## 7. Lane C authoring seed shape

本書をそのままJSONへ写さない。

Lane CでRRB-P1を作る場合の最小authoring sourceは概念上:

```text
provenance
  external design-pressure sources
  no copied artifact

procedure
  id = TEI-original
  purpose = candidate or evidence-backed wording
  trigger = candidate/unresolved if exact source absent
  actor = candidate/unresolved if exact source absent
  ordered business actions
    business wording
    realization refs only when TEI accepted authority actually exists
    manual / physical / external / unresolved otherwise

handoff / continuation
  only when accepted meaning requires it
```

既存`business-procedure-reference-v0`へ無理にfieldを追加することは本書の要件ではない。

## 8. Acceptanceとの接続

```text
This seed
  -> RRB-P1 authoring input

TEI-native source accepted
  -> RRB-P1

checker / tests integrated
  -> RRB-P2

actual runner green
  -> RRB-P3
```

第二Referenceから具体的なcontract insufficiencyが出なければGate 15cは開かない。

## 9. 禁止するshortcut

```text
CONCEPTWARE/TALON structureをそのままTEI schemaへ写す
subsystem -> Application と固定する
responsibility -> Authorization Role と固定する
material receipt -> one canonical State transition と固定する
physical completion -> Command Result と固定する
all Actions -> Application Function と固定する
```

公開ReferenceはTEI designを批判するためのpressure sourceであり、TEI ontologyを決めるauthorityではない。
