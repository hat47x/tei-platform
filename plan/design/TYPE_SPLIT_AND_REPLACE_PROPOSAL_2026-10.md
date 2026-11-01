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

**P0停止条件（2026-11-01、最小実装をDraftに保存）**：複数先への割当を含むSplitは、明示Reassignment・未割当ゼロ・移行先の型同一性・全影響が検査可能になるまではCandidate生成を**拒否**する。`-split` と `-candidate-out` の併用を拒否する小修正・オプション検査テストをこのDraftに追加した（新規Go実行は未実施）。読取り専用のImpactは維持する。将来は専用のSplit→候補変換を採用しても、既存のShared AcceptanceとAuthoritative Sourceを再利用し、第二のPatch DSL／Approvalを作らない。

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


## 9. 2026-10-31 横断追補：参照の付替えと「統合完了」を混同しない

**状態：現行mainの `tools/reference-type-reorg-impact/impact.go`、`candidate.go`、`internal/semantic/model.go` および現行Issueの静的照合に基づく追加設計。今回Go/CLI・Studio・実DBはNOT RUN。**

### 9.1 追加で判明した到達点の差

- `Analyze` は統合元型を使うAttributeだけでなく、型を直接参照するCapability Input/Outputと、統合元を基底とするAlias型を**影響一覧**に出す。
- 一方、`Candidate` が生成する `set-attribute-type` は **`Impact.Attributes` のみ**を列挙する。Capability Input/OutputのTypeRefは書き換えない。Alias型が存在する場合、Candidateは拒否する。Source Type定義自体を削除・改名する操作も生成しない。
- したがって、候補のShared Acceptanceで属性のTypeRef付替えが受理されても、「旧型を参照する全利用先が移行済み」「旧型を削除可能」「意味型の統合完了」を自動的に意味しない。既存の`set-attribute-type`のaccept/reload合成試験は、属性の変更の成立範囲として保持する。
- `physical-storage-differs` の現行実装は、**統合元の型を使用している属性のStorage集合間**の差異を検出する。統合先の型を使用しているAttributeのStorageをその比較集合に加えていない。そのため、この診断が無くても、移行元と移行先の全使用箇所が同じ物理表現・値域制約を満たすとは言えない。これは参照用途の観察範囲に関する注意で、実DBのMigrationが必要になると確定したという主張ではない。

### 9.2 型ライフサイクルの実用的な区分（正式Enumではない）

1. **型参照の候補**：元型→先型への使用箇所単位の変更案。未受理の提案。
2. **使用箇所の付替え**：Attribute等、変更対象とした参照について既存Acceptanceを通過。別の利用先は残り得る。
3. **利用先の整理**：Capabilityの入出力、Alias、Validation、Key/Relation、外部Workspace等について、移行・互換維持・未解決の区別を表示。全利用先の自動検出を保証しない。単にリストが空だから外部参照が不存在と断定しない。
4. **旧型の廃止判断**：参照・互換性・履歴の処遇と受理を確認した場合のみ。名称変更、Canonical ID置換、型定義削除を別の演算として扱う。旧型を残す選択も正当。
5. **物理適用**：データやスキーマに影響する場合は別のProvider/Migration/Target資格で観測。意味の受理だけでDBが更新済みとしない。

上記は設計者が理解できる**操作の進行状況**を説明するものであり、新しいCore状態enumや型Registryの導入を要求しない。

### 9.3 最小の反例と受入票（すべて提案・NOT RUN）

| ID | 既存モデルと操作 | 必要な判定 |
| --- | --- | --- |
| TM-01 | 型Xを参照するAttribute AとCapability入力Pが存在。AだけをX→Yに付替え | Shared AcceptanceのA変更を確認しても、Pの型はXのまま。「統合完了」と表示しない |
| TM-02 | XのAttributeは1件だけでStorageはvarchar、YのAttributeには別Storageがある | 現行`physical-storage-differs`無しを「元／先の格納互換PASS」としない。比較できたScopeと未評価を明示 |
| TM-03 | 旧型Xを基底にしたAlias Zが存在 | Candidate拒否の既存境界を維持。Zを黙ってYの派生型へ付替えない |
| TM-04 | すべての現行Attribute参照を付替えたが、旧型X定義と外部Consumerがある | 内部参照移行と旧型廃止判断・外部互換性を分離 |
| TM-05 | 同じ型IDの下で値の制約を厳しくする構想、既存値適合・運用時点が未検証 | 型の参照一致／名前一致のみで値適合と運用可能性を認定しない |

### 9.4 ユーザーが最初から厳密な型管理をしなくてよい編集体験

Studioはまず**基本型を指定する通常の項目定義**を許す。意味の共通性に気付いた段階で、既存TypeRef（意味型）を参照する変更案と利用先への影響を提示する。人による同一性判断を経て、項目単位に段階的に共有へ移行できる。

