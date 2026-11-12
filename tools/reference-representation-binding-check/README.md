# reference-representation-binding-check（#4353 / #4350 S3）

意味を伴う表現インスタンスが**必ずFunction Data Contextの中に置かれ**、Context・Scope・grain・軸role・Selection・操作資格へ束縛されているかをread-onlyで検査する参照です。Renderer・UI・実Mediaは扱いません。#4351／#4352の契約形状をそのまま使います。

```text
go run ./tools/reference-representation-binding-check/cmd/tei-representation-binding-check \
  -input plan/fixtures/representation-binding/cases.json \
  -expected plan/fixtures/representation-binding/expected.json
```

全件一致なら終了コード0、不一致なら1、入力不正なら2です。`-contract`は契約の自己記述をJSONで返します。

## 判定

| 判定 | 意味 |
| --- | --- |
| `bound` | 1つのContextに所有され、必須の束縛列がすべて宣言されている |
| `unqualified` | 所有されているが、必須列（`crs`・`clock`・`grain`・`scope`・`operation`など）が欠けている |
| `rejected` | 宣言が境界と矛盾する（layout を意味とする、Selection をIdentityとする、2つのContext、Scopeの再利用、集約点の逆写像） |
| `unknown` | 15表現型の外、または証拠が無い |

`unqualified`のときだけ`unqualified`列に不足列を列挙します。`rejected`は矛盾そのものが理由であり、列の欠落とは別の問いなので空にします。

## 15表現型（閉じた語彙）

table、form、tree、graph、chart、gantt、map、matrix、markdown、formula、drawing、image、audio、video、composite。

## 時間軸4role

`domain`、`observation`、`presentation`、`structural`。**Domain TimeをPlayback Timeとして報告すると`AXIS_ROLE_CONFLATED`で拒否**します。両方の軸を宣言する場合（観測軸と再生軸の対応）は正当で、`clock`と同期誤差の有無で資格を決めます。`clock`が不明なら`unqualified`、同期を主張して誤差が無ければ`SYNC_CLAIM_WITHOUT_BOUND`。

## 生成軸

`generated: true`の軸（ページ番号・reflow・朗読時間など）は**元データの軸ではありません**。`declared as generated`であれば`bound`のまま`GENERATED_AXIS_NOT_SOURCE`を情報として記録し、生成軸を元の軸として報告した場合だけ拒否します。

## fixture（14ケース）

`plan/fixtures/representation-binding/`の`cases.json`が入力、`expected.json`が**手書きの独立Expected**です。根拠は#4350のRSP-01〜06表と#4133／#2497／#2506の本文です。

| ID | 内容 | 判定 |
| --- | --- | --- |
| RSP-01 | 表の行indexを業務Identityとする | `rejected`（SELECTION_IS_NOT_IDENTITY） |
| RSP-02 | Treeのnodeを一意な更新対象とする（hidden edgeあり） | `rejected`（TREE_NODE_IS_NOT_IDENTITY） |
| RSP-03 | Chartの集約1点を元Entityの代表とする | `rejected`（AGGREGATE_IS_NOT_ENTITY） |
| RSP-04 | MapがCRSを宣言しない | `unqualified`（crs） |
| RSP-05 | 数式の表示一致を計算意味の一致とする | `rejected`（DISPLAY_EQUALITY_IS_NOT_MEANING） |
| RSP-06 | Domain TimeをPlayback Timeとして報告 | `rejected`（AXIS_ROLE_CONFLATED） |
| RSP-07 | 表の行をlineとして束縛する正常系 | `bound` |
| RSP-08 | Context外に置かれた意味ある表現 | `rejected`（CONTEXT_MISSING） |
| RSP-09 | 内部トポロジーをデータの意味とする | `rejected`（TOPOLOGY_PROMOTED_TO_MEANING） |
| RSP-10 | 別Contextで旧Scopeを再利用 | `rejected`（SCOPE_REUSED_ACROSS_CONTEXTS） |
| RSP-11 | Ganttの時間軸と同期誤差を宣言する正常系 | `bound` |
| RSP-12 | 生成軸を宣言したMarkdown | `bound`（GENERATED_AXIS_NOT_SOURCE は情報） |
| RSP-13 | 集約された複合表現に逆写像を宣言 | `rejected`（AGGREGATE_IS_NOT_ENTITY） |
| RSP-14 | clock不明のまま同期済みと宣言 | `unqualified`（clock） |

## 既知Mutant

| Mutant | 落とすべきケース |
| --- | --- |
| `AcceptContextless` | RSP-08 |
| `TreatTopologyAsMeaning` | RSP-09 |
| `TreatSelectionAsIdentity` | RSP-01 |
| `AcceptUnknownClock` | RSP-14 |

## 実行した検証（2026-11-12、Windows／go1.27.0）

- `go test ./tools/reference-representation-binding-check/ -count=1`：**PASS**（9テスト）
  - 14ケースのExpected一致（`matched=14/14`）
  - 3判定の網羅、Mutant 4種の判別、Context必須、Domain≠Playback、生成軸の情報扱い、集約の逆写像拒否
  - 語彙外の拒否、契約の機械可読性（15表現型・4軸・6必須列）
- `go vet`：PASS、`gofmt -l`：差分なし
- `git diff --stat internal cmd`：空

## 実行していない検証（NOT RUN）

- 実Renderer・Widget・UI・Studio・SHO・実Media再生／同期・実画像生成。すべて合成descriptorのみ。
- 実Consumerの受領、Product Contextの実装、Linux CI（PRで確認する）。
- Ownerの意味受理、契約のADR採択。

## この参照が主張しないこと

- 新Representation Registry・Renderer Plugin Contract・Context型の採用。
- 表現型のCore固定Enum化。
- 実Targetでの表示品質・知覚負荷・アクセシビリティ。
- 業務意味の受理。`bound`は合成descriptorに対する比較結果です。
