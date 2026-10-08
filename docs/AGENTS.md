# docs/ — 説明・実装状況・適合範囲

このルールは`docs/`以下全体に適用する。ルート`../AGENTS.md`を先に読む。

## 階層別の保存先

| 保存先 | 管理上の規則 |
| --- | --- |
| `docs/` | 利用者・新規参加者向けの案内、Architecture、設計思想、セキュリティ、命名、導入手順。`docs/architecture.md`と`docs/business-system-design.md`はそれぞれ対象領域のcurrent説明基線。Product RequirementやADRの第二正本にしない。 |
| `docs/status.md` | **現在の実装状況**の正本。計画・設計・synthetic試験・actual external Evidence・production採用を区別して書く。変更前に実際のcommitとEvidenceを確認する。 |
| `docs/conformance/` | 宣言した境界に対し、どこまで実装・検証したかのConformance説明。対象revision・検証条件・非保証範囲を明示する。 |
| `docs/adr/` | 過去の参照URLを保つ互換案内。ADRの決定本文は`../plan/adr/`にのみ置く。新しいADRをここへ追加しない。 |
| `docs/audits/` | 監査時点・対象・方法・観測結果を残す。過去の記録を現在の判定に上書きしない。 |
| `docs/specification/` | 読者向けの仕様整理・閲覧用資料。要求・決定・受入判定の独立した権威を持たせない。 |

## 更新時の確認

- 説明文に新しいProduct Requirementや採用判断が必要なら、先に`../requirements/`または`../plan/adr/`の担当文書へ戻す。
- 対応するIssue・Reference・検証結果を照合し、**動くこと、検証したこと、設計として想定していること**を一つの「対応済み」にまとめない。
- `docs/status.md`の現在地を古い説明へ転載し続けない。必要ならリンクでつなぐ。
- 文書変更ではリンクと正本性、同じ概念の名称、証拠の鮮度、日本語の自然さを確認する。`documentation-quality.md`を参照する。
