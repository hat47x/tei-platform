# ISSUE-000077: Semantic ExtensionをCore変更なしで追加できる最小Referenceを実証する

- 状態: In Progress — E0 candidate統合済み、敵対的dependency pressureを追加中
- 起点: RQ-031 / RQ-032 / RQ-033、PR #919
- 関連: ISSUE-000040 / ISSUE-000047 / ISSUE-000052
- Product value: 固定モデルではなく、後から意味体系を安全に追加・合成できること
- 主なReference pressure: SUI hierarchy adoption
- 依存解決設計基線: `plan/SEMANTIC_EXTENSION_DEPENDENCY_PRE_ADR_2026-10.md`
- 敵対的評価: `plan/SEMANTIC_EXTENSION_ADVERSARIAL_REVIEW_2026-10.md`

## 目的

TEIのCoreへ新しい業務概念やValidation enumを追加しなくても、Plugin / DSL側から新しい意味論を宣言し、既存のCanonical meaningと関係付け、Tool / AIが発見・検証・再利用できることを最小Referenceで実証します。

このIssueでいうSemantic Extensionは、現在のprocess Pluginと同じものではありません。

現在のPluginは、`docs/plugins.md`が定めるとおり **Capability実装を配布する単位**です。Runtime、Permission、Artifact、Plugin Lock等の実行境界を持ちます。

一方、本Issueで追加したいものは、次のような**意味の宣言**です。

```text
Semantic Extension
  what concept / rule exists
  where it applies
  what parameters / dependencies it has
  what conflicts make it unresolved
  what deterministic validation is required
  how Tool / AI discovers it

Evaluator / Runtime Plugin
  how a declared rule is evaluated or realized
```

両者は配布上同梱できても、identityとAuthorityを同一にしません。

## 防ぎたい失敗

RQ-031〜033を実装するとき、次を防ぎます。

1. 新しい意味概念が必要になるたびにCoreのGo enum / parser / IR kindを増やす。
2. Runtime Pluginがコードを追加しただけで、その意味がCanonical authorityになったことにする。
3. Plugin package名やload順をsemantic identityや優先順位として使う。
4. 同じnamespaceへ互換性のない意味定義が来たとき、後勝ちで黙って上書きする。
5. 意味宣言が存在するだけで、Runtime execution / Database / Network等のAuthorityまで得る。
6. evaluatorが無い、versionが合わない、依存が欠けている状態を「Validation成功」とみなす。
7. AIが未知のextensionを無視して、Core内蔵語彙だけで近似実装する。
8. 一つのSUI固有gapだけを根拠に、`std.*`の標準語彙へ昇格する。

## 最初のReference pressure — SUI hierarchy

`ISSUE-000052`では、SUIの`IslandHierarchyContractV1`をTEIへread-only mappingした結果、現行Coreだけでは次をlosslessに表せないことが確認されています。

- nullable parent、すなわち0..1 presence
- parent targetの存在
- parent-child整合
- cycle禁止

現状はこれらをReference-local shadow validationとして扱っています。

このgapを、Coreに`acyclic`等の新しいValidationKindを直接追加せず、Semantic Extensionで表現できるかを最初のpressureにします。

## 最初に標準化しない

最初のextension identityは`std.*`にしません。

概念上はReference-localなidentityを使います。

例:

```text
reference.hierarchy.integrity
```

この名称自体を最終仕様として固定しません。

SUI以外の第二Referenceでも同じ意味が必要であり、同じ契約で再利用できることを確認した後で、標準namespaceへ昇格するかを判断します。

## Semantic Extensionの最小契約

最終syntaxを先に固定しません。まずmachine-readable contractとして、少なくとも次を表現できることを要求します。

### Identity

- extension logical identity
- namespace
- semantic version
- source / package provenance

### Applicability

- どの既存subject / relation / definitionへ適用できるか
- 適用に必要な既存semantic identity
- parameter schema
- 必須dependency extension
- compatible version range

### Meaning

- extensionが導入するsymbol / constraint / relationの意味
- parameterの意味
- 成立条件
- unknown / unsupportedの扱い
- 他extensionとのconflict key

### Validation / Conformance

- declarationを静的に検証する規則
- instance / definitionへ適用した結果を検証する規則
- evaluatorが必要な場合、その**Semantic evaluator requirement**
- evaluator不在時にfail-openしないこと

### Tool / AI discovery

- Source Index / Knowledge Contextからextension identityへ辿れる
- 適用対象から「利用可能なextension」を列挙できる
- extensionからSource / version / dependencies / Validationへ戻れる
- AIがcustom implementationより前にextensionを候補として発見できる

