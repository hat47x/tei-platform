# entryが渡したstateを自分で見ると、何が区別できるか

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.37 は、条件5の「state分離」が child の申告との文字列一致であると限定しました（[実測](TEI_GO_SELF_BUILD_STATE_SEPARATION_SCOPE_2026-11-11.md)）。本記録は、**entry が child に聞かずに見たもの**を足したとき、その構成がどう見えるかを測ったものです（[設計 §4.55](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)）。

**これは G4 の判定ではありません。** `qualified` は `false` のままです。**同梱の child に欠陥があるという記録でもありません。** 測ったのは検査が何を区別できるかです。

## 何を測ったか

測定は二つあります。対象の版が違うので分けて書きます。

| 測定 | 対象の版 | 内容 |
| --- | --- | --- |
| A. 試作の観測器 | `main` `4b1a3973ba3918b9445ef9b74e51c422858eab04` | entry の外から child を二回起動し、child の `stateDir` 申告を判定に使わない試作（[observer.py](TEI_GO_SELF_BUILD_STATE_OBSERVATION_2026-11-12/observer.py)）。OSの境界あり・なしを比べる |
| B. entry への反映 | `main` `8c80dbff0b30e80d4b3e1bd18cf3ffb9c81715d4` に本変更を載せた版 | `tei self-build` 自身が出す `stateObservation` |

環境はどちらも WSL2 Ubuntu 24.04、`linux/amd64`、go1.26.11 です。**他のOS・アーキテクチャ・CIでは実行していません。**

child の構成は三つです。いずれも本物の `reference-go-build-from-plan` を包むラッパーで、報告の `stateDir` は**常に渡された値**です。

