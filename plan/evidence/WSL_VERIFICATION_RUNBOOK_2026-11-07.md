# pwsh が不安定なときの WSL 検証手順

DSH の `pwsh` ツールが不安定な間、このリポジトリの検証は WSL2 上のネイティブ Go で実行します。この文書は実行手順だけを定めます。製品の仕様・計画の正本ではありません。

## 前提（2026-11-07 時点の実測）

- WSL2 ディストリビューション: `Ubuntu-22.04`
- ネイティブ Go: `/usr/local/go/bin/go`、`go1.27.3 linux/amd64`
- Windows 側 Go: `1.27.0 windows/amd64`（pwsh が動く場合はこちらでも同じ試験を回す）
- 作業ツリーは `/mnt/d/GIT/hat47x/...` として見える

## 落とし穴1: worktree の `.git` が Windows パスを書いている

`git worktree add` が書く `.git` は `gitdir: D:/GIT/hat47x/tei-platform/.git/worktrees/<name>` の形です。WSL の git はこの絶対パスを解決できないため、`fatal: not a git repository: .../D:/GIT/...` で落ちます。

対処（その worktree の `.git` を相対パスへ書き換える）:

```powershell
$p = "D:\GIT\hat47x\<worktree>\.git"
[System.IO.File]::WriteAllText($p, "gitdir: ../tei-platform/.git/worktrees/<worktree>`n", (New-Object System.Text.UTF8Encoding($false)))
```

Windows 側の Explorer が `.git` を隠し属性にしている場合があるので、必要なら先に属性を落とします。

```powershell
attrib -h -s D:\GIT\hat47x\<worktree>\.git
```

書き換えたあとも Windows の git はそのまま動きます（相対パスは両方で解決できる）。

**この破損が試験の失敗として現れる形**：`gobuilder.ResolveSourceClosure`は`git ls-files`で追跡ファイルを確認するため、git が使えないと次のように報告されます。

```text
self-contained target was not closure-qualified: rejected (read Git-tracked source paths: exit status 128)
```

一見「閉包の資格がない」という**内容の失敗**に見えますが、実際は環境の破損です。閉包の`rejected`を読むときは、まず git 自体が動くかを確かめてください。実際にこの誤読を起こしました。

## 落とし穴2: ログインシェルの PATH が壊れている

`bash -lc` は WSL の `/etc/profile` などを読み、Windows の PATH を取り込もうとして `export: ...: not a valid identifier` を大量に出します。検証では **ログインシェルを使わない** こと。

```bash
env -i PATH=/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin HOME=/home/hat47x TERM=dumb \
  bash --noprofile --norc -c 'cd /mnt/d/GIT/hat47x/<worktree> && go test ./tools/<tool> -count=1'
```

`env -i` で環境を捨てると、Host の Go 関連設定（`GOFLAGS` など）も混入しません。

## 落とし穴3: 引用符の入れ子で `pwsh` 側の解析が壊れる

`wsl ... bash -c "..."` の中に `"` や `$` を書くと、PowerShell が先に解釈して壊れます。**外側は単一引用符**（`'...'`）で囲み、内側の bash へ渡す文字列に `$` を使わないでください。複数行の処理は一時スクリプトへ書き出し、それを `bash /mnt/d/.../script.sh` で実行するのが確実です。

## 落とし穴4: Go module cache の読み取り専用ファイル

`go list`／`go build` は抽出済み Module cache のファイルを `0444` にします。`t.TempDir()` の自動削除はこのままだと権限エラーで失敗し、**試験本体が PASS していても FAIL と報告されます**。

回避策はコード側に持たせています（`tools/reference-go-module-supply-verify/temp_test.go`、`tools/reference-go-build-with-supply/main_test.go` の `stateDir(t)`）。権限を戻してから削除する cleanup を使い、試験の成否と cleanup の成否を混同しません。

## 標準手順

```bash
# 1) 変更したパッケージの整形・静的解析
env -i PATH=/usr/local/go/bin:/usr/bin:/bin HOME=/home/hat47x bash --noprofile --norc -c \
  'cd /mnt/d/GIT/hat47x/<worktree> && gofmt -l tools/<tool> && go vet ./tools/<tool>'

# 2) 単体試験
env -i PATH=/usr/local/go/bin:/usr/bin:/bin HOME=/home/hat47x bash --noprofile --norc -c \
  'cd /mnt/d/GIT/hat47x/<worktree> && go test ./tools/<tool> -count=1'

# 3) module 全体のビルド（新規 tools/ ディレクトリを足したとき）
env -i PATH=/usr/local/go/bin:/usr/bin:/bin HOME=/home/hat47x bash --noprofile --norc -c \
  'cd /mnt/d/GIT/hat47x/<worktree> && go build ./...'

# 4) 文書の相対リンクと索引
env -i PATH=/usr/local/go/bin:/usr/bin:/bin HOME=/home/hat47x bash --noprofile --norc -c \
  'cd /mnt/d/GIT/hat47x/<worktree> && go test ./tools/repository-plan-check -count=1'

# 5) 新規 tools/ ディレクトリの配置検査
env -i PATH=/usr/local/go/bin:/usr/bin:/bin HOME=/home/hat47x bash --noprofile --norc -c \
  'cd /mnt/d/GIT/hat47x/<worktree> && go test ./tools/repository-architecture-check -count=1'
```

長い試験（`repository-architecture-check` は 5 分前後、`gobuilder` は 2〜4 分）は、`pwsh` の既定タイムアウトを超えることがあります。バックグラウンドジョブで回し、`job_output` で受け取ってください。

## Windows と WSL の結果をどう扱うか

- 両方で PASS したことを「両OSで通った」と書けます。片方だけなら、通った側だけを主張し、他方は NOT RUN と書きます。
- **artifact digest の OS 間一致は主張しません。** Go のビルドは Target ごとに別物です。
- 経路（file:// の綴り、`GOMODCACHE` の既定値、モジュール cache の配置）が OS で違うため、URL や物理パスの扱いは両方で確かめます。

## いつ Windows へ戻すか

`pwsh` が再び安定したら、同じ手順を Windows 側でも 1 回回して差分がないことを確認します。WSL 手順は「pwsh が使えない間の代替」であり、常用の置き換えではありません。
