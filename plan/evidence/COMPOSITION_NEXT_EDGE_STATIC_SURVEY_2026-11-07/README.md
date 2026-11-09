# 構成Gateの「次の一辺」を、依存グラフの形で測る（2026-11-07）

- 対象: [構成Kernelの移行判定](../../gate/COMPOSITION_KERNEL_MIGRATION_GATE_2026-10.md)の4.1末尾「製品としての次の一辺」
- 位置づけ: **静的なimport関係の観測**。実行時の接続・意味の受理・権限の証明ではない。
- 計測した版: `origin/main` `db62a36a` の作業ツリー（このワークツリーの追加は`plan/evidence/`配下のGo以外のファイルだけで、Goのimport関係に影響しません）。環境は`environment.txt`。

## 1. 何を測ったか

4.1の実行記録（[COMPOSITION_FIRST_CONSUMER_RUN_2026-11-07](../COMPOSITION_FIRST_CONSUMER_RUN_2026-11-07/README.md)）で、A・B・Cはそれぞれ成功しました。残る問いは「AのBusiness Action／Operationの識別子と、BがHost所有の現在Route・Service Acceptanceから実Providerへ渡すContractを、**同じRequest identityとして結ぶCallerが実在するか**」です。

これを実行で試す前に、まず**依存の形**で測りました。

- `scan_imports.py` がリポジトリ全体の`*.go`を読み、`tei/internal/*`のimport文字列を集計しました（結果`import-graph.json`、要約`import-summary.txt`）。
- `go list -json -deps ./...` で各パッケージの推移的閉包を取り、実行単位（`main`パッケージ）ごとに主要パッケージが入るかを表にしました（要約`package-summary.txt`）。

どちらも**import宣言の形**を見るだけで、実際にそのコードが呼ばれたかは示しません。

## 2. 観測

### 2.1 ファイル単位

| 項目 | 件数 |
| --- | --- |
| 走査した`*.go` | 2,876 |
| `tei/internal/composition`をimportするファイル | 49 |
| `tei/internal/operations`をimportするファイル | 47 |
| その**両方**をimportするファイル | **0** |
| `tei/internal/servicehost`をimportするファイル | 3（すべて`_test.go`） |
| `tei/internal/plugin/runtime`をimportするファイル | 114 |
| `composition`と`plugin/runtime`の両方をimportするファイル | 17（うち非試験は1） |

「両方をimportするファイル0」は、Aの操作面とBの構成選択を**同じGoファイルで突き合わせている場所が無い**という意味です。合成の場（Composition）と操作面（Operation Surface）は、別々のツリーにいます。

### 2.2 実行単位（`main`パッケージ）単位

`go list -deps`の推移閉包で見ると、リポジトリには410パッケージ・167の実行単位があります。

| 実行単位 | `composition` | `operations` | `plugin/runtime` |
| --- | --- | --- | --- |
| `tei/cmd/tei` | false | true | true |
| `tei/tools/reference-first-web-target` | true | false | false |
| `tei/tools/reference-plugin-capability-contract` | true | false | true |
| `tei/tools/reference-project-advice-lit` | true | false | false |
| `tei/tools/reference-studio-static-composition` | true | false | false |

- `composition`を含む実行単位は**4件**、`operations`を含む実行単位は**3件**。**その両方を含む実行単位は0件**。
- `cmd/tei`（AのCLI）は`composition`を含みません。`reference-first-web-target`（Bの起動側）は`operations`を含みません。
- **`internal/servicehost`を閉包に含むパッケージは0件**です。`SelectedServiceCarrierInvoker`は、非試験のコードからどこも呼ばれていません（importしているのは試験3ファイルだけ）。

### 2.3 合成の入口がどこで使われているか

- `composition.Registry`を作る非試験のコードは`tools/reference-project-advice-lit`・`tools/reference-external-adoption-authority`・`tools/reference-studio-static-composition`の3箇所です。このうち**`Registry.Select`まで呼ぶのは`reference-project-advice-lit`だけ**で、ほか2つは登録と`Apply`までです（`internal/composition`自身の実装を除く）。`cmd/tei`にも`tools/reference-first-web-target`の起動側にもありません。
- B経路の保護ルート（`withProtectedSessionCommand`）は、**非試験では定義だけがあり、呼出しは試験ファイルだけ**です。実Serverの組み立て（`main.go`の`runCombinedWebTargetWithHost`）は、この関数を呼んでいません。

## 3. この観測が示すこと

