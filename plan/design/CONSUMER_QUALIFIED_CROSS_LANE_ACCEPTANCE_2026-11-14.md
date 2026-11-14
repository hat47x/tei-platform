# Sourceから実Consumerへ届くまでの横断受入設計

更新日：2026年11月14日  
位置付け：既存製品計画へ各担当の検収条件を返す設計案。新しい共通Runner、認可機関、状態テーブル、独立Expectedの正本を設けない。

## 1. なぜ横断の検収が必要か

TEIでは、設計文書からモデルを読み、FunctionとProviderを選び、CLIやStudioから操作し、その結果を人・外部システム・別のアプリケーションが利用する。一つの段階で成功しても、次の段階まで同じ意味、同じ対象、現在有効な権限で到達したとは限らない。

2026年11月14日の検討では、Queryの隔離PostgreSQL試験、取得範囲の観測、生成CLI、BuildのToolchain観測、外部資料からの候補抽出、組織の二時点照会、段階別Plugin Contribution、B2Cの主体別開示が、それぞれ別の境界を持つことが明らかになった。これらの局所成果を一つの「製品完成」という表示にまとめるのではなく、**利用者が実際に必要とする結果に到達できたか**でつなぐ。

この横断設計は、各契約と操作へ付ける*照合観点*である。全製品に共通する新しい資格Enumや承認エンジンの導入を提案するものではない。採用と実装は各Ownerの既存Issueで判断する。

## 2. 一つの利用例をどこまで追うか

例は「会員が自分の受注履歴を、指定日時以降で検索する」とする。B2Cの専用Hostを新設する必要はなく、既存の型付きQuery、Function Data Context、共通Operation、TargetごとのFrontendを利用する。