## 依存規模を成立条件へ含める

標準機能を含む多くの意味体系をSemantic Extensionとして提供すると、Extensionの推移依存は大きくなります。このReferenceでは、一つのExtensionを読み込めるだけでは成立としません。

詳細な設計判断は`plan/SEMANTIC_EXTENSION_DEPENDENCY_PRE_ADR_2026-10.md`を基線とし、少なくとも次の条件をReferenceでは変えないものとして扱います。

- Semantic Extensionの依存解決をEnvironment固有のRuntime Plugin解決へ混ぜない。
- 特定packageのidentityが必要な依存と、Semantic Interfaceを満たせばよい依存を区別する。
- 異なる意味体系を横断する規則は、どちらか一方へ相互依存を埋め込む前にBridge Extensionを検討する。
- Extension packageのdependency cycleはAccepted graphへ入れない。
- 同じ入力のresolved graphをload順やcatalog列挙順で変えない。
- Accepted Semantic Setはexact version、declaration fingerprint、解決済みdependency / interface wiringを再現できる形で固定する。
- compatibleな新versionが見つかってもAccepted Semantic Setを自動更新しない。
- 標準`std.*`を追加する段階でも、同じdependency / conflict / lock / discovery contractを利用する。
- resolverはresolved setだけでなく、導入理由とconflict causeを人とAIへ説明できる。
- 有効解が存在する依存graphを、greedyなversion選択や探索順だけを理由にconflict扱いしない。
- Semantic Interfaceはversioned contractとしてidentity / fingerprint / Sourceへ辿れ、providerの自己申告だけで適合済みとしない。
- namespace ownershipをfirst claimantへ与えず、`std.*`はstandard authorityへ予約する。
- Acceptanceに使うSemantic Evaluatorの選択をEnvironment Runtimeのprovider選択へ委ねない。
- root requirements、resolution policy、canonicalization規則が変わったSemantic Lockをそのまま再利用しない。

最初から高度なsolver実装を固定しません。E0では、大規模化した後でもsolver実装を差し替えられるよう、Requirement、resolved graph、conflict explanationの入出力契約を先に定めます。

### 物理packagingをExtension境界から分離する

標準機能をPlugin / Semantic Extensionとして分割しても、一つのExtensionごとに別process、別binary、別Wasm moduleを要求しません。

Referenceでは、logical identityとTarget realizationを分けます。

- declaration-only ExtensionはRuntime artifactなしで成立できる
- 複数Extensionを一つのbuiltin / linked / Wasm artifactへ統合できる
- 未使用の実行implementationをTarget artifactから除外できる
- 共通runtime / support artifactをlogical Pluginごとに複製しない
- 実行artifactをlazy loadしても、Semantic Setの意味は消えない
- 同じSemantic SetをWeb / Desktop等で異なるpackagingへ変換しても、Semantic Set fingerprintはTarget packagingに依存しない
- Web Targetではnative processを必須にしない
- Target非対応のRuntime要件を暗黙fallbackで隠さない

物理最適化後も、diagnostic / Evidence / impactから元のlogical Extension / Pluginへ戻れることを要求します。

## 意味と実装を分ける

Semantic Extensionそのものへprocess実行Authorityを持たせません。

```text
semantic declaration
  reference.hierarchy.integrity
        ↓ requires evaluator capability if needed
semantic evaluator requirement
        ↓
Capability resolution
        ↓
Runtime Plugin
        ↓
existing Permission / Lock / Sandbox / Lease boundaries
```

extensionを読み込めたことは、evaluator processを実行してよいことを意味しません。

逆に、evaluator Pluginがinstallされていても、そのPluginがextensionの意味を勝手に変更できません。

同一Semantic Extensionを複数の実装Pluginが評価できる構造は許容します。ただし、そのEvaluator結果がAcceptance / Conformanceへ影響する場合、実装選択をEnvironment固有のCapability negotiationへ委ねません。Semantic Validation用のEvaluator解決scopeをRuntime Environmentから分け、選択したcontract version、implementation identity / artifact、Conformance EvidenceをValidation Evidenceへ紐付けます。

## 最初のReference declaration

SUI hierarchyの最初の宣言では、少なくとも次を分けます。

### H1 — optional parent presence

parent relationは各childについて0または1件である。

これは「single-valued relation」と「parentが無くてもよい」を分けて表現します。

### H2 — parent target existence

parent referenceがある場合、そのtarget identityが存在する。

### H3 — acyclic

parent relationを辿ってcycleを形成してはならない。

### H4 — parent / child consistency

source contractがparentとchildrenの両方向を持つ場合、両者が矛盾してはならない。

