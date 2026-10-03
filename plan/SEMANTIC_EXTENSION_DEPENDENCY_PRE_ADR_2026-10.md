# Semantic Extension依存解決 Pre-ADR — 2026-10

- 状態: Pre-ADR / Reference前
- 主担当要求: RQ-031 / RQ-032 / RQ-033
- 実証レーン: ISSUE-000077
- 関連: ISSUE-000040 / ISSUE-000047 / ISSUE-000053
- 対象: Semantic Extensionの依存、解決、固定、説明可能性
- 非対象: package配布サービス、Marketplace、Runtime sandboxの再設計

## 1. このPre-ADRで決めたいこと

TEIでは、標準機能を含む多くの意味体系をSemantic Extensionとして提供する方向を採る。その場合、Extension数と推移依存は大きくなる。

問題は依存数そのものではない。Semantic Extensionでは、解決された依存集合が変わると、同じApplication sourceの意味解釈が変わる可能性がある。通常のpackage managerで「buildできるか」を決める問題より上流にあり、Canonical meaningの再現性に直接関係する。

そのためTEIでは、次を同時に満たす必要がある。

1. 大きな推移依存を人が手で列挙しなくてよい。
2. 同じ入力から同じSemantic Extension集合を決定論的に得られる。
3. なぜそのExtensionが選ばれたか、なぜ競合したかを説明できる。
4. RuntimeやToolingの都合でCanonical meaningが暗黙に変わらない。
5. 標準機能も第三者Extensionと同じ契約を使い、Coreだけの隠れた意味追加経路を増やさない。
6. Extension更新を通常の意味変更と同じShared Acceptance境界で扱える。

この文書では最終syntaxやresolver実装を固定しない。ISSUE-000077のReferenceでPressureすべき不変条件を先に整理する。

## 2. 既存方式から採るもの

### Emacs

Emacsから採るべき点は、本体機能と追加機能が近い拡張面を使い、後から機能を参加させられることにある。Hookやgeneric functionは、既存処理の全体を書き換えずに拡張を参加させる。

一方、TEIではAdviceのように既存意味を外側から任意に差し替える方式をCanonical pathへ持ち込まない。load順や偶然の実装順序で結果が変わる余地を残すと、同じsourceから同じmeaningを再現できなくなる。

### MLIR

MLIRのDialectとInterfaceはTEIの意味拡張に近い。新しいDialectを追加しても、既存analysisが各Dialectの具体型を列挙せず、Interfaceを通して横断的に扱える。

TEIでも、特定Extension packageへの依存より、必要な意味契約を表すSemantic Interfaceへの依存を優先する。特定packageの意味そのものが必要な場合だけpackage identityへ直接依存する。

### OSGi

OSGiのRequirement / Capability modelは、推移依存とphaseごとの有効性を明示できる。TEIでも、何を要求するかと、どのExtensionがそれを満たすかを分ける。

ただし、OSGiのclass space consistencyをそのまま移植するのではない。TEIでは「同一Canonical World内で互換性のない意味体系が二重に有効にならないこと」をcoherence条件として扱う。

### Cargo / Go / Bazel

Cargoのlock-first、Goの小さい解決規則とgraph pruning、Bazelのcompatibility levelは参考になる。

ただしSemantic Extensionでは、「互換範囲内なら新しいversionを自動的に選んでも意味は同じ」とは仮定しない。version rangeは候補探索の条件に使い、Accepted Worldはexact versionとdeclaration fingerprintで固定する。

### PubGrub

依存が複雑になった場合、単に解決不能を返すだけではAIも人も修正できない。PubGrub型のroot-cause derivationは、「どの要求がどの要求と両立しなかったか」を説明する考え方として有力である。

ただしISSUE-000077 E0でPubGrub実装そのものを必須にしない。最初に必要なのは、resolverが構造化されたresolution trace / conflict causeを返せる契約である。

## 3. TEIでは依存グラフを一つにしない

最低でも次の解決面を区別する。

| 解決面 | 主な内容 | 基本scope | Canonical meaningへの影響 |
| --- | --- | --- | --- |
| Semantic | concept、constraint、relation、Semantic Interface | Workspace / Application | 直接影響する |
| Realization | Projection、Target向けlowering、変換 | Target / build context | 原則derived |
| Runtime | Provider、Executor、外部Runtime | Environment | 実行方法へ影響 |
| Tooling | Editor、lint、importer、analysis補助 | Authoring context | 原則影響させない |

最初から四種類のlock fileを作る必要はない。ただし少なくとも、Semantic Extensionの解決を現行Environment Plugin Lockへ混ぜない。

