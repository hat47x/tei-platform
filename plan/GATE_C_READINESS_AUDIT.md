# Gate C Readiness Audit（第一Web Target）

- 判定日: 2026-09-19
- 対象: `order_entry` / 第一Web Target（`reference-first-web-target` / `go-net-http`）
- 判定: **Ready**
  - Handover Baseline `B`
  - `baselineRepositoryRevision R = e5231839a5dcc62cce1181a2731fcb7c70d16736`
  - `targetBuildRevision T = 0ba87a4e0a96f70a198b03ff7652fb00e3826abe`
- Product Acceptance Scenario: PAS-04 primary
- 契約: [`GATE_C_EVIDENCE_CONTRACT.md`](GATE_C_EVIDENCE_CONTRACT.md)
- 実行手順: [`GATE_C_FIRST_WEB_TARGET_EVIDENCE_RUNBOOK.md`](GATE_C_FIRST_WEB_TARGET_EVIDENCE_RUNBOOK.md)
- Baseline: [`GATE_B_TO_C_HANDOVER_BASELINE_B.md`](GATE_B_TO_C_HANDOVER_BASELINE_B.md)
- Repository Issue: [`issue/000028-first-web-target-conformance-evidence.md`](issue/000028-first-web-target-conformance-evidence.md) / GitHub #270、[`issue/000007-first-web-target-projection.md`](issue/000007-first-web-target-projection.md) / GitHub #70

## 目的と読み方

Gate Cは「Webの画面ができた」ではなく、Gate Bを通過したApplication meaningが第一Web Targetでも保たれていることを、Evidenceから判定するものです。この文書は、その**判定記録**です。判定の規則は契約、実行手順はRunbookが正本であり、ここには複製しません。

`B / R / T`は別のidentityです（`T`は、Handover Baseline文書などでいう第一Web Target build `T1`と同じものです）。`R`は意味のbaseline（Gate B Ready時点）、`T`はWeb Targetを含むbuild revisionです。`T`は`R`より後のrevisionであり、`R → T`でCanonical Workspaceと生成Worldが変わっていないことも判定に含みます。

## 現在の結論

**Gate C: Ready**です（`B / R / T`は上記）。

Semantic Conformance、Interaction / Functional Evidence、Visual Fidelity Evidence、architecture conformanceが同じ`B / R / T`へ揃い、最終監査`tools/reference-first-web-target-gate-c-readiness-audit`が10チェックすべて成功、`gateCReady = true`を返しました。未解決事項はすべて分類済みで、blockingなものはありません。

## Evidence trail

すべてRepositoryへcommitしていません。stable referenceはSHA-256です。

