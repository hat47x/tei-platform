# 契約クエリを CLI で設計し、診断の言語混在を検出（2026-11-11、第62ラウンド）

- 対象: [#4174](https://github.com/hat47x/tei-platform/issues/4174)（起票）、[#4169](https://github.com/hat47x/tei-platform/issues/4169)（A 案の材料集め）。
- 前段: [第61ラウンド](../TEI_DOGFOODING_ROUND61_SYNTAX_PLACEMENT_2026-11-11/README.md)。
- 実行日: 2026-11-11（Asia/Tokyo）。
- 状態: **契約クエリを CLI だけで設計。診断の言語混在を検出して起票。**

## 1. 目的

#4169 の A 案（配置の誤りを1件ずつ名指しする）を進めるため、**まだ設計したことのない層**を実際に設計し、踏む誤りを集める。database contract query を選んだ。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `a16c0f84a`（`origin/main`） |
| CLI実行物 | `sha256:e1acb51df0a0f015d80f79b7b98da4f81fdf3108852baeaab7446a9550af5301` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

## 3. 設計課題（着手前に宣言）

> `sales_core` に契約クエリを追加する。ある顧客の未処理注文を返すクエリを、アプリケーションから参照できる形で宣言する。

## 4. 経過（4往復）

| Attempt | 書いたもの | 応答 | 学んだこと |
| --- | --- | --- | --- |
| 1 | `query <id> { ... }` を contracts 配下に | `expected "contract"` | 先頭は `contract` |
| 2 | `contract <id> { ... }` | `unexpected token "{"` | ID の後に版が要る |
| 3a | `contract <id> v1 { }` | `ok` | **版は裸のトークン（`v1`）** |
| 3b | `contract <id> ^1 { }` | `unexpected token "^"` | `^` は requirement 側だけ |
| 3c | `contract <id> version 1 { }` | `invalid version "version"` | 版の位置に来た語は版として解釈される |
| 4 | `query <name> { sql ... }` を中に | `Query signatureではinputまたはresultを指定してください` | `input` か `result` が要る |

**4往復で契約クエリの骨格に到達。** 3c の `invalid version "version"` は**良い診断**で、語の位置から期待を的確に伝えている。

## 5. 検出: 診断の言語が混在している

Attempt 4 のメッセージだけが**日本語**だった。同じ `PARSE_ERROR` クラスの中で、英語と日本語が混在している。

```text
databases/sales-core/contracts/open_orders_by_customer.tei:3:9
  error PARSE_ERROR Query signatureではinputまたはresultを指定してください
```

同じクラスの英語の例（第61ラウンドで追加）:

```text
applications/probe/application.tei:4:5
  error PARSE_ERROR "browse" starts an application pattern, which is not declared
  in the application body; declare it as "pattern <application>.<id> browse { ... }"
  in its own file
```

### 5.1 範囲の実測

非ASCII の文字列リテラル（テスト除く）は **66ファイル / 555件**。

| パッケージ | 件数 |
| --- | --- |
| `internal/driver` | 214 |
| `internal/validation` | 99 |
| `internal/security` | 66 |
| `internal/database` | 50 |
| `internal/applicationview` | 42 |
| `internal/realization` | 33 |
| `internal/syntax` | 17 |
| `internal/semantic` | 15 |
| `cmd/tei` | 10 |
| `internal/workspace` | 6 |
| `internal/execution` | 3 |

`validation`（99）と `syntax`（17）は、**そのまま `validate` の診断文になる層**である。4層（`syntax`・`validation`・`workspace`・`semantic`）で **137件**。

### 5.2 正確に書いておく限界

- **1回の実行で両方を同時に観測してはいない。** 別々の実行で観測した（`validate` は最初の構文エラーで止まる）。
- 555件のうち**どれが CLI に出るかは対応付けていない。** `driver`・`security` の多くは内部エラーで、外に出るとしても包まれている可能性がある。
- **「日本語であること」自体が欠陥かは方針の問題**である。オーナーが日本語話者であることは承知しており、全訳を私が決めるべきではない。

問題は言語の選択ではなく**統一されていないこと**なので、[#4174](https://github.com/hat47x/tei-platform/issues/4174) として案を5つ示して起票した。**A（利用者に出る診断のみ英語、対象は4層の137件）が最小**と書いた。

**私は 555件の変換を行っていない。** 方針判断だからである。

## 6. #4169 への材料

今回踏んだ4つの誤りのうち、**3つは既に良い診断**だった。

| 誤り | 診断 | 評価 |
| --- | --- | --- |
| `contract` を書かない | `expected "contract"` | 妥当（語を名指し） |
| 版を書かない | `unexpected token "{"` | **弱い**（何が要るか言わない） |
| `^1` を使う | `unexpected token "^"` | 弱い |
| `version 1` と書く | `invalid version "version"` | **良い** |
| `input`/`result` 無し | 日本語で「inputまたはresultを指定」 | 内容は妥当、言語が違う |

`unexpected token "{"` が最も弱い。**「版が要る」と言えば1往復減る。** #4169 の A 案の次の対象として妥当だと判断した（本ラウンドでは実装せず、材料として記録）。

## 7. 読み取れないこと（NOT RUN）

- 555件のうち CLI に出るものの対応付け。
- #4174 の案の採否（方針判断）。
- #4169 の A 案の実装（`unexpected token "{"` の改善を含む）。
- #4128 の A・B・D。`UNBOUND_DATASOURCE` の解消手順。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 8. 位置付け

**測ってから起票した回**である。言語混在は最初「1件のメッセージが日本語」に見えたが、測ると 555件・66ファイルの広がりがあった。同時に「1回の実行で両言語を観測したわけではない」「どこが CLI に出るかは未対応付け」という限界も確定できた。

**方針判断を自分で決めずに起票で止めた**のは意図的である。555件の一括変換は不可逆性が高く、オーナーの言語選択に関わる。
