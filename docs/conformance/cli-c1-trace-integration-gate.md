# CLI C1の追跡統合：検証対象と未確定事項

対象はGitHubの課題 #1165 と下書きのプルリクエスト #1172（`feat/cli-c1-typed-trace-integration`）です。この文書は**検証手順**であり、合格を示す記録ではありません。

## 検証対象

- 共通のGo操作は`operations.InspectSubject`と`operations.TraceWorkspace`です。コマンドライン側は、引数・JSON・終了コードを受け渡す薄い接続部とします。
- 出所の区別：`worldFingerprint`と`sourceIndexFingerprint`は独立しています。続きの取得に使う`result.snapshotFingerprint`には、参照関係のグラフ・規則・検索対象・探索方向・全体モデル・入力元索引を反映します。グラフだけに対応する`result.trace.snapshotFingerprint`とは異なります。
- 入力の検証：`trace`と`inspect`に未知・不適切な引数が渡された場合や、型が不正な場合、対象がない・特定できない場合、古い照合値が指定された場合は機械可読のJSONで失敗を返します。
- 表記の統一：グラフの節点・関係・検証根拠や入力元索引の並び順に左右されない照合値を使います。ただし、重複した宣言や関係は消さず、検出対象として残します。
- 入力元の境界：全体モデルの意味が変わらなくても、元ファイルの移動や意味を変えない編集で入力元の識別値は変わります。参照できないパスや許可範囲外のパスは拒否します。入力元索引を編集権限や共通受理の権限として扱いません。

## ローカルへの引継ぎと実行方法

GitHubへ認証済みで、Go 1.23.2以降を使えるローカル環境を用意します。**既存の開発作業領域とは別に取得したリポジトリ**を使用します。プルリクエスト #1172 の先頭コミットは変わり得るため、以下のコマンドで毎回取得します。GitHubへの接続には、認証済みのHTTPSまたはSSHを使います。

```bash
git clone --no-checkout git@github.com:hat47x/tei-platform.git /tmp/tei-cli-c1-qualify
cd /tmp/tei-cli-c1-qualify
git fetch origin refs/heads/feat/cli-c1-typed-trace-integration refs/heads/main

PR_HEAD="$(git ls-remote origin refs/heads/feat/cli-c1-typed-trace-integration | cut -f1)"
MAIN_HEAD="$(git ls-remote origin refs/heads/main | cut -f1)"
git checkout --detach "$PR_HEAD"
bash tools/verify-cli-c1-local.sh "$PR_HEAD" "$MAIN_HEAD" "/tmp/tei-cli-c1-receipt-$PR_HEAD-$(date -u +%Y%m%dT%H%M%SZ)"
```

`tools/verify-cli-c1-local.sh`は、対象プルリクエストの正確なコミット、リモートのmain、ブランチから切り離された変更のない作業領域を実行前後に照合します。既存の**`tools/repository-health-check.sh`を省略せず実行します**。結果の`receipt.txt`と`repository-health.log`はリポジトリの外へ保存し、作業領域に変更を残しません。**既存の結果は上書きせず**、実行のたびに別の保存先を指定します。

**終了コードが0でも、それだけで統合してよいわけではありません。**先行プルリクエストとの依存関係（#1139→#1171、#1159→#1169→#1172）、実際のmainとの関係、下書きの解除、候補とmainの正確なコミットを確認します。合格記録とこれらの条件がそろうまではmainへ統合しません。

## 信頼できる環境での検証手順

