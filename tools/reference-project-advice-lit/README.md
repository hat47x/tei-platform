# 案件専用Go補助プラグイン × Web標準/Lit（最小参照）

対象：#951 Service Contract、#997 UI構成キット、#1016 案件Composition。生成AIが短い要求から案件専用の小さなGoロジックとLit Web Componentを作り、動かすための**参照用の技術的な最初の縦断**です。

- Goの案件固有処理：[顧客メモの補助チェック](../../plugins/reference-project-customer-advice/README.md)。`VIP-`で始まる架空コードにメモが無ければ**提案**を表示するだけです。
- Web：LitのCustom Elementと標準の`<form>`、名前付き入力、キーボード操作、`aria-live`状態領域。
- 接続：診断用Go HTTP Hostは、案件プラグインを明示的に呼び出し、JSONを返します。HTTP結果は正式なTEI Service Contract / Provider selection / Host Authorityを証明しません。
- **重要**：本サンプルは現在の`order_entry`受注Actionに接続しておらず、受注のAccepted Validation、認証、認可、DB更新を行いません。案件の補助提案を、受注を拒否する根拠に使わないでください。既存の受注APIへの接続は次段階です。

## 実行と検証

リポジトリ直下で：

```sh
go test ./plugins/reference-project-customer-advice ./tools/reference-project-advice-lit
node --test tools/reference-project-advice-lit/web/advice-action.test.mjs
go run ./tools/reference-project-advice-lit
```

http://127.0.0.1:8787/ をブラウザで開き、顧客コード`VIP-123`・メモ空欄で補助確認を実行してください。結果は`review_suggested`ですが、業務の承認ではありません。

**外部依存の留保**：UIは公式Litドキュメントにある配布用ES moduleを、参照画面に限りCDNから読み込みます。ネットワークが利用できなければ表示できません。本番配布ではLitをローカルでバンドルし、依存の版・資産Digest・CSPを固定してください。このReferenceでは実ブラウザの描画・Computer Use操作・バンドル適合性をまだ自動検証しません。GoテストはHTTP HostとGoロジック、NodeテストはUI通信境界をそれぞれ評価するものです。

## 以降の段階

1. 実TEI Application / Actionに案件Extensionを接続し、既存の正式なValidation/Authority/Resultを保持する。補助提案を第二の業務正本にしない。
2. Litをローカルバンドルし、実ブラウザでアクセシビリティ・結果表示・ネットワーク断時の不確実性を確認する。
3. TEIの契約解決と現在のHost Authorityを通して案件Pluginを呼ぶ。本Reference内だけのGo直接importを製品Runtime採用の証拠としない。
4. 生成AIによる要求→コード生成→試験→差分の追跡を測定する。第三者向け配布、性能証明、Marketplace対応は実需要があれば追加する。
