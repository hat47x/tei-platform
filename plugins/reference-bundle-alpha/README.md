# 参照プラグイン Alpha

一つの実行ファイルに複数の論理プラグインを結合する試験のための、独立したソース単位です。論理IDは`tei.reference.bundle.alpha`、機能IDは`process.bundle.alpha`です。

Go標準ライブラリだけに依存し、受領応答を組み立てる関数を公開します。実際の業務処理や権限判定は行いません。[構築と接続の試験](../../tools/reference-plugin-bundle/README.md)を参照してください。
