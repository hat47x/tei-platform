# 独立した処理実行プラグイン（参照実装）

このサブプロジェクトは、TEIのプラグインを**コア内部のGoパッケージに依存せずに構築できること**を確認するための最小の参照実装です。製品用の配備機構ではありません。

## 提供する機能

旧実行環境の「処理実行」呼出し方式に合わせて、要求を標準入力から受け取り、受領結果を標準出力へ返します。ビジネス上の処理そのものやデータベース操作は行いません。

必要な要求・応答の項目だけを内部型として使用しています。**ここで旧通信仕様を再定義したり、新しい正式なプラグイン契約を確定したりはしません。** 現行仕様への互換性は[実プロセス接続試験](../../tools/reference-composition-process-bridge/README.md)によって検査します。新しい契約中心の設計へ移行する際は、[ADR-0032](../../plan/adr/0032-composition-kernel-replacement.md)に従って書き換えます。

## 構築と試験

リポジトリのルートから次を実行します。

```sh
go test -count=1 ./plugins/reference-process-executor
go build -o /tmp/tei-reference-process-executor ./plugins/reference-process-executor
bash tools/check-reference-plugin-isolation.sh
go test -count=1 ./tools/reference-composition-process-bridge
```

最後から二番目の手順は、ソースを一時領域へ複製し、**独立したGoモジュール**をその場で用意してオフラインで構築・試験します。現在の単一Goモジュールの構成は維持します。生成した`go.mod`はこのサブプロジェクトには保存しません。

## 依存関係と限界

- 製品コードと単体試験の依存先はGo標準ライブラリだけです。TEI内部の型やプラグイン管理機構を参照しません。
- プラグインの論理IDは`tei.process.reference`、機能IDは`process.reference`です。ソースディレクトリ名、成果物名、プラグインIDは同一視しません。
- 通信方式の互換性と、署名検証、信頼できる権限付与、サンドボックス、実配備、処理完了の保証は別の課題です。
- 参照プラグイン自身にホストの権限判定や成果物の検証を任せません。ホストが必要な検証と許可を済ませてから実行します。

プラグインのサブプロジェクト化の方針は[ADR-0033](../../plan/adr/0033-core-and-plugin-subprojects.md)を参照してください。
