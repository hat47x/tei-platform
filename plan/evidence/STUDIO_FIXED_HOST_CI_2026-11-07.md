# Studio固定Host負例のCI実行記録（2026-11-07）

- 対象PR: [#3059](https://github.com/hat47x/tei-platform/pull/3059)
- 対象workflow: `.github/workflows/tei-studio-host-asset-negative-ci.yml`
- 固定Host: `4bc437c72e63a3b5d578b1efe2c60be510a9f8a9`
- 生成器: `fe8e37b1c735dc138c36cb2d37db94ee44424b9e`
- 実行環境: GitHub Actions Ubuntu 24.04、Go 1.26.10（`go-version-file: go.mod`）、Node 22

## 目的

#2948で見つかったGo版の直指定を`go-version-file: go.mod`へ揃えた後、workflowが**試験本体まで到達するか**を確認する。あわせて#2985の「固定Hostのbuildで`go.mod`更新要求」を、使い捨てスナップショットのmodule metadataだけを適合させて解消する。

## 実行結果

1回目（run `42264817250`）: `go: updates to go.mod needed, disabled by -mod=readonly`。Go版は解決したが、固定Hostの`go.mod`が現在のroot moduleの`golang.org/x/text`などを要求していなかった。

2回目（run `42265470943`）: module metadataを`GOFLAGS=-mod=mod`で適合させ、buildは成功。しかし`internal/studiohost`が新旧ファイル混在でcompile error（`h.CommandLifecycle undefined`）。

3回目（run `42265608217`、head `bd21fda8f`）: `cmd/tei-studio`と`internal/studiohost`を固定Hostのsnapshotで完全に置換。buildとProbeの前段が通り、**試験本体へ到達**した。

```text
=== RUN   TestStudioHostConfiguredAliasReplacementKeepsPinnedRealRoot
--- PASS: TestStudioHostConfiguredAliasReplacementKeepsPinnedRealRoot (0.01s)
=== RUN   TestStudioHostResolvedRootReplacementNeverReadsOutsideWorkspace
    studiohost_root_retarget_test.go:122: SECURITY: allowlisted Workspace ID followed the replaced physical root into the outside Workspace (foreign_catalog=true, outside_namespace_visible=true)
--- FAIL: TestStudioHostResolvedRootReplacementNeverReadsOutsideWorkspace (0.00s)
```

## 読み取れること

- #2948のGo版ドリフトは解消した。workflowはGoのsetupで止まらず、`go build`とProbeの前段まで進む。
- #2985のbuild失敗は、固定Hostの`go.mod`が現在のroot moduleの依存と一致しないことによる。固定Hostのsourceを変えず、**使い捨てsnapshotのmodule metadataだけ**を適合させればbuildは通る。
- 固定Host `4bc437c7` は、Workspace IDの許可リストが物理rootの置換に追随し、Workspace外のCatalogを読める。これは#3061として記録した。現在のmainの`internal/studiohost/readonly.go`は`rootInfos`と`os.SameFile`でroot identityを検査しており、同じ負例を防ぐ実装になっている（`TestStudioHostResolvedRootReplacementNeverReadsOutsideWorkspace`がmainで通ることは別途確認する）。

## この記録が主張しないこと

- 固定Hostの修正完了。
- 現在のmainがすべてのroot置換競合を防ぐこと。
- 製品の受入・本番適格性。
- OSのfd raceやin-flight操作の安全性。