現行Runtime PluginはEnvironmentごとに解決できる。一方、同じApplicationのSemantic Extension集合がdevelopmentとproductionで異なれば、環境によってCanonical meaningが変わり得る。Semantic SetはEnvironmentより上流で固定する。

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

この区別により、横断機能が個別Extensionを列挙する構造を避けられる。

Package identity dependencyを禁止はしない。あるExtensionのsymbolやparameter semanticsを直接参照する場合は、曖昧なInterface dependencyへ置き換えない。

## 5. 異なる意味体系の接続はBridge Extensionへ寄せる

Data、Interaction、Procedure等の各Semantic Extensionが互いを直接知り始めると、標準機能だけで依存網が密になる。

たとえばDataとProcedureの相互検証が必要でも、両方を相互依存させない。

```text
data extension       procedure extension
       \               /
        \             /
         data-procedure bridge
```

Bridge Extensionは、両者のSemantic Interfaceへ依存し、横断ValidationやRelationを提供する。

この方式には二つの利点がある。

- 個々の意味体系を単独でも利用できる。
- cross-view ruleをどちらか一方の意味へ埋め込まずに済む。

Bridgeが増えすぎる場合は、共通Interface不足またはCore primitive不足のEvidenceとして扱う。Bridge数そのものを隠すために巨大なumbrella Extensionへ統合しない。

## 6. Extension package dependencyはDAGを基本とする

Extension packageのload dependency cycleは原則として拒否する。

循環が必要に見える場合は、次の順で見直す。

1. 共通Semantic Interfaceを下位へ抽出できないか。
2. 双方向関係をBridge Extensionへ移せないか。
3. package dependencyではなく、Canonical object間のRelationとして表すべきではないか。

ここで禁止するのはExtension packageの解決cycleである。業務上のRelation graphやobject referenceにcycleが存在することまで禁止しない。

## 7. Semantic compatibilityをbinary versionだけに委ねない

Semantic Extensionは少なくとも次を分ける。

- extension identity
- semantic version
- compatibility level / family
- declaration fingerprint

compatibility levelは「同じCanonical Worldで同一意味体系として共存できるか」を判定するために使う。

基本規則は、一つのsemantic identity familyについて同じSemantic Set内に一つのcompatibility levelだけを有効にすることとする。

互換性のない世代を同時に扱う必要がある場合は、別namespaceまたは明示されたsemantic universeとして分離し、両者の間にMigration / Translation境界を置く。単にresolverが二つのmajorを選べたという理由だけで同じCanonical subjectへ混在させない。

## 8. Optional dependencyで既存意味を暗黙変更しない

あるExtensionが「存在していれば」別Extensionのmeaningを変える方式は避ける。

```text
A alone       -> meaning X
A + B present -> Aのmeaningが自動的にYへ変わる
```

このようなambient activationは、Semantic Setを見ないとsourceの意味が分からなくなるだけでなく、第三者Extensionの追加が既存subjectを予期せず変える。

追加の組合せ意味が必要なら、Bridge Extensionまたは明示的なroot requirementとしてSemantic Lockへ載せる。

optional requirement自体を禁止するわけではない。欠けても意味が変わらず、追加的なdiagnosticやProjectionだけを提供するなど、影響範囲を契約で説明できる場合に限定する。

## 9. Semantic LockをCanonical interpretationの入力として扱う

Semantic Extensionの解決結果は、Accepted Worldに対してexactに固定する。

概念上のSemantic Lockには最低限、次を保持する。

- root requirements
- resolved extension identity
- exact semantic version
- compatibility level
- declaration fingerprint
- resolved requirement -> provider edge
- Semantic Interface wiring
- graph format version

Artifact binaryやRuntime Plugin digestは既存Runtime Plugin Lock側で扱う。Semantic Lockへ実行Authorityを混ぜない。

Canonical interpretationは概念上、次の入力から決まる。

```text
authoritative source
+ accepted Semantic Lock
= reproducible Canonical meaning
```

このため、sourceが一字も変わらなくてもSemantic Lockが変われば意味変更候補になり得る。

Semantic Set全体についてstable fingerprintを作り、Proposal、Decision、qualification Evidence、derived artifactが必要に応じて参照できるようにする。

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

SemVer上compatibleであることは、更新候補にできる条件であってAcceptanceの代わりではない。

Runtime Pluginだけの更新でsemantic declaration fingerprintが変わらない場合は、Semantic Lockまで無条件にinvalidにしない。

## 11. 標準機能は同じextension contractをdogfoodする

将来、Data、Constraint、Interaction、Procedure等の標準意味体系を`std.*`として提供する場合も、第三者Extensionとは別の隠れたresolverを使わない。

