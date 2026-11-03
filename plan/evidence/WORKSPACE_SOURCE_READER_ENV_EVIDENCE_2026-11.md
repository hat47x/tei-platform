# Workspace Source Reader の環境別Evidence（#2571）

- 対象: [#2571](https://github.com/hat47x/tei-platform/issues/2571)（#1017から移設した未実証範囲）
- 状態: 実行Evidence。実装は変更せず、環境ごとの許容・拒否・未保証を分類する。
- 固定版: `origin/main` `00830da90`（`Merge pull request #2645`）+ 本Evidenceのテスト追加
- 非目標: 原子的な多file snapshotの実装、第二のSource Index / Source Reader / Authority、bind mount等を無条件に安全と扱うこと

## 1. 環境

| 環境 | OS | filesystem | Go | 備考 |
| --- | --- | --- | --- | --- |
| Windows | Windows 11 Pro 10.0.26100 | NTFS（C:） | go1.27.0 windows/amd64 | 管理者権限なし。Developer Mode 無効 |
| Linux | WSL2 Ubuntu 22.04.5 LTS | ext4 | go1.27.3 linux/amd64 | 同一ホスト（i7-8750H、12 logical CPU） |

Windowsの通常ユーザーは `SeCreateSymbolicLinkPrivilege` を持たないため、`os.Symlink` は `A required privilege is not held by the client` で失敗する。symlinkを使う拒否試験はWindowsでは実行できない。代わりに、特権を要しない**directory junction**（`mklink /J`）と**UNC path**でWindows固有の境界を実行確認した。

## 2. 再現

```powershell
# Windows
go test ./internal/workspace/ -count=1 -v
go test ./internal/workspace/ -run '^TestWindows' -count=1 -v
go test ./internal/workspace/ -run '^$' -bench 'BenchmarkLoadManagedWorkspaceFiles' -benchtime=3x -count=1
```

```bash
# WSL2 Ubuntu
go test ./internal/workspace/ -count=1 -v
go test ./internal/workspace/ -run '^$' -bench 'BenchmarkLoadManagedWorkspaceFiles' -benchtime=3x -count=1
```

## 3. 分類（許容 / 拒否 / 未保証）

| 事象 | Windows | Linux | 分類 |
| --- | --- | --- | --- |
| 通常のWorkspace Load | PASS | PASS | 許容 |
| 外部を指すdirectory junction | `path escapes from parent` で拒否 | — | 拒否（Windows固有の実行Evidence） |
| Workspace内を指すdirectory junction | 拒否（保守的） | — | 拒否。in-root相対symlinkの許容とは扱いが異なる |
| ドライブを跨ぐjunction | 拒否 | — | 拒否 |
| 管理fileへのhardlinkによる重複 | `same physical file` で拒否 | PASS | 拒否 |
| 大文字小文字だけ異なるsource root（`Semantic`） | 許容（同一file） | 存在しないrootとして扱う | 環境差あり（下記4章） |
| 大文字小文字だけ異なる2つのsource root | `same physical file` で拒否 | 片方は存在しないroot | 拒否 |
| UNC path（`\\localhost\C$\...`）のWorkspace | 許容（管理共有に到達できる場合） | — | 許容。保護はshare ACLでありTEIの境界ではない |
| 外部を指すsymlink | **NOT RUN**（特権なし） | 拒否（PASS） | Windowsは未実証 |
| Workspace内を指すsymlink | **NOT RUN**（特権なし） | 許容（PASS） | Windowsは未実証 |
| 読取り中のin-place変更・file追加削除 | 未保証 | 未保証 | 原子的snapshotではない |
| bind mount / overlayfs / 特権filesystem | **NOT RUN** | **NOT RUN** | 未保証（os.Rootの保護外） |
| ネットワークドライブ・コンテナ | **NOT RUN** | **NOT RUN** | 未保証 |
| Web/WASM | fail-closedのみ | fail-closedのみ | 安全なReaderなし（未実装） |

### 実行結果

Windows:

```text
go test ./internal/workspace/ -count=1 -v
PASS=135  FAIL=0  SKIP=35  ok tei/internal/workspace 6.547s
```

35件のSKIPはすべて `A required privilege is not held by the client`（symlink作成不可）または `native_sql_test.go` の明示的なWindows skipである。**symlink越境拒否のWindows実行Evidenceは無い。**

Windows固有の追加Evidence（すべてPASS）:

```text
--- PASS: TestWindowsJunctionEscapeIsRefused
--- PASS: TestWindowsInternalJunctionIsRefusedConservatively
--- PASS: TestWindowsCaseVariantSourceRootResolves
--- PASS: TestWindowsCaseVariantRootsAreRejectedAsSamePhysicalFile
--- PASS: TestWindowsUNCPathWorkspaceLoads
--- PASS: TestWindowsCrossDriveJunctionIsRefused
```

Linux:

```text
go test ./internal/workspace/ -count=1 -v
PASS=165  FAIL=0  SKIP=0  ok tei/internal/workspace 0.792s
```

symlink越境拒否・in-root symlink許容・broken symlink拒否を含む165件がLinuxでPASSした。

## 4. 環境差として記録する挙動

1. **Windowsのpinned rootはdirectory rename自体を拒否する。** `TestDomainSourceListingUsesPinnedRootAfterWorkspaceAliasReplacement` は、UnixではWorkspace directoryをrenameしてからsymlinkで差し替える。Windowsでは開いている `os.Root` handle がdirectory renameを共有違反で拒否するため、差し替えは成立しない。どちらも「pinned rootの外へは逃げない」という結論は同じで、試験は両方を許容してpinned読取りを確認する形にした。
2. **source rootの大文字小文字。** Windowsでは `Semantic` が `semantic/` に解決される。Linuxでは存在しない任意rootとして扱われる。manifestのsource rootは移植時に環境差が出るため、project側は実在のcaseで書く。
3. **native SQL file欠落のエラー文言。** Windowsは `The system cannot find the file specified` を返す。拒否契約は同じなので、試験は文言ではなく `fs.ErrNotExist` の分類で検査するようにした。

## 5. Loadコスト実測

`BenchmarkLoadManagedWorkspaceFiles`（任意実行、`benchtime=3x`）。合成Workspaceは `semantic/` 配下に `namespace benchN` だけのfileを置く。

| 管理file数 | Windows 時間 | Windows メモリ | Linux 時間 | Linux メモリ |
| --- | --- | --- | --- | --- |
| 50 | 37.7 ms | 323 KB | 2.49 ms | 240 KB |
| 200 | 123.3 ms | 1.29 MB | 9.24 ms | 952 KB |
| 800 | 841.0 ms | 5.11 MB | 38.1 ms | 3.76 MB |
| 3200 | 5.43 s | 20.7 MB | 171.5 ms | 15.6 MB |

- メモリはfile数にほぼ比例する（3200 fileで約6.5 KB/file）。全管理Sourceを保持する設計どおりで、追加の増幅は観測していない。
- Linuxの時間はほぼ線形（4倍で4.1〜4.5倍）。Windowsは4倍で6.5倍前後で、file I/OとDefenderの走査が支配的とみられる。初回の3200 file実行は16.3 s、再実行は5.43 sだったため、**Windowsの絶対値は走査cacheの影響を受ける**。
- 現行の最大Workspaceは `reference/workspace` の60 fileで、Windowsでも40 ms程度。

### 判断

- **上限は設けない。** 現行のWorkspaceは数十fileで、上限を追加すると既存projectを不必要に拒否する。file数に比例するメモリは許容範囲である。
- 目安として、**管理fileが3,000を超えるとWindowsのLoadが秒単位**になる。その規模のprojectが出た時点で、選択的captureまたは上限の導入を別Issueで判断する。
- `scanManagedFiles` は物理同一性を総当たりで比較するため、理論上はfile数の二乗で増える。3200 fileではまだI/Oが支配的で、実測に対して最適化を正当化できない。10,000 file規模のprojectが出た場合の課題として記録する。

## 6. 主張しないこと

- 原子的な多file snapshotではない。読取り中のin-place変更・file追加削除・directory renameの窓は残る。必要なら別の契約として設計する。
- bind mount・overlayfs・特権filesystem constructsを安全とは扱わない。これらは `os.Root` の保護外である。
- UNC pathのWorkspaceは、share ACLとネットワークの信頼に依存する。TEIがshareの権限を検証するわけではない。
- Windowsのsymlink越境拒否は、このホストでは実行できていない（NOT RUN）。Developer Modeまたは特権を付与した環境での追試が必要である。
- Web/WASM向けの安全なSource Readerは無い。現状はfail-closedのみである。

## 7. 後退させていないこと

- symlink越境拒否（Linuxで165件PASS）
- Load時byteとSource Indexの束縛
- 部分snapshotのfail-closed（`IndexedLoadedSnapshot`）
- 第二のSource Index / Source Reader / Authorityを追加していない
