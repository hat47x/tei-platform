# 閉塞拒否と訂正能力の矛盾の宣言段階での報告の確認（2026-11）

## 対象

- 実装: `internal/validation/period_closure_conflict.go`、`internal/validation/validation.go`の呼出し
- 試験: `internal/validation/period_closure_conflict_test.go`
- 対象版: `main` の `d0f4f5f0b` から分岐した`feat/closed-rejects-correction-conflict-20261111`
- 環境: Windows、Go `1.27.0`

## 前ラウンドの残し

前ラウンドで、`closed-rejects update`と`correction via <capability>`（効果が`update`）の矛盾を**実行時の免除**で解きました。ただし矛盾は**ソースに残ったまま**で、検証は何も言いませんでした。「解けるから黙って通す」状態です。

## 変更

`PERIOD_CORRECTION_REFUSED_BY_CLOSURE`を追加しました。期間が閉塞で拒否する操作の種類が、訂正能力の効果の種類と一致する場合に報告します。

| 場合 | 扱い |
| --- | --- |
| 拒否が`update`・訂正能力の効果が`update` | **報告** |
| 拒否が`update`・訂正能力の効果が`create` | 報告しない（衝突しない） |
| 訂正能力が効果を宣言していない | 報告しない |
| 訂正能力を宣言していない | 報告しない |

## 設計上の判断

**実行時の免除は残します。** 免除は「既に書かれた宣言を安全に解く」役割で、検証は「その宣言が矛盾していることを著者に伝える」役割です。片方を消すと、前者は矛盾を含むソースが本番で詰まる原因になり、後者は矛盾が誰にも見えない状態になります。

効果を宣言していない訂正能力を報告しないのは前ラウンドと同じ理由です。モデルは「どの操作を行うか」を何も述べていないので、衝突を推論できません。

## 観測

- `TestClosedRejectsCollidingWithCorrectionIsReported` 成功。`closed-rejects update`＋効果`update`の訂正能力で`PERIOD_CORRECTION_REFUSED_BY_CLOSURE`が報告される。
- `TestClosedRejectsWithoutCollisionIsAccepted` 成功。効果`create`の訂正能力は報告されない（衝突しない）。
- `TestCorrectionWithoutDeclaredEffectIsNotReportedAsACollision` 成功。効果を宣言しない訂正能力は報告されない。
- `internal/validation`全体は`ok`。例と`reference/workspace`の`validate`は`ok`（既存警告のみ）。
- 診断コードの網羅ラチェット試験（新コードは試験必須）も通る。

## 読み取れること

- 矛盾する2つの宣言が、**ソースを書いた時点で**報告される。
- 衝突しない組み合わせ（拒否が`update`・訂正が`create`）は報告されない。
- 実行時の免除と検証の報告が**両方**存在し、役割が分かれている。

## 読み取れないこと

- **矛盾を著者がどう直すか**の案内（訂正能力を別の効果にするか、`closed-rejects`を外すか）。診断は事実を述べるだけで、修正案は出していません。
- 効果の**Data**の一致（別Dataを変更する訂正能力は前ラウンドの検査が扱う）。
- 実行時の免除が**他のホストでも同じ**に働くか（今は`internal/command`だけが知っています）。
- 実Provider・実DB・旧Runtimeでの観測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。矛盾の宣言段階での報告は参照実装で確認済み、修正案の提示と他ホストでの免除の共有は未実装です。
