# 計画がmodule contextを名指すなら、隔離ビルドを既定にする（§4.47）

§4.42は`-isolate`を**既定にしませんでした**。理由は「module contextが計画の外にある」でした。§4.43でそれが計画の内側に入ったので、**条件が変わりました**。

## 規則

| 計画が`go-module`を名指すか | 既定 | 記録 |
| --- | --- | --- |
| **名指す** | **隔離ビルド** | `buildSource=plan-only`、木は計画のsourceだけ |
| 名指さない | 作業ツリーを照合してビルド | `buildSource=working-tree` |

**`-isolate=false`で明示的に断れます。** 明示された指定は既定に優先します。

## なぜ「条件付きの既定」なのか

**無条件に既定にすると、module contextを持たない計画で「隔離した」と読まれます。** 実際には`go.mod`を作業ツリーから取ることになり、§4.42が避けたかった読み違えそのものです。**計画が名指すときだけ既定にする**ことで、既定の意味が「計画だけでビルドした」に固定されます。

## 五条件は変えない

**隔離を6番目の条件にはしません。** 条件を増やすのは§4.23の述語の変更であり、**隔離できないbuilderを一律に排除する**（§4.38の表が示すとおり、childの能力に依存する）。**既定を強くする**ことで通常の経路が隔離になり、記録がそれを示します。

## 実測

| 試験 | 結果 |
| --- | --- |
| `TestIsolationIsTheDefaultWhenThePlanNamesTheModuleContext` | **PASS**（34.2秒） |
| `TestIsolationStaysOffWhenThePlanNamesNoModuleContext` | **PASS**（14.1秒） |
| `TestIsolationCanBeTurnedOffExplicitly`（`-isolate=false`が既定に勝つ・木を作らない） | **PASS**（9.1秒） |
| 既存の隔離試験・回帰試験 | **PASS** |

環境：WSL2 Ubuntu 22.04、`go1.27.3 linux/amd64`。

## 言っていないこと

- **依存のある対象**（外部moduleを引く）の隔離ビルド。合成規則が要る（§4.39の案B本体）
- `replace`・`vendor/`の扱い
- 隔離を6番目の条件にすること。**今はしない**（判断の材料は揃いつつある）

## 検証

- 上の表のとおり
- `gofmt`差分なし／`go vet`：PASS

**NOT RUN**：`darwin`・`arm64`、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.47
- 隔離ビルド：[同 §4.42](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
- module context：[同 §4.43](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
