# Semantic Extension依存解決 Pre-ADR — 2026-10

- 状態: Pre-ADR / Reference前
- 主担当要求: RQ-031 / RQ-032 / RQ-033
- 実証レーン: ISSUE-000077
- 関連: ISSUE-000040 / ISSUE-000047 / ISSUE-000053
- 対象: Semantic Extensionの依存、解決、固定、説明可能性
- 非対象: package配布サービス、Marketplace、Runtime sandboxの再設計
- 敵対的評価: `plan/SEMANTIC_EXTENSION_ADVERSARIAL_REVIEW_2026-10.md`

## 1. このPre-ADRで決めたいこと

TEIでは、標準機能を含む多くの意味体系をSemantic Extensionとして提供する方向を採る。この構成では、Extension数と推移依存が大きくなる。

問題は依存数そのものではない。Semantic Extensionでは、解決された依存集合が変わると、同じApplication sourceでも意味解釈が変わる可能性がある。これは通常のpackage managerが扱う「buildできるか」という問題より上流にあり、Canonical meaningの再現性へ直接影響する。

そのためTEIでは、次を同時に満たす必要がある。

1. 大きな推移依存を人が手で列挙しなくてよい。
2. 同じ入力から同じSemantic Extension集合を決定論的に得られる。
3. なぜそのExtensionが選ばれたか、なぜ競合したかを説明できる。
4. RuntimeやToolingの都合でCanonical meaningが暗黙に変わらない。
5. 標準機能も第三者Extensionと同じ契約を使い、Coreだけの隠れた意味追加経路を増やさない。
6. Extension更新を通常の意味変更と同じShared Acceptance境界で扱える。

この文書では、最終syntaxやresolver実装を固定しない。ISSUE-000077のReferenceで先に検証すべき不変条件を整理する。

## 2. 既存方式から採るもの

### Emacs

Emacsから採るべき点は、本体機能と追加機能が近い拡張面を使い、後から機能を組み込めることにある。Hookやgeneric functionを使えば、既存処理の全体を書き換えずに拡張を参加させられる。

一方、TEIではAdviceのように既存の意味を外側から任意に差し替える方式をCanonical pathへ持ち込まない。load順や偶然の実装順序で結果が変われば、同じsourceから同じmeaningを再現できなくなる。

### MLIR

MLIRのDialectとInterfaceは、TEIの意味拡張に近い。新しいDialectを追加しても、既存analysisは各Dialectの具体型を列挙せず、Interfaceを通して横断的に扱える。

TEIでも、特定のExtension packageへ直接依存するより、必要な意味契約を表すSemantic Interfaceへの依存を優先する。特定packageの意味そのものが必要な場合に限り、package identityへ直接依存する。

### OSGi

OSGiのRequirement / Capability modelでは、推移依存とphaseごとの有効性を明示できる。TEIでも、何を要求するかと、どのExtensionがその要求を満たすかを分ける。

ただし、OSGiのclass space consistencyをそのまま移植するのではない。TEIでは「同一Canonical World内で互換性のない意味体系が二重に有効にならないこと」をcoherence条件として扱う。

### Cargo / Go / Bazel

Cargoのlock-firstとbacktracking、Goの単純な解決規則とgraph pruningは参考になる。Bazelの`compatibility_level`は過去の設計例として参照できるが、2026-10時点のBazel main documentationではdeprecated / no-opである。このため、TEIで独立したcompatibility levelを設ける直接の根拠にはしない。

Semantic Extensionでは、「互換範囲内の新しいversionなら自動的に選んでも意味は変わらない」とは仮定しない。version rangeは候補を探す条件として使い、Accepted Worldはexact versionとdeclaration fingerprintで固定する。

### PubGrub

依存が複雑になった場合、単に解決不能と返すだけでは、人もAIも適切に修正できない。PubGrub型のroot-cause derivationは、「どの要求とどの要求が両立しなかったか」を説明する方法として有力である。

