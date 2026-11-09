# DSH外部Harness実測の台帳（2026-11-09、#2509 / #2515）

- 対象Issue: [#2509](https://github.com/hat47x/tei-platform/issues/2509)（DSH外部Harness実証）、[#2515](https://github.com/hat47x/tei-platform/issues/2515)（実効隔離・全送出経路Conformance）。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)、比較計画は[TAI #168](https://github.com/hat47x/tai-pleroma/issues/168)。
- 状態: **既存の実行記録を並べた索引**。新しい実測は行っていない。個々の合否・Owner受理は各記録とIssueの担当。
- この文書の役割: 24件の実行記録が**それぞれ何を確定させ、何を確定させていないか**を1枚で見えるようにし、同じ測り方の重複を避ける。記録本文の複製はせず、各記録へ参照で結ぶ。

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
2. **読んだ内容は複数の経路で外へ出る**。モデル入力・Session Log・stdout・永続ログ（H6-E6）。Session Logの複製はTool結果1件につき2箇所（`data.message`と`data.meta`）。**Subagentを起動すると子も別Sessionとして同じ拡張を送り、独立した永続Logを持つ**（Subagent経路）ため、送出量は親Sessionの分だけでは数えられない。さらに**`subagent_fork`は親の完了Turnを子へ運ぶ**（Subagent fork）ため、親が読んだ内容が子のモデル入力にも載る。**継承しない`subagent`と継承する`subagent_fork`は同列に数えられない**。
3. **Process生成はmodeで止まらない**。子Process（H6-E2b）も候補の子起動（H6-E5）も成立した。止まるのは書込みと言語モード。
4. **利用者が指定しない入力が入る**。AGENTS.md・CLAUDE.md・Skill `description`（H6-E3）。Skill本文は載らず、一覧提示と本文Loadは別。
5. **設定の表示と実効制限が一致しない**。`DSH_PERMISSION_MODE`は空のまま（H6-E0・非対称）。モード名を権限判断の根拠にできない。
6. **送出の停止は「安全」と同義ではない**。`maxBytes`を小さくするとLogの送出が止まるが、作業は続く（分割と停滞）。止まる境界は**1byte単位で特定でき**（307で送られ306で送られない）、止めた分は上限を戻した再実行で**1要求のburstとして戻る**（停滞後の再開）。恒久的な抑止として扱えない。さらに、**止まったことはstdout・stderr・永続Logのどこにも現れない**（停滞の可視性）ため、「送出していない」と「設定が効いている」を出力だけから区別できない。非停滞と同条件で比べると、差は永続Logの`delivery-accepted`の有無だけで、**警告ではなく成功記録の不在としてしか分からない**（停滞の検知）。

## 4. まだ根拠が無いこと

- **別Identity（AppContainer・別User・別Session・mount名前空間）での読取り拒否**。同一Userに対するOSのACLによる拒否は[H6-E1 OSのACL](./DSH_H6E1_OS_ACL_2026-11-09/README.md)で確認したが、Harnessや子Processが**別の資格**で動く場合に同じ拒否が効くかは未確認。Network境界の実拒否も同様。
- **実Providerの保存・保持・二次利用**。
- **Telemetry（OTel）・MCP経路の網羅**、**子からの連鎖起動**、**実行中Turnのfork継承**。`subagent`は[Subagent経路](./DSH_SUBAGENT_EGRESS_2026-11-09/README.md)、`subagent_fork`は[Subagent fork](./DSH_SUBAGENT_FORK_2026-11-09/README.md)で確認したが、headlessプロファイルには`dsh-session-telemetry-otel`が載っておらず、`OTEL_EXPORTER_OTLP_ENDPOINT`を設定した実行でもテレメトリの送出は観測されなかった（**未観測は「無い」ではない**）。
- **停滞の検知手段を用意した場合の実効性**。3面に警告が出ないことと、非停滞との差が`delivery-accepted`の不在だけであることは[停滞の可視性](./DSH_SESSION_LOG_STALL_VISIBILITY_2026-11-09/README.md)・[停滞の検知](./DSH_SESSION_LOG_STALL_DETECTION_2026-11-09/README.md)で確認したが、Telemetry・UI・別flagを含む**他の面**、単一eventが上限を超えた場合の`delivery-accepted`の振る舞い、**他版で同じevent名・意味になるか**（この版の観測であって契約ではない）は未確認。
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
