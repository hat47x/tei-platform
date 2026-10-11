# E12：実E7/E11を組み合わせた読取り専用CLI

- **状態**：検証用CLI。Goの完全な構成判定、Production用Composer、受理済み業務意味、Runtime権限、全依存閉包ではない。
- **管理**：[Issue #1016](https://github.com/hat47x/tei-platform/issues/1016)／[Draft PR #1051](https://github.com/hat47x/tei-platform/pull/1051)。
- **前段**：[E7：Source出典比較](E7_PAIRED_OBSERVATION.md)、[E11：型付きImpactのレビュー接続](E11_TYPED_IMPACT_REVIEW_GATE.md)、[#1120のGo候補](https://github.com/hat47x/tei-platform/pull/1139)。

## なぜCLIが必要か

E11には、E7の`{report,index}` Envelopeと#1120のGo `derivedbridge.Projection`／`Validate`／`Impact`のJSON形状を**一つの読取り専用レビュー候補へ渡す関数**が既にある。E12は同じ照合関数を複製しない。

[E12 CLI](e12-bound-review.mjs)の責務は、**実際のJSONファイルを読む手順、入力制限、出力と終了コードの境界**だけである。E7のSource指紋比較とApplication→Patternの所有検査は既存E11→E7→E6→E4へ委譲する。Goの`modelref.Impact`と`Validate`はまだ別Draftのため、E12はそれらの結果を新たに計算しない。

以前のE12試作はE11の基本関数とE7を直接呼び、同一の根拠がない場合でも`limited-unchanged`を返し得る別経路を増やしていた。並行作業によりE11に**Go実JSON形式を受ける`qualifyPairedCompositionReview`**が追加されたため、重複する判断ロジックを削り、E12をこの**単一E11判定器を呼び出すCLI**へ改めた。E11の新しい判定器はGo Sourceと型付きProjectionの同一snapshotが未立証なら`unverified_shared_snapshot`を返す。E12もこれを尊重し、「変更なし」を偽の合格へ昇格させない。

## 入力の形式と操作

```sh
node reference/application-composition/e12-bound-review.mjs \
  <baseline-e5-e7.json> <current-e5-e7.json> \
  <modelref.json> order_entry order-entry-reference
```

- 第一・第二ファイル：E5の`-paired-evidence`により出力できるE7 Envelope（`{format,qualification,report,index}`）。
- 第三ファイル：Go候補のJSONを呼出し元が`{projection,diagnostics,affected}`に**まとめた未承認の交通整理用Carrier**。正式な永続Schema／Lockを追加するものではない。`projection.graph`は`tei.modelref.v0`、`projection.rules / unresolved / authorityLimits`は#1120のDerived Bridge、`diagnostics`は`modelref.Validate`、`affected`は`modelref.Impact`から提供される想定だが、**現在は同じWorkspace snapshotからの実出力をまだ取得していない**。
- 第四・第五引数：対象Application IDとscopeを明示的に選択する。名前から他Applicationへ自動fallbackしない。

**終了コード**：`1`は引数／入力ファイル／JSONの不正、`2`はレビューがまだ合格できない判定。現在の仕様では**終了コード0を返さない**。たとえ見かけ上のSource/Graphが一致しても、E11が`unverified_shared_snapshot`を返す。CLIの戻り値でCI、deploy、Shared Acceptanceを通過させない。

CLIは入力を変更しない。各入力を16 MiBまでに制限し、失敗時には元のJSON、秘密情報、ファイル名を標準エラーへそのまま表示しない。**この制限自体はReader権限、Source Indexの秘匿、ファイルシステムアクセス制御、改ざん防止を保証しない。** 実機の機微ファイルは使用・共有許可を別途確認する。

## 実行済み局所テスト

```sh
node --test reference/application-composition/e12-bound-review.test.mjs
```

2026-10-10、Node.js v22.16.0の標準テストランナーで**13件PASS、0件失敗**。テストはE12のCLIを実際に子プロセスで起動し、合成JSONファイルの読取りからE11→E7→E6→E4の実モジュール呼出し、終了コード、出力を確認した。

検査した内容：Go候補のJSON形状、Relation→Application所有Patternのレビュー候補、指紋変更、Application不一致、Source定義の食い違い、未知Relation、Graph内のPattern欠落、型付き診断、ファイルサイズ上限、不正JSON、機微文字列の出力抑止。

隔離環境で実際に読み込んだ6ファイルのGit blob SHAを、GitHub上の担当ブランチと照合した。

| ファイル | Git blob SHA |
| --- | --- |
| `e4-index-scope.mjs` | `d2a61dac60b1641e453513e3f72b7c6c2ef43501` |
| `e6-typed-index-bridge.mjs` | `9ca7f39a91a89503a7955940b254754b02581666` |
| `e7-paired-evidence.mjs` | `910ef058ed96d5830f026c0552196682580a20eb` |
| `e11-review-gate.mjs` | `512b7b9711c04306fa11d581b2f8e12d3c63617c` |
| `e12-bound-review.mjs` | `bc245d5469a27bde314c1e2a979769bfb1a6081b` |
| `e12-bound-review.test.mjs` | `f0f2dd807973852c6d57d7dfe4bfa047e6550a37` |

**局所テストの範囲**：合成のEnvelopeと合成のGo形式候補を用いており、`workspace.Load`・E5のGo CLI・#1120のGo Impact・実SUI・Provider・Browserへの結合は未実施。実Repositoryのexact-head Goテスト／全Repository Health／pre-post cleanも未取得。これらのため、Draft PRをmainへマージできる資格はない。

## 次の技術的な受入条件

1. #1120の`derivedbridge`／`Impact`のGo実出力とE5のSource出典を、**同じ固定Workspace・Loader解析済みrevision**から採取する。#1017の解析bytes一貫性を既存Loaderの責務として確認し、Composition専用の第二Resolverを作らない。
2. `order_entry.customer_reference → sales.order.customer → customer.customer`のSource変更について、**実GoによるImpact候補とE7によるSource変更を同じrevisionに結ぶ**。Localな文字列一致やCLIのJSON形状一致だけでは同一snapshotの証明にならない。
3. SUIはSemantic Stage 0の一部だけでは第二Applicationが実装済みとはいえない。Native Commit、CAS、権限に関する既存の実Sourceと接続実行を別Gateで検査する。
4. Go／Nodeの実Workspace focused tests、`tools/repository-health-check.sh`全項目PASS、pre/post clean、current mainとexact headの一致を確認してからDraft統合可否を判断する。

**結論**：E12は既存のSource変更検査と型付きImpactレビューをCLIから試すための道具であり、Application Composition Profileが製品に必要だという結論、あるいはその採用を先取りするものではない。
