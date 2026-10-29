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

### 3.3 編集可能性の非対称性

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
- [PR #2226](https://github.com/hat47x/tei-platform/pull/2226): TEI独自Host非必須の未統合Draft。旧Host必須案を復活しない。
- [PR #2298](https://github.com/hat47x/tei-platform/pull/2298): `tei.*`名前空間、任意Composition、Read Projection編集非対称性の未採用Draft。正式な標準名前空間・Binding構文とみなさない。

**未決（ここでCoreへ固定しない）**: トポロジー標準の最小primitive集合、Projection/Correspondenceの具体API、可逆編集の証明条件、全種類の動的Graphのclosure表現、複合図表レイアウト仕様、標準の唯一の図表UI、メタ生成の深さ・専用DSL。

次の実作業の提案はP0→P1のread-only Referenceである。最初からCore型・Studioフック・新ビルドエンジンを増やさず、複合図表の価値と不足する構造を観測する。
