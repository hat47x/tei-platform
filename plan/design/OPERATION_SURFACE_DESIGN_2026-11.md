# 操作面の導出と、操作ごとの自律区分（設計候補）

- 状態: **提案中（未採用）**
- 対応課題: [#1127](https://github.com/hat47x/tei-platform/issues/1127)（標準CRUD・検索・保守画面）、[#1129](https://github.com/hat47x/tei-platform/issues/1129)（副作用規則）、[#1130](https://github.com/hat47x/tei-platform/issues/1130)（一括処理）、[#2144](https://github.com/hat47x/tei-platform/issues/2144)（業務日付と期間）、[#1803](https://github.com/hat47x/tei-platform/issues/1803)（設計成果物の導出）
- 関連文書: [業務副作用の段階・確定後効果・再試行の設計](BUSINESS_EFFECT_LIFECYCLE_DESIGN_2026-10.md)、[Application Pattern v1 設計](APPLICATION_PATTERN_V1_DESIGN.md)、[三視点の意味参照と変更影響](../design/CROSS_VIEW_TYPED_REFERENCE_CONTRACT_2026-10.md)、[全体構成](../../docs/architecture.md)
- この文書の役割: 個別設計（#1127・#1129・#1130・#2144・#1803）が共通に使う**読み方と導出の規約**を一か所に置く。個別の実行規則はそれぞれの正本へ置き、ここへ複製しない。

## 1. 解決したい問題

TEIの作業領域には、業務操作を作る材料がすでに揃っています。`data` と `relation` が対象を、`capability` が入出力と効果を、`validation` が前提を、`persist` が保存先を、`pattern` がアプリケーション上の意図を持ちます。にもかかわらず、次の三つができません。

1. **「このアプリケーションで何ができるか」を一か所で読めない。** 操作は五つの宣言に分散し、一覧・入力項目・実行可能な条件・押せる操作は、どれも明示されていません。
2. **「何が足りないか」が沈黙する。** `record` パターンは対象データを示すだけで、項目・検証・更新可能性については何も言いません。情報が足りなくても診断は成功し、実行して初めて失敗します。
3. **「人がいなくても実行してよいか」を毎回人が判断している。** 夜間の締め、ファイル取込、一括確定は同じ仕組みで実行されますが、何が無人で走ってよいのかは、コードを読むまで分かりません。

三つは別々の問題に見えて、原因は同じです。**操作が、意味の層で一級の単位になっていない。** 操作は `capability`（意味）と `pattern`（アプリケーション上の意図）と `validation`（前提）に分かれて存在し、それらを束ねた「操作」という読み取りがありません。

## 2. この文書が定めること・定めないこと

定めること。

- 操作を、既存の五つの宣言から**読み取る**規約（導出規則、来歴、未解決の扱い、決定性）。
- 宣言と導出が食い違ったときの優先順位と、その食い違いを隠さない方法。
- 操作ごとの**自律区分**（無人で実行してよいか）を、宣言された事実から導く規則。
- 導出結果を人と機械が読むための出力契約（`tei operation surface`）。

定めないこと。

- 新しい中核要素、新しい固定の列挙値、新しい正本。操作面は**導出されたビュー**であり、正本は既存の `data`／`capability`／`validation`／`persist` のままである（[ADR-0033](../adr/0033-core-and-plugin-subprojects.md)の方針と、#951が求める「中核に固定の列挙値を増やさない」制約に従う）。
- 実行時の意味。実行は既存の Capability 呼出し・Effect Fence・Business Result・Observation の経路を通る。操作面は実行経路を置き換えない。
- 画面の配置や見た目。提示は実行先の責務である。

## 3. 操作とは何か

**操作**（business operation）は、業務上意味のある一回の作用である。少なくとも次を持つ。

| 要素 | 内容 | 現在の持ち主 |
| --- | --- | --- |
| 対象 | どの `data`／`relation`／業務手順に対する操作か | `data`、`relation` |
| 種別 | 一覧・参照・登録・更新・廃止・実行・集計・取込・出力など | `pattern` の種別、`capability` の意図 |
| 入力 | 型付きの入力項目と、必須・任意 | `capability` の入出力、`data` の属性 |
| 前提 | 検証、状態、期間、権限 | `validation`、期間の宣言、`authorization` |
| 効果 | 何を create／update／delete するか | `capability` の `effect`、`persist` |
| 結果 | 業務結果（名前付きの結果を含む）と実行エラー | ADR-0020 |
| 観測 | 実行後に別の読取りで確かめる方法 | ADR-0023 |
| 来歴 | 宣言か導出か、どの規則と入力から得たか | この文書 |
| 自律区分 | 無人で実行してよいか | この文書 |

種別の一覧は**導出規則の語彙**であり、中核の固定列挙値ではない。プラグインは自前の種別を追加でき、標準の種別は規則の版で管理する。この区別を崩すと、#951が解消した「中核に列挙値が増え続ける」問題が戻る。

## 4. 操作面

**操作面**（operation surface）は、あるアプリケーションについて、意味から決まる操作の集合を、宣言・導出・未解決の区別を保ったまま並べたものである。

```text
受理済みの意味（data / relation / candidate-key / capability / validation / persist）
        │
        ├─ 宣言された操作      pattern（browse / record / reference / command / header-detail）
        │                      capability（入出力・効果・結果）
        │
        └─ 導出された操作      標準の操作（一覧・参照・登録・更新・廃止）
                               規則の版と入力の照合値つき
        │
        ▼
   操作面（読み取り専用のビュー）
        ├─ operations[]     各操作の要素・来歴・自律区分
        ├─ gaps[]           導出できなかった理由（空にしない）
        └─ conflicts[]      宣言と導出の衝突
```

操作面は**実行権限を持たない**。ここに載ったことは「実行できる」ことを意味せず、「この意味からはこの操作が読み取れる」ことだけを意味する。実行の可否は、既存の認可・Effect Fence・配備の判定が決める。

## 5. 導出規則の契約

導出は、次の契約を満たす決定的な関数とする。

```text
ruleId      規則の識別子（例: tei.operation-surface/standard-crud）
ruleVersion 規則の版（意味が変わったら上げる。互換のない変更は別IDにする）
fingerprint 規則集合と入力の照合値（同じ値なら同じ結果を要求できる）
```

各操作と各 gap は、少なくとも `origin` と `basis` を持つ。

- `origin`: `declared`（宣言のみ）／`derived`（導出のみ）／`overridden`（宣言が導出を置き換えた）。
- `basis`: どの入力（data の版、persist の版、規則の版）から導いたかを示す照合値。
- `gap`: 導出できなかった理由の区分（6章）。

規則は次の性質を満たす。

1. **決定性。** 同じ受理済みの意味と同じ規則の版からは、同じ操作面（順序を含む）が得られる。ディレクトリの列挙順やマップの反復順を意味にしない。既存の [`internal/dependency.Derive`](../../internal/dependency/derive.go) が要件を導出するときと同じ規律である。
2. **非昇格。** 導出結果は正本ではない。受理・保存・実行の根拠にしない。正本を変えるには、既存の Shared Acceptance の経路（候補 → 確認 → 受理 → 再読込み → 適合確認）を通す。
3. **非隠蔽。** 導出できなかったこと、宣言と衝突したことを、空の結果で表さない。`gaps` と `conflicts` を必ず返す。
4. **入力の版への束縛。** 操作面は入力の照合値を持つ。実行時に照合値が変わっていれば、古い操作面で実行しない（7章）。

## 6. 未解決（gap）の分類

「導出できなかった」を一つの失敗にまとめると、次に何をすればよいか分からない。少なくとも次を区別する。

| 区分 | 意味 | 例 |
| --- | --- | --- |
| `missing-input` | 導出に必要な宣言がない | `persist` がないので更新操作を導けない |
| `missing-semantics` | 宣言はあるが、意味が決まっていない | 主キーも候補キーもないので一件参照を導けない |
| `conflict` | 宣言どうし、または宣言と導出が両立しない | 読み取り専用のデータソースに更新操作が宣言されている |
| `out-of-scope` | この規則の版では導出しないと決めている | 検索条件・並び替えは #1640（D2）の担当 |

`missing-input` と `out-of-scope` を混ぜると、未実装と対象外の区別が失われる。#1973の横断基準（合成データの成功を実環境の証拠にしない）と同じ理由で、この区別は表示だけでなく判定にも使う。

## 7. 宣言と導出の優先順位

- 宣言された操作は、同じ対象・同じ種別の導出された操作を置き換える。置き換えた事実は `origin=overridden` として残し、`overriddenBy` に宣言の識別子を持つ。
- 宣言が導出より**効果を増やす**場合は、効果の明示宣言を要求する。暗黙に効果を足さない。
- 宣言が導出を消すことはできない。消したい場合は、規則の対象外として明示する（`suppress` の宣言）。抑制も来歴に残す。
- 衝突（`conflict`）は自動解決しない。診断として返し、受理を止める。

## 8. 自律区分

自律区分は、**無人で実行してよいか**を示す。人が開発をどこまで自律的に進めてよいかを示す[ADR-0035](../adr/0035-autonomy-levels-and-verification-packets.md)のA0〜A4とは別の軸であり、混同しない。

| 区分 | 条件（すべて満たすこと） |
| --- | --- |
| `unattended` | 読み取り専用である。または、実行主体が環境に宣言され（サービス主体）、権限が呼出しごとの人の判断に依存せず、冪等の識別子が宣言され、人の判断を要する段階を含まない |
| `attended` | 実行主体が呼出しごとに決まる（利用者の資格情報で実行する） |
| `human-gated` | 人の判断・承認・例外判断の段階を含む、または含むかどうかが未解決である |

規則上重要なのは、**自律区分はフラグではなく、宣言された事実からの導出だ**という点である。`unattended` と書けるのは、環境にサービス主体が宣言され、冪等の識別子があり、人の判断段階がない場合だけである。判断段階が未解決なら `human-gated` に倒す（安全側）。「無人で動かしたい」という希望を宣言として受け付けない。

この規則は、#1130の一括処理、#2144の締め、#2145の取込み、#2020の提供形態のどれにも同じ形で適用できる。逆に、適用できない操作が見つかった場合は、その操作の意味が足りていない（何が人の判断かが書かれていない）ことを示す。

## 9. 出力契約

既存の読取りコマンド（`tei model catalog`）と同じ二段の型にします。CLIの外側が `ok`／`result`／`error` を持ち、`result` が操作面そのものです。

```console
$ go run ./cmd/tei operation surface -workspace reference/workspace -application order_entry
{
  "schemaVersion": "tei.cli.operation-surface/v0",
  "ok": true,
  "result": {
    "schemaVersion": "tei.operations.operation-surface/v0",
    "effect": "read-only",
    "application": "order_entry",
    "worldFingerprint": "sha256:...",
    "modelRevision": "sha256:...",
    "rule": {"id": "tei.operation-surface/standard-crud", "version": "v1", "fingerprint": "sha256:..."},
    "operations": [
      {"id": "order_entry.list.sales.order", "kind": "list", "target": "sales.order",
       "origin": "derived", "autonomy": "unattended", "effects": [], "basis": "sha256:..."}
    ],
    "gaps": [{"target": "sales.order", "kind": "update", "reason": "missing-semantics",
              "detail": "version attribute or expected-state comparison is not declared"}],
    "conflicts": [],
    "summary": {"operations": 1, "derived": 1, "declared": 0, "unattended": 1, "gaps": 1},
    "authorityLimits": [
      "operation-surface-is-a-derived-read-only-projection",
      "operation-surface-is-not-execution-authority",
      "derived-operations-do-not-create-canonical-elements",
      "autonomy-class-does-not-grant-runtime-authority"
    ]
  }
}
```

型の骨格（[`internal/operations/model_catalog.go`](../../internal/operations/model_catalog.go) の `ModelCatalog` と同じ作りにします）。

```go
// OperationSurface is a read-only, target-neutral derivation over the accepted
// meaning. It is a completeness and navigation aid, not an execution authority.
type OperationSurface struct {
	SchemaVersion    string            `json:"schemaVersion"` // tei.operations.operation-surface/v0
	Effect           string            `json:"effect"`        // read-only
	Application      string            `json:"application,omitempty"`
	WorldFingerprint string            `json:"worldFingerprint"`
	ModelRevision    string            `json:"modelRevision"`
	Rule             SurfaceRule       `json:"rule"`
	Operations       []Operation       `json:"operations"`
	Gaps             []SurfaceGap      `json:"gaps"`
	Conflicts        []SurfaceConflict `json:"conflicts"`
	Summary          SurfaceSummary    `json:"summary"`
	AuthorityLimits  []string          `json:"authorityLimits"`
}

type SurfaceRule struct {
	ID          string `json:"id"`
	Version     string `json:"version"`
	Fingerprint string `json:"fingerprint"`
}

type Operation struct {
	ID            string   `json:"id"`
	Kind          string   `json:"kind"`   // list, read, create, update, retire, ...
	Target        string   `json:"target"` // canonical ID
	Origin        string   `json:"origin"` // declared, derived, overridden
	OverriddenBy  string   `json:"overriddenBy,omitempty"`
	Inputs        []string `json:"inputs,omitempty"`
	Preconditions []string `json:"preconditions,omitempty"`
	Effects       []string `json:"effects,omitempty"` // create:<data>, update:<data>, ...
	Autonomy      string   `json:"autonomy"`          // unattended, attended, human-gated
	Basis         string   `json:"basis"`
}

type SurfaceGap struct {
	Target string `json:"target"`
	Kind   string `json:"kind"`
	Reason string `json:"reason"` // missing-input, missing-semantics, conflict, out-of-scope
	Detail string `json:"detail"`
}
```

契約上の条件。

- 失敗時は `{"schemaVersion": "...", "ok": false, "error": {"code": "...", "message": "..."}}` を返し、成功時と形を変えない（[#2354](https://github.com/hat47x/tei-platform/issues/2354)）。
- `authorityLimits` を必ず載せる。操作面が実行権限や正本でないことを、受け取る側が読めるようにする。
- 出力は秘密情報・業務データの値・内部パスを含めない。識別子と理由だけを返す。
- `gaps` は件数ではなく内容を返す。空配列は「未解決がない」ことを意味し、「調べていない」ことを意味しない。
- `tei commands` の一覧へ `operation surface`（read-only）を追加する。

## 10. 実行時の照合

操作面を実行の根拠にしない代わりに、実行時に次の照合を行う。

1. 実行しようとする操作が、現在の受理済みの意味から導かれる操作面に存在するか。
2. 操作面の `basis` が、現在の意味の照合値と一致するか。
3. 一致しない場合は実行せず、`OPERATION_SURFACE_STALE` として停止する（暗黙に最新の意味で実行しない）。

この照合は、既存の `STALE_SNAPSHOT`（CLI C1）と同じ考え方である。実行のたびに操作面を再導出するのではなく、照合値の一致だけを確かめる。

## 11. 効果仮説と観測（#1973の基準）

新しい抽象を足すので、何が可能になり、どう観測し、効果がなければどう撤回するかを先に決める。

**可能になること。**

- 初めて作業領域を読む人が、`tei operation surface` の一件で「このアプリで何ができるか」「何が足りないか」「どれが無人で走るか」を読める。
- 画面・バッチ・成果物の導出が、同じ規則と同じ照合値から出るため、食い違いを機械で検出できる。
- 自律実行の可否を、コードではなく宣言から判定できる。

**観測。**

| 指標 | 測り方 | 現状の基準 |
| --- | --- | --- |
| 初回読解までの時間 | 作業領域を知らない人が、操作の一覧と未解決に到達するまでの時間 | 未測定（#1168のFirst Value UXと同じ計測方法を使う） |
| gap の検出数 | 既存の参照作業領域で、`missing-input`／`missing-semantics` が何件出るか | 未測定 |
| 実行時 drift の検出 | 意味を変えた後に古い操作面で実行しようとしたとき、停止できるか | 未測定 |
| 自律区分の判定一致 | 人がコードを読んで判断した区分と、導出した区分の一致率 | 未測定 |

**撤回条件。**

- 既存の参照作業領域で `missing-input` が支配的で、操作面が「宣言が足りない」以外の情報を出さない場合は、規則を細かくするのではなく撤回し、操作の一覧は各 Issue の本文へ戻す。
- 実行時 drift の照合が、正当な実行を頻繁に止める場合は、照合を実行の入口ではなく診断に限定する。
- 自律区分の判定が、人の判断と一致しない例が続く場合は、区分を廃止し、`unattended` の宣言だけを残す。

## 12. 既存の設計との関係

| 既存 | この文書との関係 |
| --- | --- |
| [Application Pattern v1](APPLICATION_PATTERN_V1_DESIGN.md) | パターンの五種類を、操作の宣言として読む。パターンの契約（Kind ごとに受理する field）は変更しない |
| [業務副作用の段階・確定後効果・再試行](BUSINESS_EFFECT_LIFECYCLE_DESIGN_2026-10.md) | 操作の効果の実行時規則はこの文書が正本。操作面は「どの操作がどの効果を持つか」を読むだけで、段階を再定義しない |
| [三視点の意味参照と変更影響](CROSS_VIEW_TYPED_REFERENCE_CONTRACT_2026-10.md) | 変更影響の投影はこちら。操作面は影響の計算をしない |
| [Critical Lowering契約](CRITICAL_LOWERING_CONTRACT_DESIGN.md) | 宣言から実行先への接続はこちら。操作面は Lowering の入力候補になるが、置き換えない |
| [全体構成](../../docs/architecture.md) の四層 | 操作面は意味の層に置く。実行先の能力や表示を持ち込まない |
| #951（機能契約の拡張） | 種別を中核の固定列挙値にしない。規則の版で管理する |

## 13. 非目標

- 新しい中核スキーマ、新しい正本、新しい保存形式の追加。
- 画面のレイアウト・項目配置・遷移の決定。
- 実行時スケジューラの実装（#1130の担当）。
- 認可の判定そのもの（既存の `authorization` の担当）。
- 既存の五つのパターンの意味の変更。

## 14. 作業分解

| # | 作業 | 変更する場所 | 完了条件 | 規模 |
| --- | --- | --- | --- | --- |
| W1 | 規則の骨格と決定性 | `internal/operations/surface.go`、`surface_test.go`（新規）。照合値は既存の `workspaceProjectionFingerprints` と `sourceindex.ModelRevision` を使う | 同じ作業領域から二回導出して同じ結果（順序を含む）。`worldFingerprint`・`modelRevision`・規則の照合値が出力に載る | M |
| W2 | 未解決と衝突の分類 | 同上 | `missing-input`／`missing-semantics`／`conflict`／`out-of-scope` を区別して返す正負試験 | S |
| W3 | 宣言と導出の優先順位 | 同上 | `origin`／`overriddenBy` が正しく付き、抑制が来歴に残る試験 | S |
| W4 | 自律区分の判定 | 同上 | 8章の四条件のうち一つでも欠ける `unattended` を拒否する試験 | S |
| W5 | CLI と出力契約 | `cmd/tei/cli_surface.go`（新規）、`main.go` の入口と usage、`cli_contract.go` の一覧 | `tei operation surface` が動き、`tei commands` に read-only として現れ、失敗時も同じエンベロープになる | M |
| W6 | 実行時の照合 | `internal/operations/`、実行の入口 | `OPERATION_SURFACE_STALE` を返す試験。15章の未決が決まるまでは診断に限定する | S |

W1 → W2 →（W3・W4 は並行）→ W5 → W6 の順に進めます。W1・W2 は #1127 の W1・W2 と同じ作業になるため、別々に実装しません。

**実装状況（2026-11-01）。** W1・W2 の参照実装を [`internal/operations/surface.go`](../../internal/operations/surface.go) に追加しました。`list`・`read` の導出、キーの優先順位、未解決（`missing-semantics`・`conflict`）の分類、決定性の試験を含みます。標準操作の `create` と、`update`・`retire` の未解決は、[#1127の設計](DERIVED_CRUD_OPERATION_DESIGN_2026-11.md)のW3・W4に従って同じ実装で扱います。`go test ./internal/operations -run Surface` が通ります。W5（CLI）の読み取り部分を追加しました。`tei operations -workspace <root> -application <id>` が操作面（操作・gap・集計・規則）を JSON で出します。`-out <file>` を付けると、その JSON をファイルへ書きます。書き込みは行わず、実行の可否判定には使いません（15章の未決のまま）。W3（上書き）・W4（書込みの自律区分）・W6（実行時の照合）は未着手です。

## 15. 未決（Owner判断が要る点）

1. 操作面を CLI の読取りコマンドとして公開してよいか（公開面を増やす判断）。代替は、Studio の内部ビューと診断だけに留める案。
2. 操作の種別の語彙を、規則の版として管理することで足りるか、それともプラグインの契約（#951）へ移すか。
3. `unattended` の条件に、環境のサービス主体の宣言を必須とするか（本稿の案）それとも別の権限表現を待つか。
4. 実行時 drift の照合を、既定で有効にするか、明示の指定がある場合だけにするか。

これらは意味と公開面の判断であり、この文書だけでは決めない。決めるまでは、操作面を**診断専用**として扱い、実行の可否判定には使わない。
## 追記（2026-11-03）: 期間の状態

操作面の各操作は、データに期間が宣言されている場合に `period`（期間の識別子）と `periodRejects`（閉じた期間が拒否する操作）を持ちます。自律区分とは別の軸です。

- 自律区分は「誰が判断するか」（`unattended`・`attended`・`human-gated`）。
- 期間の状態は「いつ実行できるか」。

閉じた期間を `human-gated` に読み替えません。両方を持つことで、判断の主体と実行の可否を混同せずに読めます。