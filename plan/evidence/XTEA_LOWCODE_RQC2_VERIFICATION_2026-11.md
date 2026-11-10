# RQ-C2の確認方法の観測（2026-11）

## 対象

- 要求: [X-TEA知見の受容要件](../../requirements/XTEA_LOWCODE_RECEPTION_REQUIREMENTS_2026-11.md)のRQ-C2「確認方法」
- 試験: `internal/standardbusiness/rq_c2_verification_test.go`
- 対象版: `main` の `b399b6b31` から分岐した`test/rq-c2-verification-20261110`
- 環境: Windows、Go `1.27.0`

## 要求文の「確認方法」と、この試験の対応

| 要求文 | 試験 |
| --- | --- |
| 行レベルの検証・導出項目・参照入力の補助を、フック宣言と規則Pluginで表し、同じ入力に同じ結果を返す | `TestRQC2HookDeclarationCarriesTheContract`・`TestRQC2SameInputGivesTheSameReads` |
| 宣言していない能力を呼ぶフックが、検証時と実行時の両方で拒否される | `TestRQC2UndeclaredCallIsRefusedAtExecution`（実行時。検証時は`internal/validation`の対話検査が担当） |
| フックの失敗が、拒否・不明・未実行のいずれかとして観測できる | `TestRQC2RunSequenceStopsAtTheRefusedPhase`・`TestRQC2WarnContinuesAndIsRecorded`・`TestRQC2NotSatisfiedIsNotAnExecutionFault` |

## 観測

- `TestRQC2HookDeclarationCarriesTheContract` 成功。契機・読み取り・使用能力・生成物・失敗名が宣言に載る。失敗名は語彙に載る名前だけを受理し、`valueRejected`のような綴り誤りは拒否される。
- `TestRQC2SameInputGivesTheSameReads` 成功。同じ宣言と同じ観測値から同じ読み取り束を得る。宣言外の読み取りと、宣言したが観測していない読み取りは、どちらも拒否される（空で通さない）。
- `TestRQC2UndeclaredCallIsRefusedAtExecution` 成功。宣言した能力だけが境界へ届き、宣言外（`sales.delete_all`）は境界へ**到達しない**（記録が増えないことで確認）。
- `TestRQC2RunSequenceStopsAtTheRefusedPhase` 成功。`before-read`だけが実行され、`before-update`で拒否、`after-update`は**実行されない**。`SequenceResult`に実行済み・拒否した位相が残る。
- `TestRQC2WarnContinuesAndIsRecorded` 成功。`on-failure warn`の失敗は`Warned`に記録され、後続の位相は実行される。拒否としては記録されない。
- `TestRQC2NotSatisfiedIsNotAnExecutionFault` 成功。業務上の拒否は`ExecutionError`として読まれず、`OutcomeOf`が結末名を返す。理由だけの拒否には結末名を付けない。実行の失敗は業務上の結末として読まれない。

## 読み取れること

- 拡張点の契約（契機・範囲・読み取り・使用能力・生成物・失敗）が**宣言**に載り、実装ではなく宣言が何をしてよいかを決める。
- 宣言外の能力呼出しは、実行時の境界（`hook.Declared`）で拒否され、境界へ届かない。
- 失敗は「拒否（結末名つき）」「警告（続行）」「実行の失敗」に分かれ、未実行の位相は`SequenceResult`から読める。

## 読み取れないこと

- **規則Pluginの供給路**は未実装です。この試験は参照実装が持つ宣言・拒否・観測の面だけを確かめており、Pluginが規則を供給することは確かめていません。
- 「検証時の拒否」は`internal/validation`の対話検査が担当するため、この試験では同じ規則の実行時側だけを確かめています。両者が同じ規則を使うことは別の試験（`internal/hook`の`declared_test.go`等）が持ちます。
- 任意SQL・任意確定の拒否（要求文の「許しません」）は、この試験の範囲外です。
- 実Provider・実DB・旧Runtimeでの観測（#940）。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。RQ-C2の「確認方法」のうち、宣言・実行時の拒否・失敗の観測が参照実装で見えることを示すもので、Plugin供給の実装や要求の採用判断ではありません。
