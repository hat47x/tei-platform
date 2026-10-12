# Example-driven ER slide reference

状態: **参照実装はPR #1128から2026-10-11にmainへ統合済み**。ただし統合時の全体品質ゲートをPASSした証跡は未確認。この文書の機能説明は現在の実装を基準とし、確認済みの動作実績とは区別する。

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

SVGは最小の描画確認であり、完全な専用ER記号・衝突回避・印刷調整・ブラウザ表示・WASM動作を保証しません。現在の描画では表示対象の全レコードを扱い、Projectionにも全表示レコードを保持します。

実データの安全な利用、AI候補生成、業務上の最終判断、およびShared Acceptanceでの正本変更までの実証は後続の検証対象です。現在はWorld由来のRelationKeyで説明用Caseのレコードを限定的に照合しますが、部分データや型不確実性は未解決として保持します。**関係線を描けたことを参照整合性検証済みと読んではいけません。** この参照実装は独立した`std.*` Semantic Extensionの採用根拠にもなりません。

設計の詳細は [M1設計候補](../../plan/design/EXAMPLE_ER_SLIDE_M1_2026-10.md) を参照してください。

## 三値の関連評価（追加した実装候補）

スライドの`relationFindings`には、対象Snapshotに存在するSourceレコードごとの関連評価を追加した。

| result | 限定された意味 |
| --- | --- |
| `supported` | 対象テーブルの全レコードがこの説明用Snapshot内に存在すると**Case自身が申告**した範囲で、RelationKeyによる照合が成立した。実際の業務やProviderの正しさを証明しない |
| `contradiction` | `cardinality=one`の関連で、一つのSourceレコードから**複数の一致Target**が観測されるなど、手元の具体例だけで否定できる条件 |
| `unresolved` | 対象テーブルが部分的、RelationKey未定義、キー値不足、比較不能、ゼロ件の場合の必須性不明など、成立も不成立も証明できない |

`Example.completeTables`は**基底（base）Snapshotに限る**完全性申告である。後続のStageは各自の`Stage.completeTables`で申告する。後続Stageで無指定の場合、基底Snapshotが全件を含むとしても完全性を引き継がず部分抜粋と扱う。これはユーザーまたは生成AIが供給した**例に対する仮定**であり、実環境から確認された網羅性のEvidenceではない。Source Indexや実データ観測から正当化されない限り、Productionの完全性保証へ昇格しない。

RelationKeyがない（あるいは対象の属性値を比較できない）場合は、自動結合を試みず`unresolved`とする。`one`で一致Targetが複数見つかった場合だけ、Datasetが部分的でも矛盾が確定する。一致Targetが0件でも、意味モデルに必須性が無ければ、欠如を直ちに`contradiction`にはしない。

複合RelationKeyは各キー対の一致を調べる。**評価結果はキーの記述順序に左右されない**。一部のキー値が不明でも、別のキー値で確実に不一致なら、そのTargetは候補から除外できる。一方、既知のキー値に不一致がなく、不明なキー値が残る場合は対応未確定とする。整数や小数のJSON表記は有理数で正確に比較し、巨大な指数を含む値は比較不能として資源消費を制限する。

### 回帰試験の拡充

`relation_assessment_test.go`で、複合キー、大整数の値精度、部分Datasetでの安易な合格、複数一致Target、欠落・未定義キー、出現0件、Stageとスライド順序、完全テーブル宣言の誤りを反証する。

`world_adapter_test.go`では、実際の正規化済みWorldの`sales.order.customer`を合成レコード値で照合し、同じWorldにあるKeylessな`sales.order.lines`が安易に合格しないことを確認する。どちらも実Providerや本番データの検証ではない。

**検証実行の状態:** 本文書の参照実装について、統合済みmainのexact headでGo test・gofmt・full Repository-healthをPASSしたEvidenceは未取得である。PR #1128は既にマージされており、未統合という表現は古い。

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

出力に`answer-handoff.json`と`change-triage.json`を追加する。回答、主張された回答者、適用文脈、根拠参照、質問への厳密な対応、未検証状態、次に必要なレビューを保持する。`confirmationRoute`は既存Confirmationの`identity.kind=example-review` / `identity.subject=例とCatalogの内容に束縛したハッシュID` / `basisSubjects=[{kind:semantic-relation,id:relation-id}]`へ対応させる**入力候補**である。同じ質問ID・同じCanonical Relationであっても、例のレコードやCatalogが変われば別の確認対象として再確認する。独自の回答台帳は増やさない。

### 複数の権限境界

1. **回答案の束縛:** `BindAnswers`は古いReview fingerprint、存在しない質問、重複回答、未知のdisposition、根拠参照なしの項目等を検出する。正本は更新しない。
2. **根拠照合と確認記録:** 既存`tools/reference-shared-acceptance confirm-record`が、WorkspaceのSource Indexへ`basisSubjects`を解決して記録する。回答Handoffから**自動でconfirm-recordを実行してはいけない**。既存Confirmation自体もactor種別を自己申告で受け取るため、第三者による人間認証まで成立しているわけではない。
3. **意味モデルの変更:** 確認記録を作成してもモデルは変更されない。必要になった具体的なdeltaだけを既存Shared Acceptanceで改めてpreview / review / acceptする。

`claimedBy.kind=human`は回答JSON上の申告であり、実際に人間が判断した証明ではない。AI回答は人間レビューを要する。`evidence`の参照やfingerprintもこの段階では自己申告であり、対応する原本の現在性・文脈適合性を検証したEvidenceではない。根拠無しでは`evidence-needed`、調査継続なら`investigation-needed`として返す。

