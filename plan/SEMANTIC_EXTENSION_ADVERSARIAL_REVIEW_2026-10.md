# Semantic Extension依存解決 敵対的評価 — 2026-10

- 状態: Adversarial Review / Pre-ADR補強
- 対象: `plan/SEMANTIC_EXTENSION_DEPENDENCY_PRE_ADR_2026-10.md`
- 実証レーン: ISSUE-000077
- 主担当要求: RQ-031 / RQ-032 / RQ-033
- 評価目的: 構造を肯定するためではなく、意味の再現性・依存解決・Authority分離を破れる反例を探す

## 1. 結論

Semantic ExtensionとRuntime Pluginを分離し、Semantic SetをEnvironmentより上流で固定する方向は妥当である。

一方、現行Pre-ADRをそのまま実装すると、少なくとも次の経路で構造を破れる。

1. greedyなversion選択が、実際には解がある依存graphを「解決不能」と誤判定する。
2. Semantic Interfaceを文字列の自己申告だけで提供できると、互換性のないproviderが同じInterfaceを名乗れる。
3. namespace所有を最初に現れた宣言へ委ねると、`std.*`を含むidentityを先取りできる。
4. exact versionだけをlockしても、root requirement、wiring、resolution policy、canonicalization規則が変われば同じlockから同じ意味を再現できない。
5. semantic Validationに使うEvaluatorをEnvironmentごとに選ぶと、developmentとproductionで同じsourceの受入可否が変わり得る。
6. optional ExtensionがdiagnosticやValidationを通じてAcceptanceへ影響すると、明示されないSemantic dependencyになる。
7. Standard Profileの内容をversionなしで変更できると、Profile名が同じでもroot requirement集合が変わる。
8. Bridge Extensionを機械的に増やすと、意味体系数に対して組合せが増え、Coreを小さくした代わりに外周の結合が爆発する。
9. semantic versionとは別の`compatibility level`を早期に固定すると、二つのversion軸が矛盾し得る。
10. 同じmeaning fingerprintを持つ宣言でも、Source / publisher / trustが異なれば供給元として同等とは限らない。
11. 巨大または敵対的な依存graphによってresolverやAI context生成を過剰に消費させられる。
12. Accepted sourceが参照しているExtensionを外したとき、意味を黙って消す実装にするとCanonical Worldが縮退する。

したがって、現行構造は**条件付きで妥当**と評価する。上記のうち1〜7はADRへ進む前にReference不変条件へ反映する。8〜12はReferenceで継続して圧力を掛ける。

## 2. 評価方法

次の観点で、現在のPre-ADRと`tools/reference-semantic-extension-catalog`を敵対的に読む。

- 正しい入力に対して誤ったconflictを返せないか
- 不正なExtensionが正当なidentity / Interface / namespaceへ割り込めないか
- lockが存在しても意味が変わる入力を外から持ち込めないか
- Runtime / Tooling / Environmentを経由してSemantic meaningへ逆流できないか
- 「optional」「Profile」「Bridge」等の便利な仕組みが暗黙依存を作らないか
- solverを高機能化した結果、別の第二正本や隠れたpolicyが生まれないか
- AI向け説明面がauthorityへ昇格しないか
- 規模を大きくしたときに依存graph自体が運用不能にならないか

外部方式は、採用理由だけでなく破綻条件も参照した。

- Cargo resolverは、最高versionを優先しつつ、必要ならbacktrackして別versionを試す。単純なhighest-compatible選択だけでは完全ではない。
- OSGi Resolverは、mandatory requirementをすべて満たす一貫した推移依存集合を要求し、policyをResolve Contextへ分離する。
- PubGrub系solverは、解が無い場合にその理由を導出木として保持する。
- Bazelの`compatibility_level`は過去の参考にはなるが、2026-10時点のBazel main documentationではdeprecated / no-opとされている。TEIで独立したcompatibility軸を固定する根拠には使わない。

参照:

- https://doc.rust-lang.org/cargo/reference/resolver.html
- https://docs.osgi.org/download/r8/osgi.core-8.0.0.pdf
- https://github.com/pubgrub-rs/pubgrub
- https://bazel.build/rules/lib/globals/module.html

