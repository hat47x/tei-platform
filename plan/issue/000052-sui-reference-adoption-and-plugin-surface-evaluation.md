# ISSUE-000052: SUI SensemakingをTEI Reference Adoptionとして移管し、Plugin surfaceと開発負荷低減を検証する

- 状態: Proposed — Gate / supported scopeに応じてpre-designとReferenceを段階実施
- 提案日: 2026-09-18
- 性格: Product validation / Reference adoption / Plugin discovery
- 主担当要求: RQ-004 / RQ-008 / RQ-010 / RQ-011 / RQ-012 / RQ-013 / RQ-017 / RQ-021 / RQ-023
- 関連: SUI `ARCH-TEI-MIGRATION-01` / ISSUE-000046 / ISSUE-000027 / ISSUE-000040

## このIssueを最初に読むときの要点

このIssueは、SUI SensemakingをTEIへ移すこと自体を成功とするものではありません。

確かめたいのは、

> **実際に複雑なUI・domain logic・安全境界・AI連携を持つ既存OSSをTEIへ移管すると、Application開発と保守の負荷が明確に下がるか**

です。

同時に、SUI固有に見える要求のうち、どれがTEIの再利用可能なPlugin / Adapter / Capabilityへ一般化できるかを探索します。

SUIはTEIのdemo applicationではありません。SUI自身のProduct Value、Domain、SafeMode、review / approval boundaryを保ったまま、TEIのProduct Valueが現実のApplicationで成立するかをPressureするReference Adoptionです。

## 減らすProduct上の不確実性

1. TEIのCanonical / Projection / Target / Capability / Acceptance境界が、既存の非自明なApplicationを実装する負荷を実際に減らすか。
2. TEIは一般的な業務システムだけでなく、spatial canvasやAI-assisted sensemakingを含むApplicationにも耐えられるか。
3. TEIに不足する機能を、Core growthではなくPlugin / Adapter / Capabilityとして吸収できるか。
4. 現行Applicationから段階移管するとき、既存native assetを保持しつつTEIへ責務を移せるか。
5. 「TEIを使った結果、別の二重管理が増えた」という失敗を検出できるか。
6. SUIで得たPluginが第二Applicationでも再利用できるほど一般的か。

## 基本方針

### TEI-firstと全面TEI-native化を区別する

TEI-firstは、すべてをTEI DSLやTEI Coreへ書き直すことではありません。

```text
Application meaning / domain rule / validation / projection / target
  -> TEIを第一候補

specialized interaction / algorithm / external runtime
  -> Plugin / Adapter / Capability / native asset

low-level kernel / performance-critical engine
  -> native implementationを許容
```

SUI移管では、既存TypeScript / worker / import-export / specialized canvas logic等を、必要に応じてnative assetとして段階的に接続できます。

### GapはCoreへ直行させない

SUI側で不足が見つかった場合は、次の順で分類します。

1. **Existing TEI** — 既存のCanonical / Pattern / Projection / Target / Capabilityで表現できる
2. **Plugin / Adapter / Capability** — 複数Applicationへ再利用できるextensionとして切り出せる
3. **Native asset / custom implementation** — SUI固有または専門性が高く、TEIからBindingすべき
4. **Core / Product Requirement candidate** — 複数Applicationに共通する意味上の不足がReference Evidenceで確認できる

4へ進む前に、1〜3で解けない理由を明示します。

## Stage 0準備状況

2026-09-18時点で、実装前の準備として次を固定した。

- TEI側設計基線: `plan/SUI_REFERENCE_ADOPTION_DESIGN_2026-09.md`
- SUI側既存handoff: `IslandHierarchyContractV1`
- TEI側固定fixture: `reference/sui-adoption/`
  - M1 root — pass
  - M2 3-level hierarchy — pass
  - M3 missing parent — fail
  - M4 cycle — fail

Stage 0では新しいSUI↔TEI共通schemaを発明せず、SUIが既に持つdeterministic projectionをread-only inputとして再利用する。

