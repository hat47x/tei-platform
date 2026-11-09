# Gate C Readiness Audit（第一Web Target）

- 判定日：2026年9月19日
- 対象：`order_entry`／最初のウェブ実行先（`reference-first-web-target`／`go-net-http`）
- 判定：**Ready**
  - Handover Baseline `B`
  - `baselineRepositoryRevision R = e5231839a5dcc62cce1181a2731fcb7c70d16736`
  - `targetBuildRevision T = 0ba87a4e0a96f70a198b03ff7652fb00e3826abe`
- 製品受入シナリオ：PAS-04を主な対象とする
- 契約： [`GATE_C_EVIDENCE_CONTRACT.md`](GATE_C_EVIDENCE_CONTRACT.md)
- 実行手順： [`GATE_C_FIRST_WEB_TARGET_EVIDENCE_RUNBOOK.md`](GATE_C_FIRST_WEB_TARGET_EVIDENCE_RUNBOOK.md)
- 基準版： [`GATE_B_TO_C_HANDOVER_BASELINE_B.md`](GATE_B_TO_C_HANDOVER_BASELINE_B.md)
- リポジトリの課題：ISSUE-000028-first-web-target-conformance-evidence.md／GitHub #270、ISSUE-000007-first-web-target-projection.md／GitHub #70

## 目的と読み方

判定段階Cは、ウェブ画面が完成したかどうかではなく、判定段階Bを通過したアプリケーションの意味が、最初のウェブ実行先でも保たれているかを検証根拠から判定します。この文書はその**判定記録**です。判定規則は契約文書、実行方法は実行手順書を正本とし、ここには複製しません。

`B / R / T`はそれぞれ異なる識別情報です（`T`は引継ぎ基準版の文書などでいう、最初のウェブ実行先の構築版`T1`と同じです）。`R`は判定段階Bの合格時に固定した意味の基準版であり、`T`はウェブ実行先を含む構築版です。`T`は`R`より後の版で、`R → T`の間に正本の作業領域と生成したモデルが変わっていないことも判定対象に含みます。

## 現在の結論

**判定段階C：Ready**です（対象の`B / R / T`は上記のとおり）。

意味の準拠確認、操作と機能の検証根拠、表示の忠実度の検証根拠、構造上の準拠確認が、同じ`B / R / T`に揃いました。最終監査`tools/reference-first-web-target-gate-c-readiness-audit`では10件の確認項目すべてに成功し、`gateCReady = true`を返しています。未解決事項はすべて分類済みで、判定を妨げるものはありません。

## Evidence trail

検証成果物はいずれもリポジトリへコミットしていません。安定した参照にはSHA-256を使います。

