# 第一Web TargetのGoパッケージ閉包を観測する

このツールは、既存の[配備単位の設計](../../plan/design/RUNTIME_TARGET_BINARY_COMPOSITION_DESIGN.md)における **E0（実際のGoソース依存の観測）** を補助する読み取り専用Referenceです。[Issue #259](https://github.com/hat47x/tei-platform/issues/259)のTarget／実行境界検証に属します。

Plugin Lock、Build Plan、Provider、Capabilityの選択は**行いません**。`go list -deps -json`の結果から、現在のGoメインモジュールに含まれるコンパイル対象パッケージのImportPath、Go/Cgoファイル、直接import先を整列したJSONへ変換するだけです。標準ライブラリと外部モジュールのpackage情報は出力から省きますが、import辺には保持します。Cgoの`import "C"`は`go list`が独立packageとして出力しない疑似importなので、Cgo対象ファイルがある場合に限り例外として許容し、その辺を出力に残します。

## 実行手順

依存関係を取得できるGo環境の、**変更のないTEIリポジトリのルート**で実行します。結果はチェックアウト外に保存してください。

```sh
go test -count=1 ./tools/reference-target-package-evidence

go run ./tools/reference-target-package-evidence \
  -entry ./tools/reference-first-web-target \
  > /tmp/tei-first-web-go-packages.json
```

- 開始・終了時に`git status --porcelain --untracked-files=all`とHEADを確認し、変更があれば失敗します。さらに、`go list`がコンパイル対象として挙げたメインモジュールのGo/Cgoソースに加えて、`go list`が列挙する埋込ファイル（`//go:embed`）とC/C++/ヘッダー/Assembly/SWIG等の補助入力がGitの追跡対象であることを検査します。補助入力は`auxiliaryInputs`に`kind`と相対パスを出力します。Goのメインモジュールもリポジトリ直下に固定し、Git管理対象の通常ファイル`go.mod`と照合します。通常の`git status`では見えない**ignore対象の未追跡Goファイル**も、コンパイルに使われるなら失敗させます。またGitで追跡されていても、シンボリックリンク経由で別のソースを読み込む場合は拒否します。ただしこの検査は全入力の改ざん防止や`tools/repository-health-check.sh`のexact-head full PASS／統合資格を保証しません。
- Git indexの`assume-unchanged` / `skip-worktree`が設定されたファイルは、内容が変わっても`git status`で検出できない。`git ls-files -v -z`を使い、通常の追跡状態以外を**作業ツリーが空でも拒否**する。疎なチェックアウトなど意図的なGit index設定がある環境も、このE0証拠では例外扱いしない。
- `GOENV=off`、`GOWORK=off`、`GOFLAGS=`を`go list`に適用し、隠れたユーザー設定・Workspace・build flagによる入力変化を避けます。必要なbuild tagsは`-tags`で明示し、GOOS／GOARCH／CGO_ENABLED／Go版を結果に記録します。さらに、`GOAMD64`・`GOARM`・`GOARM64`・`GO386`・`GOMIPS`・`GOPPC64`・`GOWASM`・`GOEXPERIMENT`等の有効なアーキテクチャ特性を`targetFeatures`に保持します。たとえば同じ`linux/amd64`でも`GOAMD64=v1`と`v3`では`//go:build amd64.v3`のGoソース選択が異なるため、GOOS/GOARCHだけを比較キーにしません。
- Goのツールチェーンが依存を解決できない、JSONが不正、参照先パッケージが欠落している等の場合には不完全な結果を成功として出力しません。

## 二つの起動点の閉包比較（E1の前処理）

**異なるGoビルド入口から生成した二つのE0 JSON**がある場合、次のように読み取り専用で比較できます。現状の第一Web Targetは一つの実行時選択入口だけであり、このコマンドを加えただけでApplication別のビルド入口が成立するわけではありません。

```sh
go run ./tools/reference-target-package-evidence \
  -compare-before /tmp/tei-unified-e0.json \
  -compare-after /tmp/tei-application-e0.json \
  > /tmp/tei-package-closure-delta.json
```

比較は`SourceRevision`、メインGoモジュール、Go版、GOOS、GOARCH、CGO、`GOFLAGS`、`targetFeatures`が一致しない場合に拒否します。`-tags`による意図的なソースの差は許容しますが、指定されたbuild tagsは結果へ明示し、同一ビルド条件の成果物であるとは主張しません。

結果は、片方にしかないパッケージ（`beforeOnly`・`afterOnly`）と、共通パッケージでコンパイル対象Go/Cgoファイル、埋込・補助入力、import辺が変わったもの（`changed`）を区別して返します。パッケージやファイルの列挙順序の差だけでは変更とは判定しません。取り込むJSONは厳格にデコードしたうえで、**メインモジュール内の全パッケージが指定entryから静的import経由で到達可能であり、参照されたメインモジュール内パッケージが全て含まれること**を構造検査します。これにより、閉包に存在し得ない幽霊パッケージを`beforeOnly`に入れて「除去成功」と誤判定する事態を拒否します。ただし、構造的な整合性と来歴の正しさは別です。**入力JSONの真正性を検証するAuthorityではありません**。作成元のclean checkout／exact HEADと、実際に使ったビルド条件を独立して確認する必要があります。

一時Gitリポジトリを利用した結合テストには、同一HEAD上の二つの**異なるコンパイル時Go入口**（機能を参照する`cmd/app`と、参照しない`cmd/minimal`）を作り、実際の`go list`から二件のE0を得て`beforeOnly`・`afterOnly`を比較するケースを追加しました。これも現HEADのテストPASSをまだ取得していません。

E1の最終判定には、さらに`go build`後のbytes、digest、link済みsymbol、機能・認可の回帰試験が必要です。共通パッケージに未使用コードが残っているか、リンク時に削除されたかは、この比較だけでは分かりません。

## この証拠で言えること／言えないこと

**観測対象**は一つの静的なGoビルド入口です。第一Web Targetの`-application`はサーバーの**実行時引数**なので、それを二通りにしても別のGo package closureにはなりません。アプリごとに異なる実装を物理的に除外したければ、異なるcompile-time入口を作り、同じ条件でそれぞれを測る必要があります。

このJSONだけでは、Go linkerの不要コード除去、最終成果物のサイズ・digest・symbol、実動作、Secret／認可、独立したWeb UIのJSやHTMLの配布物は証明できません。`go list`の補助ファイル一覧も、Cの推移的な`#include`、`pkg-config`、システムSDK、外部リンカ入力、ツールチェーンが生成するファイルやGo以外のアセットビルド全体を網羅するものではありません。また`go list`が収集する静的import先は、GoのReflection・Pluginの動的参照や外部プロセスを完全には表しません。

次段階のE1では、同じ受理済みApplication meaning（Handover B/R）に対する**別の配備入口**で同じ観測を行い、さらに`go build`成果物のbytes・digestと必要に応じsymbol／実行結果を照合します。既存のGate C第一Web Targetを置き換えず、Build Artifactの正本は[#943](https://github.com/hat47x/tei-platform/issues/943)に委ねます。


## 二つのGo起動点に対する局所実証（TEI本体ではない）

`compare_integration_test.go`は、Gitで固定した一時Goモジュールに`cmd/unified`と`cmd/order`の**二つの異なるcompile-time入口**を設け、実`produceEvidence`と`compareReports`が返すソース閉包を検査し、続けて`go build`と`go tool nm`でリンク結果に差があることを確認するテストです。実TEIの`order_entry`・`repair_intake`、実Plugin実装、Gate Cの成否を表すテストではありません。

同じ構成を別の**隔離したGo 1.23.2/linux-amd64環境**で手動再現し、`CGO_ENABLED=0`、`GOENV=off`、`GOWORK=off`、`GOPROXY=off`、`go build -trimpath -ldflags=-buildid=`の下で、次の局所結果を取得しました（TEIのactual binaryではありません）。

| compile-time入口 | メインモジュール内パッケージ | 生成Goバイナリ | `internal/repair.Run` |
| --- | --- | ---: | --- |
| `cmd/unified` | `cmd/unified`, `internal/order`, `internal/repair` | 2,124,733 bytes | シンボルあり |
| `cmd/order` | `cmd/order`, `internal/order` | 2,124,232 bytes | シンボルなし |

この局所の差は**501 bytes**であり、Goリンク時のサイズ改善が常にこの値・比率で生じるという根拠にはならず、アプリの正しさや権限境界の証明でもありません。局所のバイナリdigestは各build結果を識別するために取得しており、TEI Build Artifact Lockや配布物の正本ではありません。**PR現headの`go test`を成功させた記録ではない**点も区別します。実TEIのE1受入には、同じ意味・契約を保つ実アプリ別入口、full health、実成果物の実測と認可の回帰確認が必要です。

## 検証段階

**初期版のGoソース3件のみ**、隔離したGo 1.23.2モジュールで当時のGit blob SHA一致を照合し、局所`go test`・`go vet`・`gofmt`および小さなGo entryに対する`go list`を確認しました。その後、実GoによるCgo疑似importとGit ignore対象のGoファイル混入を再現し、パーサー／ソース確認／負例テストを追加しています。**追加修正後の現headではGoテスト・Go vet・gofmt・実第一Web Targetの実行結果は未取得**です。Go 1.23.2/Linuxの隔離モジュールにおいて`GOAMD64=v1`は`main.go`のみ、`GOAMD64=v3`は`amd64v3.go`も選ぶことを実測し、この差と`targetFeatures`の一致性を確認する実Git/Go結合テストを追加しています（追加テストの結果は未取得）。さらに、Git indexの状態フラグが`git status`の結果を偽装し得ることを独立の一時Gitリポジトリで再現し、拒否する検査と回帰テストを追加した。**新しいGit index対策の最小ヘルパーは隔離したGo 1.23.2でテスト・vet・gofmtにPASSしたが、TEIリポジトリ内の追加コード全体のPASSではない**。初期版のPASSを現headへ流用せず、TEI trusted clean full checkoutでのexact-head `tools/repository-health-check.sh`と前後clean確認を統合条件とします。Draft PRのまま扱います。
