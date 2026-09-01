# ADR-0017: Application PatternをApplicationが所有する意味として検証する

- 状態: 採用
- 日付: 2026-09-01

## 背景

TEIでは、`browse`、`record`、`reference`、`header-detail`、`command`のように、業務システムで繰り返し現れる構造を再利用可能なApplication Patternとして扱う構想があります。

これらは画面設計と強く関係しますが、単なる画面テンプレートではありません。たとえば`reference`は、入力欄に検索ダイアログを付けるというUI上の都合だけでなく、「このApplicationが別の意味対象を参照し、その解決に特定のDatasourceやCapabilityを利用する」というApplication側の意図を含みます。`command`も、ボタンを表示することではなく、業務上の操作をCapabilityへ結び付ける意味を持ちます。

一方、これらをSemantic Kernelのprimitiveとして早い段階から固定すると、最初のWeb画面で見えた形を業務上の意味そのものと誤認したり、将来のDesktop、API、MCP、VUIなどへ不要な制約を持ち込んだりする可能性があります。

逆に、すべてをProjectionやEditorの表示情報として扱うと、Driverと複数Targetで共有すべきApplication上の意図が失われます。Targetごとに同じ意味を別々に記述することになれば、RQ-011やRQ-017とも整合しません。

そのため、v1 Reference Applicationである`order_entry`を使い、Application Patternをどの層が所有するかを明示したうえで実証を進めます。

## 判断

v1の設計検証では、Application Patternを**Semantic KernelのprimitiveでもTarget固有の画面テンプレートでもなく、Applicationが所有するCanonicalな意図**として扱います。

ただし、`browse`などの名称や現在の分類をTEIの永久的な標準語彙として固定するわけではありません。v1ではPatternをApplication層へ置くことで実証可能な境界を作り、その後のDriver、第一Web Target、第二Targetから得られるEvidenceをもとに分類や粒度を見直します。

概念上の関係は次のように整理します。

```text
Semantic上の意味
  Data / Relation / Capability / Type
            ↑
            │ 参照
            │
Application上の意図
  browse / record / reference / header-detail / command
            ↓
            │ Projection
            ↓
Presentation / Interaction
            ↓
Web / Desktop / Mobile / API / MCP / VUI
```

## Application Patternが持つもの

Application Patternには、Targetが変わっても残るApplication上の意図だけを持たせます。

たとえば次のような情報です。

- どのSemantic DataやApplication Recordを対象とするか
- どのQueryを読み取り経路として利用するか
- どのCapabilityを業務上の操作として呼び出すか
- どのRelationやReferenceをApplication上で利用するか
- どのDatasource aliasを利用するか
- read-onlyか、変更を伴うか
- ValidationやAuthorizationとどのように接続するか
- HeaderとDetailを一つのApplication上の編集単位として扱うか

これらを一つの巨大なPattern構造へ押し込む必要はありません。PatternからCanonical IDで別の定義を参照できる形を優先し、同じ意味を重複して記述しないようにします。

## Application Patternが持たないもの

次のようなTarget固有の表現は、Application Patternの意味には含めません。

- HTML要素やCSS
- Web framework固有のcomponent名
- pixel単位の配置
- Desktop固有のwindow管理
- Mobile固有のgesture
- APIのHTTP routingそのもの
- MCPやVUI固有のtransport表現

これらはProjectionまたはTarget Adapter側で扱います。

## Patternのidentityを物理構造へ結び付けない

Patternのidentityは、file pathや画面ファイル名へ依存させません。

Source Indexを通して正本へ戻れるようにしつつ、ファイル移動だけでPatternの意味上のidentityが変わらない境界を維持します。

## Semantic RelationとUI構造を同一視しない

`header-detail`を導入しても、1:NのSemantic Relationが存在するだけで自動的にheader-detail画面を生成するとは限りません。

逆に、header-detailとして扱いたいというApplication上の意図からDatabase Foreign Keyを暗黙に生成することもしません。

