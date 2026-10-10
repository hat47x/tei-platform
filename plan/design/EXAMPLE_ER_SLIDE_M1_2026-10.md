# 具体例からER図スライドへつなぐ参照実装（M1）

状態: 実装候補。読み取り専用World Adapterは実装候補として追加済み。PR全体のGoテスト・整形・統合Gateは未実施。

## なぜ必要か

設計者が記述したデータ構造は抽象的であり、発注、部分入荷、検収などの具体例にすると初めて「残数量は何を意味するか」などの疑問が見える。これまでの自由記述スライドの有用性を維持しながら、生成AIが作ったデータ例を検証可能にする。

この文書は既存RQ-026（三視点からの相互検証）の評価案であり、独立したRequirementやCanonical schemaを新設しない。

## M1の完了条件

1. 既存TEIの正本から読み取り専用で導出したテーブル・項目・関連を入力として利用する。参照実験では合成Catalogを明示し、Worldへの実結合とは区別する。
2. モデルrevisionに固定したDatasetを読込み、必須フィールド、型、レコード識別、参照先の存在を決定論的に検査する。
3. 静的な構造説明と状態変化説明を混同しない。説明順とScenarioの実行順を区別する。
4. 手動配置のER図を維持し、テーブル定義の項目表示既定値を個別の図で差分上書きできる。
5. スライドの切替えによって座標、元のDataset、意味モデルを変更しない。非表示の外部キーでも関連線は保持する。
6. 同じCaseを構造検証、スライド、質問、回帰検証へ再利用できる。未確定の業務ルールはUnknownであり、Passにはしない。
7. 設計変更は既存Shared Acceptanceを経由し、提案だけで正本を変更しない。

## 実験の最小構成

- Input: 読み取り専用Catalog、具体例Case（基底データ・段階差分）、手動配置Diagram、表示順を持つPresentation
- Deterministic: Schema/Reference Validator、段階Snapshot再構成、図への投影
- Output: スライドごとの表示用Projection、構造診断、未解決Question
- Optional: 小さなSVG出力。既存ER図固有の描画・全記号体系まで保証しない

基底データとして発注10個、その後入荷6個、検収合格4個・不合格2個を扱う。これは合成事例であり、実際の業務ルールが確定した証拠ではない。「不合格数量は履行残に入るか」を明示的な未解決事項に保つ。

## 過剰な実装を避ける境界

- Exampleは独立したCase資産。DataEntity・Action・Rule等を拡張しない。
- Stageのデータ差分は説明用であり、Action実行や外部事実を意味しない。
- Presentationの順番だけで時間・責任移転・Handoffを導出しない。
- AI生成はCase候補の入口に過ぎない。ValidatorとAuthorityを兼ねない。
- 旧スライドの自由記述を排除しない。
- Runtime Providerへの接続、実データ取込、WASM/VS Code展開はM1の完了条件ではない。

## 先に潰す反例

- 同じテーブルを複数の図に配置しても表示項目を独立指定できる。
- 合成Caseの明示的なrecord-ID参照が欠落した場合は拒否する。一方、Semantic RelationKeyの照合では部分抜粋や任意性を区別し、不明を矛盾と誤判定しない。隠した外部キーの関連線も消さない。
- モデルrevisionを変えた例を従来のPassとして表示しない。
- スライド順を入れ替えてもデータの変化順は変わらない。
- 必須項目欠落、型不一致、存在しないStage、重複IDを検出する。
- 取消・部分入荷・同時更新は、既存Actionの実行試験が成立するまで業務的にPassと断言しない。
- 画面レイアウトの変更は意味モデルの変更として記録しない。

## 続きの判断材料

まず合成Caseで型・参照・表示機構の縦断試験を作る。次に既存TEIのWorldから必要な定義を取得する読み取り専用Adapterを接続する。Semantic Patchとの結合は、検出した具体的な設計ギャップが確定してから行う。対象となる機構は標準プラグインで実装可能な設計とするが、この段階でstd.*の永久identityを先行固定しない。

関連文書:
- [Product Requirements](../../requirements/TEI_PRODUCT_REQUIREMENTS.md)
- [Business Design Requirement](../../requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md)
- [Business System Design](../../docs/business-system-design.md)
- [実務シナリオ縦断実証](../issue/000075-practical-scenario-mvp-demonstration.md)
- [Definition-centered Feedback Loop](../../docs/definition-centered-feedback-loop.md)

