# 停止したSession Log送出は、利用者から見えない（2026-11-09、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`／`P0-b`。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-09（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）
- 状態: **ローカルMock Endpointのみ**。実Providerへの送出は**NOT RUN**。
- 先行記録: [しきい値](../DSH_SESSION_LOG_MAXBYTES_THRESHOLD_2026-11-09/README.md)・[停滞後の再開](../DSH_SESSION_LOG_STALL_THEN_GROW_2026-11-09/README.md)。本記録は台帳の未確定「**止まったことが利用者に見えるか**」を確かめる。

## 1. 何をしたか

`maxBytes=306`（最初のeventが入らない上限）で4 Tool呼出しを実行し、**3つの面を別々に**採取した。

1. **標準出力**（`--json`のNDJSON）
2. **標準エラー**（`2>`で別ファイルへ。先行の記録は`2>&1`で混ぜていたため、どちらの面に出るかを区別できなかった）
3. **永続Session Log**（`session.v4.jsonl.zstd`を**frame単位で展開**し、51行を得た）

各面について、`maxBytes`・`session-log`・`too large`・`blocking event`・`stall`の語を数え、警告種別のeventがあるかを調べた。

## 2. 観測した結果

| 面 | 実測 | 警告の手掛かり |
| --- | --- | --- |
| 標準出力 | 6,176 bytes、event種別は`final`・`session`・`status`・`text`・`tool_call`・`tool_result`のみ | **該当語0件** |
| 標準エラー | **0 bytes** | 該当語0件 |
| 永続Session Log | 圧縮19,040 bytes → 展開**51,196 bytes・51行**。event種別20種（`request/header`・`tool/result`・`turn/end`等） | **該当語0件、警告／エラー種別のeventなし** |

- **fieldは6要求すべてで送られていない**（`--mock`側の捕捉で確認）。
- **どの面にも「止まった」ことを示す出力は無い**。終了コードは`0`、Sessionは`completed`で終わった。
- 永続Logのevent種別に`warn`・`error`系は**無い**。

## 3. この記録が示すこと

- `maxBytes`で送出が止まったとき、**利用者が通常見る3面（stdout・stderr・永続Log）のいずれにも警告が出ない**。
- したがって「Logを送出していない」と「`maxBytes`が効いている」を、**出力だけから区別できない**。送出が止まったことは、**捕捉側（Mock/Provider）で欠落を数えるまで分からない**。
- これは台帳の未確定項目「止まったことが利用者に見えるか」に対する**最初の実測**であり、答えは「**見えない**」（この構成・このHarness版・この3面について）。

## 4. この記録が示さないこと

- **他の面**（Telemetry・`stderr`以外のログ・HarnessのUI・`--verbose`等の別flag）に警告が無いこと。本記録は3面だけを見た。
- **止まっていない実行との対照**。先行の非停滞実行は`2>&1`でstderrを混ぜて採取していたため、同条件の比較にならない（**比較はNOT RUN**）。
- 単一eventが上限を超えた場合の振る舞い。
- 実Providerへの送出、Provider側の保存・保持期間、Crash後の重複送信。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `visibility.json` | Session ID、3面それぞれのbyte数・SHA-256（圧縮/展開の双方）・event種別・該当語の出現数・警告種別の有無、`fieldSent`、`exitCode`、`notProven` |

生の要求bodyはコミットしていない。**合成データのみ**。

再現する場合の要点:

```sh
# stderr を stdout と混ぜない（先行記録の 2>&1 は面を区別できない）
dsh --profile headless --patch <overlay maxBytes=306> --permission-mode read-only \
    --approval-policy never --json "Run the steps." > out.jsonl 2> err.txt
```

永続Logは**連結zstd frame**である。単一frameとして展開すると大半を取りこぼすため、**frameごとに展開**する（[保存面の記録](../DSH_H6E6_SINK_INVENTORY_2026-11-07/README.md)）。

## 6. `H6-E2`への含意（判断はしない）

「上限を小さくすれば送出を止められる」は成立するが、**止まったことが出力に現れない**ため、統制として使うなら**検知を別に用意する必要がある**（捕捉側の欠落検知、水位の監視、明示の状態表示など）。何を用意するかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
