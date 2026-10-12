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
