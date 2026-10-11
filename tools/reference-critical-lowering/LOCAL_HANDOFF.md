# #1121 ローカル実行担当への引き継ぎ

対象: [Issue #1121](https://github.com/hat47x/tei-platform/issues/1121) / [Draft PR #1141](https://github.com/hat47x/tei-platform/pull/1141)  
Repository: `hat47x/tei-platform`（private）  
Work branch: `work/issue-1121-lowering-contract` / target: `main`

## 0. この引き継ぎが意味すること

チャット経由のGitHubファイル編集とAPI上でのDiff・既存ソース照合は実施した。**完全なprivate Repository Checkout、Go packageのコンパイル、Gofmt、実Web server/HTTP smoke、全Repository Health、Production Host Authority、DBの外部実行は未確認**。これまでのPASSは、隔離した`qualify.sh`/mock Git ShellのGuardテストだけである。

以下の一連の作業はローカル環境で行う。**CIは利用しない**。このファイルをGitHubへ追加したこと、また`qualify.sh`が存在すること自体はPASS証拠ではない。条件が揃うまでDraftを解除せずmainへマージせずIssueを閉じない。

## 1. 認証済み完全Checkoutと変更の確認

`go.mod`はGo 1.23.2。該当Sourceは`tools/reference-critical-lowering/`以下（この文書を除き7ファイル）。GitHubに権限のあるCloneで実行する。

```bash
git fetch origin main work/issue-1121-lowering-contract
git switch --track -c work/issue-1121-lowering-contract \
  origin/work/issue-1121-lowering-contract  # ローカルに同名Branchがない場合のみ
git status --short
git merge-base --is-ancestor origin/main HEAD || \
  echo 'BLOCK: main advanced; work branch must incorporate reviewed main'
git diff --check origin/main...HEAD
go version
```

既にローカルBranchがある場合は、勝手に消去・force resetせず差分とHEADを照合し、同期方法を判断する。remote mainが進んだ場合は、最新mainを作業Branchへ非forceで取り込み、競合を調査し、以降すべての検証を**更新後のHEAD**でやり直す。他のPR/Branchに変更を混入させない。

## 2. Go formatterを先行し、必要ならコミット

`qualify.sh`はGofmt差分が1行でもあれば最初に停止する。Reference Goファイルには、手書きインライン式など**未整形の可能性が高い箇所**がある。これはローカル担当が最初に確かめる具体的な阻害候補である。

```bash
gofmt -l tools/reference-critical-lowering/*.go
# 出力があったときだけ、作業Branch上で:
gofmt -w tools/reference-critical-lowering/*.go
git diff --check
git diff --stat
# Goファイル以外に変更がないことを確認してから:
git add tools/reference-critical-lowering/*.go
git commit -m 'style: format critical lowering reference Go sources'
git push origin work/issue-1121-lowering-contract
```

Gofmt以外の修正は、まず`go test`や`go vet`の実際のエラーメッセージを取得し、その原因に限定して修正する。修正後に**新しいHEADで再検証**する。

## 3. 完全検証と改竄防止Evidence

Repository rootで、Sourceは変更せず、ログだけはRepository**外**に保存する。外部接続・Git credential・Local Provider Secretsをログへ書かない。

```bash
git fetch origin main work/issue-1121-lowering-contract
git switch --detach origin/work/issue-1121-lowering-contract
export EXPECTED_HEAD="$(git rev-parse HEAD)"
export EXPECTED_MAIN="$(git rev-parse origin/main)"
test -z "$(git status --porcelain --untracked-files=all)"
git merge-base --is-ancestor "$EXPECTED_MAIN" "$EXPECTED_HEAD"

evidence_dir="$(mktemp -d "${TMPDIR:-/tmp}/tei1121-qualify.XXXXXXXX")"
printf '%s\n' "HEAD=$EXPECTED_HEAD" "MAIN=$EXPECTED_MAIN" \
  "UTC=$(date -u '+%Y-%m-%dT%H:%M:%SZ')" "$(go version)" \
  > "$evidence_dir/identity.txt"
set -o pipefail
EXPECTED_HEAD="$EXPECTED_HEAD" EXPECTED_MAIN="$EXPECTED_MAIN" \
  bash tools/reference-critical-lowering/qualify.sh 2>&1 \
  | tee "$evidence_dir/qualify.log"
printf '%s\n' "${PIPESTATUS[0]}" > "$evidence_dir/exit-code.txt"
```

**注意**: `set -e`が有効な実行shellの場合、失敗したpipeの直後に停止する可能性がある。ログと終了コードの採取が必要なら`set +e`を局所的に適用し、最後に結果を明示的に判定する。Scriptが`LOCAL_CHECKS_PASSED HEAD=... MAIN=...`を出力し、Exit Code=0で、前後の実remote mainが不変だった場合だけLocal Qualification PASS。Script中で`-run`の0件実行・Skipを明示PASSとは数えない。

実行項目: shell guard、GoFmt、Go Vet（当該package）、Go Unit/Integration、既存Shared Acceptance、既存Driver/Webのscratchテスト、**実起動**Web TargetのHTTP正常/認証拒否/権限拒否/実行失敗、`tools/repository-health-check.sh`（全Repositoryのgofmt/vet/test/Workspace validate）。

### 実行失敗のとき

- **Formatter**: §2でフォーマットしてコミット。変更後HEADを更新。
- **コンパイル失敗**: `go test -count=1 ./tools/reference-critical-lowering`を単独実行して初発のビルドエラーを修正。修正後再実行。
- **HTTP失敗**: 実Web binaryのBuild/port/fixture/Revision R・T/Authorization段階を切り分ける。Gate B固定R=`e5231839a5dcc62cce1181a2731fcb7c70d16736`、T=今回のHEAD。別Targetや本番Authorityの証明とは混同しない。
- **Repository Health失敗**: 全Repositoryの既存問題と#1121によるRegressionを分けて記録し、無関係なファイルをこのPRで修正しない。必要なら独立Issueへ。
- **mainが検証中に進む**: exit非ゼロで正しい。mainを取り込み、改めて全検証。
- **Networkがない**: private GitHub checkoutと依存の事前取得が必要。完全checkout・dependency cacheのない環境で成功とみなさない。

## 4. Go Testが全PASSしても残る製品受入条件

Issue #1121の受入条件は、単なる合成Referenceの検証を超える。以下をそれぞれ実行記録・trace・受入判断とともに確認する。

| 条件 | 現在の状態・必要な残作業 |
|---|---|
| 2系統の実Plugin/DSL Producer | 現在の`tei.language.dsl` / `tei.language.sql-projection`は**文字列IDだけの合成Fixture**。既存2系統の実Source→Producer実行→同等Canonical意味→異なるprovenanceの検査が必要。 |
| 選択的Cache無効化 | synthetic key unit negative controlは作成済み。実Source変更→既存Resolver/Builder cacheのbefore/after依存グラフが必要。 |
| Capability・Version・Permission fail-closed | stub Trustの負例は作成済み。既存Authority Owner、current Host Permission、Conformance/Lock、対象DB Instance Versionと照合し、実dispatch前に拒否された証拠が必要。 |
| Web E2E三視点 | 既存Source/DriverとHTTP fixtureの段階的Traceを用意。#1119のscratch `live-r1-smoke.mjs`（E1-A）は**test-only Acceptance / synthetic Provider**。実受理Receipt→同一Targetの追跡とE1-B実ルール拒否・独立Observationは未証明。 |
| 第二Target | synthetic Desktop Providerの差分テストのみ。既存の現実的な第二Targetの選択、同じAccepted意味の実消費、Target独自制約、生成・または解釈での差分Evidenceが必要。 |
| Build Artifact既存契約 #943 | 今のReferenceが参照する`BuildArtifactLock`等の実オーナーが発行するArtifact/Revision/解決証拠との結合が必要。別Authority/Lock schemaを新設しない。 |

関連Owner: 親 #1119、Capability #951、Build #943、Runtime/Target側。完了条件を子Issueに強制的に仮実装して閉じず、上流OwnerのEvidences/該当PRとの整合を確認する。

## 5. 引き継ぎ先が報告する成果

1. 対象Go version、作業Branchと最新`main`の40桁SHA、開始/終了時clean状態。
2. Gofmt変更コミット（該当する場合）とGo Test / Go Vet / Web Smoke / 全Healthの完全Exit code・ログ、再実行後の最終PASS証拠。
3. 表にある6受入条件ごとの`PASS / BLOCKED / OUT-OF-SCOPE`（ただしIssueの条件を勝手に外さない）の判定、real/synthetic別の証拠。
4. PR #1141のレビュー・差分/Authority gateの記録。最新mainの再確認。
5. **すべて揃ったときだけ**Draft解除→mainへの通常Merge→正確なMerge SHAを記録→Issue #1121をclose。未達の場合はPR Draft / Issue openで、失敗ログと担当Ownerへの依頼を残す。

この文書は、実行ができないチャット環境からローカル担当への**正確な引き継ぎ指示**であり、PASS/マージ完了を証明するものではない。
