# 種（seed）は成果物のどこまで届くか

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のセルフビルドは、stage0 の Go を外部の種として明示しています（SB-C0 §3.2、§6）。本記録は、**種の違いが成果物のどこへ現れるか**を測ったものです。

**これは G4 の判定ではありません。** `qualified` は `false` のままです。**Seed Trust を確立する記録でもありません。**

## 結論

| 測ったこと | 結果 |
| --- | --- |
| 種の違う `tei`・builder に、同じ toolchain でコンパイルさせる | **成果物はバイト一致**。駆動役（`tei` と builder）は種を成果物へ持ち込まなかった |
| stage1 の `tei` が stage2 を駆動する（intent つきの全鎖） | **stage2 の成果物 ＝ stage1 の `tei`**（バイト一致） |
| 同じ計画を、別の toolchain でビルドする | **成果物のバイトが違う。五条件はどちらも成立** |
| toolchain を、もう一方の種でソースから作り直す | 実行ファイル10本が配布物と**バイト一致**（両方向）。モジュール zip のハッシュは不一致で、原因は調べていない |

読み取れること:

- **`tei self-build` にはコンパイラがありません。** バイトを作るのは Go の toolchain で、`tei` と builder は何をどう渡すかを決める駆動役です。成果物は「コンパイルした toolchain」で決まり、駆動役をどの種で作ったかには依りませんでした。
- **L3（stage1 が stage2 を駆動する）は、駆動役が自分を作り直せることを示します。Seed Trust は減りません。** stage2 が stage1 と一致するのは、同じ toolchain が同じ source をコンパイルしたからです。
- **計画は toolchain を名指していません。** `-go` は child の引数で、計画にも五条件の照合にも現れません。「成果物はその計画の出力である」は、計画の外にある toolchain に相対的な主張です。

## 測り方

環境は WSL2 Ubuntu 24.04、`linux/amd64`。対象の版は `main` `8c80dbff0b30e80d4b3e1bd18cf3ffb9c81715d4`。**各構成は一回の観測です。**

種は二つです。どちらも Go の公式配布の toolchain モジュールで、取得時にチェックサムデータベースと照合されたものです。

| 記号 | 版 | `bin/go` の digest |
| --- | --- | --- |
| X | go1.26.11 | `sha256:95499375…` |
| Y | go1.27.0 | `sha256:1db869c5…` |

対象は `./cmd/tei` です（宣言した source は83件）。経路は本物の `tei self-build` で、catalog から entry が計画を導き、builder は `reference-go-build-from-plan`、二回ビルドの再現確認つきです。手順は [run.sh](TEI_GO_SELF_BUILD_SEED_REACH_2026-11-13/run.sh)、生の結果は [raw/results.txt](TEI_GO_SELF_BUILD_SEED_REACH_2026-11-13/raw/results.txt) です。

## 1. 駆動役の種 × コンパイルする toolchain

駆動役の組（`tei` と builder）を、種 X と種 Y でそれぞれビルドしました。

| 駆動役の種 | コンパイル | `l2b.claimed` | 計画の fingerprint | 成果物の digest |
| --- | --- | --- | --- | --- |
| X | X | `true` | `072a3b31…` | `sha256:67690bf0…` |
| Y | X | `true` | `a054533a…` | `sha256:67690bf0…` |
| X | Y | `true` | `072a3b31…` | `sha256:3889193e…` |
| Y | Y | `true` | `a054533a…` | `sha256:3889193e…` |

| 比較 | 結果 |
| --- | --- |
| コンパイル X: 駆動役 X の成果物 ＝ 駆動役 Y の成果物 | 一致 |
| コンパイル Y: 駆動役 X の成果物 ＝ 駆動役 Y の成果物 | 一致 |
| コンパイル X: stage2 の成果物 ＝ 種 X で直接ビルドした `tei` | 一致（`67690bf0…`） |
| コンパイル Y: stage2 の成果物 ＝ 種 Y で直接ビルドした `tei` | 一致（`3889193e…`） |
| 駆動役 X: コンパイル X の成果物 ＝ コンパイル Y の成果物 | **不一致** |
| 駆動役 X: コンパイル X の計画ファイル ＝ コンパイル Y の計画ファイル | **バイト一致** |

- 計画の fingerprint は駆動役の種で変わります。計画が builder の digest を含むためです。それでも成果物は同じでした。
- stage2 が stage1 と一致したので、stage3 は回していません。駆動役のバイトが同じなら結果も同じです。

## 2. toolchain を、もう一方の種でソースから作り直す

