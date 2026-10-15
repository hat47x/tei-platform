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

**再描画時の注意:** #1849 Workbenchはノードを選ぶと同期的に`graph.update(related.input)`を行います。旧実装はその時点でフォーカス中のSVG `<g>` を削除しており、選択後にフォーカスが`body`へ抜け、次のSpaceキーが効きませんでした。更新前にフォーカス中のノードIDを取得し、更新後の同じIDへフォーカスを復元します。更新後にそのノードがない場合はスクロール領域へ復元します。別の入力へフォーカスがある場合は奪いません。

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
