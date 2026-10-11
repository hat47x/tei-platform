# 第一Web Targetから固定Projectionを観測するCLI

このReferenceは、**すでに起動している**第一Web Targetの `GET /manifest.json` を読み取り、明示されたApplication・Handover B/R・Target build Tと一致することを確認して、取得した**元のJSON bytes**を標準出力に書きます。既存のWeb Target Sourceを変更せず、業務Runner、認証、Command、BuilderやPlugin選択を呼び出しません。

## 使用例

第一Web Targetの実行には、既存の[実行方法](../reference-first-web-target/README.md)を使用します。例えば、検証対象checkoutで次のように起動します。

```sh
go run ./tools/reference-first-web-target \
  -workspace reference/workspace \
  -application order_entry \
  -baseline-id B \
  -baseline-revision "$ACCEPTED_BASELINE_R" \
  -target-revision "$(git rev-parse HEAD)" \
  -listen 127.0.0.1:8080
```

別のターミナルから、**同じB/R/Tを引数で独立に指定**して観測します。

```sh
go run ./tools/reference-first-web-target-manifest-capture \
  -url http://127.0.0.1:8080/manifest.json \
  -application order_entry \
  -baseline-id B \
  -baseline-revision "$ACCEPTED_BASELINE_R" \
  -target-revision "$FIRST_WEB_TARGET_T" \
  > /tmp/tei-observed-first-web-manifest.json
```

`$FIRST_WEB_TARGET_T`は第一Web Targetを**実際に起動した**checkoutのコミットです。取得時に計算したraw bytesのSHA-256を**observed hash**として標準エラーへ表示します。呼出し側が独立のAcceptance/Build Artifact記録で照合して確定しない限り、その値は**受理済みのdigestや署名ではありません**。

- 取得URLは正確に`http://127.0.0.1:<port>/manifest.json`だけを許可します。HTTPS、hostname、別IP、query、fragment、userinfo、redirectを拒否します。OSのHTTP proxy設定を使わず、10秒のtimeoutを設定します。 HTTPによる自動gzip展開も無効にし、Content-Encodingが`identity`以外の場合は拒否します。
- HTTP 200、`application/json`、8 MiB以下の本文、Application・B/R/T一致、およびJSON構文とtop-level IDの重複なしを確認します。失敗時は内容を標準出力しません。
- 検証後のJSONを再シリアライズせず、受信したbytesをそのまま出力します。出力先はcheckout外にしてください。Shellの`>`リダイレクトは事前に空ファイルを作るため、失敗時に残る可能性があります。これを受理済み成果物に見せず、取得成功時にのみ次の承認段階へ渡してください。
- E0源泉依存のテストで、当CLIはTEIメインモジュールの`reference-runtime-snapshot`だけをimportします。Workspace/Driver/DSLを直接起動しません。

## 検証

```sh
go test -count=1 ./tools/reference-first-web-target-manifest-capture
go vet ./tools/reference-first-web-target-manifest-capture
```

テストはloopback上のHTTP fixtureで、JSON raw bytesの保存、Application/B/R/Tの不一致、duplicate key、redirect、HTTP status/Content-Typeの異常、本文サイズ制限、外部URLを拒否することを検査します。これは**第一Web Target本体のFunctional/Visual/Gate C結果ではありません**。

観測したProjectionを[固定Runtime Reference](../reference-frozen-manifest-target/README.md)に受け渡すには、元のHandover／Build条件と観測されたSHA-256を**独立に検証・固定**してから、固定Runtimeに`-manifest`と期待ID/digestを渡します。このcapture CLIの成功だけでは、配備物の真正性、Business Result、認証認可、Browser/API E1の分離、TEIプロダクト適合性を主張できません。
