# 案件固有の顧客向け補助チェック — Go参照実装

#951 / #997 / #1016 の生成AI主導・案件別実装を検証するための最小実装です。

`Evaluate` は顧客コード `VIP-` で始まり、メモが空の場合にレビュー提案を返します。これは**架空の案件固有ルール**であり、実際の受注条件やTEIのAccepted Business Validationではありません。外部I/O・DBアクセス・認可・注文更新を行いません。

Go標準ライブラリのみで動作し、次のコマンドで単体試験できます。

```sh
go test ./plugins/reference-project-customer-advice
```

UIとHTTP Hostは [参照実装](../../tools/reference-project-advice-lit/README.md)を参照してください。これは製品Runtimeで登録された正式Service Providerではありません。正式な受注可否、Runtime Authority、Conformance、第三者向け配布品質は保証しません。

## 独立した実行ファイルの検証（#976）

`cmd/reference-advice-process/` は、同じ案件Go補助処理から**単独のOS実行ファイル**を構築する参照用入口です。現在のGoテスト実行ファイルを子プロセスとして流用する方式とは別に、配布候補となる独立成果物のbytesをSHA-256で検査します。

```sh
go build -trimpath -o /tmp/reference-advice-process ./plugins/reference-project-customer-advice/cmd/reference-advice-process
go test -count=1 ./tools/reference-runtime-capability-identity ./plugins/reference-project-customer-advice
```

テストでは、既存のGeneric Runtime Capability Refから導出される**合成の固定dispatch ID**だけを受け付け、想定外の操作・入力を拒否します。既存の`StdioProcessLauncher`／`HostSandboxLauncher`を通して別プロセスを呼びます。変更された実行ファイルは、実行前のDigest照合で拒否します。Linuxでの参照試験であり、ほかのTargetへ自動的に適用しません。

**制限**：固定dispatch IDの元となるContract `1.0.0`、Provider、Resolution Lock、Environment Grantは依然として**仮定した試験材料**です。コンパイルとSHA-256照合は署名・発行主体の信頼、正式なContract/Conformance受理、起動FDの固定、実OS Sandbox、運用中のGrant／Leaseの再検証、受注Actionの認可を証明しません。製品用の第二ResolverやAuthorityを導入するものでもありません。

## 案件所有の独立Goプロセス用アダプター

`cmd/reference-project-advice-process` は、このパッケージの`Evaluate`を既存TEIプロセス通信へ写す薄い参照用実行体です。ホストが導出するdispatch IDを起動引数として照合しますが、アダプター自身はそのIDの受理や実行権限を判断しません。検証例は`tools/reference-runtime-capability-identity/project_advice_built_pinned_linux_test.go`にあります。

独立したGo実行体として構築・実行できることと、**正式なプラグインの受理、署名、権限付与、業務Actionへの登録は別**です。既存`Plugin Lock`や`Host Authority`を置き換えません。
