# REF-SPEC-01：受注のValidation参照差の静的照合（2026-10-09）

## 目的と判定の範囲

業務手順を含む複数の設計視点を、一つの変更から追跡できるかを評価するための**読み取り専用の静的確認**である。新しい業務仕様や修正要求を受理するものではない。関連する価値と実証計画は[技術リサーチ統合方針](../../plan/TECH_RESEARCH_INTEGRATION_2026-10.md)のREF-SPEC-01に置く。

- 照合対象のGit revision：`797ce7e99851b0c5cd054bf0b93824e329653eaf`
- 取得方法：GitHub上の同一revisionから正本Sourceを個別取得し、Applicationの`uses validation`宣言、Sales側のValidation定義、Business ProcedureのBP-05の明示参照、Customer編集probeを読み取った。
- 照合方法：`.tei`の対象行を限定した文字列抽出と、Business ProcedureのJSON解析。TEIの全構文解析、Workspace Loader、Validation Engineや実際のCLI実行は行っていない。
- Sourceの更新：なし。出力はこの文書だけに記録する。

## 照合したSource

| 参照元 | Git blob SHA |
| --- | --- |
| `reference/workspace/applications/order-entry/application.tei` | `00c29ac0e557cb50fdcd13f2477d0b538a528c80` |
| `reference/workspace/semantic/sales/validation.tei` | `fd9e3ad6a81e3201ade8e987377137a3ba702d5f` |
| `reference/business-system-design/order-entry-business-procedure.json` | `ac1980d90ddd702a362078e1d4dd8dd70f5bc1a2` |
| `reference/business-system-design/order-entry-customer-edit-conflict.json` | `2786138549a4b3f227f06969f57bf2a6ba47a99f` |

## 得られた事実

`order_entry`の`uses validation`宣言には、以下の3件が存在する。

1. `sales.order_line.quantity_positive`
2. `sales.order.lines.line_number_unique`
3. `sales.order_line.line_number_positive`

対象とした`sales`のValidation定義Sourceにも、同じ3件のIDが記載されていた。

一方、`order-entry.order-handling`の業務手順BP-05（`validation-support`）で明示されたValidationは上記の1、2の2件であり、3番目は存在しなかった。したがって、**Application側にのみ記載されたIDは`sales.order_line.line_number_positive`の1件、業務手順側のみに記載されたIDは0件**である。

Customer編集probeは、同じ業務手順のBP-02に対し、`order_entry.customer_reference`を対象に`read-write`を要求していた。この事実だけでは、現行checkerが期待の拒否診断を実際に返したと主張できない。

## 判断を残す事項

BP-05の明示参照は、Application全体のValidation宣言を網羅する義務を持つとは定められていない。したがって、上記の集合差をそのまま欠落・不整合・自動同期対象と判断しない。業務手順の適用範囲に行番号の正値検査が必要なのか、Application側から導出した一覧を表示すれば十分なのかを、人が確認できる候補として示す。

現行`reference-business-procedure-check`の標準出力は、Business Procedureが明示したValidationの解決と、Patternから導出できる意味を扱っている。今回の集合差を**実際に表示したという証拠はまだない**。任意の比較表示を追加する試作については、コードとテストを別途検証する。

## 次の検証と完了条件

1. 同一exact headのclean checkoutで、既存のbaseline checkerとCustomer編集probeを実行し、診断・終了値・実行前後のcleanを記録する。
2. 比較表示を指定した場合だけ、上記のApplication-only IDを`review-needed`として返せるか確かめる。標準動作とblocking診断を変えない。
3. 文脈未指定、ID重複、参照不能を未解決として返し、意味の推測や権限の補完をしないことを負例で確かめる。
4. 実際の人のレビュー、業務上の採否、実Provider、UI経由の実行結果は別に確認する。