## 3. A1 — greedy resolverはfalse conflictを作れる

現行Reference resolverは、同じExtension IDのcompatible majorから最高versionを先に選び、そのversionの依存を再帰的に固定する。

次のgraphを考える。

```text
root
  -> A@1
  -> B@1

A@1
  -> X@1

B@1
  -> Y@1

X 1.2.0
  -> Z@2

X 1.1.0
  -> Z@1

Y 1.0.0
  -> Z@1
```

`X 1.1.0`を選べば全体は解決できる。

しかしA側を先に辿って`X 1.2.0`を固定すると`Z@2`が選ばれ、その後B → Y → Z@1でconflictになる。これは「依存graphが解けない」のではなく、「greedyな選択が外れた」だけである。

### 判定

**Blocker。**

「deterministicであること」と「正しく解けること」は別である。常に同じ誤答を返しても再現性は満たせるが、resolverとしては不十分である。

### 設計反映

resolverには、少なくとも宣言したconstraint languageの範囲で次を要求する。

- 解が存在する場合、探索順や最初のhighest-compatible選択だけを理由にunsatisfiableと判定しない。
- 解が存在しない場合だけblocking conflictを返す。
- 複数の解がある場合、Acceptedなresolution policyで一意に決められなければ、solver固有の探索順で一つを選ばない。

E0-Bへ「greedy false conflict」のnegative controlを追加する。

## 4. A2 — Semantic Interfaceを文字列だけで名乗らせない

将来、

```text
requires interface relation-acyclic-validator/v1
```

を導入した場合、provider側が単に同じ文字列を宣言するだけでは不十分である。

悪意がなくても、二つのExtensionが`relation-acyclic-validator/v1`を異なる入出力・unknown規則・diagnostic意味で解釈できる。悪意があれば、偽providerが同じInterface名を先に提供できる。

### 判定

**Blocker。**

Semantic Interfaceはpackage名を隠す別名ではなく、意味契約でなければならない。

### 設計反映

Semantic Interface自体に少なくとも次を持たせる。

- logical identity
- namespace authority
- contract version
- contract / declaration fingerprint
- Source / provenance
- required input / output semantics
- unknown / failure semantics
- provider conformance requirement

providerの`implements`宣言だけでは適合済みと扱わない。

同じInterfaceを複数providerが満たす場合も、明示的なselection policyまたはAccepted wiringが無ければ、package IDやversion順で暗黙選択しない。

### 2026-10-05 Reference candidate

PR #927ではA2の反例に対し、Semantic Interfaceをversioned contractとしてReference実装へ追加した。contractはidentity、version、正規化fingerprint、input / output semantics、unknown / failure semantics、provider conformance profileを持つ。

provider descriptorはInterface名だけでなくexact contract versionとcontract fingerprintを宣言する。ただし、この宣言だけではprovider候補として採用しない。resolverへ別入力のConformance Evidenceが必要であり、そのEvidenceはprovider、Interface、exact contract fingerprint、conformance profileに結び付く。

複数の適合providerが存在する場合、package IDやversion順では選ばない。明示wiringが無ければUnresolvedにし、明示wiringもexact contractとConformance条件を満たすproviderだけを受け入れる。provider wiringはSemantic resolution identityに含め、Conformance Evidence自体のfingerprintはEvidence identityへ分離する。

これにより「同じInterface文字列を名乗ればproviderになれる」という反例はReference candidate上では塞いだ。一方、Conformance Evidenceの発行Authority、実際のevaluator conformance suite、SUI hierarchyを使った実consumer pressureは未実証である。A2をProduction上の解消済みBlockerとはまだ扱わない。
## 5. A3 — namespace ownershipをfirst claimantへ与えない

現行Referenceは複数Extensionが同じnamespaceを所有する場合をUnresolvedにできる。一方、まだ誰も宣言していないnamespaceでは、最初のExtensionがその名前を使える。

この方式をproductionへ広げると、第三者Extensionが先に`std.data`等を宣言するdependency confusionが成立し得る。

### 判定

**Blocker。**

namespaceはCatalog内の競合検出だけでなく、authorityの問題である。

### 設計反映