次の実装probeへ進む場合も、最初はfixture inputを変更せず、TEI Product codeやCapabilityKindを変更しない隔離されたReference tool / testとして開始する。

current TEI標準Validationは`positive` / `relation-unique`の最小集合であり、SUI hierarchyのcycleやparent/child整合をそのまま表現できるとはまだ言えない。

加えて、current Semantic Relationは`one | many` cardinalityを持つが、SUIのnullable `parentIslandId`が必要とする0..1 presence semanticsを独立表現するcontractは確認できていない。Stage 0 reportではこの点を`unresolved`かつ`lossless = false`として残す。

Stage 0 report schema v2では、この不足を単独のbooleanだけで扱わず`constraintCoverage`として機械可読化する。parentのsingle-valued shapeだけを`shape-only / tei-definition`、nullable presenceを`unresolved / source-contract`、duplicate / existence / parent-child consistency / cycleを`shadow-only / reference-local-shadow`として分離する。これにより、SUI正本のconstraintをReference toolが再現しただけのものをTEI native機能へ誤って格上げしない。

これらを理由にValidationKindやRelation optionality fieldを先に追加せず、

- Reference-local validationで十分か
- reusable extensionが必要か
- Application-native ruleとして残すべきか
- 複数Referenceで共通failureが出てProduct gapになるか

を順に判断する。

## Stage 0 read-only probe

Stage 0の最小実装を`tools/reference-sui-adoption`として隔離する。

実装範囲:

- `reference/sui-adoption/workspace`に`sui.island`とself-parent relationだけを定義
- SUI既存`IslandHierarchyContractV1` fixtureをread-only入力として利用
- M1 / M2をvalid、M3 / M4をinvalidとしてconsumer-side shadow validation
- TEI definitionとSUI instance dataをレポート上で分離
- write / canonicalizationは行わない

このprobeはProduct Gate Evidenceではなく、Reference Adoptionのmapping hypothesisを実行可能にするresearch Evidenceである。

次の判断は、probe結果を見てから行う。

1. hierarchy定義だけで十分にTEIへ意味を寄せられるか
2. cycle / parent-child consistencyをApplication-native validationとして残すべきか
3. reusable validation extensionが必要か
4. placardを次のhandoffへ追加する価値があるか

この時点ではValidationKindやPlugin CapabilityKindを増やさない。

## Stage 0 → Semantic Extension evaluator pressure — 2026-10-05

Stage 0でReference-local shadowとして保持していたhierarchy validatorを、Semantic Extensionの実consumer pressureへ再利用するcandidateを追加した。SUIの`IslandHierarchyContractV1`やfixtureは変更せず、既存M1〜M4 semanticsを実装していた評価ロジックを共有Reference packageへ切り出し、`tools/reference-sui-adoption`と`tools/reference-semantic-extension-catalog`の両方から同じ実装を使う。

この変更でSUI側のAuthorityはTEIへ移らない。SUI native contractが引き続き正本であり、共有Evaluatorはそのpinned contractをshadowするReference implementationである。TEI Canonical dataへinstanceを昇格せず、writeも行わない。

Semantic Extension側では、`sui.island` subjectへhierarchy integrity Constraintを追加し、そのEvaluator requirementをversioned Semantic Interfaceとして明示する。provider自己申告だけではConstraintを有効化せず、exact contractへboundしたConformance Evidenceが必要である。さらにValidation Planでimplementation identity / artifact digestを固定し、Environment Runtimeのprovider選択でAccepted validityを変えない。

M1 root / M2 three-levelはpass、M3 missing parent / M4 cycleはfailとして、同じshared evaluatorをValidation Plan経由で評価するcandidate testを追加した。これは「SUI固有ValidationKindをCoreへ追加しなくても、実Application由来の意味不足をSemantic Extensionで扱えるか」をpressureする。

ただしexact-headのfocused test / repository healthはこの環境では未取得であり、Stage 0の`constraintCoverage`をnative coverageへ昇格しない。Semantic ExtensionのProduct Evidence、SUI移管判断、標準語彙化のいずれもこのcandidateだけでは更新しない。
## 最初のReference slice候補

