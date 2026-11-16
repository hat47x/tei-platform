# X-TEA の一括更新（XF110）の部分成功と、TEI の `partial_success` の接続（2026-11-17）

## 対象

- 知識目録 **F-14**（一括更新の実行順）と **F-13**（値パネル）
- `internal/failure/failure.go` の `PartialSuccess`／`PartialError`
- `internal/execution/effects.go` の `compensationFailure`（実際の生成箇所）
- `internal/command/invoke.go` の `ReportedFailure` と `ExecutionError`
- 設計文書 `XTEA_LOWCODE_RECEPTION_ARCHITECTURE_2026-11.md` §XF110 と「未決定」の該当行

## 見つけたこと

### X-TEA 側の観察（F-14）

一括更新は「行をバッチキーで並べ替え → キー区切りでバッチ表へ制御レコードを挿入 → 明細を1行ずつ更新 → バッチ表の`AC`スクリプトとバッチレコードごとの機能を実行」という順で進む。知識目録は次の2点を弱点として記録している。

1. **途中の失敗で全体をロールバックするが、先に実行した外部機能の効果は戻せない。**
2. **部分成功を表す結果がない。**

つまり X-TEA は「DBは戻したが外部呼出しは戻っていない」状態を、**成功でも失敗でもない何か**として表す手段を持っていなかった。

### TEI 側の現状

`internal/failure` は `PartialSuccess` という分類を持ち、`PartialError` 型もある。**実際に生成されている箇所が1つある**。

`internal/execution/effects.go` の `compensationFailure` がそれで、補償（compensating action）の記録を見て「一部だけ適用された」場合に `COMPENSATION_PARTIAL` を返す。

```go
if applied == 0 || applied == len(records) {
    return cause
}
return &failure.PartialError{
    Code:    "COMPENSATION_PARTIAL",
    Message: fmt.Sprintf("%d of %d compensations were applied", applied, len(records)),
    Cause:   cause,
}
```

**コメントが示す設計意図が重要**である。「エラーを数えるのではなく記録を読む。`RunEffect` は失敗した補償を `Succeeded=false` かつエラー無しの記録として返すため、エラーを数えると、まさに適用されなかった補償を見落とし、半分適用された実行を clean と報告してしまう」。

これは **X-TEA の弱点2（部分成功を表す結果がない）に対する、TEI 側の答え**になっている。

### しかし接続されていない

`internal/command` 側は `partial_success` を特別扱いしていない。

- `ReportedFailure` は `EffectMayBeUnknown` を先に見て `COMMAND_EFFECT_UNKNOWN` を返し、次に `ExecutionError.Code` をそのまま返す。**`partial_success` という分類は出口に出てこない。**
- `ExecutionError` は `EffectUnknown bool` を持つが、「一部だけ適用された」を表す欄は無い。
- 既存の試験 `TestExecutionErrorPartialSuccess` は名前と裏腹に、`EffectUnknown: true` を立てた `ExecutionError` が `TechnicalFailure` / `LookupFirst` になることを確かめている。**`partial_success` を確かめてはいない。**
- **裸の `PartialError` は、出口でコードを1つも返さない。** `ReportedFailure` は `EffectMayBeUnknown`（`ExecutionError.EffectUnknown` を見る）を先に判定し、次に `ExecutionError` かどうかを見る。`PartialError` はそのどちらでもないため `("", "", false)` になる。試験 `TestReportedFailureDropsAPartialError`（`internal/command`）で実測した。**意味はあるのに、呼出し元には何も届かない。**

## なぜ問題か

X-TEA の一括更新（XF110）を TEI で受けるとき、**「一部だけ適用された」を呼出し元へ伝える経路が無い**。

