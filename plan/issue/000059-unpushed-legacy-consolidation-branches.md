# ISSUE-000059: ローカルにだけ残る旧作業branch 2本（3commit）の扱いを決める

- 状態: Done — 2本とも不要と判断して削除した。復元用にbundleを退避した（2026-09-20）
- 提案日: 2026-09-19
- 性格: Repository hygiene / documentation consolidation follow-up
- 主担当要求: RQ-023
- 関連: ISSUE-000056 / ISSUE-000042 / ISSUE-000048 / CONTRIBUTING.md「Branch / PR運用」§4・§9

## 何に困っているのか

`D:\GIT\tei-platform`のローカルにだけ、remoteに存在しない作業branchが2本あります。どちらも上流を持たず、2026-09-09〜10のmainを土台にしています。リポジトリ全体のbranch整理（マージ済みbranchの削除、未マージbranchの一覧化）で「未マージ・要判断」として挙がったまま、判断されていません。

| branch | commit | 内容 |
| --- | --- | --- |
| `docs/business-procedure-design-consolidation-20260909` | `a08e5793`（2026-09-09） | plan直下の圧力・監査文書80本を、設計本文とADR草案へ吸収（98file、+1087 / −24026行） |
| 同上 | `88d074e0`（2026-09-09） | Issue番号の重複の是正（Gate B系を000048〜000053へ改番）と、状態語彙の正規化（15件）、Requirement Gap AuditへRQ-025 / RQ-026を追加 |
| `work/lane-c-two-blocker-ownership-sync` | `033dec02`（2026-09-10） | Lane Cの下流担当を「二つのmaterial blocker」へ揃える（4file、+29 / −23行） |

未pushなので、この作業は`D:\GIT\tei-platform\.git`にしか存在しません。

## 現在のmainとの関係

2026-09-19のmainへ、各commitを試験的にcherry-pickしました（何も変更していません）。すべて競合します。

| commit | 未解決のfile数 |
| --- | --- |
| `a08e5793` | 21 |
| `88d074e0` | 7 |
| `033dec02` | 2 |

競合の背景は、その後のmainの前進です。

- **ADR番号の衝突**: `a08e5793`が追加した3つのADR草案（`0026-business-procedure-minimum-canonical-contract` / `0027-business-semantic-plane-separation` / `0028-business-source-authority-cutover`）の番号は、mainでは別の決定（partition child realization、PostgreSQL partition drop、human-AI builder shared knowledge plane）に使われています。
- **Issue番号の衝突**: `88d074e0`の改番先000048〜000053は、mainで別の作業に使われています（`ISSUE-000056`）。
- **文書の前進**: 削除対象80本のうち、2026-09-19時点でmainに**まだ存在するのは54本**です。残る26本は、同日にmergeした整理PR（#690 / #692 / #693 / #696 / #717 / #718、旧文書を主題別に統合）などで既になくなっています。

つまり、80本のうち約3分の1は別の方法で整理済みで、約3分の2（54本）は、`a08e5793`が目指した形の整理が**まだ行われていません**。

## 内容ごとの価値の見立て

利用者の判断の材料としての見立てです。決定ではありません。

- **`a08e5793`（80本の吸収）**: 設計内容を含みます（`plan/design/BUSINESS_PROCEDURE_V1_DESIGN.md`、ADR草案3件、未解決論点の一本化、Pre-ADR Auditを601行→172行へ絞る等）。単なる移動ではなく、残る54本の整理方針として価値が残る可能性があります。一方で、Lane Cはその後も研究が進んでおり（Actor Context R3系、second Application）、当時の設計本文がそのまま有効かは確認が必要です。
- **`88d074e0`（番号・状態語彙）**: 番号の是正は、改番先が使えないため作り直しになります。状態語彙の正規化とRQ-025 / RQ-026の追加のうち、RQ-025 / RQ-026の行は現在のmainのGap Auditに既にあります。状態語彙は現在も21件が逸脱しており、`ISSUE-000056`で扱います。
- **`033dec02`（Lane Cの担当）**: 現在の`docs/status.md`は既に「material blockerは二つ」と記述しており、内容は概ね反映済みと見られます。競合するfileも少なく、破棄してよい可能性が高いものです。

## 決定と実施（2026-09-20）

利用者の方針は「不要なのであれば削除」です。次のとおり判断し、実施しました。

| branch | 判断 | 理由 |
| --- | --- | --- |
| `work/lane-c-two-blocker-ownership-sync` | 削除 | 内容は現在の`docs/status.md`等へ反映済み |
| `docs/business-procedure-design-consolidation-20260909` | 削除 | 改番はADR・Issue番号の衝突で使えず、状態語彙とRQ-025 / 026は別の作業（`ISSUE-000056`、現行のGap Audit）で扱う。80本の吸収は、mainがその後大きく前進して競合し、そのまま適用できない。整理が必要な文書は、最新のmainに対して行う |

削除の前に、3commit（`a08e5793` / `88d074e0` / `033dec02`）を`git bundle`でWSL上（`~/archive/tei-platform-unpushed-legacy-branches-2026-09.bundle`、約64KB、SHA-256は`~/archive/SHA256SUMS`）へ退避しました。WSLは日次バックアップの対象です。必要になった場合は、`git bundle unbundle`または`git fetch <bundle> <branch>`で復元できます。

## 決めること（以下は起票時の選択肢。上のとおり決定済み）

branchごとに、次のいずれかを選びます。

1. **破棄**: 内容がmainで代替済み、または不要と判断する。
2. **移植**: 有効な部分だけを、最新のmainに対する新しいPRとして作り直す（ADR・Issueの番号は最新mainで採番し直す）。
3. **保管**: 判断を保留し、復元できる形（bundle等）で退避する。

`a08e5793`については、残る54本の整理を進めるか（進めるなら、移植の粒度と、ADR草案を今採用してよい段階かの確認）を含めて決めます。

## 完了条件

- [x] branchごとの判断（破棄 / 移植 / 保管）を記録する（2本とも破棄。復元用にbundleを退避）
- [x] 移植する場合、最新のmainに対して作り直す（移植しないため対象なし。整理は最新のmainに対して`ISSUE-000056`等で行う）
- [x] 破棄・保管する場合、ローカルbranchの扱いを決める（判断後に`git branch -D`した）
- [x] 判断内容を`ISSUE-000056`（番号・状態語彙）と整合させる

## 非目標

- この起票時点でのbranchの削除・書き換え
- ADR草案の採用（ADRの採用判断はこのIssueの範囲外）
- Gate判定・Canonical採用・Evidence主張の変更

## 関連

- `plan/issue/000056-issue-numbering-uniqueness-and-status-vocabulary.md`
- `plan/issue/000042-business-procedure-cross-view-reference.md`
- `plan/issue/000048-business-procedure-contract-adr-readiness.md`
