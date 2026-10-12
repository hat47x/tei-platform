# DFD編集候補の受け渡し（#1125）

状態: **Reference限定の未受理候補**。この経路はBusiness Procedureを正本として編集する機能ではなく、DFDレビュー用の`authority=proposal`データに対して、意図した変更だけを作るための橋渡しです。

## 一つの操作を小さく保つ

`dfd-edit-proposal.mjs`は、すでに存在する業務Stepと図レビュー案を入力とし、次の4操作だけを認めます。

| 操作 | 処理する意味 | 保持する境界 |
| --- | --- | --- |
| `assign-business-role` | 既存Stepへ既存業務Roleの責任候補を追加 | Authorization Roleへ自動昇格しない |
| `replace-trigger-label` | 明示した契機の説明を変更 | Trigger ID、種類、Runtime Bindingを変更しない |
| `add-control-transition` | 手順間の順序・分岐・例外候補を追加 | 情報Flowと混同しない。分岐・例外は条件必須 |
| `add-information-flow` | From / To / payloadを持つ情報授受候補を追加 | 少なくとも一端をStepとし、外部主体→Storeの直接操作を仮定しない |

いずれも、対象IDの存在、重複、許可されたキー、参照の種類を事前点検します。任意のJSON Patch、Procedure Stepの削除、既存Canonical Sourceの変更、Role→RBAC、Runtime Trigger、DBへの書込みは認めません。エラー時は使える候補を返さず停止します。

結果の `schema=tei.reference.business-dfd-edit-preview/v0` は、**Shared AcceptanceのChange Candidateではありません**。常に`authority=unaccepted-proposal-only`、`readyForAcceptance=false`、`sourceWrites=[]`、`acceptanceStatus=not-accepted`です。

## 一往復の再現手順

リポジトリルートを作業ディレクトリとして、次の操作を行います。書込み先は一時ディレクトリだけです。

```sh
scratch="$(mktemp -d)"
cp reference/business-system-design/dfd-order-approval/order-approval-procedure.json "$scratch/procedure.json"
cp reference/business-system-design/dfd-order-approval/review-proposal.json "$scratch/review.json"
cp reference/business-system-design/dfd-order-approval/procedure-set.json "$scratch/procedure-set.json"

node --input-type=module - "$scratch/review.json" "$scratch/edit.json" <<'JS'
import {readFileSync,writeFileSync} from "node:fs";
import {createHash} from "node:crypto";
const [review,editFile]=process.argv.slice(2);
const sha="sha256:"+createHash("sha256").update(readFileSync(review)).digest("hex");
writeFileSync(editFile,JSON.stringify({
  schema:"tei.reference.business-dfd-edit/v0",
  id:"assign-submit-approver",
  origin:"studio",
  intent:"受注コマンドを依頼する業務Stepに承認担当の責任候補を追記する",
  baseReviewSHA256:sha,
  procedureId:"order-entry.approval-and-shipment-prep",
  operation:{kind:"assign-business-role",step:"APP-40",role:"approver"}
},null,2)+"\n");
JS

# 現在のレビュー案の実際のSHA-256をNode CLI側で再計算する。
# Procedure・review・editを再読込みし、バイト列が変われば失敗する。
node reference/business-system-design/dfd-edit-cli.mjs \
  -procedure "$scratch/procedure.json" \
  -review "$scratch/review.json" \
  -edit "$scratch/edit.json" > "$scratch/preview.json"

# 出力の候補レビュー案だけをScratchへ書く（accepted Sourceではない）。
node - "$scratch/preview.json" "$scratch/review-next.json" <<'JS'
const fs=require("node:fs");
const p=JSON.parse(fs.readFileSync(process.argv[2],"utf8"));
fs.writeFileSync(process.argv[3],JSON.stringify(p.proposedReview,null,2)+"\n");
JS

# 2つの未受理レビュー案の差分をGoの既存Reference checkerで点検する。
go run ./tools/reference-business-dfd \
  -procedure "$scratch/procedure.json" \
  -review "$scratch/review.json" \
  -review-next "$scratch/review-next.json" \
  -set "$scratch/procedure-set.json" \
  -format json > "$scratch/diff.json"

# ここで差分・未解決診断を人が検討する。自動Acceptはしない。
```

`baseReviewSHA256`は入力ファイルの実バイト列で検証されます。純粋関数`proposeDFDReviewEdit`を単独で使う場合、引数の現在SHAは**呼び出し側提供値**にすぎません。CLIでは実際にハッシュを計算しますが、それでもWorkspaceのCanonical World / Source Index / Shared Acceptanceのrevisionを保証しません。

## 意味の混入を拒否する

未受理のレビュー案では、要素ごとに次の項目だけを認めます。ここにない項目は、内容を解釈して補うのではなく、入力を拒否します。

| 要素 | 許可する項目 |
| --- | --- |
| 起動契機 | `id`、`kind`、`targetStep`、`label` |
| 順序・分岐・例外 | `from`、`to`、`kind`、`condition` |
| 業務上の役割・外部主体 | `id`、`label` |
| 概念上の情報保管先 | `id`、`label`、任意の`semanticData` |
| 業務上の担当 | `step`、`role` |
| 情報の授受 | `id`、`from`、`to`、`payload` |

単なる順序の矢印に条件を付けることも拒否します。認可設定、時刻による実行設定、データベースの更新指示などは、この業務レビュー案の属性ではありません。将来それらを扱う場合も、実行側の契約を明示的に参照し、別途検証する必要があります。既存のレビュー案を変更する操作で、未知の属性をそのまま次の候補へ運ぶことを防ぎます。

二つの既存受注ケースについて、通常の変更候補と、要素ごとの不適切な属性の混入を試す回帰試験を追加しました。

```sh
node --test reference/business-system-design/dfd-edit-proposal.test.mjs \
  reference/business-system-design/dfd-edit-boundary.test.mjs
```

これは**レビュー候補の形式検査**です。正本で採用された業務上の意味や、実行権限の妥当性を証明するものではありません。

## Studio・Typed Referenceとの統合条件

1. **選択**: #1125の`-select`が返す`reference-procedure:<procedureId>`のStepとSource位置を表示する。同名のAccepted Taskと同一視しない。
2. **編集意図**: 上の明示操作として一時的なレビュー候補を構成する。図の位置・線の形・見た目から意味変更を逆推定しない。
3. **差分レビュー**: 旧レビュー案・新レビュー案と元のProcedure/Handoffの指紋を照合し、未解決を示す。
4. **正本の更新**: 今回は**未実装**。#1124のSource Acceptanceと#1120の型付きRefにより、同一対象・同一revision・受理主体を確認できた場合だけ別契約に変換する。単にこの候補ファイルをSourceへコピーする操作は受理とは呼ばない。
5. **再投影**: 将来Accepted Sourceを変更した場合に限り、Workspaceを再ロードしData / Function / Businessを同一revisionから取り直す。

## 実行Evidenceの扱い

```sh
node --test reference/business-system-design/dfd-edit-proposal.test.mjs
node --test reference/business-system-design/dfd-edit-cli.test.mjs
```

GitHubへの追加時点では上記Nodeテストをfull checkout上で実行した証跡はありません。V8に読み込んだ実モジュールと既存Reference fixtureを用いた、4件の正例と7件の否定ケースの限定スモークのみPASSです（NodeのファイルI/O、SHA実測、Go checker実行を含みません）。

新しいProvider、Micro Kernel kind、Canonical Procedure、Workflow engine、Source書込み権限は追加しません。
