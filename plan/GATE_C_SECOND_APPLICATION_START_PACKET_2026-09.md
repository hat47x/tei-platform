# Gate C → Second Application Reference Start Packet

- Status: Active — S0-S3 complete / S4 revised after full-validation preflight / S5 review regeneration pending
- Date: 2026-09-19
- Primary owner after entry: ISSUE-000040
- Inputs: ISSUE-000047 / ISSUE-000053 Phase 5 / ISSUE-000027 / ISSUE-000039
- Pre-Gate contracts:
  - `plan/SECOND_APPLICATION_REFERENCE_PRE_GATE_CONTRACT_2026-09.md`
  - `plan/SECOND_APPLICATION_SCHEMA_SUFFICIENCY_PRESSURE_2026-09.md`
  - `plan/KNOWLEDGE_PORTABILITY_HANDOFF_2026-09.md`
- Canonical Application impact before Gate C Ready: None

## 1. 目的

Gate C Ready後に第二Application Referenceへ着手するとき、探索・Knowledge Portability・composition・Acceptanceを別々の作業正本へ分裂させず、一つのISSUE-000040 workstreamとして継続できる状態を用意する。

この文書は第二Application sourceではない。

```text
Start Packet
  != Gate C Ready
  != Application Proposal
  != Shared Acceptance
  != Canonical source
```

## 2. Entry Gate

Entryの機械的preflightは次を正本とする。

- `plan/GATE_C_SECOND_APPLICATION_ENTRY_PREFLIGHT_2026-09.md`
- `tools/reference-business-procedure-check/gate_c_second_application_entry_preflight_pressure_test.go`

第二Application workstreamを開始するには、Gate C Readyを**明示Evidence**で確認する。

計画文書の見出しや将来予定からReadyを推測しない。

開始時に次を一つのentry recordへ固定する。

```text
gateC.status = ready

baseline:
  handover baseline B
  baseline repository revision R

first target:
  Web build revision T1

evidence:
  semantic conformance = <ref>
  interaction / functional = <ref>
  visual fidelity = <ref>

target differences:
  <explicit summary>

unresolved:
  <remaining items>
```

### Stop

次のいずれかなら第二Application source authoringへ進まない。

- Gate C statusがReadyでない
- `R`と`T1`が識別できない
- Semantic Evidenceがない
- Functional Evidenceがない
- Visual Evidenceがない
- Target差分が未整理で、second Application設計へ混入する可能性がある

Visual Evidenceだけで開始条件を満たしたとは扱わない。

## 3. Development-flow / workstream continuity

Repositoryの現在のsolo-development flowでは、branch / PRをProduct状態やreview境界として使わず、mainへ直接commitする。

第二Application Referenceの継続性はbranchではなく、ISSUE-000040、Proposal fingerprint、Source / Evidence、Acceptance状態で管理する。

原則:

```text
current main
  ↓
ISSUE-000040 qualification / portability
  ↓
composition proposal
  ↓
Shared Acceptance boundary
  ↓ Accept後のみ
authoritative source
  ↓
reload / validation
  ↓
Environment / Driver / Target evidence
  ↓
change propagation Evidence
```

### 禁止

- exploration / portability / Pattern / Environment / Driverごとにremote branchを作る
- 同じ第二Applicationの正本候補を複数系列で並行authorする
- branch名やPR状態をAcceptance / Canonical authorityとみなす
- concurrent sessionの古いmain snapshotからそのまま変更を適用する
- health checkなしにコード変更をmainへ入れる

各作業はpush直前にcurrent mainへrebase / freshness確認し、コード変更時はRepository正本のhealth checkを通す。文書だけの変更でも、link、状態、Gate / Evidence claim、正本性を確認する。

## 4. First Reference scope

最初の実Referenceはread-only customer lookupへ限定する。

### Shared authority candidates

- `customer.customer`
- `customer.customer_id`
- `master_data.customer ^2`
- `master_data.customer/find_by_id`

### Destination-owned decisions

Gate C後に初めて決定する。

- Application Canonical ID
- Business context
- destination Relation
- destination Pattern ID
- datasource alias
- Authorization requirement
- first-target surface

### Environment-owned decisions

- development binding
- production binding
- Resource / Permission if actually required

### Out of scope

- customer write
- customer maintenance ownership
- new write Contract
- write Capability
- second Target
- general package manager
- portability inheritance / override
- cross-repository sync

