# TEIの実装状況

- 整理日: 2026-10-11
- 段階: **実験的な開発基盤・Referenceによる実証**
- 品質の注意: **生成AIを用いて開発中であり、コード・設計・セキュリティに対する体系的な人的レビューは未実施**。本番利用を前提としません。

TEIは、業務モデルと実装・Database・実行先を分離し、CLIとReference Workspaceで意味・実行・観測の接続を検証しています。現時点では完成したローコード製品やGUI Editor、汎用Target Runtimeを提供していません。全体像は[README](../README.md)、価値と対象範囲は[Product Value](../product/value/TEI_PRODUCT_VALUE.md)と[製品要求](../requirements/TEI_PRODUCT_REQUIREMENTS.md)を参照してください。

## 実装・検証で確認できること

| 領域 | 状況 | 確認先 |
| --- | --- | --- |
| Go CLI・Workspace | `cmd/tei`とCanonical Workspaceの読込・Validation・Source Index等を実装。具体的なコマンドは導入手順に記載 | [Getting Started](getting-started.md)、[Reference](../reference/README.md) |
| Application / Driver | `order_entry`でbrowse、record、reference、header-detail、commandなどの意味境界を実証 | [Architecture](architecture.md)、[v1計画](../plan/V1_REFERENCE_APPLICATION_PLAN.md) |
| Gate A | **Ready**。Application PatternからDriverへ渡す意味境界を確認した過去の判定 | [Gate A判定](../plan/GATE_A_READINESS_AUDIT.md) |
| Gate B | **Ready**。固定revision `e5231839a5dcc62cce1181a2731fcb7c70d16736`（2026-09-19）に対するRepository再現性、生成物、actual PostgreSQLの同一回のEvidence | [Gate B判定](../plan/GATE_B_READINESS_AUDIT.md)、[実行手順](../plan/GATE_B_EVIDENCE_RUNBOOK.md) |
| Gate C / 第一Web Target | **Ready**。固定build `0ba87a4e0a96f70a198b03ff7652fb00e3826abe`（2026-09-19）のSemantic・Functional・Visual Evidence。**Visual Fidelityは単一AI評価者による評価で、人的承認ではない** | [Gate C判定](../plan/GATE_C_READINESS_AUDIT.md)、[Evidence契約](../plan/GATE_C_EVIDENCE_CONTRACT.md) |
| Shared Acceptance・第二Application | `repair_intake`の一部Semantic Sourceを明示的受理でReference Workspaceへ取り込む経路などを検討・実装。受理範囲と実Targetの成立は別 | [Reference Workspace案内](../reference/README.md)、[実装構造](architecture.md) |

Gate A/B/Cの「Ready」は**それぞれの判定時点・固定Source／Build／Evidenceに対するもの**です。現在のmain、すべてのProvider、未知の業務、production環境や別のTargetが自動的にReadyになることはありません。新しい変更の資格を過去の結果から自動転用しません。

## 現在も未検証・未完成のこと

- **統合モデラー／Studio**: Human/AIの入力からData・Function・Businessの三視点を一貫して編集・受理・実行する製品としては未完成。
- **第二Targetと多様な配備先**: 汎用Web・Desktop・Mobile・組込み・IoTを同等の業務定義から成立させたProduct Evidenceは未取得。
- **外部接続と実Provider**: Referenceでのsynthetic fixtureと、Oracle等を含む実環境での認証・接続・Command実行の成立を区別。未取得のEvidenceを成功扱いしない。
- **再現可能な実Target操作**: Gate Cの初回結果と、後続のR7 replay / repeatabilityの実行・反復確認は別。未実施分を含む。
- **安全性・品質**: セキュリティ・プライバシー・第三者素材のライセンス・権限設定の包括的監査、体系的な人的レビュー、production durability / HAは未完了。
- **GitHub Actions / 完全な現行検証**: Actionsは利用制限のため停止中。過去の特定revisionのPASSを最新mainの`go test ./...`やRepository HealthのPASSとして扱わない。

検証可能な実装と未確定の設計を区別して進めます。開発上の優先順位は[TEI Master Plan](../plan/MASTER_PLAN.md)、技術境界は[Architecture](architecture.md)、業務設計の考え方は[Business-System Design](business-system-design.md)を参照してください。

## 最初に動かす

Go toolchainを用いた主要なローカル確認は次のとおりです。

```bash
go test ./...
go run ./cmd/tei validate -workspace reference/workspace
```

より詳しい検証コマンドとReferenceの範囲は[Getting Started](getting-started.md)を参照してください。この手順を掲載したことは、公開候補や最新HEADでテストを実施・合格したという意味ではありません。
