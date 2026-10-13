# 参照モデルとプラグイン接続の開発候補の引継ぎ

2026-10-13。[接続計画](REFERENCE_MODEL_PLUGIN_EXECUTION_2026-10.md)に対する固定候補の検証結果と、残る接続条件を記録する。現在の実装説明は[実装状況](../../docs/status.md)と各実行手順を正本とする。本書は製品全体の完了判定や新しいGateではない。

## 対象版と確認単位

コード候補は `6be8f55ba274337b0caeae262f311b69a77482d3`、作業ブランチは `codex/reference-model-composition-20261012`。作業開始時に確認した `origin/main` の `800f54bc` を通常の統合で取り込んだ。変更のないLinuxチェックアウトで、次の範囲を同じ候補版として確認した。

| 開発候補の機能 | 実行した確認 | 未成立の境界 |
| --- | --- | --- |
| 選択済みサービスを使う受注・修理受付 | 複合・受注専用・修理専用の実HTTPで従来結果との同等性、実行物識別、除外経路・依存を確認 | Oracle顧客参照と認証は模擬実装 |
| PostgreSQLの受注・明細・確定・別観測 | 通常ホストと固定Worldで実DBへ接続。許可失効、古い版、重複確定、DB停止、保存障害を検査 | 本番データ、汎用の権限基盤、認可正本との原子的な結合は未成立 |
| ソース変更から再実行 | リポジトリ外の複製で提案・確認・受理・選択版・DB許可・明示的再起動を接続。未受理・古い案・直接変更を拒否 | 人による承認、任意の外部書込みとの原子性、無停止のモデル交換は未成立 |
| 帳票・一括処理・停止後診断 | CSV／JSON、部分成功、取消、進行ログ、実CLI強制終了、ID指定観測、ログを入力にした診断を確認 | 永続的な要求ID、自動再開、PDF、本番監査証跡は未成立 |
| 別プロセスの世代管理 | 実プロセスの構築・起動・実行物固定・交換・旧処理保持・準備失敗・解放・復旧を関連試験で確認 | DB更新と交換の一体運用、本番署名・認可、可用性目標は未成立 |

この候補は、受注の実DB操作から停止後の診断までをローカルで再現できる。表の未成立項目を含む製品要求全体の完了にはしない。

## 再現手順

Go・Python・Dockerを利用できるLinuxで、固定した候補版の変更のないチェックアウトから実行する。出力先は新しいリポジトリ外のディレクトリとする。

```sh
python3 tools/verify-postgres-order-execution.py --source-loop --non-screen --batch-crash --output /absolute/new/live-evidence
python3 tools/verify-postgres-order-execution.py --frozen-api --non-screen --batch-crash --output /absolute/new/frozen-evidence
python3 tools/verify-composed-targets.py --output /absolute/new/composed-evidence
go test -race -count=1 ./internal/composition ./plugins/reference-process-executor ./plugins/report-renderer ./tools/reference-composition-process-bridge ./tools/reference-first-web-target
go test -race -count=1 -tags=tei_order_only,tei_api_only,tei_frozen_world ./tools/reference-first-web-target
go test -race -count=1 -tags=tei_order_only,tei_api_only ./tools/reference-first-web-target
go test -race -count=1 -tags=tei_repair_only ./tools/reference-first-web-target
go vet ./internal/composition ./tools/reference-first-web-target
```

実DB試験は通常の単体試験では省略する。検証器が専用DBを作り、実DB試験が一件も省略されなかったことを検査する。模擬HTTPの同等性、実DBへの作用、実プロセスの交換は別の根拠として扱う。ブラウザー・表示品質・人による利用価値は今回の確認に含めない。

確定結果の不明を扱う試験では、実DBのCOMMIT後に試験用の内部接続点から失敗を返す。先頭件は確定済み、後続件は未処理であり、保存ログの診断と別観測を行っても再実行しないことを確認した。これは応答喪失の注入試験で、実ネットワークの切断・応答喪失の再現ではない。通常の起動には注入設定を公開しない。

## 次に満たす接続条件

1. 実際の認証提供元と認可正本を定め、その主体リンク・所属・規則版から正規の版付き許可を取得する。模擬認証から版を作って代用しない。
2. 既存の永続作用の契約に従い、受注更新・現在の認可版の検査・要求IDの確定記録を同じ物理DBトランザクションへ結ぶ。現在の参照ホストの実行許可表や進行ログを第二の認可正本にしない。
3. 現在の権限を再検査してから確定記録を照会し、同じ要求内容・対象・主体の再送を識別する。異なる要求や失効した権限からの照会を拒否する。
4. 実ネットワークでの確定応答喪失、実運用のDB接続、Oracle、表示・操作、負荷と可用性をそれぞれの担当条件で確認する。

これらは現在の候補の成功から推定しない。接続先と認可正本が未指定の段階で、新しい権限モデルや自動再開を追加しない。

## 統合状態

変更はローカルの作業ブランチへコミット済み。pushとmainへの統合は未実施。Draft PRの統合は行っていない。後続のmain変更への適合、人的レビュー、本番への配備は、この固定候補の確認に含めない。
