# `tei validate -h` が受け付けない70個のflagを並べていた（修正）

- 記録日: 2026-11-10
- 対象: `cmd/tei/main.go`、`cmd/tei/cli_commands.go`
- 関連: #3943、#3944（`-qualification-requirements` の help 文言）、#3185

## 観測（修正前）

`-h` は「このコマンドの文法」を答える要求ですが、一部のコマンドは**共有 FlagSet の全flag**を並べていました。

| 呼出し | `-h` が出す flag 数 | 実際に受け付ける数 |
| --- | ---: | ---: |
| `tei validate -h` | **70** | 3（`-workspace`・`-world`・`-json`） |
| `tei world -h` | **70** | 2（`-workspace`・`-world`） |
| `tei dbcheck -h` | **70** | 1（`-dsn`） |
| `tei inspect -h` | 3 | 3（正しい） |

`validate` の help に `-action`・`-advance`・`-grant-capability`・`-qualification-requirements` などが出ますが、いずれも `INVALID_ARGUMENT` で拒否されます。

```text
tei validate -workspace <ws> -json -grant-capability x
# exit 2, error.code = INVALID_ARGUMENT
```

## 原因

`helpText` は「allowlist にあるコマンドはそのflagだけを書く」設計で、`inspect`・`trace` は正しく絞られていました。しかし `-h` は **`fs.Parse` の中で Go の flag パッケージが処理**します。

- `inspect`・`trace` などは `flag.ContinueOnError` で解析するため `flag.ErrHelp` が戻り、`printHelp` に届く。
- `validate`・`world`・`dbcheck` は `flag.ExitOnError` で解析するため、flag パッケージが**自前の usage（共有 FlagSet 全体）を出して `os.Exit`** し、`printHelp` に届かない。

さらに `dbcheck` は allowlist にも catalogue にも無いため、届いたとしても 0 flag になっていました。

## 変更

- `requestedHelp(args)` を追加し、`-h`・`-help`・`--help` を**flag 位置でのみ**認識する（`--` 以降の値は対象外）。
- `ExitOnError` で解析するコマンドでは、**`fs.Parse` より前に** `printHelp` を呼んで返す。これで全コマンドが同じ絞り込み経路を通る。
- `helpText` に `dbcheck` の分岐を追加し、読む唯一のflag `-dsn` を書く（catalogue に出さない方針は変えない。allowlist へ足すと「広告されていないコマンドが flag 一覧を持つ」として既存試験が落ちるため、`model catalog`・`model locate`・`self-build` と同じ扱いにした）。

## 検証

- `gofmt -l cmd`: 空、`go build ./...`: PASS
- `go test ./cmd/tei -count=1 -timeout 30m`: PASS
- 実測（修正後の実バイナリ、全20コマンド＋非広告3コマンド）:

| コマンド | `-h` の flag 数 |
| --- | ---: |
| `validate` | 3 |
| `world` | 2 |
| `dbcheck` | 1 |
| `inspect` | 3 |
| `trace` | 9 |
| `view` | 15 |
| `self-build` | 20 |

  いずれも「そのコマンドが読む flag」と一致し、共有 FlagSet の他コマンドの flag は出ない。
- 追加試験 `cmd/tei/help_grammar_test.go`:
  - `validate`・`world`・`requirements`・`dbcheck` の help が**受け付ける flag だけ**を出し、`-pattern`・`-advance`・`-grant-capability`・`-qualification-requirements`・`-period-states` を出さないこと。
  - `requestedHelp` が flag 位置の `-h` だけを認識し、`--` 以降や値としての `-h` を認識しないこと。
- 負例: `requestedHelp` の分岐を無効化すると `tei validate -h` は **70 flag** に戻る（コードは戻した）。

## 主張しないこと

- `validate`・`world`・`dbcheck` の**実行時の受理範囲**は変えていない（従来どおり `INVALID_ARGUMENT` で拒否する）。help の表示だけを直した。
- `dbcheck` を catalogue へ広告していない（read-only 選択集合の方針は不変）。
- 実 Host・認可・外部送出は NOT RUN。
