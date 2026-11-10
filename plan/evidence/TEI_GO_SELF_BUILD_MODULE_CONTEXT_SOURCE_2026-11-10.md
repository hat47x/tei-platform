# module contextも計画のsourceにする（§4.43）

§4.42の`-isolate`は、**`go.mod`・`go.sum`を作業ツリーから取る**ので、隔離してもTOCTOUが残りました。本記録は、**module contextも計画のsourceにする**ことで、その例外を消したことを残します。

## 規則

**`kind: "go-module"`のsourceとして`go.mod`・`go.sum`を宣言できる。** 宣言すれば、隔離ビルドはそれらを**他のsourceと同じように写し、同じようにdigestを照合**します。

| 計画がmodule contextを名指すか | `moduleFiles` | 何が言えるか |
| --- | --- | --- |
| **名指す** | **空** | **木は計画のsourceだけでできている** |
| 名指さない | `go.mod:<digest>`など | 木は計画＋作業ツリーのmodule contextでできている（境界に明記） |

**「例外は無い」と言えるのは、名指した場合だけです。**

## 実測

| 試験 | 結果 |
| --- | --- |
| `TestIsolatedTreeTakesNoModuleFileFromTheWorkingTreeWhenThePlanNamesThem`（`moduleFiles`が**空**、写したmembersは**3件**＝package＋`go.mod`＋`go.sum`） | **PASS** |
| `go test ./tools/reference-go-build-from-plan -count=1`（全試験） | **PASS**（79.4秒） |

環境：WSL2 Ubuntu 22.04、`go1.27.3 linux/amd64`。

## 途中で直した欠陥

**単一ファイルのsourceを写す処理が、そのパスをディレクトリとして作っていました。** ディレクトリのsourceしか写していなかったので表面化していませんでしたが、`go.mod`をsourceにした瞬間に`is a directory`で落ちました。**ファイルとディレクトリを分けて書く**ように直しました。

## 言っていないこと

- **`go-module` sourceの検証**（`go.mod`という名前に限るか、任意のmoduleファイルを許すか）。今は**kindの意味だけ**を決め、名前は呼び手に任せている
- `go.sum`が無いmoduleでの扱い
- `-isolate`の既定化（module contextを名指す計画が一般的になってから）
- `replace`・`vendor/`の扱い

## 検証

- 上の表のとおり
- `gofmt`差分なし／`go vet`：PASS

**NOT RUN**：`darwin`・`arm64`、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.43
- 隔離ビルド：[同 §4.42](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
