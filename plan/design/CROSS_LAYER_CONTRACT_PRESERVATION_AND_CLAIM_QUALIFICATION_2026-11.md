# TEI横断：契約保存・Claim単位の資格化と証拠の非昇格（設計候補）

- **状態**: 評価待ち（未採用）。要求・API・Core型・共通Runner・新Gateの決定ではない。新しいコード実行・実Provider/実DB・Studio GUI・Owner業務受理は **NOT RUN**。
- **静的照合基準**: tei-platform main `8c3d1a5582cf3e9a40b52b3a2ebaa041739b4fd2`（2026-11-12）。既存Issue・設計文書・PRの静的読取。改版後は再照合する。
- **目的**: 構造、業務意味、能力、生成、Build、実行、配送、Studioの間で、局所PASSを別のClaimへ無根拠に転用しない。反例と責務を横断的に接続する。
- **正本**: [現行アーキテクチャ](../../docs/architecture.md)、[現行実装状況](../../docs/status.md)、[ADR-0043](../adr/0043-target-native-artifacts-without-tei-host.md)、[ADR-0048](../adr/0048-shared-application-api-and-transport-adapters.md)、[重要Issueの計画](../workstreams/TEI_PRODUCT_CRITICAL_PATH_2026-10.md)。本書は正本の置換ではない。
- **既存領域設計**: [図表・Pipeline](TOPOLOGY_COMPOSITE_VIEWS_AND_PIPELINE_2026-10.md)、[Trait履行](TRAIT_ATOMIC_COMPOSITION_AND_FULFILLMENT_2026-11.md)、[ETL/Batch](ETL_BATCH_FUNCTION_COMPOSITION_DESIGN_2026-11.md)、[セルフビルド再入](SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)。これらの試験・実装Ownerを移さない。

## 1. 採用を提案する最小原則（新規基盤機構ではない）

1. **Meaning / Contract first**: 意味・要求操作・制約はCanonicalな意味の受理境界へ、実現方式・公開方式・Runtime・物理成果物は別のBindingへ置く。TEI固有Host、中央HTTP、ETL専用DSL、第二Graph/Resolver/Scenario SSOTは必要条件にしない。
2. **Evidence before promotion**: Sourceに記述がある、Parserが読んだ、型が合う、Providerが選ばれた、Runnerが成功した、Artifact byteが一致した、Consumerへ到達した、実Effectが確認された、Ownerが業務意味を受理した、は別々の主張である。一つから隣の成立を推測しない。
3. **Scope-relative qualification**: 一つの検証結果は対象Subject・Scope・問い・前提・Reader/Consumer・Source/Artifact版・実行Attempt・Policy時点に相対化する。Claimの対象を変えたら必要な根拠を再照合する。
4. **No invisible completeness**: 「図にないなら無い」は問われた範囲の全入力・Reader/実消費・関係/役割・未解決・開示条件が閉じた場合だけの限定結論。見えていない処理、未観測のWriter、未取得のExternal/Private情報は無いことにしない。
5. **Minimal meta-generation**: 共通部品の合成で解ける間は専用Generator-of-Generatorsや万能Pipeline Engineを追加しない。異なる実Target間で実証済みの重複・変更負担が残ったとき、再利用可能なTransformの最小生成を検討する。

四層（Authoring / Canonical Semantic / Binding・Realization / Implementation・Execution）は現行構造を維持する。横断の評価表は新しい第五層、認証主体、Claim Registry、永続的な製品Schemaではない。

## 2. Claim間の引継ぎ規則（必要条件・十分条件ではない）

一件のClaimを照合するための**既存Evidenceの一時的な比較列**を、少なくとも以下に分ける。名前を永続API・Core enumに固定しない。

| 比較列 | 判定したいこと | 典型的な欠落 |
| --- | --- | --- |
| Subject / Scope / Question | 何の、どの範囲の、どの問いへの結果か | 同名異Scope、Query述語未対応、Source無ID |
| Required closure | その問いへの必要入力・関係・Role・時点が列挙済みか | 暗黙入力、Graph非表示辺、外部Writer |
| Source / Meaning revision | 実読込byte・Source Index・必要な受理済み意味が一致するか | Source位置とSemantic IDの混同、推定の正式昇格 |
| Mapping / Trait / Provider | 必須の構造・値・時間・操作保証を保存または履行したか | 非単射Projection、未履行Required Trait |
| Realization / Artifact / Consumer | 選択Targetと実Build消費・Artifact一式・受領byteが結ばれたか | declared-only、Proxy child、部分公開、Reader未受領 |
| Effect / Authority / Acceptance | 実副作用の独立観測・現在のGrant・Owner受理が対象Claimに適合するか | HTTP成功＝DB確定、ACK＝業務完了、古いGrant |

**引継ぎできる場合**: 送り側と受け側で対象・版・Scopeを対応付け、必要なFacet/Contractが写像で保存され、当該Claimに必要な実Consumer・独立Expected・現在の許可を既存Ownerが検査した範囲だけ。条件の列挙自体はPASSの証拠ではない。

**引継ぎできない場合**: 欠落・損失・不明・非開示・未観測をゼロやPASSに丸めない。読み取り成功は編集逆写像の許可ではない。Provider conformanceは実Targetの権限ではない。実行時の古い資格は同digestに戻っても自動復活しない。逆にLayoutだけの変更は、意味に影響しないことを確かめられた範囲で不要なMeaning再受理を発生させない。

区別すべき二種類の証拠は、**不変な履歴事実**（特定の版・Attemptで何が起こったか）と、**現在の使用資格**（Grant、Disclosure、Provider選択、読取先、Owner判断等）である。履歴PASSを消す必要はないが、現時点の使用資格へそのまま再利用できない。

## 3. 他チャットの成果を結ぶ最小反証

下表は**既存Issueへ渡す比較観点**であり、独立Expected/Scenarioの第二正本や新しい製品試験IDではない。実試験結果を主張しない。

