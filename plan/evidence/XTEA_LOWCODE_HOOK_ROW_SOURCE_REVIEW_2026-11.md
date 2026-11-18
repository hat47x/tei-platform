# 行フックの出所宣言への独立レビューと、指摘の反映（2026-11-18）

## 対象

- レビュー対象: #5460「行スコープのフックが行の出所を宣言する」
- 反映: `internal/validation/application_pattern.go`（出所の一致検査）、`internal/hook/reach.go`（skip理由の文言）
- 試験: `internal/validation/application_pattern_hook_row_source_differs_test.go`

## レビューの結論

**blocking 0件 / non-blocking 4件 / question 3件。** 設計の芯（語彙を Query だけに限定・既定の参照実行は依然 skip・証拠文書が限界を明記）は実装と一致していた。

## 反映したもの

### non-blocking 1: 「通るのに走らない」が1つ残っていた

**バリデーションはフックごとに独立して出所を検査するが、フック間の一致を見ていない。** 一方 `internal/hook/row_source.go` は全ての row スコープ宣言に**同一の queryID** を要求する。したがって**異なる出所を2つ書いた pattern は validate 緑・実行は必ず `HOOK_ROW_SOURCE_MISMATCH`** になる。これは D-9（受理されるが実行されない）の形そのもので、証拠文書の検証表にも記載が無かった。

`APPLICATION_PATTERN_INTERACTION_HOOK_ROW_SOURCE_DIFFERS` を足し、**1 Pattern 内で出所が食い違えばエラーで拒否**するようにした。理由: 1回の呼出しには行集合が1つしか渡らないので、**2つの出所は原理的に両立しない**。

試験2件（食い違えば拒否・同じ出所なら受理）。**陰性対照を実測**: 検査を無効化すると `two row sources were accepted: []validation.Diagnostic(nil)` で落ち、戻すと通る。

### non-blocking 3: skip 理由の文言が不正確

空ページ（0行）は「照会したが該当なし」と「呼出し元が渡していない」を区別できず、「pass the source's rows to apply it」 と出る。実際にはホストは行を渡している（0件）。**「宣言した出所の行がこの実行へ渡されていない」**へ直した。

## 反映していないもの（記録した判断）

- **non-blocking 2**: 「一致確認」は queryID の一致と「宣言 reads が各行にキーとして存在」だけである。**同じ Canonical ID を持つ別定義の結果**や、呼出し元が組み立てた列はすり抜ける。`hook.Runner.Rows` は宣言と無関係に設定できるため、**境界での強制ではない**。ホスト入口経由なら防げるが、`Runner` を直接使う経路は残る。**未修正**（境界の型を変える設計変更なので独立に扱う）。
- **non-blocking 4**: 警告コード名 `..._HAS_NO_SUPPLIER` は「供給者が無い」を意味するが、今は「出所が宣言されていない」の意味に狭まった。**コード名は文字列定数で、改名は契約変更**なので独立に扱う。
- **question 2**: `rows from query` かつ `scope` 未指定（`Scope == ""`）は `ROW_SOURCE_SCOPE_MISMATCH` で拒否されるが専用試験が無い。**未追加**。
- **question 3**: 証拠文書の「cmd/tei も ok」は、この環境では `/bin/sh` 不在で再現しない（`TestDriveSelfBuildInKeepsTheChildAsTheRecordedBuilder`、#4578 由来・環境依存）。**記録側で区別すべき**（未修正）。

## 読み取れること

- **異なる出所を2つ宣言した Pattern は、validate で拒否される**（実行時に必ず失敗する宣言を、宣言時に止める）。
- 同じ出所を2つのフックが共有する形は**受理される**（1つの行集合が両方に使える）。

## 読み取れないこと

- `Runner.Rows` を直接設定して宣言照合を迂回する経路（境界の強制が無い）。
- Relation 出所を語彙から外したことで実際の用途が塞がれていないか（必要用途の根拠が証拠文書に無い）。
- 既定の参照実行で Query を実行する配線（依然として無い）。

## 位置付け

参照実装のレビュー記録。製品の受入や本番適格性を示さない。仮承認。
