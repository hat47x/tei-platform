# 三要素分析方法論とDiagram Projectionの接続契約（設計候補）

- **状態**: §1〜18の図法・具体方式は評価待ち（未採用）。**§19の『全体モデルを図表で理解・探索でき、標準固定箇所とCustomを明示する』は2026-11-08に利用者が明示したTEI製品標準要件**。その要求を満たす個別Schema/Generator/GUIの実装・検証は未完であり、文書更新をProduction適合とみなさない。
- **整理日**: 2026-10-31
- **調査基線**: TEI main commit 4325ca79a52bf0d576db6fb4027bdc516b1023fd。XeadModeler M49公開ソース commit 68f1535b1c5dd80a327dfced37be401ecd3093c9（静的確認、M50/実GUI未実証）。
- **目的**: Data / Function / Businessの三視点、X-TEA系の図法、具体例、定義ジャンプ、図上編集を、TEIの既存の意味正本と共通構造から再現する最小縦断設計。
- **関係Issue**: [#1124](https://github.com/hat47x/tei-platform/issues/1124)（Studio統合）、[#1126](https://github.com/hat47x/tei-platform/issues/1126)（Data/ER）、[#1125](https://github.com/hat47x/tei-platform/issues/1125)（Business/DFD）、[#1127](https://github.com/hat47x/tei-platform/issues/1127)（Function生成・利用）、[#1131](https://github.com/hat47x/tei-platform/issues/1131)（具体例）、[#950](https://github.com/hat47x/tei-platform/issues/950)（legacy Source Adapter）、[#951](https://github.com/hat47x/tei-platform/issues/951)（Capability Contract）、[#267](https://github.com/hat47x/tei-platform/issues/267)（Shared Acceptance）。
- **非目標**: X-TEAの画面・Java/Swing・内部ID・XMLをTEIの正本へ移植すること、一般Canvas/Plugin Resolver/Acceptance/外部UI制御の第二実装、旧資産の無許諾再配布。

## 1. 決める境界

三要素分析プラグインは、ER図などを個別に描くRendererの寄せ集めではなく、**意味体系、図法、操作解釈、検証を束ねて提示するAuthoring Method**として設計する。ただし一つの巨大なプラグイン実行単位を必須としない。

| 層 | 所有するもの | 所有しないもの |
| --- | --- | --- |
| Canonical Semantic / Semantic Extension | Data/Relation/Function/Businessの受理済み意味、採用した方法論固有の意味規則 | 図形の座標・DOM・個々のConsumer状態 |
| Authoring Method Profile | 図の種別、必要なSemantic Interface、利用するProjection/Validation/編集操作の**明示的な組合せ** | Accepted meaningの第二正本、暗黙のAcceptance権限 |
| Diagram Projection | Qualified Refから表示Occurrence・slot・connectionへの導出、欠落・未解決の報告 | 暗黙のSource更新、配置を意味に昇格すること |
| Notation Profile | Glyph、Port、Line、Label、表示文法・操作の提示 | 関係が業務的に成立するとの判定 |
| View Layout / Presentation | 座標、折れ点、フィルタ、図の段階表示、手動注記 | Semantic ID、業務Action順序の確定 |
| Studio Consumer | 描画、Selection snapshot、Focus、Navigation、readonly Inspect | 実行Host、受理Authority、業務Effect |
| Source Adapter | 元構成要素、Source Token、未知情報、原典の出典と曖昧性 | 旧ID→Canonical IDの自動昇格 |

**決定候補A — Method ProfileとSemantic Extensionの区別**: Profileを導入しただけでAccepted意味は変わらない。関係種別の成立規則や、Acceptanceを遮断するValidationなど、意味の解釈に影響するものは任意の表示Pluginではなく、明示したSemantic Extension/Interfaceとそのexact fingerprintをSemantic Setへ含め、必要なShared Acceptanceを要求する。図法・座標・閲覧専用の補助はMeaningを変更しない。Profileのパッケージ順、インストール済みPlugin、Registryの最新公開版で現在の意味を暗黙に変えない。[Semantic Extension依存のPre-ADR](../research/SEMANTIC_EXTENSION_DEPENDENCY_PRE_ADR_2026-10.md)を再利用する。

**決定候補B — Contract/Provider/Instance**: 論理Method、Capability Contract、実装Provider、配置されたPlugin Instance、配布Package/Artifactを分離する。同じRenderer Typeを異なるStyle/Workspaceで複数Instance化しても同じ意味体系としない。製品固有Core enum追加や第二Resolverは行わず[#951](https://github.com/hat47x/tei-platform/issues/951)へ接続する。tei.experimental.*等は[#2298](https://github.com/hat47x/tei-platform/pull/2298)（2026-11-03 UTCにmerge済み、提案中）の識別子**候補**であり正式な予約・実装済みではない。分散公開Registryは発見/第三者検証に限定し、Semantic Authority/Provider Conformance/Consumerの利用許可を与えない。

**決定候補C — Host/Target**: [#2226](https://github.com/hat47x/tei-platform/pull/2226)（2026-11-02 UTCにmerge済み。[ADR-0043](../adr/0043-target-native-artifacts-without-tei-host.md)）の「TEI独自Host原則非必須」を前提に、純粋なProjection/ValidationとUI Adapterを分離する。Wails/Theia/Browser/Golang processのいずれも必須基盤としない。ADR番号の衝突は[#2341](https://github.com/hat47x/tei-platform/issues/2341)へ委譲し、ここで番号や採否を確定しない。

## 2. 共通Projectionの最小構造（Reference候補、確定schemaではない）

既存の[modelref契約](CROSS_VIEW_TYPED_REFERENCE_CONTRACT_2026-10.md)による Key{Scope,Subject(kind,id)} / Ref / Evidenceを再利用する。画面ID・旧XML ID・Source位置・物理PK/行キーはQualified Keyではない。

| 項目 | 役割・最低不変条件 |
| --- | --- |
| DiagramScope | 対象のQualified Key、include kinds、選択したrelation集合、フィルタ |
| ProjectionIdentity | Source model revision、Semantic Setのexact fingerprint、方法論の意味規則fingerprint、projection schema/rule version |
| Occurrence | view-only occurrence ID。subject Refは任意。**同じSubjectの複数Occurrence可** |
| Slot | 図形内部の行/ラベル/Port。必要なら別のAttribute/ActionのQualified Refへ接続 |
| Connection | source/target occurrence/port、typed relation Refまたはderived provenance、presentation-onlyの区別 |
| Evidence | accepted / proposal / source-observation / derived。表示名や観測tokenだけでacceptedとしない |
| Coverage | 指定scope/kinds内のrepresented/omitted/unresolved、対象集合と列挙根拠、complete/partial/unverified |
| Layout | projection対応の手動座標、edge route、visibility、annotation、layout revision。Semantic Sourceとは分離 |
| NavigationSnapshot | active render epoch、scope付きviewID→Key対応、projection/source版。仮置きmapは描画成功まで非公開 |

図形全体はDataを、slotはFieldを指すことができる。意味上のSubjectに対応しない「メモ」「囲み線」も描画可能だが、意味編集やCanonicalジャンプを与えない。同一Subjectが別図面に存在してもOccurrence・配置・表示項目はそれぞれ独立する。

**三種のConnection**: (1)正本Relationを参照するSemantic Edge、(2)明示した規則と依存参照から導いたDerived Edge、(3)図面だけのPresentation Edge。Derived Edgeには導出経路・rule fingerprintを残す。Task→Function→Dataという導出線から、Task自身のC/R/U/Dや実行時Effectを捏造しない。

**完全性の限界**: complete-for-scopeを宣言できるのは入力対象集合が現在版で完全に列挙され、フィルタ・未解決・未対応が数量化できる場合だけ。「図中になければ存在しない」は、固定したscope/kinds/rule versionの中でのみ成立する。モデルが業務要件を網羅する保証は別。非表示要素数とunknownを常に区別する。

## 3. 透明な図面パイプライン

TEIの標準型付きデータ構造の合成で、以下の実際の処理をすべて宣言可能にする。図面は独立した開発パイプライン正本ではなく、その実際のノード/辺/版/Evidenceから生成する。

1. Accepted World + exact Semantic Setを解決する（別のSource Index/観測結果をCanonicalとしない）。
2. Qualified Target Setを列挙する。未対応/未解決/非表示を記録する。
3. Method規則を明示してProjectionを導出し、rule fingerprint・派生根拠・coverageを固定する。
4. Notationを適用してGlyph/Port/Labelへ変換する。表記で新しい意味を創作しない。
5. Layoutを別Revisionで照合し、失効/ambiguousなら自動移譲しない。
6. Rendererへ表示し、選択Snapshotを正しいepochで公開する。

いずれかの処理が図中に存在しないのにバックグラウンドの隠れたhookで意味を追加することは禁止する。逆に、Viewerが表示のために行う測定・ヒットテスト等の技術的処理はMeaning pipelineと区別してよい。Maven/Gradle等の特定Build知識に依存せず、一般的なtyped transform・relation・evidenceの合成として扱う。

## 4. TEA固有の解釈・旧X-TEA互換

| 旧機能/証拠 | 扱い | 既存Owner |
| --- | --- | --- |
| Table/Field/Key/Relationship、FAMILY/REFFER/SUBTYPE | raw type/ID/SortKey/Source locatorを保持。論理キーと物理FK、候補キーを自動同一視しない。意味規則はTEA Semantic Extension候補 | #950 / #1126 |
| Data Model上のInstance Text / SortKey | 旧テキストと表示順をsource-faithfulに保存。型付きCaseは明示したmappingと検証を経由する別候補 | #1131 / 既存Example ER Slide M1 |
| Business DataflowのProcess/Subject/Store/手作業、SlideNumber、固有Glyph | Glyph種別をそのままCanonical業務Subject kindへ列挙しない。業務Process/Action/Event/Roleと表示記号を別に解釈 | #1125 |
| Function/TaskAction/TaskFunctionIO/IOTableの連鎖 | Function-level C/R/U/DとAction-specific effectを区別。未解決の参照を先着一致で自動bindしない | #950 / #1031 / #1132 |
| 旧Sourceのインポート・同名一致・再採番 | XML ID/SortKeyの一致をSemantic同一性・更新規則の根拠にしない。元Sourceの重複・欠落をLoss/Unresolvedへ | #950 / #1805 |
| 三視点の参照/ジャンプ | type/scope/revision付きReferenceへ。Readerのsource locatorと正本定義ジャンプは別の操作 | #1120 / #1124 |

M49の[DialogAddRelationshipOnDatamodel.java](https://github.com/xead/XeadModeler/blob/68f1535b1c5dd80a327dfced37be401ecd3093c9/src/xeadModeler/DialogAddRelationshipOnDatamodel.java)では、一本の関係線の作成がField/Key/Relationship/IOTableFieldの複数変更へ及び得る。したがってSource変更は単純な描画操作で確定せず、型付きの**複合Change Plan**を用いる。現行TEIが複数Sourceの原子的受理を保証すると未確認のまま仮定しない。

M49の[DialogEditInstanceOnDatamodel.java](https://github.com/xead/XeadModeler/blob/68f1535b1c5dd80a327dfced37be401ecd3093c9/src/xeadModeler/DialogEditInstanceOnDatamodel.java)は自由記述のInstance Text/Slide順を扱う。Slide順を業務時系列やAction effectに変換しない。M50の同等性、GUIでの表示忠実度、未公開Reference Modelの権利は未確認。公開Repoには未許諾の原資料を転記しない。

## 5. 図上操作・定義ジャンプ

| ジェスチャー | 解釈 |
| --- | --- |
| node/slot/edge select | Consumer側でactive epoch→Qualified Key/Provenance→Inspect。曖昧・staleは読み取り候補/診断のみ |
| node/field double-clickまたはEnter | 提供可能なViewを列挙し、Canonical定義・旧Source観測・派生説明を区別してnavigate |
| move/resize/route/filter/slide | Layout/Presentationの編集。Semantic Changeではない |
| connection creation/deletion | 「非表示か意味の変更か」を選ばせ、必要ならMethod-specific typed Edit Intentへ |
| semantic edit | current Source readback→precondition/impact/ambiguity→typed Change Plan→既存Shared Acceptance→別readback/reprojection |
| AIが編集案を提案 | 人手のGestureと同じ適格性/受理境界。AIがMethod規則や権限を自己授与しない |

編集Intentには必ず origin diagram/occurrence、Qualified targets、操作種別、base Semantic revision/Set、projection fingerprint、意図を含める。Change PlanはそのIntentから導出される候補であり、未受理時はsourceWrites=[]。Functionの読み取りProjectionからCustomerへの書込みを逆推論しない。UndoはLayout UndoとSource Revisionに基づく意味変更候補を分ける。

**Consumerの負例は省略しない**。[Studio freshness設計候補](STUDIO_CONSUMER_FRESHNESS_BOUNDARY_2026-10.md)のとおり、G2の描画がmissing_endpointで拒否された時、G2 mapを公開して残存G1 DOMのclickをG2 Subjectとして解釈しない。Source R10→R11へ更新後にDecision ledgerが失敗しても、旧Inspect結果をcurrent表示に戻さず、Acceptを自動再試行しない。Source/ledger/Owner acceptance/Target readinessは別証拠であり、Graph epochだけではSourceの鮮度を保証しない。

Studioの選択/ジャンプと[#2391](https://github.com/hat47x/tei-platform/issues/2391)の外部UI操作は別。後者のlocate/act/wait/observeおよびTest/Production Artifact GateをMethod Profileの一部として再実装・必須化しない。

## 6. 最小Referenceと撤回条件

**TEA-D0（read-only）**: 独自の受注・受注明細・得意先Dataと論理キー/関係を利用し、次を一つのFixtureで確認する。既存[Example ER Slide M1](EXAMPLE_ER_SLIDE_M1_2026-10.md)のCase/Projection、[modelref](CROSS_VIEW_TYPED_REFERENCE_CONTRACT_2026-10.md)、[Studio Graph](../../reference/tei-studio-graph/README.md)、[DFD Layout](../../reference/business-system-design/DFD_VIEW_LAYOUT_REFERENCE.md)を比較・再利用。初回に新たなCanonical schema、大型Canvas、Plugin Registryを作らない。

| ID | 最小Assertion / expected failure |
| --- | --- |
| D01 | 同じSource/Rule/Scopeから同じQualified Projection。同一Subjectを別Diagram/別Occurrenceへ置いても参照は不変 |
| D02 | Data nodeとField slotがそれぞれ正しいQualified Refへ解決し、同名別Scopeを誤選択しない |
| D03 | 描画更新失敗時はpending map非公開。旧DOMが残ってもSource鮮度が変わればInspectを起動しない |
| D04 | move/route/slideでSemantic fingerprint不変。表示の独立した保存/Undoが可能 |
| D05 | View再生成時に同一Subjectの配置は候補として再使用。split/merge、ambiguous移行を自動確定しない |
| D06 | hidden/unsupported/unresolvedのCountを保持。入力対象集合が不明ならcompleteを主張しない |
| D07 | Relationship追加IntentはKey/Field/Function影響の複合候補へ。Source writes=0、Accepted=false |
| D08 | legacy同名・重複ID・異なるSortKeyをCanonicalへ黙って対応付けない |
| D09 | 表示Style plugin追加でAccepted meaningとSemantic Lockは不変。規則変更を暗黙有効化した場合は拒否 |
| D10 | 遅延InspectとAccept後ledger failureを同一時系列で扱い、旧current-readyを失効、auto Accept=0 |
| D11 | スライド順だけを変えてもCase Snapshot/業務実行順は変わらない |
| D12 | manual/external StepからFunctionの存在・CRUD effect・Ownerの受理を推定しない |

**進行順**: E0＝Projection/Slot/Connection/Coverageの最低形を既存Fixtureで型検証 → E1＝既存Graph ConsumerからData/Fieldジャンプと更新拒否負例 → E2＝TEA Data NotationとM49 source-only比較 → E3＝複合Edit Intent→既存Proposal/Impact/Acceptance（新規Source書込み前に停止）→ E4＝Business/Function/Exampleと相互参照。E0/E1は#1124、E2は#1126、Businessは#1125、Caseは#1131、plugin契約のProduction化は#951へ委譲する。

**評価指標**: 隠れた意味変換数0、誤Scopeジャンプ数0、source-fact→accepted無断昇格0、非表示要素の説明率、未解決を残せる率、同じProjectionの決定論、旧手動配置を安全に引き継げる割合、三視点での利用者タスク達成（今後の独立実測）。定量閾値は独自Fixtureと人による評価を取得するまで固定しない。

**撤回条件**: 既存modelrefとDiagram Projectionの二重正本を要求する、Layoutを永続化するためにSemantic IDへ画面IDを混ぜる、Method ProfileのinstallだけでAccepted意味が変わる、追加抽象が既存Referenceに比べて有用な検証/操作を生まない、と分かった場合は新規抽象を縮減する。

## 7. 既存Ownerとの重複排除と残課題

[#1120](https://github.com/hat47x/tei-platform/issues/1120)の型付き参照、および[#1132](https://github.com/hat47x/tei-platform/issues/1132)の横断診断は既存完了範囲を尊重し、新しいReference Graph/Validatorを再実装しない。#1124のConsumer guardとEdit↔Sourceを接続する未実証箇所、#1126のER種別規則、#1125の業務図法をそれぞれOwnerへ戻す。

現時点で未決: TEA Semantic ruleの最小Extension interface、図形/edge/slotの最小schema、旧FAMILY/REFFER/SUBTYPEの厳密な意味対応、Businessの正本SourceとAction Tree、M49/M50・実GUIの対応、Layoutの長期保存/再対応付け、ノード密度とアクセシビリティの実機合格、Provider/Plugin Conformance、原子的複数Source更新、方法論間のRule競合。設計文書で推論により埋めず、各OwnerのReference Gateへ残す。

**検証**: 2026-10-31時点、公開GitHubの関連Issue・main文書・公開M49 sourceを静的照合。本文は設計仮説と参照先の接続のみ。コード実装、Go/Nodeテスト、GUI、M50挙動、人間による意味受理、repository-health-checkは**NOT RUN / NOT VERIFIED**。

## 8. 図中にない対象を不存在と判定するための五層（追加設計候補）

他チャットの「図中になければ実際に存在しない」要求は、**対象を完全に列挙した閉じた管理範囲**についてだけ判断可能とする。未表示・権限不足・Source未読込・外部Plugin・実Runtimeを、要素数ゼロと取り違えない。

| 完全性の軸 | 最小の確認事項 | 不足時 |
| --- | --- | --- |
| 1. 要素 | Source/Scope/種類/Filterに属する対象集合を漏れなく列挙した根拠 | 未表示と不存在の識別不能 |
| 2. 関係 | 対象間のtyped relation、複合key、導出関係のRule・端点role・未解決を列挙 | 線がないことを無関係の証拠にしない |
| 3. 表示 | 対象ごとに rendered / hidden / omitted / unsupported / unresolved のいずれかへ対応付け、理由・数を報告 | 描画の省略を対象の消失として解釈しない |
| 4. 構成 | 実際のMethod/Notation/Provider/Component Instance/Rule集合と依存が確定しているか、当該図には無関係として除外した理由 | install済みPackageや宣言を利用中の実Instanceと推定しない |
| 5. 時点 | Source、Semantic Set、Projection、Consumer snapshot、必要なProvider/Build観測の版・適用範囲 | 過去の完全性を現在の不存在主張へ流用しない |

単純なDiagram Coverage.countだけでなく**対象集合、対象定義、使った規則、適用版、列挙の根拠**を最低限記録する。現行のモデルを読むだけのER図にBuildやProviderを要求することはないが、その場合「実行構成については主張しない」を明示する。通常の画面表示から、モデル全体や業務要件全体、外部環境の不存在を推定しない。

**不存在判定の三値**: specified closed scopeにおける absent（不存在を確認）、present（存在を確認）、unknown（未観測・権限不足・未解決）。absenceを返せるのは、該当kindを含む完全なSource集合、Relation規則と対象範囲、適用版、非表示の区別が成立し、問いがその閉包内にある場合だけ。UIで非表示でも、元の対象が存在するならpresent。秘密・参照権限が不足する場合は、その対象の有無や件数を無断公開せずunknown/非開示とする。Coverageは安全性・開示判定の代わりではない。

**対応する既存資産**: [三視点参照のSource/Evidence](CROSS_VIEW_TYPED_REFERENCE_CONTRACT_2026-10.md)、[既存ER/物理モデル比較](../../tools/reference-er-table-crossview/projection.go)、[Xead Cross-viewのSource再照合](../../tools/reference-xead-cross-view/README.md)。前二者は全対象の存在・実業務正しさを一括保証しない。Graphの描画成功もSource列挙の完全性を保証しない。

## 9. 二項Edgeで表現しきれない関係の扱い

現行 modelref.Edge は From→To の**二項の型付き参照**、現行Studio Graphは直線の二項表示である。三要素分析では、Composite RelationKey、TaskAction→TaskFunctionIO→Function→IOTable→Table、複数のactor/step/conditionを持つHandoff等を**一つの直線へ意味的に縮約できない**場合がある。新しいCore Hyperedgeを先に作らない。

**最小投影形**: まず既存の正本側で個々のRelationとそのKey配列、複数の型付き参照、出典集合を保持する。図面上は次のどちらかを選べるようにする。

1. 二項の表示線 + 説明可能な役割付きslot/詳細Inspector。クリックした線から関係そのもの、key対応、導出経路の一覧を開く。
2. 必要な場合だけ「関係Occurrence」をview-only nodeとして置き、各参加要素から role付きportへ接続する。relationを表すview node IDはCanonical identityへ昇格せず、出典が正本を持つ場合だけQualified Relation Refを付ける。

例：TaskAction A、TaskFunctionIO I、Function F、IOTable T、Data Dの観測経路をA→Dという一本に簡略表示しても、**source paths A→I→F→T→D と参照を導いたrule/evidence**を保持し、AがDのwrite effectを起こしたとは表示しない。キー構造もComposite Keyを一本の「FK」と誤記せずField pair集合へ展開できるようにする。契約上の端点roleが未解決ならグラフの欠落として黙って落とさずdiagnosticに残す。

**図表間の関連表**: 同じQualified RefがData表、ER図、Function図、Business図で何回現れたかを、view occurrenceの逆索引として導出する。異なるDiagramでIDが一致するという理由で同じmeaningとしない。複数の意味体系の断面を示すことが利用者/AIの認知負荷を減らしたかは、実操作比較で評価する。

## 10. Declared / Resolved / Observedと業務証拠の直交性

Plugin公開基盤の議論で用いるDeclared（宣言）/Resolved（選択・合成）/Observed（実際の観測）に対応する視点を**既存の証拠から読み取り専用で投影**できるようにする。これをCanonicalな「第四のWorld」や独自Registry台帳へしない。将来Method Pluginを配布する際も、公開Registryでの検証成功やArtifact digest一致だけではConsumerの利用権限、現行Providerの適合、Accepted meaningを証明しない。

同時に、Action/Interaction Evidenceで検討されている四つのレーンは別の軸である。

| 業務・操作上のレーン | 示せること | 示せないこと |
| --- | --- | --- |
| Operation / Selection | 図上の意図、UIで選択した要素、操作要求 | 実対象が決まったこと、業務が確定したこと |
| Declared Transition | 正本で宣言された条件・期待状態遷移 | 当該遷移が実際に起きたこと |
| Observed Effect | 特定Provider/Target/時点/対象を独立に観測した結果 | 人・Ownerの業務受理、Source変更の原子性 |
| Owner-Accepted Meaning | 既存正本で受理された業務意味と、その適用範囲 | 現在の実DB/配布Build/画面状態の保証 |

公開・配布の三層と、この四レーンを単一の「validated」「green」「ready」に畳み込まない。Diagramには各線・図形の出典、対象scope、revision、選択map、適用時点、missing/unknown理由をInspectorから到達できるようにし、図面自体には必要最小限の区別を置く。

図面内の「Dataを開く」「Actionを表示する」はStudio read-only navigationであり、業務記録の更新ではない。**UI locatorの一意性と意味参照の一意性は独立**。StudioのviewIDが一意でもQualified Refが曖昧ならInspect/Source確定を拒否し、逆に意味参照が一意でも現行画面のlocatorが曖昧なら外部UI actを拒否する。

[#2391](https://github.com/hat47x/tei-platform/issues/2391)は2026-10-31現在、通常WebのA/B/C（Production外部Runner、Test外部Runner、Test optional instrumentation）とWails内蔵Application MCPのW0を**実証計画**として追補している。これはStudioのmethod/rendererの必須依存ではなく、任意の外部UI制御の別Gateである。Test/Production同一Scenario、配布実物の物理除去、business effect readback、Owner判断はそれぞれ別の結果であり、A/B/C/W0はNOT RUN。

## 11. TEA-D0を実装可能な小課題へ切る（P0優先）

**前提の厳密化**: 現行[Graph Reference](../../reference/tei-studio-graph/README.md)は GraphNode.id,label,x,y、GraphEdge.id,source,target,label と node-only onSelect を提供。**Data内部のField行クリック/Port/Hyperedgeは未実装**。最初から「既存GraphでField slotがクリック可能」とは主張せず、Data一覧・Field一覧など既存Consumerの別選択入口を一つのqualified snapshotへ結ぶところから開始する。

| Step | 入力と最小変更 | Assertion | Owner / 依存 |
| --- | --- | --- | --- |
| G0：静的資格 | current World/Data/Relation、既存modelref、existing ER-crossviewのread-only出力 | scope/kind/id/版を固定。RelationKey未解決をunknownと保持。新しいSemantic Sourceなし | #1126 / #1120 |
| G1：Diagram Binding | 2 Scopeに同名Data、同じDataの2図面Occurrence、Field一覧行のview IDを同じConsumer Bindingへ | Graph・List・Fieldの選択時にscope付きKey一致。view ID再利用で別Scope Inspect 0 | #1124（Graphは既存Reference） |
| G2：描画拒否 | R10/G1のview ID K→Scope A、R11/G2用K→Scope Bのpending map、G2はmissing_endpoint | G1 DOMは残るがM2はpublishしない。M1はR11にはstale。Kクリック後Inspect/Source Locateとも0 | #1124 / 現行freshness設計 |
| G3：表示・不存在 | Scope対象DataとRelationの閉包一覧、一部Hidden、一件Unsupported、Source版不明 | Hiddenはpresent。Unsupported/版不明はunknown。absent主張0。後続差分はlayout-only | #1126 / #1124 |
| G4：Methodの意味境界 | exact Semantic Setに未採用のTEA意味規則候補、別Notation Providerを追加 | Notationだけの交換でAccepted意味・Semantic Lock不変。Meaningを変えるRuleは明示受理なしに適用しない | Semantic Extension既存契約 / #951 |
| G5：複合Relation Intent | M49静的source factsと独自受注fixture、Key/Field不足ケース | 複数Change candidateとImpactのみ、Source書込み0、受理済み扱い0 | #1126 / #267（次Gate） |

**最小出力物**: 1) Derived Projection/BindingのFixture、2) G1/G2のSpy回帰、3) closure manifestの正例・負例、4) read-only画面のData/Field/Relation選択、5) 既存Contractと衝突しない未対応の一覧。新たなプラグイン配布サービスやGeneral Canvasを先行要件にしない。

**停止・撤回**: (i) Graph/list/Fieldを共通Bindingへ通せず表示IDがSourceへ漏れる、(ii) closed-scope absenceを正しく拒否できない、(iii) 同一SourceでProvider追加だけでAccepted意味が変わる、(iv) derived Relationが業務Action CRUDやEffectとして表示される、(v) target UIの実操作とOwnerの受理が混同される場合は受入不可。大きな抽象が必要だと判明しても、まず既存#1124/#1126/#951の境界に戻して原因を記録する。

## 12. 今回の静的確認と未検証

2026-10-31後続のmain head f89344bfca438a101b7ba93d357911bad8d3b609で、[Graphの入力・更新](../../reference/tei-studio-graph/graph.ts)、[ER/物理Cross-view Projection](../../tools/reference-er-table-crossview/projection.go)、[Business DFD diagram context](../../tools/reference-business-dfd/diagram_context.go)、[modelref](../../internal/modelref/model.go)、[Studio freshness](STUDIO_CONSUMER_FRESHNESS_BOUNDARY_2026-10.md)、[Xead Cross-view](../../tools/reference-xead-cross-view/README.md)を静的照合。#1124/#1125/#1126/#2391の現行Issue本文も再確認。Code/Testの実行、Studioの統合操作、実X-TEA GUI/M50、外部UI制御、Provider効果、Owner acceptanceは未実施。新設のG0〜G5は**提案/NOT RUN**であり、既存Graph単体の過去の合格を新しいConsumer受入とみなさない。



## 13. 最初の実装結果と三つの鮮度の分離（2026-11-01）

他チャットの[#2352 複合図表・トポロジー設計](https://github.com/hat47x/tei-platform/pull/2352)を再読し、汎用の多項関係・図間対応・不在証拠をこのTEA文書へ再定義しないことを確認した。TEA Method側の独自課題は、方法論の関係種別（FAMILY/REFFER/SUBTYPE）を**既存意味契約へどう解釈するか**と、そのUI Consumerが古いID対応を誤使用しないことである。共通Projectionの標準schema固定は両Draftでまだ行わない。

### 実装候補（別Draft、限定的な成立）

[PR #2457](https://github.com/hat47x/tei-platform/pull/2457)で `reference/tei-studio-graph/qualified-selection.mjs` と小さい負例テストを独立に追加した。PR #2397は引き続き横断設計のみとし、同じコードを取り込まない。

| 鮮度の軸 | #2457が扱うもの | 未実装で他Ownerに残すもの |
| --- | --- | --- |
| 1. UI Display | `stage`した表示ID→Qualified Refを、同期Renderer更新成功後だけ`publish`する。更新失敗時は旧map | 実DOM/Workbenchの統合、Graph eventの非同期発火/再描画競合 |
| 2. Model/Projection | 呼出元が観測した`workspaceId/modelRevision`に紐付け、`observeSource`で変更・ABAを通知されたらmapを無効化 | Projection fingerprint、Semantic Setの現在性、複数資源の原子的Snapshot |
| 3. Source/Acceptance/Runtime | 何も保証しない。`select`の`resolved`は**観測済みSnapshot内の対応**を返すのみ | Sourceの独立readback、accepted/ledger/owner、Current Authority、Target readiness、Effect |

#### 初回Referenceの合否と不足する事実

- **限定的に確認**: GitHub上の`qualified-selection.mjs` blob `976b6192e2502e5be01d43f1a415ff271f2d5462`と手元再現ファイルのGit blobが一致。Node 22.16.0でsyntax checkとそのファイルへの局所assertion 7件がPASS。
- **未実行**: GitHub保存済みの`qualified-selection.test.mjs`の全ケース、既存Graphとの実DOM、Workbench、Studio統合、Source Acceptance後の遅延Inspect拒否、外部UI操作。branch checkout・repository-health-check・GUIの資格は未取得。
- **既存実装の動作事実**: `reference/tei-studio-workbench/main.ts`ではGraphの`onSelect(key)`から現在の`entries.find(subjectKey)`を参照し、`inspect`が再度Graphを更新する。新規Guardを追加しただけでは既存経路を変更せず、#1124のP0は完了しない。
- **次の最小統合Gate**: 同じWorkspaceでR10→R11、G1 DOMはK→Scope A、G2の候補K→Scope Bを仮置き、`missing_endpoint`でG2更新拒否。旧G1が画面上に残ってもInspector呼出し0、Source locator確定0を実Consumer Spyで確認する。R10がなお現在と独立に確認できる場合のG1正常選択は不必要に失効させない。

**安全性の注記**: `modelRevision`はCallerから供給された観測値であり、真の現在Sourceを発見する仕組みではない。通知を逃した変更、版の再利用/偽装、別Sourceの割込み、現在Authority未確認をGuardが検出する保証はない。Source/Decision ledger readback、現在性、Scope別の権限、開示条件はStudio Consumerと既存OwnerのGate。したがって`resolved`を`accepted/current/ready/authorized`のいずれにも自動変換しない。

### Registry・Plugin構成の横断判断

2026-10-31の分散Registry設計では、公開のContent Identity/Origin/検証Evidenceと、Consumerが実際に選択・利用できるかを分ける方針が強まった。本TEA設計も、Method Definitionを公開Registryから取得できた、あるいはNotation Providerをインストールできたという事実を`Semantic Extension`のAccepted Lockや`Studio Consumer`の現在Authorityに使わない。宣言・解決・観測の各版は、汎用のPlugin/Artifact/Evidence正本に逆参照する。追加のTEA専用Registry、Resolver、Build Lockは作らない。

今後の推奨順は (a) #2457の焦点試験を現行checkoutで確認、(b) #1124 Consumerへ最小結合してG1/G2負例を実行、(c) #1126の方法論固有Relation規則をSource-only fixtureと照合、(d) 図上Intent→Proposal（Source writesなし）を確認、の順。ここを超える意味規則の正式採用・本番Provider接続には別の承認/適合Gateが必要。


## 14. 三要素×分析層と帰納・演繹の証拠境界（2026-11-02）

2026-11-02に[三要素分析法の説明記事](https://atmarkit.itmedia.co.jp/fbiz/cbuild/complete/doa/01/02.html)と現在のTEI main `3784a4432a28502ca04057325689e409d14d9140`、Studio [Draft PR1 #2583](https://github.com/hat47x/tei-platform/pull/2583)、[Draft #2457](https://github.com/hat47x/tei-platform/pull/2457)を照合した。出典記事は「データ構成・機能構成・業務構成」という三つの観点と、論理要件／物理制約／実装を区別している。**それは歴史的な方法論の紹介であり、TEIの既存Semantic schemaやX-TEA M49の全挙動を検証した証拠ではない**。三要素を一つの図法に平坦化しないためのPressureとして使う。

| 各観点の独立軸 | 論理要件・設計意味 | 物理制約・実現の構成 | 実装・観測結果 |
| --- | --- | --- | --- |
| Data | 論理的なEntity/Key/FD/Relationと業務識別 | DB schema、型、制約、Provider mapping | 生成されたschema、Actual DBの観測 |
| Function | 動作の宣言、入力/出力・参照、機能境界 | Target Capability・Invocation/構成・操作binding | Function実行の観測、結果・Effect witness |
| Business | Manual/External Step、Actor、Trigger、Handoff、手順宣言 | 実施主体、UI・外部接続、実現上の制限 | 実際の作業記録・結果、Ownerによる別途受理 |

これは**一つのStandard Matrix schemaの新設を要求しない**。Data/Function/Businessの全セルに必ず実装値があるという主張でもない。既存Semantic / Mapping / Actual / Evidenceの各Ownerへtyped referenceで接続し、未対応セルを`unknown/not-observed`として残す。

### 帰納・演繹をMethod出力の権限と混同しない

他チャットのCSWの議論を設計Pressureとして、出力ごとに「どの材料から何が言えるか」を識別する。追加の`Authority` enumを設けず、既存`modelref.Evidence`（accepted / proposal / source-observation / derived）、qualified identity、Artifact/Rule fingerprintと表示用説明を組み合わせる。

- **Source観測**: 旧X-TEA/XML/図法上の宣言・名称・関係型。Source profileと版に固定。TEIの業務意味正本へ自動昇格しない。
- **演繹的導出**: 明示的な前提、Rule/version、適用Scope、導出経路から結果を再現できる場合。主張の範囲は前提と規則の範囲内に限る。Ruleに照らして正しいことは、元の業務命題が真であることと別。
- **帰納的候補**: 複数モデルのパターン、具体例、X-TEA観測、生成AIが提示した新しいRelation/Method Rule案。曖昧・矛盾・未知の反例を維持した`proposal`であり、MethodのinstallやNotation切替による暗黙の意味変更を禁止。
- **Owner受理・実観測**: 人の意味受理と実ProviderのEffect観測は導出・提案とは別のAxis。Ruleの型検証や試験成功をOwner-Accepted/current-ready/authorizedへ変換しない。

例えば、M49の静的コード上の`FAMILY/REFFER/SUBTYPE`分類が入力Key形状から追えるとしても、それは**旧分類アルゴリズムの導出**であり、TEIの論理Relationが業務上正しい、DBの実FKが存在する、Functionや業務Actionがその関係を実際に利用した、という四つの主張を導かない。

**反例の最小組**: (1) M49 Key形状の分類が一致しても業務IDは未受理、(2) 物理FKが観測されても論理FD・Business identityは推定不能、(3) 同じ意味Worldに別Notationを入れてもSemantic Lockは不変、(4) 演繹規則の入力Source/provenanceが欠落したら関係を`verified`ではなく`unresolved`、(5) AIによるパターン仮説からのSemantic Source writeは0。これらが新Core型なしに既存Evidenceへ紐付けられるかを評価する。

### Studio ConsumerのP0順序と表示の同一性Witness

本設計の前節G0〜G5はTEA固有の横断検証項目であり、**Studio P0 PRの実装順序ではない**。2026-11-02現在、先に[#1124 PR1 Draft #2583](https://github.com/hat47x/tei-platform/pull/2583)（Catalog ambiguity保持＋共通Inspect guard）、次にPR2（Graph/list/initial/commandの4入口結合）、次にPR3（Graph stale/表示epochとqualified binding）を通す。PR1はNode focused/typecheckが合格した一方、embedded bundleとWorkbench snapshot pinによるCI failureが残り、未統合。PR1/2/3を`#2457`単独の採用に読み替えない。

[#2457](https://github.com/hat47x/tei-platform/pull/2457)は**実Graph inputとselection bindingの同一性不足**を確認し、`publishGraph(token, graphView, graphInput)`を追加した。正規化したGraph Inputに**存在しないview IDへのbindingを拒否**し、非同期renderer・renderer例外では旧mapを失効する。表示上のunbound nodeはInspectを許さない。原案の`publish(token, render)`はGraph node集合を照合しない低水準入口であり、Graphの現在のbinding資格には使わない。

**限定検証**: GitHub blob `30f1b0133693c421cc2826028790ed90baf96344`をV8内で実行（Graph `normalizeGraph`はtest double）し、表示外ID、G2失敗、ABA、別Scope、非同期戻り値等10 assertionがPASS。**PR保存済みNode試験一式、実DOM、Graph/Workbench/4入口E2E、Source readback、Accepted/Effect、M50はNOT RUN**。この限定検証を本番GraphやSourceの保証へ昇格しない。既存Draft PR #2457の旧CI successful checksも**新headの資格として流用しない**。

将来のPR3で実証すべき同一性の式は、`PublishedBinding.viewIds ⊆ normalizedGraph.nodes[*].id`、`PublishedBinding.version = acceptedDisplaySnapshot.version`、`Source現在性・権限・Inspect eligibilityは別Gate`。一図面の境界で成立させてから、複合図表（複数Graph/List/Field）のsurface別snapshot/epochと図表間対応（#2352）を評価する。


## 15. Script-first三要素Methodの縦断契約とSelf-build Fit/Gap（2026-11-06）

**位置づけ**: 2026-11-06 main `36bda76271db4db72ddd41a8566d281f02764b8a`、[#1124](https://github.com/hat47x/tei-platform/issues/1124)の2026-11-04 Script-first / Everything is Data宣言と11-05追補、統合済み[#2397](https://github.com/hat47x/tei-platform/pull/2397)/[#2457](https://github.com/hat47x/tei-platform/pull/2457)/[#2583](https://github.com/hat47x/tei-platform/pull/2583)、[#2352](https://github.com/hat47x/tei-platform/pull/2352)、[Plugin self-build契約](PLUGIN_SELF_BUILD_PRODUCTION_CONTRACT_2026-10.md)、[DSL Language/Operator候補](DSL_OPERATOR_PLUGIN_COMPOSITION_DESIGN.md)を静的照合。従来の図面編集優先という説明は履歴とし、**Script/DSL/Markdown等のauthoritative Source → generic Property/Table/Tree → 三要素Diagram Projection** を初期経路にする。現行Studio UI E2E、方法論固有Plugin、セルフビルド実行はこの節によって成立しない。

### 15.1 Method Profileは三要素の新しい正本ではなく、Role付きの解釈束

Data / Function / Businessは既存の標準データを異なる**意味Role**で見る方法であり、全モデルへ独立した「TEA Entity/Function/Process」テーブルを必須にしない。Method Profileが明示的に束ねるのは (a) 受理済みSemantic Interfaceと正確なRule/Extension version、(b) 必要なRole/Relation/制約、(c) read-only Projection・Notation・診断、(d) Sourceに逆到達できると証明した編集Intentの範囲、(e) unresolved/Disclosureの解釈である。Generic Source inspectionが先に成立し、TEAの表記Providerなしでも未知Roleをread-onlyに示せることを優先する。

| 横断役割 | TEA Methodでしてよいこと | してはならないこと |
| --- | --- | --- |
| Data/ER | Key/FD/RelationのSource宣言と物理Mappingを別レーンで閲覧し、Rule不足を診断 | Sourceにない論理業務IDをPK/FKから補完し、外部DB観測を受理済み意味とする |
| Function | FunctionとData I/O候補を結び、Operation/Effectの独立Evidenceへnavigate | Function-wide CRUDや生成されたUIから特定Business Actionの実Effectを断定 |
| Business/DFD | Manual/External Step・Actor・Trigger・Handoffを、宣言と候補として描く | ManualをFunction実行、RoleをAuthorization Principal、図上順序を確定Runtime順序とする |
| Cross-view | qualified key、relation role、source/provenance、representation coverageを照合 | 表示ID、legacy ID、Source Locator、ハッシュ表記の一致だけでSubjectと現在Authorityを確定 |

Methodの**構造上の役割**と**業務意味規則**は別契約とする。異なるMethod ProfileのProjectionを同時に開いても、同じSemantic SetのAccepted meaningを表示Plugin登録順で変えない。公開Plugin名や`tei.std.*`の所属はContract/Provider/Artifact/Instance/受理証拠の代わりにならない。汎用の型付き関係・多項role・被覆・図間対応は[#2352](https://github.com/hat47x/tei-platform/pull/2352)の既存評価へ委譲する。

### 15.2 TEA-V0：同一合成Sourceによる最小の五ケース（受入仕様／NOT RUN）

合成受注モデルを一つだけ使う。ヘッダ/明細Dataと論理Relation、Functionが参照するData、受注確認のManual Step＋Function参照を**別Source role**で持ち、Data・Function・Business・Source一覧が同じqualified subjectへ到達できるようにする。Sourceの意味内容、Source位置、UI選択版、Disclosure、Artifact/Providerを別々に固定・変異させる。元のX-TEA Referenceの本文・旧IDを使用しない。

| Case | 固定した前提と変異 | 最小期待結果 | 担当／未実証 |
| --- | --- | --- | --- |
| **T01 Source移動（meaning同一）** | 同じSource byte・論理意味を別Pathへ移す。World/Meaning fingerprint同じ、Source IndexとTrace locator変更 | 既存[#2626](https://github.com/hat47x/tei-platform/pull/2626)の局所証拠を期待値にする。Data/Function/Businessのmeaning投影は再利用候補だが、**旧Source jump/span・引用Occurrenceはstale**。再読取後のみ新位置へnavigate。移動がpath-sensitiveなBuild入力ならbuild/closureは独立再評価 | #1124 / #950 / #2604。三要素UI未実証 |
| **T02 閲覧資格だけ失効** | Source byteとMeaning版は同じ、現行Grant/Policy/Role pinが失効 | 旧DOM/ARIA/一覧/診断/Tool JSONと新規後着応答の開示資格を別々に失効。低権限側から非公開Data/Function/Manual Stepの有無・件数を推測させない。別Scopeの許可済み表示は無関係なら維持 | #1124 / #2516。実Disclosure E2E未実証 |
| **T03 Graph非可逆編集** | Composite RelationKeyを一本の線として表示し、図上で線を消去・付替え | 原Sourceに複数Field/Key/Relation/Function参照が存在するなら逆写像を**一意と推定しない**。read-only表示＋Source navigateまたは明示的な複合Change Proposalまで。Source writes=0。可逆なField scalar editだけ別経路でtyped Preview→既存Shared Acceptanceへ | #1126 / #1124 / #267。複数Source atomic commit未証明 |
| **T04 DFDとETL/ELTの重なり** | 同じDataに対し人手Business Step、バッチTransform、File Relay/Delivery候補が併存 | それらを標準Operation/Artifact/Deliveryの**異なるRole・型付き関係**として表示。業務Process矢印からbatch実行順、file到達、replay/idempotency、外部Effectを推定しない。隠れた外部処理はopaque/unverified | #1125 / #1130 / #2391／パイプライン設計。連携実Provider未実証 |
| **T05 Method・Notationと実行の独立** | 同じAccepted SourceでTEA NotationまたはLayout Providerのみ差替え。別ケースでSemantic Rule/Provider/Targetを変更 | Style/LayoutのみならMeaning/Accepted fingerprint不変。Semantic Ruleは明示pinと適切な受理なしに変えない。Provider/Target変更は利用資格/Artifact/Effectを独立再評価。未解決ならDiagramの`ready`なし | #951 / #943 / #1124。Plugin Conformanceと実Consumer未実証 |

**検証方法**: 先に既存の`modelref`、ER cross-view、Business DFD、Xead read-only、Source Index、qualified selection、Shared Acceptanceの各Referenceで入力を再利用できるかを棚卸しする。次に一つの`Source reader→typed model→TEA Method Projection→generic Consumer`で`Input/Rule/Source Revision/Scope/Effective Policy/Expected/Actual`を同じCase IDへ帰属させる。試験PASSは個々の層で記録し、別PRの局所PASSの足し算でTEA-V0一巡を成立させない。S01 Source Acceptance、S02 realization readiness、Modeler選択、実Command/Effect、Owner受理は独立。

**認知負荷の反証**: 三視点で同一項目を探し、RelationKey変更の影響を説明する2タスクをGeneric Source一覧のみ・TEA複合図面で比較する。探索手数、誤Scope/誤Source jump、未知情報の説明漏れ、変更波及、必要な再読取数を測る。図が増えても情報の正本や探索量が増えるならMethod特化UIを縮減する。閾値・人手評価は未設定・未実施。

### 15.3 Self-build Fit/Gapを、TEA Methodの成果物で検査する（既存Self-build Ownerへの提案）

**Fit**: 現在の[Plugin self-build Reference](../../reference/plugin-self-build/README.md)は、Descriptor Source / Implementation Source / Build Plan / Artifact / Lock / Runtime invocationの分離、SB-1/2/3の異なる証拠資格を持つ。TEA Methodにも`Method Definition`（Semantic Interface/Rules/Projection/Notation参照の宣言）と`Method implementation`（Reader・Projection・Notation Provider）が区別できる見込みがある。既存のDSL Operator/Compositionを使い、Method専用Builderや第二Plugin Lockを先に作らない。

**優先Gap**: (G-SB1) Model/Method Sourceがauthoritativeであること、同じSource/Rule pinから同じ**意味投影**を再現できること、(G-SB2) derived Projection/Notation packageの実Build Plan・artifact closure・binaryとSourceからのtrace、(G-SB3) 生成したMethodを別Consumerが実load/inspectし、未知Roleではread-only・Scope不明では拒否すること、(G-SB4) Method自身のbuilder stage1/2収束や配布Providerの実行は、**既存Self-build SB-3と別証拠**として残すこと。特にT01のSource移動はmeaning no-opでもpath-sensitive builderならBuild Plan no-opでない。

**進行順**: Generic read-only TEA Projection＋T01/T03のSource-boundary負例を最初の高優先Gapとし、その後T02の開示/Consumer証拠、T04/T05の他領域比較を行う。Self-buildをTEA-V0の前提にせず、SB-1 package→SB-2 implementation→SB-3 builderのOwner既存受入へ順次接続する。「TEAでself-buildできる」ことを「TEI全体がself-hosted」「元Method意味が正しい」「実ProviderのEffectが成立」とは呼ばない。

### 15.4 限界・所有と採用前の停止条件

- **既存実装**: #2397/#2457/#2583/#2352はmain統合済み（2026-11-03）。[#2747](https://github.com/hat47x/tei-platform/pull/2747)はStudio Command参照を統合。#2626の実CLI Source移動は限定Evidenceであり、TEA画面のSource位置追従は未実装・未検証。過去節のDraft/CI記載は当時の履歴であり、**現在のPR状態は本節を優先**する。
- **継続中**: #1124 M0〜M4の同一実Consumer接続、#1125 Business意味/Role責務、#1126 Data/ER、#951 Provider/Method適合、#267 Shared Acceptance。[#2833](https://github.com/hat47x/tei-platform/pull/2833)は合成4レーンPresenterのDraftであり、実Policy/Owner/Effect受理ではない。
- **停止**: Methodの追加で既存Sourceが第二正本になる、表示Lineから実外部Effectが発生したことにする、意味のRuleがPluginの更新/ロード順で変わる、Scope/Disclosureが未解決なのに存在を漏らす、Script以外の図形を常に可逆編集と扱う、TEA package一致をself-hostingや業務Acceptanceへ昇格する場合。
- **検証状態**: この節は2026-11-06のGitHub現行main・既存Issue・設計・Reference説明の静的照合に基づく**設計/テスト仕様**である。TEA-V0の実装、Go/Node focused test、WebView/GUI、Private X-TEA原本、MethodのBuild/Load、現実のETL・File Relay、Self-buildは**NOT RUN**。採用・実装・Owner受理と区別する。


## 16. 三視点の開示閉包とRole解釈の局所資格（2026-11-06追補）

**照合根拠**: 2026-11-06 main `bf2e7b0ffe21747a308cd2b37328eb440535f696`、[#2620 Everything is Data](https://github.com/hat47x/tei-platform/issues/2620)のSDX-01〜15（Role bootstrap／必要依存閉包の候補）、[#2516](https://github.com/hat47x/tei-platform/issues/2516)の用途相対Construction Context・二世界Disclosure、[#1124](https://github.com/hat47x/tei-platform/issues/1124)のH0〜H4・M0〜M4、[#2604](https://github.com/hat47x/tei-platform/issues/2604)のArtifact Set実Reader境界、[ETLとSelf-build二ConsumerのDraft #2880](https://github.com/hat47x/tei-platform/pull/2880)を照合。各汎用OwnerのSchema／Policy／RuntimeをこのMethod設計に複製しない。

### 16.1 Data / Function / Businessの複合Projectionは「開示済みSlice」だけを表す

同じModelに属する図や表を合成すると、個別Nodeを隠しても、Relation線・端点数・配置・折り畳み・図間リンク・Coverage・診断・ARIA・Export・AI向けJSONから非公開要素の存在が漏れ得る。**三要素Methodが独自のPolicy Engineを持つのではなく、既存OwnerからConsumer用途・現在のGrant/Policy/as-ofに適合した公開可能Sliceと未公開部分への照合禁止条件を受け取る。** 未資格の全Source Graphに先にレイアウトを施し、後でラベルだけ隠す方式は非干渉の証拠にならない。

```text
actual Source / exact selected Role+Rule / qualified relations
           ├─ private interpretation + provenance (Owner side)
           └─ current Disclosure decision (existing Policy owner)
                 ↓ authorized, bounded claim slice
           TEA Data + Function + Business projection / relation table
                 ↓ public render + DOM/ARIA/count/diagnostic/export
```

**受入不変条件（候補）**: 同じ許可済み公開Inputと現在Policy・Consumer条件を固定した二世界 `W0=private item なし`、`W1=private item を追加` で、公開図面のNode/Edge一覧・ラベル・順序・座標・図間対応・分岐・件数・Coverage用の説明・Diagnostic・検索候補・ARIA・Tool/JSON payloadが、非公開差分だけを理由として変化しないこと。ただし、公開してよい独立情報まで一律に削る要件ではない。Private集合が空でも、Consumerに全Sourceの閉包を証明できなければ、未開示全体について `absent` を主張しない。存在を示唆する`withheld=1`や隠れた端点のGhost IDも無断公開しない。

完全性判定は**公開された有限のQuery/Scope/Relation kind**にのみ相対化し、`complete-for-public-scope`と`complete-for-all-source`を混同しない。private情報を含む完全性の証拠は内部に保持しても、外部Consumerへ現在の開示権限なく伝搬しない。Source・Role/Rule・Grantが変わったときは公開済みDOM/ARIA/Exportと遅延応答を**別々に失効**させる（既存#1124/#2516へ接続）。既存の許可済み別Scopeは必要依存が交差しない限り失効しない。

### 16.2 Roleの自己解釈は「表現成功」「意味解釈成功」「利用可能」を分離する

TEA Methodは同じ標準値・Relationに複数Roleを適用できるが、**carrierが同じこと**、**Roleラベルの名前が一致すること**、**図面Glyphが描けたこと**を、Accepted meaningや現在の利用資格へ短絡しない。

| 層 | Methodが確認すること | 未成立時 |
| --- | --- | --- |
| Plain Source | 実Readerが受け取ったbytes、Source span、Scope、Source Index、未読込Required Sourceの有無 | Generic read-only Sourceへ退避、Missingを空集合と呼ばない |
| Selected Interpretation | Role/Operator/Language/必要Semantic Extensionのexact Pinと推移eager依存、Required Shape、Relation端点とrole、Conflict/循環 | 当該Roleのsemantic projectionを`unresolved`、別Roleの利用資格を無条件に停止しない |
| Accepted Meaning | qualified subjectと意味差分、受理範囲/Decisionを既存Semantic ownerから確認 | proposal/derived/observedと区別し、Acceptedに昇格しない |
| Current Consumer Use | Projection版、出典位置、現在Disclosure・Reader世代・表示epoch・用途に必要な閉包を確認 | 当該公開/Source jump/Inspectを拒否し、旧Source位置へ自動ジャンプしない |
| Actual Effect / Delivery | 対象Operation/Attempt/Provider/Target/独立Readや受領byteを他OwnerのEvidenceで接続 | 図の矢印・CommandのCommitted・ManifestだけではEffect/Deliveryを主張しない |

Role解釈成功後も、現在のTarget ProviderやBusiness Actorの権限は得られない。Role/Operatorの**同名・同版でも実選択した依存bytesだけ**変化したら再資格化し、無関係Providerが追加されても有効なexact selectionは暗黙に切替えない（#2620 SDX-07）。同じSource byteとMeaning fingerprintでもSource位置だけ移動したらSource navigationの資格は失効する（本書T01）。Role不明時のSource読取の存続と、Role意味のunknownを同時に保持する。

### 16.3 TEA-V0の追加反例：複合Viewでの非干渉とRole Pin（**NOT RUN**）

| Case | 操作・相互比較 | 判定（実証前） | Owner |
| --- | --- | --- | --- |
| **T06 三視点二世界** | Public注文Data/Function/Businessは固定。W1にだけPrivate DataとそれへのPrivate Relationを追加して再投影 | Public Node/Edge、Graph geometry、cross-view table、ARIA/件数/診断/Tool JSONにPrivate差分がなく、全Source不在の断定なし。Disclosureの実施主体・観測Readerを固定できなければ`NOT_PROVEN` | #1124 + #2516 |
| **T07 複数Roleの選択** | 同じSource carrierにData Roleはaccepted、Business Roleはmissing/ambiguous、別Scopeに同名Function | Data generic inspectは可、Business意味は`unknown`、別Scopeを先勝ちInspectしない。Role providerを暗黙fallbackしない | #2620 + #1124 + #1125 |
| **T08 同名Pinの実依存変更** | Source/Role ID・表示名・意味Subject IDを固定し、選択したOperatorの実bytesまたはeager依存を差替え。対照は未選択Providerだけ追加 | 前者では当該Interpretation/Projectionの再資格が必要。後者で既選択meaningを勝手に変更しない。Artifact/ProviderのConformance・現在Authorityは別 | #2620 + #951 + #2600 |
| **T09 公開成果物の版混合** | 三視点Markdown/JSON/DiagramをR1で生成後、R2で一部だけ更新し、実ConsumerがR1 ManifestとR2 memberを読む | digest/Manifest生成成功を同一世代の実読取と呼ばない。版不一致を拒否。既存#2604の出版/受領Evidenceを再利用 | #2604 + #1124 |

最初の独立Referenceに必要なのは**T06の二世界比較＋T07のunknown Role**である。両者を同じ公開合成注文fixtureのread-only入力で比べ、次に既存[#2586](https://github.com/hat47x/tei-platform/issues/2586)のM0 CLI→M1同一Workbench Consumerの受入へ渡す。T08は#2620のRole bootstrap、T09は#2604の公開/Read証拠を再利用し、TEA専用Resolver/Publisherの新設はしない。CLI→Workbenchの4入口/Graph stale、実Provider/Host Grant、実SHO/外部AI受領、Owner Meaning Acceptanceまでは、この文書の追記では一切成立しない。

**次の計測対象**: (a) Public世界を二つの秘密構成に変えても出力が等しいか、(b)未選択Roleのエラーが無関係なData読取を止めないか、(c)独立Source movementが旧定義jumpだけを失効させるか、(d)事後に実Source・Provider・Decisionを追加した場合のみClaimsを段階的に強められるか。前節の認知負荷比較は、この安全性を満たしたread-only Methodの比較で行う。仕様・標準Role型・Plugin ABI・評価値の採用は未決。


## 17. T06/T07合成Referenceの観測と次の実Consumer境界（2026-11-06）

[独立Draft PR #2972](https://github.com/hat47x/tei-platform/pull/2972)に、**既に公開資格がある有限Slice**を入力とする三視点のread-only Referenceを追加した。実装は`reference/tei-three-element-method/public-slice.mjs`。Data/Function/BusinessのRole別Occurrence・qualified ref、role付きRelation table、固定した公開入力からのみ求める座標、`allSourceCompleteness=not-proven`を導出する。これは**TEA Methodの入力契約へのPressure**であり、Policy Engine、Role Resolver、Source Reader、現行Studio Consumerでもなく、Product Schemaではない。

### 17.1 投影と開示の非可換性

公開版の図形だけを最後に消す方式は、非公開Element数がレイアウト、並び、匿名端点、関連表、診断へ伝搬し得るため、以下を一般には交換可能としない。

```text
Safe candidate:    Source -> existing Policy/Role qualification -> Public Slice -> TEA Projection -> Consumer
Unsafe candidate:  Source -> TEA Projection (global layout/count/relations) -> remove private labels
```

**本Referenceの狭い主張**: `publicSlice(W0) = publicSlice(W1)`を**前提条件として外部から与えた場合**、当Referenceの`Project`出力は一致する。これは現行PolicyがW0とW1から正しい同一Public Sliceを生成すること、実DOM/ARIA/Tool/Exportの内容が等しいこと、Timing/Cache/通信の非干渉を保証しない。後者は#2516/#1124で、実Source Readerと実Disclosure判断、実Consumer観測に束縛した二世界比較が必要。公開対象外の`count=1`やghost edge、データ保護対象の`withheld`理由を知らせない。

**局所性**: `Data Role=accepted`と公開資格が与えられ、`Business Role=unknown/ambiguous`の場合、明示的な必要依存がない限りDataのread-only閲覧は止めない。一方、RoleのSource実bytes/選択Pinが未確認ならAccepted MeaningやExecution Eligibilityには昇格しない。曖昧なRoleと同名別Scopeの先勝ちを区別する。

### 17.2 限定試験の実際の証拠

Node.js 22.16.0で`node --test /tmp/tei-tea-ref/public-slice.test.mjs`を実行して**8件PASS**。GitHubに保存した`public-slice.mjs`（blob `e7543bd049f341d43c23dbd9c726efc68291fa66`）、`public-slice.test.mjs`（`bde7015eceb87d58ad9ebb1028cd8ad2d93152c1`）、README（`01fbf70dae6485b3709df3448df9fec92a1c2b42`）は実行したローカルbyteのGit blob SHAと各一致を確認した。

合格範囲は T06: 事前公開Sliceの二世界出力同値、秘密の存在数を使ったレイアウトの負対照、未公開Endpoint拒否、T07: unknown/ambiguous Roleの局所保持、別Scope・重複key拒否、入力順序独立、raw Source path非出力。**実際の秘密SourceのDisclosure、実Role/Operator Provider、独立Source read、CLI→同一Workbench、実Graph/DOM/ARIA、現在Grant・失効、Source Acceptance、実Target/ETL/Build、GitHub CIやtrusted full checkout healthはNOT RUN**。局所試験をProduct Gate合格へ合算しない。

### 17.3 次の実証への最小引継ぎ

- **P0（#2516/#1124）**: 同じ合成受注SourceのW0（公開のみ）／W1（非公開要素＋非公開Relation）に対し、既存Policyが実際に公開したSliceのbytesと版・Consumer目的を取得。次に同一WorkbenchでData/Function/BusinessのGraph/Table/関連表、DOM/ARIA、件数、診断、Tool JSONを比較する。候補が出ていないことと非公開要素がないことは別のClaim。未受領なら`NOT_PROVEN`。
- **P0並行（#2620/#951）**: 公開RoleのSelection/Interpretation依存Pin・Source Readerの実bytesを固定し、Business側unknownをData read-onlyへ波及させない。ただし同一Required Closureの欠損なら、その局所Claimだけを失効する。新Resolverは作らない。
- **次段（#1124/#267）**: Source移動の旧Jump失効（T01）とComposite Relationの非可逆編集拒否（T03）を同じSource/Consumerへ結び、意味編集はtyped Proposal→既存Acceptanceへ限定する。
- **他Owner**: ETL/MFTの四粒度・実2-member受領は[Draft #2827](https://github.com/hat47x/tei-platform/pull/2827)、標準I/Oの二Consumerは[Draft #2880](https://github.com/hat47x/tei-platform/pull/2880)、Plugin最適化のContract/Effect同等性は[Draft #2933](https://github.com/hat47x/tei-platform/pull/2933)。三要素Method内に第二のPipeline/Build/Delivery保証を置かない。


### 17.4 事前公開Sliceの境界を厳格化し、元SourceのRole構文を別資格にする（2026-11-06）

[Draft #2972](https://github.com/hat47x/tei-platform/pull/2972)の初版T06は、`privateSubjects/privateEdges`の追加キーを入力に入れたまま黙って捨てていた。**これは「実際のPrivate WorldからPolicyが正しいPublic Sliceを作れる」ことの実証ではない。** 最新Referenceは、二世界の`private` stateをProjectorの**入力外**に置き、公開Sliceが外部で正規に選ばれたという前提を明示。同時に最小Fixtureのトップレベル/Subject/RoleStates/Relation/Endpointの**未宣言field・symbol・不適切prototype・getter**を定型`unqualified_public_slice`で拒否する。Projectorにsecret raw pathを渡して「出力にないから安全」とみなさず、入力資格外として停止する。Roleに`accepted`と書いてあっても、それ自体はOwnerのAccepted Meaningではない。

**新しい独立実測**: Node 22.16.0で、GitHub最新Module blob `9b46cf13f3683607a384ea629eebba99a59ce872`、test blob `bc02920d11005ca53c6b928802e022bfead07f72`に一致するローカルファイルを用い、`node --test public-slice.test.mjs`で**9/9 PASS**。T06は二世界の公開入力同値とPrivate stateの入力混入拒否、T07はRole/Scope独立、追加負例はraw Source metadata・getterを評価せず拒否すること。前節の初版8/8はその時点の履歴であり、現在の検証対象はこの9件。**実Policyにおける二世界の非干渉、実Source→公開Sliceの生成、Web/Studio、実ProviderはなおNOT RUN**。旧HEADのCI成功は新HEADの資格ではない。

さらに[#2620](https://github.com/hat47x/tei-platform/issues/2620)では、同一Attributeの重複Role宣言が既存Parserで後値上書きとなり得る反例を固定。修正候補[Draft #2898](https://github.com/hat47x/tei-platform/pull/2898)は重複宣言のSource位置付き拒否とread-only Role値anchorを扱うが、未統合。**解析済みの`roleStates` Mapから、元Scriptで上書きされたRoleの二重指定は復元不能**。最小受入を、①実Source Readerが実際に読んだSourceと未読込Required Set（[#2823](https://github.com/hat47x/tei-platform/issues/2823)）→②ParserのRole重複・Source span診断（#2898）→③選択Role/Operator・推移Pin・必要閉包（[#2860](https://github.com/hat47x/tei-platform/pull/2860)）→④Policy Ownerによる公開Slice生成→⑤TEA投影→⑥同一WorkbenchのConsumer実受領、と分割する。個々のGo/Node Test PASSを結合縦断と呼ばず、第二Parser/Role Resolver/Policy/Source Writerをここで追加しない。

**残る反証の最小二件**: (A)同じPublic Sliceなのに元Source内の秘密でProducerが公開Node座標・件数を変えてしまう（Policy→Projection間の実結合負例）、(B)同じAttributeに二重RoleがあるのにParserが片方を落とし、TEAが意味確定済みと誤表示（Source→Role Pin負例）。両方ともTEAだけの純粋Projection単体では検出不能であり、Ownerの実Reader/Parser/Policyを介した試験が必要。


## 18. 意味的Traitの三視点ProjectionとIntent／Observation／Assessmentの照合（2026-11-06）

**位置づけ／一次Owner**: [Draft #3027：Semantic Trait Claims / Conformance](https://github.com/hat47x/tei-platform/pull/3027)を、Trait Definition、Intent、Observation、Assessmentとその正負判別の**唯一の横断設計Owner**として参照する。本書が追加するのはTEAのData/Function/Business複合表示で**同じClaimを別Authorityへ誤昇格させない**最小Consumer契約。[#2495](https://github.com/hat47x/tei-platform/issues/2495)の構造Facet、[#1126](https://github.com/hat47x/tei-platform/issues/1126)のCandidateKey/FD、[#2620](https://github.com/hat47x/tei-platform/issues/2620)のRole自己解釈、[#2516](https://github.com/hat47x/tei-platform/issues/2516)の実Disclosureは別Owner。2026-11-06 main `f275843b2bd35117f3bbb8d6090a59678d3c33ed`を基線とした静的照合。本節自体は**未採用の設計・試験仕様**である。

### 18.1 「どの視点で見たか」はTraitの根拠資格を増やさない

同じ意味Subjectに対する`Trait Definition`・人間のIntent・観測されたImplementation・比較Assessmentを、TEA Methodが各視点へ**read-onlyで投影**する。型付きRefは表示OccurrenceのIDとは別。既存`modelref`/Source Indexのqualified Keyと実Source span/revision、選択したRole/Rule/Use-site、現在許可済み公開Sliceを参照する。現行`public-slice.mjs`はこのOverlayを持たず、`roleStates.accepted`を業務Traitの正式受理として扱わない。

| TEA視点 | 表示候補 | 自動的に断定してはいけないこと |
| --- | --- | --- |
| **Data** | `status`の宣言値域、業務分類の母集団U、候補キー、FD、多重度、NULL、強属性の独立候補Facet | Java Enum/DB PK/単なる重複ゼロだけで業務値域の網羅性・Business Identity・「強属性」成立とする |
| **Function** | 値の導出式、Validator、Use-site別の強制点、Input/Outputと例外、Actual Provider観測 | Validatorが一つあれば全Writer経路で強制される、観測済みFunctionが業務意味を受理済みとする |
| **Business** | 状態の排他・網羅の業務意図、許可された遷移、例外・手作業・有効期間、担当者による未決判断 | Workflowの矢印や`default`枝だけで業務例外が漏れなく処理される、Roleが認可権限になる |
| **共通Inspection** | `Intent / Observed Fact / Assessment / Unverified Obligation`を独立表示。各項にSubject/Scope、Source/Artifact/Rule、as-of、確認手段、開示資格 | 同名の物理FieldとSemantic Subjectを先勝ち結合、過去PASS/Providerの存在を現在保証に昇格 |

**図表描画契約**: Annotationや線の表示には、少なくとも`(Qualified Subject, Trait predicate, claim-scope/U, exact semantic revision, evidence source/observation revision, Role/Rule pin, use-site, consumer/policy as-of)`の対応可能な範囲を要求する。情報が不足すればその部分の`assessment=unverified`を残し、矛盾を隠すために別Scopeの観測や現在未確認のSQL/ETL結果を流用しない。これは**現行の正式schema名ではない概念的な照合キー**である。

### 18.2 Closed Domain・MECE・Strong Attributeを一つの印へ畳み込まない

- `Closed Domain`（ある適用条件下で許容値域を固定）と`Mutual Exclusion`（分類述語が重ならない）、`Collective Exhaustiveness`（指定母集団が分類の和集合に含まれる）は独立。Javaの列挙型、3つのタグ、SQL `NOT NULL`の存在はそれぞれ全体MECEの保証ではない。NULL/欠損/Unknown/非公開/有効時点/別Writerも別軸。
- 画面表示は`declared-intent`、`observed-constraint`、`counterexample-observed`、`coverage-not-proven`等の**説明**に留める。これらを新製品の閉じたStatus Enumへ固定しない。制約の**未実装**は、実データに**違反が発生した**ことと異なる。
- 「強属性」は原典の正確な定義／適用条件が未照合。Change Identity、Immutable、Candidate Key/FD、Ownership、Authoritative/Derivedを独立した質問のまま保持し、`strong-attribute`を`tei.std` Roleに登録・推定しない。出典の版と定義が特定された時点で[#3027](https://github.com/hat47x/tei-platform/pull/3027)のOwnerへ差分を返す。
- **相互参照だけでなく反例への到達**を要求する。たとえばBusinessの`Order Status`をクリックした際に、Dataの状態項目とFunctional Validatorを飛び越えて「適合」とするのではなく、制約が不在のDB Writerや未観測のFile Relayを`unverified`として示す（表示を許可された範囲のみ）。完全なWriter inventory自体が未取得なら`no gap observed`でも`all writers enforced`ではない。

### 18.3 三視点の最小Case（TEA-T10〜T13／全件NOT RUN）

同一の合成受注Source`sales.order.status`を固定し、業務意図（まだOwner受理前）：`[NEW, ACTIVE, CLOSED]`、Java Enum観測：3値、DB観測：`VARCHAR NOT NULL`・値域`CHECK`なし、File Writer：未読込・不明、とする。DBに不正行が存在する、実DB実装がすべて違反する、といった結論をこのFixtureから出さない。

| Case | 変異・表示経路 | 合格に必要な区別 | 根拠Owner |
| --- | --- | --- | --- |
| **T10 共通Claimの三視点照合** | Businessで`status`のclosed-domain Intentを示す→DataでEnum/DDLの制約を表示→FunctionでParser/Validator・書込み入口を確認 | `Intent=proposal`、`Java enum=static fact`、`DB value-check enforcement=not observed`、`actual violating rows=not measured`、`business MECE=not proven`の五つを同じSubject/Scope/Revisionから提示。別書込み経路を勝手に閉じない | #3027 TR-01〜03、#1124、#1126 |
| **T11 強属性・同一性誤昇格** | 代理PK・候補キー・不変に見えるFieldを相互参照する | 物理PK・1サンプルで重複ゼロ・更新が未観測、の三つから`Business Identity`や「強属性」を導出しない。原典定義不明はunresolved | #3027 TR-04〜05、#1126 |
| **T12 Source/Artifact版の食い違い** | 受理済み意味と表示IDは同一に見えるが、Java EnumはA1、DB schemaはB2、Method RuleはR3、あるいはSource移動で旧jump | 証拠のjoinが成立する条件を一つずつ説明。古い事実を履歴として示せても、現在の適合・現在Source jumpは拒否。Scope/Artifact/Target/as-ofをViewで省略しない | #3027 TR-09、既存T01/#2620/#1124 |
| **T13 非公開Traitの存在漏れ** | W0にPrivate Constraintなし、W1にPrivate Trait/違反候補を追加。両者の許可済み公開情報は同一 | Public Node/線/Badge以外の診断・Overlay件数・検索候補・ARIA/Tool JSONも非公開差分を漏らさない。既存T06実二世界試験へ付属。公開資格がないなら診断の存在ごと保留 | #3027 TR-10、既存T06/#2516 |

**初回の最小縦断**: T10の`sales.order.status`をSemantic Data/Business State/Function Validatorの三視点で**同一Source、同一Attempt、同一公開資格**へjoinするread-only fixtureを検討する。T11〜13は既存#1126・#2516・#2620と実験Ownerを合わせて順次実証。初回の`T01/T03`（Source移動・不可逆編集）、`T06/T07`（公開Slice・Role不明）を置き換えない。Trait Overlayをsourceへ書き戻す新Writer、全Trait自動検出器、別のSource Registry/Policy/Reasoner、単独Studio Editorは作らない。

**現状**: [Draft #3027](https://github.com/hat47x/tei-platform/pull/3027)のTR判別票と本節を静的照合した設計成果であり、上記T10〜13の実Fixture、Source Index・DB/Java観測Reader、実Workbench/Policy、Go/Node、独立Expected、Owner受理・実Effectの新しい実行証拠は**NOT RUN/NOT PROVEN**。現行[#2972](https://github.com/hat47x/tei-platform/pull/2972)の9件局所PASSは、Trait Overlayや実Source制約の適合を証明しない。

## 19. TEI標準のモデル全容マップ・イベント／Lifecycle・Patternの可視化契約（2026-11-08）

**製品上の基準要件（利用者の明示）**：TEIが扱う各ドメインの**構造、責務、イベント群、状態・遷移、操作・効果、Function、Pattern、Provider、外部連携**を、TEI自身が導出する**相互参照可能な図表の体系**として示す。利用者が当該ドメインの予備知識を持たなくても、全体像から個別要素の目的・影響・拡張箇所を辿れることを、特定の組織ModelやStudioの追加機能でなく**TEI標準機能の成立条件**とする。図を単一画像に詰め込むことは「完全」ではない。俯瞰・階層化・表・詳細・出典・未確定箇所を**同じモデル集合から**導出し、どの図からも行き来できることが必要である。

これは標準**要求**であり、現行TEIに完成したGenerator、標準Model Map API、全ドメイン対応が実装済みとの宣言ではない。本節を新しいCanonical World／専用Graph Editor／第二モデル正本として採用せず、既存#1124（Modeler・共通Diagram Projection）、#1125（Business Flow）、#1126（Data Graph）、#1127（Function Pattern・Operation）、#2497（表現Projectionの忠実性）、#2506/#2507（Context/Pattern）、#2619/#2620（Everything is Data／Role）の実装・検証Ownerへ受入条件を渡す。

### 19.1 「全容」の対象集合と、初心者が辿る主経路

対象Worldを`W`、対象ドメイン／Function／Source集合を`S`、採用した意味解釈・Projection/Generatorの版を`G`、閲覧Consumerと公開Scopeを`C`とする。**全容の適格性は`(W,S,G,C)`相対**である。選択可能なPlugin・外部Provider・Sourceの存在を「TEIが知っているすべて」と誤認しない。必要定義の集合を確認できない場合は**全容未確認**を表示し、有限な取得済み範囲だけを`qualified partial`として説明する。Hidden／Not Loaded／Not Found／Permission Denied／Unsupported／Conflicted／Staleは「モデルにない」という表示へ丸めない。

標準の**ナビゲーション根**は「このモデルは何を扱い、誰／何が関与し、どのイベントが何を起こし、状態がどう変わり、利用者はどこを拡張できるか」とする。専用用語を知らない利用者向けに、**概要→業務の流れ→関係するデータ／役割→イベント・状態変化→Function/Pattern→実行と出典→編集・カスタム候補**の道筋を標準で提供する。どの図面・表の要素にも、平易な短い説明、型付きQualified Ref、受理意味とSourceへのジャンプ、影響先、適用Scope、更新版と証拠資格を辿る操作を用意する。意味説明が原典に無ければAIがもっともらしい説明を確定するのでなく`description-unverified`を表示し、Ownerが説明を受理するまで候補に留める。

### 19.2 固定の図表体系（異なる意味を一枚に潰さない）

| 標準図表／入口 | 必須の可視化対象（存在を確認できた範囲） | 明示すべき混同・欠落 |
| --- | --- | --- |
| **全体マップ / Navigation Catalog** | ドメイン、サブシステム、Function、Pattern、Role、Data、Event Group、Lifecycle、外部Actor、Plugin/Extension、Sourceへのリンクと集約数 | 不明・非公開・未取得を数と理由で示す。表示1枚＝全件検証ではない |
| **Data / Organization Topology** | Entity、属性、Relation occurrence、Position/Membership/Occupancy等の異なる関係、多重度・時点・粒度、物理対応 | 同一Personの兼務、JOIN増幅、同label別ID、FKと受理Relationを区別 |
| **Business/Event Flow** | 入口Trigger（人／Timer／外部Event／API／VUI）、Event group／correlation、Producer、Consumer、手作業、Branch、Handoff、拒否・再試行・終端、関連Function | Event発生・受信・適用・完了を別に描く。外部未受領を成功経路で隠さない |
| **Lifecycle / State Transition** | State identity、from/to、遷移Trigger、Guard、Owner Action、Effect/Receipt、観測・補償・失敗、許されない／未定義遷移 | 物理状態コードとMeaningを混ぜず、`allowed`と`authorized`と`committed`を別記 |
| **Function / Pattern Composition** | 既存`browse/record/reference/command/header-detail`等のFunction内での役割・入出力・Context/Result Slice・Data/Query・Hook・Plugin実装の束縛 | Pattern単品の説明でなく、全体のどのEvent、State、Data、Effectを扱うかを辿れる |
| **Cross-reference Matrix** | Data×FunctionのCRUD・Event×Transition・Transition×Action/Validation・Function×Pattern・Pattern×Representation/Target・Requirement×Evidence | 宣言／導出／実観測／未検証の各セルを別資格で示す。欠落セルを実行なしと即断しない |
| **Execution & Evidence Trace** | Logic／Query／Provider／Target／Consumer、Domain time／known/observed time、Scope／Source revision、実Read／Effect／Receipt | Diagramから実行成功や現在Grantを推定しない。実行・生成資格のない辺は`unverified` |

図表は共通の**意味付きノード／関係／イベント／時点／作用／参照**のProjectionであり、特定表現を唯一の図法として固定しない。**同じ要素の異図表示は同じQualified IDへ収束**する。一方でBusiness eventと状態、Operation intentと副作用、データ関係と表示上の接続は、見た目が同じ線でも意味が異なるので必ず種別を持つ。Event Groupを全ドメインで必ず同じ物理テーブルにすることは要求しない。

### 19.3 標準固定領域とカスタム領域を同一図表で見分ける

**「カスタム可能か」を図から隠さない**。図中の各要素・Edge/Hookに、(a)由来＝標準導出／案件宣言／Plugin実装／観測結果、(b)操作区分＝固定／設定可／置換可／拡張可／閲覧のみ／未確定、(c)現在の意味版・Source・権限／Provider資格、(d)カスタム点の明示契約を表示する。視覚上の色や線種だけに依存させず、同等の表／CLI／SHO説明にもテキスト属性で出す。

| 区分 | 固定するもの／許容するもの | 代表例 |
| --- | --- | --- |
| **Standard invariant** | ID/Scope/型・Meaning受理・副作用境界・順序制約・出典保存・検証前後条件は、表示や案件Hookで黙って削除・迂回できない | 状態遷移のCommit時再照合、権限境界、Source版失効、Eventの相関Identity |
| **Standard configurable** | 許された値域の設定／表示／有効化、版・選択を宣言で変更できる | 表示階層、図法、選択可能なLayout、既定Label、利用する標準Pattern |
| **Extension slot / Hook** | Trait・Contractと型付き入力／出力、適用位置、実行前後条件、失敗時動作、優先順位を宣言して差し込み、合成後の**有効フロー**を再描画する | Event handler、状態遷移Guard追加、FunctionのData取得／検証／後続Action、Target Generator差替 |
| **Project-owned replacement** | 明示的に置換可能と認められた節だけ、互換性と意味保存・失効影響を検収し、旧節と置換節のDiffを提示する | 顧客固有Approvalフロー、業務上の追加State/Role、VUI専用対話フロー |
| **Observation-only / Unqualified** | Providerのread-only実観測、権限不明、生成未対応、Source未解決の辺は編集許可や確定意味にしない | 外部Eventの観測、実DB物理FK、History欠落・未接続Consumer |

**フックポイントの原則**：標準処理にカスタムが差し込まれた場合、標準フローとカスタム差分の**両方**、そして合成後の**実効フロー**を図で見られるようにする。標準図にはない隠れたHook、未資格のPlugin置換、実行しない死んだHookを「有効」と表記してはならない。TEI生成ロジックもTEI自身がモデル化した定義を参照してよいが、データの変換・コード生成・Runtime束縛の版は別資格で保持する。実行TargetごとにHook可否が異なるならTargetごとの差を明示し、非対応Targetで標準動作を黙って変えない。

VUIのような異質媒体は「音声用の別意味正本」を作らず、Event受信／Flow制御／状態・Contextの操作として標準マップに接続する。音声再生・確認・対話順序の媒体差分をカスタムRepresentationへ落とし、業務State Commitを音声UI状態で代用しない。同様にWeb/Desktop/API/CLI/MCPは**同じ意味モデル**の異なる入口・出口として辿れることを要求する。

### 19.4 標準ガイド・ナビゲーションとカスタム編集の体験

初期表示は**何を扱うシステムか／主要な主体・イベント・ゴール／標準と独自の割合／未解決件数**の概要。利用者が`CaseRecord`を選択すると「何の案件か→誰が扱うか→どのイベントで担当が変わるか→どの状態遷移が可能か→どのFunction/Patternが閲覧・変更するか→どのRule/Provider/Sourceで成立するか」へ型付きリンクで段階的に掘れる。導入利用者には一般語の説明・用語集・代表Scenarioを提示し、専門家には原DSL・Source Position・Rule・Diff・Evidenceへ即座に到達させる。ナビゲーションの導出はMethod/Modelが責任を持ち、Studio固有の手動リンク集を標準要件の代わりにしない。

カスタム時は **①全体で対象の位置を特定 → ②変更可能領域・影響・不適格の理由を表示 → ③標準との差分と合成後フローをPreview → ④Source/Rule/Targetの版と型・Guard・副作用・Source閉包を検査 → ⑤既存Shared Acceptanceで受理 → ⑥再生成・差分再検収** の経路を共通にする。Layout変更だけなら意味受理を再要求しない。Hookの適用順やAction/State/ProviderのMeaningを変えたとき、見た目の変更として再検証を省かない。AIも同じ入力・出力／検証・権限を使い、人間と別の秘密の編集経路を持たない。

### 19.5 標準受入ケースと「完全」の不成立条件

既存の`organization`/Case C1→D1、Membership/Occupancy二時点、標準Action/Query、Event/Handoff、Lifecycleのケースを再利用し、**新しい全体像Scenario正本を作らない**。初回はread-onlyで同一Source/Scope/Consumerの次の負例を含む受入票を#1124/#2514/#1180へ接続する。

| 受入ID | 期待する機械的区別／操作 | 失格とする実装 |
| --- | --- | --- |
| **MAP-01** | 全体マップの全てのFunction/PatternがBusiness Event・Data・Stateの少なくとも一つの**説明付き型付き参照**に位置づく。非該当/未束縛はgapへ列挙 | Patternが検索結果の孤立項目として並ぶだけ |
| **MAP-02** | Event→Transition→Action→Effect/Observation→Handoffの正負・失敗・補償を辿れる。非同期遅延・取消・未受信を別ノード/状態で示す | 成功一本道の描画、手作業／外部工程の自動完了 |
| **MAP-03** | State×Eventの禁止／未定義遷移とGuard、依存Rule、現在Grant未確認をMatrix／Graphで区別 | 表示上の矢印を実行許可やCommit成功と表示 |
| **MAP-04** | 標準固定節／設定可／明示Extension Slot／案件置換／観測限定の区別が図・表・CLIに同義で表れる | 固定処理をUIだけで上書き、暗黙のHookを非表示 |
| **MAP-05** | Plugin追加・変更・拒否で標準フロー、Extension差分、実効フローの三者を比較し、Target/Generator/Source版の変化を追跡 | Generator変更をLayout-only扱い、未束縛Pluginを有効表示 |
| **MAP-06** | 初学者が概要→イベント→状態→Pattern→Data/Sourceへジャンプでき、どの段階でも「何・なぜ・どこで変える」が説明される | 予備知識がなければPattern名しか見えない、説明が出典なし推測 |
| **MAP-07** | Scopeを固定して対象集合／描画集合／Unknown/Hidden/Unsupported/Incomplete集合と被覆率・理由を提示する | 未取得を不存在と表示、部分グラフを「完全な全容」と表示 |
| **MAP-08** | 同じModelをGraph、表、CLI、SHO/Studio候補へ投影してもID・Role・Source rev・公開資格が整合する | Graphだけが別SSOT、非公開属性・存在をFigure Labelから漏えい |
| **MAP-09** | 図上のカスタム変更はtyped Intent→Diff/Impact→Shared Acceptance→再読取へ。標準不変条件と現在Authorityは変更不可 | 見た目の線やHookボタンから無条件のDB/業務書込 |
| **MAP-10** | 必須図表やLink/Scenarioに未対応・欠落があればModel Map自身が**不足の場所と原因**をガイドとして表示する | 欠落を黙殺して「完成」と認定 |

**初期検証順**：M0＝同じSourceのNode/Relation/Event/State/Pattern/Hook catalogとUnknown CountをCLI/read-onlyで導出 → M1＝組織の代表Business EventとState/Patternをつないだ2階層Graph＋横断Matrix（未解決を明示） → M2＝Studio/SHOの表示一致・ナビゲーション／初学者向け説明・利用者の理解課題（非AIの人的評価ができない場合はAI盲検/客観課題で代替し、その限界を明記） → M3＝標準/Custom差分・実効フロー・Typed Change Intent → M4＝実Provider/Host/Target生成物に対する欠落/適用成績の独立確認。**M0が未成立ならM1の美しい図を製品の全容保証と呼ばない**。試験票・Owner・Source集合と検証結果は既存Issue/CIを使い、本節に第二台帳を作らない。

**対応する実装Owner**：モデル全体のDiagram/Navigate #1124、Business Flow/Event/Handoff #1125、Data/Org Topology #1126、Function/Pattern #1127/#2507、Lifecycle/Guard #2620/#1174、Projection意味保存 #2497、完全性/Source #1706/#2516、独立Expected #2514/#1180、組織最小例は[Draft #3243](https://github.com/hat47x/tei-platform/pull/3243)。現行の限定Reference試験や[Draft #3500](https://github.com/hat47x/tei-platform/pull/3500)のNative Query→Pattern Go試験はこの10件の全体受入を証明していない。**MAP-01〜10は受入提案・新実行NOT RUN**。

### 19.6 コード生成時のカスタムフロー可視化：Target間で異なるカスタム実装を同じモデルから説明する

Code Generatorの入力は「TEIのAuthoringで宣言したPattern」だけではない。**選択された標準Plugin／個別Plugin、Traitの手動/生成Implementation、Provider、Target制約、適用済みHook順序**を解決した**実効Composition候補**が必要である。ただし宣言・選択・実Build／Load・実Host Invoke・Effectは別資格であり、宣言だけで「利用中」と表示しない。Model Mapには、実効Compositionの**有資格範囲**と未解決の部分を帰属させる。実バイナリに差し込まれた処理の証拠を、Source Diagramの見た目から創作しない。

| Model Mapに表示する処理段 | TEI標準の固定境界 | 明示的なカスタム点（存在すれば） | 一致が必要な実行・生成資格 |
| --- | --- | --- | --- |
| **Receive / Normalize** | Source/Request/Messageの出所、入力型・Epoch・Correlationの確定、秘密／非開示の境界 | 媒体ごとのDecoder／VUI Recognition／入力Validator | 実Adapter選択、Contract/Version、Trusted Origin、未取得の拒否 |
| **Resolve / Bind** | 意味ID／Relation/Pattern/Function Contextと実Providerの一致、Scope資格、曖昧拒否 | 許容範囲の選択規則、追加Binding、Target固有Adapter | 実Provider/Manifest/Source rev／選択Epoch、current Grantは独立 |
| **Validate / Decide / Transition** | Declared State/Guard、現在Validation/AuthorityとCommit条件の分離、禁止遷移の拒否 | 業務Rule／承認段階／遷移前後の追加業務判定 | 採用されたRole／Validation、Contractのpre/post、Action結果 |
| **Execute / Persist / Effect** | Operation ID、Concurrency/Period、Effect/Receipt、重複処理境界、失敗・部分副作用 | Provider実装／保存フック／承認済み追加Action | Transaction/Effectの実証、独立Read、Host Side-effect許可 |
| **Publish / Observe / Handoff** | ConsumerのDisclosure、Observed vs Accepted/Committed、送出・受領の適用集合 | Web/Desktop/CLI/MCP/VUIの表示/通知/後続Workflow | Consumer実受信、ACK対象／版、Result Slice、再試行／失効 |
| **Generate / Package / Deploy** | 受理WorldとTarget/Generator/Plugin versionの束縛、未対応Capability明示 | 置換可能なTemplate／生成Section／Manifest付き実装差込み | 生成Source/Artifact/Evidence、実Target接続・実行資格 |

この表の「固定」は**標準が検証しなければならない不変条件**を指し、「特定のMiddleware/認証方式/一つの実装」を全Targetで強制することではない。Targetにその段の機能がない、または外部Hostが担当するときは**外部実装の束縛／未確認Gap**を表示する。ユーザーが差し替えたFlowが標準Contractのpre/postと整合しない場合、Generatorは出力を「標準適合済み」として扱わず、Diagnostics→Preview差分→既存Acceptanceへ返す。

**完成Model Mapでの見せ方**：①標準参照Flow、②変更したHook/Stepと入力・出力・権限／版、③生成Targetごとの合成後Flow、④同じ対象のGraph/Matrix/Traceにおける影響集合、をリンクして提示する。CLIなど図形描画不可のConsumerには同型の階層表/ASCII graph/Typed JSONを用意する。これは特定の正規化JSON Schema採用ではなく、既存のRole付標準データとProjectionを使う標準受入要求である。

**追加負例**（既存MAP-05/08/10へ集約）：Generator Templateのみ変更でSemantic World fingerprint不変の場合もTarget生成資格を再評価する。Plugin宣言だけ追加・実選択なしなら合成後Flowに「非稼働」と示す。標準Guardの無いTargetへ配置する場合は警告を消して固定部分を描くのではなく`unsupported/unqualified`を明示する。Hookで前後のEvent順を入れ替えてContext/Effectを破壊する場合は適合拒否。**コードジェネレータのHook実装やTarget実証は本PRではNOT RUN**。