- `std.*`はTEI standard authorityへ予約する。
- namespace owner / delegateはAccepted metadataとして明示する。
- 「最初にloadされた」「最初にregistryへ現れた」をownershipの根拠にしない。
- subnamespace委譲が必要なら、ownerからの明示delegateとして表す。
- semantic identityとpackage distribution sourceを分ける。

### 2026-10-05 Reference boundary

PR #927ではnamespace authorityをExtension descriptorから分離し、Accepted Authority Evidenceを別入力にするReference candidateへ進んだ。`std` / `std.*`はauthority無しでは`namespace-authority-missing`となり、Catalogへ先に現れたExtensionがfirst claimantとして取得できない。

Authority Evidenceがnamespaceとowner Extensionを明示した場合だけ、そのExtensionのsemantic contributionを公開する。同じnamespaceの別claimantは`namespace-authority-mismatch`として残し、Coreが既に所有するnamespaceはAuthority Evidenceでも上書きできない。Authority Evidence fingerprintはprovenanceとしてEvidence identityへ含め、namespaceとownerが同じままEvidenceだけを更新してもSemantic resolution identityを変えない。

これでfirst claimant経路と正当なstandard Extensionの導入経路を分ける候補はできた。ただしsubnamespace委譲、publisher / Source trust、Authority Evidence自体の発行Authorityは未実証である。A3をProduction上の解消済みBlockerとはまだ扱わない。
## 6. A4 — exact versionだけではlockにならない

次が変わると、Extensionのexact versionが同じでも解釈やgraphが変わり得る。

- root requirements
- requirement -> provider wiring
- Semantic Interface contract
- descriptor canonicalization
- resolution policy
- lock formatの解釈
- Accepted source側のExtension参照

### 判定

**Blocker。**

Semantic Lockはpackage version一覧では足りない。

### 設計反映

Semantic Lockまたは同等表現へ、少なくとも次を含める。

- lock format version
- root requirementsとそのfingerprint
- exact Extension identity / semantic version
- declaration fingerprint
- Semantic Interface contract fingerprint
- resolved requirement -> provider wiring
- resolution policy identity / version
- fingerprint canonicalization version
- unresolvedが0であることを示せる状態

既存lockがある場合は、それを候補探索のhintとして扱わず、まずexact graphとして検証する。Catalogへ新versionが増えただけでは再解決しない。

root requirementsまたはpolicyが変わった場合は、既存lockをそのまま有効とは扱わず、candidate re-resolutionとShared Acceptanceへ進める。

## 7. A5 — EvaluatorをEnvironmentへ任せるとmeaningが揺れる

現行ISSUE-000077には、同じSemantic Extensionを複数Pluginが評価でき、EnvironmentがCapability negotiationで実装を選べる案がある。

しかし、そのEvaluatorのpass / fail / unknownがShared AcceptanceやConformanceへ影響するなら、developmentでEvaluator A、productionでEvaluator Bを選ぶだけで同じsourceの受入可否が変わり得る。

Evaluator outputがCanonical meaningそのものではなくても、Accepted Worldへ入れる条件を変えるならsemanticに重要である。

### 判定

**Blocker。**

Runtime providerの交換可能性と、Semantic Validationの交換可能性を同一視してはいけない。

### 設計反映

- Acceptance / Conformanceへ使うSemantic Evaluatorの解決scopeをEnvironment Runtimeから分離する。
- Validation Evidenceへevaluator contract version、implementation identity / artifact digest、conformance Evidenceを紐付ける。
- 複数実装を交換可能と扱うには、同じcontractに対するConformance Evidenceを要求する。
- EnvironmentがRuntime用途のproviderを選ぶことは許容しても、その選択でAccepted semantic validityを再解釈しない。
- evaluator不在、非対応version、結果不一致はpassへ丸めない。

## 8. A6 — optional contributionからAcceptanceへ逆流できる

「既存meaningを変えず、diagnosticやProjectionだけを追加するoptional Extension」は一見安全に見える。

しかし、そのdiagnosticがblockingになったり、Shared Acceptance Previewの判定材料になったりすれば、存在するだけでAccepted Worldが変わり得る。

### 判定

**High。**

「meaningを書き換えない」だけではoptional性の条件として弱い。

### 設計反映

Contributionの具体enumを今すぐ固定しないが、少なくとも次を区別する。

