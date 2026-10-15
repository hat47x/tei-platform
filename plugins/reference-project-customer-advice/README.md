# 案件固有の顧客向け補助チェック — Go参照実装

#951 / #997 / #1016 の生成AI主導・案件別実装を検証するための最小実装です。

`Evaluate` は顧客コード `VIP-` で始まり、メモが空の場合にレビュー提案を返します。これは**架空の案件固有ルール**であり、実際の受注条件やTEIのAccepted Business Validationではありません。外部I/O・DBアクセス・認可・注文更新を行いません。

Go標準ライブラリのみで動作し、次のコマンドで単体試験できます。

```sh
go test ./plugins/reference-project-customer-advice
```

UIとHTTP Hostは [参照実装](../../tools/reference-project-advice-lit/README.md)を参照してください。これは製品Runtimeで登録された正式Service Providerではありません。正式な受注可否、Runtime Authority、Conformance、第三者向け配布品質は保証しません。

## 独立Goプロセスの唯一の参照入口（#976）

`cmd/reference-project-advice-process/` を、案件所有の `Evaluate` を既存TEI Process Request/Responseへ写す**唯一の独立実行アダプター**とします。以前の `cmd/reference-advice-process/` は、別のPRで同時期に追加された重複入口のため廃止しました。案件本体の業務ロジックを二重実装しません。

```sh
go build -trimpath -buildvcs=false -o /tmp/reference-project-advice-process ./plugins/reference-project-customer-advice/cmd/reference-project-advice-process
go test -count=1 ./tools/reference-runtime-capability-identity ./plugins/reference-project-customer-advice
```

Hostは、既存Generic Runtime Capability Refから導出したdispatch IDを `--tei-expected-dispatch=<ID>` という起動引数で渡します。これは契約の受理や権限の付与ではなく、**要求のdispatch先が起動時のHost選択と一致すること**を試験するための境界です。

検証は次の二方式で同じアダプターを使用します。

- `project_advice_independent_binary_test.go`：単独でビルドした実行ファイルのSHA-256を起動前に照合し、通常の `StdioProcessLauncher` で起動する。改ざん、偽のdispatch、未知の操作、不正な入力、Hostの起動引数欠落を拒否する。
- `project_advice_built_pinned_linux_test.go`：既存の `PinLinuxProcess` が保持するLinux FDから実行し、検査後のパス置換が実行内容を変えないことを確認する。

二つはTransport・Artifact固定方式の異なる試験であり、プラグイン作者に二つの実行入口を要求するものではありません。

**非達成**：Contract version/fingerprint、Provider、Resolution Lock、Execution Grantは試験用の合成候補であり、正式なContract/Conformance受理ではありません。署名・発行者の信頼、OS Sandbox適用、運用時のPrincipal/Action認可、実Resource Lease、本番RegistryのGeneric Capability統合は証明されません。独立したResolver、Permission Store、Composition Profileを追加せず、これらは既存の#951/#976の担当境界に残します。
