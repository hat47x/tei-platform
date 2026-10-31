# 意味の型の分割・置換を変更案として扱う（#1806）

ステータス: **設計案（PoC）**。型の分割（1 型 → 複数型）と置換（rename）を変更案として扱う最小経路を定める。物理型の変更は既存の移行の安全境界を通る。#1806 の Issue 本体に従う。

## 1. 目的

旧モデラーは型の統合（項目を別の型へ移し、元の型を削除）を許していた。TEI には意味の型を扱う基盤があるが、**意味の型の再編・統合と、その影響**は担当がない。#1806 では型の統合・分割・置換を変更案として扱い、影響（キー、関連、導出規則、Validation、Capability）を示す。

本設計は、統合に続く 2 操作（分割・置換）の最小経路を定める。統合は `reference-type-reorg-impact` で影響を列挙し、`set-attribute-type` 候補を既存 Shared Acceptance へ渡す参照経路がある（PR #1993）。一方、分割は影響一覧の参照経路に留まり、複数の移行先を指定した正しい候補生成・受理は未実証。型定義そのものの削除・改名も未実証である。

## 2. スコープ

- 型の分割: 1 型 → 複数型。属性の再割り当てを変更案として扱う。影響を列挙する。
- 型の置換（rename）: 型の Symbol / ID を変更する。Canonical ID が変わるため、参照の付け替えを変更案として扱う。
- 物理型の変更と意味の型の変更を分ける（物理型変更は移行の安全境界を通る）。

## 3. 効果仮説と観測（#1973）

- 従来の限界: 型の統合は影響を列挙するのみで、分割・置換の経路がない。型の再編が手作業になり、参照の整合を守れない。
- 効果: 型の分割・置換を変更案として扱える。影響（キー、関連、導出規則、Validation、Capability）をプレビューできる。受理前は正本が変わらない。
- 観測: 1 件の型を 2 型へ分割し、影響を列挙できる。1 件の型を rename し、参照の付け替えを変更案として出力できる。
- 撤回条件: 受理境界（#267）の変更案形式が変わり、より直接の経路ができた場合は廃止する。
- 未確認: 分割後の属性の再割り当ての UI、rename による canonical ID 変更の影響範囲、受理後の再投影。

## 4. 分割（split）

入力: `From`（1 型）と `Into`（複数型）と、属性の再割り当て（どの属性がどの型へ移るか）。

- 影響は `reference-type-reorg-impact` と同様に列挙する（属性、キー、関連、Validation、Capability、Alias 型、物理ストレージ）。
- 分割は意味の型の変更であり、物理型の変更は別扱い（移行の安全境界を通る）。
- 変更案の operation: 既存の `add-type` + `set-attribute-type` の組み合わせで表現する。新しい operation 種別は追加しない（未確認）。

## 5. 置換（replace / rename）

入力: `From`（旧型）と `Into`（新型）。

- `From` のすべての参照を `Into` へ付け替える変更案を生成する。
- Canonical ID が変わる場合、参照の付け替えが正本の意味を変えるため、受理境界で人間の判断を要求する。
- 変更案の operation: `set-attribute-type` の繰り返し + 型の削除（未実装の operation）。現時点では影響の列挙のみ。

## 6. 受入条件

- [ ] 分割前に、影響を受ける属性・キー・関連が列挙される
- [ ] 置換前に、影響を受ける参照が列挙される
- [ ] 受理前は正本の型定義が変わらない
- [ ] 物理 DB の型変更は既存の移行の安全境界を通る

## 7. 未確認・境界

- 分割後の属性再割り当ての UI
- rename による canonical ID 変更の影響範囲
- 受理後の再投影
- 旧ツールの実行規約や画面操作を、TEI の意味の正本へ持ち込まない（#1802 の境界に従う）

## 8. 2026-10-31 横断照合：既存実装の到達点と必要な負例

**状態：現行mainの該当Goソースと既存PRを読んだ静的レビュー。今回のGo/CLI/実DB試験は NOT RUN。下記は設計・実装の改訂提案であり、正式な仕様の受理ではない。**

### 8.1 現在あるもの／まだないもの

