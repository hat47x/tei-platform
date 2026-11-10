# 計画が、ビルドが実際にコンパイルするpackageを覆っているか（§4.41）

§4.39の案Aは「計画と作業ツリーを照合してから、そのツリーで`go build`する」形にしました。しかし**照合するのは計画が名指したsourceだけ**で、**ビルドが計画に載っていないmodule内packageを引き込んでいないか**は見ていませんでした。

本記録は、その検査を足した結果を残します。**TOCTOUを閉じるものではありません**（それは§4.39の案B）。

## 何をするか

| 段 | 内容 |
| --- | --- |
| 1 | `go list -deps -f '{{if not .Standard}}{{.ImportPath}}\t{{.Dir}}{{end}}'`を、ビルドと同じ隔離環境・同じ供給で走らせる |
| 2 | module内（root配下）の各packageのdirが、計画の`go-package` sourceの**いずれかの配下**にあるかを調べる |
| 3 | **覆われていないpackageがあればビルドしない**。理由にpackageとdirを並べる |

**標準ライブラリは対象外**（`{{if not .Standard}}`）。**外部moduleは`external`として別に記録**します——供給から来るもので、計画のsourceが覆う対象ではないためです。

## 実測

| 試験 | 結果 |
| --- | --- |
| `TestCheckDependencyCoverageRefusesAnUncoveredModulePackage`（`./internal/buildintent`を、無関係なsourceだけを名指す計画でビルド → **覆われていないと判定**） | **PASS** |
| `TestCheckDependencyCoverageAcceptsASelfContainedPackage`（module内依存の無いpackageは自分のsourceで覆われる） | **PASS** |
| `TestRunBuildsFromThePlanAndReportsWhatItDid`（依存の無いprobeで実build） | **PASS** |
| `TestSelfBuildEstablishesL2BFromARealGoBuild`（回帰試験） | **PASS**（16.1秒） |

環境：WSL2 Ubuntu 22.04、`go1.27.3 linux/amd64`。

## この検査が意味すること

**`./cmd/tei`のビルドは、`cmd/tei`だけを名指す計画では通りません。** 実際に試すと、`internal/`配下の30弱のpackageが「覆われていない」と出ます。

**これは正しい。** 計画が`cmd/tei`だけを名指している以上、その計画は「`cmd/tei`をコンパイルする」ことを記述していません。

**したがって、宣言の`sources`はmodule内の閉包を含まなければなりません。** 宣言を書く側の負担は増えますが、**「計画が記述していないビルド」を通さない**という意味を持ちます。

## 言っていないこと

- **TOCTOUは閉じていない。** 照合と`go list`と`go build`の間にファイルは変わり得る
- 外部moduleの完全性。**供給のTrustの範囲**であり、ここでは`external`として並べるだけ
- 覆われていないpackageを自動で計画へ足すこと。**計画は呼び手が書くもの**
- `vendor/`の扱い。**今は対象から外している**

## 検証

- 上の表のとおり
- `gofmt`差分なし／`go vet`：PASS

**NOT RUN**：`darwin`・`arm64`、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.41
- 案A・案B：[同 §4.39](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
