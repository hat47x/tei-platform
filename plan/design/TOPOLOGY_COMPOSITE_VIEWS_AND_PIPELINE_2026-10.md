# トポロジー・複合図表・開発パイプラインの共通表現（設計・実証計画）

- **状態**: 評価待ち（未採用）。2026-10-29の構想整理と検証設計であり、新しいCore型・Studio API・実効ビルド保証の成立を主張しない。
- **静的照合基準**: `ea293ae56a3817701ec0b8116ccc36978526367b`（main）。本文・既存Issue/PRの読取のみ。コード実装、Go/Node/GUI/実Provider試験は **NOT RUN**。
- **目的**: 人と生成AIが、構成要素の構造・関係・意味・実現・観測を混同せず、図表から変更箇所と不存在を判断できるための最小の共通表現・実証順を整理する。
- **正本の所在**: 製品要求は [`requirements/TEI_PRODUCT_REQUIREMENTS.md`](../../requirements/TEI_PRODUCT_REQUIREMENTS.md)、現行実装は [`docs/architecture.md`](../../docs/architecture.md) / [`docs/status.md`](../../docs/status.md)、採用判断はADR、受入と進捗は各Issue。本書はこれらの第二正本ではない。

## 1. 到達目標と原則（設計判断ではなく評価基準）

**主目的は抽象化の階層を増やすことではなく、人間と生成AI双方の理解・変更・検証負担を減らすこと。** コード量・画面数・図のノード数の削減は代替指標にすぎない。構造的な複雑性を除去できたか、別のブラックボックスへ隠しただけかを判定する。

- TEIの標準データ構造・標準プラグインは、図表化できる領域を広く受け止める**構造的・トポロジー的な語彙**を提供する。ドメイン固有の意味論をCoreの固定列挙へ増殖させない。
- **意味構造、表示構造、実行計画、実行観測**は異なる。関連付けは可能だが、表示位置から意味や実行順を推定しない。
- 標準データ構造の組み合わせで複合図表・変換グラフ・ビルド依存・メタ生成を表現する。新しい専用`Pipeline`正本、第二Graph/Resolver/Lock、Studio独自Acceptanceを作らない。
- 標準プラグインは汎用の投影・検査・変換・出力・編集補助を、領域プラグインは具体的な意味・制約・適合・表示最適化を、案件Overlayは固有の構成差を担う。
- 既存資産・外部ツールを排除しない。外部操作の内部は不透明と明示する。TEI独自Host、旧Wails/Theia、固定のMaven/Gradle DSLを前提にしない。
- **見せられないものを「無い」と判定しない。** 完全性の証拠がない図表、フィルタ済み図表、未解決Providerは不在の根拠にならない。

## 2. トポロジーと意味の責務境界

標準が取り込むべきなのは図の名称ではなく、複数領域に再利用可能な**構造上の区別**である。採用候補の能力分類は次のとおり。現行型の存在や追加を前提とせず、既存の型・関係・`modelref`・`composition`等との重複を評価する。

| 能力群（候補） | 識別する構造上の差 | 異なるドメインでの利用 |
| --- | --- | --- |
| 集合・多重集合・系列 | 存在、多重度、列挙順 | データ構造、工程一覧、値列 |
| 型付き二項／多項関係 | 役割、端点、対応、多対多 | ER、モデル間対応、関連グラフ |
| グラフ・ポート・境界 | 接続方向、端点、部分構造 | 依存図、データフロー、変換構成 |
| 木・包含・階層 | 所属、親子、入れ子 | 構成、スコープ、状態階層 |
| 順序・区間・時間 | 前後、区間、発生、観測時刻 | ビルド順、ガント、イベント |
| 座標・幾何・量 | 位置、距離、値と単位 | チャート、空間図、レイアウト |
| 写像・式・制約 | 入出力対応、条件、導出 | モデル変換、コード生成、検証 |

単なる「辺」の存在から、依存／実行順／因果／参照／変換を同一視しない。これらの**領域上の解釈**は、その関係が持つ型付き役割、制約、プラグインの契約によって決める。関係の向きと表示上の矢印も分ける。

「数学的なトポロジー」が数量・論理・時間の全てを包含するわけではない。図表を広く受け止めるために必要な汎用の値・制約・式を、構造的な能力の組合せとして検証する。最初から包括的な新Core型体系を設計しない。

### Complectを設計上の検査対象にする

次の混線を検査する。ただし「図が複雑」「ノードの次数が大きい」だけではComplectとは判定しない。

| 分離したい関心事 | 混線している兆候 |
| --- | --- |
| モデルの意味 / 図のレイアウト | 座標移動で受理済み意味が変わる |
| 接続 / 依存 / 順序 / 因果 | 線の追加だけで実行・権限まで推定する |
| 生成規則 / Target実現 / 実行副作用 | Target変更で無関係な意味規則を修正する |
| 構成時点 / 実行時点 | 版を固定したProvider解決を現在の実行Authorityとみなす |
| build-only / runtime dependency | ビルド器が実行時依存に無条件混入する |
| Source / Proposal / Accepted / Derived / Observation | 表示や生成・試験結果から正本を逆算する |
| 表示ID / qualified identity | 別scopeの同名ノードを単一対象としてInspectする |

**反証可能な簡素化判定**: 同じ機能変更について、理解に必要な正本・関係数、無関係な変更波及、診断不明点、修正・再検証範囲が減るか。抽象化で不透明な隠れた依存や文脈探索が増えるなら撤回する。

## 3. 複合図表：関連グラフ・対応表を一つの関係から導出する

一つのモデルを複数の図表へ投影するだけでは、次の二つを扱えない。

1. **同じqualified要素の複数表現**: 例として工程ノードと工程一覧の同一`Operation`。
2. **別要素の意味上の関係**: 例として工程`Operation`、成果物`Artifact`、その検証証拠`Evidence`の関係。

この二つを一種類の「図間リンク」として曖昧にしない。表示上の交差リンクと意味上の関係も区別する。

概念モデル（**新しい製品schemaの提案ではない**）:

~~~text
Canonical / Authoritative Source (ownerごと)
    ↓ typed mapping [evidence / unresolved を保持]
Standard structural relations (qualified identity + relation roles)
    ├─ Projection A → Diagram A, View-element IDs
    ├─ Projection B → Table B, Row IDs
    ├─ Projection C → Diagram C, View-element IDs
    └─ Correspondence projection
          ├─ Related graph: 構造要素・ビュー要素の関係を描く
          └─ Relation table: qualified subjects / roles / provenance / status
~~~

### 3.1 関連の区別

- **Identity correspondence**: 複数の表示要素が同一のqualified model keyを投影する。画面IDの一致からは導かない。
- **Semantic relationship**: 別々のqualified subjectがSource/受理済みモデルで持つ関係。意味づけはdomain側。単なる隣接や同時表示から生成しない。
- **Derived relationship**: 型付き規則・入力版・導出元を持つ関係候補。受理済み意味と区別する。
- **Navigation-only correspondence**: ビュー上の行、図、セクションを相互参照する表示上の導線。Canonicalのsemantic edgeへ昇格させない。
- **Evidence / trace relationship**: 生成物、実行記録、診断等への追跡。現在の権限・Ownerの受理と区別する。

関係が二者以上の役割を持つ場合は、無理に全組を二項エッジへ変換して意味を失わず、**多項関係を標準構造の合成**として表現できるか調べる。関係の表示を「関係ノード＋役割付きエッジ」に落とすか「対応表」に落とすかはProjectionの選択であり、意味の変更ではない。

### 3.2 関連表・関連グラフの共通入力

~~~text
Relation View（説明例。実APIではない）
- relation identity or derived origin
- role → qualified subject(s) [scope / kind / ID]
- relation type / interpretation owner
- source or derivation evidence / version
- unresolved / ambiguous / unsupported
- view occurrence mapping: (view ID, local node/row ID, projection revision)
~~~

関連表に行があることは、その関係が受理済みであることを意味しない。source由来・推論由来・未解決を表示する。図表どうしの対応を手作業で別の正本として維持しない。

**複合図表全体も投影**: 図A、表B、図Cを組み合わせたWorkspaceの構成（どの図を見せるか、折り畳むか、どこへナビゲートするか）はビュー構成であり、意味モデルの第二正本ではない。図を跨ぐ同一対象の選択・検索・変更影響はqualified identityと同じsnapshot上の対応に限定する。

### 3.3 図表そのものを結ぶメタ関係と、複合図表の被覆範囲

関連グラフは**図内要素同士**だけでなく、**図表自体をノード**として扱うこともできる。例: 「データ構造図が定義する型を、工程グラフが入力として参照」「工程一覧の選択に応じて成果物来歴図を開く」。ただし、図A→図Bという線だけから、元モデルのA→Bという意味関係を捏造しない。

- **図表間の関係**（共通対象を被覆する、別viewへnavigateする、型定義を参照する等）と、**図表内の要素間の関係**（OperationがArtifactを生成する等）は区別する。
- 図表間対応表は、view identity、projection snapshot、被覆しているqualified subject集合・関係種別、対応の役割・根拠・未解決を行として表す。行はView構成または導出結果であり、Sourceのsemantic relationではない。
- その表から「図表をノードとする関連グラフ」を生成でき、逆にグラフ上の選択から表の該当行・関係元のsubjectへnavigateできる。表示ごとに別の対応付けを保守しない。
- **複合図表の完全性は各Viewの完全性の足し算ではない**。View AがOperation、View BがArtifactを全件表示していても、両者の依存関係が完全に被覆されているとは限らない。対象scope・関係型・Source closureに対する被覆情報を別途検証し、partial/unknownを維持する。

これにより、多様な図表を組み合わせても専用の「複合図表言語」やUI正本を増やす必要がないか評価できる。詳細なView identity・被覆witnessのスキーマは、複数Consumerで不足を確認するまで固定しない。

### 3.4 編集可能性の非対称性

