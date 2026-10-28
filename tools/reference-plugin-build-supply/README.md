# reference-plugin-build-supply

技術検証用サンプル[`plugin-build-supply`](../../reference/plugin-build-supply/README.md)(サンプル7)の工程1〜4を、一つの通しの記録として実行する実装です。設計の正本は[技術検証用サンプルの統合設計 4.7節](../../plan/design/TECH_VALIDATION_SAMPLES_2026-10.md#47-プラグインの構築意図から自己構築成果物の固定までplugin-build-supply)です。

この文書は作業中です。調査の結論(工程ごとの既存資産を呼べるか)を先に記録します。

## 既存資産を呼べるか(調査の結論)

工程1・2の参照実装は、いずれも`package main`で、関数を外から呼べません。CLIが利用者の入力を受け取るかどうかで、呼べるかが決まります。

| 工程 | 既存資産 | 呼べるか | 理由と扱い |
| --- | --- | --- | --- |
| 1 | `tools/reference-build-authoring-intent` | CLIで呼べる | `-intent`、`-catalog`、`-source-root`で利用者の宣言とカタログを受け取る。ただしソースの照合値は資産内の固定入力用のディレクトリ照合(`DirSourceResolver`)で計算し、自己構築の実行器のGoのソース閉包の照合値(`tools/internal/gobuilder`)とは別の値になる。資産のREADMEも、言語別のソース閉包(#1096)は接続していないと明記している |
| 1 | `tools/reference-builder-contract-resolution` | 固定入力でだけ動く | CLIに入力の引数がなく、資産内の固定の提供元で解決・固定・再生を示すだけ。利用者のカタログを渡せない。信頼の失効は扱っていない |
| 2 | `tools/reference-build-attempt-evidence` | 固定入力でだけ動く | CLIは模擬の失敗一件を出して終了状態1で終わる(意図した動作)。試行の記録と発生識別子の型は非公開 |
| 2 | `tools/reference-build-artifact-store` | 固定入力でだけ動く | CLIは一時ディレクトリへ固定のバイト列を出し入れするだけ。`PutBytes`と`GetBytes`は非公開。保管の配置(`sha256/<64桁の16進>`)は公開の説明があり、同じ配置を使える |
| 3 | `tools/reference-plugin-self-build` | CLIで呼べる | `-repo`、`-go`を受け取り、検証根拠のJSONを標準出力へ出す。計画は実行器が自分で組み立て、外から渡せない。構築した成果物は一時ディレクトリごと削除され、バイト列は外へ出ない |
| 3 | `tools/reference-native-plugin-builder`、`reference-plugin-package-builder`、`reference-transform-runtime-engine`、`reference-transform-seed-builder` | 自己構築の実行器が内部で構築・実行する | このサンプルからは直接呼ばない。実行器の検証根拠に、各構築器の二回構築と段階の収束の結果が残る |
| 4 | `tools/reference-process-plugin` | 構築済みの実行体として呼べる | 製品の`StdioProcessLauncher`(処理プラグインの標準入出力の通信、プロトコル版1)で起動できる |
| 共通 | `tools/internal/gobuilder` | ライブラリとして呼べる | Goのソース閉包、ツールチェーンの照合値、隔離した構築を、自己構築の実行器と同じ関数で計算できる |