- 初回から全項目のSemantic Typeを設計する義務は置かない。
- **Domain**は対象領域、**Value Domain**は値域、**Semantic Type**は共有する一つの値の意味、という名称案を維持する（正式な名称採用はOwner判断）。
- 現行`semantic.Type`には長さ・精度・文字集合等の汎用制約を直接持たない。初期Studioが`VARCHAR(10)`等の全値制約をCanonicalで保持できるかは未実証であり、実装済みとして提示しない。
- 値の制約・利用先の必須性・参照先の存在や状態・実DBへの書込み条件は別責務。Smart DataのRuleはData側の所属とFunctionの明示拡張点で検討されているが、FunctionのBind/Precheck成功は全更新経路・Commit後不変条件の保証ではない（#1129/#1174）。
- 共有の増加に伴う影響解析は、正本のTypeRef・依存グラフ・Source revisionを利用し、表・ER・Query Explain・SHO文書は同じAccepted Sourceから投影する。SQL/CTE別名の一致をTypeRef同一性の根拠にしない（#2387）。

**次の最小実証はTM-01とTM-02を既存Referenceテストで判別すること**。型の全利用先を常に閉世界で列挙できるという前提や、旧型定義の廃止機構・新しい共通型Registryを先に実装しない。

**Owner判断待ち**：Studioで「型参照変更」「使用箇所の移行」「統合・廃止」の言葉をどう表示するか、旧型の互換維持にどの範囲のConsumer確認を要するか、Value Domain制約を既存Validation/Pluginでどの程度まで表現するか。

## 10. 2026-11-01 追補：変更意図を失わない変換と、閉じた検証範囲

**位置付け：#1806 のDraft PR #2404での限定的コード修正と、他チャットの #2387 SQL Scope／#1570 外部Semantic／#1935 Shared Acceptance の責務比較。以下のGoテストと実CLI動作は、執筆時点では未実行。正式な型仕様やOwner受理は未確定。**

### 10.1 Impact→Candidateの変換は「変更意図」の保存を条件にする

分割X→{Y,Z}の宣言を考える。意味型の基底型が同じでも、A→Y・B→Zという**割当先の対応関係**を失えば、型検査がPASSしても元の変更意図とは違う。

- 入力：固定されたSource版、対象TypeRef、対象のAttribute identity、移行先候補集合、項目別の意図された割当。
- 出力候補の成立条件：**入力で明示したAttribute集合と、候補の変更対象集合の一致**、各属性の元・先TypeRefの保存、未割当・余剰・重複・別Scope混入が無いこと。意図して対象外にした項目は対象外と明示し、「未発見」を「存在しない」に昇格させない。
- Source→Impact→Candidate→Preview/Acceptという段階が一致していても、その段階で扱っていないCapability直接参照や外部Consumerの契約まで「型統合済み」とは判断しない。
- 上記は既存 `structureRevision` と Shared Acceptance の `baseModelRevision` の**意味の違い**も保存することを要請する。型の使用先の構造版が固定されていても、直前のSource受理対象版は別に再照合する。

この考え方は #2387 のSQL列解決と共通する。そこでは、同じ列名・aliasであってもScopeとSource occurrenceを保存して初めて正しい参照となる。同様に、型の分割では同じ基底表現でも属性別の割当先を保存する必要がある。ただし、Query IRやSQL Scopeを型管理の第二正本にしない。

### 10.2 最小実装と、実際に確認すべき対象

今回のDraft PR #2404には、`cmd/tei-type-reorg-impact/main.go` の入力境界で `-split` と `-candidate-out` の併用を拒否する条件を追加した。`main_test.go` にオプションの4ケース（読取り専用split、候補なしmerge、merge候補、split候補拒否）を追加し、READMEには分割の読取り専用契約を明記した。

| ID | 確認する入力と出力 | 完了条件・証拠境界 |
| --- | --- | --- |
| TS-02a | `-split -candidate-out bad.json` | 非0終了、診断、bad.jsonを作らず、Workspace不変（要実CLIの負例） |
| TS-02b | `-split`だけ | 影響JSON・未割当診断を保持、候補出力なし（既存機構の退行確認） |
| TS-02c | `-candidate-out`で通常の統合 | 従来どおり候補を生成、既存のShared Acceptanceへpreview可能（退行確認） |
| TS-02d | `-split -candidate-out`で実際のWorkspaceが存在しない | 実Workspaceの読取りや書込みに入る前に組合せ拒否（入口順の確認） |

- **コード保存済み**と**GoテストがPASSした**は分ける。手元のGo 1.23.2では、現行`go.mod`指定のGo 1.26.10を満たさず、trusted exact checkoutでの `gofmt`、`go test ./tools/reference-type-reorg-impact/... -count=1` と実CLI実行が残る。
- この修正は**候補生成の誤変換の入口を閉じる**だけで、正しいSplit候補の実装、Studio上の操作、意味型のOwner受理、実DB値適合、外部Hostの型互換性まで合格にしない。
- 将来のSplit→Candidateが必要になったときは、TS-01・TS-03と、入力割当↔候補割当の完全性を独立した固定fixtureで実証してから入口を開く。新しい変更案スキーマや型Registryの採用は先行しない。

### 10.3 Domainを横断する再利用の説明

