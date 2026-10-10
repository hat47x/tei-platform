# 具体例からER図スライドへつなぐ参照実装（M1）

状態: 実装候補。Goテスト・統合Gate・実TEI World連携は未確認。

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
- 参照先の無い明細は拒否する。隠した外部キーの関係は消さない。
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
- Semantic RelationKeyによる**具体レコードの対応・参照整合性検証**は未実装である。図面に線を引けることと検証済みは別である。
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