1. GitHubへ認証でき、Go 1.23.2以降が使える環境で、**プルリクエスト #1172 の最新コミットと、その時点の`main`のコミットを記録**します。先行する #1169 などの先頭コミットも確認し、ブランチ名を再取得した結果だけで判断しません。
2. 共有中や未コミットの変更がある作業領域は使わず、新しくリポジトリを取得します。対象プルリクエストの**正確なコミット**をブランチから切り離して取り出し、`git rev-parse HEAD`と`git status --porcelain`の結果を保存します。期待したコミットと異なる場合は停止します。
3. `GOENV=off GOWORK=off GOFLAGS= tools/repository-health-check.sh`を**省略せず**実行します。これには`gofmt -l .`、`go vet ./...`、`go test -count=1 -timeout 60m ./...`、参照用作業領域の検証が含まれます。標準出力・標準エラー・終了コード・実行前後のコミット・変更の有無を記録します。
4. 問題が見つかった場合は、**プルリクエスト #1172 の作業ブランチへ通常のコミットで修正**します。先頭コミットが変わったら以前の合格記録は現在の版には使えないため、同じ手順で検証し直します。共有ブランチの強制更新・履歴の付け替え・巻き戻しや、mainへの直接変更は行いません。
5. 全項目の合格後も、対象と先行プルリクエストの依存関係、現在のmainとの差分、正本の文書、必要な判定段階を確認します。**下書きのまま統合してはいけません**。統合直前には、対象コミット・取り込み先・最新のmainと合格記録の対応を確かめます。

### 検証候補（リポジトリ全体の健全性検査の代わりにはならない）

```bash
go test -count=1 ./internal/modelref/... ./internal/sourceindex/... ./internal/operations/... ./internal/workspace/... ./cmd/tei/...
go run ./cmd/tei trace -workspace reference/workspace -scope reference-observation -subject-kind semantic-relation -subject-id sales.order.customer -mode summary
go run ./cmd/tei inspect -workspace reference/workspace -subject-kind semantic-relation -subject-id sales.order.customer
```

次のページを取得するときは、`trace.result.snapshotFingerprint`を`-expected-fingerprint`へ指定します。`trace.result.trace.snapshotFingerprint`は、作業領域の継続取得に使う識別値ではありません。

## 検証していない点

- この文書を作成した環境では、GitHubからの直接取得がDNSの名前解決に失敗しました。そのため、**対象コミットでの整形・ビルド・静的解析・テスト・リポジトリ全体の検査に合格した記録はありません**。
- 入力元の読み取りでは、入力元索引のダイジェストと読み取り範囲を毎回検証します。ただし、モデルの読み込み・入力元索引の作成・ファイルの読み取りを一つのOSトランザクションとして固定するわけではありません。シンボリックリンクのすり替えによる競合を含む #1017 は別の課題です。
- 参照用の試験データと合成した作業領域によるテストは、実際の提供元での動作、実環境への書き込み権限、人による意味の受理を示すものではありません。
- `full`では、明示的にページを分けなければ全候補の参照を組み立てます。大規模な作業領域での性能や最大メモリ使用量は実測できていません。処理量の上限を理由に結果を知らせず打ち切ることはしません。

この文書は判定段階について新たな正本を作るものではありません。`AGENTS.md`と`CONTRIBUTING.md`、対象の課題にある合格条件を適用するための補足手順です。

## 設計上の知見と未解決課題の参照先

- **#1165（Issue本文）**：採用したCanonical/Source/Typed Graphの責務、Scopeの非権威性、複合tokenの使い分け、五つの初期kind、重複宣言と不完全Evidenceの扱い、CLIとGoのAdapter分離、未取得のGo品質Gate証拠、親PR依存を集約する。
- **#1017**：入力元の読み取りにおけるシンボリックリンクと、検査時点・使用時点のずれを扱います。C1のハッシュ・ダイジェスト・パスの範囲確認でも、読み取り全体が一つの時点で行われることは保証しません。
- **#1177**：大規模な作業領域でのページ分割と内部の全量走査、入力元索引の再検証、メモリ使用量を分けて扱います。結果を知らせず省略することを禁じ、全量出力してからまとめて解析する案も評価します。
- **#1178**：`read-only`を機密情報の開示権限と同一視しません。意味の正本、入力元のパス、関係、未解決事項、周辺情報を大規模言語モデルへ渡す際の扱いを、既存の開示制御と結び付けます。
- **#1163**：一つの版について受理・実行先・観測を通した検証根拠と、CLIが読み取りに使う識別値を区別します。後者は実行や書き込みの権限を証明しません。

C1の品質検査に合格した後も、#1017・#1177・#1178・#1163は**独立した課題**として扱います。C1のプルリクエストへ範囲を際限なく広げ、完了を遅らせることはしません。
