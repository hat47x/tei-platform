# plan/ — 計画・判断・作業の管理

このルールは`plan/`のすべての子ディレクトリにも適用する。着手前にルート`../AGENTS.md`と関連する`README.md`、現在のMaster Plan・採用済みADR・担当Issueを確認する。

## 階層別の保存先

| 保存先 | 管理上の規則 |
| --- | --- |
| `plan/`直下 | `MASTER_PLAN.md`、`MILESTONES.md`、Product Acceptance Scenario、Gate / Evidence取得計画・監査・Runbook、横断的な計画・調査の基線。実装コードや一機能だけの詳細設計を新規追加しない。 |
| `plan/issue/` | 作業の**目的・担当境界・進行状態・完了条件**の正本。ファイルは6桁連番`000001-description.md`。GitHub Issueの番号とは別物であり、両者を混同しない。 |
| `plan/adr/` | 継続的な**設計判断と理由**の正本。4桁連番`0001-description.md`。採用済みADRの判断を黙って変更せず、新しい判断は後続ADRで扱う。 |
| `plan/design/` | 複数の作業にまたがる**機能内設計**。Issueの進捗、ADRの採用判断、`docs/architecture.md`の全体構造を重複定義しない。単独Issueで足りる場合は設計ファイルを増やさない。 |
| `plan/design/mockups/` | 画面・操作検討用の単独実行できるHTML等。UIの契約、動作するTarget、実環境Evidenceを兼ねない。 |

## 変更時の確認

1. 上位`product/`・`requirements/`との関係と、減らしたいProduct上の不確実性を特定する。既存Issue・ADR・PRと所有pathが重ならないか確認する。
2. 提案・評価待ち・採用・実装済み・Gate通過を区別する。**未採用の構想を実装の根拠にしない**。Issue状態（Proposed / Ready / In Progress / Blocked / Done / Superseded）とADR状態（提案中 / 採用 / 置き換え済み / 却下）を混同しない。
3. GateやEvidenceの記述を変更するときは、source revision、対象Provider、実行場所、再現条件、未確認範囲の主張が正しいか確認する。過去の成功を現在の環境の保証にしない。
4. 計画・Gateのcurrent判断と、過去の観測・引継ぎ記録を区別する。既存`plan/`直下には歴史的な領域別文書が多数あるため、一括移動せず、移動が必要な場合は参照・正本・並行branchへの影響を先に評価する。
5. 設計本文を追加・変更したら、`plan/design/README.md`等の入口と相互参照を確認する。索引へ可変な状態を二重転記しない。

詳細は`plan/README.md`、`plan/issue/README.md`、`plan/adr/README.md`、`plan/design/README.md`、`../docs/documentation-quality.md`を参照する。これらの案内は要求や決定の第二正本ではない。
