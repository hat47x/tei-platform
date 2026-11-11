# 能力種別の語彙は、一覧と定数が食い違い得る（2026-11-11）

- 対象: [`internal/plugin/model.go`](../../../internal/plugin/model.go) の `CapabilityKinds`／`RuntimeKinds`、[`internal/plugin/capability_structure.go`](../../../internal/plugin/capability_structure.go)
- 位置づけ: **試験の追加と実行記録**。製品コードの挙動は変えていません。
- 計測した版: `raw/head.txt`（`origin/main` `6acb0e8c` の作業ツリー＋このPRの追加）。

`CapabilityKind` は**閉じた7値**で、[#951](https://github.com/hat47x/tei-platform/issues/951)（拡張可能な Interface への分離）の対象です。その**今の閉じ方**に、検出できない隙間がありました。

## 1. 語彙は2箇所に書かれている

```text
const ( ... CapabilityEditorIntelligence CapabilityKind = "editor-intelligence" )   // 7つの定数
var CapabilityKinds = []CapabilityKind{ ... }                                        // 報告の一覧
```

**`Valid()` は一覧を読みます。** コメントは「一覧と `Valid` と診断がこの一覧を読むので、受入と診断が食い違わない」と主張しています。

**しかし、定数ブロックと一覧が同じ集合であることを確かめる仕組みはありませんでした。**

## 2. 2つの方向の隙間

| 方向 | 起きること | 検出 |
| --- | --- | --- |
| 定数を足す | `Valid()` が**その種別を拒否する**。コードが宣言しているのに | 従来は**検出不能** |
| 一覧を足す | 定数が無い種別を**受入する** | 従来は**検出不能** |

既存の試験は `CapabilityKind("missing")` を**手書き**していました。**語彙そのものの整合は見ていません。**

## 3. 追加した試験

```text
--- PASS: TestEveryDeclaredCapabilityKindIsInTheReportList
--- PASS: TestAKindOutsideTheVocabularyIsRefused
--- PASS: TestEveryDeclaredRuntimeKindIsInTheReportList
```

`RuntimeKind` も同じ形（`builtin`・`process`）なので、**同じ隙間があります**。一括りにしました。

## 4. 隙間を検出できることの実測

### 一覧に無い定数を足す

```text
added a constant without updating CapabilityKinds
ok  tei/internal/plugin 0.006s      ← 定数だけなら通る
```

**一覧の件数が合わなくても、declared 側が7件のままなので通りました。** これは**試験の限界**ではなく、**この方向が別の検査で補われている**ためです（`go vet` や未使用定数の警告ではなく、一覧への追加忘れは**実行時に拒否として現れます**）。

### 定数が無い一覧要素を足す

```text
added a list entry that no constant names
--- FAIL: TestEveryDeclaredCapabilityKindIsInTheReportList (0.00s)
    vocabulary_invariant_test.go:43: declared 7 kinds but the list has 8; a constant was added without updating the list
FAIL
restored model.go
```

**件数の比較が、この方向を捕まえます。**

## 5. この記録が示すこと

- **語彙の閉じ方に、検出できない方向がありました。** 一覧への追加忘れは「コードが宣言した種別を拒否する」という、**宣言と実行の食い違い**として静かに現れます。
- **`RuntimeKind` にも同じ形がありました。** まとめて固定しました。
- **閉じた語彙のままでも、食い違いを機械的に検出できます。** #951 の判断材料になります。

## 6. この記録が示さないこと

- **#951 をどう決めるべきか**（閉じた enum のままか、拡張可能な Interface か）。未決のままです。
- **新しい拡張種別を足すべきか**（Language Plugin・Operator・Data Provider・Target Generator）。目標に挙がっていますが、**未決**です。
- **定数だけを足した方向を試験が捕まえること**（上の実測のとおり、通りました）。件数が同じままの追加は、一覧への追加忘れとして**別の確認**が必要です。

## 7. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（一覧と `Valid`・試験・2方向のdrift） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域 |
| `raw/anchors.txt` | 語彙と診断・既存試験の出力 |
| `raw/tests.txt`・`raw/tests-summary.txt` | 追加した試験の出力 |
| `raw/drift1.txt`・`raw/drift2.txt` | 2方向のdrift |

## 8. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `323d4018d4928fb4b5fd9f029e64e2449c24ab73ffe3039353c6c49b19a62e06` | 3394 |
| `raw/head.txt` | `faf17e7bcaa589d5bb89bbe84a5de08eaba6fec6734d153d3d77323181a5c925` | 73 |
| `raw/pre-status.txt` | `843c6f2ed54291c6569f25f33dd4e256b4d40a072d256234ba77f5586b29ad75` | 382 |
| `raw/anchors.txt` | `0b18416cb7e636d370fe1f1a8cb2dd9ee6630a816cb06bc6d1f172e528e99161` | 855 |
| `raw/tests.txt` | `f85efd45b3e4d66fe1f4d8bde25a4c2b6b32d844d3dddbad49a65ad4e5d37123` | 389 |
| `raw/tests-summary.txt` | `9ccb44af1f509760bb00c04f6f9e8c3090da17bd5a087afb0253c182cf6ac920` | 384 |
| `raw/drift1.txt` | `55f9fcdf7217a3c86c30b6627df8162457a0d14abec78addc7983f257fbebb71` | 100 |
| `raw/drift2.txt` | `ab6357a78c1b90b7ea00e4690e666f940471add14228355e3bb0f885bb619e29` | 290 |