**現時点の結論：** 同一revisionのSource間に、確認候補となる明示参照の差が一つ存在することまでは静的に確認できた。これを検出する製品機能の成立、checkerのPASS、業務上の欠落の確定、利用者の確認負担の削減は、いずれも未検証である。

## 隔離環境での原文コード確認（Go 1.23.2）

プルリクエスト内の`tools/reference-business-procedure-check/validation_scope_review.go`を、出典のまま隔離用のGoモジュールへ転記し、`git hash-object`の結果がGitHubに保存されているblob SHA `c061d451c774a25999cd2fd7a8e18dc250d6018c`と**完全一致**することを確認した。比較処理そのもののSource bytesは一致する。

隔離モジュールでは、Application、World、fixture、Validation IDの件数取得を最小限の代替型・関数として用意し、以下の6件を実行した。

1. 受注のApplication-only Validation 1件とJSON出力
2. Application文脈の未指定・同一ID重複
3. Validation IDの未定義・意味定義の同一ID重複
4. Application・Procedure内のValidation宣言重複
5. 集合一致時の扱い
6. Application-onlyとProcedure-onlyが同時に存在する場合の双方向差

`GOENV=off GOWORK=off GOFLAGS= go test -count=1 -v ./...`は6件PASS。`GOENV=off GOWORK=off GOFLAGS= go vet ./...`もPASSした。`gofmt -l tools/reference-business-procedure-check/validation_scope_review.go`の出力は空だった。

**限界:** ここで使用したApplication・World・fixture・semanticCountsは隔離モジュール内の最小の代替実装であり、TEI本体の型とWorkspace Loaderは実行していない。さらに、PRの既存`main.go`変更、GitHubにある`validation_scope_review_test.go`、リポジトリ全体のテストは、この記録では検証していない。したがって、隔離テストのPASSはプロダクトの受入・統合資格にならず、PRのexact-headでのfull Repository healthとclean前後記録が必須である。

## Source Indexによる戻り先の検証（2026-10-09）

静的な集合差だけでは、利用者が判断に必要な正本へ戻れない。そこでDraft PRの比較結果へ、次の読取情報を追加した。

- `procedureSource`: Business Procedureの参照を記した入力fixtureのパス。Canonicalな意味IDではない。
- `sourceEvidence`: 既存Source Indexを根拠とするApplication定義と差分Validation定義の相対パス・開始行・ファイル指紋。
- `unresolvedSource`: Source Index内の定義が不在・重複・指紋欠落の場合、推測で補完せず残す対象の意味ID。

比較機能を指定しない既存CLI実行では、Source Indexを追加生成しない。比較指定時にSource Index自体の検査に失敗した場合はエラーとして停止する。成功時のsource evidenceも**意味や業務判断の受理・権限付与を表さない**。

今回GitHubに登録した`tools/reference-business-procedure-check/validation_scope_review_sources.go`の原文を隔離用のGoモジュールへ転記し、`git hash-object`がblob SHA `a86ee7dc38a1e1d6f7fa6563511f17e9f70bce66`に一致することを確認した。隔離モジュールは`tei/internal/subject`と`tei/internal/sourceindex`の最小代替型を持つ。Go 1.23.2で次の4件のテストがPASSし、`go vet ./...`もPASSした。

1. ApplicationとValidationの出典をそれぞれ1件に絞って返す。
2. 同じ意味IDに二つのSource定義が存在すれば、選ばず未解決に残す。
3. ファイル指紋が無ければ、根拠として返さない。
4. 意味参照自体が未解決であれば、Source探索から勝手に意味を作り出さない。

新規ファイルの`gofmt -l`出力は空だった。これは**原文の一ファイルを隔離した試験**である。本体の`workspace.BuildSourceIndex`、既存のSource Index、`runWithOptions`の接続、テストファイルを含むPR全体の`go test`は未実施である。結論として、実際の業務アプリに対するsource provenanceの成立と、人が確認できる実際の所要時間の短縮は、引き続き未確認である。
