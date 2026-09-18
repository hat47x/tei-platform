# Business Procedure Identity — 2026-09

Status: **Design pressure audit / no ADR yet / identity-only adoption suspended**

## この文書について

本書は、Business Procedureに関する identity 調査を時系列で行った次の3文書を1本へ統合したものである。

1. `BUSINESS_PROCEDURE_IDENTITY_PRESSURE_2026-09.md`（元・Pressure 1〜6の広い探索）
2. `BUSINESS_PROCEDURE_IDENTITY_SHELL_PRESSURE_2026-09.md`（元・ID-onlyのminimal shell probe）
3. `BUSINESS_PROCEDURE_IDENTITY_OPTION_AUDIT_2026-09.md`（元・4案の比較audit、最終結論）

いずれもorphan（他文書から参照されていない）で、単一の連続した調査だったため統合した。個別ファイルの履歴はGit historyから追跡できる。

## 目的

`Procedure`をtop-level Canonical identityへ昇格させる前に、Business Procedure Referenceでどのidentityが本当に必要かを段階的に確認する。

この文書はCanonical schema追加の提案ではない。

review / approval continuityとProcedure間handoffの追加圧力は、`BUSINESS_PROCEDURE_REVIEW_HANDOFF_PRESSURE_2026-09.md`へ分離して記録する。

---

# Part 1 — 広いPressure（Pressure 1〜6）

## Authoring / Derivedの前提

対象は`order_entry`の二つのReference authoring fixtureである。

- `reference/business-system-design/order-entry-business-procedure.json`
- `reference/business-system-design/order-entry-follow-up-procedure.json`

現在は次を明示する。

```text
artifactRole = reference-authoring-fixture
canonical = false
Procedure identityStatus = reference-local-stable
```

`reference-local-stable`はReference corpus内のauthoring keyであり、Canonical World identityではない。

fixtureにはApplication / Semantic Worldから導出できないBusiness側の意味を保持する。

```text
purpose
trigger / event candidate
actor / job context candidate
manual / external / unresolved Business Action
judgment / continuation
```

一方、Application Patternから得られるData / Relation / Query / Capability / Datasource、Business Action × Pattern Matrix、diagnosticはderived Projectionとして扱う。

## Pressure 1: multiple Procedure -> same Pattern

二つのauthoring fixtureは同じ`order_entry.order_record`を利用する。

```text
order-entry.order-handling / BP-01
  -> order_entry.order_record

order-entry.follow-up / FU-01
  -> order_entry.order_record
```

期待する性質:

- Pattern identityは同じまま再利用する
- Business Actionを一つへ畳まない
- Procedureごとに別Capabilityや別Data identityを捏造しない
- 同じPatternを使うこと自体を曖昧性エラーにしない

### 観測

基本的なmultiple Procedure -> same Pattern traceabilityにはCanonical Procedure identityは不要である。

Reference-local Procedure key、step key、explicit Pattern identityで区別できる。

## Pressure 2: Procedure key renameとcross-view meaning

Procedure keyだけを変更し、Business Action / step / Pattern referenceを変えない場合、Pattern-derived cross-view meaningは変化しない。

これはProcedure keyがApplication Pattern meaningのauthorityではないことを示す。

一方、review / handoffでstable authoring keyとして使う段階では、key renameはlogical authoring identity changeとなる。

つまり次を分ける。

```text
Pattern-derived meaning
  Procedure key renameだけでは不変

review / handoff target
  Procedure key renameで変化
```

## Pressure 3: artifact path rename

同じProcedure / step / PatternでもReference authoring fixtureのpathを変えると、pathを含むlocatorは変化する。

file pathをBusiness semantic identityとして使わない。

## Pressure 4: 既存Source Indexでどこまで足りるか

TEIには`internal/sourceindex`とneutralな`subject.Subject`がある。

```text
Subject
  = path-independent semantic / logical target

Artifact
  = path + format + content fingerprint

Reference
  = Subject -> ArtifactPath + Range + Role
```

### 4A pure file move

同一bytesを別pathへ移した場合、Pathは変わるがArtifact Fingerprintは一致する。

pure moveのcontent continuityには既存Fingerprintを利用できる。

### 4B formatting-only change

同じJSON meaningでもbytesを変えればFingerprintは変化する。

したがってFingerprintはsemantic identityではない。