```text
Accepted meaning / validityへ影響する contribution
  -> Semantic Setへ明示
  -> lock / fingerprint / Acceptance対象

advisory diagnostic / non-authoritative projection
  -> optionalにできる
  -> absence / presenceだけでAccept / Rejectを変えない
```

Validationを通じてAcceptanceへ影響するものは、optional Toolingとして扱わない。

## 9. A7 — Standard Profileも意味に影響する

Pre-ADRではStandard Profileをroot requirement setとして提供し、Profile自体をSemantic authorityにはしないとしている。

この分離は妥当だが、Profile定義が変われば選ばれるSemantic Setが変わる。Profileはauthorityではなくても、意味解釈に影響する入力である。

### 判定

**High。**

### 設計反映

Standard Profileにはidentity、version、Source、fingerprintを持たせる。

Accepted ApplicationがProfileを参照する場合は、Profileのexact identityまたは展開済みroot requirement fingerprintをSemantic Lockへ反映する。

同じProfile名の内容を更新してAccepted Semantic Setを黙って変えない。

## 10. A8 — Bridgeは循環を減らすが組合せ爆発を起こせる

Bridge Extensionは相互依存を避けるうえで有効である。

一方、意味体系がN個あり、各組合せにBridgeを作ると最大でO(N²)に近い接続面が生まれる。三体系以上の整合性ではさらに複雑になる。

### 判定

**Highだが、Bridge方針を棄却する理由ではない。**

### 設計反映

Bridgeは「cross-facet ruleがあるたびに作る標準手段」ではなく、次の順で使う。

1. 既存Semantic Interface / Relation contractで表せないか確認する。
2. 一方の体系に属さない横断意味であることを確認する。
3. Bridgeとして独立させる。
4. 同種Bridgeが反復する場合、共通Interface / Relation contractの不足を再評価する。
5. Bridge graphのfan-in / fan-out、個数、重複constraintをReferenceで測る。

巨大なumbrella Extensionへまとめて複雑さを隠すこともしない。

## 11. A9 — compatibility levelを第二のversion軸として固定しない

初期Pre-ADRではsemantic versionとは別に`compatibility level / family`を必須候補としていた。

しかし、semantic version、version range、family、levelを同時に持つと、どの軸が互換性を決めるかが曖昧になる。

さらに、参考にしていたBazelの`compatibility_level`は、2026-10時点のBazel main documentationではdeprecated / no-opである。

### 判定

**設計の過剰固定。**

### 設計反映

現段階で不変条件にするのは次だけとする。

- semantic versionとRuntime binary versionを分ける。
- dependency compatibilityを明示する。
- Accepted Worldはexact lockで固定する。
- 互換性のない意味を同じCanonical subjectへ暗黙混在させない。

別のcompatibility family / epoch / levelが必要かは、version rangeだけで表現できない実Referenceが出てから決める。

## 12. A10 — meaning identityとsupply-chain trustを混ぜない

現行Referenceは、正規化descriptorから得る`resolutionFingerprint`と、Source来歴を含む`evidenceFingerprint`を分けている。この分離は敵対的評価でも維持すべきである。

同じdescriptor contentを別のSourceから取得した場合、meaningとして同一でも、信頼できる供給元かどうかは別問題である。

### 判定

**現行方向は妥当。**

### 設計反映

- semantic fingerprintが同じことをpublisher trustの同一性とみなさない。
- namespace authority、署名、provenance、distribution policyはEvidence / supply-chain側で検証する。
- trust policyの違いだけでsemantic fingerprintを変えない。
- ただし、信頼条件を満たさないSourceをAccepted Semantic Setへ導入してよいとは扱わない。

## 13. A11 — resolver自体をDoS対象にできる

大量のversion候補、深いdependency chain、広いInterface provider集合、意図的なconflictを与えると、backtracking solverは計算量を消費できる。

AIへdependency closureを展開する経路も同様である。

### 判定

**Medium。**

### 設計反映

- resolutionには明示的なresource / complexity budgetを設ける。
- budget超過を「依存なし」「conflict」と誤認せず、専用のUnresolved / diagnosticとして返す。
- AI contextは完全graphを毎回展開せず、scoped closureとdrill-downを使う。
- budget値そのものを意味仕様へ埋め込まず、operational policyとして分離する。

