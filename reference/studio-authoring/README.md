# 三視点Studio authoring・最小Reference（Issue #1124）

状態: **初回G0/G1向けの部分実装**。現時点で三視点統合Editorの製品受入・Source再投影・実行Evidenceは成立していません。

既存の[Shared Acceptance](../../tools/reference-shared-acceptance/README.md)と[Studio authoring計画](../../plan/issue/000046-studio-shared-acceptance-authoring-loop.md)を使用します。このディレクトリに新しいCanonical正本、Patchエンジン、Validation、Acceptance、Authorizationを作りません。親の[#1119](https://github.com/hat47x/tei-platform/issues/1119)と[#1120](https://github.com/hat47x/tei-platform/issues/1120)の最終契約が確定するまでは、ここにあるJSONは**Reference限定**の運搬形式です。

## 検証する最小境界

- 入力はTEIの既存World JSONと、同じWorkspaceから得たmodelRevision。BridgeはWorldのData/Relation/AttributeとApplication Patternの**元のID**を使い、表・ER・機能の近傍Projectionを読み取り専用で返します。
- Accepted Business Procedureは現在のWorld JSONに含まれていないので、Business projectionは明示的に `unresolved` です。UIに架空のActor/Step/typed edgeを作りません。Source Indexの正確な位置との結合も未実装で、`sourceNavigation.status=unresolved`とします。
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

Worldとrevisionを先に取得してCLIへ渡す場合は、`-workspace`の代わりに`-world-file world.json -revision-file revision.json`を同時指定できます。いずれも取得の原子性を保証せず、Accepted Sourceのrevision一致はShared Acceptanceで再確認します。JSONファイルを任意の第三者から受け取った場合、その内容がTEIのauthoritative Sourceと一致することはBridge単独では証明できません。

## 試験と境界

```sh
node --test reference/studio-authoring/bridge.test.mjs
# 正確な候補PR headのtrusted full checkoutで、上記に加え
tools/repository-health-check.sh
```

| Scenario | 初期実装 | 未検証・未対応 |
| --- | --- | --- |
| V00 | Worldから表/ER/機能近傍を同じ論理IDへ戻す | BusinessとSource Index locatorは未結合。4Projection受入は**未達** |
| V03 | ERのRelation追加・表のValidation追加をcandidate化 | PreviewのSemantic Diff / ImpactはShared Acceptance実行が必要 |
| V04 | Shared Acceptance CLIへの手順を定義 | exact-head scratch E2Eのactual実行、Source reload比較は**未実行** |
| N05 | Derived直書き/任意PatchをBridgeが拒否 | 全Viewのeditable metadataの網羅 |
| N06 | candidateへ基準revisionを保持 | 実Workspaceでのstale/並行変更試験はShared Acceptance側で要確認 |
| Invalid ref | exactな入力subject/Relation target不在・重複を拒否、出力の参照切れはunresolved | 全モデルKind・全typed edge・Business参照 |
| N12 | layout書換えは意味操作へ変換しない | 独立したView Metadataの保持・再生 |
| 性能 | focus近傍だけを返す、合成増量ケースあり | realistic 1万+件の時間/メモリ実測 |

### 次の開発順序

1. #1120のtyped subjectとSource Index anchorを正本にしたAdapterを設け、Sourceに戻れるlocatorを付与する（推測したfile pathを出さない）。
2. #1125のAccepted Business Procedure、#1126のData定義、#1127の機能投影を**同一revision**で読み、4Projectionへ拡張する。Designer layoutは派生metadataへ分離する。
3. #267のShared AcceptanceへのPreview/Review/Accept/Undoを実Workspace scratchで検証し、N06のstale再現・競合時のSource byte不変・Decision ledgerの追跡を独立試験にする。
4. 運用規模の入力で性能測定を行い、Source上の変更箇所へnavigate→図/表変更→再投影をUI利用者に対して検証する。

このReferenceは製品Studio完成の証拠ではありません。V05/V08（Target/Runtime/Observation）は#1121/#1127の責務です。