### 4C existing Pattern Subject

複数Business Procedureが同じApplication Patternを参照しても、既存path-independent Pattern Subjectへ複数Referenceを結び付けられる。

Pattern変更からBusiness Procedure利用箇所を辿るためだけなら、新しいCanonical Procedure kindは不要である。

## Pressure 5: Business authoring residualはPatternから再生成できるか

primary fixtureにはPatternを持たないBP-04 / BP-09がある。

```text
BP-04
  manual-unresolved

BP-09
  manual-external-unresolved
```

これらはauthoring fixtureには存在するがPattern-derived meaning rowsには存在しない。

### 観測

Pattern-owned meaningだけからBusiness Procedureを再構成するとmanual / external / unresolved Business Actionを失う。

したがってBusiness Procedure authoring sourceはPattern-derived cacheではない。

purpose / trigger / actor candidateも同じくBusiness側のauthoring meaningである。

### Business-only wording change

manual Business Action文言だけを変更した場合:

```text
Business-side Projection
  changed

Pattern-derived Application meaning
  unchanged

cross-view diagnostic
  unchanged unless a real cross-view conflict changes
```

Business視点をFunction/Data視点の単なる表示へ還元しない。

## Pressure 6: review / handoffによるstable authoring keyの必要性

前段のPressure 1〜5では、Canonical Procedure identityは不要というEvidenceが中心だった。

review history continuityとProcedure間handoffを加えると、安定したauthoring-level logical keyには具体的な価値が出る。

そのためReferenceでは`procedure.id`を次として扱う。

```text
identityStatus = reference-local-stable
```

ただしこれはCanonical `Procedure`への昇格ではない。

詳細:

- `plan/BUSINESS_PROCEDURE_REVIEW_HANDOFF_PRESSURE_2026-09.md`
- `reference/business-system-design/order-entry-procedure-reference-set.json`
- `tools/reference-business-procedure-check/review_handoff_pressure_test.go`

## 000039との境界

`plan/issue/000039-business-change-rehearsal.md`は将来のrevision-aware semantic diff / dependency propagationを所有する。

レーンCはそのgraphを先取り実装しない。

今回固定するidentity区別は000039の入力として利用できる。

```text
same Procedure key + changed revision/fingerprint
  = same review target, changed source snapshot

changed Procedure key
  = logical authoring identity change
```

## Part 1 時点の判断

### 維持する境界

- Business Procedure fixtureはReference authoring input
- Matrix / diagnosticはderived Projection
- manual / external Business Actionをfake Functionへ変換しない
- Pattern-owned meaningをBusiness Procedureへ重複記述しない
- Source Index Artifact fingerprintをsemantic Procedure identityとして流用しない
- `procedure.id`はReference-local stable authoring keyとして使う
- `Procedure` / `Step`はCanonical Worldへ追加しない

### identityを混同しない

```text
A. Canonical business semantic identity
B. Reference-local authoring identity
C. Source / Evidence locator
D. Repository revision provenance
```

この時点ではBの必要性にはEvidenceが出たが、Aの必要性はまだ実証されていない。

---

# Part 2 — Minimal Identity Shell Probe

## 目的

Business Procedureのpath-independent identityを、source pathやruntime semanticsから分離できるか確認する。

このPressureはCanonical `Procedure`採用ではない。

## 1. shellの上限

probe typeは意図的に次だけを持つ。

```text
CanonicalProcedureIdentityShellProbe
  ID
```

次は持たせない。

- purpose
- trigger
- actor
- step
- Business Action
- instruction
- handoff
- branch
- lifecycle
- state
- timer
- retry
- authorization mapping
- source path
- fingerprint
- Source range
- runtime orchestration

## 2. authoritative sourceから導出する

identity shell用の第二ID fileは作らない。

```text
file-native Business Procedure source
  ↓ strict decode
procedure.id
  ↓ normalized
identity shell
```

同一IDを複数authoritative sourceが宣言した場合はfail closedにする。

## 3. 成立したidentity性質

### pure file move

source bytes / Procedure IDが同じならidentity shellは不変。

### Business wording change

purpose / Business Action等が変わってもProcedure IDが同じならidentity shellは不変。

### logical rename

`procedure.id`変更はidentity change。

名称類似からrenameを推測しない。

### Source Index target

```text
kind = business-procedure
id   = Procedure ID
```

