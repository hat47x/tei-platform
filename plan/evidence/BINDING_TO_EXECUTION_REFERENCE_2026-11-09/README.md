# 宣言の Binding から実行までを、参照実装で繋いでみる（2026-11-09）

- 対象: [判断材料: 実行の側をどこで宣言へ繋ぐか](../../workstreams/COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)の **C-1**（`routing.Router` を組み立てる）と、[環境の配備実行Bindingは、どこまで使われるか](../COMPOSITION_COLOCATION_RESURVEY_2026-11-08/README.md)
- 位置づけ: **参照実装と実行記録**。製品へ繋ぐべきかは示しません。
- 計測した版: `raw/head.txt`（`origin/main` `11dbc19c` の作業ツリー）。

前の判断材料は、**計画と実行の間に宣言を読む繋ぎ目が無い**ことを示し、4案（C-1〜C-4）を並べました。この記録は、そのうち **C-1（`routing.Router` を組み立てる）を実際に作って動かします。**

## 1. 何を作ったか

`tools/reference-binding-to-execution`（新規・読み取り専用）。

- **選択**: `routing.RouteCommand`（環境の `deployment-executor` Binding を使う既存の関数）
- **実行**: `routing.Router.Dispatch`（既存の関数）
- **確認**: `routing.ValidateResult`（結果が Binding の executor から来たかを照合）
- **登録**: Executor の registry は**呼出し側（Host）が渡す**。参照は自分で Executor を組み立てません
- **拒否**: Binding が無い／曖昧／未登録の executor を指す場合は、**実行せずに理由を残す**
- **しない**: 認証・権限発行・永続的な receipt・外部状態の確認

**新しい解決器も、新しい選択の仕組みも作っていません。** 既存の3関数を順に呼ぶだけです。

## 2. 観測

```text
== bound ==
  environment: development
  ran: 1  refused: 0
    action=db/apply executor=migration.atlas bindingResolved=True

== unregistered ==
  environment: development
  ran: 0  refused: 1
    action=app/start executor=runtime.reference bindingResolved=False
    error=EXECUTOR_NOT_REGISTERED: the binding names an executor this Host did not register

== unknown-env ==
  environment: staging
  ran: 0  refused: 0
```

- **Binding が指す executor を登録してあれば、選択から実行まで通り、結果は Binding の executor と照合されます**（`bindingResolved=true`）。
- **Binding が未登録の executor を指す場合は、実行せずに拒否**します。`development` の `application-start` は `runtime.reference` を指しますが、参照は `migration.atlas` だけを登録しているためです。
- **存在しない環境**は 0件で終わり、失敗として扱いません（記録に環境名だけ残ります）。

## 3. この観測が示すこと

- **C-1 は成立します。** 既存の `routing.RouteCommand` と `routing.Router.Dispatch` だけで、Binding から実行まで繋げられます。新しい部品は不要でした。
- **`routing.Router` を組み立てる非試験の箇所が0件**という前の観測は、「組み立てられない」ではなく「**まだ誰も組み立てていない**」ことを示します。この参照は、組み立てられることを実演しています。
- **未登録の executor を指す Binding は、実行前に拒否できます。** `ValidateResult` による結果の照合も含め、Binding と実行の対応を機械で確かめられる形です。
- ただし **Executor の registry は Host が渡す**ため、この参照だけでは「本物の配備」にはなりません。参照が登録するのは記録用の Executor で、外部効果はありません。

## 4. この記録が示さないこと

- **製品 CLI へ繋ぐべきか**。判断材料の C-1〜C-4 は未決です。
- **実配備の成立**。参照の Executor は記録用で、実DB・実クラウド・実移行ツールに触れません。
- **C-2（`servicehost` を入れる）との比較**。C-2 は実装していません。
- **認証・権限・receipt**。この参照は何も発行しません。
- **実行時の並行性**。1コマンドずつ順に実行します。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（3つの場合を実行し、要約する） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域（新規ツールと`aw/`のみ） |
| `raw/bound.json` | Binding が解決した場合 |
| `raw/unregistered.json` | Binding が未登録の executor を指す場合 |
| `raw/unknown-env.json` | 存在しない環境 |
| `raw/unregistered.exit.txt` | 2番目の終了コード |
| `raw/summary.txt` | 上の表を作る元の一覧 |

## 6. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `bc1b0604d1ba2c734462d528a1f49766c4f3ee6c2c90df42d38bb976f9a1d002` | 2867 |
| `raw/head.txt` | `9d9529522fac4a385b0df18206630191f23dfc9ea0e9137fea3369957e99192d` | 73 |
| `raw/pre-status.txt` | `368c4ab11ea3633249a8f87dcc8aac54d655373dc3f2e4cf0b5bee8414d70fda` | 287 |
| `raw/bound.json` | `74579d138254cb38fefab014eddaa436b58f30cb021ddeed91aa600feef41548` | 916 |
| `raw/unregistered.json` | `30d2d61fcfbe95dc1aea517ac28435793b8a797011a4d6097b02cfbd287f7027` | 950 |
| `raw/unknown-env.json` | `a5a6b3c38293344f7fc55740e3a33e8ea7069b8a229f098a9fac8128e78f62b0` | 728 |
| `raw/unregistered.exit.txt` | `4d8dc4050b690ebd914147add79f148db3d3277ce9d9ae39d8f2bd6d09c40ab4` | 20 |
| `raw/summary.txt` | `c5e37ba3d7fa39a8c2411008971ca8cdf4d4702a82447ab51d4570af4ae9097f` | 484 |

規則の指紋は `sha256:329b1d5311a748ffece0c60c1cd723dc6323884fceae35b4ad6c3591a3613c35` です。

`raw/unregistered.exit.txt` の内容は `unregistered exit=0` です。**拒否はエラーではなく値として扱う**ため、この参照は拒否時にも JSON を出して 0 で終わります。拒否の内容は `error` 欄に残ります。
