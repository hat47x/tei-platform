# rea-intake が候補生成を失い、並行Relationの回帰3件が壊れている（2026-11-17、第87ラウンド）

- 起票: [#5248](https://github.com/hat47x/tei-platform/issues/5248)
- 前段: [第86ラウンド](../TEI_DOGFOODING_ROUND86_SCOPED_FIXTURE_TYPE_2026-11-17/README.md)
- 状態: **起票のみ。私は修正しない（設計判断のため）。**

## 1. 目的

`repository-health` に残る `TestREAIntake*` 系3件を、前ラウンドと同じ手法（表面のメッセージではなく実体を計測する）で調べる。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `fc3fe59da`（`origin/main`） |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.26.11 linux/amd64` |

## 3. 実測

3件とも**単独実行で FAIL**（テスト順序依存ではない）。

```text
--- FAIL: TestREAIntakeParallelRelationsStayUnacceptedInPreview
    separate edge occurrences collapsed: map[]
--- FAIL: TestREAIntakeCompleteGraphDoesNotProveFunctionOrBusiness
--- FAIL: TestREADualEntryCapturedInputAndIdentityBoundaries
```

## 4. 計装: 実際に生成されている候補

```text
DIAGNOSIS candidate kinds: map[data-element:3] (total=3)
```

fixture には**エッジが2本ある**（`graph.edges=1` ＋ テストが1本追加）のに、候補は `data-element` が3件だけ。**`cross-reference` 候補が1つも生成されていない。**

## 5. いつ壊れたか（二分探索）

```text
3429c7891  test(rea): 保存REA原本の二入口と並行Relation回帰を統合
            -> 3件とも PASS（追加時は緑）

606134e63  refactor(rea): restrict intake to evidence receipts
            -> ここで壊れた
```

## 6. 原因: 実データ由来の候補が固定3件のプレースホルダになった

```diff
-    set, err := reacandidate.GenerateCandidates(evidence)
-    port, err := reacandidate.NewWorkspaceReadOnlyModel(parsed.Workspace)
-    result, err := reacandidate.BindCandidates(port, set, mappings)
-    preview, err := reacandidate.PreviewAdoption(set, model)
+    "candidates": []map[string]interface{}{
+        {"candidateId": "candidate-1", "kind": "data-element", ...},
+        {"candidateId": "candidate-2", "kind": "data-element", ...},
+        {"candidateId": "candidate-3", "kind": "data-element", ...},
+    },
```

`reacandidate.GenerateCandidates` の呼び出しが削除され、固定3件に置き換わっている。

## 7. なぜ私が直さないか

`606134e63` は「**intake を receipt に限定する**」という意図的な設計変更に見え、**テストの要求と真正面から衝突**している。production 自身がこう書いている:

> This legacy CLI has synthetic candidate IDs, not source-backed CandidateSet identities.

つまり実装者は placeholder を認識しており、**テストが追随していない**状態。どちらを正とするかは設計判断である。

| 案 | 内容 | 判断 |
| --- | --- | --- |
| A | `rea-intake` が再び `GenerateCandidates` を使う | リファクタの意図と衝突しうる |
| B | 3テストを削除し placeholder を仕様として固定 | **並行エッジの区別という回帰検査を失う** |
| C | 私が A を実装 | **リファクタの設計意図を私が決めてしまう** — 採らない |

**A か B のどちらも所有者の判断**である。

## 8. 正確な事実整理

- `reacandidate` パッケージ（`GenerateCandidates`・`CandidateKindCrossReference`）は**今も存在**し、別経路から使われている。
- 失われたのは **`rea-intake` 経由での**候補生成だけ。
- よって「候補生成が消えた」ではなく、**`rea-intake` から切り離された**のが正確。

## 9. 読み取れないこと（NOT RUN）

- A か B かの設計判断（#5248）。
- 他経路（`inspect` 等）での候補生成の健全性。
- 実DB・実Provider・実Host、HTTP・Desktop。

## 10. 位置付け

**「緑だったテストが、意図的な設計変更で赤になった」回**である。壊れたのは事実だが、**直すには「どちらの設計が正か」を決める必要がある**。私は判断を起票して返し、事実（いつ・何が・どう変わり・何が失われたか）を確定させた。
