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

入力の `nodes` は `{id,label,x,y}`、`edges` は `{id,source,target,label?}`、全体は `{width,height,nodes,edges}`。表示位置`x,y`はSVG viewBox内の座標であり、**Studioも別consumerも自分で用意**します。IDは画面用の厳密一致キーで、Canonial Subject、Workspace path、権限を意味しません。

## 自動配置を実装しない理由

初期PoCでは自作レイアウトを持たず、位置指定を必須とします。既存のGraph描画/配置OSSを#3と選定し、必要ならこの表示層を置き換えます。複雑なグラフの自動配置、線の交差回避、pan/zoom、ループ、複数エッジの経路、折りたたみ、編集等は対象外です。`source===target`の辺は表示誤りにせず `unsupported_self_edge` で拒否します。既存[#997](https://github.com/hat47x/tei-platform/issues/997)のSlot/renderer構成を置換せず、UI登録からAuthority/Actionを生成しません。

Host/Operation、TEIのSource Index、Go/Wails、Reactなどには依存しません。HTMLを組み立てる文字列を入力から生成せず、ノード名/辺名は`textContent`へ代入します。選択callbackを呼ぶ以外に副作用はなく、workspace読取り、shell、ファイル書込み、編集・受理を公開しません。型付き入力でも実行時の検証を省略しません。

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