最初からこの四つを一つの`hierarchy`booleanへ畳みません。

第二Referenceで個別再利用できる可能性を残します。

## E0 — declaration / discovery / dependency resolutionを先に実証する

SUI instance validationの実行まで一度に進めません。E0は一つのworkstreamの中でE0-A〜E0-Cに分けます。

### E0-A — declaration / discovery

1. extension declarationをSourceからloadできる。
2. logical identity / version / dependencyがSource Indexへ入る。
3. SUI hierarchy definitionから適用extensionへexactに辿れる。
4. Knowledge Contextからextension Source / version / applicabilityへ戻れる。
5. extensionが無い状態をCore標準Validationへ暗黙変換しない。
6. incompatible duplicate definitionをload順で上書きしない。
7. extensionを削除しても無関係なCanonical subjectsの意味は変わらない。

### E0-B — typed dependency / conflict

Reference-localな小規模の依存graphを追加し、次を検証します。

- diamond dependencyが一つの互換なproviderへ決定論的に収束する
- package identity dependencyとSemantic Interface dependencyを区別する
- Semantic Interfaceがidentity / contract version / fingerprint / Sourceを持つ
- providerの`implements`自己申告だけでInterface適合済みとしない
- 複数providerが同じInterfaceを満たす場合、Accepted wiringまたは明示policyなしに暗黙選択しない
- incompatible version constraintをload順に依存せずblocking conflictへする
- higher compatible versionでは衝突するがlower compatible versionなら解けるgraphをfalse conflictにしない
- package dependency cycleを検出する
- Bridge Extensionを外した場合、両側の独立Extensionが互いを暗黙requireしない
- optional Extensionの存在だけで既存meaningを変更しない
- optional diagnostic / ProjectionがAcceptanceを変える場合はSemantic dependencyへの明示を要求する
- 第三者Extensionが`std.*` namespaceをfirst claimantとして取得できない

SUI hierarchy Extensionを実際のroot consumerとして使い、syntheticなdependency graphだけでRQ-032の成立を主張しません。

2026-10-04の実装候補として、PR #927でAX-01のbacktracking resolverを追加しています。compatible major内では高いversionを先に試しつつ、推移依存が衝突した場合は低いcompatible versionへ戻ります。version固有のdependency cycleから代替versionへ退避するcaseと、代替が無いcycleを拒否するcaseもtestへ追加しています。

この時点ではcandidateです。独立したGo 1.23.2 probeでは同じ探索アルゴリズムのfalse-conflict / version-specific-cycle caseがPASSしていますが、repository exact-head focused testとfull health gateは未実施です。したがってAcceptance checkboxやEvidence tierはまだ更新しません。

Productionの`internal/plugin.ResolveEnvironment`も、2026-10-04時点ではhighest-compatibleを先に固定するDFSを使っています。同じ種類のfalse conflictを起こし得ますが、Semantic resolutionとRuntime Plugin resolutionは別scopeとして維持します。PR #927でReferenceの解法とnegative controlを確定した後、同じ反例をRuntime Plugin resolverへ適用し、共通solver primitiveへ寄せるべきか、別実装のまま同じConformanceを要求するべきかを判断します。Referenceが通る前にProduction resolverを同時変更しません。

PR #927では、複数の有効解がある場合の選択規則も隠れた探索順にしないため、Reference catalogへ`resolutionPolicy`を追加しました。現在のcandidate policyは`highest-compatible-complete-search/by-extension-id/v0`です。policy identityをSemantic resolution fingerprintへ含め、policy変更を意味解決上の差として検出します。このpolicy名や規則自体はReference candidateであり、ADR成立前にProduction contractへ固定しません。

同PRでは、backtrackingの探索量を制御するためのoperational budget境界も追加しています。candidate attempt上限を超えた場合は専用の`semantic resolution search budget exceeded`として返し、dependency conflictと区別します。十分なbudgetを与えた場合にunbounded解決と同じ結果になることもtestします。budget値自体はSemantic Set fingerprintへ入れず、意味仕様ではなく運用policyとして維持します。

#### A2 / A3 candidate response — 2026-10-05

PR #927では、敵対的評価A2に対するReference candidateとしてSemantic Interfaceのcontractとprovider wiringを追加した。package identity dependencyとは別に、Interface identity、contract version、正規化contract fingerprint、input / output semantics、unknown / failure semantics、provider conformance profileを宣言できる。

providerの`implements`相当の自己申告だけでは適合済みと扱わない。provider descriptorはInterface identity、exact contract version、contract fingerprintを宣言するが、resolverは別入力のConformance Evidenceを要求する。Conformance Evidenceはprovider identity、Interface identity、exact contract fingerprint、conformance profileへ結び付ける。contract fingerprintまたはprofileが異なるEvidenceを流用しない。

