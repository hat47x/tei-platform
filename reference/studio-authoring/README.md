# 三視点Studio authoring・最小Reference（Issue #1124）

状態: **Studio参照実装の部分実証**。一時作業領域でのSource受理・再読込・表/ER再投影と、四つの参照視点・HTML審査表示は自動試験で成功しています。ただし**製品としての統合編集画面・受理済み業務正本の四視点連携・人による受理・実Providerでの動作**は未達です。

既存の[Shared Acceptance](../../tools/reference-shared-acceptance/README.md)とStudio authoring計画を使用します。このディレクトリに新しいCanonical正本、Patchエンジン、Validation、Acceptance、Authorizationを作りません。親の[#1119](https://github.com/hat47x/tei-platform/issues/1119)と、統合済みの型付き参照契約[#1120](https://github.com/hat47x/tei-platform/issues/1120)を踏まえても、Studioが扱うJSONは**参照実装限定**の運搬形式です。元Sourceの位置を特定できることは、編集権限・意味の受理・実行権限を保証しません。

## 検証する最小境界

- 入力はTEIの既存World JSONと、同じWorkspaceから得たmodelRevision。BridgeはWorldのData/Relation/Attribute/ValidationとApplication Patternの**元のID**を使い、表・ER・機能の近傍Projectionを読み取り専用で返します。CLIではWorkspace revisionをWorld取得前後で照合し、読み込み中の変更を検知すればProjectionを拒否します。
- Accepted Business Procedureは現在のWorld JSONに含まれていないため、Businessは標準で `unresolved` です。#1125のread-only DFD JSONを任意入力として与えた場合に限り、**明示的なApplication Pattern ID**で局所的に結合します。手作業・外部作業はFunctionへ紐付けずに別表示します。この結果は`reference-only`（未受理のFixture由来）で、Sourceとの同一revisionを証明しません。業務意味を勝手にacceptedへ昇格させません。既存の`tei source-index`を同じWorkspaceから取得すると、Source定義を`semantic-data`等のexact kind/IDで特定できます。生の作業領域を指定する`inspect -workspace`では、さらに対象ファイルを読み直してSHA-256指紋とバイト位置・行列位置を照合します。検証結果は`sourceNavigation.byteVerification.status=verified-at-read`です。`-world-file`等の単独資料を渡す場合は`not-checked`に留めます。取得時はrevisionの前後一致を確認します。検証はファイルを読んだ一時点の観測であり、同時編集を排除するロックでも書込み権限でもありません。対象パスのシンボリックリンクや作業領域外参照、通常ファイル以外の対象（ディレクトリ・名前付きパイプなど）、古い指紋や位置情報は拒否します。Sourceへ書き込む位置はShared Acceptance Previewの`sourceEdits`で確かめます。
- ER操作の`add-relation`と表操作の`add-validation/positive`だけを、既存 `tei.reference.change-candidate/v0` に変換します。Derived rowの直接書換え、任意のpatch、layout変更から意味を生成することは禁止します。
- 実際のValidation、Semantic Diff、Impact、source edit、stale revision、Accept、Rollback、Decisionは**既存Shared Acceptance**が担当します。Bridgeのprecheckは利便性のためだけで、これらの検査を代替しません。
- file/native CLI/AI/Studioのorigin値はprovenanceにとどめ、meaningを入口により変えません。候補の差分が同じなら最終的な意味の比較はShared Acceptanceへ委譲します。

## 実Workspaceでの一往復（必ずscratch copy）

以下は**一時作業領域での再現手順**です。PRに含めた自動試験とは異なり、手順を実行しただけでは人間の受理を証明しません。既存Gate B/Cの固定Workspaceに書かないため、まず独立コピーを用意します。

```sh
scratch="$(mktemp -d)"
cp -R reference/workspace "$scratch/workspace"

# V00: source由来のWorldから局所横断Projection（JSON）
node reference/studio-authoring/cli.mjs inspect \
  -workspace "$scratch/workspace" \
  -focus relation:sales.order.lines

# V03: 図から新しい逆方向Relationを提案（実編集なし）
cat > "$scratch/edit.json" <<'JSON'
{
  "schema": "tei.reference.studio-edit/v0",
  "id": "reverse-order-relation",
  "origin": "studio",
  "intent": "明細から受注へ戻るRelationを明示する",
  "view": "er",
  "subject": "data:sales.order_line",
  "operation": "add-relation",
  "relationID": "sales.order_line.order",
  "symbol": "order",
  "targetData": "sales.order",
  "cardinality": "one"
}
JSON
node reference/studio-authoring/cli.mjs propose \
  -workspace "$scratch/workspace" -edit-file "$scratch/edit.json" \
  > "$scratch/candidate.json"

# 表/ER編集→Shared Acceptance Preview/Reviewまでを一つのread-only応答へ
node reference/studio-authoring/cli.mjs review-edit \
  -workspace "$scratch/workspace" -edit-file "$scratch/edit.json" \
  > "$scratch/studio-review.json"

# review-editが出力するcandidate/preview/reviewはいずれもread-only。
# sourceWriteByThisCommand=false、explicitAcceptance=not-performed。
# preview.acceptableがfalseなら決してAcceptへ進まない。

# Shared Acceptanceはsourceの現revisionを再検査し、overlay上で評価する
go run ./tools/reference-shared-acceptance preview \
  -workspace "$scratch/workspace" -candidate "$scratch/candidate.json" \
  > "$scratch/preview.json"
go run ./tools/reference-shared-acceptance review \
  -workspace "$scratch/workspace" -candidate "$scratch/candidate.json" \
  -format json > "$scratch/review.json"

# V04: Preview/Reviewの差分・impact・unresolvedを確認した上で
# 利用者が明示的に受理すると決めた場合だけ次を実行する。
# 自動Acceptしない。fingerprintだけでは人間の判断は証明できない。
review_fp="$(node -e 'console.log(require(process.argv[1]).reviewFingerprint)' "$scratch/review.json")"
go run ./tools/reference-shared-acceptance accept \
  -workspace "$scratch/workspace" -candidate "$scratch/candidate.json" \
  -expected-review-fingerprint "$review_fp" \
  > "$scratch/decision.json"

# Sourceから再読込みして各Projectionへ戻る
node reference/studio-authoring/cli.mjs inspect \
  -workspace "$scratch/workspace" \
  -focus relation:sales.order_line.order
```

一括`review-edit`は既存`preview`と`review -format json`を呼び出す**読み取り専用の薄いconsumer**です。候補のJSONは一時ディレクトリだけに作成し、処理後に消去します。手動の`preview`/`review`を置き換えられますが、正本編集・承認・Runtime authorityを付与しません。取得前後のrevision不一致に加え、PreviewとReviewの**候補ID・候補指紋・根拠指紋・基準revision・受入可否が互いに食い違う場合も拒否**します。`reviewConsistency.readyToAccept=false`は人による判断が未実施であることを表します。Accept時には元のShared Acceptance自身が再検査します。外部snapshotファイルのみで`review-edit`を行う操作は認めません。

`review.previewAcceptable`がfalse、review内容が未確認、あるいはsourceの競合があればAcceptを行わず、未解決として止めてください。Previewの`unresolved`を勝手に「変更なし」へ変換しません。現行Shared Acceptanceの既知仕様では、Relation追加によってPersistence Mappingの意味まで自動確定するとは限りません。

表上の編集は`view:"table"`、`subject:"attribute:sales.order_line.unit_price"`、`operation:"add-validation"`、`validationKind:"positive"`、`validationID:"sales.order_line.unit_price_positive"`とし、別候補として提出できます。**ただしValidationの宣言のみでは、既存Web CommandがそのRuleを評価するとは限りません。** 実際に`order_entry`から使用する意図があるなら、同じ編集入力に次のフィールドを明示して、**規則とApplicationの選択を一つの候補**として提出します。

```json
{
  "applicationUse": {
    "application": "order_entry",
    "placement": "applications/order-entry/application.tei"
  }
}
```

この追加フィールドは上記の`add-validation`編集入力へ組み合わせる部分指定です。単体のJSONとして提出するものではありません。生成される`delta`は既存の`add-validation`と`add-application-validation-use`の2操作になります。TEI既存実装`tools/reference-shared-acceptance/practical_validation_loop_test.go`と同じ契約を利用し、別の書込み器やRule実行器を追加しません。

`placement`は呼出し側が**正本で確認したApplication定義の位置**を指定します。Bridgeはパスの大まかな形式とSemantic namespace利用を確認しますが、source ownership・型検査・二重宣言・書込み可否はShared AcceptanceのPreview/Reviewが最終確認します。Application側の明示的選択が受理済みであることも、実Target・Providerで規則が実行された証拠ではありません。

`e2e.mjs`は表のケースを2操作の候補として実施し、受理後に意味・Application双方のSourceが変化し、再投影でも利用宣言が分かることを検査します。Studio専用CIで一時作業領域上の実行を確認済みです。Attribute型・Validationの妥当性はShared Acceptanceへ委ねます。

各Projectionが返す`realizationReadiness.status=not-verified`は重要です。Source Acceptanceは正本変更に限った判断であり、DB migration、Target配備、実行時権限、Provider書込み、Observationを証明しません。表示の変更だけで不要なDB migrationを要求しない一方、意味変更だけで旧schemaが利用可能と断定もしません（#1119 S01/S02）。実際の適用可否は#1121/#1126/#1127側から明示的に受け取る必要があります。

既存Relationのcardinalityを`set`するoperationは未実装です。未対応編集を直接Source patchへ落とし込まないでください。

Worldとrevisionを先に取得してCLIへ渡す場合は、`-workspace`の代わりに`-world-file world.json -revision-file revision.json`を同時指定できます。`inspect`では`-source-index-file index.json`も指定でき、そのIndexにexact一致する定義位置を表示します。Workspace指定時はWorldとSource Indexを同じWorkspaceから読み、取得前後のrevision一致を検査します。ただし完全な原子snapshotやファイルの再読込時点の一致は保証せず、Accepted Sourceのrevision一致はShared Acceptanceで再確認します。JSONファイルを任意の第三者から受け取った場合、その内容がTEIのauthoritative Sourceと一致することはBridge単独では証明できません。

### Source revision R0 → R1 の追跡（#1119）

`e2e.mjs`は各ケースの結果を`tei.reference.studio-source-loop-receipt/v0`として標準出力に含めます。採取対象は`baseModelRevision`（R0）、Previewの`currentModelRevision`と予測`modelRevisionAfter`、Accepted Decisionの`modelRevisionBefore/After`、Sourceを再ロードした実際のR1、candidate/review指紋、変更ファイル一覧です。これらの整合を**一つのScratch受理操作の内部で検査**し、旧R0候補のPreviewとAcceptがR1を上書きしないことも検査します。

Receiptは**一時作業領域で実際に試験を実行した記録**です。2026年10月12日に[Studio専用CI #38246987101](https://github.com/hat47x/tei-platform/actions/runs/38246987101)でNode.jsの39テストと、一時作業領域での80項目の試験が成功しました。後者はER・表の正本変更から再投影する2件に加え、Go製業務DFDの生成、四つの視点からの参照、業務手順からValidationへの逆向き参照を含みます。この記録の対象はPR head `6239cbbf9ebcf0d67569f746c4745b8cb49d6c75` で、Studio専用CI・全体健全性CI・E1-Aがいずれも成功しています。人間の受理、実際のデータベースや実行先、後続観測の証拠は含まず、`not-observed`として明示します。Source IDとSource RevisionをWebのCommand/Observationへ引き渡す実際の連結は#1121/#1127の責務で、ここで実行Readyとは表示しません。

現行Source Indexは#1120の`scope + kind + ID`契約と同じスコープ情報を提供しません。Source Indexの参照自身、または`subject`内へ呼出し側が`scope`を付けてもC0解決済みと扱わず、`sourceNavigation.status=unresolved`にします。パスの重複区切りや`.`区間、開始・終了位置の形式不備も索引の段階で拒否し、ソース位置を誤って確定しません。複数scopeを持つモデルを安全に編集するには、#1120のtyped Resolverを同一revisionへ接続する必要があります。単一scopeの現Reference向けflat `data:ID` 形式を、汎用のC0表現として固定しません。


### ブラウザでの編集候補作成（正本は未変更）

`edit-form` は、Workspaceで受理済みのWorldからDataまたはAttributeを選び、オフラインの編集候補作成画面を生成します。ERのRelation追加と、表の正値Validation追加に対応し、後者は意味名前空間の利用を明示したアプリについてだけ、利用宣言も一つの候補にできます。

```sh
node reference/studio-authoring/cli.mjs edit-form \
  -workspace "$scratch/workspace" \
  -focus data:sales.order_line > "$scratch/relation-editor.html"
node reference/studio-authoring/cli.mjs edit-form \
  -workspace "$scratch/workspace" \
  -focus attribute:sales.order_line.unit_price > "$scratch/validation-editor.html"
```

生成HTMLをブラウザで開き、候補の識別子・変更意図・対象などを入力し、「候補JSONを生成」を選択してください。JSONはブラウザ内の読取り専用領域に出力されるので、そのままJSONファイルへ保存し、`review-edit -workspace ... -edit-file ... -format html`へ渡して審査します。**HTMLは正本へ書き込まず、承認を代行しません**。外部通信は禁止し、ブラウザスクリプトは内容指紋で固定します。

自動試験では、実際に生成したフォームの埋込みJavaScriptをNode.jsの隔離された模擬DOM上で実行し、そこで作成されたJSONをStudio CLIからGoの共通受理機構に渡して、**Preview・Review・テスト用Accept・Source再読込・再投影・旧版の拒否**まで確認します。これは実ブラウザやWebViewでの操作を観測したものではなく、人的な受理の証拠でもありません。 編集フォームの選択項目が改ざんされて存在しないRelation参照先やアプリを指定した場合にも、Studio側で候補の審査を拒否し、正本と版を変更しないことをER・表で検査します。

編集JSONの`expectedModelRevision`には**画面生成時の版**が必ず含まれます。後続のStudio CLIが異なる版を取得した場合は`STUDIO_EDIT_STALE_MODEL_REVISION`で拒否し、最新状態へ暗黙に編集意図を移しません。従来の手書きJSONでは省略可能ですが、画面から作る候補では必須です。ブラウザ内の検査は入力補助であり、型検査・Source所有権・変更影響・受理可否はすべて既存Shared Acceptanceに委ねます。最初のフォームで扱えない意味変更は、無理にDerivedを更新せず既存のSource契約を用います。

### 実Chromeによる編集フォーム操作の自動回帰

既存のWeb Target用Chrome CIと同じ `playwright-core@1.56.1` とGitHub-hosted Ubuntu runnerのChromeを使用し、`edit-form -workspace` が実Workspaceから生成した **ER・表の2画面**を、実ChromeのDOM上で操作する自動試験を追加しています。`reference/studio-authoring/real-browser.mjs` は候補ID・意図・対象値の入力、フォーム送信、CSP付きインラインスクリプトの実行、出力JSONの検査、`propose`→`review-edit` による**既存のGo共通審査**への手渡しを行います。外部HTTP通信が無いこと、レビュー前後のmodelRevision不変も確認し、実画面のスクリーンショットと判定JSONをCIに短期保存します。 さらに、実Chromeでセレクト要素へ存在しないData／Applicationの選択肢を挿入する改変負例を両画面で試し、画面側の入力制限に依存せず、Studio Bridgeが`INVALID_RELATION_PROPOSAL`と`APPLICATION_DOES_NOT_DECLARE_SEMANTIC_USE`で拒否すること、拒否後のモデル版が不変であることを確認します。インストールは作業領域外に限定し、ソースツリーを変更しません。

```sh
# Chromeのある環境で。依存はWorkspace外へ固定して導入
npm install --prefix /tmp/tei-studio-playwright \
  --no-audit --no-fund --ignore-scripts --no-save --no-package-lock \
  playwright-core@1.56.1
node reference/studio-authoring/real-browser.mjs \
  /tmp/tei-studio-playwright /tmp/tei-studio-chrome-evidence
```

これは**自動化された実ブラウザ操作**であり、人間の判断・受理／実業務環境でのユーザビリティ確認ではありません。このBrowser試験は実Sourceを書き換えず、Goへの`propose/review`で止まります。既存のNode VM→Scratch Accept→R1再投影試験とは異なる境界を観測しており、両者の成功を「人間が実ブラウザでSourceを受理した」という一つの実績へ合算しません。正式なBusiness正本との同一revisionも依然未実証です。

### 読み取り専用のHTML画面

既存のJSONと同じ投影・審査結果を、独立した静的HTMLとして確認できます。ブラウザを使っても正本を直接編集する機構は追加しません。HTML内の表・ER・機能・業務は同じWorldの参照用表示です。参照用業務手順は受理済み正本ではなく、提案の審査も人間による受理ではありません。

```sh
node reference/studio-authoring/cli.mjs inspect \\
  -workspace "$scratch/workspace" \\
  -focus relation:sales.order.lines -format html > "$scratch/studio.html"
node reference/studio-authoring/cli.mjs review-edit \\
  -workspace "$scratch/workspace" -edit-file "$scratch/edit.json" \\
  -format html > "$scratch/review.html"
```

変更案の審査画面には、共通受理機構へそのまま渡すための**変更候補JSONの全文**、基準版、候補指紋、審査指紋、再審査のためのCLIコマンドも載せます。内容を変えずに候補JSONとして保存し、**改めてShared Acceptanceで審査**してください。指紋の存在は承認権限を意味せず、古い版・受理不可・審査不一致では「受理案内を保留」とします。画面上に受理ボタンや正本への書込み経路は設けません。

HTMLは実行用スクリプト、フォーム、外部通信、外部画像を含まず、表示文字列をHTML用に符号化して出力します。共通受理機構の審査区分A1〜A8（意味差分、影響、診断、未解決点、権限への影響、ネイティブコードへの依存、逆追跡、承認単位）を評価範囲・根拠とともに表示します。区分が欠落・重複した場合は「未評価」と明示します。共通受理機構やプラグインから未知の審査区分が返された場合も、表示から削らず未評価で示し、審査済みと誤認させません。ブラウザ内の視点切替は同じ文書内の移動であり、異なる対象IDへ移動する場合はCLIで焦点を指定して再生成します。正本の編集・受理には既存の共通受理機構が必要です。単独のWorld/Source Indexファイルを使ったHTMLでは元ソースとの一致が未確認であることを明示します。

## 業務モデルの参照用結合（#1125 / 統合済みPR #1138）

#1125の独立DFD Readerが配置されたcheckoutでは、同じScratch Workspaceを指定してDFD JSONを取得できます。

```sh
go run ./tools/reference-business-dfd -format json \
  -workspace "$scratch/workspace" > "$scratch/dfd.json"
node reference/studio-authoring/cli.mjs inspect \
  -workspace "$scratch/workspace" \
  -focus relation:sales.order.lines \
  -business-dfd "$scratch/dfd.json"

# 逆方向: 非CanonicalなBusiness Stepから、元World上のFunction / Data /
# Relation / Validationへの読み取り専用の参照候補を取得
node reference/studio-authoring/cli.mjs inspect \
  -workspace "$scratch/workspace" \
  -focus step:BP-05 \
  -business-dfd "$scratch/dfd.json"
```

- 逆向きの`step:ID`選択は`-business-dfd`が必須です。`tei.reference.studio-business-step-navigation/v0`を返し、Business Stepのスコープ`reference-procedure:<procedureId>`と、元Worldの型付きfocus候補`data:ID` / `relation:ID` / `validation:ID` / `pattern:ID`を分離します。ExactなID照合が成立しても`identity-match-only`であり、同一revisionとの結合は未証明です。対象Stepに一件でも未解決の参照がある場合、全体の`status`は`unresolved`とし、`resolutionSummary`に一致・未解決・生成元エラーの件数を出します。Data・Relation・Validation・Functionの側からStepを見る場合も、表示中の参照だけでなく**当該Stepの全宣言を照合**します。別の参照が切れていればStepを`unresolved`にし、一致した項目だけで完全な手順と判断しません。参照先が一件も見つからない場合も`unresolved`です。Studio側に受理済みCapabilityの解決機構がないときは、その参照を一致と推定しません。解決済み候補を個別に表示しても、手順全体を正本として扱いません。`editable=false`、`acceptanceTargetStatus=unresolved-use-shared-acceptance`を維持します。Procedureに当該Stepが存在しない、Producerがerrorを返した、Source参照が不一致の場合はfail-closedです。
- DFDの`functionRefs[].id`が同じWorldのApplication Pattern IDにexact matchしたStepについて、そのfocusとの関係を表示します。`data`/`relation`/`capability`参照の相違は`unresolved`にします。さらに#1125の`steps[].explicitMeaning`が示す`semantic-data` / `semantic-relation` / `semantic-validation`の**明示されたkind/ID**を、現在のWorldの正確な定義と再照合します。正確に一致するときに限り、FunctionのないStepからも該当Data/Relation/Validationへ`reference-only`の閲覧リンクを返します。これはBusiness StepをCanonicalに受理したことも、実行Functionを生成したことも意味しません。
- 原DFDの`source.file`/`source.pointer`は**非Canonicalな参照Fixtureの位置**であり、Source Indexの編集先ではありません。
- `manualOrExternalUnlinked`にはFunctionなしの手作業等を別掲します。これらは**Functionに未接続**であり、明示された意味参照まで存在しないという意味ではありません。近接していることだけを理由に依存辺を捏造しません。大きなProcedureでは20件までとし切詰めを明示します。
- DFDにはWorkspace revisionのbindingが無いので、たとえ同一IDでも`freshness=unverified-no-model-revision-binding`です。旧DFDがSource revisionと一致すること、Business Procedureがacceptedであることは**証明できません**。最新Worldからの再取得と#1120 contractによる厳密な結合は後続です。
- #1138の業務DFD参照実装はすでにmainへ統合されていますが、生成される業務手順は現在も受理済みの正本ではありません。ツールの存在と業務意味の承認を混同しないでください。

### 元ソース位置の確認

通常の`inspect -workspace`は索引とソース本体を照合し、位置情報を表示します。検査では作業領域内の通常ファイルに限定し、**検査対象と同じバイト列**をSHA-256関数へ渡して指紋を求め、開始・終了位置と照合します。呼び出し元から渡された指紋だけで検証済みとは判断しません。確認後には作業領域の版をもう一度取得します。途中で版が変われば結果を拒否します。

独立した`-world-file`・`-revision-file`・`-source-index-file`の組合せは、元ソースを保有していないため、この照合を実施できません。表示された位置を根拠に編集先を自動決定することはありません。通常の`inspect`でも、利用者の受理や編集権限は得られず、正本の変更は既存の共通受理機構に委ねます。

## 試験と境界

`e2e.mjs`のAcceptは**検証用Scratchに限定した自動テスト操作**であり、人間が承認したEvidenceではありません。テストは一時WorkspaceのSource digest、Preview/Review中の不変、受理後の限定変更、再ロード/再投影、旧候補のstale拒否を確認し、一時ファイルを削除します。実行用フラグ`TEI_STUDIO_E2E=1`を意図的に要求します。

```sh
node --test reference/studio-authoring/bridge.test.mjs

# exact headのfull checkoutを保有する環境で、scratch Workspaceだけを変更する統合試験
TEI_STUDIO_E2E=1 node reference/studio-authoring/e2e.mjs

# 1万Data近傍の高ファンイン合成入力について、投影・HTMLの実測値をJSONで出力
node reference/studio-authoring/performance.mjs > studio-performance.json

# 全体健全性検査も実施する場合。統合判断は変更範囲に応じた試験を基準とする
tools/repository-health-check.sh
```

大規模な合成Referenceでは、Data 10,001件（うち参照元10,000件）、Attribute・Relation・Application Pattern各10,000件を組み立てます。同一Dataへ10,000件のRelationが集中する条件で、既存の `projectWorld` と `renderStudioHTML` を実行し、投影時間、HTML生成時間、HTMLバイト数、NodeプロセスRSSを `tei.reference.studio-performance-receipt/v0` として記録します。HTML一覧が各100件を上限に切り詰められることと、元のID・読み取り専用状態が維持されることも検査します。

計測値はCI実行環境に依存する単回観測であり、時間・メモリの性能保証や実ブラウザの描画FPSを示しません。合成Worldは実Workspaceの受理済みSourceではなく、Business Procedureの同版性・人の受理・Source書込み・実Targetも実証対象外です。合格判定にマシン依存の固定ミリ秒閾値を持ち込まず、領収データに値と限界を残します。

| Scenario | 初期実装 | 未検証・未対応 |
| --- | --- | --- |
| V00 | 表・ER・機能の同一ID参照に加え、Goの既存業務DFD生成器を一時作業領域で動かし、同じWorldから四つの参照視点を読み取り専用で照合。`BP-05`からValidationへの逆向き参照も実行済み。Source Indexの元定義アンカーを表示 | 業務DFDは参照用でありAccepted Businessではない。同revisionの四つの**受理済み**Projectionや実画面操作、実DB・実行先との連携は**未達** |
| V03 | ERでRelationを追加し、表からValidationとApplication側の利用宣言を提案する。共通受理機構の差分・影響・審査を経由する一往復を、一時作業領域で実行済み | 実画面からの操作、実データベースや実行先での可否は未確認 |
| V04 | 共通受理機構を使うER・表の2件の受理・再投影、旧候補の拒否に加え、Go製業務DFDからの四視点参照・逆向き参照を一時作業領域で実行。合計80項目を検査 | 参照用の業務手順を受理済み正本とみなさない。人間の受理・実画面・実Providerでの確認は未実施 |
| N05 | Derived直書き/任意PatchをBridgeが拒否 | 全Viewのeditable metadataの網羅 |
| N06 | 一時作業領域で、異なる編集者の独立した候補を同じR0で審査し、一方の受理後にもう一方の古い審査指紋による受理を拒否する試験を追加。正本のR1が保持されることも検査 | OS上の完全同時書込み、複数編集者による実環境の競合は未試験 |
| Invalid ref | exactな入力subject/Relation target不在・重複を拒否。Source Indexの曖昧な定義、範囲不正、危険なpathを拒否。DFDの参照切れはunresolved | 全モデルKind・全typed edge・Source byte最新性とBusinessの受理済み意味 |
| N12 | layout書換えは意味操作へ変換しない | 独立したView Metadataの保持・再生 |
| 性能 | 1万件高ファンインの合成Referenceで投影時間・HTML生成時間・HTMLサイズ・RSSをCI実測しJSON記録。表示一覧100件上限を確認 | 実業務分布の1万件超・実ブラウザ/WebView FPS・複数ユーザー負荷・実Providerを含むSLOは未測定 |

### 次の開発順序

1. #1120のscope付きtyped subjectと現在のSource Indexのkind/ID対応を正式化する。現行のexact anchorは最初の足場とし、locatorのfile fingerprint最新性・source ownership・逆参照と編集可能箇所を結び付ける。
2. #1125の非CanonicalなDFD候補と受理済みBusiness Procedureを区別し、後者がSource Acceptanceの対象になってから#1126 Data、#1127 Functionと**同一revision**で4Projectionへ拡張する。Designer layoutは派生metadataへ分離する。
3. #267のShared AcceptanceへのPreview/Review/Accept/Undoを実Workspace scratchで検証し、N06のstale再現・競合時のSource byte不変・Decision ledgerの追跡を独立試験にする。
4. 運用規模の入力で性能測定を行い、Source上の変更箇所へnavigate→図/表変更→再投影をUI利用者に対して検証する。

このReferenceは製品Studio完成の証拠ではありません。V05/V08（Target/Runtime/Observation）は#1121/#1127の責務です。