最初からSUI全体を移しません。少なくとも次の性質を一つの小さいsliceでPressureできるものを選びます。

- Card / Island / Relation等の意味データ
- 保存・再読込
- Canvasまたは空間Projection
- 一つのValidation / SafeMode rule
- 一つのreview / proposal acceptance
- 一つのAI / Cognitive Capability
- export / provenanceの最小経路

初回sliceの具体的な対象は、TEIのsupported scopeとSUI側の比較基線を見て決めます。

## 固定済みのReference設計

TEI側の移管設計基線は `plan/SUI_REFERENCE_ADOPTION_DESIGN_2026-09.md` を参照する。

SUI側では実履歴から次の3 challengeを固定済みである。

| Challenge | Historical SUI PR | Changed files | Changed lines |
|---|---:|---:|---:|
| hierarchy / placard | #235 | 11 | 338 |
| polygon vertex editing | #286 | 5 | 219 |
| explicit merge apply transaction | #2849 | 3 | 465 |

補助Evidenceとして、hierarchy concernがexport / import / view metadataへ横断したSUI PR #241（10 files / 260 changed lines）も用いる。

初期移管戦略は **encapsulation before translation** とする。現行SUI `DocumentV1`を当面のauthoritative native assetとして保持し、TEI側はread-only mapping / projectionから開始する。二つのauthoritative sourceを作らない。

また、current TEI Pluginの`CapabilityKind`にはspatial canvas / presentation専用種別は存在しない。この一件だけを根拠にkindを追加せず、Target Adapter、native component、既存Capability、将来のreusable extensionのどれが自然かをReferenceから判断する。

## TEI適性の第二軸 — structure-to-application leverage

SUI Reference Adoptionでは、Application Planeの共通化だけでなく、SUIの図表的・構造的な設計資産からApplicationをどこまで導けるかを検証する。

比較では、次を観測する。

- 一つのstructured definitionから何個のValidation / Projection / Target / documentation / AI contextを導けるか
- 同じ意味の重複記述が何箇所減るか
- view変更でdomain definitionを書き換えずに済むか
- domain意味変更が複数UI / export / validationへ自動的または機械的に伝播できるか
- native codeへ残る部分が、特殊Interaction / algorithm等へ局所化するか

これは **structure-to-application leverage** のEvidenceとして記録する。

TEI利用率が高くても、構造化した意味から派生物を導けず二重管理が増えるならReference Adoptionは成功としない。

## 開発負荷の比較

SUI側の同一change challengeと比較し、TEIで次がどう変わるかを記録します。

- hand-written code / config / schema差分
- 触るファイル・層・正本の数
- test fixture / regression testの更新量
- 同じ意味を重複記述する箇所
- generated / derived artifactの再生成可能性
- Validation / Conformanceが自動で捕捉できる不整合
- manual synchronization step
- Plugin / native assetとして残るapplication-specific code
- migration friction / rollbackの容易さ

中心指標は**change amplification**です。

```text
one semantic change
  -> how many layers / files / tests / docs must move together?
```

単純な生成コード量削減やTEI記述率を成功指標にはしません。

## Plugin discovery

初期のPlugin候補として、次を仮説として保持します。

- spatial / graph canvas projection
- drag / resize / grouping / polygonal enclosure等の複雑Interaction
- offline bundle / artifact import-export
- review / diff visualization
- local cognitive provider integration

ただし、SUIで必要だったという理由だけでTEI標準Pluginにはしません。

Plugin候補は最低でも、

1. SUIで実用価値がある
2. TEI Coreの意味を増やさず接続できる
3. 第二Applicationで再利用可能性をPressureできる

ことを確認してから昇格します。

## Gateとの関係

TEIの現在のProduct Gateやsupported scopeを、SUI移管のためだけに飛び越えません。

- pre-design
- mapping
- change challengeの固定
- Plugin boundaryの仮説
- Reference fixture準備

は先行できます。

