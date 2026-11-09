# 判断材料: 操作面の未決4件と、決める順序（#1127 / #1124 / #1640）

- 状態: **判断材料・未確定**。決定はこの文書では行いません。
- 対象の決定: [操作面の設計候補](../design/OPERATION_SURFACE_DESIGN_2026-11.md)の15章にある4件（公開面／語彙の置き場所／`unattended` の条件／drift 照合の既定）
- 下流の正本: [操作面の設計候補](../design/OPERATION_SURFACE_DESIGN_2026-11.md)、[Application Pattern v1](../design/APPLICATION_PATTERN_V1_DESIGN.md)、[プラグインと配布一式](../../docs/plugins.md)、[CLI の JSON 出力契約](../design/CLI_JSON_OUTPUT_CONTRACT_2026-10.md)
- この文書の役割: 4件の**観測済みの材料**を1か所に束ね、**どの判断から決めると後続が解けるか**を示す。決定そのものは書かない。

## 1. 材料の出所

4件とも、`main` 上の実行記録が付いています。この文書は数値や結論を複製せず、参照だけを置きます。

| 未決 | 実行記録 | 記録が示すこと（要約） |
| --- | --- | --- |
| 1. 公開面を増やすか | [OPERATIONS_COMMAND_PUBLICATION_GAP](../evidence/OPERATIONS_COMMAND_PUBLICATION_GAP_2026-11-07/README.md) | `tei commands` は19件だけを載せ、CLIは53件を受け付ける。`operations` は非公開で、成功時は単段JSON、失敗時はstdout 0バイト |
| 2. 語彙の置き場所 | [OPERATION_VOCABULARY_PLACEMENT](../evidence/OPERATION_VOCABULARY_PLACEMENT_2026-11-07/README.md) | プラグインの `CapabilityKind` は7種の閉じた列挙で未知を拒否、操作面の種別は検査が無い |
| 3. `unattended` の条件 | [OPERATION_AUTONOMY_DECLARATION_SURVEY](../evidence/OPERATION_AUTONOMY_DECLARATION_SURVEY_2026-11-07/README.md) | `unattended` は読み取り専用だけで決まり、`Environment` にサービス主体の欄が無く、`human-gated` は未生成 |
| 4. drift 照合の既定 | [OPERATION_DRIFT_CHECK_SURVEY](../evidence/OPERATION_DRIFT_CHECK_SURVEY_2026-11-07/README.md) | 照合値は出力にあるが、渡す口も比べるコードも無い。属性追加では `basis` が変わらない |

## 2. 決める必要があること（4件）

| # | 決定 | 選択肢（材料から読める範囲） | 決めないと止まること |
| --- | --- | --- | --- |
| D-1 | 操作面を CLI の公開面に出すか | (a) `tei commands` に載せ、失敗時もエンベロープにする (b) 出さない（Studio と診断だけに留める） | `operations` の出力契約（二段か一段か）が決まらない。W5 の完了条件2・3が未達のまま |
| D-2 | 操作種別の語彙をどこに置くか | (a) 規則の版（`tei.operation-surface/*`）(b) プラグインの契約（#951） | 未知の種別の扱いが決まらない。`operationKindRank` の既定 rank に落ちる経路が残る |
| D-3 | `unattended` にサービス主体の宣言を必須にするか | (a) 必須にする（宣言の欄と生成規則を新設）(b) 現行のまま（読み取り専用で足りる） | `human-gated` を出す経路の有無が決まらない。8章の安全側の規則が未実装のまま |
| D-4 | drift 照合を既定で有効にするか | (a) 既定で有効 (b) 明示指定のときだけ | 照合値の受け渡し経路（フラグ／保存済み操作面／再導出）が決まらない |

**どの選択肢も、この文書は推薦しません。** 材料は各記録にあります。

## 3. 決める順序（材料から読める依存）

4件は独立ではありません。**先に決めるほど後続が解ける**関係にあります。

