# reference-tool-neutral-development

RQ-034 / ISSUE-000083（GitHub #972）の最初の実行可能な参照実装です。

目的は、TEI独自のエージェント実行環境や接続部、構造化した提案、独自の変更セット、セッション・作業領域・チェックポイントなどを使わずに、通常のリポジトリ開発だけで次の一連の作業を実行できるか確かめることです。

```text
clean TEI Repository checkout
  -> local task clone
  -> TEI CLIでcurrent meaningを読む
  -> file-native sourceを直接編集
  -> build / targeted test / validate / Canonical reload
  -> ordinary Git commit
  -> original baseから作ったseparate integration clone
  -> task commitを通常Gitでfetch / fast-forward integration
  -> rebuild / test / validate / Canonical reload
```

## Scenario

検証用プログラムは、**現在のTEIリポジトリの作業領域全体**を`git clone --local`で一時ディレクトリへ複製します。元の作業領域に変更が残っている場合は開始せず、mainや現在の作業領域も変更しません。

作業用の複製では`cmd/tei`をビルドし、既存の知識・文脈に関するテストを実行してから、参照用作業領域をコマンドラインで読み込みます。

基準となる状態には、次の規則が定義されています。

```tei
validation sales.order_line.quantity_positive positive {
    attribute sales.order_line.quantity
}
```

作業用の複製では、既存の検証規則と同じ種類の意味を使い、次の内容をファイルへ直接追加します。

```tei
validation sales.order_line.unit_price_positive positive {
    attribute sales.order_line.unit_price
}
```

この変更では、共通受理の候補形式を経由しません。提案を使う方法を否定するのではなく、元の形式のファイルを直接編集してGitで管理する開発も、正式な方法として成立することを確かめます。

## Tool固有stateを前提にしないpressure

現在の参照用作業領域には`.tei/decisions`があります。これは共通受理に関する判断の記録であり、意味の正本をもう一つ作るものではありません。

検証用プログラムは作業用の複製から別の一時作業領域を作り、そこだけで`.tei/decisions`を削除します。その状態で`validate`と`world`を再実行し、判断の記録を残した作業領域と比べて、検証の出力と意味の正本となる全体モデルが、それぞれバイト単位で一致することを要求します。

この検査では`.tei/plugin-lock.json`を削除しません。元の作業領域にプラグインの固定情報があり、一時作業領域へ複製した後もバイト単位で同じであることを確かめます。プラグインの選択を固定する情報と、意味の第二の正本ではない判断記録では、役割が異なるためです。

## 検査内容

検証用プログラムでは、少なくとも次の条件を確認します。

- 元のリポジトリがGitの作業領域であり、変更が残っていない
- 作業環境が現在のリポジトリ全体をローカルで複製したものである
- 作業用の複製内で`cmd/tei`をビルドできる
- 知識・文脈に関する対象を絞ったテストを実行できる
- `workspace-files`が`semantic/sales/validation.tei`を管理対象のファイルとして列挙する
- 基準となる作業領域で`tei validate`が成功する
- 既存の検証規則について、`source-index`から`semantic/sales/validation.tei`まで入力元をたどれる
- 既存の検証規則を`knowledge-context`から参照できる
- `.tei/decisions`を削除しても検証結果と意味の正本となる全体モデルが変わらない
- `.tei/plugin-lock.json`が存在し、判断の記録を削除する前後でバイト単位の一致を保つ
- ファイルを直接編集した後も`tei validate`が成功する
- 追加した検証規則についても、`source-index`から同じ記述元をたどれる
- 追加した検証規則を`knowledge-context`から参照できる
- 意味の正本となる全体モデルが基準状態から実際に変化する
- ファイル編集後も関連する対象を絞ったテストを実行できる
- コミット前に変更したファイルが意図した一つだけである
- コマンドライン・ビルド・テストの実行でリポジトリ内に別の未追跡の生成物が残らない
- 通常のGitコミットで変更を確定できる
- 統合側は元のリポジトリから作る別のローカル複製であり、作業用の複製そのものを複製しない
- 統合側の複製が作業側と同じ基準コミットから始まり、作業結果のコミットを通常のGitの取得と早送りで取り込む
- 統合後の先頭コミットが作業結果のコミットと一致する
- 統合側の複製でもコマンドラインをビルドし、関連する対象を絞ったテストを実行できる
- 統合側の複製でも、追加した検証規則の入力元索引から同じ記述元へ戻れる
- 統合側の複製からも同じ検証規則を参照できる
- 統合側の検証出力と意味の正本となる全体モデルが、作業側の結果とそれぞれバイト単位で一致する
- 読み取り専用で再取得した後も統合側の複製に変更が残らない