1. **Definition**：固定したSource内にQueryの対象、必須引数 \`from\` と \`std.datetime\` 型、返す行の意味を記述する。SQLのプレースホルダーが存在するだけでは型宣言済みと判断しない。
2. **Binding**：Workspace・Environment・実Database Instance・Provider・Query・引数を、呼び出す直前の対象へ対応付ける。読取り用の参照Toolが通ったことを製品CLIの実接続へ転用しない。
3. **Authority**：現在の主体が匿名利用者、ゲスト、会員、運用者、分析主体のどれであるかを識別し、Operationと対象のRead Scopeを評価する。公開可能なAPI名と実行権限は別である。
4. **Execution**：通常の \`tei\` CLIまたは製品Operationが、選ばれた実Providerを呼んだことを記録する。呼出し先は選択名ではなく、実行されたProvider・接続先・試行で確かめる。
5. **Acquisition**：Rowsetが返った場合にも、取得できた範囲、行・列の由来、RLSやLIMIT等の制限、未取得を残す。Native SQLの行の完全性を推測しない。
6. **Delivery**：GUI、CLI、B2C Frontend、独立したConsumerが、認可された情報だけを実際に受け取り、対象ID・日時条件・結果本文を保持したことを確認する。画面が応答しただけでは内容の正しさが確定しない。
7. **Acceptance**：独立したExpectedと、正例・負例の実観測を比較する。業務的な意味の受理や外部作用の許可は、担当OwnerとHost側が別に判断する。

**匿名の一覧閲覧と、会員自身の注文履歴を取得する操作は異なる。** どちらも同じQueryの機能を再利用し得るが、検索結果の存在を知ってよい範囲、返してよい項目、キャッシュの共有、Grant失効時の応答が異なる。Guestカート・運用者の業務照会・分析用の集計へ安易に同じ資格を転用しない。

## 3. 受入票で分ける時点と証拠

次の表は既存Receipt、Build報告、Source参照、Driver Scenario等をどう結び付けて読むかの例である。新しい中央保存形式の採用を求めない。

| 観点 | その段階で確かめること | その観測だけでは確定できないこと |
| --- | --- | --- |
| Sourceと意味 | Sourceの原文バイト、Source Revision、World Fingerprint、参照位置、Schema版 | 最新の意味受理、利用者の操作権限 |
| 選択した構成 | Contract、Target、Provider、依存Closure、実行環境 | 実Providerの呼出し、成果物の実使用 |
| 実行時 | Attempt／Operation ID、開始時の資格、実Call対象、入力、失敗分類 | 外部の副作用や、途中で失効したGrantの現在性 |
| 取得したデータ | Rowset、観測できたLimit・RLS・関係、Coverage、NULL由来の資格 | Native SQLの全件性、すべての列の真正な来歴 |
| Consumerの受領 | Audience、受領した対象IDと版、表示・受信の範囲、対応する試行 | 他のAudienceへの開示適格性、業務意味の受理 |
| Buildと配布 | Build Intent、Resolver、実ChildのDigest、Toolchain、Artifact、Consumerで読まれたBundle | 計画外のToolchainに対する再利用資格、Seed Trust、別OSでの動作 |
| 作用と復旧 | 外部更新を確認したReceipt、再読取り、結果不明の区間 | 受付成功だけによる外部作用の完了、無条件の安全な再送 |

**Definition時点、Execution時点、Consumer受領時点の資格は独立に更新される。** 以前は許可されたデータでも、現在のGrantが失効した後に古い表示やAIの応答を新しい結果として再利用してはならない。SourceがA→B→Aと戻り、現時点のバイト列が以前と一致しても、途中の変更がなかったことや当時の権限が継続していることは証明できない。

### Coverageは「見えた制限」と「完全性の証明」を分ける

[#4647](https://github.com/hat47x/tei-platform/pull/4647)のW1〜W3 Draftでは、隔離PG16でRLS、LIMIT、JOIN、View等の観測から\`partial\`と理由を返す参照経路を検証した。反面、Native SQLの関数による暗黙の絞込みは、計画中の関係やLIMITに現れない場合がある。このため、計画上の制限が見つからないだけで\`complete\`や「本当に0件」を宣言しない。列の由来を追うW4やCurrent Grantも別の課題である。

Coverageは、**観測できた事実を保守的に説明するもの**であって、開示を許可する能力ではない。RLSで隠された行の存在を、表示件数や診断から漏らさない条件も別に検査する。

### Toolchainと生成物の関係を省かない

[#4648](https://github.com/hat47x/tei-platform/pull/4648)では、同じBuild計画でも異なるGo Toolchainで生成した実行ファイルのバイト列が異なり得ること、報告にある実行ファイルのDigest一致はそのToolchainで実際にBuildした証拠とは異なることを確認した。**ToolchainをBuildIntentの正式入力に含めるか**はOwner判断を残す。別HEADのCode試験がそれぞれ成功しても、生成TS→Frontend Bundle→Go埋込み→実Windows Stage2の受領までを一つの試行で通したことにはならない。

## 4. 横断する各領域の停止線

| 領域 | 現時点の具体的な成果・足場 | 必ず残す未受理の境界 | 既存担当 |
| --- | --- | --- | --- |
| Studio・セルフビルド | Native Wailsの限定試験、Builderの宣言・実Child PinとToolchainの局所検証 | 非Plugin N0〜N3、生成Stage1→Stage2、実WebView2での成果物受領 | [#2831](https://github.com/hat47x/tei-platform/issues/2831)・[#4583](https://github.com/hat47x/tei-platform/issues/4583)、[PR #4634](https://github.com/hat47x/tei-platform/pull/4634)・[#4648](https://github.com/hat47x/tei-platform/pull/4648) |
| QueryとFunction Context | RA-0/1/2の同一Source・実PG参照検証、W1〜W3の取得制限観測 | 通常製品CLI→実Provider（RA-2P）、独立Consumer（RA-3）、W4列由来、Current Grant | [#2586](https://github.com/hat47x/tei-platform/issues/2586)、[PR #4644](https://github.com/hat47x/tei-platform/pull/4644)・[#4647](https://github.com/hat47x/tei-platform/pull/4647) |
| Script Launch | Input/Effectの事前拒否と既存Function・生成CLIのReference | 宣言済み型を通常Entryから実束縛し、Host選択Providerの実Callと受領を確かめる | [#4495](https://github.com/hat47x/tei-platform/issues/4495)、[Script Launch計画](SCRIPT_LAUNCH_ENTRY_TEST_QUALIFICATION_2026-11.md) |
| Plugin | 同じ意味版にStudio／Generator／Runtime／TestのContributionを対応付けるReference | 選抜・生成物への収録・実Provider呼出し・現行権限・実結果を別に検証 | [#951](https://github.com/hat47x/tei-platform/issues/951)、[PR #4646](https://github.com/hat47x/tei-platform/pull/4646) |
| 外部資料・REA解析 | Host固定Selectorとbyte-only R2投影、限定したローカル別Process試験 | Worktree・Source現在性、Audience別の開示、異なるTransportの独立Consumer、Owner意味受理 | [#3394](https://github.com/hat47x/tei-platform/issues/3394)、[PR #4637](https://github.com/hat47x/tei-platform/pull/4637) |
| 組織・履歴 | 同じ所属のOccurrence IDと、更新ごとのRecord Versionを区別する設計、二時点の合成Expected | 実DBの人物・所属履歴・権限・二時点の再取得、独立業務Consumer | [#2386](https://github.com/hat47x/tei-platform/issues/2386)、[PR #4633](https://github.com/hat47x/tei-platform/pull/4633) |
| B2C | 会員履歴Queryと認証主体・Audienceの区分、隔離PGによるRA-2参照検証 | 本人性、行・列の開示、Guest/会員/運用者のCache分離、失効と実Consumer受領 | [#2586](https://github.com/hat47x/tei-platform/issues/2586)、[B2C計画](../workstreams/PRODUCT_COMPLETION_PLAN.md) |
| AIと文書Consumer | Goal／Session／Activity／OperationとSource参照、作業票、Skill比較の設計 | 外部Harnessの実送受信、完全な結果、Source/Grantの再資格化、独立試験 | [#3262](https://github.com/hat47x/tei-platform/issues/3262) |

上表は**2026年11月14日時点の特定PRに基づく作業境界**である。PRがDraftのまま、あるいは最新HEADのCIが未確認の場合、別HEADの成功を現在のmainの正式な動作確認に含めない。

## 5. 既存Driverへ渡す負例の例

| Case | 故障条件 | 必須の結果 |
| --- | --- | --- |
| CL-01 | \`:from\`はSQLにあるがQueryのInput宣言がない | 製品公開を拒否し、型を推測しない |
| CL-02 | Input宣言は\`std.datetime\`だが文字列・NULL・未知名が送られる | Executorを呼び出さず診断する |
| CL-03 | RLS／LIMIT／関数が結果を制限しているがConsumerが0件を取得 | 全件不存在や\`complete\`を主張しない |
| CL-04 | 会員Aの結果・Cacheを会員Bの操作で参照する | Audience・現在のGrantを再評価し、不正開示しない |
| CL-05 | 同名の組織MembershipがあるがOccurrence ID・履歴版が異なる | 異なる履歴を一つの現所属へ統合しない |
| CL-06 | 外部EvidenceがA→B→Aと変化し、過去のDigestへ戻る | 現在のByte一致だけで履歴現在性を主張しない |
| CL-07 | Provider Bを選択したが実Child／実Call先はA | 実行前または受領前に不一致として停止する |
| CL-08 | Build計画が同じでもToolchainが異なりArtifactが変わる | 再利用資格を保留し、生成器・Toolchainの区別を残す |
| CL-09 | DB更新後にReceipt受領が途切れる | 結果不明を保持し、独立再読取りなしで無条件再実行しない |
| CL-10 | 同じ結果本文が別のSource RevisionやAudienceへ届けられる | 元の試行と受領先を照合し、資格を引き継がない |

各Caseは既存OwnerのScenario・テストへ割り当てる候補であり、独立した検査台帳や第二のFixture正本を作る提案ではない。正常例と負例は、同じSource・World・Attempt・適用権限を固定した条件で比較する。

## 6. P0からの実装順と統合条件

**P0-A：TEI Studio自己再構築を最優先する。** 現行Wails Native同一プロセスを基準に、生成されたInspector、esbuildの実入力、生成JS、Go埋込み、Desktop ArtifactSet、生成Stage1によるStage2構築、Windows WebView2での実受領をつなぐ。Worker・別Server・IPCの方式比較、SaaSの実行分離、汎用Graph Compilerを先行条件へ増やさない。

**P0-B：通常製品CLIで一つの実Readを完成させる。** 選ばれたQuery・引数・Providerが同一Attemptで呼ばれ、Rowset→Function ContextがConsumer Aと独立Consumer Bへ、それぞれ権限に適合した形で届くことを確認する。W1〜W3の限定観測は活用し、W4とCurrent Grant、Native SQLの完全性の問題を残したまま「全件」と表示しない。

**P0-C：公開する操作の資格を個別に評価する。** Script Launch、B2C、REA、組織の各Consumerは、A/Bの成功を借りて自動合格にしない。現在のSource・意味の受理・Audience・Host Grant・外部作用を必要に応じて再確認する。

**P1：PluginとTargetの縦断的な適合へ拡張する。** 一つの意味版に対する段階別Contributionを、選抜、実Byte、Build、実Call、最終Consumerまで確認する。独立したもう一つのRunnerやRegistryを開発する前に、既存Contract Resolver／Driver／Scenarioの不足を具体的に調べる。

**将来：共有Studio・SaaSと複数業務領域を評価する。** Source Workspace、Git Branch、DB Instance・Schema、開発実行環境、Principal/Grantを独立に束縛する。WailsをStudioの構成へ選んでも、生成アプリのTargetへWailsやLit、TEI独自Hostを強制しない。

## 7. 更新する計画と判断の担当

この文書を[PRODUCT_COMPLETION_PLAN](../workstreams/PRODUCT_COMPLETION_PLAN.md)から参照し、各Ownerは自分のIssueで「次に何を実際に動かし、何を受け取れば検収するか」を管理する。レビュー時は別PRの成功や設計例を、製品の正式な受入へ昇格させない。

採用を急がない事項は、ToolchainをBuildIntentの正規入力にする粒度、列の由来W4の実装方法、共通のAudience・Grant契約の公開形、Plugin Contributionの永続形式、共有Serverの隔離方式である。これらは実Reader・Builder・Consumerから得られた反例と測定に従ってOwnerが決める。

**製品完了の定義は、要求した仕事を実際の利用者と実Consumerが正しい対象・内容・現在の資格で終えられることである。** 宣言・構成・局所テストを増やすこと自体を完成条件にしない。
