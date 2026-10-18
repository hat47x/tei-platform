# 横断検索・置換の受理実行と再投影（#1804）

ステータス: **設計案（PoC）**。#1804 の受理実行段（acceptance execution）を定める。置換を変更案として受理境界（#267）へ渡し、受理後に各視点へ再投影する最小経路。#1804 の Issue 本体および `CROSS_SEARCH_AND_REPLACE_PROPOSAL_2026-10.md` に従う。

## 1. 目的

#1804 の検索・置換は読み取り専用で、置換を変更案へ変換するところまで実装済みである。受理実行段では、変更案を Shared Acceptance（#267）の `preview → review → accept` に渡し、受理後に Source と各視点へ再投影する。

## 2. 前提（実装済み）

- `reference-model-search`: Symbol / Description / Display / Script の検索、Propose（置換案）、Candidate（`set-symbol` 変更案への変換）
- `reference-shared-acceptance`: `preview / review / accept / reject`、Source 書き込み、Conformance、Decision Index
- 手順は `reference-model-search/README.md` の「Shared Acceptance への接続」に記載済み

## 3. 受理実行の流れ

1. 検索: `Search(world, q)` で hit を得る
2. 提案: `Propose(world, res, replacement, selected)` で Proposal を得る
3. 候補変換: `Candidate(proposal, acceptanceBaseRevision)` で `tei.reference.change-candidate/v0` を得る
4. Preview: Shared Acceptance の `preview` で Conformance / impact を確認
5. Review: 人間が確認（`review`）
6. Accept: Shared Acceptance の `accept` で Source へ書き込み
7. 再投影: 受理後の Source を再読み込みし、各視点（Data / Capability / Validation）へ再投影する

## 4. 再投影の境界

- 再投影は受理後の Source から canonical meaning を再構築する。受理前の意味は変えない。
- 再投影の結果が受理前の意味と異なる場合、`LOGICAL_RULE_RETRACTION_REQUIRES_REVIEW` 等の診断を出す。
- 受理後の Source が外部から変更された場合は `SOURCE_MANIFEST_CHANGED_DURING_ACCEPT` で拒否する。

## 5. 効果仮説と観測（#1973）

- 従来の限界: 置換を変更案として扱う経路はあったが、受理実行と再投影が未接続だった。
- 効果: 検索 → 提案 → 候補変換 → Preview → Review → Accept → 再投影の一続きが通る。
- 観測: 1 件の Symbol 置換が `preview` の全段階を通る。受理後、Source と各視点が再投影される。
- 撤回条件: #267 の変更案形式が変わり、より直接の経路ができた場合は廃止する。
- 未確認: 受理後の再投影の自動化、受理失敗時のロールバック、大規模 Workspace での性能。

## 6. 受入条件

- [ ] 検索 → 提案 → 候補変換 → Preview の手順が文書化されている（済み）
- [ ] Preview が Conformance / impact を確認できる
- [ ] Accept 後、Source が更新される
- [ ] 受理前は正本の意味が変わらない

## 7. 未確認・境界

- 受理後の再投影の自動化（現時点では手動で Source を再読み込み）
- 受理失敗時のロールバック
- 大規模 Workspace での性能
- 旧ツールの実行規約や画面操作を、TEI の意味の正本へ持ち込まない（#1802 の境界に従う）