実装がbuiltinであることは許容する。ただし、意味のidentity、dependency、Interface、conflict、lock、discoveryは同じcontractへ載せる。

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

標準構成を使いやすくするため、Standard Profileはroot requirement setとして提供できる。

```text
profile business-app
  -> std.data
  -> std.constraint
  -> std.interaction
  -> std.procedure
  -> required bridges
```

Profile自体を新しいSemantic authorityにはしない。

## 12. AI / Toolへresolverの理由を公開する

Pluginが増えた場合、人もAIもresolved setだけでは保守できない。

少なくともmachine-readableに次へ到達できる必要がある。

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

AIはCatalog全体を毎回promptへ入れない。Semantic Lockとdependency indexからtaskに関係するclosureを切り出し、必要なら理由を辿って拡張する。

## 13. Resolver実装は差し替え可能にする

初期Referenceで複雑なsolverを先に作らない。

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

最初の制約集合で単純なdeterministic resolverが十分ならそれを使う。Referenceでbacktrackingや複雑なversion range conflictが実際に必要になった段階で、PubGrub型のsolverへ差し替えられる境界を保つ。

「将来大規模になるから」という理由だけで、E0からsolver実装自体をProductの中心にしない。

## 14. 現行Reference実装との関係

PR #922の`tools/reference-semantic-extension-catalog`は、このPre-ADRより先に動き始めたReference実装である。2026-10-04時点では、次の境界まで実装している。

- Extension identity、namespace、semantic version、Source fingerprintをReference descriptorから読み込む
- concrete Extension IDとcompatible majorによる推移依存を決定論的に解決する
- dependency cycleとexact version重複を拒否する
- namespace ownership conflictをsilent overrideせずUnresolvedとして保持する
- descriptor由来のresolution fingerprintとSource来歴のevidence fingerprintを分離する
- Extension selectionをAcceptanceやRuntime Authorityへ昇格させない

この実装はE0-AとE0-Bの初期pressureとして利用できる。ただし、次はまだ実証していない。

- Semantic Interfaceを介したtyped dependency
- compatibility level / family
- Bridge Extensionによるcross-facet composition
- Accepted Semantic Lock
- root requirementからのwhy / reverse dependency
- structured conflict derivation
- Runtime Plugin更新とSemantic Set fingerprintの独立性

したがって、PR #922のReferenceを完成形へ広げてから設計を合わせるのではなく、現在の小さいresolverをnegative / positive controlとして使い、上記の不足をE0-B / E0-Cで順にPressureする。Pre-ADRは既存Referenceの成立Evidenceを否定せず、その先で必要になる不変条件を定義する。

## 15. ISSUE-000077で追加するdependency pressure

E0のdeclaration / discoveryに加えて、E0内で次の小さいdependency pressureを入れる。

### D1 — diamond

同じSemantic Interfaceへ複数経路から到達しても、一つの互換なproviderへ決定論的に収束する。

### D2 — incompatible range

二つのExtensionが両立しないversion / compatibility levelを要求した場合、load順に依存せずblocking conflictになる。

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

最初のSUI hierarchy Extensionをreal root consumerとして使い、D1〜D7の構造pressureにはReference-local Extensionを必要最小限だけ追加する。synthetic graphだけでSemantic Extension成立を主張しない。

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

E0-A〜Cを別remote branchへ分裂させる必要はない。同じISSUE-000077 workstreamで小さいcommitとしてmainへ収束させる。

## 17. ADRへ進める判断

次のEvidenceが揃うまで、具体的なresolver algorithmや最終manifest syntaxをADRで固定しない。

- real SUI consumerを含むdependency graph
- diamond / conflict / cycleのnegative control
- Semantic Lock repeatability
- load order非依存
- dependency reason / conflict explanation
- semantic updateによるimpact
- Runtime Plugin更新との分離

一方、以下は現時点からReference不変条件として扱う。

1. Semantic resolutionをEnvironment Runtime resolutionへ混ぜない。
2. silent override / last-winsを使わない。
3. Extension package cycleをAccepted graphへ入れない。
4. accepted semantic setをexact lock / fingerprintで再現可能にする。
5. std.*も同じExtension contractを使う。
6. Semantic dependency updateをShared AcceptanceなしでAccepted Worldへ反映しない。
7. dependency explanationを人とAIの共通surfaceにする。

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
- Bazel Bzlmod
  - https://docs.bazel.build/versions/main/bzlmod.html
- Dart pub, PubGrub solver design
  - https://github.com/dart-lang/pub/blob/master/doc/solver.md
