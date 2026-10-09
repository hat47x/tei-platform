# 案件固有の生成機構を共通化する最小実証

この参照実装は、[Issue #1067](https://github.com/hat47x/tei-platform/issues/1067)の最初の検証に使うものです。二つの案件で個別に作られた小さな生成処理から、共通する検証規則だけを取り出し、元の期待結果と振る舞いが保たれるかを確かめます。

**現在の状態は、合成データを中心とした段階的な試作です。** 共通契約の抽出と案件別採用はReference内で検証し、その採用・版固定・Build入力追跡をTEI既存のPlugin Lock／Build機構へ接続する試験も追加しています。共有生成器そのものは外部Authorityを必要とするCapabilityではなく、まず決定論的Transformとして扱う方針へ整理しました。一方、二つの実案件への導入、Transformによる実生成、実配備はまだ行っていません。

## 確かめること

`projects/order.mjs`と`projects/invoice.mjs`は、もともと別案件が所有していたという想定の生成処理を、それぞれ独立して表しています。いずれも識別子の必須入力と、金額が安全な非負整数であることの検証計画を作ります。抽出前の関数は共有モジュールを参照しません。

`shared/required-fields.mjs`は、この二つに共通する処理だけを表した候補です。業務名や承認手順は扱いません。項目名と意味契約を受け取り、決定論的に同じ検証計画を構築します。対応していない意味契約、無効な項目名、同一項目への二重指定は受理しません。

承認の要否は案件側に残します。受注は10万円以上で承認が必要ですが、請求は海外向けかつ5万円以上という異なる条件です。これらを「承認」という名前の一致だけで共通化してしまうことを防ぎます。

## 局所的な実行

リポジトリのルートから、抽出前後の振る舞いを確認します。

```bash
node --test reference/cross-project-extraction/regression.test.mjs
```

テストは、抽出前後の生成計画が各案件の独立した期待値に一致すること、入力の正負例に対する判定が一致すること、意味契約の不一致を拒否すること、承認条件を案件側に残すことを確認します。期待値は共有ジェネレータから生成せず、テスト内に独立して記述しています。

## 二つの実リポジトリとの比較

[実リポジトリの比較記録](real-repository-comparison.md)では、SUIの文書取り込み検証とMMIのAI候補選択検証について、コミットと対象ソースを固定して読み取っています。

両者とも外部入力を検査していますが、対象の意味、正規化の許容範囲、時刻への依存、診断、採用権限が異なります。そのため、現在の段階では一つの共通検証プラグインへ統合しない判断としました。共通化を見送る理由を示す比較であり、実案件での共有プラグイン導入を実証したものではありません。

## 案件ごとの採用と切り戻し

[採用の回帰試験](adoption-regression.test.mjs)は、受注と請求が、それぞれ独立して共有生成器を選択できるかを合成データで確認します。共通生成器の候補は、識別子・候補版・意味契約・Git blob SHAを一組として固定します。同じ候補版を名乗っていてもソース内容が変わった場合、選択を拒否します。暗黙の最新版選択や、別案件への設定の伝播は認めません。

受注を従来の案件内生成器へ残したまま、請求だけ共有候補へ切り替える場合と、その逆の場合を検査します。その後、各案件が従来の生成器を再び選べることを確認します。両案件の異なる承認条件は、共有生成器の選択によって変更しません。候補版・意味契約・ソースの不一致や、選択方法が不明な場合には停止します。

リポジトリルートから次のコマンドを実行できます。

```bash
node --test reference/cross-project-extraction/adoption-regression.test.mjs
```

このNode.js試験は、既存のプラグイン解決機構やPlugin Lockの代替実装ではありません。回帰試験内で明示的に指定した候補を、読み取り専用で比較する補助です。

## 既存Plugin Lock／Build機構との接続

[Plugin Lock接続試験](plugin_lock_integration_test.go)は、独自の採用Lockや新しいResolverを作らず、TEIに既にある次の機構をそのまま利用します。

- `Environment`ごとのPlugin要求
- Runtime Plugin LockによるPlugin ID・版・Definition fingerprint・artifact digestの固定
- `BuildPlan`とBuild出力Lockによるsource digest・builder・target・build入力の固定
- Runtime Plugin LockへのBuildPlan fingerprintの結び付け
- `VerifyRuntimePluginBuild`によるBuild入力・artifact・Runtime選択の横断照合

試験では、受注と請求を別々のEnvironmentとして扱います。両方が共有Pluginの1.0.0を利用している状態から、請求だけ1.1.0へ採用し、受注のLockが1.0.0に留まることを確認します。請求を以前のLockへ戻せることも検査します。これにより、「別案件が新版を採用しただけで自案件へ伝播する」経路を作りません。

さらに、共有候補の実ソースを`BuildPlan.Sources[].Digest`へ結び付けます。固定後にsource digest、artifact、Build出力の対応が変わった場合は、既存の検証機構で失敗させます。

局所的には次の試験を実行できます。

```bash
go test ./reference/cross-project-extraction
```

このGo試験を追加した時点では、trustedな完全checkoutでの実行結果はまだ取得していません。PR統合資格として扱うには、exact headを固定した`tools/repository-health-check.sh`の全項目PASSと、実行前後のclean状態が別途必要です。

## 共有生成器の実行境界

[実行境界の整理](transform-boundary.md)では、現在の共有生成器を汎用Capabilityではなく、まず決定論的Transformとして扱う理由を整理しています。

現行JS候補は、契約identity・項目名・項目の重複を検査する責務と、検証計画を組み立てる責務を一つに持っています。TEI本体へ接続するときは、前者をSemantic／Authoring側のtyped reference検証へ残し、後者だけを`internal/plugin/transform`へ移します。外部read/writeを必要としない純粋変換に、Provider ResolutionやRuntime Authorityを持ち込みません。

現行Transform IRで今回の生成を表すには、複数の式から配列を組み立てる最小primitiveが不足しています。#979側へ`construct-array(values: Expr[])`相当のpressureを引き渡します。条件分岐や正規表現をTransformへ追加してJS側の意味検査を複製するのではなく、受理済みtyped inputからderived planを作る範囲へ限定します。

汎用Capability ContractのReferenceは[#951](https://github.com/hat47x/tei-platform/issues/951)で進めています。並行PR #1058のGeneric Invocationは、Source AdapterやContract Queryなど実行サービスの境界として利用し、今回の純粋生成器の必須依存にはしません。将来、外部schema取得やDB introspection等が必要になった場合だけ、そのreadをCapabilityへ分離します。

## 既存の配列写像によるTransform実証候補

[配列写像のGo試験](transform_projection_test.go)では、[TEIの変換定義](required-fields-projection.transform.json)を実際の`ParseSourceJSON`で解析し、既存の`map-array`と`construct-object`を使って、順序付き規則配列から受注・請求の派生検証計画を構築できるか確認します。期待される出力は試験側に独立して記述し、未対応の演算子・必須項目の欠落を拒否する試験も含めます。

```bash
go test ./reference/cross-project-extraction -run 'TestExistingMapArray|TestProjection'
```

この経路で十分かどうかは、**Authoring側が正しい規則配列を独立して提供できるか**に左右されます。Transformは規則名が案件の意味に合致するかを判断しません。既存の受理済み規則から必要な配列を得られるなら、新しい配列構築演算子は追加せずに済む可能性があります。逆に、二つの項目参照しか入力されない場合には、Transformが二規則を構築する方法を別途検討する必要があります。

検討の根拠と境界は[実行境界の整理](transform-boundary.md)に記録しています。これは合成入力の能力試験であり、既存の業務仕様から規則を自動導出した証拠ではありません。現時点ではtrustedな完全checkoutでのGo試験実行結果も取得していません。

## 実際の業務仕様を起点とする規則投影

[業務意味モデルからの投影試験](semantic_validation_source_test.go)では、実際の`reference/workspace`を既存の`workspace.Load`で読み込み、`validation.Validate`で意味検証を実施します。受注アプリケーション`order_entry`が明示的に参照する3件の業務規則を、`semantic.Validation`から宣言順に取り出し、[既存Transformの変換定義](semantic-validation-projection.transform.json)へ渡します。新しいルールを試験コードに書き直して入力する方式ではありません。

確認対象は、数量と明細番号の`positive`、明細番号の`relation-unique`です。正の値を「非負」に置き換えず、関係内一意性を単なる項目検査へ弱めず、識別子・規則種別・属性・Relationを保持します。検証結果の期待JSONは試験側に独立して記述します。Source Indexも参照し、正本の定義ファイルとアプリケーションから遡れるかを確かめます。

```bash
go test ./reference/cross-project-extraction -run 'TestActualOrderEntrySemantic|TestSemanticValidationProjection'
```

これは既存の業務仕様から**派生する規則情報**をTransformで表せるかの試験です。生成された計画が各種のデータ検査を実行できること、実際のProvider・UIに適用されたこと、異なる二つの実案件で同じPluginを採用したことは示しません。現時点でGo試験はtrustedな完全checkoutで未実行です。

この経路で確認できるのは受注アプリケーション一つです。独立した二番目のConsumerと、実際の業務検証ロジックを共通化できるという証拠は引き続き必要です。

## 既存の実行機構を優先する判断

[実行境界の整理](transform-boundary.md)のE8では、TEI本体にすでにある`driver.ResolveApplicationValidations`と`validationeval.Evaluate`、First Web Targetの実際の検証経路を確認しています。

受注アプリケーションの現在の業務検証では、これらの機構が**正本の検証規則を直接消費**します。したがって、受注の実行のために、別の規則計画を生成する共通ジェネレータを追加しません。

このReferenceの[意味規則の投影試験](semantic_validation_source_test.go)も、規則の選択を独自に再実装せず、既存Driverの解決結果を入力として使う形に修正しました。元の業務規則を異なる形式で必要とする明確な消費側が見つかるまでは、投影した計画は製品の実装対象ではなく、**既存Transformで表現可能かを確かめる試作**として扱います。

## 共通化の効果測定

[共通化の効果測定](sharing-economics.md)では、現在の二案件fixtureについて、抽出前後の生成処理の記述量を比較しています。

現在の実装をそのまま数えると、抽出前は非空18行、共有化後は31行で、二案件だけでは約72%増えています。重複したrule宣言は4個から2個へ減りますが、共有契約の検査という固定費が上回ります。単純な行数モデルでは現在の構造の損益分岐は概ね7案件目です。

この値を一般的な閾値にはしません。重要なのは、共通化候補が見つかったことと、共通化する価値があることを分けることです。変更箇所・試験・レビュー負担まで含めて利益が出なければ、案件内の小さな重複を維持することも正常な判断とします。

## この試作では分からないこと

今回の接続により、案件別の版固定とBuild入力の追跡を既存TEI機構で表現できることは検証対象に入りました。一方、Node.js側の簡単な評価器はTEIの決定論的Transform実行機構とは異なり、共有生成器をTEI-native Transformとして実行したわけではありません。

また、実際の二つの案件リポジトリへ同じ共有Pluginを導入し、それぞれの所有者が独立して受理・切り戻しを行った事実はまだありません。実Provider、署名・権限、配備、Runtime AuthorityもこのReferenceだけでは確認できません。

次の正の実証では、まず#979側の最小Transform表現が成立した後、JS共有生成器とTEI Transformの出力を受注・請求双方で比較します。そのうえで同一の意味契約を本当に共有できる二つの独立Consumerを固定し、それぞれの受入試験を保持したまま採用差分とLock更新を比較します。外部Authorityが必要になった場合だけ#951のCapability契約へ接続し、新しい並行正本を作りません。

生成コードの単体試験を機械的に案件ごとへ複製することは目的ではありません。共通部分の契約試験は共有側へ寄せ、案件ごとの業務上の期待値と接続検証は引き続き保持します。定義量と保守負担の削減効果は、実案件の比較が可能になってから測定します。
