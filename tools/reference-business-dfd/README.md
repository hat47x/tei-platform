# Business DFD read-only Reference (#1125)

このツールは、既存の Business Procedure / Handoff Reference と正規化済みの Application Pattern を参照し、**一階層の業務図を描くための派生情報**を JSON / Graphviz DOT で返します。Canonical World や手順の正本は書き換えません。実装形態は独立した薄い Projection adapter の実験であり、TEI の標準 Plugin として登録されたことを意味しません。

## 実行

リポジトリのルートから実行します。

```sh
go run ./tools/reference-business-dfd -format json
go run ./tools/reference-business-dfd -format dot > /tmp/order-entry-review.dot
go test ./tools/reference-business-dfd
```

DOT の描画には任意で Graphviz を使えます。ツール自体は Graphviz に依存しません。DOT の node 配置・装飾は一時的な表示情報で、Procedure の意味や step の同一性を構成しません。

## 図からの局所ナビゲーションと変更プレビュー（G1候補）

`-select kind:id` はSourceを書き換えず、選択要素の参照先と元Source位置を返します。Stepは**Procedure内に限るReference-local識別**であり、同じ文字列のAccepted Taskを作ったり推測して接続したりしません。

```sh
# 電話確認StepのSource・前後の候補遷移・業務Roleを表示
go run ./tools/reference-business-dfd \
  -procedure reference/business-system-design/dfd-order-approval/order-approval-procedure.json \
  -review reference/business-system-design/dfd-order-approval/review-proposal.json \
  -set reference/business-system-design/dfd-order-approval/procedure-set.json \
  -select step:APP-10

# Function側から、そのIDを明示参照するBusiness Stepへ戻る
go run ./tools/reference-business-dfd \
  -procedure reference/business-system-design/dfd-order-approval/order-approval-procedure.json \
  -review reference/business-system-design/dfd-order-approval/review-proposal.json \
  -set reference/business-system-design/dfd-order-approval/procedure-set.json \
  -select pattern:order_entry.order_record
```

選択可能な種類は`step`、`pattern`、`trigger`、`business-role`、`party`、`store`、`information-flow`、`semantic-data`、`semantic-relation`、`semantic-validation`、`semantic-capability`です。Data／Validation側から、そのIDを明示したStepやPattern由来の参照元へ戻ることもできます。ただし逆参照も`reference-only`であり、未受理の業務意味を確定したとは扱いません。探索できないIDは`unresolved`で返し、別IDを類推しません。既存のGo Worldから参照したPattern、Data、Relationも同一性の照合にとどまり、Source IndexやAcceptanceの書込み先を保証しません。

出力の`sourceSnapshots[].sha256`はProcedure・図レビュー案・Handoff setの**具体的な参照ファイルのバイト列**に対する確認用指紋です。実行中に参照ファイルが変化した場合は中断します。ただしWorkspace全体の`modelRevision`、Accepted Sourceとの因果関係、`sourceindex`のScopeを確定するものではなく、`worldRevisionStatus=unverified-no-model-revision-binding`とします。#1120のスコープ付き契約と#1124のStudio側ソース確認に接続するまでは`reference-only`のままです。

未受理の図レビュー案を二つ比較するときは`-review-next`を使います。

```sh
go run ./tools/reference-business-dfd \
  -procedure reference/business-system-design/dfd-order-approval/order-approval-procedure.json \
  -review reference/business-system-design/dfd-order-approval/review-proposal.json \
  -review-next /path/to/changed-review-proposal.json \
  -set reference/business-system-design/dfd-order-approval/procedure-set.json \
  -format json
```

結果には追加・変更・削除の候補差分と、両レビュー案の指紋、元Procedure／Handoff指紋、診断が入ります。Stepの正本やaccepted Worldを変更しません。Control transitionにはまだ独立した論理IDがないため、同じ矢印でも内容が変われば**削除＋追加**として表示し、勝手に同一業務経路とみなしません。この比較はShared AcceptanceのPreviewではなく、`readyForAcceptance=false`かつ`sourceWrites=[]`固定です。

## Source / authority の対応

| 入力 | 意味 | Projection 側の扱い |
| --- | --- | --- |
| `order-entry-business-procedure.json` | 非 Canonical な Reference authoring fixture | `procedure.id` と `steps[].id` を exact identity として参照。actor / trigger は候補のまま保持 |
| `order-entry-procedure-reference-set.json` | 独立した Handoff Reference | `from` / `to` / `condition` のみ投影。手順の自動実行は含意しない |
| `order-entry-dfd-review-proposal.json` | **未受理の図レビュー案** | human / event / schedule の契機候補、順序・分岐・例外、業務 Role・責任割当て、外部主体・概念 Store・情報の授受。いずれも `authority=proposal` |
| `reference/workspace` | 現行 Workspace の正規化 Application Pattern | Step の明示 Pattern ID だけを exact resolution し、Data / Relation / Capability を派生表示 |