## 2026-10-10の候補実装（未検証）

独立した`tools/reference-example-slide/`とCLI候補を追加した。既存の`internal/`や`reference/workspace`には書き込んでいない。

- `CatalogFromWorld(ir.World)`: 既存の正規化済みSemantic Worldを読み取り、Canonical Data / Attribute / Relation / RelationKeyを表示用Catalogへ導出する。RelationのKeysなしはKeysなしのまま残す。単一・複合のキーを、物理FKや文字列のRecord ID参照へ勝手に変換しない。
- 合成Caseの`Build`: 段階差分を検証しSnapshot化する。Relationの向きを保った接続情報と手動座標を投影する。
- CLI: 合成Caseの`projection.json`と`slide-001.svg`等を明示的な出力先に書く。正規化済みWorldからCatalog/revisionを表示する別モードを持つ。
- 回帰試験: Checked `reference/world.json`のCustomer/Order/OrderLineにある現実のRelation、RelationKey、KeysなしのRelation、型alias、revision、古いCase拒否を検査対象とした。

### テスト結果と残余の境界

上記は**コミット済みの実装候補**であり、Goテスト・gofmt・全Repository-healthがPASSしたと確認したものではない。Draft PRを閉じず、実行確認とコード整形、exact-head Gateが揃うまでmergeしない。

- Worldの`Attribute`だけでは必須性は導出できない。`Required=false`が業務上任意である証拠ではない。
- `std.identifier`、`std.decimal`、`std.money`等は、このReferenceの基本型検証では`opaque`とし、厳密な値の妥当性を主張しない。
- Semantic RelationKeyを使う具体レコードの照合は**候補実装として追加した**。部分抜粋・未定義Key・不明な必須性では`unresolved`とし、元の業務仕様や実データの整合性を証明しない。図面に線を引けることと検証済みは別である。
- Catalog revisionはWorld内のSemantic名前空間のハッシュであり、業務の実データの観測・時点、Application、Target/Provider状態を含まない。
- 現段階ではReferenceが意味定義の必須性・データ型を完全再現するわけではなく、Studioの正本受理経路も使っていない。

この境界を維持しながら、次は実際のCaseをSemantic RelationKeyへ適合させて検証可能にするか、未知・不完全を返すかの最小契約を反証例から定める。データ例が一部しかない説明用スライドで、参照先が存在しないことを不整合と即断してはいけない。

## 部分データの扱い — RelationKey評価の追加候補

データモデル上の関連線の表示と、具体的なレコードの関連照合を別機構にした。図に関連が表示されるだけでそのケースの参照整合性を確認済みとすることはできない。

`Example.completeTables`は、各Snapshotについて「このテーブルのレコードをすべて掲載した」という**Case内部の申告**である。指定していないテーブルは部分的な抜粋とし、Targetが見えないだけでは矛盾を主張しない。この申告には業務事実の採用権限を与えない。

三値の`relationFindings`は、relation ID / source record IDを単位に出す。判定結果はこの説明用Datasetの範囲だけに限定する。

- `supported`: Sourceに必要なキー値があり、Targetが完全と申告され、キー値が一致し、`cardinality=one`の場合は一致Targetが1件（`many`では列挙可能）。これは**モデルが実務上正しい証拠ではない**。
- `contradiction`: `one`に対して可視の2件以上のTargetが一致した。この矛盾は部分Datasetでも証明できる。
- `unresolved`: RelationKey未定義、比較不能、キー欠落、Targetが部分、0件に対する必須性未定義など。空欄を否定事実へ変換しない。

キー照合は順序付きの複合RelationKeyを保持し、JSON numberは浮動小数点へ丸めずに比較する。巨大な指数等では資源制約のため比較不能とする。異なるJSON型を自動変換せず、型の意味が未確定なら`unresolved`で残す。

### M1検証の段階

`relation_assessment_test.go`と`world_adapter_test.go`に、複合キー・部分抜粋・2件一致矛盾・キー欠落・Keysなし・大きいID・表示順差・World内の受注→顧客を含むテスト候補を追加した。