ただし、ISSUE-000077 E0でPubGrub実装そのものを必須にはしない。先に必要なのは、resolverが構造化されたresolution trace / conflict causeを返せる契約である。

## 3. TEIでは依存グラフを一つにしない

最低でも次の解決面を区別する。

| 解決面 | 主な内容 | 基本scope | Canonical meaningへの影響 |
| --- | --- | --- | --- |
| Semantic | concept、constraint、relation、Semantic Interface | Workspace / Application | 直接影響する |
| Realization | Projection、Target向けlowering、変換 | Target / build context | 原則derived |
| Runtime | Provider、Executor、外部Runtime | Environment | 実行方法へ影響 |
| Tooling | Editor、lint、importer、analysis補助 | Authoring context | 原則影響させない |

最初から四種類のlock fileを作る必要はない。ただし、Semantic Extensionの解決結果を現行Environment Plugin Lockへ混ぜない。

現行Runtime PluginはEnvironmentごとに解決できる。一方、同じApplicationでdevelopmentとproductionのSemantic Extension集合が異なれば、環境によってCanonical meaningが変わり得る。Semantic SetはEnvironmentより上流で固定する。

## 4. Requirementをpackage名だけにしない

Semantic dependencyには少なくとも二種類がある。

### Package identity dependency

特定Extensionが定義する意味そのものへ依存する場合に使う。

概念例:

```text
requires extension reference.hierarchy.integrity compatible ^1
```

### Semantic Interface dependency

必要な意味契約を満たす実装があればよい場合に使う。

概念例:

```text
requires interface relation-acyclic-validator/v1
```

この区別によって、横断機能が個別Extensionを一つずつ列挙する構造を避けられる。

Package identity dependencyそのものは禁止しない。あるExtensionのsymbolやparameter semanticsを直接参照する場合は、曖昧なInterface dependencyへ置き換えない。

### Semantic Interfaceの粒度

Semantic Interfaceは、あるdomainのvalidatorや機能をまとめた便宜的な単位ではなく、consumerが独立して要求できる意味契約として切ります。

第二consumerでInterface全体ではなく一部の意味だけが再利用できた場合、既存Interfaceを別domainへ無理に適用しません。共通部分に独立した意味があることを確認し、小さいInterfaceへ切り出してdomain側で合成することを優先します。

一方、rule一つごとに機械的にInterfaceを作ることもしません。Interfaceの細分化は、次を満たす場合に行います。

- 二つ以上の独立consumerで同じ意味契約として利用できる
- input / output / unknown / failure semanticsをdomain語彙なしで説明できる
- 切り出した後も、各domain固有のValidation責務が明確に残る
- 分割によってprovider selectionやConformance Evidenceが不必要に増えない

SUI hierarchyとDatabase Migrationを使ったE3 candidateでは、hierarchy integrity全体ではなくdirected graphのacyclicだけがこの条件を満たす可能性を示しています。exact-head Evidenceが揃うまでは最終contractとして固定しません。

同じalgorithmが複数domainに現れても、それだけでSemantic Authorityを統合しません。たとえばSolution Releaseは既にCoreの`releaseplan.Compile`がcycleを拒否します。この場合、共通graph utilityを内部実装として共有する余地と、Semantic Extensionとして同じValidation ruleを付与することは分けて判断します。二重Authorityを作らないことを優先します。

Semantic Interfaceは単なる文字列ラベルにはしない。Interface自体がlogical identity、contract version、namespace authority、contract fingerprint、Source / provenanceを持つ意味契約である必要がある。providerの`implements`自己申告だけを適合Evidenceにはしない。

同じInterfaceを複数providerが満たす場合、Accepted wiringまたは明示的なselection policyが無ければ、package ID、version、catalog列挙順等で暗黙に一つを選ばない。

## 5. 異なる意味体系の接続はBridge Extensionへ寄せる