## 5. Work sequence

### S0 — Gate C entry capture

Gate C Evidenceをentry recordへ固定し、entry preflightで`entry-evidence-complete`を確認する。

PreflightはGate C authorityそのものではなく、accepted Gate C Evidenceがsecond Application開始に必要な形で揃っているかを確認する。

Output:

- Gate C Evidence refs
- `R`
- `T1`
- target difference summary
- unresolved

No source change.

### S1 — ISSUE-000047 candidate qualification

intent:

> destination Business contextからCustomerをIDで参照したい。

候補をcurrent Repositoryからread-only探索する。

最低対象:

- Customer Semantic
- Customer ID Type
- Customer Contract
- Contract Query
- origin reference Pattern provenance

判定:

- applicable
- incompatible
- unknown

Outputにはauthority source / discovery provenance / qualification reason / missing conditionを残す。

### S2 — Phase 5 portability relation

S1 candidateを次へ分類する。

- linked reuse
- conditional linked reuse
- adopted local variation
- rebind-required

保持:

- authority owner
- dependency subject
- provenance
- freshness input
- destination requirement

`applicable`を`accepted`へ変換しない。

### S3 — Destination decision packet

次の未決定を明示的に解く。

1. Application ID
2. Business context / Relation
3. Pattern ID
4. datasource alias
5. Authorization requirement
6. development / production binding policy
7. first-target presentation scope

未決定が残る項目は`unknown`として保持し、originから補完しない。

### S4 — ISSUE-000040 composition candidate

既存grammarだけでProposalを作る。

概念形:

```text
application <destination> {
    uses semantic customer

    database <destination-alias> {
        target master_data
        access read
        requires {
            contract master_data.customer ^2
        }
    }
}
```

reference Patternはdestination-owned。

origin `order_entry.customer_reference`をコピー正本にしない。

### S5 — Shared Acceptance

composition candidateをISSUE-000027 / ADR-0022のAcceptance boundaryへ渡す。

Acceptance前:

- authoritative sourceを変更しない
- Environment bindingを作らない
- AI Proposalをaccepted Decisionとみなさない

Acceptance後のみsource authoringへ進む。

### S6 — Source / reload / validation

accepted compositionを通常のTEI sourceとしてauthorする。

確認:

- AI / exploration indexなしでreload可能
-同じCanonical meaningを再構成
- Validation / Source Indexが成立
- dedicated portability grammar不要という仮説を再評価

### S7 — Environment Binding Proposal

Application source acceptance後にEnvironment側を別authorityとして扱う。

確認:

- destination Application / alias
- target Database instance
- Database definition
- Provider
- Permission / Resource requirement

origin `order_entry.master` bindingをcopyしない。

### S8 — Driver / First Web Target

read-only customer lookupを、

1. Driver
2. 第一Web Target

で再現する。

同じBusiness / Semantic meaningをTarget側で再定義しない。

### S9 — Change propagation Evidence

最低限次をPressureする。

- shared Semantic change → dependency requalification
- Contract compatible minor → compatibility再評価
- Contract major → dependency再評価
- origin Pattern change → provenance drift
- destination Pattern change → originへback-propagateしない
- origin alias / Environment binding change → destinationへ伝播しない
- unrelated origin change → stale-allしない

ISSUE-000039が利用可能なら共通impact解析を使う。

独自impact engineをISSUE-000040へ作らない。

## 6. Stop conditions

workstream中に次が発生した場合、次stepへ自動進行しない。

### Candidate unknown

必要Evidenceを追加取得する。

originの選択を推測して埋めない。

### Candidate incompatible

別候補またはdestination設計変更へ戻る。

Semantic identity自体まで不用意にincompatibleへ広げない。

### New schema required

既存grammarで表現不能な具体的Evidenceを記録する。

先にKnowledge Package / inheritance / overrideを作らない。

### Authorization unresolved

read routeが分かっても実行authorityを推測しない。

### Environment unresolved

Application acceptanceからBindingを自動生成しない。

### Target semantic divergence

Target実装都合でCanonical meaningを黙って変えない。

必要ならbaseline / accepted source変更へ戻る。

## 7. AI Builder handoff

AI Builderへ渡すcontextは少なくとも次を含む。

```text
Gate C entry
  R / T1 / Evidence

intent
  read-only customer lookup

shared candidates
  identity / authority / qualification

local decisions
  accepted / unresolved

provenance
  origin Pattern / source

portability
  reuse relation / freshness

acceptance
  current stage

implementation
  allowed source kinds
  forbidden shortcuts
```

