# 契約ヘッダの版を名指しする（2026-11-11、第63ラウンド）

- 対象: [#4169](https://github.com/hat47x/tei-platform/issues/4169) の A 案、2件目。
- 前段: [第62ラウンド](../TEI_DOGFOODING_ROUND62_LANGUAGE_MIX_2026-11-11/README.md)で「次の対象は `unexpected token "{"`」と記録した。
- 実行日: 2026-11-11（Asia/Tokyo）。
- 状態: **契約ヘッダの版欠落を名指しするようにした。実装中に自分のバグを1つ作り、試験で捕まえた。**

## 1. 目的

第62ラウンドで契約クエリを設計したとき、`contract sales_core.probe { ... }` が `unexpected token "{"` になり、何が足りないか分からなかった。この1点を直す。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `c3755316e`（`origin/main`） |
| CLI実行物（修正前） | `sha256:d68e96ca3ea2c143abb846ab5d05c9f50215f1fcc829d281f9a2ae9b730c05b6` |
| CLI実行物（修正後） | `sha256:562d713e86b2500f5da7b0510c2a13dd4dfafd1d05f7fbfa80c0f1196bed8f84` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

## 3. 修正前の実測

```text
contract sales_core.probe {      -> unexpected token "{"     <- 何が要るか言わない
contract sales_core.probe ^1 {   -> unexpected token "^"
contract sales_core.probe version 1 { -> invalid version "version"   <- こちらは良い
contract sales_core.probe v1 {   -> ok
```

## 4. 修正

`ParseContract` のヘッダで、版の**前**に次のトークンを見る。

```go
if next, err := p.peek(); err == nil && next.Kind != TokenIdent {
	return database.Contract{}, p.errAt(next, fmt.Sprintf(
		"expected a version after the contract id %q: write it as \"contract %s v4 { ... }\" (the id and the version are separate tokens)",
		idTok.Text, idTok.Text))
}
```

### 4.1 修正後

```text
no version      -> expected a version after the contract id "sales_core.probe":
                   write it as "contract sales_core.probe v4 { ... }"
                   (the id and the version are separate tokens)
caret version   -> 同上
keyword version -> invalid version "version"        （変わらず）
correct v1      -> ok
```

**対照（変わらないこと）**: `contract {` は `unexpected token "{"`、本体の未知語は `expected read, write, or query`、`read sales.order` は `ok`。

## 5. 実装中に作った自分のバグ

**最初の版は `p.expect(TokenIdent)` を呼んだ後で `p.peek()` していた。** `expect` は失敗時にトークンを消費するので、peek は次のトークンを見てしまい、条件が成立しない。**ビルドは通り、既存の試験も通った。**

CLI で測ったら**修正前と同じ `unexpected token "{"` が返り**、気づいた。

**現在の版は `expect` の前に peek する。** 変異 M2 がこの順序バグを再現する形になっており、試験が捕捉する。

```text
M2: expect() の後に判定を置く -> FAIL message does not ask for a version
```

**この誤りは「テストが通った」だけでは見つからない。実物を走らせて初めて分かった。**

## 6. 試験と変異

| 試験 | 内容 |
| --- | --- |
| `TestContractWithoutVersionNamesTheShape` | `{` と `^` の両方が「版が要る」と言い、正しい形を示し、汎用メッセージでない |
| `TestContractWithAnInvalidVersionTokenKeepsItsMessage` | **版が在るが不正な場合は「欠落」と報告しない**（別の誤りなので混同しない） |
| `TestContractHeaderControlsAreUnchanged` | `contract {` と本体の未知語は変わらない |
| `TestContractMissingVersionRangePointsAtTheBrace` | 範囲が `{` を指す（id ではない） |

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | 版の判定を削る | FAIL `message does not ask for a version` |
| M2 | 判定を `expect` の後に置く（**実際に犯したバグ**） | FAIL 同旨 |
| M3 | 版が在っても「欠落」と報告する | FAIL `a present-but-invalid version must not be reported as missing` |

## 7. 回帰確認

| 対象 | 結果 |
| --- | --- |
| `./internal/syntax` | ok |
| `./internal/workspace` | ok |
| `./cmd/tei` | ok（302.8s） |
| `go vet` / `gofmt -l .` | 指摘なし / 空 |

## 8. #4169 の残り

A 案で直したのは2件（第61ラウンドのパターン配置、本ラウンドの版欠落）。

第62ラウンドで弱いと判定したもののうち、**`unexpected token "^"` は本ラウンドで一緒に直った**（版の位置で `^` を書いた場合）。残る弱い診断は、まだ踏んでいない層にある。

## 9. 読み取れないこと（NOT RUN）

- #4169 の A 案の他ケース（踏んでいない配置の誤りは特定できていない）。
- #4174 の案の採否。`UNBOUND_DATASOURCE` の解消手順。#4128 の A・B・D。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 10. 位置付け

**「実物を走らせないと分からない誤り」を1つ記録した回**である。peek の順序バグはビルドも既存試験も通った。CLI の出力が修正前と同じだったことで気づいた。**前ラウンドで「打ち込みではなく実物で測る」と決めた理由が、ここで実証された。**