## 14. A12 — Extension removalで意味を消してはいけない

Accepted sourceがExtension固有symbol / constraintを参照している状態で、そのExtensionをCatalogまたはSemantic Setから外す。

このとき参照を無視して残りのWorldだけをloadできる実装は、意味を静かに削る。

### 判定

**High。**

### 設計反映

- Accepted sourceから参照されるExtensionが欠けた場合はUnresolved / blocking diagnosticにする。
- Extension removalをSemantic Changeとしてimpactへ載せる。
- unrelated subjectだけは影響を受けないことを同時に確認する。
- 「Extensionを削除しても無関係subjectは変わらない」と「参照しているsubjectまで成功扱いする」を混同しない。

## 15. 敵対的評価後の構造

更新後の構造は次を基本とする。

```text
Accepted source
   +
Accepted root requirements / Profile identity
   +
Namespace / Interface authority contracts
        |
        v
Semantic Resolver
  - complete for declared constraint language
  - no traversal-order winner
  - explicit provider selection policy
  - bounded resolution
        |
        v
Resolved candidate graph
  - no unresolved
  - exact interface wiring
  - conflict derivation
        |
        v
Semantic Lock
  - root fingerprint
  - exact versions
  - declaration / interface fingerprints
  - wiring
  - policy / canonicalization version
        |
        v
Semantic Set fingerprint
        |
        +--> Shared Acceptance / semantic diff / impact
        |
        +--> Semantic Evaluator resolution
                - Environment Runtimeとは別scope
                - implementation / artifact / conformance Evidenceを固定
        |
        +--> Runtime realization
                - Environment Plugin Lock
                - Permission / Resource / Sandbox
```

意味の宣言、意味の依存解決、意味のValidation、Runtime実行を一つのPlugin graphへ戻さない。

## 16. Referenceへ追加する敵対的ケース

ISSUE-000077へ次を追加する。

| ID | ケース | 期待 |
| --- | --- | --- |
| AX-01 | higher versionがtransitive conflict、lower compatible versionなら解ける | false conflictにせず有効解へ到達 |
| AX-02 | 同じSemantic Interfaceを二providerが提供 | accepted wiringが無ければUnresolved |
| AX-03 | Interface名だけ同じでcontract fingerprintが異なる | 同一Interface扱いしない |
| AX-04 | 第三者Extensionが`std.*` namespaceを宣言 | namespace authorityで拒否 |
| AX-05 | root requirementsだけ変更しlockを再利用 | stale lockとして拒否 |
| AX-06 | resolution policy / canonicalization version変更 | lock再検証またはcandidate updateを要求 |
| AX-07 | optional diagnosticがAcceptance結果を変えようとする | Semantic dependencyへの明示を要求 |
| AX-08 | Profile内容だけ変更 | Semantic Set変更候補として検出 |
| AX-09 | EnvironmentごとにEvaluator A/Bが異なる結果を返す | Accepted validityをEnvironment選択へ委ねない |
| AX-10 | Extension removal後もaccepted sourceがsymbolを参照 | blocking Unresolved |
| AX-11 | Sourceだけ異なる同一descriptor | semantic identityは同じ、Evidence / trustは別 |
| AX-12 | resolution budget超過 | conflictへ偽装せず専用Unresolved |

## 17. A13〜A18 — Plugin化による性能・storage・可搬性の劣化

Semantic Extensionや標準Pluginの数が増えたとき、logicalな分割をそのまま物理分割へ写すと、拡張性のためにTarget品質を失う。

### A13 — one Extension = one process

標準機能100個を100 processとして起動する構成を考える。個々のPluginが軽量でも、process startup、scheduler、IPC、memory、supervisionの固定費が積み上がる。

**判定: Blocker。**

logical Plugin identityはprocess boundaryを要求しない。process isolationが必要な実装だけをprocessにし、標準機能はbuiltin / linked / coalesced realizationを選べる必要がある。

### A14 — one Extension = one Go Wasm

Web Targetで各Extensionを独立したGo `js/wasm` binaryとして配布すると、language runtimeやsupport codeの固定費が繰り返される可能性がある。

