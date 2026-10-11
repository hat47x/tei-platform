# 公開前の第三者ライセンス初期棚卸し

- 実施日: 2026-10-11
- 対象: 統合候補のcurrent tree。全Git履歴・外部資料全件の権利監査ではない。
- 判定: **確認できた2コンポーネントの通知を集約した。公開に必要な全資料・assetの人間審査は未完了。**

## 確認したコンポーネント

| Component | Evidence | License / status |
| --- | --- | --- |
| TEI source | Root `LICENSE` | MIT; third-party notice does not replace it. |
| `github.com/lib/pq v1.12.3` | `go.mod` pins v1.12.3; `go.sum` pins module checksums; upstream tag contains MIT `LICENSE`. | MIT notice and complete license text added to root `THIRD_PARTY_NOTICES.md`. The current shell lacks Go, so `go list -m all` was not run in this work session. |
| Ponytail skill | `skills/ponytail/SKILL.md` and nested `LICENSE` SHA-256 both match byte-for-byte with `DietrichGebert/ponytail` v4.10.0 files. | MIT; original nested license retained and upstream source/version recorded in `THIRD_PARTY_NOTICES.md`. |

## 公開前に残る確認

- `#941` CONCEPTWARE Reference Model and derivative-material boundaries remain unresolved; keep raw model/manual and substantial extracts out of the public repository unless their redistribution terms are clarified.
- Human review is still needed for documentation bibliography/quoted material, screenshots, generated artifacts, other non-code assets, and any historical-only components outside the current tree.
- The Go dependency inventory reflects the declared module in the current `go.mod`; it is not a completed build-target/SBOM inventory. Repeat against the exact approved public candidate and its actual distribution targets.
- `THIRD_PARTY_NOTICES.md` is a source-tree notice. Confirm the release/package process includes it in every applicable distribution archive or binary notice bundle.

This is a scoped inventory and attribution improvement, not legal clearance for all repository contents.
