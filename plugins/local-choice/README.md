# ローカル認知・型付き選択（参照プラグイン）

関連 #5807、#5808、#5809、#2021、#2533。**参照実装。製品の新規契約ではない。**

## 利用する機能

文章と選択肢ごとの判断基準を与え、選択肢IDを一つだけ返す。判断できないときは `unknown`。問い合わせ分類、設計資産の分類、型の適合候補の絞込みなどに同じ処理を使える。モデルの回答は候補であり、業務状態・設計の変更を行わない。

既存の認知提案ポートと `internal/composition.Registry` を使う。CLIで `-plugin local-choice` を指定すると、静的に結合した実装のContractを選択し、既存Request/Provider/Proposal/Receiptの検査・出所管理を経て呼び出す。HTTPやモデル固有の変換は本プラグインに閉じる。従来の `-command` 経路は変更しない。第二の実行Host、固定CapabilityKindの追加、独自の受理機構は設けない。

## 実行

1. 既存のローカル推論サーバーを起動する（OpenAI互換chat completions、loopbackのみ）。モデル別名を確認する。
2. [Provider例](testdata/provider.json)をコピーし、model・version・endpointを環境に合わせる。versionは利用者の申告であり、実モデルの内容照合はしない。
3. リポジトリのルートで実行する。

```sh
go run ./tools/reference-cognition-port/cmd/tei-cognition-propose -plugin local-choice -request plugins/local-choice/testdata/inquiry.json -provider plugins/local-choice/testdata/provider.json
```

例のproviderを編集したコピーを使う場合は `-provider` にそのパスを指定する。実モデルの応答は未確認。期待する選択候補は `billing`。結果は既存のreceipt JSONで、`proposal.candidates[0].payload.choice` に候補ID、`basis` に入力と基準の参照が出る。`accepted`、`grantsAuthority`、`applied` はfalse。unknown/failed/blockedは終了コード1。汎用ハーネスが使用している同じローカルサーバーを利用できる。

## 契約の絞込みと基盤との関係

- 入力は既存Contextの観測で、ID `input` の一件（statementとsubject reference）、および `option.<id>` の2〜16件（statementは判断基準、referenceは空）。選択肢は小文字ASCIIのIDで、unknownは予約。未対応の制約・受理済み参照を黙って無視せず拒否する。これは参照Consumerの入力投影であり、新しいDSLの採用ではない。
- `reference-cli` scopeで既存合成Registryのexact Contractを選ぶ。CLIの静的構成が実装を指定する。宣言上の選択は正式Contract発行、署名、実行権限、製品Runtimeの準拠性を保証しない。HTTPをアプリケーション実行用の旧CapabilityKindに押し込まない。
- 既存ポートには固定されたlocal／external command以外の静的な呼出し入口がなかった。共通検証を複製せず、選択済みProviderの呼出しを注入する入口を追加した。ADR-0032/0033/0043に沿う参照段階の統合であり、将来の生成Targetは同じ意味契約を各実現方式へloweringする。ADR-0038の提案・受理分離を維持する。
- 現行offlineはnetwork:noneに限定されるため、loopback HTTPでもconnected/network:requiredと宣言する。通信先は明示したloopback IPだけで、proxy、redirect、クラウドへのfallbackはない。サーバー自身の外部送信やOSの閉域は未検証。汎用的な通信範囲の表現が必要なら #951/#2515 で既存の配置・権限契約と統合して判断する。
- 一度に一件を同期で実行し、出力32 token、観測8192 byte、HTTP応答64 KiB、descriptorの期限を上限とする。CLIを別プロセスで実行するだけで前景UIを待たせずに利用できるが、キュー、アイドル制御、増分結果の再利用、実アプリへの反映は未実装。認知処理を保存やビルドの必須条件にしない。
- 短答の形式や候補範囲を検査できても、判断の正しさ、根拠の妥当性、入力内命令に対する意味的な耐性を保証しない。モデル自己申告の確率は要求しない。候補の採否は既存の受理境界へ戻す。

## 試験

```sh
go test -count=1 ./plugins/local-choice ./tools/reference-cognition-port/...
```

テストは実HTTPの代役サーバーを用い、固定された期待値で候補・棄権・候補外・出力打切り・offline・別Providerを区別する。外部IPとredirectを拒否する。これは実モデルの品質・CPU処理能力・正式な製品実行権限の試験ではない。

実機での初回受入は、例の問い合わせ、技術的不具合、判断不能の三件を実モデルで実行し、独立期待と所要時間を照合する。通常利用に必要な品質と待ち時間が得られるかはこの観測で判断する。次は同じContractのStudio文脈／業務Contextへの接続を #5808 と #5809 で別々に検討する。
