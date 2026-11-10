# 隔離木が再現できないmodule contextは、ビルドしない（§4.48）

§4.43でmodule contextが計画のsourceになりました。しかし**計画が名指していても、その中身が隔離木では再現できない場合**があります。本記録は、その二つを**拒否**することを決めた結果です。

## 何を拒否するか

| module contextの中身 | なぜ再現できないか |
| --- | --- |
| **`replace`**（`replace x => ../x`） | 置換先は計画のsourceではない。隔離木にそのパスは無く、あっても「計画が名指したもの」ではない |
| **`vendor/`** | 依存がmoduleの隣に置かれている。隔離木は`vendor/`を写さないので、ビルドは供給の側から取ることになる |

**どちらも「ビルドは通るが、計画が記述したビルドではない」**種類の食い違いで、§4.41・§4.46と同じ族に属します。

## 実測

| 試験 | 結果 |
| --- | --- |
| `TestCheckModuleContextRefusesAReplaceDirective` | **PASS** |
| `TestCheckModuleContextRefusesAReplaceBlock` | **PASS** |
| `TestCheckModuleContextRefusesAVendorDirectory` | **PASS** |
| `TestCheckModuleContextAcceptsAPlainModule`（`require`だけなら通る） | **PASS** |
| `TestCheckModuleContextIgnoresAMissingModuleFile` | **PASS** |
| 既存の隔離試験・回帰試験 | **PASS**（29.8秒／27.5秒） |

環境：WSL2 Ubuntu 22.04、`go1.27.3 linux/amd64`。

## TEIのmodule contextはこの規則を通る

**`go.mod`に`replace`は無く、`vendor/`もありません。** したがってこの規則は、TEI自身を対象にした隔離ビルドを妨げません。**依存のある対象でも、`require`だけで解決できるなら隔離ビルドできる道が残ります。**

（依存のある対象を実際に隔離木でビルドすることは、**まだ測っていません**。）

## 言っていないこと

- **依存のある対象の隔離ビルドの実測**。今回拒否したのは「再現できない形」であり、残る「再現できる形」を試したわけではない
- `replace`を**計画で名指せるようにする**案（置換先をsourceとして宣言できれば再現できる）
- `vendor/`を計画に含める案

## 検証

- 上の表のとおり
- `gofmt`差分なし／`go vet`：PASS

**NOT RUN**：`darwin`・`arm64`、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.48
- module context：[同 §4.43](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
- 隔離の既定化：[同 §4.47](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