適合するproviderが一つならwiringを確定できる。複数providerが残る場合はpackage ID、version、Catalog列挙順で選ばず、明示wiringが無ければ`interface-provider-ambiguous`としてUnresolvedにする。明示wiringも、指定providerがexact contractとConformance条件を満たさなければ拒否する。

provider wiringはSemantic resolution identityへ含める。同じcontract、provider、conformance profileに対するEvidenceの更新だけはmeaningの変更とせず、`evidenceFingerprint`側で追跡する。これによりSemantic identityとConformance provenanceを分離する。

A3については、namespace authorityをExtension descriptorから分離し、AcceptedなAuthority Evidenceを別入力にするcandidateへ進んだ。`std` / `std.*`はauthority無しでは`namespace-authority-missing`となり、first claimantでは取得できない。Accepted authorityがnamespaceとowner Extensionを明示した場合だけ、そのExtensionのsemantic contributionを公開する。Core所有namespaceはauthority入力でも上書きできず、同じnamespaceの他claimantは`namespace-authority-mismatch`として残す。Authority Evidence fingerprintはEvidence identityへ含め、同じnamespace / ownerに対するEvidence更新だけではSemantic resolution identityを変えない。subnamespace delegateとpublisher trustの最終contractは未実装である。

これらは構造pressureのcandidateであり、A2 / A3をProduct Evidenceとして解消したとは扱わない。exact-head focused testとrepository healthに加え、SUI hierarchyを実root consumerとしてInterface依存へ接続し、provider Conformance Evidenceを実際のValidationへ結び付けるまでE0-Bの完了状態は更新しない。
### E0-C — Semantic Lock / explanation

同じroot requirements、同じCatalog、同じAccepted sourceからは、列挙順を変えても同じresolved graphとSemantic Set fingerprintが得られることを確認します。

最低限、次の情報へ機械可読な形で辿れることを要求します。

- root requirementsとそのfingerprint
- exact resolved extension identity / semantic version
- declaration fingerprint
- Semantic Interface contract fingerprint
- requirement -> provider / Semantic Interface wiring
- resolution policy identity / version
- fingerprint canonicalization version
- root requirementから各Extensionまでのdependency path
- reverse dependency
- conflict cause / unresolved requirement

Runtime Pluginの実装versionだけが変わり、semantic declarationが同じ場合は、Semantic Set fingerprintに不要な差分が入らないこともnegative controlで確認します。

root requirements、resolution policy、canonicalization versionのいずれかが変わった場合は、既存lockを有効なまま再利用しないことも確認します。Unresolvedを含むgraphからAccepted Semantic Lockを作りません。

E0ではRuntime Pluginを起動しない。

#### Semantic Lock candidate — 2026-10-05

PR #927ではA4に対するReference candidateとしてSemantic Lockを追加した。Unresolvedを含むcatalogからはlockを作成しない。lockはroot requirements、requirements fingerprint、resolution policy、canonicalization policy、Core subject fingerprint、exact Extension identity / version / descriptor fingerprint、namespace owner、Semantic Interface contract fingerprint、provider wiring、Semantic Set fingerprintを保持する。

既存lockの再検証では、現在のCatalogを通常のresolverで再解決しない。lockに固定したexact Extension versionだけをavailable declarationsから取り出し、同一descriptor fingerprintを確認したうえでsemantic graphを再構成する。したがってCatalogへcompatibleな新versionが増えただけでは既存lockを自動更新しない。一方、同一versionのdeclaration変更、root requirements、resolution policy、canonicalization、Interface wiring、namespace ownerが変わればreplayを拒否する。

Semantic LockはSource / Conformance / namespace Authority Evidenceのfingerprintをmeaning identityへ含めない。Interface wiringとnamespace ownerは意味解決に含めるが、同じcontract / provider / ownerに対するEvidence更新だけではSemantic Set fingerprintを変えない。providerが唯一候補だったか明示wiringだったかという選択経路もmeaning identityから外す。

CLIには`-semantic-lock-out`と`-verify-semantic-lock`を追加した。後者が保証するのはexact graphの`semanticReplayValid`であり、現在のConformance Evidence、namespace Authority Evidence、publisher trust、Runtime Authorityの有効性ではない。A10のsupply-chain trust境界と混同しない。

