# Desktop Studio Host：外部編集競合とDraft復旧 Reference（#1529）

本ディレクトリは **Host固有のWorkspace状態・競合検知・再起動後の未受理状態復旧を読取り専用で検査する小さな参照** であり、TEIの正式なSource Authority、Application Service、Plugin Resolver、第二の受理Controller、受入済みAPIではありません。実装主体は `hostdraftrecovery.go`、受入は `hostdraftrecovery_test.go`、起動点は `cmd/tei-host-draft-recovery/` です。`正本`を書き換える経路はありません。

設計の入手先は [DESKTOP_STUDIO_HOST_INTEGRATION_DESIGN](../../plan/design/DESKTOP_STUDIO_HOST_INTEGRATION_DESIGN.md)。仕様の採用と実装を区別します。

## 区別する対象

| 対象 | 扱い |
| --- | --- |
| `modelRevision`（受理済みの版） | 版だけが一致しても受理根拠にしない |
| Sourceバイトfingerprint（`sha256:<hex>`） | バイト列から計算し、ラベルを信頼しない |
| 未受理Draft／Candidate | 表示・保存・再読込のみ。明示Reviewが終わるまで Accepted へ昇格しない |
| 外部副作用（Run／Observation） | 完了が不明なら既存の記録へ照会。勝手に再送しない |

## 判定する意味（採用前）

| 条件 | 期待 | 結果コード |
| --- | --- | --- |
| 明示Review後の新鮮なDraft | 受理入口へ移譲できる | `ACCEPT_ALLOWED` |
| `modelRevision` 不一致 | fail closed | `REVISION_STALE` |
| revision一致・バイト不一致（外部編集後の旧Preview） | 受理を拒否 | `SOURCE_BYTES_MISMATCH` |
| 保存時再検査なし | Watcher欠落・重複・順序逆転に依存しない保護 | `WRITE_RECHECK_MISSING` |
| 別プロセス・別ウィンドウが編集中で再検査なし | 書込み直前の拒否 | `CONCURRENT_WRITER_REJECTED` |
| 再起動・切断後で明示Review未了 | 昇格しない。Accepted Source を読み直す | `DRAFT_NOT_PROMOTED` |
| 副作用の完了が不明でObservationなし | 再送しない | `SIDE_EFFECT_UNKNOWN_NO_RESEND` |
| 副作用の記録が既存Observationにある | 再送ではなく照合 | `VERIFY_FROM_OBSERVATION` |
| 受理を伴わないDraft表示 | 表示・保存・再読込のみ許可 | `DRAFT_DISPLAY_ONLY` |

Reject／Preview では受理済み Source のバイトと revision を変更しません。本参照はどの経路にも書込みません。

## 合成Fixture（受入条件との対応）

| 操作 | 本参照の結果 |
| --- | --- |
| read → Draft → Preview → Accept → 再読込 | `ACCEPT_ALLOWED`（既存共通Operationと同じ意味、書込みは別経路） |
| revision同一・外部編集でバイト不一致の旧Preview受理 | `SOURCE_BYTES_MISMATCH`、Watcher通知の有無に依存しない |
| ファイル監視の欠落・重複・順序逆転 | 保存時の再検査が無ければ `WRITE_RECHECK_MISSING` |
| WebView再起動を模した切断・再接続後のDraft | `DRAFT_NOT_PROMOTED`（明示Review完了で`ACCEPT_ALLOWED`） |
| Host途中終了で副作用完了不明 | `SIDE_EFFECT_UNKNOWN_NO_RESEND`／記録があれば `VERIFY_FROM_OBSERVATION` |
| 多重ウィンドウ・別プロセス | `CONCURRENT_WRITER_REJECTED`（書込み直前の再検査で初めて解消） |

シンボリックリンク／パス差替え・Workspace外書込みについて、Hostの実際のResource Enforcementでの拒否は**合成だけでは未実証**です。

## 実行方法

```sh
go run ./tools/reference-host-draft-recovery/cmd/tei-host-draft-recovery -input request.json
```

`-input -`（既定）は標準入力から読みます。入力例（外部編集後の旧Preview受理）：

```json
{
  "workspace": { "workspaceId": "ws-repair", "revision": "r7", "sourceBytes": "externally edited bytes" },
  "draft": { "baseRevision": "r7", "baseSourceFingerprint": "sha256:f9e6d6c9a4a522e63bb23e517f106ec6035a2d1d1a4a3d1e1c4b6a1b5d3c2e1a0" },
  "wantAccept": true,
  "writeRecheck": true
}
```

## 責務と分界

- 型付きOperation契約は #1158、RelationのPreview／Acceptと三視点Projectionは #1124、Trace／Sourceの開示制御は #1178、Desktop Target Evidenceは #259 が所有します。
- 受理入口・受理Controllerの第二実装、Source Authority・Application Service・Plugin Resolverの再作成をしません。
- 実Wails／実WebViewの起動・OS別配布は別Gateであり、本参照の合成Host PASSをその実証として扱いません。

## 限界

- 合成入力に対する静的な判定です。実Host・実Watcher・実ファイルシステムでの競合測定は未実施です。
- 書込み直前の排他制御は Host の実際の Resource Enforcement に依存します。本参照はそのための実装を持ちません。
- `-race` は cgo を伴わない可搬Goでは実行していません。
- Crash後のDraft復元で「保存できない機微情報の非永続化」の実装は本参照の外であり、未実証として残します。
