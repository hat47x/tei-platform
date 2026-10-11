# 信頼済み開発HostのE0事前診断 (#1173)

このReferenceは、ChatGPTから実行先を選択する前に、既存のGit・Go環境で指定したリポジトリのcommit関係を読み取れるかを確認する。TEI独自Agent Runtime、Shell executor、資格判定器を追加するものではない。

## 実行例

    go run ./tools/reference-chat-cli-host --root . --expected-head CANDIDATE_SHA --expected-main MAIN_SHA

出力は機械可読JSON (tei.dev-host-probe/v0)。成功はexit 0、前提不足はexit 1、不正引数はexit 2で返す。実Git checkoutを前提とし、GitとGoの存在、ルート、HEAD、ローカルorigin/main、main祖先関係を検査する。加えてtracked/untrackedのclean状態、ignored artifact、assume-unchanged / skip-worktreeの隠蔽フラグを**独立に**確認する。状態だけを返し、ファイル名やGit remote、Credentialを出力しない。

eligibleForLocalGate=true は「ローカル資格確認を試行するための最小前提がある」という意味に限る。既存の#972資格確認に代わるclean証拠、実際のGitHubへの接続・認証、live remote main、全Goテストを確認した意味ではない。JSONのliveRemoteMainとcredentialsは常にnot-checked、repositoryHealthはnot-runである。Remote URLやCredentialは出力しない。引数から任意のShellコマンドを実行しない。

## 次段

Hostが信頼できるGitHub経路からfresh checkoutと最新origin/mainを確保し、別の明示権限を持つときだけ、既存 tools/reference-tool-neutral-development/qualify.sh を用いる。これは完全なRepository HealthとRQ-034を実行する正式な資格Referenceであり、本事前診断が置き換えない。資格receiptは候補HEADとmain SHAに固定し、最新mainが動けば再取得する。

テスト: go test ./tools/reference-chat-cli-host。合成した隔離Gitリポジトリに対して成功、HEAD不一致、origin/main欠落・不一致、祖先関係不一致、サブディレクトリ誤指定、未知引数、Remote URL中のsecret非出力を検査する。実GitHub通信・実TEIフルcheckoutの検証ではない。

Gitの`refs/replace`やlegacy graftsは、表示上のコミットSHAを変えずに祖先判定を変更できるため拒否します。Git呼び出しは`--no-replace-objects`で保護し、リポジトリが指定した`fsmonitor`は実行しません。元ソースが未信頼なら、そもそもShellで内容を実行する前に別経路でCheckoutと資格を検査する必要があります。

Gitのread-only問い合わせには1コマンドあたり15秒のtimeoutと8MiBのstdout上限を設ける。超過やキャンセル時は**完全な観測を得られなかった失敗**とし、部分出力を状態判定へ利用しない。Gitのstderrも資格情報を含み得るため報告へ混ぜない。この上限も、`qualify.sh`による全体検証のtimeoutやログ保管契約を置き換えない。

実行環境側の`GIT_DIR`、`GIT_WORK_TREE`、`GIT_INDEX_FILE`、代替object store、Git configuration injection等が設定されていれば、`-C`で別checkoutを指定したように見せかける危険があるため`ambientGitOverrides=present`で停止する。子Gitプロセスへ`GIT_NO_REPLACE_OBJECTS=1`を明示する。この検査もHost認証と最新remoteへの接続を証明するものではない。