注: この参照機能が使う`ModelRevision`はSemantic Worldのハッシュであり、Shared Acceptanceの管理対象Workspace全ファイルから計算した`baseModelRevision`とは同一ではない。変更候補を作る際は**後者を現在のWorkspaceから改めて取得**しなければならない。

GoコードとテストはPR #1128に追加したが、対象BranchのGoテスト・Format・Repository Health Checkはまだ実行していない。

## 回答案から安全な次の変更経路を選ぶ

`change-triage.json`は、レビューに結び付けられた各回答の`disposition`を次のいずれかへ振り分ける、**read-onlyの進路確認**である。

| 回答の方向 | 次の経路 |
| --- | --- |
| `clarify-example` | Case編集案を人間が確認。Canonical Sourceを書き換えない |
| `retain-model` | 根拠と適用条件を確認し、必要に応じて既存Confirmationへ記録 |
| `investigate` | 根拠収集。未解決のまま残す |
| `propose-model-change` | 原因を確定し、Source Index・現在のWorkspace revision・実装済みtyped operationを点検。Shared Acceptanceへ直接投入しない |

この段階では`candidateReady=false`を常に出力する。回答を受け取っただけでSemantic Patchを作ったり、実行可能性を保証したりしない。

既存Shared Acceptanceの現在のtyped operationには`add-relation`があるが、既存RelationのRelationKeyを編集する`update-relation`や`add-relation-key`は無い。そこで、既存Relationのキー不足を理由とした変更希望は`existing-relation-key-edit-not-supported-by-current-typed-operations`と分類する。別のIDのRelationを追加して既存の関係を置き換えることはしない。

また、多重度`one`なのに複数の一致先がある場合、原因は業務モデル、RelationKey、合成Caseのいずれにもあり得る。この段階での変更先は`model-versus-example-cause-not-determined`とし、変更箇所を自動決定しない。

CLIのCase入力も未知fieldを拒否する。例えば`completeTables`の綴り違いを黙って無視すると検証範囲が変わるため、誤入力として扱う。

これらはPR #1128の候補実装であり、対象BranchのGoテストやRepository-health GateのPASSは未確認である。

## RelationKeyの型同一性 — 表面的な値の一致を過信しない

`CatalogFromWorld`では、従来の表示用`type`（string / integer / opaqueなど）に加えて、各属性の**Canonical typeRef**を`semanticType`として保持する。

値がたまたま`"001"`同士であっても、片方がSupplier ID、もう片方がCustomer IDなら、それだけで業務上の同一性を証明できない。このため、RelationKeyで結合する両属性の`semanticType`が異なる場合、または片側だけ型identityを持つ場合は、`relation-key-type-not-confirmed`として`unresolved`にする。表示用基本型しか無い合成Catalogでは、両基本型が同一であることを要求する。

これはCanonicalな型IDの**同一性に関する保守的な条件**であり、異なるAliasが認証済み変換で相互運用できるか、あるいは型が同一なら実値のドメイン制約を満たすかは判断しない。ValueやReferential Integrityの完全検証に読み替えない。

`relation_assessment_test.go`へ、文字列一致なのに意味的な型が異なる場合、双方の型IDが同じ場合、片側の型IDだけがある場合の否定・肯定試験を追加した。対象PR全体のGoテストは未実行。

## 段階ごとに独立した網羅性申告

同じモデルの説明であっても、基底時点で抽出した全件データが、受入・検収など別Stageでも全件の根拠になるとは限らない。従来のCase全体の`completeTables`を基底Stage専用へ限定し、後続Stageでは`Stage.completeTables`を独立して指定する契約へ変更した。**Stageの無指定を自動的な継承に解釈しない**。同じ明細が図に載っていても完全性の根拠が別であれば三値評価は`unresolved`に留まる。

追補したテストでは、後続Stageへ基底の申告が漏れないこと、後続Stage独自の申告でのみ`supported`が許されること、未定義テーブルへのStage申告を拒否することを確認対象とした。これは説明用Case内での申告であり、各時点の実Providerを完全観測したEvidenceにはならない。

## SVGの複数レコード表示とレビュー概要（2026-10-10）

従来のSVG表示は各テーブルの先頭レコードの値だけを描いていたため、データセットに二つの一致Targetがあり`contradiction`を検出できても、**生成された図ではその証拠の一部が見えない**というギャップがあった。

SVGレンダラーを変更し、各配置テーブルについてSnapshot内の**全表示用レコード**を順に描画する。レコードIDと表示許可済みの各フィールド値を行として表示し、件数に応じて箱の高さを拡張する。説明用レコードが0件の場合もスキーマ上のフィールド名を残し、`(no example records)`と明示する。

スライド上部へタイトル、Stage、関連評価の`Supported / Contradictions / Unresolved`件数を表示する。線にRelation IDまたはsynthetic reference fieldをアクセシブルなラベルとして追加する。自由な文字列はXML文字としてescapeし、長いラベルだけを**SVG上で省略表示**する。投影JSONの元値は省略しない。既存の手動配置座標やField Show/Hideは引き続き尊重し、配置の衝突を勝手に解消しない。

この描画方式も正式なER図記法の完全実装ではない。線のルーティング、重なりの解消、印刷向けのページ分割、件数が大きい場合のページネーションは後続の検証対象とする。

新しいレンダラーと三つの否定・表示テストの同等コードを独立Go 1.23.2 scratch moduleで`go test -count=1 -v ./...`（3件）と`go vet ./...`実行し、PASSを確認した。ただしこれは**新レンダラーの隔離した動作検証**であり、PR head全体のGo testやRepository-health GateのPASSではない。
