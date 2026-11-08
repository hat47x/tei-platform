# 外部Harnessの送出請求を要約する参照ツール

外部AIハーネス（DSH等）がMock Providerへ送った**HTTP要求body**を、後から比較できる形へ要約します。要求bodyをそのまま保存・公開せず、**何が載ったか**だけをJSONへ落とします。

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)（H6-E2・P0-b）、[#2509](https://github.com/hat47x/tei-platform/issues/2509)
- 関連する測定記録: [Session Log拡張の実送出](../../plan/evidence/DSH_SESSION_LOG_EGRESS_MOCK_2026-11-07/README.md)、[既定・再試行・切断](../../plan/evidence/DSH_SESSION_LOG_RETRY_AND_DEFAULT_2026-11-07/README.md)、[再開と既送出Session Log](../../plan/evidence/DSH_H6E7_RESUME_AFTER_DISCLOSURE_2026-11-07/README.md)
- 状態: **要約専用の参照実装**。送出の許可・拒否、機密Profileの適合、Provider側の保存は判定しません。

## 何を解くか

`#2515`のP0-bは「モデル本文とは異なるRequest extension／Session Eventの実送出byteをLocal Mockで確認し、disable/in-flight/backlogの状態差を区別する」ことを求める。測定のたびに手作業でbodyを読み直すと、**モデル入力と拡張fieldを取り違える**、**byte数を混ぜて数える**、**圧縮や連結frameで取りこぼす**といった事故が起きる。

このツールは、1要求ごとに次を**別の欄**として出す。

| 欄 | 内容 |
| --- | --- |
| `bytes` / `sha256` | 受け取ったbodyの実バイト数とdigest |
| `topLevelKeys` | 最上位のキー名（値は出さない） |
| `model` / `stream` / `messageCount` / `toolCount` | モデル入力側の形 |
| `modelInputBytes` | `messages`・`system`・`tools`だけを直列化したバイト数 |
| `sessionLog.fieldBytes` / `keys` / `afterSeq` / `throughSeq` / `eventCount` / `eventTypes` | 拡張fieldの形と水位 |
| `modelInputHits` / `sessionLog.markerHits` | 合成markerが**どちらに**現れたか（回数） |

**モデル入力と拡張fieldは決して合算しません。** 同じmarkerが両方に載る場合も、それぞれの欄に別々の回数として残します。

## 使い方

```sh
# Mock Endpointが保存した *-request.json を入力にする
go run ./tools/reference-harness-wire-summary \
  -requests <capture-dir> \
  -markers "SYNTHETIC_MARKER_A,SYNTHETIC_MARKER_B" \
  -out <summary.json>
```

- `-out`は**存在しないパス**である必要があります。既存ファイルを上書きしません（測定記録を事故で消さないため）。
- 要求bodyを読みますが、**新たな通信はしません**。認証情報も読みません。
- 並びはファイル名順で固定し、同じ入力から同じ出力が得られます。

## 出力の読み方と限界

`limits.notProven`に、この要約が**示さない**ことを明示的に書きます。

- 他の経路（Telemetry・MCP・Subagent）で何も送られなかったこと
- 記録されたfieldの送出が**許可されていた**こと
- Providerがbodyを保存・保持・再利用したか

また、次は本ツールの対象外です。

- **圧縮・連結された永続ログ**の展開。HarnessのSession Logは連結zstd frameで保存されることがあり、**単一frameとして展開すると内容の大半を取りこぼす**（[保存面の測定記録](../../plan/evidence/DSH_H6E6_SINK_INVENTORY_2026-11-07/README.md)）。本ツールは**HTTP要求bodyだけ**を扱う。
- 実OS/ACL・Network境界での拒否、Providerの実保存。
- 業務意味・権限・Owner受理。

## 実装の分割

| 場所 | 役割 |
| --- | --- |
| [`tools/internal/harnesswire`](../internal/harnesswire/) | bodyの解析と要約（純粋関数。I/Oなし） |
| [`tools/reference-harness-wire-summary`](./) | ディレクトリの読み取り、順序、出力の拒否規則 |

解析を`internal`へ置くのは、**同じ要約を別のConsumer（CLI・検証script）から再利用するため**であり、Harness専用の契約や第二Registryを作るためではありません。`SessionLogField`という名前は**この版のHarnessが使うfield名の記録**であり、TEIの製品契約ではありません。

## 検証

- `go test ./tools/internal/harnesswire ./tools/reference-harness-wire-summary -count=1`
- `go vet ./tools/internal/harnesswire ./tools/reference-harness-wire-summary`

試験は次を固定します。

- markerがモデル入力と拡張fieldの**それぞれ別の欄**に現れること（混ざらないこと）
- 拡張fieldが無い要求・`null`の要求を「無し」として扱うこと
- **解析できないbodyを捨てず**、バイト数とdigestを残したまま`problem`を添えること
- 既存の出力ファイルを**上書きしない**こと
- `-markers`や`-requests`の欠落を拒否すること

## 作らないもの

- Harness専用の共通Schema、第二のAuthorizer、Registry。
- Provider・ホストの実効隔離の判定。
- 実secretの取り扱いを前提にした機能。