**保管先（locator）**（2026-09-20追記）: GitHub Release [`gate-c-evidence-T-0ba87a4`](https://github.com/hat47x/tei-platform/releases/tag/gate-c-evidence-T-0ba87a4)（tagは`T`を指す）のasset `tei-gate-c-evidence-T-0ba87a4e.tar.gz`、SHA-256 `de5f587a038c56c0f4b5776953b2f6729e6521d4eda67ebcc246567638fa12a3`。展開して`MANIFEST.sha256`で各file（Screenshot 6枚を含む）を再照合できます。

| Evidence | nature / tool | 結果 | SHA-256 |
| --- | --- | --- | --- |
| Semantic / Functional | `run-first-web-target-gate-c-evidence.sh` → `reference-first-web-target-evidence-audit`（`semantic-functional-summary.json`、run `first-web-target-0ba87a4e0a96`） | source qualification / Semantic Conformance / Functional Evidence すべて`passed = true`、Visualは`pending` | `78c4c3b1031a70eb62683233fb5f1794cf7f71fd8b420ffa741d5ebe3f649d6b` |
| Architecture negative | `reference-first-web-target-architecture-evidence`（`architecture-conformance.json`） | R→T Canonical不変 / Web Target authority boundary / shared Command boundary すべて`passed = true` | `c4d25a3fea751dbee671964d17df87729b38ea7cefd9fffe56641040d4ba32d1` |
| Visual manifest | 評価記録（run `visual-T`） | 7 criteria すべて`pass` | `18e132c3e166b1d3b7cfbcd7e155fd802067bca3b6e3986774f420b0f299b684` |
| Visual audit | `reference-first-web-target-visual-evidence-audit`（`visual-audit.json`） | required captures完備、criteria全resolved、`visualFidelityPassed = true`（単独では`gateCReady = false`） | `61858f3d89891381684b5f8a363fbbfa0471ae313286cf239e2903aa800e2180` |
| 最終Readiness audit | `reference-first-web-target-gate-c-readiness-audit`（`gate-c-readiness-audit.json`） | 10チェック成功、`gateCReady = true` | `b796e38db9a249ef98f2cb600eb354afc70f9adf57d2c83d47116f9968a0ff6c` |

Semantic / Functional run のraw artifact（`read.json`、`submit-*.json`、`custom-submit-*.html`、`server.log`、`health.txt`等）のexact bytes SHA-256は`semantic-functional-summary.json`の`artifacts`に保持しています。

### Screenshot（Visual raw artifact）

viewport 1440x3000（全高を1枚で撮影）のPNGです。各captureは、同一runのTarget-rendered HTML（exact SHA-256）へ`sourceArtifact / sourceSHA256`で結び付いています。

| capture | Screenshot SHA-256 | source HTML SHA-256（先頭16桁） |
| --- | --- | --- |
| `projection-index` | `b1e1848aac28ed34554196e138c6149fe224300884a0664f1bc9a24244875ebb` | `814ea5e68d77e4d0` |
| `custom-submit-initial` | `d19f2cfc4e99dd69423fba2ded7aaa6d0fc2213103c395adff12d197d31a004f` | `f7ae386c2935de9a` |
| `custom-submit-success` | `3c46c9ac0dc81f0dd2ac7ce84c34f8f329898de07ffc578922cb75207ae42068` | `981ab18ce58a387c` |
| `custom-submit-authorization-denied` | `e0d29670fc41e9a718557e52e4d5c18a4aab46683c2f8b884619cbe641ba0fd4` | `6f01aebf9a69d1dd` |
| `custom-submit-validation-not-satisfied` | `de33f953978cebc6f626b5383ad9b7247561e4da807cda42c1b7a5d82972b1cb` | `a264b95e248a17de` |
| `custom-submit-execution-error` | `751c9563d08719ba9ee902258a94b87aca3d698c0d31fe1eba886e25a8f16d8f` | `5d5cceb753e4f0bd` |

## 実行環境

- OS / Host: Ubuntu 22.04（WSL2）、`go run`で起動したnative process、loopback bind
- Go: `go1.25.14 linux/amd64`、`GOENV=off` / `GOWORK=off` / `GOFLAGS=`
- 対象checkout: `T`のclean full checkout（tracked / untracked / ignored local stateなし、index entryは`H`のみ）。run前後で同じ`T`
- Screenshot: Playwright同梱のChrome for Testing系`chrome-headless-shell`（build 1208）でsame-run HTMLのreplay（`tools/reference-first-web-target-visual-replay`）を撮影
- Reference credentialはruntime inputとしてのみ使い、raw artifact / manifestへ保存していない（auditが混入を拒否）

## 実行経緯

Gate Bと同様に、CI（GitHub Actions）が止まっている間に紛れ込んだ「実際に動かして初めて分かる」不具合を修正しながら進めました。旧`T`のEvidenceを新`T`へcarry-forwardしていません。

1. `T = 9d2fd694`: Runbookのsource qualificationで停止。`tools/reference-first-web-target-architecture-evidence/main_test.go`の文字列literalに`\n`ではなく実改行が入りpackageがcompileできず、他のFirst Web Target toolにもgofmt差があった。PR [#815](https://github.com/hat47x/tei-platform/pull/815)で修正（挙動変更なし）。
2. `T = ce95dcf`: Semantic / Functional / Architecture Evidenceは成功。Visual reviewで標準Projectionのheader（Baseline R / Target T / Adapter）の長いrevision値が隣の列へ重なり判読できないことを確認し、`field-grouping-readability = fail`と記録。最終監査はVisual不成立でfail closedした。PR [#816](https://github.com/hat47x/tei-platform/pull/816)で標準Projection CSSに折り返しを追加（presentation-onlyの変更）。
3. `T = 0ba87a4`: 上記を最初から再取得し、すべて成功。本書の判定はこの`T`に対するものです。

## Gate C判定条件との対応

[`GATE_C_EVIDENCE_CONTRACT.md`](GATE_C_EVIDENCE_CONTRACT.md)「Gate C判定」との対応です。最終監査のチェックID（`checks`）を併記します。

- [x] すべてのGate C runが明示Handover Baseline `B` / revision `R`へ追跡できる — `same-B-R-T-and-target-identity`
- [x] Target build revision `T`をbaseline `R`と混同していない — 同上（`B / R / T`をraw artifact間でexact照合）
- [x] `T`でbaseline subjectの意味が変わっていない — `canonical-world-invariant-R-to-T`（`reference/workspace/**`と`reference/world.json`が`R`から`T`まで差分なし）。意味変更は発生していないため新Handover Baselineへの切替は不要
- [x] DriverとWebが同じApplication / Pattern meaningを共有している — Semantic Conformance（5 Application Pattern identity、Runtime Binding）
- [x] Semantic Conformanceが成立している — `source-semantic-functional-passed`
- [x] Reference ScenarioのFunctional Evidenceが揃っている — 同上（read / 成功Command / Authentication denied / Authorization denied / Validation not-satisfied / Execution Error）
- [ ] replay可能なbehaviorを利用する場合の追跡 — 今回のGate C Evidenceでは**利用していない**（任意）。Visual replay helperは保存済みHTMLの配信でありReplayable Behaviorではない
- [x] Visual Fidelityが別Evidenceとして残っている — `visual-fidelity-separately-passed`
- [x] Result / Execution Error / Observationの境界が保たれている — `result-execution-error-observation-boundaries-preserved`
- [x] Web固有実装をCanonicalへ逆流させていない — `web-target-authority-boundary-passed`（`.tei` / `.sql` / native SQL本文を持たない）
- [x] Web Adapterを除去してもCanonical Worldが変わらない — `canonical-world-invariant-R-to-T` + `web-target-authority-boundary-passed`（Adapterは`R`より後に追加され、Canonicalは不変）
- [x] standard ProjectionとCustom UIが同じ上位意味へ接続できる — `standard-custom-shared-command-boundary-passed`（同じ`commandScenarioRunner`）
- [x] Target Capability / Runtime Binding / execution artifact・ABI・Host / Execution・Trust差を第二Targetへ渡せる — `target-difference-handoff-recoverable`
- [x] unresolvedを成功へ読み替えず明示できる — `all-unresolved-explicitly-classified` / `no-blocking-unresolved`

## Visual Fidelityの評価者とその限界

Visual Fidelityは主観を含む判定であり、次のとおり**単一のAI評価者**によるものです。

- evaluator identity: `claude-sonnet-5`、version `1`
- evaluation method: `vlm-screenshot-review`（保持したScreenshotを目視相当で確認し、criterionごとに`pass / fail / unresolved`と根拠をmanifestへ記録）
- 人による確認は行っていません。契約は評価方式（人 / VLM / image comparison）を限定していませんが、評価者を属性として残すことを要求します。

保持したScreenshotとmanifestは別の評価者（人を含む）が再評価できます。再評価で`fail`が出た場合、Gate Cは同じ`T`または新しい`T`で再監査します。Visual Evidenceや評価者のscoreだけでSemantic / Functional Conformanceを合格にしていない点は契約どおりです。

記録した非blockingの見た目の課題（`pass`とした上で残る）:

- 狭い値列で長いidentifierが途中で折り返される（例: `sales.order.custome` / `r`）
- 標準Projectionの先頭cardの見出しが末尾1文字折り返される
- Custom UIで2列panelと`Order lines` panelの境界線が接する
- Custom UIのScenario state表示が、状態が変わっても段階の強調を変えない。Execution Error時のbadgeが`command`（段階名）になる
- validation `satisfied` / `not-satisfied`のtagは文字のみで区別され、視覚的な強調差がない（monochrome Referenceの設計）

## 未解決事項（carry-forward）

最終監査が分類済みです。いずれもblockingではありません。

| id | disposition | owner | 理由 |
| --- | --- | --- | --- |
| `actual-external-command-observation-not-acquired` | carry-forward | external-runtime-evidence | Reference Observationはsynthetic。actual external / production Evidenceへ格上げしない |
| `browse-environment-instance-binding-not-exposed-by-current-driver-api` | carry-forward | `ISSUE-000026` | 現行Driver APIがEnvironment instance selectionを公開しない。Web側で推測しない |
| `general-target-neutral-presentation-taxonomy-not-yet-defined` | carry-forward | `ISSUE-000026` | Gate CはTarget-neutral Presentation taxonomyの完成を要求しない |
| `target-capability-taxonomy-remains-candidate-until-second-target` | carry-forward | `ISSUE-000026` | 第一Targetの観測は、十分に異なる第二Targetと比較するまで候補 |
| `visual-fidelity-evidence-not-acquired` | resolved-by-visual-evidence | — | 独立にauditしたVisual Evidenceで解消 |

## 第二Targetへのhandoff

`semantic-functional-summary.json`の`targetDifferenceHandoff`に、observed Target Capability候補（`http-html-projection` / `http-json-manifest` / `http-read-scenario` / `http-command-scenario` / `http-custom-submit-form`）、Runtime Binding（`sales_core` PostgreSQL / `master_data` Oracle read-only）、execution artifact（`go-run-native-process`）、ABI（`linux/amd64`）、Execution / Trust観測、Web固有差、Canonical化してはいけない差（HTTP route / status、HTML element、CSS class、Referenceのsynthetic fixture値）を保持しています。これは第一Referenceの観測recordであり、Target CapabilityやExecution Profileの最終taxonomyではありません。`ISSUE-000026`で第二Targetの圧力を受けて再検討します。

## Ready の意味と非主張

Gate C Readyは、契約に定めた第一Targetの意味・操作・視覚・architecture境界が、同一provenanceのEvidenceから再確認できたことだけを意味します。次は意味しません。

- production-readyなWeb product
- actual external Oracle / production Commandの成功（Reference Observationはsynthetic）
- Target-neutral Presentation taxonomyの完成
- 第二Targetが不要になったこと
- Visual Fidelityの人による承認
- unresolvedの消滅

`T`でbaseline meaningを変更する必要が生じた場合は、Web固有のfieldやruleで埋めず、authoritative sourceへ戻して新しいHandover Baseline `B'` / `R'`を発行します。

## 再現

Semantic / Functional / Architectureは、`T`のclean checkoutでRunbookの手順を再実行して再取得できます。Screenshotは描画環境（browser build / font）に依存するため、byte-for-byte一致は保証しません。再取得したScreenshotは新しいVisual runとして評価します。

## 関連

- `GATE_C_EVIDENCE_CONTRACT.md`
- `GATE_C_FIRST_WEB_TARGET_EVIDENCE_RUNBOOK.md`
- `GATE_B_TO_C_HANDOVER_BASELINE_B.md`
- `issue/000007-first-web-target-projection.md`
- `issue/000026-target-capability-execution-profile-boundary.md`
- `issue/000028-first-web-target-conformance-evidence.md`
