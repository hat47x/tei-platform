# reference-plugin-capability-coverage

プラグインが `provides` で宣言した能力のうち、**どの環境の Binding からも選ばれていないもの**を一覧する、読み取り専用の参照です。選ばれていないことは欠陥の証明ではありません。

~~~text
go run ./tools/reference-plugin-capability-coverage/cmd/tei-plugin-capability-coverage \
  -workspace reference/workspace
~~~

Workspaceは変更しません。Provider解決・実装選択・権限発行はしません。

## 数える選択

| Binding | 数える | 理由 |
| --- | --- | --- |
| `deployment-executor` | する | 能力IDをそのまま指す |
| `observation-provider` | する | 同上 |
| `evidence-provider` | する | 同上 |
| `database-instance` | する（技術名で解決） | 指すのは提供技術名（`postgres`）。能力IDとは文字列で比べず、`Capability.SupportsDatabaseProvider` で解決する（既存の validation と同じ問い） |
| `lease-policy`・`grant`・`external` | しない | 選択ではなく、資源・権限・外部要件 |

`database-provider` も、instance の提供技術に対応する能力が選ばれたものとして扱います。技術名と能力IDを文字列で比べないため、`postgres` と `database.postgres` の食い違いは起きません。

## 出力

- `capabilities[]`: 宣言ごとの `status`（`selected`／`unselected`）と、選んだ `environments[]`。
- `unselectedCount`: 選ばれていない宣言の数。
- `notes[]`: この一覧が示さないこと。

## 扱っている範囲

- 宣言であり、実装の存在・適合・信頼・権限ではありません。
- `selected` は「環境がその能力IDを名指しした」ことだけを示します。
- 読み取り専用です。`-workspace` の外へは書きません。

## 未実装

- `database-provider` の解決が、instance と能力の対応を**宣言**から引くこと（現行は能力側の `databaseProviders` を引いており、instance 側の宣言ではありません）。
- Bundle経由の間接選択を、選んだ主体ごとに分けること。
- 実Provider・実Host・権限との照合。

実行記録は [`plan/evidence/PLUGIN_CAPABILITY_COVERAGE_REFERENCE_2026-11-08`](../../plan/evidence/PLUGIN_CAPABILITY_COVERAGE_REFERENCE_2026-11-08/README.md) にあります。

## 履歴Evidenceの再利用境界（2026-11-08／#3267と連携）

このReferenceの[2026-11-08初回観測記録](../../plan/evidence/PLUGIN_CAPABILITY_COVERAGE_REFERENCE_2026-11-08/README.md)は、**当時の導出規則で「12宣言中6能力が未選択」と数えた歴史的証拠**である。その後、main統合済み[#3350](https://github.com/hat47x/tei-platform/pull/3350)・[#3363](https://github.com/hat47x/tei-platform/pull/3363)で、DB Instanceの技術名をCapability IDの文字列と比較するのでなく、既存Validatorと同じ`SupportsDatabaseProvider`により判定する規則へ更新した。**初回の生JSONやREADMEの数値を「現行最新の未選択数」へ読み替えたり、過去のraw記録を上書きしたりしない。**

`historical_policy_evidence_test.go`は、実`reference/workspace`から現行Ruleを通した読取結果と、過去に保存された生JSONを比較する。Schema/RuleVersionというラベルが一致しても`ruleFingerprint`が違えば、**同じRuleで得た現在の選択証拠として流用できない**。同じPlugin/Version/Kind+IDのDB能力について、過去`unselected`→現行`selected`となった行を判別する。変更したRuleや対象Workspaceの実消費Source/Scopeが確定しない場合も再確認が必要となる。

この試験で成立するのは**宣言／Environment Binding由来の局所差分**だけである。現行の`selected`はProvider一意解決、実Go Link/Load/Call、Trait履行、現在Grant、Test PASSを意味しない。未選択の任意能力を欠陥と分類しない。Verified BOM（#3267、RQ-036）へ渡すときはRuleの来歴・同一Source/対象Scope・exact Lock/Target/Artifact・独立Expectedと同一Attemptを**別々に照合**する。BOM用の第二Resolver/Runner/Evidence Storeは設けない。

**検証状況**：新試験コードはmainから派生した独立Draftで実行対象。exact-head CI結果が揃うまでは新しいPASSを宣言しない。実Provider/Host呼出しと正式BOM発行はNOT RUN。
