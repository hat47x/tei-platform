# 番号の桁数を綴る helper が1桁しか扱えず、検査が静かに止まっていた（2026-11-09、第34ラウンド）

- 対象: [#3919](https://github.com/hat47x/tei-platform/issues/3919)（第33ラウンドで起票・クローズ済み。本ラウンドは残した所見の解消）。
- 実行日: 2026-11-09（Asia/Tokyo）。
- 状態: **第33ラウンドで「変更していない」と記録した欠陥を、範囲を確認したうえで修正。**

## 1. 目的

第33ラウンドで、`tools/repository-plan-check/plancheck_test.go` の `itoa` が1桁しか変換しないことを記録し、「影響範囲が読めないので変更しない」とした。これを閉じる。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `ddb566c63`（`origin/main`） |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

## 3. 観測

### 3.1 呼出し元は1箇所だけだった

```go
func numberedFiles(t *testing.T, dir string, digits int) map[string][]string {
	pattern := regexp.MustCompile(`^([0-9]{` + itoa(digits) + `})-.+\.md$`)
```

`numberedFiles` の呼出しは3つで、渡す `digits` は **4 と 6**。

| 呼出し | digits |
| --- | --- |
| `plan/issue` の Issue 番号 | 6 |
| `plan/adr` の ADR 番号 | 4 |

### 3.2 実測: 4 と 6 は偶然正しく動く

```text
itoa(0)  = "0"
itoa(4)  = "4"
itoa(6)  = "6"
itoa(9)  = "9"
itoa(10) = ":"
itoa(11) = ";"
itoa(23) = "G"
itoa(514)= "Ȳ"
```

`string(rune('0' + n))` なので 0〜9 は正しく、10 以上は記号や別の文字になる。**現状の動作は正しい。バグは潜在している。**

### 3.3 第33ラウンドの私の記述は誤っていた

第33ラウンドでは「呼出し元は ADR 番号や Issue 番号（1〜3桁）を渡している」と書いた。**誤り。** 実際に渡しているのは**桁数**（4・6）であり、番号そのものではない。

この誤りのせいで、私は「3桁で化ける」と見当違いの危険度評価をしていた。実際は「10桁以上を要求したとき」に壊れる。

### 3.4 壊れ方は「静かに止まる」

`digits = 10` のとき:

```text
itoa(10) = ":"
pattern  = ^([0-9]{:})-.+\.md$
```

`regexp.MustCompile` は**panic しない**（`{:}` は「`:` の1回」を意味する有効な数量子）。コンパイルは成功し、**どのファイルにも一致しなくなる**。

```text
matches 1234567890-x.md: false
matches 1234-x.md:       false
```

つまり、桁数を10以上に変えた瞬間、`assertUnique` は空の集合に対して「重複なし」と報告する。**検査が落ちるのではなく、検査が無くなる。**

## 4. 修正

```go
func itoa(n int) string {
	return strconv.Itoa(n)
}
```

`go.mod` に新しい依存は不要（標準ライブラリ）。

## 5. 追加した試験

`tools/repository-plan-check/itoa_test.go`。

| 試験 | 内容 |
| --- | --- |
| `TestItoaSpellsMultiDigitCounts` | `itoa(0/4/6/9/10/23/514)` が `"0"/"4"/"6"/"9"/"10"/"23"/"514"` を返す |
| `TestDigitCountPatternStillMatches` | `digits` が 1・2・4・6・10・12 のいずれでも、その桁数ちょうどの名前を作って正規表現が一致すること。かつ1桁の名前を誤って受理しないこと |

### 5.1 変異試験

修正を旧実装へ戻すと、両方の試験が落ちる。

```text
--- FAIL: TestItoaSpellsMultiDigitCounts
    itoa_test.go:30: itoa(10) = ":", want "10"
--- FAIL: TestDigitCountPatternStillMatches
    itoa_test.go:47: digits=10 pattern "^([0-9]{:})-.+\\.md$" does not match "7777777777-entry.md"
```

## 6. なぜ第33ラウンドで直さなかったか、直せた理由

第33ラウンドでは「既存試験の挙動を変えると影響範囲が読めない」として保留した。本ラウンドで**呼出し元が1箇所、渡す値が4と6だけ**と確認できたので、安全に修正できた。

**保留の判断自体は妥当だったが、記述（「ADR 番号や Issue 番号を渡している」）が事実と違っていた。** 呼出し元を数えずに書いた。

## 7. 読み取れないこと（NOT RUN）

- `numberedFiles` に 10 以上の桁数を渡す予定があるか（現状は無いが、Issue 番号が7桁になれば 6 のままで足りない別の問題になる）。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 8. 位置付け

「検査が静かに無くなる」型の欠陥を閉じた。`regexp.MustCompile` が panic しないため、壊れた数量子でも**テストは緑のまま通る**。第33ラウンドで見つけた「試験が静かに通る状態」と同族であり、同じラウンドで2回続けて遭遇したことになる。

前ラウンドの記述の誤りも併せて訂正した。
