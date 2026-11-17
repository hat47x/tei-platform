# TEI自己記述の第一縦断：三要素モデル・具体例・図面レビュー（2026-11-17）

> **位置づけ：設計・実証準備。** [三要素Method §33](THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md#33-tei自身の要件全体をモデルとしてレビューする三要素method標準構造インスタンスx-tea移植2026-11-17設計統合)の最小実行票。図とデータはレビュー用の説明例であり、TEIの正式DSL・採用済みMeaning・X-TEA原典実測ではない。ここに示すSDMケースの製品試験・GUI試験・Owner受理は **NOT RUN / NOT ACCEPTED**。

## 1. 何を一続きでレビューするか

TEIの要件のうち、まず [RQ-011](../../requirements/TEI_PRODUCT_REQUIREMENTS.md)（複数Projection）、RQ-026（三要素分析法のモデルによる相互検証）、RQ-031（特定のMethodをCoreで特権化しない）の三つを選ぶ。これらは**対象となる実在の要求項目**である。一方、下図に示す設計者のレビュー業務、図面の組み合わせ、Graphの例示インスタンスは、今後の実証候補であって実装済みの機能を示さない。

この一件を「要件索引 → データモデル／機能モデル／業務モデル → モデル定義 → 具体例 → Source → 検証結果」の順で追えることを第一縦断の目的とする。全要件を一枚の図で表示することは求めない。三要素分析法は原典に忠実な**Authoring Method**とし、TEI固有のMeta・標準表現型・Generator・Provider・Evidenceへのリンクを、原典と区別した派生Projectionで重ねる。三つのモデルをCoreの三つの固定クラスへ変換する意味ではない。

### 図A：三要素とTEI固有のレビュー情報

```mermaid
flowchart LR
    R["要求索引<br/>RQ-011 / RQ-026 / RQ-031"]
    D["データモデル<br/>構造・関係・型付き例"]
    F["機能モデル<br/>入出力・参照・CRUD"]
    B["業務モデル<br/>担当業務・人手作業・契機"]
    P["派生Projection<br/>表・ER・Graph・説明図"]
    S["Source / Revision<br/>根拠位置"]
    E["独立Expected / 実測<br/>受理資格は別"]
    R --> D
    R --> F
    R --> B
    D --> P
    F --> P
    B --> P
    D -.-> S
    F -.-> S
    B -.-> S
    P -.-> E
```

この矢印はレビュー時の**参照経路**の例であって、Canonical Relationの定義でも、業務アクションが実際にデータを書き換えた証拠でもない。正式な関係と派生経路、描画の線を区別して保つ。

## 2. データモデルとインスタンスを並べて示す

標準Graphの例は、型の説明と実際の値を混同しないため、次の二つの記録を別々に扱う。

| 区分 | この例で確かめる内容 | モデル上で残す情報 |
| --- | --- | --- |
| モデル定義 | ノードと有向辺に独立した識別子があり、同じ端点間に複数の辺を持てる | 型、ID、source/target、辺属性、多重性、制約 |
| インスタンス | 同じ表示名の二つのノードと二本の異なる辺 | 各要素ID、関係ID、向き、属性値、Source/版 |
| 表現型 | 同じ実例をGraph図と表で表示する | 表示対象、描画IDと意味IDの対応、省略・集約・表示順 |
| 検証 | どちらの図面でも同じ四要素を区別できる | 保存した意味、不明・失われた意味、独立期待値との照合 |

### 図B：同名ノードと平行有向辺（説明用の小例）

```mermaid
flowchart LR
    N1["node-01<br/>表示名：対象"]
    N2["node-02<br/>表示名：対象"]
    N1 -->|"edge-01 / kind=reference"| N2
    N1 -->|"edge-02 / kind=dependency"| N2
```

| 要素ID | 種別 | 表示名または端点 | 例示属性 |
| --- | --- | --- | --- |
| node-01 | Node | 対象 | role = source |
| node-02 | Node | 対象 | role = target |
| edge-01 | directed Edge | node-01 → node-02 | kind = reference |
| edge-02 | directed Edge | node-01 → node-02 | kind = dependency |

図が二本の辺を重ねて描画しても、一覧では **edge-01とedge-02が別要素**であると分かる必要がある。ここでのIDは説明例のIDであり、表示Occurrence IDや描画エンジンのEdge IDではない。表示名でnode-01とnode-02を統合しない。候補Relationと受理済みRelationを混在させる場合は、さらにAuthority/Provenanceを分離する（SDM-07）。

### 値の有無を扱う別の小例

| 例示ID | 属性／集合 | 値の状態 | レビュー時の意味 |
| --- | --- | --- | --- |
| presence-01 | note | `null` | 取得済み・明示的NULL |
| presence-02 | note | 欠損 | 項目が与えられていない |
| presence-03 | lines | `[]` | 取得済みの空集合 |
| presence-04 | lines | 未取得 | 完全性を主張できない |

これらは同一フィールドに対する単一の型表記ではなく、異なるインスタンスの**状態の比較例**である。許される状態は対象モデルと取得Scopeに依存する。未取得を空集合、欠損をNULLに自動変換してはならない。X-TEAの自由なInstance TextやSlide順を、根拠なしに型付きデータ・実時刻・実Effectへ変換しない（#1131）。

## 3. 機能モデル・業務モデルとの関係は帰属の強さを表示する

X-TEAとの最初の照合対象は、#1031で観測対象にしている次の経路である。

```mermaid
flowchart LR
    T["Task"] --> A["TaskAction"]
    A --> TF["TaskFunctionIO"]
    TF --> F["Function"]
    TF --> UI["IOPanel / IOSpool"]
    F --> C["IOTableの<br/>Function-level CRUD"]
    C --> DT["Table"]
```

- **原典の明示関係**：Task、TaskAction、TaskFunctionIO、Function、IOTableのSource上の参照を、それぞれ出典位置・版とともに表示する。
- **派生経路**：ActionからFunctionを経てTableへ到達できることと、ActionがTableへCUDを実行することは異なる。Function-level CRUDをAction固有の実Effectへ自動昇格させない。
- **人手・外部手順**：Functionを呼ばないTaskActionは正当な業務モデルの要素になり得る。未接続Functionの自動補完や業務実行の成功判定は行わない。
- **TEI固有の表示**：型付きI/O、Operator、標準表現型、実呼出しの証拠は、原典の図法そのものではなく追加の情報として表示する。

**第一段階はread-only**。Cross-viewの参照候補と正式Relation、表示上のEdgeを同一視しない。実GUIの編集はSubject・Source Span・Revision・被参照の確認を経て、現在開いている原Sourceへ明示保存し、再読込後に再投影する（#1124、#3788）。第二のMeaning正本は作らない。

## 4. レビュー用の一件票

図面を見るだけでは判断できないため、対象ごとに次の情報を伴わせる。既存のSource Index、Projection、独立Expected、Evidenceを**組み合わせて表示**するための項目であり、新しい永続Schemaや管理サービスの提案ではない。

| 項目 | 最初の一件で必要な判断 |
| --- | --- |
| 対象・Scope | どの要求・Source版・Method Profile・開示範囲について判定するか |
| 対象集合の完全性 | Data / Function / Businessごとの対象総数、観測数、未観測・unsupported・unfetchedの理由。0件と未調査を分ける |
| 型・関係・例 | Model定義とInstanceが対応するか。実例は説明例／独立Expected／実観測のどれか |
| Projectionの忠実度 | 元構造に対して保存・省略・集約したIdentity、向き、多重性、順序、Presence、単位、時点 |
| 出典と帰属 | Source位置、Revision、導出規則、正式関係／派生経路／候補の区別 |
| 変更時の扱い | read-only、明示保存可能、非一意のため変更不可、のいずれか |
| 資格・判定 | 定義の読取り、図面表示、独立期待値の一致、実Consumer、DB Effect、Owner受理を別々に記録 |

この第一縦断において、「RQ-011を図で探せた」だけではRQ-011の適合もRQ-026の網羅も認めない。逆に、Function/BusinessのReaderが未対応であれば、**未対応と報告すること自体が正しい検証結果**となる。分母を確定できないCoverageを100%と表示しない。

## 5. 実装担当への具体的な引渡し

| 順序 | 既存の担当・試験 | 最小成果と停止条件 |
| --- | --- | --- |
| **1：read-only基線** | #1124 / #1031 / #1126、SDM-01・SDM-03・SDM-06・SDM-08 | Source版付きでデータモデルと三要素参照を表示し、同名異ID・Function未観測・原文Instanceの未型付けを壊さない |
| **2：型付き例の照合** | #1131 / #2497、SDM-02・SDM-12 | GraphとPresenceの例で、表／図の間の意味保存・表示損失を説明できる |
| **3：独立判定** | #1180 / #2514、既存Scenario/Expected | 正例・一因子負例を実装とは独立に用意し、Source版・対象集合・観測範囲に合う場合だけ結果を認定する |
| **4：Sourceへの変更** | #1124 / #3788、SDM-04・SDM-05・SDM-11 | 一意な編集対象と現在のRevisionが確定する場合のみPreview → Save → Re-read → Re-project。曖昧・古い版は拒否 |
| **5：実利用の別検収** | #2586 / #2831 / #4583、SDM-09・SDM-10 | 通常製品CLIの実ProviderとStage1→Stage2の実Windows Consumerは別P0。図面の成功を実DB更新・セルフビルド成功にしない |

**この文書による実施結果**：上記は設計と照合用の具体例である。SDMケース実行、原典GUIとの画面比較、XeadDriver/正式Provider/DB実験、Stage2実Windows起動、CI、Owner意味受理は行っていない。既存のP0へ新しいHard Gateを追加せず、作業は各IssueのOwnerの範囲で進める。
