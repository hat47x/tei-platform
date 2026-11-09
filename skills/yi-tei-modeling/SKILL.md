---
name: yi-tei-modeling
description: >-
  TEIの既存Workspace・モデル・Relation・Sourceを、実在するread-only CLIで
  発見、検証、参照追跡するときに使う。対象版と診断、未解決を保持する。
  TEIでの書込み・モデル受理・生成物の実配布を暗黙に行わない。
---

# TEIモデリング調査（Product Skill）

> **2026-11-05 検証状態**：このProduct Skillの文書・参照資料は[merged PR #2607](https://github.com/hat47x/tei-platform/pull/2607)によってmainに統合済み。[GitHub Actions #41532988114](https://github.com/hat47x/tei-platform/actions/runs/41532988114)では、対象版 `d0de429ae5aac38b7949c2cfe6eacafff059675f` の公開 `reference/workspace` について、ビルド済みの実TEI CLIから `commands`・`model catalog`・`inspect`・`trace`・`validate` を読み取る**限定S1試験**が成功した。一方、外部AI HarnessへのSkillインストール・実ロード・実呼出し、MCP/Provider/業務Effect、Ownerの正式な意味受理は**NOT RUN／未受理**。他の版や任意Workspaceへ試験成功を一般化しない。
> **2026-11-05 実行資格の更新**：このSkillの初期ファイルは[PR #2607](https://github.com/hat47x/tei-platform/pull/2607)でmainへ統合済み。対象HEAD `d0de429ae5aac38b7949c2cfe6eacafff059675f`の[Product Skill CI #41532988114](https://github.com/hat47x/tei-platform/actions/runs/41532988114)では、静的チェッカーと、公開Reference Workspaceに対する実Goバイナリの`commands`／`model catalog`／`inspect`／`trace`／`validate`が限定的に成功した。一方、**外部ハーネスでのSkill実ロード・Tool呼出し、実OS/Provider権限・開示、業務Owner受理と正式Releaseは未実証**。利用時は現在の実環境とSource対象を改めて確認する。

## いつ使うか

TEIのSource・モデル要素・入力元・参照関係・Validationを調べ、結果を再現可能な根拠とともに説明したいときに用いる。TEIの単独利用にTAIは必要ない。複数製品をまたぐ活動の進行はTAI横断Skillが担当する。

**このSkillはTEI Kernel Plugin、TEIで開発したLibrary、外部Harness Plugin、MCP Serverそのものではない。** TEIで生成したLibraryを外部Java/.NET/Wasm等が利用する際に、このSkill/TEI CLIを配布依存にしない。

## 実行の流れ

1. **実環境を確認**：TEIリポジトリのcheckoutとWorkspaceの所在・利用目的・利用可能な情報範囲を確認する。`go.mod`の`go`行の要求版と`go version`の出力を照合し、不足なら実行を止めて環境不備として記録する（TEIの結果と区別する）。CLIの使用可能なread-only入口を`go run ./cmd/tei commands`で確認する。ビルド済み実行ファイルがあれば`tei version -json`などで版を照合し、Gitの改訂・Working Treeを別記する。**発見したCLIと実際の呼出しで使用した実行物（path/bytes/argv/cwd）を区別する**。必要な場合は固定したビルド成果物を使い、異なる実行物を同じ試行と見なさない。`go run`からVCS revisionが得られるとは仮定しない。
2. **対象を発見**：`go run ./cmd/tei model catalog -workspace <workspace>`の`entries[].subject.kind/id`から対象候補を選び、重複・曖昧なら特定を保留する。名前や一覧順を正本IDとして推測しない。
1. **実環境を確認**：TEIリポジトリのcheckoutとWorkspaceの所在・利用目的・利用可能な情報範囲を確認する。CLIの使用可能なread-only入口を`go run ./cmd/tei commands`で確認する。ビルド済み実行ファイルがあれば`tei version -json`などで版を照合し、Gitの改訂・Working Treeを別記する。**発見したCLIと実際の呼出しで使用した実行物（path/bytes/argv/cwd）を区別する**。`go run`例は探索用であり、**比較試験の実行物固定**には一度ビルドした同一バイナリを全コマンドで使い、その実bytesとGit/Go/Workspace改訂を別々に記録する。ビルド済バイナリの版表示だけでビルド入力全体の閉包を保証しない。CLI使用中にSource/Rootが変化した場合は同一試行として混ぜず、現在の実Sourceを再照合する。`go run`からVCS revisionが得られるとは仮定しない。
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

**現在の評価状態**：Product Skillのmain統合と、公開Workspaceでの実ビルド済TEI CLIの限定read-only S1試験は確認済み（上記のexact HEAD／CI）。しかしSkill本文を読んだAIが**実際にToolを呼べる／正しく使える**ことは未検証。外部Harnessの実install/load/invoke、Tool-onlyとSkill付きのH2/H3独立4 Session比較、実外部送出・全Source完全性・正式Release/Owner受理はいずれも未実施。比較のOwnerは[TAI #168](https://github.com/hat47x/tai-pleroma/issues/168)／[Draft #270](https://github.com/hat47x/tai-pleroma/pull/270)。
**現在の評価状態（2026-11-05）**：mainにある試作Skillの静的チェッカーと公開Fixtureの**実CLI単体CIは限定成功**。その実行はSkillを外部ハーネスからロード・呼出した試験ではない。次の資格は、必要な利用者定義Sourceが実解析済みであることの確認、同じTaskでのH2 Tool-only／H3 Tool＋Skillの実四Session比較、独立Expectedによる採点、適切な現在開示・Tool権限の確認である。二Harness互換、実外部Host/Provider、業務Owner受理、正式Releaseは**NOT RUN**（[TEI #2605](https://github.com/hat47x/tei-platform/issues/2605)／[TAI #168](https://github.com/hat47x/tai-pleroma/issues/168)）。
