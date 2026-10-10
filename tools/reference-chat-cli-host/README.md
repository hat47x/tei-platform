# 信頼済み開発HostのE0事前診断 (#1173)

このReferenceは、ChatGPTから実行先を選択する前に、既存のGit・Go環境で指定したリポジトリのcommit関係を読み取れるかを確認する。TEI独自Agent Runtime、Shell executor、資格判定器を追加するものではない。

## 実行例

    go run ./tools/reference-chat-cli-host --root . --expected-head CANDIDATE_SHA --expected-main MAIN_SHA

出力は機械可読JSON (tei.dev-host-probe/v0)。成功はexit 0、前提不足はexit 1、不正引数はexit 2で返す。実Git checkoutを前提とし、GitとGoの存在、ルート、HEAD、ローカルorigin/main、main祖先関係を検査する。

eligibleForLocalGate=true は「ローカル資格確認を試行するための最小前提がある」という意味に限る。checkoutのclean状態、実際のGitHubへの接続・認証、live remote main、全Goテストを確認した意味ではない。JSONのliveRemoteMainとcredentialsは常にnot-checked、repositoryHealthはnot-runである。Remote URLやCredentialは出力しない。引数から任意のShellコマンドを実行しない。

## 次段

Hostが信頼できるGitHub経路からfresh checkoutと最新origin/mainを確保し、別の明示権限を持つときだけ、既存 tools/reference-tool-neutral-development/qualify.sh を用いる。これは完全なRepository HealthとRQ-034を実行する正式な資格Referenceであり、本事前診断が置き換えない。資格receiptは候補HEADとmain SHAに固定し、最新mainが動けば再取得する。

テスト: go test ./tools/reference-chat-cli-host。合成した隔離Gitリポジトリに対して成功、HEAD不一致、origin/main欠落・不一致、祖先関係不一致、サブディレクトリ誤指定、未知引数、Remote URL中のsecret非出力を検査する。実GitHub通信・実TEIフルcheckoutの検証ではない。
