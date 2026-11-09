# 鎖はWindowsでも通り、catalogは両ホストで同じバイトになった

§4.33の鎖（**能力宣言 → catalog → 計画 → 成果物 → L2-bのClaim**）は、これまでWSL上のLinuxでだけ通していました。本記録は、**Windowsでも通ること**と、**catalogが両ホストで同じバイトになること**を測った結果です。

## 測ったこと1：鎖がWindowsで通る

`cmd/tei`の決定論的な2試験を、**WindowsのGoで直接**実行しました。

| 試験 | 結果 | 所要 |
| --- | --- | --- |
| `TestSelfBuildEstablishesL2BEndToEnd`（authoring経由） | **PASS** | 25.13秒 |
| `TestSelfBuildWithoutTheRebuildCheckNamesTheMissingCondition` | **PASS** | 9.63秒 |

`go test ./cmd/tei -run 'EstablishesL2B\|WithoutTheRebuildCheck'`が`ok`で終わりました。**SKIPではありません。**

これらは**ツールでcatalogを作り**、`artifactDigest`が実際に走るbuilderのバイトと一致することを確かめ、そのうえで`l2b.claimed=true`を要求します。**鎖全体がWindowsでも成立します。**

## 測ったこと2：catalogが両ホストで同じバイトになる

同じ宣言（`artifact`は相対パス`builder`）と**同じバイトのbuilderファイル**から、`windows/amd64`と`linux/amd64`でcatalogを作りました。

| 値 | Windows | Linux |
| --- | --- | --- |
| `catalogDigest` | `sha256:f7e1c879f233fb022e4e5463625056accace7002a11a865dd2797ae760e2320c` | **同じ値** |
| catalogのバイト数 | 667 | 667 |
| `artifactDigest` | `sha256:cabf793a281ecf0060fc1834d8a944fc861416ba3390023fc7cc3ccaf0ebd855` | **同じ値** |

**catalogのバイトが一致します。** 宣言の`artifact`を相対パスにしたことと、ツールが絶対パスをcatalogへ書き込まないことによる。

## なぜ一致するのか

| 段 | ホスト非依存である理由 |
| --- | --- |
| `artifactDigest` | ファイルのバイトの`sha256`。パスを含まない |
| catalogのJSON | `json.MarshalIndent`の出力で、`-root`の位置もartifactの絶対パスも入らない |
| `catalogDigest` | そのバイトの`sha256` |

**したがって、同じ宣言と同じbuilderがあれば、どのホストでも同じ`catalogDigest`を呼び手に渡せる。**

## この実測が言えること

- 鎖が**Windowsでも**通る（`l2b.claimed=true`まで）
- catalogが**両ホストで同じバイト**になる

## この実測が言えないこと

- **すべてのOS**で通ること。観測したのは`linux/amd64`と`windows/amd64`だけ（`darwin`・`arm64`は未観測）
- catalogが**正しいこと**（§4.22 K-3、SC0-26のまま）
- **誰が能力を宣言してよいか**（権限）
- 既定の試行でのClaim（§4.30のまま）
- 能力宣言の**版付け**

## 検証

- WindowsのGoで`cmd/tei`の2試験を直接実行：PASS（25.13秒／9.63秒）
- 両ホストで同じ宣言・同じbuilderバイトからcatalogを作り、`catalogDigest`・バイト数・`artifactDigest`を比較
- 文字化け検査：該当なし

**NOT RUN**：`darwin`、`arm64`、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.33
- 鎖の回帰試験：[`TEI_GO_SELF_BUILD_CHAIN_GATE_2026-11-10.md`](TEI_GO_SELF_BUILD_CHAIN_GATE_2026-11-10.md)
- ツール：[`reference-build-catalog-authoring/README.md`](../../tools/reference-build-catalog-authoring/README.md)
- 二つのホストでの再現（§4.26）：[`TEI_GO_SELF_BUILD_CROSS_OS_DIGESTS_2026-11-09.md`](TEI_GO_SELF_BUILD_CROSS_OS_DIGESTS_2026-11-09.md)
