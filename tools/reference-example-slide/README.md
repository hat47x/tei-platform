# Example-driven ER slide reference

状態: **独立したGo参照実装候補（未統合・テスト未実行）**。

発注10個・一部入荷6個・検収合格4個／不合格2個の合成データを、手動配置のER図上で三つのスライドとして表示するための最小実験です。データ構造だけでは分からない業務上の疑問（不合格数量を履行残に含めるか）を未解決のまま保持します。

## 実行

リポジトリルートから:

~~~sh
# 参照パッケージとCLIを対象に試験
go test ./tools/reference-example-slide/...

# 合成Caseを指定して三枚のSVGと投影JSONを出力
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