- 一括更新の途中で外部機能が失敗し、DBは補償で戻ったが外部呼出しは戻らない、という状況は F-14 が明示している。
- TEI には `PartialError` があるのに、Command 境界から出てこない。**意味はあるが、出口が無い**（D-9 の変種）。
- 呼出し元は `COMMAND_EFFECT_UNKNOWN`（効果不明）を受け取るが、これは「適用されたか分からない」であって「**一部は確実に適用された**」ではない。**再送の判断が変わる**（効果不明なら照会してから再送、部分成功なら残りだけ実行する必要がある）。

## 選択肢

### A. Command 境界から `partial_success` を報告する

`ReportedFailure` に分岐を足し、`IsPartialSuccess(err)` のときは `COMMAND_PARTIAL_SUCCESS` を返す。`EffectMayBeUnknown` より**先**に判定するか後にするかを決める必要がある。

- **利点**: 意味が出口に出る。呼出し元が「残りだけ実行」を選べる。
- **欠点**: 順序の判断が要る。「一部適用」と「適用不明」はどちらも「全部は適用されていない」だが、**再送の危険度が違う**（部分成功は既知の適用があり、不明は未知）。
- **コスト**: 小（分岐1つと試験）。

### B. `ExecutionError` に「適用済みの範囲」を持たせる

`EffectUnknown bool` の隣に、適用された範囲（どの効果が適用されたか）を持たせる。

- **利点**: 呼出し元が「何が残っているか」を知れる。
- **欠点**: 一括更新の「範囲」の宣言（まだ設計されていない）と結びつく。今は範囲を表す型が無い。
- **コスト**: 中。

### C. 何もしない（`COMMAND_EFFECT_UNKNOWN` のまま）

- **利点**: 変更ゼロ。安全側（照会してから再送）。
- **欠点**: 「一部は確実に適用された」という**既知の情報を捨てる**。照会の結果、適用済みと分かっても「残りを実行する」判断は呼出し元が独自に行うしかない。

## 既定案: A を**段階的に**（まず意味を出口へ出す）

理由:

1. **B は範囲の宣言が前提**であり、XF110 の「一括の範囲と対象行の宣言」はまだ設計されていない。順序が逆になる。
2. C は既知の情報を捨てる。`PartialError` が実在するのに出口が無いのは、**まさに D-9 の形**である。
3. A は分岐1つで意味を可視化でき、`EffectMayBeUnknown` との**順序だけ**を決めれば済む。

順序については、**`partial_success` を先に判定する**ことを提案する。理由は「効果不明」より情報が多いから（不明は何も分からないが、部分成功は「一部が適用された」ことが分かっている）。ただし呼出し元が照会を省いて再送しないよう、**`retryable` は `lookup_first` のまま**にする。

覆す条件:

- 一括の範囲宣言が設計されたら B を検討する（何が残っているかを伝える方が有用になる）。
- 実運用で「部分成功を受けた呼出し元が、照会せずに再送して二重適用した」事象が観測されたら、A の順序を逆にする（`COMMAND_EFFECT_UNKNOWN` を優先）。

## いま明記しておくこと

- **`partial_success` は意味として存在する**（`internal/failure`、`internal/execution` の補償で実際に生成される）。
- **Command 境界からは出てこない**ため、呼出し元は「一部適用」を知れない。
- 既存試験 `TestExecutionErrorPartialSuccess` は名前と内容が一致していない（`technical_failure` を確かめている）。**この名前は誤解を招く**。

## 読み取れないこと

- X-TEA の一括更新が、外部機能の失敗時に**どこまでを「戻した」とみなしていたか**（`AC` スクリプトの後処理に依存し、宣言が無い）。
- 実 PostgreSQL での `COMPENSATION_PARTIAL` の到達（`effect_real_pg_test.go` に試験があるが、実DB未設定では skip される）。
- 一括更新の「範囲」を意味としてどう宣言するか（`BatchKeyFields` 相当）。

## 位置付け

**設計判断の候補**であり、実装は未了。A を既定案とするが、順序（`partial_success` と `effect_unknown` のどちらを先に報告するか）は呼出し元の再送判断に直結するため、実施時に再確認する。仮承認。
