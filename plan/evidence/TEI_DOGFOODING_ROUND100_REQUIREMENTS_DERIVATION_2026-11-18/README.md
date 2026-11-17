# `tei requirements` は宣言から導出され、その出所を名指しする（2026-11-18、第100ラウンド）

- 起票: **なし**（欠陥を1件も作らなかった回）
- 前段: [第99ラウンド](../TEI_DOGFOODING_ROUND99_MULTIFILE_AND_TEXT_JSON_PARITY_2026-11-18/README.md)
- 状態: **導出は成立し、出所も名指しされる。私が読んだ設計文書の1行は「提案」であって現状説明ではなかった。**

## 1. 目的

目的文の「再びドッグフーディングに戻る」を実行した。未使用だった最後の面、**`tei requirements` と `tei world`** を使う。問いは「**ワークスペースが満たすべき要件が、宣言そのものと区別して読めるか**」。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `53dcc34ba`（`origin/main`） |
| CLI実行物 | `sha256:09cc2db57eca6dae2fe3428c2114cf975fddafe2c7a912c929eb131d5128414c` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.26.11 linux/amd64` |

## 3. 導出は成立し、出所を名指しする

`requirements` は**宣言の一覧ではなく、宣言から導いた要件**を返す。各要件は `origins[].introducedBy` と `reason` を持つ:

```json
{"environment": "development", "scope": "build", "kind": "toolchain", "name": "go",
 "origins": [{"introducedBy": "custom-go:order_entry.compose_order_summary",
             "reason": "build custom Go realization"}, ...（計4件）]}
```

**「何が要件か」と「なぜそれが要件か」が同じ entry にある。** これは宣言の写しではない。

## 4. 自分の宣言が要件を増やすことを実測した

`equip_app` に `realize equip.loan_register with custom go` を追加し、環境に束縛した。

| 段階 | requirements 件数 | 自分の寄与 |
| --- | --- | --- |
| 束縛なし | 6 | **なし**（`equip` を含む entry が0） |
| `bind equip_app.equipment` を追加 | **8** | `staging/build/toolchain: go`、`introducedBy=custom-go:equip.loan_register` |

**宣言 → 要件の因果が、件数と出所の両方で確認できた。**

## 5. 途中で「穴」と疑い、コードで否定した

束縛なしの状態で `requirements` が `equip` に触れないのを見て、**「custom-go 宣言があるのに要件が出ない」**と疑った。`internal/dependency/derive.go:100` を読むと条件が明示されていた:

```go
for _, b := range w.Bindings {
    for _, env := range w.Environments {
        if _, ok := bindingsByEnv[string(env.ID)][string(b.Application)]; !ok {
            continue
        }
        add(string(env.ID), ScopeBuild, KindToolchain, "go", ...)
```

**要件は「環境ごと」の概念**であり、束縛の無い環境に要件を作らないのは**設計**である。私の疑いは誤りだった。

## 6. 設計文書の1行を「現状」と誤読した（訂正）

`CLI_FIRST_INTERFACE_DESIGN.md:229` は「`world`・`requirements`・`plugin-lock`・`plugin-resolve`・`plugin-runtime-plan` は内部構造体の直接出力であり、外部契約か内部デバッグ用かの宣言が先に必要（Owner判断）」と書いている。**私はこれを現状説明として読み、catalog の advertisement と矛盾すると疑った。**

しかし:

- その1行は **`### 10.3 版固定が必要なコマンドと追加案（提案）`** の下にあり、**提案**である。
- 同じ文書の §194 が「**§10.1の表は2026-10-31時点の記録であり、現在の実装ではない**」と明記し、`requirements` が envelope を持つようになったと書いている。
- `CLI_JSON_OUTPUT_CONTRACT_2026-10.md` §132 が **A案（推奨）** として、まさにこの提案の帰結を記録し、`requirements` 等を外部契約へ上げたと書いている。

実測でも5コマンドのうち**4つが `enveloped`**（`world` のみ `raw`）。**文書は「提案」と「現状」を正しく区別しており、私が区別しなかった。**

## 7. 読み取れないこと（NOT RUN）

- 実DB・実Provider を用いた要件の充足。
- `world` の `raw` 出力が外部契約かどうか（§132 は `live-r1-smoke` の生バイト証跡照合のため対象外と記録）。
- 実Host、HTTP・Desktop。

## 8. 位置付け

**「要件は宣言から導かれ、出所を名指しする」ことを確認した回**である。同時に私は**2回疑い、2回ともコードと文書で否定された**（束縛条件と、提案行の読み方）。**疑いを確定させる前に出典の区分（提案か現状か）を読む**必要があった。
