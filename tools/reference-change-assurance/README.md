# 変更に応じた検証根拠の再利用 — 読取り専用Reference

関連：[#1445 開発工程プラグイン](https://github.com/hat47x/tei-platform/issues/1445)、[#1180 Driver Scenario](https://github.com/hat47x/tei-platform/issues/1180)、[#1163 統合Qualification](https://github.com/hat47x/tei-platform/issues/1163)、[#1067 共有Plugin昇格](https://github.com/hat47x/tei-platform/issues/1067)。

## 目的

変更のたびにすべてのテストを実行するのではなく、**過去の検証結果が現在も同じ主張を裏付け得る範囲を示す**アルゴリズムを、小さな合成データで検討します。受理済みの業務意味、契約、構成、Scenario、Target、実行権限はそれぞれ既存の責務を持ち、本ツールに第二の正本を作りません。

既存の `tools/reference-purpose-preflight`（#1445 PR #1632）は**業務規則の意味が未確定か、実行時の値が未取得なのか**を区別します。本ツールは別の問題である**過去のEvidenceを候補として再利用できるか**だけを扱います。互いに独立した実証であり、製品への統合はまだ行っていません。

## 実行

```sh
go run ./tools/reference-change-assurance -input tools/reference-change-assurance/example.json
go test ./tools/reference-change-assurance -count=1
```

JSONに記録するのは、変更対象・変更された側面、主張が依存する対象、依存関係の網羅性、影響する変更の側面、現在の比較文脈、過去の試験受領記録、実行時に再確認が必要な条件です。

出力の `action` は次の意味を持ちます。

| 値 | 意味 |
| --- | --- |
| `reuse-candidate` | 完全な依存閉包、非影響の変更、同じ比較文脈、独立した期待値による過去の成功という**fixture内の宣言条件**が一致した。履歴Evidenceの参照候補に過ぎない。 |
| `recheck` | 影響する変更、古い実行先・Provider、独立の期待値欠如などにより検証をやり直す提案。 |
| `unresolved` | 依存閉包が不透明、または変更の側面がその主張の影響規則に含まれず、再利用可否を判断できない。 |

どの場合も `executionAllowed=false` です。`reuse-candidate` をPASS・実行許可・Release Gate合格として扱ってはいけません。`runtimeChecks` は過去の静的な根拠では代替できない現行権限や副作用確定条件を列挙します。これらの検査や実行はこのプログラムでは行いません。

## 実Workspaceの型付き変更影響（POC-8）

`#1120` の既存 `workspace.Load` → `index.Build` → `modelref/derivedbridge.FromIndex` → `modelref.ReviewImpact` を**読取り専用**でつなぐモードです。架空のGraphを新たに正本化せず、`reference/workspace` の受注アプリの宣言済み相互参照を使います。

```sh
go run ./tools/reference-change-assurance \
  -workspace reference/workspace \
  -scope order-entry-reference \
  -change-subject-kind semantic-relation \
  -change-subject-id sales.order.customer \
  -change-kind version-changed \
  -consumer-kind application-pattern \
  -consumer-id order_entry.customer_reference
```

変更は**ユーザーが仮定した検証シナリオ**であり、実際のSource変更を観測・証明したものではありません。出力の `impact` には既存APIの診断と影響先が含まれます。変化先と対象Consumerが型付き参照上で直接・間接に結ばれた場合は、他の投影ギャップが残っていても安全側に `recheck` を提案します。Graph検証の失敗、未知の変更種類、Consumer不在は `unresolved` とします。**影響先として現れなかった場合も、投影ギャップの残る現行Workspaceでは `unresolved`** であり、影響なしの証明とは扱いません。現行受注管理WorkspaceではDatabase/Database Contract関連の型が投影対象外となり、`projectionUnresolved` に残ります。型付き参照が届かないConsumerも、過去のOracleとreceipt、依存閉包が結び付いていないため**`reuse-candidate`には昇格させません**。

既存の合成fixtureモード `-input` は保持しますが、合成fixtureの `coverage=complete` とは別の判断権限です。実Workspaceモードは `executionAllowed=false` を固定し、Runtime Authority・Effect・Provider適合・Scenario期待値を確認しません。Business ProcedureのCanonical意味を勝手に生成せず、実Sourceのrevisionと署名・権限も確認しません。

### 残る接続課題

- #1120のTyped Graphが描く**観測範囲外の依存**（nativeコード、動的な外部Effect、未投影のBusiness手順）を推測して「非影響」とみなさない。
- #1180のScenario・期待値、#1163のEvidence Packetを介し、根拠とOracleを対象の変更前後Sourceに結びつける。
- #951のProvider/Target fingerprintと実Runtime権限を別責務として保持する。
- 将来の実Change Diffは `ReviewImpactDiff`（前後2スナップショット）を使用し、今回の`-change-kind`仮説を実際の差分と混同しない。

## 二つの実WorkspaceのSource差分と型付きImpact（POC-9）

`-workspace`は **仮定されたChange** の評価でした。`-before-workspace`と`-after-workspace`を併用するモードでは、二つのディレクトリを個別に`workspace.Load`し、既存`workspace.BuildSourceIndex`の **管理対象Source指紋** と `sourceindex.ModelRevision` を比較します。型付きグラフは既存の `modelref.ReviewImpactDiff` が比較し、その結果を検証計画の**候補**として取り込みます。

```sh
go run ./tools/reference-change-assurance \
  -before-workspace /path/to/before/tei-workspace \
  -after-workspace /path/to/after/tei-workspace \
  -scope order-entry-reference \
  -consumer-kind application \
  -consumer-id order_entry
```

このコマンドはWorkspaceを変更しません。出力には変更前後のモデルSource Revision、`sourceChanges`（追加/削除/指紋変化した管理対象ファイル）、型付き`impact`、投影で未解決の参照、判断理由を含めます。既存の`-input`と`-workspace`も維持します。

型付き参照グラフの投影規則（`modelref.Rules`）自体が変化して影響範囲の網羅性を保証できない場合、全体の判定は`unresolved`のままです。ただし、同時に型付き参照から対象Consumerへの直接変更や波及が**明確に観測されている場合**は、既存`reasonCodes`に`typed-projection-rules-changed`と`consumer-directly-changed`／`typed-reference-impact-candidate`の両方を残します。「既知の再確認対象」と「依然として未知の範囲」を一つの`action`へ押し込めないための診断であり、`recheck`を`PASS`へ変更したり、変更閉包の完全性、過去の証拠再利用、実行許可を認定したりするものではありません。

**重要な反例：** `reference/workspace` の `recent_orders.sql` で `ORDER BY ... DESC` を `ASC` へ変更すると、SQLの意味は変わりますが、現在の `derivedbridge.FromIndex` が投影する型付きグラフにはSQL本体の変更が現れません。この場合、`sourceChanges` とモデルRevisionの変更を保持し、`source-changed-without-typed-graph-diff`として `unresolved` にします。グラフ差分なしを「変更なし」「検証済み」とは解釈しません。

有効な型付きグラフでConsumer自身または型付き依存先の変更が確認されれば`recheck`を提案します。未解決参照、Consumer不在、Rule fingerprintの差、管理対象Source変更の影響不明は`unresolved`です。いずれも`reuse-candidate`を発行せず、`executionAllowed=false`です。

二つの`Load`は**各々が原子的なmulti-file snapshotではありません**。Source Indexが扱わない未管理ファイル、Runtime Effect、旧資産の意味同値、Provider artifact、外部データや権限、独立Expectationとreceiptの真正性を証明しません。`sourceChanges`はビジネス意味の差分ではなくバイト指紋の差分です。実際にSourceが変わっていない場合でも、独立した検証証拠がないため再利用を認定しません。

次段階では、#1180/#1163の受入根拠と`Source Revision + Scenario/Oracle + Target/Provider`を照合し、未管理Sourceの扱いを導入します。Graph投影の欠落を推測で補う実装や新たな品質Gateは作りません。

## Driver Scenarioと生成CLI PlanのSource Revisionへの読取り接続（POC-10）

既存の`tei scenario generate`（#1180）が保存する`tei.cli-scenario-plan/v0`を、今回の検証ツールが**読取りだけ**で照合します。二つのWorkspaceの`Source Revision`差分と、既存のScenario/Fixture/World/Plugin Lockを参照した同一Planの現行性を、一つの出力に並べます。ScenarioやCLI Planの新しい正本、第二のDriver、実行機構は作りません。

```sh
go run ./cmd/tei scenario generate \
  -scenario reference/driver-scenarios/order-entry-pattern-smoke.json \
  -workspace /path/to/after-workspace > /tmp/order-entry-plan.json

go run ./tools/reference-change-assurance \
  -before-workspace /path/to/before-workspace \
  -after-workspace /path/to/after-workspace \
  -scope order-entry-reference \
  -consumer-kind application -consumer-id order_entry \
  -scenario reference/driver-scenarios/order-entry-pattern-smoke.json \
  -plan /tmp/order-entry-plan.json
```

`-scenario`と`-plan`は対で指定します。`-plan`は既存CLIで生成・保存したPlanの**完全一致**を求め、Scenarioの原文SHA-256、すべてのFixture原文SHA-256、解析済みWorldのSHA-256、Plugin LockのSHA-256、Step数・順序・CLI argvを照合します。編集済みPlan、staleなScenario、Fixture、World、Plugin Lockや外部ディレクトリを指すFixtureは受け付けません。Fixtureを持つStepは既存`scenariotest`の厳格なParserで読みます。

出力は`scenarioId`/`scenarioSha256`/`fixtureSha256`/`planMatches`と、POC-9の`diff.beforeSourceRevision`/`diff.afterSourceRevision`を並置し、同じ読取り時点で観測したSource Revisionを`observedAfterSourceRevision`として表示します。

**重要**：現行の生成CLI Planは**Source Indexの`ModelRevision`そのものをpinしていません**。一方、`WorldSHA256`は保持するため、Canonical Worldに取り込まれるSQL本文の`DESC → ASC`変更は既存Planをstaleとして拒否できます。逆に、管理対象Semantic Source末尾の空行追加など、**Byte指紋とModelRevisionは変わってもWorld表現が変化しない編集**についてはPlan指紋が一致したままです。`planPinsSourceRevision=false`を明示し、この違いを`diff`の指紋変化とともに残します。Plan照合の合格を「受理済みSource Revisionへの暗号学的な束縛」とみなしてはいけません。

`qualification=unresolved`、`executionAllowed=false`で固定し、Scenarioに明示した期待値と合成Fixtureの存在を**独立Oracle認定・本番Provider観測・実行済みTest PASS**へ自動昇格させません。Scenario定義・Fixture、Planが一致しても、実TargetのEffect/Permissionと業務意味・受理済みSource、Scenarioの網羅性は未証明です。#1180のCLI/Web生成と#1163の正式Evidence Packetは既存の担当機構のまま保持します。

次は、既存のAccepted Source RevisionとExecution Receiptから**実際に照合可能な証明項目だけ**を追加し、証拠が欠ける場合は引き続き`unresolved`へ留めます。

## #1163の既存8件の合成Web receiptとSource Revisionの照合（POC-11）

#1163 / #1119の既存`tools/reference-critical-path-evidence-audit`はEvidence Packetの**形式・申告欄の整合**を検査しますが、指定されたreceiptの実バイト列を読んでSource Revisionと照合する機能ではありません。本モードは**正式なEvidence Packetの代替ではなく**、既存の`live-r1-smoke.mjs`がリポジトリ外に保存する8個の合成専用receiptを、変更前後の実Workspaceと**読取りだけで**照合します。

```sh
go run ./tools/reference-change-assurance \
  -before-workspace /path/to/test-only/r0-workspace \
  -after-workspace /path/to/test-only/r1-workspace \
  -scope order-entry-reference \
  -consumer-kind application -consumer-id order_entry \
  -archived-witness-dir /path/to/unpacked/existing-eight-e1-receipts
```

8件の元の名前・`evidenceNature`・`integrationHead`と各ファイルのバイトSHA-256を記録したうえで、主に下記の**receipt内の主張**を照合します。

- `workspace-r1.json` のR0/R1と`decisionModelRevisionAfter`を、二つの`BuildSourceIndex`から計算した`ModelRevision`と比較する。
- `target-launch.json` のTarget build revision、application、起動前に観測したSource R1、`web-positive.json`と`web-negative-unit-price.json`のR1申告を照合する。
- `n11-unobserved-control.json`のCommand完了・Observation read試行・read結果未記録、および`stale-source-control.json`が異なるR0を実際に読んだと**申告しているか**を調べる。
- 8件が別々のファイルとして存在すること、予期しない種類・ヘッド相違・Source版相違・読めないファイルを診断・拒否する。ファイルのSHA-256は証拠の真正性や独立採取の保証ではありません。

出力の`linkStatus=declared-links-consistent`は**試験ハーネスが出した申告の文字列同士が整合する候補**でしかありません。`qualification=unresolved`・`executionAllowed=false`は固定です。独立DB観測、人間の受理、Targetが実際にR1版を強制したこと、現行Runtime権限、第三者の署名・artifact真正性、#1180のScenarioそのものがこの#1163 Web試験で実行されたことは**一切認定しません**。既存Evidence preflightのE1/E2資格判定や#1119のGateも呼び出しません。証拠の回帰テストは合成receiptを用いるもので、本番実行の実証ではありません。

既存`-scenario/-plan`モードとの同時指定は禁止します。#1180のCLI Scenarioと#1163のWeb合成試験を、同じScenario ID・同一実行だと誤認しないためです。前後Workspace・証跡の収集時点の原子的整合も保証しません。#1163の実receipt 8件はアーカイブされる短期成果物であり、現行リポジトリに保存されているわけではありません。

## 旧E1合成証拠の保存済み原本と任意のSource再現を接続（POC-12）

#1163/#1119が実施した[GitHub Actions #38267656696](https://github.com/hat47x/tei-platform/actions/runs/38267656696)（2026-10-12、成功）には`tei-e1-synthetic-38267656696-1`という8 receipt+manifestのArtifactが存在します。GitHubが2026-10-15時点で返した当該Artifact IDは`11755671316`、ZIP digestは`sha256:8519e0bb06377f1943a97bb4b7f839f46df339ec12690df3f465630902ad235a`、期限は**2026-10-19 15:05:57 UTC**です。これらは取得時のGitHubメタデータの申告であり、本ReferenceがGitHub上の実行者を独立に認証したわけではありません。PR head `ef0ea673...`と受理当時のGitHub Actions PR merge checkout `466b1d100...`は**異なるコミット**であり、manifestの`exactCheckoutHead`は後者です。

8件のバイト原本とmanifestはすでに`plan/evidence/tei-e1-synthetic-38267656696/`と`plan/evidence/TEI_E1_SYNTHETIC_MANIFEST_20261012_38267656696.json`へ保全されています。既存の`tools/verify-critical-path-e1-archive.py`が固定manifest SHA-256、全8ファイルのSHA-256、R0/R1/R2と合成実行の申告を検査します。**この検査を本ツールで再実装しません。**

```sh
# 読取りのみ：#1119既存の原本検査を再利用する
python3 -B tools/reference-change-assurance/e1_archive_reconcile.py

# 任意：GitHub Actionsから期限内に取得済みの元ZIPがある場合
python3 -B tools/reference-change-assurance/e1_archive_reconcile.py \
  --artifact-zip /path/to/tei-e1-synthetic-38267656696-1.zip \
  --artifact-digest sha256:8519e0bb06377f1943a97bb4b7f839f46df339ec12690df3f465630902ad235a

# 任意：**実際の当時の**R0/R1 Workspace全体のスナップショットを取得・復元できた場合のみ
python3 -B tools/reference-change-assurance/e1_archive_reconcile.py \
  --before-workspace /path/to/historical-r0 \
  --after-workspace /path/to/historical-r1

# オフライン負例を含む回帰テスト
python3 -B tools/reference-change-assurance/test_e1_archive_reconcile.py
```

ZIP照合は**呼出元が別に確認したGitHub ArtifactのSHA-256**を必須とし、ダウンロードしたZIPの全9エントリが保全済みmanifest/8件とバイト一致するか確認します。余分なエントリ、同名重複、外部へのパス、Symlink、過大ファイル、差替えを拒否します。GitHub APIからの取得や資格情報の取扱いは行いません。

Source照合は両方のディレクトリが指定された場合に限り、**既存**`go run ./tools/reference-change-assurance -before-workspace ... -after-workspace ... -archived-witness-dir ...`を起動し、その出力と保全manifest内の`sourceBefore/sourceAfter/exactCheckoutHead`を突き合わせます。さらに`workspace-r1.json`に保全された**変更対象2ファイル**（`applications/order-entry/application.tei`、`semantic/sales/validation.tei`）について、`changedManagedSourceFiles`と`changedSourceDigests`の変更前後SHA-256を、既存Source Indexから得た`sourceChanges`のpath・種別・前後指紋と**完全一致**比較します。Revision文字列だけ一致しても、変更ファイル一覧が欠落・増減・差替えされていれば拒否します。照合済みの管理対象ファイル数を`changedManagedFilesVerified`に表示します。現在のmainを当時の受理R1とみなすことや、receiptからSource全体を復元したと偽ることはありません。

どちらも入力されなければ`artifactZip.status=not-supplied`、`sourceSnapshots.status=not-supplied`を明示します。この追加照合は同一の保存済み合成receiptと、呼出元が提供したWorkspace内容の対応を厳密化するもので、receiptの発行元・Sourceの取得来歴・原子的時点・管理対象外のファイルを認証するものではありません。古い8件のreceiptは**一時R0/R1 Source全体のコピーではありません**。Source版を突き合わせるためには、その時点の管理対象Sourceを含むWorkspaceスナップショットが別に必要です。整合しても試験用の履歴記録に過ぎず、常に`qualification=unresolved`、`executionAllowed=false`、`productGate=UNQUALIFIED`を返します。RuntimeでのVersion拘束・実DB独立Observation・人の受理・#1180 Scenarioとの同一実行を認定しません。

## 収録する反例

- 表示名だけの変更：業務規則の検証は再利用候補、Web表示の検証は再確認。
- 承認閾値の変更：境界規則は再検証し、閉包上無関係な受付操作には波及しない。
- Providerの成果物差し替え：業務意味の検証は保持できても実Providerの適合は再確認。
- nativeコードに不明な外部副作用：`opaque` として保留。
- 実行時の権限失効：静的なEvidence候補があっても `runtimeChecks` を必ず残す。
- 未対応の変更種類・Targetビルド差異：安全だと推定しない。
- 実装自身が導いた期待値：独立したoracleの代わりにできない。

## 未実装の重要な部分

本ツールの `coverage=complete`、過去のreceiptの `outcome=pass`、`oracleOrigin=independent`、fingerprintは**すべて合成fixture側の自己申告**です。ソース解析からの依存閉包の証明、Scenarioの期待値の独立性検証、署名・改ざん検出・受領記録の信頼性確認、現行のShared Acceptance/Authorityとの照合は未実施です。したがって本ツールの判定は、実案件のテスト省略を正当化しません。

次の統合候補は、[#1120](https://github.com/hat47x/tei-platform/issues/1120)の型付き参照と差分、[#1121](https://github.com/hat47x/tei-platform/issues/1121)のLowering、[#1180](https://github.com/hat47x/tei-platform/issues/1180)のScenario/fixture/独立Expectation、[#951](https://github.com/hat47x/tei-platform/issues/951)のContractとexact Artifact、[#1163](https://github.com/hat47x/tei-platform/issues/1163)のEvidence packetの**参照・検査**です。新しい品質Gate、認可、受理DBをこのツールへ組み込まない方針です。
