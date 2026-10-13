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

### 対応レコードの証拠を保持する

関連評価では、対象の具体例に現れた参照先を次のように区別する。

- `matchedTargetIds`：定義された全キー値が比較可能で、値が一致した参照先レコードの識別子。
- `possibleTargetIds`：既知のキー値に不一致はないものの、一部の値が欠落・比較不能なため、一致を否定できない参照先レコードの識別子。

この二つは、投影結果の`relationFindings`と、未解決・矛盾を扱う`review.json`の`issues`に保持する。既知の値が一つでも明確に不一致である参照先は、`possibleTargetIds`に含めない。両方ともレコードID順で安定し、同じ段階のスライドを複数並べてもレビュー項目を重複させない。

これらは**説明用の具体例内に存在するレコードの識別子**である。外部データベースのレコードの存在証明、参照の完全性や業務的な承認とは区別する。表示設定によって参照先の表が図に含まれていない場合でも、診断の根拠は入力された具体例の範囲に残る。

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

出力に`answer-handoff.json`と`change-triage.json`を追加する。回答、主張された回答者、適用文脈、根拠参照、質問への厳密な対応、未検証状態、次に必要なレビューを保持する。

**レビューの根拠を回答段階まで維持する。** 派生した質問の回答には`sourceIssue`を付け、検証時点の関連ID、レコードID、段階、スライドID、判定理由、確定一致した参照先（`matchedTargetIds`）と一致の可能性が残る参照先（`possibleTargetIds`）を、その時点のレビューから複写する。設計変更の振り分け資料にも同じ`sourceIssue`を引き継ぐ。人が自由記述の回答へ参照先IDを書き足しても、その記述から機械的な根拠を生成しない。

人やAIがもともとCaseへ記載した質問（`authored`）には派生元の診断が存在しないため、`sourceIssue`を付けない。いずれも回答は引き続き`unverified`であり、変更候補の`candidateReady`は`false`のままとする。`sourceIssue`の追加は回答資料の内容指紋にも反映されるため、異なる根拠を持つ旧回答を同一資料として扱わない。`confirmationRoute`は既存Confirmationの`identity.kind=example-review` / `identity.subject=例とCatalogの内容に束縛したハッシュID` / `basisSubjects=[{kind:semantic-relation,id:relation-id}]`へ対応させる**入力候補**である。同じ質問ID・同じCanonical Relationであっても、例のレコードやCatalogが変われば別の確認対象として再確認する。独自の回答台帳は増やさない。

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


## 関数従属性を具体例から反証する（#1126）

`ProbeBaseFunctionalDependencies`は、設計者が**仮説として明示した**同一表内の関数従属性を、説明用Caseの基底レコードと突き合わせる読み取り専用の検査です。入力には`Catalog`、`Example`、`FDHypothesis`を渡します。仮説には決定項目の組と、それに従属する項目の組を指定します。

- 決定項目がすべて一致し、従属項目に明確な不一致がある**二つのレコード**を見つけたときだけ、`contradiction`とレコードIDの組を返します。複合項目や、巨大なJSON数値の正確な比較にも対応します。
- 不明な値や型（`opaque`）が残る組は、無理に同値・相違と判定しません。既知の項目の不一致があれば、その組は対象から除外します。
- 反例が見つからなくても結果は必ず`unresolved`です。完全性の自己申告、全件がたまたま一致した例、物理主キーの一意性から、業務上の関数従属性を**採用済みの意味**へ昇格させません。
- レコードID順・仮説ID順に安定した結果を返します。入力は検証し、古いモデル版、存在しない項目、重複したレコード、重複した仮説は拒否します。

この小さな検査は基底時点の説明用レコードだけを扱います。段階ごとのスナップショット、実際のデータベース、候補キー・論理キーの正式な宣言、物理DBのPK/FK、正本への編集操作とは接続していません。例えば異なる物理行IDを持つ二件で同じ「受注番号＋明細番号」に別の商品が入っていれば、その**仮説に矛盾する具体例**として示せますが、どの項目が本来の業務識別子かは決めません。

