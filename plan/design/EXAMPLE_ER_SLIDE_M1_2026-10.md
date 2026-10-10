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
