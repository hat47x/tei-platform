# TEI Studio向けGraph表示部品 — PoC C試験実装

担当: [TEI Studio開発#9 / #1824](https://github.com/hat47x/tei-platform/issues/1824)。Studio画面とTEI以外の小アプリへ渡すための**読取り専用の表示部品**です。現在はReferenceであり、標準プラグイン・汎用Graphエンジン・正式なUI Composition Kitではありません。

## 入口と責務

- `graph.ts`: `mountReadonlyGraph(container, input, {onSelect})`。明示的なノード・エッジ・表示位置をSVGへ投影し、ノード選択IDだけを通知します。返り値は`update(nextInput)`、`setSelectedNode(id|null)`、`getSelectedNode()`、`dispose()`。
- `sample-graph.mjs`: **TEI知識が一切不要**な一般的なビルド工程の入力例。#8の別consumer用fixtureとして転用できます。
- `graph.test.mjs`: 入力拒否、文字列のHTML非解釈、更新/選択/破棄の小さいDOM代替試験。

入力例:

```ts
import {mountReadonlyGraph} from "./graph.ts";
import {buildPipelineGraph} from "./sample-graph.mjs";

const view = mountReadonlyGraph(document.getElementById("graph")!, buildPipelineGraph, {
  onSelect(nodeId) {
    // 表示上の選択通知のみ。実際の操作・許可はHost/Operation側で扱う。
    console.log(nodeId);
  },
});
// 選択中ノードを画面側で更新する場合:
// view.setSelectedNode("check");
// 破棄時: view.dispose();
```

入力の `nodes` は `{id,label,x,y}`、`edges` は `{id,source,target,label?}`、全体は `{width,height,nodes,edges}`。表示位置`x,y`はSVG viewBox内の座標であり、**Studioも別consumerも自分で用意**します。IDは画面用の厳密一致キーで、Canonical Subject、Workspace path、権限を意味しません。

## 自動配置を実装しない理由

初期PoCでは自作レイアウトを持たず、位置指定を必須とします。既存のGraph描画/配置OSSを#3と選定し、必要ならこの表示層を置き換えます。複雑なグラフの自動配置、線の交差回避、pan/zoom、ループ、複数エッジの経路、折りたたみ、編集等は対象外です。`source===target`の辺は表示誤りにせず `unsupported_self_edge` で拒否します。既存[#997](https://github.com/hat47x/tei-platform/issues/997)のSlot/renderer構成を置換せず、UI登録からAuthority/Actionを生成しません。

Host/Operation、TEIのSource Index、Go/Wails、Reactなどには依存しません。HTMLを組み立てる文字列を入力から生成せず、ノード名/辺名は`textContent`へ代入します。選択callbackを呼ぶ以外に副作用はなく、workspace読取り、shell、ファイル書込み、編集・受理を公開しません。型付き入力でも実行時の検証を省略しません。

## 幅240px前後のパネルでの可読性

#8のReading Journal（`reference/tei-studio-independent-consumer/`）は3列の表示領域が約240pxになるケースがある一方、`journalGraph()`の入力座標は`width: 480, height: 170`です。#Workが実ブラウザーで確認した初期実装では、SVGの`width=100%`が**480座標を240pxへ縮め、ラベルも約半分へ縮小**していました。ノード選択→picker・本文更新は確認済みでしたが、文字がほぼ読めない問題が残りました。

**このReferenceの表示方針**: SVGは`width:100%; min-width:<graph.width>px; max-width:none; height:auto`とし、グラフの座標1単位を**最低1 CSSピクセル**として扱います。ラベルの既定フォントサイズは14pxです。部品自身の親`div[data-graph-reference]`を`width:100%; max-width:100%; min-width:0; overflow-x:auto`のスクロール領域にし、Tabでフォーカス可能にします。**240pxの領域には480pxのGraphを縮小せず、横スクロールで閲覧**できます。480pxより広い領域では横幅に合わせて拡大できます。表示位置/辺ルーティングの再計算はしません。

- **#3 / Studio**: Graphの親パネルにグラフ全幅を収めるための`transform:scale()`や強制的な外側縮小をかけないでください。狭幅では部品のスクロール領域を使用し、実際のStudio画面で横スクロールと選択追従、マウス・キーボード操作を確認してください。
- **#8 / Reading Journal**: `journalGraph()`の480幅と3列レイアウトは変更不要です。約240pxのGraph slotでラベルの可読性、スクロール到達、選択後のpicker/本文連携、ページ破棄を実ブラウザーで再検証してください。Graph更新による再ビルドが必要です。実ブラウザーでの**修正版**表示確認は#9の単体試験だけでは証明されません。
- 入力の`width`・位置・エッジ配置はconsumerが決めます。長い**ノード**ラベルの表示省略と全文アクセスはこのReferenceが扱いますが、エッジラベルの長文、極端な座標端・極小viewBox、ラベル間の重なりは未解決です。選択IDの設定は表示状態の更新であり、スクロール位置への自動移動はしません。

## 読取り専用ノードのクリック領域

#Workの実ブラウザー試験では、幅対応後の `2d7148c3` で`clientWidth=225 / scrollWidth=480 / svgWidth=480 / labelFont=14px` を観測し、縮小問題の改善とEnterキーによるノード選択・picker/本文反映が確認されました。一方、**ポインターによるノード中央のクリックでは選択が変わらない**不具合を確認したため、これを別の未合格項目として扱います。

原因: SVGの`<g role="button">`自身には描画されたヒット領域がなく、`circle fill="none"`の場合、円内部はポインターイベントのターゲットになりません。このReferenceでは**輪郭はそのまま、`circle fill="transparent" pointer-events="all"`**とし、中心クリックから親`g`の既存`click`ハンドラーへイベントを伝播させます。`keydown`のEnter/Space、`onSelect`、選択状態・破棄の契約は変えません。透明塗りによるノード中央のクリック可能化は権限付与やTEI operationの実行ではありません。

追加した小DOM試験ではcircle属性、子から親へのclick伝播、Enter/Space選択を確認します。ただし**実際のSVGヒットテストやスクリーンリーダー対応の合格証明ではない**ため、#Workには修正後exact headを再取得して実ブラウザーでノード中央クリック・ロールbuttonクリック・選択/picker反映を改めて確認してもらいます。#3のStudio側の実画面再利用も別途未確認です。

## 長いノードラベルの表示方針

#9独立検証で、480pxのSVGに実測3037pxの長いノード名を入れると描画領域からはみ出し、横スクロールでも全文を読めない問題を再現しました（[Issue #1824](https://github.com/hat47x/tei-platform/issues/1824#issuecomment-6181365204)）。

- **表示**: ノードの横位置と入力幅から表示できる文字列を安全側に近似し、省略時だけ末尾を`…`にします。文字列は`textContent`で代入し、`x/y`・直線エッジ・レイアウトは変更しません。文字幅近似であり、あらゆるフォント・言語で表示領域内に厳密に収まる保証ではありません。
- **全文**: ノードの`aria-label`とSVG `<title>`には省略しない元ラベルを保持します。省略があるノードへキーボードでフォーカスするか選択すると、Graph領域内の`data-graph-full-label`へ全文を折り返して表示します。通常の短いラベルでは補足欄を表示しません。テキストは`innerHTML`に渡しません。
- **選択契約**: `node.id`、`getSelectedNode()`、クリック・Enter・Spaceで通知する`onSelect(nodeId)`は不変です。見た目のラベルを縮めてもsubject IDを変更しません。古いイベント・dispose後の通知抑止も不変です。
- **確認範囲**: TypeScript strict、Node DOM代替8件、Chromiumで狭幅225px・480px SVG・ノード長文の枠内表示・全文フォーカス・円中央クリック・Enter/Spaceの同一subject IDを確認。実Host接続画面、全フォント・スクリーンリーダー・Desktop WebViewは別の資格です。

## キーボード操作・アクセシブル名と再描画時のフォーカス

ノードはSVGの `<g role="button" tabindex="0">` として公開し、`aria-label`に省略前の `node.label`（空の場合は`node.id`）を設定します。選択状態は`aria-pressed`、Enter/Spaceの操作は既存`onSelect(node.id)`へ通知します。表示名の省略とsubject IDの選択を混同しません。

**再描画時の注意:** #1849 Workbenchはノードを選ぶと同期的に`graph.update(related.input)`を行います。旧実装はその時点でフォーカス中のSVG `<g>` を削除しており、選択後にフォーカスが`body`へ抜け、次のSpaceキーが効きませんでした。更新前にフォーカス中のノードIDを取得し、更新後の同じIDへフォーカスを復元します。**再描画でノード座標が画面外へ移った場合も、フォーカスしたノードが見えるよう通常のブラウザースクロールを許可**します（`preventScroll:true`は指定しません）。更新後にそのノードがない場合はスクロール領域へ復元します。別の入力へフォーカスがある場合は奪いません。

- **自動試験:** Node小DOM代替試験に`onSelect → update → setSelectedNode`の後も、Enterに続くSpaceが同じノードを選択し、消えたノードのフォーカスがスクロール領域へ戻る回帰を追加しました。
- **独立実Chromium DOM試験:** role=button、アクセシブル名、focusable、aria-pressedをChromiumのアクセシビリティツリーで確認し、Tab/Enter/Space/円中央クリックが同じIDを返すこと、再描画の前後でフォーカスが維持されることを確認します。
- **生成モード実画面の受入:** #1832 Host + #1833生成Go + #1849 Workbench + 本Graphのexact headを固定snapshotへ抽出して、実Wails ServerでGraphをTab選択、Enterによる`InspectSubject`表示後もフォーカスを保持し、Spaceで再選択できるかを別途確認してください。独立Chromium試験と生成モードの実Host画面は別の証拠であり、全a11yやスクリーンリーダー合格を意味しません。

## 検証

Node.js 22系（TypeScript型除去を明示）とTypeScript CLIが利用できる場合:

```sh
tsc --noEmit --strict --target es2022 --module esnext --lib es2022,dom reference/tei-studio-graph/graph.ts
node --experimental-strip-types --test reference/tei-studio-graph/graph.test.mjs
```

これらはTypeScript型とDOM呼出しの**小さい代替試験**であり、実ブラウザー、実WebView、Studio、OSS互換、2 consumerによる実画面再利用の証拠ではありません。

## #3・#8への引継ぎ

- **#3（画面統合）**: Graph用の実DOM containerを渡し、選択時の `nodeId` を画面の選択状態へ投影してください。Hostとの意味的な接続・UI Kitへの登録・依存やmanifest/lock変更は#3が管理します。UIマウント時に生成し、更新時は`update`、アンマウント時は`dispose`を呼んでください。
- **#8（別アプリでの再利用）**: `buildPipelineGraph`をそのままTEI非依存consumerに渡せます。実UIとGraph部品を同時に実行し、TEI importなしで選択・破棄ができる証拠を取得してください。fixtureだけでは2 consumer Gateを満たしません。

採用するGraph OSSのライセンス・版・更新負荷・SVG代替の評価、実ブラウザー/WebView試験、画面のフォーカス/IME/a11y検証は未実施です。

## #1124: Consumer側のqualified選択と描画epoch（設計候補）

> [#1124 第3追補](https://github.com/hat47x/tei-platform/issues/1124#issuecomment-6405343434)・[第4追補](https://github.com/hat47x/tei-platform/issues/1124#issuecomment-6405774769)の**Graph Consumer実装で補う条件**。ここで示すmap/guardは未実装・未実行であり、Graph表示部品の新しいTEI依存・Core ID・認可機構ではない。固定照合main: `b36ffdfd609bbc73bb1be93867576280c913f9d9`。

- **意味Identityの供給元**: `GraphNode.id`はview-only key。`modelref.Key{Scope,Subject}`、`Ref{Scope,Subject,Version}`は上流の**Scope確定済み**ModelRef Graph/Projectionから取得する。現行Workbenchの`Subject{kind,id}`、scope-less `Parent`、namespace絞込、ラベル、表示順、あるいは`subjectKey(kind,id)`からScopeやqualified edgeを推定しない。未確定・曖昧な対象は候補表示を維持し、操作対象として未解決にする。
- **画面内snapshot**: Consumerが描画ごとに`{renderEpoch, graph/source revisionまたはfingerprint, keyToViewID, viewIDToKey}`を所有し、1つのsnapshot内で表示IDとqualified Keyを一対一に対応させる。両mapは同じ検証済みProjectionから構築し、不足・重複・端点不明を暗黙補完しない。view IDはsnapshotを越える正本ID／永続ID／Source locator／権限IDではない。RefのversionとSourceの版照合はmapとは別に保持する。
- **失効と公開の順序**: Workspace切替だけでなく**同一Workspace内Graph差替**でも、差替受理時点で旧Consumer snapshotを**先に失効**させ、旧選択を停止する。新Graph・対応mapの検証と設置が完了してから新epochを一体として公開する。構築・更新失敗時は旧意味選択を自動復活させず、復旧する場合も新epochとして公開する。Graph部品が再描画後も同じ`node.id`の視覚選択・フォーカスを保持できることは、そのIDに紐づく**旧意味選択**を承認する理由にならない。
- **旧callback拒否**: 選択イベントに結び付けたsnapshot/epochが現在のactive snapshotと一致しないなら、`viewIDToKey.get(viewNodeID)`**より前**にrejectし、InspectもSource locator解決も起動しない。現行`graph.ts`の`generation !== revision`による旧SVGイベント拒否は**実装済み**だが、`GraphOptions.onSelect(nodeId: string)`はepochを渡さない。Consumerがイベントの元epochを確認するには、描画snapshotに結び付いたcallback/adapter（必要なら再mount）を設ける必要がある。可変な最新mapから旧IDを逆引きしてはならない。`responses.ts`の非同期応答世代は別責務。
- **操作境界**: Graph／一覧／初期自動選択／Commandは同じConsumer側`selectForReadOnly`相当の判定へ集約する。Graphの表示IDを`InspectSubject(kind,id)`、SourceBridge`Locate(key)`、Source path、Candidate／Acceptへ直接流したり、文字列解析で変換したりしない。論理RefのResolve成功はSourceの一意性・認可を意味しない。現在の`InspectSubject`はscopeなしの照会で、Catalogの`ambiguous`やSourceBridgeの`subject-scope-ambiguous`ではその呼出し・Source確定を保留する。Graph/Source Index/Readerは同じWorkspace revisionで照合し、Source locator成功もAccepted意味・書込み権限を与えない。

**最小Consumer回帰（未実行）**: 同じWorkspace Wにおいて、G1/epoch30の`view-node-0 → qualified Key(scope A)`をG2/epoch31の`view-node-0 → qualified Key(scope B)`へ差し替える。旧callbackを差替開始直後とG2公開後に呼んでも旧map lookup **0件**、Inspect **0件**、Source locator解決 **0件**。新callbackだけがscope BのKeyを得る。Source Indexがscope未確定なら`subject-scope-ambiguous`を表示してSource確定・scopeなしInspectは**0件**。Graphのview IDがInspect／Source locatorへの引数に現れない。既存`graph.test.mjs`の「stale events and disposed views」は表示部品の旧DOM拒否のみを実証し、このConsumerテストの代わりにはならない。


### 更新成功と構造検査拒否の分岐（#1124 [追加受入 #6426483819](https://github.com/hat47x/tei-platform/issues/1124#issuecomment-6426483819)、未実装）

**Consumer snapshotの安全側の順序**：次Graphのscope-qualified Projectionとmapを`pending`として組み立てる（**非公開**）→Graph差替を受理する時点で旧`active`をretire（選択不可）→`graph.update(nextInput)`を実行→**正常returnした場合のみ**Graphの描画と対応するpending map/epochを`active`としてpublishする。失敗したpending mapは破棄する。Consumerの選択epochはRenderer内部revisionの単なるコピーではない。旧描画を再び意味選択可能にするには、現在のGraph/Catalog/Source版を独立に再確認し、**新たな**snapshot/epochとそのcallback起源の拘束を作る必要がある。自動rollbackで旧mapを有効化しない。

| 独立ケース | Renderer側の事実 | Consumer側のmap／選択契約 |
| --- | --- | --- |
| **A: 正常update**（G1→有効なG2） | 新DOMへ置換、描画revision増分。旧DOMイベントは`generation !== revision`で拒否。見た目の同じnode IDの選択・focusを維持する場合がある | G2の配置成功後だけpending mapをpublish。旧epochのcallbackはlookup前に拒否。同一view ID `K`でも新Mapのqualified Key Bだけを選択し、表示IDをInspect/locator引数へ変換しない |
| **B: 構造検査拒否**（G2に`missing_endpoint`） | `normalizeGraph`が`replace`より前に例外。G1 DOM・Graph内部revision・表示選択は**変わらない**。残存G1ノードのイベントはRendererから`onSelect(K)`へ届き得る | G2 mapはpublishせず破棄。retire済みG1 mapも暗黙に復活させず、Consumerは`active=null`相当で選択を拒否。G1の閲覧は維持しても意味選択・Source確定は`stale-or-unresolved`相当 |

**最小Inspect spy受入（未作成／NOT RUN、Bを独立に検証）**：W固定。G1の可視`K → Key(scope one, semantic-data/shared.id)`を表示中、G2用に`K → Key(scope two, semantic-data/shared.id)`をpendingに用意する。ただしG2入力には存在しないedge endpointを混入する。更新拒否後に**旧DOMのKを実際にclick**し、①旧DOMが残存、Graph内部の旧選択状態が保持、②Rendererの`onSelect(K)`が**1回通知**（無効化された旧DOMイベントの試験ではない）、③Consumerの旧map・pending map双方の`get`は**各0回**、④`inspectSubject` spy **0回**、SourceBridge`Locate` spy **0回**、⑤G2のKey/epochをactiveへ公開しない、⑥未呼出しのSourceBridge診断を実際の`subject-scope-ambiguous`観測と偽らない、を同時にassertする。旧描画が現在のSource snapshotと一致することを別途実証できない限りInspectを再開しない。

**Aの対照条件**：別の正常なG2で同じview ID `K`を再利用した場合、旧DOMクリックは通知0、新DOMクリックだけがKey Bを選択する。**scopeなしInspect呼出し**は別途Catalog全体でBのCanonical Subject一意性を確認できた場合だけ許容（許容時はBの`kind/id`のみ1回でview IDなし）。同kind/IDが異scopeに重複するfixtureでは、新Mapの選択が成功してもInspect spyは**0回**とする。Renderer単体の既存`graph.test.mjs`だけではこのConsumer側のA/B判定を証明しない。

実装配置・受入は[#1124](https://github.com/hat47x/tei-platform/issues/1124)。型付きRef解決は既存[#1120](https://github.com/hat47x/tei-platform/issues/1120)を再利用し、TEI独自Hostを必須化しない（[Draft PR #2226](https://github.com/hat47x/tei-platform/pull/2226)、提案中・main未統合）。


## #1124: Qualified Selection Binding Reference（2026-11-01、独立試作）

[qualified-selection.mjs](qualified-selection.mjs) は、**現在のStudioへ接続していない**read-only Consumer Bindingの限定的なReferenceです。現行Graphはnodeの `onSelect(viewId)` しか通知せず、Workbenchの `main.ts` は同時点の `entries` を検索します。これだけでは異なるScopeの意味IDや、Graph更新拒否時の旧DOMと新候補mapの混在を防ぐ契約を実証できません。

- `observeSource({workspaceId, modelRevision})`: 観測したCatalog/Source文脈を指定。異なる版・Workspace、未確認への遷移は現行選択mapを失効。
- `stage({workspaceId, modelRevision, bindings:[{viewId,scope,kind,id,version?}]})`: 観測済み版だけを受け付け、Scope/Kind/IDを必須にする。複数Occurrenceが同一Subjectを指してよいが、同一Snapshot内の重複viewIdを拒否。ここではまだ選択mapを公開しない。
- `publish(token, () => graph.update(input))`: **同期・失敗時に旧描画を保持する表示置換**の成功後に限り、新mapを公開する。更新が `missing_endpoint` 等で拒否されれば旧mapのまま。Sourceがその間に変われば公開せずstaleとする。成功したRendererを持つ場合も現在のSource/Authority/Effectを保証しない。
- `select(viewId)`: `resolved` の場合のみ、`modelref.Key`相当の `{scope,subject:{kind,id},version?}` を返す。ほかは `stale` / `unbound`。**Source locatorやInspect実行をここで行わない**。呼出側が現在版・権限・開示条件を別途確認する。
- `invalidate()`: Sourceの現在性を証明できなくなった場合や再接続時に使う。Source改訂の通知が届かない場合の現在性までは証明できず、ABA/同時snapshotの問題もSource Ownerの独立readbackが必要。
- 初回はGraph node選択とList/Field行のbindingを同じ形式で渡す候補。**Field行クリック、Cross-view UI、実Studioへの結線は未実装**。

focused test（Node 22以上で `graph.ts` を型除去して動かす試験）:

```sh
node --experimental-strip-types --test reference/tei-studio-graph/qualified-selection.test.mjs
```

このReferenceの試験は、Graphの既存 `normalizeGraph` を通した失敗時の候補map非公開、R10→R11でのG1残存、Scope違い、同一Subject複数Occurrence、ABA、render中Source変化の**小さい単位試験**です。実DOM、WebView2、元X-TEA GUI、Shared Acceptance、外部UIのLocator/操作、実Providerは確認しません。意味上の「完全・不存在」は[#2352の複合図表設計](https://github.com/hat47x/tei-platform/pull/2352)と[TEA設計#2397](https://github.com/hat47x/tei-platform/pull/2397)の別Gateです。正式なConsumer受入は[#1124](https://github.com/hat47x/tei-platform/issues/1124)が所有します。
