# コマンドパターンを CLI で設計し、interaction の形を名指しさせる（2026-11-11、第65ラウンド）

- 対象: [#4169](https://github.com/hat47x/tei-platform/issues/4169) の A 案、4件目。
- 前段: [第64ラウンド](../TEI_DOGFOODING_ROUND64_UNBOUND_DATASOURCE_2026-11-11/README.md)。
- 実行日: 2026-11-11（Asia/Tokyo）。
- 状態: **コマンドパターンを設計し、`interaction` の形を名指しするようにした。設計した成果物は CLI で読み返せた。**

## 1. 目的

目的文の「再びドッグフーディングに戻る」を実行する。まだ触っていない層として **command パターン**を選び、踏む誤りを集める。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `e615163f9`（`origin/main`） |
| CLI実行物（設計時） | `sha256:fa5f01edf7315563ebb10ca55ef363ee0321c847c3ce282c55be3fab2e3d1e5e` |
| CLI実行物（修正後） | `sha256:55e4107acd1b665401fa461fba84925c67277e9b24508321714e36078da67312` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

## 3. 設計課題（着手前に宣言）

> returns スライスに「accept-return」コマンドを足す。注文 ID を取り、返品メモを書く。CLI で読み返す。

## 4. 経過

| Attempt | 応答 | 学んだこと | 情報源 |
| --- | --- | --- | --- |
| 1 | `command pattern ... does not accept data` | command は `data` を取らない | CLI |
| 2A | `UNKNOWN_APPLICATION_PATTERN_CAPABILITY ... unknown semantic capability sales.record_return` | 能力を先に宣言する必要がある | CLI |
| 2B | `unexpected token "command"` | `interaction command "accept"` は通らない | CLI（限界） |
| 2C | `expected data, relation, query, capability, datasource, contract-query, or interaction` | 本体の語彙は引けた | CLI |
| 4 | `expected "namespace"` | 意味ファイルには `namespace` ヘッダが要る | CLI |
| 5 | `ok` | `uses semantic` 済みの名前空間に `capability` を置けた | CLI |
| 6/7 | `ok` | コマンドパターンが解決した | CLI |

**`interaction` の形（行かブロックか）だけは CLI で分からなかった。** 2B の `unexpected token "command"` は「この語は知らない」としか言わない。

## 5. 設計した成果物の読み返し

```text
model catalog: returns.accept_return が application-pattern として現れる
inspect -subject-kind application-pattern -subject-id returns.accept_return
  {"id": "returns.accept_return", "kind": "command", "capability": "sales.record_return"}
```

第58ラウンドで足した `application-pattern` の受理が効いている。

## 6. 修正: `interaction` の形を名指しする

```go
if next, perr := p.peek(); perr == nil && next.Kind != TokenLBrace {
	// パターン種に合った例だけを出す
}
```

### 6.1 二度書き直した

| 版 | 出力 | 問題 |
| --- | --- | --- |
| v1 | `for example "interaction { command command }"` | 違反トークンをそのまま例に埋め込み、**無意味な例**になった |
| v2 | `for example "interaction { command <capability name> }"` | command パターンでは**契約が即座に拒否する例**を出していた。往復が1回増える |
| v3（採用） | record には `"interaction { item <data>.<attribute> }"`、それ以外には「この種は interaction を受けない」 | 種に合った案内 |

**v1 と v2 はどちらも CLI で走らせて初めて気づいた。** v2 は特に、「例を示す」ことが**別の誤りに誘う**という失敗だった。

### 6.2 修正後

```text
record + 行形式  -> expected "{" after interaction: interaction is a block,
                    for example "interaction { item <data>.<attribute> }"
command + 行形式 -> expected "{" after interaction: interaction is a block
                    "interaction { ... }", but this command pattern does not accept one;
                    the first interaction contract is limited to record patterns
```

## 7. 試験と変異

| 試験 | 内容 |
| --- | --- |
| `TestInteractionLineFormNamesTheBlock` | record ではブロック形と例を示す |
| `TestCommandPatternInteractionLineSaysItIsNotAccepted` | **command では「受けられない」と言い、拒否される例を出さない** |
| `TestInteractionBlockFormsAreUnchanged` | ブロック形の挙動不変 |
| `TestInteractionExampleFollowsThePatternKind` | `isRecordPatternKind` が record だけ真 |

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | ブロック形の判定を削る | **ビルドが壊れた**（下記） |
| M2 | 全種に record の例を出す | FAIL `does not say the command pattern rejects an interaction` |
| M3 | 全種を interaction 受容とみなす | FAIL `"browse" must not be treated as accepting an interaction` |

**M1 は最初ビルドを壊して「失敗しなかった」。** 削除の範囲が `default:` の途中まで食い込み、`expected '}', found 'default'` になった。第67ラウンドの M2 と同じ類型で、**変異が壊れていることに気づく必要がある**。

## 8. 回帰確認（正直に書く）

| 実行 | 結果 |
| --- | --- |
| `./internal/syntax`（単独） | ok |
| `./internal/workspace`（単独） | ok |
| `./cmd/tei`（単独） | **ok（564.8s）** |
| `./cmd/tei`（変異作業と同時実行） | **FAIL（600.1s）** |

**単独実行では通る。同時実行では `go test` の600秒上限に達した。** 564.8s は上限に近く、CPU 競合で超えたと考えられる。**これは機能の失敗ではなく実行環境の競合**だが、記録には両方残す。

## 9. #4169 の進捗

A 案は4件（パターン配置・契約の版・bind・interaction の形）。

## 10. 読み取れないこと（NOT RUN）

- 600秒上限に近い `cmd/tei` の実行時間そのもの（別の課題として記録）。
- #4174（言語混在）の案の採否、#4128 の A・B・D。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 11. 位置付け

**案内文を2回書き直した回**である。どちらも「例を示す」ことが別の誤りに誘う失敗で、CLI で走らせて初めて分かった。案内文は正しさだけでなく**その案内に従ったときに通るか**まで確かめる必要がある。

`cmd/tei` が600秒上限に近いことも判明した。単独では通るが、これは別に扱うべき課題として記録した。
