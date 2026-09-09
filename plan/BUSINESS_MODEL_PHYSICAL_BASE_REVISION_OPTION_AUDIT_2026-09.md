# Business Model Physical Base Revision Option Audit — 2026-09

Status: **Lane C / Shared Acceptance physical-base option audit / no revision API or token schema adoption**

## 1. 目的

Lane Cではproduction Acceptanceについて:

```text
Business source freshness
  != Business semantic revision
  != cross-view Conformance base
```

を固定している。

一方、`ISSUE-000027` / `ISSUE-000046`はcandidateが必要に応じてopaqueな`baseModelRevision`を持ち、staleなら黙ってlatestへ適用しない原則を持つ。

残る問いは:

> Business source baseとWorld / Conformance baseを、productionで物理的に一つのtokenへ束ねてよいか。

本書では現行TEIが単一Git repository / file-native workspaceを使う事実をPressureにしつつ、Lane C独自のrevision type、Git API、hash algorithm、transaction engineを採用しない。

## 2. Correctness criterion

物理tokenの第一目的はsemantic equalityではなく**stale rejection**である。

したがって優先順位は:

```text
false-negative stale
  = unacceptable

false-positive stale
  = conservative but acceptable
```

とする。

つまり:

- relevant Business sourceが変わったのにfreshと判定する -> 不可
- relevant World sourceが変わったのにfreshと判定する -> 不可
- unrelated docだけ変わってstale扱いする -> 精度は低いがcorrectnessは壊さない

である。

## 3. Option R0 — Git HEAD commit SHAだけをbaseModelRevisionにする

Conceptual:

```text
baseModelRevision = git HEAD
```

### 利点

- 単一repository revisionでBusiness / World source familyのcommit状態を一括固定できる。
- opaque tokenとして扱いやすい。
- unrelated changeでもstaleになるため保守的。
- Lane C専用revision storeを作らなくてよい。

### Critical problem

Git HEADはworking treeの未commit変更を表さない。

```text
HEAD = R

Preview
  Business / World source bytes at R

then
  authoritative file edited in working tree
  HEAD still R

Accept checks only HEAD
  -> falsely fresh
```

となり得る。

したがって:

```text
Git HEAD equality alone
  != complete source freshness proof
```

である。

**判定: reject as unconditional sufficient token.**

## 4. Option R1 — repository revision + clean authoritative-base invariant

Conceptual:

```text
baseModelRevision = repository revision R

precondition:
  all authoritative Business + World sources at Preview are exactly represented by R
  no relevant dirty/untracked authoritative source state
```

Accept時に:

```text
current authoritative base still exactly R
```

を確認する。

### 利点

- Business / Worldが同一repository revisionで管理される現在形では、一つのtokenで両source familyをconservativeにcoverできる。
- unrelated commitもstaleになるがfalse-negativeを避けやすい。
- Lane C固有のBusinessRevisionを作らない。

### 制約

- shared Acceptanceが「clean authoritative base」を定義・検証する必要がある。
- candidate自体のstaging changesをdirty判定へ混ぜない境界が必要。
- untracked sourceがauthoritative discoveryへ入るなら、それもRだけではcoverできない。
- Git以外のconnected / generated / external authoritative sourceへ一般化できるとは限らない。

**判定: current single-repository configurationではviable conservative option。ただしLane Cから採用決定しない。**

## 5. Option R2 — scoped Business source snapshot + scoped World source snapshot

Conceptual:

```text
BusinessBase
  = relevant logical Business source set + content state

WorldBase
  = relevant workspace source / normalized World source state
```

### 利点

- unrelated repository changesでstaleにならない。
- BusinessとWorldが独立更新されても明示的に検出できる。
- dirty working treeもcontent stateを含めれば扱える。

### 問題

- relevant source closureを安全に定義する必要がある。
- source addition / removalをsnapshotに含めないとfalse-negativeになる。
- Business / Worldそれぞれのsnapshot machineryが増える。
- Lane Cからgeneric source-version subsystemを先行設計する危険がある。

**判定: precision option / defer to shared Acceptance implementation pressure.**

## 6. Option R3 — repository envelope + scoped source fingerprints

Conceptual:

```text
baseModelRevision
  = repository-level base R

plus internal Preview evidence
  = Business source snapshot
  + World / Conformance base snapshot
```

### 利点

- repository revisionで大きなstale envelopeを作る。
- scoped fingerprintsでdirty / source-specific changesを補強できる。
- semantic revisionとは分離したまま運用できる。

### 問題

- tokenが物理的に一つか複数かという問いを実質内部構造へ移す。
- current Gate Eのためだけにcomposite revision contractを固定する必要はない。

**判定: plausible implementation direction / not adopted.**

## 7. Single opaque tokenを許せる条件

Lane Cはproduction Shared Acceptanceが一つのopaque `baseModelRevision`を使うこと自体を禁止しない。

ただしそのtokenは概念上、Preview時点の:

```text
A. authoritative Business source base
B. authoritative World / cross-view source base
```

の両方について、Accept時にstaleをfalse-negativeなく検出できなければならない。

したがって:

```text
one physical token
  = allowed

one conceptual freshness condition
  = not allowed
```

である。

実装内部で一つのGit-backed snapshot tokenに束ねても:

```text
Business source changed?
World / Conformance base changed?
```

という二つのcorrectness obligationは消えない。

## 8. Business semantic revisionはtoken代替ではない

