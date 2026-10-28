# reference-plugin-build-supply

技術検証用サンプル[`plugin-build-supply`](../../reference/plugin-build-supply/README.md)(サンプル7)の工程1〜4を、一つの通しの記録として実行する実装です。設計の正本は[技術検証用サンプルの統合設計 4.7節](../../plan/design/TECH_VALIDATION_SAMPLES_2026-10.md#47-プラグインの構築意図から自己構築成果物の固定までplugin-build-supply)です。

この実装はサンプル固有の処理で、`internal/`の基盤には入れていません。製品コードは、`internal/plugin`(`BuildPlanFingerprint`、`BuildDefinitionFingerprint`、`LockBuildOutput`、`RealizeBuiltDefinition`)、`internal/plugin/runtime`(`BuildPlan`、`BindRuntimeEngineLock`、`DeploymentExecutor`、`StdioProcessLauncher`、`ValidateDeploymentInvocation`)、`internal/syntax`、`internal/workspace`と、`tools/internal/gobuilder`をそのまま呼びます。工程3の自己構築の実行器(`tools/reference-plugin-self-build`)と工程1の構築意図の参照実装(`tools/reference-build-authoring-intent`)は、変更せずにCLIとして構築して呼びます。

構築するのは、自己構築の実行器と同じ`tei.process.reference`(`tools/reference-process-plugin`)です。

## 構成

| 役 | 実体 | 担当 |
| --- | --- | --- |
| 試験(編成) | `go test`のプロセス | 工程1〜4、失敗注入、通しの照合 |
| ホスト | 試験のプロセス内の`Host`(`stage2.go`) | 構築の試行を一回起動するごとに、起動の前に発生識別子を発行し、OSが返した終了状態(終了コードまたはシグナル)を型つきで記録する |
| 構築器 | 試験のバイナリを`TEI_PBS_ROLE=builder-attempt`で起動した子プロセス(`child.go`) | 既定では`gobuilder.BuildPackage`で実際に`go build`する(自己構築の実行器と同じ隔離した設定)。失敗注入のときだけ、部分出力を残して終わる模擬と、毎回違う出力を出す模擬になる。どの方式かは試行の記録に残る |
| 保管庫 | 一時ディレクトリの`<root>/sha256/<64桁の16進>`(`store.go`) | `tools/reference-build-artifact-store`と同じ配置と規則。照合値が合わない物は拒否し、修復しない |
| 自己構築の実行器 | `tools/reference-plugin-self-build`を構築したCLI | 工程3。二回構築、変換の構築器の段階の収束、配布一式とTEI-native実現、実行時の固定情報(ホスト登録)の検証根拠を出す |
| プロセスプラグイン | 保管庫から取り出したバイト列を`tei-process-reference`として置いた実行体 | 工程4。製品の`StdioProcessLauncher`から、プロセス通信の版1で一回呼ばれる |

## 既存資産を呼べるか

工程1・2の参照実装は、いずれも`package main`で、関数を外から呼べません。CLIが利用者の入力を受け取るかどうかで、呼べるかが決まります。

| 工程 | 既存資産 | 呼べるか | 理由と扱い |
| --- | --- | --- | --- |
| 1 | `tools/reference-build-authoring-intent` | CLIで呼べる | `-intent`、`-catalog`、`-source-root`で、このサンプルの宣言([`testdata/intent.json`](testdata/intent.json))と観測から作ったカタログを受け取る。ただしソースの照合値は資産内の固定入力用のディレクトリ照合(`DirSourceResolver`)で計算し、自己構築の実行器のGoのソース閉包の照合値とは別の値になる。そのため、確定計画はこの資産の出力ではなく、同じ宣言からサンプル側(`Derive`)で導く。資産の出力とは、ソース以外の項目(プラグインの識別、定義の照合値、構築器の識別、手順、出力の種類、対象、構成、パラメータ)が一致することと、提供元が二つのときに両方の順序で停止することを照合する |
| 1 | `tools/reference-builder-contract-resolution` | 固定入力でだけ動く | CLIに入力の引数がなく、資産内の固定の提供元で解決・固定・再生を示すだけ。利用者のカタログを渡せず、信頼の失効も扱わない。解決・固定・再生・失効はサンプル側(`ResolveBuilder`、`ResolutionLock`、`ReplayLock`、`TrustStore`)で作った。CLIは単独確認として実行し、終了状態だけを記録する |
| 2 | `tools/reference-build-attempt-evidence` | 固定入力でだけ動く | CLIは模擬の失敗一件を出して終了状態1で終わる(意図した動作)。試行の記録と発生識別子の型は非公開。試行の記録はサンプル側(`Host`、`History`、`DeriveReproducibility`)で作った |
| 2 | `tools/reference-build-artifact-store` | 固定入力でだけ動く | CLIは一時ディレクトリへ固定のバイト列を出し入れするだけ。`PutBytes`と`GetBytes`は非公開。公開の説明にある配置と規則を`Store`で再現した |
| 3 | `tools/reference-plugin-self-build` | CLIで呼べる | `-repo`、`-go`を受け取り、検証根拠のJSONを標準出力へ出す。計画は実行器が自分で組み立て、外から渡せない。そのため、実行器の計画が工程1の確定計画と同じ照合値であることを、検証根拠の側で照合する(`BindStage3`)。構築した成果物は実行器が一時ディレクトリごと削除し、バイト列は外へ出ない |
| 3 | `tools/reference-native-plugin-builder`、`reference-plugin-package-builder`、`reference-transform-runtime-engine`、`reference-transform-seed-builder` | 実行器の内部で構築・実行される | このサンプルからは直接呼ばない。実行器の検証根拠に残る二回構築、段階の収束、エンジンの束縛を照合する |
| 4 | `tools/reference-process-plugin` | 構築済みの実行体として呼べる | 製品の`StdioProcessLauncher`(プロセス通信の版1)で起動する |
| 共通 | `tools/internal/gobuilder` | ライブラリとして呼べる | Goのソース閉包、ツールチェーンの照合値、隔離した構築を、自己構築の実行器と同じ関数で計算する |

## 工程の流れ

1. 工程1(`Derive`、`stage1.go`): 作業ツリーが清浄であることを確かめ、宣言を読みます。プラグイン定義は実ファイルのSHA-256と構文解析器の結果から、Goのソースは`gobuilder.ResolveSourceClosure`(隔離した`go list`)から照合値を導きます。Goは実行体、`go version`、GOROOTの閉包から照合値を作り(実行器の`goToolchainDigest`と同じ式)、これを提供元`tei.builder.go.seed`の成果物の照合値としてカタログへ載せます。カタログは観測から作り、照合値を手で書きません。提供元を一つに解決して固定記録(`ResolutionLock`)を作り、確定計画とその照合値を得ます。同じ宣言とカタログを構築意図のCLIにも渡し、ソース以外の項目の一致を確かめます。
2. 工程2(`Host.Run`、`stage2.go`): 確定計画と固定記録で、同じ計画の試行AとBを同時に起動し、Bが起動を報告した時点でBのプロセスグループを`SIGKILL`で止めます。続けて試行Cを起動します。履歴(`History`)は、このホストが発行した発生識別子を一度だけ受け付けます。再現性(`DeriveReproducibility`)は、成功した別々の発生識別子の二件(AとC)から導き、一致したバイト列だけを保管庫へ入れます。
3. 工程3(`RunSelfBuild`、`BindStage3`、`stage3.go`): 既存の実行器を、ホストのGo設定(`GOFLAGS=-mod=mod`と、`GOCACHE`、`GOMODCACHE`、`GOPATH`を別々の空ディレクトリ)を与えた環境で実行します。検証根拠を、ツールチェーン、ソース閉包、版、実行器が組み立てた計画の照合値、二回構築の一致、工程2の保管済みのダイジェスト、成果物の固定(`LockBuildOutput`)、ホスト登録(実行時の固定情報の`artifactDigest`と`buildPlanFingerprint`)、変換の構築器の段階の収束、配布一式の計画へのエンジンと構築器の束縛の順に照合します。工程1の固定記録の構築器が失効していれば拒否します。
4. 工程4(`InvokeFromStore`、`stage4.go`): 保管庫からダイジェストでバイト列を取り出し、記述子が名指す`tei-process-reference`として置きます。自己構築用の作業領域と、工程3がホスト登録した実行時の固定情報から環境計画を組み(`pruntime.BuildPlan`)、TEI-native実現のエンジンの固定情報を束縛して、製品の`DeploymentExecutor`と`StdioProcessLauncher`で一回呼びます。検証器は置いた実行体のバイト列を直接ハッシュします。
5. 通しの照合(`VerifyThrough`、`chain.go`): 工程ごとの記録を前の記録の照合値でつなぎ、読むだけで照合します。

## 失敗注入と合否

観測値は、下の「安定性の記録」の実行(試験の版`3437771`)で得た値です。

| 要件 | 試験 | 注入 | 合否の観測 |
| --- | --- | --- | --- |
| P-GOV-01(工程1) | `TestPGOV01ResolutionStopsAndRevocationRefusesUse` | 観測したGoの提供元に、同じ契約・手順・対象・構成の模擬の提供元を加える(二つの順序)。固定記録の後に提供元を加える。固定記録の構築器の信頼を失効させる | 新規の解決はサンプル側も構築意図のCLIも`ambiguous`で停止し、二つの順序で同じ誤り文。固定記録の再生は提供元の追加後も同じバイト列で、計画の照合値も同じ。失効後も再生(再現)はでき、試行の起動(発行した発生識別子0件)と工程3の照合は`builder trust is revoked`で拒否 |
| P-JOB-01(工程2) | `TestPJOB01ConcurrentAttemptsKeepDistinctOccurrences` | 同じ確定計画で二つの試行を同時に起動し、一方を起動の報告の直後に`SIGKILL`で止める | 止めた試行は`failed`、終了状態は`SIGKILL`(OSの報告)、成果物なし。三つの試行は別々の発生識別子(発行3件)で、再現性はAとCから導かれ、Bは含まれない。数えた試行を再び加えると`already counted`で拒否 |
| P-JOB-01(工程2) | `TestPJOB01FailuresAndMismatchesDoNotYieldReproducibility` | 部分出力を残して終了状態2で終わる構築器(模擬)。毎回違う出力を出す構築器(模擬)の二回構築。一回の実行Aを、順序番号だけ変えた写しとして加える、または履歴に二度並べる | 部分出力(42バイト)は退避され、成功0件で再現性は導かれない。二回の出力が違うと`not reproducible`で、保管庫は受け付けない。写しは`not issued by this host`、二度並べた履歴は`already counted`、一件だけでは`two successful attempts`が必要で、いずれも再現性を導かない |
| 固有(工程3) | `TestClosureThatCannotBeFixedStops` | 未追跡のファイルを置いた複製(汚れた作業ツリー)。外部モジュールをimportし`go.mod`に要求を加えてコミットした複製。プロセスプラグインのソースを一行変えてコミットした複製。同じGoを包むシェルスクリプトを構築器とする観測。ホストのGo設定 | 汚れた作業ツリーは工程1と実行器の両方が停止。外部モジュールは工程1(隔離した`go list`が`missing go.sum entry`)と実行器(終了状態1)が停止。ソースの変更は工程1が別の計画を導き、工程3の照合が`source closure`の不一致で停止。包んだGoは`go version`が同じでも照合値が変わり、工程3の照合が`toolchain`の不一致で停止。ホストのGo設定を与えても三つのディレクトリは空のままで、実行器の成果物は工程2(設定なし)と同じダイジェスト |
| 固有(工程1〜4) | `TestThroughDetectsSwappedIdentityAndMissingStage` | 工程3の三つのダイジェストを揃えて別の値に差し替える。工程2の計画の照合値を差し替える。工程2を除く | 差し替えでは四つの工程の単独確認(`SelfCheck`)はすべて通り、通しの照合だけが`stage 3 ... differs`、`stage 2 plan fingerprint ... differs`で不合格。工程2を除くと`missingStages=[2]`で不合格。同じ試験で、工程1・2の固定入力の資産(解決、試行の証跡、保管庫)の単独確認を実行すると、このサンプルの識別とは無関係に終了状態0、1(意図した失敗の例示)、0を返す |

## 通しの完了条件

`TestThroughPluginBuildSupply`が、四つの条件をすべて確かめます。

- 工程1で確定した計画の照合値(観測値`1ac3a258…`)が、工程2の試行と再現性の記録、工程3の実行器の計画・成果物の固定・ホスト登録、工程4の呼出しの来歴(`buildPlanFingerprint`)まで同じ値です。計画は工程1で一度だけ導き、実行器が自分で組み立てた計画はそれとの照合にだけ使います。
- 保管庫から再取得したバイト列のダイジェスト(`sha256:f596a2ae…`)が、工程2の二回の実構築、工程3の実行器の二回構築、ホスト登録時のダイジェスト、工程4で検証器が呼出しの直前にハッシュした値と同一です。
- 実行器の検証根拠で、変換の構築器の第一段階と第二段階が収束し(同じダイジェスト)、配布一式の計画が変換エンジンの識別と構築器のダイジェストを束縛し、配布一式とTEI-native実現が再現します。処理プラグインの構築器は初期のGoの構築器(`tei.builder.go.seed`)のままで、収束によって消えません。TEI-native実現のエンジンの識別は、工程3の検証根拠と工程4で束縛した環境計画で同じ値です。
- 構築済みのプロセスプラグインをプロセス通信の版1で呼ぶと`accepted`が返り、引数の目印が保たれ、来歴の成果物のダイジェストは検証済み(`digestVerified=true`)で、環境計画の固定情報の照合値は工程3の値と同じです。工程が欠けたとき、照合は欠けた工程を`missingStages`として返し、通しの完了としません。

通しの記録は`chain.json`(`TEI_PBS_OUT_DIR`)に書きます。

## 実行

作業ブランチのコミットを浅いcloneした清浄な複製の中で実行します。工程3の実行器は変更や未追跡のファイルがある作業ツリーを拒否し、試験はその拒否も合否に使うためです。

```bash
git clone --depth 1 --branch <作業ブランチ> file:///<リポジトリ> /tmp/tei-pbs
cd /tmp/tei-pbs
python3 tools/verify-plugin-build-supply.py --out /tmp/tei-pbs-receipt
```

- スクリプトは`go env GOROOT`(`GOPROXY=off`で、ツールチェーンを取得しない)から正確なGoの実行体を選び、`TEI_PBS_REPO`、`TEI_PBS_GO`、`TEI_PBS_OUT_DIR`と`GOPROXY=off`、`GOFLAGS=-mod=readonly`、`GOTOOLCHAIN=local`を与えて`go test -json -count=1 -p 1`を実行します。全試験がPASSでskipが0件、通しの照合が完了、試験の後も作業ツリーが清浄なときだけ合格です。
- 開発に使ったWSL2では、`~/sdk/go`はGo 1.25.14で、`GOTOOLCHAIN=auto`によりモジュールキャッシュに既にある`golang.org/toolchain@v0.0.1-go1.26.9.linux-amd64`が選ばれます。スクリプトはこのGOROOTの`bin/go`を使います。取得は行っていません(receiptの`goRootFromModuleCacheToolchain`)。
- 環境変数がないと、通しの試験はskipします(単体試験はどの環境でも動きます)。通しの試験はLinux専用です(プロセスグループと、記述子の実行体名をPATHで解決する起動)。
- 一回の実行は約2分です(工程2の実構築2回と止めた1回、実行器の一回の実行を含みます)。

## 時間に依存する判定

時刻で合否を決める判定はありません。強制終了の時点は、構築器が標準出力へ`started`を書いたことで決め、時間待ちを使いません。試験の時間制限(`context`の20分、`StdioProcessLauncher`の2分)は異常時に止めるためだけのもので、合否には使いません。

## 安定性の記録

WSL2(Ubuntu、12 CPU、Go 1.26.9 linux/amd64)で、作業ブランチの浅いcloneを毎回作り直して`tools/verify-plugin-build-supply.py`を実行しました。

| 試験の版 | 回数 | 結果 | 試験の時間(秒) |
| --- | --- | --- | --- |
| `3437771` | 1 | 11件PASS、FAIL 0、SKIP 0 | 127.0 |
| `e60abd7`(試験コードは`3437771`から診断の表示一行だけ変更) | 5 | 5回とも11件PASS、FAIL 0、SKIP 0、試験後も作業ツリーは清浄 | 200.3、130.0、90.1、189.6、126.6 |

不安定な試験は観測していません。時間のばらつきは、隔離した空のキャッシュでの実構築の時間によるものです。計画の照合値(`1ac3a258…`)と成果物のダイジェスト(`sha256:f596a2ae…`)は、同じ試験コードの版の間で同じ値でした(版が変わると作業ツリーの照合値は変わるが、処理プラグインのソース閉包とGoが同じなので計画と成果物は変わらない)。

## 基盤の不足

| 要件 | 不足 | サンプルでの扱い |
| --- | --- | --- |
| 固有 | 構築意図から確定計画を導く入口がコアになく、構築意図の参照実装はGoのソース閉包を解決しない(`DirSourceResolver`は固定入力用) | サンプル側の`Derive`が`gobuilder`で閉包を解決する。構築意図のCLIとはソース以外を照合し、ソースの照合値は二つの方式で違う値になることを記録する(観測値で二件とも不一致) |
| 固有 | 自己構築の実行器が計画を外から受け取らず、自分で組み立てる。成果物のバイト列を外へ出さない | 実行器の計画を工程1の計画へ照合する(`BindStage3`)。保管庫へ入れるバイト列は工程2の実構築のもので、実行器の二回構築と同じダイジェストであることで結ぶ。実行器の出力そのものを保管したわけではない |
| P-GOV-01 | 構築器の提供元の解決、固定、信頼の失効をコアが持たない。参照実装は固定入力だけで、失効を扱わない | サンプル側で解決・固定・再生・失効を作った。適合の検証根拠(`reference-builder-contract-resolution`の`Conformance Evidence`)は扱っていない |
| P-JOB-01 | 構築の試行の型つきの終了状態と発生識別子をホストが供給しない | サンプルの`Host`が発生識別子を発行し、`os.ProcessState`から終了状態を読む。製品のホストの型ではない |
| 固有 | 保管庫の公開の入口がない(`PutBytes`、`GetBytes`は非公開) | 同じ配置と規則を`Store`で再現した |
| 固有 | 処理プラグインの計画はエンジンを束縛しない。変換エンジンの識別はソース閉包とツールチェーンの照合値で、実際にリンクされた実体の識別ではない | 照合するエンジンの識別は、工程3と4のTEI-native実現の固定情報と、工程3の配布一式の計画への変換エンジンの束縛に限る |
| 固有 | 配布一式、TEI-native実現、変換の構築器の計画は、工程1で導いていない(実行器が組み立てる) | 工程1が導くのは`tei.process.reference`の計画だけ。他の計画は実行器の検証根拠の内部の整合だけを照合する |

## 範囲外

- 実際の署名、信頼の起点、提供元の適合の検証根拠。失効は合成の`TrustStore`です。
- OSレベルの隔離。構築器の隔離は環境変数とディレクトリによるもので、`gobuilder`の範囲です。
- 二回構築の不一致と部分出力の失敗は模擬の構築器で起こしました。実際の`go build`の非決定性やリンカの異常終了は起こしていません。
- 遠隔の保管庫、複数のホスト、本番の配備。
- `tools/reference-build-execution-binding`(構築計画の型の分割)と`tools/reference-plugin-bundle`は、設計どおり工程に入れていません。