- 表示の移動・折り畳みはビュー設定を変更し、原則として意味は変えない。
- 構造の変更は、**唯一の正本の所有箇所**に対するtyped candidateへ変換し、影響・conformance・受理を必要とする。
- Derived関係、曖昧な多対多対応、read-only projectionは、逆写像またはOwner Commandが明示されない限り編集不可。
- Cross-viewナビゲーションの成功はSource書込み可能性ではない。関係表のセル編集を、そのまま意味変更として扱わない。
- 複合図表内の旧イベント・旧対応表は、Scope/Revision/Epochが不整合ならpublishしない。Graph更新拒否時に旧DOMへ新mappingを結合しない（[#1124の既存負例](https://github.com/hat47x/tei-platform/issues/1124#issuecomment-6426483819)を再利用）。

## 4. 開発パイプライン：ドメイン構成を標準構造へ写像する

開発パイプライン専用のCore`Pipeline`型や固定の`Build`フェーズを新設しない。必要なモデルは、型付きOperation/Artifact、入出力役割を持つ関係、順序・条件・制約、構成・Overlay、来歴の組合せとして表現可能かを検証する。

- **Data dependency**: Artifact入力を要求する。順序はこれから導出可能。
- **Ordering constraint**: データの受け渡しがない先行制約。データ依存と混同しない。
- **Composition / selected Provider**: 標準規則、案件差分、解決版、適合状況。探索順位だけで確定しない。
- **Execution effect**: 書込み・外部実行のCapability / Authority / Effectは実行境界が審査する。Graphの見た目で許可しない。
- **Build artifact / runtime binding**: 既存[#943](https://github.com/hat47x/tei-platform/issues/943) と [#1121](https://github.com/hat47x/tei-platform/issues/1121)を流用。Build-onlyとRuntimeの依存を分ける。

Studioに追加すべきものは新しいビルドエンジンではなく、**解決済み実効グラフを汎用の図・表・関連グラフ・来歴ビューへ安全に投影する接続**である。

### 4.1 「図にないなら無い」の適用条件

対象を特定した `snapshot + environment + target + selected profile + lock + scope` に対し、入力closure、プラグイン挿入工程、動的展開の許容範囲、非表示・折畳み要素、opaqueな外部操作が全て記録され、解決の完全性を検証できる場合に限り、**TEIが管理する工程の不在**を主張できる。

- 完全解決かつ全要素が存在する展開ビュー: 対象範囲のTEI管理工程の不存在を判断可能。
- filter/collapse: 非表示件数、scope、展開導線を示し、「無い」と扱わない。
- unresolved/opaque: 情報不足・外部内部・動的未展開の存在を明記し、不在を主張しない。
- Source変更・Provider変更・標準Overlay更新でclosureが古くなれば、旧表示のCompleteを無効化し、再解決を要求する。
- 計画上の存在、実際の実行／スキップ、独立した結果観測は別証拠。`complete`を`executed`や`authorized`へ昇格させない。

これは**図表の製品状態enumを直ちに追加する要求ではない**。まず既存のResolution/Build/Source evidenceから範囲限定の完全性witnessを構成できるか試す。

## 5. TEI Studioが受け止める拡張境界（専用フックの乱立を避ける）

| 関心 | 提供・所有する主体 | 最小限の接続と禁止事項 |
| --- | --- | --- |
| Topology mapping | domain plugin → 標準構造 | qualified key、関係のrole、scope、来歴、未解決を保持。UI IDを正本IDにしない |
| Projection / composite assembly | 標準プラグイン＋Studio consumer | 同じmodel snapshotの図・表・関連グラフを合成。ビューは第二正本にならない |
| Presentation / layout | Studio UI kit、任意のdomain presentation plugin | 配置・glyph・集約・navigation。domain意味や実行Authorityを増やさない |
| Interaction / selection | Studio consumer | view occurrence → qualified key、同一描画snapshot、epoch guard。旧mappingや曖昧scopeを拒否 |
| Edit translation / review | 対応するSource/Owner＋既存Acceptance | 候補→preview diff/impact→明示Accept→source再読込。Derivedは既定read-only |
| Structural conformance | 標準型・関係の検証＋domain追加検証 | structural validとdomain semantic valid、Provider conformanceを混同しない |
| Transformation / generation | 標準・domain generator plugin | 入出力・制約・失敗/unknown・効果を明示。必要な最小の生成器だけ追加する |
| Effective graph / observability | #1121/#943/#1016を再利用しStudioが消費 | 解決の完全性、Trace、Evidence、Target readinessを別に扱う |

これらの名称は**責務の分類**であり、固定の8段階実行順・新しい8本のCore APIではない。既存のContract/Provider/Projection/Shared Acceptanceを拡張可能な範囲で利用する。あるPluginが他のPluginを要求する場合も、その要求は明示され、Provider解決と実行許可は分離される。

標準パイプラインの強い標準化は、共通の構成・検査・必要Contractと、案件Overlayに対する差分・衝突検出で行う。標準の更新による案件変更の消失を許さず[#2091](https://github.com/hat47x/tei-platform/issues/2091)をOwnerとして扱う。

## 6. 実証する順序：成熟した境界から始める

| 段階 | 小さいReference | 得たい根拠 | 次に進まない条件 |
| --- | --- | --- | --- |
| P0 読取棚卸し | 既存のmodelref/Graph/Projection/IO Mapping/Build | 既存標準で表現できる構造と不足の正確な一覧 | 既存APIで再現できるのに第二Model/Resolverを提案する |
| P1 複合read-only | 一つのOperation/Artifactモデル→Graph＋表＋関連表、さらに異ドメイン（ERか状態） | 同一qualified ID、役割付き関係、表示のみの相違、曖昧性の保持 | View IDから正本や意味を推定する |
| P2 完全性の限定実証 | 同じ構成の実効グラフを標準/Overlay適用前後で比較 | source closure、解決版、隠れた工程、opaque・unresolvedの区別 | 未解決状態で「存在しない」と主張する |
| P3 安全な編集 | 関係の1件変更をcandidateとしてpreview→Accept→独立再投影 | 二重正本なし、Source更新とView更新の分離、失敗時の旧snapshot拒否 | acceptance / owner authorityの迂回 |
| P4 単純化検証 | 従来コード/専用UIと共通構造＋複合図表を比較 | 人とAIの変更時間、必要参照量、誤変更、影響・再試験範囲 | 数値・観察でコストが増加する |
| P5 必要時のメタ生成 | 重複したgenerator構成部分のみ共通化 | 導入前後で理解・変更負荷低減、生成器conformance | 抽象層だけ増え、負担が減らない |

P1の一例: Source/Operation AがArtifact Xを生成し、検証Operation BがXを読む。Graphは依存、表はOperation/Artifactと型、関連表はoperation→artifact→evidenceの**役割**を示す。ビューを跨ぐ選択が同一qualified keyへ到達することを検証する。二つの別subjectの意味関係を「同じ表示要素」と誤認しない。

### 6.1 必須の負例（試験候補、**NOT RUN**）

1. 同名・同kindの別scopeが、Graphと表から同一Inspect対象に解決されることを拒否。候補表示は残し、Inspect spyは0回。
2. G2への不正更新を拒否し旧G1を描画したまま、新対応map M2をpublishしない。旧DOMから新Scope対象へのInspectは0回。
3. 関連表の多項関係を全組の意味エッジとして誤展開しない。役割とn-ary同一性を保つ。
4. 関係が不在なのか未観測/unknownなのかを区別し、Sourceと図表の差分を欠落の自動証拠にしない。
5. 表示位置・折り畳みだけの変更で、意味のSource revisionを変更しない。
6. 標準Plugin/案件Overlayで追加される工程を省略した投影で、Complete/不存在を主張しない。
7. Source Accept後にDecision ledger記録のみ失敗した場合、旧ビューを失効しSource/ledgerを別々に再読込する。Owner受理・Target readinessへ推定しない（[#1124の既存受入候補](https://github.com/hat47x/tei-platform/issues/1124#issuecomment-6470746209)）。
8. 変更影響範囲が不明な動的参照を`unaffected`と誤表示しない。Provider/Authorityの静的資格を現在権限として再利用しない。
9. 標準Plugin更新によるOverlay衝突を暗黙上書きしない。rollback可能な元版を保つ。
10. 既存Maven/Gradle/外部Builderを一つのopaque Operationとして接続したとき、外部内部の完全性まで主張しない。

### 6.2 測定・撤回

- 同じ要求変更で、人間とAIが読む必要のある説明/コード/関係、質問・確認回数、修正ファイル数、誤った参照・影響の件数、正しい変更までの時間、実行・受入試験の範囲を対照比較する。
- 図面上での単純化と変更の局所性を別に計測する。図が綺麗でも意味の連鎖を追えない場合、成功としない。
- 対応表の手入力やメタ生成器の複雑化が増加した場合、新しいCore機構を採用せず既存Projectionまたは限定Pluginへ撤回する。
- 実際に必要な表現ギャップが複数の独立ドメインに再現するまで、新標準型の追加を決めない。

## 7. 既存の所有境界と未決

- [#1120](https://github.com/hat47x/tei-platform/issues/1120): qualified identity・型付き関係・変更影響は共通契約として完了。新規の別Graph正本を作らない。
- [#1124](https://github.com/hat47x/tei-platform/issues/1124): Studioの複数Projection、選択/epoch、Sourceへの編集・受理接続。現行Consumer完成とは扱わない。
- [#997](https://github.com/hat47x/tei-platform/issues/997): Target/Framework非依存の共通UI Kit、Renderer、編集UIの交換性。
- [#267](https://github.com/hat47x/tei-platform/issues/267): Human/AI/CLI共通Acceptance。Studio独自の書込みAuthorityは作らない。
- [#1121](https://github.com/hat47x/tei-platform/issues/1121): Lowering・Target実現・依存/Effect/Build境界。受入未完。
- [#943](https://github.com/hat47x/tei-platform/issues/943): self-build/Artifact/Lock。Issue Closedと実SB-3・Production接続未検証を区別。
- [#951](https://github.com/hat47x/tei-platform/issues/951) / [#1016](https://github.com/hat47x/tei-platform/issues/1016) / [#1223](https://github.com/hat47x/tei-platform/issues/1223): Contract/Provider/Composition/入出力の既存Ownerを尊重。
- [#2091](https://github.com/hat47x/tei-platform/issues/2091): 標準定義更新と案件差分の保全。
- [PR #2226](https://github.com/hat47x/tei-platform/pull/2226): TEI独自Host非必須。2026-11-02 UTCにmerge済み（[ADR-0043](../adr/0043-target-native-artifacts-without-tei-host.md)）。旧Host必須案を復活しない。
- [PR #2298](https://github.com/hat47x/tei-platform/pull/2298): `tei.*`名前空間、任意Composition、Read Projection編集非対称性の未採用の提案（2026-11-03 UTCにmerge済み。統合された設計文書の状態は提案中）。正式な標準名前空間・Binding構文とみなさない。

**未決（ここでCoreへ固定しない）**: トポロジー標準の最小primitive集合、Projection/Correspondenceの具体API、可逆編集の証明条件、全種類の動的Graphのclosure表現、複合図表レイアウト仕様、標準の唯一の図表UI、メタ生成の深さ・専用DSL。

次の実作業の提案はP0→P1のread-only Referenceである。最初からCore型・Studioフック・新ビルドエンジンを増やさず、複合図表の価値と不足する構造を観測する。

## 8. 2026-10-31 他領域との交差検証：共通構造が本当に共通か

- **状態**: 本節は別チャットの設計仮説と現行mainの限定的な静的コード調査を接続した**未採用の追加受入候補**。main照合時点 `4325ca79a52bf0d576db6fb4027bdc516b1023fd`。新たなコード・試験・Pluginの完成を主張しない。
- **コード上の現状（限定）**: `internal/modelref/model.go` の `Key` は `Scope + Subject`、`Graph` は `Nodes + Edges`、`Edge` は `From + To + Relation + Evidence` の**二項参照グラフ**であり、一般のn項関係・図表間対応表の製品APIではない。`reference/tei-studio-graph/graph.ts` は表示専用の `id/label/geometry` と二端点Edgeを持ち、現Referenceの直接自己ループを拒否する。Rendererのview IDを意味ID、実行/編集権限と見なせない。`reference/tei-studio-workbench/presentation.ts` の `Subject` は `kind/id` であり、qualified選択を横断して実装完了した状態ではない（#1124のConsumer安全性は未完）。
- **方式**: 異なる2〜3個のConsumerを、同じ構造部品・投影・対応・診断によって表せるか試す。単純な二項Graphへ全てを押し込めず、既存の型・集合・役割付き関係・表の合成を先に試す。多項関係を描画上「関係ノード＋役割エッジ」に変換しても、元の関係の同一性・役割・来歴を失わないことを検証する。

| 異なるConsumerによる圧力 | 共通機構の受入候補 | 領域側に残す意味／異なるAuthority |
| --- | --- | --- |
| 三視点・受注明細（#1124） | Data/Function/Businessの別Projectionから同一qualified subjectへnavigate、関係表と影響図を生成 | Business Procedureの正式受理、関係の意味、Source書込み判断はOwner |
| 三要素分析の方法論プラグイン（設計案） | 同一subjectの複数図面Occurrence、図間の関係表、Notation・View StateとMeaningの分離、関連定義へのjump | 方法論上の妥当性・固有の記法・X-TEA互換性は専用Plugin。旧資産の表示から受理済み意味を推定しない |
| 開発パイプライン（#943/#1121/#1016） | Operation/Artifact/Dependency/Overlayの図＋表＋関連グラフ、条件付き工程のcoverage、opaque工程の明示 | ビルド・キャッシュ・Provider選択・配布資格・実副作用は既存Owner |
| 外部UI制御（#2391、任意） | ModelRefと実UI出現を異なる識別子として対応付け、選択snapshot/更新世代を再確認 | locate/act/wait/observeはUI Adapter。業務Command/Owner受理/Effectは#1158/#1174/#267、Scenario正本は#1180 |

三要素分析を**Rendererの寄せ集め**とみなさず、構造上の役割を方法論上の意味・記法・編集制約へ写像する領域Pluginの圧力として試す。ただし三要素分析のNotation Grammar・Source互換・形式固有の生成規則を、本書でCore標準へ採用しない。Action Diagramも説明・検証用のProjectionであり、実行プロトコルを代替しない。

**失敗判定を明確にする**: 共通の二項Graph/表への射影で多項の役割、順序、包含、Occurrence、多重度、時間、出典のいずれかを復元できない場合、「図を描けた」だけでカバー済みと判定しない。共通標準の真の不足と、領域Pluginの意味定義不足と、単なるRenderer不足を切り分ける。2つ以上の独立Consumerで反復する不足が見つかるまでは、新Core primitiveや新Studio APIを固定しない。

## 9. 「ない」を断定するための限定された不在証拠

図表が複数あり、図表間の関連グラフがあっても、要素の完全性と**関係の完全性**は別である。次の問いを独立に評価する。名称は実装上の列挙型ではなく、レビュー時の観点。

| 完全性の観点 | 検証する対象 | 「不明」の典型 |
| --- | --- | --- |
| 対象集合の被覆 | 指定scope/型/条件で対象となる要素が全件投影されたか | フィルタされた・未解決Source・動的追加 |
| 関係とroleの被覆 | 関係の種類、方向、全ての端点/roleが正しく投影されたか | n項関係の一部を二項Edgeとして捨てた |
| Projectionの可視被覆 | 折畳み・非表示・省略した要素/関係を計数・展開できるか | 複合図表の別Viewに関係が隠れている |
| 構成の解決閉包 | 標準Plugin・案件Overlay・Provider/契約・Build入力・条件付き処理が解決済みか | 未選択Provider、外部Builder内部、動的参照 |
| Snapshotの整合 | Source/Accepted/Projection/Plugin Lock/Target/Environment/選択epochが整合するか | Source改訂後の旧Graphと新しい関係表を混用した |

不在判定の概念規則（**製品APIではない**）:

~~~text
absent(typedQuery, boundedSnapshot) is supported only if:
  query.scope is closed and explicitly bounded
  AND subject coverage = complete
  AND relation-role coverage = complete
  AND view coverage is accounted for (including folded/hidden)
  AND required composition/input closure = resolved
  AND all participating revisions are coherent
  AND unresolved/opaque regions do not intersect query.scope
  AND there is no matching subject/relation
otherwise:
  unknown / not-proven   # not "absent"
~~~

この規則は**TEI管理下で宣言・解決されたモデル／工程の範囲**にだけ成立し得る。外部コンパイラ内部、未公開の外部副作用、実際のOwner Business Effectの不存在まで推論しない。また、completeな計画は実行完了・現在のAuthority・効果確定の証明ではない。

### 9.1 版の異なるView同士のJoinは受理しない

複合図表の図A、表B、関連表Cで`qualified key`が同じであっても、それぞれの表示が異なるSource/Projection revisionから生まれた場合、同じ現在事実として関係を結合しない。図AがG1、表BがG2なら、共通Viewとして確定するか片方をstaleにするまで、相互ナビゲーション・Source locator確定・Edit Intentを抑止する。図Aの描画更新が拒否された場合、旧DOMに新View mapを公開しない。これは#1124の既存Consumer不変条件を複合図表へ**適用する**提案であり、新しいglobal epochを発明しない。

### 9.2 拡張プラグインの資格を図表完全性や実行権限に昇格しない

別チャットの分散Registry構想では、**配布物の識別・形式・署名・Conformance**、**案件Compositionの解決・依存閉包・Target適合**、**実行時のAuthority/Resource/Effect Fence**を区別する。公開Registryで一つのPluginが検証済みでも、案件がそのPluginを含めた実効Pipelineを完全に解決したことにはならず、実行許可も付与されない。

本設計で必要なのは既存#951（Contract/Provider）、#1016（案件Composition）、#943（Build/Artifact）、#1174（Effect/Outcome）への**根拠付き参照**であり、Registry ID/Genesis形式、Portable Verification Package、透明性サービスをStudioや基本トポロジーの固定APIへ導入することではない。Plugin Type、複数Instance、各Bindingのscope/契約/版も、図上の単一ノード名で同一視しない（#2298は2026-11-03 UTCにmerge済みだが、内容は提案中で未採用）。

## 10. 次の最小Referenceの具体的な受入判定（P0→P1に限定）

1. **同一構造・二つのView**：同じqualified `modelref.Key` の要素をGraphとTableで異なる表示IDへ投影し、選択が同一対象に帰着する。異なるscopeの同kind+IDは候補表示にとどめ、Inspect/Locate呼出しは0。
2. **多項関係と関連表**：例として「生成Operation・入力Artifact・出力Artifact・検証Evidence」のrole付き関係を、対応表と関係ノードの図に投影する。失われたroleやSource来歴があればFail。既存二項`modelref.Graph`への直接埋込みを強制しない。
3. **三要素分析の投影**：一つの意味上の対象が別図で複数Occurrenceを持つ。Notation/座標/選択は表示側、意味の関係はDomain側。図上のジャンプはqualified参照の読取に限定する。
4. **Complectの負例**：依存と順序を単一種類のEdgeに押し込んだ場合の情報損失、もしくはSource変更とView layout変更の連動を検出し、分離前後の変更影響を比較する。
5. **Completenessの反例**：要素は全件表示したが関係が一部欠落する図、Overlay追加工程が隠れている図、旧Viewに新mappingを結合する図では、「ない」を拒否する。

**出口条件**: 既存Reader/Graph/Table部品を再利用できる部分と不足を記録し、構造的・領域的・表示上の欠落を分類する。実装と試験の担当を#1124等のOwnerと合意し、P1のread-only完全性すら証明しない段階で、双方向編集・メタ生成・本番Production除去の受入へ進まない。新標準型の採用、AI生成Pluginの安全な実行、Studio複合図表UIの製品化はこのReferenceの成果を見て別途判断する。

**検証状態**: 現行mainと関連Issueの静的照合に基づく設計追補。既存コードに対する新たな自動テスト、real UI/Provider/Target、実際の認知負荷比較、X-TEA三要素のSource互換証明はいずれも **NOT RUN / NOT PROVEN**。

## 11. 2026-11-02 横断レビュー：Projectionの被覆根拠と独立Consumerの実証入口

- **状態**: 本節は他レーンの成熟した設計・限定試験と現行GitHubを照合した**未採用の評価補足**。確認main `9c9ef4d160640908e06ad13afb6c2b45c664c748`。本Draft自身はmainへ未統合であり、下記の他Draftを正式仕様・実装済みとは扱わない。
- **Ownerとの接続**: [三要素Methodと図法のDraft #2397](https://github.com/hat47x/tei-platform/pull/2397)、[型付き標準SemanticとPlugin I/OのDraft #2592](https://github.com/hat47x/tei-platform/pull/2592)、[任意Compositionと型／Instance分離の#2298（2026-11-03 UTCにmerge済み）](https://github.com/hat47x/tei-platform/pull/2298)、[Studio Catalog曖昧性guard候補 #2583](https://github.com/hat47x/tei-platform/pull/2583)、[選択binding独立候補 #2457](https://github.com/hat47x/tei-platform/pull/2457)。各Draftで定義したMethodの意味規則、標準型候補、実装差分をこの文書へ複製しない。

### 11.1 「表せる」と「欠落なく表せる」と「操作可能」を別々に証明する

汎用構造を増やす判断では、少なくとも以下の**独立した三つの問い**を立てる。

| 問い | 提示すべき根拠 | 禁止する短絡 |
| --- | --- | --- |
| 表現可能性 | 型・順序・cardinality・role・端点・出現回数・Sourceを保持した構造対応 | 一枚の見た目が描けたのでlosslessと主張 |
| 問いに対する完全性 | bounded query、対象集合と関係集合のclosure、非表示・unknown・opaque・参加snapshotの整合 | 図／表が空なので対象は不存在と主張 |
| 操作資格 | 表示occurrence→qualified keyの同一版対応、編集先OwnerとAcceptance、実行時の独立Authority | 図上で選択できるので書込み・実行が許されると推定 |

**図表の合成では、個別図表の「complete」表示を単純に足して全体completeとはしない。** 例えば図Aは全Operation、図Bは全Artifactを表示していても、両者を結ぶ`produces`のrole/証拠が未表現なら、`produces`関係への不在問い合わせは`unknown`となる。図間関連グラフや対応表はこの欠落を表示できるが、それ自身の表示成功は元の関係集合の完全性証明ではない。

評価用の概念表現（**新製品スキーマではない**）:

~~~text
ProjectionWitness(query, boundSnapshot) =
  represented-elements / missing-elements / unknown-elements
  represented-relations-and-roles / missing-or-unknown-relations
  view occurrence -> qualified subject + projection revision
  omitted-or-collapsed coverage
  derivation / source / plugin-resolution evidence
  validity boundary (scope, target, environment, effective lock)

Absent(Query) is defensible only for the closed subset proven complete.
Else absence remains unknown; no universal closed-world assumption.
~~~

「source上に宣言されたTEI管理Operationが無い」と、「実行された外部Builderがその内部で未知のActionを行わない」とは別である。前者に必要なgraph/closureの被覆証拠、後者に必要な実行環境の観測・enforcementを同一の`complete`へ畳み込まない。これは[#2391](https://github.com/hat47x/tei-platform/issues/2391)のTest/Production物理除去・Artifact同一性・独立効果観測とも別Gateである。

### 11.2 プラグイン型とインスタンスを一つの表示ノードへ混同しない

同じTypeを持つ二つのComponent Instanceは、Binding、Scope、Input、Output、Effect、選択Providerが異なり得る。型レベルの図表、インスタンス構成図、Provider解決図、実行観測図をそれぞれ別Projectionにして、**同じInstanceを指す場合のみ**qualified identityと選択版に基づき結ぶ。

一つのStandard PluginがOperationを宣言しても、それがその案件で採用・解決・配布・実行されたとは限らない。[#951](https://github.com/hat47x/tei-platform/issues/951)のContract/Provider適合、[#943](https://github.com/hat47x/tei-platform/issues/943)のArtifact/Build、[#267](https://github.com/hat47x/tei-platform/issues/267)のAcceptanceと実行Authorityを別に残す。新しいPlugin Registry、Runtime Resolver、Graph正本は本書で追加しない。

### 11.3 Complectを「局所変更の独立性」で検査する

ノード／エッジ数を削減するより、**直交すると期待した変更が本当に局所化されるか**で測る。P1以降のReferenceで、同一モデルに対する次の対照操作を用意する。

| 対照変更 | 本来変わる対象 | 変化したら混線を疑う対象 |
| --- | --- | --- |
| 図表レイアウトだけ変更 | view layout / occurrence coordinates | Source意味revision、依存やビルド工程 |
| Target実装を差替える | Provider binding / Target実現 / Artifact | 受理済みドメイン関係や業務規則 |
| 依存関係だけ変更 | typed relation / derived order / impact | 無関係な表示ID、Ownerの意味受理記録 |
| 生成規則だけ差替える | Transformation / derived artifacts | 暗黙の実行認可や外部Effect |
| 図表のフィルタだけ変更 | visible subset / coverage status | Canonical要素の存在・不存在 |

ただしドメイン上の必然的な影響は失敗ではない。分離後に必要な制約を失わず、意図しないSource/Authority/Effectへの波及だけが減ったか測定する。生成AIのコンテキスト量、変更箇所、修正回数、誤選択・誤推論、再検証範囲を併記し、メタ抽象による探索コスト増加も観測する。

### 11.4 先行するConsumer安全性と、後続の汎用複合図表を分ける

現行の優先順位は、[#1124](https://github.com/hat47x/tei-platform/issues/1124)の**PR1: Catalog ambiguity保持・共通Inspect guard → PR2: Graph/List/initial/command 4入口 → PR3: Graphの旧DOM/qualified map/epochの不一致拒否**を先行させる。Draft #2583の限定CI成功を4入口・旧DOM・Source鮮度までの成功として拡大しない。Draft #2457の限定Graph binding試験も実Workbenchの受入証拠にしない。

本書P0→P1は、PR1の意味的資格を代替・ブロックする新たな汎用UI実装ではなく、**既存のread-only正本参照から図＋表＋関連表を作り、情報の保存・欠落・不明を分類する独立Reference**とする。その後の意味編集、Owner受理、Headless E0→E4、Production物理除去は各Ownerの独立Gateに残す。

**次の最小証拠**: (a)同一qualified要素の複数View occurrence、(b)role付き関係のGraph/Table間の完全往復（描画・意味変更なし）、(c)queryを制限して初めて成立する「不在」の正負例、(d)Layoutのみ変更でSource不変、(e)旧Snapshotまたは曖昧scopeではInspect/Locate呼出し0。まずこの五つのread-only検査を確認し、標準構造不足の観測なしにCoreやStudioフックを追加しない。

**検証状態**: 本追補は既存Issue/PRとコード公開契約に対する静的設計照合である。Reference本体の追加、実GUI、実Provider/Runtime、独立Observation、認知負荷実測、実Production Artifact検査は **NOT RUN / NOT PROVEN**。

## 12. 2026-11-06：P1限定不在判定Reference（合成・採用前）

**既存§9〜11の実装候補をコード化した最小実験**：[reference/tei-topology-coverage](../../reference/tei-topology-coverage/README.md)。基準main `36bda76271db4db72ddd41a8566d281f02764b8a`。本節は基準設計を置換せず、新たなCore型・正本Graph・Source Reader・実行Host/Authorityを採用しない。

- 一つの合成Source/Projection snapshotから、scope付きSubjectをGraphとTableへ投影。同じkind/IDでも別scopeなら別keyとし、view IDは画面内のOccurrence識別子のまま維持する。
- 多項関係（Operation、Artifact、Evidence）を「関係ノード＋role付きEdge」と「role付き関係表」で同時に表す。端点欠落・role脱落・Relationの描画欠落は不在判定ではなく`unknown`とする。
- `boundedAbsence`は**合成fixtureが宣言した閉包内の不在だけ**`absent-in-declared-snapshot`と返す。scope未閉鎖、Plugin/Composition/Reader未確認、版混在、省略・opaque領域、Disclosure未資格では`unknown`。表示上に要素が存在しても、非開示なら公開Claimを出さない。
- レイアウトだけの変更でSource/表/閉包判定が変わらないことを確認する。対照ケースには重複qualified subjectと欠落端点を含む。

**実行証拠と非主張**：Node v22.16.0で`node --test reference/tei-topology-coverage/coverage.test.mjs`をローカル実行し、**12/12 PASS**（合成fixture、2026-11-06）。ただし本Evaluatorは`subjectsComplete=true`等の**申告値の真正性を証明しない**。実TEI `modelref`/CLI/Source Index、実Reader/Policy/Provider closure、旧DOMと4入口、非公開二世界の非干渉、実Target/Effect、認知負荷の改善は**NOT RUN／未証明**。#1124 M0/M1より先にこのReferenceの製品導入を要求しない。

**セルフビルドと非画面パイプラインへの適用境界**：[TEI全体セルフビルド #2831](https://github.com/hat47x/tei-platform/issues/2831)のSB-P0.1は**宣言した工程と実際に消費・起動・生成した工程/bytesの照合**がOwnerであり、本Referenceの「宣言Graph→表示Graph/Tableの損失検査」だけでは代用できない。実Run側の独立Read/Traceがない場合、図に無い実行工程・動的ロード・pre/post hookの不在を推測しない。非画面ETL/ELT・バッチ・ファイル中継も、Source上のstage記述／実Producer・Transfer・Consumer／再送・ACK/Receipt／外部Effectを別レーンに保持し、未観測はopaque/unknownとする（[非画面実証 #1130](https://github.com/hat47x/tei-platform/issues/1130)、[Scenario SSOT #1180](https://github.com/hat47x/tei-platform/issues/1180)）。#2831の実Build witnessと#1124のConsumerは別のOwner／別資格であり、この試験はその両者を結合したと主張しない。

**次の反証可能な接続**：#1124の実CLI→同一Consumerで同じqualified SubjectとSource/Model revisionを取得し、(a)元Source/関係roleの閉包が実Readerで確認できる範囲、(b)不完全で`unknown`となる範囲、(c)Graph・Table・関連表間の局所変更が意味Sourceを変えないこと、を別の証拠として測る。実験が必要な新Core primitiveを示さなければ既存Standard Structureで継続する。

## 13. 2026-11-06 横断追補：表示同一性の整合と「不存在」の主張資格

**状態：合成Referenceの局所実証。新Core/Open-World API/Policy Engineの採用ではない。** 根拠は、Script-firstの三要素分析 [Draft #2878](https://github.com/hat47x/tei-platform/pull/2878)、Role自己記述・必要閉包 [Draft #2860](https://github.com/hat47x/tei-platform/pull/2860)、実工程セルフビルド [#2831](https://github.com/hat47x/tei-platform/issues/2831)／[Draft #2876](https://github.com/hat47x/tei-platform/pull/2876)、Studioの実Consumer [#1124](https://github.com/hat47x/tei-platform/issues/1124)、許可済み合成Evidence [Draft #2833](https://github.com/hat47x/tei-platform/pull/2833)。他Draftの設計・限定PASSを、本Referenceの製品資格へ合算しない。

### 13.1 Claimごとに異なる三つの「閉じている」を分ける

| 区分 | 限定された肯定に必要な観測 | 不足時の扱い |
| --- | --- | --- |
| **表示整合** | 同一SnapshotのTable行（scope/kind/id/key/view ID）、Graphノード、Relation role/端点/Edgeが**値とIDの両方**で一致。SourceIndexと投影版が変われば再結合する | stale / unresolved。誤ったView選択からInspect・Editへ昇格しない |
| **宣言された問いへの閉包** | bounded scope、対象種別・Relation IDを含むquery、subject/relation/role/viewの完全な列挙、opaque/omittedなし、Reader/Plugin Compositionの申告 | unknown。個別図の完全性や単純な件数一致から不存在を断定しない |
| **現物・利用資格** | そのSource/Artifact/Plugin/Target/Providerの**実際のReaderと消費byte**、現在のPolicy/Disclosure、実Run/Delivery/Receipt、Owner受理を、それぞれの必要Claimへ対応付け | **本Referenceは証明しない**。Synthetic `evidenceId`やboolの申告をAuthority・現物閉包・独立Oracleにしない |

同じSubjectやMeaningVersionが不変でも、Sourceの位置移動・Role/Operator Pin差替え・Policy失効・Reader世代更新・外部工程追加は異なる依存へ作用し得る。単一のグローバルepochやモデル全体Digestだけで各Claimを一括資格化しない。また低権限Consumerへ`unknown`の内訳を返すこと自体が非公開依存の存在を漏らす可能性があるため、公開画面への変換は[#2516](https://github.com/hat47x/tei-platform/issues/2516)の既存Policyが許可したsliceを用いる（本Referenceは**Policyにより選別済みの合成入力のみ**を対象にする）。

### 13.2 実装から得た具体的な負例

[read-only Reference](../../reference/tei-topology-coverage/README.md)で、次を追加して確認する。View全体の版文字列だけを確認しても防げない事例を中心にした。

| ケース | 反例 | 局所的な期待 |
| --- | --- | --- |
| TC-I01 | Table行の`scope`だけが別値、保存されたqualified keyは旧値 | 表示一致ではなく`unknown`、選択は`unresolved` |
| TC-I02 | Graph nodeの`viewId`を残したままkeyが別scopeへ再束縛 | `unresolved`、scopeなしInspectへ接続しない |
| TC-I03 | Graph nodeを重複表示し、別nodeの欠落を件数で隠す | 不在Claimは`unknown` |
| TC-R01 | 同じRelation型が存在するが、queryのrelation IDは別 | 別Relationを存在証明に流用しない。閉じたfixture内に限り限定的不在 |
| TC-R02 | Relation role文字列またはGraph Edgeのrelation IDのみ改変 | `unknown`。二項Edgeの見た目で元の多項意味を復元しない |
| TC-F01 | Meaning/Model revision不変でSourceIndex fingerprintだけ改変 | 旧View選択は`stale`、複合表示は`unknown` |

**今回の実測**：新規[専用GitHub Actions](https://github.com/hat47x/tei-platform/actions/runs/42155243921)でNode 24.21.0・**18/18 PASS**。当該runがcheckoutしたexact head `577bbe60d961c58e142502ab5835c9a7610214df`、`coverage.mjs` blob `5cc3ca120e2995650bcaba1121a2416b3be8010a`、試験blob `6ece714d49a7b6f8dfa4ff7d376dfe4eb61e7f4d`、終了時clean treeを確認。これはSource Closure/Policy/製品UI/実Target E2Eではなく**synthetic identity/coverage logic**の検証。第12節の12/12は旧時点の限定記録として残す。

### 13.3 次の研究で評価する効果と撤回条件

- **Studio優先**：#1124のM0実CLI→同一Consumer、M1四入口0/1 RPC、PR3旧DOM・後着・scope epochを先行。Referenceの`selectView`が返すcandidateは実Source locator/Inspect可能性を証明せず、製品UIへ直接組み込まない。
- **Self-build/ETL優先**：#2831は宣言Buildと実消費Input/実行工程/Artifactの一致を独立Traceで確認する。#1130のETL/Batch/File Relayでは転送Producer/Consumer、再送、ACK/Receipt、外部Effectを区別。合成Viewの閉包は実処理の完全性に昇格しない。
- **Role・Plugin共通性**：Role Typeの検証と、複数InstanceごとのBinding・Operator Pin・Role coverageは別。二つ以上の独立Consumer/domainで標準構造の不足を反証するまで、Core primitiveやStudio固定フックを追加しない。
- **認知負荷の検証**：正答率（不存在の誤断定率）、Sourceへ辿る操作数、Graph/Table/関連表の再照合回数、局所変更の無関係波及、AIの見落とした不明点とToken/修正回数を、従来の単一図と複合図表で比較する。**測定未実施**であり優位性を前提化しない。

**採用停止線**：実Source/Role/Reader closure・Policy/Consumer・Build Runnerの各Ownerが証明できる前に、`boundedAbsence`の合成ステータスを「実TEIに存在しない」「Productionで実行されない」「Ownerが受理した」へ昇格させない。#2881はP1独立実験のDraftを維持する。

### 13.4 Relation自身のscopeと端点scopeの独立性（2026-11-06追補）

他チャットの[Role自己記述 #2620](https://github.com/hat47x/tei-platform/issues/2620)／[Draft #2860](https://github.com/hat47x/tei-platform/pull/2860)、[TEI全体セルフビルド #2831](https://github.com/hat47x/tei-platform/issues/2831)、[ファイル・メッセージ連携 #2542](https://github.com/hat47x/tei-platform/issues/2542)の横断レビューから、**Relationの識別scopeと、Relationが結ぶ各Subjectのscopeを同一視しない**という反例を独立させた。これは既存ModelRef/Relation Schemaの採用変更ではなく、Read-only Projectionが保持しなければならない情報に対する圧力試験である。

- 例：scope `batch-A`で宣言したTransfer Relation `flow`が、scope `file-bridge`のArtifactとscope `erp`のConsumerへ結ばれても、Relationの宣言元が端点のどちらかへ自動的に移ることはない。セルフビルドでもBuild Operation・Target Artifact・Toolchainの所属は一致するとは限らない。
- 試験FixtureのRelationは、**明示された`scope + relation ID`** を自身のqualified keyとする。別scopeの同じrelation IDは区別し、役割・端点は別のscope付きSubject keyへ参照する。GraphのRelation node/各role EdgeとTableのRelation recordは**key、type、role、端点**を同一Snapshotで照合する。
- `query.scope`のRelationの不存在を問うときは、端点のscopeを検索範囲の代用にしない。Relation自身の宣言scopeが未取得・未解決なら、実Consumerは`unknown`のままにし、対象集合を勝手に閉じない。これは内部Referenceで明示scopeを**要求する**理由であって、既存TEI SourceのRelationがすべてそう表現されているという主張ではない。
- Reference上の負例は、(1)他scopeの端点を持つRelationでも宣言scopeのみで存在判定、(2)別scopeの同ID Relationの併存、(3)宣言scope欠落・同scope重複の拒否、(4)Table宣言scopeまたはGraph relation keyの後編集、(5)端点scopeからの誤った存在推定、(6)GraphとTableのRelation type齟齬。Roleの重複宣言を後値で上書きする[別Parser問題 #2898](https://github.com/hat47x/tei-platform/pull/2898)の正式修正/受理とは別問題である。

**実装・証拠上限**：同一合成RelationをGraph/Tableへ無損失に投影するためのReference契約を検査する。実TEI ModelRef/Reader、未宣言関係の実不存在、Build・ETLのactual execution/transfer/Receipt、Current Policy、Owner受理は証明しない。[#2877のP0-A0/A1](https://github.com/hat47x/tei-platform/pull/2877)が扱うRequired/Declared/Selected/ActuallyConsumed/Observed/Undischargedを表示完全性に足し算して、実Pipeline完全性に昇格させない。

**追加の実行証拠**：[専用Node CI #42156720965](https://github.com/hat47x/tei-platform/actions/runs/42156720965) は、変更後exact head `760597b2aaddde1bf3990ee8f57a35deb9bdb9ea`で **26/26 PASS**。対象コードGit blob `c20ceb222a690d486ce5b8f95782dba6c84d9a13`／試験Git blob `9ec51ea2518b05499405bc86e21398b91856084a`、Node 24.21.0、テスト前後のclean treeを確認した。これらは**合成のscope/type/role/Graph/Table内部整合だけ**の証拠であり、元Source・実Consumer・実Build・実Transferの保証ではない。第13.2節の18/18は当時の限定記録として残す。

**次段の最小受入**：既存Go Reader/Source Indexから(1)実Relation identity、(2)Relationの宣言元scopeを取得できる契約、(3)端点のqualified key、(4)roles/typeとSource/Projection revision、(5)開示済みscopeの閉包根拠を確認する。いずれかが欠けたら、図表は候補・部分表示にとどめ`absent-in-declared-snapshot`のProducer資格を実Sourceへ移さない。既存#1124/#2620/#2831/#2542の担当境界を変更せず、新共通Relation Registry/第二Resolverを追加しない。

### 13.5 Query相対の条件被覆と、無視されたPredicateからの偽の不存在を禁止（2026-11-06追補）

[研究計画 Draft #2966](https://github.com/hat47x/tei-platform/pull/2966)は、合成Referenceとは独立した実Source Readerと認知負荷比較を所有する。本節はその研究を再実装せず、**既存#2881の合成Evaluator内で確認できるQuery意味の欠落**だけを扱う。

**具体的な不具合候補（固定Sourceを静的に確認）**：従来の`boundedAbsence`は`kind/scope/type/id`のみを読んでいる。したがって、たとえば`{kind:"relation",scope:"two",type:"produces",participantScope:"two"}`という問い合わせで`participantScope`を無視し、宣言scopeが`two`に無いだけで`absent-in-declared-snapshot`を返し得る。しかし、scope `one`に所属するRelationがscope `two`のSubjectへ接続していることと矛盾しない。この「無視した条件で得た不在」は、閉包をどれほど正しく検査しても元の問いへの答えにならない。

**最小対策（新Query DSLではない）**：この合成Referenceで受理する条件を`kind`、`scope`、`type`、任意の`id`に限定し、それ以外の`participantScope`、`role`、`where`、`window`、`subjectKey`等は **`unknown:unsupported_query_predicate`** へ戻す。Relation queryの`scope`は**Relationの宣言scope**であり、接続先のscopeや、特定Roleを通じた入射関係、依存の推移閉包を表さない。これらの問い合わせを今後実装するときは、必要なEndpoint/Role/Scope/Window/Source closureを独立に列挙し、そのQueryについての完全性を示すまで不在断定を禁止する。

| 反例 | 誤った判定の型 | このReferenceの許容する判断 |
| --- | --- | --- |
| 別scopeのSubjectへ接続するRelationの端点条件を無視 | 所属Relationが無い→接続Relationも無い | `unknown` |
| `role=receiver`を無視してRelationの型だけで存在確定 | 別Roleの関係を同じ解釈へ昇格 | `unknown` |
| ETLの`window/end`、BatchのACK条件を無視 | 対象期間の全件処理・配送完了を宣言Graphから推測 | `unknown`、実Event/Receiptは別Owner |
| Buildの`where/target/selectedModules`を無視 | 未選択Dependency・実消費Inputの不存在を断言 | `unknown`、#2831のSelected/Consumed証拠は別 |
| 無効ID、未対応の式・伝播条件の暗黙削除 | 広いQueryの結果を狭いQueryの答えに転用 | `invalid_query`または`unsupported_query_predicate` |

これは単なる入力Validationの拡充ではなく、**意味解析していない条件を処理済みと見なさない**というEverything is Data/Role自己解釈の共有原則に対応する。正本Source、意味解釈部品、Projection、Query、必要な観測の各区分で「未解決」を保持する。既存実装の`present-in-snapshot`と`absent-in-declared-snapshot`はいずれも**fixture内部**のClaimであり、Owner Authority、現在Policy、実Readerの完全性を意味しない。

**受入／未受入**：同一合成Fixtureで、Queryに未対応Predicateを追加したとき、存在系・不在系双方が`unknown`となることを回帰試験で固定する。実TEIのQuery/Role DSL、Source closure、Studio/AIでの問い解釈、ETL WindowやSelf-build Target/Selected Inputs、現在の開示非干渉は**別Owner／NOT RUN**。新Core Query型・汎用Solver・第二Authorityは導入しない。

### 13.6 実Go ModelRef／Source IndexのFit/Gap（2026-11-06静的照合）

固定main `3572dcedfa50c3de2903dd4d8ca4ad45ccfa1417` の既存コードを確認した。これはGo実装を変更・実行したことを意味しない。別担当[研究計画 Draft #2966](https://github.com/hat47x/tei-platform/pull/2966)にある**独立実Source照合・認知負荷研究**を重複実装しない。

| 既存の事実（該当Source） | 実Readerと合成Referenceの間に残る不足 |
| --- | --- |
| [`modelref/model.go:15–56`](https://github.com/hat47x/tei-platform/blob/3572dcedfa50c3de2903dd4d8ca4ad45ccfa1417/internal/modelref/model.go#L15-L56)：`Key{Scope,Subject}`、`Edge{From Key,To Ref,Relation string,Evidence}`、`Graph{Nodes,Edges}`が存在 | Edgeの`Relation`は**関係種別文字列**であり、すべてのRelationが独立の`declaringScope+relationID`、多項role列、Source anchorを保持するとは証明しない。合成RelationをGo Edgeへ1:1対応させない |
| [`modelref/trace.go:13–55`](https://github.com/hat47x/tei-platform/blob/3572dcedfa50c3de2903dd4d8ca4ad45ccfa1417/internal/modelref/trace.go#L13-L55)：`TraceDirect`は直接宣言Edge、方向、full/summary、ページ、snapshot fingerprint、unresolved件数を提供 | **直接Edgeの候補観測**には利用可能。Graph全体のRelation実体・多項role・全Source閉包、未読込Pluginの不存在を証明しない。fullのページングと失効も別確認 |
| [`sourceindex/model.go:41–56`](https://github.com/hat47x/tei-platform/blob/3572dcedfa50c3de2903dd4d8ca4ad45ccfa1417/internal/sourceindex/model.go#L41-L56)：Source Index Referenceは`Subject(kind,id),ArtifactPath,Range,Role` | Scope付きRelation/Subjectの独立Source位置が直接結び付く仕様ではない。形式上ある`Role`はSource anchorのdefinition/authoring等の**証拠役割**であり、業務関係の多項Roleと同一視しない |
| [`modelref/sourcebridge/source.go:21–37`](https://github.com/hat47x/tei-platform/blob/3572dcedfa50c3de2903dd4d8ca4ad45ccfa1417/internal/modelref/sourcebridge/source.go#L21-L37) ／ [`resolver.go:85–110`](https://github.com/hat47x/tei-platform/blob/3572dcedfa50c3de2903dd4d8ca4ad45ccfa1417/internal/modelref/sourcebridge/resolver.go#L85-L110)：Source Indexにscopeがなく、同kind+IDの異scope衝突は`subject-scope-ambiguous`で停止 | 欠けたscopeをnamespaceや表示ラベルから推定しない。Source locatorがあることを編集権限や現在のOwner受理へ昇格させない |
| [`operations/model_catalog.go:15–33`](https://github.com/hat47x/tei-platform/blob/3572dcedfa50c3de2903dd4d8ca4ad45ccfa1417/internal/operations/model_catalog.go#L15-L33)：Catalogは`ModelRevision`・World/Source FP・Subject・global `ambiguous`を持つ | Catalogはread-onlyの候補表示であり、全Relation宣言scope・roles・用途相対Closureの実Reader証拠ではない |

**最小Fit/Gap判断**：共通のqualified Subject、direct Edge候補、版付きTrace、Source位置は既存の足場として再利用できる。一方、(a)Relation宣言scope・個別ID、(b)多項Relationのすべてのroleと端点、(c)Input/Source/Plugin/Providerの問い相対なRequired/Consumed被覆、(d)QueryのWHERE/Role/Window等の型付き解釈、(e)Current Disclosure/Authorityは別資格。合成Referenceの真偽値から逆算して実契約へ追加しない。**不足の反証が二つ以上の独立Consumerで確認された場合のみ**、#2620等のOwnerが既存構造／Plugin／Readerの拡張を判断する。

**低権限向けの停止線**：合成Referenceでも、開示資格の無いfixtureはGraphの世代・要素数・relation欠落・unsupported条件の差を公開結果に反映しない。内部関数の同一返答だけではDOM/ARIA/Count/Tool JSON/実Providerの二世界非干渉ではない（#2516）。合成の開示フラグ自体は偽造可能で、実Policy受理を代替しない。

**確認・未確認**：上記はGo Sourceの静的読取と合成Node試験であり、実Go Reader→合成Graph/Tableの変換、実多項Relation／Source Anchorの往復、実CLI/Studio 4入口、実Policy/Disclosure、実Build/ETL・認知負荷の検証は**NOT RUN**。次は#1124のM0実CLIが取得した同一Source/Subject候補を、独立Reader/Expectedと比較する最小一例から始める。

## 14. 2026-11-06 実CLI direct-edge観測と多項Relation再構成の停止線

**研究成果の段階変更**：§13.6のGo Source静的Fit/Gapを受け、既存`cmd/tei trace`→read-only Node Consumerの小さな実Workspace縦切りを、[PR #2881](https://github.com/hat47x/tei-platform/pull/2881)の合成Referenceと独立に追加。CLIは既存の`operations.TraceWorkspace`と`modelref.TraceDirect`、SourceBridgeを使用する。本実験のために新しいReader、Relation Runtime、正本Graph、共通Host、Policy Engineを実装しない。

### 14.1 同じ「関係」を扱っても識別資格が異なる

| 表現または事実 | 既存実装で確認可能なもの | **導出してはならないもの** |
| --- | --- | --- |
| **Actual Workspace CLI Trace** | `tei.cli.trace/v0`のread-only成功、`tei.operations.trace/v0`、scope付きSubjectとdirect Edgeの`From/To/Relation`、World/SourceIndex/Trace fingerprint、resolved/unresolved、SourceNavigation | すべてのRelation instanceの`declaringScope + relationID`、多項roleと端点の完全列挙、Source closure、編集・実行資格 |
| **Reference direct-edge observation** | 実CLI出力を既存JSON schema・scope／方向・row cardinalityで検査し、**direct-edge候補**のみ返す。traceのページ不足、誤版、解決不能を`unknown`へ退避 | 型名`Relation`を独立した業務Relation IDとして再解釈すること、直接Edge 0件から不存在を推論すること、Claim `qualified`への昇格 |
| **Synthetic composite view** | Fixtureが明示したRelation ID／宣言scope／role配列をGraph/Tableへ損失なく投影し、fixture内の`unknown`と限定的不在を区別 | 実CLI direct Edgeを仮想的な多項Relationへ「復元できた」こと、fixtureの`subjectsComplete=true`を実Source Readerの独立証明とすること |
| **Actual Source Closure / Disclosure** | 既存Source Index、Load、Traceの個別限定証拠と別Ownerの実Policy/Reader/Oracleの候補 | 実Sourceの全Root、Plugin未選択、外部File/Build/ETL、Current Grantが完全だという無条件の主張 |

この区別はEverything is Dataの**Role型／Instance／意味解釈／Source／実現・観測**の分離と一致するが、**合成の新Core型を採用したという意味ではない**。また #2831 Self-BuildのL1/L2/L3や#2542 ETL/MFTのArtifact／Transfer Attempt／ACK／Effectは、直接Graphがあることと異なるClaimであり、横断図表の被覆合格へ加算しない。

### 14.2 固定Workspaceの実証と再現手順

`reference/workspace`の`semantic-relation:sales.order.customer`について、既存CLIのfull/direct traceを取得し、別のNode Consumer `reference/tei-topology-coverage/verify-cli-trace.mjs`が独立Expectedとして以下を要求する。

- incoming：scope `reference-observation`の`application-pattern:order_entry.customer_reference` → `semantic-relation:sales.order.customer`。
- outgoing：同じscopeの`semantic-relation:sales.order.customer` → `semantic-data:customer.customer`。
- SourceNavigationは`located-navigation-only`まで。結果の`relationEntityIdentity`は`not-provided`、`multiRoleClosure`/`sourceAndPluginClosure`は`unproven`、`absentClaim`は`unknown`、`authority`は`read-only-candidate`から昇格しない。

専用[GitHub Actions #42205912384](https://github.com/hat47x/tei-platform/actions/runs/42205912384)はexact source head `6b99fed018dc9cc9c44bd0cee2031aecb1939a7b`で、**Go 1.26.10実CLI→Node観測 PASS**、Node 24.21.0のSynthetic Graph/Table **31/31**とTrace境界の合成負例 **7/7 PASS**、試験前後clean treeを確認。実CLIの既存Go試験を再実装したのではなく、**別言語のConsumerに届く輸送・表示前の意味の損失／過剰昇格**を最小限検証した。Go CLIから受け取ったPrivate Source path等の生JSONをPublic Consumerへ配布しない。

### 14.3 次段階の受入と優先順位

1. **P0・別Owner**：[#1124](https://github.com/hat47x/tei-platform/issues/1124)のM0実CLI→同一Workbench Consumer、M1のGraph/List/Initial/Command 0/1 Inspect、M2のSource/Grant別失効は先行。今回のNode直結成功を実Workbench/DOMやOwner認可の成功へ転用しない。
2. **P0並行・別Owner**：[#2831](https://github.com/hat47x/tei-platform/issues/2831)のGo module実閉包、実Build入力・Processとstage1駆動のre-entry、[#2542](https://github.com/hat47x/tei-platform/issues/2542)のMFT実Delivery／ImportのReceiptは別証拠として保持する。
3. **P1・本研究**：実Source Readerに**独立のRelation宣言ID・宣言scope・roleと端点**が存在する対象を特定する。実Go Index/SourceSpanの再読取で独立Expectedを作り、その要素だけGraph/Tableへ無損失に投影できるか確認する。無いものは`unknown/partial`。二つの異なるConsumerで不足が明確化するまで新Core構造を足さない。
4. **研究・別Owner**：[Draft #2966](https://github.com/hat47x/tei-platform/pull/2966)が人／AIの偽の不存在、到達操作数、Complect/変更波及を比較し、表示の理解負荷低減を仮説のまま測定する。今回の実CLI 2本のPASSからその優位性を推測しない。

**公開境界**：この実Sourceからの読取は実PolicyのDisclosureを判定しない。外部Consumer/AI/Studioへ実データを公開する場合は[#2516](https://github.com/hat47x/tei-platform/issues/2516)が資格化したpublic sliceを先に要求し、非公開Sourceの個数、Path、診断、roleを別経路から漏らさない。独立Policy/Source/Owner Oracleが欠けるClaimは、候補または`unknown`のままとする。

### 14.4 caller-provided scopeとSource-declared identityを分ける実CLI負例

固定Go `internal/modelref/derivedbridge/projection.go` の`FromIndex(d, scope)`は、**callerが指定したscope**をDerived Indexの各Node/Edgeへ付与する。ここにSourceが独立に宣言したscopeを読み取る処理はない。また`sourceindex.Reference`にはscopeが直接存在せず、`sourcebridge`は同kind/ID異scopeの衝突を安全側へ拒否する。したがって`TraceWorkspace`の`trace.subject.scope`が形式上qualifiedであっても、元Sourceの権威ある宣言scopeとの一致は**未証明**。

**実行した反例**：同じ実`reference/workspace`、同じ`semantic-relation:sales.order.customer`に対して、実`tei trace`を`-scope reference-observation`と`-scope alternate-observation`で**別々に**実行し、二つのJSONを独立Node Consumerで比較した。

| 独立に比較した事実 | 結果と意味 |
| --- | --- |
| 実CLIが返すDirect Edgeの`From/To`（scope以外）、Relation種別 | 同一。Sourceが同じ二つの観測で直接宣言参照は維持 |
| `WorldFingerprint`／`SourceIndexFingerprint` | 同一。Source/Worldの意味や版が変わったことではない |
| `trace.subject.scope`と各`From/ResolvedTo.scope` | Caller引数に応じて変更。実Sourceの自動発見scopeの証拠ではない |
| `SnapshotFingerprint`（selectorを含む） | 変更を確認。古いselectorの継続tokenを別scopeへ流用しない |
| `relationEntityIdentity`／`sourceAndPluginClosure`／`absentClaim` | 引き続き`not-provided`／`unproven`／`unknown`。scope変更後も昇格しない |

**exact HEAD証拠**：[GitHub Actions #42206842519](https://github.com/hat47x/tei-platform/actions/runs/42206842519)、head `8bf4347fa587a66bbc500a8a2fd5927d150437d2`、Go1.26.10実CLI二回＋Node24.21.0の受信比較 **PASS**、合成試験 **31+7 PASS**。追加後に継続する運用・CIは別資格。既存`TraceWorkspace`そのものを変更せず、read-only Reference Consumer側だけを修正した。

**不変条件／採用停止線**：`Source-declared scope`、`Projection caller-scope`、`Workspace/Observation scope`、`Principal/AuthZ scope`は同じ文字列であっても、生成元と証拠が異なる限り同一概念へ畳まない。画面の表示グループ、namespace filter、ラベル、実CLIの`-scope`からSource側Relation ID/declaring scopeを推測しない。製品のScope/Source identity契約を拡張する場合は#1120/#1124/#2620の既存Ownerへ、実Source Readerと独立Source anchor・必要閉包の証拠を渡して判断する。**新しいScope Registry/Resolver、正式Role Schemaはここで作らない**。

## 15. 2026-11-07：Intent／Realization／Assessment間のSource位置を同一Subjectへ束縛

**位置づけ**：[研究計画PR #2966](https://github.com/hat47x/tei-platform/pull/2966)（2026-11-07 main統合済み）のIntent／Realization／Assessmentを前提とした、既存read-only実CLI観測Referenceの**具体的な境界修正**。三視点Source設計 [#2878](https://github.com/hat47x/tei-platform/pull/2878) とPlugin最適化要件 [#2933](https://github.com/hat47x/tei-platform/pull/2933) も同日時点で統合済み。これらの研究内容を第二正本化せず、PR #2881が所有する実CLI→Node ConsumerのSourceナビゲーション誤帰属だけを扱う。確認main `680646b8629ce7af5b9bb0ba7db3529151b63a4c`（2026-11-07）。

### 15.1 成熟した小さな反例

既存Go [`sourcebridge.Navigation`](https://github.com/hat47x/tei-platform/blob/680646b8629ce7af5b9bb0ba7db3529151b63a4c/internal/modelref/sourcebridge/source.go) は `Key` と `Status`（located/unresolved）、`References`、`AuthorityLimits` を持つ。一方、修正前のReference `observeDirectTrace` は **`source.status === "located"`だけ**を確認し、`trace.subject`と`source.key`が別Subject／別scopeでも`sourceNavigation:located-navigation-only`を返すことができた。これは同じCLI JSONに含まれた**Realizationの誤帰属**であり、Graph/Viewのkey整合とは独立した問題である。

最小修正は、SourceNavigationの`key`を同一Traceの選択Subject（scope＋kind＋id）に照合し、SourceNavが`located`を名乗るなら空でない`References`を要求すること。未解決SourceNavは**直接Edgeの候補観測とは独立**して`unresolved`に保ち、Source位置や編集資格を捏造しない。型の不正な`ProjectionUnresolved`を「未解決0件」と解釈することも拒否する。

| 境界 | 修正後の局所結果 | なお要求される独立根拠 |
| --- | --- | --- |
| **Direct Edge candidate** | `tei trace`のsubject/方向・完全な1ページとread-only資格を検査 | 実業務Relation ID／多項Role／完全なSource・Plugin依存の証明なし |
| **Same-packet SourceNavigation** | Trace選択対象とNavigationのKeyが一致。LocatedならSource参照が少なくとも1件 | 同一時点の物理Source再読取・現在Grant・Authoring権限ではない |
| **Query/Closure** | 未対応Role/WHERE/Windowや不明なProjectionは`unknown`。非存在の断定なし | 要求入力の全Root、Readerによる実解析、独立Expected |
| **Disclosure** | 内部Referenceの結果を公開スライスと誤認しない | #2516のCurrent Policy／W0/W1二世界、実Producer→Consumer非干渉 |
| **Intent／Assessment** | 正本Intentと今回のbounded observed direct edgeを区別し、同じSubjectへの帰属だけ検査 | Accepted Meaning・Source/Version一致・独立受入者・正否の業務判断 |

この5段階は新たなCore enumではない。Role観測について #2620 が整理した **Inventoried → Read-by-Loader → Parsed/Anchored → Claim-Qualified → Currently Disclosable** を勝手に一本の成功フラグへ集約せず、Product Studio／AI／ETLと共通する**別Claimの集合**として扱う。特にSource IndexにNavigationが存在してもParser/Role意味が取得・受理されたとは判断しない。

### 15.2 検証と停止線

`direct-trace.test.mjs` に、(1)他Subjectのlocated Source Key、(2)同Subject ID／別scope、(3)Source Key欠落・Source未提供、(4)locatedだがSource参照が0、(5)unresolved Sourceは未解決のまま、(6)ProjectionUnresolvedの不正型を拒否する負例を加える。さらに実CLIで取得したJSONを**一箇所だけ意図的に変更**し、Navigation Keyの取り違えがNode Consumerで拒否されることを、実Go Source・実Consumer間のnegative controlとして確認する。

**証拠階層**：合成負例PASS＝局所パーサーの防御。実CLI JSONを変更した負例PASS＝同一Attemptの通信文書に対する防御。実Source Reader＋独立Oracleの成功＝別途必要。実Current Policy、WebView/DOM/ARIA、Owner Meaning Acceptance、実Build/ETL/DB、認知負荷の改善は **NOT PROVEN**。旧CIは新HEADの証拠に転用しない。

### 15.3 優先順位と他チャットの成果

#1124のM0実CLI→同一Workbench M1四入口／M2 stale UI・Grantを直接P0とし、本PRのNode側をWorkbenchの受入に流用しない。#2831はBuildのDeclared／Selected／ActuallyConsumedとStage1生成物の再入を別に実証。#2542/#2827はTransfer/ImportのReceipt・ACKと外部Effectを独立判定する。#2620のRole自己記述と#2972の公開三視点は独立Producer/Policyの資格を持ち、今回のSourceNavigation Key検証を役割意味・開示保証へ昇格しない。複合図表研究に新Resolver、Host、Policy Engine、Writer、関係の第二正本を導入しない。

## 16. 2026-11-12：Studioの問い相対な判断可能性・操作適格性（設計・受入候補）

**状態**：既存のScript-first Studio、Everything is Data、三視点／複合図表、Intent／Realization／Assessment、Current Disclosureを同一Consumerの利用判断へ接続する設計候補。新Core enum、全機能共通Verifier、Proof Store、第二Source・Graph・Resolver・Host・Runner・Acceptanceは導入しない。Ownerの採否判断・製品実装・E2E完了を主張しない。参照先は[#1124](https://github.com/hat47x/tei-platform/issues/1124)、[#2497](https://github.com/hat47x/tei-platform/issues/2497)、[Model View横断資格](../research/MODEL_VIEW_CROSS_CONSUMER_QUALIFICATION_2026-11.md)、[#2516](https://github.com/hat47x/tei-platform/issues/2516)とし、既存のGate／受入番号を置換しない。

### 16.1 表現できる・判断できる・操作できるは別Claim

- **表現可能性**：Value／Record／Sequence／Relation／Graph／Expression／Typed RefとRoleからGraph／Table／Tree／式／Sourceへ投影できる。Authoritative Sourceから解析したCanonical Worldと表示Occurrenceを別Identityとする。投影できた事実は、Required Source閉包とOwner意味受理を意味しない。
- **判断可能性**：問い Q ごとの対象、Scope、時点、Relation種、Universe、必要Reader・Source・Provider、partial／unresolved／opaqueを識別する。存在の限定陽性Witnessと、不存在を認定する全件閉包は非対称。全称を否定する一件の適格な反例と、全称成立を認定する適格な全件検証／不変条件も非対称。少数の正常SampleやGraphの線の欠落から不存在・全称を主張しない。
- **操作適格性**：一意なSource anchor・現在Revision、可逆性と保存条件、現在のDisclosure／Grant、Owner／Shared Acceptance、Target／Effect Authorityを操作単位で独立照合する。Mapのresolved、Catalogのvalidation OK、CI PASS、Artifact digestを編集・実行・業務受理へ昇格させない。

この三分類は新しい製品資格Schema／一括Verifiedフラグではない。**問い・Subject・Consumerを固定しない「図が完全」というグローバル表示は避ける。** 非公開Relationの有無、件数、Source pathや理由を低権限AudienceへGraph・DOM／ARIA・診断・Tool JSON・AI Contextから漏らさない。公開可能な「判断不能／この表示は対象外」と内部の不足理由を分離する。

### 16.2 同一Consumerでの最小Evidence Join（概念上の評価文脈。新APIではない）

| 分けて保持する項目 | 防ぎたい偽の短絡 |
| --- | --- |
| Q／Purpose、qualified Subject（kind＋id＋Source-owned scope）、表示Occurrence、caller-supplied scope | 同名異scope、表示位置／名称を正本Identityと取り違える |
| Workspace／Manifest、Source byte／Anchor、World／Index／Model Revision、Reader Attempt、selection・render epoch | Fingerprint一致／別Loadを原子的Snapshotと見なす。Source-only変更／ABAで旧位置を再利用する |
| Qに必要なRule／Relation／Source／Provider、Required・Declared・Selected・Actually Consumed、coverage／omitted／opaque | 未読込Writer、未選択Plugin、隠れたBuild工程、部分ETLを「存在しない」とする |
| Intent／Accepted Meaning、Resolved Realization、Artifact／Effect／Observation、独立Expected | 宣言・選択・実行・観測・意味受理を一つの成功段階へ畳む |
| Audience／Current Disclosure／Grant／Purposeと操作別Authority | 内部で読めることをSHO／AI／MCPへの現在開示許可に変える |

Catalog／Validate／Map／Locate等が同一fixtureを別Loadする場合、Fingerprintが一致しても同じ物理Root、同一Reader Attempt、同一Principal、現在Grantの証明にはならない。型付きModel Map M0のcoverageは部分投影であり、Event／Effect／Function／Business全体の不存在は未証明。[Draft #4392](https://github.com/hat47x/tei-platform/pull/4392)のSource-root-only errorとMap埋込Validation不一致はProducer側の既存修正候補を再利用し、Studioに第二Validatorを作らない。

### 16.3 操作導線と、表示済み情報・後着応答の独立失効

1. **View／Navigate**：現行の許可された範囲だけread-only表示。Occurrenceからqualified Subjectを解決してSourceへnavigate。未知のRole／Pluginでは汎用inspect／診断を優先し、未解釈ならunknownのままにする。
2. **Proposal／Preview**：一意かつ可逆なSource変更だけをtyped candidate化し、差分・影響・Source Revision・必要Validationを提示。Derived／集約／多対一／Macro／不明Providerの逆編集は拒否し、Source navigateに戻す。
3. **Accept／Apply**：既存Owner／Shared Acceptanceと現在Grantを通した場合のみApply。Sourceを再Readしてreprojectし、Source変更済み／Decision ledger失敗のpartial successは別readbackし自動再Acceptしない。実EffectとReceiptも別Ownerが観測する。
4. **Invalidate**：Source byte／位置、Plugin／Reader Pin、Disclosure／Grant、selection／render epochの変化を独立扱いする。失効を観測した際は、旧応答の後着拒否と**既表示DOM・ARIA・Graph・AI Contextの旧資格撤回を別々にassert**する。別Scopeの許可済み表示は不必要に全停止しない。変化をConsumerが観測できない期間の即時失効は保証しない。

### 16.4 他領域の問いを同じ標準Viewで扱う

| 対象 | 限定的に表示できるもの | 表示から推定してはならないもの／Owner |
| --- | --- | --- |
| 三視点・TEA | Source根拠のRelation／CRUD候補、Function／Businessへの関係 | 未読Function・外部Writerなし、Physical FKがOwner意味受理済み。#1124／#4487 |
| Subsystem分割 | 宣言済direct CUD、must-linkと制約相対の候補評価 | unknown＝越境CUDゼロ、最適Partition＝業務最適／Effect移管。#3250 |
| Plugin／Trait | Declared／Required／Selected／Resolved／実呼出の差 | 継承＝代替可能、宣言＝実Provider使用、Session間のGrant継承。#3279／#1251 |
| Self-build | 宣言Graph、実Consumed Input、実Build Attempt、Artifact digest | Artifact一致＝工程完全閉包、Stage収束＝Seed信頼。#2831 |
| ETL／Batch／MFT | Flow、Transfer Attempt、ACK、Import Unit、Business Receiptの別ID | ACK＝業務受理、partial＝全件完了、mixed-generation＝一式公開。#2542 |
| AI／Harness | 限定Selection Context、Diff、独立Expected | 選択文字列からSource／Grant推測、自己生成Expectedによる自己採点、自動Apply。#3262／#2514 |

### 16.5 既存Ownerに渡す最小反証（本節ではすべて NOT RUN）

| ケース | 変異・対照 | 独立Expected／既存担当 |
| --- | --- | --- |
| STQ-01 | Source-root-only errorとwarning-onlyを別Workspaceで比較 | transport OKとValidation OKを区別し、Catalog／Map／typed Validateで不一致を残さない。#4392 |
| STQ-02 | 同WorldでSource移動・byte変更・A→B→A、旧Inspect応答後着 | 現在Anchorを再資格化し、後着拒否と既表示失効を別assert。#1124／#2497 |
| STQ-03 | Public Slice同一、Private Relationのみ異なる二世界 | 公開Graph／件数／診断／Tool／AI ContextがPrivate有無を漏らさない。#2516 |
| STQ-04 | 宣言外Build工程あり、生成Artifact byteは一致 | 全Graph／実現一致を認定しない。Consumed／ObservedとPlanを照合。#2831 |
| STQ-05 | ETL二Memberの片方だけ新Generation、転送ACKは成功 | 一式・Import・業務完了を誤昇格しない。#2542／#2604 |
| STQ-06 | Subsystem direct CUDゼロ、未読Writer／間接Effectあり | 越境更新ゼロのClaimを保留し、疑似最適をOwner受理へ変えない。#3250 |
| STQ-07 | 非可逆Derived Relationの編集、または同名異Scope選択 | 変更対象を推測せず編集拒否。許可済みSource navigateのみ。#2497／#1124 |
| STQ-08 | AI Context作成後にGrant／Source Revision／Plugin Pin変更 | 旧Contextの現在資格を撤回しProposal→Applyを停止。#3262／#2516 |

**実証順**：P0は#4392のexact-head CIと#1124の実Workbenchでの旧画面・後着応答拒否を既存担当で継続。P1は実Catalog／Map／Locate／Traceの同一Source対照から、上記の問い相対のread-only表示・限定Evidence JoinをConsumerに接続する。Build・ETL・Subsystemは既存Ownerが実取得したEvidenceだけを再利用。P2は[#2018](https://github.com/hat47x/tei-platform/issues/2018)の同一Task／独立Expectedで、正答・false-qualified・Unknown保持、調査／編集／再試験時間、人手／Tool費用を比較する。Studioの図表完成を業務実Read／実Provider選択・Invoke／Self-buildの新Hard Gateにしない。抽象化で認知・検証負担が増えるなら限定read-onlyへ戻す。

**資格の上限**：この章は設計候補で、STQの成功、実Studio／Wails・Production Disclosure／Grant、実Provider／DB、Owner Meaning、認知負荷削減は未実証。既存PRの異なる局所PASSを一つの製品E2E成功に合算しない。

### 16.6 2026-11-12 実StudioのProducer→Consumer切断点（main静的照合）

**固定main** ea9e4560a835323c1d6d748486216b9ea011a965。確認対象：internal/operations/model_location.go、model_map.go、model_catalog.go、cmd/tei-studio/main.go、assets/client.mjs／main.js／preview-scope.mjs、internal/studiohost/command.go。この節は設計・受入候補であり、コード変更・実画面実証ではない。

- **既存Producer**：operations.LocateWorkspaceはloadCatalogReadの一回のWorkspace LoadからSubject／親・子・兄弟／typed Incoming・Outgoing、Total／Omitted、Coverage.Derived／NotDerived、Validation／Attention、World・Index・ModelRevisionを返す。これは他Operationとの同一Read Attempt・FS原子的Snapshotではない。operations.MapWorkspaceは別のread-only部分投影で、Coverageはpartial-projection。
- **切断点**：現StudioBridge／assets/client.mjsが公開するのはCatalog／Validate／Inspect／View／Command等で、MapWorkspaceとLocateWorkspaceの公開メソッドはない。assets/main.jsにも対応する実表示入口がない。Operationの存在だけで実Studio受信を主張しない。**初回はLocate-first**で一つの実Relationをread-only表示し、Graph Rendererや第二Model Storeを先行実装しない。
- **Validation**：現mainのMapWorkspaceはCatalogWorldの後にValidationを設定しないため、Map埋込Catalogの初期値をValidation失敗・成功の実根拠にしない。[Draft #4392](https://github.com/hat47x/tei-platform/pull/4392)が既存Producer側の修正担当。Locateは一回のcatalogWorkspaceLoadedでValidationを構成するが、それでも外部Provider・Owner Meaning・現在Grantを証明しない。
- **鮮度**：assets/preview-scope.mjsのcontext／sequenceは、画面の入力・Workspace変更で旧Promiseのpublishを拒否する仕組みであり、外部Source／Plugin／Policy／Grant変更の自動観測ではない。既描画DOM／ARIA／AI Contextも自動撤回しない。観測不能期間の即時失効を保証せず、再読取と別Ownerが提供する有効な失効通知を区別する。
- **Commandの資格**：StudioBridgeにはPlan／Execute系が存在する。internal/studiohost/command.goが扱うのはreference executorと非永続MemoryStoreで、画面由来allowed一覧に対するreference authorizer（空一覧ならreference-allow-all）を使う。ReadOnlyという実装名や画面のExecute成功を、正式な本番Grant／実DB Commit／Business Completeの証拠にしない。mode／durability／Effect／Receiptを別々に表示する。

### 16.7 同一Workbenchでの最初の縦切り（既存Ownerへ渡す受入）

**対象**：公開reference/workspaceの semantic-relation:sales.order.lines 一件。既存[#3865](https://github.com/hat47x/tei-platform/pull/3865) Model Locateと[#3866](https://github.com/hat47x/tei-platform/pull/3866)のtyped Map限定照合を再利用。Ownerは[#1124](https://github.com/hat47x/tei-platform/issues/1124)のConsumer実接続、[#1824](https://github.com/hat47x/tei-platform/issues/1824)の既存Workbench技術資産。Pin／Focus／Transient Selectionの契約は[Draft #4473](https://github.com/hat47x/tei-platform/pull/4473)／[Draft #4479](https://github.com/hat47x/tei-platform/pull/4479)に委譲し、第二Storeを設けない。

1. **Transport**：既存許可Workspace IDから既存rootを解決する薄いread-only Locate Adapterを使う。任意のfilesystem path・新Hostを公開せず、Subject kind/idとlimitを正確に渡し、失敗を既存分類へ委譲する。実CLI／Go Operationで応答できたことと、実Wails/Browser Consumerが同一要求・結果を受領したことを別に記録する。Hostは現実のAdapterであり、TEI全体の必須アーキテクチャにしない。
2. **View**：親Data→Relation→参照先Data、SourceStatus、typed role、Validation／Attention、Coverage.NotDerived、Total／Omitted、観測Revisionをread-only表示。Catalog Validation NGでも観察は許すが、編集・実行権限に昇格しない。公開可能なSource／診断だけを表示し、privateな存在・件数を漏らさない。
3. **Staleness**：Workspace／選択／Qの変更による旧応答後着拒否と、既表示の資格撤回は別assert。Source移動／A→B→A／Grant失効をProducerから実観測した場合の再資格化も別assert。実観測のない変更はunknownのまま、「常に最新」を名乗らない。
4. **Editing**：一意のSource Anchor／Revision、可逆なtyped patch、Preview、Shared Acceptance、適用後の独立Readbackが成立するまでは図面編集を完成扱いしない。実Business Effect／SHO Writer／AI外部送出は別Ownerで検証。

| Case（新規試験は全てNOT RUN） | 仕込み | 独立Expected／担当 |
| --- | --- | --- |
| STQ-09 | Go Locate成功、Studio BridgeにLocate未公開 | Producer成功と実Workbench受領を混同しない。#1124／#1824 |
| STQ-10 | Locate limit=1、未導出のQuery依存あり | Omitted／NotDerivedを保持し、非表示の辺を不存在としない。#3865／#1124 |
| STQ-11 | TypeRef Validation NG、Relationはindexed | Navigation成功＋Validation NGを並記し、編集・実行不可を維持。#4392／#1124 |
| STQ-12 | Source移動またはGrant失効を観測、旧Locate応答が後着 | 後着拒否と既描画撤回を独立検証。未観測期間の即時失効は未証明。#1124／#2516 |
| STQ-13 | Reference Commandのallowed入力／MemoryStoreを変更 | reference mode・非永続性を保持し本番認可・実DB Effectを主張しない。#2997／#1124 |

**資格と優先**：P0は#4392の既存Validation修正のexact-head照合と#1124の安全なSelection、P1は上記Locate-firstの実Consumer受領、P2はSource逆編集と問別の認知負担を検証する。実Wails／WebView／実DB・Provider、Current Disclosure／Owner Acceptance、原子的な複数Read、Source watcher、STQ-09〜13の新試験はNOT RUN／NOT PROVEN。別Graph正本、Runner、Host必須化、Verifier、Registryを追加しない。
