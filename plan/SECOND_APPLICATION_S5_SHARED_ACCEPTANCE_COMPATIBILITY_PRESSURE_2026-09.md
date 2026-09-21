# Second Application Reference — S5 Shared Acceptance Compatibility Pressure

- Status: Current Shared Acceptance coverage gap identified / compatibility Preview not yet executable
- Date: 2026-09-21
- Primary workstream: ISSUE-000040
- Shared boundary owner: ISSUE-000027
- Revised S4 source Proposal SHA-256: `c18ae0fdef8e01aa965e2ad84c86ecfc21ca0608c07b1808e73368a5f434458d`
- Current deterministic Shared Acceptance implementation: `tools/reference-shared-acceptance`
- Authoritative destination source created: No
- Acceptance attempted: No

## 1. Why this pressure exists

2026-09-21、`tools/reference-shared-acceptance` に決定論的なShared Acceptance Boundaryの最初のincrementがmainへ入った。

この実装は重要な前進だが、現在のReference candidateは `add-validation` だけを扱い、source containment / semantic diff / architecture-policyもsemantic層内の一変更を前提にしている。

Second Applicationのrevised S4 Proposalは次のthree-source compositionである。

```text
semantic/repair-intake/intake.tei
applications/repair-intake/application.tei
applications/repair-intake/patterns/customer-reference.tei
```

したがって、既存boundaryを迂回して直接authoritative sourceを書くのではなく、Second Applicationを次の実測pressureとして使い、Shared Acceptance Boundaryがmulti-layer compositionを安全に扱える最小incrementを要求する。

## 2. Current boundary and exact incompatibilities

current implementationをsource inspectionした結果、少なくとも次の3点がSecond Applicationをそのまま受け付けない。

### 2.1 Candidate operation coverage

`tools/reference-shared-acceptance/candidate.go` は現在:

```text
supported operation = add-validation
```

のみである。

Second Applicationでは新規namespace / Data / Attribute / Relationに加え、新規Application / datasource requirement / Application Patternを同一candidateで扱う必要がある。

### 2.2 Semantic diff coverage

`tools/internal/changeanalysis.Diff` は `World.Semantics` だけをflattenし、namespace / type / data / attribute / relation / capability / validationのsubject diffを返す。

Second ApplicationではApplicationとApplication Patternもauthoritative meaningとして増えるため、semantic-only diffでは「宣言したmulti-layer deltaと実際のWorld差分が一致した」ことを証明できない。

### 2.3 Architecture-policy / changed collections

current `checkExpectedDelta` はchanged collectionがexactly `semantics` 1件であることを要求する。

Second Applicationでは正当な変更として少なくとも:

```text
semantics
applications
```

が変わる。

このためcurrent policyを単純に緩めるのではなく、candidateが宣言したtyped layerだけが変化したことをfail-closedで検査する必要がある。

## 3. What must not be introduced

このpressureを理由に次を先行実装しない。

- 第二Application専用の別Acceptance engine
- AI専用 / Studio専用のwrite path
- arbitrary text patch / generic Patch DSL
- whole-model regeneration
- Environment bindingの暗黙copy
- Runtime authority grant
- inheritance / override / auto-sync protocol
- shared Customer Semantic / Contractの書換え

Acceptance Boundaryは一つのまま維持する。

## 4. Minimum next increment

exact candidate schemaはcode pressureから決め、ここでは固定しない。ただしbehaviorとして次を必須とする。

1. revised S4 three-source Proposalを、meaning-owning layerを保持したtyped multi-layer deltaへ変換できる。
2. `baseModelRevision` によりstale candidateをfail closedできる。
3. overlay上で3 source unitをatomicに適用し、`workspace.Load` と通常の `validation.Validate` を通す。
4. source containmentは宣言された3 unitだけを許し、Persistence / Environment / Target等の無関係source変更を拒否する。
5. Canonical World差分はSemantic subjectsとApplication-owned subjectsの双方を説明できる。
6. changed collectionはcandidateが宣言したlayer集合とexact一致し、余分なcollection変更を拒否する。
7. Preview / Rejectはauthoritative sourceをbyte単位で変更しない。
8. Acceptは全検査を再実行し、post-write reload / Validation / declared diff不一致時にはrollbackする。
9. origin（file / CLI / Studio / AI）によって同じmeaning deltaの判定を変えない。
10. AcceptanceはEnvironment bindingやeffect-time authorityを付与しない。

## 5. Proposal identity and Acceptance identity

二つのidentityを混同しない。

```text
S4 source Proposal fingerprint
  c18ae0fdef8e01aa965e2ad84c86ecfc21ca0608c07b1808e73368a5f434458d

Shared Acceptance candidate fingerprint
  = current baseModelRevision + normalized multi-layer delta
  = not yet obtained
```

S4 fingerprintはreview対象となったthree-source payloadのexact identityである。

Shared Acceptance candidate fingerprintはcurrent model revisionを含むため、base revisionが変われば変わり得る。

したがってcompatibility adapter / pressureは:

- input three-source payloadからS4 fingerprintを再計算して `c18ae...` と一致させる
- 同じpayloadからnormalized multi-layer deltaを生成する
- current model revisionと合わせてAcceptance candidate fingerprintを得る
- Preview Evidence上で `S4 proposal fingerprint -> Acceptance candidate fingerprint -> baseModelRevision` のtraceを失わない

ことを要求する。

このtraceはauthoritative business meaningではなく、Acceptance Evidenceである。

## 6. Required Preview pressure

human Acceptanceの前に、sourceを書かないPreviewで少なくとも次を確認する。

```text
decode / normalize                 PASS
base revision                      PASS
typed composition resolve          PASS
overlay apply                      PASS
reload                             PASS
full Validation                    PASS
architecture / containment         PASS
multi-layer diff                   PASS
impact / unresolved reporting      PASS
acceptance policy                  PASS
sourceChanged                      false
environment binding mutation       0
runtime authority grant            0
```

さらにnegative caseとして:

- stale base revision
- existing destination source collision
- undeclared fourth source edit
- Semanticだけ追加してApplication / Patternが欠落
- Applicationだけ追加してRelationが存在しない
- unknown Relation
- Contract Query cardinality mismatch
- undeclared World collection change

をfail closedする。

## 7. Revised pre-Acceptance gates

Second Application S6 authoritative authoringへ進む前の停止条件を、次の3つへ分離する。

```text
Gate A:
  revised three-source executable pressure
  focused tests + reference runner + repository health PASS

Gate B:
  Shared Acceptance compatibility
  exact c18ae... Proposal -> current-base candidate conversion
  read-only Preview PASS

Gate C:
  explicit human Acceptance decision
  exact reviewed Proposalに対するaccept / reject
```

Gate AのPASSをGate B / Cへ読み替えない。
Gate BのPreviewをhuman Acceptanceへ読み替えない。
通常の「進めてください」をGate Cへ読み替えない。

## 8. S6 entry

Gate A / B / Cがすべて成立した後だけ、同じShared Acceptance Boundaryの`accept`経路でauthoritative sourceへ反映する。

反映対象はS4で定義済みの3 source unitに限定し、その後に:

```text
reload
  -> full Validation
  -> exact accepted diff
  -> destination source Evidence
```

を取得する。

Environment bindingはその後の別Proposalであり、このAcceptanceへ混ぜない。

## 9. Implementation packet

次incrementの具体的なcode pressureは:

- `plan/SECOND_APPLICATION_S5_SHARED_ACCEPTANCE_IMPLEMENTATION_PACKET_2026-09.md`

に分離する。