Semantic Relation、Application Pattern、Persistence Mappingは、それぞれ意味の異なる層として明示的に接続します。

## commandをCRUDへ還元しない

`command`は「更新ボタン」の一般化ではありません。

`sales.order.submit`のように業務上の意味を持つCapabilityへ接続し、単純なRecord更新と区別できることを前提とします。Command acknowledgementと実際の状態変化を必要に応じて分離するRQ-018の原則も維持します。

## Custom UIを例外扱いしない

標準Patternから直接Projectionできない画面があっても、Application上の意味を捨ててTarget固有実装だけへ移行しません。

Custom UIは、Application PatternやCapabilityなどのCanonicalな意味へ接続する別のProjectionとして扱います。Custom UIを採用したことを理由に、同じ業務上の意味をWeb側へ重複定義しないことを原則とします。

## v1で検証するPattern

最初のReferenceでは次の5種類を扱います。

1. `browse`
2. `record`
3. `reference`
4. `header-detail`
5. `command`

これらは網羅的な標準Pattern一覧ではありません。

`order_entry`の業務シナリオを一続きに検証するために必要で、DriverとWeb Targetの双方から共有できる可能性が高い最小集合として選びます。

## PatternをSemantic Kernelへ昇格させる条件

将来、Patternの一部をSemantic Kernelへ持ち上げる場合は、少なくとも次をEvidenceで確認します。

- 複数Applicationで同じ意味として再利用される
- 複数Targetでも同じ意味が残る
- UI上の慣習ではなく、業務上の意味として独立している
- Coreへ置いてもTargetやProviderへの依存が増えない
- Kernelへ移すことで二重管理が減る
- 独立したConformanceを定義できる

これらを確認できないものは、Applicationが所有する定義またはProjection情報として残します。

## PatternをProjectionへ下げる条件

反対に、実証の結果、ある情報がWebなど特定Targetでしか意味を持たないと分かった場合は、Application Patternから外します。

初期実装へ入れたこと自体を正当化の根拠にはしません。第二Targetで不要だった情報や、Targetごとに大きく異なる情報は、Projection側へ移せる構造を保ちます。

## 影響

### 良くなる点

- Driverと複数Targetが同じApplication上の意味を共有できる
- Web画面の都合からSemantic Kernelを膨らませにくくなる
- Application Patternを標準UIテンプレートへ閉じずに済む
- Custom UIでもCanonicalな意味との接続を維持できる
- 第二Targetで境界を検証し直しやすい
- Patternの再利用とTarget固有表現を分けて評価できる

### 増える負担

- Semantic、Application、Projectionの三層を意識して設計する必要がある
- v1の段階ではPattern分類が確定しない部分が残る
- Webだけを作る場合より、Source Indexやidentityの設計が必要になる
- 何をApplication上の意図として残すかをReferenceごとに検証する必要がある

この負担は、第一Targetの実装都合をTEI全体の意味モデルへ固定しないために受け入れます。

## 検証

`ISSUE-000005`では、少なくとも次を確認します。

- 5種類のPatternを同じApplication定義から識別できる
- PatternがSemantic Data、Query、Capability、Datasourceなどをidentityで参照できる
- file pathをPatternの意味上のidentityへ含めない
- DriverがPatternを利用できる
- 第一Web Targetが同じPatternを利用できる
- Web固有の情報をCanonical Semantic Modelへ追加しない
- native SQLの正本をPattern側へ複製しない
- Custom UIを接続してもApplication上の意味を失わない
- Source Index / DiagnosticからPattern定義へ戻れる

## 関連文書

- `requirements/TEI_PRODUCT_REQUIREMENTS.md` RQ-001、RQ-011、RQ-012、RQ-013、RQ-014、RQ-017、RQ-021
- `plan/issue/000005-application-pattern-reference.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000007-first-web-target-projection.md`