Go公式WebAssembly Wikiは、標準GoのWasm binaryが小さなプログラムでも比較的大きくなりやすく、library importでさらに増えることを説明している。

**判定: Blocker。**

Extension contractとcompiler / module boundaryを分離し、複数Extensionを一つのWasm artifactへまとめたり、TinyGo等の小型toolchainや別実装をTargetごとに選べるようにする。

### A15 — eager download everything

Semantic SetにExtensionが含まれることを理由に、その全実行artifactをWeb初期表示前にdownloadすると、初期転送量と起動時間がExtension数へ比例しやすい。

**判定: High。**

意味のdeclaration / Lockと実行codeを分離する。意味解決に必要なmetadataはeagerに利用できるようにし、実行artifactはreachabilityとTarget条件に応じてprune / lazy loadできるようにする。

### A16 — physical packagingをSemantic fingerprintへ混ぜる

同じSemantic SetをDesktopではstatic link、WebではWasm bundleとして実現したとき、artifact分割方法をSemantic Set fingerprintへ含めるとTargetごとに別meaningとして扱われる。

**判定: Blocker。**

Semantic Set fingerprintとTarget artifact fingerprintを分離する。Target artifactからSemantic Set / Plugin Lockへ逆追跡できればよい。

### A17 — browser非対応Runtimeを暗黙fallbackする

process-only ProviderがWeb Targetで利用できないため、別Providerへ暗黙fallbackし、その実装差で結果や意味が変わるケースを考える。

**判定: Blocker。**

Targetが必要なRuntime classを満たせなければbuild / Conformance段階で非対応またはUnresolvedとする。fallbackを行う場合も、Accepted Binding / policyで明示する。

### A18 — shared runtime / artifact duplication

複数Pluginが同じWasm runtime、library、model artifactをそれぞれpackageへ複製すると、storageとnetwork costがlogical Plugin数に比例して増える。

**判定: High。**

content digestによるdeduplication、shared runtime、link-time coalescing等をTarget packagingで選べるようにする。logical Pluginごとのprovenanceは、物理byteの複製ではなくmappingで保持する。

## 18. 性能・可搬性を含めた追加Reference条件

次をReferenceで確認する。

| ID | ケース | 期待 |
| --- | --- | --- |
| PX-01 | 10個のdeclaration-only Extension | Runtime artifact増加を要求しない |
| PX-02 | 複数standard Extensionを一つのTarget artifactへ統合 | logical identity / Evidenceへ逆追跡できる |
| PX-03 | 未使用evaluatorを含むCatalog | Target artifactから実行codeを除外できる |
| PX-04 | 複数Pluginが同じsupport artifactを参照 | byteを重複格納しない余地がある |
| PX-05 | Web Target | native process起動なしで最低限の標準Semantic Setを利用できる |
| PX-06 | process-only implementationしかないCapability | 暗黙fallbackせず非対応 / Unresolvedになる |
| PX-07 | 同一Semantic SetをWeb / Desktopへbuild | Semantic Set fingerprintは同じ、Target artifact fingerprintは別 |
| PX-08 | Extension数を段階的に増やす | startup / transfer / memory / storageの増分を観測できる |

## 19. ADRへ進むための追加条件

従来の条件に加え、次を満たすまでresolver / lock / Interface contractをADRで固定しない。

- AX-01でgreedy false conflictを排除できる
- Semantic Interfaceが自己申告文字列ではなくversioned contractとして成立する
- `std.*` namespace authorityをfirst claimantから分離できる
- Semantic Lockがroot / wiring / policy / canonicalizationを含めて再現可能である
- semantic Validation用EvaluatorがEnvironment Runtime selectionから独立する
- optional contributionとAcceptance影響の境界を説明できる
- Profile変更をSemantic Changeとして検出できる
- logical Extension / Pluginとphysical artifact / process境界を分離できる
- Web Targetでnative processを要求せずSemantic Extensionを成立させられる
- unused executable implementationの除外とshared artifactの重複排除をTarget packagingで行える
- Target packaging差がSemantic Set fingerprintを変えない

この条件が成立した後でも、solver algorithmそのものをCoreのProduct Valueにはしない。必要なのは、意味の再現性と説明可能性を保つ解決契約である。
