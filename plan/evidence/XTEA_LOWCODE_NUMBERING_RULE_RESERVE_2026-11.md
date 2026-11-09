# 採番規則から発行までの一続きの経路の確認（2026-11）

## 対象

- 実装: `internal/numbering/rules.go`（`ResolveRules`・`Rules.BindRule`・`Rules.ReserveRule`・`ReserveRuleWithoutTransaction`）
- 試験: `internal/numbering/rules_test.go`
- 対象版: `main` の `070834044` から分岐した`feat/numbering-rule-reserve-20261108`
- 環境: Windows、Go `1.27.0`

## 実行

```powershell
go test ./internal/numbering -run 'TestResolveRules|TestBindRule|TestReserveRule' -count=1 -v
```

## 観測

- `TestResolveRulesBindsTheNamedRule` 成功。1つのDataの`order`（`ORD-`・41）と`invoice`（`INV-`・7）を、規則名で別々に束縛する。
- `TestBindRuleRefusesAMissingRule` 成功。行の無い規則は拒否され、別規則の番号を発行しない。
- `TestBindRuleRefusesTwoRowsForOneRule` 成功。1つの規則に2行ある場合は拒否される（どちらのカウンタか決められない）。
- `TestReserveRuleIssuesFromTheNamedRule` 成功。規則名から契約を束縛し、取引内で`ORD-00042`を発行し、Commit後にカウンタが42。
- `TestReserveRuleWithoutTransactionRollsBackOnRelease` 成功。取引を持たない経路でも、Releaseでカウンタが7へ戻る。
- `TestReserveRuleReportsAConcurrentChange` 成功。先行予約が後続の発行に追い越された場合、先行のRollbackは`ErrNumberingConflict`で拒否され、後続の`ORD-00043`を上書きしない。
- パッケージ全体は`ok`。`internal/validation`も`ok`。

## 読み取れること

- 「規則名 → 契約の束縛 → 取引内の発行」が1回の呼出しで閉じる。呼出し側が契約と状態を2段階で持ち回る必要がない。
- 1つのDataに複数の規則を置き、規則ごとに独立した番号を発行できる。
- 追い越された予約の取消は、後続の発行を壊さずに競合として報告される。

## 読み取れないこと

- Workspace例への複数規則Dataの追加。
- Command境界の`UnitOfWork`実装との共通化（今は番号側の小さな契約）。
- 実DB・旧Runtimeの実測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。規則名から発行までの経路は参照実装で確認済み、例への追加とCommand側UoWとの共通化は未実装です。
