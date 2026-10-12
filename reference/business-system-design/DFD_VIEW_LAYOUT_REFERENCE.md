# 業務DFDの配置情報を独立させる

#1125の一階層DFDに対し、図の座標を**業務手順・機能・データの意味から分離**するための参照実装です。既存のDFD生成器や正本を変更しません。

## 役割と範囲

`dfd-view-layout.mjs`は、既存の`business-dfd-projection-v0`と、独立した表示配置のJSONを受け取ります。最初は空の配置から始め、`move-node`だけを受け付けます。位置を与えられるのは、そのDFDに現れるStep、起動契機、業務上の役割、外部主体、概念Store、明示的なFunction参照だけです。

- 配置の形式は`tei.reference.business-dfd-view-layout/v0`、`authority=view-only`とします。
- 座標は`kind`、`id`、`x`、`y`だけです。識別子は元のDFDに厳密一致する必要があります。
- 表示配置には対象ProcedureとDFD投影のSHA-256を記録します。古い投影、重複した位置、未知の要素、範囲外や数値ではない座標を拒否します。
- 結果は`tei.reference.business-dfd-layout-preview/v0`です。`sourceWrites=[]`、`canonicalMeaningChanged=false`、`readyForAcceptance=false`を固定し、正本を変更しません。
- アプリケーションの認可、業務上の承認、Source Acceptance、実行時の権限を付与しません。

これは**図の表示情報を受け渡すための候補**であって、Studioのドラッグ操作や表示エンジンまで実装したものではありません。元のDFDは、依然として非Canonicalな参照手順を元に作られています。

## 一時領域での確認

リポジトリのルートから実行します。

```sh
scratch="$(mktemp -d)"

go run ./tools/reference-business-dfd \
  -procedure reference/business-system-design/dfd-order-approval/order-approval-procedure.json \
  -review reference/business-system-design/dfd-order-approval/review-proposal.json \
  -set reference/business-system-design/dfd-order-approval/procedure-set.json \
  -format json > "$scratch/dfd.json"

node - "$scratch/dfd.json" "$scratch/layout.json" "$scratch/edit.json" <<'JS'
const fs = require('node:fs');
const crypto = require('node:crypto');
const [dfdPath,layoutPath,editPath] = process.argv.slice(2);
const digest = bytes => 'sha256:' + crypto.createHash('sha256').update(bytes).digest('hex');
const dfd = fs.readFileSync(dfdPath);
const procedureId = JSON.parse(dfd).procedureId;
const layout = {
  schemaVersion:'tei.reference.business-dfd-view-layout/v0',
  authority:'view-only',
  procedureId,
  projectionSHA256:digest(dfd),
  nodes:[]
};
fs.writeFileSync(layoutPath,JSON.stringify(layout,null,2)+'\n');
fs.writeFileSync(editPath,JSON.stringify({
  schema:'tei.reference.business-dfd-layout-edit/v0',
  id:'move-phone-review',
  origin:'studio',
  intent:'電話確認の位置を見やすくする',
  procedureId,
  projectionSHA256:digest(dfd),
  baseLayoutSHA256:digest(fs.readFileSync(layoutPath)),
  operation:{kind:'move-node',nodeKind:'step',nodeId:'APP-10',x:120,y:80}
},null,2)+'\n');
JS

node reference/business-system-design/dfd-view-layout-cli.mjs \
  -projection "$scratch/dfd.json" \
  -layout "$scratch/layout.json" \
  -edit "$scratch/edit.json" > "$scratch/layout-preview.json"
```

この操作は入力ファイルの実バイト列を読み取り、実行中に変わっていないことを再確認します。更新済みの配置を永続化する操作や、受理済みの業務モデルとの版一致は提供しません。投影の版を提示する責任と、その真正性の確認は上位の呼び出し元に残ります。

## 試験

```sh
node --test reference/business-system-design/dfd-view-layout.test.mjs \
  reference/business-system-design/dfd-view-layout-cli.test.mjs
```

2026年10月12日、独立したNode.js 22.16.0環境で7件の試験がすべてPASSしました。GitHubに登録した4ファイルのBlob SHAが、試験に使用した4ファイルと一致することも確認しています。ただし、**これをTEI全体の検証とは扱いません**。信頼できる全リポジトリのチェックアウトによる`tools/repository-health-check.sh`、既存Goテスト、Studio実画面の動作確認は別途必要です。

## 参照画面での操作（追加）

`dfd-view-studio.html`と`dfd-view-studio.mjs`は、既存のDFD投影を読み込み、図の配置を一時的に操作するための小さなブラウザー用参照画面です。Studio本体への正式な組込みや、正本編集画面としての利用は想定していません。

リポジトリのルートで、前節のGoコマンドを用いて`$scratch/dfd.json`を作成した後、次のように静的ファイルを公開します。

```sh
python3 -m http.server 8765
```

