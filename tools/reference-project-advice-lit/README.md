# 案件専用Go補助プラグイン × Web標準/Lit（最小参照）

対象：#951 Service Contract、#997 UI構成キット、#1016 案件Composition。生成AIが短い要求から案件専用の小さなGoロジックとLit Web Componentを作り、動かすための**参照用の技術的な最初の縦断**です。

- Goの案件固有処理：[顧客メモの補助チェック](../../plugins/reference-project-customer-advice/README.md)。`VIP-`で始まる架空コードにメモが無ければ**提案**を表示するだけです。
- Web：LitのCustom Elementと標準の`<form>`、名前付き入力、キーボード操作、`aria-live`状態領域。
- 接続：診断用Go HTTP Hostは、案件プラグインを明示的に呼び出し、JSONを返します。案件補助チェックは既存の `internal/composition.Registry` に案件Scope・Contract・Providerを宣言し、`Select` で適格なProviderを選んでからGoの案件ロジックを呼び出します。**ただしSelectionは宣言上の適格性に限り、正式なTEI Host AuthorityやRouter/Coordinator経由の実行許可ではありません。**
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

**Lit依存**：公式の[Lit配布バンドル](https://github.com/lit/dist/tree/v3.3.3/core) の `v3.3.3/core/lit-core.min.js` をこの参照実装内へ版固定で同梱します。ライセンスは `web/LIT-LICENSE`（BSD-3-Clause）です。上流のGitタグ `v3.3.3` にある配布バンドルのGit blob SHAは `30071d620d392394d56721f05fc4808e02e038e3` で、CIが同梱ファイルと照合します。UI配信時はCDNへアクセスせず、同じHostからロードし、CSPは `script-src 'self'` です。配布前には資産Digest/供給経路を既存Builder/Lockと接続する必要があります。GitHub Actionsの [`tei-reference-project-advice-lit-browser-ci.yml`](../../.github/workflows/tei-reference-project-advice-lit-browser-ci.yml) で、PR正確HEADから実Goの受注APIと案件Hostを別プロセス起動し、Google Chromeを通してLitのShadow DOM・キーボード操作・補助提案・実読取結果・断線時の結果不確実性を検証します。1280×800と375×812を対象に画面を撮影し、SHA-256を持つ診断ReceiptをRunner外部Artifactとして保存します。テスト用ChromeとPlaywrightはRunner上で用意し、Browser配信は同一Originのローカルバンドルのみを使用します。なお、これは実Chrome上の機能検証であり、人間によるVisual受理や本番のService Contract/Authority・安全性認証ではありません。

## 以降の段階

1. 実TEI Applicationの読み取り専用参照は追加したが、正式なApplication ActionとService Contract/Host Authorityへの案件Extension接続は未完了。既存のValidation/Authority/Resultを保持し、補助提案を第二の業務正本にしない。
2. Litをローカルバンドルし、実ブラウザでアクセシビリティ・結果表示・ネットワーク断時の不確実性を確認する。
3. TEIの契約解決と現在のHost Authorityを通して案件Pluginを呼ぶ。本Reference内だけのGo直接importを製品Runtime採用の証拠としない。
4. 生成AIによる要求→コード生成→試験→差分の追跡を測定する。第三者向け配布、性能証明、Marketplace対応は実需要があれば追加する。

## 実ブラウザ検証で確認すること

- Lit 3.3.3のローカルバンドルが埋め込まれ、ブラウザから外部CDNへ接続しないこと。
- 案件補助チェックの「提案あり」「提案なし」を実Go Host経由で表示すること。キーボードによるフォーム送信を含む。
- 実際の`order_entry` APIの`GET /scenario/read`と一致する参照内容をLitが表示すること。既存の`POST /scenario/submit`は呼び出さないこと。
- 補助チェック通信断と、受注参照API停止時に、成功を表示しないこと。
- デスクトップ・モバイルの実画面と検証Receiptを保存し、未成立のAuthority・受入範囲を明示すること。

作業領域を汚さずに同等の検証を実施する手順は専用Workflowを参照してください。実Chrome確認済みであっても、業務Actionの変更・認証/認可・正式Service ContractへのPlugin登録は別の課題です。

## 既存Composition Registryとの最小接続

`contract_bridge.go` は新しい汎用Resolverを実装せず、既存の `composition.Contract`・`Provider`・`Registry.Apply`・`Registry.Select` を利用します。案件ごとのScopeを分け、Providerの削除・不一致時は補助確認を失敗として扱います。Go関数の実行は診断Hostのローカル呼び出しですが、既存の `NewSelectedCoordinator` と `Acquire` / `Release` を通すようにしました。Reference用の固定Bindingと合成Digest、簡易Host Guardを使用するため、**Production向けの実Artifact検証、現在の権限・fencing検証は未成立**です。`contract_bridge_test.go` で案件Scope隔離とProvider削除時の拒否を検証します。正式Runtime統合の達成を意味しません。
