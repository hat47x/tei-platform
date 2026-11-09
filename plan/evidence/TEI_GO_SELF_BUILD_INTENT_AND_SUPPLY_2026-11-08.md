# `tei self-build`のSC0-01（Intent宣言）とSC0-02（供給照合）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.6で「未実装」と記録した二条件を埋めた記録です。

**`qualified`は常に`false`です。** artifactができたことは、受理でも、閉包資格でも、Runtime Authorityでもありません。

## 何を足したか

| 条件 | 実装 |
| --- | --- |
| SC0-01 | `-intent-file`を必須にし、JSON object・`formatVersion`・`subject.id`を検査。**宣言を読んでdigestを取るだけで、BuildPlanは導出しない** |
| SC0-02 | `-supply-verifier`を指定した場合、その実行ファイルを起動し、`verified`が真でなければ**childを起動する前に停止** |

### 設計上の判断

1. **分類器を`cmd/tei`へ複写しない。** `tools/internal/gobuilder`は`cmd/`からimportできない（[SB-C0 §3.1](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)）。供給の分類は、呼び手が**exactな実行ファイル**として渡し、`cmd/tei`はそのJSONを読むだけにした。§35.2が禁じた第二実装を作らない。
2. **検証器を指定しない場合は「委譲」と記録し、合格とは読まない。** `state: delegated-to-child`と`note`を残す。指定しないことが黙示の合格にならないようにする。
3. **`schemaVersion`が無い検証器出力は証拠として扱わない。** `verified: true`だけを返す未知の出力を受け入れない。
4. **Intentは「宣言」と「解決」を分ける。** `resolved: false`と`resolutionNote`を必ず出し、`IntentDeclaration`にdigest・formatVersion・subjectId・outputKind・recipe entrypointを記録する。宣言が読めたことを、解決済みBuildPlanの成立と読み替えない。

## 実測（WSL2 Ubuntu 22.04 / Go 1.27.3 linux/amd64）

対象packageは`./cmd/tei`。検証器は`tools/reference-go-module-supply-verify`、childは`tools/reference-go-build-with-supply`。

| 欄 | 値 |
| --- | --- |
| `ok` | `true` |
| `intentDeclaration.digest` | `sha256:6660187445ea7ee63f0aef64c7c31ff7ae95d1b68ed3d4b0c8eb16e2a3cc9775` |
| `intentDeclaration.resolved` | **`false`** |
| `intentDeclaration.subjectId` | `tei.plugin.authoring_example` |
| `supplyCheck.state` | `verified` |
| `supplyCheck.verified` | `true` |
| `supplyCheck.verifierDigest` | `sha256:7b694d37…` |
| `childRun.artifactCrossCheckMatched` | `true` |
| `childRun.invoked.executableDigest` | `sha256:f50cc9d9…` |
| `qualified` | **`false`** |

### 退け方を確かめた入力

| 入力 | 実測 |
| --- | --- |
| `-intent-file`なし | `SELF_BUILD_PRECONDITION_FAILED`、child未起動 |
| IntentがJSONでない | `SELF_BUILD_INTENT_INVALID`、child未起動 |
| 検証器が`verified: false`を返す | `SELF_BUILD_SUPPLY_UNVERIFIED`、child未起動 |
| 検証器の対象packageが外部Moduleを選ばない | 上と同じ（`the target selects no external module; the supply was never exercised`） |
| 検証器を指定しない | `state: delegated-to-child`、`verified: false`（黙示合格にしない） |

4行目は実際に踏んだ。最初、検証器へ`./tools/reference-build-authoring-intent`（外部Moduleを選ばない）を渡してしまい、`verified: false`で止まった。**検証対象はchildが構築する対象と同じでなければならない**ため、実測では`./cmd/tei`に揃えた。

## 検証

- `gofmt -l cmd/tei`：差分なし
- `go vet ./cmd/tei`：PASS
- `go test ./cmd/tei -count=1`：PASS（primitive 5件＋外形 14件）
- 実プロセスのend-to-end：上表のとおり成功

## 未実装・限界

- **Intentを解決していません。** `resolved: false`のままです。Build Intent→BuildPlanの導出は`tools/reference-build-authoring-intent`が持つ（`cmd/tei`はimportできない）。
- **L2（TEI-defined build）は成立しません。** Graphから実Buildへの結線はまだ無い。
- 供給のTrust・署名は未検証。`go.sum`自体のTrust根拠も未決。
- catalogへ載せていないため`tei commands`からは見えません（D-3）。
- G4は`UNQUALIFIED`のままです。
