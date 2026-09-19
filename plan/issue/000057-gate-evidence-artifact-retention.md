# ISSUE-000057: Gate Evidence artifactの保管先を決める

- 状態: Done — 容量が小さいためGitHub Releaseへ格納し、locatorを文書へ追記した（2026-09-20）
- 提案日: 2026-09-19
- 性格: Evidence provenance / operational continuity
- 主担当要求: RQ-021 / RQ-022
- 関連: ISSUE-000029 / ISSUE-000030 / ISSUE-000037 / ISSUE-000028 / ISSUE-000058

## 何に困っているのか

Gate B / Gate Cは、実際に取得したEvidence artifactのSHA-256を根拠にReadyと判定されています。RunbookはEvidence artifactを**Repository外**へ保存し、Repositoryへcommitしないと定めています（credential・DSN・Screenshotをリポジトリ履歴へ入れないため）。この規則自体は妥当です。

問題は、Repository外の**保管先が決まっていない**ことです。現在のartifactは、1台のマシンのWSL2ホーム配下にしかありません。

| Evidence | 場所（1台のマシン上） | 規模 |
| --- | --- | --- |
| Gate B（`R = e5231839…`、採用分） | `~/gate-b-evidence/run-R-main/` | 約32KB（JSON 5件） |
| Gate C（`T = 0ba87a4e…`、採用分） | `~/gate-c-evidence/run-T2/` | 約1.3MB（JSON・HTML・log・PNG 6枚） |

`GATE_B_TO_C_HANDOVER_BASELINE_B.md`と`GATE_C_READINESS_AUDIT.md`はSHA-256を記録していますが、artifactを取得できる場所は書かれていません。マシンを失う、またはWSLを再作成すると、fingerprintを再照合できなくなります。

再現性にも差があります。Semantic / Functional / Architectureは同じ`T`のclean checkoutで再実行できますが、**Screenshotは描画環境（browser build・font）に依存し、byte一致は保証されません**。Visual Fidelityの評価は、保持したScreenshotへ別の評価者が当たれることが前提です（`ISSUE-000058`）。

## 防ぎたい失敗

- Ready判定の根拠artifactを、後から誰も開けなくなること。
- 取得できないまま「fingerprintがある」ことだけで再監査可能とみなすこと。
- 逆に、保管のためにcredential・DSN・Screenshotをリポジトリ履歴へ入れてしまうこと。

## 守る原則

- artifactをRepositoryへcommitしない。
- DSN・credential・secret pathを、保管するartifactへ含めない（現行のaudit toolが混入を拒否します）。
- Canonicalの第二正本にしない。保管先は参照先であり、意味の正本ではありません。

## 決めること

1. 保管先: private storage（例: 個人のクラウドストレージ）、Repositoryとは別のprivate repositoryのrelease asset、社内のバックアップ等。private repositoryであることと、artifactに機密が含まれないことを、audit済みartifactで確認できる点を前提にします。
2. 保持期間と、置き換えられた古いrun（`run-R` / `run-Rpp`、`run-T`）の扱い。
3. 文書からの参照の書き方。SHA-256に加え、保管先を指す**locator**（secretを含まないkey）を`GATE_B_TO_C_HANDOVER_BASELINE_B.md` / `GATE_C_READINESS_AUDIT.md`へ追加するか。
4. 保管したartifactを再照合する手順（fingerprint再計算）と、その頻度。
5. 今後のGate（Gate B'・Gate C'、successor Handover）でも同じ保管規則を使うか、Runbookへ書くか。

## 決定と実施（2026-09-20）

利用者の方針は、**容量が小さければ圧縮してGitHubへ格納し、大きければWSL上に留める**（WSLは日次バックアップの対象）です。Gate B（約32KB）とGate C（約1.3MB）はどちらも小さいため、GitHubへ格納しました。

- 保管先: このRepository（private）のGitHub Release（prerelease）。Repositoryのtreeや履歴には入れません。tagは`R` / `T`のcommitを指します。
  - Gate B: [`gate-b-evidence-R-e523183`](https://github.com/hat47x/tei-platform/releases/tag/gate-b-evidence-R-e523183)、`tei-gate-b-evidence-R-e5231839.tar.gz`（SHA-256 `1af53d6e…4395`）
  - Gate C: [`gate-c-evidence-T-0ba87a4`](https://github.com/hat47x/tei-platform/releases/tag/gate-c-evidence-T-0ba87a4)、`tei-gate-c-evidence-T-0ba87a4e.tar.gz`（SHA-256 `de5f587a…12a3`）
- 各archiveは、各fileのSHA-256を並べた`MANIFEST.sha256`を同梱しています。展開後に再照合できます。
- 格納前にDSN・credential・secret pathが含まれないことをgrepで確認し、格納後にassetを再ダウンロードしてSHA-256の一致を確認しました。
- 置き換えられた古いrun（`run-R` / `run-Rpp` / `run-T`）は採用していないため、格納していません。
- locatorを`GATE_B_TO_C_HANDOVER_BASELINE_B.md`と`GATE_C_READINESS_AUDIT.md`へ追記し、保管規則を`GATE_B_EVIDENCE_RUNBOOK.md`と`GATE_C_FIRST_WEB_TARGET_EVIDENCE_RUNBOOK.md`へ書きました。今後のGateも同じ規則を使います。

## 完了条件

- [x] 保管先と保持期間を決める（GitHub Release、期限なし。大きい場合はWSL）
- [x] Gate BのRequired Evidence（same-run child 3件、final、artifact audit）を保管する
- [x] Gate Cの採用run（raw artifact、Screenshot 6枚、manifest、各audit）を保管する
- [x] 保管物のSHA-256が文書の記録と一致することを確認する
- [x] 文書へlocatorを追記する（過去の判定は書き換えず、追記として残す）
- [x] Runbookへ保管規則を追記する

## 非目標

- Evidenceの取り直し（同じ`R` / `T`のEvidenceを別runで置き換えない。same-runの原則を守る）
- artifactのRepositoryへのcommit
- Screenshotのbyte再現性の保証

## 関連

- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/GATE_C_FIRST_WEB_TARGET_EVIDENCE_RUNBOOK.md`
- `plan/GATE_B_TO_C_HANDOVER_BASELINE_B.md`
- `plan/GATE_C_READINESS_AUDIT.md`