Data、Interaction、Procedure等のSemantic Extensionが互いを直接参照し始めると、標準機能だけでも依存関係が密になる。

たとえばDataとProcedureの相互検証が必要でも、両方を相互依存させない。

```text
data extension       procedure extension
       \               /
        \             /
         data-procedure bridge
```

Bridge Extensionは両者のSemantic Interfaceへ依存し、横断的なValidationやRelationを提供する。

この方式には二つの利点がある。

- 個々の意味体系を単独でも利用できる。
- cross-view ruleをどちらか一方の意味へ埋め込まずに済む。

Bridgeが増えすぎる場合は、共通InterfaceかCore primitiveが不足しているEvidenceとして扱う。Bridge数を減らして見せるためだけに、巨大なumbrella Extensionへ統合しない。

## 6. Extension package dependencyはDAGを基本とする

Extension packageの読み込み依存にcycleがある場合は、原則として拒否する。

循環が必要に見える場合は、次の順で見直す。

1. 共通Semantic Interfaceを下位へ抽出できないか。
2. 双方向関係をBridge Extensionへ移せないか。
3. package dependencyではなく、Canonical object間のRelationとして表すべきではないか。

ここで禁止するのは、Extension packageを解決するためのcycleである。業務上のRelation graphやobject referenceにcycleが存在することまで禁止しない。

## 7. Semantic compatibilityを別のversion軸へ早期固定しない

Semantic Extensionでは、少なくとも次を区別する。

- extension identity
- semantic version
- declaration fingerprint
- dependency compatibility constraint

semantic versionはRuntime binary versionと分ける。一方、semantic versionとは別に`compatibility level / family / epoch`を必須軸として今の段階で固定しない。

独立したcompatibility軸を設けると、semantic versionとの優先関係や不整合時の判定規則が新たに必要になる。Referenceでversion rangeだけでは表現できない実例が確認されるまでは、互換範囲、明示conflict、exact lockでPressureする。

現時点の不変条件は次である。

- 依存側が受け入れるsemantic version範囲を明示できる。
- 互換性のない意味を同じCanonical subjectへ黙って混在させない。
- Accepted Worldはexact versionとdeclaration fingerprintを含むSemantic Lockで固定する。
- 別世代の意味体系を同時に扱う必要がある場合は、namespace、translation、migration等の境界を明示する。

別のcompatibility family / epoch等が必要かは、複数ReferenceのEvidenceから判断する。

## 8. Optional dependencyで既存意味を暗黙変更しない

あるExtensionが「存在していれば」別Extensionのmeaningを変える方式は避ける。

```text
A alone       -> meaning X
A + B present -> Aのmeaningが自動的にYへ変わる
```

このようなambient activationを許すと、Semantic Setを確認しなければsourceの意味が分からなくなる。さらに、第三者Extensionの追加だけで既存subjectの意味が予期せず変わる。

追加の組合せ意味が必要なら、Bridge Extensionまたは明示的なroot requirementとしてSemantic Lockへ載せる。

optional requirement自体は禁止しない。欠けても既存の意味が変わらず、追加のdiagnosticやProjectionだけを提供するなど、影響範囲を契約で説明できる場合に限定する。

ただし、diagnosticやValidationの有無がShared AcceptanceのAccept / Rejectへ影響するなら、それは実質的にSemantic dependencyである。その場合はoptional Toolingとしてambientに追加せず、Semantic Setへ明示してlock / fingerprint / Acceptanceの対象にする。

## 9. Semantic LockをCanonical interpretationの入力として扱う

Semantic Extensionの解決結果は、Accepted Worldに対してexactに固定する。

概念上のSemantic Lockには最低限、次を保持する。

- root requirementsとそのfingerprint
- resolved extension identity
- exact semantic version
- declaration fingerprint
- Semantic Interface contract fingerprint
- resolved requirement -> provider edge
- Semantic Interface wiring
- resolution policy identity / version
- fingerprint canonicalization version
- graph / lock format version