ブラウザーで `http://localhost:8765/reference/business-system-design/dfd-view-studio.html` を開きます。まず「DFD投影」から作成したJSONを選択します。必要に応じて、保存済みの配置ファイルも選択できます。要素をクリックすると、参照元の位置・Function・直接意味参照・条件付きHandoffを表示します。明示されたControl／起動契機／担当業務Role／Function参照／情報の授受だけを元に、図中の関連要素へのリンクを表示します。例えばFunctionからそれを明示参照するStepへ戻り、StepからFunctionやRoleへ辿れます。外部主体・概念Storeを選択した場合も、明示された情報授受の相手Stepと参照元位置を確認できます。StoreのSemantic Data IDは参照文字列を示すだけで、実DBや受理済みDataへの権限を示しません。古い投影で生成したリンクは投影差替え後に使えません。

図の上部では、投影器が生成した診断コード・重要度・SourceファイルとJSON Pointerを表示します。未参照Functionは**当該Procedureの範囲で明示参照がないという確認事項**であり、Application全体で不要とは判定しません。`review-question`や`warning`は非実行の参照を許容しますが、`error`を含むDFD投影は図・エクスポート・変更候補を禁止し、診断の出典表示だけを残します。出典位置は投影器の自己申告であり、受理済みの同一revision Source Indexとの一致を確認した証拠ではありません。DFD投影の解除・差替え時に古い診断表示も破棄します。

要素をドラッグすると、座標だけの変更候補が表示されます。キーボードではTabで要素を選び、Enterで詳細を確認し、矢印キーで20単位、Shift＋矢印キーで1単位ずつ移動候補を作れます。キャンセルされたドラッグや、別のポインターによる操作は移動として扱いません。**「配置だけを反映」**を選んでも変更されるのはブラウザー内の一時表示だけです。配置ファイルを保存する操作も、業務手順やCanonical Sourceは変更しません。初回表示時の並びは簡易的な格子配置であり、DFD上の業務順序や役割分担を意味しません。Stepが多い場合は画面領域を拡張し、初期領域外の要素もスクロールして表示します。存在しないStepへの順序矢印、参照先のない起動契機・責任割当て・情報Flow、重複した矢印・責任割当て、Stepを一つも持たない手順は、省略して描かず、投影全体を拒否します。

WorldのJSONと版を利用者が入力した場合に限り、#1124の既存`navigateReferenceBusinessStep`を読み取り専用で呼び出します。表示される参照は識別子の照合候補であって、受理済みの業務Stepや最新Workspaceとの版一致を証明しません。入力した版文字列だけでSource Acceptanceを認めることはありません。Worldの再選択・選択解除と版文字列の変更時には選択済み要素の参照表示・未受理候補を無効化し、Worldの古い非同期読込み結果を採用しません。新しい参照結果を見るには要素を選び直します。

### 表示上の区別

- 灰色の矢印：レビュー案に記述された順序
- 茶色の矢印：条件付きの分岐（条件を併記）
- 赤色の矢印：例外経路（条件を併記）
- 青緑色の破線矢印：明示された情報の授受（内容を併記）
- 紫色の点線矢印：業務上の責任割当てまたはFunction参照
- 黄色のノード：Application Functionを必要としない作業

倉庫など別ProcedureへのHandoffは、現時点では選択したStepの詳細に表示し、同じ画面の自動実行経路としては描きません。すべてが参照または未受理候補であり、実際に処理が完了したことは示しません。

### 試験と残る検証

```sh
node --test reference/business-system-design/dfd-view-studio.test.mjs
node --test reference/business-system-design/dfd-view-layout.test.mjs \
  reference/business-system-design/dfd-view-layout-cli.test.mjs
```

画面ロジックは模擬DOMによるNode.js試験で、投影の読込み・選択・移動プレビュー・一時反映・古い配置の拒否を確認できます。実ブラウザーによる操作試験は別途必要です。画面の操作・入力JSON・ローカルのJavaScriptがそのまま信頼できるStudio製品実装であるとは主張しません。全リポジトリでのGoテストと`tools/repository-health-check.sh`も引き続き統合前の必須確認事項です。

追加の回帰試験では、参照先の欠落、分岐と例外の矢印、キーボードからの座標変更、ポインター中断、多数Stepの表示範囲、Procedure切替え時の出典表示の破棄を検査します。いずれも参照画面内の動作を対象とし、実ブラウザーでの操作品質や、正本の編集権限を確認した証拠ではありません。

2026年10月12日追補：この参照画面の更新後、GitHub上のJavaScriptをV8の模擬DOMで直接評価し、読み込み、矢印の種類と条件、キーボード操作、一時反映、長い図、誤参照の拒否、Procedure切替え時の状態破棄の8項目がすべてPASSしました。暗号学的な指紋の照合・実ブラウザーの描画・リポジトリ全体の試験は含みません。
