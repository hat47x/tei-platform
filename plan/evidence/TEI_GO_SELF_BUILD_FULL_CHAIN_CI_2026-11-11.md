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

**workflow 自体はまだ走っていません。** この変更を含む PR が `paths` に一致するので、マージ前の PR で走ります。

## 言っていないこと

- **CI での実行時間**（ランナーでの実測はまだ無い。ローカルでは 254〜980秒）
- `repository-health` 側の扱い（**変更していない**。全鎖は skip されたまま）
- 失敗時の切り分け（workflow が赤いとき、全鎖 때문か他の段かは**段の名前で判断する**）

## 検証

- 上の表のとおり
- `python3 tools/self-build-ci-contract-test.py`：PASS

**NOT RUN**：GitHub Actions での本実行（PR のチェックで走る）、`darwin`・`arm64`。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.53
- 重い観測の扱い：[同 §4.51](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
- 全鎖の観測：[同 §4.50](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
