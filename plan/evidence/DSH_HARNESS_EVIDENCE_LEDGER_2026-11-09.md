# DSH外部Harness実測の台帳（2026-11-09、#2509 / #2515）

- 対象Issue: [#2509](https://github.com/hat47x/tei-platform/issues/2509)（DSH外部Harness実証）、[#2515](https://github.com/hat47x/tei-platform/issues/2515)（実効隔離・全送出経路Conformance）。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)、比較計画は[TAI #168](https://github.com/hat47x/tai-pleroma/issues/168)。
- 状態: **既存の実行記録を並べた索引**。新しい実測は行っていない。個々の合否・Owner受理は各記録とIssueの担当。
- この文書の役割: 44件の実行記録が**それぞれ何を確定させ、何を確定させていないか**を1枚で見えるようにし、同じ測り方の重複を避ける。記録本文の複製はせず、各記録へ参照で結ぶ。

## 1. なぜ台帳が要るか

`#2515`のPressure Case（H6-E0〜E7）と`#2509`のH2/H3比較は、**記録が10件を超えて個別のREADMEに散らばった**。個別記録は「この計測で何が言えるか」を丁寧に書いているが、**横に並べると見えること**（たとえば「どのmodeでも読取りは止まらない」）は、どれか1件を読んだだけでは分からない。

台帳は、**主張ごとに「根拠になる記録」と「まだ根拠が無いこと」を対応づける**ための索引である。新しい正本や第二のEvidence Schemaを作るものではない。

## 2. 一覧

「確定」は**その記録が観測した範囲**で言えること。「未確定」は同じ範囲で言えないこと。

| 記録 | 対象 | 確定したこと（観測範囲） | 未確定 |
| --- | --- | --- | --- |
| [H2 Tool-only起動](./DSH_H2_TOOL_ONLY_READ_ONLY_2026-11-07/README.md) | H2・実Tool起動 | 固定実行物をDSHの実Sessionから起動し、引数・出力・Session IDを同一ログで突合できた | 正答、Skill効果、Egress、OS隔離 |
| [H2/H3 四Session](./DSH_H2_H3_FOUR_SESSION_2026-11-07/README.md) | H2/H3比較 | 同一Taskで4 Sessionを完了し、H3の2件が同一byteのSkill本文を受け取った | 回答の正否、Skillの増分価値（Task path不一致・Session再利用あり） |
| [回答のFact出現](./DSH_ANSWER_FACT_PRESENCE_2026-11-08/README.md) | 四Session回答 | 4回答とも、CLI応答の事実13件を文字として含む | 意味の正しさ、未知の保持、Skill効果 |
| [Session Log送出](./DSH_SESSION_LOG_EGRESS_MOCK_2026-11-07/README.md) | 送出 | 拡張fieldが要求に載り、Tool結果がモデル入力とは別経路でも出る | 実Providerの保存、他経路の網羅 |
| [既定・再試行・切断](./DSH_SESSION_LOG_RETRY_AND_DEFAULT_2026-11-07/README.md) | 送出の状態差 | 出荷既定でON、非2xxと空streamでは未受理で再送 | agent要求自体の自動再試行、Crash重複 |
| [maxBytes上限](./DSH_SESSION_LOG_MAXBYTES_2026-11-08/README.md) | 送出の上限 | 上限が小さいと水位で止まり、運べるevent数が減る | 分割送出（追測で確認）、単一eventの停滞 |
| [backlog分割と停滞](./DSH_SESSION_LOG_SPLIT_AND_STALL_2026-11-08/README.md) | 送出の上限 | 後続要求へ分割送出（`6`→`9`）、上限256ではfieldが一度も作られない、上限判定は生byte | 004以降が載らない理由、停滞の警告 |
| [maxBytesのしきい値](./DSH_SESSION_LOG_MAXBYTES_THRESHOLD_2026-11-09/README.md) | 送出のしきい値 | 307では載り水位`4`まで1eventずつ、**306では6要求すべてで載らない**。しきい値は最初のeventを包むfieldの直列化サイズ | 同じSessionで上限を上げた再実行（次行で確認）、停滞の警告 |
| [停滞後の再開](./DSH_SESSION_LOG_STALL_THEN_GROW_2026-11-09/README.md) | 停滞分の再送出 | 上限を上げた同じSessionの再実行で`afterSeq -1`から`9`まで10 event・1,273 bytesが1要求に載り、合計13 eventが流れた（停滞分は失われない） | `maxBytes`変更だけで自動送出されるか、詰まっているeventの可視化 |
| [停滞の可視性](./DSH_SESSION_LOG_STALL_VISIBILITY_2026-11-09/README.md) | 停滞の検知 | `maxBytes=306`でfieldが6要求すべて送られない状態を、**stdout・stderr・永続Logの3面を別々に**調べた。stdout 6,176 bytes・stderr **0 bytes**・永続Log展開51,196 bytesのいずれにも該当語0件・警告種別eventなし・終了コード0 | 他の面（Telemetry・UI・別flag）、非停滞との同条件比較、単一eventが上限超過時の挙動 |
| [停滞の検知](./DSH_SESSION_LOG_STALL_DETECTION_2026-11-09/README.md) | 停滞の検知可能性 | 停滞（306）と非停滞（4096）を**stderrを混ぜずに同条件**で比較。stdout・stderrは**両方とも警告なし**、差は永続Logの`session-log-deepseek/delivery-accepted`のみで非停滞4件・停滞**0件** | Telemetry・UI・別flag、単一event超過時の挙動、他版で同じevent名・意味になるか |
| [Subagent経路](./DSH_SUBAGENT_EGRESS_2026-11-09/README.md) | 送出経路 | `subagent`を1回強制し、親5要求と子1要求を採取。Session IDが2種類あり、子は`afterSeq -1`から`sessionFormatVersion 4`の拡張を送る。`prompt`のcanaryは親にも子にも載り、子は独立した永続Log（展開36,774 bytes・19行・`delivery-accepted` 1件）を持つ | Telemetry・MCP・`subagent_fork`・連鎖起動、実Providerの保存 |
| [Subagent fork](./DSH_SUBAGENT_FORK_2026-11-09/README.md) | 文脈の継承 | Turn 1で`pwsh`から親canaryを出して完走させ、Turn 2で同じSessionを再開してforkした。子は`afterSeq -1`の別Sessionだが**メッセージ数5**で親の完了Turnを引き継ぎ、**親canaryが子の本文にも永続Logにも現れる**（`subagent`の子はメッセージ数1） | 実行中Turnの継承、forkの連鎖、Telemetry・MCP |
| [Subagentの連鎖](./DSH_SUBAGENT_CHAIN_2026-11-09/README.md) | 連鎖の上限 | Sessionごとに1回`subagent`を強制して連鎖させると、Sessionは**3つ（親1・子2）で止まり孫は0**。子は`subagent depth 2 exceeds maxDepth 1`を**`isError: true`のTool結果**として受け取り、**その文言が子の次のモデル入力にも載る**。拒否後も子は要求を消費（各3要求） | `subagentMaxDepth`を上げた場合、forkの連鎖、Telemetry・MCP |
| [深さ上限を上げた連鎖](./DSH_SUBAGENT_DEPTH_RAISED_2026-11-09/README.md) | 上限の可変性 | `--dump-config-schema`で上限の在処を特定（`subagent`の`maxDepth`、既定`1`、`volatile`、同schemaに`maxActiveSubagents`既定`8`）。`--patch`で`3`にすると**Sessionは4つ（深さ0〜3）になり、深さ4で`subagent depth 4 exceeds maxDepth 3`**（`isError: true`）。12要求・合計765,677 bytes | 既定との正確なbyte比較（Mockの発火方針が違う）、`4`以上、`maxActiveSubagents`との相互作用、forkの連鎖、Telemetry・MCP |
| [forkの連鎖](./DSH_SUBAGENT_FORK_CHAIN_2026-11-09/README.md) | 継承と深さ | `maxDepth 3`で`subagent_fork`を連鎖させると**4 Session（深さ0〜3）で止まり、深さ4は`isError: true`で拒否**。forkした子の初回要求は**どれもメッセージ数1**で、**同一Turnの途中では完了Turnを継承しない**。送出量はspawn連鎖とほぼ同じ（765,350 vs 765,677 bytes、差0.04%） | 完了Turnを挟んだforkの連鎖、`4`以上、`maxActiveSubagents`との相互作用、Telemetry・MCP |
| [Turn境界をまたぐfork連鎖](./DSH_SUBAGENT_FORK_TURN_CHAIN_2026-11-09/README.md) | 継承の条件 | Turn 1で`pwsh`からcanaryを出して完走させ、Turn 2で同じSessionを再開してforkを連鎖させると、**forkした子の初回要求はどれもメッセージ数5で親canaryを含み、深さ3まで届いた**。14要求・786,786 bytes（同一Turn内は12要求・765,350）。深さ4は`isError: true`で拒否 | Turn 1が複数ある場合の増加量、`4`以上、`maxActiveSubagents`との相互作用、Telemetry・MCP |
| [同時起動の上限](./DSH_SUBAGENT_CONCURRENT_LIMIT_2026-11-09/README.md) | 同時数 | 1メッセージで12体の`subagent`を並列要求すると**8体が起動し4体が拒否**された。拒否は`isError: true`のTool結果で本文は`active child limit: 8`。**子が即応答する対照でも8/4で変わらない**ため、上限は**起動時に見る同時数**で、子の完了を待って順に通す形ではない。親1＋子8で13要求・1,074,677 bytes | 時間を分けた起動、`maxActiveSubagents`の変更、`maxDepth`との同時作用、Telemetry・MCP |
| [同時起動の上限は設定値に従う](./DSH_SUBAGENT_ACTIVE_CAP_2026-11-10/README.md) | 上限の可変性 | `subagent`の`maxActiveSubagents`を**3**にして1メッセージで**6体**を並列要求すると、**3体が起動し3体が拒否**された。拒否の本文は**`active child limit: 3`**で、**設定値がそのまま制限値**として現れる。既定`8`では8/4だったので、**上限は既定値の話であり設定で変わる** | 時間を分けた起動、`maxDepth`との同時作用、`0`や`1`にした場合、Telemetry・MCP、実Provider |
| [時間を分けた起動](./DSH_SUBAGENT_SEQUENTIAL_2026-11-10/README.md) | 同時性と総量の区別 | `maxActiveSubagents`**3**のまま、**1ターンに1体ずつ**子を起動し起動直後に完了させて4回繰り返すと、**4体すべて起動し拒否0**（子Sessionも4つ）。並列6体では3/3だったので、**この上限は「同時に生きている子」に効き、起動した総数の上限ではない** | 上限を大きく超える順次起動、子が完了しないままの要求、`maxDepth`との同時作用、レート制限 |
| [2つの上限の同時作用](./DSH_SUBAGENT_TREE_CAP_2026-11-10/README.md) | 資源の奪い合い | **`maxDepth: 3`と`maxActiveSubagents: 2`**を同時に設定し、根が子2体を並列起動（2体とも成功）、各子が孫2体を並列起動しようとすると、**孫は0体**で、子2体は**`active child limit: 2`**の拒否を受け取った。**深さ上限による拒否は0件**、**作られたSessionは3つ**（根＋子2）。**同時数は木全体で数え、先に埋まった方が勝つ** | 子が完了してから孫を起動した場合、3段以上の木、`maxActiveSubagents`を4以上にした場合、`subagent_fork`の木 |
| [Telemetry経路](./DSH_OTEL_PATH_2026-11-09/README.md) | 送出のgate | `session-telemetry-otel`は`dsh-base`が**既定で登録**しており、`mode`は`DSH_TELEMETRY_MODE || FEEDBACK_ONLY`、`exporter.url`は`DSH_TELEMETRY_OTLP_URL ?? 'https://dsh-otel-collector.deepseeksvc.com/v1/logs'`。`DSH_TELEMETRY_OTLP_URL`をローカルMockへ向けた1往復で**OTLP要求は0件**（モデル要求は2件・143,308 bytes）。**止めているのはプラグインの不在ではなく`FEEDBACK_ONLY`のgate**で、headlessにはfeedbackの面が無い | feedbackを与えた場合の送出内容、web・tui面、`DISABLED`の実測、`maxRequestBytes`超過、MCP |
| [feedback面の不在](./DSH_FEEDBACK_SURFACE_2026-11-10/README.md) | 送出の契機 | OTLP宛先をローカルへ向け`FEEDBACK_ONLY`にして**headlessで`/feedback`を打った**が、**コマンドではなく通常のプロンプトとしてモデルへ渡り**（canaryが要求に入った）、**OTLP要求0件・正本Logの`feedback/**` event 0件**。同梱実装では送出の契機は**`feedback/record`（と同一Sessionの`feedback/message-put|delete`、`seq >= inheritedEventCount`）だけ**で、**headless・ACP・JSON-RPCはslash commandを提供しない**。**導入済みProfileは`desktop`と`headless`の2つだけ** | Webクライアントでの実測、feedbackを与えた場合の送出内容、`DISABLED`との差、fork継承時の`inheritedEventCount` |
| [MCP経路](./DSH_MCP_PATH_2026-11-09/README.md) | 既定の面 | 既定のheadlessの面は**Tool 24個で名前に`mcp`を含むものは0**。`mcp-resources`は`dsh-base`が**登録済みだがサーバー未設定**で、`--dump-config-schema`の`configRef`は`unknownConfig`。`mcp-client`を`servers.synthetic {transport: http, url}`で挿すと`--dump-config`は受理するが、実行後も**Tool 24個・MCP Tool 0・MCP要求0件**（モデル要求は2件・143,308 bytes） | 正しい設定形を与えた場合の送出、MCPサーバー側の保存、stdio transport、認証、`mcp-resources`の実挙動 |
| [MCPの設定形](./DSH_MCP_CONFIGURED_2026-11-10/README.md) | 接続の条件 | 同梱の`dsh-mcp-client/lib/index.js`の`Config`は**1インスタンス1サーバー**で、`transport`は`stdio`か**`streamable-http`**（`http`は無く、`servers`入れ子も無い）。その形で挿すと**`server/discover`→`initialize`→`notifications/initialized`→`tools/list`の4件**が届き、**モデルのToolが24→28**（`mcp__synthetic__synthetic_echo`と`mcp-resources`系3つ）、要求bytesも約9,170増えた | `tools/call`の実行、stdio、認証、複数サーバー、`mcp-resources`の実挙動、実MCPサーバー |
| [MCPのstdio経路](./DSH_MCP_STDIO_2026-11-10/README.md) | 別transport | `transport: stdio`＋`command: python`で自作サーバーを起動すると、**`initialize`→`notifications/initialized`→`tools/list`の3行**が届き（**streamable-httpと違い`server/discover`が無い**）、**モデルのToolは24→28**（`mcp__stdiomock__stdio_echo`）。**子Processは環境変数49件を引き継ぎ**（合成canaryと`DEEPSEEK_BASE_URL`を含む）、**作業ディレクトリも親と同じ**。ただし**`DEEPSEEK_API_KEY`は渡っていない** | `env`設定で明示的に渡した場合、`cwd`設定、認証・再接続、`tools/call`のstdio経路、他の資格情報の可視性 |
| [MCPの`headers`](./DSH_MCP_HEADERS_2026-11-10/README.md) | 送り方 | streamable-httpで`headers`に合成canaryを入れると、**4件すべてのMCP要求（`server/discover`・`initialize`・`notifications/initialized`・`tools/list`）に同じ11個のヘッダ**が載り、**canaryは`Authorization`として毎回送られた**。**相手を確かめる前の最初の要求にも載る**。設定外のヘッダは**HTTPクライアントの標準のみ**（Cookie・環境の資格情報は付かない）。`user-agent`は`node`。**canaryはモデル要求には現れない** | `reconnect`時の再送、TLS、複数サーバー、`headers`の値がログへ残るか |
| [MCPの複数サーバー](./DSH_MCP_TWO_SERVERS_2026-11-10/README.md) | 並置 | `id`を分けた2エントリで2サーバーを挿すと、**両方が独立に4件ずつハンドシェイク**し（合計8件・到着は交互）、**モデルのToolは24→29**（`mcp__servera__tool_a`と`mcp__serverb__tool_b`＋`mcp-resources`系3つ）。**`serverName`が名前空間になるので衝突しない**。**1つだけ許可しても、エントリを増やせば増える** | `serverName`を同じにした場合、片方が落ちた場合、stdioとの混在、片方だけの`headers` |
| [MCPの`tools/call`](./DSH_MCP_TOOL_CALL_2026-11-10/README.md) | 往復 | `mcp__synthetic__synthetic_echo`を1回呼ぶと、MCPサーバーへ**`tools/call`（123 bytes）**が届き、**`arguments.text`にモデル生成のcanaryがそのまま入っていた**。**ワイヤ上のTool名は`mcp__<serverName>__<toolName>`ではなく元の名前**。サーバーの応答は**3回目のモデル要求に引数と結果の両方が載って**戻る | 引数の検閲・上限、stdio、認証、複数サーバー、`read_mcp_resource`、実MCPサーバー |
| [MCPの引数の大きさ](./DSH_MCP_ARG_SIZE_2026-11-10/README.md) | 引数の上限 | **200,000 bytes要求→200,018 bytes受信**、**1,000,000 bytes要求→1,000,018 bytes受信**で、**どちらもcanaryが先頭にあり末尾100文字が全て`z`**（切り詰めも拒否も無い）。**Tool結果後のモデル要求は約1.05MB／約5.05MBへ膨らむ**（引数が文脈に残るため） | 1MB超、Provider側の上限、内容の検査、`maxInstructionBytes`との関係、stdio・認証・複数サーバー |
| [上限を超える単一event](./DSH_SESSION_LOG_OVERSIZE_EVENT_2026-11-09/README.md) | 送れない単位 | `maxBytes 4096`で約8,000 bytesの`pwsh`結果を出した後も呼び出しを続けると、**要求001〜004は拡張あり（水位`-1`→`9`→`11`→`12`）、005以降は`dsh_session_log` field自体が無い**。水位は`12`で凍り作業は96 event・maxSeq 94まで進む。**`delivery-accepted`は4件で止まり増えない**。**同じ上限で小さい結果のみなら拡張は付いたまま**（3/3） | Telemetry・UI・別flagでの検知、上限を戻した回復、`maxRequestBytes`との関係、他版での再現 |
| [上限を戻した回復](./DSH_SESSION_LOG_OVERSIZE_RECOVERY_2026-11-10/README.md) | 回復 | `maxBytes 4096`のRun 1は**10要求中3要求だけ拡張あり・水位`11`で凍結**（22 event・606,472 bytes）。**同じSessionを既定の上限で再開**すると、**最初の1要求で86 eventが届き水位`11`→`97`**、以後は通常配送（5要求・121 event）。**凍っていた分は失われずburstで戻る** | Telemetry・UI・別flagでの検知、burstの上限、`maxRequestBytes`との関係、他版での再現 |
| [backlogのburst](./DSH_SESSION_LOG_BURST_2026-11-10/README.md) | 分割の有無 | `maxBytes 256`で30回の呼び出しを強制したRun 1は**32要求すべてで拡張なし・送出0**（Logは2,809,857 bytes分 溜まる）。**同じSessionを既定の上限で再開**すると、**最初の1要求で243 event・375,623 bytesが届き水位`-1`→`242`**、**分割されない**。その中に**21,297 bytesのevent**（上限256を超える）も含まれた。Run 2合計483 eventで**取りこぼしなし** | 243 event・375KBを超える場合、`maxRequestBytes`との関係、Telemetry・UIでの検知、他版での再現 |
| [H6-E0 Host棚卸し](./DSH_H6E0_HOST_INVENTORY_2026-11-07/README.md) | 実行環境 | read-onlyはConstrainedLanguage、親のenvは子へ継承されない、loopbackへ到達可 | 別Host、他Tool経路、実Egress |
| [H6-E1 read境界](./DSH_H6E1_READ_BOUNDARY_2026-11-07/README.md) | 読取り | `read-only`でもraw配下・`.env`・cwd外を読める | AppContainer・別User・mount名前空間での拒否 |
| [H6-E1 OSのACL](./DSH_H6E1_OS_ACL_2026-11-09/README.md) | 読取りの強制点 | `icacls`で拒否したファイルは`read-only`でも`danger-full-access`でも`PermissionDenied`。同Userへ権限を足すと同じProbeが成功する（正負対） | 別User・AppContainer・mount名前空間、Network egress、存在の推測 |
| [workspace-write非対称](./DSH_WORKSPACE_WRITE_ASYMMETRY_2026-11-07/README.md) | 書込み | root外への書込みは拒否、同じroot外の読取りは通る | 読取り拒否の実OS構成 |
| [子Process到達範囲](./DSH_CHILD_PROCESS_REACH_2026-11-08/README.md) | Process | 子`cmd.exe`は親の読取り範囲を引き継ぎ、書込みはmodeに従う | 別OS資格での子起動、連鎖起動 |
| [H6-E3 暗黙Context](./DSH_H6E3_IMPLICIT_CONTEXT_2026-11-07/README.md) | 暗黙入力 | AGENTS.md・CLAUDE.md・Skill `description`がTool未読で載る | 解決範囲、Tool poison |
| [H6-E4 Lane分離](./DSH_H6E4_LANE_SEPARATION_2026-11-07/README.md) | 並列Lane | ディレクトリ分離だけでは相手Laneのcanaryを読める | 別Identity/ACL/mount/egress の実拒否 |
| [H6-E5 候補逸脱](./DSH_H6E5_CANDIDATE_ESCAPE_2026-11-07/README.md) | 生成候補 | 隣接Source読取りと子Process起動は止まらない、自己書換えは言語モードで停止 | workspace-writeでの自己書換え、D1→D2 |
| [H6-E6 保存面](./DSH_H6E6_SINK_INVENTORY_2026-11-07/README.md) | ローカルsink | stdoutと永続Session Logに残る、storages/cache/attachmentsには残らない | Issue/CI/外部Provider、保持期間 |
| [H6-E7 再開](./DSH_H6E7_RESUME_AFTER_DISCLOSURE_2026-11-07/README.md) | 再開 | Session Logは水位から先を再送しない、モデル入力には過去会話が再構成される | Grant失効、ACK不明replay |

## 3. 横断して言えること（複数記録の一致）

1. **DSHのSandbox modeは読取りの境界を与えない**。`read-only`（H6-E1）・`workspace-write`（非対称）・子Process（H6-E2b）の3経路すべてで、宣言root外の読取りが成立した。境界になるのは**OSのACL**であり、そこを拒否すればmodeに関係なく止まる（H6-E1 OSのACL・正負対）。
2. **読んだ内容は複数の経路で外へ出る**。モデル入力・Session Log・stdout・永続ログ（H6-E6）。Session Logの複製はTool結果1件につき2箇所（`data.message`と`data.meta`）。**Subagentを起動すると子も別Sessionとして同じ拡張を送り、独立した永続Logを持つ**（Subagent経路）ため、送出量は親Sessionの分だけでは数えられない。さらに**`subagent_fork`は親の完了Turnを子へ運ぶ**（Subagent fork）ため、親が読んだ内容が子のモデル入力にも載る。**継承しない`subagent`と継承する`subagent_fork`は同列に数えられない**。`subagent`の**連鎖は既定の`maxDepth 1`で止まる**（Subagentの連鎖）が、止め方は**Toolエラー**であって送出の停止ではなく、**拒否の文言も子のモデル入力に載る**。しかも**上限は設定で緩められ**、`maxDepth 3`では深さ3まで連鎖して**段ごとにSessionと送出が増える**（深さ上限を上げた連鎖）ため、**既定値に依存した「止まる」を保証として書けない**。`subagent_fork`の連鎖も**同じ上限で止まる**が、**同一Turn内では完了Turnを継承しない**ため文脈は積み上がらず、送出量もspawn連鎖とほぼ同じ（forkの連鎖）。**forkが文脈を運ぶのは完了Turnを挟んだ場合**で、**Turn境界をまたげば親の完了Turnが深さ3まで運ばれる**（Turn境界をまたぐfork連鎖）。つまり**「forkの連鎖は文脈を積み上げない」は同一Turn内に限った話**である。また**同時起動数は既定で8体**（同時起動の上限）で、9体目以降は**Toolエラー**として拒否される。深さと同時数のどちらも**上限値は設定でき**、拒否は**Toolエラー**であって送出の停止ではない。**Telemetry経路も既定で登録済み**で、宛先の既定値を持ち、**明示的なfeedbackが無い限り送らない**（Telemetry経路）。したがって**「経路が無い」ではなく「gateが閉じている」**として扱う。**MCPはこれと逆向き**で、既定の面に**Toolとして現れない**（MCP経路）ため、**「登録されている」と「モデルから使える」を分けて確かめる**必要がある。ただし**設定形（`streamable-http`・1インスタンス1サーバー）を合わせれば繋がり**、**起動時にハンドシェイク4件が外へ出て、サーバーのTool定義がモデル入力に載る**（MCPの設定形）。さらに**`tools/call`はモデルが生成した引数をそのまま外部へ送り、外部の応答がTool結果として文脈に戻る**（MCPの`tools/call`）ため、**MCPは「読む経路」ではなく「往復する経路」**として扱う必要がある。しかも**引数に少なくとも1MBまでの上限は無く**（MCPの引数の大きさ）、**引数は以後のモデル入力にも残る**。また**`--dump-config`の受理は実行時に経路が開くことを意味しない**。
3. **Process生成はmodeで止まらない**。子Process（H6-E2b）も候補の子起動（H6-E5）も成立した。止まるのは書込みと言語モード。
4. **利用者が指定しない入力が入る**。AGENTS.md・CLAUDE.md・Skill `description`（H6-E3）。Skill本文は載らず、一覧提示と本文Loadは別。
5. **設定の表示と実効制限が一致しない**。`DSH_PERMISSION_MODE`は空のまま（H6-E0・非対称）。モード名を権限判断の根拠にできない。
6. **送出の停止は「安全」と同義ではない**。`maxBytes`を小さくするとLogの送出が止まるが、作業は続く（分割と停滞）。止まる境界は**1byte単位で特定でき**（307で送られ306で送られない）、止めた分は上限を戻した再実行で**1要求のburstとして戻る**（停滞後の再開・上限を戻した回復）。**単一eventが上限を超えて拡張が丸ごと消えた場合も同じで**、上限を戻すと**86 eventが1要求で戻った**。**backlogが243 event・375,623 bytesでも1要求に収まり、分割されなかった**（backlogのburst）。恒久的な抑止として扱えない。**単一eventが上限を超えると拡張が丸ごと消え、`delivery-accepted`も増えない**（上限を超える単一event）ため、上限は**合計だけでなく1件の大きさ**としても効く。さらに、**止まったことはstdout・stderr・永続Logのどこにも現れない**（停滞の可視性）ため、「送出していない」と「設定が効いている」を出力だけから区別できない。非停滞と同条件で比べると、差は永続Logの`delivery-accepted`の有無だけで、**警告ではなく成功記録の不在としてしか分からない**（停滞の検知）。

## 4. まだ根拠が無いこと

- **別Identity（AppContainer・別User・別Session・mount名前空間）での読取り拒否**。同一Userに対するOSのACLによる拒否は[H6-E1 OSのACL](./DSH_H6E1_OS_ACL_2026-11-09/README.md)で確認したが、Harnessや子Processが**別の資格**で動く場合に同じ拒否が効くかは未確認。Network境界の実拒否も同様。
- **実Providerの保存・保持・二次利用**。
- **MCPの1MB超の引数**と**内容の検査**（大きさは[MCPの引数の大きさ](./DSH_MCP_ARG_SIZE_2026-11-10/README.md)で1MBまで確認済みだが上限の有無は未確認）、**stdio transport・認証・複数サーバー**の相互作用、**Telemetryでfeedbackを与えた場合の送出内容**、**web・tui面のfeedback経路**、**`maxDepth`を`4`以上にした場合**、**`maxActiveSubagents`を時間を分けて起動した場合**、**`maxDepth`と`maxActiveSubagents`の同時作用**、**Turn 1が複数ある場合のfork連鎖の増加量**、**実行中Turnのfork継承**。既定の同時数が8体であることは[同時起動の上限](./DSH_SUBAGENT_CONCURRENT_LIMIT_2026-11-09/README.md)で確認したが、上限値を変えた場合と、順に起動した場合は未確認。Turn境界をまたぐfork連鎖が親の完了Turnを深さ3まで運ぶことは[Turn境界をまたぐfork連鎖](./DSH_SUBAGENT_FORK_TURN_CHAIN_2026-11-09/README.md)で確認したが、履歴が長い場合に各段がどこまで増えるかは未確認。継承しない`subagent`の連鎖が既定で深さ1に止まることは[Subagentの連鎖](./DSH_SUBAGENT_CHAIN_2026-11-09/README.md)で、`maxDepth 3`で深さ3まで伸びることは[深さ上限を上げた連鎖](./DSH_SUBAGENT_DEPTH_RAISED_2026-11-09/README.md)で確認したが、**既定との正確なbyte比較は発火方針が違い未実施**。`subagent`は[Subagent経路](./DSH_SUBAGENT_EGRESS_2026-11-09/README.md)、`subagent_fork`は[Subagent fork](./DSH_SUBAGENT_FORK_2026-11-09/README.md)で確認した。**Telemetryについては以前の記述を訂正した**：`session-telemetry-otel`は`dsh-base`が既定で登録しており、この版が読むのは`DSH_TELEMETRY_OTLP_URL`（標準の`OTEL_EXPORTER_OTLP_ENDPOINT`ではない）。宛先をローカルへ向けても**`FEEDBACK_ONLY`のgateにより送出は0件**で、headlessにはfeedbackの面が無い（[Telemetry経路](./DSH_OTEL_PATH_2026-11-09/README.md)）。
- **停滞の検知手段を用意した場合の実効性**。3面に警告が出ないことと、非停滞との差が`delivery-accepted`の不在だけであることは[停滞の可視性](./DSH_SESSION_LOG_STALL_VISIBILITY_2026-11-09/README.md)・[停滞の検知](./DSH_SESSION_LOG_STALL_DETECTION_2026-11-09/README.md)で確認したが、Telemetry・UI・別flagを含む**他の面**、**243 event・375KBを超えるbacklog**（この範囲では分割されなかったが上限の有無は未確認）、**他版で同じevent名・意味になるか**（この版の観測であって契約ではない）は未確認。単一eventが上限を超えると拡張が消え`delivery-accepted`が増えないことは[上限を超える単一event](./DSH_SESSION_LOG_OVERSIZE_EVENT_2026-11-09/README.md)で確認した。
- **Skillの増分価値**。四Sessionは未採点で、Task path不一致とSession再利用を含む。
- **独立Expectedによる採点**（RunbookのReviewer-only）。

## 5. この台帳が示さないこと

- 各記録の合否、`#2515`の受入条件の達成、`#2509`のH2/H3判定、Owner受理、#1119のGate。
- 記録に無い経路の安全性。「一覧に無いから存在しない」とは扱わない。
- 実装の修正。本台帳は**観測の索引**であり、DSHやTEIのコードを変更しない。

## 6. 更新の規則

- 新しい記録を足したら、**この表の行を1つ足す**。表の列（確定／未確定）を埋められない記録は、まだ台帳へ載せない。
- 記録の**主張を変える訂正**をしたら、この表の該当行も同じ変更で直す（前例: [maxBytesの訂正](./DSH_SESSION_LOG_MAXBYTES_2026-11-08/README.md)）。
- 横断の結論（§3）は、**2件以上の記録が一致したときだけ**足す。1件の観測を横断の結論に昇格させない。
- この台帳は**索引であって正本ではない**。数値・条件は各記録を参照する。
