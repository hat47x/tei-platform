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

### 19.7 Lifecycle・イベント・対話を型付きGraphとして合成し、生成器入力へつなぐ（他チャットのPlugin/Lifecycle/AI Chat横断、採用前の方式候補）

**【同一テストの重複回避】**：Lifecycle Graph/Patchの方式検討・Runtime/Target Loweringへの引渡しは[Draft #3507 のHook/Interaction Flow設計](https://github.com/hat47x/tei-platform/pull/3507)を、原子的TraitとGraph全経路保証のOwnerは**main統合済み[#3510](https://github.com/hat47x/tei-platform/pull/3510)**の[設計§15](TRAIT_ATOMIC_COMPOSITION_AND_FULFILLMENT_2026-11.md)を参照する。本§19.7が所有するのは**RQ-036の図表への表示、分かりやすいナビゲーション、欠落診断、標準／Patch／実効Graphの同一Ref・Revisionでの比較受入**のみ。MAP-11〜16はそこでの表示・資格の観点を追加するもので、#3510のTLG-01〜08や#3507のLGXケースと別のScenario SSOT・Runner／Validatorを増やす提案ではない。実装を採用・実行した証拠は本節から発行しない。

**設計上の補正**：§19.3／§19.6で「標準固定」と呼ぶのは、全ドメインに単一の固定**手続き順**を強いることではない。固定対象は、型付き参照／Guard/Authorityの迂回拒否、承認が必要なEffectのCommit資格、出典追跡、Effect/Receipt、失敗の伝播、版・Scopeの整合、要求する前提と事後保証などの**成立義務**である。これを守りながら、業務Graphの経路／Step／イベント群／State／Plugin Hook／Target適用を構成時に可変とする。互換なGraph Patchの導入は**意味変更**として扱う。Layout/Notationだけの変更とは区別する。

受理された同じWorldに、**Roleが付いた型付きData/Graph**をProjectionできることを最小要件とし、新しいWorkflow専用Canonical、固定Runtime Dispatcher、独立Graph Storeを導入しない。各Viewは以下の**論理属性**を持つが、この表を新たなGo公開構造体や正式DSL宣言として固定しない。

| Graph要素／束縛 | 必要な意味・由来・検証 |
| --- | --- |
| Node / Relation / Edge | Qualified identity、役割（Event/State/Guard/Action/Effect/Observation/Interaction/Join/Retry/Timeout/Handoff）、端点ID／方向／多重性／必要なら順序・条件、意味版とSource位置 |
| Port / Trait / Contract | Input/Output型・Result/NULL/partial、粒度、前提／保証、Owner、Hook挿入契約。Traitは原子的義務であり**Hook Slotという編集許可自体ではない** |
| Event Group / Correlation | 発生源、業務時点と観測時点、correlation/causation、重複・遅延・欠落、start/end/acceptance。発生・受信・適用・完了を混同しない |
| State / Transition | 有効な状態とfrom/to、禁止・不明、Guard、Owner Action、Commit再評価、明示取消／補償。認証の方式は固定せず、束縛した場合にだけ挿入する |
| Graph Extension / Patch | 変更対象のQualified node/edge、base revision、追加・削除・置換可能なslot、Plugin identity/Version、適用Scope/Target、期待する型・前提／保証、競合と順序。任意edgeの書換権限を暗黙付与しない |
| Effective Graph / Generation | Standard Graphと受理済みProject Graph、選択Plugin/Provider/Hookの実Bindingを合成。各Targetへの生成候補・未対応・生成器版・Artifact/Evidenceを別版で持つ |

**処理構成の順**：①Source/Role/Scopeと必要集合の閉包を列挙→②標準Graphに受理済み案件Graphと宣言されたPatchを合成→③同一node/edge・Priority・強制順序を巡る競合、型・Trait・pre/post・入出力粒度を診断→④Guard/Authorization／Effect Fenceを迂回する経路、到達不能Node、合流の条件不一致、無期限待機、部分成功の未処理、Cancel/TimeoutやRetryの欠落を検査→⑤Target別に実装可能な部分だけを生成し、Source/Generator/Plugin版を束縛→⑥実Targetが本当にそのFlowをロード／Invoke／Receive／Commitしたかは別の実証で確認。①〜⑤の一段でも不足があれば**生成済み図があること**をRuntime適合や完全性の証拠としない。

Graphを**描いていること**、Graphの**構文と型が通ること**、受理したMeaningとRequired Contractを**意味保存していること**、Targetへ**生成可能であること**、選択したPluginが**実呼出しされたこと**、実行Effectが**確定したこと**は全て別の合格列。Model Map／Pattern詳細／Generator Previewはこの区分を同一IDとRevisionで表示する。編集中にSourceが変更されたときは、初期Graph・Patch対象・Revisionを再解決して競合を拒否する。すでに走っている長期Executionを新Graphへ暗黙に移し替えず、実行継続／Migration／Resumeの資格は#1174/#2390等既存Runtime Ownerの別課題とする。

**異質な検証領域：生成AIのWebチャット**。Webチャットの`message`／`turn`／`run`／`streaming`／`tool-call`／`tool-result`／`approval`／`regenerate`／`branch`は、**Interaction FlowのState/Event候補**として同じGraph体系に位置づける。Tool応答の未受信・部分Streaming・中断・承認待ち・二重送出・Branch Resumeを明示し、会話画面上の「回答完了」を業務TransactionのCommitや現在Tool Grantへ自動昇格しない。UI Adapter/ProtocolはGraphの入出力であり、AG-UI等のEvent Transport、A2UI等の宣言的UI記述、MCP等のOperation Adapterは**可能な外部比較対象**にとどめ、いずれかの採用や実対応をこの要求から強制しない。AIがGraphを解釈・改善提案しても、Ownerの意味受理やHostの実行権限を代行しない。

**独立反例（MAP-11〜16、全件新規試験NOT RUN）**：

| ケース | 独立に比較する期待 | 実装を不合格とする条件 |
| --- | --- | --- |
| MAP-11 | 標準FlowのGuardを保持しつつ、許可済みApproval Stepを途中へ加えた合成後GraphとTarget別Diff | 形状が異なるだけで一律拒否、またはGuardを迂回して成功 |
| MAP-12 | 同じExtension Slotを二Pluginが異なるOutput契約で同時置換した際のConflictと未選択表示 | 先着Pluginを黙って採用、見た目だけ競合解消 |
| MAP-13 | State Graphの未接続／到達不能／合流Guard矛盾／Timeout未処理を独立Diagnosticsにする | 正規化だけ成功して全Graphが有効と表示 |
| MAP-14 | 配置・Notationのみ変更ならMeaning/Generator inputは不変、型付きEdge/Hook変更は生成・検証対象 | 表示変更で意味変更を起こす、意味変更なのにLayout更新として無検証 |
| MAP-15 | Graph Revision変更後も既存Running Attemptを自動切替せず、別案／移行資格／停止を明示 | 新しいGraphができたので実行中も暗黙に切替 |
| MAP-16 | AI chatのStreaming/Tool/Approval/BranchをInteraction側に示し、Tool未受信・Host Grant未照合をUnknownに保持 | メッセージ表示済み＝外部Effect確定／AIの承認済みと表示 |

**既存Ownerへの分担**：Everything is Data・Role #2619/#2620、PatternのAssume/Guarantee #2507、Rule/Operator #988、標準Plugin/Trait #951/#3278、生成器契約 #2090/#2092/#2096、Event/Business Flow #1125、Lifecycleの採否 ADR-0046/#1174、実AI Interaction/Tool #2390、Graph表示／参照 #1124、独立Expected #2514/#1180。三要素Methodは投影・案内・差分の**共通受入窓口**であって、これらのDefinition/Runtime正本を複製しない。

### 19.8 図表を全体のガイドとして保つ：現在地・往復・階層化、生成AI向け表現、設計書から実装への導線（2026-11-09、利用者明示の設計原則）

**原則（利用者の明示）**：大きなBusiness／Event Flow、状態遷移図、構成図のように機能や処理の全体像を描く図表は、個別要素を確認・編集している間も**全体の道案内**として働かなければならない。利用者は全体像の中の現在地を保ったまま個別要素へジャンプし、内容を確認し、元の位置へ戻れる。対象ドメインの知識がない人でも、大きな概念の中で個別要素を把握し、迷わずに作業できることを狙う。同じ案内は生成AIにも、直接読みやすい表現で提供する。

§19.1の主経路と§19.4の段階的な掘り下げは「どこへ進めるか」を定めていた。本節は、進んだ後も**どこにいるか**を失わない条件を加える。新しい正本、専用Graph Store、Studio固有のリンク集は作らない。案内は§19.2の図表体系と同じ意味集合から導出する。

#### 19.8.1 案内として成立する条件

| 条件 | 利用者に見えること | 導出元・責務 |
| --- | --- | --- |
| **G1 現在地** | 個別の定義・表・詳細図を開いている間も、全体像のどの区画・段階・階層にいるかを示す。GUIでは概観図上の強調、テキストでは上位からの経路で表す | 選択対象のQualified IDから、Model Mapの包含・所属関係を引いて求める。画面座標から逆算しない |
| **G2 往復** | 概観→詳細→概観の移動で、元の概観の表示位置・展開状態・選択が戻る。詳細から隣接要素へ横移動しても経路を辿り直せる | 表示位置・展開・履歴はPresentation state（[表現型間の連携設計§2・§4](REPRESENTATION_CROSSVIEW_COORDINATION_2026-11.md)）。意味変更として扱わず、Source Revisionの変更後は古い位置を現在の対象として復元しない |
| **G3 近傍と位置づけ** | 詳細側に、上位要素、前段と後段（何から来て何へ渡るか）、同じ段階の兄弟、関係するEvent・State・Function・Dataを示す | §19.2の型付き関係から導出する。関係の種別（業務の流れ、状態遷移、データ関係、所属）を同じ線に畳まない |
| **G4 階層化した概観** | 一枚で読めない規模の図表は、区画（サブシステム、業務段階、Event Group、Lifecycle）単位の概観と、その内訳の詳細図に分ける。概観の各区画から内訳へ入れ、内訳から概観へ戻れる | 区画は意味上の包含・Group・段階から決める。描画器の自動クラスタリングや見た目の近さを、意味上の区画として確定しない |
| **G5 集約の伝播** | 概観の区画に、内訳にある未解決・欠落・非公開・カスタム・Extension Slotの件数と種別が集約されて見える。詳細へ入らないと異常に気づけない状態にしない | §19.3の区分と§19.1の欠落区分を区画単位で集計する。集計値から非公開要素の存在・件数を漏らさない（#2516の開示判定に従う） |
| **G6 配置の安定** | 再生成や詳細からの復帰で、概観の主要区画の相対配置が意味の変更なしに入れ替わらない。利用者が形で覚えた全体像を壊さない | Layoutの保持はPresentation側の責務。意味の変更で配置が変わる場合は、変わった区画を示す |
| **G7 平易な入口** | 各区画と要素に、専門語を知らない人向けの短い説明（何をする所か、なぜここにあるか）を付ける | §19.1と同じく、原典に説明がなければ`description-unverified`とし、AIの推測を確定説明にしない |

これらは特定のGUI部品（ミニマップ、パンくず、分割表示など）の採用を強制しない。満たすべきは、現在地・往復・近傍・集約が**どのConsumerからも得られる**ことである。

#### 19.8.2 生成AIにとって読みやすい表現

生成AIも同じ全体像の中で作業する利用者として扱う。描画済みの画像、座標付きの図形データ、全件を展開した大きなGraph JSONを、AIが毎回直接読み解く前提にはしない。CLI・MCP等の既存の入口から、次の性質を持つ表現を得られるようにする。

1. **概観の目次**：Model Mapを区画の階層として返す。各項目に型、Qualified ID、平易な一行説明、内訳件数、未解決・カスタムの集約（G5）を付ける。深さと件数は呼出し側が指定でき、省略した部分は「省略した」と明示する。
2. **現在地の問合せ**：Qualified IDを渡すと、上位からの経路（G1）、前段・後段・兄弟・関係要素（G3）、属する図表とその中での位置、定義・Source・Evidenceへの型付き参照を一度に返す。AIが何度も全体を読み直さずに位置づけを把握できる。
3. **段階的な展開**：区画IDや要素IDを指定して一段ずつ内訳へ進む。一度に返す量に上限を設け、上限で切った場合は続きの取得方法と残件数を示す。
4. **欠落の明示**：未取得・未解決・非公開・未対応を、テキスト上でも「存在しない」と区別して返す（§19.1、MAP-07と同じ区分）。
5. **人間向け図表との一致**：同じ要素について、人間向けの図表とAI向けの表現で、Qualified ID、版、公開資格、区画の割当てが一致する（MAP-08の拡張）。

この表現は**AI専用の新しい正本や要約台帳ではない**。同じAccepted Meaning／Sourceからの別Projectionであり、具体的なMCP Toolの形やCLIの出力形式は既存の担当（#2516の用途相対Context Projection、[ADR-0048](../adr/0048-shared-application-api-and-transport-adapters.md)の共通API方針、[RQ-027](../../requirements/TEI_PRODUCT_REQUIREMENTS.md)の再入場Context）で決める。本節は受入条件だけを渡し、新しいSchemaを固定しない。AIが要素の位置づけを把握できても、それは意味の受理、編集権限、実行資格を与えない。

#### 19.8.3 図表は実装である以前に設計書であり、実装の直接のガイドになる

**原則（利用者の明示）**：Model Mapと§19.2の図表は、実装の状態を後から描く説明図ではない。まず**設計書**であり、その設計書が**直接的に実装のガイド**になる。

1. **設計書として読めること**：実装がまだ無い要素も、責務、入出力の型、前段と後段、State／Guard／Effectの不変条件、§19.3の操作区分（固定・設定可・Extension Slot・置換可）を図表と要素の説明から読める。実装が無いことは「要素が無い」ことではなく、`未実装`という資格として表示する。
2. **設計の資格を区別すること**：要素ごとに、設計の段階（提案中・受理済み）と、実装の段階（未実装・生成済み・手書き実装あり・検証済み）を別の列として持つ。図表が描けたことを実装済みや検証済みの証拠にしない（§19.7の「描けた／意味保存／生成可能／実呼出し／Effect確定は別の合格列」と同じ扱い）。
3. **要素から実装へ辿れること**：各要素から、実現を担う生成器・Template、Plugin／Trait実装、Hook、手書きのnative実装、試験・Driver Scenario、Evidenceの所在へ型付き参照で移動できる。対応する実装が無い要素、試験の無い要素は、概観の区画にも集約する（G5）。実装者は概観から「次に実装すべき未実装の要素」を選び、その要素の設計（G3の近傍、不変条件、変更可能箇所）を見ながら実装し、終わったら同じ要素へ戻って資格の変化を確認できる。
4. **方向を逆にしないこと**：設計書は受理済み意味（Accepted Meaning／Source）から導出する。実装コードや生成物から構造を読み取って設計書を上書きしたり、正本にしたりしない。実装からの観測は§19.3の`Observation-only`として設計との差分に示し、設計を変える場合は通常の受理経路（§19.4のカスタム経路⑤）へ戻す。
5. **生成AIの実装者にも同じ導線を渡すこと**：§19.8.2の現在地の問合せは、要素の設計内容と並べて、実装・試験の所在と未実装の範囲を返す。AIは全体像を読み直さずに、今実装する要素の設計上の位置づけ、守るべき不変条件、触ってよい範囲を把握できる。この導線はAIに編集権限や受理資格を与えない。

ここでいう「実装への参照」は、既存のSource Index、Generator・Plugin・Trait実装の束縛（§19.6、§19.7）、Driver Scenario（RQ-012）の所在を引くものであり、新しい実装台帳を作るものではない。参照が引けない場合は`unbound`として示す。

#### 19.8.4 受入ケース（MAP-17〜23、全件NOT RUN）

既存のOrg Case／Orderの受入対象を再利用し、新しいScenario正本を作らない。

| 受入ID | 期待する区別・操作 | 不合格とする実装 |
| --- | --- | --- |
| **MAP-17** | 状態遷移図の一つのTransitionから、Guard定義、関係するFunction、Source位置へ移動しても、全体の何段目・どの区画にいるかが表示され続ける | 詳細を開くと全体像との対応が消え、利用者が自分で元の図を探し直す |
| **MAP-18** | 概観→詳細→隣接要素→概観と移動したとき、概観の表示位置・展開・選択が戻る。途中でSource Revisionが変わった場合は、古い位置を現在の対象として復元せず、変わったことを示す | 戻ると初期表示にリセットされる。または旧版の選択を新版の同じIDとして黙って復元する |
| **MAP-19** | 一枚に収まらない業務Flowを、意味上の区画に基づく概観と内訳に分け、両方向に移動できる。区画にまたがる線は、概観では区画間の関係として集約され、内訳では端点が分かる | 全要素を一枚に縮小して読めない。描画器の自動配置の塊を意味の区画として扱う |
| **MAP-20** | 内訳にある未解決・カスタム・Extension Slotの件数が概観の区画に表示され、非公開要素は公開資格に従って件数も伏せる | 異常が詳細でしか見えない。集計値から非公開要素の存在が分かる |
| **MAP-21** | AIがCLI/MCPで、概観の目次、指定要素の現在地と近傍、一段ずつの展開を、上限付きの構造化テキストで得られ、人間向け図表とIDと区画が一致する | AIが画像や全件Graphを読まないと位置づけが分からない。AI向けの別要約が人間向け図表と食い違う |
| **MAP-22** | 予備知識のない利用者（人とAIを区別して記録）が、概観から一要素の目的・前後関係・変更可能箇所を説明し、概観へ戻って隣の区画へ進める | 個別要素は読めるが、全体のどこにあるかを説明できない |
| **MAP-23** | 実装が一部しか無いFlowで、未実装の要素も責務・入出力・不変条件が読める。各要素から生成器・Plugin・Hook・手書き実装・試験の所在へ移動でき、設計の段階（提案中・受理済み）と実装の段階（未実装・生成済み・検証済み）が別に示される。未実装・未試験の件数が概観の区画に集約される | 実装済みの部分しか図に出ない。図があることで実装済みと表示する。実装コードから読み取った構造で設計書を上書きする |

MAP-22は理解課題であり、§19.5のM2と同じく、人の評価が得られない場合はAI盲検や客観課題で代替し、その限界を記録する。

#### 19.8.5 既存段階への位置づけと担当

- **成熟段階**：G1・G3・G4とAI向けの目次・現在地問合せ（§19.8.2の1〜2）は、M0 Catalog／M1 Graphの上に**read-onlyで**成立させる。設計の段階と実装の段階の区別、実装の所在への参照（§19.8.3、MAP-23）もM0／M1で`unbound`を含めて出し、生成・実呼出しの資格はM4で確認する。GUIの往復・配置の安定（G2・G6）と理解課題（MAP-22）はM2に含める。M0が未成立のまま、見た目の概観図だけを先に作ってRQ-036の案内が成立したと呼ばない。
- **実装の現在地（2026-11-10）**：G1（経路）とG3（近傍）の最小形を、CLI `tei model locate`とMCP参照adapterの`tei_model_locate`としてread-onlyで実装した。`model catalog`と同じ読込みから、最上位からの経路、上限付きの子・兄弟（`total`・`omitted`・種類別件数）、宣言参照6種（Relationの参照先、属性の型、PatternのData・Relation・Query・Capability）の入出力を返す。参照先が無いものは`resolved:false`で残し、重複宣言は位置を選ばず`SUBJECT_AMBIGUOUS`にする。導出していない関係（業務イベントの流れ、状態遷移、Queryのデータ依存、実装・試験との対応）は`coverage.notDerived`に名指しする。これで成立したのは§19.8.2の2（現在地の問合せ）のData・Pattern部分だけであり、G2・G4〜G7、§19.8.3の実装への参照、MAP-17〜23の受入は**未成立**。
- **実装の現在地（2026-11-10、G5の最小形）**：CLI `tei model overview`とMCP `tei_model_overview`で、§19.8.2の1（概観の目次）を最上位要素（意味名前空間、Application、Query）の区画として返す。各区画には部分木の件数・種類別件数と、部分木のどこかにある**要注意の集約**（validationのerror・warning、未解決の宣言参照とそのrole別件数、重複宣言、Source参照の無いWorkspace宣言要素）を付ける。区画一覧を`-limit`で切っても`totals`は全区画の合計であり、切ったことで問題を隠さない。どの要素にも帰属しないfinding（subjectの無い診断、catalogに無い種類の診断、Source rootのfinding）は`unattributed`に種類別で残す。`model locate`も同じ数え方の`attention`を、位置を示した要素の部分木について返すため、詳細と概観の数は食い違わない。**集約していないもの**：カスタム・Extension Slot（標準とカスタムの区分がまだ型付きで導出できない）、業務イベント・状態遷移の欠落、実装との対応、Consumerごとの開示判定。開示判定は未適用であり、見える範囲は`model catalog`と同じである（G5の「非公開要素の件数を漏らさない」は、#2516の開示判定をこの経路へ通すまで**未成立**）。
- **担当**：Model Mapの包含関係と現在地の導出は#1124、区画をまたぐBusiness/Event Flowの集約は#1125、選択・表示位置の連携はPresentation stateとして[表現型間の連携設計](REPRESENTATION_CROSSVIEW_COORDINATION_2026-11.md)と#2500の境界、AI向け表現と開示判定は#2516、CLI・MCPの入口は[ADR-0048](../adr/0048-shared-application-api-and-transport-adapters.md)の共通API、要素から生成器・Plugin実装への参照は§19.6／§19.7の担当（#2090/#2092/#2096、#951/#3278）、独立したExpectedは#2514/#1180。本節は表示・案内の受入条件を加えるだけで、これらの正本を複製しない。
- **撤回条件**：区画化や現在地表示が、利用者の理解課題（MAP-22）で全体を一枚で見せる場合より改善しないと分かった場合、G4の分割方式を見直す。現在地・往復・欠落表示（G1・G2・G5）は、改善の有無にかかわらず誤った全体像を見せないための条件として維持する。

## 19. Trait根拠の三視点Bind Reference（2026-11-07、限定試験）

**依存関係の更新**: 本設計の旧[Draft #2878](https://github.com/hat47x/tei-platform/pull/2878)およびTrait定義・Intent／Observation／Assessmentの[Draft #3027](https://github.com/hat47x/tei-platform/pull/3027)は**2026-11-07にmain統合済み**。両者の設計文書を新たな第二正本に複製しない。本節は前節T10〜T13を、独立したread-only Reference [Draft PR #2972](https://github.com/hat47x/tei-platform/pull/2972)でどこまで実証したかを追記するもの。Product採用・本番Provider・Meaning Acceptanceではない。

### 19.1 意図・事実・適合を三視点の表示Occurrenceへ明示結合

追加した`reference/tei-three-element-method/trait-attachments.mjs`の`bindPublicTraitClaims(publicProjection, envelope)`は、既存`projectPublicTeaSlice`が返す**公開資格済みData/Function/Business Occurrence**を入力とし、外部Ownerが選択・開示したTraitの参照だけを束縛する。演繹や判定は行わず、`Intent`、`Observation`、`Assessment`の来歴・局所状態・Evidence参照を**そのまま**表示向けに運ぶ。概念上は`Trait Reference → (Qualified Subject, Role, View Occurrence, exact caller snapshot)`という対応。Reference内で新しいTrait Definition/Proof/Assessment/Policy/Source ReaderやWriterは実装しない。

- `sales.order.status`相当の同一合成Subjectで、Businessのclosed-domain意図は`proposal`、DataのJava Enum・DB制約は独立`source-observed`、FunctionのWriter保証不足は`not-proven`という**入力側の異なる主張**として表示する。Enum/DDLからAssessmentを生成せず、業務MECE・実DB違反件数・実Provider適合を自動確定しない。
- `View ID`の型付き表示対応を使い、同名異Scope、欠落Subject、Role不一致、重複Claim、旧`snapshot`、秘密属性が混入した不正Inputは`unqualified_trait_attachment`で拒否する。新旧一貫性の判断は呼出側が供給する`snapshot`文字列の**一致検査だけ**であり、Sourceの現在性/Role実実装/現在Grantの独立readbackではない。
- 「強属性」は厳密な原典定義未確認につき追加しない。物理PK・業務Identity・FD・immutable等の異なるClaimを自動統合せず、新`tei.std` Role/Facetや「Strong」というBadgeを生成しない。ClosedDomainとMECEの母集団/排他/網羅も別。
- 入力は**すでに開示資格確認済み**である必要がある。非公開TraitやSecret Source Pathを投入して「出力から消えたから安全」とみなさない。#2516の実二世界Disclosure判断、#1124の実DOM/ARIA/Tool/Workbenchと旧応答失効は独立検証。新しい`lanes/states`定数は合成Fixture内の表現であってTEI正式Enumではない。

### 19.2 追加Referenceの限定実証

2026-11-07、**Node.js 22.16.0**で、既存`public-slice.mjs`→新`trait-attachments.mjs`を**同一テスト内で実行**した。10件すべてPASS。主な検査はT10（Intent/Observation/Assessmentを区分して表示・静的観測から適合を捏造しない）、T11（強属性/業務Identityを捏造しない）、T12（別Scope/旧Snapshot拒否）、T13（Public投影に存在しないPrivate Trait拒否）、入力順序、Claim ID重複、未知Role/過剰Metadata、Getter非実行。ローカル実行ファイルとGitHubのBlob SHAを照合した。

| 固定ファイル（PR #2972） | Git blob SHA |
| --- | --- |
| 既存`public-slice.mjs` | `9b46cf13f3683607a384ea629eebba99a59ce872` |
| 新`trait-attachments.mjs` | `72d3a993792bfd09715271b0d0030249a052fd42` |
| 新`trait-attachments.test.mjs` | `4f6e625bff475ed5e9fb529912123edadd2c4ae3` |

**未実行・未資格**: 実Java/Go/.NET Enum Reader・実DDL/DB値・未読込Writer inventory、実Source Index/Role/Operator Pins、実PolicyがPrivateから正しいPublic Sliceを生成すること、完全な同一AttemptでのSource→Studio 4入口・Graph/DOM/ARIA/Tool/AI、現在Grant/Source readback、Owner受理/Target Effect、trusted全Repo Health。旧`public-slice`の9件と新10件は**別Fixture資格**であり、両者を合算してTEI製品E2E合格とはしない。JavaScriptの任意Proxy・特殊配列による入力side effectも一般保証しない。

### 19.3 次のP0への引渡しと撤回条件

直接クリティカルパスは従来どおり[#1124](https://github.com/hat47x/tei-platform/issues/1124) M0実CLI→M1同一Workbenchの曖昧0/一意1 InspectとGraph旧DOM拒否。三要素MethodはこのConsumerへ**すでに資格確認されたSlice/Role/Claim**を渡すのを優先し、Method独自の認可・Source Writer・Plugin Resolver・Assessmentを新設しない。並行Referenceの次の最小一巡は`sales.order.status`について同一Source/Scope/Attemptを固定し、(1)実Script/Roleの必須Source・Parser診断、(2)実Sourceで観測されたJava/DB/Function制約、(3)既存Trait Ownerによる比較・現在Disclosure、(4)実Workbench三視点の同一根拠表示を順に結ぶ。実Providerが不在なら`unverified`を出し、架空の独立Expectedを採用しない。

**停止**: 方法論Plugin更新や表示だけでAccepted Meaningが変わる、Java EnumからMECEを証明したことにする、未知Writerを0件とみなす、Private Traitの存在を件数やBadgeで漏らす、局所PASSを全Consumer／Production／セルフビルドの資格へ昇格する場合には、受入対象を縮小・撤回して既存Ownerへ返す。

### 19.4 三者Snapshotの結合不足を閉じる限定Reference（2026-11-08）

[Draft #2972](https://github.com/hat47x/tei-platform/pull/2972)の前版は、`Envelope.snapshot = Claim.snapshot`を検査していたが、**実際の公開ProjectionにSnapshotがなく**、同じ`viewId`を持つ旧図へ新Claimを結べた。Sourceの現在性とは別の、合成Reference内部の欠陥である。新Referenceは`projectPublicTeaSlice(publicSlice, callerSnapshot)`が公開されたOccurrenceに**任意の呼出元世代Witness**を付与できるようにし、`bindPublicTraitClaims`で`Projection.snapshot = Envelope.snapshot = 各Claim.snapshot`を要求する。Stampなしの従来Projectionは単独閲覧を維持するが、Trait結合を拒否。旧版と新版で同じ`viewId`のまま、前者を新Envelopeに渡す反例とStamp未提供の負例を加えた。

**追加の限定検証**: GitHubから実ソース4件を取得し、そのままV8内で読み込んだNode test互換ハーネスで**22/22件PASS**（`public-slice`10件＋`trait-attachments`12件、うち新規T12 Projection世代差・未Stamp拒否）。これは正規の`node --test`による実行ではない。これを補うため[別Draft #2972](https://github.com/hat47x/tei-platform/pull/2972)へ専用の[exact-head Node 22 CI](https://github.com/hat47x/tei-platform/blob/reference/tea-public-slice-two-world-20261106/.github/workflows/tei-tea-three-view-reference-node.yml)を追加し、同じ10件＋12件を実Node test runnerで確認する契約を置いた。[PR #2972 exact head `7289e86581af9e91a74e7e3b73667293886c5f9f`の専用synthetic-nodeジョブ](https://github.com/hat47x/tei-platform/actions/runs/42385331823/job/128441389821)は**completed / success**。ワークフローはNode 22.16.0と実headを固定し、上記公開Slice10件・Trait Bind12件を正規`node --test`で実行する。これは新しい**局所Native Node資格**であり、Repository Health・脆弱性スキャン・Studio製品受入の成功を含意しない。新コードの固定Git blobは`public-slice.mjs=265206ab45a3a8d357e113dd9e02a65e2137ed59`、`trait-attachments.mjs=2d9e2eefdb2e4edb2370471c0a1c1ce0c7b29b13`、テストは`ca83ee66a97550213653db5a8a0bf8c8282afbe0`／`3795cbedbbff2cf1bf6f7f2dee55468ec9a41f19`。旧§19.2の10件Node PASSは更新前の履歴で、今回の最新資格と混ぜない。

**重要な非主張**: このSnapshotは呼出元が割り当てる**等値Witness**であって、改版した実Source、Source Index、Role/Operator実bytes、現在のGrant、Producerが実読んだ対象、Callback時の表示世代を独立して証明しない。同じ文字列の再利用・ABAや偽Stampを単体では検知できない。#1124 M0/M1と#2516の実Consumerでは、Source/Role/Policy/Consumerの独立現行証拠と画面上の失効・後着拒否を別assertする。いま達成したのは、**明示的に異なる表示世代の合成Reference内誤結合を拒否すること**だけである。

## 20. サブシステム分割候補の三視点表示（2026-11-08、未採用・未実行）

他チャットの[Issue #3250](https://github.com/hat47x/tei-platform/issues/3250)／[Draft #3255](https://github.com/hat47x/tei-platform/pull/3255)で提案される**更新凝集を基準としたサブシステム分割**を、本TEA Methodからは「計算アルゴリズム」ではなく**提案と根拠の三視点Projection**として扱う。#1132は越境Direct CUDのread-only診断Ownerであり、TEAは最適化・Owner受理・Shared Acceptance・実DB Effectを再実装しない。

| Data | Function | Business |
| --- | --- | --- |
| 業務Dataの所有候補、Relation、ER範囲と外部参照を保持。物理DB配置・代理PKと混同しない | **宣言上のDirect C/U/D**、Read、別Functionへの委譲、**潜在／観測済みEffective Effect**を別Relationとして示す | Procedure、Manual Step、担当Actor、Trigger、委譲による業務責務・制約を示す。Actorは実Authorization Principalでない |

**局所契約**: `Accepted Source/Qualified Subject/Scope/Revision/解析Policy/必要Closure/現在Disclosure`を固定できる部分についてだけ同じDiagram/CRUD表/Subsystem別ER図を照合する。分割案は`proposal`、規則に照らした比較は`derived assessment`、実現されたFunction/DB/Effectは別Evidence。提案の見た目や境界移動だけで実SourceのDirect CRUDを消したことにしない。未知のFunction/Relation、未読込Source、非公開CUDは`0`へ変換せず、母集団の被覆不明なら「図にない＝全Sourceに存在しない」と主張しない。View間の同名端点はQualified identityとRelation roleが確定するまで保留。公開用配置・線は既存T06の開示Slice境界を守り、秘密の越境Edge数・評価点を漏らさない。

| TEA専用受入候補 | 反証すべき誤結合 | 所有する独立証拠 |
| --- | --- | --- |
| **T14 Scope固定の越境Direct CUD** | Sales FunctionがInventory Dataへ直接`Update`を持つとき、別Subsystemへの`Read`だけと同列に扱う／未読込辺を0本とする | #3250の計算・#1132のCUD診断、#1124の同一Consumer、#2881の限定被覆 |
| **T15 委譲による境界修復** | Function所属変更と、Command委譲による実行契約の変更を同じ可逆図面操作とする／Direct CUD削除だけでEffective Effect・Tx・Recoveryが保証されたとする | #3255のpartition-only／repair区別、#1121/#1174の実Effect、#267 Shared Acceptance |
| **T16 Proposalの表示と開示** | 最適化候補やOwner/ScoreをAcceptedと表示する／非公開Dataの追加でPublic図の位置・候補数・評価を変える | #3250の候補証拠、#2516/T06の二世界、#1124 Studio |

**優先順・停止線**: #1124の実CLI→同一Workbench Consumer結合を最優先のまま維持する。T14は既存#3250/#1132の*同じAccepted Sourceと解析範囲*を持つread-only結果が得られた後、T15/T16は委譲/Current PolicyのOwner証拠が揃った場合のみ検証を起動する。新たなSubsystem ID正本、最適化エンジン、Graph schema、権限主体、実行Provider/DB移行、トランザクション制御をTEAに導入しない。**T14〜T16は設計Caseであり、Go/Node・実Source/Policy/Studioの試験もOwner採用もNOT RUN。**

## 21. WorkspaceとSnapshot同一性、およびRead／Validation／Coverageの独立資格（2026-11-09）

### 21.1 Workspaceを区別しないSnapshot一致は三視点根拠のJoin条件として不足

[Draft #3321](https://github.com/hat47x/tei-platform/pull/3321)は、**異なる実WorkspaceでもWorld／SourceIndex／Snapshot Fingerprintが等しい**事例を実Go CLIで再現している。ManifestのNameだけ異なるCopyはManifest差を検出できるが、Manifestもbyte-identicalの別Rootは既存Load-time値だけでは区別できない。**表示ID・同名Subject・Source fingerprint・Snapshot文字列の一致だけで「同じWorkspace」「同じ実Read Attempt」を証明しない。**

[TEA Reference Draft #2972](https://github.com/hat47x/tei-platform/pull/2972)では合成限定の追加受入として、事前公開された`publicSlice.workspaceObservation`を`publicProjection`へ渡し、`Projection / Envelope / 各Trait Claim`の`workspaceObservation`が**同じCaller公開用Witness**であることを必要条件に加えた。従来の`snapshot`三者一致とは独立。**T12-W1** 同じSnapshot・ViewIDでも異なるWorkspace観測なら拒否、**T12-W2** Claim側の異WorkspaceまたはWorkspace観測欠落を拒否、**T12-W3** Public SliceのWorkspace観測なし／空は拒否。秘密のManifest path/物理Root識別子を公開Payloadへ持ち込まず、開示資格済みopaque値をCallerが用意する前提を明記する。

これは**実WorkspaceのIdentityを保証する実装ではない**。Callerが完全コピーした別Rootへ同じ`workspaceObservation`を誤発行すれば見分けられない。Root/Source Reader/Attempt/現在Grant・Policyの実測と照合するのは#1124/#2516/#3181等のOwnerであり、ここでは新UUID、Hash方針、Context Registry、第二Policy Engineを採用しない。

**限定試験の更新**: GitHub実ソース4件を同じV8互換test runnerで実行して`12 public-slice + 15 Trait = 27/27 PASS`（合成Referenceのみ）。[PR #2972 exact head `7627f5d59dc66cbb161067c0ca7bfb44a7587c12`の専用Node 22 CI](https://github.com/hat47x/tei-platform/actions/runs/42591779028)は**completed / success**。対象は公開Slice 12件とTrait Bind 15件（合成27件）であり、GitHubへのSource/Fixture保存と別に正式Node test runnerで再現できた。**Repository Health・実Policy/Studio/Root identity・Owner受理は別資格**。以前の`10 + 12`や旧head CI成功を新headの合格とは扱わない。**実CLI #3321のSource照合と局所Referenceの成功を足し合わせても、同一Workspaceの実Consumer受入は未実証**。

### 21.2 各三視点Claimを一つのVerifiedへ縮約しない

最新の実装境界と反例から、TEA複合図は少なくとも下表の**独立した確認事項**を混同しない。各欄を合成RoleのState Enumへ固定するのではなく、既存Ownerの型付き結果・版・根拠リンクを対応付ける表示設計とする。

| 確認する問い | 実際の境界／現況 | 図でしてはいけない断定 |
| --- | --- | --- |
| **Read**：実Catalog/Inspect/Traceを取得したか | [Draft #3543](https://github.com/hat47x/tei-platform/pull/3543)は実CLI A/Jと同一WorkbenchのInitial/List操作を限定実証。曖昧AのInspect増分0/0、一意Jは1。Graph/Command、現在開示は別 | 実CLI値一致だけで全四入口と旧DOM操作まで合格 |
| **Validation**：ソース／意味制約を検証したか | [PR #3728](https://github.com/hat47x/tei-platform/pull/3728)は検証NGでもCatalog読取に成功するのを保ち、`validation.ok`を別提示する候補。World Fingerprintは同じでもRead OKとValidate NGが同時成立し得る | `catalog.ok=true`やSnapshot一致を`validation.ok=true`に昇格 |
| **Identity**：Workspace/Source/Role/Subject/Attemptが同一か | #3321では実Workspace同士でSnapshot同値。#3543の実CLIとStudio Catalog比較は原子的同一Read Attemptではない | 同名＋同FPだけでRoot/Read Attempt同一を断定 |
| **Coverage**：必要Source/Relation/Role/Writerをすべて取得したか | [Draft #3518](https://github.com/hat47x/tei-platform/pull/3518)は不完全なModel Mapを限定的に表示。旧#2881はGraph/Table合成Reference。未読込・非公開は0件と別 | 図に線がないことから全SourceでRelation不在、越境CUD 0を断定 |
| **Eligibility**：現在の用途で表示・Inspect・提案を許せるか | #2516の現在Disclosure、#1124のSelection/Consumer freshness、#267のShared Acceptanceを再利用 | Render可能・Validation成功・Meaning一致から認可／Effect／Owner受理を得たとする |

### 21.3 次の反証可能なTEA Consumer引継ぎ（全て新規実行前）

- **T18 正常Read・検証NG**：同じWorkspaceでCatalog内容は取得しつつValidationがNGのData/Function/Businessビューを表示できる場合、`Read成功`と`Source健全`を別の表示根拠とし、編集提案への資格を勝手に付与しない。#3728の実行証拠をそのままTEA GUI実証に転用しない。
- **T19 別Workspace同Token**：#3321の実Workspace W0/W1と同じSnapshot・同じSubjectを、別の観測Load Contextとして実Consumerへ提示する。CallerのWorkspace文脈が違えば旧Claim添付を拒否。完全コピーで同じLoad Witnessを誤発行する場合は未検出と明示し、実Reader/Root側の独立証拠を別Ownerへ返す。
- **T20 部分被覆と曖昧選択**：#3543のA/J実Workbench証拠に#3518の部分Model Map・#3728のValidationを**同じWorkspace/Subject/Attemptで**接続できる範囲のみ再利用。Ambiguous AはInspect 0、許可済み一意Jは1、未知Relation/未検証Ruleを非存在へ昇格させず、公開Policyは別に検査する。

**計画への反映**: #1124 M0→M1の直接P0は、実CLI A/J→同一Workbench Initial/Listの**限定成功**を再利用し、残るGraph/Command・旧DOM・Source/Role/Grant/current Disclosureとの実合流へ絞る。TEAは独立のTarget/UI/Policy/Validatorを設けない。T18〜20は**設計受入候補であり、実Studio GUI・実Policyを結合した試験はNOT RUN**。Builder/Self-build、ETL/MFT、#3250のSubsystem計算・実DB Effect、Plugin意味受理は既存Ownerの別証拠を待つ。

## 22. 型付きRead/ValidationとPublic Projectionの入力境界を分離する（2026-11-09）

**追加で実際に照合したOwner証拠**: [merged #3728](https://github.com/hat47x/tei-platform/pull/3728) は`operations.ModelCatalog.validation.{ok,errorCount,warningCount}`を`validate`と同じ規則で出す一方、検証NGのWorkspaceでも**Catalog readが成功する**契約を実装。[merged #3492](https://github.com/hat47x/tei-platform/pull/3492) は実Source IndexのPath/Range/Fingerprintが構造的に正しくても、そのReferenceのSubject IDを偽装すると`Index.Validate`とTraceが通り得て、実`InspectSubject`が`SUBJECT_NOT_FOUND`となる負例を追加。これらの実Goテスト・Source Reader・Validatorを本TEA Referenceに複製しない。

### 22.1 Read / Validation / Semantic Binding / Disclosureはそれぞれ非同値

| TEAが受け取る情報 | 適用資格と禁止する推論 |
| --- | --- |
| **Catalog read**: CLI envelopeの`ok`、表示可能なEntries | Read可能な場合でも`validation.ok=false`を維持できる。読めたことから正常／Accepted／Edit可能を導出しない |
| **Validation summary**: `ok`、errorCount、warningCount | これは**観測Workspace全体**に対する結果であり、各Data/Function/Business Subjectの個別違反件数やRelation不存在証明ではない。詳細の正本は`validate -json`でありTEAが複製しない |
| **Source/Subject binding**: Source Index Artifact/Range/Byteと実Canonical Inspect | Byte/Rangeの構造妥当性は意味Subject実在性ではない。Pathだけ合致する旧Indexや偽装SubjectをGraph Source jump/可逆編集根拠へ昇格しない |
| **Disclosure**: 現在Consumer用途に公開可能なSlice/診断 | 生のCatalog全体ErrorCountや非公開Subjectに起因する診断を、そのまま公開図・検索件数・ARIA/Tool JSONへ投影しない。全体Validation公開可否そのものは#2516の現在Policyへ委譲 |

Readに失敗した場合はValidationが`false`なのではなく**未観測**。Validationに失敗しても、現在許可される**read-onlyでのData/Function/Business閲覧**を機械的に全停止しない。ただし出典Span・Use-site・Role/Operatorの必要依存・現在開示が不明なら、その局所Source Jump/Inspect/Proposalは保留する。公開図で`errorCount=0`の意味を「見えないエラーなし」から「全Sourceにエラーなし」へ強めない。

### 22.2 公開済みSliceの入力配列は非実行のデータに限定

[TEA Reference Draft #2972](https://github.com/hat47x/tei-platform/pull/2972) の従来`onlyFields`はObject fieldのgetterを拒否していたが、`subjects`・`relations`・`claims`・`views[role]`の**配列要素getter／独自`Symbol.iterator`／疎配列／余剰Metadata／改変Prototype**を拒否できなかった。新しい`plainArray`は各配列について普通のArray prototype、Own key総数`length+1`、全要素のOwn Data Descriptorを確認してから反復する。匿名公開端点やPrivate Stateへの意図しないgetter実行を防ぐ**局所Referenceの制限**であり、入力の意味的資格や開示判断は引き続き外部Ownerが供給する。

新T06/T13負例を**4件**追加。GitHub上の更新済み4モジュール/試験のbyteを直接取得してV8 test互換ハーネスで**31/31 PASS**を確認したが、**Node test runnerの正規CI・Repository Healthは更新headの完了結果を別途照合する**。従来27件のCI成功を新31件の合格として流用しない。任意Proxy/Array prototype全域のmonkeypatch、入力規模による資源攻撃、タイミング/副作用の一般非干渉はこのReferenceでは保証しない。正式な実入力隔離と認可は既存#2516/#1124に接続する。

### 22.3 追加の反例候補と実行順

| ID | 反証 | 状態／責務 |
| --- | --- | --- |
| **T21 Read-success / Validation-fail** | Catalog Readに成功した実Workspaceで`validation.ok=false`、かつ許可済みread-only三視点は表示可能。全Source正常／Owner受理は出さない | #3728のGo局所実証はmerged。TEA実Consumerへの結合は**NOT RUN**（#1124） |
| **T22 Structurally-valid / Semantic-invalid Locator** | Source Byte/Range照合が通ってもSubjectを偽装すると実Inspectが拒否。Source Jump、Graph/CUD Cell、Proposalを有効化しない | #3492の実Workspace Go反例はmerged。TEA実GUI Source Jumpは**NOT RUN** |
| **T23 Private diagnostic-count noninterference** | 非公開制約の追加で生Catalogの総Warning/Error数が変わる場合、Policyで開示不可な低権限Graph/Table/ARIA/Tool JSONの件数を変えない | #2516の実Disclosure二世界検査に接続。TEAでの実Policy実行は**NOT RUN** |

**ロードマップ**: #1124の同一Workbenchで実CLI A/JのInitial/Listに続きGraph/Command/旧DOM/Current Policyの負例を先行。並行してT21〜T23を既存#3728/#3492/#2516のOwner証拠へ照合し、Sourceの現在位置／意味Subject・読取／Validation／開示の資格を独立表示する。新Source Validator・Trait Evaluation・Policy/Authority判定・Second SSOT・専用HostやModel Mapを作らず、Self-build/ETL/SubsystemのOwnerと既存Gateを変更しない。

### 22.4 T18の限定Read/Validation表示Reference（2026-11-09夜）

[Reference Draft #2972](https://github.com/hat47x/tei-platform/pull/2972)へ`catalog-validation.mjs`と`catalog-validation.test.mjs`を追加。これは[main統合済み#3728](https://github.com/hat47x/tei-platform/pull/3728)の`ModelCatalog.validation`を**計算し直す実装ではない**。Callerが既に開示資格を確認した`catalogReadSucceeded:true`と`validation.{ok,errorCount,warningCount}`を、同じ`scope/snapshot/workspaceObservation`の三視点Public Projectionへ添えて、Read成功とValidation NGを独立表示するread-only Fixture。CLIの`ok`や実Read Attemptを本Referenceが検査したわけではなく、`catalogReadSucceeded`の真偽も呼出側の申告である。

**新たな6件の合成T18負例**: (a) Read成功＋Validation NGでも三視点read-only Occurrenceを保持、(b)warning>0でも`validation.ok=true`をそのまま表示、(c)Read失敗/未観測・Validation欠落から`false`を発明しない、(d)同じ表示IDでも別Workspace/Snapshot/Scopeは拒否、(e)件数・booleanの型不正や負数を拒否、(f)秘密Source metadataやGetterを評価せず拒否。現行の合成Reference計**37件（Public Slice14 / Trait17 / Catalog6）**をGitHubから取得した正確な3ソース・3テストでV8互換ハーネスにより**37/37 PASS**。これは**正規Node test runnerの完了結果ではなく、実Go Workspace・実Catalog JSON・現在Policyを受領したテストでもない**。更新後exact-headの専用Node CIとRepository Healthは別途確認する。

特に**グローバルなValidationのError/Warning総数**は非公開Sourceに由来し得るため、公開可能かの判定は#2516が先に行う。未開示の生Countsを「TEAが実際に秘密を除去する」目的で流し込まない。合成RuntimeはAssessment・個別Subjectの健全性・Editable・Owner Meaningを創作しない。今後T21〜T23へ接続するには、**実Catalog Producerの読取版・Workspace・Scope・現在Disclosure→同一Workbench Consumer**の別資格を必要とする。

## 23. Lifecycle Graph・AI Harness・三視点Modelerの統合資格（2026-11-12）

**状態**：RQ-036/#3503/#1124/#3278/#1180/#2514への設計引渡し。Graph Compiler、Diagram Writer、AI Chat/VUI/Harness実Target、実DB/Effect、Owner受理はNOT RUN。Draft #3507の旧Branchは現時点で参照できないため、本節を独立した既存設計への追補とする。第二Graph正本/Resolver/Workflow Host/Scenario Runnerは作らない。

### 23.1 Authoritative / Proposed / Effective / Observedを分離

| 対象 | 意味・資格 | 確認点 |
| --- | --- | --- |
| Authoritative Source / Accepted Meaning | 業務意味・型・Relation・Operation・Owner受理 | Source/Revision/Scope/Meaning |
| Proposed Graph Patch / AI Candidate | 人/AI/Harnessが提案したNode/Edge/Guard/Binding変更 | stable ID、対象選択、競合、出典。未受理なら実行不可 |
| Effective Graph / Selected Provider | 受理済みSourceと選択Pluginを合成した構造 | Lock、型、全Path Obligation、未選択Plugin非干渉 |
| Generated Target / Actual Invocation | Artifact、実Host/Consumer、Effect/Receipt | 選択B→実B呼出し、Target Coverage、Effect/Unknown |
| Diagram / Session / Observed Trace | 編集・表示・文脈・実行観測の投影 | Read/Disclosure、現在Grant、同一Attempt/Scope |

**固定Selection Context**：UIのハイライト/行番号ではなく、stable qualified Subject ID＋Source span＋Workspace/World/Snapshot/Revision＋Scopeを保持する。選択解除、Canvas切替、Harness切替で対象を暗黙変更しない。旧Revisionの変更は競合として提示し、AIが意味対応を推測して自動再結合しない。

### 23.2 各製品への投影

- **TEI Modeler**：Data/Function/Business、Event/State、Lifecycle/Plugin、CRUD/Relation Cross-referenceを標準Graph/Roleから投影。Edge ID、多重性、向き、NULL/Absent、Source位置を欠くProjectionはread-onlyとする。
- **SHO Composer**：文章・表・図への引用と挿入でSource/Accepted Meaningの資格を保持。引用先が同名異内容へ変化した場合は保存・再開時に再解決。SHO編集をTEI業務正本へ自動昇格しない。
- **SUI/CSW**：Card/Island/KJ法的な探索Clusterは候補構造であり、業務Relation/OperationのAuthorityではない。Owner受理前はProposed Graphに留める。
- **共有AI Harness**：セッション・文脈・Selection Contextは製品間で受け渡せるが、現在Grant/Disclosure/Effect/Source資格はOperationごとに再検証。DSH/Codex/Claude Code等は等価なAdapter候補として扱い、標準Graphを特定実装へ従属させない。
- **AI Chat/VUI**：Stream delta、partial tool arguments、ASR候補は未確定Event。Tool Intent確定→Host権限→Application Operation→Effect/Observationを区別し、Regenerate/Branch/Interrupt/Resumeで旧Effectを暗黙再実行しない。

### 23.3 独立Expected候補（既存Scenario SSOTへ返す）

| ID | 敵対的入力 | Expected |
| --- | --- | --- |
| LGX-17 | UI選択固定後に別Canvasへ移動してHarnessへ送信 | 元Subject/Source/Revisionを保持し、対象の暗黙変更なし |
| LGX-18 | AIがEdge変更を提案するがOwner未受理 | Proposed表示可、Effective/Artifact/Effectは不変 |
| LGX-19 | 同名の多重Edgeを図表編集 | Edge IDごとに識別。曖昧ならread-only/拒否 |
| LGX-20 | Source A→B→A、Sessionは継続 | 旧Meaning/Grant/Artifact/Evidenceを自動復活させない |
| LGX-21 | Chat partial Tool入力・音声候補でUI更新 | 表示更新可、Application Operation実Callは0 |
| LGX-22 | CLI→Studio→HarnessでFunction/Source/Attempt照合 | Selected/Loaded/Invoked/Received/Effectを別証拠とし欠落はUNKNOWN |
| LGX-23 | SHO引用後に同名異内容のSourceへ差替え | 旧Source同一性を主張せず再解決/差分を要求 |
| LGX-24 | B2C Headlessが標準Rendererを使わない | Operation契約維持、TEI必須Browser Runtime/DOM/hopなし |

上記はテスト実行結果ではない。#1180/#2514がExpected/Scenarioを所有し、既存LG/TLG/MAPとの重複を統合する。Selection Contextは#1124/#3262、Plugin/Graph合成は#3503/#3278、実Provider/Function Readは#1251/#2586へ引渡す。

### 23.4 実装順序と停止条件

1. **P0**：通常CLI/Studioの選択Provider実呼出し、Function→Query→実SQL/Rowset→Data Contextを同一Attemptで観測。合成Go試験と製品実行を区別。
2. **P1 read-only**：実WorkspaceのM0 Model Map/Source Indexでstable ID・多重Edge・対象固定・Scope/Disclosureを確認。
3. **P1 Graph Patch**：既存OperationへのValidation追加と条件分岐。Guard/Effect/Authority迂回を拒否し、Source帰還→Target生成を検証。
4. **P2**：Chat/VUI/Harness/SHO/B2Cを異質Adapter/Projectionとして検証。Session継続とAuthority継続を分離。

**停止**：第二正本/権限Engine/Hostを導入、候補を受理済みMeaningへ自動昇格、同名をIdentity扱い、実Provider未呼出しでE2E PASS、Session継続をGrant継続扱い。いずれも方式再検討の対象。

## 24. 敵対的レビュー：Graphの可変性を「受理・生成・実行」へ誤昇格させない（2026-11-12）

§23を実装可能にするための**追加設計候補**。新たなGate/Graph Compiler/Workflow Host/永続台帳は導入しない。既存#3503/#3278/#1124/#1180/#2514/#1251の責務を保持する。

### 24.1 5種類の差分と失効範囲

| 変更 | 必ず再検査する対象 | 原則として失効させない対象 |
| --- | --- | --- |
| Diagram Layoutだけ | 表示・操作可能領域、選択位置 | Accepted Meaning、Semantic Graph、生成Artifact |
| Graph Node/Edge/Guard/Role | Source差分、**意味が変わる範囲の既存レビュー**、全Path Obligation、Target Plan/Scenario | 無関係な別Scopeの意味・生成物 |
| Plugin選択・実装・Lock | Provider/Hook適合、合成Graph、生成物、実Invoker証拠 | 無関係な未選択PluginのConsumer |
| Harness Session/AI提案 | 対象ID/Scope/Sourceの現在性、開示、提案の採否 | 未受理提案だけではBusiness State/Effect |
| 実行中のGraph/Entity/Grant改版 | Attempt/Operation identity、再承認、Resume/Migration、Effect unknown | 旧Effectの事実そのもの。新実行への自動再利用は不可 |

**失効は証拠の用途別**：Source Observation、Structural Candidate、Accepted Meaning、Build Artifact、Execution Receiptを一括破棄・一括再利用しない。A→B→Aでbyteが一致しても、途中の権限・Generator・Plugin・Owner受理の変更をなかったことにしない。Scopeの異なるConsumerへ「同じIDだから同じ資格」を流用しない。

### 24.2 Graph Patchの実装に必要な二段階の受理

- **構造受理**：Node/Edge stable ID、source revision、type/NULL/Absent、Patch precondition、Provider/Target capability、合成競合、到達可能性、全Path Obligationを検査する。静的UNKNOWNはPASSでなく、対象TargetにRuntime Guardの実証が必要。
- **意味受理**：承認条件、業務Rule、Function/Relationの役割、操作対象、開示範囲を、**既存の通常開発と同等のレビュー・変更管理**で担当者が必要な範囲を確認する。新しいTEI専用の多段階承認エンジンや、全項目の個別署名を要件にしない。構造受理だけで業務意味が正しいとはみなさず、生成AIの推測は出典のある事実へ自動昇格させない（#267/#2621/#4143）。
- **実行資格**：さらに実Host Grant、Source/Provider/Artifact Revision、対象Entity版、Effect identity/receiptを照合する。Owner受理済みでも古いGrantや未接続Providerでは実行不可。

Graph編集を「編集可能」と表示する条件は、Projectionがlosslessであること、対象Source/Scopeが特定できること、現在の編集Grantがあること、変更をSourceへ反映できることを**別々に満たす**必要がある。Graphが見えるだけではEditableではない。

### 24.3 高度なAI Harnessの安定した対象指定

対象選択をUIの一時ハイライトから独立した**Selection Context（読み取り専用の対象指定値）**として扱う。GUIで選択したNode/Edge/Range/Graph/Document/Relationと、CUIで指定したqualified refは同じ対象表現へ正規化する。

- Selection Contextは複数対象、集合、除外、Source範囲、Snapshot、Revision、Consumer Scopeを表現できる。UIは選択を固定/解除する明示操作を提供する。
- AIへの送信時は「対象」「依頼」「参照文脈」「変更可能範囲」「禁止事項」「期待する証拠」を別々に渡す。大きな選択集合は対象数/未解決数/非開示数を表示し、暗黙の切り捨てをしない。
- AI応答は**Proposed Patch/Document Edit/Action Plan**に分類する。どれも自動的なOwner意味受理やHost Grantではない。採用操作はSourceの現在性と差分を再照合する。
- 複数製品間でSessionを共有しても、ConsumerごとにRead/Disclosure/Write/Effect権限を評価する。DSH・Codex・Claude Code等のハーネスはAdapterを通じて同じ選択契約へ接続する。
- 選択固定後に対象が削除/移動/改版された場合は、候補の再解決を提示しても自動的に別対象へ編集しない。曖昧・欠落・非開示は区別する。

### 24.4 実行状態の直交性

**Graph Definition**（どの経路があり得るか）、**Interaction State**（何が確定したか）、**Business State**（何が成立したか）、**Execution/Effect State**（何を実際に呼び出したか）を分離する。

ChatのStream delta、Voiceのpartial ASR、HarnessのTool提案、B2CのExperiment exposure、SHOの文章変更は、それだけではBusiness Operation/Effectの確定を意味しない。Resume/Regenerate/Branch/Timeout/Cancelでは、旧Operation/Effectの結果がunknownの場合に「未実行」と捏造しない。実行中のDefinition改版は旧版継続・明示Migration・拒否のいずれかを対象Scopeで決める。

### 24.5 検証の優先順位と判定方法

1. **最小の実証対象**：既存Operation一件、Provider B、CLI/Studio二Consumer、Node/Edge IDが重複表示名に依存しないread-only Model Map。#1251の実Call証拠が先。
2. **同じIndependent Expected**で、元のOperation、Graph投影、Validation Patch、条件分岐Patchを比較。構造が同じでもProvider違い・Effect欠落なら不合格。
3. **図面とCUIの二経路**で同じEdge変更を提案し、Source/Revision/Owner受理/Target生成への帰還を比較。Layout-onlyはSemantic digest不変。
4. **AI Chat/VUI/B2C/SHO**は各々の固有Event/Representation/Source契約を別検証し、CLIのPASSから全媒体へ外挿しない。

**追加の反例観点（既存LGX-17〜24に付帯）**：大量Selectionの一部が未取得、Graph同名多重Edgeの誤編集、Graph Patch順序非可換、Owner未受理の条件変更、実行中Grant失効、同名異内容Source、Plugin版A→B→A、選択Bと実Invoker A、Chat partial ToolがEffectを起こす、B2C非関与Routeへの必須JS混入。これらを#1180/#2514の独立Expectedへ還元し、第二テストSSOTを作らない。

**採否停止線**：第二正本/第二Auth/第二Resolverが必要になる、AI候補をOwner受理と同一化する、静的UNKNOWNをPASSにする、未実行Targetを適合済みと表示する、Sourceへ帰還できないProjectionを編集可能と称する。この場合は設計を増築せず既存契約へ戻す。

## 25. 実装引渡し：Selection Context・Patch・Execution Evidenceの最小縦断契約（2026-11-12）

§23〜24の設計を実装可能な単位へ分解する。**新しい公開APIや正式Schemaの採用ではなく、既存ContractへのFit/Gapを調べるためのReference**である。#1124（Modeler）、#3262（Source Context）、#3503（Lifecycle）、#1251（Provider実Consumer）、#1180/#2514（Scenario/Expected）をOwnerとし、二重の正本・Resolver・Hostを作らない。

### 25.1 Selection Contextは「表示選択」ではなく変更対象の契約

最低限の情報を以下に分離する。

| 区分 | 必須情報 | 禁止する解釈 |
| --- | --- | --- |
| Identity | qualified Subject ID、Kind、**Sourceで裏付けられるRelation occurrence ID**、複数対象の集合演算 | 表示名・行番号・座標・描画専用Edge IDを業務Relation IDにする |
| Provenance | Workspace、World/Snapshot、Source URI/Span、Revision、fingerprint | 同名Source・同bytesの再出現を無条件に同一資格とする |
| Scope | Consumer、選択時の可視範囲、未取得/非開示/曖昧の件数 | 見えた対象だけを完全集合と主張する |
| Intent | read / explain / propose / edit / invoke の要求種別 | 選択操作だけでWrite/Effect Grantが成立したとする |
| Lifetime | 固定時点、期待Revision、再解決方針、失効理由 | セッション存続で対象が常に現行とみなす |

**権限はSelection Contextへ永続Grantとして埋め込まない**。選択時のScopeは診断用であり、実際のRead/Write/Effect時にHostが現在権限を判定する。固定選択の解除は明示操作とし、旧対象が消失した場合は自動で近傍の別対象へ付け替えない。

**Focus／一時Selection／Pinned Target Set／Activity Context／Harness Sessionは同義ではない**（#4479/#4515）。最初の製品縦切りはWorkbench内のread-only Subject Pinに限り、元のSubject kind/ID/Workspace、現在のInspect結果、現在資格の不明点を保持する。現在のGraph描画が作る`parent:`形式のEdge IDは意味Relationの識別子ではないため、Sourceに裏付けられる正式なRelation occurrence IDが得られるまでRelation Pinをunsupportedとする。Pinned Targetを保持できたことだけで、Source/DB Scope/Disclosure/Grantの現在性や外部Harnessへの安全な送信まで証明したことにはしない。

### 25.2 Proposed Patchから実行までの状態遷移

~~~text
Selected Context
  -> Proposed Change (AI / Human)
  -> Structural Validation
  -> Owner Meaning Acceptance [業務意味を変える場合]
  -> Source Commit / Revision
  -> Effective Graph + Plugin Lock
  -> Target Artifact + Build Evidence
  -> Host Authorization + Actual Invocation
  -> Outcome / Effect Receipt / UNKNOWN
~~~

- **構造のみの変更**：layout-onlyはSource業務意味・生成Artifactを変更しない。単なる表示設定も対象・Scopeの可視性検査は必要。
- **意味を変える変更**：Edge/Guard/Operation/Relationの変更は通常開発の変更管理で判断した対象と範囲を示し、未確認のAI候補ならSource候補として保留する。編集者が手作業で確定した変更まで無条件に専用の再承認フローへ流さない。
- **実行中の変更**：旧Attemptを黙って新Graphへ移さない。継続、明示Migration、拒否を区別し、既存Effect receiptとunknown commitを保持する。
- **外部参照**：Source move/content change/permission-denied/unavailable/partial/not-foundは別状態。A→B→Aでも旧資格を自動復活させない。

### 25.3 実Consumerを通した三つのReference Slice

| Slice | 入力 | 実証する出口 | 主Owner |
| --- | --- | --- | --- |
| S0: Inspect | 実Workspace、選択Provider B、CLI/Studioの同一Subject | 選択B=実Invoker B、実Consumer受領、Source/Attempt一致 | #1251 / #1824 |
| S1: Modeler Selection | **第一段は実WorkbenchのSubject Pin**（kind/ID/Workspace）。第二段でSource裏付けのあるRelation occurrence・同名多重Edge・AI提案を扱う | 第一段はFocus/IME/Canvas移動でもPin維持・旧応答拒否。第二段では描画Edge IDを拒否し、同名多重Relationを混同しない。いずれもOwner未受理ならEffect 0 | #1124 / #1824 / #4515 / #3262 |
| S2: Graph Patch | 既存OperationへValidation追加と条件分岐、同一Independent Expected | 合成Graph、Source帰還、Target実Artifact、全Path Guard/Effect | #3503 / #3278 / #1121 |

S0未実証の段階でS2の実Provider parityをPASSとしない。S1のread-only表示は編集・実行資格の証拠ではない。S2でTargetを実行しない場合はDesign/Buildまでの限定PASSに留める。

### 25.4 Evidence Joinの最小照合キー

**同じログが存在することと、同じ実行に由来することは別**。Evidenceを突き合わせる際は、少なくともWorkspace/World/Snapshot、Source revision、Operation/Attempt identity、Consumer/Target、selected Plugin/Provider Lock、Artifact revision、Host authority context、Effect identity/receiptを用途別に照合する。全部を一つの巨大IDに押し込まない。取得不能な項目はUNKNOWNとして示し、類似するタイムスタンプや表示名で結合しない。

### 25.5 受入票と失敗時の報告

各Sliceの報告には「Source実体」「選択Provider」「実Invoker」「Consumer受領」「Independent Expected」「Effect/Receipt」「Target coverage」「実行SHA/環境」「未検証事項」を記載する。局所合成テスト、CI成功、実Workspace実行、Owner意味受理は別欄とする。

**採用判定**：S0で実Consumerが成立し、S1で安定Selection/Source資格が成立し、S2で少なくとも一Targetの実生成・実行と全Path Obligationが成立した場合に、Graph編集→実Targetの限定縦断を主張できる。Chat/VUI/SHO/B2C全Targetの適合はその後に個別検証する。

**状態**：本節は設計と受入票の追加のみ。S0/S1/S2の実行、Target生成、Owner受理、CI検証は本変更ではNOT RUN。

## 26. S0〜S2実装バックログと統合判定（2026-11-12）

§25のReference Sliceを、**成果物・独立Expected・実行証拠・停止条件**で検収できる最小作業へ分解する。これは既存Issueへ割り当てる設計引渡しであり、新たな製品Gate、第二Scenario SSOT、独立Workflow Engineを作らない。

| 順序/既存Owner | 作業と成果物 | 必須の反証・受入証拠 | 不合格・保留条件 |
| --- | --- | --- | --- |
| A / #1251 #1824 | 通常CLIとStudio Bridgeが選択済みInspector Provider Bを実呼出しする配線、同一Subject/Attemptの結果表示 | B選択→B実Invoker→CLI/Studio受領、BをAに差し替えた負例、Provider error/Grant失効 | 既存Direct実行や局所Go呼出しだけ、実Bridge未接続 |
| B / #1124 #1824 #4515 #3262 | **B0 Subject Pin**を現行WorkbenchのInspectに接続。**B1 Relation Pin**はSource裏付けのあるoccurrence IDが利用可能になってから検証 | B0: Focus/IME/Esc/Canvas移動、別Workspace同Fingerprint、後着旧応答。B1: 同名多重Edge・CRUD Relationの曖昧拒否。各段でRevision/Disclosure未知を明示 | 描画専用Edge IDを意味ID化、Pin意図を現行Grantとみなす、B0成功だけでB1/外部HarnessをPASSにする |
| C / #3503 #3278 #1121 | 最小Validation Patch→Effective Graph→Source帰還→一Target生成、次に条件分岐 | 全Path Authority/Effect、未受理Patch非反映、Patch順序競合、layout-only生成不変 | Guard迂回、Owner未受理、静的UNKNOWNをPASS |
| D / #1180 #2514 #259 | A〜Cの同一Independent Expectedを既存Scenario/Conformanceに収容 | 実行SHA・Workspace・Source・Provider Lock・Artifact・Attempt・Receiptの突合、対象外Targetの明記 | 他PR/別SHAのCIや合成テストを製品実行証拠へ転用 |
| E / #3503 #3509 #1124 | Chat/VUI/SHO/B2Cの異質Consumerへの限定接続 | partial tool/ASR=実Call 0、SHO同名異内容、B2C非関与Routeに必須JSなし | CLIでの合格を全Targetの合格とする |

### 26.1 実装時の選択肢を閉じないための判断

- Selection Contextのシリアライズ方式、UI保持場所、Harnessへの転送形式は**未採用**。まず同一対象を二Consumerで失わずに扱えることを優先する。公開ABIや新Core型の導入は必要性が立証された場合に限る。
- Graph Patchの記述は既存標準データ表現を第一候補とし、Mermaid/Graphviz/Markdown/GUIはProjectionまたは糖衣として扱う。Layoutと業務意味の差分は別ハッシュ/失効規則で検証する。
- 非同期実行・Wait/Resumeを扱う際は、Graph定義版、Attempt、Business Entity、Grant、Effect Receiptを分離する。新しいTEI常駐Hostを前提にしない。
- Pluginの未選択実装の故障や、B2CでTEIが関与しないRouteは、無関係なConsumerを停止させない。ただし当該Scopeの必須安全義務を省略してよいという意味ではない。

### 26.2 完了報告テンプレート（各作業共通）

| 項目 | 記載内容 |
| --- | --- |
| Scope | Workspace/World/Source Revision、Consumer/Target、対象Operation/Subject |
| Build | commit SHA、Artifact/Generator/Plugin Lock、実行環境 |
| Evidence | Selected / Loaded / Invoked / Received / Effect のうち実際に観測した段階 |
| Expected | 既存Scenario ID、独立Expected、正常/異常/NULL/Absent/部分集合 |
| Result | PASS / FAIL / UNKNOWN / NOT RUN（未実証のTargetを明示） |
| Authority | Owner意味受理、現在Host Grant、Disclosureをそれぞれ記載 |
| Follow-up | 再現手順、失効要因、次のOwner、Blocker |

### 26.3 実装の着手条件

Aは既存Direct Baselineを破壊せず、実Provider B呼出しが可能な最小Consumerを選ぶ。Bは実Source Indexのstable identityを利用し、既存Model Map M0のread-only範囲を超えた編集資格を主張しない。CはA/Bの証拠を参照できる場合のみE2Eの限定受入候補とし、そうでなければCompiler/Projection単体検証と明示する。DはA〜Cと並行してExpectedを整備できるが、実行結果を先取りしない。EはA〜Dの結果を外挿せずConsumer固有の契約を再検証する。

**この追補の状態**：バックログ/受入条件の設計のみ。コード変更、CI、実Consumer、Owner受理はNOT RUN。作業順序は依存関係であり、実装完了宣言ではない。

## 27. テーブル定義とUI項目の連動：明示的な構造変更と安全な逆写像（2026-11-12）

**利用者が明示した操作原則を具体化する。** 本節は、業務開発で選択した**三要素分析法**（データモデル・機能モデル・業務モデル）による表現に加え、TEIの汎用Source/Mapping/Projectionにも適用する変更連動を扱う。三要素分析法を汎用Coreの必須モデルとはしない。データモデル内の**テーブル／列の定義表示は、権威あるテーブル定義の写像**とし、図面を第二の定義正本にしない。業務キー・Relation・責任の意味は、物理PK/FKやテーブル定義の事実から自動受理しない（#1126/#4120）。ここでいう「テーブル定義」は選択したSource上のDesired/Declared構造であり、実DBで現在観測できたActual Schemaや未適用Migrationと混同しない。

### 27.1 編集操作別の既定動作

| 操作 | データモデル・Function/UIでの既定動作 | 別途必要な明示操作・確認 |
| --- | --- | --- |
| UI項目の追加 | UI要素だけを追加。対応する既存項目があればマッピング候補を提示してよい | **テーブル列は自動追加しない**。新列が必要なら対象テーブル・型・NULL/Default・制約を確認し、別の構造変更として明示 |
| 既存のテーブル列をUIへ割当 | Source裏付けのある一意な項目だけMappingを設定。複数候補は未解決 | 対象の明示選択。編集/更新の場合はFunction/Operationと権限を別途検証 |
| UI項目の削除 | UI定義／そのMappingのみ削除。データモデルの列は残す | テーブル列削除を意図するなら別の構造変更が必要 |
| **権威あるテーブル定義から列を削除** | そのテーブルのデータモデル写像から列が消える。対応UI要素は残し、**有効な項目MappingはUI定義から除去し、未マッピング警告を表示**。旧対応は監査・影響調査用の履歴に限る | 関連Query/Function/CRUD/Constraint/Generatorへの影響を列挙し、MigrationのPlan/Applyは別実行 |
| テーブル列のrename／split／merge | Source identity・旧新対応・依存の変化を検査。曖昧なら旧Mappingを自動付替えしない | 新しい対応付けの明示受理。単なる同名・近似型の一致から復活させない |
| 実DBだけの外部Schema変更 | Actualとの差分・鮮度不足として観測。権威あるSource上の列定義やUI Mappingを**黙って消さない** | Source/Desiredへの採用、Migration/逆取込み等は別操作・別資格 |
| Schema変更のPlan／Apply | 設計差分・DDL候補が出ても、実DBの変更完了を主張しない | Provider能力・対象環境・現行Grant・破壊性・データ損失・実行結果・再観測を照合（#3270） |

**一操作で複数の変更を提案してもよいが、UI追加、テーブル定義変更、DB適用は別々の効果である。** ワンタッチの作成支援は許す一方、ユーザーが画面項目を配置しただけで列を新規作成したり既存DBを変更したりしない。明示確認済みの複合操作であっても、Source Commitが成功したこととDB Migrationが適用されたことは別のResultとして扱う。

### 27.2 削除後の整合状態と操作可能範囲

- **Source／Model**：列削除を権威あるSourceへ確定した時点で、対応するテーブル列のモデル写像は最新定義から除去する。ただし過去のSource Revisionや実DB Observationの履歴まで消去したことにはしない。
- **Function／UI**：対応していたUI項目を消さない。有効なMappingを外し、未マッピングUI項目として残す。旧Mappingの根拠・出典は履歴や診断に残せるが、実行時のBindingには用いない。描画可能性と値取得・保存可能性を分ける。列がないのに旧値を表示することや保存先を推測することは禁止する。
- **CRUD／クロスリファレンス**：列単位のCRUD参照、Function Input/Output、Query Lineage、依存Rule、テストFixtureを再評価する。Unresolved/partialを「依存なし」に読み替えず、影響を確認できない場合は変更適格を保留する。
- **編集／実行**：未マッピングUI項目の表示・編集可能性は分離する。画面上で入力できても、対応先が未確定なら旧列・別列への書込Intentは生成・実行しない。既存コード、外部Host、手書きの処理には独立した影響確認を要する。
- **選択の固定**：列を参照するPinned Targetは旧Intentの記録として残り得るが、失効後に似た列へ自動再束縛しない。現在のSource/Schema/Scope/Host Grantを再資格化する。

### 27.3 同じ操作をGUIとCLIで説明・検証する

製品上のコマンド名・JSON Schemaはこの段階で固定しない。既存のModeler/Studio/Data Source Workbench/Source編集契約を使い、次の**意味上の操作**を共通化する。

1. 対象と現在版を明示して編集提案を作る（UI-only / Map-existing / Schema-desired-change / Schema-apply）。
2. 影響範囲を取得する。Source Index、三要素図、CRUD/Query、UI Mapping、Migration Planから、解決済みと未知を別に返す。
3. 明示した対象・変更範囲に対してSourceの現行性、構造妥当性、必要な意味受理を確認する。
4. Sourceへ反映し、UI Mappingを再評価した結果と、実DBへ適用するためのPlanを**別々に表示**する。
5. DB Applyを指示した場合だけ、外部Provider/Host権限の下で実行し、結果を独立に再観測する。

CLIでは非対話操作も、承認済み変更の明示的な入力と対象版の指定を必要とする。確認UIがないことを自動作成・自動削除の理由にしない。通常コーディングを排除しないため、既存コードとの対応や未追跡依存は診断対象とし、モデル変更だけで既存処理まで更新済みとしない。

### 27.4 独立Expectedへ渡す反例（設計候補・NOT RUN）

| Case | 条件 | 必須結果 |
| --- | --- | --- |
| DMU-01 | UIへ未マッピング項目を一つ追加 | UI要素+1、テーブル列+0、実DB DDL 0 |
| DMU-02 | UIから「新しい保存列」を明示要求し、Schema Source確定前に中断 | テーブル定義変更0、実DB Apply 0。候補のみ保持 |
| DMU-03 | 共有列Cをテーブル定義から削除し、二つのFunction UIがCを使用 | 最新データモデル列C=0、UI要素2件は維持、有効Mapping 0・警告2件、旧列への書込0 |
| DMU-04 | Source上の列削除後、DBへMigration未適用 | DesiredとActualの差を保持。物理列削除成功・実Commitを捏造しない |
| DMU-05 | 外部DBで列削除、Source/Desiredは変更なし | Schema driftを警告。SourceのModel列・UI Mappingを無音で削除しない |
| DMU-06 | 列renameに見えるdelete+add、同名／同型・Source A→B→A | 旧Mappingを自動復活させず、Identity/Revisionを再検査 |
| DMU-07 | partial Catalog／非開示Relationに依存する列変更 | 影響「なし」やSafe Applyと断定しない。未知範囲を明示 |
| DMU-08 | 描画Edge IDしかないUI／Pinned TargetからRelation列を更新 | 意味Relationへの書込0。Source裏付けのあるIDと編集資格を要求 |
| DMU-09 | Schema Apply拒否／部分適用／結果不明 | Source Commit・Provider呼出し・DB Observation・Effect Receiptを別状態にし、成功と捏造しない |

これらは**既存#1180/#2514のScenario・独立Expectedへの追加候補**であり、新しいテスト正本ではない。GUI・CLIの振る舞いを同じExpectedで比較する。ただし実DBの適用試験はProvider/権限の独立証拠が必要であり、合成テストのPASSから外挿しない。

### 27.5 Ownerと実装順序

- **#1126 / #1143**：テーブル定義と論理・物理Mapping、rename/delete影響、三モデル間の変更資格。
- **#1124 / #1824 / #4515**：実Sourceに裏付いたModeler表示、read-only Subject Pin、未マッピングUIの警告。描画EdgeをSemantic Relation IDへ昇格させない。
- **#3270 / #4120**：Desired/Actual差分、Schema Plan/Apply、外部変更・実DB観測とSource資格。
- **#2500 / #2497 / #2586**：Function UIの表示と、実データ更新Intent、read-only Rowset受領、写像喪失・部分集合を分離。
- **#1180 / #2514**：DMU-01〜09を既存Expectedへ結合。**実Provider Read・Plugin CLI/Studio P0をブロックしない**。

着手順は、(a)現行Sourceからテーブル列とUI Mappingの**read-only**な依存表示、(b)削除時の未マッピング診断・提案差分、(c)GUI/CLIの同一Source編集資格、(d)明示的なDesired Schema変更、(e)実DB Applyの独立検証。正式API・Migration Engine・承認基盤・Graph正本の新設は、この設計だけを理由に行わない。

**状態**：設計と検証候補の追補のみ。GUI/CLI編集、Migration、DB再観測、DMUの実行、Owner意味受理はすべてNOT RUN。

### 27.6 テーブル変更からEntry/Trigger/実行までの影響資格（#4502/#4495への引渡し）

第27節の「列・UIの変更」は**モデル編集の操作**である。実行に向けては、Function/Script Source、型付きOperation、公開Entry、Trigger Binding、実際にLinked/LoadedされたArtifact、実行Attemptまで、**同じ変更がどの依存を無効化するか**を別に確認する。Scriptの存在とEntry公開、Triggerの存在と実起動、Job受付と業務Effectを同一視しない（#4502/#4495）。ここでは起動器・Scheduler・Effect Engineを新設せず、既存契約に要求する**依存の追跡とAdmissionの停止条件**だけを整理する。

| 変更・観測 | 変更後に照合する対象 | 断定・自動操作してはいけないこと |
| --- | --- | --- |
| UI項目の追加・削除 | UI Mapping・Form/Page・関連FunctionとEntryの入力要件への影響 | UI追加から列・Script・公開Entryを自動作成、UI削除だけで公開Entryを一律削除 |
| Sourceのテーブル列削除／rename | Data Modelの現行列、全UI Mapping、Queryの参照列・`SELECT *`・`NATURAL JOIN`等の暗黙列集合、CRUD/Rule/Script入力、生成Artifact | 旧Function/Scriptの修復先を同名・同型で推測。非出力列を「依存なし」と判断 |
| SourceのFunction/Operation契約変更 | Source/Contract Revision、必須引数（missing/NULL/unfetched）、公開Entry、Triggerの引数束縛、Linked/Loaded Artifact | Functionが存在するだけで起動可能、古いEntryやArtifactを現行扱い |
| Actual DBだけの外部Schema変更 | Actual/Desiredの差分、取得範囲/時点、Provider/DB alias、Query/Field Mapping、次のInvocationの現在資格 | PartialなSchema観測から不存在を確定、Source/Desiredを無断変更 |
| Triggerの改版・停止／Grant失効 | EntryとTarget、予定/配送Event、現在のAdmission、実Attempt/Effect | Trigger停止＝既存Effect取消、以前のGrant＝現在権限、設定済み定時刻＝実Run成功 |
| Source/Provider/Artifactの差替え（A→B→Aを含む） | Selectionの起点、Revision、実Loaded bytes、選択Providerと実Invoker、Attempt/Effect | 元のFPに戻っただけで旧実行資格・旧選択・旧承認を復活 |

**影響検出は依存の確実性を表示する。** Source上の直接参照だけでなく、Queryの列集合依存、式・Rule、手書きコード、外部Source/Providerに渡したMappingの未観測・曖昧さを区別する。依存を全件確認できない場合、影響「0件」ではなく`partial/unresolved`とし、影響するOperationの次回起動は必要な条件を再確認する。一方、関係のないOperation/Plugin/Consumerを無条件で停止させない。

**保存から実実行までの段階**：①Source変更候補と影響一覧→②Source確定とモデル/UI再投影→③必要なFunction/Entry/Trigger契約の再評価→④Artifact生成・実ロード版照合→⑤Invocation時のTarget/Subject/Current Grant・Source/Artifact/Provider版の再照合→⑥実Attempt/Effect Receiptの独立観測。①〜④を終えても、⑤の現在資格や⑥の実行結果は成立しない。逆に既に確定したEffectは、後からSource/Entry/Triggerを削除しても取り消されたことにならない。Queued済み/実行中/ACK不明のAttemptは、#1174/#4502の既存Recovery・Cancel契約に委ねる。

**検証の最小交差点**（新しいテストID・第二Scenario正本を作らない）：

1. **DMU-03/04 × #4495 EL-03**：二画面で利用している列をSourceから削除。UI要素2件は保持、有効Mappingは0。依存Script/Entryが旧Contract/Artifactを使う起動要求は拒否。実DB列が旧まま残っていればDesired/Actual差を維持。
2. **DMU-05/07 × #2586 HJ-01〜04**：外部DBの列変更またはpartial取得。Schema情報の不明点を保持し、Query/Contextの列・grain・完全性を確認できないClaimを確定しない。
3. **DMU-06/08 × #4515 PIN-S0**：同名列再作成、Subject/Source A→B→A。固定UI/AI対象の旧Intentは残せても現在資格を復活させず、描画Edgeから業務更新先を推測しない。
4. **#4502 N02/N06・#4495 EL-07 × DMU-03**：Trigger発火後にSource/Grant/Artifactが失効した場合は実起動時に拒否。Commit後のACK喪失ならEffectを`unknown`として保持し、再送で二重実行しない。Trigger停止・取消をRollback成功と呼ばない。

**Ownerの範囲**：Data/Logical/Physical変更は#1126/#1143、Workbench/モデル編集は#1124/#4515、SQL/Query依存は#2387/#2586、UI逆写像は#2500/#2497、Entry/Trigger/Invocationは#4502/#4495、Actual DB/Migrationは#3270、Effect/Recoveryは#1174/#1129、独立Expectedは#1180/#2514。先行する**P0実Read/実Provider呼出しは継続**し、上記交差ケースの実Target実行は各担当の証拠が集まるまで`NOT RUN`とする。

## 28. 方法論プラグインと通常開発の受理：Coreに固定しない協業契約（2026-11-12）

### 28.1 三要素分析法は重要な標準構成、しかし唯一の方法論ではない

TEIは汎用的なプログラム・データ・文書・業務システムの構築基盤であり、**三要素分析法そのものをCoreの必須モデル構造や実行時依存にはしない**（#1124/#2620/#2621、組織モデルの別Draft #4500 §10.7）。Core側で必要なのは、既存の標準データ構造・型/Facet・Source/Revision/qualified reference・Projection/Mapping・Operation・Plugin選択・診断の共通能力である。Data/Function/Business、イベント付き業務フロー、ER/CRUD/職務/アクションツリーの原典に忠実な表現は、**業務向けMethod/Profile/Pluginの組合せ**で検証する。汎用GraphやFunction計算のRoleを揃えただけで原典を再現したとは主張しない。

通常のコーディングはCore上の正規の開発経路として残す。業務開発でデータやFunctionの定義・整合確認を行うことは原則重要だが、**開発者に図面エディターの使用や全三モデルの作成を一律に強制する意味ではない**。ドメイン専門家が画面・業務手順・文章・例を入力し、開発者がKey/Relation/制約/データ移行などの高度なモデリングを支援し、生成AIは既存定義の調査と候補作成を担う。責務の境界は実案件の通常開発の分担に従う。

| 利用形態 | 期待する利用条件 | 必須にしないもの |
| --- | --- | --- |
| 汎用コード中心 | Source/型/Plugin/既存Testに接続し、必要に応じ標準UIを使う | 三要素のモデル一式・TEI専用承認 |
| 業務向け三要素分析法 | 原典の図法・相互参照・CRUD・Source往復をMethodとして追加 | 三要素専用の第二Core/Graph正本 |
| 業務向け簡易データ連動 | UI追加と既存項目割当を支援し、構造変更の効果を明示 | UI追加からの列作成・DB DDL暗黙実行 |
| 異質Domain/Target | 対象固有Pluginで必要なRole/Projection/Targetを構成 | Business Stateやテーブル必須化 |

### 28.2 業務意味の確認は通常開発相当、AIによる事実誤認を局所で防ぐ

§23〜27に記した「Owner意味受理」は、**通常の要求確認、設計レビュー、差分確認、Pull Requestレビュー等の作業に相当する**。Graph/Mapping変更一件ごとに独自の多段階ワークフローを新設することではない。業務上の意味を判断する必要がある部分と、単なる構文変更・配置変更・Refactorを区別し、既存の手続きと受入対象に基づいて確認する。

AIが推測したMapping/Relation/Rule/Columnを「発見した事実」として自動反映しないため、必要な場合は、候補の対象・出典となるSource/DocumentとRevision・推論内容・適用範囲・反例・未解決事項を通常の変更差分や既存Authoring資料から辿れるようにする。**OKFはその記述・検査の一候補**（#4143/#2621）であり、すべての変更に新しいEvidence台帳/形式の必須記載を課さない。OKF Readerが形式を解釈できたことは、Meaning/Grant/Effectの受理を意味しない。Sourceに出典のないAI候補は、採否判断まで未解決として扱う（#267）。

これは「実行時権限」の簡略化を意味しない。Source編集の通常レビューと、実DB Schema Apply／業務更新／実Effectにおける**現在のHost/DB権限・実行対象の確認**は別の責務である。

### 28.3 実装に渡す追加反例（既存Expectedの補強）

| Case | 入力・条件 | Expected |
| --- | --- | --- |
| DMU-10 | UIに計算式・外部API・一時入力項目を追加 | UI/Functionの表現は可能。永続テーブル列+0、暗黙DDL 0。更新可能性は別Operationと対応先で判定 |
| DMU-11 | 業務Method Pluginを未選択にした一般コード/データ構造のSource | 汎用inspect/Source navigation/BuildがMethodの不在だけで停止しない。三要素図はunsupportedでもCore実行へ波及しない |
| DMU-12 | AIが自然言語から新Relation/Columnを推定、原資料には裏付けがない | Candidate/出典不足の診断。既存Source列+0、Accepted Meaning変更0、実DB Apply 0 |
| DMU-13 | 同じSourceから原典方式に近い業務モデル図と汎用Graph投影を生成 | 図法・CRUD・対象IDの情報欠落/追加を比較表に記載。表示類似だけで原典互換やMeaning同一と断定しない |
| DMU-14 | 通常のコード修正のみ、またはUIレイアウトのみの変更 | 必要な型・依存・テストは検証するが、無関係な三要素/OKF/業務Ownerの追加承認を要求しない |
| DMU-15 | Code/Sourceの変更提案がある一方、OKFなど参考資料は stale/partial/unavailable | 参考資料の欠落状態と、対象Sourceの現在性を別表示。旧文書のAI要約でMappingを無断再確定しない |

DMU-01〜09、他DraftのORG-MAP-01〜10（#4500）と期待値・入力を照合し、重複するテストは**#1180/#2514の既存Scenarioへ統合**する。方法論の不在で一般コードが動かない負例は#2620/#951、AI候補の採否は#267、OKF形式Readerは#4143、三要素の原典再現は#1124/#1031がそれぞれOwnerを保持する。

**計画への位置付け**：P0の実Provider・実Read（#1251/#2586）を変更しない。並行P1では最小のSource→Data/UI Mapping→未マッピング警告をGUI/CUIで確認し、方法論Pluginの有無と既存コード経路を対照にする。原典への忠実度評価・OKFを用いた資料管理は独立に段階導入する。今回の追補はdocs-onlyであり、DMU-10〜15の実試験・Owner確認・実DB Apply・実Targetは**NOT RUN**。


## 29. 問いに応じた高速参照移動と「未使用」判定の資格（2026-11-12、横断設計）

§23〜28を、X-TEAの項目レベル参照・被参照ジャンプ、TEI Studioの最小マウス操作、AIへの固定対象引渡しへ接続する。**既存Source/Model Indexを利用し、三要素分析法のMethod Plugin、図面Renderer、AI Harness Adapterが独立した意味正本・参照Resolver・常駐Hostを持たない。** 一般コード開発やMethod未選択Workspaceも基本的な参照操作を利用できるようにする。

### 29.1 「参照0件」と「未使用」は異なる命題

| 表示する判断 | 必要な資格 | 誤判定を避ける条件 |
| --- | --- | --- |
| 範囲内で参照N件を観測 | Workspace/Source Indexの版、typed relation、対象Scope/Reader Attempt | 観測件数、範囲、未索引・解決不能件数を区別 |
| 限定Scopeで参照なし | **探索対象の閉包**、欠落Source・未導出・Current Disclosureを評価 | 不完全なら「観測0件・未確認あり」。単なる0件を未使用としない |
| 変更しても影響しない | Query暗黙列集合、Script/Trigger/Entry、Artifact、Provider、外部Consumerを問いに応じて照合 | 未探索・非開示・依存未知はpartial/unknown |
| 安全に削除・改名できる | 変更影響、現行Source/Target、逆写像、現在Write Grant、必要な意味確認 | read-onlyの不存在判断をEdit/Apply資格へ昇格しない |
| 実業務で使われていない | 上記に加え必要なRuntime/DB/実Provider/外部Effectを期間限定で観測 | 静的索引のみでは判定しない |

一件の陽性Witnessで限定的な参照の存在を示せるが、不存在・全件・安全削除には問いごとのRequired Source/Relation/Consumer Closureが必要。非公開参照を表示して漏洩させず、Policyが許す範囲で**判定不能であること**を示す（#2516）。表示上の「0件」が非公開世界の不存在を推測させないようにする。

### 29.2 Locate・References・Pin・実操作の縦断

1. **Locate → References**：qualified Subjectを既存Source Indexで再解決し、inbound/outboundのtyped relationを辿る。Source位置はjump先であってidentityではない。図面描画用Edge IDを正式Relation IDに読み替えない。
2. **Pin → Explain/Propose**：参照元・先の複数選択を固定し、Revision・Scope・部分集合・未開示・除外条件を保持してAIへ引き渡す。固定対象はGrantではない。PinのOwnerは#4479であり、§25に第二Storeを作らない。
3. **Preview → Source/Edit/Apply**：同じSubject/Source/Revisionの変更候補と影響範囲を確認。Sourceへ安全に帰還できない図表はread-only。編集採用とDB Apply/Host Effectは別。
4. **Verify → Observation**：MapとLocateが同一Fingerprintでも、別Load間の一致は原子的Readを証明しない。実Consumer受領、Host Invocation、Effectは各々独立に観測する。

**関連PRと資格（2026-11-12夜のIssue照合時点）**：[#4524](https://github.com/hat47x/tei-platform/pull/4524)は同Kind同IDの誤索引を防ぐ限定修正、[#4392](https://github.com/hat47x/tei-platform/pull/4392)はMap埋込Validation等の修正で、双方ともIssue上のPR状態は**closed**へ変化している。ただし**closedだけではmerged・現行mainの適合・CI成功を証明しない**。[#4521](https://github.com/hat47x/tei-platform/pull/4521)は実SourceのMap/Locateを限定比較し、旧HEADのValidation不一致／新HEADの検証範囲を区別している。受入担当は各PRの**merged_at、mainにあるBlob、exact HEADの試験、実ConsumerのRead**を独立に確認する。これらを一括して「Modeler完成」とは呼ばない。 旧X-TEAの空・欠落IDは一時Occurrenceとして読取・移動できても、永続Semantic IDや逆編集先を捏造しない（#1031/#4510）。

### 29.3 Studioの高速性を損なわない拡張境界

- identity、kind、source span、inbound/outbound adjacency、diagnostic/coverageは**既存Producerの読取Projection**として必要最小限で取得。Graph Layout、統計、広範な依存探索は必要時に計算し、全Plugin/全Sourceの同期走査を初期Navigationの必須処理にしない。
- 参照数・被参照数・未解決数の表示では**Index済み観測値と完全件数を区別**。非公開の生CountsはPolicy Owner #2516の判断を先行させる。
- 増分Index/Cacheの再利用はWorkspace/World/Source Revision、Generator/選択Plugin、Projection設定、Consumer/Disclosure Scopeを用途別に照合する。無関係なPlugin変更で無関係なIndexを失効させず、関係するA→B→A・Grant変更では現在資格を再確認する。
- キーボード中心に「参照へ移動」「被参照へ移動」「定義に戻る」「Pin/解除」「影響表示」「AIへ送出Preview」を一連の操作として設計。具体的なキー配列は衝突検証後に決め、IME・画面遷移・空白クリックでPinを解除しない。
- 重い言語Plugin/Method Plugin/DB Providerは必要な操作だけに参加する。Methodが存在しない場合も基本Locateを無効化しない。性能はcold/warm、初回Locate、連続jump、参照一覧、Pin→Previewについて**実Workspaceの規模・Consumer別**に測定し、実測前に一律のミリ秒基準を採用しない。

### 29.4 既存Scenarioへ戻す独立Expected候補

| Case | 入力と期待する限定結果 | Owner |
| --- | --- | --- |
| NAV-01 | 実SourceでSubject→参照→被参照→定義へ戻る。typed RelationとSource位置が一致 | #1124/#1120 |
| NAV-02 | 同Kind同ID重複。曖昧Subjectを一意に丸めずdiagnosticを表示 | #4475/#4524 |
| NAV-03 | 別Kind同ID・同label多重Edge。Qualified ID/Edge IDで別対象として追跡 | #1124/#1120 |
| NAV-04 | 観測参照0だが未索引Source、非開示/未導出依存がある。未使用の断定なし | #2516/#1124 |
| NAV-05 | Completeな限定Scopeで参照0を報告。Runtime未使用・安全削除へ昇格しない | #1124/#1180 |
| NAV-06 | 同FPでもMap/Locate Validation不一致。未実施/失敗を区別 | #4392/#4521 |
| NAV-07 | Legacy SourceでID欠落。Occurrence移動のみ可能で永続Writer先は未確定 | #1031/#4479 |
| NAV-08 | Pin中にSource A→B→A、別Workspace複製、Grant失効。黙って再結合しない | #3262/#4479/#2516 |
| NAV-09 | 無関係Method Pluginの故障/遅延。基本LocateとP0 Provider Consumerを無関係に停止しない | #951/#1124/#1251 |
| NAV-10 | 同一実Workspaceでcold/warm・連続jump・Pin Previewを計測。Scopeと負荷を別記 | #1124/#1824 |

これらは**#1180/#2514に渡す独立Expectedの候補でありNOT RUN**。#4479の固定対象UX、#4510のQuestion-bound STQ、#4487のTEA-Q、#4504のSource再資格と同じFixture/反証は各Ownerを再利用し、第二Scenario正本を作らない。

### 29.5 計画上の優先順位と停止条件

- **P0維持**：#1251の通常CLI/Studio実Provider Invoke、#2586のFunction→Query→実Rowset→Consumer受領。Navigatorを先行Hard Gateにしない。
- **P1a**：既存Map/Locate/Validationの整合と曖昧IDのfail-closed。#4392/#4524のexact-head CI、merge可否を各Ownerが確認。
- **P1b**：実Sourceでread-only Locate→references→jump→backをStudio/CUIへ一件接続。Coverage/未索引を示す（#1124/#1824）。
- **P1c**：固定Pin→AI PreviewでScope/Revision/Disclosureを再検査（#4479/#2516）。
- **P2**：Graph/CRUD逆編集、変更影響、Artifact/実Effectを§25 S2と#1143/#2497へ接続。
- **P3**：実Workspaceの性能・認知負担を測ってIndex/Cache/Plugin最適化を優先順位付け。

**撤回条件**：観測0件を無条件に未使用と表示、表示名・位置から編集先を創作、partialをcompleteとして表示、別Loadを原子的Readと呼ぶ、SelectionをGrantとして扱う、Navigation初期表示に全Plugin同期ロードを要求する、実測なしで高速化を主張する。いずれも限定read-onlyへ戻して再検証する。

**状態**：設計・バックログ・Expected候補のみ。新Navigation/Index/Pin UI、NAV-01〜10、性能実測、CI、現在Disclosure、Owner意味受理は**NOT RUN / NOT ACCEPTED**。


### 29.6 参照ナビゲーションとScript近傍テストの統合受入（#4525/#4502）

**実装で追加するのは第二のテスト体系ではなく、同じSourceから「どこを参照するか」と「何が実行されたか」を別々に観測する経路である。** #4525の近傍直接テスト（Rule/Operator/Function/Hook）と、#1180の複数Function/Driver Scenario、#4495の製品Entry/Invocation、#2586の実Query/Provider/Consumerは責務を分ける。以下は既存Expectedの**引渡し・照合条件**であり、新しい正式DSL/Runner/Oracle/Consoleコマンドを定義しない。

| 受入経路 | 必要な一致と観測 | 合格の上限 |
| --- | --- | --- |
| A：Source→Locate/Map→参照往復（§29 P1a/b） | 実Workspace Root/Source Revision、qualified Subject、Relation occurrence、Source range、Validation済みか、Coverage/Disclosure。別Loadなら別Attemptとして記録 | read-only navigationの限定合格。編集資格・実Function実行ではない |
| B：Source→Rule/Function近傍直接テスト（#4359/#988/#2620/#4525） | 近傍Source/Operator/Plugin版と独立Expected、入力のmissing/NULL/partial、**実際に参照したSource集合**。実行位置とテスト位置の一致だけを根拠にしない | 局所意味/実装特性の合格。Product Entry/Hostの資格ではない |
| C：Function→Entry/Trigger→Product CLI Process（#4495/#4502） | 実Entry公開版、型付き引数、selected Provider、Linked/Loaded bytes、実Invoker、Operation/Attempt、結果の観測。CLIと共通APIの二入口を同じMeaningへ接続 | 観測した一Target/Consumerの実起動。Trigger設定や擬似Executorだけで実Effectは証明しない |
| D：Query→Rowset→Function Data Context→Consumer（#2586） | 実Source/DB alias/Schema、Provider選択と実Read、Rowset grain/Scope/Completeness、Consumer受領と独立Expected | 実Readの一往復。参照0件、集計値、欠落から不存在や更新適格を推測しない |
| E：Operation→Host Effect/Recovery（#1174/#1129） | current Grant、実Commit/Receipt、独立Observer、ACK喪失と未知結果の扱い | 実行したEffectだけの受入。B/C/Dの結果を流用しない |

**同一Fixtureの条件**：A〜Eに共通の意味上のSubjectを使っても、異なるWorkspace Root、Model Revision、Source-only改版、DB/Grant/Plugin更新、Load時点を同一のObservationとして結合しない。Content FingerprintだけでSource Identity/Current Eligibilityを証明しない。直接テストのExpectedは、生成AIが作成した実装コードや生成テスト結果から逆算して確定させない。

**実装担当への直近の小さな引渡し**：
- #4392/#4524の**取り込み実態と現main実テスト**を確認したうえで、#4521の実Relation一件をAのread-only基線にする。已存の負例（同Kind同ID、Map/Locate Validationの不整合、同FP別Root）を流用する。
- #4495/#4525はB/Cのテストを受け持ち、Aで見える参照・出典を起動契約へ渡しても、**Mapに存在する＝公開Entryあり**とは扱わない。
- #2586のDは最優先の実Read主線。A/BのGUI・文書対応が未完成でも開始できる。
- Scopeを限定できる領域では独立した無関係Plugin/MethodのエラーでA〜D全体を止めない。ただし**問いに必要なRequired Closure**に欠落がある場合は、その問いの結果だけ`unknown/blocked`を残す（#2582/#4518）。参照数0や未索引0を「絶対未使用」の証拠にはしない。

**優先順位と停止条件**：P0は#1251/#2586と#4495の実Provider/CLI経路。P1aはMap/Locate/Validation/Source Indexの実現状確認、P1bは一件の参照往復、P1cは近傍直接テストとのSource/Scope照合。P2はPin→AI共有・逆編集影響、P3は異質Consumerと性能。追加の横断Test SSOT、独自Acceptance Engine、同期全Pluginロード、実証前のTarget普遍適合宣言は行わない。**本節はdocs-onlyでありA〜Eの新しい通し実行はNOT RUN。**

