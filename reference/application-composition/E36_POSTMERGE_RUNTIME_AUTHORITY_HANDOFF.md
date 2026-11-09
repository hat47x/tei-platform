# E36：案件Plugin参照実装のmain統合後に残る実行Authorityの接続条件

- **区分**：#1016の読取り専用の構成評価。新たなComposition Profile、Resolver、Lock、Grant発行器、Runtimeの実装提案ではない。
- **確認時点**：2026-10-14。ここでの「統合済み」はGitHub PRのmerged状態についての観測であり、現在のEnvironmentでの実行資格ではない。
- **所有境界**：案件選択・差分の評価は[#1016](https://github.com/hat47x/tei-platform/issues/1016)、Service Contract／Provider適合・受理は[#951](https://github.com/hat47x/tei-platform/issues/951)、Generic Runtime Ref／Execution Authorityと現行実行への接続は[#976](https://github.com/hat47x/tei-platform/issues/976)、サブプロジェクト・成果物の管理は[#1251](https://github.com/hat47x/tei-platform/issues/1251)が担う。

## 前段から何が成立したか

| GitHub上の証拠 | 観測可能な範囲 | それだけでは成立しないもの |
| --- | --- | --- |
| [#1567](https://github.com/hat47x/tei-platform/pull/1567)、main統合（merge `94a026d1034b60c975e5131bf6e6f7a0a0ac2a6a`） | 案件Owned Go助言とLit画面、既存Compositionを用いる診断Bridge、受注明細へのread-only接続。実Go/Chrome局所CIの記録 | 正式Service Contract、受理済みProvider、受注Actionの認可、製品配備 |
| [#1781](https://github.com/hat47x/tei-platform/pull/1781)、main統合（merge `06270f16d1d5abd0e018f0cd42bcb376271f8830`） | Generic Runtime Ref候補でContract／Artifact／Grantの欠落・差替え、二重Authorityを拒否するReference | 合成候補のissuerによる正式Acceptance、実Artifact起動 |
| [#1789](https://github.com/hat47x/tei-platform/pull/1789)、main統合（merge `0a25cb7e055bc5df70868f49ea55baac1af5e09a`） | 案件Goの`Evaluate`を実子プロセスで起動し、`os.Executable()`の実bytesをSHA-256照合。既存`StdioProcessLauncher`／`HostSandboxLauncher`を経由 | 配布用Plugin Artifactの署名・来歴・FD固定、実OS隔離、運用Lease、正式なAction認可 |
| [#1691](https://github.com/hat47x/tei-platform/pull/1691)、main統合（merge `43c510683308acf6784efbb2946f871e1d33ae61`） | 別経路の業務見積りCandidateでCanonical認可→Web Host→Composition→Service Carrierの参照縦断（試験用認証・Guard／Verifier） | 案件Go助言自体の正式な業務Actionへの採用や、両経路の同一Artifact／Authority接続 |

**E35との関係**：[E35](E35_PROJECT_CI_AUTHORITY_BOUNDARY.md)は#1567の*過去の固定HEAD*について、Go二プロセス・Chrome・PR健全性のCI事実を照合した記録である。現在のmainの状態や#1789の追加保証を示すものに書き換えない。E36は後から統合された別個の証拠を、重複実装せずに接続するための境界整理である。

## 次の最小縦断 — 既存の所有者でつなぐ

実行時に独立して判定しなければならない条件を順序づける。前段が通っても後段の成立を推定しない。

1. **案件Owner／意味**：`reference-project-customer-advice`は顧客コード・メモからレビュー提案を返す案件補助であり、`order_entry`の受注Validationや受理済みActionではない。接続先のAction・入力意味・read-onlyな作用域・所有者を明示する。試験fixtureの業務認可を実運用Granted Actionに見立てない。
2. **契約と選択（#951）**：既存のService Contract／Provider受理機構で、Contract ID・version・fingerprint・scope・Provider実装・Conformanceの証拠を扱う。案件用Bridgeの`v0`、説明的Shape、合成Digestは受理証拠にできない。既存Registry選択とResolution Lockを利用し、第二Resolverを作らない。
3. **実成果物（#943／#1251／#976）**：テスト実行ファイルではなく、案件Ownedソースから独立に構築した**配布対象の実Artifact**を検証する。Build入力とartifact digestを区別し、検査対象のbytesと起動対象のbytesが同一であることを確認する。署名・発行主体や実行内容の固定を必要とする環境では、既存Verifier／Trustの責務に委譲する。単にGoのSHA-256を計算できるだけではAuthenticityは成立しない。
4. **現在のEnvironment Authority（#976／#951）**：Host側の信頼されたApplication文脈から有効Planを選び、正確なRuntime Ref、Grant、Permission、Resource、Lease、失効世代を実行直前に照合する。BrowserのHeader／Body由来のApplication・Provider・Action値をAuthorityの選択根拠にしない。既存のResource Broker／Host Sandbox／Service Carrierを通し、独自のPermission StoreやGuardを製品機構に昇格させない。
5. **呼出しと観測**：認可済みAction、選択済みProvider、固定Artifactが同一要求に属することを追跡できる形で呼び出す。正常・検証失敗・実行直前の失効・契約drift・対象Target非対応を分離する。拒否時には子プロセス起動・外部Effectが起きないことを対象の限定試験で確認する。

### 完了証拠を混同しない

| Evidence段階 | 必要な証拠 | 成功しても主張できないこと |
| --- | --- | --- |
| Reference構成 | Source上の明示wiring／既存Registryの選択 | Provider受理・実行可否 |
| 試験用Authority | 合成Candidateで欠落・二重化・driftを拒否 | 本番Issuer Trust・Grant |
| 実ファイル起動 | 実bytes Digest一致、別OSプロセスの動作 | 配布Artifactの署名・来歴・Permission |
| Host受理と現在権限 | 同一Runtime Ref／Environmentの現行受理・失効／Lease証跡 | 受注ActionのAccepted意味 |
| 業務Application | 信頼済み文脈から正式Actionへ到達し、意味・権限・Effectを分離した実Consumer試験 | 別のTargetや別案件へ自動的に適用可能なこと |

## 今回の判断と並行作業の切り分け

- #1567／#1781／#1789の既存コードを**移植・再作成しない**。#1691で実装済みのCanonical認可を案件Goへ暗黙流用しない。次の実装担当は#951／#976側で、上記2〜4を*同一要求*へつなぐ最小差分を所有する。
- #1016側では、既存Source／Registry／Lock／Environment bindingから導出できない**案件固有の選択差分**と、その変更時の衝突を観測する。実Consumerの接続作業時間・AIへの質問数は未計測のまま保ち、実測なしに独立Composition Profile採用を結論しない。
- 各Pluginを同型のmanifest・専用`go.mod`・一律の性能証明に従わせない。案件ごとの生成AIによる小さな実装を許容し、**使用する環境に必要な**契約・権限・性能・適合性のみを具体化する。
- この文書自体はSource／実プロセス／CIを新たに実行していない。GitHubの統合済みPR metadataを読んで残課題を分類した記録であり、Production資格の付与でもIssueの完了判定でもない。