AIへRepository全体を無差別に読ませることを標準入口にしない。

## 8. Human review handoff

Human Reviewではgenerated codeより先に、

- intent
- shared authority
- destination-owned meaning
- Environment-owned choices
- unresolved
- impact candidates
- exact source refs
- Acceptance status

を提示する。

```text
less code reading
  != less Evidence
```

を維持する。

## 9. Commit granularity

ISSUE-000040 workstream内では、意味の境界に沿ってmainへcommitする。

推奨単位:

1. Gate C entry / qualification Evidence
2. portability / destination decision
3. composition Proposal
4. accepted source
5. Environment Binding
6. Driver / Target Evidence
7. change propagation Evidence

「ファイルごと」ではなく「検証可能な意味単位」でcommitする。

一つの意味変更を複数の作業系列へ分けず、並行sessionがある場合はcurrent mainと関連Source / Evidenceを直前に再確認する。

## 10. main commit completion

各意味単位のcommitは、最低限次のうち該当する情報を正本文書またはEvidenceへ残す。

- Gate C entry Evidence
- shared / local / Environment boundary
- reused Canonical identities
- destination-owned identities
- unknown / incompatible
- Acceptance Evidence
- reload / Validation Evidence
- Driver / Target Evidence
- propagation Evidence
- 追加schemaを導入したか / 見送ったか、その理由

branch / PRを完了条件にはしない。コードを変更した場合はpush前のRepository health checkを必須とし、文書だけの場合もリンク・状態・Gate / Evidence claimを確認する。

## 11. 現在状態

2026-09-19、第一Web TargetはGate C Readyに到達した。

actual entry:

- `plan/SECOND_APPLICATION_GATE_C_ENTRY_2026-09.md`
- `B / R / T = B / e5231839a5dcc62cce1181a2731fcb7c70d16736 / 0ba87a4e0a96f70a198b03ff7652fb00e3826abe`

S0は完了した。

S1 / S2は、

- `plan/SECOND_APPLICATION_S1_S2_QUALIFICATION_PORTABILITY_2026-09.md`
- `tools/reference-business-procedure-check/second_application_s1_s2_actual_pressure_test.go`

でSource-definedまで進んだ。

現在も第二Canonical Applicationのauthoritative destination source、Environment binding、Target実装は作成していない。Shared Acceptanceのactual Decisionも未成立である。

S3はSource-defined後、S6 preflightでdestination Relation owner / keyをaddendumした。旧S4 2-source ProposalはAcceptance前にsupersededし、current revised Proposalは `plan/SECOND_APPLICATION_S4_FULL_VALIDATION_PREFLIGHT_2026-09.md`、fingerprint `c18ae0fdef8e01aa965e2ad84c86ecfc21ca0608c07b1808e73368a5f434458d` である。

2026-09-21時点で、revised executable pressure、multi-layer Shared Acceptance実装、actual `reference/workspace` read-only Preview、別生成AIによる独立レビュー（`review-clear-with-conditions`）まで完了した。required pre-Acceptance fixは無い。残るS5停止条件はexact `c18ae...`に対するexplicit human Acceptance decisionだけである。

Acceptance実行契約:
- `plan/SECOND_APPLICATION_S5_ACCEPTANCE_EXECUTION_PACKET_2026-09.md`

Acceptance成功後のS6 Source / reload / validation契約:
- `plan/SECOND_APPLICATION_S6_SOURCE_RELOAD_VALIDATION_PACKET_2026-09.md`

一般的な作業継続指示をAcceptance witnessへ読み替えない。

## 12. Start Packet完了条件

- [x] Gate C entry recordの必須項目を固定した
- [x] 一つのISSUE-000040 workstreamへ収束し、remote branchを増やさない規則へ更新した
- [x] first Reference scopeをread-only customer lookupへ限定した
- [x] ISSUE-000047 → Phase 5 → ISSUE-000040 → Acceptanceの順序を固定した
- [x] Application / Environment authorityを分離した
- [x] unknown / incompatibleのstop条件を固定した
- [x] existing grammar firstを維持した
- [x] AI Builder / Human Review handoffを定義した
- [x] direct-to-main / health-check / concurrent freshness規則へ更新した
- [x] Gate C前にsecond Application sourceを追加していない