| 順 | 決定 | 決めると解けること | 決めるまで止まること |
| --- | --- | --- | --- |
| 1 | **D-4 drift 照合の既定** | 照合値をどう渡すかが決まり、`operations` の出力に「照合値を受け取る口」を足すかどうかが決まる | D-1 の出力契約（成功時と失敗時の形）が確定しない |
| 2 | **D-1 公開面** | `tei commands` に載せるかが決まり、CLI 契約（#2824）へ反映できる | D-2 の語彙を「公開する面の語彙」として扱えるかが決まらない |
| 3 | **D-2 語彙の置き場所** | 未知の種別の扱い（拒否か `out-of-scope` か）が決まり、規則の版を上げるか #951 へ移すかが決まる | D-3 の `unattended` の条件を「規則の語彙」として書けるかが決まらない |
| 4 | **D-3 `unattended` の条件** | `human-gated` を出す経路を新設するかが決まる | 何も止まらない（他3件と独立に決められる） |

**理由**: D-4 は「操作面をどう保存し、どう突き合わせるか」という**出力の形**を決めます。D-1 はその形を公開するかどうかを決めます。D-2 は公開する面に載る語彙の閉じ方を決めます。D-3 は語彙とは別軸（誰が判断するか）なので、最後でも独立でも構いません。

## 4. 最初に答えるべき一つの問い

**「drift 照合の照合値を、どこから渡すか」**（D-4 の前段）。

これは4件のどれよりも先に答えられ、**追加の意味を必要としません**。材料は既にあります。

- `operations` は `worldFingerprint` と `modelRevision` を出力している（[記録](../evidence/OPERATION_DRIFT_CHECK_SURVEY_2026-11-07/README.md)）。
- 同じ形の `STALE_SNAPSHOT` が `trace` にあり、`-expected-fingerprint` で実装済み（`cmd/tei/main.go`）。
- 属性を1つ足すと `worldFingerprint` は変わるが `basis` は変わらないため、**`basis` を照合値にすると見逃す**。

選択肢は3つあり、どれも既存の形の再利用で足ります。

| 案 | 形 | 既存の前例 |
| --- | --- | --- |
| (a) CLI フラグ | `tei operations -expected-fingerprint <値>` | `trace -expected-fingerprint` |
| (b) 保存済み操作面との突き合わせ | 書き出した操作面を読み、現在の意味と比べる | `artifacts -check` の整合検査 |
| (c) 実行時の再導出 | 実行の入口で再導出し、渡された値と比べる | `execution.Invoke` の期間ガード |

## 5. この文書で決めないこと

- 決定そのもの（採用・却下・期限）。4件は15章のOwner判断です。
- 選択肢の推薦。材料の提示に留めます。
- 下流文書の書き換え。決定後に、影響を確認してから行います。
- D-4 の照合値をどの案で渡すか。前段の問いへの答えもOwner判断です。

## 6. この文書が示さないこと

- **4件を決めないと作業が止まるという主張**。操作面は現行でも読み取り専用のビューとして動いており、決定を待たずに他の作業は進められます。
- **順序が唯一の正しい順序であること**。D-3 は独立に決められ、D-2 を先に決めても D-1 は解けます（語彙が決まれば「公開する面に何を載せるか」が具体化するため）。
- **材料が網羅的であること**。各記録は実行1回ぶんの観測で、`main` が進むと数値は変わります。

## 7. 関連

- [操作面の設計候補](../design/OPERATION_SURFACE_DESIGN_2026-11.md)（15章が未決の正本）
- [Application Pattern v1](../design/APPLICATION_PATTERN_V1_DESIGN.md)（Pattern の `Kind` と受理fieldの正本）
- [CLI の JSON 出力契約](../design/CLI_JSON_OUTPUT_CONTRACT_2026-10.md)（`tei commands` と envelope の正本）
- [ADR-0035](../adr/0035-autonomy-levels-and-verification-packets.md)（開発の自律段階。操作面の自律区分とは別の軸）
- [マイルストーンと契約の判断材料](MILESTONE_CONTRACT_DECISION_BRIEF_2026-11.md)（同じ形式の先例）
