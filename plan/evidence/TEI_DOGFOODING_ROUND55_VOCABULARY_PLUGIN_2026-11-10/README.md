# 語彙診断の第3弾: plugin 系統5件（2026-11-10、第55ラウンド）

- 対象: [#4094](https://github.com/hat47x/tei-platform/issues/4094)（第3弾）。
- 前段: [第54ラウンド](../TEI_DOGFOODING_ROUND54_VOCABULARY_BATCH2_2026-11-10/README.md)。
- 実行日: 2026-11-10（Asia/Tokyo）。
- 状態: **plugin 系統の5件を修正。残り8件。**

## 1. 目的

#4094 の残り13件のうち、`plugin` にまとまっている5件を処理する。前ラウンドで「`plugin` 系統5件がまとまっているので効率的」と判断した。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `f1d1d58e1`（`origin/main`） |
| CLI実行物（修正前） | `sha256:89d93cd378f644992810d9adde6426df5ec6c0336cd8218a701208f36539c0b7` |
| CLI実行物（修正後） | `sha256:791bf87d2588b5538cab43f0b500d6ff61a08e3d4fa2ef4a518de5ffc48adbff` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

## 3. 修正した5件

| コード | 語彙 | 値 |
| --- | --- | --- |
| `INVALID_PLUGIN_RUNTIME` | runtime kind | builtin, process |
| `INVALID_PLUGIN_SANDBOX_POLICY` | sandbox policy | none, preferred, required（**または省略**） |
| `INVALID_PLUGIN_CAPABILITY_KIND` | capability kind | deployment-executor, observation-provider, evidence-provider, database-provider, contract-query-executor, migration-provider, editor-intelligence |
| `INVALID_PLUGIN_EXTERNAL_SCOPE` | external scope | editor, build, test, runtime, migration |
| `INVALID_PLUGIN_EXTERNAL_KIND` | external kind | toolchain, runtime, language-package, tool |

単一情報源として `plugin.RuntimeKinds`・`plugin.SandboxPolicies`・`plugin.CapabilityKinds`・`plugin.ExternalScopes`・`plugin.ExternalKinds` を追加し、各 `Valid()`／`ValidScope()`／`ValidKind()` がそれを読む。

### 3.1 sandbox だけ扱いが違う

`SandboxPolicy.Valid()` は**空文字も受理する**（「宣言していない」を意味する）。そこで:

- 一覧 `SandboxPolicies` には空を含めない（著者に空を書けと言うべきではない）
- 診断文は `valid values are none, preferred, required (or omit the policy)` とし、省略可能であることを示す

この区別を試験で固定した（`TestSandboxDiagnosticSaysThePolicyMayBeOmitted`）。

## 4. 修正後の実測（CLI で確認）

```text
runtime:
  plugin probe.runtime has invalid runtime gibberish; valid values are builtin, process

sandbox:
  plugin probe.sandbox has invalid sandbox policy gibberish;
  valid values are none, preferred, required (or omit the policy)

capability kind:
  plugin probe.cap capability probe.thing has invalid kind gibberish;
  valid values are deployment-executor, observation-provider, evidence-provider,
  database-provider, contract-query-executor, migration-provider, editor-intelligence

external scope:
  plugin probe.scope external requirement thing has invalid scope gibberish;
  valid values are editor, build, test, runtime, migration

external kind:
  plugin probe.kind external requirement thing has invalid kind gibberish;
  valid values are toolchain, runtime, language-package, tool
```

正常なワークスペースは `ok` のまま。**5件すべて CLI から到達できた**（前ラウンドの `INVALID_SEMANTIC_TYPE_KIND` と違い、到達不能なものは無い）。

## 5. 試験

| ファイル | 試験 |
| --- | --- |
| `internal/plugin/vocabulary_test.go` | 5つの一覧と述語の一致。宣言数との一致。空の sandbox が受理されるが一覧には載らないこと |
| `internal/validation/plugin_vocabulary_diagnostic_test.go` | 5つの診断がそれぞれの有効値をすべて名指し、「valid values are」で導入する。sandbox は省略可能であることを示し、空を選択肢として並べない |

### 5.1 変異試験

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | external scope 診断から一覧を削る | FAIL `external scope: message does not name "editor"` |
| M2 | `SandboxPolicies` に空を加える | FAIL（空を選択肢にしないことを固定している試験） |
| M3 | `CapabilityKinds` から `editor-intelligence` を落とす | FAIL `CapabilityKinds has 6 entries but 7 are declared` |

## 6. 回帰確認

| 対象 | 結果 |
| --- | --- |
| `./internal/plugin/...`（6パッケージ） | ok |
| `./internal/validation` | ok |
| `./internal/workspace ./internal/operations` | ok |
| `./cmd/tei` | ok（195.7s） |
| `go vet` / `gofmt -l .` | 指摘なし / 空 |

## 7. 残り8件

#4094 の17件のうち、第54ラウンドで4件、本ラウンドで5件を直したので **8件**が残る。

| 系統 | コード |
| --- | --- |
| `migration` | `INVALID_MIGRATION_RECOVERY`・`INVALID_MIGRATION_STEP_KIND` |
| `releaseplan` / `deployment` | `INVALID_DATABASE_RELEASE_DATA_READINESS`・`INVALID_SOLUTION_RELEASE_STEP_KIND`・`INVALID_SOLUTION_RELEASE_PHASE`・`INVALID_ROLLOUT_STRATEGY` |
| 到達不能 | `INVALID_SEMANTIC_TYPE_KIND`（第54ラウンドで修正済み、CLI から観測不能） |
| その他 | `INVALID_PLUGIN_ARTIFACT` 等（語彙ではなく構造の検証） |

## 8. 読み取れないこと（NOT RUN）

- 残り8件の修正。
- `migration`・`releaseplan`・`deployment` の語彙が CLI から到達できるか（本ラウンドの手法で個別に測る必要がある）。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 9. 位置付け

**前ラウンドで「plugin 系統5件がまとまっている」と見立て、そのとおりに進んだ回**である。5件とも CLI から到達でき、うち1件（sandbox）は空を許す点で他と扱いが違ったため、そこだけ記述を分けた。

`#4094` は 17→8 件。まだ完了ではないが、残りは `migration`・`releaseplan`・`deployment` の3系統に絞られた。
