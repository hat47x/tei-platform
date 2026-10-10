# Example-driven ER slide reference

状態: **独立したGo参照実装候補（未統合・テスト未実行）**。

発注10個・一部入荷6個・検収合格4個／不合格2個の合成データを、手動配置のER図上で三つのスライドとして表示するための最小実験です。データ構造だけでは分からない業務上の疑問（不合格数量を履行残に含めるか）を未解決のまま保持します。

## 実行

リポジトリルートから:

~~~sh
# 参照パッケージとCLIを対象に試験
go test ./tools/reference-example-slide/...

# 合成Caseを指定して三枚のSVG、投影JSON、設計レビューJSONを出力
go run ./tools/reference-example-slide/cmd/tei-example-slide \
  -input tools/reference-example-slide/testdata/partial_inspection.json \
  -out /tmp/tei-example-slides

# 正規化済みWorldから、表示用Catalogとrevisionを読み取り専用で確認
go run ./tools/reference-example-slide/cmd/tei-example-slide \
  -world reference/world.json -catalog
~~~

`testdata/partial_inspection.json` に、読み取り専用Catalog、Case、Diagram、Presentationを含みます。GoテストではJSON読込から全段階のSnapshot投影とSVG文字列生成までを検証します。現時点でこれらのコマンドがPASSしたという記録はありません。CLIでは実WorldとCaseのrevisionが異なる場合、黙って修正せず拒否します。

## この実装が保証を目指す範囲

- 現在のモデルrevisionに固定した例を受け取り、古い版の例は拒否する。
- 必須項目、基本型、レコード識別子、外部参照の存在を検証する。
- Stageの説明用差分からSnapshotを構築する。Actionの実行や現実の業務事実を証明しない。
- 手動配置を保ち、図ごとの表示項目上書きによるフィールド選択ができる。
- フィールドを非表示にしても、定義済みの参照関係が図から消えない。
- スライド表示順とStage差分の順序を区別する。
- 元の入力を変更しない。疑問に勝手に回答せず、Questionを出力へ引き継ぐ。
- `CatalogFromWorld`で、実際の`reference/world.json`の`ir.World.Semantics`からData・Attribute・Relation・RelationKeyを読み取り専用で導出する。複合のRelationKeyやKeys未定義のRelationを単純なFKに変換しない。
- SVGでは文字列をエスケープする。

## 非対応・今後の検証課題

合成Catalogによるスライド実験に加え、実TEIの正規化済みWorldのSemantic部分から読み取るAdapterを実装候補として追加しました。ただしSource Indexの根拠まで結び付けるものではなく、実World由来の型については`std.identifier`、`std.decimal`、`std.money`等を安易にstring/float64へ狭めず`opaque`とします。これは型検証済みを意味しません。またWorldのAttribute定義に無い必須性や表示既定値を推測しません。

SVGは最小の描画確認であり、完全な専用ER記号・衝突回避・印刷調整・ブラウザ表示・WASM動作を保証しません。各テーブルの例示値は先頭レコードのみを表示しますが、Projectionは表示対象の全レコードを保持します。

実データの安全な利用、AI候補生成、業務上の整合性の判定、詳細設計への提案とShared Acceptance接続は後続の検証対象です。WorldのRelationKeyが存在しても、説明用Caseのレコード間で業務キーを検査する機構は未実装です。**関係線を描けたことを参照整合性検証済みと読んではいけません。** この参照実装は独立した`std.*` Semantic Extensionの採用根拠にもなりません。

設計の詳細は [M1設計候補](../../plan/design/EXAMPLE_ER_SLIDE_M1_2026-10.md) を参照してください。

## 三値の関連評価（追加した実装候補）

スライドの`relationFindings`には、対象Snapshotに存在するSourceレコードごとの関連評価を追加した。

