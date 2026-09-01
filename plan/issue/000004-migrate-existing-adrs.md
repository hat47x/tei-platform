# ISSUE-000004: 既存ADRを`plan/adr`へ段階的に移行する

- Status: Proposed

## 背景

ADR-0013で、今後の上位・横断的なADRは`plan/adr/`へ置く方針を採用しました。

一方、既存のADR-0001〜0012は`docs/adr/`にあり、README、Architecture、Status、Conformance等から多数参照されています。単純な一括移動は、既存リンクと過去の文脈を壊す可能性があります。

## 関連するProduct Requirement

- RQ-021 Reference / Conformance
- RQ-022 実装済み範囲と設計上の主張の区別

## 作業内容

- `docs/adr/0001`〜`0012`の参照元を列挙する
- ADR番号は維持する
- `plan/adr/`へ移す場合のredirect / compatibility方針を決める
- README、Architecture、Status、Conformanceからのリンクを一貫して更新する
- `docs/adr/README.md`の役割を終了するか、互換入口として残すかを決める
- 移行後もGit historyから元の文脈を追えることを確認する

## 制約

- 番号を振り直さない
- 既存ADRの意味を移行作業のついでに変更しない
- リンクを壊した状態を中間成果としてmainへ入れない
- 新旧の二つを恒久的な正本として残さない

## 完了条件

ADR-0001以降を一つの連続した判断履歴として追跡でき、現在の正規保存場所が`plan/adr/`であることをRepository全体から一貫して確認できること。
