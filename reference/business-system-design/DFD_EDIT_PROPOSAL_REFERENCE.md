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

## 図の選択と変更対象を結び付ける

`dfd-selected-edit-bridge.mjs`は、図で選択した要素から、既存の限定的な業務レビュー変更を作成するための接続です。図の入力、元の業務手順、元のレビュー案の三者を照合します。

- 業務手順のID、StepのID・表示内容・Function参照、起動契機、遷移、業務Role、外部主体、概念保管先、責任割当て、情報の授受が食い違えば変更候補を作成しません。
- 選択したStepへの担当割当て、選択した起動契機のラベル変更、選択したStepからの遷移追加、選択した端点からの情報の授受追加を認めます。それ以外の編集や、別要素へのすり替えは拒否します。
- 戻り値は`tei.reference.business-dfd-selected-review/v0`であり、`sharedAcceptanceCandidate=null`、`readyForAcceptance=false`、`sourceWrites=[]`です。選択したStepのScopeは`reference-procedure:<手順ID>`で、採用済みの意味IDではありません。
- この照合だけでは、受理済みWorldやSource Indexとの版一致は成立しません。同じID・同じ内容でも、採用済みの業務仕様だと認めてはなりません。

`dfd-selected-edit-cli.mjs`は四つの入力ファイル（DFD投影、業務手順、レビュー案、変更指示）の実バイトからSHA-256を計算し、指示に含まれる三つの指紋と一致しなければ拒否します。変更候補生成後にも、四つのファイルが途中で変わっていないことを確認します。**このCLIは標準出力へ未受理の候補を出すだけで、いずれのファイルも書き換えません。**

実行例（すべて一時領域に保存）：

```sh
node reference/business-system-design/dfd-selected-edit-cli.mjs \
  -projection "$scratch/dfd.json" \
  -procedure "$scratch/procedure.json" \
  -review "$scratch/review.json" \
  -binding "$scratch/selected-edit.json" > "$scratch/selected-preview.json"

node --test \
  reference/business-system-design/dfd-selected-edit-bridge.test.mjs \
  reference/business-system-design/dfd-selected-edit-cli.test.mjs
```

`selected-edit.json`は`schema=tei.reference.business-dfd-selected-edit/v0`と、`projectionSHA256`、`procedureSHA256`、`reviewSHA256`、`selection:{kind,id}`、`operation`、`id`、`origin`、`intent`を明示します。SHAは各ファイルの実バイトから作成してください。選択や画面配置から自動的に編集対象を推測しないことが重要です。

この接続の次段階は、#1124が受理済みBusiness Procedureの正本と同一版のSource位置を認証した後に、**別の正式な候補**として受理可否を審査することです。未受理のDFDレビュー案をShared Acceptance候補へ自動変換することは認めません。

### 投影器が付けた状態も照合する

図上の識別子や業務上の文言が元の手順と一致していても、投影器が付けた状態を無条件に信頼しません。選択から変更候補への接続では、次を確認します。

- 手作業かどうかが、元の手順で宣言されたApplication Function参照の有無と一致すること。
- 明示されたFunction参照の解決状態が`resolved`であること。
- 業務Roleから物理的な認可Roleを推測していないこと。
- 起動契機の状態が`proposal`、実行への接続が`unbound`であること。
- 順序・分岐・例外、業務Role、外部主体、概念保管先、担当、情報の授受が、いずれも`proposal`として示されること。
- 概念保管先のData参照について、宣言がなければ`not-declared`、明示された場合は`resolved`と示されること。

食い違った場合は未受理の変更候補であっても生成しません。ただし`resolved`は参照実装内での一致を意味するだけで、採用済みWorldの同一版や、情報の鮮度を証明しません。Go生成器が使う状態値に合わせ、模擬DFDを用いた回帰試験でも同じ区別を保持します。

### 図の選択元を参照資料の位置へ照合する

変更の対象を選んだ際、その要素が指す出典の位置も検査します。Stepなら元の業務手順の`/steps/番号`、起動契機ならレビュー案の`/triggers/番号`、外部主体なら`/externalParties/番号`、概念保管先なら`/stores/番号`に一致する必要があります。同じ資料に属する要素のファイル名が一つだけ異なる場合も拒否します。

結果には`sourceLocatorStatus=reference-local-pointer-checked`と`sourceFileAuthority=unverified`を明示します。ここでの照合は、現在の参照資料内の要素を取り違えないための検査です。**ファイルの真正性や、Source Index上の採用済み版との一致を認証するものではありません。** 保存先や受理対象のファイルパスとして、この出典を無条件に利用してはなりません。

## 参照画面からの業務変更プレビュー

`dfd-view-studio.html`では、DFD投影に加え、元の業務手順と未受理レビュー案のJSONを選択できます。図上の起動契機を選択すると説明文の変更候補を、業務Stepを選択すると、**担当する業務Roleの追加**、**別Stepへの順序・分岐・例外**、**別のStep・外部主体・概念Storeへの情報の授受**のいずれかを選択して入力できます。業務Roleは元のレビュー案で宣言されているものに限定し、そのStepにすでに割り当てられたRoleは選択肢から除きます。手作業Stepにも、Application Functionを新たに付けずに担当を提案できます。

「未受理の変更候補を確認」を押すと、読み込んだ三つのファイルの実バイトからSHA-256を再計算し、初回の読み込み時と一致することを確認します。そのうえで`dfd-selected-edit-bridge.mjs`を呼び、選択したStepまたは起動契機、参照手順、レビュー案、局所的なSource位置を照合して未受理の変更案を作成します。図上の線を自動的に実行可能な業務フローへ変換するものではありません。

- 候補は画面内で表示し、希望する場合にのみ独立したJSONとして保存します。元のProcedure/Review/DFDファイルを書き換えません。
- 起動契機は**説明文だけ**を変更できます。Stepでは、担当業務Roleの追加、別Stepへの順序・分岐・例外の提案、選択したStepを出発点とする情報の授受を追加できます。分岐・例外には条件の明示が必要で、単なる順序には条件を付けません。情報の授受には識別子、宛先、情報の内容を明示します。重複する遷移や情報の授受は拒否します。いずれもRBAC認可、Function実行、Storeへの実データ更新、業務手順の受理を伴いません。
- 外部主体（Party）や概念Storeを図上で選択した場合は、**その要素から業務Stepへ入る情報の授受**だけを提案できます。宛先種類はStepに固定し、担当Roleの追加や制御遷移への変更はできません。Storeからの授受の記述は概念上の関係であり、データベースへの読取り・書込み権限や操作の承認を意味しません。
- 選択対象の切替え、元資料の変更、変更種類・宛先・条件・情報の内容などの入力変更によって、表示中の未受理候補を破棄します。
- 保存したJSONは`tei.reference.business-dfd-selected-review/v0`であり、正式なShared Acceptanceへの投入資料ではありません。`readyForAcceptance=false`、`sharedAcceptanceCandidate=null`、`sourceWrites=[]`のままです。

実ブラウザーによる画面操作は未検証であり、Nodeの模擬DOM試験はその代用にはなりません。さらに、この照合は受理済みWorldの同一revisionや真正なSource Indexを証明しないため、変更候補を正本へ自動適用してはなりません。

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
