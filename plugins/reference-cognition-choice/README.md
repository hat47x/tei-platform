# ローカル認知による型付き選択（参照プラグイン）

[#5807](https://github.com/hat47x/tei-platform/issues/5807) の最初の実用場面は、自由記述の問い合わせを、利用者が定義した分類基準に沿って分類することです。[#5809](https://github.com/hat47x/tei-platform/issues/5809) の業務側の第一縦断と、[#5808](https://github.com/hat47x/tei-platform/issues/5808) のStudio側の候補選択に共通する処理です。既存のCognition Portを使い、判断を正本へ自動適用しません。

## 使い方

ローカル推論サーバーが起動済みであることを前提とします。examples/provider.json の model をサーバーが公開する実モデルIDへ、version を実際に使用するモデル版へ変更してください。必要に応じてポートを変更します。外部への通信や認証を必要とするサーバーはこの参照実装の対象外です。

リポジトリのルートから実行します。

```sh
go run ./tools/reference-cognition-port/cmd/tei-cognition-propose -local-choice -request plugins/reference-cognition-choice/examples/inquiry.json -provider plugins/reference-cognition-choice/examples/provider.json
```

例の独立した期待は billing です。成功すると、receipt の proposal.candidates[0].payload.optionId に分類候補を返します。情報が不足する場合は unknown、形式違反・候補外・接続失敗は failed です。分類対象・選択肢・判断基準を request 側で変更しても、プラグインや生成器を変更する必要はありません。

一要求ごとに終了するため、既存のタスクキューやOSのスケジュールから呼び出せます。サーバーは外部で共有し、モデルを起動する責務をTEIへ持たせません。まず同時数1で運用し、120秒の期限は実環境に応じて調整します。長文生成や自動ツール実行は扱いません。バックグラウンドキュー・優先順位・キャッシュ・現在性の再確認は未実装です。

## プラグイン契約との関係

- ADR-0032/0033/0043に従い、判断契約、提供実装、選択、実行先を分ける。独自Hostは追加しない。
- 既存Cognition Portに choice の質問（観測ID、選択肢IDと説明）を追加した。返答は一つの宣言済みIDまたは unknown。根拠IDの存在は検査するが、判断の正しさを保証しない。既存kindは維持する。
- 既存の外部Provider呼出しに、信頼済み実装を静的に渡す入口を設けた。CLIの -local-choice はこの参照用の結合であり、Providerの設定にある transport、endpoint、model が実際の呼出しを決める。Go標準ライブラリだけで実装し、プラグイン側はTEI内部型へ依存しない。
- 従来の外部コマンド経路は保持する。新たな固定CapabilityKind、専用Registry、認可Engineは追加しない。
- 汎用Service Contractの解決・Plugin Lock・Artifact資格・生成時の静的Loweringからこの実装を選択する正式な経路は未接続。現CLIの静的結合を製品プラグイン機構の完成と扱わない。参照結果に基づき、契約の入出力・任意依存・期限・Provider設定の束縛を既存契約へ還元する。
- network: none は外部ネットワーク不要というProvider宣言。実装はliteral loopback HTTP、proxyなし、redirectなしに限定するが、サーバープロセス自身の外部通信を隔離するものではない。

## 検査

```sh
go test -count=1 ./tools/reference-cognition-port/... ./plugins/reference-cognition-choice
```

参照試験では実HTTP通信の代役を使って、正常候補・棄権・候補外・不正な長答、ポート側での候補集合検査を区別する。実モデル、Windowsでの動作、実際の業務精度、生成UI/Targetは別の受入項目。実モデルでの速度をこの試験から推定しない。

次に実モデルで同じ例を実行し、入力処理を含む所要時間、回答、receiptを確認する。利用者の入力・判断基準を変更した例でも動作することを確認し、正式なプラグイン選択への接続を既存Ownerと設計する。