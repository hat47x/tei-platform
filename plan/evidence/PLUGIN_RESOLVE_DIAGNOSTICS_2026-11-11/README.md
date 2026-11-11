# プラグイン解決の拒否理由を、文字列から値へ（2026-11-11）

- 対象: [`internal/plugin/resolve.go`](../../../internal/plugin/resolve.go)、[`internal/plugin/requirements.go`](../../../internal/plugin/requirements.go)、[`internal/plugin/diagnostics.go`](../../../internal/plugin/diagnostics.go)
- 位置づけ: **製品コードの変更と実行記録**。
- 計測した版: `raw/head.txt`（`origin/main` `99fe97df` の作業ツリー＋このPRの変更）。

前の調査で、ロード時の拒否が**文字列だけ**で、呼出し側が `err.Error()` の部分一致でしか分岐できないことを記録しました。**その1点だけ**を直しました。

## 1. 何を変えたか

`fmt.Errorf` の11箇所を、**code と subject を持つ値**に置き換えました。

```text
DiagnosticDuplicateRequirement = "PLUGIN_RESOLVE_DUPLICATE_REQUIREMENT"
DiagnosticInvalidRequirement   = "PLUGIN_RESOLVE_INVALID_REQUIREMENT"
DiagnosticNoCompatiblePlugin   = "PLUGIN_RESOLVE_NO_COMPATIBLE_PLUGIN"
DiagnosticDuplicatePluginVersion = "PLUGIN_RESOLVE_DUPLICATE_PLUGIN_VERSION"
DiagnosticNoCompatibleBundle   = "PLUGIN_RESOLVE_NO_COMPATIBLE_BUNDLE"
DiagnosticDuplicateBundleVersion = "PLUGIN_RESOLVE_DUPLICATE_BUNDLE_VERSION"
DiagnosticIncompatibleTransitive = "PLUGIN_RESOLVE_INCOMPATIBLE_TRANSITIVE"
DiagnosticDependencyCycle      = "PLUGIN_RESOLVE_DEPENDENCY_CYCLE"
DiagnosticEnvironmentSelection = "PLUGIN_RESOLVE_ENVIRONMENT_SELECTION"
```

**`resolve.go` に残る `fmt.Errorf` は0件**になりました。

### メッセージは1文字も変えていません

`ResolveDiagnostic.Error()` は**以前と同じ文字列**を返します。型を足しただけで、**読む側の文字列は変わっていません**。

```text
== how many existing tests branch on the message text ==
  11ファイル・26箇所
```

**この26箇所は移行していません。** そのまま通っています（下の試験結果）。

## 2. 追加した API

| 関数 | 用途 |
| --- | --- |
| `ResolveDiagnostics(err)` | 連鎖している診断を**外側から順に**返す |
| `ResolveCode(err)` | 最も外側の code を返す（無ければ `""`） |

**`ResolveCode` が `""` を返すのは「理由が不明」であって「成功」ではありません。** その区別を試験で固定しています。

**連鎖を返すのは、拒否が入れ子になるためです。** 環境の選択一覧が不正な場合、外側は `PLUGIN_RESOLVE_ENVIRONMENT_SELECTION`、内側が具体的な理由（重複など）になります。**片方だけでは原因が分かりません。**

## 3. 観測

```text
--- PASS: TestResolveDiagnosticsCarriesACodeAndTheSameMessage
--- PASS: TestACycleReportsTheCycleCode
--- PASS: TestAnOrdinaryErrorHasNoResolveCode
--- PASS: TestADuplicateRequirementReportsItsOwnCode
--- PASS: TestAnInvalidRequirementReportsItsOwnCode
```

既存の試験はすべて通っています。

```text
ok  tei/internal/plugin
ok  tei/internal/plugin/implementation
ok  tei/internal/plugin/inputprofile
ok  tei/internal/plugin/runtime
ok  tei/internal/plugin/transform
```

**文字列に依存する26箇所が壊れていないこと**が、メッセージを変えていないことの実測です。

## 4. この記録が示すこと

- **ロード時の拒否理由が、機械可読になりました。** `ResolveCode` で分岐でき、`ResolveDiagnostics` で連鎖全体を報告できます。
- **既存の読み手は無傷です。** メッセージを保ったので、26箇所の文字列一致はそのまま通ります。**移行は別の判断**として残しました。
- **#3184 の分類（`IsInvalidLock`）と同じ形です。** 既存の `InvalidLockError` は `errors.As` で分類する型でした。**その前例に合わせました。**

## 5. この記録が示さないこと

- **26箇所を移行すべきか。** コードで分岐する方が頑健ですが、**移行は別作業**です。
- **code の語彙を Owner が承認したこと。** これは resolver の rule vocabulary として私が置いたもので、**受理ではありません**。
- **`formatversion.Diagnostic` と統一すべきか。** あちらは意味要素向けの欄（`ElementID`・`Attribute`）を持ち、**Plugin 向けの欄（対象 ID・要件・版）を持ちません**。統一は設計判断です。
- 動的ロード・外部プロセス方式。

## 6. 検証の制約

**`cmd/tei` の全試験は600秒で打ち切られます。** これは**clean main でも同じ**（別 worktree で再現）なので、**私の変更に起因しません**。代わりに、解決に関わる CLI 試験だけを実行しました。

```text
go test ./cmd/tei -run 'Plugin|Lock|Resolve' -count=1   ok (55.5s)
```

## 7. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（code 一覧・残存 `fmt.Errorf`・文字列依存の件数・試験） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域（このPRの未追跡ファイルのみ） |
| `raw/anchors.txt` | 上の1節の出力 |
| `raw/new-tests.txt`・`raw/new-tests-summary.txt` | 追加した試験の出力 |
| `raw/tests.txt`・`raw/tests-summary.txt` | 既存を含む package 試験の出力 |

## 8. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `94eac15962a9950c5735d32d927a534b5a163029f7eb5d17ea6fe79225b32678` | 2089 |
| `raw/head.txt` | `e75657b8630893f9a4866b315d678a39e0a22fc150246f11fc7886a0753f89e9` | 73 |
| `raw/pre-status.txt` | `c9ea0cc4340d22e0a742ccb1c5979aa2f5e4d56424d0ceeb8c5a1c4b38a0cc2c` | 403 |
| `raw/anchors.txt` | `1ef0a208287c2db0b1fd4a50f0e0fd69053fdad7a5621df5ca1963fe6d99e855` | 1414 |
| `raw/new-tests.txt` | `f7cd4ddae305ea755461d300ca985cc7c6870639de9d19583c97b1da16303770` | 7453 |
| `raw/new-tests-summary.txt` | `e56dbdc13b063736afd8813581fe3f4cf33e80a0ac38deb799324dbd675022c9` | 5156 |
| `raw/tests.txt` | `51ffa158509ceb3bcd05e180073e9f0ac12ef11896cb171ce46911af09bcec79` | 271 |
| `raw/tests-summary.txt` | `5adc76eaa1cb9b2ac2bd42ce54399ffec13aa1daf63a1a3da711e45eb211b20a` | 207 |
