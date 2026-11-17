# セルフビルドを外側から観測した実測（三段、正例と負例、変更実験）

[#2831](https://github.com/hat47x/tei-platform/issues/2831)と[#2600](https://github.com/hat47x/tei-platform/issues/2600)の実測記録です。判断は[設計文書](../design/SELF_BUILD_DECLARED_REPORTED_OBSERVED_2026-11.md)にあります。

**G4の判定ではありません。** `qualified`は`false`のままです。**TEI全体のセルフビルドの達成でもありません。** 対象は、Goで書かれた三つのコマンドです。

## 条件

| 項目 | 値 |
| --- | --- |
| 対象の版 | `main` `dcb076699c636ffefde682fe8370da5c203ab4da`に、本ブランチの変更を載せた作業木 |
| ホスト | WSL2 Ubuntu 24.04、`linux/amd64`、1台 |
| Go | go1.26.11。closure digest `sha256:f2186c6e99baa3ca77f2204e4beef05672a9aabd6b42e7f640ccd1229ac00e3c`（builderが計算した値） |
| 外部moduleの供給 | ホストのmodule download cache。供給の検査は行っていない（`supplyCheck.verified: false`） |
| 観測 | [`reference-build-process-observation`](../../tools/reference-build-process-observation/README.md) |
| 回数 | 各1回 |

**ほかのOS、ほかのアーキテクチャ、CIでは実行していません。**

対象の版には注意が要ります。実行に使った木は、コミット済みの木と同じ内容ですが、gitの同一コミットを取り出したものではありません。Windows側の作業木からWSL側の複製へファイルを写して実行しました。写した後に、コードの5ファイルが一致することを確かめています。

## 1. 三段の実行

対象は三つです。`./cmd/tei`（CLI）、`./tools/reference-go-build-from-plan`（builder）、`./tools/reference-build-catalog-authoring`（catalogを作るツール）。

| 段 | 動かしたもの |
| --- | --- |
| Stage0 | 外部のGoで三つをビルドした（`go build -trimpath`） |
| 宣言 | 三つの宣言を`go list -deps`から書いた。toolchainはdigestで名指した。以後は書き換えていない |
| Stage1 | Stage0の`tei`が、Stage0のbuilderを起動して三つを建てた |
| Stage2 | Stage1の`tei`が、Stage1のbuilderを起動して三つを建てた |

どの試行も、`tei self-build`を`-build-catalog`と`-hand-over-plan`で呼びました。計画は`tei`がin-processで解決し、builderへ渡しています。各試行は二回ビルドします（再現の検査）。

Stage全体を観測器の下で実行しました。Stage2では開いたパスも記録しました。

### 成果物

| 対象 | Stage0 | Stage1 | Stage2 | Stage1＝Stage2 |
| --- | --- | --- | --- | --- |
| `tei` | `82b3d48fb0bc9860…` | `f502475c14b3e821…` | `f502475c14b3e821…` | 一致 |
| builder | `53ee6e04206c2d57…` | `e3934aabbc0014dd…` | `e3934aabbc0014dd…` | 一致 |
| catalog作成ツール | `d97bb87bcceb74cc…` | `5effd04b563329a5…` | `5effd04b563329a5…` | 一致 |

Stage0とStage1は三つとも違います。Stage0は作業木で直接ビルドし、Stage1以降は計画の入力だけを写した木でビルドするためです。このため、Stage0の実行物とStage1の実行物はdigestで見分けられます。

### 動いたプログラム

判定は[`verify_l3.py`](TEI_SELF_BUILD_OBSERVED_PIPELINE_2026-11-17/verify_l3.py)で行いました。読むのは、ディスク上の実行物と観測記録です。動いたプログラムを、そのdigestでStage0の三つ・Stage1の三つ・toolchainの下・それ以外に分けます。

| 分類 | Stage1 | Stage2 |
| --- | --- | --- |
| Stage0の`tei` | 3 | **0** |
| Stage0のbuilder | 6 | **0** |
| Stage0のcatalog作成ツール | 1 | **0** |
| Stage1の`tei` | — | 3 |
| Stage1のbuilder | — | 6 |
| Stage1のcatalog作成ツール | — | 1 |
| toolchainの下 | 1,644 | 1,644 |
| それ以外 | 9 | 9 |
| 合計 | 1,663 | 1,663 |

- builderを起動したのは、Stage1ではすべてStage0の`tei`、Stage2ではすべてStage1の`tei`でした
- 「それ以外」の9は、Stageの手順を進めるスクリプトです（`bash` 1、`mkdir` 4、`cp` 3、`sha256sum` 1）
- Stage2が開いた前のStageのファイルは、Stage1の`tei`の成果物（3回）とbuilderの成果物（1回）だけでした。Stage0のファイルは開いていません
- 六つの試行はすべて、`ok: true`、`l2b.claimed: true`、`planHandover: handed-over`、`reproduced`、`builderBinding: bound`でした
- 所要時間は、Stage1が107秒（実行だけを観測）、Stage2が560秒（開いたパスも観測）でした

### この節の限界

- **Stage2の観測記録には、読めなかった停止が10件あります。** 三段の実行に使った観測器は、スレッドの終了直後に問合せが失敗する競合を、追跡の失敗として数えていました（その後に直しました）。動いたプログラムの集計は、この影響を受けません。「Stage2が開いた前のStageのファイル」は、開いたパスの記録に依るので、この10件の分だけ不確かです
- Stageの中で三つの対象を順に呼ぶのは、スクリプトです
- 宣言は`go list`から作りました。常設の宣言ではありません
- 宣言のsubjectは三つとも`kind: "plugin"`です。Plugin以外を宣言できないためです
- Goのtoolchainは外部のseedです

## 2. 一つの試行を、宣言範囲と照合する

Stage1の`tei`で`./cmd/tei`を建てる試行を、単独で観測し、宣言範囲と照合しました。builderを二通りに替えています。

| 構成 | 宣言したbuilder |
| --- | --- |
| honest | Stage1のbuilder |
| copy | ラッパー。二回目の起動の前に、一回目のstateを二回目のstateへコピーし、Stage1のbuilderへ処理を渡す（[source](TEI_SELF_BUILD_OBSERVED_PIPELINE_2026-11-17/copywrap/main.go.txt)） |

copyは、[SB-C0 §4.55](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)の表の最後の行（entryが見逃す構成）です。

### entryの報告

二つの構成で、entryの報告は同じでした。

| 欄 | honest | copy |
| --- | --- | --- |
| `ok` | true | true |
| `l2b.claimed` | true | true |
| `artifactReproducibility.state` | reproduced | reproduced |
| `stateSeparation.basis` | entry-post-hoc | entry-post-hoc |
| `stateObservation.kind` | effect-observed | effect-observed |
| `builderBinding.state` | bound | bound |
| `toolchainObservation` | 計画が名指し、一致 | 計画が名指し、一致 |

### 観測と照合

| 項目 | honest | copy |
| --- | --- | --- |
| 動いたプログラム | 793 | 406 |
| うちentry | 1 | 1 |
| うち宣言したbuilder | 2 | 2 |
| うちtoolchainの下 | 790 | 401 |
| **宣言にない実行** | **0** | **2**（Stage1のbuilder。ラッパーが起動した） |
| 一回目のBuildの実行数 | 396 | 397 |
| 二回目のBuildの実行数 | 396 | **8** |
| **別のstepが持つ領域への読み書き** | **0** | **8,669** |
| 読取り専用の領域への書込み | 0 | 0 |
| 観測器が読めなかった呼出し | 0 | 0 |
| `boundedAbsence` | not-proven | not-proven |

copyの二回目のBuildは8プロセスでした。ラッパー、Stage1のbuilder、toolchainの6です。honestでは395のtoolchainのプログラムが動いています。copyの二回目は、コピーしたキャッシュからの再生でした。

### honestで残った、宣言にない読み書き

honestでも`boundedAbsence`は`not-proven`でした。宣言範囲の外で成功した読み書きは3,940件で、内訳は次のとおりです。

| 読み書き | 件数 | 開いたもの |
| --- | --- | --- |
| 二回目のBuildの隔離木（`/tmp/tei-plan-tree-N`） | 3,934 | builderが書いて削除し、`go`とcompilerが読んだ |
| 共有の一時ディレクトリ（`/tmp`）を開く | 3 | entry、builder（二回） |
| 時刻帯のデータ（`/usr/share/zoneinfo/…`） | 2 | `go`（各Buildで一回） |
| 出力先の隣の一時ファイル（`artifact-go-tmp-umask`） | 1 | `go` |

隔離木の3,934件は報告の不足です。entryの報告（`childIsolation.root`）は、一回目のBuildの隔離木だけを載せます。残りの6件は、計画が述べていない入力と書込みです。

### 役割ごとの読み書き（honest）

| 役割 | パス | 読取り | 書込み | 削除 |
| --- | --- | --- | --- | --- |
| リポジトリ（入力の元） | 1,793 | 38,330 | 0 | 0 |
| 隔離木（一回目） | 1,793 | 13,303 | 1,978 | 1,793 |
| toolchain | 12,843 | 39,803 | 0 | 0 |
| 外部moduleの供給 | 36 | 72 | 0 | 0 |
| state（二回分） | 19,482 | 29,547 | 19,294 | 4,579 |
| OSとの接点 | 5 | 3,964 | 4 | 0 |

toolchainとリポジトリへの書込みは0でした。供給から読んだのは36パスです。

## 3. 宣言した入力と、コンパイラが開いた入力

honestの一回目のBuildについて、隔離木へ写されたファイルを、開いたプログラムで分けました（[`consumed.py`](TEI_SELF_BUILD_OBSERVED_PIPELINE_2026-11-17/consumed.py)）。

| 分類 | ファイル | byte | 内訳 |
| --- | --- | --- | --- |
| 宣言した入力（隔離木へ写された） | 1,672 | 9,870,416 | |
| `compile`・`asm`・`link`が開いた | 487 | 3,539,607 | `.go` 487 |
| `go`だけが開いた | 1,085 | 5,769,712 | `_test.go` 1,079、他OS向けなどの`.go` 4、`go.mod`と`go.sum` |
| toolchainのどのプログラムも開かなかった | 100 | 561,097 | `_test.go` 52、`.go` 29、`.md` 8、`.json` 4、`.yaml` 3、`.tei` 2、`.jsonl` 2 |

コンパイラが開いたのは、ファイル数で29%、byteで36%でした。

`go list -deps`が述べるコンパイル対象（`GoFiles`・`SFiles`・`EmbedFiles`）は487ファイルで、観測した487ファイルと一致しました。

## 4. 変更実験

`./cmd/tei`について、編集ごとに三つを比べました（[`inc.py`](TEI_SELF_BUILD_OBSERVED_PIPELINE_2026-11-17/inc.py)）。

| 比べたもの | 求め方 |
| --- | --- |
| 宣言の鍵 | resolverが返す計画のfingerprint |
| コンパイル対象の鍵 | `go list`が述べるコンパイル対象の名前とdigest、`go.mod`と`go.sum`のdigest |
| 成果物 | builderが新しいstateで作り直した実行物のdigest |

編集は、リポジトリの写しに対して一つずつ行い、編集ごとに写しを元へ戻しています。

| 編集 | 宣言の鍵 | コンパイル対象の鍵 | 成果物 |
| --- | --- | --- | --- |
| （編集なし、二回目） | 同じ | 同じ | 同じ |
| `cmd/tei`の試験ファイルに追記 | 変わる | 同じ | 同じ |
| `core/effect/README.md`に追記 | 変わる | 同じ | 同じ |
| `cmd/tei/testdata`のファイルに追記 | 変わる | 同じ | 同じ |
| 他OS向けのファイルに追記 | 変わる | 同じ | 同じ |
| コンパイル対象の末尾にコメントを追記 | 変わる | 変わる | 変わる |
| コンパイル対象の先頭にコメントを追記 | 変わる | 変わる | 変わる |
| コンパイル対象の文字列を変更 | 変わる | 変わる | 変わる |
| コンパイル対象のファイルを追加 | 変わる | 変わる | 変わる |
| 試験ファイルを追加 | 変わる | 同じ | 同じ |
| Windows向けのファイルを追加 | 変わる | 同じ | 同じ |
| `go.mod`にコメントを追記 | 変わる | 変わる | 同じ |
| ルートの`README.md`に追記（宣言の外） | 同じ | 同じ | 同じ |

| 鍵 | 誤った再利用（鍵は同じ、成果物は変わる） | 無駄な再構築（鍵は変わる、成果物は同じ） |
| --- | --- | --- |
| 宣言の鍵 | 0 | 7 |
| コンパイル対象の鍵 | 0 | 1 |

### この節の限界

- 比べたのは鍵です。再利用する実装はありません
- 編集は13種類で、網羅ではありません。埋込みファイル（`//go:embed`）の編集、cgo、ビルドタグの変更、toolchainの差し替え、外部moduleの変更は試していません
- 変更実験は、三段の実行より前に書いた宣言（toolchainを名指さない）で行いました。鍵と成果物の関係には影響しません
- 待ち時間は測っていません

## 5. 途中で起きたこと

実測の途中で、断絶が三つ見つかりました。記録に残します。

**名指したGoが、別のGoへ切り替わった。** 最初の試行では`~/sdk/go/bin/go`を`-go`に渡しました。このファイルは1.25.14で、`GOTOOLCHAIN=auto`の下では1.26.11へ自分で切り替わります。補助ツールをビルドした起動側のシェルでは切り替わり、builderが作る環境（`GOTOOLCHAIN=local`）では切り替わりませんでした。builderは`go.mod`の要求を満たせず、ビルドしませんでした。

**その失敗の理由が、entryの報告に無かった。** builderは`buildSucceeded: false`と理由を出して終了コード0で終わりました。entryの報告には「成果物を読めない」とだけ出ました。`childBuildError`を足して直しています。

**計画と違う入力を、builderが拒否した。** 全鎖の実行中に、同じ作業木へ別の修正を写してしまいました。一回目と二回目のBuildの間で`cmd/tei`の内容が変わり、builderは二回目を「計画が名指すdigestと違う」として止めました。意図した負例ではありませんが、止まることを実対象で見たことになります。

## 6. 再現の手順

スクリプトは[記録のフォルダ](TEI_SELF_BUILD_OBSERVED_PIPELINE_2026-11-17/)にあります。linuxのファイルシステム上の作業木と、Go 1.26.11のtoolchainのディレクトリが要ります。

```bash
# 三段。<checkout>は作業木、<work>は作業用の空ディレクトリ、<goroot>はtoolchain
bash l3.sh <checkout> <work>/l3 <goroot>
python3 verify_l3.py <work>/l3 <goroot>

# 一つの試行の観測と照合。三段の結果（Stage1の実行物と宣言）を使う
JUDGE_L3=<work>/l3 JUDGE_REPO=<checkout> bash judge.sh honest
JUDGE_L3=<work>/l3 JUDGE_REPO=<checkout> bash judge.sh copy
```

`judge.sh`と`inc.py`は、作業用のディレクトリを`~/fable3`の下に置く前提で書いてあります。ラッパーのsourceは、Goのパッケージとして拾われないように`main.go.txt`という名前で置いています。使うときは`judge.sh`と同じ場所の`copywrap/main.go`へ写します。

生の出力は[`raw/`](TEI_SELF_BUILD_OBSERVED_PIPELINE_2026-11-17/raw/)にあります。利用者名を含むパスは`/home/user`に置き換えています。観測記録そのもの（一つ数十MB）は置いていません。

## 実行した検証

- `go test -count=1 ./tools/reference-build-process-observation`: PASS（linux/amd64）
- `go vet ./tools/reference-build-process-observation ./cmd/tei`: 指摘なし
- `go test -count=1 -run 'TestSelfBuild|Reserved|MarkPlan|ConfirmPlan' ./cmd/tei`: PASS（実Goビルドを含む。`TEI_SELF_BUILD_FULL_CHAIN`が要る試験はskip）
- 三段の実行、honestとcopyの観測と照合、変更実験: 上のとおり

**NOT RUN**: `go test ./...`の全体、`TEI_SELF_BUILD_FULL_CHAIN=1`の全鎖の試験、`repository-health`、CI、Windows・darwin・arm64、別のGo版、Studioの構築（Node、Wails、WebView2）、供給の検査を伴う試行。
