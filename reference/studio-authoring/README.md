# 三視点Studio authoring・最小Reference（Issue #1124）

状態: **初回G0/G1向けの部分実装**。現時点で三視点統合Editorの製品受入・Source再投影・実行Evidenceは成立していません。

既存の[Shared Acceptance](../../tools/reference-shared-acceptance/README.md)とStudio authoring計画を使用します。このディレクトリに新しいCanonical正本、Patchエンジン、Validation、Acceptance、Authorizationを作りません。親の[#1119](https://github.com/hat47x/tei-platform/issues/1119)と[#1120](https://github.com/hat47x/tei-platform/issues/1120)の最終契約が確定するまでは、ここにあるJSONは**Reference限定**の運搬形式です。

## 検証する最小境界

- 入力はTEIの既存World JSONと、同じWorkspaceから得たmodelRevision。BridgeはWorldのData/Relation/Attribute/ValidationとApplication Patternの**元のID**を使い、表・ER・機能の近傍Projectionを読み取り専用で返します。CLIではWorkspace revisionをWorld取得前後で照合し、読み込み中の変更を検知すればProjectionを拒否します。
- Accepted Business Procedureは現在のWorld JSONに含まれていないため、Businessは標準で `unresolved` です。#1125のread-only DFD JSONを任意入力として与えた場合に限り、**明示的なApplication Pattern ID**で局所的に結合します。手作業・外部作業はFunctionへ紐付けずに別表示します。この結果は`reference-only`（未受理のFixture由来）で、Sourceとの同一revisionを証明しません。業務意味を勝手にacceptedへ昇格させません。既存の`tei source-index`を同じWorkspaceから取得すると、Source定義を`semantic-data`等のexact kind/IDで特定できます。生の作業領域を指定する`inspect -workspace`では、さらに対象ファイルを読み直してSHA-256指紋とバイト位置・行列位置を照合します。検証結果は`sourceNavigation.byteVerification.status=verified-at-read`です。`-world-file`等の単独資料を渡す場合は`not-checked`に留めます。取得時はrevisionの前後一致を確認します。検証はファイルを読んだ一時点の観測であり、同時編集を排除するロックでも書込み権限でもありません。対象パスのシンボリックリンクや作業領域外参照、古い指紋や位置情報は拒否します。Sourceへ書き込む位置はShared Acceptance Previewの`sourceEdits`で確かめます。
- ER操作の`add-relation`と表操作の`add-validation/positive`だけを、既存 `tei.reference.change-candidate/v0` に変換します。Derived rowの直接書換え、任意のpatch、layout変更から意味を生成することは禁止します。
- 実際のValidation、Semantic Diff、Impact、source edit、stale revision、Accept、Rollback、Decisionは**既存Shared Acceptance**が担当します。Bridgeのprecheckは利便性のためだけで、これらの検査を代替しません。
- file/native CLI/AI/Studioのorigin値はprovenanceにとどめ、meaningを入口により変えません。候補の差分が同じなら最終的な意味の比較はShared Acceptanceへ委譲します。

## 実Workspaceでの一往復（必ずscratch copy）

以下は**操作手順**であり、本PR headで実行・PASS確認した記録ではありません。既存Gate B/Cの固定Workspaceに書かないため、まず独立コピーを用意します。

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

初期`e2e.mjs`は表のケースを2操作の候補として実施し、受理後に意味・Application双方のSourceが変化し、再投影でも利用宣言が分かることを検査します。**このE2Eスクリプト自体の実行は未確認です**。Attribute型・Validationの妥当性はShared Acceptanceへ委ねます。

各Projectionが返す`realizationReadiness.status=not-verified`は重要です。Source Acceptanceは正本変更に限った判断であり、DB migration、Target配備、実行時権限、Provider書込み、Observationを証明しません。表示の変更だけで不要なDB migrationを要求しない一方、意味変更だけで旧schemaが利用可能と断定もしません（#1119 S01/S02）。実際の適用可否は#1121/#1126/#1127側から明示的に受け取る必要があります。

既存Relationのcardinalityを`set`するoperationは未実装です。未対応編集を直接Source patchへ落とし込まないでください。

Worldとrevisionを先に取得してCLIへ渡す場合は、`-workspace`の代わりに`-world-file world.json -revision-file revision.json`を同時指定できます。`inspect`では`-source-index-file index.json`も指定でき、そのIndexにexact一致する定義位置を表示します。Workspace指定時はWorldとSource Indexを同じWorkspaceから読み、取得前後のrevision一致を検査します。ただし完全な原子snapshotやファイルの再読込時点の一致は保証せず、Accepted Sourceのrevision一致はShared Acceptanceで再確認します。JSONファイルを任意の第三者から受け取った場合、その内容がTEIのauthoritative Sourceと一致することはBridge単独では証明できません。

### Source revision R0 → R1 の追跡（#1119）

`e2e.mjs`は各ケースの結果を`tei.reference.studio-source-loop-receipt/v0`として標準出力に含めます。採取対象は`baseModelRevision`（R0）、Previewの`currentModelRevision`と予測`modelRevisionAfter`、Accepted Decisionの`modelRevisionBefore/After`、Sourceを再ロードした実際のR1、candidate/review指紋、変更ファイル一覧です。これらの整合を**一つのScratch受理操作の内部で検査**し、旧R0候補のPreviewとAcceptがR1を上書きしないことも検査します。

Receiptは実行に成功したときだけ出力される**テスト実行記録の構造案**であり、現PR headのactual E2E PASSを意味しません。人間のAccept、実Provider、Runtime/Target、後続Observationの証拠は入らず、`not-observed`として明示します。Source IDとSource RevisionをWebのCommand/Observationへ引き渡す実際の連結は#1121/#1127の責務で、ここで実行Readyとは表示しません。

現行Source Indexは#1120の`scope + kind + ID`契約と同じスコープ情報を提供しません。Source Indexの参照へ呼出し側が`scope`を付けただけではC0解決済みと扱わず、`sourceNavigation.status=unresolved`にします。複数scopeを持つモデルを安全に編集するには、#1120のtyped Resolverを同一revisionへ接続する必要があります。単一scopeの現Reference向けflat `data:ID` 形式を、汎用のC0表現として固定しません。

## Business参照の任意結合（#1125 / Draft PR #1138）

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
node reference/studio-authoring/cli.mjs inspect \\
  -workspace "$scratch/workspace" \\
  -focus step:BP-05 \\
  -business-dfd "$scratch/dfd.json"
```

- 逆向きの`step:ID`選択は`-business-dfd`が必須です。`tei.reference.studio-business-step-navigation/v0`を返し、Business Stepのスコープ`reference-procedure:<procedureId>`と、元Worldの型付きfocus候補`data:ID` / `relation:ID` / `validation:ID` / `pattern:ID`を分離します。ExactなID照合が成立しても`identity-match-only`であり、同一revisionとの結合は未証明です。`editable=false`、`acceptanceTargetStatus=unresolved-use-shared-acceptance`を維持します。Procedureに当該Stepが存在しない、Producerがerrorを返した、Source参照が不一致の場合はfail-closedです。
- DFDの`functionRefs[].id`が同じWorldのApplication Pattern IDにexact matchしたStepについて、そのfocusとの関係を表示します。`data`/`relation`/`capability`参照の相違は`unresolved`にします。さらに#1125の`steps[].explicitMeaning`が示す`semantic-data` / `semantic-relation` / `semantic-validation`の**明示されたkind/ID**を、現在のWorldの正確な定義と再照合します。正確に一致するときに限り、FunctionのないStepからも該当Data/Relation/Validationへ`reference-only`の閲覧リンクを返します。これはBusiness StepをCanonicalに受理したことも、実行Functionを生成したことも意味しません。
- 原DFDの`source.file`/`source.pointer`は**非Canonicalな参照Fixtureの位置**であり、Source Indexの編集先ではありません。
- `manualOrExternalUnlinked`にはFunctionなしの手作業等を別掲します。これらは**Functionに未接続**であり、明示された意味参照まで存在しないという意味ではありません。近接していることだけを理由に依存辺を捏造しません。大きなProcedureでは20件までとし切詰めを明示します。
- DFDにはWorkspace revisionのbindingが無いので、たとえ同一IDでも`freshness=unverified-no-model-revision-binding`です。旧DFDがSource revisionと一致すること、Business Procedureがacceptedであることは**証明できません**。最新Worldからの再取得と#1120 contractによる厳密な結合は後続です。
- #1138は別レーンのDraftであり、本PRへのコードコピーやCore変更は行いません。このコマンドは#1138のツールが存在するcheckoutでのみ動きます。通常のmainではDFDを渡さずBusinessを`unresolved`に保ちます。

### 元ソース位置の確認

通常の`inspect -workspace`は索引とソース本体を照合し、位置情報を表示します。検査では作業領域内の通常ファイルに限定し、**検査対象と同じバイト列**をSHA-256関数へ渡して指紋を求め、開始・終了位置と照合します。呼び出し元から渡された指紋だけで検証済みとは判断しません。確認後には作業領域の版をもう一度取得します。途中で版が変われば結果を拒否します。

独立した`-world-file`・`-revision-file`・`-source-index-file`の組合せは、元ソースを保有していないため、この照合を実施できません。表示された位置を根拠に編集先を自動決定することはありません。通常の`inspect`でも、利用者の受理や編集権限は得られず、正本の変更は既存の共通受理機構に委ねます。

## 試験と境界

`e2e.mjs`のAcceptは**検証用Scratchに限定した自動テスト操作**であり、人間が承認したEvidenceではありません。テストは一時WorkspaceのSource digest、Preview/Review中の不変、受理後の限定変更、再ロード/再投影、旧候補のstale拒否を確認し、一時ファイルを削除します。実行用フラグ`TEI_STUDIO_E2E=1`を意図的に要求します。

```sh
node --test reference/studio-authoring/bridge.test.mjs

# exact headのfull checkoutを保有する環境で、scratch Workspaceだけを変更する統合試験
TEI_STUDIO_E2E=1 node reference/studio-authoring/e2e.mjs

# 統合資格の判定では上記だけでなく、以下の完全実行とpre/post clean receiptが必要
tools/repository-health-check.sh
```

| Scenario | 初期実装 | 未検証・未対応 |
| --- | --- | --- |
| V00 | Worldから表/ER/機能近傍を同じ論理IDへ戻す。Source Indexのexact kind/IDで元定義のアンカーを表示。#1138 JSONの非Canonical BusinessについてPatternおよび直接Data/Relation/Validation参照をread-only結合し、Step選択からの逆向きfocusも提供 | Accepted Businessとの同revision結合は未実装。Live inspectの単一時点でのSource byte照合は追加したが、再読込を伴う一貫した画面表示・4つのAccepted Projection受入は**未達** |
| V03 | ERのRelation追加・表のValidation追加に加え、必要ならApplicationによるValidation利用選択も同一candidateで提案。一括`review-edit`で既存Shared AcceptanceによるDiff/Impact/Reviewを表示し、実行Readyは未確認とする | read-only consumerのexact-head実GoテストとTarget/Provider readinessは未確認 |
| V04 | Shared Acceptance CLI手順とscratch自動検証スクリプトを追加。規則+Application選択はSource2ファイルを検査するテストとして追加。Accepted Relation/Validationを表/ERへ再投影可能 | exact-head scratch E2Eのactual実行、Source reload比較は**未実行** |
| N05 | Derived直書き/任意PatchをBridgeが拒否 | 全Viewのeditable metadataの網羅 |
| N06 | candidateのR0・Reviewの同一性を確認し、scratch E2EにAccepted DecisionのR0/R1照合と旧候補のstale負例を追加 | 実WorkspaceでのN06実行とaccept途中の並行write試験は未実施 |
| Invalid ref | exactな入力subject/Relation target不在・重複を拒否。Source Indexの曖昧な定義、範囲不正、危険なpathを拒否。DFDの参照切れはunresolved | 全モデルKind・全typed edge・Source byte最新性とBusinessの受理済み意味 |
| N12 | layout書換えは意味操作へ変換しない | 独立したView Metadataの保持・再生 |
| 性能 | focus近傍だけを返す、合成増量ケースあり | realistic 1万+件の時間/メモリ実測 |

### 次の開発順序

1. #1120のscope付きtyped subjectと現在のSource Indexのkind/ID対応を正式化する。現行のexact anchorは最初の足場とし、locatorのfile fingerprint最新性・source ownership・逆参照と編集可能箇所を結び付ける。
2. #1125の非CanonicalなDFD候補と受理済みBusiness Procedureを区別し、後者がSource Acceptanceの対象になってから#1126 Data、#1127 Functionと**同一revision**で4Projectionへ拡張する。Designer layoutは派生metadataへ分離する。
3. #267のShared AcceptanceへのPreview/Review/Accept/Undoを実Workspace scratchで検証し、N06のstale再現・競合時のSource byte不変・Decision ledgerの追跡を独立試験にする。
4. 運用規模の入力で性能測定を行い、Source上の変更箇所へnavigate→図/表変更→再投影をUI利用者に対して検証する。

このReferenceは製品Studio完成の証拠ではありません。V05/V08（Target/Runtime/Observation）は#1121/#1127の責務です。
