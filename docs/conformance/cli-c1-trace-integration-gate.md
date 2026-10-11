# CLI C1 trace integration: 検証対象と未確定事項

対象: GitHub Issue #1165 / Draft PR #1172（`feat/cli-c1-typed-trace-integration`）。この記録は**検証指示**であり、PASS receiptではない。

## 検証対象

- 共通Go操作: `operations.InspectSubject`、`operations.TraceWorkspace`。CLIはarg/JSON/終了コードの薄いAdapter。
- 出所の意味: `worldFingerprint` と `sourceIndexFingerprint` は独立。継続用 `result.snapshotFingerprint` はModelRef Graph/Rule/検索対象/方向/World/Source Indexから求め、`result.trace.snapshotFingerprint`（Graph限定）とは区別する。
- 入力契約: `trace` と `inspect` の未知・不適切なフラグ、型不正、対象不存在・曖昧、古いfingerprintは機械可読JSONで失敗する。
- 正規化: GraphのNode/Edge/EvidenceとSource Indexの並び順に依存しないfingerprint。ただし重複宣言・重複edgeは消さず検出対象とする。
- Source境界: 同じWorldを保った元ファイル移動・非意味的編集でもSource側tokenが変わる。参照不能や逸脱pathはfail-closed。Source Indexを編集権限やShared Acceptanceへ昇格させない。

## ローカル引き継ぎ：実行コマンド

GitHubの認証情報を利用できるGo 1.23.2以降のローカル環境で、**既存の開発作業ツリーとは別の新規checkout**を使う。PR #1172のhead SHAは更新され得るため、以下のコマンドで都度取得する。実際のGitHub URLは認証済みのHTTPSまたはSSHを使用する。

```bash
git clone --no-checkout git@github.com:hat47x/tei-platform.git /tmp/tei-cli-c1-qualify
cd /tmp/tei-cli-c1-qualify
git fetch origin refs/heads/feat/cli-c1-typed-trace-integration refs/heads/main

PR_HEAD="$(git ls-remote origin refs/heads/feat/cli-c1-typed-trace-integration | cut -f1)"
MAIN_HEAD="$(git ls-remote origin refs/heads/main | cut -f1)"
git checkout --detach "$PR_HEAD"
bash tools/verify-cli-c1-local.sh "$PR_HEAD" "$MAIN_HEAD" "/tmp/tei-cli-c1-receipt-$PR_HEAD-$(date -u +%Y%m%dT%H%M%SZ)"
```

`tools/verify-cli-c1-local.sh` は、exact PR head、remote main、クリーンなdetached checkoutを実行前後に照合し、既存の **`tools/repository-health-check.sh` を省略なく呼ぶ**。証跡 `receipt.txt` と `repository-health.log` はリポジトリ外に置き、作業ツリーを汚さない。**既存のreceiptは上書きしない**ので、実行ごとに新しい証跡ディレクトリを指定する。

**exit=0でもマージ権限の自動発行ではない。** 親PR依存（#1139→#1171、#1159→#1169→#1172）と、actual mainへの統合ベース、PRがDraftでなくなったか、candidate/mainのexact-SHAを再確認する。PASS receiptとこれらの条件を揃えるまではmainへ統合しない。

## 信頼できる検証環境での手順

1. GitHub認証済みのGo 1.23.2以降が利用可能な環境で、**最新のPR #1172 head SHAとcurrent `main` SHAを記録**する。親PR #1169など並行PRのheadも確認し、対象ブランチ名の再解決だけで判断しない。
2. 新しいクリーンなcheckoutを取得する（共有ワークツリーや未コミット修正済みディレクトリを用いない）。PR headの**正確なコミットSHA**でdetached checkoutし、`git rev-parse HEAD` と `git status --porcelain` を保存する。期待SHAに一致しなければ停止する。
3. `GOENV=off GOWORK=off GOFLAGS= tools/repository-health-check.sh` を**省略なし**で実行する。これは`gofmt -l .`、`go vet ./...`、`go test -count=1 -timeout 60m ./...`、Reference Workspace検証を含む。標準出力・標準エラー、exit、検証開始/終了のSHAとclean状態を保管する。
4. 問題を検出した場合、**PR #1172 branch上に通常コミット**で修正する。PR headが動いたら既存PASS証跡はstaleとして破棄し、同じ手順で再検証する。共有branchのforce/rebase/resetや直接main変更はしない。
5. full PASS後、PRや各親PRの依存、現在mainの差分、正本ドキュメントと必要なGateを確認する。**Draftのまま統合しない**。マージ直前にexact PR head、base、current mainとPASS証跡の一致を再確認する。

### 検証候補（full Repository Healthの代替ではない）

```bash
go test -count=1 ./internal/modelref/... ./internal/sourceindex/... ./internal/operations/... ./internal/workspace/... ./cmd/tei/...
go run ./cmd/tei trace -workspace reference/workspace -scope reference-observation -subject-kind semantic-relation -subject-id sales.order.customer -mode summary
go run ./cmd/tei inspect -workspace reference/workspace -subject-kind semantic-relation -subject-id sales.order.customer
```

ページ継続時は`trace.result.snapshotFingerprint`を`-expected-fingerprint`として使用する。`trace.result.trace.snapshotFingerprint`は継続用のWorkspace tokenではない。

## 検証していない点

- この環境からのGitHub直接checkoutはDNS解決に失敗しており、**exact-headに対するgofmt/ビルド/vet/test/full Repository HealthのPASSは無い**。
- Source読取は毎回Source Indexのdigestと範囲を検証するが、モデルload・source索引化・filesystem readを単一のOSトランザクションとして固定しない。symlink-swap raceを含む#1017の課題は別途扱う。
- Reference Fixtureと合成Workspaceを用いるテストはProvider実動・実環境での書込み権限・人間による意味の採用の証拠にならない。
- `full` は明示ページ分割しない場合、全候補参照を組み立てる。大規模Workspaceに対する性能・最大メモリの実測は未取得。走査予算による黙った打切りは行わない。

この文書はGate判断の正本を増やすものではなく、`AGENTS.md` / `CONTRIBUTING.md` と対象Issueの合格条件を適用するための補足手順である。

## 設計知見・未解決課題の恒久参照

- **#1165（Issue本文）**：採用したCanonical/Source/Typed Graphの責務、Scopeの非権威性、複合tokenの使い分け、五つの初期kind、重複宣言と不完全Evidenceの扱い、CLIとGoのAdapter分離、未取得のGo品質Gate証拠、親PR依存を集約する。
- **#1017**：Source Readerのsymlink/TOCTOU。C1のhash＋digest＋path containment検証はatomic snapshotを保証しない。
- **#1177**：large Workspaceのページングと内部全量走査・Source Index再検証・メモリ使用量の分離。黙った走査予算による欠落を禁止し、全体テキスト出力後の一括解析案も評価する。
- **#1178**：`read-only`を機密情報開示の認可へ昇格させない。Canonical/Source path/edge/unresolved/近傍情報のLLM送出と既存Disclosure機構を接続する。
- **#1163**：one-revision end-to-end Acceptance/Target/Observation Evidenceとの区別。CLIの読取tokenは実行や書込権限の証拠ではない。

C1の品質Gateが通った後も#1017/#1177/#1178/#1163は**独立した未解決課題**として残す。C1 PRへ無制限に取り込んで完了を遅らせない。
