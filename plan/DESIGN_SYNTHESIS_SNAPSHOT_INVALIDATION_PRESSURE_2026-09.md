# Design Synthesis Snapshot Invalidation Pressure — 2026-09

Status: **Lane D / R6 read-only evaluation / no production snapshot schema**

## 1. この評価の位置づけ

`plan/design/DESIGN_SYNTHESIS_CROSS_VIEW_PROOF_DESIGN.md`では、cross-view proofを**evaluated snapshotへ束縛し、accepted authority変更後にstaleなproofを再利用しない**ことを設計仮説として置いている。

同文書の初期評価計画では、change / stale再評価を`R4`としていた。その後、実際のLane Dでは次の評価が間に入った。

```text
R1  order_entry P1〜P4
R2  manufacturing / actuality / Handoff P5〜P6
R3  Context Packet / proof boundary
R4  candidate-local proof selection / contraction
R5  candidate discovery / provenance
```

既にmerge済みのR3〜R5を後からrenumberしない。本書では、初期計画に残っていたchange / stale再評価を**R6**として実施する。

このR6は新しいProduct Gate、Requirement、Canonical kind、production `AcceptanceSnapshot` schemaを作るものではない。

## 2. 見つかった具体的なgap

R1〜R5により、次はPressure済みである。

- `supported / contradiction / unresolved`の意味分離
- candidate scopeに関係するproofだけを選ぶ
- exact contradictionだけでcandidateを除外する
- unresolvedを残す
- discovery provenanceとcurrent qualificationを分ける
- stale derived retrievalをcurrent candidate discoveryに使わない

一方、test-only candidate contractionでは、選ばれたproof result自体が**どのaccepted authority集合に対して評価されたか**を機械的に照合していない。

このままでは概念上、次が起こり得る。

```text
snapshot S1
  P3 contradiction
        ↓
accepted authority change
        ↓
snapshot S2
  old P3 contradictionをそのまま再利用
        ↓
candidateを誤ってeliminate
```

staleな`contradiction`は、安全なdeterministic elimination reasonではない。

同様に、staleな`supported`をcurrent supportとして扱ってはならず、staleな`unresolved`もcurrent missing-authority状態を証明しない。

## 3. R6で確認する問い

### Q1 — selective invalidation

proofが依存しているaccepted authorityだけが変わったとき、そのproofをstaleにできるか。

### Q2 — unrelated change

proofが依存していないauthority変更で、無関係なproofまで一律staleにしないか。

### Q3 — locator-only change

source file move等でlocatorだけが変わりsemantic fingerprintが不変なら、無条件にproofをstaleへしないか。

### Q4 — missing / ambiguous authority

current snapshotでrequired authorityがexact-oneに解決できない場合、old proofをfreshとみなさずfail closedできるか。

### Q5 — contraction safety

stale proofをcandidate-local contractionへ渡さず、特にstale contradictionによるeliminationを防げるか。

## 4. global repository revisionをfreshness条件にしない

最も単純な実装は、proof resultへrepository commit SHAを付け、SHAが変われば全proofをstaleにする方法である。

R6ではこれをdefaultにしない。

```text
repository revision changed
  != every accepted meaning changed
```

文書修正、無関係なReference追加、別domainのSemantic変更だけで、すべてのproofを再評価する必要はない。

必要なのはproof questionに関係するauthorityのchange detectionである。

概念的には次の程度で足りる可能性がある。

```text
Proof evaluation stamp
  proof id
  required authority keys[]
    authority key
    semantic fingerprint at evaluation
```

current snapshot側では:

```text
authority key
  exact resolution count
  semantic fingerprint
  locator, if useful
```

を照合する。

これはproduction schema案ではない。test-onlyでinformation lossがあるかを確認するためのconceptual shapeである。

## 5. fingerprintとlocatorを分ける

TEIは既にfile pathとsemantic identityを分離する方向を持つ。

R6でも:

```text
locator changed
semantic fingerprint unchanged
  -> freshnessを失わせる根拠にはしない
```

をPressureする。

逆にlocatorが同じでもaccepted meaningが変わりfingerprintが変化したなら、関連proofはstaleである。

したがって:

```text
path equality != semantic freshness
```

を維持する。

## 6. proof dependencyはfamily全体ではなくquestion scopeへ閉じる

同じproof familyでも必要authorityは異なり得る。

例えばR1の代表例では概念上:

### P3-07 — submit mutation authority

```text
Business / Application realization context
Capability identity / semantics
Database Contract write authority
accepted attribution edge, if any
```

の変更がresultへ影響し得る。

Legacy Authorization policyだけが変わっても、P3-07 mutation resultを無条件にstaleにする必要はない。

### P4-06b — Business protected target

```text
Business source
Security policy corroboration
relevant target identity context
```

が影響し得る。

Database Contract write authorityだけが変わっても、P4-06bを無条件にstaleへしない。

R6では「P3だからこのsource set」というfamily-global dependency tableをproduction化しない。まずproof question単位で必要authorityをtest-onlyに明示する。

## 7. freshness判定

conceptually、evaluated proofをfreshと呼べる最低条件は次である。

