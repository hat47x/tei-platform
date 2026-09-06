# Business Procedure File-native Source Pressure — 2026-09

Status: Evidence in progress

## 目的

Business ProcedureをReference fixtureのまま閉じず、file-nativeなauthoring sourceとして扱った場合に、現在のWorkspace / Source Index / Shared Acceptance境界でどこまで成立するかを確認する。

この実験ではproductionの`reference/workspace`正本を変更しない。

`tools/reference-business-procedure-check/accepted_source_pressure_test.go`が、test用temp Workspaceへ既存Reference Workspaceを複製して圧力を与える。

## 1. Workspace source rootの既存余白

現行`workspace.Load()`はmanifestの既知rootを各loaderへ渡す一方、未知rootをforward-compatible layoutとして受理する。

```text
known root
  -> loader
  -> normalized ir.World

unknown / future root
  -> accepted in manifest
  -> ManagedFilesへscan
  -> no automatic World expansion
```

したがってtemp Workspaceでは次を追加できる。

```yaml
sources:
  ...
  - procedures
```

`procedures/order-handling.json`はManagedFileになるが、それだけでは`ir.World`へProcedure objectを追加しない。

### Pressure result

元のReference Workspaceと`procedures` root追加後のWorkspaceをloadし、`ir.World`が一致することをtestする。

これはfile-native source layoutを試すためにCanonical Procedureを先に導入する必要がないことを示す。

## 2. file-native Business Procedure source

実験sourceには現在のReference authoring formをそのまま使う。

```text
artifactRole = reference-authoring-fixture
canonical = false
procedure.identityStatus = reference-local-stable
```

新しいproduction syntaxをこの段階で増やさない。

sourceはstrict JSON decoderで読み、未知field / unsupported schemaVersion / wrong artifact roleをfail closedにする。

## 3. reload pressure

同一sourceをWorkspace reload後に再度読み込む。

確認する性質:

- Procedure keyが同一
- artifact fingerprintが同一
- normalized `ir.World`が同一
- Business Action × Pattern derived Projectionが同一

このreloadにはCanonical Procedure objectを必要としない。

## 4. Business-only change pressure

BP-04のBusiness Action文言だけを変更する。

```text
same Procedure key
changed Business source bytes
changed artifact fingerprint
same Application / Semantic World
same Pattern-derived meaning
changed Business-side Projection
```

旧fingerprintをbaseとしてacceptしようとした場合は`stale base fingerprint`として拒否する。

新fingerprintを明示したfresh acceptanceだけを通す。

### 含意

Reference-local stable keyは「何を変更しているか」を束ね、fingerprintは「どのsource snapshotか」を固定できる。

Business-only変更をApplication semantic changeへ偽装せず、stale sourceへの書込みも防げる。

## 5. file move pressure

Business Procedure source bytesを変えず、

```text
procedures/order-handling.json
```

から

```text
procedures/sales/order-handling.json
```

へ移動する。

期待結果:

```text
Procedure authoring key
  same

artifact fingerprint
  same

source locator
  changed

Canonical World
  unchanged
```

file pathをBusiness Procedure identityにしない。

## 6. Source Indexで構造的には表現できること

`workspace.Result`には`SourceContributions`があり、既存実装ではQueryのnative SQL等を既存logical subjectへ`RoleAuthoring`として追加している。

実験では同じmechanismをin-memoryで利用し、Business Procedure sourceを次のReferenceへ投影する。

```text
Subject
  kind = business-procedure-authoring
  id   = reference-local Procedure key

Reference
  role = authoring
  artifactPath = procedures/...json
```

`BuildSourceIndex()`はManagedFileをArtifact化し、このauthoring Referenceを同じIndexへ保持できる。

## 7. ただしSource Indexの正式契約には未解決点がある

現行`subject.Subject`のcontractは、path-independentな

> semantic or logical target inside a TEI World

である。

一方、この実験のBusiness Procedure keyは意図的に`ir.World`へ追加していない。

したがって、

```text
business-procedure-authoring Subject
```