Artifact binaryやRuntime Plugin digestは既存Runtime Plugin Lock側で扱う。Semantic Lockへ実行Authorityを混ぜない。

Canonical interpretationは概念上、次の入力から決まる。

```text
authoritative source
+ accepted Semantic Lock
= reproducible Canonical meaning
```

そのため、sourceが変わっていなくても、Semantic Lockが変われば意味変更候補になり得る。

Semantic Set全体についてstable fingerprintを作り、Proposal、Decision、qualification Evidence、derived artifactから必要に応じて参照できるようにする。

既存lockがある場合、Catalogへ新versionが追加されたことだけを理由に再解決しない。まずlockに記録したexact graphとfingerprintを検証する。root requirements、resolution policy、canonicalization規則が変わった場合はstale lockとして扱い、candidate re-resolutionとShared Acceptanceへ進める。

Unresolvedを含むgraphからAccepted Semantic Lockを作らない。

## 10. 更新はdependency updateではなくSemantic Changeとして扱う

新しいversionが公開されてもAccepted Semantic Lockを自動更新しない。

更新は次の順を基本とする。

```text
current Semantic Lock
  -> candidate re-resolution
  -> candidate Semantic Lock
  -> semantic diff
  -> affected subjects / reverse dependency
  -> Validation / Conformance
  -> unresolved / conflict
  -> Shared Acceptance
  -> accepted Semantic Lock
```

SemVer上compatibleであることは、更新候補にできる条件にすぎず、Acceptanceの代わりにはならない。

Runtime Pluginだけを更新し、semantic declaration fingerprintが変わらない場合は、Semantic Lockまで無条件にinvalidとしない。

## 11. 標準機能は同じextension contractをdogfoodする

将来、Data、Constraint、Interaction、Procedure等の標準意味体系を`std.*`として提供する場合も、第三者Extensionとは別のresolverを標準機能だけに用意しない。

実装をbuiltinにすることは許容する。ただし、意味のidentity、dependency、Interface、conflict、lock、discoveryには同じcontractを適用する。

Coreへ残すのは、Extensionを安全に成立させるためのmeta-level mechanismを中心とする。

例:

- identity / namespace
- extension registry
- Requirement / Semantic Interface
- dependency resolution contract
- Source / provenance
- diagnostic / Unresolved
- lock / fingerprint
- Shared Acceptanceとの接続
- authority boundary

どの標準語彙までCore primitiveとして残すかは、複数ReferenceのEvidenceで決める。

標準構成を使いやすくするために、Standard Profileをroot requirement setとして提供してもよい。

```text
profile business-app
  -> std.data
  -> std.constraint
  -> std.interaction
  -> std.procedure
  -> required bridges
```

Profile自体を新しいSemantic authorityにはしない。ただし、Profileはroot requirement集合を選ぶため、意味解釈に影響する入力である。Profileにはidentity、version、Source、fingerprintを持たせ、Accepted Applicationが参照するProfileの変更をSemantic Changeとして扱う。

### 11.1 Extensionの論理境界と物理packagingを分ける

Semantic Extension、Runtime Plugin、物理artifactを一対一にはしない。

```text
Logical Extension / Plugin identity
        |
        | semantic / capability contract
        v
Target realization plan
        |
        +-- builtin / static linked
        +-- shared native library or host implementation
        +-- portable Wasm module
        +-- isolated process
        +-- external service binding
        |
        v
Target artifact / deployment image
```

一つのTarget artifactへ複数のlogical Extension / Plugin実装をまとめてもよい。逆に、一つのlogical PluginがTargetごとに異なるartifactで実現されてもよい。

この再配置で変えてはいけないものは、logical identity、semantic contract、Capability contract、Accepted Lock、Authority境界、provenanceへの逆追跡である。

したがって、Semantic Set fingerprintへ物理artifactの分割方法を混ぜない。Targetごとのpackaging / build artifactには別のfingerprintを持たせ、どのSemantic Set / Plugin Lockから導出したかを参照させる。

