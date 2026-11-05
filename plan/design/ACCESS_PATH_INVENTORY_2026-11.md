# アクセス経路の棚卸し（2026-11）

- 状態: **棚卸しのみ**。強制を実装したものではありません。
- 目的: 権限の強制点をどこに置くかを決めるため、ファイル・ネットワーク・環境変数に触る箇所を数える。
- 関連: [権限の現状調査](PERMISSION_ENFORCEMENT_SURVEY_2026-11.md)
- 数え方: `internal`・`cmd`・`tools` の非試験 `.go` を対象に、`os.ReadFile`・`os.WriteFile`・`os.OpenFile`・`os.Create`・`os.Remove`・`os.MkdirAll`、`net.Dial`・`net.Listen`・`http.Get`・`http.Post`・`http.NewRequest`、`os.Getenv` を数えた（2026-11-06 時点）。

## 1. 件数

| 種類 | 件数 | 主な場所 |
| --- | ---: | --- |
| ファイル | 626 | `tools` 541、`cmd/*` 各数件、`internal` は `sourceacceptance` 8・`opensshwitness` 5・`security` 3 など |
| ネットワーク | 15 | ほぼ `tools`（`reference-order-assist-provider-offline` 4、`reference-order-migration-dual-db` 2 など） |
| 環境変数 | 31 | スイッチと検証用の設定 |

## 2. 分かったこと

1. **実行の中核（`internal/driver`・`internal/execution`・`internal/plugin/runtime`・`internal/servicehost`）が直接触るファイル操作は1件だけ**（`internal/plugin/runtime/process_pinned_linux.go` の一時ファイル作成）。中核は意味と契約を扱い、外部へのアクセスは**アダプタと参照ツール**が持っています。
2. したがって、権限の強制を能力の呼出し直前（`execution.Invoke`）に置いても、**実際のアクセスは素通り**します。アクセスを持つのは `tools` 側のアダプタであり、そこを強制点にしない限り「宣言と実際」はつながりません。
3. `os.Getenv` は 31 件あり、秘密の取得が宣言（`secret`）を通っているかは別に確かめる必要があります。

## 3. 強制点の候補

| 候補 | 利点 | 欠点 |
| --- | --- | --- |
| アダプタ境界（`tools` の各アダプタ） | 実際のアクセスに最も近い | アダプタごとに実装が必要。数が多い |
| 中核の共通アダプタ層（新設） | 一箇所で済む | 既存アダプタの移行が必要。ツールが中核を通らない経路が残る |
| OS のサンドボックス（権限を落として起動） | 宣言と独立に守れる | 実行環境の対応が必要。TEI の外側の仕組み |

**推奨（仮説）**: まず「中核の共通アダプタ層」を一箇所作り、そこを通るアダプタだけを対象にする。通らない経路は**通らないと記録**し、対象を広げる。OS のサンドボックスは別途、実行環境の担当として切り分ける。

## 4. この棚卸しが主張しないこと

- すべての経路を数え切ったこと（正規表現による近似であり、`os` 以外の手段は含まない）。
- 強制の実装方針の決定。
- 現在の権限宣言が実際のアクセスと一致していること。
## 5. 追記（2026-11-06）: 判定と記録の最小実装

`internal/access` に、判定と記録を一箇所にした最小の実装を追加しました。

```go
guard := access.Guard{Granted: grants, Journal: access.File{Path: "access.jsonl"}}
decision, err := guard.Check("sales.close", requirements)
```

- 判定は `permission.Missing`（付与が要求を覆うか）を使う。
- **すべての判定を記録**する（許可も拒否も）。記録できない場合はエラー（記録できない判定を黙って通さない）。
- 記録は JSON Lines。壊れた行はエラーとして報告し、その行より前は返す。

**まだ強制ではありません**。アダプタ（`tools`）がこの `Guard` を通るようになるまで、実際のアクセスは止まりません。通るアダプタを一つずつ増やし、**通らないものは通らないと記録**する方針です。