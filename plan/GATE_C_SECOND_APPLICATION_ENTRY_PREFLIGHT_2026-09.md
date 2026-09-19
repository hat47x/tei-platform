# Gate C → Second Application Entry Preflight

- Status: Prepared / test-only
- Date: 2026-09-19
- Parent: \`plan/GATE_C_SECOND_APPLICATION_START_PACKET_2026-09.md\`
- Related:
  - \`plan/GATE_C_EVIDENCE_CONTRACT.md\`
  - \`plan/GATE_B_TO_C_HANDOVER_BASELINE_B.md\`
  - \`plan/V1_REFERENCE_APPLICATION_PLAN.md\`
  - \`plan/MASTER_PLAN.md\`
- Canonical Application impact: None
- Product Gate impact: None

## 1. 目的

Gate C Ready後にsecond Application workstreamへ入る際、

- Gate Cのstatus文字列だけ
- Visual Evidenceだけ
- baseline revision不明
- Web Target build revision不明
- Target差分未整理

の状態で誤ってApplication authoringを開始しないよう、Start Packetのentry条件をtest-onlyにPressureする。

このPreflightはGate Cを判定する正本ではない。

\`\`\`text
Gate C authority
  = Gate C Evidence / readiness decision

Entry Preflight
  = already accepted Gate C Evidenceが
    second Application開始に必要な形で揃っているか確認
\`\`\`

である。

## 2. Authority boundary

Preflightは次を導出しない。

- Gate C Readyそのもの
- Semantic Conformance合否
- Functional Evidence合否
- Visual Evidence合否
- first Web TargetのTarget差分分類
- second ApplicationのAcceptance
- Canonical Application ID

Gate C statusは外部authority inputとして受け取る。

\`\`\`text
gateCStatus = ready
  != preflight automatically complete
\`\`\`

である。

## 3. 必須entry inputs

最低限次を要求する。

### Gate C decision

\`\`\`text
gateCStatus = ready
\`\`\`

### Handover baseline

- Handover Baseline ID
- baseline Repository revision \`R\`

current first baseline:

\`\`\`text
B
R = e5231839a5dcc62cce1181a2731fcb7c70d16736
\`\`\`

将来successor baselineを使う場合も、IDとrevisionを明示する。

### First Target build

- Target build revision \`T1\`

baseline \`R\`とTarget build \`T1\`を混同しない。

### Evidence classes

- Semantic Conformance Evidence ref
- Interaction / Functional Evidence ref
- Visual Fidelity Evidence ref

Visualだけで他二つを代替しない。

### Target differences

第一Web Targetで観測した差を明示記録する。

少なくとも、

- Target Capability候補
- Runtime Binding差
- execution artifact / ABI / Host差（観測可能な範囲）
- Execution / Trust差
- Web固有実装
- Canonicalへ持ち上げない差
- unresolved

のどこに何が存在したかを後続workstreamから参照できるようにする。

「差がなかった」場合でも、その確認結果をexplicit recordとして残す。

## 4. Preflight result

test-only qualificationは次の二状態のみとする。

\`\`\`text
entry-evidence-complete
blocked
\`\`\`

これはProduct authorityではない。

### entry-evidence-complete

必要entry Evidenceが揃っている。

意味:

> S0 Gate C entry captureを完了できる入力形がある。

意味しないもの:

- second Application sourceを作ってよいという自動Acceptance
- Application IDが決まった
- candidate qualificationが終わった
- Environment Bindingを作ってよい
- Shared Acceptance済み

### blocked

必要entry inputが不足している。

不足項目を明示する。

## 5. Current repository state

2026-09-19時点:

- Gate B = Ready
- Handover Baseline = \`B\`
- \`R = e5231839a5dcc62cce1181a2731fcb7c70d16736\`
- First Web Target build \`T1\` = 未成立
- Gate C = Not Ready

したがってcurrent stateはPreflight上も\`blocked\`である。

これはGate B未成立によるblockではない。

\`\`\`text
Gate B / Handover ready
  ↓
First Web Target / Gate C Evidence pending
  ↓
second Application entry blocked
\`\`\`

である。

## 6. Negative pressure

### A. Visual only

\`\`\`text
gateCStatus = ready
visualEvidence = exists
semanticEvidence = missing
functionalEvidence = missing
\`\`\`

結果:

\`\`\`text
blocked
missing:
  semantic-evidence
  functional-evidence
\`\`\`

Visual evaluator / screenshot scoreだけでentryを開かない。

### B. Ready label only

\`\`\`text
gateCStatus = ready
B / R = missing
T1 = missing
\`\`\`

結果:

\`\`\`text
blocked
\`\`\`

Gate Cの文字列だけをsecond Application開始authorityにしない。

### C. Target difference recordなし

Evidence三種が揃ってもTarget差分が記録されていなければblockする。

理由:

second Application / knowledge laneが第一Target固有差をCanonical meaningと誤認する危険があるため。

### D. Current pre-target state

\`\`\`text
B / R = known
Gate C = not-ready
T1 = missing
Gate C Evidence = missing
\`\`\`

は明示的にblocked。

## 7. Test-only source

- \`tools/reference-business-procedure-check/gate_c_second_application_entry_preflight_pressure_test.go\`

確認すること:

1. completeなsynthetic entryは\`entry-evidence-complete\`
2. current pre-target形はblocked
3. Visual-onlyはblocked
4. \`ready\` labelだけではblocked
5. Target difference recordなしはblocked
6. qualification typeへauthoring shortcutを追加しない

## 8. Prohibited shortcuts

test-only qualificationへ次を置かない。

- \`AutoStart\`
- \`CreateApplication\`
- \`Accepted\`
- \`Approved\`
- \`CanonicalApplicationCreated\`
- \`SourceAuthoringAttempted\`
- \`EnvironmentBindingCreated\`
- \`TargetImplementationStarted\`

Preflightはentry evidenceの完全性だけを扱う。

## 9. Executable Evidence runner

portable runner:

- \`tools/run-knowledge-portability-reference-evidence.sh\`

対象prefixへ、

- \`TestGateCSecondApplication\`

を追加する。

GitHub-hosted runner availabilityが回復した場合は、

- \`.github/workflows/knowledge-portability-reference-evidence.yml\`

から同じscriptを実行する。

successful runが得られるまでExecutable Evidence PASSとは扱わない。

## 10. Master Plan同期

\`MASTER_PLAN.md\`はcurrent Evidenceに合わせ、

- Gate B Ready
- Handover Baseline \`B\` / \`R\`固定済み
- Gate C Not Ready
- First Web Target \`T1\`未成立

を一貫して示す。

上位plan同士のstatus不整合をsecond Application開始条件へ持ち込まない。

## 11. Gate C Ready後

Gate C Readyが正式に成立した場合:

1. Gate C Evidence正本を確認
2. Preflight inputへ\`B/R/T1\`と三Evidence、Target差分を記録
3. \`entry-evidence-complete\`を確認
4. \`GATE_C_SECOND_APPLICATION_START_PACKET_2026-09.md\`のS0を完了
5. 同じprimary ISSUE-000040 workstream branchでS1へ進む

Preflight PASSだけでS4以降へ飛ばない。

## 12. 完了条件

- [ ] Gate C authorityとentry completenessを分離した
- [ ] B / R / T1を別identityとして要求した
- [ ] Semantic / Functional / Visualを別Evidenceとして要求した
- [ ] Visual-onlyを拒否した
- [ ] Target差分recordを必須にした
- [ ] current pre-target stateをblockedとして説明できる
- [ ] authoring shortcutをqualificationへ持ち込んでいない
- [ ] portable runnerへ接続した
- [ ] Master PlanのGate B stale記述を修正した
