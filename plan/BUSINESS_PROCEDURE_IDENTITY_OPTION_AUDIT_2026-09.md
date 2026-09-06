# Business Procedure Identity / Canonical Meaning Option Audit — 2026-09

Status: Design pressure audit / no ADR yet / identity-only adoption suspended

## 目的

Business Procedureのfile-native accepted sourceをSource Index / Shared Acceptance / cross-view diffへ接続する最小contractを比較する。

比較対象は次の四案である。

- A. minimal Canonical Procedure identity shell
- B. Subject contractをWorld外accepted design targetへ拡張
- C. Authoring identity registryを別層として正式化
- D. thin normalized Canonical Business Procedure meaning

full Runtime Workflow modelは比較対象外とする。

## 1. 前段Evidence

Canonical Procedure modelなしでもReferenceで次は成立した。

- Business Action -> Application Pattern cross-view
- manual / external Business Action
- multiple Procedure -> same Pattern
- Pattern-owned meaning derivation
- cross-view diagnostic
- Reference-local stable Procedure key
- Procedure間handoff
- review target continuity
- source fingerprintによるstale検出
- file move / logical rename分離
- Preview / Reject / Accept / reloadの機械的安全性

したがってWorkflow Engineやfull Procedure objectを先に導入する必要はない。

## 2. identity-only仮説に対する新しい反証

前段では残件を主にRQ-002の

```text
authoring source -> Canonical subject
```

と捉え、Option Aを第一候補とした。

しかしformal architectureは、

```text
Authoring
  -> parse / resolve / accept
  -> Canonical Semantic meaning / IR
```

を基線とし、Data / Function-Interaction / Businessを同じaccepted meaningの異なる視点として扱う。

現在のfile-native source pressureでは、BP-04等のBusiness-only wordingをAcceptしても`ir.World`は不変である。

Option Aの

```text
BusinessProcedureIdentity{id}
```

を追加しても、同じProcedure IDの内容変更ではidentity shellは変わらない。

したがってIDだけでは次を表せない。

- Business-only Semantic Diff
- Business change -> Data / Function impact
- accepted Business meaningのread-only exploration
- reload後のBusiness Conformance
- 000039 change graph上のBusiness meaning change

**Canonical trace targetが存在することと、accepted Business meaningがCanonical側で保持されることは別問題である。**

## 3. 評価基準

1. RQ-002のSource-to-Canonical trace
2. RQ-003の単一正本
3. RQ-017の二重管理最小化
4. RQ-023のone-maintainer性
5. RQ-025のmanual / external Business Action保持
6. RQ-026の三視点cross-check
7. Business-only Semantic Diff / impact
8. Source Index reuse
9. Shared Acceptance target
10. 000039への拡張
11. file move / logical rename分離
12. concept creep抑制
13. Pattern-owned meaningの非重複
14. Runtime Workflow semanticsの非侵入

## 4. Option A — minimal Canonical Procedure identity shell

### 形

```text
BusinessProcedureIdentity
  id
```

file-native sourceの`procedure.id`からnormalized resultとして導出する。

### 強み

- RQ-002のCanonical Subjectを作りやすい
- Source Index targetへ直接使える
- Shared Acceptance targetをpathから分離できる
- duplicate IDをfail closedにできる
- file move / logical renameを自然に区別できる
- source path / fingerprintをWorldへ入れずに済む

### 限界

Business Procedureのaccepted contentが変わってもIDが同じならCanonical meaning差を表せない。

```text
same ID
changed purpose / Business Action / trigger / actor
```

をidentity shellだけでは区別できない。

### 現在の位置づけ

**必要なidentity primitive候補ではあるが、最小十分contractとはみなさない。**

ID-onlyをproductionへ先行導入する判断は保留する。

## 5. Option B — Subject contract拡張

### 形

現行Subjectの

```text
inside a TEI World
```

を広げ、World外accepted design targetもSubjectにする。

### 強み

- `ir.World`へProcedure fieldを追加しなくてよい
- Reference-local keyをSource Index targetへ使える

### 問題

- RQ-002の`Canonical subject`意味を変える
- SubjectにあることとCanonicalであることが混線しやすい
- Business-only meaning自体をどこに正規化するかは解決しない

したがってidentity-only問題の回避にはなるが、Canonical meaning問題の解答ではない。

## 6. Option C — Authoring identity registry

### 形

```text
AuthoringTarget
  key
  locator(s)
  fingerprint
  role
```

### 強み

- review / handoff / file move targetを明示しやすい
- WorldへProcedureを追加しなくてよい

### 問題

- `AuthoringTarget -> Canonical meaning`のlinkが別途必要
- Source Index / Registry / Acceptanceの三層化
- RQ-023上の保守負担
- Business-only semantic diffのCanonical representationは依然未解決

現時点で新registryをproductionへ追加する根拠は弱い。

## 7. Option D — thin normalized Canonical Business Procedure meaning

### 形