実際のGitHub PRのcheckoutはこの環境では取得できず、**当該候補ブランチのGoテスト・format・full Repository-healthは未実行**。独立したGo 1.23.2 scratch probe（関係評価の抜粋・型スタブ）では三値評価の主要7ケースがPASSし、`go vet`も完了したが、PRのソース全体・Adapter・CLI・リポジトリの検証Evidenceを代替しない。

未確定: Datasetの網羅性宣言をどのEvidenceで裏付けるか、Source Indexによる業務キーの権威追跡、実Provider観測と説明用スナップショットの区別、複雑なデータ型の比較、完全ER図記法との互換。

## 検証結果から設計者への問いへ — Review Packet（2026-10-10）

三値の関連評価で重要なのは、`unresolved`を放置しないこと。ただしAIがもっともらしい値を補完して`supported`へ変換してはならない。そこで独立した派生資産`ReviewPacket`を作り、同じCaseのStage・Relation・Source Record・Frame IDs・モデルrevisionを固定して、必要な確認を生成する。

`Review(input)`は新しいProjectionを内部で検証してから作る。未検証の外部ProjectionやAIの診断を既存の事実として受け取らない。元のCaseにある`authoredQuestions`と決定論的に生成した`issues`は分離し、提示内容を何も自動受理しない。

| 結果の理由 | 必要になる確認 | まず確認する資産 |
| --- | --- | --- |
| RelationKey未定義 | 対応属性を定義する必要があるか。関係そのものが別の意味か | Semantic Relation・Business identity |
| Sourceキー不足 | 例の記載不足か、業務上の任意値か | Example Dataset・必須性の根拠 |
| Targetキー不足 | 参照先候補のどの値で照合できるか | Target Dataset・Attribute |
| Target表が部分的 | 何をもって全件と確認したか | データ抽出範囲・完全性Evidence |
| Targetが0件 | その業務で参照先の存在は必須か | Business rule・例の目的 |
| `one`で2件以上が一致 | キー定義の誤りか、実例の重複か、多重度の設計不備か | Relation・具体例・identity evidence |

レビュー候補のIDはCase・モデルrevision・Stage・Relation・Source Record・原因から生成する。同じStageを複数フレームで見せても同一論点とし、Frame IDsだけ増やす。表示順が変わってもIDは変えないが、モデルが変われば古いIDを使い回さない。

導出物に含まれる`ReviewTarget`は担当すべき**検討領域**の手掛かりであり、書き込み操作ではない。例えば「RelationKey未定義」から自動で`AddRelationKey`を作るのは誤りで、キーを持たない関連が意図的である場合もある。真の業務上の問いは、一次資料、業務担当者、設計者の確認を経て、既存のShared Acceptanceで受け入れる差分に初めて変換できる。

CLI出力先へ`projection.json`とSVGに加えて`review.json`を生成する。`review.json`は承認済みのDecision、Semantic Patch、Business Ruleを含まない。対象StageやRelationの変更、根拠の追加後は、再生成して差異を確認する。

### 継続的な品質境界

数値は`float64`へ丸めずJSON numberを正確な有理数として検証する。例えば`9007199254740993.5`を巨大な整数へ丸めて誤って許可しない。

`many`の関連で一致先が0件しかないときも、具体的な対応関係を確認できたという`supported`にしない。最小多重度の意味は自動決定せず、`unresolved`とする。

**状態**: 該当コード・テスト・文書はPR #1128の作業ブランチにコミットした。対象PRのGoテスト、format、Repository-health Gateは依然として未実施。レビュー質問の生成結果を承認済みの業務仕様と混同しない。

## 回答Handoffへの縦断拡張（2026-10-10）

Reviewから出た質問を、実際のモデル・具体例・適用文脈に結び付けるため、次を候補実装とした。