を既存Source Indexへ入れることは**API上は構造的に可能**だが、現行Subject contractをそのまま満たすproduction設計とはまだ言えない。

このexperiment-only SourceContributionを根拠にSubject contractを黙って拡張しない。

## 8. RQ-002との圧力

RQ-002はauthoring sourceから「どのCanonical subjectへ寄与したか」を追跡できることを要求する。

Business Procedureをfile-native accepted sourceへ進めながらCanonical Procedure objectを持たない場合、次の問いが残る。

```text
Business Procedure authoring source
        ↓
何をCanonical subjectとして追跡するか？
```

これは今回初めて具体化した圧力である。

## 9. 候補を先に一つへ決めない

現時点では少なくとも次の候補がある。

### A. minimal Canonical Procedure identity shell

Business Procedure本体を巨大なCanonical schemaへせず、path-independent identityと最小contractだけをWorld側へ持つ。

利点:

- RQ-002 / current Subject contractへ素直に接続できる
- Source Index / impact / Acceptanceのtargetを既存方式で扱える

リスク:

- identity shellから徐々にWorkflow / role / state等を吸収するconcept creep
- Reference-local keyで足りる段階でCoreを増やす可能性

### B. Subject contractをWorld外のaccepted design targetへ広げる

利点:

- Business ProcedureをWorldへ追加せずpath-independent Source Index targetを持てる

リスク:

- `Subject`がCanonical / authoring / evidence targetを兼用し始める
- RQ-002の「Canonical subject」意味を弱める可能性

### C. Authoring identity registryを別に持つ

利点:

- Business semantic identityとWorld Canonical identityを明確に分離できる

リスク:

- identity体系が増え、Source Index / Shared Acceptance / impact analysisが複雑化する

### D. full Canonical Procedure model

現時点ではEvidenceに対して重すぎる。

Workflow Engineや詳細Business Flow schemaまで同時に導入する理由はない。

## 10. 現時点の判断

今回のEvidenceからは次までは言える。

1. `procedures`というfile-native source root自体は、現在のWorkspaceのforward-compatible構造でWorldを変えずに保持できる
2. strict decode / reload / Business-only change / stale base / file moveはReference-local key + fingerprintで扱える
3. derived Matrix / diagnosticを同じnormalized Worldから再生成できる
4. Source Index APIはexperiment-only authoring Referenceを構造的に表現できる
5. しかしcurrent Subject contractとRQ-002を満たす**正式なCanonical trace target**は未解決

したがって、Canonical `Procedure`が必要だと確定したわけではない。

一方、「Canonical Procedureは不要」とだけ言い切れる段階からも進み、**file-native accepted sourceへ昇格する時点でCanonical trace target問題を解く必要がある**ことが明確になった。

## 11. 次のPressure

次に優先するのはschema追加ではなく、Shared Acceptanceを最小限再現してtarget contractを圧力試験することである。

確認する問い:

1. Business Procedure change proposalは何を`baseModelRevision` / target identityとして持つべきか
2. Reference-local keyだけでaccept / reload / re-derive / impact説明を安全に完結できるか
3. RQ-002を満たすためにCanonical identity shellが不可避になる場面はどこか
4. minimal identity shellを導入する場合、本体Procedure内容までWorldへ入れる必要があるか
5. Source Index Subject contractを広げる方が小さいか、identity shellの方が小さいか

この比較Evidenceが出るまでCanonical Procedure ADRへ進まない。

## Verification

主なtest:

- `TestForwardCompatibleProcedureSourceRootDoesNotChangeWorld`
- `TestAcceptedProcedureSourceUsesAuthoringReferenceWithoutWorldExpansion`
- `TestAcceptedProcedureSourceReloadKeepsKeyAndDerivedMeaning`
- `TestBusinessOnlyAcceptedSourceChangeRequiresFreshAcceptance`
- `TestAcceptedProcedureSourceMoveChangesLocatorNotAuthoringSubject`

これらの`SourceContribution`登録はexperiment-onlyであり、production Source Index contractの採用決定ではない。

actual Repository runnerがcheckout / Go testへ到達するまではEvidence in progressとする。
