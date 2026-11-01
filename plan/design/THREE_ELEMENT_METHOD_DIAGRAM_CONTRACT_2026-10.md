# 三要素分析方法論とDiagram Projectionの接続契約（設計候補）

- **状態**: 評価待ち・未採用。要求・ADR・Production schemaではない。コード変更・実行試験なし。
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

**決定候補B — Contract/Provider/Instance**: 論理Method、Capability Contract、実装Provider、配置されたPlugin Instance、配布Package/Artifactを分離する。同じRenderer Typeを異なるStyle/Workspaceで複数Instance化しても同じ意味体系としない。製品固有Core enum追加や第二Resolverは行わず[#951](https://github.com/hat47x/tei-platform/issues/951)へ接続する。tei.experimental.*等は[#2298 Draft](https://github.com/hat47x/tei-platform/pull/2298)の識別子**候補**であり正式な予約・実装済みではない。分散公開Registryは発見/第三者検証に限定し、Semantic Authority/Provider Conformance/Consumerの利用許可を与えない。

**決定候補C — Host/Target**: [#2226 Draft](https://github.com/hat47x/tei-platform/pull/2226)の「TEI独自Host原則非必須」を前提に、純粋なProjection/ValidationとUI Adapterを分離する。Wails/Theia/Browser/Golang processのいずれも必須基盤としない。ADR番号の衝突は[#2341](https://github.com/hat47x/tei-platform/issues/2341)へ委譲し、ここで番号や採否を確定しない。

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
