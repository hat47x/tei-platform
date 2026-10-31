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
