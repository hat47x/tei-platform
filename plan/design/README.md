# 機能設計文書

機能ごとの設計文書を置きます。対象は、複数のIssueやPRにまたがる一つの機能について、意味モデル、authoring、Validation、実行境界、Conformanceをまとめて示す文書です。

- 何を行うかと完了条件: [`../issue/`](../issue/README.md)
- なぜその方針を採ったか: [`../adr/`](../adr/)
- 何から進め、何をEvidenceとするか: [`../README.md`](../README.md)
- 実装構造と技術境界のcurrent説明: [`../../docs/architecture.md`](../../docs/architecture.md)
- 実装・Evidenceの現在地: [`../../docs/status.md`](../../docs/status.md)
- 画面モック: [`mockups/`](mockups/README.md)

設計文書はIssueとADRの代わりではありません。作業の進行はIssue、採用した判断はADRが正本です。ここに置くのは、その二つだけでは分断される機能単位の設計本文です。小さい機能では新しい文書を作らず、Issue本文へ書きます。

Architecture文書の第二正本にもしません。`docs/architecture.md`が層と境界の全体像を示し、ここでは一機能の内部設計を扱います。

## 体裁

新規追加時は[`docs/README.md`](../../docs/README.md)の「設計文書の基本体裁」に従い、冒頭で役割、正本との関係、対象範囲、現在の結論、根拠、非主張範囲を明らかにします。項目を埋めること自体を目的にはしません。

状態は各設計文書の冒頭に書き、その文書自身を文書状態の正とします。この索引には可変な状態を重複記載しません。状態語彙も固定二択にはせず、その文書の性格に応じて、たとえば`実装済みの現行基線`、`実装前設計`、`評価待ち（未採用）`のように区別します。

ここで表すのは**文書の性格**です。Issueの進行状態、Gate判定、actual Evidenceの取得状況などの可変な現在地は、対応Issue、`docs/status.md`、Gate監査等を正とし、設計文書の状態行へ重複して持ちません。

## 一覧

