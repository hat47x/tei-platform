# reference-xead-source-inspect

Repository外に置いたXead/Xeaf source snapshotを、RMT-2のsource-faithful adapterと
`xead-source-transcription-contract.synthetic.json`で検査するReference toolです。

このtoolはCanonical sourceを変更しません。標準出力も、原modelのraw XML、業務名称、
legacy ID、完全なsource locatorを複製しない集計Evidenceに限定します。

~~~text
go run ./tools/reference-xead-source-inspect \
  -format xead \
  -source /path/to/local-model.xead
~~~

主な出力:

- schema version（現在は `reference-xead-source-inspection-v4`）
- source format / SHA-256
- XML construct inventory
- unknown construct件数（名称は公開出力しない）
- candidate classificationの集計
- 項目拡張型（`TypeOptions`／`FieldOptions`）の指定を意味別の軸へ分けた集計
- legacy / runtime keyとRelation mappingの集計
- DERIVABLEなkey memberの件数
- DERIVABLEなkeyを参照するRelationshipの件数
- unresolved diagnostic codeと、source digest / resolution policy revision / diagnostic ordinalから作るopaque diagnostic token
- relational source fact全体の整合状態を示す `relationalResolutionStatus = resolved | partial | unresolved`
- fact種別ごとの `resolved / unresolved / unvalidated` 件数を示す `relationalFactResolutions`
- readiness判定算法のrevisionを示す `relationalResolutionPolicyRevision`
- transcription contract fingerprint
- normalized classification policy fingerprint
- authority limit

出力はReference Modelの利用条件を変更しません。public RepositoryへEvidenceを保存する場合は、
[`docs/business-system-design.md`](../../docs/business-system-design.md)に記載された第三者資料の参照・複製・再配布の原則と、
[Issue #941](https://github.com/hat47x/tei-platform/issues/941)のReference Modelごとの権利確認状況を確認してください。
原典やその実質的な抜粋・変換物の公開可否が未確認の場合は、公開判断をOwnerへ戻し、公開Evidenceに含めません。
構造検証の成功やsource factの観測は、意味受理にも公開許諾にもなりません。


`relationalResolutionStatus`と`relationalFactResolutions`は、source factをCanonical meaningとして承認したことを示しません。`relationalResolutionPolicyRevision`も判定算法のprovenanceであり、semantic authorityではありません。
`resolved`は、その検証scopeで構造diagnosticが残っていないことだけを表します。
`partial`は、diagnosticは無いものの現行validatorでは安全に検証できないfactが残る状態です。
`unresolved`の場合も観測したsource factはEvidenceとして保持します。

全体statusが`unresolved`でも、無関係なfactまで自動的に意味不明になるとは限りません。
内部Inspectionは各factを`resolved / unresolved / unvalidated`へ分けます。ただし局所`resolved`はloweringの「候補資格」であり、
Canonical meaning、business identity、functional dependency等の承認ではありません。局所loweringを行う場合も、
そのfactが依存するKey / Table / Field / Relationshipに未解決が伝播していないことをexplicitなResolution Projectionから確認します。

公開reportではraw locatorやlegacy IDを出さず、fact種別ごとの件数だけを出します。

## 項目拡張型の指定を意味別の軸へ分ける

`fieldDesignations`は、`TypeOptions`／`FieldOptions`に書かれた指定を、値そのもの、表示名、
値域、入力時の補助、更新可能性、物理DB型、導出、保存の有無、表示書式という別々の軸へ
割り当てた結果を件数で示します。対象は`KUBUN(...)`、`VALUES(...)`、`BOOLEAN(...)`、
`VIRTUAL`、`NO_EDIT`の5つに限定します。

| 指定 | 読み取れる軸 | 読み取ってはいけない軸 |
| --- | --- | --- |
| `KUBUN(...)` | コードの同一性、値域の存在、入力時の補助 | 更新可能性、物理DB型、導出、保存の有無 |
| `VALUES(...)` | 値域、入力時の補助 | 表示名の分離、更新可能性、物理DB型、導出、保存 |
| `BOOLEAN(...)` | 値域（真偽に対応する保存値）、入力時の補助 | 更新可能性、物理DB型、導出、保存、表示名 |
| `VIRTUAL` | 導出の存在、保存しないこと、物理列にしないこと | 値域、更新可能性、表示書式、入力時の補助 |
| `NO_EDIT` | 表示書式のみ | 更新可能性、値域、物理DB型、導出、保存 |

`NO_EDIT`は更新可否の指定ではありません。旧Driverで更新可否を決めるのは`NoUpdate`属性で、
`NO_EDIT`は先頭ゼロや桁区切りを保つ表示書式として読まれています。本toolは`NoUpdate`を
読みません。

`KUBUN(...)`が参照するコード表の中身はmodel fileに無く、外部表の行として存在します。
そのため`KUBUN`は`unresolvedAgainstSourceByKind`へ計上されます。値域が宣言されていることと、
その値域の中身が分かっていることを区別します。

`KUBUN`と`VALUES`のように同じ軸を別の意味で主張する組は`conflicts`として件数を出し、
どちらかを勝者にしません。旧Driverは画面ごとに逆順で検査しており、一方を正本にできません。

解釈しないもの:

- 対象5語以外の指定（`AUTO_NUMBER`、`YMONTH`、`PERCENT`など）は`unknownTokenCount`へ計上し、
  意味を与えません。
- `NO_EDITFYEAR`のように既知の語で始まり続く指定は、`ambiguous-concatenation-not-split`として
  数え、分割しません。旧Driverも1語として読むため、分割すると挙動が変わります。
- 括弧内のカンマで分割が崩れた場合は`unbalanced-parenthesis-after-comma-split`として数え、
  読めなかったことを残します。
- 宣言された文字列そのもの（コード表名、保存値、旧項目ID）は公開reportへ出しません。

この集計は、宣言された文字列をどう分けるかの規則
（`tokenizationPolicyRevision`）を記録します。旧DriverとEditorは4通りの異なる読み方を
しており、どれも「X-TEAの意味」の正本ではないためです。


diagnostic tokenもsource locator文字列のhashにはしません。構造pathに業務名や独自schema名が含まれる場合、単純なlocator hashは候補pathから辞書照合できるためです。v3ではexact source digest、readiness判定policy revision、diagnostic ordinalからtokenを生成します。このtokenは同一report内の相関用であり、source node identityやCanonical identityとして使いません。