- `ReviewPacket`へ正規化JSON由来の`exampleFingerprint`、`catalogFingerprint`、`fingerprint`を追加。既存のモデルrevisionだけでは、同じ診断を生んだ別の実例を区別できないためである。表示順だけを変更した場合は同じレビューとして扱う。
- `AnswerInput`は`schema`、`caseId`、`modelRevision`、`reviewFingerprint`と質問ごとの`question.kind/id`を必須にする。無関係・旧版・重複した回答は拒否する。
- 各回答はテキストだけでなく、`disposition`、適用範囲を表す`context`、自己申告の`claimedBy`、文書や観測結果への参照である`evidence`を持つ。**回答と根拠の真正性はまだ確認していない**。
- `BindAnswers`は入力CaseからReviewを再生成して厳密に照合し、`AnswerHandoff`を返す。出力は常に`status=unverified`であり、Confirmation、Decision、Semantic Patchへ昇格させない。
- Relation由来の質問に限り、`confirmationRoute`として既存`confirm-record`での対象identityと`semantic-relation`のBasis subjectを候補提示する。自由記述の質問にはCanonical subjectを無根拠に作らず、route無しとする。
- CLIは`-answers FILE -out DIR`で`answer-handoff.json`を出力できる。不正・古い回答は生成物を書き始める前に拒否する。

### 既存のConfirmationと共有Acceptanceを再利用する

既存のShared Acceptanceには、`confirm-record`/`confirm-check`、変更候補への`preview`/`review`/`accept`が存在する。新しい回答の保存台帳や承認エンジンを`exampleslide`に重ねて実装してはならない。

**回答案→確認候補→根拠と権限の確認→既存Confirmationでの記録→必要な場合だけ新しい変更candidate**という順を維持する。

- `AnswerHandoff`は必ずしも人間の回答を表さない。`claimedBy.kind=human`は自己申告である。既存Confirmationでもactorの実証は行わないため、両者の自己申告を合成して認証済みの判断へ格上げしない。
- Source IndexのBasisはWorkspaceから取得し直す。Handoffに記載された任意のevidence pointerをそのまま権威あるEvidenceとして扱わない。
- `ReviewPacket.ModelRevision`はSemantic Worldのfingerprintであり、Shared Acceptanceが要求するWorkspace全体の`baseModelRevision`とは異なる。後者は変更候補作成時に再取得する。
- RelationKeyの有無や表示条件が変われば、回答・質問とモデルの関係を再評価する。Reviewの固定化は、記録の同一性を示すだけで業務的な採否を決めない。

### M1で残る実証

1. 対象PR exact-headのGoテスト、gofmt、Go vet、必須Repository Health Check。
2. Source Indexから実RelationへBasis subjectが正しく解決できるかの、実Workspaceを使った読み取り専用縦断検査。
3. `completeTables`が現在はCase全Stageに一括適用されることを踏まえた、Stageごとの完全性・時点・データ出典の表現。
4. 確認後の設計変更について、既存Shared Acceptanceの対応するtyped operationだけを使用する。未実装の操作を無条件に生成しない。

今回もProduction、UI、意味モデルの正本は変更していない。

## Confirmationの再確認漏れと設計変更Triage（2026-10-10）

既存ConfirmationはCanonincal Source Indexの`Basis`ファイルのfingerprintによって「前提が変わったか」を判定する。これはそのファイルの意味変更には適するが、データモデルを変えずに**説明用Case内のレコード値だけが変化した**場合は、前提の差異を観測できない。

以前の`confirmationRoute.identity.subject=issue.ID`のままでは、同じRelationと同じ原因を持つ別の具体例に対して、古い確認済み回答を再利用し得る。今回、識別子を`CaseID/ModelRevision/ExampleFingerprint/CatalogFingerprint/IssueID/QuestionText`の安定ハッシュに変更する。RelationのCanonical Source Index Basisはそのまま維持する。例のレコードが変われば再確認し、スライドの表示順だけを変えても不用意に再確認を要求しない。

### 回答後の振り分け

`TriageAnsweredReview`は`BindAnswers`を再実行して現在の質問との一致を確認した後、回答の意図を次の設計操作候補へ振り分ける。

| 回答の意図 | 意味モデル変更を許すか | 次に必要な確認 |
| --- | --- | --- |
| `clarify-example` | いいえ | 説明用Caseの修正と再検証 |
| `retain-model` | いいえ | 根拠の真正性、適用文脈、必要なら既存Confirmation |
| `investigate` | いいえ | 追加根拠や実務担当者からの確認 |
| `propose-model-change` | **回答だけでは不可** | 誤りの主体特定、Source Index、current Workspace revision、実装済みtyped operationの資格確認、Shared Acceptance Preview |