toolchain モジュールに含まれるソースから、もう一方の種をブートストラップにして `make.bash -distpack` で作り直し、配布物と比べました。手順は [rebuild-toolchains.sh](TEI_GO_SELF_BUILD_SEED_REACH_2026-11-13/rebuild-toolchains.sh)、結果は [raw/results-rebuild-toolchains.txt](TEI_GO_SELF_BUILD_SEED_REACH_2026-11-13/raw/results-rebuild-toolchains.txt) です。

| 作り直した対象 | ブートストラップ | 所要 | `bin/go`・`bin/gofmt`・`pkg/tool/linux_amd64/*` の10本 | モジュール zip の SHA-256 |
| --- | --- | --- | --- | --- |
| go1.26.11 | go1.27.0 | 214秒 | **すべて配布物とバイト一致** | 不一致 |
| go1.27.0 | go1.26.11 | 227秒 | **すべて配布物とバイト一致** | 不一致 |

- **zip のハッシュは一致しませんでした。原因は調べていません。** zip の一致は主張しません。
- 一回目の実行では10本中9本が不一致でした。**原因は実験側の設定**で、環境に `CGO_ENABLED=0` を置いたために、作り直した `go` の既定値が `CGO_ENABLED='0'`（配布物は `'1'`）になっていました。設定を外した二回目が上の表です。一回目の記録は `raw/results.txt` に残しています。
- **これは Go の配布物についての外部の検証であり、TEI の生成物ではありません。**

## 3. toolchain を記録へ出す（`main` `c6bf44634a83d67a6fcefc1f12cd44f90374dabb` ＋ 本変更）

上の3行目（同じ計画・別の toolchain）は、変更前の報告では見分けられませんでした。toolchain が現れるのは child の argv だけだからです。本変更は**記録を足すだけ**で、計画の契約も五条件も変えていません。

| 欄 | 内容 | 出どころ |
| --- | --- | --- |
| builder の報告の `toolchain` | `-go` が解決した実行ファイルとその digest、その周りの toolchain（`VERSION`・`bin/go`・`pkg/tool/<os>_<arch>`・`src`）の digest | builder の申告。`gobuilder.HashToolchain` を再利用 |
| `toolchainObservation.state` | `matched`／`mismatched`／`unreported` | entry が、申告された実行ファイルを自分で読み直して比べる |
| `toolchainObservation.namedByPlan` | 常に `false` | 計画が toolchain を名指していないことの明示 |

`matched` は「申告されたパスのファイルが、申告されたバイトである」ことで、builder がそれを実行したことではありません。toolchain を報告しない child（Go 以外の builder、既存の試験用 child）は `unreported` になり、拒否されません。

TEST_RESULTS

## 言えること・言えないこと

| 言えること | 言えないこと |
| --- | --- |
| この対象・このホスト・この二つの種で、駆動役は成果物へ種を持ち込まなかった | 対象や条件を選んで発動する混入が無いこと（有限回の観測では否定できない） |
| stage1 と stage2 がバイト一致した | それが Seed Trust を減らすこと（同じ toolchain なら自明に成り立つ） |
| 配布された実行ファイル10本は、同梱のソースを別の種でコンパイルしたものと一致した | 両方の種に共通する混入、ソース自体の混入が無いこと |
| 同じ計画が、toolchain によって別のバイトになる | 成果物が宣言から一意に決まること |
| toolchain が記録に出る | 計画が toolchain を名指すこと（契約の変更であり、決めていない） |

**二つの種は独立な系統ではありません。** 同じ配布元の隣接する版です。種から独立であると主張するには、別系統の実装か、ソースからの長いブートストラップ鎖が要ります。実行していません。

## 実行していないもの

- `windows`・`darwin`・`linux/arm64`、CI
- 系統の独立な種
- builder 自身（`reference-go-build-from-plan`）を stage2 で作り直すこと。対象は `tei` だけです
- 外部モジュールの供給の独立性（ホストのモジュールキャッシュを使用）
- モジュール zip の不一致の原因の調査

## 持ち主の判断が要る点

1. **toolchain を計画の入力にするか、どの粒度の digest にするか。** 入れた場合、計画と違う toolchain では起動前に止めることになります。
2. G4 が Seed Trust について何を求めるか（外部の再構築の引用で足りるか、TEI 側の再実行を求めるか）。
3. SB-C0 §4.52 が G4 の「組込み機能」を L3 で埋めている行の文言。L3 が示すのは駆動役の作り直しまでです。
