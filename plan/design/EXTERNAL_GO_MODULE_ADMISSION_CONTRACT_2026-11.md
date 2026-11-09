# 外部Go Module入力の受入契約（判断材料）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-Bを`cmd/tei`で進めるには、**既存`ResolveSourceClosure`が外部Module根を拒否する方針**をどう扱うかを決める必要があります。この文書は、その判断に必要な材料を**観測済みの事実**と**未決の選択肢**に分けて置くものです。

**採用済みの設計判断ではありません。** 方針の採否は[#1090](https://github.com/hat47x/tei-platform/issues/1090)（closed）と本Issueの所有者が決めます。本書は、決めるための材料を並べるだけです。

## 観測済みの事実（2026-11-07）

| 事実 | 記録 |
| --- | --- |
| `cmd/tei`は`GOPROXY=off`の隔離観測で`missing-module`になる | [SB-0a](../evidence/TEI_GO_MODULE_CLOSURE_PREFLIGHT_2026-11-07.md) |
| 実選択12件は`go.sum`の`h1`と一致し、供給だけでオフライン再解決できる | [SB-0b](../evidence/TEI_GO_MODULE_SUPPLY_VERIFICATION_2026-11-07.md) |
| 供給だけで`cmd/tei`を実Buildできる（exit 0、artifact digest取得） | [X1](../evidence/TEI_GO_BUILD_WITH_SUPPLY_X1_2026-11-07.md) |
| 同じadapterの`ResolveSourceClosure`は同じ対象を`rejected`とする | 同上 |
| 二つの独立試行が同じartifact byteを出す（別々の試行ID・別state） | [SB-N8](../evidence/TEI_GO_BUILD_REPRODUCIBILITY_2026-11-07.md) |
| `cmd/tei`の選択入力は、リポジトリ内318件と外部Module 12件・325メンバーに分かれる | [選択Sourceの観測](../evidence/TEI_GO_SOURCE_CLOSURE_OBSERVATION_2026-11-07.md) |
| 供給zipの1 byte改ざんは`corrupt-supply`として供給不足と別に分類される | [壊れた供給の分類](../evidence/TEI_GO_MODULE_CORRUPT_SUPPLY_2026-11-07.md) |

つまり、**「Buildできる」「byte一致する」は成立しているが、「資格ある閉包」は成立していない**という状態です。

## 未決の選択肢

| 案 | 内容 | 得られるもの | 失うもの／危険 |
| --- | --- | --- | --- |
| **A. 方針を維持** | 外部Module根は拒否したまま。`cmd/tei`のartifactと閉包資格を別Claimとして扱い続ける | 現行の安全側の停止線をそのまま保つ。既存Referenceの検証根拠が変わらない | `cmd/tei`を「資格ある閉包」付きで扱えない。SB-Bの完了条件を「artifact + 別途の入力一覧」へ弱める必要がある |
| **B. 検証済み供給のみ許可** | 供給byteが`go.sum`と一致し、対象が実際に選択したModuleだけを、名前空間付きメンバーとして閉包へ入れる | `cmd/tei`が資格ある閉包を持つ。`docs-only変更`で無関係なartifactが失効しない（#1090の目的） | 外部Moduleの供給元Trust・署名は未検証のまま。`go.sum`自体のTrustが閉包の信頼根拠になる |
| **C. 段階導入** | 既定は拒否。`-allow-external-modules`のような明示opt-inのときだけ、A案の証拠（`go.sum`一致と実選択）を条件に許可する | 既存呼び出し元の挙動を変えない。opt-inした実行だけ新しいClosureを持つ | 二つの閉包意味が併存する期間が生じる。どちらを使ったかをEvidenceへ必ず記録する必要がある |

## 判断に必要な追加の観測（未取得）

- **方針B/Cの証拠を既存の`BuildPlan`へ載せられるか**：`internal/plugin.BuildSource`は`Kind`/`ID`/`Locator`/`Digest`を持つ。外部Moduleメンバーを`Kind: "go-module-closure"`として入れる設計が、既存`schema`と衝突しないかの確認。
- **`go.sum`をどの時点で受理したか**：`go.sum`はリポジトリに追跡されているため、「誰がいつ受理したか」はGit履歴でしかない。別のTrust根拠（署名済みSupply Manifest等）を持つかどうかは未決。
- **WindowsとLinuxで閉包digestが同じになるか**：メンバー名をmodule相対にしたので理論上は一致するはずだが、実測はLinuxのみ。
- **供給の再現手順**：`go mod download`で補った2件（`google/uuid`、`x/sys`）は観測用fixture。本番の供給手順は未決。

## 推奨（既定案）

**C（段階導入）を既定案として推奨します。** 理由は次の三点です。

1. 既存呼び出し元（`tools/reference-plugin-self-build`等）の挙動を変えずに済む。#1090の検証根拠を壊さない。
2. `cmd/tei`の閉包を「資格あり」として扱う経路を、明示opt-inに限定できる。
3. Evidenceに「どちらの閉包意味を使ったか」を必ず書ける。

ただし**これは提案であり、採用はしていません**。Aを選ぶ場合、SB-Bの完了条件を「artifact digestと入力一覧（資格なし）」へ書き換える必要があります。Bを選ぶ場合は、`go.sum`のTrust根拠を別途決める必要があります。

## 覆す条件

- 方針B/Cで、供給byteと`go.sum`の一致が実Buildの入力と一致しない反例が出た場合（例：`go list`の選択と実コンパイラの読み込みがずれる）。
- 別OSで閉包digestが一致しない、または短縮パス等で同じファイルが別メンバーとして二重に数えられる場合。
- 供給元のTrustが、`go.sum`だけでは不足だと判断された場合。

## 参照

- 計画: [`plan/workstreams/TEI_PRODUCT_CRITICAL_PATH_2026-10.md`](../workstreams/TEI_PRODUCT_CRITICAL_PATH_2026-10.md) §35.2
- 既存方針: `tools/internal/gobuilder/source_closure.go` の `ResolveSourceClosure`
- 観測と観測入口: `tools/internal/gobuilder/source_closure_observation.go`
- Gate: [#1119](https://github.com/hat47x/tei-platform/issues/1119)、横断: [#2831](https://github.com/hat47x/tei-platform/issues/2831)