一方、未成立のTarget / Acceptance / Runtime機能をproduction-readyとして扱う実装は、対応GateのEvidenceを待ちます。

## 非目標

- SUI Product ValueをTEI Product Valueへ吸収すること
- SUIのDomain objectをTEI Core kindとしてそのまま追加すること
- SUIを一括rewriteすること
- SUIが使うからという理由でTEI Coreを拡張すること
- TEIへの移管率100%を目標にすること
- 専門runtimeや高性能canvas engineまでTEI-nativeで再実装すること
- 一つのReference成功だけで「すべてのOSSをTEI化できる」と主張すること

## 完了Evidence

このIssueは、少なくとも次が揃ったとき判断可能になります。

1. SUI側の現行baselineと3種類以上のchange challenge
2. TEI上の最小Reference slice
3. 同一change challengeの比較結果
4. gapのExisting / Plugin / Native / Core candidate分類
5. 少なくとも1つのPlugin候補についての実利用Evidence
6. SUI移管を broad / incremental / limited / deferred のどれにするかの判断
7. TEI側で、今回のReferenceからProduct Requirementを増やす必要があるか否かの明示判断

## 完了後に可能になる判断

- SUIをTEIへ本格移管する価値があるか
- TEIを他の自作OSSのApplication Implementation Layerとして標準利用する妥当性があるか
- どのPluginを先に育てるべきか
- TEI Coreへ不足があるのか、extension surfaceだけで十分なのか
- SEI / EKI等の管理面・UI面へ横展開できるか


### 実装で顕在化した追加Pressure — GestureとDomain Actionの非同一性

**観測（2026-10-08、SUI実装のソース読取＋TEI Reference E1/E2）**

- SUI `CardView`はpointermoveごとにスクリーン座標の差分を通知し、`CanvasShell`がzoomでworld deltaへ換算する。`App.handleCardMove`はdrag中の**in-memory Document履歴**を随時更新し、設定に応じてgrid snapを適用する。
- `App`のpointerup **とpointercancelの両方**が現在は一つのundo stepの確定経路へ入り、drop後にCardのIsland所属を更新することがある。単純なCard座標更新だけでは既存のdomain actionを再現できない。
- 永続化は別の`handleSave`で、`putDocument`へ（存在する場合）ETagを渡す。SUIの`PUT`は`If-Match`欠落時にLWWを許すため、Reference E2では**強いサーバETag必須／無条件PUT禁止**に限定してHTTP境界を検証した。
- Reference E1は独立Gestureと`position.set`、E2はloopback HTTP上のGET/If-Match PUTであり、実際のSUI React Canvas接続、Domain full validator、TEI Driver/Permission連携、実SUI Backendの条件付き更新の実証は**未了**。

**設計への帰結**

「Gesture preview →座標commit」だけを一般化しても、特殊Applicationの実装が可能になったとは言えない。必要なのは、Native Interactionのイベント粒度と、Applicationが所有する**複合Action（Card座標・島所属・undo単位）**、UI履歴の確定と永続化の確定を結ぶ可変な境界である。TEIはその接続・Authority・Bindingを提供し、SUI固有のIsland geometryやdomain policyをCoreへ引き上げない。

**次の受入試験**: native pointer lifecycle、zoom、grid snap、cancel policy、Islandへのdrop、undo、Save/reloadを一つの実行シナリオとして観察し、現行SUIと比較する。pointercancelの意味を勝手に変更しない。E2のmock HTTP成功をSUI live-Backend準拠と読み替えない。詳細Reference: `tools/reference-sui-interaction/`、PR #994・#995、Issue #993。

## 非業務・非汎用Applicationによる構成力の圧力試験（2026-10-08）

- Native Canvas / Interaction Hostの実装窓口: GitHub Issue #993（本節を設計・Acceptanceの正本とする）

### 判定したい問い

SUIのように専用の空間UI、探索的な状態、利用者による保留・判断、任意の認知処理を持つApplicationを、**TEI CoreにSUI固有概念を追加せず**組み立てられるかを検証する。SUIを移植した割合ではなく、同じ意味変更を行ったときの保守負荷、既存native資産の再利用、Validationの一貫性を成果とする。

