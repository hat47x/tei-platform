# reference-accounting-period（#2144）

業務日付（計上日）とシステム時刻を分け、会計期間の状態（開いている・締め中・締め済み）を宣言し、締め済み期間への直接更新を拒否して赤黒訂正へ導く読取り専用の参照です。締め後の到着は、宣言した方針（当月で調整／保留）に従います。

帳簿・DBトリガー・ライフサイクル実行器ではありません。締めや訂正は合成入力に対する判定であり、保存・確定は行いません。

## 実行

```bash
# 1つの操作を判定する
go run ./tools/reference-accounting-period/cmd/tei-accounting-period \
  -input tools/reference-accounting-period/testdata/closed-update.json

# 業務順でイベントを再生する
go run ./tools/reference-accounting-period/cmd/tei-accounting-period -mode replay \
  -input tools/reference-accounting-period/testdata/replay.json
```

## 入力（evaluate）

```json
{
  "systemTime": "2026-05-10T09:00:00Z",
  "periods": [
    {"id": "2026-04", "state": "closed", "start": "2026-04-01", "end": "2026-04-30"},
    {"id": "2026-05", "state": "open",   "start": "2026-05-01", "end": "2026-05-31"}
  ],
  "slips": [
    {"id": "S-APR-1", "period": "2026-04", "businessDate": "2026-04-15", "amount": "1000", "status": "posted"}
  ],
  "action": {"kind": "update", "slipId": "S-APR-1", "businessDate": "2026-04-15", "amount": "1200"},
  "latePolicy": "hold"
}
```

- `periods[].state`は`open`・`closing`・`closed`
- `start`と`end`は業務日付（`YYYY-MM-DD`、両端を含む）
- `systemTime`は記録するだけで、期間の判定に使わない
- `latePolicy`は`adjust-current`（当月の開いている期間で調整）または`hold`（保留）。省略時は`hold`

## 判定

| 操作・条件 | 結果 |
| --- | --- |
| 開いている期間への登録 | `POSTED` |
| 締め済み期間の伝票を直接更新 | `CLOSED_PERIOD_REJECTED`（赤黒訂正を使う） |
| 締め済み期間の元伝票を、開いている期間の訂正で逆仕訳 | `CORRECTION_POSTED`（`correctionOf`と`linkedSlips`） |
| 訂正の元伝票がない | `CORRECTION_MISSING_ORIGINAL` |
| 締め済み期間へ訂正を計上 | `CLOSED_PERIOD_REJECTED` |
| 締め済み期間に届いた登録（`adjust-current`） | `LATE_ARRIVAL_ADJUSTED`（開いている期間へ） |
| 締め済み期間に届いた登録（`hold`） | `LATE_ARRIVAL_HELD`（値を変えない） |
| 締めの再開 | `REOPEN_NOT_PERMITTED`（権限と監査の判断が必要） |
| 未処理の伝票がある締め | `CLOSE_BLOCKED_PENDING` |
| すべて登録済みの締め | `CLOSE_ACCEPTED`（判定であり保存ではない） |
| 期間に含まれない業務日付 | `UNKNOWN_PERIOD` |

## 再生（replay）

`orderKey`が業務上の順序、`version`が同じイベントIDの版です。到着順は使いません。

- 同じイベントIDでは**最も高いversion**を採用し、古い版は`stale`へ残す
- 締め済み期間に届いたイベントは`latePolicy`に従い、`late`（当月で調整）または`held`（保留）へ分ける
- 締め済み期間の合計は変えない。`adjust-current`は開いている期間の合計へ加える
- 到着順を並べ替えても、合計と`digest`は同じ

## 締め時点の記録

`CloseDigest`は締め時点の伝票（ID・期間・業務日付・金額・状態）をID順にハッシュします。締め後に遅延到着を処理しても、締め時点の記録から再出力したdigestは変わりません。締め済みの値を無断で書き換えないことを、この照合値で確かめられます。

## ADRで決める点（本参照の外）

- 期間の状態と確定行の不変性を、コアの意味モデルに持つか、プラグインの範囲に置くか（ADR-0024・0025との関係）
- 期間の再開を許すか、許す場合の権限・監査
- 締め処理のトランザクション境界と、締め時点の記録の保存先

これらは [#2539](https://github.com/hat47x/tei-platform/issues/2539) ではなく、[#2540](https://github.com/hat47x/tei-platform/issues/2540)（本Issueから移設）で扱います。

## 範囲と非主張

- 合成入力に対する静的な判定です。実DB・実帳簿・実Actionの保存・締めを観測した証拠ではありません
- 合成データや同じ機械の上のDBでの成立を、本番環境や異種DBでの成立の証拠にしません
- 電子帳簿保存法などの制度への適合、汎用のイベントソーシング、DBトリガーを主張しません
- サンプル固有の表・トリガーをコアへ移しません
- `-race`はcgoを伴わない可搬Goでは実行していません