### 11.2 declarationをeager、実行codeを必要時だけloadできる構造にする

Semantic declaration、identity、dependency、Lockは、Tool / AI / buildが意味を理解するために必要である。一方、すべての実行codeを起動時に読み込む必要はない。

基本方針は次とする。

- declaration-only ExtensionはRuntime artifactを持たなくてよい。
- Semantic SetのdeclarationとLockは小さく保ち、意味解決に必要な範囲をeagerに利用できるようにする。
- evaluator / Provider等の実行artifactは、Target buildで到達可能なものだけを含める。
- Acceptance / Conformanceに必要なSemantic Evaluatorは、判定時までに解決・実行できなければならない。Application起動後まで未検証のまま遅延してはいけない。
- Runtimeで呼び出すProvider等は、起動時に不要ならTarget条件に応じてlazy loadできる。
- 同じruntimeやsupport codeを各Pluginへ複製せず、共有またはlink時に統合できるようにする。
- content digestが同じartifactをversionやProfileごとに重複保存しない余地を保つ。
- lazy load失敗を意味の不存在へ読み替えない。必要な実装を取得できなければRuntime / Realization側のUnresolvedとして扱う。

意味の依存graphをpruneするのではなく、**実行artifactの到達可能性をpruneする**点を区別する。Accepted Semantic Setに含まれる意味は、実行codeをまだloadしていなくても消えない。

### 11.3 Web Targetをprocess Pluginへ依存させない

Webブラウザでは、native process起動や任意filesystem access等を前提にできない。したがって、標準Semantic Extensionや標準Capabilityをprocess-only実装へ固定しない。

Web Targetでは、少なくとも次の実現経路を候補にできる構造を保つ。

- declaration-only
- host JavaScript / Web API integration
- browser-compatible Core WebAssembly
- build時にhostへ統合したbuiltin / linked implementation
- browserから利用可能な外部service binding

Core WebAssemblyは現在のブラウザで広く利用でき、streaming compile / instantiateも利用できる。一方、WebAssembly Component Modelのbrowser native integrationは2026-10時点でも将来目標を含む段階にある。そのため、TEIのPlugin ABIをComponent Modelのnative browser supportへ依存させない。

Go標準の`js/wasm`はTarget候補の一つだが、各Extensionを独立したGo Wasmへ機械的に変換する方針にはしない。Go公式資料でも、小さなWasm binaryにruntime由来のサイズ負担があり、importするlibraryによって大きくなり得るとされている。TinyGo等の小型toolchain、将来の軽量Go→Wasm compiler、JavaScript、static linking等をReferenceで比較し、Extension contractとcompiler選択を分離する。

参照:

- Go WebAssembly Wiki: https://go.dev/wiki/WebAssembly
- WebAssembly `instantiateStreaming`: https://developer.mozilla.org/docs/WebAssembly/Reference/JavaScript_interface/instantiateStreaming_static
- WebAssembly Component Model goals: https://github.com/WebAssembly/component-model/blob/main/design/high-level/Goals.md
- TinyGo WebAssembly: https://tinygo.org/docs/guides/webassembly/

### 11.4 Target buildで最適化しても追跡性を失わない

Target packagerは、Accepted Semantic SetとRuntime / Capability Bindingからderivedなpackaging planを作る。

概念上は次の順を取る。

```text
Accepted Semantic Set
 + Target
 + Capability / Runtime Binding
        ↓
reachable implementation analysis
        ↓
Target realization plan
        ↓
unused executable implementation elimination
shared runtime / dependency deduplication
compatible implementation coalescing
compression / cache plan
lazy-load boundary
        ↓
Target artifact
```

Target realization planはCanonical authorityではない。削除して再生成できるderived artifactとして扱う。

複数Pluginを一つのartifactへまとめた場合も、diagnosticやEvidenceから元のlogical Plugin / Extensionへ戻れるmappingを保持する。