Business semantic revisionはnormalized meaningの比較である。

例えば:

```text
source formatting only change
  -> physical source base changed
  -> Business semantic meaning unchanged
```

があり得る。

このとき:

```text
same Business semantic revision
  -X-> source fresh
```

とは言えない。

逆にrepository revisionは:

```text
unrelated docs change
  -> repo revision changed
  -> Business semantic meaning unchanged
```

となり得る。

したがって:

```text
repository/source freshness token
  != Business semantic revision
```

を維持する。

## 9. Conformance resultもtoken代替ではない

Preview時にcross-view PASSだったことだけではAccept時のcurrent Worldを保証しない。

```text
Preview PASS against World W1

World changes to W2
Business source unchanged

old PASS
  != Conformance against W2
```

したがってone repository tokenを使う場合も:

- token freshならW1=Wcurrentを物理的に保証できること
- 保証できない場合はAccept直前にcurrent Worldへrevalidationすること

のどちらかが必要。

## 10. Dirty worktree / candidate staging boundary

File-native Shared Acceptanceではcandidateを作るためのtemporary changesとauthoritative baseを区別しなければならない。

避けるべき形:

```text
candidateをauthoritative fileへ直接書く
  -> worktree dirty
  -> Preview
  -> Accept判定
```

これはPreview時点でauthorityを既にmutationしており、Reject semanticsを壊す。

Current preferred shape:

```text
authoritative base
  -> read

candidate
  -> separate object / temp source set / non-authoritative staging

Preview
  -> no authoritative mutation

Accept
  -> atomic apply
```

したがってR1の`clean authoritative-base invariant`は、candidate staging areaまでcleanであることを要求するものではない。

要求するのは:

```text
candidate staging
  != authoritative discovery
```

である。

## 11. Workspace discoveryとの接続

`BUSINESS_AUTHORING_WORKSPACE_DISCOVERY_BOUNDARY_PRESSURE_2026-09.md`では、workspace manifest source-root membershipをphysical authoritative discovery候補としてPressureした。

この方向ではbase freshnessがcoverすべきsource setは少なくとも:

```text
declared authoritative Business source root state
+ current World source roots / relevant workspace source state
+ discovery membership itself
```

を含む。

特にsource fileだけでなく:

```text
source added
source removed
source-root membership changed
```

もstaleを発生させ得る。

個別file fingerprintだけ見てsource-set membership changeを見逃してはいけない。

## 12. Git commitをsemantic authorityにしない

Single-repository R1を採る場合でも:

```text
Git commit
  -> physical source snapshot envelope
```

として利用するだけである。

次にはしない。

- Business semantic revision
- Business Procedure identity
- Approval identity
- Conformance PASS identity
- Source Index subject identity

Git historyはexisting version controlとして利用できるが、Business meaningの正本そのものではない。

## 13. Independent versioning reopen condition

将来:

```text
Business authoring source repository A
World / Application source repository B
```

またはconnected external authoring等へ分かれた場合、一つのrepository SHAでは両者をcoverできない。

その時は少なくとも:

```text
Business source base check
+ World / Conformance base check
```

を独立に成立させる必要がある。

物理tokenを二つにするかcomposite one-tokenにするかはshared Acceptance実装が決める。

## 14. Current option ranking

```text
R1 repository revision + clean authoritative-base invariant
  -> current single-repository configurationでviable conservative option
  -> shared Acceptance implementation candidate, not Lane C adoption

R3 repository envelope + scoped snapshots
  -> plausible if dirty/source-specific precision is needed
  -> defer

R2 separate scoped Business + World snapshots
  -> precise but more machinery
  -> defer until concrete need

R0 Git HEAD only, no clean/snapshot invariant
  -> insufficient / reject
```

## 15. Gate E implication

Gate Eの問い:

> semantic revision identity / source freshness / Conformance baseを物理的に一つのrevision tokenへ束ねるか分離するか

は、Lane Cとして次まで狭められる。

```text
physical one-token
  -> allowed if complete conservative stale envelope

physical multiple tokens
  -> allowed

Business semantic revisionとの同一化
  -> reject

Git HEAD-only freshness proof
  -> reject unless authoritative base is known clean/exactly represented

false-positive stale
  -> acceptable tradeoff

false-negative stale
  -> reject
```

したがってLane CがADR前にplatform-wide token shapeを決める必要はない。

Production Shared Acceptance prototypeでcomplete stale envelopeを実証すればよい。

## 16. Non-adoptions

本auditから次を採用しない。

- Git SHAをplatform-wide `baseModelRevision` formatへ固定
- `BusinessRevision` production type
- Business専用revision store
- Business source fingerprint algorithm
- composite ETag schema
- mandatory clean Git repository as product-wide rule
- Git commit = Business semantic revision
- Git commit = Conformance result
- cross-repository revision protocol

## 17. Current conclusion

```text
Conceptual identities
  = three-way separation maintained

One physical base token
  = acceptable when it covers Business + World source state completely

Git HEAD alone
  = not sufficient in dirty/untracked authoritative-source cases

Current single-repository deployment
  = repository revision + exact/clean authoritative-base invariant is viable conservative option

Future independently versioned source planes
  = require separate detectable stale conditions

Lane C revision API
  = not adopted

Gate E token-shape question
  = no longer needs Lane C-specific pre-ADR design decision

Pre-ADR
  = still Not Ready for actual execution / production cutover / second realistic Handoff reasons
```
