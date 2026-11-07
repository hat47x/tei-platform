# 操作種別の語彙はどこに置くか — 現行の二つの閉じた集合（2026-11-07）

- 対象: [操作面の設計候補](../../design/OPERATION_SURFACE_DESIGN_2026-11.md)の15章 未決2（語彙を規則の版に置くか、#951のプラグイン契約へ移すか）
- 位置づけ: **実行して観測した記録**と、その比較。採否はOwner判断。
- 計測した版: `raw/head.txt`（`origin/main` `4994a4f0` の作業ツリー）。

設計15章の未決2は「操作の種別の語彙を、規則の版として管理することで足りるか、それともプラグインの契約（#951）へ移すか」です。判断には、**現行で語彙がどう閉じているか**の事実が要ります。この記録はそれを観測します。

## 1. 観測したこと

見積もり用の小さなGoプログラムを一時ディレクトリに置いて実行し、直後に削除しました（リポジトリへは書きません）。

```text
unknown plugin CapabilityKind accepted: false
  rejected with: capability billing.executor has invalid kind "billing-executor"
known plugin CapabilityKind accepted: true
derived operations for a World with no persistence: 0
  gaps: 0
  rule: tei.operation-surface/standard-crud/v1 sha256:92adf0fa…9ae0
```

| 語彙 | 置き場所 | 閉じ方 | 未知の値 |
| --- | --- | --- | --- |
| プラグインの`CapabilityKind` | `internal/plugin/model.go` の定数と`Valid()` | **閉じた列挙**（7種） | `ValidateCapabilityStructure`が**拒否**する |
| 操作の種別（`list`/`read`/`create`/`update`/`retire`） | `internal/operations/surface.go` の生成コードと`operationKindRank` | **規則の中の語彙**（定数は無い） | 語彙を検査する関数が無い。`sort`のrankが既定値5になる |

- プラグイン側は、未知の種別を**宣言の時点で拒否**します。追加は`model.go`の定数と`Valid()`の両方を変える必要があり、その追加は履歴上1回（`e4d63b3de`、6種→7種）だけです。
- 操作面側は、種別を**生成コードが文字列として書く**だけで、閉じた集合として検査していません。`operationKindRank`は並べ替えのための関数で、未知の種別を拒否しません（rank 5）。

## 2. 二つの選択肢の現状

| | 規則の版に置く（設計の案） | #951のプラグイン契約へ移す |
| --- | --- | --- |
| 語彙の追加 | `surface.go`の生成コードと`ruleVersion`を上げる | `plugin.CapabilityKind`の定数と`Valid()`を変える |
| 未知の種別の扱い | **現行は検査しない**（決めたら検査を足す必要がある） | 現行でも宣言時に拒否する |
| 影響範囲 | 操作面の規則（`tei.operation-surface/*`）だけ | プラグインの宣言・ロック・解決・runtime planまで |
| プラグインが自前の種別を足せるか | 設計は「足せる」と書く（49行）が、**実装は足せない**（生成コードが固定） | 足せない（閉じた列挙） |

設計49行は「種別の一覧は導出規則の語彙であり、中核の固定列挙値ではない。プラグインは自前の種別を追加でき、標準の種別は規則の版で管理する」と書いています。**この観測は、その後半（規則の版で管理する）がまだ実装されていないことを示します。** 現行の操作面は規則の版を持ちますが、語彙を版で管理する仕組み（未知の種別を拒否する検査、版ごとの語彙表）は持っていません。

## 3. この記録が示すこと

- 語彙は**二箇所に別々に**あり、閉じ方が違います。プラグイン側は閉じた列挙で拒否し、操作面側は開いた文字列で検査しません。
- 「規則の版に置く」を選ぶと、**現行に無い検査を足す**ことになります（未知の種別を拒否するか、`out-of-scope`として残すか）。「#951へ移す」を選ぶと、操作面の語彙がプラグイン宣言の語彙になり、**プラグインの宣言・ロック・runtime planの版まで影響**します。
- どちらを選んでも、**いまの「操作面は種別を検査しない」状態は変わります**。現状のまま放置すると、`operationKindRank`の既定値（5）に落ちる未知の種別が、エラーではなく「並び順の最後」として通ります。

## 4. この記録が示さないこと

- **どちらを選ぶべきか**。15章の未決2はOwner判断です。この記録は材料だけを出します。
- **プラグインが種別を追加できるようにすべきか**。設計の49行は案であり、実装されていません。
- **未知の種別の扱い**（拒否か`out-of-scope`か）。設計6章の語彙に`out-of-scope`はありますが、操作面の生成コードで使っているのは`list`〜`retire`だけです。
- **実DB・実Provider・権限**。合成のWorldと、パッケージの検証関数だけです。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（一時的なGoプログラムを作り、実行し、削除する） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域（0行） |
| `raw/vocabulary-probe.txt` | 実行結果 |
| `raw/summary.txt` | 上の表を作る元の一覧 |

## 6. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `0a87e44d4e744d0509ecb97921c8a7d8ea0f0f76ad261cdc8b47682c07db730c` | 3697 |
| `raw/head.txt` | `c13eee016a062bb9d302bee4446fc2b3036906c95d501fa75d43392700dfc2cb` | 73 |
| `raw/pre-status.txt` | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`（空） | 0 |
| `raw/vocabulary-probe.txt` | `36fa1a836acb388bc3ac7971e0a16af053fc3d548876c2a6783c373e61a060c8` | 353 |
| `raw/summary.txt` | `24598be094657726cc254c50284a3d7ffba4f438948541ea6ec9eeaf8bc8ad6a` | 408 |

`raw/vocabulary-probe.txt`が353バイトと短いのは、確認した問いが4つだけだからです。網羅的な語彙の一覧ではありません。