### 11.5 性能・storage・可搬性をExtension Referenceの品質条件にする

Plugin化によるcostは、機能数ではなく増分として測る。

ReferenceではTargetごとに少なくとも次を観測する。

- cold start / warm start
- compressed initial transfer
- install / cache storage footprint
- steady-state memory
- resolver時間
- evaluator / Capability dispatch overhead
- Extensionを一つ追加したときの増分artifact size / memory / startup cost
- unused executable contributionを除外できたか
- 共通runtime / dependencyを重複格納していないか
- Web Targetでnative process依存が残っていないか

現時点で一律の上限値は置かない。RQ-024に従い、First / Second Targetのactual Evidenceからcontext-specific thresholdを決める。

重要なのは「Plugin化前より必ず高速であること」ではない。**拡張可能性を得るために、Target固有の実行・配布最適化を構造上禁止しないこと**を不変条件とする。

## 12. AI / Toolへresolverの理由を公開する

Pluginが増えると、人もAIもresolved setだけでは依存関係を保守できない。

少なくとも、次の情報へ機械可読な形で到達できる必要がある。

- why: なぜこのExtensionが入ったか
- path: どのroot requirementから到達したか
- reverse dependencies: 外した場合に何が影響するか
- interface wiring: どのRequirementを何が満たしたか
- conflict cause: どの制約同士が両立しないか
- unresolved: 解決に足りない情報は何か
- semantic impact: candidate lockへ更新した場合に影響するsubject

CLI名はまだ固定しないが、将来のsurfaceは概念上次を扱えることを想定する。

```text
extension tree
extension why <id>
extension reverse-deps <id>
extension conflicts
extension explain-resolution
extension impact <candidate>
```

AIへCatalog全体を毎回渡すことはしない。Semantic Lockとdependency indexからtaskに関係するclosureを切り出し、必要に応じて理由を辿りながら範囲を広げる。

## 13. Resolver実装は差し替え可能にする

初期Referenceでは、複雑なsolverを先に作らない。

Core側では、少なくとも次の入出力契約を先に固定する。

入力:

- root requirements
- available extension declarations
- compatibility constraints
- Semantic Interface requirements / providers
- existing accepted lock if any
- resolution policy

出力:

- exact resolved graph
- stable graph fingerprint input
- resolution trace
- unresolved requirements
- structured conflict cause

最初の制約集合を単純なdeterministic resolverで扱えるなら、それを使う。ただし、resolverは宣言したconstraint languageの範囲でsoundかつcompleteである必要がある。解が存在するのに探索順やgreedyなhighest-compatible選択だけを理由にunsatisfiableと判定してはいけない。

複数の有効解があり、Acceptedなresolution policyでも一意に決められない場合は、solver内部の探索順で一つを選ばず、候補またはUnresolvedとして表す。

現行Referenceのhighest-compatible先行選択には、より低いcompatible versionへbacktrackすれば解けるgraphをfalse conflictにできる反例がある。E0-Bでこのnegative controlを追加し、その結果に応じてbacktrackingまたは同等の探索が必要かを判断する。

将来大規模になるという予測だけを理由に、E0の時点から特定solver実装そのものをProductの中心にはしない。必要なのは、意味の再現性を保つ解決契約である。

## 14. 現行Reference実装との関係

PR #922の`tools/reference-semantic-extension-catalog`は、このPre-ADRより先に着手されたReference実装である。2026-10-04時点では、次の境界まで実装している。

- Extension identity、namespace、semantic version、Source fingerprintをReference descriptorから読み込む
- concrete Extension IDとcompatible majorによる推移依存を決定論的に解決する
- dependency cycleとexact version重複を拒否する
- namespace ownership conflictをsilent overrideせずUnresolvedとして保持する
- descriptor由来のresolution fingerprintとSource来歴のevidence fingerprintを分離する
- Extension selectionをAcceptanceやRuntime Authorityへ昇格させない

