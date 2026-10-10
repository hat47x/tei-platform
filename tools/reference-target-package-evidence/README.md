# 第一Web TargetのGoパッケージ閉包を観測する

このツールは、既存の[配備単位の設計](../../plan/design/RUNTIME_TARGET_BINARY_COMPOSITION_DESIGN.md)における **E0（実際のGoソース依存の観測）** を補助する読み取り専用Referenceです。[Issue #259](https://github.com/hat47x/tei-platform/issues/259)のTarget／実行境界検証に属します。

Plugin Lock、Build Plan、Provider、Capabilityの選択は**行いません**。`go list -deps -json`の結果から、現在のGoメインモジュールに含まれるコンパイル対象パッケージのImportPath、Go/Cgoファイル、直接import先を整列したJSONへ変換するだけです。標準ライブラリと外部モジュールのpackage情報は出力から省きますが、import辺には保持します。Cgoの`import "C"`は`go list`が独立packageとして出力しない疑似importなので、Cgo対象ファイルがある場合に限り例外として許容し、その辺を出力に残します。

## 実行手順

依存関係を取得できるGo環境の、**変更のないTEIリポジトリのルート**で実行します。結果はチェックアウト外に保存してください。

```sh
go test -count=1 ./tools/reference-target-package-evidence

go run ./tools/reference-target-package-evidence \
  -entry ./tools/reference-first-web-target \
  > /tmp/tei-first-web-go-packages.json
```

- 開始・終了時に`git status --porcelain --untracked-files=all`とHEADを確認し、変更があれば失敗します。さらに、`go list`がコンパイル対象として挙げたメインモジュールのGo/CgoソースがGitの追跡対象であることを検査します。通常の`git status`では見えない**ignore対象の未追跡Goファイル**も、コンパイルに使われるなら失敗させます。またGitで追跡されていても、シンボリックリンク経由で別のソースを読み込む場合は拒否します。ただしこの検査は全入力の改ざん防止や`tools/repository-health-check.sh`のexact-head full PASS／統合資格を保証しません。
- `GOENV=off`、`GOWORK=off`、`GOFLAGS=`を`go list`に適用し、隠れたユーザー設定・Workspace・build flagによる入力変化を避けます。必要なbuild tagsは`-tags`で明示し、GOOS／GOARCH／CGO_ENABLED／Go版を結果に記録します。
- Goのツールチェーンが依存を解決できない、JSONが不正、参照先パッケージが欠落している等の場合には不完全な結果を成功として出力しません。

## この証拠で言えること／言えないこと

**観測対象**は一つの静的なGoビルド入口です。第一Web Targetの`-application`はサーバーの**実行時引数**なので、それを二通りにしても別のGo package closureにはなりません。アプリごとに異なる実装を物理的に除外したければ、異なるcompile-time入口を作り、同じ条件でそれぞれを測る必要があります。

このJSONだけでは、Go linkerの不要コード除去、最終成果物のサイズ・digest・symbol、実動作、Secret／認可、Web UIのJSやHTMLの配布物は証明できません。また`go list`が収集する静的import先は、GoのReflection・Pluginの動的参照や外部プロセスを完全には表しません。

次段階のE1では、同じ受理済みApplication meaning（Handover B/R）に対する**別の配備入口**で同じ観測を行い、さらに`go build`成果物のbytes・digestと必要に応じsymbol／実行結果を照合します。既存のGate C第一Web Targetを置き換えず、Build Artifactの正本は[#943](https://github.com/hat47x/tei-platform/issues/943)に委ねます。

## 検証段階

本追加のGoソース3件は、隔離したGo 1.23.2モジュールで**同じGit blob SHA**を照合した上で、局所`go test`・`go vet`・`gofmt`および小さなGo entryに対する`go list`の動作を確認しました。これはTEIリポジトリ全体や実第一Web Targetに対する実行確認ではありません。Trusted clean full checkoutでの`tools/repository-health-check.sh`は未取得です。Draft PRのまま扱います。
