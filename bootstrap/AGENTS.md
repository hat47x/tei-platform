# bootstrap/ — 初期構成・再構成の手順と実体

このルールは`bootstrap/`以下に適用する。ルート`../AGENTS.md`を先に読む。

- `final-source/`、`source-safe/`、`payload/`、`materialize/`、`publish/`等は、何を入力し、どの成果物を作り、どこへ公開するかが異なる段階として扱う。段階をまたぐ操作の責務を、名前の似たスクリプトという理由だけで混ぜない。
- 変更前に入力の正本、生成先、既存ファイルの上書き可能性、再実行時の副作用、失敗時の中断・復旧手順を確認する。特に公開・書込み処理を「検査しただけ」と表現しない。
- bootstrap artifact・履歴・一時的な生成結果を`requirements/`や`reference/workspace/`の代替正本にしない。
- 実行した環境・revision・検証範囲と、未実行の手順を区別する。codeを変えたときはルート`AGENTS.md`のtrusted full Repository Health Gateを適用する。