| 構成 | 実際に使う state |
| --- | --- |
| honest | 渡された state（同梱 builder の通常の動作） |
| share | 二回とも同じ共有ディレクトリ（[#4299](https://github.com/hat47x/tei-platform/pull/4299) の構成） |
| copy | 渡された state。ただし二回目の開始前に、一回目の state を二回目へコピーする |

## A. 試作の観測器（`4b1a3973`）

対象は外部依存のない小さな Go コマンドです。計画は `reference-build-catalog-authoring` と `reference-build-authoring-intent` で作りました。

| 構成 | 変更前の entry | 境界なしの観測 | linux の user・mount 名前空間 |
| --- | --- | --- | --- |
| honest | `l2b.claimed=true` | PASS-LIMITED。各回の state に545ファイル・33,561,683 byte | PASS-LIMITED。二回とも実ビルド（6.7秒・6.1秒） |
| share | **`l2b.claimed=true`、バイト一致** | **FAIL**。「成功したのに、渡した state へ何も書いていない」。二回目は0.3秒（一回目6.4秒） | **FAIL**。共有先へ書けずビルドできない |
| copy | **`l2b.claimed=true`、バイト一致** | **PASS-LIMITED（見逃す）**。二回目は0.5秒 | PASS-LIMITED。二回目は6.6秒かかった |

読み取れること:

- **成果物のバイトが一致していても、share は「state へ効果がない」ことで区別できます。**
- **share の二回目は0.3秒でした。** 共有した state では二回目がキャッシュの再生になり、バイト一致は二回の独立した観測を意味しません。
- **境界なしの観測は copy を区別できません。** 渡したディレクトリに新しいファイルがあるからです。
- 名前空間の中では、作業領域・`$HOME`・`/dev/shm`・`/var/tmp`・`/mnt/c` への書込みがすべて失敗することを、起動のたびに確かめました。読取り専用にできなかったマウントは `/tmp/.X11-unix` の1つで、新しい `/tmp` の下に隠れます。
- 起動直後に読んだ `/proc/<pid>/exe` は、ラッパーでは `/usr/bin/env` や `python3.12` でした。`-child` に名指ししたファイルと、OSが動かしたイメージは別です。

生の要約は [raw/summary.txt](TEI_GO_SELF_BUILD_STATE_OBSERVATION_2026-11-12/raw/summary.txt)、手順は [run.sh](TEI_GO_SELF_BUILD_STATE_OBSERVATION_2026-11-12/run.sh) です。

## B. entry への反映（`8c80dbff` ＋ 本変更）

entry は名前空間を張りません。足したのは「境界なしの観測」に当たる記録だけで、**五条件と `l2b.claimed` の式は変えていません。**

本物の Go ビルド二回での対（`TestStateObservationOnARealGoBuild`。`TEI_SELF_BUILD_FULL_CHAIN=1` の opt-in、`linux/amd64`、24.8秒）:

| 構成 | `ok` | `reproduced`・バイト一致 | `l2b.claimed` | `stateObservation.kind` | 観測 |
| --- | --- | --- | --- | --- | --- |
| 同梱の `reference-go-build-from-plan` を直接起動（正例） | `true` | はい | `true` | **`effect-observed`** | 一回目・二回目とも545ファイル・33,561,719 byte |
| share のラッパー（負例） | `true` | **はい** | `true`（式は不変） | **`declared-only`** | findings: 「child は一回目に `…/state` を使ったと申告したが、そのディレクトリは実行後に存在しない」。二回目も同じ |

対象は既存の `cmd/tei/testdata/l2b-go-command` で、計画は entry が catalog から導いたものです。

軽い試験（本物の Go ビルドを使わない fixture の child）:

| 試験 | 確かめること | 結果 |
| --- | --- | --- |
| `TestStateObservationSeesAChildThatUsesTheStateItWasPassed` | 正例。`effect-observed`、findings なし | PASS |
| `TestStateObservationDoesNotAcceptAReportThatHidesSharedState` | share。バイト一致・`reproduced` のまま、`declared-only`。両方の回と「申告したディレクトリが存在しない」を名指し | PASS |
| `TestStateObservationDoesNotRejectAChildThatKeepsNoState` | 対照。state へ何も書かない child を拒否しない（`ok=true`、`declared-only`） | PASS |
| `TestStateObservationCannotSeeStateCopiedBetweenRuns` | 限界。copy は `effect-observed` になり、`notObserved` と `boundary` がそれを明記する | PASS |
| `TestSelfBuildRecordsTheProcessImageBesideTheNamedFile` | 直接起動は一致、スクリプト経由は不一致を記録。試行は失敗させない（linux のみ） | PASS |
| `TestAssessL2BRepeatsStateSeparationWithoutDependingOnIt` | `l2b.stateSeparation` は写しで、`claimed` を左右しない | PASS |
| `TestAssessStateObservationNamesEachReason` | 同一の実体・既存ファイルあり・未使用・読取り失敗を名指す | PASS |

既定の `go test ./cmd/tei -count=1`（opt-in なし）は **PASS**（188.3秒）でした。既存の試験ファイルは変更していません。`gofmt -l cmd/tei` は出力なし、`go vet ./cmd/tei` は指摘なし、`TestPlanReadmesMentionEveryEntry` は PASS です。

## 言えること・言えないこと

| 言えること | 言えないこと |
| --- | --- |
| entry は、渡した二つの state に各回が新しいファイルを残したかを、child に聞かずに見る | 二回が state を共有しなかったこと（copy を区別できない） |
| share の構成は、バイトが一致しても `declared-only` になり、申告との食い違いが findings に出る | share の構成で Claim が立たなくなること（式は変えていない。`l2b.claimed` は `true` のまま） |
| state へ何も書かない child は拒否されない | その child の二回が独立だったこと（`declared-only`） |
| linux の名前空間で share と copy が止まることを一度測った | entry がその境界を張ること（入れていない）。他のOSでの同等の手段 |
| `-child` のファイルと実プロセスイメージの食い違いが記録に出る（linux） | builder の同一性。計画が選んだ builder との照合（[#4114](https://github.com/hat47x/tei-platform/pull/4114)）は `main` に無い |

## 実行していないもの

- **`windows`・`darwin`・`linux/arm64`。** `processImage` は linux 以外では `notObserved` を返すだけで、その経路は linux 上の試験で関数を直接呼んだ範囲しか見ていません
- **CI。** `.github/workflows/tei-self-build-qualification.yml` での実行結果は、PR を出した後の run を見るまで分かりません。GitHub の `ubuntu-24.04` ランナーで非特権の名前空間が使えるかも未確認です（entry は使いません）
- **`./cmd/tei` 全体を対象にした全鎖**（`TestSelfBuildFullChain…`）での `stateObservation`
- **名前空間で copy が止まった直接の証拠。** 根拠は二回目の所要時間（0.5秒→6.6秒）で、コピーの失敗そのものは記録していません
- ネットワーク・IPC・常駐プロセス経由の共有、既存ファイル（ホストのキャッシュ）の読取り、継承した環境変数

## 限界

- 各構成は一回ずつの観測です
- `effect-observed` の判定は「ファイルが増えたか」です。builder が state をどう使うかの意味は見ていません
- プロセスイメージは起動直後の一回の読取りです。後からイメージを替えるプロセスは見えません。すぐ終了する child では読めず `notObserved` になります
- 値はコミットとホストに依存します
