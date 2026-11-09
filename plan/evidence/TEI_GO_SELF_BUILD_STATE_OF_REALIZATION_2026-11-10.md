# セルフビルドの実現状況の総括（SB-C0の終了時点）

本記録は、SB-C0（[SB-C0の入口と再入場の契約](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)）が**どこまで到達したか**を、根拠の在り処つきで一度に示すものです。**新しい主張はしません。** 既に記録した観測を、現在の`main`で検証し直したうえで並べます。

## 到達した状態

**能力宣言 → catalog → 計画 → 成果物 → L2-bのClaim**が、製品の契約を一つも変えずに、二つのホストで通ります。

| 層 | 何が言えるか | 根拠 |
| --- | --- | --- |
| **L1** | 同じsourceから同じバイナリが出る | [`TEI_GO_SELF_BUILD_ARTIFACT_REPRODUCIBILITY_2026-11-09.md`](TEI_GO_SELF_BUILD_ARTIFACT_REPRODUCIBILITY_2026-11-09.md) |
| **L2-a** | entryが宣言を引き渡し、builderが読んだバイトとentryの保持バイトが一致する | [`TEI_GO_SELF_BUILD_CHILD_DECLARATION_2026-11-09.md`](TEI_GO_SELF_BUILD_CHILD_DECLARATION_2026-11-09.md)・[`TEI_GO_SELF_BUILD_ENTRY_INTENT_FINGERPRINT_2026-11-09.md`](TEI_GO_SELF_BUILD_ENTRY_INTENT_FINGERPRINT_2026-11-09.md) |
| **L2-b（解釈）** | entry自身が宣言を解釈して計画を導く | [`TEI_GO_SELF_BUILD_INPROCESS_RESOLUTION_2026-11-09.md`](TEI_GO_SELF_BUILD_INPROCESS_RESOLUTION_2026-11-09.md) |
| **L2-b（(c)）** | その計画がそのartifactを生んだことを、二回のbuildの一致で観測する | [`TEI_GO_SELF_BUILD_ARTIFACT_REPRODUCIBILITY_2026-11-09.md`](TEI_GO_SELF_BUILD_ARTIFACT_REPRODUCIBILITY_2026-11-09.md) |
| **L2-b（Claim）** | 五条件の同時成立を述語として、`l2b.claimed`で述べる | [`TEI_GO_SELF_BUILD_L2B_CLAIM_2026-11-09.md`](TEI_GO_SELF_BUILD_L2B_CLAIM_2026-11-09.md) |
| **L3** | stage1のバイナリがstage2を動かす | [`TEI_SELF_BUILD_CMD_TEI_L3_2026-11-08.md`](TEI_SELF_BUILD_CMD_TEI_L3_2026-11-08.md) |
| **鎖** | 宣言を書くところからClaimまでが一本で通り、CIで毎回回る | [`TEI_GO_SELF_BUILD_CHAIN_GATE_2026-11-10.md`](TEI_GO_SELF_BUILD_CHAIN_GATE_2026-11-10.md) |
| **二つのホスト** | `linux/amd64`と`windows/amd64`で同じ計画・同じartifact・同じcatalog | [`TEI_GO_SELF_BUILD_CROSS_OS_DIGESTS_2026-11-09.md`](TEI_GO_SELF_BUILD_CROSS_OS_DIGESTS_2026-11-09.md)・[`TEI_GO_SELF_BUILD_CHAIN_WINDOWS_2026-11-10.md`](TEI_GO_SELF_BUILD_CHAIN_WINDOWS_2026-11-10.md) |

## 製品の契約を変えずに済んだ理由

|catalogの扱い|決定|効果|
| --- | --- | --- |
| 誰が読むか | **K-3**（呼び手が名指しする・digest必須。[§4.22](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)） | 新しいAuthorityを作らない |
| どこに書くか | **製品の外**（[§4.32](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)） | Workspaceの`formatVersion`もplugin構文も変えない |
| 誰が作るか | **ツール**（[§4.33](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)） | 能力宣言からcatalogを作り、digestは実ファイルから計算する |

**`tei.workspace.yaml`も`.tei`の構文も一行も変わっていません。** 既存readerは未知の入力を拒否する設計なので、欄を足す案は古いbinaryとの互換を壊しました（[§4.31](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)）。

## 既定の判断（[§4.34](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)）

| 問い | 決定 |
| --- | --- |
| `-rebuild-check`を全試行で既定に | **しない**（child経路では費用が倍になるだけ） |
| 既定経路をin-processへ | **据え置き**（名指しされていないcatalogを読むのはK-3の取り消し） |
| `l2b.claimed`を既定で | **立てない**（Claimは呼び手の行為） |
| D-1 | **採らない**（外在consumerを測れない） |
| `-strong` | **採らない**（§4.30で足りる） |

## この環境で確認し直したこと

`main`の`d264173ad`で実行しました（WSL2 Ubuntu 22.04 / `go1.27.3`）。

| 検証 | 結果 |
| --- | --- |
| `go test ./cmd/tei -run 'EstablishesL2B\|WithoutTheRebuildCheck\|AssessL2B\|ApplyEvidenceDefaults'` | **PASS**（11.6秒・7件） |
| `go test ./tools/repository-plan-check` | **PASS**（68秒） |
| `go test ./internal/buildintent ./tools/reference-build-catalog-authoring ./tools/reference-build-authoring-intent` | **PASS** |

## 言えないこと（まとめ）

| 項目 | 状態 |
| --- | --- |
| `darwin`・`arm64`・別のGo版 | **未観測。このホストでは実行できない**（macOS無し・`qemu-aarch64`無し） |
| 意味の受理・Runtime Authority | **未主張。** `qualified`は常に`false` |
| 供給のTrust・Seed Trust | **未主張** |
| catalogが正しいこと | **未主張**（K-3、SC0-26） |
| 誰が能力を宣言してよいか（権限） | **未決**。catalogは「buildできると名乗るデータ」であり、名乗りの妥当性は範囲外 |
| 能力宣言の版付け | **未決。** 第二のconsumerが現れた時点で決める |
| G4（適格性） | **`UNQUALIFIED`のまま。** 設計ではなく適格性の判定であり、本記録の範囲外 |

## 終了条件の状態

設計文書§7の終了条件は、**45項目がチェック済み、未チェックは2項目**です。

| 未チェック項目 | なぜ未チェックか |
| --- | --- |
| `darwin`・`arm64`・別のGo版の観測 | このホストでは実行できない |
| 能力宣言の版付け | 第二のconsumerが現れた時点で決める（今は互換の約束をしない） |

**どちらも「やっていない」ではなく「今はできない／今は決めない」であり、理由と再開条件を記録しています。**

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
- 追補（既定の判断）：[同 §4.34](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
- 記録の索引：[`README.md`](README.md)