このPressureでは次を混同しない。

- TEIのCanonical *definition* と、SUIで利用者が作るCard / Island等の *instance data*
- TEI sourceのShared Acceptanceと、SUI上で利用者が行うProposal / Hold / Mergeの判断
- 観察・仮説・保留というSUI domain stateと、一般的な開発作業の検証状態
- Canvas上の一時的なpointer / selection / preview状態と、確定して保存するDocument state
- SUIの原Source / DocumentV1のauthorityと、TEIのread-only mapping・derived projection
- Targetで可能な操作と、Runtime実装の配置・権限・Resource条件

「Canvasが表示できた」「JSONを読み込めた」「Pluginに名前が付いた」だけでは、ApplicationをTEIで実装可能と判定しない。

### Gap inventoryと配置の第一候補

| SUIからのPressure | 現状の境界・未成立点 | 最初に配置する先 | 判定に必要なEvidence |
| --- | --- | --- | --- |
| Islandのnullable parent、異種endpoint、任意の関係種別 | Relationのone/manyだけでは0..1をlosslessに表現できない。SUIのEdge instanceをTEI Relation definitionへ変換できない | SUI domain schema + Semantic Extension / Validation Interface | 0..1・参照先存在・cycle・関係の方向を個別に検証。未知種別の往復保持 |
| アプリケーション固有の意味語彙 | 固定kindだけで新しい意味・規則を表すとCore変更が必要 | RQ-031〜033 / ISSUE-000077 | Extensionの宣言・依存解決・pin・通常Validationへの接続・AI discoveryを一つの経路で実証 |
| 独自DSL構文・Operator | Operatorだけの追加ではParserや意味変換を拡張できない | #988（DSL Language Provider） | 外部Language Providerの選択・構文解析・意味変換・衝突検出を順序非依存で実証。SUIのためにTEI固定Parserへ追加しない |
| Canvas / polygonなどの専用UI | 第一Web Targetは業務Pattern中心。任意の既存ReactコンポーネントをApplicationへ結ぶ実証がない | Target Adapter内native UI → 再利用性確認後にInteraction / Projection Extension | native componentからRead / Preview / Commitへの接続を実行。UI実装をCoreへ複製しない |
| Pointer操作・複合選択・一時preview | 操作中の一時状態と永続変更をどう接続するかが未実証 | UI Host + Application-owned Action | previewはread / write副作用なし、commit時だけValidation / current authority / revision整合を確認 |
| Document-levelな変更・共有・復元 | SUI DocumentV1をTEI定義へ全面移管すると二重正本化する | native DocumentV1を保持、scope単位で必要ならauthority移行 | Save / Reload / Export / Importで意味が保全。移行する場合のみscope / roundtrip / rollbackを明示 |
| AIの統合提案と人間の判断 | 開発時Shared AcceptanceとSUIのdomain decisionは別。実行時Capabilityの一体接続が未実証 | SUI domain transaction + TEI Capability / Permission / Provider境界 | AIは候補のみ。reject / holdは原Cardを改変せず、accept後のmutationとObservationを区別 |
| SafeMode・出力時の秘匿 | 表示、AI入力、export、importで異なる経路を通る | SUI policyを正本とした実行時Policy / native safety gate | 未review本文・秘密情報が既定で意図せず外部へ出ない。認可不足はfail closed |
| Browser / offline / Plugin packaging | builtin / process中心の実装をBrowserへそのまま移せない | #951のCapability Interface、#957のself-build、RQ-024 / #259 | 論理Extension・Runtime実装・Target artifactを区別し、起動時間・転送量・メモリ・追加依存を計測 |

「未成立」はその能力が原理的に実装不可能という意味ではない。すでにReference candidateがある項目も、productionのApplication end-to-endに接続できたEvidenceがない限り成立済みとしない。

### 最初の縦切り（段階的な構成実証）