## Git mergeとSemantic conflict

別のnegative caseでは、current Repositoryから独立したcloneを作り、二つのbranchが別fileへ同じCanonical Data identityを追加します。

```text
branch A: semantic/sales/tool-neutral-collision-a.tei
        \
         +-- Git merge succeeds
        /
branch B: semantic/sales/tool-neutral-collision-b.tei

merged workspace
  -> tei validate
  -> DUPLICATE_SEMANTIC_ELEMENT
```

各ブランチだけなら検証に通ることを要求します。二つのファイルはパスが異なるため、Gitでテキストを統合しても競合しません。統合後にはじめて意味上の識別子が重複し、既存の意味検証がそれを拒否することを確かめます。

TEIがGitの統合機能を置き換えるのではありません。**Gitで正常に統合できた場合も、意味上の不整合があればTEIが検出して停止できること**が、この検査の目的です。

## Qualification

`run.sh`はRQ-034を検証する実行可能な参照実装です。ただし、コードを変更したプルリクエストをmainへ統合してよいかどうかは、この結果だけでは判断できません。

この資格確認の手順では、コードを変更した候補について、**対象となる正確なコミットで`tools/repository-health-check.sh`の全検査に合格し、前後に変更が残っていないこと**を求めます。ここで「変更がない」とは、追跡済み・未追跡のファイルだけでなく、`.gitignore`などで無視した生成物も残らず、`assume-unchanged`や`skip-worktree`のように追跡中の変更を隠せる指定も使われていない状態です。この専用手順の条件と、一般のプルリクエストに適用する変更範囲に応じた検証条件は区別します。

条件を取り違えないよう、検証を呼び出す小さなスクリプトを用意しています。

```bash
tools/reference-tool-neutral-development/qualify.sh \
  --expected-head <candidate-sha> \
  --expected-main <current-main-sha>
```

このスクリプトは次の順番で確認し、条件を満たせなければ処理を中止します。

```text
tracked / untracked / ignored clean pre-state
  -> exact candidate HEAD
  -> optional origin/main binding
  -> expected main is an ancestor of candidate HEAD
  -> full repository-health-check.sh
  -> HEAD / tracked / untracked / ignored clean-state再確認
  -> run.sh
  -> HEAD / tracked / untracked / ignored clean-state再確認
  -> candidate / main bindingを含むPASS receiptをstdoutへ表示
```

`--expected-main`を指定する場合は、先にリモートの情報を再取得し、`origin/main`がその時点のmainを指すことを確認します。さらに、そのコミットが候補の先頭コミットの祖先であることも要求します。これにより、ローカルの`origin/main`だけが正しくても、候補ブランチに最新のmainが含まれない状態では合格としません。スクリプト自身はネットワークへ接続したり参照情報を書き換えたりしません。

`--expected-main`によって、ローカルの`origin/main`の値と、候補の履歴にそのコミットが含まれることを確認します。ただし、リモートのmainを更新できないよう固定するものではありません。検査後にmainが更新されたら、以前の合格結果をそのまま利用せず、リモートの最新状態を取得して候補へ統合し、資格確認をやり直します。

このスクリプト自身が実行前後の追跡済み・未追跡・無視対象の状態を確認するため、呼び出し元による事前検査だけに依存しません。

合格時には、`candidate-head`、`origin-main`、`candidate-contains-expected-main=true`と、リポジトリ全体・参照実装・資格確認の結果を標準出力へ返します。これはTEI製品の永続的な状態ではなく、その時点のGitのコミット間の関係と検査結果を示す記録です。必要な引継ぎの際に、通常の課題やプルリクエストの検証根拠として保存できます。


### Gitの履歴・checkout境界の負例（#972 / #1173）

