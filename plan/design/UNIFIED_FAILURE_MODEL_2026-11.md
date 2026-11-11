# 共通の例外・失敗モデル（2026-11）

- **状態：参照実装（A2）／一部適用済み。** 意味の受理・権限の正本・本番配備は扱わない。
- **対象コード**: `internal/failure/`、`internal/command/invoke.go`、`internal/hook/run.go`、`internal/security/effectcommit/commit.go`、`internal/database/sqlgen/failure.go`、`internal/studiohost/readonly.go`、`internal/studiohost/command.go`、`cmd/tei/cli_view.go`、`cmd/tei/cli_inspect.go`、`tools/reference-generator-selection/generator.go`・`templates/TeiFailure.java.tmpl`・`templates/Main.java.tmpl`・`templates/Main-long-running.java.tmpl`・`templates/LineAmounts.java.tmpl`・`templates/template.go.tmpl`
- **関連**: [#3807](https://github.com/hat47x/tei-platform/issues/3807)（境界ごとに違っていた失敗コード）、[#2996](https://github.com/hat47x/tei-platform/issues/2996)、[#2997](https://github.com/hat47x/tei-platform/issues/2997)、[#1174](https://github.com/hat47x/tei-platform/issues/1174)、[#2853](https://github.com/hat47x/tei-platform/pull/2853)、[#2854](https://github.com/hat47x/tei-platform/pull/2854)
- **設計の正本**: 故障分類と回復の方針は[Script起動の失敗状態・再実行・回復設計](SCRIPT_INVOCATION_FAILURE_AND_RECOVERY_DESIGN_2026-11.md) §3（F01〜F14）、Command境界の扱いは[Command境界の実行前検査](COMMAND_BOUNDARY_SAFETY_2026-11.md)を正本とする。本書は**Goの型として実装した共通契約**だけを扱い、方針を複製しない。

## 1. なぜ必要か

同じ失敗が、入口ごとに違う形で外へ出ていた。

| 入口 | 同じ失敗の告げ方 |
| --- | --- |
| CLI `view -execute` | `ExecutionError`の`Code`（21種類） |
| CLI `inspect` 等 | `cliInspectionFailure`（`Code`のみ） |
| Studio bridge | `studiohost.Failure`（`Code`/`Stage`/`Message`） |
| HTTP/IPC Adapter | `interfaceadapter.Failure`（`Code`/`Message`） |
| Hook境界 | `hook.Error`（`Code`のみ） |

呼出し側は「これは業務の拒否か、こちらの故障か」「もう一度送ってよいか」「副作用が残ったかもしれないか」を、**文字列コードの一覧を各自で持つ**ことで判断していた。一覧は入口ごとにずれ、#3807では21コードのうち15が`COMMAND_FAILED`へ潰れていた。さらに`ExecutionError`は`EffectUnknown`を持っていたが、`hook.Error`は同等の印を持たず、畳む側が落としていた。

業務拒否を技術障害として報告すると、呼出し側は「いつか治る」と判断して再送し、**重複した副作用**を作る。逆に、技術障害を業務拒否として報告すると、効果が残っているのに「要求が通らなかっただけ」と処理して**不正な成功状態**を残す。どちらも型で防ぎたかった。

## 2. 判断したこと

### 2.1 三つの軸を一つの契約にする

`internal/failure.Failure`は`error`を満たし、さらに3つの軸を持つ。

| 軸 | 型 | 値 | 意味 |
| --- | --- | --- | --- |
| 分類 | `Kind` | `business_rejection` | 要求は妥当だが、規則・認可・ポリシーで拒否された。部分効果は期待しない |
| | | `technical_failure` |  infra・接続・実行側の障害。効果が適用されたか分からない |
| | | `partial_success` | 一部だけ成立した。効果が適用された可能性がある |
| 再試行 | `RetryPolicy` | `retryable` | 同じ操作identityのまま送り直せる |
| | | `not_retryable` | 要求を変えずに再送してはならない |
| | | `lookup_first` | **先に同じ操作identityで結果を照会**してから再送を決める |
| 効果不明 | `HasEffectUnknown()` | `bool` | 失敗後に副作用が残った可能性がある |

加えて`FailureCode()`（機械可読な理由）、`FailureMessage()`（利用者向け）、`FailureCause()`（開発者向けの原因）を持つ。発生箇所は`Code`と、応答に残る`Stage`で表現する。

**再試行の3値が要点である。**「リトライ可/不可」の2値では、commitのACKが失われた失敗を安全に扱えない。同じIDで送れば二重適用、別のIDで送れば重複——どちらも事故である。`lookup_first`は「**再送より照会が先**」を型で強制し、[Script設計 §3 のF07/F10](SCRIPT_INVOCATION_FAILURE_AND_RECOVERY_DESIGN_2026-11.md)と[#2854の別Key再適用という反例](https://github.com/hat47x/tei-platform/pull/2854)に対応する。

### 2.2 既存の型を壊さず、後方互換に統合する

新しいラッパー型を増やすと、`errors.As(err, &command.ExecutionError{})`を使う既存の呼出し側がすべて壊れる。そのため**既存の型にメソッドを追加**した。

| 型 | 追加したもの |
| --- | --- |
| `command.ExecutionError` | `FailureCode/Message/Cause`、`Kind()`、`Retryable()`、`HasEffectUnknown()` |
| `hook.Error` | 同上 |
| `effectcommit.CommitUncertainError` | 同上 |
| `interfaceadapter.Failure` | 同上 |
| `database/sqlgen.DatabaseFailure` | 同上。加えて`Unwrap()`で原因を標準ライブラリに曝す |

フィールド（`Code`・`Message`・`Cause`・`EffectUnknown`）は変更していない。`var _ failure.Failure = ...`で各型の適合をコンパイル時に固定する。

### 2.3 分類はコードの閉集合から決める

分類を自由文字列にすると、入口ごとに違う分類が生まれる。`Kind()`は**コードの閉集合に対するswitch**にする。未知のコードは`technical_failure`／`not_retryable`に倒す——「分からない」を安全側（再送しない）へ寄せる。

| Command | 分類 | 再試行 |
| --- | --- | --- |
| `INVALID_COMMAND_INPUT`、`COMMAND_NOT_DECLARED`、`COMMAND_UNAUTHORIZED`、`COMMAND_AUTHORIZER_UNAVAILABLE`、`COMMAND_PERIOD_CLOSED`、`COMMAND_BUSINESS_DENIED`、`COMMAND_RESOURCE_CONFLICT` | business_rejection | not_retryable |
| `COMMAND_TRANSACTION_BEGIN_FAILED`、`COMMAND_ROLLBACK_FAILED`、`COMMAND_COMMIT_FAILED`、`COMMAND_EXECUTION_FAILED`、`COMMAND_RECORD_FAILED`、`COMMAND_PERIOD_*_UNAVAILABLE`、`UNDECLARED_COMMAND_EFFECT`、`UNDECLARED_COMMAND_OUTCOME`、`TIMEOUT` | technical_failure | retryable |
| 上記の技術障害で`EffectUnknown` | technical_failure | **lookup_first** |
| `COMMAND_EXECUTOR_UNAVAILABLE` | technical_failure | retryable |

`EffectUnknown`が立つ技術障害はすべて`lookup_first`へ昇格する。これは`Retryable()`の中で`base == retryable && e.EffectUnknown`として実装し、個別のコードを列挙しなくて済むようにした。

`COMMAND_COMMIT_FAILED`は常に`EffectUnknown`を立てるので、常に`lookup_first`になる。commitの成否が確定できない失敗を、そのまま再送させないためである。

### 2.4 畳む側は分類を落とさない

応答の形（`cliInspectionFailure`・`studiohost.Failure`）に`Kind`・`Retryable`・`EffectUnknown`を追加し、**元のerrorが契約を持つならそれを写す**。写す元がない場合の既定は次のとおり。

| 応答の作り | 分類 | 再試行 | 効果不明 |
| --- | --- | --- | --- |
| 要求自体の拒否（認可なし・版不一致・対象が解決しない） | business_rejection | not_retryable | false |
| Host側の障害（読込失敗・index構成失敗・未分類のerror） | technical_failure | lookup_first | true |

未分類のerrorはtechnical側に倒す。未知の失敗を業務拒否として報告すると、呼出し側が諦めてしまうためである。

`cmd/tei/cli_view.go`の`failWithFailure`と`internal/studiohost/readonly.go`の`commandFailure`がこの写像を行う。`studiohost`は`command.EffectMayBeUnknown`でも効果不明を拾うので、契約を持たないerrorからも落ちない。

## 3. 具体例

commitが失敗したCommandを、CLIとStudioがどう告げるか。

```go
// internal/command/invoke.go
if err := transaction.Commit(ctx); err != nil {
    _ = transaction.Rollback(ctx)
    return Result{}, wrapTimeout(&ExecutionError{
        Code:          "COMMAND_COMMIT_FAILED",
        Message:       err.Error(),
        Cause:         err,
        EffectUnknown: true,
    })
}
```

```json
// tei view -command sales.order.submit -execute -format json
{
  "ok": false,
  "error": {
    "code": "COMMAND_COMMIT_FAILED",
    "message": "commit failed",
    "kind": "technical_failure",
    "retryable": "lookup_first",
    "effectUnknown": true
  }
}
```

呼出し側は`retryable`を見て、同じ操作identityで照会へ回る。`kind`を見て、これは業務の拒否ではないので要求を修正しても無駄と判断する。

## 4. 検証したこと

| 試験 | 確認内容 |
| --- | --- |
| `internal/failure/failure_test.go` | `AsFailure`の解決、分類・再試行・効果不明の各述語 |
| `internal/command/execution_failure_test.go` | `ExecutionError`の契約適合、業務拒否（期間クローズ・業務拒否・資源競合）、技術障害（commit失敗・実行失敗）、包んでも残ること |
| `internal/hook/hook_failure_test.go` | `hook.Error`の適合、スコープ拒否＝業務拒否、実装障害＝技術障害 |
| `internal/security/effectcommit/commit_failure_test.go` | `CommitUncertainError`が`lookup_first`＋効果不明を保つこと、包んでも失われないこと |
| `internal/studiohost/command_failure_classification_test.go` | 入力不正・認可拒否が`business_rejection`/`not_retryable`で出ること、commit失敗が`technical_failure`/`lookup_first`/効果不明で出ること、読込失敗の既定 |
| `cmd/tei-studio/bridge_failure_axes_test.go` | Wailsブリッジが三軸を落とさないこと（Go構造体とJSONの両方） |
| `tools/reference-generator-selection`（Java） | 生成したJavaが4軸を持ち、業務拒否と技術障害が違う値を出すこと。`javac`でコンパイルし実行して確認 |
| `reference/interface-adapter` | トランスポート拒否（`UNAUTHENTICATED`・`FORBIDDEN`・`STALE_REVISION`）＝業務拒否 |
| `cmd/tei/cli_view_failure_test.go` | `commandFailure`の既存の契約（未知のerrorは変換しない） |

```
ok  tei/internal/failure                1.981s
ok  tei/internal/command                3.441s
ok  tei/internal/hook                   1.996s
ok  tei/internal/security/effectcommit  3.505s
ok  tei/internal/studiohost             9.854s
ok  tei/reference/interface-adapter     5.856s
ok  tei/cmd/tei-studio                  6.980s
ok  tei/tools/reference-generator-selection (TestJava* / TestGoAnd*)  139.931s
```

## 5. 適用済みの範囲と、残っている範囲

### 適用済み

Go実行基盤の主要な境界——Command・Hook・EffectCommit・Studio bridge・CLI・参照Adapter。加えてJava生成ターゲット（`Main`・`Main-long-running`）と、Effect補償（`execution.Compensate`）。

#### `partial_success` を返す境界

`execution.Compensate` は、失敗した効果に対する補償を宣言順に全て試行します。1件が適用され別の1件が適用できなかった場合、従来は error も nil も返さない（補償の失敗は record の `Succeeded=false` として残るだけだった）ため、呼出し側は「全部やり直す」か「全部諦める」かの二択になっていました。どちらも事故です（前者は成立した補償の重複、後は成立した補償の喪失）。

`failure.PartialError` を追加し、この状態を独立した分類にしました。`Retryable()` は常に `lookup_first`、`HasEffectUnknown()` は常に true です——何が適用されたかは error からは分からないので、呼出し側はまず record を読む必要があります。

判定は error の数ではなく **record の `Succeeded` を数える**ことで行います。`RunEffect` は invocation が失敗した補償を「error なし・`Succeeded=false` の record」として返すため、error を数えるちょうど「適用できなかった補償」が見えなくなります。

補償が1件しかない場合は分割できないので、従来どおり error はそのまま（失敗は record、journal 失敗は素の cause）です。

### 未適用

| 範囲 | 状態 |
| --- | --- |
| Data Provider / Operator 層 | 未着手。`internal/database/sqlgen`が`main`に無いため、このPRの範囲外。SQLSTATE/result codeでの分類は別ブランチ（`feat/sql-to-standard-query-conversion-20261111`）に実装済みで、sql-dialect作業が`main`へ入ってから持っていく |
| `LineAmounts.java.tmpl`（java-library-template） | `main`にテンプレートが無いため範囲外。業務/技術の区別はあるが再試行・効果不明の軸が無い |
| Web Host（Wails）の実ブラウザー確認 | ブリッジが三軸を落とさないことはGo試験で確認済み。実WebView2での表示は未確認 |
| 他の部分成功 | RunPlan の「後の Step が失敗しても前の Step の効果は残る」ケース（[Script設計のF09](SCRIPT_INVOCATION_FAILURE_AND_RECOVERY_DESIGN_2026-11.md)）は未実装。RunPlan には製品の呼出し元がなく、API の形は設計判断 |

### 生成ターゲットへの展開方針

`Kind`・`RetryPolicy`・`HasEffectUnknown`は**文字列の閉集合**としてJSONに出るので、言語を選ばない。Java側は検査例外の階層を作らず、生成コードが`FailureCode`/`Kind`/`Retryable`/`EffectUnknown`をフィールドに持ち、言語固有の例外は1つのAdapterでこの4つへ変換する。Goの`failure.Failure`インターフェースは「変換後の照合票」として使い、各言語が同じ照合票を実装する形にする。

## 6. 混同を防ぐための不退転条件

1. **業務拒否に`lookup_first`を付けない。**業務拒否は効果がないので、照会する理由がない。
2. **効果不明のない技術障害だけが`retryable`。**効果不明なら`lookup_first`。
3. **未知のコードは`technical_failure`/`not_retryable`。**安全側に倒す。
4. **畳む側は契約を持つerrorから分類を写す。写す元がないときだけ既定を使う。**
5. **応答に`kind`を出さない口を作らない。**呼出し側がコード一覧を持たなくて済うようにするため。

## 7. 既存動作への影響

- `ExecutionError`・`hook.Error`等の**フィールドと既存メソッドは不変**。`errors.As`による既存の参照はそのまま動く。
- JSON応答には`kind`・`retryable`・`effectUnknown`が**追加**される。いずれも`omitempty`なので、契約を持たないerrorでは従来と同じJSONになる。
- `cli_view.go`の`commandFailure`は戻り値を変えていない（#3807の既存試験がそのまま通る）。
- `studiohost.Failure`は`MarshalJSON`を持つ。未分類のままだと三軸を補うため、**同じJSONを出すがフィールドが増える**。明示値を優先するので、境界がerrorから写した分類は上書きされない。
- `cmd/tei`は`three-element`のallowlist不在理由をコメントで記録した（コードの挙動は変わらない）。
- 生成Javaは`{"state":"error","code":...}`に4軸が増える。Goテンプレートも同じ4軸を出す。
- `execution.Compensate` は「補償の1件が適用され別の1件が適用できなかった」ときに error を返すようになる（従来は nil）。補償が1件だけの場合は従来どおり nil のままで、`TestCompensateRecordsItsOwnFailure` がその契約を保ちます。