Current Shared Acceptanceのtyped operationには`add-relation`が存在する一方、**既存RelationのKeyを更新する操作や、RelationKeyだけを追加する操作はない**。このため、既存Relationのキー不足を理由に、新しいIDのRelationを追加して差し替える案を勝手に作らない。ここは`existing-relation-key-edit-not-supported-by-current-typed-operations`と記録し、原モデル上の意図確認と新しいtyped editの設計判断を先に求める。

また、`one`方向の重複参照が発見された場合、実例のレコードの誤りなのか、RelationKeyの誤りなのか、多重度の業務定義の誤りなのかを確認する。すべての要素がまだ未確定なので`model-versus-example-cause-not-determined`とし、既存Ruleや関係へ即時編集しない。

CLIは`-answers`時に`change-triage.json`を出力するが、全件で`candidateReady=false`とする。既存Shared Acceptanceの`preview`、`review`、`accept`に直接渡せるCandidate、必要な権威ある根拠、本人による承認を生成したとは主張しない。

### ローカル動作検証の制約

Container環境のGoは1.23.2だが、github.comのDNS解決ができず、このブランチのclean checkoutは取得できなかった。したがってPRの正確なソース一式を用いた`go test`、`gofmt`、`go vet`、`Repository-health`は未実施である。欠陥があれば修正し、exact-headのGateを通す前に統合してはならない。

JSON入力で`completeTables`を綴り間違えた場合にも、検査対象の完全性を黙って変更しないよう、Case入力に未知フィールドを拒否する読み込みを追加した。

### 後続で必要な契約

- **Stage単位の網羅性と観測時点**: 現状`completeTables`は全Stageで共有されるため、時点別の観測範囲を厳密に証明できない。
- **Source Indexの根拠付け**: Caseが示すEvidencePointerの文書・観測結果・Source Indexが実在し、現在の仕様へ適用できることの独立検証。
- **人間の権限検証**: 申告の`claimedBy.kind=human`を本当のHuman authorizationへ昇格させない。
- **既存Relationを編集するtyped operationの必要性**: ユースケースを確定した後に、追加・修正・置換の意味的境界と差分検証を検討する。未検証の汎用文字列Patchを代用品にしない。

## 型同一性の否定試験（2026-10-10）

`relation_assessment.go`に、RelationKeyの**値の一致と型の同一性を別々に扱う条件**を追加した。たとえばCustomer IDとSupplier IDがどちらも文字列`"001"`なら、単にJSONの値が一致しただけでは正しい対応付けを証明できない。

既存の正規化済みWorldには各Attributeの`typeRef`があるため、Adapterの表示用Catalogに`semanticType`として欠落なく保持する。両端が同じCanonical typeRefであると確認できる場合だけ、既存のキー値比較へ進む。片端が未定義か、Canonical typeRefが異なる場合は`relation-key-type-not-confirmed`としてレビューを起こす。合成CatalogにtypeRefが無いときは同じprimitive表示型同士であることを最低条件とする。

ここでは**Aliasの変換可能性**はまだ判断しない。Canonical typeRefの一致は業務上のドメイン制約の充足や、識別子・コード値の実データの正当性まで証明するものではない。

否定試験をPRに追加した。さらに実装のキー型比較ロジックを**型スタブへ切り出したGo 1.23.2の独立環境**で、意味型一致、異なる意味型、片側のみ型情報あり、基本型一致、基本型不一致の5ケースについて`go test -v ./...`がPASSし、`go vet ./...`もPASSした。

ただし、この独立スモークテストは、PR #1128の実コードをチェックアウトして実行したものではない。実World連携・既存テスト全体・gofmt・Repository-healthの合格Evidenceとは区別する。

### 統合までの実行手順

trusted clean checkoutの**exact PR head**とcurrent mainを取得して固定したうえで次を順に実行し、clean pre/postと出力を保存する。Go sourceを手動修正した場合はheadが変わるのでGateを最初から取り直す。

```sh
git status --porcelain
gofmt -l .
go vet ./tools/reference-example-slide/...
go test -count=1 ./tools/reference-example-slide/...
tools/repository-health-check.sh
git status --porcelain
```

PRのDraft解除やmain統合は、既存リポジトリ規約により**full Repository-health PASSとexact head/base/current mainの一致**を確認してから判断する。GitHub Actionsを起動しない。