資格確認と統合前の事前検査では、変更済み・未追跡・無視対象のファイルや索引の指定だけでなく、**Gitが実際にどのソースとコミットの履歴を読み取ったか**も確認します。Gitの`refs/replace`や旧形式の`info/grafts`は、表示されるコミットの照合値を変えずに、親子関係やファイル構成を変えられます。`git merge-base --is-ancestor`だけでは、偽の祖先関係を見逃すおそれがあります。

- 対象となる作業領域の実際の起点が、呼び出し元の指定と一致しなければ拒否します。Gitの`core.worktree`で別の変更のないディレクトリへ読み替えても認めません。
- `GIT_DIR`、`GIT_WORK_TREE`、`GIT_INDEX_FILE`、`GIT_COMMON_DIR`など、オブジェクト保存先・名前空間・履歴の差し替え・設定の注入につながる環境変数の上書きを拒否します。検証中は`GIT_NO_REPLACE_OBJECTS=1`と、読み取り専用のGit呼び出しに付ける`--no-replace-objects`を使います。
- `refs/replace`や`info/grafts`があれば**処理を継続しません**。候補と統合側の作業領域で、実行開始時と終了時に確認します。資格確認中は、リポジトリ固有の`fsmonitor`を実行しません。
- 検証用プログラム自身と合格記録が対象とする版の対応、開始・終了時に変更がないこと、リモートの最新mainの確認は従来どおり維持します。

`qualification_test.go`では、試験用のGitリポジトリに履歴の差し替え・接ぎ木・作業領域のすり替え・環境変数の上書きを加え、資格確認と統合前の検査が**両方とも拒否すること**を回帰テストで確かめます。これはGitの動作を模した試験であり、実際のTEIリポジトリ全体に対する検査の合格記録ではありません。

### machine-readable qualification receipt

通常Git mergeによる統合でもGateを取り違えないため、`--receipt-file`を指定すると、すべてのGateが成功した後だけmachine-readable receiptをRepository外へ出力します。

```bash
CANDIDATE_HEAD="$(git rev-parse HEAD)"
TESTED_MAIN="$(git rev-parse origin/main)"
RECEIPT_DIR="$(mktemp -d)"
RECEIPT="$RECEIPT_DIR/rq034-qualification.receipt"

tools/reference-tool-neutral-development/qualify.sh \
  --expected-head "$CANDIDATE_HEAD" \
  --expected-main "$TESTED_MAIN" \
  --receipt-file "$RECEIPT"
```

receiptは少なくとも次を固定します。

- exact candidate HEAD / candidate tree
- exact `origin/main` / expected main
- candidateがexpected mainを含むこと
- tracked / untracked / ignored clean pre/post
- `assume-unchanged` / `skip-worktree`等、tracked内容を隠せるindex flagが無いこと
- full Repository Health PASS
- RQ-034 Reference PASS
- `repository-health-check.sh` / `qualify.sh` / `run.sh` / `integration-preflight.sh` のcandidate上のGit blob SHA
- Git / Go / Python version

receiptはRepository内へ書けず、出力先はまだ存在しないpathでなければなりません。既存receiptの上書きやsymlink置換を避けるため、すべてのqualification検査を終えた後にだけ、同じdirectory内の一時fileからhard-linkで原子的に公開します。receiptを可視化した後に失敗し得るqualification判定を置かず、「失敗runがPASS形状のreceiptを残す」経路を作りません。receipt出力自体によってqualification checkoutをdirtyにしません。

### integration preflight

統合直前には、**資格確認済みcandidateのclean checkout**と、receiptが固定したexact mainをHEADに持つ**別のclean integration checkout**を用意します。preflight自身がcandidateで初めて追加・変更される場合でも自己適用できるよう、preflightはcandidate側のfileを実行し、`--integration-root`で統合側checkoutを明示します。

```bash
CANDIDATE_CHECKOUT=/path/to/qualified-candidate
INTEGRATION_CHECKOUT=/path/to/clean-integration-main

git -C "$INTEGRATION_CHECKOUT" fetch --prune origin main
git -C "$INTEGRATION_CHECKOUT" switch --detach "$TESTED_MAIN"

"$CANDIDATE_CHECKOUT/tools/reference-tool-neutral-development/integration-preflight.sh" \
  --receipt-file "$RECEIPT" \
  --candidate "$CANDIDATE_HEAD" \
  --expected-main "$TESTED_MAIN" \
  --remote origin \
  --integration-root "$INTEGRATION_CHECKOUT"
```