| result | 限定された意味 |
| --- | --- |
| `supported` | 対象テーブルの全レコードがこの説明用Snapshot内に存在すると**Case自身が申告**した範囲で、RelationKeyによる照合が成立した。実際の業務やProviderの正しさを証明しない |
| `contradiction` | `cardinality=one`の関連で、一つのSourceレコードから**複数の一致Target**が観測されるなど、手元の具体例だけで否定できる条件 |
| `unresolved` | 対象テーブルが部分的、RelationKey未定義、キー値不足、比較不能、ゼロ件の場合の必須性不明など、成立も不成立も証明できない |

`Example.completeTables`で、説明用Snapshotの対象テーブルが全件列挙されていると申告できる。省略時はすべて部分抜粋として扱う。これはユーザーまたは生成AIが供給した**例に対する仮定**であり、実環境から確認された網羅性のEvidenceではない。Source Indexや実データ観測から正当化されない限り、Productionの完全性保証へ昇格しない。

RelationKeyがない（あるいは対象の属性値を比較できない）場合は、自動結合を試みず`unresolved`とする。`one`で一致Targetが複数見つかった場合だけ、Datasetが部分的でも矛盾が確定する。一致Targetが0件でも、意味モデルに必須性が無ければ、欠如を直ちに`contradiction`にはしない。

Composite RelationKeyもキー対を順番どおり照合する。整数や小数のJSON表記を比較するときは有理数による完全な数値比較を使い、巨大指数を含む値は比較不能に分類してリソース消費を制限する。

### 回帰試験の拡充

`relation_assessment_test.go`で、複合キー、大整数の値精度、部分Datasetでの安易な合格、複数一致Target、欠落・未定義キー、出現0件、Stageとスライド順序、完全テーブル宣言の誤りを反証する。

`world_adapter_test.go`では、実際の正規化済みWorldの`sales.order.customer`を合成レコード値で照合し、同じWorldにあるKeylessな`sales.order.lines`が安易に合格しないことを確認する。どちらも実Providerや本番データの検証ではない。

**検証実行の状態:** Go test / gofmt / full Repository-healthは未実行。これらを実行したというEvidenceは存在しない。

## 設計レビューへの接続（Review Packet）

`Review(input)`は、同じCaseを再検証した結果から、レコード・関連・ステージ単位の診断を**設計者への質問**に変換する。CLIで`-out`を指定すると`review.json`も生成する。

- `authoredQuestions`: 設計者やAIがCaseに記載した既存の未解決の問いを、そのまま保管・表示する。
- `issues`: `supported`以外のRelationFindingから決定論的に導出するレビュー候補。モデルrevision、Stage、Relation、Source Record、Frame IDs、判定理由、必要なEvidence、確認すべき設計対象を含む。
- Frameが重複しても同じStageの同じ論点を増殖させない。スライド順序の変更でも同じIDに収束し、revision変更では古いレビューIDを使い回せない。
- 対応先は`example-data`、`example-completeness`、`semantic-relation`、`business-meaning`、`cross-view-contradiction`などを示す**レビューの手掛かり**であり、書き込み命令ではない。
- `contradiction`も、AIが勝手にKeyやCardinalityを書き換えるのではなく、どちらが誤っているか確認を要求する。

このReview Packetは、確認済みの設計判断やSemantic Patchそのものではない。確認者、適用対象、根拠を確定する作業と、既存Shared Acceptanceによる正本変更は別段階として扱う。`relation-keys-not-declared`から`AddRelationKey`のような操作が必要だと自動決定することもない。

`Review(input)`自体も入力のWorld、Case、Diagramを変更しない。局所のunit testでは、質問の出典参照、安定ID、スライドの重複排除、回答・Patchの捏造防止などを受入対象とした。**対象PR全体のGoテストは未実行。**

## 回答を検討資料として受け渡す（M1回答Handoff候補）

