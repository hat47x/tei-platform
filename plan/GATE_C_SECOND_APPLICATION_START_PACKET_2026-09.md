# Gate C → Second Application Reference Start Packet

- Status: Active — S0 complete / S1-S4 Source-defined / S5 review-ready + freshness captured / explicit Acceptance witness pending
- Date: 2026-09-19
- Primary owner after entry: ISSUE-000040
- Inputs: ISSUE-000047 / ISSUE-000053 Phase 5 / ISSUE-000027 / ISSUE-000039
- Pre-Gate contracts:
  - \`plan/SECOND_APPLICATION_REFERENCE_PRE_GATE_CONTRACT_2026-09.md\`
  - \`plan/SECOND_APPLICATION_SCHEMA_SUFFICIENCY_PRESSURE_2026-09.md\`
  - \`plan/KNOWLEDGE_PORTABILITY_HANDOFF_2026-09.md\`
- Canonical Application impact before Gate C Ready: None

## 1. 目的

Gate C Ready後に第二Application Referenceへ着手するとき、探索・Knowledge Portability・composition・Acceptanceを別々のbranchや別々のKnowledge正本へ分裂させず、一つのworkstreamとして開始できる状態を用意する。

この文書は第二Application sourceではない。

\`\`\`text
Start Packet
  != Gate C Ready
  != Application Proposal
  != Shared Acceptance
  != Canonical source
\`\`\`

## 2. Entry Gate

Entryの機械的preflightは次を正本とする。

- `plan/GATE_C_SECOND_APPLICATION_ENTRY_PREFLIGHT_2026-09.md`
- `tools/reference-business-procedure-check/gate_c_second_application_entry_preflight_pressure_test.go`

第二Application workstreamを開始するには、Gate C Readyを**明示Evidence**で確認する。

計画文書の見出しや将来予定からReadyを推測しない。

開始時に次を一つのentry recordへ固定する。

\`\`\`text
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
\`\`\`

### Stop

次のいずれかなら第二Application source authoringへ進まない。

- Gate C statusがReadyでない
- \`R\`と\`T1\`が識別できない
- Semantic Evidenceがない
- Functional Evidenceがない
- Visual Evidenceがない
- Target差分が未整理で、second Application設計へ混入する可能性がある

Visual Evidenceだけで開始条件を満たしたとは扱わない。

## 3. Branch policy

Gate C Ready後の第二Application Referenceは、一つのprimary workstream branchへ収束する。

原則:

\`\`\`text
main
  ↓
one ISSUE-000040 workstream branch
  ↓
qualification
  ↓
portability
  ↓
composition proposal
  ↓
acceptance-related source change
  ↓
reload / validation
  ↓
Environment / Driver / Target evidence
  ↓
PR
  ↓
main
\`\`\`

### 禁止

- exploration用branchを長期残存させる
- portability用branchを別系列として残す
- Pattern用 / Environment用 / Driver用にbranchを恒常分割する
- merge済みworkstreamから新しい枝を無制限に増やす
- 同じ第二Applicationの正本候補を複数branchで並行authorする

補助的な短命branchが必要になった場合も、同じprimary workstreamへ直ちに戻し、単独の長期系統にしない。

## 4. First Reference scope

最初の実Referenceはread-only customer lookupへ限定する。

### Shared authority candidates

- \`customer.customer\`
- \`customer.customer_id\`
- \`master_data.customer ^2\`
- \`master_data.customer/find_by_id\`

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
- \`R\`
- \`T1\`
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

\`applicable\`を\`accepted\`へ変換しない。

### S3 — Destination decision packet

次の未決定を明示的に解く。

1. Application ID
2. Business context / Relation
3. Pattern ID
4. datasource alias
5. Authorization requirement
6. development / production binding policy
7. first-target presentation scope

未決定が残る項目は\`unknown\`として保持し、originから補完しない。

### S4 — ISSUE-000040 composition candidate

既存grammarだけでProposalを作る。

概念形:

\`\`\`text
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
\`\`\`

reference Patternはdestination-owned。

origin \`order_entry.customer_reference\`をコピー正本にしない。

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

origin \`order_entry.master\` bindingをcopyしない。

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

\`\`\`text
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
\`\`\`

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

\`\`\`text
less code reading
  != less Evidence
\`\`\`

を維持する。

## 9. Commit granularity

primary workstream内では、意味の境界に沿ってcommitする。

推奨単位:

1. Gate C entry / qualification Evidence
2. portability / destination decision
3. composition Proposal
4. accepted source
5. Environment Binding
6. Driver / Target Evidence
7. change propagation Evidence

「ファイルごと」ではなく「検証可能な意味単位」でcommitする。

一つの意味変更を複数branchへ分けない。

## 10. PR / merge completion

第二Application workstreamのPRは、最低限次を説明する。

- Gate C entry Evidence
- shared / local / Environment boundary
- reused Canonical identities
- destination-owned identities
- unknown / incompatible
- Acceptance Evidence
- reload / Validation Evidence
- Driver / Target Evidence
- propagation Evidence
-追加schemaを導入したか / 見送ったか、その理由

merge後はprimary workstream branchを\`main\`へ戻し、次作業の起点として枝を残さない。

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

現在も第二Canonical Application source、Shared Acceptance、Environment binding、Target実装は作成していない。

S3 Destination Decision Packetは `plan/SECOND_APPLICATION_S3_DESTINATION_DECISION_PACKET_2026-09.md` でSource-definedまで進んだ。`repair_intake` はS3の明示destination candidateであり、`repair_intake-test-only`からの自動昇格ではない。S4 composition candidateも `plan/SECOND_APPLICATION_S4_COMPOSITION_CANDIDATE_2026-09.md` でSource-definedまで進んだ。S5は `plan/SECOND_APPLICATION_S5_SHARED_ACCEPTANCE_REVIEW_PACKET_2026-09.md` でreview-readyまで進み、`plan/SECOND_APPLICATION_S5_FRESHNESS_CAPTURE_2026-09.md` で current base / shared authority freshness も exact source fingerprint付きでcaptureした。残る前提はexplicit Acceptance witnessである。したがってShared Acceptanceはまだ成立しておらず、authoritative Application / Pattern source、Environment binding、Target実装は作成していない。

## 12. Start Packet完了条件

- [x] Gate C entry recordの必須項目を固定した
- [x] 一つのprimary workstream branchへ収束する規則を固定した
- [x] first Reference scopeをread-only customer lookupへ限定した
- [x] ISSUE-000047 → Phase 5 → ISSUE-000040 → Acceptanceの順序を固定した
- [x] Application / Environment authorityを分離した
- [x] unknown / incompatibleのstop条件を固定した
- [x] existing grammar firstを維持した
- [x] AI Builder / Human Review handoffを定義した
- [x] commit / PR / merge規則を定義した
- [x] Gate C前にsecond Application sourceを追加していない
