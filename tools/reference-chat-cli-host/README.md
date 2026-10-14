# 信頼済み開発HostのE0事前診断 (#1173)

この参照実装は、ChatGPTから実行先を選ぶ前に、指定したリポジトリのコミットの関係を、既存のGit・Go環境で確認できるかを調べます。TEI独自のエージェント実行環境、シェル命令の実行機構、資格判定機構を追加するものではありません。

## 実行例

    go run ./tools/reference-chat-cli-host --root . --expected-head CANDIDATE_SHA --expected-main MAIN_SHA

出力は機械が読み取れるJSON（tei.dev-host-probe/v0）です。正常終了は終了コード0、前提条件を満たさない場合は1、引数が不正な場合は2を返します。実際のGit作業領域を対象とし、GitとGoの有無、リポジトリの起点、先頭コミット、ローカルのorigin/main、mainとの祖先関係を検査します。追跡対象・未追跡ファイルの変更の有無、無視対象の生成物、変更を隠すassume-unchangedとskip-worktreeの指定も**別々に**確認します。状態だけを返し、ファイル名、リモート接続先、認証情報は出力しません。

eligibleForLocalGate=trueが意味するのは「ローカルで資格確認を試すための最低限の条件を満たす」ということだけです。課題 #972 の検査に代わる変更なしの証拠や、GitHubへの実際の接続と認証、リモートにある最新のmain、Goの全テストを確認したわけではありません。JSONではliveRemoteMainとcredentialsが常にnot-checked、repositoryHealthがnot-runとなります。リモートのURLや認証情報は出力しません。また、引数に含まれる任意のシェル命令を実行しません。

## 次段

ホストが信頼できるGitHub接続を通して新しい作業領域と最新のorigin/mainを取得し、別途必要な権限が明示されている場合に限り、既存のtools/reference-tool-neutral-development/qualify.shを使用します。これはリポジトリ全体の検査とRQ-034を実行するための正式な資格確認用の参照実装であり、この事前診断で置き換えることはできません。合格記録は候補の先頭コミットとmainのコミットに結び付け、mainが更新されたら取得し直します。

関連するテストはgo test ./tools/reference-chat-cli-hostです。隔離して作成したGitリポジトリを使い、正常終了、先頭コミットの不一致、origin/mainの欠落や不一致、祖先関係の不一致、誤ったサブディレクトリ、未知の引数、リモートURLに含まれる秘密情報を出力しないことを確認します。実際のGitHub通信やTEIリポジトリ全体を取得した状態での検証ではありません。

Gitの`refs/replace`や旧形式の履歴の接ぎ木は、表示されるコミットの照合値を変えずに祖先関係を変えられるため拒否します。Gitの呼び出しには`--no-replace-objects`を使い、リポジトリが指定した`fsmonitor`は実行しません。取得元を信頼できない場合は、シェルで内容を実行するより前に、別の経路で作業領域と実行資格を確認する必要があります。

Gitへの読み取り専用の問い合わせには、一回当たり15秒の時間制限と、標準出力8 MiBの上限を設けます。制限を超えたり処理が取り消されたりした場合は、**必要な情報をすべて取得できなかった失敗**とします。一部の出力だけを使って状態を判定しません。標準エラー出力にも認証情報が含まれ得るため、報告に混ぜません。この制限は、`qualify.sh`による全体検証の制限や記録の保管方法に代わるものではありません。

実行環境側に`GIT_DIR`、`GIT_WORK_TREE`、`GIT_INDEX_FILE`、別のオブジェクト保存先、Gitの設定を差し替える指定などがある場合、`-C`で指定した作業領域とは別の場所を参照させられる危険があります。そのため、`ambientGitOverrides=present`として停止します。子のGitプロセスには`GIT_NO_REPLACE_OBJECTS=1`を明示します。ただし、この検査だけではホストの認証や最新のリモートへの接続を証明できません。

## E1のHost側handoff手順（未実行）

これは**信頼できるホストでだけ**実行する手順です。ChatGPTの接続機能が、そのホストのシェルを既に操作できることを意味しません。非公開リポジトリへの認証設定はホスト側で行い、トークンをチャットへ貼り付けないでください。

```bash
set -euo pipefail
REPO=hat47x/tei-platform
BRANCH=work/issue-988-operator-pin-convergence   # 初回consumer、別PRなら置換
CHECKOUT="$(mktemp -d)"
EVIDENCE="$(mktemp -d)"

# 認証済みGit transportが前提。CLIが使えることとアクセス権は別。
git clone --no-checkout "https://github.com/$REPO.git" "$CHECKOUT"
git -C "$CHECKOUT" fetch --prune origin main "$BRANCH"
MAIN_SHA="$(git -C "$CHECKOUT" rev-parse --verify refs/remotes/origin/main)"
CANDIDATE_SHA="$(git -C "$CHECKOUT" rev-parse --verify "refs/remotes/origin/$BRANCH")"
git -C "$CHECKOUT" switch --detach "$CANDIDATE_SHA"

# Candidateへ最新mainが未統合なら資格確認を進めない。
git --no-replace-objects -C "$CHECKOUT" merge-base --is-ancestor "$MAIN_SHA" "$CANDIDATE_SHA" || {
  echo "candidate does not contain current main; integrate in its work branch first" >&2
  exit 1
}

# E0 probeがこのCandidateに存在する場合だけ任意で先行実行する。
if [[ -f "$CHECKOUT/tools/reference-chat-cli-host/main.go" ]]; then
  (cd "$CHECKOUT" && go run ./tools/reference-chat-cli-host \
    --root . --expected-head "$CANDIDATE_SHA" --expected-main "$MAIN_SHA")
fi

# 既存#972 runnerが本物のテストを実行する。出力とreceiptはRepo外。
(cd "$CHECKOUT" && tools/reference-tool-neutral-development/qualify.sh \
  --expected-head "$CANDIDATE_SHA" \
  --expected-main "$MAIN_SHA" \
  --receipt-file "$EVIDENCE/qualification.receipt") \
  >"$EVIDENCE/qualification.log" 2>&1
```

ここまで成功しても、**その時点の候補コミットとmainについて資格確認を試みられる状態**でしかありません。リモートのmainが更新されないよう固定するものではありません。統合直前には、変更のない別の作業領域と最新リモートを参照できる環境で、既存の`integration-preflight.sh`を実行します。mainが進んだ場合は、通常のGitの統合で候補ブランチへ取り込み、新しい先頭コミットで資格を確認し直します。

失敗した場合はログ・終了コード・候補コミットを保存して停止し、**失敗の記録を合格記録へ書き換えません**。ログにはローカルのパスや環境情報が含まれ得るため、外部の大規模言語モデルやGitHubへ送る前に、既存の情報開示規則に従って確認します。認証情報やリモートのURLに含まれる秘密はそのまま添付しません。ホストへの接続、必要な権限、リポジトリの取得、CLIの実行、検証根拠の返却は、このリポジトリのソースだけでは有効になりません。

