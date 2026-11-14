# TEI Markdown Graph／Rule 最小検証入力

このディレクトリは、`*.tei.md`の最初のReader／意味解釈／Goコード生成を、**同一の入力群と独立した期待結果**で検証するための固定例です。入力と期待結果は実装に先行して用意しました。

ファイルは通常のMarkdownです。**現行TEIの正式なReaderに読み込ませてよい、という意味ではありません。** 管理対象Workspaceでは`TEI_MD_PARSER_UNBOUND`による明示拒否を維持し、Reader/Language Pluginの選択とSource位置の照合ができるまでは、ここから意味受理・実行・セルフビルドの成功を主張しません。

## 定義と表示の役割

- **定義元**：`## ノードの定義`と`## 関係の定義`にあるMarkdown表。Node IDとEdge ID、起点、終点は表に記載します。
- **規則の候補**：`## 確認する規則`の文章。Reader/Roleの契約が整うまでは実行可能な定義とみなしません。
- **表示図**：`mermaid`の`flowchart`。N1/N2は表示上の識別子であり、意味上のNode IDは`n-1`/`n-2`です。図から定義表のIDや向きを上書きしません。
- **独立Expected候補**：`independent-expected.candidate.json`。コード生成器・Providerの出力から逆算していない値です。正式採用と検収は既存[#1180](https://github.com/hat47x/tei-platform/issues/1180)・[#2514](https://github.com/hat47x/tei-platform/issues/2514)が担当します。

| ファイル | 一因子の差 | 判定の要点 |
| --- | --- | --- |
| `baseline.tei.md` | 基準 | 同じ表示名でもNodeは2件。同じ端点のEdgeもIDごとに2本保持 |
| `edge-reversed.tei.md` | `e-2`の方向を逆転（表と表示図） | 出辺は`n-1=[e-1]`、`n-2=[e-2]`になる |
| `layout-only.tei.md` | Mermaidの`LR`を`TB`へ変更 | 原文・表示版の差は残し、意味上の出辺結果は基準と同じ |
| `duplicate-edge-id.tei.md` | 二行目のEdge IDを`e-1`に変更 | 重複IDを拒否し、一つのEdgeへ黙ってまとめない |
| `diagram-stale.tei.md` | 表の`e-2`だけ反転し、表示図は旧向きのまま | 正しい意味上の結果は反転後。導出図の不一致を示し、どちらも無断で上書きしない |

## 最初の実装に渡す範囲

1. **Source／Reader**：元のUTF-8 byte列と表Cell・Fenceの位置を保存し、どの部分を正式にRole/Operatorへ選択したかを明示します。例示表・説明・Mermaidを拡張子だけで実行可能とみなしません。未対応なら`TEI_MD_PARSER_UNBOUND`等の既存の未対応診断を保持します。
2. **意味解釈**：Node ID、Edge ID、向き、多重性、規則の出典と版を固定します。Mermaidの配置や表示ラベルは意味IDの代用品にしません。
3. **コード生成**：出辺IDを返す判断と重複IDの検査が、TEI側の受理済み意味から生成したGoに含まれることを確認します。固定JSONの生成、手書きGoへの全面委譲は自己生成の合格対象外です。
4. **実ビルド・利用**：実Builderが読んだ入力、生成物、選択Plugin/Providerの実呼出し、通常CLIと独立Consumerの受領を、それぞれ別の証拠として記録します。
5. **変更と再利用**：レイアウト変更時もSource再確認の必要性は失わず、意味不変を示せた下流工程だけ再生成省略の候補にします。旧EvidenceやCurrent Grantは自動復活させません。

既存[製品完成計画のSB-0〜4](../../plan/workstreams/PRODUCT_COMPLETION_PLAN.md)、[実行モデル§37〜39](../../plan/design/PROGRAM_EXECUTION_MODEL_PROJECTIONS_2026-11.md)、[#2621 Markdown Authoring](https://github.com/hat47x/tei-platform/issues/2621)、[#988 Language Plugin](https://github.com/hat47x/tei-platform/issues/988)、[#2600 増分Build](https://github.com/hat47x/tei-platform/issues/2600)へ渡すための**Sourceと期待値候補**です。新しいScenario正本、Parser、Graph Compiler、Host、Plugin Registryは設けません。

**現在の資格**：入力Fixtureと期待値候補を固定した段階。正式`*.tei.md` Reader、Source-backed Meaning、実Go生成、実Plugin/CLI、Stage1→Stage2、正式Owner受理は未実行・未成立です。