**保管先（locator）**（2026年9月20日追記）：GitHub Release [`gate-c-evidence-T-0ba87a4`](https://github.com/hat47x/tei-platform/releases/tag/gate-c-evidence-T-0ba87a4)（タグは`T`を指す）にある成果物`tei-gate-c-evidence-T-0ba87a4e.tar.gz`。SHA-256は`de5f587a038c56c0f4b5776953b2f6729e6521d4eda67ebcc246567638fa12a3`です。展開後は`MANIFEST.sha256`で、画面画像6枚を含む各ファイルを再照合できます。

| Evidence | nature / tool | 結果 | SHA-256 |
| --- | --- | --- | --- |
| Semantic / Functional | `run-first-web-target-gate-c-evidence.sh` → `reference-first-web-target-evidence-audit`（`semantic-functional-summary.json`、run `first-web-target-0ba87a4e0a96`） | source qualification / Semantic Conformance / Functional Evidence すべて`passed = true`、Visualは`pending` | `78c4c3b1031a70eb62683233fb5f1794cf7f71fd8b420ffa741d5ebe3f649d6b` |
| Architecture negative | `reference-first-web-target-architecture-evidence`（`architecture-conformance.json`） | R→T Canonical不変 / Web Target authority boundary / shared Command boundary すべて`passed = true` | `c4d25a3fea751dbee671964d17df87729b38ea7cefd9fffe56641040d4ba32d1` |
| Visual manifest | 評価記録（run `visual-T`） | 7 criteria すべて`pass` | `18e132c3e166b1d3b7cfbcd7e155fd802067bca3b6e3986774f420b0f299b684` |
| Visual audit | `reference-first-web-target-visual-evidence-audit`（`visual-audit.json`） | required captures完備、criteria全resolved、`visualFidelityPassed = true`（単独では`gateCReady = false`） | `61858f3d89891381684b5f8a363fbbfa0471ae313286cf239e2903aa800e2180` |
| 最終Readiness audit | `reference-first-web-target-gate-c-readiness-audit`（`gate-c-readiness-audit.json`） | 10チェック成功、`gateCReady = true` | `b796e38db9a249ef98f2cb600eb354afc70f9adf57d2c83d47116f9968a0ff6c` |

意味・機能を確認した実行で取得した元の成果物（`read.json`、`submit-*.json`、`custom-submit-*.html`、`server.log`、`health.txt`など）のバイト列そのものに対するSHA-256は、`semantic-functional-summary.json`の`artifacts`に保持しています。

### Screenshot（Visual raw artifact）

画面表示領域は1440×3000で、ページ全体を1枚に収めたPNGです。各画像は、同じ実行で実行先が生成したHTML（元のバイト列のSHA-256）と、`sourceArtifact / sourceSHA256`で対応付けています。

| capture | Screenshot SHA-256 | source HTML SHA-256（先頭16桁） |
| --- | --- | --- |
| `projection-index` | `b1e1848aac28ed34554196e138c6149fe224300884a0664f1bc9a24244875ebb` | `814ea5e68d77e4d0` |
| `custom-submit-initial` | `d19f2cfc4e99dd69423fba2ded7aaa6d0fc2213103c395adff12d197d31a004f` | `f7ae386c2935de9a` |
| `custom-submit-success` | `3c46c9ac0dc81f0dd2ac7ce84c34f8f329898de07ffc578922cb75207ae42068` | `981ab18ce58a387c` |
| `custom-submit-authorization-denied` | `e0d29670fc41e9a718557e52e4d5c18a4aab46683c2f8b884619cbe641ba0fd4` | `6f01aebf9a69d1dd` |
| `custom-submit-validation-not-satisfied` | `de33f953978cebc6f626b5383ad9b7247561e4da807cda42c1b7a5d82972b1cb` | `a264b95e248a17de` |
| `custom-submit-execution-error` | `751c9563d08719ba9ee902258a94b87aca3d698c0d31fe1eba886e25a8f16d8f` | `5d5cceb753e4f0bd` |

## 実行環境

- OS・ホスト：Ubuntu 22.04（WSL2）、`go run`で起動したネイティブプロセス、ループバックへの接続待受け
- Go：`go1.25.14 linux/amd64`、`GOENV=off`／`GOWORK=off`／`GOFLAGS=`
- 対象の作業領域：版`T`でリポジトリ全体を取得し、追跡対象・追跡対象外・無視対象のローカル状態がなく、索引項目は`H`だけでした。実行の前後とも同じ`T`です。
- 画面画像：Playwright同梱のChrome for Testing系`chrome-headless-shell`（構築版1208）を使い、同じ実行で取得したHTMLを再表示（`tools/reference-first-web-target-visual-replay`）して撮影
- 参照用の認証情報は実行時の入力にだけ使い、元の成果物や一覧には保存していません（監査で混入を拒否）

## 実行経緯

判定段階Bと同様、GitHub Actionsの自動検査が停止していた間に残った、実際に動かして初めて判明する不具合を修正しながら検証を進めました。旧版`T`で得た検証根拠を、新版`T`にそのまま引き継いではいません。

1. `T = 9d2fd694`：実行手順書のソース確認で停止しました。`tools/reference-first-web-target-architecture-evidence/main_test.go`の文字列リテラルに、`\n`ではなく実際の改行が入り、パッケージをコンパイルできませんでした。他の最初のウェブ実行先用ツールにもgofmtの差分がありました。PR [#815](https://github.com/hat47x/tei-platform/pull/815)で修正しました（挙動の変更なし）。
2. `T = ce95dcf`：意味・機能・構造に関する検証根拠の取得には成功しました。一方、表示を確認すると、標準の派生表示の見出し（Baseline R／Target T／Adapter）にある長い版の値が隣の列へ重なり、読み取れませんでした。`field-grouping-readability = fail`と記録し、最終監査では表示に関する条件が成立しなかったため、安全側に倒して不合格としました。PR [#816](https://github.com/hat47x/tei-platform/pull/816)で標準の派生表示のCSSに折返しを追加しました（表示だけの変更）。
3. `T = 0ba87a4`：上記の検証根拠を初めから取得し直し、すべて成功しました。この文書の判定は、この`T`についてのものです。

## Gate C判定条件との対応

[`GATE_C_EVIDENCE_CONTRACT.md`](GATE_C_EVIDENCE_CONTRACT.md)が定める「Gate C判定」との対応を示します。最終監査で使用した確認項目のID（`checks`）も記載します。

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

表示の忠実度には主観的な判断が含まれます。今回の評価は、次の**単一のAI評価者**によるものです。

- 評価者の識別情報：`claude-sonnet-5`、版`1`
- 評価方法：`vlm-screenshot-review`（保存した画面画像を目視に相当する方法で確認し、評価項目ごとに`pass / fail / unresolved`と根拠を一覧に記録）
- 人による確認は行っていません。契約は評価方式を人・視覚言語モデル・画像比較のいずれかに限定しませんが、評価者の情報を記録することを求めます。

保存した画面画像と評価の一覧は、人を含む別の評価者でも再評価できます。再評価で`fail`となった場合は、判定段階Cについて、同じ`T`または新しい`T`で監査をやり直します。表示の検証根拠や評価点だけで意味・機能の準拠確認を合格にしていないことは、契約に沿っています。

`pass`と判定した後も残る、合格を妨げない表示上の課題は次のとおりです。

- 幅の狭い値の列で長い識別子が途中で折り返される（例：`sales.order.custome`／`r`）
- 標準の派生表示にある最初のカードの見出しで、末尾の1文字が折り返される
- 個別実装の画面で、2列のパネルと`Order lines`パネルの境界線が接する
- 個別実装の画面で、シナリオの状態が変わっても段階の強調表示が変わらない。実行時エラーの際には、バッジが`command`（段階名）になる
- 検証結果の`satisfied`／`not-satisfied`は文字だけで区別され、表示の強調に違いがない（単色の参照事例としての設計）

## 未解決事項（carry-forward）

最終監査で分類しています。いずれも合格を妨げるものではありません。

| id | disposition | owner | 理由 |
| --- | --- | --- | --- |
| `actual-external-command-observation-not-acquired` | carry-forward | external-runtime-evidence | Reference Observationはsynthetic。actual external / production Evidenceへ格上げしない |
| `browse-environment-instance-binding-not-exposed-by-current-driver-api` | carry-forward | `ISSUE-000026` | 現行Driver APIがEnvironment instance selectionを公開しない。Web側で推測しない |
| `general-target-neutral-presentation-taxonomy-not-yet-defined` | carry-forward | `ISSUE-000026` | Gate CはTarget-neutral Presentation taxonomyの完成を要求しない |
| `target-capability-taxonomy-remains-candidate-until-second-target` | carry-forward | `ISSUE-000026` | 第一Targetの観測は、十分に異なる第二Targetと比較するまで候補 |
| `visual-fidelity-evidence-not-acquired` | resolved-by-visual-evidence | — | 独立にauditしたVisual Evidenceで解消 |

## 第二Targetへのhandoff

`semantic-functional-summary.json`の`targetDifferenceHandoff`には、観測した実行先の能力の候補（`http-html-projection`／`http-json-manifest`／`http-read-scenario`／`http-command-scenario`／`http-custom-submit-form`）、実行時の対応付け（`sales_core`のPostgreSQL／読取り専用の`master_data` Oracle）、実行成果物（`go-run-native-process`）、ABI（`linux/amd64`）、実行と信頼についての観測結果、ウェブ固有の違い、正本に取り込んではいけない違い（HTTPの経路・状態コード、HTMLの要素、CSSのクラス、参照事例の合成試験データの値）を保持しています。これは最初の参照事例の観測記録であり、実行先の能力や実行方式の最終的な分類ではありません。`ISSUE-000026`で二つ目の実行先と比較し、再検討します。

## Ready の意味と非主張

判定段階CのReadyは、契約で定めた最初の実行先における意味・操作・表示・構造の境界を、来歴の揃った検証根拠から再確認できたことだけを意味します。次のことまでは意味しません。

- 本番運用に耐えられるウェブ製品であること
- 実際の外部Oracleや本番の命令実行に成功したこと（参照事例での観測は合成データによるもの）
- 実行先に依存しない表示方式の分類が完成したこと
- 二つ目の実行先が不要になったこと
- 表示の忠実度を人が承認したこと
- 未解決事項がすべて解消したこと

`T`で基準版の意味を変更する必要が生じた場合、ウェブ固有の項目や規則で補いません。正本となる入力元へ戻り、新しい引継ぎ基準版`B'`／`R'`を発行します。

## 再現

意味・機能・構造については、変更のない版`T`を取得した作業領域で実行手順書に従えば、検証根拠を再取得できます。一方、画面画像は描画環境（ブラウザーの構築版やフォント）に左右されるため、バイト単位での一致は保証しません。撮影し直した画面画像は、新しい表示検証の実行として評価します。

## 関連

- `GATE_C_EVIDENCE_CONTRACT.md`
- `GATE_C_FIRST_WEB_TARGET_EVIDENCE_RUNBOOK.md`
- `GATE_B_TO_C_HANDOVER_BASELINE_B.md`
- ISSUE-000007-first-web-target-projection.md
- ISSUE-000026-target-capability-execution-profile-boundary.md
- ISSUE-000028-first-web-target-conformance-evidence.md
