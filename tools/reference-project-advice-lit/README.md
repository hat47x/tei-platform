# 案件専用Go補助プラグイン × Web標準/Lit（最小参照）

対象：#951 Service Contract、#997 UI構成キット、#1016 案件Composition。生成AIが短い要求から案件専用の小さなGoロジックとLit Web Componentを作り、動かすための**参照用の技術的な最初の縦断**です。

- Goの案件固有処理：[顧客メモの補助チェック](../../plugins/reference-project-customer-advice/README.md)。`VIP-`で始まる架空コードにメモが無ければ**提案**を表示するだけです。
- Web：LitのCustom Elementと標準の`<form>`、名前付き入力、キーボード操作、`aria-live`状態領域。
- 接続：診断用Go HTTP Hostは、案件プラグインを明示的に呼び出し、JSONを返します。案件補助チェックは既存の `internal/composition.Registry` に案件Scope・Contract・Providerを宣言し、`Select` で適格なProviderを選んでからGoの案件ロジックを呼び出します。**Registry Selectionは宣言上の適格性に限定されます。** 診断Hostは既存`NewSelectedCoordinator`と`Acquire`/`Release`を使いますが、合成Bindingと参照用Guardであり、正式なHost Authorityには達していません。
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

**ループバック診断Hostの境界**：実Listenerは `127.0.0.1:8787` に固定し、すべての静的ファイル・補助POST・読取GETの手前で受信 `Host: 127.0.0.1:8787` 以外を `421` で拒否します。ループバックへのbindだけでは防げないDNSリバインディングへの局所的な対策です。Goユニットテストと実Goプロセスを起動する `smoke_order_entry.sh` の両方で検証します。これは診断Hostのアクセス境界であって、ユーザー認証や正式なTEI認可の代替にはなりません。

既存の受注Commandを変更せず、案件側が実Applicationと同じ画面で参照状況を確認できる足場です。実Chromeでの表示と操作はCIで検証していますが、人間による画面受入は別途必要です。

**Lit依存**：公式の[Lit配布バンドル](https://github.com/lit/dist/tree/v3.3.3/core) の `v3.3.3/core/lit-core.min.js` をこの参照実装内へ版固定で同梱します。ライセンスは `web/LIT-LICENSE`（BSD-3-Clause）です。上流のGitタグ `v3.3.3` にある配布バンドルのGit blob SHAは `30071d620d392394d56721f05fc4808e02e038e3` で、CIが同梱ファイルと照合します。UI配信時はCDNへアクセスせず、同じHostからロードし、CSPは `script-src 'self'` です。配布前には資産Digest/供給経路を既存Builder/Lockと接続する必要があります。GitHub Actionsの [`tei-reference-project-advice-lit-browser-ci.yml`](../../.github/workflows/tei-reference-project-advice-lit-browser-ci.yml) で、PR正確HEADから実Goの受注APIと案件Hostを別プロセス起動し、Google Chromeを通してLitのShadow DOM・キーボード操作・補助提案・実読取結果・断線時の結果不確実性を検証します。1280×800と375×812を対象に画面を撮影し、SHA-256を持つ診断ReceiptをRunner外部Artifactとして保存します。テスト用ChromeとPlaywrightはRunner上で用意し、Browser配信は同一Originのローカルバンドルのみを使用します。なお、これは実Chrome上の機能検証であり、人間によるVisual受理や本番のService Contract/Authority・安全性認証ではありません。

## 以降の段階

1. 実TEI Applicationの読み取り専用参照は追加したが、正式なApplication ActionとService Contract/Host Authorityへの案件Extension接続は未完了。既存のValidation/Authority/Resultを保持し、補助提案を第二の業務正本にしない。
2. Litのローカル配信、実Chromeでのキーボード操作・結果表示・通信断は局所検証済み。包括的なアクセシビリティ検査、人間のVisual受入は今後の段階。
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

`contract_bridge.go` は新しい汎用Resolverを実装せず、既存の `composition.Contract`・`Provider`・`Registry.Apply`・`Registry.Select` を利用します。案件ごとのScopeを分け、Providerの削除・Contract変更・他案件Scopeとの不一致時は補助確認を失敗として扱います。HostがWithdrawされた場合は、Registryの宣言が残っていてもCoordinatorが新規Leaseを拒否します。Retired generationのReap後も自動復活しないことを追加試験します。Go関数の実行は診断Hostのローカル呼び出しですが、既存の `NewSelectedCoordinator` と `Acquire` / `Release` を通すようにしました。Reference用の固定Bindingと合成Digest、簡易Host Guardを使用するため、**Production向けの実Artifact検証、現在の権限・fencing検証は未成立**です。`contract_bridge_test.go` で案件Scope隔離とProvider削除時の拒否を検証します。正式Runtime統合の達成を意味しません。

### main側ビルド版T固定との整合

Go結合CIは通常のPull Requestマージ結果を検証し、実Chrome CIはPR正確HEADを検証します。最新mainの第一Web APIは、実行時指定のTarget Tだけでなく、ビルド時に埋め込んだTとの一致を要求します。`smoke_order_entry.sh`では既存E1ビルダーと同じ `-ldflags="-buildid= -X main.referenceAPISourceRevision=..."` を指定し、実行時の`-target-revision`と合わせます。この一致検査は**診断用Target同一性**であり、正式なArtifact Lockやリリース署名ではありません。