| 文書 | 対象 |
| --- | --- |
| [`APPLICATION_PATTERN_V1_DESIGN.md`](APPLICATION_PATTERN_V1_DESIGN.md) | Application Patternの設計・実装基線（[ISSUE-000005](../issue/000005-application-pattern-reference.md) / ADR-0017 / ADR-0018） |
| [`LEGACY_AUTH_V1_DESIGN.md`](LEGACY_AUTH_V1_DESIGN.md) | 既存認証仕様を保持してAuthentication / Authorization Capabilityへ接続する設計（[ISSUE-000003](../issue/000003-legacy-authentication-compatibility-reference.md) / ADR-0019） |
| [`CAPABILITY_OUTCOME_V1_DESIGN.md`](CAPABILITY_OUTCOME_V1_DESIGN.md) | CapabilityのBusiness Result / named OutcomeとExecution Errorの分離（[ISSUE-000009](../issue/000009-capability-outcome-and-execution-error.md) / ADR-0020） |
| [`APPLICATION_DRIVER_V1_DESIGN.md`](APPLICATION_DRIVER_V1_DESIGN.md) | Pattern解決からCapability実行、Query / Contract Query実行、Observationまでの実行境界（[ISSUE-000006](../issue/000006-application-driver-vertical-slice.md) / ADR-0019 / ADR-0020 / ADR-0021 / ADR-0023） |
| [`DATABASE_CONTRACT_QUERY_V1_DESIGN.md`](DATABASE_CONTRACT_QUERY_V1_DESIGN.md) | Contract Query signature、SQL realization、executor Binding、provider runtimeの四層（[ISSUE-000018](../issue/000018-database-contract-query-execution.md)〜[ISSUE-000021](../issue/000021-oracle-contract-query-sql-runner.md) / ADR-0007 / ADR-0021） |
| [`BUSINESS_LIFECYCLE_V1_DESIGN.md`](BUSINESS_LIFECYCLE_V1_DESIGN.md) | 業務の状態遷移をCanonicalな意味として定義・実行・観測する設計（[ISSUE-000038](../issue/000038-business-lifecycle-semantics.md) / ADR-0024 / ADR-0025） |
| [`DESIGN_SYNTHESIS_CROSS_VIEW_PROOF_DESIGN.md`](DESIGN_SYNTHESIS_CROSS_VIEW_PROOF_DESIGN.md) | RQ-026のcross-view照合をbounded proof obligationとしてDesign Synthesisの候補縮約へ接続する評価待ち設計 |
| [`PROJECT_PLUGIN_COMPOSITION_DESIGN.md`](PROJECT_PLUGIN_COMPOSITION_DESIGN.md) | 案件単位のPlugin / UI Kit / 仕様Template / Architecture Profileの構成境界（[GitHub #1016](https://github.com/hat47x/tei-platform/issues/1016)、評価待ち） |

## 関連Reference / Evaluation

- [`../DESIGN_SYNTHESIS_ORDER_ENTRY_PROOF_REFERENCE_2026-09.md`](../DESIGN_SYNTHESIS_ORDER_ENTRY_PROOF_REFERENCE_2026-09.md) — Cross-view proof設計のP1〜P4を既存`order_entry`へ適用し、`supported / contradiction / unresolved`の実用性とauthority境界を確認するR1評価
- [`../DESIGN_SYNTHESIS_MANUFACTURING_ACTUALITY_HANDOFF_PROOF_REFERENCE_2026-09.md`](../DESIGN_SYNTHESIS_MANUFACTURING_ACTUALITY_HANDOFF_PROOF_REFERENCE_2026-09.md) — Manufacturing / inventory pressureへP5 / P6を適用し、Command / Observation / actuality / Business acceptance / Handoffの非短絡を確認するR2 negative評価
- [`../DESIGN_SYNTHESIS_CONTEXT_PACKET_PROOF_BOUNDARY_2026-09.md`](../DESIGN_SYNTHESIS_CONTEXT_PACKET_PROOF_BOUNDARY_2026-09.md) — R1 / R2のproof結果をbounded `DesignContextPacket`へ渡すときのauthority、candidate contraction、unresolved、LLM残余判断の境界を整理するR3評価
- [`../DESIGN_SYNTHESIS_CANDIDATE_LOCAL_PROOF_PACKET_PRESSURE_2026-09.md`](../DESIGN_SYNTHESIS_CANDIDATE_LOCAL_PROOF_PACKET_PRESSURE_2026-09.md) — candidate-local proof選択、compact elimination certificate、trace-preserving unresolved集約をR1 / R2へ当てるR4 read-only評価
- [`../DESIGN_SYNTHESIS_CANDIDATE_DISCOVERY_PROVENANCE_BOUNDARY_2026-09.md`](../DESIGN_SYNTHESIS_CANDIDATE_DISCOVERY_PROVENANCE_BOUNDARY_2026-09.md) — exact existing asset、Reference Pattern、associative retrieval等のdiscovery経路とcandidate meaning / current qualificationを分離するR5 read-only評価
- [`../DESIGN_SYNTHESIS_SNAPSHOT_INVALIDATION_PRESSURE_2026-09.md`](../DESIGN_SYNTHESIS_SNAPSHOT_INVALIDATION_PRESSURE_2026-09.md) — proofをrequired accepted authorityのfingerprintへ束縛し、selective stale再評価とstale contradictionのcandidate elimination禁止を確認するR6 read-only評価
- [`../DESIGN_SYNTHESIS_SCENARIO_ACCEPTANCE_DRAFT_2026-09.md`](../DESIGN_SYNTHESIS_SCENARIO_ACCEPTANCE_DRAFT_2026-09.md) — Design Synthesis LoopのS1〜S5について、Gate C前後を分けたAcceptance素案、R1〜R6、およびS1 semantic candidate diversity pressureからのEvidence接続を整理するLane D scenario draft
- [`../DESIGN_SYNTHESIS_ORDER_ENTRY_S1_S2_FRONTIER_PRESSURE_2026-09.md`](../DESIGN_SYNTHESIS_ORDER_ENTRY_S1_S2_FRONTIER_PRESSURE_2026-09.md) — current `order_entry` sourceを使い、existing supported candidate / unresolved proposal / contradicted proposalが同時に存在するS1→S2 frontierをPressureし、contractionとwinner selectionを分離するread-only評価
- [`../DESIGN_SYNTHESIS_CONTEXT_PACKET_MIXED_FRONTIER_REFINEMENT_2026-09.md`](../DESIGN_SYNTHESIS_CONTEXT_PACKET_MIXED_FRONTIER_REFINEMENT_2026-09.md) — R3の初期`proposal_origin`表現をR5とS1→S2 frontier Evidenceで補正し、candidate subject / discovery trace / current proof qualificationを分離したままmixed frontierをS3へ渡すR3 refinement
- [`../DESIGN_SYNTHESIS_PROVIDER_FEASIBILITY_SCOPE_PRESSURE_2026-09.md`](../DESIGN_SYNTHESIS_PROVIDER_FEASIBILITY_SCOPE_PRESSURE_2026-09.md) — current PostgreSQL fail-closed sourceを使い、Target / Provider feasibility contradictionがsemantic candidateではなくmatching realization variantへ向くこと、hard provider constraint時だけcomposite candidateへ拡張できることをPressureするS2 refinement
- [`../DESIGN_SYNTHESIS_PROVIDER_FEASIBILITY_OUTCOME_CLASSIFICATION_PRESSURE_2026-09.md`](../DESIGN_SYNTHESIS_PROVIDER_FEASIBILITY_OUTCOME_CLASSIFICATION_PRESSURE_2026-09.md) — generic provider failure surfaceをmissing Evidence / observed incompatibility / structural unsupported / operational recovery / supportedへ分け、S2 contractionとS5 owner routingでreason / owner / scopeを失わないことをPressureするS2-S5 refinement

`DESIGN_SYNTHESIS_CROSS_VIEW_PROOF_DESIGN.md`の評価順は、後続のread-only実績に合わせてR1〜R6へ同期済みである。当初change / stale再評価として想定した論点は、R3〜R5を先に独立評価した後、R6 snapshot invalidationとして扱う。

R3の初期conceptual shapeにある`proposal_origin`はcurrent production field候補ではない。R5およびmixed-frontier refinementを優先し、candidate meaningとdiscovery trace / provenanceを別軸として読む。

## Design Synthesisのpre-Gate-C停止点

Design Synthesis / local cognitionのLane Dでは、Gate C前に安全に進められるread-only設計評価をR1〜R6、S1 / S2 frontier pressure、mixed-frontier context refinement、provider feasibility scope / outcome-classification pressure、S1〜S5 Acceptance素案まで実施した。

ここで得たcurrent boundaryは次である。

```text
candidate discovery
  -> exact / Pattern / associative provenanceを分離

candidate qualification
  -> current accepted authorityへ戻る

cross-view proof
  -> supported / contradiction / unresolvedを分離

candidate contraction
  -> exact scopeのcontradictionだけを決定論的に除外
  -> unresolvedは残余dimensionとして保持
  -> frontierを必ず1 candidateへ縮約しない
  -> survived != supported
  -> supported existing candidate != preferred / winner / accepted candidate

provider feasibility
  -> semantic candidate != realization / binding variant
  -> missing provider Evidenceはsemantic contradictionにしない
  -> provider-specific incompatibilityはmatching realization variantへ閉じる
  -> hard Target / Provider requirement時だけcomposite candidate eliminationへ拡張する
  -> provider limitationからBusiness / Canonical meaningを逆authorしない
  -> same error surfaceをsame resultへ畳まず、reason / owner / scopeへ戻す
  -> missing Evidence / operational recoveryはunresolvedとして保持する
  -> observed incompatibility / structural unsupportedだけをexact realization claimへ縮約する

context packet
  -> candidate subject != discovery trace != candidate-local proof / qualification
  -> R3初期の単一proposal_originへ戻さない
  -> supported existing + unresolved proposalをflattenせずS3 / human reviewへ渡す

proof freshness
  -> required accepted authorityのsemantic fingerprintへ束縛
  -> stale contradictionをcandidate eliminationへ使わない

LLM boundary
  -> bounded residualをProposalとして扱う
  -> minimum semantic deltaはS2 proofではなくS3 / human comparisonで扱う
  -> Acceptance / authorityを所有しない
```

Gate C前には、これらを根拠として次をproduction contractへ先行固定しない。

- `DesignCandidate`のCanonical kind
- production `DesignContextPacket` schema
- candidate originの排他的enum
- persisted proof lifecycle / result enum
- persisted `AcceptanceSnapshot` / proof evaluation stamp
- production Pattern Memory schema
- local / remote LLM provider contract
- vector DB / graph DB / generic solver
- automatic Shared Acceptance
- semantic-delta persistence model
- feedback event ontology
- `ProviderFeasibility` production enum / candidate hierarchy
- provider error / outcome taxonomyのproduction固定

新しいread-only作業を追加するのは、R1〜R6、S1 / S2 frontier pressure、mixed-frontier context refinement、provider feasibility scope / outcome-classification pressureでも説明できない具体的なinformation loss、authority contradiction、provenance / freshness / scope / owner-routing gapがcurrent Referenceで見つかった場合に限る。単に将来必要そうという理由で抽象化やtest-only shapeを増やさない。

### Gate C後のdownstream handoff guard

Gate C後に[`ISSUE-000047`](../issue/000047-knowledge-first-semantic-exploration.md) / [`ISSUE-000051`](../issue/000051-derived-cognitive-retrieval-evaluation.md)を再開するとき、R5 / R6で確認済みの境界を別のschemaや認知層として再設計しない。

`ISSUE-000047`へ渡す既知条件は次である。

- semantic candidateとdiscovery mechanism / discovery traceを分離する
- 同一candidateがexact / Pattern / associative等の複数経路で見つかっても、candidateを複製せず複数traceを保持できる
- Reference Patternのprovenance、associative retrieval representationのbuild-input provenance / scoped freshness trace、current M1 qualificationを別物として扱う
- retrieval score / similarity / 過去利用回数をapplicability / Evidence / Acceptanceへ変換しない
- exact existing assetも「存在する」だけでapplicableとせず、current identity / dependency / scopeとrequired accepted authority / Evidenceへ戻ってqualifyする

`ISSUE-000051`へ渡す既知条件は次である。

- derived cognitive representationはM2のregeneratable resourceであり、M1 authorityではない
- candidate retrieval方式を変えてもsemantic candidate identityやaccepted meaningを変えない
- derived indexのbuild-input provenance / freshness / stale判定と、M1 accepted authority / proof freshnessを混同しない
- derived representationがfreshでも、current applicabilityやcross-view proofはrequired accepted authority / Evidenceへ独立に戻って確認する
- R6のproof freshnessは「proofが依存するaccepted authority fingerprint」の問題であり、cognitive index用のglobal revision counter導入根拠にはしない
- index削除・方式差し替え後もCanonical Worldが変わらないことを維持する

したがって再開時に、次を改めて導入するところから始めない。

```text
candidate.origin = exact | pattern | associative
という排他的origin enum

retrieval score -> confidence / evidence strength
というauthority変換

repository全体のglobal revision counter

cognitive index / pattern memoryを
accepted semantic sourceとする構造
```

actual Gate C Referenceでこれらの境界では情報が失われることが観測された場合にだけreopenする。Gate C後の最初の仕事は境界の再設計ではなく、実在するS1 candidate frontierへ000047のcurrent qualificationを当て、必要性が出たときだけ000051のretrieval方式を比較することである。

### 再開条件

通常の再開点はProduct Gate C後とする。少なくとも次が実在してから、S1〜S5のactual Referenceへ進む。

```text
Gate B Ready
  -> Handover Baseline B / baseline revision R
  -> First Web Target build T1
  -> Gate C Evidence
```

再開後の最初の順序は[`../DESIGN_SYNTHESIS_SCENARIO_ACCEPTANCE_DRAFT_2026-09.md`](../DESIGN_SYNTHESIS_SCENARIO_ACCEPTANCE_DRAFT_2026-09.md)に従う。

```text
S1 realistic intent / candidate frontier
  -> S2 incremental cross-view contraction
  -> actual information-lossを見てbounded packetを調整
  -> S3 local LLM / minimum semantic delta
  -> Shared Acceptanceで小さいsemantic changeを1件Accept
  -> S4 implementation / test derivation
  -> S5 one observable feedbackをmeaning ownerへroute
  -> closed loopを1周
```

このclosed loopのactual Evidenceを得る前に、R1〜R6のtest-only shapeやS1 / S2 frontier pressure / mixed-frontier refinement / provider feasibility pressureのconceptual shapeをproduction schemaへ昇格しない。

## まだ設計文書を置いていない領域

実装とADR、Conformanceは存在しますが、機能単位の設計本文をまだ置いていない領域です。必要になった時点で追加し、それまでは各ADRとConformanceを正本とします。

| 領域 | 現在の正本 |
| --- | --- |
| Source-to-Canonical Traceability / Source Index | [ADR-0004](../adr/0004-source-to-canonical-traceability.md)、[`docs/conformance/source-traceability-m0.23.md`](../../docs/conformance/source-traceability-m0.23.md) |
| Database物理schemaのDesired / Actual / Diff / Unresolved、Advanced Index、Migration rendering | ADR-0008〜ADR-0015、`docs/conformance/`のm0.26〜m0.35 |
| Plugin / Process Runtime、Permission、Resource Broker、fencing / takeover | [ADR-0005](../adr/0005-supervisor-and-runtime-fencing-authority.md)、[ADR-0006](../adr/0006-process-command-and-argv.md)、`docs/conformance/`のm0.22.1 / m0.24 / m0.25 |
| Target / Presentation境界 | Gate C前のため未確定。[Gate C Evidence契約](../GATE_C_EVIDENCE_CONTRACT.md)と[ISSUE-000026](../issue/000026-target-capability-execution-profile-boundary.md) |

Gate前に確定していない領域へ、先回りして設計文書を作りません。
