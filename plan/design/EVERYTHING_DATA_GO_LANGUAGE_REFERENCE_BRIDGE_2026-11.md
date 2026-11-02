# Everything is Data：既存Go Language Plugin Referenceとの最初の接合（2026-11-02）

- **状態**：既存TEI `reference/dsl-language-plugin` を直接呼ぶ**Goの試験追補（Draft）**。TEI正式DSL/Syntax ABI/Canonical Schema/対象Host実行・Owner意味受理の採用ではない。
- **親**：[Everything is Data #2619](https://github.com/hat47x/tei-platform/issues/2619)、[#2620 Role/自己記述](https://github.com/hat47x/tei-platform/issues/2620)、[#988 DSL/Operator](https://github.com/hat47x/tei-platform/issues/988)。
- **既存合成Reference**：[PythonのSyntax Sugar比較 #2625](https://github.com/hat47x/tei-platform/pull/2625)（自作変換・23件の局所PASS）、[Script-first Studio #2617](https://github.com/hat47x/tei-platform/pull/2617)（14件の局所PASS）。本試験はこれらを製品PASSへ格上げするものではない。
- **次段の横断計画**：[Draft #2629](https://github.com/hat47x/tei-platform/pull/2629)、Studio open-world [Draft #2631](https://github.com/hat47x/tei-platform/pull/2631)、Function mapping [Draft #2624](https://github.com/hat47x/tei-platform/pull/2624)、Provider read [Draft #2628](https://github.com/hat47x/tei-platform/pull/2628)。
- **追加ソース**：[`reference/dsl-language-plugin/everything_data_cross_surface_test.go`](../../reference/dsl-language-plugin/everything_data_cross_surface_test.go)。

## 1. 既存の「本物のGo Reference」を使う

2026-11-02のmainから直接確認した既存実装：

| 実コード | 実際に持っている限定能力 | この試験で再利用すること |
| --- | --- | --- |
| `selection.go` | `Selection`／`Resolve[T]`／`ParseSelected[T]`。Languageと実装fingerprintの**既受理と仮定されたexact pin**に従う。Candidate返却はAcceptanceでない | 二つのテスト用Language callbackからどちらを使ったかを区別する |
| `pipeline_syntax.go` | `ParsePipelineSyntax`のUTF-8 **byte SourceSpan**を持つpipe記法 `rows/order |> std.where |> std.sum` | 構文→Source位置の検証 |
| `pinned_lowering.go` | `LowerPinnedPipeline`で各Operatorを固定契約と照合。型連鎖が不正なら診断し、元Spanを維持する | 同じ候補を`OperatorID/Version/Fingerprint/Input/Output/Effect`を含む**テスト用データ投影**へ渡す |
| `operator.go` | `ResolveOperator`のID/版/fingerprint/型・Effect検査。E2 Referenceは`pure`に限定 | 型不一致・旧版→新版暗黙置換・Effectの不正なpure昇格を拒否 |
| `pinned_composition.go` | `ComposePinned`で固定契約を照合した順序付き構成 | 将来、Source→Function構成→実Targetへの追補に再利用 |
| `internal/semantic/model.go` | 既存Canonical `Namespace/Data/Type/Relation/Validation`等の**実在する構造** | 新しい標準`PlainData` Core型と混同せず、正式Semantic解釈を既存Ownerへ戻す |

Test-only JSON adapterを明示的に用意し、Go本体の`ParseSelected`に通す。ただし、このJSON adapterが本番利用できるLanguage Pluginとして解決された、とは主張しない。二つの`Language Selection`は**テストに与えた固定値**で、現行WorkspaceのAccepted Lockではない。

## 2. P0の検査対象

一つの純粋な架空Pipelineに限り、2表層：

A（既存Parser）：
```text
　rows/order |> std.where |> std.sum
```

B（Test-only JSON adapter）：
```json
{"input":"rows/order","operators":["std.where","std.sum"]}
```

から得た`PipelineSyntax`（SourceSpan付き）に対し、同じ**外から固定した**Operator契約を通す。同形の`fixturePlainCandidate`に`Role=function`、input ref、各段のOperator ID/Version/Fingerprint、Input/Output Type、Effectを**データ値として投影**する。

このデータ構造は**正式内部Normal Form・新Core型ではない**。原Sourceのbyte fingerprint、span、表層Plugin、Meaningの受理決定、Target実行資格は別に保持する。等しいProjectionのJSONは**同じ限定候補データ**の観測であって、Source同一・意味同値・業務的な受理や公開許可を意味しない。

### 実際に検査する7ケース

| ID | 判別 |
| --- | --- |
| EG-01 | Source A/Bを別の固定Language実装から解析し、**実Go LowerPinnedPipeline**へ通した場合に同じ候補構造を得る。Source原文hashは別であり、byte Spanは両方正しく各Tokenを指す |
| EG-02 | Acceptedと仮定したOperator v1が消え、Catalogがv2になっても**自動upgradeしない** |
| EG-03 | 同じ名前・型でもEffect=writeなら**pure Pipelineへ誤昇格しない** |
| EG-04 | 上流Outputと下流Inputの型が合わなければ拒否 |
| EG-05 | JSON実装のLanguage exact pinがProviderにない場合、他のLanguageへフォールバックしない |
| EG-06 | Test-only JSON記法で未知`execute`属性があれば、そのまま実行要求を許可しない |
| EG-07 | 不完全なpipe Sourceを拒否 |

EG-01の投影で**Operator meaning law、filter条件、集計Grain/NULL/Bag/順序/Timeは証明できない**。現行`OperatorContract`に当該情報が存在しない範囲では、これらについて同値判定を主張せず、#2620/#988/#2593/#2622の本体試験へ残す。Predicateなしの`std.where`は**本番用Query定義ではなく参照用の型連続性試料**である。

## 3. 欠ける実証をどこへ返すか

- **Syntax／Role**（#988/#2620）：意味を持つ引数・パラメータ・scope/lexical binding・Source span・版、Function/Model/TestのRole適用。今回のSourceにある`role=function`はテストに設定した表示ラベルである。Sourceから自動Role受理していない。
- **Data Context／Scope**（#2506/#2624）：`rows/order`というType文字列だけで、Relation Path・必要Field・全件集合・Identity・観測完全性を推測しない。
- **Data Operations**（#2622/#2387）：`where/sum`のRDBやAPIへのpushdown、NULL/重複/粒度/Measure law・native SQL Source・実Provider Read witnessは別の契約を検証する。
- **Studio／SHO**（#1124/#1824/#2621）：異表層Sourceの編集位置・Undo・Source locator再解決、Partial/Unknown/Disclosure、説明表と実定義の区別、Source→Preview→Acceptance→reparseは未検証。
- **実行／Target**（#1121/#1174）：pure固定候補の説明だけで実Binary/Host実行、Action権限、Receiptを発行したことにならない。
- **AI-native DSL**（#988/#2605）：Goa風/MukuroL風/Markdown/Typed DOT等のエージェント生成・局所修正の質を比較する。今回の2形式＋7テストではAIのfirst-pass成功率は未測定。

## 4. 統合受入へ渡す順

1. **E0**：本Goテストの正規Repo環境での `go test ./reference/dsl-language-plugin` と静的診断・保存Blobを確認。Python23件と本Go test7件は**別試験**として報告する。
2. **E1**：#2620のRole/Operator ID/Versionを本物の既存Semantic Resolutionへ束縛。実Source ID/Namespaceと意味Constraintを確認。
3. **E2**：#2624から明示的Function Data Context Mappingを同版で取得し、実QueryとCollection completeness/#2628 Read witnessへ接続。
4. **E3**：#2622/#2598でData Operations→型付きAction/RequestとProvider実呼出しを検証。既存#1174の現在Grant、実Effect/Receipt/独立Readを別資格とする。
5. **E4**：#1124/#2631のStudio標準UI、SHO/AIの二Consumerで**実際に読み取ったSource/Artifact/版と開示**が揃うことを確認。Manifestや生成コードの一致だけでReader成功を推定しない。

## 5. 合格資格

- **書込時点**：新規Go試験追加・Planを保存しただけでGoテストがPASSしたとは扱わない。CI/ローカルで結果が得られた場合のみ、exact branch/commit・実行物・環境と結び報告。
- **Go Referenceの限界**：既存`reference/dsl-language-plugin`は部分的な実装であり、正式Language Plugin ABI/Target runtime/Canonical Acceptanceへの本番統合を意味しない。
- **研究の自由度**：新しいSyntax/Operator/Domainの並列試行を継続し、Coreや専用UIの新種は必要性が反証試験で明確になった部分を採用する。固定の候補数上限を設けない。

**成果の目的**：Everything is Dataの抽象計画を、既にあるTEI Go製品コードに1段階だけ接近させる。実装を複製せず、何が接続できたかと未成立の保証を同時に記録する。

## 6. 2026-11-02 EG-08/09追加：実Semantic接合とData ID同一性の限界

先行EG-01〜07はPipeline構文とexact-pinned Operatorの**型連鎖に限定した同値**を観測する。以下の二試験を、実`semantic.Namespace`への接合と、意味を取り落とす危険の明示的な負例として追加した。

| 試験 | 具体的な実Go接合 | 主張しないこと |
| --- | --- | --- |
| **EG-08** | `ParseSelected`に、実`syntax.NewParser(...).ParseSemanticFile()`とtest-only JSONの二実装を渡す。`sales.order`とそのAttribute Typeが両方で一致する場合のみ`BindPinnedSemanticInput`から同じ型結果を得るか確認。 | JSONが正式言語として登録されたこと、Semantic WorldがOwner受理されたこと、Predicate/Query/Host実行が一致すること。 |
| **EG-09** | 同じ`sales.order` IDのまま`sales.order.amount`の型を`std.decimal`→`std.string`へ変更し、`BindPinnedSemanticInput`の結果型だけは一致し得ると示す。 | **同じData IDと同じOperator Pinだからモデル意味は不変**、あるいはキャッシュ/Consumer資格の再検査が不要だと判断すること。 |

EG-09は既存Binderの責務不足を**即座に実装バグ・新API要求**とみなすものではない。現BinderはData IDの一意性とOperator契約の型連鎖を確認する限定Referenceで、全属性の意味・Role/Context・Accepted Revisionを確認する責務は定義されていない。二つの`Namespace`値は完全なWorkspace World/正式Validationの通過を意味しない。

**横断計画への成熟差分**：`Syntax/Role→Model (Attribute/Relation/Rule)→Function Context/Query hidden dependencies→Provider Route→Operation/Artifact→Consumer Loaded Bytes/Disclosure`を、結果型やData IDではなく、用途に必要な**依存閉包と実証された各version/epoch**で接合する。`#2629`のQDC-01〜10、`#2636`のF1/S1再利用選択、`#2638`のRole改版影響にEG-09を引き渡す。モデル全体変更だから必ず全Artifact無効、あるいはIDが変わらないから再確認不要、のどちらも無条件の正解としない。出力されないWHERE/ORDER/JOIN等の依存については`#2634`で別に検証中。

**試験資格**：EG-08/09の追加後、計**9件のGoテスト**。追加前head `5be242b664e1cb77d4af2f97930e52a7681fe480`ではGitHub Actions Repository Health successかつ `go test -count=1 ./...`ログ内の `tei/reference/dsl-language-plugin` PASSを確認したが、それは**旧7件が載る版のEvidence**。新headでのGo/CI成功は再確認後にのみ表明する。Python23件/JS19件等、別Subjectの局所PASSを1件の横断E2Eへ加算しない。
