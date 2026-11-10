# 未束縛の datasource に、書くべき bind を名指しさせる（2026-11-11、第64ラウンド）

- 対象: [#4169](https://github.com/hat47x/tei-platform/issues/4169) の A 案、3件目。第60ラウンドから**未起票のまま繰り越していた**摩擦。
- 前段: [第63ラウンド](../TEI_DOGFOODING_ROUND63_CONTRACT_VERSION_2026-11-11/README.md)。
- 実行日: 2026-11-11（Asia/Tokyo）。
- 状態: **未束縛 datasource の警告が、宣言すべき `bind` を名指しする。3件目を解消。**

## 1. 目的

第60ラウンドで `UNBOUND_DATASOURCE` を踏み、「何が足りないかは言うが何を書けばよいかは言わない」と記録した。**正しい解消手順を特定できていないので起票しなかった**（第52ラウンドの誤りの再発を避けるため）。

本ラウンドでついに**正しい手順を確定できた**ので、起票せずにそのまま直す。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `61488b024`（`origin/main`） |
| CLI実行物（修正前） | `sha256:119dd0e6d242f1204fb52e07767dbfa90cf7307b0ce6c81eeb4352fadb7fdb2a` |
| CLI実行物（修正後） | `sha256:56c7288360b6af823a625fcc6a02e18c82549deb1f828379f5e61648ad012462` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

## 3. 正しい手順の確定（これが本ラウンドの核心）

警告は再現できた。**解消手順は、モデル自身の `.tei` を読んで確定した**（道具のソースではなく設計対象）。`environments/development/environment.tei` に:

```tei
database-instance sales_core.dev {
    definition sales_core
    provider postgres
}

bind order_entry.sales -> sales_core.dev
```

**`bind <application>.<alias> -> <database-instance>` を environment ブロックの中に書く。** 実測で確認した:

| 操作 | 結果 |
| --- | --- |
| `bind task67.sales -> sales_core.dev` を environment ブロック**内**に追加 | **警告が消える（`ok`）** |
| その行を削る | 警告が戻る |

### 3.1 途中で自分が踏んだ誤り

最初、`environment.tei` の**閉じ括弧の後ろ**に `bind` を追記した。結果は `PARSE_ERROR unexpected trailing token "bind"`。**置き場所を間違えると、構文エラーになる。** ブロック内に移すと通った。

（これは #4169 の「配置」問題の一例でもある。本ラウンドでは記録に留めた。）

## 4. 修正

警告のメッセージに、**World から導出した**案内を添える。

```go
"query %s datasource %s is not bound in any environment; %s",
q.ID, q.Datasource, unboundDatasourceHint(w, a, q.Datasource)
```

`unboundDatasourceHint` は:

1. application の当該 alias の `TargetDatabase` を引く
2. environment の `Instances` から**定義が一致するもの**を探す
3. 見つかれば `add "bind <app>.<alias> -> <実際のインスタンス>" inside environment <env>, which already declares that instance`
4. 見つからなければ `add "bind <app>.<alias> -> <database-instance>" inside an environment block`

**実在しないインスタンスを例示しない**（fail-closed）。別データベースのインスタンスを借りて案内すると、そのとおりに書いた利用者が**間違った束縛を作る**。

### 4.1 修正後の実測

```text
warning UNBOUND_DATASOURCE  query task67.q datasource sales is not bound in any environment;
  add "bind task67.sales -> sales_core.dev" inside environment development, which already declares that instance
```

**案内のとおりに1行足すと `ok` になる。** これを CLI で実演した。

## 5. 試験

| 試験 | 内容 |
| --- | --- |
| `TestUnboundDatasourceNamesTheBindingToAdd` | 宣言されたインスタンスが無いとき、`bind <app>.<alias>` と `<database-instance>` の形を示す |
| `TestUnboundDatasourcePrefersADeclaredInstance` | 対象データベースのインスタンスが宣言済みなら**その実名**を出す |
| `TestUnboundDatasourceDoesNotBorrowAnotherDatabasesInstance` | **別データベースのインスタンスを案内しない**（fail-closed） |

### 5.1 変異試験

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | 旧メッセージに戻す | FAIL `message does not contain "bind returns_intake.sales"` |
| M2 | 対象データベースの一致を無視して任意のインスタンスを案内 | FAIL `message names another database's instance: ... master_data.dev` |
| M3 | 常にプレースホルダを使う | FAIL `message does not name the declared instance` |

**M2 が最初は失敗しなかった。** 最初の変異は `continue` を消してビルドが壊れた（試験以前の問題）。書き直した変異（`_ = target` で一致判定を実質無効化）で初めて**意図した挙動の変異**として落ちた。第61・63ラウンドと同じく、**変異が壊れていることに気づいて直した**。

## 6. 回帰確認

| 対象 | 結果 |
| --- | --- |
| `./internal/validation` | ok |
| `./cmd/tei` | ok（310.7s） |
| `go vet` / `gofmt -l .` | 指摘なし / 空 |

## 7. #4169 の進捗

| ラウンド | 直したもの |
| --- | --- |
| 61 | パターン種を application 本体に書いたとき、配置を名指し |
| 63 | 契約ヘッダの版欠落を名指し（`^` も含む） |
| **64（本ラウンド）** | **未束縛 datasource に、書くべき `bind` を名指し** |

A 案は3件。**いずれも実際に踏んだ誤りから作られている。**

## 8. 読み取れないこと（NOT RUN）

- `bind` の置き場所（ブロック内であること）を診断が教えるか（今回は構文エラーで気づいた）。
- #4174（言語混在）の案の採否、#4128 の A・B・D。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 9. 位置付け

**「起票しない」判断が正しかったことを示した回**である。第60ラウンドでは正しい手順を知らなかったので起票しなかった。本ラウンドでモデルを読んで手順を確定し、**起票せずに直接直した**。第52ラウンドの誤り（壊れた探査から一般化して起票）を避けられた。

同時に、**案内文が実在するインスタンスを名指すべき**という原則を fail-closed で実装し、変異で固定した。誤った案内は、誤った設計より悪い。
