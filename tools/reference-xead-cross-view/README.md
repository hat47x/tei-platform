# Xeadの三視点の対応関係を調べる

この参照実装は、Xeadの設計資料に記述された業務、機能、データの対応関係を読み取り専用で調べます。課題 [#1031](https://github.com/hat47x/tei-platform/issues/1031) に対応します。

利用者が指定したXML形式の設計資料と、読み取り方を定める契約ファイルから、資料に明記された関係と複数の記述をたどって得られた関係を区別します。

通常の出力は件数などの集計結果に限ります。設計資料の本文、旧識別子の値、個別の関係をたどるための識別情報は公開しません。

```bash
go run ./tools/reference-xead-cross-view \
  -input /path/to/local-model.xead \
  -contract reference/xead-source-transcription-contract.synthetic.json
```

上記の契約ファイルは**合成試験用の不完全な例**です。外部の参照モデル全体について、記述要素の網羅性や意味の正しさを保証しません。実際の資料を照合する際は、対象の形式・版・要素に合う契約を別途確認してください。参照モデルの原本や詳細な解析結果は公開リポジトリへ保存しません。

個別の対応関係を手元で調べる場合に限り、次の指定を追加します。この出力は非公開の調査資料として扱ってください。

```bash
go run ./tools/reference-xead-cross-view \
  -input /path/to/local-model.xead \
  -contract reference/xead-source-transcription-contract.synthetic.json \
  -local-projection
```

## 結果の読み方

- 表の利用宣言に有効な作成・参照・更新・削除操作が一つもない場合、操作関係を作らず、`iotable-operation-missing`として未解決に残します。表の記述自体は集計対象です。
- 未知の操作指定があれば、理解できた操作だけを取り出すことはせず、`iotable-operation-marker-unsupported`として未解決に残します。
- 機能から表への操作宣言は、その機能全体での利用候補です。特定の業務作業による更新、実行時の副作用、取引の確定、認可を証明するものではありません。

集計結果は設計資料を読み取った証拠であり、TEIの正本への受理や、元のX-TEA Driverで実行した結果を意味しません。実機での動作確認は別課題 [#940](https://github.com/hat47x/tei-platform/issues/940) が担当します。