preflightはmerge / commit / push / ref更新を行いません。ただし、remote-tracking refが古いままでも通らないよう、`git ls-remote`で指定remoteのlive `main`を開始時と終了時にread-only確認します。次をfail closedで照合します。

- integration checkoutのHEAD == expected main
- local `origin/main` == expected main
- live remote `main` == expected main（開始時 / 終了時）
- candidate HEAD / tree == receiptのcandidate
- candidateがexpected mainを含む
- receiptの全PASS条件
- receiptに記録された4つのrunner blob == candidate上の現在のblob
- 実行中の`integration-preflight.sh`自身 == candidate上のpreflight blob
- candidate checkoutとintegration checkoutの双方がtracked / untracked / ignored cleanで、tracked内容を隠せるindex flagも持たない
- preflightの開始から終了までcandidate HEAD / candidate ref / integration HEAD / local `origin/main` / live remote `main`が不変

mainがreceipt取得後に動いた場合、local remote-tracking refが古い場合、candidateやrunnerが変わった場合、preflight自身がreceipt取得時と異なる場合、receiptを書き換えた場合、またpreflight実行中にreceipt内容が変化した場合は失敗します。その場合は新しいmainをcandidateへ取り込み、**新しいexact candidateでqualificationから取り直します**。

preflightがPASSした後の統合対象は、`--candidate`に渡したbranch名をもう一度解決した結果ではなく、preflightが出力した**exact `candidate-head` SHA**です。branch refがpreflight後に動いても別commitを誤統合しないため、通常Gitのfetch / merge / pushでもこのSHAを引継ぎます。preflightはremoteをlockしないため、その後remote mainが進んだ場合はnon-force pushのnon-fast-forward拒否を安全網とし、最新mainを取り込んだcandidateでqualificationから取り直します。

このreceiptは暗号学的な署名やGitHub branch protectionの代替ではありません。人間・AIが通常Gitで統合するときに、別revisionの成功記録を誤って流用しないための機械的なhandoffです。TEI Product state、Agent Session、独自Change Setへ昇格させません。

### 回帰テスト

`qualification_test.go` は、actual TEI semanticsを模倣せず、qualification receipt / integration preflightのGit制御だけを合成Repositoryで回帰検証します。

- exact candidate / candidate tree / expected mainを持つv3 receipt
- exact candidate + live mainのpreflight成功
- 既存receiptの非上書き
- candidate tree改ざんの拒否
- local `origin/main`が古いままlive remote `main`だけ進んだ状態の拒否

このテストはfull `go test ./...`へ含まれるため、将来のreceipt/preflight変更で境界が退行すればRepository Healthから検出できます。ただし、合成Git repository上のmechanics testであり、actual TEI CLI / Canonical meaning / RQ-034 E2EのEvidenceではありません。


## Run

```bash
bash tools/reference-tool-neutral-development/run.sh
```

runner自身はremote Git operation、GitHub Actions、developer-facing MCP serverを必要としません。Repository内のlocal Git clone、Go toolchain、通常CLIだけを利用します。

ただし、Go moduleがlocal module cacheに存在しない環境では、通常の`go build` / `go test`と同様にdependency取得が必要になる可能性があります。offline hermetic buildをRQ-034の成立条件とはしません。

MCP parityはISSUE-000083 E3で後続検証します。MCPを追加する場合も、CLIと別のCanonical解釈やAuthority modelを作りません。

## Non-claims

このReferenceだけでは次を主張しません。

- ChatGPT / Codex / Claude Code固有integrationが完成した
- developer-facing MCP serverが完成した
- すべてのTEI変更が直接file editだけで十分である
- Shared Acceptance / Semantic Diff / Reviewが不要である
- local Git write authorityがApplication Runtime authorityを与える
- production-grade CI / sandboxが完成した
- `run.sh`単体のPASSだけでcode-changing PRのmerge authorityが得られる
- offline hermetic Go buildが完成した
- TEIが独自のGit merge engineを必要とする
- Decision ledgerが不要である

このReferenceが示すのは、**Agent / Acceptance固有の非正本stateをCanonical meaningの第二正本にせず、SourceとGitを境界にして開発環境を交換できるか**という最小の一往復です。現時点で除去して実証する対象はDecision ledgerであり、Plugin Lock等の別責務のstateまで不要だとは主張しません。
