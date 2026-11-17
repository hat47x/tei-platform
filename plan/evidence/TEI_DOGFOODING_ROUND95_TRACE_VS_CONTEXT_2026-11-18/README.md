# trace と knowledge-context が、同じ capability で違う本数を返す（2026-11-18、第95ラウンド）

- 起票: [#5426](https://github.com/hat47x/tei-platform/issues/5426)
- 前段: [第94ラウンド](../TEI_DOGFOODING_ROUND94_VIEW_ARGFOR_FAILURE_REASONS_2026-11-17/README.md)
- 状態: **起票のみ。私は修正しない（辺の同一性の契約判断のため）。**

## 1. 目的

目的文の「再びドッグフーディングに戻る」を実行した。**設計の問い面**（`trace`・`knowledge-context`）で「この能力はなぜ在り、何に支えられているか」を訊く。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `3dff6131c`（`origin/main`） |
| CLI実行物 | `sha256:243bcd2bcedba2d8395b17da46d7cda75c6dfcd802d9cec602c5931abf37f316` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.26.11 linux/amd64` |

## 3. 動いたこと（先に確認）

| 問い | 結果 |
| --- | --- |
| `trace` で能力の出典を訊く | `artifactPath: semantic/equip/desk.tei`、行・列、artifact fingerprint |
| `trace -mode full` | `rows` に辺が並ぶ（`relation: input` / `outcome`） |
| 存在しない主体 | `SUBJECT_NOT_FOUND` |
| `knowledge-context` | `outgoing`・`references`・`authorityLimits` 4件 |

**どちらも読める。** 問いは成立している。

## 4. 見つけた差: 同じ主体で本数が違う

```text
capability equip.loan_register {
    input loan equip.loan
    outcome created equip.loan        # 2つの outcome が同じ payload
    outcome overlapping equip.loan
    effect create equip.loan
}
```

| コマンド | 返り値 |
| --- | --- |
| `trace -mode full` | `total=2`（rows 2） |
| `knowledge-context` | `outgoing=3` |

## 5. 切り分け実験

| 宣言 | trace total | context outgoing |
| --- | --- | --- |
| outcome 2つ（同じ payload） | **2** | **3** |
| outcome 1つ（同じ payload） | 2 | 2 |

**outcome を1つ減らしても trace の本数は変わらない。** trace が**同じ payload を指す2つの outcome を1本に潰している**ことが確定した。

## 6. 原因（コードで確認）

`internal/modelref/derivedbridge/projection.go:118`:

```go
// The identity of an edge is what a caller can see: the endpoints and
// the relation. A repeat is the same statement, not a second one.
identity := fromKind + "\x00" + fromID + "\x00" + relation + "\x00" + toKind + "\x00" + toID
if edgeSeen[identity] { return }
```

**辺の同一性が (from, relation, to) だけで決まる。** `internal/index/index.go:339` は outcome ごとに依存を積むので index は3本、trace は2本。

## 7. 何が失われるか

`inspect` は outcome 名を保っている（`['created', 'overlapping']`）。**trace では、その2つが別々の業務結果であることが読めない。** 呼び手は `total=2` を「outcome は1つ」と受け取る。

`knowledge-context` の `authorityLimits` にもこの差の記述はない。

## 8. なぜ私が直さないか

| 案 | 内容 | 判断 |
| --- | --- | --- |
| A | 同一性キーに outcome 名を含める | **正当な宣言の表現力を変える** |
| B | 「payload が同じ outcome は1本に潰す」と明文化する | 挙動不変。**最小で安全**だが文言と置き場所は契約判断 |
| C | 辺に代表件数を持たせる | スキーマ拡張 |

**A も C も採らない。** B が妥当に見えるが、**明文化は契約の変更**なので所有者判断として起票した。

**テストで固定されていない**（outcome の重複除去を検査するテストは0件）。**現在の挙動は誰も固定していない。**

## 9. 読み取れないこと（NOT RUN）

- A / B / C の選択（#5426）。
- 実DB への接続、実 Provider 起動。
- 実Host、HTTP・Desktop。

## 10. 位置付け

**「2つのコマンドが同じ対象について違う数を言う」を見つけた回**である。どちらも正しく、**辺の同一性の定義が違う**だけだった。**私は数を合わせるのではなく、どちらが何を数えているかを確定させて起票した。**
