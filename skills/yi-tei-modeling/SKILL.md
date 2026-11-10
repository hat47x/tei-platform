---
name: yi-tei-modeling
description: >-
  TEIの既存Workspace・モデル・Relation・Sourceを、実在するread-only CLIで
  発見、検証、参照追跡するときに使う。対象版と診断、未解決を保持する。
  TEIでの書込み・モデル受理・生成物の実配布を暗黙に行わない。
---

# TEIモデリング調査（Product Skill）

> **2026-11-10 時点の証拠範囲**：初期Product Skillは[merged #2607](https://github.com/hat47x/tei-platform/pull/2607)でmain統合済み。固定HEAD `d0de429ae5aac38b7949c2cfe6eacafff059675f`の[CI #41532988114](https://github.com/hat47x/tei-platform/actions/runs/41532988114)は、公開Reference Workspaceに対する同一実Goバイナリのread-only S0/S1を限定確認した。さらに[TAI #168](https://github.com/hat47x/tai-pleroma/issues/168)／[TEI #2509](https://github.com/hat47x/tei-platform/issues/2509)の2026-11-07実DSH記録では、H3-A/Bで固定Skill本文と参照資料の実Tool Readが成立した。ただし固定Task/Source配置の不一致、OS読取隔離不足、不完全Tool Reply、H2 Session再利用等があり、**H2/H3比較適格0/4、Skillの増分効果・二Harness互換・実Provider/業務Effect・Owner正式受理はNOT RUN／未成立**。これは最新mainでの再実行証明ではなく、各固定版の履歴Evidenceである。

## いつ使うか

TEIのSource・モデル要素・入力元・参照関係・Validationを調べ、結果を再現可能な根拠とともに説明したいときに用いる。TEIの単独利用にTAIは必要ない。複数製品をまたぐ活動の進行はTAI横断Skillが担当する。

**このSkillはTEI Kernel Plugin、TEIで開発したLibrary、外部Harness Plugin、MCP Serverそのものではない。** TEIで生成したLibraryを外部Java/.NET/Wasm等が利用する際に、このSkill/TEI CLIを配布依存にしない。

## 実行の流れ

1. **実環境を確認**：TEIリポジトリのcheckoutとWorkspaceの所在・利用目的・利用可能な情報範囲を確認する。`go.mod`の`go`行の要求版と実際の`go version`を照合し、不足する場合は環境不備として停止する（Productの意味検証失敗と区別）。CLIの使用可能なread-only入口を`go run ./cmd/tei commands`で確認する。ビルド済み実行ファイルがあれば`tei version -json`などで版を照合し、Gitの改訂・Working Treeを別記する。**発見したCLIと実際の呼出しで使用した実行物（path/bytes/argv/cwd）を区別する**。`go run`例は探索用であり、**比較試験の実行物固定**には一度ビルドした同一バイナリを全コマンドで使い、その実bytesとGit/Go/Workspace改訂を別々に記録する。ビルド済バイナリの版表示だけでビルド入力全体の閉包を保証しない。CLI使用中にSource/Rootが変化した場合は同一試行として混ぜず、現在の実Sourceを再照合する。`go run`からVCS revisionが得られるとは仮定しない。
2. **対象を発見**：`go run ./cmd/tei model catalog -workspace <workspace>`の`entries[].subject.kind/id`から対象候補を選び、重複・曖昧なら特定を保留する。名前や一覧順を正本IDとして推測しない。**標準 `std` 定義と利用者定義を区別し、目的に必要な利用者のSubject kind/idを明示して照合する。**Catalog非空だけでは利用者定義が読まれたとは言えない（[#2822](https://github.com/hat47x/tei-platform/issues/2822)）。
3. **対象を読む**：`go run ./cmd/tei inspect -workspace <workspace> -subject-kind <kind> -subject-id <id>`で指定要素を調べ、必要な場合のみ`trace`で宣言済み直接参照を追う。Source indexの候補と元ファイルの所在を分ける。
4. **検証する**：`go run ./cmd/tei validate -workspace <workspace> -json`の終了コード、`ok`、`diagnostics`、Warning、エラーをそれぞれ確認する。exit 0や`ok=true`だけからWarningなし・業務受理済みと判断しない。**空Workspace・未読込Sourceでも`ok=true`となり得る**ため、必要なSource root／実Parsed・Loaded Subjectの到達と検証成功を別に評価する（[#2822](https://github.com/hat47x/tei-platform/issues/2822)、[#2823](https://github.com/hat47x/tei-platform/issues/2823)）。
5. **版と結果範囲を照合**：結果ごとの`worldFingerprint`と`sourceIndexFingerprint`、必要なSubject/Trace固有の照合値を確認する。複数コマンド実行中のWorkspace変更、異なるScope、未取得/部分的な参照、古い結果を混ぜて一回の全件検証と呼ばない。`trace`のページ継続では**外側**`result.snapshotFingerprint`を`-expected-fingerprint`に渡し、`result.trace.snapshotFingerprint`と区別する。`trace`の`-scope`は読取りの文脈名であり、業務権限を与える値ではない。
6. **用途に必要な結果を返す**：対象ID・Source位置・コマンドと版・診断結果・確認できた範囲・未解決点を明示する。外部LLMや別Productへ結果を送るときは、Source保持側が許可する用途と開示範囲を守る。通常の組織Dataの利用を一律に制限する趣旨ではなく、秘密・保護対象がある条件で実Host/OS権限と開示資格を確認する。
7. **続きの作業へ引き渡す**：モデル変更、受理、コード生成、外部Host配布、DB更新が必要なら、現在の実装・関連Issue・既存Ownerの操作契約を改めて確認し、未実装は`unavailable/unsupported/not-run`として報告する。架空のCLI/MCP名を生成して成功としない。実作用の権限・業務Outcomeは当該Host/Providerが確認する。

詳しいコマンド例・Sourceとfingerprintの境界は[read-onlyモデル調査の資料](references/read-only-model-investigation.md)を読む。必要なときだけ参照し、Skill本文と同じ説明を常時読み込ませない。

## 作業結果に含める内容

対象Workspace/Source改訂・使用したCLI/Go/環境の識別、対象`subject.kind/id`、実行したread-onlyコマンド、各終了状態、Fingerprintと不一致、確認できた意味・Sourceと残るUnknown、Warning/Validation finding、未実行の操作、次に必要なOwner/Provider/Host検証を必要十分な範囲で示す。**「モデル検証済み」「実行できた」「正本へ受理済み」「業務Effect確認済み」は異なるClaim**として記す。

## 関連

- [このSkillの設計と試験計画](../../plan/design/TEI_PRODUCT_SKILL_HARNESS_BINDING_2026-11.md)
- [TEI CLI利用案内](../../docs/getting-started.md)
- [TEI Product Skill課題](https://github.com/hat47x/tei-platform/issues/2605)
- [YIシリーズ横断Skill設計（TAI Draft）](https://github.com/hat47x/tai-pleroma/pull/270)

**現在の評価状態（2026-11-10）**：Portable Skillがmainに存在すること、固定版の実CLI read-only S0/S1、DSH H3-A/Bでの限定Skill本文・参照実Readはそれぞれ観測済み。一方、同一Root/権限/Sourceで隔離した独立H2/H3比較、実回答の独立Expected採点、全Tool Reply受領、現在のOS/Network開示資格、二Harness接続、Owner正式受理・実業務Effectは**NOT RUN／未成立**。[TEI #2605](https://github.com/hat47x/tei-platform/issues/2605)と[TAI #168](https://github.com/hat47x/tai-pleroma/issues/168)の実証Gateを参照する。