- **統合：参照経路あり。** PR [#1993](https://github.com/hat47x/tei-platform/pull/1993) は main に統合済み。既存 `set-attribute-type` は、対象Attribute IDを維持し、同じ基底表現に解決される型参照へ変更する。scratch WorkspaceでPreviewの各段階、AcceptでのSource変更と再読込を確かめるテストも存在する。これは**参照実装の限定的受理**であり、Studio製品体験、実DBの値適合・Migration、異なる型の意味同一性を保証しない。以前の「型参照変更の受理演算が未実装」という記録は、この到達範囲では古い。
- **統合：影響と候補の区別あり。** `tools/reference-type-reorg-impact/impact.go` は属性、Alias、Validation、Capability、Key、Relation等を列挙する（PR #2006/#2007）。`candidate.go` は**統合の候補**を `set-attribute-type` の列へ写す。影響の読取りが成功しても、人の意味受理や物理変更は自動実行しない。
- **分割：read-onlyに留まる。** `split.go` は `Split.Reassignment` を内部入力として持ち、未割当や不明な移行先を診断する。`cmd/tei-type-reorg-impact/main.go` の `-split` 入口は `-from/-into` だけで、属性ごとのReassignmentを指定できない。さらに `-candidate-out` は統合／分割で共通経路を通る。
- **値制約：未表現を表現済みとしない。** 現行 `internal/semantic/model.go` の `Type` は `ID/Symbol/Kind/Underlying` の意味型参照・Aliasを保持するが、桁数、文字集合、decimal precision/scaleなどを直接定義しない。`internal/database/persistence/model.go` の `ColumnMapping` は `Storage/Nullability` などを保持するが、ここに上記すべての値制約があるわけではない。従って「文字列(10)を意味型へ昇格」のUI構想を、**既存Canonicalの値域制約まで実装済み**とは見なさない。検証条件の所有先は既存Validation／Provider／標準Pluginの実証を経て決める。

### 8.2 最優先の負例：SplitをMerge用Candidateに誤変換しない

現行Sourceを静的に追うと、`SplitImpact` が返す `Impact.Merge.Into` は移行先配列の**先頭**であり、属性別移行先は `Use.Target` に入る。他方、共通 `Candidate(Impact,...)` は**すべての属性**を `Impact.Merge.Into` へ書き換える。しかも `Candidate` のblocking診断集合は `split-attribute-unassigned` を含まない。

そのため `-split -candidate-out` を単なる書込み候補とみなすと、**「AはType1、BはType2に分割」したい意図が「AもBもType1」に失われる危険**がある。型の基底表現まで異なる例なら後続Previewが拒否し得るが、二つの移行先の基底表現が同じ例では、技術的な型検査だけで誤配分を見逃し得る。

**P0停止条件（修正案）**：複数先への割当を含むSplitは、明示Reassignment・未割当ゼロ・移行先の型同一性・全影響が検査可能になるまではCandidate生成を**拒否**する。最小修正として `-split` と `-candidate-out` の併用を拒否し、読取り専用のImpactを残す。将来は専用のSplit→候補変換を採用しても、既存のShared AcceptanceとAuthoritative Sourceを再利用し、第二のPatch DSL／Approvalを作らない。

| 試験候補 | 入力 | 期待 |
| --- | --- | --- |
| TS-01 | 属性A・Bの旧型Xを、同じ`std.string`に基づく新型Y・Zへ分割。A→Y、B→Z | 各属性の指定を保持する。統合用Candidateへの写像は不可 |
| TS-02 | CLIで`-split -candidate-out`を指定し、再割当指定なし | **誤った候補を出力せず、明示拒否**（修正するまで未保証） |
| TS-03 | 分割先Yは存在するが、Bの行き先が未決 | `split-attribute-unassigned`を成功・受理可能へ昇格しない |
| TS-04 | 型の基底表現は同じだが、片側の値制約が厳しくなる | 値適合は未確認。型参照変更PASSをDB値適合・移行不要の証明にしない |

上記は**未実行の試験設計**であり、現在のCLIが安全に拒否するという主張ではない。既存 `split_test.go` は`SplitImpact`の一部を検査するが、CLIの`-split -candidate-out` 経路をこの票の意味で実証したものではない。

### 8.3 計画・用語の責務境界

- **Domain** はTEIの対象領域を表す語として確保し、単一項目の型を単に「ドメイン型」と呼ばない。**Semantic Type（意味型）**は名称候補、**Value Domain（値域）**は許容値の範囲。前者の同一性、後者の包含、物理格納の互換性は独立判定（[既存#1806追補](https://github.com/hat47x/tei-platform/issues/1806#issuecomment-6536808156)）。
- **#1806**：型統合／分割の影響と候補、型の利用先。**#1935/#267**：共有受理と更新系演算。**#1124**：Studioの編集・差分・再投影。**#1126／既存Migration**：物理値・格納。**#1705/#2056**：Decimal・NULL/missing。コード名前空間やStudio GUIだけの第二の意味正本を作らない。
- **表示名の変更／型Refの付け替え／Canonical IDの付け替え**は別の操作として説明する。単なるSymbol変更をIDのrenameと呼ばず、ID変更は参照の付け替えと下流影響を要する。既存`set-symbol`と`set-attribute-type`の成立範囲を超えた「rename完了」を宣言しない。
- 型統合で物理格納差がないという診断は、`Persistence`の**既存AttributeごとのMapping**についての静的観察に限る。Target型自身に唯一のStorageがあるとみなさず、SQLの実データ適合、文字長・scale、Provider差、既存値検査までPASSにしない。意味変更から実データへの適用は[既存の移行設計（評価待ち）](SEMANTIC_CHANGE_DATA_EVOLUTION_DESIGN_2026-10.md)を参照する。

### 8.4 完了条件と撤回条件

1. まずTS-01〜03の**誤配分防止**をGo/CLI負例で再現し、影響表示を失わない最小拒否を実証する（対象：#1806）。割当指定と候補生成の拡張はその後の独立判断とする。
2. TS-04では静的モデルの型参照変更と、値制約・既存データの観測を区別し、未表現の制約は`unknown`へ残す。全DB/全Targetの検査を最初の受入必須条件にしない。
3. 型の分割に対する新しい汎用契約が不要で、既存の更新系操作と明示割当で安全に表せるなら、新たなType Registryや専用Approval Engineを導入しない。追加抽象が効果を示さなければ撤回する。

**Owner判断待ち**：Semantic Typeの正式名称、値域制約をどの既存契約/Pluginで表すか、Splitの配分確定時に必要な人のレビュー範囲、値適合が未確認なモデル改版の採用・配備停止の区別。
