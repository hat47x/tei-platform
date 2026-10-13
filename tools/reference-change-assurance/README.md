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
