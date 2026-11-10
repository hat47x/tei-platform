# フック範囲の貫通試験の確認（2026-11）

## 対象

- 試験: `internal/hook/scope_through_test.go`
- 対象版: `main` の `2fdca879c` から分岐した`test/hook-scope-through-20261110`
- 環境: Windows、Go `1.27.0`

## なぜ書いたか

前ラウンドで、範囲（scope）を層ごとに足した結果**各層の試験は緑のまま、貫通する経路だけが存在しない**状態が起きました。`row`は宣言できず、実装全体が到達不能でした。層ごとの試験では、この隙間が見えません。

そこで「宣言（構文）→ 検証 → 実行」を**1本の試験**で通すことにしました。次に同種の機能を足すときの型になります。

## 試験の内容

### `TestScopeIsDeclarableValidatedAndRun`

`.tei`のソースから書き起こしたPatternを、3つの層に通します。

1. **構文**: `hook before-update { scope <word> ... }`が読める。
2. **検証**: `validation.Validate`がエラーを出さない。
3. **実行**: `hook.Runner`が実際にフックを呼ぶ（`row`なら行の数だけ、`record`と範囲なしなら1回）。

範囲なし・`record`・`row`の3通りを回します。

### `TestScopeValidationAndRunAgreeOnRefusal`

逆方向も確かめます。`item`はrecord Patternでは検証に拒否され、**実行側も同じく拒否**します（`HOOK_SCOPE_UNAPPLIED`）。検証を迂回した呼出しでも、黙って効かないフックは走りません。

## 試験が本当に隙間を捉えるかの確認

試験を書いただけでは「たまたま通った」のか「隙間を捉える」のか分かりません。そこで**前ラウンドの修正を一時的に戻して**、試験が落ちることを確かめました。

```
--- FAIL: TestScopeIsDeclarableValidatedAndRun/scope=row
    scope_through_test.go:102: validation refused the declaration:
    APPLICATION_PATTERN_INTERACTION_HOOK_SCOPE_MISMATCH record pattern
    order_entry.order_record hook before-update declares scope row that the
    pattern kind does not look at
```

修正を戻すと`row`だけが落ち、戻すと通ります。**この試験は前ラウンドの隙間を実際に捉えます。**

## 観測

- `TestScopeIsDeclarableValidatedAndRun` 成功（範囲なし・`record`・`row`の3副試験）。
- `TestScopeValidationAndRunAgreeOnRefusal` 成功。
- `internal/hook`・`internal/validation`・`internal/syntax` は`ok`。

## 試験を書く過程で分かったこと

構文のパーサは**1ファイル1名前空間**しか読まないため、意味の宣言とPatternの宣言を1つのソースに置くと`expected type, data, ...`で落ちました。2つに分けて解析する必要があります。これは例のWorkspaceが意味とPatternを別ファイルに置いている理由でもあります。

## 読み取れないこと

- **`row`範囲の実運用**。貫通試験は`Runner.Rows`を明示的に渡しており、CLI/Studioが行集合を供給する経路は未実装です。
- `browse`・`header-detail`の`interaction`（現行の契約では宣言できません）。
- `item`・`header`の適用。
- 実Provider・実DB・旧Runtimeでの観測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。範囲の宣言・検証・実行が一致することを1本で確認するもので、実運用の経路と`browse`系の`interaction`は未実装です。