SUI固有のCore追加や全面rewriteをしない。既存DocumentV1とCanvasを正本／native資産として保持し、独立したfixture上で次を順に確認する。

1. **A — Semantic / Validation**: 既存M1〜M4を維持し、nullable parent、target existence、cycle、parent-child consistencyを独立判定する。SUI正本validatorと共通Evaluatorの結果の差は、domain constraint、consumer compatibility policy、未実装の三つへ明確に分ける。#932で第二consumerと共有できた非循環検証は、SUI全域の規則の再利用成立へ拡大解釈しない。
2. **B — Native Interaction Host**: Canvasでカードを選択・移動し、視覚previewと永続commitを分離する。操作イベント全件をTEIへ永続化しない。初回はSUI既存コンポーネントのadapterで実施し、Canvas自体の汎用Plugin化は保留する。
3. **C — Domain Action / Decision**: グルーピング候補を提示し、利用者のaccept / reject / holdを扱う。SUIのdomain decisionをTEI authoring Decision ledgerへ書き込まない。acceptedな変更だけをSUI domain validatorと現在の実行権限で審査し、Documentへ反映する。
4. **D — Reload / Export / Cognition off**: 保存したDocumentを再読込して同じ構造・保留・来歴が残ることを検証する。SafeMode既定値でExportとAI入力を点検する。Cognitive Providerが無効でも主要な操作が成立することを確認する。
5. **E — Change-amplification / second consumer**: 同等のSUI既存変更Challenge A（hierarchy）、B（polygon）、C（merge apply）をSUI-native経路とTEI接続経路で比較する。少なくとも一つの共通部品をSUI以外のApplicationでも再利用し、SUI特有の偶然の一般化を排除する。

各段階で**構成できた／外部native実装で回避した／TEIの共通機構が不足した／判断材料が不足**を区別する。read-only Stage 0のpassはB〜Eのpassを意味しない。

### 検証時に失敗させるべきケース

- parent不在は許すが、存在しないparent参照、階層cycle、parent-child不整合はSUI正本と同じ規則で検出する。
- Canvas previewの開始・取消し・Pointer中断で、Document revisionや永続stateを変えない。
- 予期しないDocument更新後の古いpreviewを、現在revisionへ黙ってcommitしない。
- Componentが別ApplicationのActionや許可されていないResourceへ到達しない。
- AI候補を返すProviderの変更・不在・errorを、利用者によるacceptや確定済みの意味へ読み替えない。
- reject / holdや表示上の非表示だけを、原Card削除や意味の確定に読み替えない。
- SafeModeのUI表示だけを信頼せず、export / share / external AIへのデータ境界でも漏出を防ぐ。
- 未対応のEdge kind、横断的Affiliation、由来のあるinstance dataを、導入時に黙って破棄しない。

### 成功条件と既存作業の担当

このPressureの**成功条件**は次のとおり。いずれも現時点で完了とは主張しない。

1. SUI domain固有の新規Core kind / 固定CapabilityKindを追加せず、A〜Dの最小シナリオが通常実行経路で成立する。
2. Semantic Extensionの解決、実行時のPermission、Target Binding、Application Data保存、safe outputをそれぞれ別の所有境界で説明できる。
3. 専用UIや特殊アルゴリズムがnativeのまま残っても、TEIで再利用する構造・規則・契約の二重管理が増えない。
4. Challenge A〜Cの少なくとも一つで、変更対象・重複記述・手動同期・再検証の負担がSUI-native baselineより実質的に下がる。
5. 性能・配布サイズ・オフライン利用・ブラウザ可搬性の悪化を隠さず測定し、限定適用が適切なら限定適用と結論できる。

既存の担当境界を維持する。Meaning / ConstraintはISSUE-000077、DSLのProviderは#988、Generic Capabilityは#951、Target差は#259、AIと開発時Acceptanceは#267等を参照する。本Issueは**非業務Applicationとしてのend-to-end統合と比較**に責任を限定する。重複するCore実装・新Product Requirement・SUI専用標準Pluginを、本記録だけで確定しない。