関数従属性の正式な受理、モデル図での表示、Source Indexを伴う編集と変更影響は[#1126](https://github.com/hat47x/tei-platform/issues/1126)の後続作業です。結果を自動的にCanonicalやShared Acceptanceへ書き込む処理はありません。変更範囲の試験は次で実行します。

```sh
go test -count=1 ./tools/reference-example-slide/...
go vet ./tools/reference-example-slide/...
```


## 候補キーの反例検査（#1126）

`ProbeBaseCandidateKeys`は、設計者が明示した`CandidateKeyHypothesis`について、説明用Caseの基底レコードから**同じキー値を持つ異なる二件**を探す、読み取り専用の検査です。複合キーの各成分は既存のFD検査と同じ厳密な値比較を使い、巨大なJSON数値の丸めや記述順序の違いで結果が変化しないようにします。

例えば物理行IDが異なる二つの受注明細が、同じ「受注番号＋明細番号」を持つなら、業務識別子の仮説に対する`contradiction`としてレコードIDの組を示します。`Record.ID`やデータベースのPKが一意だからといって、仮説上の業務キーが一意とは判断しません。

- `collisions`：候補キーの全成分を比較でき、同値と判明した二レコードの組。これが一件でもあれば`contradiction`です。
- `unusableRecordIds`：欠損・null・比較不能な値（`opaque`型を含む）を持つレコード。これらを勝手に同じ値や別の値とは扱いません。
- `indeterminatePairs`：既知のキー成分には相違がなく、少なくとも一成分の比較を完了できないレコード対の件数。既知の相違がある対は`distinctPairs`に数えます。
- 一意に見える具体例、空の例、あるいは`completeTables`による自己申告があっても、結果は`unresolved`に留めます。**候補キーの最小性、必須性、永続的な一意性を証明しません**。

候補キーと関数従属性は関連しますが、検査結果を同一視しません。キー候補が衝突することと、関数従属性の従属項目が異なることは別の反例です。検査は`Catalog`と`Example`の同じモデル版を要求し、入力とTEIの正本を変更しません。

この機能はまだ`ir.World`の正式な業務キー、Source Index、ER図画面、物理PK/FK、Shared Acceptance、Runtimeへ結び付いていません。今後の設計では、これらの意味と変更権限を分離したまま横断参照を実装します。受入条件全体は[Issue #1126](https://github.com/hat47x/tei-platform/issues/1126)で管理します。


## 回答の出典参照を原資料の内容と照合する

`AuditAnswerSourceEvidence`は、回答の再束縛から始め、既存の入力元索引と原資料のバイト列を使って、出典参照を検査する**読み取り専用の関数**である。利用者が提供する読込関数は、対象の作業領域に閉じたものでなければならない。検査側が任意のパスを直接開くことはない。

まず入力元索引の全資料について、内容指紋と位置の妥当性を検証する。検証に失敗した場合は、個別の参照を部分的に合格扱いせず、検査全体を失敗させる。

派生質問の`sourceIssue.relationId`について、回答者が申告した`source-index`参照が同じ`semantic-relation`を指すか調べる。定義として索引化された場所が一つに定まり、回答に記載された資料指紋が現在のバイト列と一致し、指定位置の識別子も一致する場合だけ`source-range-checked`を返す。欠落・異なる識別子・古い指紋・位置だけの粗い参照・複数定義は、区別した未確認状態にする。原資料が検査途中で変化した場合も、成功を返さない。

文書・観測記録の参照には、この関数が読むための契約がないので`not-checked`を返す。利用者が直接書いた質問には派生元のRelationが無いため、入力元索引に同じ識別子が存在していても対応付けを作り出さない。

**`source-range-checked`は「指定された作業領域のバイト列に、対応する範囲と識別子が存在する」ことの確認に限る。** 索引の生成過程、記述の真偽、作成者の本人性、業務への適用可能性、実提供元の記録、確認・採用の権限までは証明しない。また、現在の実装は関数呼び出しに限られ、CLIでの安全な作業領域読込みや自動的な確認記録への連携は未実装である。これらを実装する際も、既存の作業領域の境界確認と資料の同一性確認を迂回しない。

## ER図から候補キー・関数従属性のレビューへ進む（#1126）

`-hypotheses`に候補キーと関数従属性の**仮説**を記載したJSONを与えると、従来の`projection.json`、`review.json`、SVGに加え、`model-hypothesis-review.json`を出力します。既存の回答・変更検討で使う`review.json`の指紋は変えません。

```sh
go run ./tools/reference-example-slide/cmd/tei-example-slide \
  -input tools/reference-example-slide/testdata/partial_inspection.json \
  -hypotheses /path/to/local-hypotheses.json \
  -out /tmp/tei-example-review
```

`local-hypotheses.json`の合成例（この例は基底段階に発注レコードが一件だけあるため、**反例が見つからなくても採用には進みません**）:

```json
{
  "schema": "tei.reference.example-model-hypotheses/v0",
  "candidateKeys": [
    {"id": "purchase-number-candidate", "table": "purchase_order", "fields": ["number"]}
  ],
  "functionalDependencies": [
    {"id": "quantity-to-number", "table": "purchase_order", "determinants": ["quantity"], "dependents": ["number"]}
  ]
}
```

入力の表名・属性名には表示名や推測した物理列ではなく、`Catalog`が持つ識別子を指定します。実Worldを読み取った場合は`CatalogFromWorld`の意味属性IDが使われますが、その属性へのSource Index根拠がこの検査で確認されるわけではありません。未知の項目、不正な候補、古いモデル版、未知のJSONプロパティは出力開始前に拒否します。

出力には、仮説ごとに次の区別を保ちます。

- `kind`は`candidate-key`または`functional-dependency`。両者の反例の意味は異なります。
- `table`と`fieldIds`は評価した意味上の表と属性。`diagramPlacementIds`には、この表が置かれた**すべてのER図上の配置**を記録します。属性を非表示にした配置も含み、表示設定で診断根拠を落としません。
- `stage=base`は、検証したレコードが基底時点だけであることを示します。`frameIds`には基底時点を表示するスライドだけを記載し、後続段階へ結果を流用しません。
- `recordPairs`は実例内で検出した矛盾の二レコード。比較不能なデータは`indeterminatePairs`等に残します。
- `exampleFingerprint`、`catalogFingerprint`、`hypothesesFingerprint`、`fingerprint`で評価に使った入力を区別します。仮説やデータが変わればレビューを再生成し、古い結果を転用しません。配置順や仮説の記述順だけの変更はレビューの意味を変えません。

`model-hypothesis-review.json`は既存の`review.json`や`answer-handoff.json`とは**別の未確認資料**です。回答を受理した、候補キーを採用した、関数従属性を意味モデルへ書き込んだ、あるいは既存の共通受理を通過したことを表しません。出力はファイル権限0600で保存し、元の具体例とともに機微なローカル検討資料として扱います。

正式な業務識別子・関数従属性の宣言、対象の実際の出所確認、段階別の評価、DB主キーとの対応、図上の直接編集と受理は、引き続き[課題 #1126](https://github.com/hat47x/tei-platform/issues/1126)の未完了範囲です。


## 候補キー・関数従属性の仮説と入力元の照合（#1126）

前段の`model-hypothesis-review.json`に含まれる表・属性の識別子を、既存の入力元索引と実際の意味定義に結び付ける読み取り専用監査を追加しました。

```sh
go run ./tools/reference-example-slide/cmd/tei-example-slide \
  -input /path/to/workspace-matched-case.json \
  -hypotheses /path/to/model-hypotheses.json \
  -workspace-source /path/to/tei-workspace \
  -out /tmp/tei-example-source-review
```

この例で`workspace-matched-case.json`の`catalog`は、指定した作業領域の意味モデルから導出した`Catalog`と一致している必要があります。`modelRevision`という文字列だけを合わせても受け付けません。`-workspace-source`は`-hypotheses`、`-input`、`-out`が必須であり、独立の`-world`指定とは併用できません。入力の照合が不成立なら**ファイル出力を開始する前に停止**します。

`model-hypothesis-source-audit.json`は、各仮説が参照した意味上の表（`semantic-data`）と属性（`semantic-attribute`）について、実際の定義場所を調べた結果を記録します。`source-range-checked`は、その作業領域を読み込んだ時点の入力元索引について、原資料のバイト列・指紋・対象宣言の識別子・位置が一致したことを意味します。各結果には資料の相対パス、内容指紋、該当範囲を含めます。定義が無い、複数ある、位置が別の宣言を指す場合は区別して未確認とします。

重要な境界があります。

- **表・属性の定義が確認できても、候補キーや関数従属性の業務上の意味は確認されません。** 現在のTEI意味モデルには、それらの仮説を正式に宣言する型付き操作がありません。入力元索引に存在しない「業務キー定義」の場所を捏造せず、既存の共通受理用の別操作へ読み替えもしません。
- 検査対象の`Catalog`は作業領域の読み込み結果から再導出し、同一読み込みのスナップショットと入力元索引を用います。独立したファイルの後追い読み取りを出所として使わず、スナップショットに無いパスも読み取りません。
- スナップショットの一致は、ファイル読み込み後の実ファイルの最新性、OSの競合状態、情報の真偽、作成者の本人性、受理する権限を証明しません。
- 出力は従来の`model-hypothesis-review.json`と別のファイルであり、`modelReviewFingerprint`と`hypothesesFingerprint`により同じレビュー対象へ結び付けます。既存の`review.json`・回答指紋を変更しません。保存は0600です。

正式な業務キー・関数従属性の宣言方式、意味モデルへの差分操作、出所付き人間確認、ER図からの編集は引き続き[Issue #1126](https://github.com/hat47x/tei-platform/issues/1126)で追跡します。


## 業務キー・関数従属性の宣言候補を型付きで持ち回る（#1126）

既存の仮説検査、ER図レビュー、Workspaceの原資料出所監査を通った後に、**まだ採用されていない**業務上の候補キーと関数従属性を、独立した型付きの宣言候補として生成できます。いまの共通受理が対応しない業務規則を、`add-relation`などの別操作へ読み替えることはありません。

```sh
go run ./tools/reference-example-slide/cmd/tei-example-slide \
  -input /path/to/workspace-matched-case.json \
  -hypotheses /path/to/model-hypotheses.json \
  -workspace-source /path/to/tei-workspace \
  -rule-intents /path/to/model-rule-intents.json \
  -out /tmp/tei-example-model-draft
```

`model-rule-intents.json`は次の形です。二つの指紋および`issueId`には、直前の`model-hypothesis-review.json`と`model-hypothesis-source-audit.json`から**実際の値**を指定します。例示文字列をそのまま使っても検証を通過しません。

```json
{
  "schema": "tei.reference.model-rule-draft-intent/v0",
  "modelReviewFingerprint": "sha256:<モデル仮説レビューの実指紋>",
  "sourceAuditFingerprint": "sha256:<出所監査の実指紋>",
  "requests": [{
    "issueId": "<レビュー内の実issueId>",
    "proposedRuleId": "sales.child.business-key",
    "rationale": "業務上この属性で一意に識別できるか確認する",
    "claimedEvidence": [{
      "kind": "document",
      "ref": "documents/business-rules",
      "fingerprint": "sha256:<参照文書の実指紋>",
      "claimedBy": "author-claim-only"
    }]
  }]
}
```

`claimedEvidence`は出所や発言者の**自己申告**であり、提出された文書の内容・本人性・承認権限を検証したものではありません。空の`claimedEvidence`も許可しますが、資料を省略しただけで承認できるわけではありません。

生成する`model-rule-declaration-draft.json`は、候補キーなら`keyFields`、関数従属性なら`determinants`/`dependents`を**異なる型付き領域**として保持します。また、既存の表・属性のSource Index位置（`sourceBindings`）、反例の判定、未解決の作業、レビューと出所監査の指紋を保持します。宣言候補のIDは意味上の表IDを前置し、仮のローカル名にはハイフンを使用できます。ただし、このIDはまだ現行CoreのCanonical IDや承認済み規則IDではありません。

| `status` | 意味 |
| --- | --- |
| `blocked-counterexample` | 具体例に確定した反例があるため、そのまま承認経路へ進めない |
| `blocked-source-definition` | 表・属性の定義場所の出所照合が不足または不一致 |
| `unverified-business-rule` | 反例・定義位置の障害は見つからなかったが、業務規則の真偽・承認は未確認 |

`unverified-business-rule`は「採用可能」を意味しません。資料の申告や人名を付けても、**どの状態も自動的にAcceptedにはなりません**。現在の`tools/reference-shared-acceptance`に対応する型付きdeltaはまだ無く、この成果物から正本編集・DDL生成・物理PK/FK変更・Runtimeへの作用は実行しません。実Workspaceを再読込し、版・原資料・仮説・レビューが変われば古い入力を拒否します。CLIの出力先は明示指定、ファイル権限0600です。

正式な規則の書式、業務識別子と物理キーの対応関係、Source Indexへの業務規則自体の定義、権限のある確認者による受理と影響解析は[Issue #1126](https://github.com/hat47x/tei-platform/issues/1126)の後続範囲です。


## 未採用の業務キー・関数従属性を独立した規則ソースへ記述する（#1126）

型付きの`model-rule-declaration-draft.json`ができた後、規則そのものに**独立した記述位置**を与える最小参照実装です。まだ`semantic/model.tei`の正本文法でも、`workspace.Load`が解釈する正式な規則でもありません。規則ファイルは**外部から提出された未採用の入力**として読み込み、既存のWorkspace Source Indexとは別の、`reference-candidate-key`／`reference-functional-dependency`を持つ参照用Source Indexに記録します。

```text
tei-model-rules v0
candidate-key sales.child.business-key on sales.child fields sales.child.parent_id
functional-dependency sales.child.business-to-owner on sales.child determinants sales.child.business_id dependents sales.child.owner_id
```

行頭の`#`はコメント、属性を複数指定する場合はカンマ区切りで記述します。正式な`ModelRuleDraftIntent`が参照する規則ID、種類、対象Data、全属性の組と**一字一句ではなく意味上の組として**照合し、規則の余分な追加、欠落、重複、別のDataへのすり替え、型の変更は拒否します。規則IDは提出された文書中の識別子トークンのバイト範囲へ対応付け、内容指紋と一緒に保持します。項目名と書式の見かけだけから正本規則を生成したり、物理PKを推測したりしません。

```sh
go run ./tools/reference-example-slide/cmd/tei-example-slide \
  -input /path/to/workspace-matched-case.json \
  -hypotheses /path/to/model-hypotheses.json \
  -workspace-source /path/to/tei-workspace \
  -rule-intents /path/to/model-rule-intents.json \
  -rule-source-file /path/to/proposed-rules.txt \
  -rule-source-path proposals/proposed-rules.txt \
  -out /tmp/tei-model-rule-review
```

`model-rule-source-review.json`に参照Source Index、個々の定義位置、元の`ModelRuleDeclarationDraft`の指紋、未解決の権限・事実確認事項を記録します。`-rule-source-path`は提出者が付ける**論理的な相対パスの申告**です。`-rule-source-file`の内容がその位置にある、ファイルの所有者が誰か、Workspaceが管理していることを証明しません。危険な絶対パス・正規化できないパス・大きすぎる入力は拒否し、ファイル出力の前に全規則を照合します。

提出されたソースへの参照には、Source Indexの`authoring`ロールと、正式な`semantic-*`とは異なる`reference-*`の種別を使用します。`definition`ロールや正本の`World.Semantics`には登録しません。`unverified-business-rule`は承認済みを意味しません。正式なSource Index登録、規則の受理、適用権限、他プラグインと共有する文法は、この実装を使って要件を確認した後の設計対象です。