この実装はE0-AとE0-Bの初期検証に利用できる。ただし、次の項目はまだ実証していない。

- Semantic Interfaceを介したtyped dependencyとInterface contract authority
- greedyなhighest-compatible選択によるfalse conflictの排除
- Bridge Extensionによるcross-facet composition
- Accepted Semantic Lock
- root requirementからのwhy / reverse dependency
- structured conflict derivation
- namespace authority、特に`std.*`の予約
- Runtime Plugin更新とSemantic Set fingerprintの独立性

したがって、PR #922のReferenceを先に完成形まで広げてから設計を合わせることはしない。現在の小さいresolverをnegative / positive controlとして使い、上記の不足をE0-B / E0-Cで順に検証する。このPre-ADRは既存Referenceの成立Evidenceを否定せず、その先で必要になる不変条件を定義する。

## 15. ISSUE-000077で追加するdependency pressure

E0のdeclaration / discoveryに加え、E0内で次の小規模なdependency pressureを行う。

### D1 — diamond

同じSemantic Interfaceへ複数経路から到達しても、一つの互換なproviderへ決定論的に収束する。

### D2 — incompatible range

二つのExtensionが両立しないversion constraintを要求した場合、load順に依存せずblocking conflictになる。

### D3 — package cycle

Extension package dependency cycleを検出し、部分的にloadしたWorldをAccepted扱いしない。

### D4 — explicit bridge

二つの独立ExtensionをBridgeで接続し、一方を単独利用した場合に他方を暗黙requireしない。

### D5 — lock repeatability

同じroot requirements、同じCatalog、同じAccepted sourceから、列挙順を変えても同じresolved graphとfingerprintを得る。

### D6 — semantic / runtime separation

Runtime Pluginの実装versionだけを変更しても、semantic declarationが同一ならSemantic Set fingerprintへ不要な差分を入れない。

### D7 — explainability

各resolved Extensionについてrootからの依存pathを取得でき、conflict時には両立しないrequirementの由来へ戻れる。

### D8 — greedy false conflict

higher compatible versionを選ぶとtransitive conflictになるが、lower compatible versionなら全体を解決できるgraphを用意する。resolverが最初の選択だけを理由にunsatisfiableとしないことを確認する。

### D9 — Interface spoof / ambiguity

同じSemantic Interface名を異なるcontract fingerprintで宣言した場合に同一Interface扱いしない。複数providerが同じInterfaceを満たす場合も、Accepted wiringまたは明示policyなしに暗黙選択しない。

### D10 — namespace authority

第三者Extensionが`std.*` namespaceを宣言しても、first claimantとしてownershipを得ないことを確認する。

### D11 — lock input drift

root requirements、resolution policy、canonicalization versionのいずれかだけを変え、既存Semantic Lockを再利用できないことを確認する。

### D12 — optional semantic side effect

optional diagnostic / ProjectionがAcceptance結果を変えようとする場合、ambient Toolingとして扱わずSemantic dependencyへの明示を要求する。

最初のSUI hierarchy Extensionを実際のroot consumerとして使う。D1〜D12の構造検証にはReference-local Extensionを必要最小限だけ追加し、syntheticなgraphだけでSemantic Extensionの成立を主張しない。

## 16. 実装順序

ISSUE-000077は次の順で進める。

```text
E0-A  declaration / identity / Source / discovery
  ↓
E0-B  typed requirement / dependency graph / conflict
  ↓
E0-C  Semantic Lock / repeatability / why
  ↓
E1    deterministic evaluator boundary
  ↓
E2    SUI fixture validation
  ↓
E3    independent second consumer
  ↓
std.*昇格判断
```

E0-A〜Cは別のremote branchへ分けない。同じISSUE-000077 workstreamで小さなcommitに分け、mainへ収束させる。

## 17. ADRへ進める判断