というpath-independent Subject候補を作れ、Artifact path / fingerprint / rangeはSource Index側へ残せる。

## 4. 重要な追加反証

前段では、このidentity shellだけでRQ-002のCanonical trace targetを作れれば最小十分かもしれないと考えた。

その後の`BUSINESS_PROCEDURE_CANONICAL_MEANING_PRESSURE_2026-09.md`で、次の不足が明確になった。

```text
same Procedure ID
+
changed accepted Business meaning
```

ではidentity shellは変わらない。

したがってIDだけでは、

- Business-only Semantic Diff
- Business change -> Data / Function impact
- accepted Business meaningのCanonical exploration
- reload後Business Conformance
- 000039 change graph上のcontent change

を表現できない。

## 5. 更新後の位置づけ

identity shellは不要ではない。

むしろ、thin Canonical Business Procedure meaningの

```text
ID部分
```

として有用である可能性が高い。

ただし、**ID-onlyをproduction `ir.World`へ先行導入する判断は停止する。**

比較対象は次とする。

```text
A: identity only
vs
D: identity + cross-viewに必要な薄いBusiness residual meaning
```

この時点のEvidenceはD側を支持し始めている（→ Part 3で確定）。

## 6. Option Dとの境界

thin Business meaning候補は次を保持する。

```text
Procedure
  id
  purpose
  trigger
  actor
  ordered steps

Step
  id
  businessAction
  realization
  explicit semantic refs
```

一方、identity shellで排除したruntime semanticsはOption Dでも排除する。

- retry
- timer
- queue
- saga
- scheduler
- runtime state
- Authorization Role

したがって「identity-onlyでは足りない」ことを理由にWorkflow Engineへ飛ばない。

## 7. handoffとの関係

handoffはProcedure bodyへ埋め込むのではなく、

```text
Procedure/Step -> Procedure/Step
```

のthin Business semantic relationとして別Pressureを行っている。

詳細:

- `BUSINESS_PROCEDURE_HANDOFF_2026-09.md`（Part 1、旧`HANDOFF_CANONICAL_PRESSURE`）
- `canonical_handoff_pressure_test.go`

## 8. Shell Probe Verification

`tools/reference-business-procedure-check/identity_shell_pressure_test.go`

主なtest:

- ID-only field guard
- authoritative sourceからidentity導出
- duplicate ID reject
- pure file move不変
- Business wording changeでidentity不変
- logical renameでidentity change
- Source Index Subject候補

actual Repository runnerがcheckout / Go testへ到達するまではgreen Evidenceとは数えない。

## 9. Shell Probe段階のproduction判断

identity shell単独ではCore integrationへ進めない。

次の全体Pressureを先に満たす必要がある。

1. thin Procedure body meaning
2. thin handoff relation
3. Pattern-owned meaningの非重複
4. manual / external work保持
5. Business-only semantic diff / impact
6. file move / logical rename
7. actual Repository test green

その結果、full Business contentをCanonical layerへ置かずとも同等のdiff / impact / explorationが成立すると実証できた場合だけ、identity-only案を再評価する。

---

# Part 3 — Option Audit（現在の結論）

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

`identity_shell_pressure_test.go`（Part 2参照）

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

## Verification（全体）

```bash
go test ./tools/reference-business-procedure-check
```

主なtest:

- `TestMultipleProceduresCanReferenceSamePatternWithoutCanonicalProcedureIdentity`
- `TestArtifactAndStepEvidenceLocatorSurvivesExperimentLocalProcedureRename`
- `TestArtifactRenameStillChangesNonCanonicalEvidenceLocator`
- `TestSourceIndexArtifactFingerprintSurvivesPureFileMove`
- `TestSourceIndexArtifactFingerprintIsContentIdentityNotSemanticIdentity`
- `TestCanonicalPatternSubjectRemainsPathIndependentAcrossProcedureArtifacts`
- `TestPatternDerivedProjectionCannotReplaceManualBusinessAuthoring`
- `TestBusinessWordingChangeDoesNotPretendApplicationMeaningChanged`
- `TestPurposeAndActorCandidatesRemainAuthoringMeaning`
- identity shell pressure tests（Part 2）
- canonical meaning pressure tests（Option D）
- review / handoff pressure tests

actual Repository runnerによるgreen確認まではEvidence in progressとする。