| 反例・変更 | 禁止する誤昇格 | 既存Owner |
| --- | --- | --- |
| Sourceは同じ表示名だがScopeが異なる、または元X-TEA要素に安定IDがない | Source位置・一時View IDからPin/逆編集の資格を合成 | [#1124](https://github.com/hat47x/tei-platform/issues/1124)、[#1120](https://github.com/hat47x/tei-platform/issues/1120) |
| Graph→Tree／集計表でRelationやOccurrenceを失う | read-onlyの投影成功から元グラフの可逆編集を許す | [#2497](https://github.com/hat47x/tei-platform/issues/2497)、[#4350](https://github.com/hat47x/tei-platform/issues/4350) |
| 保存Providerの構造的特殊化で原子更新・条件付き書込みが不足 | Pluginの互換名・解決成功をRequired Trait履行と扱う | [#951](https://github.com/hat47x/tei-platform/issues/951)、[#3278](https://github.com/hat47x/tei-platform/issues/3278) |
| 旧Java Enumに3値が存在するが、別Writer・時点・未観測値を未調査 | 実装上の列挙を業務の閉値域・MECE・全Targetの履行と自動採用する | [#2495](https://github.com/hat47x/tei-platform/issues/2495)、[#3278](https://github.com/hat47x/tei-platform/issues/3278) |
| Build childがProxy経由で別実行物を呼ぶ、または宣言外のbyteを読む | Artifact digest一致を全actual consumed closure・builder由来の保証と扱う | [#2831](https://github.com/hat47x/tei-platform/issues/2831) |
| ETLの途中Stage完了・File ACK・Dataset片方のみ新世代 | Business Accepted、完全なDataset公開、冪等Commitの証拠と扱う | [#1130](https://github.com/hat47x/tei-platform/issues/1130)、[#2542](https://github.com/hat47x/tei-platform/issues/2542)、[#2604](https://github.com/hat47x/tei-platform/issues/2604) |
| 更新凝集に従いSubsystem境界を再分割し、当初の名称が役割と不一致になる | 旧Subsystem名を不変な意味IDや物理DB/配備境界へ固定する | [#3250](https://github.com/hat47x/tei-platform/issues/3250)、[#267](https://github.com/hat47x/tei-platform/issues/267) |
| FunctionのCRUD欄が空、または代表Bindingが一つ | 全Writer閉包・越境直接CUDがゼロと断定する | [#3250](https://github.com/hat47x/tei-platform/issues/3250)、[#1127](https://github.com/hat47x/tei-platform/issues/1127) |
| Private Worldを変えPublic Sliceは同一 | 公開のCount・Score・Digest・診断・DOMからPrivateの存在が分かる | [#2516](https://github.com/hat47x/tei-platform/issues/2516) |
| Source/Plugin/GrantがA→B→Aへ戻る | 昔の受理・Current Grant・実Effectの資格が復活する | [#2555](https://github.com/hat47x/tei-platform/issues/2555)、[#1174](https://github.com/hat47x/tei-platform/issues/1174) |
| 旧Legacy観測LとOwner受理Desired Dが意図的に異なり、Target観測TはDと一致 | 旧実装とのbyte/挙動一致と意図的な業務改善を同じ互換性PASSにする | [#1124](https://github.com/hat47x/tei-platform/issues/1124)、[#3394](https://github.com/hat47x/tei-platform/issues/3394) |
| TEI→EKIへのAdmissionは成功、実Worker結果・独立Readerは欠落 | 委譲受理を実Function Outcomeや業務Effectと扱う | [#4384](https://github.com/hat47x/tei-platform/issues/4384)、[#4194](https://github.com/hat47x/tei-platform/issues/4194) |

Subsystem再命名は役割・更新責務が改善された可能性を示す**設計上の発見**であり、自動採用や必ず改名するという規則ではない。Logical IDの継続・分割、Ownerの受理、既存外部参照、物理Migration/配備は別々に審査する。閉値域・強属性も、実装言語の表現から業務意味へ一方向に昇格させない。

三要素分析法（データ・機能・業務）の原典に忠実な表現と、TEIの汎用Graph/Role・AI解析は混同しない。Legacy観測・受理済み意図・Target観測は三要素分析法の三視点とも別の軸とする。

## 4. 実装順への受渡し（新しいHard Gateを増やさない）

| 順 | 確認対象 | 既存の受入先と停止線 |
| --- | --- | --- |
| **P0-A** | 既存Read-only Operationの同一Scope・版・失敗意味をCLI/MCP等で比較 | #1158 / ADR-0048。入口が同じJSONを返すだけでは独立Business Oracleにならない |
| **P0-B（並走）** | セルフビルドのDeclared→Selected→Actually Consumed→Actual Outputと実childの観測 | #2831。L2bとG4全体資格、Seed trustを分離 |
| **P0-C（並走）** | 必須Trait→解決済みProvider→実履行と、Facet→表現型のLoss/逆写像 | #3278 / #4350 / #2497。Referenceの合成PASSは実Targetへ昇格しない |
| **P1** | 一つのread-only Batch FunctionのInput ClosureとArtifact/Consumerを接続 | #1130 / #2542 / #2604。Source/Plan/Run/Published/Consumedを別Claim |
| **P1** | Source根拠つきDirect CUDとUnknownの分割案、Effective Effectは独立欄 | #3250 / #4412 / #4413。候補順位やGraph下界を全業務の証明に使わない |
| **P2（P0のHard依存なし）** | StudioのLocate-first read-only Consumerから限定変更Proposalへ | #1124 / #267。非可逆表現・無ID Sourceの編集は拒否し、Owner受理を別にする |

各実装Ownerは既存のScenario SSOT（#1180）、独立Expected（#2514）、Scope/Disclosure（#1120/#2516）を利用する。本書が新しいSchema・Engine・Contract Resolver・Policy・Recorderを所有しない。

## 5. 検証可能な研究仮説と撤回条件

**仮説**: 同じSubject/Scope/Claimを、Source→構造→操作→実現→実Consumerの各境界で明示し、局所証拠を過大昇格させない設計により、人間と生成AIの誤った肯定・誤った不存在判断と変更負担が減少する。

**測定候補**: （a）誤ったPASS・不存在断定の件数、（b）必要Source/Ownerに到達する操作数、（c）変更後に不要となる再資格化数と見落とした必須再資格化数、（d）AIが修正するまでの往復・Token、（e）新抽象の維持コスト。既存単一図/散文と、同一Sourceからの表・Graph・診断を比較する。実験は **NOT RUN**。

**撤回・縮小**: 同一のClaim表を各Ownerが二重管理する、Consumer間で意味がずれる、Scope/Policy漏えいが増える、実験で理解・修正負担が減らない、または既存Contractで十分に表現できるなら、新しい抽象や永続化は導入しない。再利用できる比較表として留める。

**本書の証拠上限**: GitHubの既存Issue/PRと設計・Sourceの静的照合に基づく横断設計だけ。新規Go/Node/Windows/Wails実行、実Build/ETL/DB/EKI Consumer、Current Grant、外部Policy、Ownerの仕様採用は確認していない。

## 6. 問いに相対化した再資格化：履歴の保存と現在の使用適格性（2026-11-12追補）

**今回の判断候補**：変更に対する影響は「Fileが変わったか」ではなく、**一つの問いとそのClaimが必要とする依存・観測・使用条件が変わったか**で判定する。新しいClaim Registry／Proof Engine／共通Schemaを作る提案ではない。

一時的なレビュー票で、対象Claimの輪郭を `Q=(subject, scope, predicate, quantifier, observationBoundary, requestedUse)` と記述できる。これらは既存のQuery/Function Data Context、Scope/Identity、Scenario、Consumer、Disclosure契約から照合する列であり、新しい固定DTOを要求しない。

- `predicate`：問いで実際に解釈された条件。無視されたWHERE/Role/Window等があれば閉包成立や不存在を宣言しない。
- `quantifier`：**存在**、**不存在**、**全件**、**変化**、**効果**など、問いの論理的な強さ。存在の一例を観測しただけでは全件処理や他要素の不存在にならない。
- `observationBoundary`：実Source Reader、Snapshot、InputCut、Artifact Generation、Consumer、Effect Reader等、どの境界まで実際に観測したか。
- `requestedUse`：read-only表示、モデル変更Proposal、逆編集、再実行、外部公開、業務受理等。read-onlyでのpreservedはEditable/Executableを含意しない。

**限定的な再利用の必要条件（十分条件ではない）**：

1. 問いの解釈・Scope・観測対象・既存証拠の版を固定できる。
2. **その問いに必要な依存集合**と解決/未解決の閉包が取得され、実際に消費した依存・Reader/Providerの範囲と区別できる。
3. 変更差分が、その依存または未解決閉包を変えないと確かめられた範囲のみ、read-only Claimの既存資格を保持候補にする。依存が不明なら「無関係」と結論しない。
4. Mapping/Trait/Targetの保存・履行、Current Grant/Disclosure、実Operationの権限、実Effect・Owner受理は、それぞれ必要なOwnerの現行条件で独立評価する。
5. Source→B→Aなど、byteやDigestが以前と一致しても、失効したGrant、Publication Generation、Owner判断、実行Attemptを自動復活させない。

**肯定・否定・全件Claimの非対称性**：あるSnapshotに一件存在するという限定的証拠は、その一件の存在に十分でも、全件処理／不存在／外部Effectには不十分。不存在または「すべて」の主張は、問われたUniverse・Predicate・期間・全Reader/Writer・未観測を閉じた証拠が必要である。追加行・遅延訂正・未観測Writer・部分InputCutが加わると、過去の全件Claimは保持できても現在の全件Claimは再評価対象になる。SQL演算の単調性だけから、Source/Policy/Runtimeを跨いだ資格保存を推定しない。

**既存研究の利用限界**：[Query Impact Draft #4335](https://github.com/hat47x/tei-platform/pull/4335) の `retain / recheck / unknown` は、合成`DerivedIR`とcallerが申告したContext束縛・取得状態に対する**read-only Claim**の判定であり、`operationEligibility=not-evaluated`。同じ語を他領域の製品Contractへ無審査で移植しない。[Required/Consumed二閉包 Draft #4393](https://github.com/hat47x/tei-platform/pull/4393) の合成booleanも実Reader/Buildの証拠ではない。

## 7. 変更種別と必要な再確認の有限行列（既存Ownerへの引渡し）

「再検証しない」は暗黙のPASSではなく、**当該Claimの完全な依存閉包に無関係と確認できた場合の限定判断**である。次表の判断は提案であり、実Source/Provider/Targetを使った試験はNOT RUN。

| 一因子変更 | 原則維持できる歴史的事実 | 再確認対象 / 許されない推論 | 主管 |
| --- | --- | --- | --- |
| 図表Layoutだけの移動 | 旧Sourceの静的事実、旧Projectionの当時の値 | 意味再受理は通常不要。ただし旧View/selection epoch、逆編集資格、公開View一式は用途別に検査 | #1124 / #2497 / Draft #4510 |
| 他FunctionのSource変更 | 旧read-only Query結果・当時のObservation | Required/Consumed・Source/Context・規則の閉包が独立なら限定retain。未解析Predicate、global rule・default Provider変更ならunknown/recheck | #2582 / #2506 / Draft #4335 |
| 選択Provider／Required Trait／Template変更 | 変更前Artifactと試験記録 | 実Binding/Target適合・Build実消費・Consumer再読取を再照合。未選択Plugin追加が無影響とはdefault選択/Lock不変の証拠なしに断定しない | #951 / #3278 / #2831 |
| Published Artifactのbyteは同じで公開世代が異なる | byte同一性と過去の受領 | 世代・Manifest・受領Attempt・current Trustを再評価。byte同一性は同一公開／同一実行を証明しない | #2604 / #2589 |
| ETLのInputCut／時間窓／Correction変更 | 旧Runの履歴Output・receipt | 必須入力閉包、Stage・Checkpoint再開条件、Publication Generation／Consumerを再評価。Commit Unknownの自動再実行は禁止 | #1130 / #2542 / #1174 |
| サブシステム名変更のみ／直接CUD→Owner Command委譲 | 元の意味と過去のDirect CRUD / Effect記録 | 名称だけなら意味IDを推測変更しない。委譲ならWriter閉包・Effect/Tx/Recovery・Owner受理を別途検査 | #3250 / #267 / #1129 |
| Current Grant／Disclosure／Publisher資格が失効 | 当時の認可・受領・静的Source事実 | 現在の公開・編集・実行を再判定。旧PASSやA→B→Aを現行Authorityに流用しない | #2516 / #1174 / #2589 |
| Private Worldのみ変更、Public Slice同一 | 許可された公開Source/見えてよい結果 | 公開Count/Score/Diagnostic/Graph/DOM/Tool応答が秘密差分から変化しないことを検証。非公開件数のredacted表示も自動許可しない | #2516 / #3250 |

**限定判定結果の読み方**：`retain`は「旧read-only Claimの限定利用を続けられる」、`recheck`は「当該契約の再取得・検証が必要」、`unknown`は「安全な選択には必要な情報が不足」。どれも現在の実行Authority・業務受理・強い不存在証明を兼ねない。前後版の比較、実際のSource closure、Current Policy、独立Expectedを既存Ownerに委ねる。

## 8. 最小の横断Witness：誤ったretainと無用なinvalidatingを同時に検出する

一つの固定Source/Contextに、`D1 = OrderLines`、`D2 = PromotionRates`、`D3 = unrelated Settings` を置く。以下は*試験用の架空データモデル*で、製品の既存`order_entry`にこれらのテーブルが存在する主張ではない。Independent Expectedを既存[#2514](https://github.com/hat47x/tei-platform/issues/2514) / [#1180](https://github.com/hat47x/tei-platform/issues/1180)に置き、第二Fixture SSOTを作らない。

| 対象Claim | 変更 | 条件付きExpected（独立Oracleへ引渡す案） |
| --- | --- | --- |
| Q1: `SUM(D1.amount)`（D1のみ読取） | D2だけの行変更 | D1しか読まない完全な解析閉包、同版・同Scopeならread-only Claimはretain候補。現在のGrant・表示Disclosureは別 |
| Q2: `SUM(D1.amount * COALESCE(D2.rate,0))`（Join） | D2変更 | recheck。外部結合・集計の非単調性やD2の部分取得を忘れない |
| Q3: `D3`を読まないQ1 | D3だけ変更。ただしGlobal Ruleやdefault Binding変更はなし | Q1の完全な依存閉包が独立ならretain候補。広域無効化のみを要求する実装はfalse invalidation |
| Q4: `該当D2行は存在しない` | D2の部分取得／未観測Writer追加 | unknown。0行の表示、同一DigestのQuery Plan、古いWindowから不存在を証明しない |
| Q5: Q1を公開したStudio Table | Context/epoch更新・Grant失効 | 静的意味は変わらなくても、旧DOMのEdit/Inspect、現在のDisclosureを再確認。Read成功は逆編集資格ではない |
| Q6: Q2をETL出力し2-member Datasetで公開 | 片方のみ新世代／ACKだけ成功 | Publication/Consumer/Business Acceptedを別に判定、世代混合を完全成果物としない |
| Q7: Legacy observed LとOwner accepted Dで税率の訂正を採用しTarget Tで結果確認 | T=DでL≠T | 受理された意図的変更は旧挙動との単純一致FAILとしない。一方Tの実EffectやOwner受理を合成Outputから推定しない |

評価では、**false retained**（再検証が必要なのに保持）、**unnecessary invalidation**（無関係と証明できるのに全失効）、**unknown erased**（未確認を成功/不存在へ丸める）の3分類を独立に計上する。実Sourceからこの比較票へ必要な列が供給できなければ、まずそのGapを既存Ownerへ返す。新しい全域Impact Engineやメタ生成器を作らない。

### Studio・三要素分析法・認知負荷への接続

一つの問いのread-only回答と、その根拠のSource/Scope/Revision、欠けた依存、表示損失、変更影響を同じConsumerで辿れることを最低基準とする。原典三要素分析法のData/Function/Business相互牽制は、Legacy observed L／Owner accepted D／Target observed Tという移行比較軸とは**直交**する。Method ProfileやTEA固有Rendererから新たなCanonical/権限を作らない（#1124と[Draft #4510](https://github.com/hat47x/tei-platform/pull/4510)がConsumer側Owner）。

比較評価は**正しいretain/recheck/unknown率、Source到達操作数、誤った全件・不存在判断、AIの修正往復、失われた編集情報の検出率**を計測する。既存文章/単一図に対する優位性は未測定。利益が確認できない新しい共通抽象は縮小する。

**§6〜8の証拠上限**：2026-11-12時点の既存GitHub Issue/PRの静的照合と、反証可能な新設計の記述のみ。新Go/Node、実SQL/DB、Studio/Browser、EKI/ETL実Provider、Current Policy、実Owner受理は本追補では**NOT RUN / NOT PROVEN**。

## 9. Evidence Join：独立Witnessを越える保証を成立させる結線資格（2026-11-13追補）

- **状態**：設計候補。2026-11-13のmain `f3b58e7b580f0ecbea1d2feb156f52c2147f63a7` と既存PR/Issueを静的照合。新規Go/Node/Windows/実PGをこの節のためには実行していない。
- **狙い**：一つのSource、操作、Builder、Artifact、Consumerについて、各工程が別々にPASSした事実を単純に足すだけで「一続きの成功」に変えない。一方で実際に結線を観測した段階は、必要以上にunknownへ落とさない。
- **非目標**：Proof Registry、Universal Event Bus、分散トランザクション、第二Source Index、永続的な共通Evidence Schema、全工程を単一Attemptへ強制すること。

### 9.1 Witnessを結ぶときに必要な五つの照合

各領域の既存Result・Manifest・RunLog・Source/Pin・Scenarioから、次の**比較列だけ**を一時的に取り出す。証拠IDが存在する／両文書に同じDigestが書かれた、というだけでは結合資格を得ない。

| 比較 | 結合可能な限定条件 | 証拠が欠ける場合 |
| --- | --- | --- |
| **対象**（Subject） | canonical Kind/IDとScope/Workspace、legacy Source occurrenceの所有元・Path・位置、問いのPredicateが意味上の同じ対象を指すことを各Readerで確認 | 同名、同bytes、同offset、同View IDだけなら対象の同一性はunknown |
| **物質**（Material） | 実Consumed input byte／Selected Artifact／実Consumer byteの対応が、各工程に必要な独立Observationで分かること | Manifestの宣言・計画・SHAだけで実Consumer到達や完全消費を推定しない |
| **試行・因果**（Attempt/causality） | Builder→Artifact→Consumerは必要なAttempt/Generationを明示して対応付ける。独立Readerは別Attemptでもよいが、どのState/Windowを観測し先行処理とどう関係するかを示す | 二つのCIがGreen、同じArtifact hash、時刻の近接だけでは一続きの実行ではない |
| **保存・閉包**（Preservation/closure） | 必須Trait、Function Data Context、Mapping loss、Required集合、実Consumed集合、完全性を**問いごと**に照合 | データが読めた、Mapが正しい、0行、部分Graphが描けたことから全件・不存在・可逆Writeは導けない |
| **現在の資格**（Authority/use） | Consumer用途、Disclosure、Current Grant、操作権限、Owner受理を対応する既存Ownerへ確認 | 正しいTargetの読取・旧時点の成功から公開/編集/Effect/意味採用を推論しない |

ここでいうEvidence Joinは、既存の独立Witness間の**限定された参照関係**であり、単一の自動`verified`状態ではない。例えば`Observed(Build→Artifact)`と`Observed(Consumer←Artifact)`を結んでも、`Qualified(Whole Required Universe)`や`Accepted(Business Meaning)`は自動生成されない。

### 9.2 独立事例で確認した誤結線と現在の上限

| 確認した事例 | 確認された事実・現実の上限 | 横断的な反証 |
| --- | --- | --- |
| [#4584 merged](https://github.com/hat47x/tei-platform/pull/4584)のSource Index | 旧Canonical Subjectの参照0件を`ok:true`で返していた不整合に対し、同一Workspace Load/Indexを使う`InspectionFromLoad`でSubjectを先に確かめ、`SUBJECT_NOT_FOUND`を拒否するよう修正 | `references:[]`はSubject不存在、未使用、Meaning受理のいずれの証明でもない。対象存在とSource参照数は独立Claim |
| [Studio/X-TEA Draft #4570](https://github.com/hat47x/tei-platform/pull/4570) のPin | 合成Source→選択Provider→Bridge→Windows WebView2のread-only道筋は限定的に試験済み。一方同一bytes・Occurrenceで別Workspace/別PathへPinが`held`になる可能性は静的な負例候補（[#4614](https://github.com/hat47x/tei-platform/issues/4614)） | Content digestとOccurrenceだけでWorkspace/Source identityが保持されるとは言えない。実再現・修正の受入は既存Owner |
| [Desktop Build Draft #4587](https://github.com/hat47x/tei-platform/pull/4587) | N2で定義したPowerShell変数を次Step N3で使って実際にCIが失敗し、Step間Artifact参照へ修正後の固定HEADで両Desktop Job成功。次のHEADではN2 esbuild選択入力24件、N3 Go選択入力303件の前後hashを記録 | 工程間の暗黙状態は証拠辺にならない。実選択ファイルの前後hashも、実read-time全byte、外部Toolchain、Stage1 TEI自身のWindows Stage2駆動の証明ではない |
| [実PG Read #4539 merged](https://github.com/hat47x/tei-platform/pull/4539) | Driver/Executor→PG16→Function Data Contextの限定実Readは成立。RLS下の2/3可視、0行、NULL/Maskは`unestablished`として区別。製品`tei` CLI→Studio Consumerは未結線 | 正しいPG結果でも0行を全集合の不存在と断定できず、独立Reader成功をStudio受領とみなせない |
| [B2C Required #4550 merged](https://github.com/hat47x/tei-platform/pull/4550) | Caller申告Graph上でRequired Consumerに至る多段上流のUnknownを保持し、無関係なSiblingの過剰失効を回避 | 単に隣接1辺だけを検査した`declared closed`やGraph全連結成分の無差別失効は、どちらも問いへの正確な再資格化にならない |

**既存Ownerの切分け**：対象存在/Source Index＝#4577/#1120、実PG→Consumer＝#2586/#2514、Studio Pin/Validation/Current Disclosure＝#4614/#1124/#2516、Build実工程＝#2831/#4583、実Effect＝#1174/#1180、Business/Subsystem＝#3250/#1125。横断原則からこれらのOwnerへ実装責務を移さない。[Architecture Draft #4527](https://github.com/hat47x/tei-platform/pull/4527)は「未確定Claim→次の最小証拠辺」の全体図、[Studio Draft #4510](https://github.com/hat47x/tei-platform/pull/4510)はGUI用途別表示と受入のOwner。本書は**証拠辺を結合した際の主張の上限**のみを扱う。

### 9.3 ひとつの縦断Witnessの受入順と、再利用の制約

**先に試すP0**：#4583 B1-A/BのN0 Inspector Source、N1 PoC B、N2 esbuild JS、N3 nested Go/Wails exe、WebView2 Consumerを**単一のBuild試行で**つなぐ。既存BuildIntent/ArtifactSetが表現できる工程を実Builderへ渡し、表現できない工程は`unsupported/unknown`のまま開示する。同一CI/同一Sourceだけで`BuildIntent-qualified`、`Stage1→Stage2 self-build`とは言わない。B2/B3の実再入と実Consumerは次段の独立資格である。

**並行する別P0**：#2586の実PG Readを通常`tei` CLI/typed Operationから受領し、同じ対象・版のFunction Data Contextへ結び、一つのStudioまたは別Consumerが受領した事実を個別に確認する。PG独立Readerは元QueryのResultを照合できるが、Source/Meaning版、Current Grant、全件Universeを自動証明しない。

**反証候補（新しいFixture正本ではない）**：

1. Builder AとConsumer Bが同じJS byte digestを持つが、Artifact generation／読取対象が異なる場合に、同一Attemptの縦断成功を主張しない。
2. 同一`.xead` byte・同一Occurrenceを別Workspace/Pathに配置した場合に、旧Pinの対象を維持したと推定しない。
3. 旧Canonical Subjectに`references:[]`が返ったとしても、Subject存在/不存在は別の実Readerで確かめる。
4. 中間Subjectの2段上流に未解決Requiredが残るとき、対象Consumerの閉包を認定しない。一方、無関係なSiblingまで一括失効させない。
5. 実PG QueryがRLS下で0行を返しても、全Dataの不存在を主張しない。
6. N2からN3へのOutputをShellローカル変数に依存させた場合に、工程間受渡しの証拠が無いとして拒否する。

各反証は[#1180](https://github.com/hat47x/tei-platform/issues/1180)／[#2514](https://github.com/hat47x/tei-platform/issues/2514)の既存Scenario/Independent Expected、各実装OwnerのCIで具体化する。**本節は追加の実装/試験を行っていない**。前段で実測されたPASS/FAILは対象のexact HEADと試験範囲の履歴として保持し、今回の設計の製品資格には合算しない。

## 10. 複数入力の整合カットと、同一bytesから保証を推定しない原則（2026-11-13追補）

- **状態**：横断設計候補・未採用。2026-11-13のPR/Issue静的照合に基づく。新しい共通Cut DTO、Timeline Database、イベントBus、Snapshot Coordinator、Host、Release Verifier、AI Task Storeは提案しない。
- **目的**：同じdigestでもScope/生成世代/公開先/Source状態/Grantが違い、逆に異なるbyteでも正しく保存された業務意味が同じ場合がある。「何が固定された集合か」「どの時点間の因果・整合が確認されたか」を使途ごとに区別する。

### 10.1 同一性は一つのHashやEpochで代用しない

一つのClaimの必要条件に限り、既存Result/Manifest/Source Index/Context/Pin/Receiptの中から次の同一性を照合する。

| 区別する軸 | 何を固定・観測したか | 誤って導いてはならないこと |
| --- | --- | --- |
| 業務意味・Subject | Accepted Meaning Revision、typed SubjectとScope | 同名・同ID風の外部表現・View occurrenceが同一Canonicalとは限らない |
| AuthoringとSource実体 | 保存済Source bytes/位置、未保存Editor buffer、Source Index fingerprint | Dirty Bufferの編集中R1がBuildに使われた、またはR0へ自動受理されたとみなさない |
| 入力集合の整合カット | Sourceごとの取得Snapshot/Cursor、業務期間、物理観測時刻、必要集合・取得粒度 | 複数Readerの取得が成功しても、**単一の整合したDB/業務時点**を観測したとは限らない |
| Buildと実消費 | 宣言Build Intent、選択Builder/Toolchain、実消費Source/Embed/外部依存、Attempt | 実Artifactが同byteでも当該Build Planが実行されたことにならない |
| 公開と配送 | Artifact bytes、Release Descriptor、署名対象、Publication Generation、配送/受領Attempt | HTTP GET成功・ACK・署名成功から現在Publisher Trust、全Member取得、業務取込みは推定できない |
| Current Use | Workspace/Source Pin、Consumer用途・Disclosure、現在Grant/Owner受理 | 同digest・過去のRead成功を現在の公開・編集・実行資格に流用しない |

**整合カット**とは、各Reader/Process/Consumerが何をどの版で観測し、その結果を一緒に扱える保証があるか、という**比較上の範囲**を指す。永続型や一律のグローバル時刻を要求する語ではない。DB Transaction Snapshot、Fileの確定Marker、HTTP Releaseのpin、Stage1/Stage2 Process Imageなど、それぞれの独立した取得境界を同じものと偽称しない。

### 10.2 結線の強さを、対象と用途に合わせて制限する

**限定的な同時利用の条件**（各Ownerが対応付け可能な場合のみ）：

1. 二つ以上の入力Sourceには、各々のRevision、Data/Business time、取得境界、可視範囲（RLS/Tenant等）、InputCutを記録できる範囲で明示する。**同じwall-clock timestamp**や同じ業務期間名から分散全体のatomic snapshotを生成しない。
2. 変換・生成の入力を結合する際、Required集合とActually Consumed集合を別扱いにし、未読込Plugin・動的依存・外部Toolchain・未確定Write/Correctionがあれば、全入力閉包は`unknown`に留める。
3. ArtifactとOutputが同じbyteでも、Production/Publication Generation、Source-Artifact Binding、実Consumerの取得Attemptを別の証拠で対応付ける。符号化・改行変換・暗号化等でbyteが異なる場合は、明示した変換とLoss/Conformanceを検証するまで内容同値としない。
4. 署名されたRelease宣言があることと、Consumer自身が**未信頼HTTP responseから宣言を取得し、検証し、各Member実byteまで照合したこと**を分ける。Current Publisher委任、RegistryのOwnership、公開先のDisclosure、Provider Runtimeの現在権限はさらに別のClaim。
5. Studioの表示中Source R0／未保存Buffer R1／AIが基にしたR0／明示Accepted Meaning／実Builder消費Inputを分ける。人間がBを編集中にPin Aを裏で再確認しても、Editor/Graph/Dirty Buffer/Undo/選択Bを無断変更しない。Source再取得やAI応答はProposal/Reviewの入口であって自動Save/Apply/Buildではない。

これらはすべて対象Claimの**必要条件候補**であり、列の存在だけで完全性や業務受理を証明しない。開示制限のあるConsumerには、Private Worldの有無・件数・欠落理由・検査Coverageも許可なしで返さない。

### 10.3 既存Ownerへ渡す圧力試験（新しいFixture正本を作らない）

| 反証対象 | 否定する危険な短絡 | 既存Ownerと優先度 |
| --- | --- | --- |
| 同じ業務期間だがDBとファイルを異なる時点に取得し、後から訂正行が到着 | `period同一 = global consistent snapshot`、`partialの0件 = 全件不存在` | [ETL #1130](https://github.com/hat47x/tei-platform/issues/1130) / [MFT #2542](https://github.com/hat47x/tei-platform/issues/2542)、P1 |
| データAは新世代、データBは旧世代で、どちらのbyteも正しい | 各member digest PASS = Dataset全体の同一世代Publication PASS | [Artifact #2604](https://github.com/hat47x/tei-platform/issues/2604)、P1 |
| [Registry E1-B merged #4529](https://github.com/hat47x/tei-platform/pull/4529)ではConsumerがHTTP Artifactを取得する一方、署名対象Release宣言はlocal fixtureから取得 | 実HTTP byte PASS＋別途Signature PASS = Consumerが署名済み宣言もHTTPで検証した | [Registry #2589](https://github.com/hat47x/tei-platform/issues/2589)のE1-C、P1。別のVerifier/Trust Rootを本書に作らない |
| AI提案R0と編集中Dirty Buffer R1が異なり、別タブのPin Aを再確認 | 古いSource/Pinned Consumerの再読込 = 編集中Bの内容と現在操作資格も更新された | [Studio協業 #4515](https://github.com/hat47x/tei-platform/issues/4515)、[UX Draft #4605](https://github.com/hat47x/tei-platform/pull/4605)、P1並走 |
| 固定Contract/生成JSを同じCIで観測したが、BuildIntentはNode/esbuild/Nested Wailsを実駆動しない | 同Attempt・同digest = Stage1自身によるStage2 Desktop Build/完全Consumed閉包 | [Self-build #2831](https://github.com/hat47x/tei-platform/issues/2831)、[Desktop P0 #4583](https://github.com/hat47x/tei-platform/issues/4583)、**最優先P0** |
| 別Workspaceに同じ`.xead` bytesと同一Occurrenceがある | 同Content fingerprint = 別Workspaceでも同じSource Pin/編集対象 | [#4614](https://github.com/hat47x/tei-platform/issues/4614)、[#1124](https://github.com/hat47x/tei-platform/issues/1124)、P1並走 |

**受入資料と実験**：各Ownerの既存Scenario SSOT [#1180](https://github.com/hat47x/tei-platform/issues/1180)、独立Expected [#2514](https://github.com/hat47x/tei-platform/issues/2514)、現在Authority/Disclosureの既存契約に委ねる。最小の正負対照は、(a)byte同一でもScope/Publication/Attemptが異なる、(b)宣言された変換によりbyteが異なるが意味同値を検証できる、(c)必要閉包未取得、(d)既知の無関係変更、(e)Private Worldのみ変更、を別々に用意し、`false retained / unnecessary invalidation / unknown erased`を記録する。

**禁止する横断的な主張**：Source digestの一致からMeaningやGrantを昇格、Manifest合格から実Builder呼出しを昇格、ファイル受領から業務Commitを昇格、合成private fixtureから実Disclosure資格を昇格、局所Readから全件/不存在を昇格。新たなHard Gate追加も、Studio DesktopセルフビルドP0の他レーン依存化も行わない。

**状態更新（2026-11-13）**：E1-B #4529はmain `b03c415e5fd13d8feec96d9669d96d6db9035e8c`へ統合済み。HTTPの実Artifact byte受領という限定事実は維持するが、signed Release Descriptor自身のuntrusted HTTP取得→Consumer独立検証→Member byte検証は次のE1-Cであり未成立。別のLocal Fixtureの署名PASSをHTTP配布の一続きの署名済み検証へ昇格しない。

**今回のEvidence上限**：既存GitHubのIssue/PR・文書の静的照合により新しい設計を記述しただけ。Registry E1-C、複数Source atomic snapshot、AI協業Studio実操作、B1/B2/B3 Desktop self-build、本書の反証E2E、Owner採否は`NOT RUN / NOT ACCEPTED`。

## 11. Decision-relevant Qualification Slice：変更を局所化するための反実仮想を含む依存閉包（2026-11-13追補）

**採否**：設計・研究候補。新しい共通Closure Solver、Proof Planner、Global Impact Cache、永続Claimスキーマ、Hard Gateは提案しない。各Ownerが持つ既存IR/Graph/Build Plan/Source Index/Scenarioの**読み取り専用比較票**として扱う。

### 11.1 観測された消費入力と「変更後に必要になり得る入力」は異なる

ある問い`Q`の限定的な使用資格を変更`Δ`の後も保持するには、**前回の実Consumed集合と変更集合が非交差**であるだけでは不十分である。少なくとも次を区別する。

| 依存層 | 閉包・証拠 | 不在と誤認しやすいもの |
| --- | --- | --- |
| Declared Required | 元のMeaning/Contractが要求するData、Function、Constraint、Result Slice、Trait、Targetの必要集合 | 宣言に載らない動的Plugin、暗黙の環境、負の条件、外部Writer |
| Resolved / Selected | その版・Scope・Context・Profile・Policyで実際に選択されたProvider、Source、Builder、Operation path | 条件分岐、選択規則、Feature flag、外部設定の変更で新たに選ばれる別経路 |
| Actually Consumed / Observed | 固定AttemptでReader/Builder/Consumerが読んだbyte・依存と、Independent Observation | 未実行分岐、late input、opaque解析、不完全Tracer、非公開依存 |
| Qualification obligations | 当該`Q`を利用・編集・公開・再実行するのに必要なCompleteness、Source identity、Output loss、Authority、Effect | read-onlyが有効でもWrite/EffectやCurrent Grantは別資格 |

**変更後の反実仮想**：旧AttemptがAだけ読んだとしても、「Bの変更が選択条件を変え、次のAttemptではBを読む可能性」があるなら`B∉OldConsumed`から`NoImpact`を導かない。Queryの`NOT EXISTS`、LIMIT/ORDER BY、JOIN・集合差、RLS、Provider既定値、Build tag、Template/Toolchain選択、ETL後着訂正、SubsystemのCommand委譲の前後も同じである。

読み取り専用`retain`を限定主張できる必要条件は、(1)質問・Subject・Scope/Context・契約版・選択規則の対応が確認でき、(2)問いに関係する**Potential/Required/negative/control**依存の閉包が十分に開示・解決され、(3)その閉包から`Δ`への影響が無いと確認され、(4)現在のRequested Use/Disclosure/Grantの別要件が満たされることである。これらの条件は**十分条件ではなく**、各Ownerの実Readerや独立Expectedを置換しない。

変更内容が実際に必要な依存へ到達した場合は`recheck`。必要な依存が未解決・非開示・動的で、変更への到達可否が判定できない場合は`unknown`であり、**無関係と断定しない**。ただし全Graph連結成分へ無差別に失効を広げることも正当化されない。明示的な拒否条件・意味不一致は単なる`unknown`に丸めない。公開Consumerへの理由・欠損件数の開示は#2516のPolicyに従う。

### 11.2 「論理更新凝集」と「変更・再検証の局所性」は別の目的関数

サブシステム分割では、外部へのDirect CUDを減らしても、委譲したOwner Commandの実Effectや多段上流の依存が増減し得る。**Direct CUDが0**は**越境再資格化が0**、**実Transactionが局所的**、**変更負担が小さい**ことの証明ではない。

横断比較では次の別々の測定値を保持する。各値は問われた範囲・Input/Output・独立Expectedに相対化する。

| 観点 | 指標候補（提案） | 本書で判断しないこと |
| --- | --- | --- |
| Ownership / Write | Declared Direct CUD越境、Unknown Writer、Owner-side Command委譲 | 実Effect/Txの保証・最終Subsystem名称 |
| Requalification locality | 実際に必要だった再検証数、誤retain、誤全失効、unknown残存と理由 | 全製品の影響解析の完全性 |
| Time / Cost | full対selectiveの再実行数・待ち時間、分析/AI再確認の往復・Token、修復費用 | 測定前の生産性改善率 |
| Reversibility / Safety | Mappingの損失、旧Source Pin、部分Source/Private Worldの開示、Edit eligibility | 展開/配備資格やCurrent Grant |

候補の優劣は、必要なOwner/Constraintを満たす候補間で**同一の問い・試験条件・観測版**を比較して初めて判断する。`#3250`のCandidate ScoreやDeclared Graph reachabilityを製品全体の正解・Globally optimalとみなさない。Measure側のOwnerは[#2600](https://github.com/hat47x/tei-platform/issues/2600)、Subsystem側は[#3250](https://github.com/hat47x/tei-platform/issues/3250)、Query側は[#2582](https://github.com/hat47x/tei-platform/issues/2582)。

### 11.3 既存Scenarioへ渡す最小の独立対照（すべて新規実行はNOT RUN）

| 対照 | 変更と観測 | 必須の判定 |
| --- | --- | --- |
| DR-01 | 旧AttemptはProvider Aだけ消費。新RevisionでFeature flagがB選択へ変わるが、旧Artifact byteは偶然同じ | `OldConsumed∩Δ=∅`だけでretainしてはいけない。Resolved selectionと実Builder/Readerを再確認 |
| DR-02 | Q1はD1だけで完全に導出。D2の値だけ変わり、Q1に到達する否定依存・default Bindingは無しと明示確認 | Q1 read-only限定retainを保持候補にする。Q2=D1×D2はrecheck。全Context失効は過剰 |
| DR-03 | `NOT EXISTS`対象が部分取得、別Readerで0行。後着D2またはRLS外行あり | 0行から不存在を主張せずunknown。特定Subjectの存在・Canon IDも別検査 |
| DR-04 | Direct CUDをOwner Commandへ移し、外部Writerは消えたが実Effectは別Subsystemへ波及 | Direct CUDの局所性とEffect/Tx/再検証局所性を独立表示 |
| DR-05 | Build Planに宣言した入力はA、実Consumedに未宣言B、外部Toolchain依存未観測 | 実Outputの成功や実消費の列挙だけで全Required閉包・SelfBuild G4を主張しない |
| DR-06 | Public Sliceは同じでPrivate Worldだけ増加、理由コード/Count/Score/DOMが変化 | Published/Studio/AI ConsumerはPrivate有無を漏らす「安全な局所説明」を出さない |

採点は正解`retain/recheck/unknown`だけでなく、**根拠と未確認の局所化**、誤全件/不存在・誤Edit/Effect許可、False Invalidation、既存Source/Ownerへの到達操作数を別に記録する。Scenario SSOTは[#1180](https://github.com/hat47x/tei-platform/issues/1180)、独立Expectedは[#2514](https://github.com/hat47x/tei-platform/issues/2514)。Query Draft [#4335](https://github.com/hat47x/tei-platform/pull/4335) と合成Required/Consumed Draft [#4393](https://github.com/hat47x/tei-platform/pull/4393) は、それぞれの固定範囲から実Target一般へ昇格しない。

**進め方**：まず#2600の既存Build/Incremental Referenceと#2582のQuery局所判定、#3250の分割評価を**同一の説明原則で照合**する。必要なら既存Ownerの一実Source・一機能・二Consumerで反例を実装する。Studio Desktop Self-build [#4583](https://github.com/hat47x/tei-platform/issues/4583) のB1-B→B2→B3を直接P0として優先し、この比較表を新たな前提Gateにはしない。測定によって理解・変更費用が下がらないなら、横断共通実装を作らず設計上の説明・試験観点のみに縮退する。

**本追補の証拠上限**：2026-11-13現行main`b03c415e5fd13d8feec96d9669d96d6db9035e8c`および既存Issue/PRの静的読取。ここで示すDR-01〜06は反証候補であり、実Provider/Builder、Query改版、実DB、Studio、Product Consumer、Current Grant、Owner受理の新しい試験は`NOT RUN`。
