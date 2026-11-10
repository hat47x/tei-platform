# 標準データ構造と標準表現型：構造間Binding・媒体時間軸の設計案（2026-11-07）

> **統合Draftの位置づけ（2026-11-07）**：この資料は以前の設計Draftの内容を保存し、現行mainからの統合計画へ取り込み直したものです。最新の横断検証順と追加反例は `plan/design/COMPOSED_REPRESENTATION_MAPPING_QUALIFICATION_2026-11.md` を参照してください。本文中の旧Draft参照・NOT RUNは当時の証拠の限界として保持します。
- **状態**：Ownerから示されたX-TEA→TEIの構想に対する設計・実証計画。既存Core/公開Schemaへの採用、実装、今回の独立試験、実Target/Provider、意味受理は **NOT RUN / 未採用**。
- **目的**：データ構造と表現型を型として独立させ、表現インスタンスをFunctionへ配置し、その両側の構造間に検証可能な写像を設ける。RDBの1:1・1:NとXF単票/明細の対応を、文書・音声・動画・科学/空間/グラフ表現にも通用する概念へ拡張する。
- **責務Owner**：[データFacet #2495](https://github.com/hat47x/tei-platform/issues/2495)、[Pressure Corpus #2501](https://github.com/hat47x/tei-platform/issues/2501)、[Projection/損失/逆写像 #2497](https://github.com/hat47x/tei-platform/issues/2497)、[Function Context #2506](https://github.com/hat47x/tei-platform/issues/2506)、[Function Pattern #2507](https://github.com/hat47x/tei-platform/issues/2507)、[表示Kit #997](https://github.com/hat47x/tei-platform/issues/997)、[編集逆写像 #2500](https://github.com/hat47x/tei-platform/issues/2500)。本資料は**受渡し用の横断比較票**で、第二の型Registry、Projection Engine、Validator、Action/Authority、Scenario SSOTを設けない。

## 1. 4つの責務と存在条件

| 責務 | 型/定義の位置づけ | 具体的な利用時の位置づけ |
| --- | --- | --- |
| 標準データ構造（Data Structure） | 物理DBから独立した値・Record・Collection・Relation・Sequence・Hierarchy・Graph・Time・Space・N次元等の合成と意味Facet | Instance、Subject/Occurrence、grain、完全性、Revision、Data Provider/read scopeに束縛 |
| 標準表現型（Representation Type） | Panel/Tab/Text/Choice/Toggle/Slider/Table/Tree/Graph/Chart/Map/Timeline/Document/Audio/Video等、配置能力と必要条件を記述する再利用可能な型 | Palette/Previewは独立してよいが、**意味ある表現インスタンスは必ずFunctionの中に配置**。SHO固有Document Sessionなど別Consumerへ概念を転用する場合、TEI Functionであると偽称しない |
| Function / Data Context | 特定の用途・対象・Operationを組み立てる契約 | 複数Context、Result Slice、表示Instanceの所有、UI状態、Role・Actionの参照を管理。見た目から業務ProcedureやHost Grantを推定しない |
| Mapping / Projection | Source構造と表現Instanceの**二つの構造間の写像**。コンテナ包含・参照・選択・同期を含む | Source→Presentationの表示資格と、Presentation Edit→Action/Sourceへの逆写像を別に審査する |

**トポロジーの使い分け**：本書の「データ側の構成トポロジー」は要素/関係/多重度/順序/同一性、「表現側の配置トポロジー」はコンテナ・Slot・子要素・Navigation・選択・同期を指す。数学の位相空間としてのtopology、物理DBのFK、画面の視覚的な並びは同義ではない。1:1対応・同型性は必要条件ではない。

表現型には内部構造がある（Tableの行/列/セル、Videoのtrack/frame/時刻、Documentの段落/page/annotation）。ただし、その内部構造をDataの意味として自動採用しない。**表現型の機構はデータとして記述できても、入力・知覚・媒体ごとの制約はRepresentation Capabilityに残る**。

## 2. Mapping Contractに必要な論理列（新Core Schemaではない）

1. **選択**：Source Subject/Relation path/Occurrence、Context Instance、Query/Result Slice、投影対象のgrain・Identity・multiplicity・order・completeness・Disclosure。
2. **構造対応**：Data側node/edgeとRepresentation側slot/item/axis/selectionの対応、多対一・一対多・欠落の扱い。Tabと1:N、Table行とEntity、Tree表示PathとGraph nodeを同一視しない。
3. **値と意味**：Required Role、量の次元/単位、point vs delta、値域、測定尺度（nominal/ordinal/interval/ratio）、精度、丸め、NULL/false/未取得/非開示。
4. **軸と媒体**：時間/空間/順序/ページ/文字位置などの軸種、座標系/timebase、媒体の再生/描画能力、生成した軸が元データ由来か。
5. **変換証拠**：Conversion/aggregation/filter/flatten/temporalizationの演算と選択済みImplementation/Version、loss・uncertainty、Source→Targetのprovenance、変換対象の版。
6. **操作資格**：read-only/逆写像候補/明示Action経由を区別し、逆写像の曖昧さ・更新対象/Expected State/現在権限を別判断する。UI Edit enabledはCommit資格ではない。

互換性は（A）**構造・意味として実現できるか**（direct/explicit transform/partial/incompatible/unknown）、（B）**用途に適切か**（recommended/conditional/unsuitable/unknown）、（C）**特定Targetで実現されたか**（observed/unverified/unsupported）を分離。これらは論理的な比較列であって固定Enum採用ではない。**Authority、意味受理、実Effect、独立再読**はさらに別である。

## 3. 時間軸・ページ・文章の重要な非同一性

「時間を扱える表現型」と「時間単位を元から持つデータ」は異なる。少なくとも次を切り分ける。

| 軸 | 意味・例 | 自動同一視してはならないもの |
| --- | --- | --- |
| Source/Domain Time | センサーのサンプル時刻、イベント発生時刻、動画フレームのPresentation Timestamp、資料の有効期間 | 取込み時刻、画面の再生位置 |
| Observation/Processing Time | Provider取得時点、後着イベント、処理/受信時刻、clock domain | Domainでの発生時刻、完全な時間窓 |
| Presentation/Playback Time | 音声/動画の再生位置、アニメーション進行、朗読用の生成時間軸 | 元の章節・ページが本来持つ時間、業務有効時点 |
| Structural Position | 章/段落/文字Offset、Layout後のページ番号、Tree Path Occurrence | 秒・日時、原文をまたぐ安定ID |

**対応区分（仮説）**
- **軸保持**：既存のtimebase/単位を明示して等価に投影する。Audio waveformと音声再生、動画frame PTSと映像時間など。サンプルレート・可変Frame Rate・欠落・clock driftを無視しない。
- **明示変換**：サンプル時間→別sample rate、映像時刻→字幕・別trackのclockなど。変換率・origin・丸め・同期誤差が必要。
- **軸生成（Temporalization）**：文書を朗読/映像化する、ページ列を時間指定でスライド再生する。元の段落/文字範囲と**生成された時間区間**の対応、音声生成器/voice・速度・言語・版・手修正を保存。ページ番号はreflowで変わり、読上げ時間は速度/声で変わる。文書原本のSemantic Timeへ無断昇格しない。
- **意味の変換（Sonification等）**：数量→音高/音量、文章→読み上げ等。量の意味・正規化・聴取可能性・非可逆/情報損失を表示。値の大小と音高の関係は宣言した規則に限定。

Audio/Videoは「元データそのもの（録音原本・映像資料）」にも「表現の出力媒体」にもなる。Source媒体ファイルの同一性/バイト/時刻・著作権等と、Representation Instanceのレンダリング能力・生成された派生Assetを**別のSubject/来歴**として扱う。Mediaが保存できる事実から、媒体に映る現実・業務意味の正しさを推定しない。

**同期の最小契約**：Source領域（文字span/章節/サンプル区間等）↔Presentation Time Intervalは多対多・部分・不確実を許容し、特定Track/Version/clock domainに条件づける。TTS voice変更、文書再flow、字幕再編集、sample rate変更で同期資格を再評価。同期が不明ならseek/highlight/編集逆写像を無資格とし、Playback成功だけをSynchronization PASSとしない。

## 4. 受入に使う独立反例（計画のみ、すべてNOT RUN）

| ID | Source→表現 | 独立Expected：誤実装をどう落とすか | 主担当 |
| --- | --- | --- | --- |
| RP-01 | Order H、別々の子集合D1=2/D2=3→Panel+Table+Tab | 2集合をJOIN fanout=6行と誤認しない。Layout/Tab変更で対象集合や業務Actionを変えない | #2506/#2507 |
| RP-02 | 同一nodeへの2 Pathを持つDAG→Tree+Graph | Tree表示上の重複・非表示のedgeを保存。表示nodeを一意な変更対象や単親Graphとみなさない | #2497/#2500 |
| RP-03 | decimal 12.345、0.1 step、NULL/未取得→Slider/Text/Toggle | Sliderの丸めを無警告の編集許可にしない。false/NULL/unfetchedを二値へ畳まない | #2507/#1705 |
| RP-04 | 48kHz waveform＋可変frame-rate映像→Audio/Video | timebaseを明示し異Track同期に誤差を記録。frames/sample index≠domain wall-clock。未知clockで「同期済み」としない | #2497/#2501 |
| RP-05 | 2段落/3章の文章→ページ→TTS音声 | ページのreflowと朗読時間を原文の本質的時間としない。text span↔生成区間を保持し、voice変更でalignment失効 | #2497、SHO #19 |
| RP-06 | 数値時系列→音高/強度へ変換したSonification | 正規化/単位/範囲/scale/lossを宣言。音声からの元数値逆編集を自動許可しない | #2497/#2501 |
| RP-07 | 1/3ページ取得、別Tab未取得→Table/Chart | 可視件数≠対象母集団、集計1点≠元Entity、Tab未表示≠0件。SnapshotとScopeを持ち回る | #2506/#2514 |
| RP-08 | 同じ表現型を別Function/異Tenantに配置 | 表示Capabilityは共通でもContext/Scope/Selection epoch/Operation/Grantは別。Function跨ぎで古いAction資格を再利用しない | #2507/#2598/#1174 |

固定Source/fixtureと独立Expectedを既存[#2514](https://github.com/hat47x/tei-platform/issues/2514)/[#1180](https://github.com/hat47x/tei-platform/issues/1180)に受け渡す。本書に第二Scenario/Expectedの正本を作らない。特に媒体生成をテストする場合は**同一Source版・同一生成器版・決定的な合成時間対応**を固定し、実TTS品質・実再生・実字幕同期とはClaimsを分ける。

## 5. 計画への接続・順序

- **P0 分類比較（#2495/#2501）**：既存Facetを用い、Structural position、source/observed/presentation time、Media Source vs Media Output、Representation topologyを比較票へ追加。新Core型は不要か圧力で判定。
- **P1 Read-only Projection（#2497/#2506）**：RP-01/02/04/05/07の少数対照で、双方向のstructural mappingの粒度・Source Revision・情報損失・Axis provenanceを説明できるか検証。初手はHeadless、静的Media alignmentでもよい。
- **P2 Pattern/Renderer（#2507/#997）**：同一FunctionのPanel/Tab/Table/Timelineや異Functionを、型の定義・Instance所有・bindingを分離してConsumerへ提示。別Target/SHO/SUIに移しても意味の正本を増やさない。
- **P3 Edit/Effect（#2500/#2597/#2598/#1174）**：Read-only mappingを一律に編集可能へ昇格させず、可逆・条件付き逆写像と現在の実Target/Grant/Receipt/独立readを個別検証。
- **P4 横断展開（#1803・SHO #19・SUIの該当Owner）**：文書レンダリングとMedia生成の差分を収集。SHOのRaw Markdown保存とTEIのMeaning Acceptance、SUI表示と実Actionを混同しない。

**停止条件**：Axis origin・clock・Source span・grain・Identity・partial/disclosure・変換Impl版が必要なのに未確定なら、推奨/逆編集/意味受理/完了を推測しない。静的Previewは構造適合の証明でなく、実UI操作/Media再生/Provider effectの証拠ではない。

**残る判断**：media-timeの正式な名称と公開Schema、同期誤差の定量基準、媒体変換Pluginと文書Producerの契約境界、表現型のアクセシビリティ適性/知覚的負荷、実Target対応の採否。Ownerの意味・権限判断とともに別票で保持する。
