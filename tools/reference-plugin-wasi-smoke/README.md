# 独立プラグインのLinux・WASI実行結果の比較

[Issue #1251](https://github.com/hat47x/tei-platform/issues/1251)の実行先別構築の後続試験です。既存の[独立参照プラグイン](../../plugins/reference-process-executor/README.md)を同一ソースからLinux/amd64とWASI（wasip1/wasm）向けに構築し、**両方を実際に実行**して結果を比較します。

## 実行

Linux/amd64上のGo 1.23.2とNode.js（組み込みWASI preview1対応）がある環境で、リポジトリのルートから次を実行します。

```sh
bash tools/check-reference-plugin-wasi-smoke.sh
```

試験は参照ソースを一時Goモジュールに複製し、外部モジュールの取得を禁止して構築します。Linux版は直接起動し、WASI版は[Node.jsによる最小実行ホスト](wasi-runner.mjs)から起動します。両者へ同じJSON要求と同じ引数を渡して、標準出力の応答をバイト列で比較します。

正常な受領に加えて、通信方式の版・操作・要求内容・起動引数の不一致を拒否する場合を検証します。壊れたJSONを受けた場合は両実行環境で失敗終了し、応答や受信した秘密文字列を出力しないことを確認します。試験はCIの[コア隔離・外部利用者検証](../../.github/workflows/tei-core-isolation-ci.yml)に組み込んでいます。

## 立証範囲と限界

この検証は、**同じ参照ソースが異なる二つの物理実行方式で同じ処理結果・拒否結果を返すこと**に限ります。WASIにおける実際のプロセス呼出しは確認しますが、Webブラウザの動作やTEIの製品Wasmホストへの接続ではありません。

使う通信形式は既存のプロセスプラグイン試験用のもので、新しい正式なサービス契約の正本ではありません。また、Node.jsの実行ホストは試験専用であり、認可、WASI capability制限の十分性、署名、成果物の来歴、サンドボックスによる実効的隔離、ホスト再起動時の引継ぎを証明しません。結果の受領（`accepted`）は外部業務処理の永続的な完了を意味しません。

Linux/arm64向け成果物の形式は既存の[クロス構築試験](../check-reference-plugin-isolation.sh)で検証しますが、本試験ではARM64の実CPU上での呼び出しは行いません。ターゲットごとの形式照合と実行時の同等性を区別します。正式な契約とHost権限は[Issue #951](https://github.com/hat47x/tei-platform/issues/951)などの管理対象です。
