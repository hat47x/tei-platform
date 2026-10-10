# E6：解析済み参照とSource Index変更検査を接続する

- **状態**：読み取り専用の接続試作（2026-10-10）。正式なApplication Composition Resolver、業務意味の受理、実行権限、第二のPlugin Lockではない。
- **管理**：[Issue #1016](https://github.com/hat47x/tei-platform/issues/1016)／[Draft PR #1051](https://github.com/hat47x/tei-platform/pull/1051)。
- **前提**：[E5](E5_PARSED_WORLD_SOURCE_JOIN.md)で既存のGo Loader・型付きWorldから限定的な参照集合を抽出し、[E4](E4_SOURCE_INDEX_JOIN.md)でSource Indexに記録されたSubjectの変更を検査した。この二つの結果を手入力の依存一覧なしで結ぶ。

## 具体的に接続したもの

[E6接続試作](e6-typed-index-bridge.mjs)は、E5のJSON形式のReportを変更前後の二つ受け取り、対応するSource Index二つと照合する。

1. E5 Reportが対象Applicationを一つ特定できており、宣言・Source所在の照合に未解決事項がないことを要求する。E5側の`observed`をそのまま全面的な適合へ読み替えない。
2. Application・Validation・論理Database・Database Contractの既知の参照種類だけを対象に、Report内の定義SourceとSource Indexの`definition`参照、Artifact指紋が一致することを確認する。出典欠落・重複・取り違えは停止する。
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

`limited-unchanged`には`fullClosure:false`、`authority:none`、`qualification:application-validation-and-database-contract-definitions-only`を明示する。

## なぜ別の構成解決器ではないか

E6は、E5で取得した明示参照をE4の既存検査へ**受け渡す接着部分**にとどめた。業務規則の解釈、Plugin Providerの選択、Target実行適合、Runtime Permissionの判定を実装しない。

入力されたE5 ReportとSource Indexは、信頼できるWorkspace読み取り境界から得られたことが**別途保証される前提**である。JSONの形と二つの入力同士の一致を検査しても、出典の真正性、完全性、改ざんされていないことは証明できない。LoaderとSource Indexの解析時同一bytes保証を扱う[#1017 / PR #1111](https://github.com/hat47x/tei-platform/pull/1111)の成果も未統合であり、ここで代替しない。

この試作で自動的に収集できる依存は**E5で扱った明示Validation、Database、Database Contract、Application自体**に限る。`SemanticUses`、Pattern、Actionの実装、認証・認可、Environment binding、UI構成キットの選択は含まれない。E5の部分的な`observed`は全依存閉包の成立を意味しない。SUI Stage 0には正式なApplicationがなく、Native Canvas/Document/Actionの契約もこの集合からは導出できないため、**第二Consumerの検証完了とはしない**。

## 検証

```sh
node --test reference/application-composition/e6-typed-index-bridge.test.mjs
```

Node.jsの隔離実行環境で、E4モジュールとE6モジュール、およびE6の14件の合成試験を実行し、**14 PASS / 0 FAIL**を確認した。GitHubへ登録したGit blob SHAと手元で試験した内容が一致した。

- E4モジュール：`d2a61dac60b1641e453513e3f72b7c6c2ef43501`（既存ファイル）
- E6モジュール：`820638daeed5d93b2c4cf3fcca2f643b8e5262aa`
- E6試験：`9646348b24394a47aa9bd969d9573d5e403da9cb`

正常例に加え、依存先Validationの変更、無関係なSource更新、宣言追加、未解決Report、資格の取り違え、Source指紋不一致、定義の重複・欠落、役割のすり替え、不正な参照種類、Applicationの不一致を検査した。

**未実施**：GitHub上の全Goソースを含むクリーンなTEIリポジトリでのE5 Goテスト・CLI実行、E5の出力と実Source Index JSONを使ったE6の一連の実行、実UI・Provider・SUI Target、exact-headの全体Repository Health。このためE6の成功を実Applicationの構成成功とは扱わない。

## 次に確かめること

既存GoのE5検証器を実Workspaceで起動し、現在のSource Indexと組み合わせてE6へ入力する。さらに、現在対象にしていないSemanticUses、PatternとUI構成／Environment結合の所在を**既存の意味・契約解析へ戻して**明らかにする。第二ConsumerのSUIにおいては、Native操作とApplication Actionの境界・保存先・権限の実在を別途検査する。

案件構成プロファイルには、現時点では**選択したKit／Target／Framework／案件差分と、それらの正本への参照**以外の新情報を増やす理由がない。汎用DSLや重複Lockの採用判断は実Consumer検証後に限る。
