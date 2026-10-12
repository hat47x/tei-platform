# 案件専用Go補助プラグイン × Web標準/Lit（最小参照）

対象：#951 Service Contract、#997 UI構成キット、#1016 案件Composition。生成AIが短い要求から案件専用の小さなGoロジックとLit Web Componentを作り、動かすための**参照用の技術的な最初の縦断**です。

- Goの案件固有処理：[顧客メモの補助チェック](../../plugins/reference-project-customer-advice/README.md)。`VIP-`で始まる架空コードにメモが無ければ**提案**を表示するだけです。
- Web：LitのCustom Elementと標準の`<form>`、名前付き入力、キーボード操作、`aria-live`状態領域。
- 接続：診断用Go HTTP Hostは、案件プラグインを明示的に呼び出し、JSONを返します。HTTP結果は正式なTEI Service Contract / Provider selection / Host Authorityを証明しません。
- **重要**：本サンプルは`order_entry`の**読み取り専用シナリオ**へ任意で接続できますが、受注Actionの呼出し、Accepted Validation、認証、認可、DB更新には接続しません。案件の補助提案を、受注を拒否する根拠に使わないでください。

## 実行と検証

リポジトリ直下で：

```sh
go test ./plugins/reference-project-customer-advice ./tools/reference-project-advice-lit
node --test tools/reference-project-advice-lit/web/*.test.mjs
bash tools/reference-project-advice-lit/smoke_order_entry.sh
go run ./tools/reference-project-advice-lit
```

http://127.0.0.1:8787/ をブラウザで開き、顧客コード`VIP-123`・メモ空欄で補助確認を実行してください。結果は`review_suggested`ですが、業務の承認ではありません。

### 実`order_entry`読取との接続

`smoke_order_entry.sh`は実第一Web TargetのGo APIをビルド・起動し、案件参照Hostを`-order-entry-api http://127.0.0.1:18977`で明示接続します。実APIの`GET /scenario/read`から`applicationId`、`scenarioId`、`selectedRowIndex`、Target buildを抽出して`GET /project/order-entry-context`として表示します。元の業務データ行、認証情報、Command結果は転送しません。

API URLは明示的な`http://127.0.0.1:<port>`のみ許可し、HostはBrowserから受け取ったCredential/Cookie等を上流へ転送しません。接続未指定の場合は対象経路を公開しません。この接続は**診断用の読取Consumer**であり、正式なTEI Provider/Runtime Authorityや業務Action統合ではありません。

既存の受注Commandを変更せず、案件側が実Applicationと同じ画面で参照状況を確認できる足場です。実ブラウザでの表示や人間の受入は別途検証します。

**Lit依存**：公式の[Lit配布バンドル](https://github.com/lit/dist/tree/v3.3.3/core) の `v3.3.3/core/lit-core.min.js` をこの参照実装内へ版固定で同梱します。ライセンスは `web/LIT-LICENSE`（BSD-3-Clause）です。UI配信時はCDNへアクセスせず、同じHostからロードし、CSPは `script-src 'self'` です。配布前には資産Digest/供給経路を既存Builder/Lockと接続する必要があります。この段階では実ブラウザ操作は別途検証するものとします。GoテストはHTTP HostとGoロジック、NodeテストはUI通信境界をそれぞれ評価するものです。

## 以降の段階

1. 実TEI Applicationの読み取り専用参照は追加したが、正式なApplication ActionとService Contract/Host Authorityへの案件Extension接続は未完了。既存のValidation/Authority/Resultを保持し、補助提案を第二の業務正本にしない。
2. Litをローカルバンドルし、実ブラウザでアクセシビリティ・結果表示・ネットワーク断時の不確実性を確認する。
3. TEIの契約解決と現在のHost Authorityを通して案件Pluginを呼ぶ。本Reference内だけのGo直接importを製品Runtime採用の証拠としない。
4. 生成AIによる要求→コード生成→試験→差分の追跡を測定する。第三者向け配布、性能証明、Marketplace対応は実需要があれば追加する。
