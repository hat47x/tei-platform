# 宣言からClaimまでの鎖を回帰試験に載せた（§4.33）

`tools/reference-build-catalog-authoring`（#3916）を足した時点では、宣言からClaimまでの鎖は**一度通しただけ**でした。本記録は、それを**毎回通る試験**にしたことを記録します。

## 何を変えたか

`cmd/tei`のエンドツーエンド試験`TestSelfBuildEstablishesL2BEndToEnd`は、これまで**参照catalogをコピー**して使っていました。今は**ツールでcatalogを作ります**。

| 段 | 試験が確かめること |
| --- | --- |
| 能力宣言 | builder childの実体を宣言が名指しする |
| authoring | ツールがcatalogを書き、`catalogDigest`を返す |
| **catalogの中身** | **`artifactDigest`が、実際に走るbuilderのバイトのdigestと一致する** |
| `self-build` | `l2b.claimed=true`・`unmet`空・`qualified=false`・exit 0・artifact存在 |

## なぜdigestの一致を試験で確かめるか

**builderの解決はdigestで照合しません。** `ResolveBuilder`はcontract・contractMajor・recipe kind・target・profileで候補を選び、`pin`が指定されたときだけdigestを見ます。つまり、**catalogが別のバイトを名指ししていても試行は通ってしまう**。

鎖の意味は「**名指ししたバイトが実際に走る**」ことにあるので、そこを固定しました。試験はcatalogを読み直し、`providers[0].artifactDigest`を実際のbuilderのdigestと比べます。

```go
if want := digestOfBytes(builderBytes); written.Providers[0].ArtifactDigest != want {
    t.Fatalf("the catalog names %s but the builder is %s",
        written.Providers[0].ArtifactDigest, want)
}
```

## 試験の中でbuilderを`-root`の中へ置く

authoringツールは`-root`の外を指す`artifact`を拒否します（§4.33）。試験が組み立てるbuilder childは別の一時ディレクトリにあるため、**同じバイトを`-root`の中へ置いてから**宣言します。バイトが同じなのでdigestも同じです。

## 観測結果

| 試験 | 結果 | 所要 |
| --- | --- | --- |
| `TestSelfBuildEstablishesL2BEndToEnd`（authoring経由） | **PASS** | 16.50秒 |
| `TestSelfBuildWithoutTheRebuildCheckNamesTheMissingCondition` | **PASS** | 3.61秒 |

環境：WSL2 Ubuntu 22.04、`go1.27.3 linux/amd64`。

## この試験が保証すること

- **宣言・catalog・Claim**の鎖が、リポジトリ内の材料だけで毎回通る
- catalogが**実際に走るbuilderのバイト**を名指ししている
- `go test ./...`に含まれるため、`tools/repository-health-check.sh`の経路で自動的に回る

## この試験が保証しないこと

- **catalogが正しいこと**（K-3・SC0-26のまま）
- **誰が能力を宣言してよいか**（権限）
- builderの実装が正しいこと（digestを計算するだけで中身は見ない）
- 既定の試行でのClaim（§4.30のまま）
- 能力宣言の**版付け**

## 検証

- 上の2試験：PASS
- `go test ./tools/repository-plan-check -count=1`：PASS
- 文字化け検査：該当なし

**NOT RUN**：Windows、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.33
- ツール：[`reference-build-catalog-authoring/README.md`](../../tools/reference-build-catalog-authoring/README.md)
- 鎖を最初に通した記録：[`TEI_GO_SELF_BUILD_CATALOG_AUTHORING_2026-11-09.md`](TEI_GO_SELF_BUILD_CATALOG_AUTHORING_2026-11-09.md)
