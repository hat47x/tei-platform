# 三視点Studio authoring・最小Reference（Issue #1124）

状態: **初回G0/G1向けの部分実装**。現時点で三視点統合Editorの製品受入・Source再投影・実行Evidenceは成立していません。

既存の[Shared Acceptance](../../tools/reference-shared-acceptance/README.md)と[Studio authoring計画](../../plan/issue/000046-studio-shared-acceptance-authoring-loop.md)を使用します。このディレクトリに新しいCanonical正本、Patchエンジン、Validation、Acceptance、Authorizationを作りません。親の[#1119](https://github.com/hat47x/tei-platform/issues/1119)と[#1120](https://github.com/hat47x/tei-platform/issues/1120)の最終契約が確定するまでは、ここにあるJSONは**Reference限定**の運搬形式です。

## 検証する最小境界

- 入力はTEIの既存World JSONと、同じWorkspaceから得たmodelRevision。BridgeはWorldのData/Relation/Attribute/ValidationとApplication Patternの**元のID**を使い、表・ER・機能の近傍Projectionを読み取り専用で返します。CLIではWorkspace revisionをWorld取得前後で照合し、読み込み中の変更を検知すればProjectionを拒否します。
- Accepted Business Procedureは現在のWorld JSONに含まれていないため、Businessは標準で `unresolved` です。#1125 Draft PR #1138のread-only DFD JSONを任意入力として与えた場合に限り、**明示的なApplication Pattern ID**で局所的に結合します。手作業・外部作業はFunctionへ紐付けずに別表示します。この結果は`reference-only`（未受理のFixture由来）で、Sourceとの同一revisionを証明しません。業務意味を勝手にacceptedへ昇格させません。既存の`tei source-index`を同じWorkspaceから取得すると、Source定義を`semantic-data`等のexact kind/IDで特定できます。取得時はrevisionの前後一致を確認します。ただし表示するのはIndexのdigest/位置情報であり、実際の編集権限や最新バイト検証を担保するものではありません。Sourceへ書き込む位置はShared Acceptance Previewの`sourceEdits`で確かめます。
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

`review.previewAcceptable`がfalse、review内容が未確認、あるいはsourceの競合があればAcceptを行わず、未解決として止めてください。Previewの`unresolved`を勝手に「変更なし」へ変換しません。現行Shared Acceptanceの既知仕様では、Relation追加によってPersistence Mappingの意味まで自動確定するとは限りません。

表上の編集は`view:"table"`、`subject:"attribute:sales.order_line.unit_price"`、`operation:"add-validation"`、`validationKind:"positive"`、`validationID:"sales.order_line.unit_price_positive"`とし、別候補として提出できます。Attribute型・Validationの妥当性はShared Acceptanceで最終確認します。既存Relationのcardinalityを`set`するoperationは未実装です。未対応編集を直接Source patchへ落とし込まないでください。

Worldとrevisionを先に取得してCLIへ渡す場合は、`-workspace`の代わりに`-world-file world.json -revision-file revision.json`を同時指定できます。`inspect`では`-source-index-file index.json`も指定でき、そのIndexにexact一致する定義位置を表示します。Workspace指定時はWorldとSource Indexを同じWorkspaceから読み、取得前後のrevision一致を検査します。ただし完全な原子snapshotやファイルの再読込時点の一致は保証せず、Accepted Sourceのrevision一致はShared Acceptanceで再確認します。JSONファイルを任意の第三者から受け取った場合、その内容がTEIのauthoritative Sourceと一致することはBridge単独では証明できません。

## Business参照の任意結合（#1125 / Draft PR #1138）

#1125の独立DFD Readerが配置されたcheckoutでは、同じScratch Workspaceを指定してDFD JSONを取得できます。

```sh
go run ./tools/reference-business-dfd -format json \
  -workspace "$scratch/workspace" > "$scratch/dfd.json"
node reference/studio-authoring/cli.mjs inspect \
  -workspace "$scratch/workspace" \
  -focus relation:sales.order.lines \
  -business-dfd "$scratch/dfd.json"
```

- DFDの`functionRefs[].id`が同じWorldのApplication Pattern IDにexact matchしたStepのみ、そのfocusとの関係を表示します。`data`/`relation`/`capability`参照の相違は`unresolved`にします。
- 原DFDの`source.file`/`source.pointer`は**非Canonicalな参照Fixtureの位置**であり、Source Indexの編集先ではありません。
- `manualOrExternalUnlinked`にFunctionなしの手作業等を別掲します。近接していることだけを理由に、選択Dataとの依存辺を捏造しません。大きなProcedureでは20件までとし切詰めを明示します。
- DFDにはWorkspace revisionのbindingが無いので、たとえ同一IDでも`freshness=unverified-no-model-revision-binding`です。旧DFDがSource revisionと一致すること、Business Procedureがacceptedであることは**証明できません**。最新Worldからの再取得と#1120 contractによる厳密な結合は後続です。
- #1138は別レーンのDraftであり、本PRへのコードコピーやCore変更は行いません。このコマンドは#1138のツールが存在するcheckoutでのみ動きます。通常のmainではDFDを渡さずBusinessを`unresolved`に保ちます。

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
| V00 | Worldから表/ER/機能近傍を同じ論理IDへ戻す。Source Indexのexact kind/IDで元定義のアンカーを表示。#1138 JSONの任意参照で非Canonical BusinessとのPattern結合を候補実装 | Accepted Businessとの同revision結合、Source byte再検査は未実装。4つのAccepted Projection受入は**未達** |
| V03 | ERのRelation追加・表のValidation追加をcandidate化 | PreviewのSemantic Diff / ImpactはShared Acceptance実行が必要 |
| V04 | Shared Acceptance CLI手順とscratch自動検証スクリプトを追加。Accepted Relation/Validationを表/ERへ再投影可能 | exact-head scratch E2Eのactual実行、Source reload比較は**未実行** |
| N05 | Derived直書き/任意PatchをBridgeが拒否 | 全Viewのeditable metadataの網羅 |
| N06 | candidateへ基準revisionを保持。Projection時の前後revision不一致を拒否し、scratch E2Eにstale負例を追加 | 実WorkspaceでのN06実行とaccept途中の並行write試験は未実施 |
| Invalid ref | exactな入力subject/Relation target不在・重複を拒否。Source Indexの曖昧な定義、範囲不正、危険なpathを拒否。DFDの参照切れはunresolved | 全モデルKind・全typed edge・Source byte最新性とBusinessの受理済み意味 |
| N12 | layout書換えは意味操作へ変換しない | 独立したView Metadataの保持・再生 |
| 性能 | focus近傍だけを返す、合成増量ケースあり | realistic 1万+件の時間/メモリ実測 |

### 次の開発順序

1. #1120のscope付きtyped subjectと現在のSource Indexのkind/ID対応を正式化する。現行のexact anchorは最初の足場とし、locatorのfile fingerprint最新性・source ownership・逆参照と編集可能箇所を結び付ける。
2. #1125の非CanonicalなDFD候補と受理済みBusiness Procedureを区別し、後者がSource Acceptanceの対象になってから#1126 Data、#1127 Functionと**同一revision**で4Projectionへ拡張する。Designer layoutは派生metadataへ分離する。
3. #267のShared AcceptanceへのPreview/Review/Accept/Undoを実Workspace scratchで検証し、N06のstale再現・競合時のSource byte不変・Decision ledgerの追跡を独立試験にする。
4. 運用規模の入力で性能測定を行い、Source上の変更箇所へnavigate→図/表変更→再投影をUI利用者に対して検証する。

このReferenceは製品Studio完成の証拠ではありません。V05/V08（Target/Runtime/Observation）は#1121/#1127の責務です。