## 一階層DFDとして区別する情報

従来の Step 間 `flows` は**業務の順序・分岐・例外候補**であり、情報の授受を表すDFDそのものではありません。今回、これとは独立して `information` Projectionを追加しました。

| 要素 | 用途 | 意味上の制限 |
| --- | --- | --- |
| `steps` | 既存 Procedure Stepを一階層の作業・処理ノードとして表示 | Stepの再分解・自動Workflow化はしない |
| `information.roles` / `responsibilities` | 営業担当や承認担当の業務上の役割とStepへの責任候補 | 認証主体・Authorization Role・RBACと同一視しない |
| `information.externalParties` | 選択した業務境界の外にいる相手方 | 外部API接続や通信実績の証拠ではない |
| `information.stores` | 情報の保管先という概念的な候補 | `semanticData`のexact ID照合はStoreの採用・物理DBの確定ではない |
| `information.informationFlows` | 送り手・受け手・内容（`payload`）が明示された**情報の授受** | 実際に送受信された、業務が完了した、という主張にはならない |

`informationFlows[].from/to` は `kind = step | party | store` と `id` の組で参照します。少なくとも一端がStepでなければ不整合として扱い、外部PartyからStoreへの直接書込み等を暗黙に認めません。内容不明、未解決の参照先、重複IDはSource位置とともに診断します。情報の授受は制御の矢印から推論しません。表示上は全て未受理であることが分かるようDOT上に`proposal`を付けています。

このCaseの`businessRoles`や`externalParties`等は独立の**設計レビュー候補**です。採用済みの業務手順Sourceへ逆書きせず、他プロジェクトへ一般化しません。JSONの`authority=derived-read-only`はProjection自身の読み取り属性です。候補要素自体の権威はそれぞれ`proposal`です。

**「手順ファイル内の Step の並び」と「業務の実際の遷移」は別物です。** 遷移の Source が無いとき、配列順から矢印を自動生成しません。提案経路は DOT では破線として表示します。3種類の契機も runtime trigger / schedule に束縛しておらず、いずれも `runtimeBinding=unbound` です。

`manual-unresolved`、`manual-external-unresolved` の Step は対応 Application Function がなくても valid な業務上の作業候補として残します。`ActorCandidate` は担当業務上の主体であり、Security / Authorization Role へ自動変換しません。`RoleAuthorizationBinding=not-inferred` は認可成功を意味しません。

StepがFunctionを持たなくても、Procedure側が明示したData／Relation／Validation／Capabilityへのdirect referenceは、`steps[].explicitMeaning`に保持します。特に既存の`BP-05`にはApplication Functionがありませんが、2件のValidationをSource位置付きで表示できます。これらは`procedure-reference-only`の参照であり、参照先のIDが既存Worldに一致しただけでは、権限・更新実績・業務完了を確定しません。

JSON の `source.file` と `source.pointer` は当該 Source の位置を示します。存在しない Pattern／direct Semantic参照、複数解決、欠けた Step、無条件の分岐・例外等は source-aware diagnostics として報告します。`unreferencedPatterns` は**この一つの Procedure から参照されていないPattern一覧**であり、Application 全体で未使用という断定ではありません。各件には正本未記載を即時エラーにしない `review-question` を付けます。

## 実証と残る作業

最初の縦切りで確かめるのは #1119 の **V02 / N07**（Function のない manual / external Step が消えないこと）と、図レビュー候補を受理済み意味へ昇格しないことです。unit / integration test はそれらに加え、3種の trigger 候補、例外、Handoff、Source locator、無効参照に加え、業務Role・外部Party・Store・情報Flowの正負例を検査するテストを追加しています。

現時点で以下は**未実証**です。

- Authoritative な Business Procedure の編集と Acceptance / 再読込（#1124）
- 業務Role・情報の授受・手作業の提案を受理済み正本へ反映する編集操作（提案データのread-only投影は実装候補として存在）
- 分岐・契機候補の業務上の確定、runtime trigger / schedule への Binding（#1121）
- Pattern IDや概念Storeからのreference-only Navigationは追加。Accepted Source Indexと一体化したFunction / Data / ProcedureのStudio UI往復は未実証（#1120 / #1124 / #1132）
- 別資料の原文互換、旧製品挙動の再現、actual Target / Provider Evidence

この Adapter によって新しい Canonical Procedure / Workflow / Authorization Role を追加しません。今後「提案された図の修正 → Source の変更提案 → Shared Acceptance → 再投影」を進めるときは、既存契約の owner（#1120 / #1124）と調整します。
