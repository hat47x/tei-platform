# Go Module供給の実観測（SB-0b 前提）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) の SB-0b（`cmd/tei` を実Buildへ進めるための外部Module供給）に先立ち、**供給の有無・破損・改変が、分類・照合・成果物へどう現れるか**を実際に観測した記録です。

**これはBuildの実行を伴う観測ですが、TEIのBuild経路（`BuildPackage` / `ResolveSourceClosure` / SB-3実行器）を通したBuildではありません。** ここで測ったのは`go build`と`go mod verify`という既存Go toolchainの振る舞いであり、TEI artifact・Lock・受入・権限・G4の判定を一切動かしません。G4は`UNQUALIFIED`のままです。

- 設計側の契約案：[セルフビルドのGo Module供給境界](../design/GO_MODULE_SUPPLY_FOR_SELF_BUILD_SB0B_2026-11.md)
- SB-0a（分類の事前観測）：[Draft PR #3091](https://github.com/hat47x/tei-platform/pull/3091)（branch `fix/gobuilder-module-closure-preflight-20261107`、head `7eaeaa09135fbecbd6e26aef2ccc1207e374be01`）。追加した観測入口と`gobuilder`の分類コードは、このPRのbranch上にあり、このcheckout（main系）には未統合

## 観測条件

| 項目 | 値 |
| --- | --- |
| 対象checkout | `D:\GIT\hat47x\tei-wt-module-closure`。PR #3091のbranch head `7eaeaa09135fbecbd6e26aef2ccc1207e374be01`（親 `4fdad69d9f360ef605930cc298f2a72efdfdee8e`） |
| Go | `go1.27.0`、windows/amd64。実行ファイルdigest `sha256:7d828191ba32519a9c9361789ab647486236ed45c660889196c7770a8ff1985c` |
| `go.mod` / `go.sum` | `sha256:ad0264180541bc910132f3f9183deb81539c89b3f4d88ba68fd2512d0382f19c` / `sha256:d5967ad9f867efa398e7fa9afaccb61819a3e8f520aaee7bbb4a5069402c55d3` |
| 固定設定 | `GOPROXY=off`、`GOSUMDB=off`、`GOWORK=off`、`GOTOOLCHAIN=local`、`CGO_ENABLED=0`、`GOENV=off`、`GOFLAGS=`、`GOOS=windows`、`GOARCH=amd64`、`-trimpath` |
| 隔離状態 | 観測ごとに独立した`GOMODCACHE`・`GOCACHE`・`GOPATH`・`HOME`・`TMP` |
| 供給元 | `C:\Users\yhata\go\pkg\mod\cache\download`（135.4 MB、298ファイル）を丸ごと複製。**Host cacheを直接参照せず、複製先のパスだけを使った** |

供給元の物理パスはマシン固有であり、内容同一性として引用しない。

## 観測1：供給の有無でSB-0aの分類が変わる

`go run ./tools/reference-go-module-closure-preflight -repo . -package ./cmd/tei -state <状態dir>` を、二つの状態dirで1回ずつ実行した。

| 供給 | 分類 | 解決package | 未解決 |
| --- | --- | --- | --- |
| 空（状態dirを新規作成しただけ） | `missing-module` | 65 | `github.com/jackc/pgx/v5`、`golang.org/x/text/unicode/norm`、`modernc.org/sqlite`（3件、いずれも`module lookup disabled by GOPROXY=off`） |
| Host cacheの複製を`cache/download`へ置いた | `external-unresolved` | 109 | 0 |

外部Module根は12件（`github.com/dustin/go-humanize`、`github.com/jackc/pgpassfile`、`github.com/jackc/pgservicefile`、`github.com/jackc/pgx/v5`、`github.com/mattn/go-isatty`、`github.com/ncruces/go-strftime`、`github.com/remyoudompheng/bigfft`、`golang.org/x/text`、`modernc.org/libc`、`modernc.org/mathutil`、`modernc.org/memory`、`modernc.org/sqlite`）。対象Module `tei` は65package。

**読み取れること**：SB-0aの`missing-module`は、Sourceや構文の誤りではなく**供給の不在だけ**を表していた。供給を置けば同じSourceが`external-unresolved`まで進む。
**読み取れないこと**：この12件のbyteが正しいこと。`external-unresolved`は「外部入力がある」という観測であり、照合済みを意味しない。

## 観測2：壊れた供給は`unknown`へ落ちる

供給のうち`golang.org/x/text@v0.41.0`のarchive（`cache/download/golang.org/x/text/@v/v0.41.0.zip`、元のsha256 `E63F35DAAAE749D0FFFF97A295AD8F4837A662938A46B7A87F18A88E85A5CBF9`）の1byteだけを変えた（変更後 `51E0501B5E9F169410D43E39B3AE9FC571E4137862B16886258F61BF476F6D29`）。同じpreflightを実行した。

| 分類 | 解決package | 未解決 |
| --- | --- | --- |
| `unknown` | 96 | `golang.org/x/text/secure/precis`、`golang.org/x/text/unicode/norm`。いずれも`unzip ...: flate: corrupt input before offset 44` |

`x/text`のModule根は一覧に残るが、`unresolvedPackages`に載るので分類は`unknown`になる（`classifyModuleRoots`が未解決を最優先で見る）。

**読み取れること**：破損はfail-closedで止まる。
**読み取れないこと**：それが「供給の破損」なのか「Sourceの誤り」なのか。`unknown`は両方を同じ箱に入れる。これが設計側で`corrupt-supply`を足す根拠である。

## 観測3：展開済みModuleの改変は、成果物digestでは検出できない

隔離`GOMODCACHE`へ供給したあと、**展開済みディレクトリ**内の`golang.org/x/text@v0.41.0/unicode/norm/normalize.go`を書き換え、`GOCACHE`を消した状態で`go build -trimpath -o <out> ./cmd/tei`を実行した。

| 供給 | 改変内容 | 改変後file sha256 | 成果物 sha256 |
| --- | --- | --- | --- |
| 未改変（試行A） | なし | — | `D60C5FCC63FFDBDA9EBB2E02EB5083AC9C6E91FC54AD05746FE9BC79D03A7B58` |
| 未改変（試行B・別状態dir） | なし | — | `D60C5FCC63FFDBDA9EBB2E02EB5083AC9C6E91FC54AD05746FE9BC79D03A7B58` |
| 試行C | **コメント行を1行追加** | `FDE42F2B0AAE1E71E45FFA878376649CC7F689A55DF5223285392651A2A638FF` | `D60C5FCC63FFDBDA9EBB2E02EB5083AC9C6E91FC54AD05746FE9BC79D03A7B58`（**未改変と同一**） |
| 試行D | 定数`NFC Form = iota`を`NFC Form = iota + 1`へ | `C6690F7FCF5108C51D4F6723536AC5C750D0E65A9CEA65DD833BE33A4EA2650D` | `504DF5E4195BF5CD8CFD4AA0CD8C98CEC91CFDACCCFC089B41E773DE976F9066`（**未改変と異なる**） |

試行Cの成果物が同一であることは、改変後に同じfileを再度digestして確認した（`FDE42F2B…`のまま）。つまり**ビルドは改変されたfileを読んだうえで、同じbyte列を出力した**。`-trimpath`により物理パスは成果物へ入らないため、pathの差も現れない。

**読み取れること**：成果物byteの一致は、供給byteの同一性を証明しない。逆に、供給の改変が成果物を変え得ることも同じ実験で示せる（試行D）。
**読み取れないこと**：どの改変が「危険」かを成果物から判別すること。コメントの追加が無害だったのは偶然であり、同じfileの定数変更は成果物を変えた。

## 観測4：`go mod verify`は供給の改変を捕まえる

同じ三つの状態dirで`go mod verify`を実行した。

| 状態 | 終了コード | 出力 |
| --- | --- | --- |
| 未改変（試行A） | 0 | `all modules verified` |
| コメント改変（試行C） | 1 | `golang.org/x/text v0.41.0: dir has been modified (...)` |
| 定数改変（試行D） | 1 | `golang.org/x/text v0.41.0: dir has been modified (...)` |

`go mod verify`はネットワークを使わず、`go.sum`の`h1:`と展開済みの内容を突き合わせる。

**読み取れること**：観測3の二つの改変は、どちらも既存toolchainで検出できる。SB-0bは第二のハッシュ実装を必要としない。
**読み取れないこと**：`go.sum`の`h1:`が正しいこと（供給元のTrust）。照合は「期待値と実際のbyteが一致する」ことだけを言う。

## 観測5：`go mod download -json`が照合に必要な値を返す

試行Aの状態で`go mod download -json golang.org/x/text`を実行すると、次を返した。

| field | 値 |
| --- | --- |
| `Sum` | `h1:vz/seA0lnX87Othu2f/0L24RcgrXD9/YFTSuGjj3rH8=`（`go.sum`の`golang.org/x/text v0.41.0 h1:`行と一致） |
| `GoModSum` | `h1:jvf1O8ajNzZqhSrQBPbutR/EB83Cc0CFrezNQIwbb5M=`（同`/go.mod`行と一致） |
| `Zip` / `GoMod` / `Dir` | 供給した物理パス（マシン固有。内容同一性として引用しない） |

**読み取れること**：供給manifestの`moduleH1` / `goModH1` は、自前計算ではなくtoolchainの申告から取れる。
**読み取れないこと**：`Zip`のbyteが`Sum`に対応すること。そこは`go mod verify`と`h1:`照合の役割である。

## 観測6：供給を用意する経路は`go mod download`では開かない

新しい状態dirで`cache/download`を空のまま`go mod download`と`go build`を試すと、`go: github.com/dustin/go-humanize@v1.0.1: module lookup disabled by GOPROXY=off`で停止した。`GOPROXY=off`ではdownloadも供給を要求する。

**読み取れること**：供給はBuild試行の**外側**で用意する必要がある。Build試行の中で取得を試みる設計は、`GOPROXY=off`と両立しない。

## 主張しないこと

- TEIのBuild経路（`BuildPackage`・`ResolveSourceClosure`・`reference-plugin-self-build`）を通したBuildは**未実行**。本観測の成果物は`go build`の直接実行であり、TEIのartifact・Lock・Evidenceではない。
- 供給byteのTrust、供給元の真正性、署名、`go.sum`の内容の正しさは**未検証**。
- 観測3〜5の`x/text`は一例であり、他の11 Moduleでも同じ振る舞いになることは確かめていない。
- 別Go版・別OS・別供給元への一般化はできない。`-trimpath`を外した場合、成果物に物理パスが入るため観測3の同一性は成立しない（試行Dの非trim版は`1A0D3FDD…`）。
- 本観測の成功はG4・SB-A・SB-B・SB-Cのいずれの合格でもない。

## 実行した検証

- `go test ./tools/internal/gobuilder ./tools/reference-go-module-closure-preflight -count=1`：24件PASS（2パッケージ）
- `go vet ./tools/internal/gobuilder ./tools/reference-go-module-closure-preflight`：指摘なし
- 観測1〜6：上記のコマンドを1回ずつ実行。再実行による値の揺れは、試行A/Bの成果物digest一致で確認した範囲のみ

**NOT RUN**：TEI Build経路での外部Module入力、SB-0bの供給manifest実装、`corrupt-supply`分類の実装、SBP-24〜30の実試験、CI、trusted full checkout。

## 参照

- SB-0aの記録：Draft PR #3091 が同じ`plan/evidence/`配下へ追加する`TEI_GO_MODULE_CLOSURE_PREFLIGHT_2026-11-07.md`（このcheckoutには未統合のため、相対リンクでは辿れない）
- 設計案：[`GO_MODULE_SUPPLY_FOR_SELF_BUILD_SB0B_2026-11.md`](../design/GO_MODULE_SUPPLY_FOR_SELF_BUILD_SB0B_2026-11.md)
- 既存の判定用ノート：[`PLUGIN_SELF_BUILD_PRODUCTION_CONTRACT_2026-10.md`](../design/PLUGIN_SELF_BUILD_PRODUCTION_CONTRACT_2026-10.md)
- 判定の正本：[`docs/status.md`](../../docs/status.md)
