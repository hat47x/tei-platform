# 案件固有の生成機構を共通化する最小実証

この参照実装は、[Issue #1067](https://github.com/hat47x/tei-platform/issues/1067)の最初の検証に使うものです。二つの案件で個別に作られた小さな生成処理から、共通する検証規則だけを取り出し、元の期待結果と振る舞いが保たれるかを確かめます。

**現在の状態は、合成データを中心とした段階的な試作です。** 共通契約の抽出と案件別採用はReference内で検証し、その採用・版固定・Build入力追跡をTEI既存のPlugin Lock／Build機構へ接続する試験も追加しています。一方、二つの実案件へ同じ共有Pluginを導入した実証、正式な汎用Capability Contractへの接続、実配備はまだ行っていません。

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

## Capability契約との境界

共有した検証生成器を、既存の`editor-intelligence`など別のCapabilityへ便宜的に押し込めません。現行productionの`internal/plugin.CapabilityKind`は固定列挙であり、任意の新しい意味契約を追加する正本にはまだなっていません。

汎用Capability ContractのReferenceは[#951](https://github.com/hat47x/tei-platform/issues/951)で進めています。並行PR #1058では、その契約を独立したConsumerから呼び出す検証が進行中です。本ReferenceはPlugin Lock／Buildとの接続だけを先に行い、Capability契約がproductionへ収束する前に独自のCapability表現を増やしません。

## この試作では分からないこと

今回の接続により、案件別の版固定とBuild入力の追跡を既存TEI機構で表現できることは検証対象に入りました。一方、Node.js側の簡単な評価器はTEI実行機構とは異なり、共有生成器を正式なCapabilityとして呼び出しているわけでもありません。

また、実際の二つの案件リポジトリへ同じ共有Pluginを導入し、それぞれの所有者が独立して受理・切り戻しを行った事実はまだありません。実Provider、署名・権限、配備、Runtime AuthorityもこのReferenceだけでは確認できません。

次の正の実証では、同一の意味契約を本当に共有できる二つの独立Consumerを固定し、それぞれの受入試験を保持したまま、共有Pluginの採用差分とLock更新を比較します。Capability契約については#951の既存機構へ収束させ、新しい並行正本を作りません。

生成コードの単体試験を機械的に案件ごとへ複製することは目的ではありません。共通部分の契約試験は共有側へ寄せ、案件ごとの業務上の期待値と接続検証は引き続き保持します。定義量と保守負担の削減効果は、実案件の比較が可能になってから測定します。