current Evidenceについては、`-verify-semantic-lock-evidence`を追加した。semantic replayが成功した後、lockへ固定したprovider / Interface contract / conformance profileと、namespace / ownerに対応するAccepted Evidenceが現在の入力に存在することを別に確認する。Evidence fingerprint自体はmeaning identityへ固定しないため、同じcontract / profileまたはnamespace / ownerに対するEvidence更新をsemantic changeにはしない。

この検査が確認するのはlockとAccepted Evidenceの結び付きまでである。Conformance Evidenceやnamespace Authority Evidenceを誰が発行できるか、publisher trustをどう検証するかは引き続き未実証である。

このlock形式もReference candidateである。exact-head focused testとrepository healthを未実行であり、Accepted source側のExtension参照、lockのShared Acceptance接続、Evidence発行Authority / supply-chain trustも未実証なのでE0-Cの完了状態は更新しない。
## E1 — deterministic evaluator boundary

E0成立後、H1〜H4のうちCore declarative primitiveだけでは評価できないものについて、deterministic evaluator boundaryを追加します。

Evaluatorは次を守ります。

- inputはexplicit subject / extension identity / accepted definition snapshotへboundする
- outputはpass / fail / unknownとdiagnostic Evidenceを返す
- evaluator outputをCanonical meaningそのものへしない
- evaluator不在 / timeout / unsupported versionをpassへ変換しない
- Runtime Pluginの場合は既存Permission / Sandbox / Artifact / Lockを迂回しない
- Acceptance / Conformanceに使うEvaluatorの選択をEnvironment固有のRuntime provider選択へ委ねない
- Validation Evidenceへevaluator contract version、implementation identity / artifact digest、Conformance Evidenceを紐付ける
- 複数Evaluatorを交換可能と扱う場合、同じcontractに対するConformance Evidenceを要求する
- EnvironmentごとのEvaluator差によってAccepted semantic validityを再解釈しない

### E1 candidate — Validation-scope binding / 2026-10-05

PR候補レーンでは、Semantic Evaluatorの選択をEnvironment RuntimeのCapability negotiationから分離するReferenceを追加した。Evaluatorを必要とするConstraintは`evaluatorInterface`を明示し、そのInterfaceをExtensionの`requiresInterfaces`へ宣言しなければdescriptorを受理しない。catalogへConstraintを公開する時点でも、Semantic Interface / provider / exact contract fingerprint / conformance profileが解決済みでなければ`constraint-evaluator-unresolved`としてfail closedにする。

さらに、resolved Semantic Setとは別に`SemanticValidationPlan`を導入した。Planはexact Semantic Set fingerprintへboundし、Evaluator-requiredな全Constraintについてimplementation identityとartifact digestを固定する。一部のConstraintだけをbindして残りをEnvironment任せにするPlanは成立させない。Plan自身にもfingerprintを持たせ、Semantic Set driftやbinding改変を再入場時に検出する。

Evaluation時はPlanに記録されたimplementation / artifactと、Constraintへ解決済みのInterface / provider Extension / contract fingerprint / conformance profileの一致を要求する。現在AcceptedなConformance Evidenceも別入力で必要とし、使用したEvidence fingerprintをEvaluation Evidenceへ残す。Environment側に同じInterfaceを実装する別provider / implementationが存在しても、Planと一致しない限りValidationへ差し替えない。

Evaluation EvidenceはSemantic Set fingerprint、Constraint / subject / validator、Interface contract、provider Extension、implementation identity / artifact digest、Conformance Evidence fingerprint、input fingerprint、`pass / fail / unknown`、diagnostic、Evidence fingerprintを保持する。`unknown`は`pass`へ畳まず、Evaluator error / cancelも成功扱いにしない。

SUI hierarchyの実consumer pressureでは、`tools/reference-sui-adoption`に既存だったM1〜M4用shadow validatorを共有Reference packageへ切り出し、Stage 0とSemantic Extension側で同じ評価ロジックを利用する。これにより、Semantic ExtensionのためにSUI contract semanticsを二重実装しない。M1 root / M2 3-levelをpass、M3 missing parent / M4 cycleをfailとする既存fixtureをValidation Plan経由で評価し、CoreへSUI固有ValidationKindを追加しないcandidateを作った。

CLIには`-evaluator-binding`、`-validation-plan-out`、`-verify-validation-plan`を追加した。これはAI / Toolが会話履歴やEnvironment設定へ依存せず、Accepted Semantic Setに対するValidation implementationの選択を再入場可能なartifactとして扱うためのReference pressureでもある。

ただし、現時点のEvaluator implementationはReference内のdirect Go functionであり、Runtime PluginとしてのPermission / Sandbox / artifact signature verificationまでは通していない。Conformance Evidenceの発行Authorityも未実証である。したがってE1のProduct Evidence成立やAcceptance checkbox更新はまだ行わない。
## E2 — SUI fixtureでpressureする