元の`review.json`には、モデルrevision、Example / Catalogの内容fingerprint、Review全体のfingerprintが入る。回答を記録する前に、**必ずその時点の`reviewFingerprint`を入力へコピーする**。元のCase、Catalog、質問内容が変われば古い回答は拒否する。同じCaseの同じStageを複数フレームで表示する順番だけの変更では、Reviewの意味を変えない。

回答JSONの例（`<...>`は`review.json`から実際の値へ置換する。固定fixtureとしては使わない）:

~~~json
{
  "schema": "tei.reference.example-review-answer/v0",
  "caseId": "multi-key",
  "modelRevision": "<review.json:modelRevision>",
  "reviewFingerprint": "<review.json:fingerprint>",
  "answers": [
    {
      "question": {"kind": "derived", "id": "<review.json:issues[0].id>"},
      "response": "このCaseでは一部の顧客レコードだけを例示している",
      "disposition": "clarify-example",
      "context": "当該Caseのbaseステージに限る",
      "claimedBy": {"kind": "human", "id": "reviewer-id"},
      "evidence": [{"kind": "document", "ref": "design-review-notes"}]
    }
  ]
}
~~~

`question.kind`は`authored`（人やAIがCaseに記載した問い）または`derived`（RelationFindingから導出した問い）。`disposition`は`clarify-example`、`propose-model-change`、`retain-model`、`investigate`の候補に限定する。いずれも**最終採否・変更操作を意味しない**。

実行例:

~~~sh
go run ./tools/reference-example-slide/cmd/tei-example-slide \
  -input tools/reference-example-slide/testdata/partial_inspection.json \
  -out /tmp/tei-slide-review

# review.json内のcaseId / modelRevision / fingerprint /
# authoredQuestionsまたはissuesを見て、対応するanswers.jsonを作る。
go run ./tools/reference-example-slide/cmd/tei-example-slide \
  -input tools/reference-example-slide/testdata/partial_inspection.json \
  -answers /tmp/answers.json \
  -out /tmp/tei-slide-answered
~~~

出力に`answer-handoff.json`を追加する。回答、主張された回答者、適用文脈、根拠参照、質問への厳密な対応、未検証状態、次に必要なレビューを保持する。`confirmationRoute`は既存Confirmationの`identity.kind=example-review` / `identity.subject=issue.id` / `basisSubjects=[{kind:semantic-relation,id:relation-id}]`へ対応させる**入力候補**である。独自の回答台帳は増やさない。

### 複数の権限境界

1. **回答案の束縛:** `BindAnswers`は古いReview fingerprint、存在しない質問、重複回答、未知のdisposition、根拠参照なしの項目等を検出する。正本は更新しない。
2. **根拠照合と確認記録:** 既存`tools/reference-shared-acceptance confirm-record`が、WorkspaceのSource Indexへ`basisSubjects`を解決して記録する。回答Handoffから**自動でconfirm-recordを実行してはいけない**。既存Confirmation自体もactor種別を自己申告で受け取るため、第三者による人間認証まで成立しているわけではない。
3. **意味モデルの変更:** 確認記録を作成してもモデルは変更されない。必要になった具体的なdeltaだけを既存Shared Acceptanceで改めてpreview / review / acceptする。

`claimedBy.kind=human`は回答JSON上の申告であり、実際に人間が判断した証明ではない。AI回答は人間レビューを要する。`evidence`の参照やfingerprintもこの段階では自己申告であり、対応する原本の現在性・文脈適合性を検証したEvidenceではない。根拠無しでは`evidence-needed`、調査継続なら`investigation-needed`として返す。

注: この参照機能が使う`ModelRevision`はSemantic Worldのハッシュであり、Shared Acceptanceの管理対象Workspace全ファイルから計算した`baseModelRevision`とは同一ではない。変更候補を作る際は**後者を現在のWorkspaceから改めて取得**しなければならない。

GoコードとテストはPR #1128に追加したが、対象BranchのGoテスト・Format・Repository Health Checkはまだ実行していない。
