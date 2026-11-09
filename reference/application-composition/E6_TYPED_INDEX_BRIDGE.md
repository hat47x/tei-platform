# E6：解析済み参照とSource Index変更検査を接続する

- **状態**：読み取り専用の接続試作（2026-10-10）。正式なApplication Composition Resolver、業務意味の受理、実行権限、第二のPlugin Lockではない。
- **管理**：[Issue #1016](https://github.com/hat47x/tei-platform/issues/1016)／[Draft PR #1051](https://github.com/hat47x/tei-platform/pull/1051)。
- **前提**：[E5](E5_PARSED_WORLD_SOURCE_JOIN.md)で既存のGo Loader・型付きWorldから限定的な参照集合を抽出し、[E4](E4_SOURCE_INDEX_JOIN.md)でSource Indexに記録されたSubjectの変更を検査した。この二つの結果を手入力の依存一覧なしで結ぶ。

## 具体的に接続したもの

[E6接続試作](e6-typed-index-bridge.mjs)は、E5のJSON形式のReportを変更前後の二つ受け取り、対応するSource Index二つと照合する。

1. E5 Reportが対象Applicationを一つ特定できており、宣言・Source所在の照合に未解決事項がないことを要求する。E5側の`observed`をそのまま全面的な適合へ読み替えない。
2. Application・Semantic Namespace・Application Pattern・PatternのData／Relation／Capability／Application所有Query・Validation・論理Database・Database Contract・Database Contract Queryの既知の直接参照種類だけを対象に、Report内の定義SourceとSource Indexの`definition`参照、Artifact指紋が一致することを確認する。出典欠落・重複・取り違えは停止する。
3. 変更前後の対象Subject集合が変わった場合は`declared_dependency_set_changed`とする。新たに追加されたValidation参照を、旧セットに含まれないという理由で見逃さない。
4. 集合が一致する場合は、E4の既存`assessIndexedSubjects`へSubject集合を渡す。E6独自のSource指紋比較器は作らない。
5. 結果が同一でも`limited-unchanged`と返す。これは「この限定的な明示参照集合について、Index上のSource情報が不変」という意味に限り、Application全体の互換性ではない。

| 条件 | 判定 | 意味 |
| --- | --- | --- |
| 明示参照集合と対象Indexの出典が一致し、関連Sourceが不変 | `limited-unchanged` | 限定した静的Source集合だけが不変 |
| Validationなどの明示参照が追加・削除された | `unresolved` | 依存集合の変化を再評価する |
| Validation等のSource内容・所在が変化した | `unresolved` | E4の参照検査で変更を確認 |
| E5が未解決、定義の重複、ReportとIndexの指紋不一致 | `unresolved` | 根拠不十分として停止 |
| 無関係なArtifactだけが更新された | `limited-unchanged` | 一律の再検証を避ける |

`limited-unchanged`には`fullClosure:false`、`authority:none`、`qualification:application-declaration-and-direct-source-definitions-only`を明示する。

## なぜ別の構成解決器ではないか

E6は、E5で取得した明示参照をE4の既存検査へ**受け渡す接着部分**にとどめた。業務規則の解釈、Plugin Providerの選択、Target実行適合、Runtime Permissionの判定を実装しない。

入力されたE5 ReportとSource Indexは、信頼できるWorkspace読み取り境界から得られたことが**別途保証される前提**である。JSONの形と二つの入力同士の一致を検査しても、出典の真正性、完全性、改ざんされていないことは証明できない。LoaderとSource Indexの解析時同一bytes保証を扱う[#1017 / PR #1111](https://github.com/hat47x/tei-platform/pull/1111)の成果も未統合であり、ここで代替しない。

この試作が受け取る依存は**E5で扱ったApplication、Semantic Namespace、Application Pattern、Patternが直接参照するData・Relation・Capability・Application所有Query、明示Validation、Database、Database Contractの定義出典**に限る。PatternのContractQueryについては直接の定義Sourceだけを扱い、**入力Bindingや結果型、実行時のDatasource適合**、Data・Relation・Capabilityの推移的依存、Action実装、認証・認可、Environment binding、UI構成キットの選択は含まれない。E5の部分的な`observed`は全依存閉包の成立を意味しない。SUI Stage 0には正式なApplicationがなく、Native Canvas/Document/Actionの契約もこの集合からは導出できないため、**第二Consumerの検証完了とはしない**。

### Contract Queryを独立したSubjectとして追跡

E5の直接参照に`pattern-contract-query`を追加したため、E6も`database-contract-query` Subjectを既存E4の変更検査へ渡す。特定QueryのSourceが欠落すれば、親のContractを見つけただけでは補完しない。

Source Indexでは、同じContractファイルに親Contractと内包Queryがある場合、**物理Artifactは一つ、論理Subjectは二つ**である。初期の合成試験では同じ物理パスを異なる指紋で二度登録していたため、偽の不一致が生じた。fixtureを修正し、Source変更時には双方のSubjectで同じファイル指紋が変わることを検査した。

現行E6の23件を、GitHub上のE4／E6コードと試験を読み込み、import/exportと`structuredClone`を置換したV8隔離実行器で再検証し、**23件成功・0件失敗**。これはNode.jsの`node --test`実行、実Go E5→E7接続やexact-headのRepository Healthではない。Source Indexに依存辺を取り込むときも、全依存の閉包や実Runtime権限が証明されたとはしない。

## 検証

```sh
node --test reference/application-composition/e6-typed-index-bridge.test.mjs
```

E4／E6／E7のGitHub上のSource原文と試験を**V8インメモリ互換ランナー**で実行し、合計**43件成功・失敗0**（うちE6の追加Pattern直接参照テストを含む）。`node:test`、`structuredClone`、基本`assert`を簡易互換実装へ置き換え、import／exportを試験用に変換した**限定的な実行**である。Node.js標準の`node --test`実行証拠ではない。

- E4 JS本体：`d2a61dac60b1641e453513e3f72b7c6c2ef43501`
- E6 JS本体：`2320d51a3c18d741dc5afb1d5adff23826884ec6`
- E6試験：`32b294e91cea36067538081d3a69d2958291c6af`
- E7 JS本体：`910ef058ed96d5830f026c0552196682580a20eb`

試験はValidation／Patternの参照先Source変更、無関係なSource更新、宣言依存集合の追加、定義欠落・重複、資格・役割のすり替え、Application不一致などを扱う。ただしE5のGoコードは**一度も実行していない**。E5が全参照を正確に抽出したとする根拠にはできない。

**未実施**：GitHub上の全Goソースを含むクリーンなTEIリポジトリでのE5 Goテスト・CLI実行、E5の出力と実Source Index JSONを使ったE6の一連の実行、実UI・Provider・SUI Target、exact-headの全体Repository Health。このためE6の成功を実Applicationの構成成功とは扱わない。

## 次に確かめること

既存GoのE5検証器を実Workspaceで起動し、現在のSource Indexと組み合わせてE6へ入力する。さらに、現在対象にしていないPatternのContractQuery／Datasource・RelationのTarget・Semanticの型参照、UI構成／Environment結合を**既存の意味・契約解析へ戻して**明らかにする。第二ConsumerのSUIにおいては、Native操作とApplication Actionの境界・保存先・権限の実在を別途検査する。

案件構成プロファイルには、現時点では**選択したKit／Target／Framework／案件差分と、それらの正本への参照**以外の新情報を増やす理由がない。汎用DSLや重複Lockの採用判断は実Consumer検証後に限る。
