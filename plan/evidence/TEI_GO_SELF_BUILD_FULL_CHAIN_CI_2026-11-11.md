# 全鎖の観測を、セルフビルドのCIで回す（§4.53）

§4.51は「CI でこの試験を回すかは**別の判断**」で終わっていました。本記録は、**回す場所を決めた**ことを残します。

## どこで回すか

**`tei-self-build-qualification.yml`**（セルフビルド実証の workflow）。理由は三つ:

1. **名前が一致している** — この workflow は「セルフビルドを実証する」ためにあり、全鎖の観測はその中心
2. **`repository-health` に4分を足さない** — §4.51 で避けたかったことそのもの
3. **独立して失敗する** — 全鎖が壊れたとき、健全性の他の項目と混ざらない

## 変えたこと

| 対象 | 変更 |
| --- | --- |
| `timeout-minutes` | 45 → **60**（全鎖が約5分、負荷時にそれ以上） |
| `paths` | `cmd/tei/**`・`tools/reference-go-build-from-plan/**`・`tools/reference-build-plan-authoring/**`・設計文書を追加 |
| 段 | **「自己構築の全鎖を実対象で確認」**を「変更範囲のGo試験」の前に追加 |

**段は `TEI_SELF_BUILD_FULL_CHAIN=1 go test -run TestSelfBuildFullChainOnTheRealSubject ./cmd/tei` を実行します。** 開発者がローカルで回すのと**同じ試験**であり、**CI 専用の第二実装を持ちません**。

## 実測

| 検証 | 結果 |
| --- | --- |
| workflow YAML の構造（job数・timeout・段数） | **妥当**（job 1・timeout 60・段15） |
| `tools/self-build-ci-contract-test.py`（既存の契約試験） | **PASS**（5件） |
| ローカルでの全鎖（§4.50・§4.51） | **PASS** |
| **PR #4254 での本実行** | **`reference-self-build` pass（8分41秒）**、段の中で `ok tei/cmd/tei 105.032s` |

PR #4254 の head `289d3eeb2d5696ed3ac3ca07c11ddcfb946fc541` に対する [Run 42941363544 / Job 130263696253](https://github.com/hat47x/tei-platform/actions/runs/42941363544/job/130263696253) は成功しました。2026-11-11 UTC の Job 全体は01:46:34〜01:55:15（8分41秒）、全鎖の段は01:48:43〜01:50:59（2分16秒）です。段のログの `ok tei/cmd/tei 105.032s` はGo試験の所要時間であり、段やJob全体の時間とは区別します。これは固定した1回の観測で、現在のheadの再検証ではありません。

## 言っていないこと

- CIの負荷による所要時間の変動や、ローカルとの差の原因
- `repository-health` 側の扱い（**変更していない**。全鎖は skip されたまま）
- 失敗時の原因の実証。失敗した段の名前とログで切り分ける

## 検証

- 上の表のとおり
- `python3 tools/self-build-ci-contract-test.py`：PASS

**NOT RUN**：本記録の更新に伴う全鎖の再実行、`darwin`・`arm64`。上記Runの成功はG4の資格判定を変更しません。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.53
- 重い観測の扱い：[同 §4.51](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
- 全鎖の観測：[同 §4.50](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