full Workflow modelではなく、Business Procedure sourceのうちaccepted Business residual meaningだけを正規化する。

```text
BusinessProcedureMeaning
  id
  purpose
  trigger
  actor
  ordered steps

BusinessProcedureStep
  id
  businessAction
  realization
  explicit Application Pattern refs
  explicit Data refs
  explicit Relation refs
  explicit Validation refs
  explicit Capability refs
```

### 入れないもの

- source path / fingerprint / range
- schemaVersion / artifactRole
- experimental identityStatus
- Target UI detail
- runtime branch / timer / retry / queue / saga / scheduler
- runtime state
- Authorization Role

### Pattern-owned meaning

Procedureが`order_entry.customer_reference`を参照しても、そのPatternが所有するData / Relation / Query / Capability / DatasourceをProcedureへコピーしない。

Procedure側はexplicit Pattern refだけを持つ。

Business側でPatternから導けないValidation等を明示した場合だけdirect refを保持する。

### 強み

- Business-only changeをCanonical semantic changeとして扱える
- IDはpath-independent targetとして同時に使える
- manual / external actionをfake Capabilityなしで保持できる
- Studio / AI / Matrix / impact analysisがsource parser専用経路だけに依存しなくてよい
- Source IndexはProcedure Subjectへ接続できる

### リスク

- dedicated Business semantic modelがCoreへ増える
- current `order_entry`だけから過度にsingle-Application前提を固定する危険
- handoffが別Reference setにあるためProcedure bodyだけではBusiness meaning全体を覆わない
- 「Canonical Procedure」からRuntime Workflowへ概念が膨張する危険

このためfield上限とnon-goalをtest / ADRで固定する必要がある。

## 8. test-only Pressure

### Option A

`identity_shell_pressure_test.go`

- ID-only field guard
- source-derived identity
- duplicate ID rejection
- file move不変
- wording changeでidentity不変
- logical renameでidentity change
- Source Index Subject候補

### Option D

`canonical_meaning_pressure_test.go`

- thin Business semantic field guard
- Business-only wording changeをmeaning差として観測
- pure file moveではmeaning不変
- manual BP-04保持
- Pattern-owned meaning非重複
- explicit Validation ref保持
- set-like ref order normalization

actual Repository runnerがGo testへ到達するまではどちらもtest-only specificationでありgreen Evidenceではない。

## 9. 比較表

| 基準 | A: ID shell | B: Subject拡張 | C: Authoring registry | D: thin Canonical meaning |
|---|---|---|---|---|
| RQ-002 trace | 強い | 要再解釈 | 別link必要 | **強い** |
| 単一正本 | source-derivedなら良 | 良 | registry第二正本化注意 | source-derivedなら良 |
| Business-only diff | **不可** | 未解決 | 未解決 | **可能** |
| RQ-025 manual meaning | IDしか保持しない | World外で保持 | registry外で保持 | **直接保持** |
| RQ-026 cross-view | target nodeのみ | target nodeのみ | mapping経由 | **meaning + refs** |
| Source Index reuse | 直接 | 契約変更して直接 | adapter必要 | **直接** |
| Shared Acceptance target | 直接 | 非Canonical target | registry target | **IDで直接** |
| 000039 | identity nodeのみ | graph意味変更 | adapter必要 | **meaning change node** |
| file move耐性 | 強い | 強い | 強い | 強い |
| concept creep | 要guard | Subject意味膨張 | identity層膨張 | **semantic field guard必須** |
| runtime Workflow化リスク | 低 | 低 | 低 | 中・明示guard必要 |

## 10. 更新後の優先順位

### 第一比較対象: D と A をセットで検証

Option DのID部分はOption Aを包含する。

したがってproductionへAだけを先に入れるのではなく、

```text
A: identityだけで十分か
vs
D: cross-viewに必要なBusiness residual meaningまで必要か
```

を同じReference Evidenceで比較する。

現時点のEvidenceはD側を支持し始めている。

### 第二候補: B

DのCore blast radiusが過大で、かつCanonical meaningをWorld外へ正式化する設計原則が別Evidenceから成立した場合のみ再検討する。

### 第三候補: C

review / handoff identityを既存Source Index / Acceptanceで扱えない反復Evidenceが出た場合のみ検討する。

## 11. production integration前の停止条件

次のいずれかならCoreへ進めない。

- actual Repository testが未実行
- Business detailとruntime Workflow semanticsの境界が曖昧
- Pattern-owned meaningをProcedureへ複製しないと成立しない
- source path / fingerprintをCanonical modelへ入れる必要がある
- actorからAuthorization Roleを導出する必要がある
- current Referenceだけではhandoff / multi-Application pressureが不足している

## 12. 次のPressure

次はhandoffをBusiness semantic relationとして扱う場合に、

- Procedure body内へ押し込むべきか
- separate Business relationとして正規化すべきか
- file-native reference setをどうaccept / traceするか

を確認する。

これが成立するまでproduction `ir.World`へProcedure fieldを追加しない。