```text
for each required authority:
  current exact resolution count == 1
  && current semantic fingerprint == evaluated fingerprint
```

一つでも満たさなければ、そのproof resultはcurrent snapshotのresultとして使わない。

### changed

fingerprintが変わったauthorityをstale reasonとして示せること。

### missing

current resolution count = 0ならstale / re-evaluation requiredである。

### ambiguous

current resolution count > 1でもfresh扱いしない。

### dependency set自体が不足している場合

「依存authorityが少ないほどfreshになりやすい」ことを成功指標にしない。

Referenceで、あるsource変更がproof resultへ影響したのにdependency setがそれを捉えられないcaseが見つかった場合、そのdependency記述が不十分である。

## 8. candidate contractionとの接続

candidate-local contractionへ渡せるのはcurrent snapshotでfreshと確認できたproofだけとする。

```text
all proof results
      ↓ freshness filter
current proof results
      ↓ candidate-local selection
supported / contradiction / unresolved
      ↓
contraction
```

### stale contradiction

```text
old contradiction
  -X-> candidate elimination
```

required proofがstaleでcurrent resultが無いなら、R4の`required proof missing -> fail closed`へ接続する。

つまり:

```text
stale contradiction
  -> remove from current proof set
  -> required proof coverage unresolved
  != candidate contradiction
```

### stale supported

current supportとして数えない。

### stale unresolved

old missing-authority状態をcurrent unresolvedとしてそのまま表示しない。再評価後のcurrent unresolvedを作る。

## 9. selective invalidation matrix

R6でPressureする代表case:

| Change | P3 mutation proof | P4 security-boundary proof |
| --- | --- | --- |
| relevant Database Contract fingerprint change | stale | unaffected if not dependency |
| relevant Security policy fingerprint change | unaffected if not dependency | stale |
| unrelated Customer Semantic change | fresh | fresh |
| source locator only changed, semantic fingerprint same | fresh | fresh |
| required authority missing | stale / fail closed | stale / fail closed when required |
| required authority ambiguous | stale / fail closed | stale / fail closed when required |

この表はproduction dependency catalogではなく、selective invalidationの意味Pressureである。

## 10. Acceptance Snapshotに持たせない意味

R6のsnapshot / stampはderived evaluation metadataである。

次を持たせない。

- Candidate Acceptance
- proof confidence
- AI score
- semantic authorityそのもの
- write-back permission
- global completeness
- source of truth marker

fingerprintが一致することは「proof resultがcurrent inputに対して再利用可能」というfreshness条件であり、proofの正しさやcandidate Acceptanceを新たに証明するものではない。

## 11. selective reuseと再評価

R6の目的はcache optimizationではない。

再評価costが十分小さければ、すべて再実行してもよい。

重要なのは、将来DesignContextPacketやinteractive Studioでcached proofを扱う場合に:

```text
staleをfreshと誤認しない
+
unrelated changeをsemantic changeへ過大解釈しない
```

ことである。

したがって、R6からincremental build system、dependency database、event bus等を導入しない。

## 12. test-only Pressure

`reference-business-procedure-check`へ、production typeを増やさずtest-onlyで次を確認する。

1. proof stampはrequired authority fingerprintへ束縛される。
2. relevant authority fingerprint変更で対象proofだけstaleになる。
3. unrelated authority変更ではfreshを保てる。
4. locator-only変更ではsemantic fingerprintが同じならfreshを保てる。
5. missing / ambiguous required authorityはfail closedする。
6. stale contradictionはcandidate eliminationへ到達しない。
7. unchanged fresh contradictionは従来どおりcandidate-local eliminationへ使える。

## 13. R1〜R6の読み方

Lane Dのcurrent evaluation sequenceは次として読む。

```text
R1  proof family P1〜P4 / order_entry
R2  P5〜P6 / actuality / Handoff negative boundary
R3  Context Packet / proof handoff boundary
R4  candidate-local proof selection / contraction
R5  candidate discovery / provenance
R6  accepted snapshot / selective stale re-evaluation
```

`DESIGN_SYNTHESIS_CROSS_VIEW_PROOF_DESIGN.md`に残る初期計画上の`R4 — change / stale再評価`は、このR6で扱う。merge済みEvaluation文書のR番号は後から振り直さない。

## 14. R6でproduction化しないもの

- global revision counter
- persisted `AcceptanceSnapshot` Canonical kind
- persisted proof evaluation stamp
- generic dependency database
- proof invalidation event stream
- incremental proof engine
- cache server
- graph database
- durable Claim identity

まずtest-only selective invalidationで意味境界だけ確認する。

## 15. R6後の判断

R6が成立すれば、pre-Gate-Cのcross-view proofについて少なくとも次の閉包が得られる。

```text
proof question
  -> accepted authority
  -> result
  -> candidate-local contraction
  -> current snapshot freshness
```

これにより、Gate C後のactual Design Synthesisでcached / reused proof結果を扱う際にも、stale contradictionでcandidateを落とす危険を避ける設計根拠ができる。

それ以上のproduction snapshot / invalidation機構は、actual S1 / S2 Referenceで再評価costやinformation lossが観測されてから決める。