既存`reference/sui-adoption/` fixtureを再利用します。

最低限:

- M1 root — pass
- M2 3-level hierarchy — pass
- M3 missing parent — fail
- M4 cycle — fail

新しいextensionを評価するためにSUI source contractを書き換えません。

TEI側のmapping / extension declaration / evaluatorで結果を説明し、SUIのnative authorityを維持します。

## E3 — second consumerで標準化をpressureする

SUIだけで成立しても、Core標準語彙へ昇格しません。

第二Referenceでは、既存のDatabase Migration modelを使います。`migration.Definition.Parents`はMigration間の依存関係を表し、cycleを許さないという意味上の必要性があります。SUI Island hierarchyとは別domainであり、単に同じJSON shapeを作るためのfixtureではありません。

2026-10-05のcandidateでは、SUIとMigrationからacyclic判定に必要な`nodes / directed edges`だけをdomain-neutralなsnapshotへ投影する小さなReference packageを追加しました。両domainで、同じSemantic Interface contract、provider Extension、evaluator implementation、artifact digestを再利用します。consumer Extension identityだけをdomainごとに分けます。

このpressureから、現時点では次の境界が妥当です。

- SUI固有の`IslandHierarchyContractV1`を共通Interfaceのinput shapeにしない。
- Migration固有のIDや`Parents`表現も共通Interfaceへ持ち込まない。
- 共通化するのはdirected graphのacyclicという意味だけに留める。
- parent target existenceやparent / child consistency等、SUIだけで必要な意味はSUI側へ残す。
- logical consumerが二つあっても、同じevaluator implementation / artifact identityを参照できる構造を保つ。実packagingでのbyte重複排除は別Evidenceとして扱う。

Reference testでは、SUIのM2をpass、M4をfailとして評価します。Migration側ではcurrent Reference Workspaceの`sales_core/20260826_001`をpassとし、`migration.Definition`で作ったcycleをfailとします。

また、SUIから共通snapshotへ変換した結果に`islands`、`parentIslandId`、`childIslandIds`、`schemaVersion`が残らないことをnegative controlで確認します。共通InterfaceにSUI固有語彙が漏れた場合は、再利用成立とは扱いません。

このcandidateが成立しても、`std.*`へ昇格しません。少なくともexact-head test、repository health、Semantic Lock / Validation Evidenceとの接続、AI / Tool discoveryでの再利用を確認した後に判断します。

### E3で見えたInterface粒度

E3では、既存SUI consumerが使う`reference.hierarchy.integrity.evaluator`全体をMigrationへ再利用できませんでした。SUIのInterfaceにはparent target existenceやparent / child consistency等、Migrationには不要な意味が含まれるためです。

一方、cycle禁止だけは独立した`reference.graph.acyclic.evaluate`として切り出すと、二つのdomainで同じ契約を再利用できます。

この結果から、Semantic Interfaceは「あるdomainのvalidator一式」を大きな一契約にまとめるより、独立して再利用できる意味単位へ分け、必要なdomain側で合成できる構造を優先します。ただし、細分化そのものを目的にはしません。二つ以上の実consumerで独立再利用できる境界をEvidenceとして判断します。

既存のSUI hierarchy Interfaceは直ちに置き換えません。E3 candidateは、より小さいInterfaceへ分解できる可能性を示すpressureであり、現在のSUI Reference contractをMigrationの都合だけで書き換える根拠にはしません。

## Namespace / conflict rule

namespaceは意味Authorityの境界です。

ownershipは「最初にloadされた」「最初にCatalogへ現れた」ことから取得しません。`std.*`はTEI standard authorityへ予約し、第三者Extensionが先に宣言してもownershipを得ないものとします。subnamespace委譲が必要な場合はownerからの明示的なdelegateとして扱います。

最初のReferenceでは、少なくとも次の状態をfail closedとして扱います。

- same extension identity + incompatible semantic version
- same symbolを互換性なしで複数extensionが定義
- required dependency version不一致
- dependency cycle
- unresolved implementation/evaluator requirement

競合をPlugin load orderで解決しません。

明示的なcompatibility / precedence contractが無い場合はUnresolvedとします。

## Versioning

Semantic versionはRuntime binary versionと独立させます。別の`compatibility level / family / epoch`は、version rangeだけでは表現できない実Referenceが得られるまで最終contractへ固定しません。

理由:

- evaluator implementationだけ更新して意味契約が変わらない場合がある
- 同じSemantic Extensionを複数Runtime実装が提供できる
- 意味互換性と実装互換性を同じversionへ畳むと、不要なCanonical invalidationを起こす