次のEvidenceが揃うまでは、具体的なresolver algorithmや最終manifest syntaxをADRで固定しない。

- real SUI consumerを含むdependency graph
- diamond / conflict / cycleのnegative control
- Semantic Lock repeatability
- load order非依存
- dependency reason / conflict explanation
- semantic updateによるimpact
- Runtime Plugin更新との分離
- greedy false conflictを起こさないresolution completeness
- Semantic Interface contractのidentity / fingerprint / authority
- `std.*` namespace authority
- root / wiring / policy / canonicalizationを含むlock再現性
- Acceptanceに使うSemantic EvaluatorとEnvironment Runtime selectionの分離
- optional contributionのAcceptance影響境界
- logical Extension数とphysical artifact数を分離したTarget packaging
- unused implementation pruning / shared runtime deduplication
- Web Targetでのprocess-free realization
- RQ-024に沿ったstartup / transfer / memory / storageの増分測定

一方、以下は現時点からReferenceで変えない条件として扱う。

1. Semantic resolutionをEnvironment Runtime resolutionへ混ぜない。
2. silent override / last-winsを使わない。
3. Extension package cycleをAccepted graphへ入れない。
4. accepted semantic setをexact lock / fingerprintで再現可能にする。
5. std.*も同じExtension contractを使う。
6. Semantic dependency updateをShared AcceptanceなしでAccepted Worldへ反映しない。
7. dependency explanationを人とAIの共通surfaceにする。
8. 有効解があるgraphを探索順だけでconflictにしない。
9. Interface / namespace authorityをfirst claimantや自己申告へ委ねない。
10. semantic validityをEnvironmentごとのEvaluator選択で変えない。
11. logical Extension / Pluginを一つのartifact / process / Wasm moduleへ固定しない。
12. Target packagingの最適化でSemantic identity / Lock / provenanceを失わない。
13. Web Targetの成立をnative process Pluginへ依存させない。
14. declaration-only ExtensionへRuntime artifactを要求しない。

## 18. 参照した外部設計

- GNU Emacs Lisp Reference Manual, Hooks / Generic Functions / Advice
  - https://www.gnu.org/software/emacs/manual/html_node/elisp/Hooks.html
  - https://www.gnu.org/software/emacs/manual/html_node/elisp/Generic-Functions.html
  - https://www.gnu.org/software/emacs/manual/html_node/elisp/Advising-Functions.html
- MLIR, Defining Dialects / Interfaces
  - https://mlir.llvm.org/docs/DefiningDialects/
  - https://mlir.llvm.org/docs/Interfaces/
- OSGi Core, Resolver Service / Resource API
  - https://docs.osgi.org/specification/osgi.core/7.0.0/service.resolver.html
  - https://docs.osgi.org/specification/osgi.core/8.0.0/framework.resource.html
- Cargo Book, Dependency Resolution / Features
  - https://doc.rust-lang.org/cargo/reference/resolver.html
  - https://doc.rust-lang.org/cargo/reference/features.html
- Go Modules Reference
  - https://go.dev/ref/mod
- Bazel module API（compatibility_levelの現行状態確認を含む）
  - https://bazel.build/rules/lib/globals/module.html
- Dart pub, PubGrub solver design
  - https://github.com/dart-lang/pub/blob/master/doc/solver.md

- 敵対的評価: `plan/SEMANTIC_EXTENSION_ADVERSARIAL_REVIEW_2026-10.md`

- WebAssembly browser loading:
  - https://developer.mozilla.org/docs/WebAssembly/Reference/JavaScript_interface/instantiateStreaming_static
- Go WebAssembly:
  - https://go.dev/wiki/WebAssembly
- WebAssembly Component Model goals:
  - https://github.com/WebAssembly/component-model/blob/main/design/high-level/Goals.md
- TinyGo WebAssembly / binary optimization:
  - https://tinygo.org/docs/guides/webassembly/
  - https://tinygo.org/docs/guides/optimizing-binaries/
