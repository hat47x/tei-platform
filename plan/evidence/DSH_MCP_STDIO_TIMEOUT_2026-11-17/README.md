# stdioの子が応答しないと`Request timed out`になり、子は殺されず`notifications/cancelled`が飛ぶ（2026-11-17、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-17（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.3`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルの合成MCPサーバーのみ**。実MCPサーバーへの接続は**NOT RUN**。
- 先行記録: [stdioの呼び出し中の死](../DSH_MCP_STDIO_MIDCALL_2026-11-17/README.md)（落ちる場合）。本記録は、そこで未確認とした「**ハングする場合**」と「**`toolCallTimeoutMs`を超えた場合**」を埋める。

## 1. 何をしたか

stdioのMCPサーバーが**ハンドシェイクと`tools/list`に答え**、**`tools/call`には応答せず眠る**ようにした。**`toolCallTimeoutMs: 3000`**。**待ち時間を2通り**（120秒・5秒）にして、**タイムアウト後に子がどうなるか**を見た。

## 2. 観測した結果

### 呼び出しはタイムアウトした

```
status: error
result: Error: Request timed out
```

- **4つ目の失敗形**である（下記）。

### 120秒のハング: 子は完了を記録しなかった

| 項目 | 値 |
| --- | --- |
| 子が完了を記録したか | **いいえ** |
| 実行時間 | 約22秒 |

- 眠り終える前に**実行が終わった**ので、**子がどうなったかは分からない**（実行と共に終了）。

### 5秒のハング: 子はタイムアウトを生き延び、キャンセルを受けた

ここが本記録の要点である。**子は殺されていなかった**。

```
tools/call 受信 t=1.453
hanging           t=1.469
survived-the-timeout t=6.469   ← タイムアウト(3秒)を超えて目覚めた
late-reply-sent   t=6.469      ← 応答を書いた
notifications/cancelled t=6.469 ← クライアントがキャンセルを送った
eof               t=6.469
```

- **5秒の待ちでは、子はタイムアウトしても殺されなかった**（目覚めて応答を書いた）。**少なくともこの1例では、タイムアウトは「子を止める」ではなく「要求を諦める」**である。**120秒側の子は実行終了と共に居なくなっており、クライアントが殺したかは分からない**。
- **諦めた後に、クライアントは`notifications/cancelled`を送った**（**1回観測**）。キャンセルの送信自体は記録できたが、**子がそれに従うかは未確認**。
- **この実行では、遅れて届いた応答は既に失敗した呼び出しに使われなかった**（`tool_result`は`Request timed out`のまま）。**別のターンで拾われるかは未確認**。

## 3. 落ち方・止まり方は4種類になった

| 状況 | 出るエラー | 出典 |
| --- | --- | --- |
| **stdio・呼ぶ前**に死ぬ | **`unknown tool`**（`UNKNOWN_TOOL`） | [子Processの死](../DSH_MCP_STDIO_CHILD_DEATH_2026-11-17/README.md) |
| **stdio・呼び出し中**に死ぬ | **`MCP connection closed`** | [呼び出し中の死](../DSH_MCP_STDIO_MIDCALL_2026-11-17/README.md) |
| **stdio・応答しない** | **`Request timed out`** | **本記録** |
| **http・落とす** | **`fetch failed`** | [サーバーを落とした場合](../DSH_MCP_KILL_MIDRUN_2026-11-10/README.md) |

- **4つとも別の文言**である。**エラー文言での検知は、測った組み合わせの外で取りこぼす**。
- **ただし4つのうち本記録が測ったのは1つ**（`Request timed out`）。他の3つは**リンク先の記録**の値であり、**本記録の生データには含まれない**。

## 4. この記録が示すこと

- **応答しない子は `Request timed out`** になる。**`MCP connection closed` とは別の文言**である（**この記録では繋がったまま**なので）。
- **この1例では、タイムアウト後も子は生きて遅れて応答を書き**、**クライアントは`notifications/cancelled`を送った**。**一般化はしない**（n=1）。
- 機密の観点では、**「タイムアウトした＝相手が止まった」ではない**。**相手は動き続けている可能性がある**。**外部Effectを持つToolでは、この区別が効く**（処理は続いているかもしれない）。

## 5. この記録が示さないこと

- **遅れて届いた応答が使われるか**（本記録では使われなかった。**別のターンで拾われるかは未確認**）。
- **`notifications/cancelled` を受けた子が従うか**（本記録の子は無視して応答を書いた）。
- **他の`toolCallTimeoutMs`値**。
- **http経路で応答しない場合**。
- **子が完全に無反応（CPU 0でブロック）の場合**との違い。
- 実MCPサーバー、サーバー側の保存。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 6. 生データ

| ファイル | 内容 |
| --- | --- |
| `stdio-hang-timeout.json` | `overlay`、`runLongHang`（120秒・完了記録なし・解釈）、`runShortHang`（5秒・**タイムアウト生存**・**キャンセル受信**・子側の全ログ）、`toolResultText`、`modelRequests`、`failureShapes`（4経路の対比）、`fourDistinctShapes`、`cancelProtocolObserved`、`caveat` |

生の要求bodyはコミットしていない。**合成のみ**（`hangmock`・`hang_echo`）。

再現する場合の要点:

```yaml
- insert:
    - id: mcp-client-stdio-hang
      name: "@deepseek-ai/dsh-mcp-client"
      config:
        transport: stdio
        serverName: hangmock
        command: python
        args: ["<script>", "<record-dir>", "5"]
        toolCallTimeoutMs: 3000
        failOnStartupError: false
```

- **待ち時間をタイムアウトより長くし、実行より短くする**。これで**「殺された」と「生き延びた」を区別**できる。**120秒だけ**では区別できなかった。
- 子は**眠りから覚めたら必ずログを書く**。**その行があるかどうかが判定**になる。
- **モデルにToolを呼ばせる**。呼ばないとタイムアウト経路に入らない。

### 手法上の注意（実際に踏んだ）

**モデル側のMockが「1回目の要求でToolを呼ぶ」実装だったため、Tool一覧が空の要求で発火して空振りした**。**「Tool一覧に目的のToolが入っている最初の要求」で発火**するよう直してから測り直している。

## 7. `H6-E2`への含意（判断はしない）

**タイムアウトは「相手が止まった」を意味しない**。**子は生きて処理を続けている可能性がある**。外部Effectを持つToolでは、**タイムアウト後の状態を「失敗」と決めつけられない**。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