Source / Decision / qualification Evidenceは、必要に応じてSemantic Extension identity、semantic version、declaration fingerprintへ紐付けます。

## Shared Acceptanceとの関係

Semantic Extensionを追加・更新すること自体も意味変更です。

AI、CLI、Studio、Plugin installerのどこから来ても、採用済みextension declarationを黙って上書きしません。

```text
extension candidate
  ↓
Preview
  semantic symbols / applicability / conflicts / affected subjects
  ↓
Accept / Reject
  ↓ Accept only
authoritative extension source
  ↓
reload / Source Index
  ↓
Conformance
```

Runtime Pluginのinstall / activationは別Authorityです。

```text
accept semantic extension
  !=
allow process / network / database execution
```

## Knowledge-first / AI railとの関係

RQ-033のProduct valueを最初から検証します。

AI / Toolが新しい実装案を作る前に、対象subjectのKnowledge Contextから次を取得できることを目指します。

- applicable Semantic Extensions
- required parameters
- required dependencies
- deterministic Validation
- evaluator requirement
- unresolved conflict
- related Capability

AIがextensionを発見した後も、自動採用はしません。

`applicable`は「使える候補」であり、AcceptanceでもRuntime Authorityでもありません。ISSUE-000047と同じ境界を使います。

## Product Evidence

このReferenceでは、「新しい仕組みを実装できたか」だけを価値として測りません。

### Core growth

- 新しい意味論を追加するために変更したCore Go enum数
- parser / IR / validatorのCore固有分岐数
- 新しいApplicationごとに追加したCore code量

最初の目標は、SUI hierarchy固有の概念をCore enumへ追加しないことです。

### Reuse / change amplification

- extension一つの変更で影響するSource / Tool / Target数
- 同じconstraintの重複記述数
- 第二Referenceで再利用するための変更量
- extensionなしcustom implementationとのhand-written差分
- Bridge Extensionの個数、fan-in / fan-out、重複するcross-facet constraint数

### AI rail

- AIが既存extensionを発見できた割合
- custom implementationを作る前に既存extensionへ到達できたか
- required dependency / unresolvedを見落とした件数
- extensionを無視して近似実装した件数

### Target cost / portability

- cold / warm start
- compressed initial transfer
- install / cache storage footprint
- steady-state memory
- resolver / evaluator overhead
- Extension追加時の増分artifact size / startup / memory
- unused executable implementationの除外
- shared runtime / support artifactの重複排除
- Web Targetでnative process依存が残っていないこと
- logical Extension数とphysical artifact数の比率

一律の閾値はこのIssueで固定しません。RQ-024に従い、Targetごとのactual Evidenceとして比較します。

### Safety

- incompatible extensionのsilent override = 0
- 有効解が存在するgraphのfalse conflict = 0
- authority不明のSemantic Interface / reserved namespace受入 = 0
- semantic declarationからRuntime Authorityへの暗黙昇格 = 0
- evaluator不在をpass扱い = 0
- EnvironmentごとのEvaluator差によるAccepted validity変更 = 0
- stale Semantic Lockの無条件再利用 = 0
- unrelated Canonical meaningの変化 = 0

Safety違反を生産性指標と相殺しません。

## 最初のcode work unit候補

E0の最初のcode work unitは、一つのReference declarationとread-onlyなloader / indexの検証に限定します。

まだ最終syntaxは決めません。

候補構成:

```text
reference/workspace/extensions/<reference-extension>/<version>/...
internal/semanticextension/   # identity / declaration / validationのみ
internal/sourceindex/         # exact source references
internal/knowledgecontext/    # read-only discovery
```

ただし、このdirectory名やpackage名は実装前に既存syntax / Source Indexの拡張方法を再監査して決めます。

最初のincrementでは次をしません。

- Runtime process起動
- new Plugin CapabilityKind追加
- SUI dataのCanonical化
- std namespaceへの昇格
- AIによる自動選択

## Acceptance — E0