- **Domain＝対象領域**の共通用語と、**Semantic Type＝意味型**／**Value Domain＝値域**の用語候補を維持する。Contextや外部Namespaceの境界は、型の名称・値域一致から意味的同一性を自動認定する根拠ではない。
- #1570の外部Semantic参照では、どのNamespaceを明示的に選んでいるかが分かる範囲を確認し、その範囲外のConsumerを「存在しない」と断言しない。閉じたReference fixtureは閉じた範囲で合格判定してよい。
- #2077の他言語への生成、#2390の外部製品によるPlugin利用は、型の構造適合・必要な契約・実行側の権限を個別に検証するレーン。型の改名・付替えだけで生成済みJava/.NET等のartifactや外部Hostの更新・実行許可を得ない。

**次の優先順位**：まず上記CLI負例と退行をexact branchで実測（#1806）。次にTM-01/TM-02の局所使用先・物理Storageの観測範囲を明確にする（#1806）。最後にStudio上の段階的な意味型化を#1124へ接続する。値域制約のCore表現、分割先の人の意味判断、旧型の廃止、複合値第一級型化はOwner判断・実例待ちのまま維持する。

## 11. 2026-11-01 追補：CLI入口だけでなく、Impact→Candidate内部APIでも誤転写を防ぐ

**背景と新しい反例**：§10のCLI入口拒否を追加した後、`SplitImpact` が返す `Impact` を他の呼出元が直接 `Candidate(impact, revision)` に渡せることを確認した。CLIのフラグ検査はその呼出経路を通らないため、同じ誤転写問題が再発し得る。

`SplitImpact` は既存の `Impact` の表示形状に合わせ、`Impact.Merge.Into` に分割先の**先頭**を置き、属性別の移行先は `Attributes[].Target` に保持している。一方、`Candidate` は統合専用のため `Merge.Into` だけを読んで変更候補を生成する。**同じJSON形状を共有できても、統合・分割の操作意味は異なる。**

### 11.1 今回の限定的修正（Draft #2404）

- `SplitImpact` に `split-candidate-unsupported` の診断を常に付与。既存の読み取り専用分析の結果・属性別の割当・キー/Validation/Capability影響は引き続き表示できる。
- 統合専用 `Candidate` は既存のblocking診断の仕組みを使い、`split-candidate-unsupported` が含まれれば拒否する。これにより、CLI外からの内部API直接呼出しも拒否対象にする。
- `candidate_test.go` に、属性別の `Reassignment` が指定済みでも `SplitImpact→Candidate` が拒否される負例を追加。現行の通常Merge→Candidateの正例は既存試験を再利用する。
- 新しいCore kind、型Registry、Shared Acceptance契約、外部Hostは導入しない。既存 `Impact` の表示互換は維持し、専用の分割用候補変換が成立するまで候補化しない。

### 11.2 実測・設計の境界

| 観点 | 今回の変更と検証対象 | 未保証 |
| --- | --- | --- |
| CLI入口 | `-split -candidate-out` をWorkspace読込・出力前に拒否。CLIビルド・起動の負例を追加 | 将来のSplit候補の受理 |
| 内部API | `SplitImpact→Candidate` の拒否診断と負例テストを追加 | 任意の利用者が診断を改変したImpactまで安全な型付けをすること |
| 既存正常経路 | 読取り専用SplitImpactと通常のMerge Candidateのテストを維持 | TypeRef変更後の実DB、外部言語・プラグインの意味保存 |
| 証拠・受理 | Candidate生成の可否と、正本のShared Acceptanceを別段階に保持 | Owner意味受理、旧型廃止判断、Provider Migration |

**既存の実測**：Draftの先行コミット `7beb2e4ba18737a2d2f3ae6050fe24f6b7e2a2e6` に対するGitHub Actionsの「TEI PR・main 健全性確認」および依存脆弱性確認は `success`（2026-11-01取得）。この事実はそのコミットの変更範囲についてのみ有効。

**追加変更の現状**：CLIビルド・起動を伴うテスト、および内部API直接呼出し拒否のテストはこの追補で新規追加した。新しいexact headのGo/CLI/CI結果は**まだ未確認**。新しいテストの定義をPASS済みと扱わず、latest headのCI終了と、CLI負例の実行ログを照合する必要がある。

### 11.3 次の採用判断

最小停止条件の実証後、分割先を個々のAttributeへ割り当てる操作をStudioで実現する場合は、「Source固定版＋Attribute identity＋明示Reassignment＋移行前後のTypeRef対応＋影響先」の保存を第一条件とする。正本への変更は既存Shared Acceptanceを利用する。

複数Targetの割当てがすべて明示されていても、型の意味の同一性・値域互換・外部Consumer適合・旧型廃止は個別の判断である。意味型の利用範囲を追加で観測できた場合、必要に応じて影響一覧の範囲を拡張するが、「世界全体を調べた」という抽象的な保証は要求しない。

**Owner未決**：本Referenceの `split-candidate-unsupported` を恒久的な公開診断に昇格させるか、将来の専用Split候補変換に合わせて削除するか。現段階はReference限定の暫定的な意味区別である。
