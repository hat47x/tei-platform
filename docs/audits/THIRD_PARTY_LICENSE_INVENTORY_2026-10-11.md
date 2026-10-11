# 公開前の第三者ライセンス初期棚卸し

- 実施日: 2026-10-11
- 対象: 統合候補のcurrent tree。全Git履歴・外部資料全件の権利監査ではない。
- 判定: **現行候補の直接依存 `pgx` のMIT通知を集約した。推移依存を含む完全なSBOMと公開に必要な全資料・assetの人間審査は未完了。**

## 確認したコンポーネント

| Component | Evidence | License / status |
| --- | --- | --- |
| TEI source | Root `LICENSE` | MIT; third-party notice does not replace it. |
| `github.com/jackc/pgx/v5 v5.7.6` | `go.mod` pins v5.7.6; exact upstream tag has MIT `LICENSE` and declares Go 1.23.0, compatible with TEI's Go 1.23.2 baseline. The official Go 1.23.2 toolchain checksum was verified; `go mod tidy` resolved checksums, and `go list -m all` completed. | MIT notice recorded in root `THIRD_PARTY_NOTICES.md`. Runtime adapter preserves the existing `database/sql` provider boundary; its simple protocol setting supports existing multi-statement fixture scripts. |
| pgx runtime transitive modules | Resolved build dependency imports: `pgpassfile v1.0.0`, `pgservicefile v0.0.0-20240606120523-5a60cdf6a761`, `puddle/v2 v2.2.2`, `x/crypto v0.37.0`, `x/sync v0.13.0`, `x/text v0.24.0`. Exact module-cache license files were inspected. | Three Jack Christensen modules are MIT; three Go x modules are BSD 3-Clause. Notices and license texts are included in `THIRD_PARTY_NOTICES.md`. This is the current runtime graph, not a complete inventory of test-only modules or all release targets. |
| Ponytail skill (developer-side) | Previously vendored at `skills/ponytail/`; exact files matched `DietrichGebert/ponytail` v4.10.0. Installed in the developer's global Codex skills directory and removed from the current product tree. | Authoring aid only; not a TEI product dependency and omitted from product notices. The existing Git history still contains earlier copies and must be included in the historical/publication-scope decision. |

## 公開前に残る確認

- `#941` CONCEPTWARE Reference Model and derivative-material boundaries remain unresolved; keep raw model/manual and substantial extracts out of the public repository unless their redistribution terms are clarified.
- Human review is still needed for documentation bibliography/quoted material, screenshots, generated artifacts, other non-code assets, and any historical-only components outside the current tree.
- Codex-side skills are installed per developer environment; they are not distributed by TEI. The corresponding install is available to Codex in a later turn. Claude Code installation is a separate developer-environment setup.
- The Go dependency inventory reflects runtime imports from the current `go.mod`; it is not a completed all-target SBOM. Repeat against the exact approved public candidate and its actual distribution targets.
- `THIRD_PARTY_NOTICES.md` is a source-tree notice. Confirm the release/package process includes it in every applicable distribution archive or binary notice bundle.

This is a scoped inventory and attribution improvement, not legal clearance for all repository contents.