- **AのOperation SurfaceとBのComposition/Host Routeは、現行のコードで同じファイル・同じ実行単位に同居していません。** 「同一の意味／Operation／Request identityとして結ぶ既存Caller」を探す対象は、今のところ**存在しない**と静的に言えます。
- 4.1のC（`internal/driver`のD2試験）が通っても、それは`internal/servicehost`の`SelectedServiceCarrierInvoker`が実Hostから呼ばれることを意味しません。**Service Carrierは現在、非試験の呼出し元を持ちません。**
- B経路の合成（選択ProviderとActionの結線）は、実Serverではまだ組み立てられていません。Bの成功は、試験が明示的に構成したHostの範囲です。

## 4. この観測が示さないこと

- **実行時の不接続**。importが無いことは、HTTP越し・別プロセス越し・設定ファイル越しの接続を否定しません。逆に、importがあることも実行を意味しません。
- **意味・権限・DB効果**。これは依存の形の話で、Business Actionの受理、Hostの現在権限、実DBのCommitを見ていません。
- **将来の形**。`cmd/tei`に`composition`を足す、`servicehost`を実Hostへ配線する、といった変更で、この結論は容易に変わります。**この観測は「今の形」に対するもの**です。
- **build tag付きの実行単位**。`go list`は現在の環境（Windows/amd64、既定のタグ）で評価しました。`tei_repair_only`等のタグを付けた構成は別に評価していません。

## 5. 次に確かめること（案・未実施）

1. `cmd/tei`の操作面が出す`ledger.update.sales.order`と、Bが扱う`order_entry.preview_order_quote`を、**同じOperation identity**として結ぶには、どのOwnerのどのContractが必要かを列挙する（新しいResolverや第二のLockを作らない範囲で）。
2. `internal/servicehost`に非試験の呼出し元を1つ作るなら、どのHost（`reference-first-web-target`か、別のConsumerか）が適切かを決める。**これは配線の追加であり、実行権限の発行ではない。**
3. B経路の保護ルートを実Serverへ組み立てる条件（診断起動か、通常起動か）を、既存の`-diagnostic`の扱いと整合させる。

## 6. 生データとSHA-256

`go list -json -deps ./...`の生出力（約2.5MB）と、全ファイル分の走査一覧（約1.2MB）は、`environment.txt`の手順で再生成できる中間物のためリポジトリには置いていません。**この記録の数値は1回の実行のもので、独立に2回走らせて一致を確かめてはいません**（スクリプト自体は決定的です）。下表のファイルが生データです。

| ファイル | 内容 | SHA-256 | バイト数 |
| --- | --- | --- | --- |
| `environment.txt` | HEAD・Toolchain・実行前のclean状態・再現手順 | `c843ade9e0d16bca1ec26b33df2f27765070b4c8a0fa5a2b98e0b76f0c755ad5` | 864 |
| `scan_imports.py` | ファイル単位の走査器 | `9c6b92a10dd15646a093abdfe21f4e15d6e90b9394db5c6593c121c3d4f75c7a` | 2051 |
| `summarize_imports.py` | 走査結果の集計器 | `4aaf521799bbafca15bdb10d4dc202133c8dc51f7183ec793efa7190022daaa7` | 2066 |
| `summarize_packages.py` | パッケージ閉包の集計器 | `eae987e2a19dde5de772a6b325cbbeb15ff6d7ea93a71fc44c5b2300c4898f8d` | 2787 |
| `import-graph.json` | 対象の`tei/internal/*`をimportする318ファイルの一覧 | `497f6604dc0fe033194e0f28e3dbcd355aa2b71c06a2d34f29bf4ce26157e341` | 112333 |
| `import-summary.txt` | 2.1の件数と該当ファイル | `523f228ca7a1714fd1aab5922270faac7fb3fbf385d5d5cdc9c89939f5bb82ab` | 2003 |
| `package-summary.txt` | 2.2の実行単位ごとの該当一覧 | `6b4f8204d8c823bf2ad3a16631795a2e106a1dae1e38709ed605a099257c0bc9` | 1362 |

`package-summary.txt`は`go list -json -deps ./...`を一時ファイルに出し、`summarize_packages.py`に通して作りました。`summarize_packages.py`と正本の`package-summary.txt`をコミットしたため、**この節の表は同じHEADから再生成できます**。`import-summary.txt`も同様に`scan_imports.py`と`summarize_imports.py`から作れます。

`go list -json -deps ./...`の生出力（約2.5MB）は再生成可能な中間物のためリポジトリには置いていません。**この記録の数値は1回の実行のもので、独立に2回走らせて一致を確かめてはいません**（スクリプト自体は決定的です）。`main`が進むと、ファイル数・パッケージ数・閉包フラグは変わり得ます。