- [ ] CoreにSUI固有ValidationKindを追加せずSemantic Extensionを宣言できる
- [ ] extension logical identity / semantic version / Sourceへ戻れる
- [ ] applicabilityをexact existing subjectへ結び付けられる
- [ ] Source Indexからextension declarationを発見できる
- [ ] Knowledge Contextから適用候補をread-only取得できる
- [ ] duplicate / incompatible definitionをsilent overrideしない
- [ ] package identity dependencyとSemantic Interface dependencyを区別できる
- [ ] Semantic Interfaceがversioned contractとしてidentity / fingerprint / Sourceへ戻れる
- [ ] providerの自己申告だけでSemantic Interface適合済みとしない
- [ ] 複数Interface providerを明示policyなしに暗黙選択しない
- [ ] diamond dependencyの解決がload順 / catalog列挙順に依存しない
- [ ] lower compatible versionなら解けるgraphをgreedyなhighest-compatible選択だけでconflict扱いしない
- [ ] dependency version / compatibility conflictをblocking diagnosticまたはUnresolvedにする
- [ ] Extension package dependency cycleをAccepted graphへ入れない
- [ ] Bridge Extensionなしで独立Extension同士を暗黙結合しない
- [ ] optional Extensionの存在だけで既存meaningを変えない
- [ ] optional diagnostic / ProjectionがAcceptanceへ影響する場合はSemantic Setへ明示する
- [ ] 第三者Extensionが`std.*` namespaceをfirst claimantとして取得できない
- [ ] resolved graphをexactなSemantic Lockまたは同等表現へ固定できる
- [ ] Semantic Lockがroot requirements / wiring / resolution policy / canonicalization versionを再現できる
- [ ] staleなroot / policy / canonicalizationに対して既存lockを無条件に再利用しない
- [ ] Unresolvedを含むgraphからAccepted Semantic Lockを作らない
- [ ] 同じ入力から同じSemantic Set fingerprintを再現できる
- [ ] resolved Extensionごとにrootからのdependency pathを説明できる
- [ ] conflict時に両立しないrequirementの由来へ戻れる
- [ ] extension不在をCore標準意味へ推測変換しない
- [ ] Semantic ExtensionとRuntime Plugin identityを分ける
- [ ] Runtime Pluginだけの変更をSemantic Set変更へ無条件に変換しない
- [ ] extension declarationだけではRuntime Authorityを得ない
- [ ] accepted extension sourceをToolなしでreloadできる
- [ ] unrelated Canonical subjectsを変えない
- [ ] declaration-only ExtensionがRuntime artifactなしで成立する
- [ ] logical Extension / Plugin identityをphysical artifact / process / Wasm moduleへ一対一固定しない
- [ ] 複数Extensionを一つのTarget artifactへまとめても元identity / provenanceへ戻れる
- [ ] unused executable implementationをTarget artifactから除外できる
- [ ] shared runtime / support artifactを重複格納しない余地がある
- [ ] Target packaging差だけではSemantic Set fingerprintを変えない
- [ ] Web Targetでnative processを必須にしない
- [ ] Web非対応Runtimeを暗黙fallbackで意味変更しない
- [ ] RQ-024のTarget cost指標を増分として測定できる

## 次へ進む条件

E0-A〜E0-Cが同じISSUE-000077 workstreamとしてexact-head full `tools/repository-health-check.sh`を通ってmainへ収束した後、E1 evaluator、E2 SUI fixture、E3 second consumerの順に進めます。

E2が成立しても第二consumer前に`std.*`へ昇格しません。

第二consumerで大きなdomain-specific変更が必要なら、共通extension化の仮説を棄却または分割します。

## 実装Gate

code変更はTEIの通常Gateに従います。

- latest main / open PR / active laneを再確認
- code PRはDraft
- exact-head full `tools/repository-health-check.sh` PASS
- clean pre/post state
- Gate EvidenceをPRへ記録
- PASSなしでmainへ統合しない

## 非目標

- Plugin system全体の作り直し
- Runtime PluginをSemantic authorityにすること
- 一つの万能Extension DSL
- arbitrary codeをmeaningとして登録すること
- SUI domain objectをTEI Core kindへ追加すること
- SUIのshadow validationをそのまま標準語彙へコピーすること
- extensionをAIが自動採用すること
- load orderでconflictを解決すること
- Semantic ExtensionとPermission / Runtime Authorityの統合
- 一つのReference成功からEmacs相当の汎用性を実証済みと主張すること

## 関連

- PR #919
- RQ-031 / RQ-032 / RQ-033
- `product/value/TEI_PRODUCT_VALUE.md`
- `product/principles/TEI_PRODUCT_PRINCIPLES.md`
- `docs/plugins.md`
- `plan/issue/000040-semantic-component-composition.md`
- `plan/issue/000047-knowledge-first-semantic-exploration.md`
- `plan/issue/000052-sui-reference-adoption-and-plugin-surface-evaluation.md`
- `plan/SEMANTIC_EXTENSION_DEPENDENCY_PRE_ADR_2026-10.md`
- `plan/SEMANTIC_EXTENSION_ADVERSARIAL_REVIEW_2026-10.md`
