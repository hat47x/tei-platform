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

**重要な反例：** `reference/workspace` の `recent_orders.sql` で `ORDER BY ... DESC` を `ASC` へ変更すると、SQLの意味は変わりますが、現在の `derivedbridge.FromIndex` が投影する型付きグラフにはSQL本体の変更が現れません。この場合、`sourceChanges` とモデルRevisionの変更を保持し、`source-changed-without-typed-graph-diff`として `unresolved` にします。グラフ差分なしを「変更なし」「検証済み」とは解釈しません。

有効な型付きグラフでConsumer自身または型付き依存先の変更が確認されれば`recheck`を提案します。未解決参照、Consumer不在、Rule fingerprintの差、管理対象Source変更の影響不明は`unresolved`です。いずれも`reuse-candidate`を発行せず、`executionAllowed=false`です。

二つの`Load`は**各々が原子的なmulti-file snapshotではありません**。Source Indexが扱わない未管理ファイル、Runtime Effect、旧資産の意味同値、Provider artifact、外部データや権限、独立Expectationとreceiptの真正性を証明しません。`sourceChanges`はビジネス意味の差分ではなくバイト指紋の差分です。実際にSourceが変わっていない場合でも、独立した検証証拠がないため再利用を認定しません。

次段階では、#1180/#1163の受入根拠と`Source Revision + Scenario/Oracle + Target/Provider`を照合し、未管理Sourceの扱いを導入します。Graph投影の欠落を推測で補う実装や新たな品質Gateは作りません。

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
