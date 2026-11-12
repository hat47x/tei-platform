# OKF K2 実Provider二Consumer受入計画（2026-11-12）

## 位置付けと更新根拠

本書は [TEI/SHO OKF Domain Lint・Document Unit計画](TEI_SHO_OKF_DOMAIN_LINT_DOCUMENT_UNIT_2026-11.md) の**実Provider二Consumer受入**に限定した追加計画。旧[PR #3329](https://github.com/hat47x/tei-platform/pull/3329)はマージ済みで元作業ブランチは削除されているため、履歴の旧HEADを更新せず、現mainから新規Draftとして起票する。旧PRの大型設計差分を現mainに存在すると推定しない。

2026-11-12再照合：TEI [#3479](https://github.com/hat47x/tei-platform/pull/3479)（Provider Source-only drift）とSHO [#27](https://github.com/hat47x/sho-composer/pull/27)（公開TEI原文のprivate SHO実Reader受領）、SHO [#29](https://github.com/hat47x/sho-composer/pull/29)（UTF-16選択→UTF-8 byte範囲）は**マージ済み**。これらは限定的な局所検証であり、K2同一Provider実Invoke、実Writer、Owner Meaning受理の証拠ではない。旧TEI HEAD `1cf0a48a` の健全性CI成功は確認できず、旧HEAD `0d035db6` の成功を新mainへ転用しない。

## 受入の不変条件

- **K2は同じSourceを二つのConsumerが読めたことではない**。一つのProvider契約で選択された実体をTEI CLIとSHO実Reader境界の二つの独立Call Siteが**実際にLoad/Invoke**した証拠が必要。
- `selected_provider_id`、`loaded_provider_id`、`actual_invoked_provider_id`、`consumer_received_provider_id`を別観測として保存。選択Bに対してAが呼ばれれば、結果のbytesが同じでもFAIL。静的Manifestの値を四欄へ転記してPASSにしない。
- Source raw SHA、Manifest digest、Provider Lock、toolchain/generator、Build Artifact、Source Revision、Consumer Receiptを別々に保持。同じdigestでもA→B→A、Owner受理、Current Host Grant、Audienceを推論しない。
- F0形式・F1必要Source/Role/Relation閉包・F2 Owner Meaning/Host Grant/Disclosureは別判定。形式が不明でもDraftとして保存可能なことと、TEI Loader/Builderの受理を混同しない。
- 正常な空`result:[]`、未取得、not-found、permission-denied、partial、unsupported、unknownを分ける。未探索の対象を「存在しない」と推論しない。
- Core最小化／Everything is Pluginを維持。第二のHost、Registry、Parser、Policy Engineを作らず、現mainのPlugin Selector、共通Application API（ADR-0048）、CLI/Inspector/SHO G1b境界を優先する。

## 実装・検収の順序

| Gate | 一件目の実作業 | 合格に必要な証拠 | 未達時の停止 |
| --- | --- | --- | --- |
| P0: 現main探索 | Capability→Selector→実Build/Load→Invoke→CLI Replyをコード/テストから列挙 | 各辺の実Call Site・固定Revision・既存Adapter・未実装箇所 | Offer宣言や選択UIだけならK2-Aへ進めない |
| K2-A | Provider A/Bを用意しB選択→B実Load→B実Invoke→TEI CLI受領 | 同一Attemptの選択/Load/Invoke/受領ID、実Source/Manifest消費、独立Expected | A誤呼出し、未観測、異Attempt Receipt混合はFAIL |
| K2-B | SHO private側の実Reader/Adapterから同じProviderを呼ぶ | private権限内で実Invoke、Raw SHA、Reader版、未解析Reference、独立Receipt | 公開TEI CIにprivate SHOのPAT/Sourceを持ち込まない |
| K2-C | 両ConsumerのRequired→Received差集合を照合 | 事前固定Expected、取得/未取得/unsupportedの別票、Coverage上限 | 2 CI緑やMarkdown tokenだけで完全性を主張しない |
| K2-D | Provider曖昧、A誤呼出し、Source-only/Manifest-only、ABA、権限撤回、partial配送 | 一因子ずつの拒否・失効Receipt、旧Artifact/旧Evidence再利用の個別判断 | 不正な自動Fallback、旧Receiptの現在資格化を拒否 |
| K2-E | 2 Provider×2 Consumerの4 exact tupleとCLI/Studio限定Parity | 実Build/Invoke、独立Expected、同じ対象Scopeの比較 | 表示一致だけでRuntime等価としない |

**実装順序上の重要点**：K2-Aに先立ち、既存の実ProviderがSource/Manifestのどちらを実消費するかを特定する。#3479のSource-only drift検証を、K2-Aの実消費証拠へ接続する。SHO #27のK1R受領はK2-Bの入力Fixture再利用に限り、Provider共通化済みとは呼ばない。

## 横断製品への責務分離

| 製品 | K2と共有するもの | K2に持ち込まないもの |
| --- | --- | --- |
| TEI | 型・Source/Meaning境界、Provider契約、共通API、Context Query、Receipt | Source一致だけによるMeaning受理 |
| SHO | 原文位置・Reader/Editor境界、Candidate Review、Working Copy | K1R読取からの正式Writer/ABA/CAS保証 |
| SUI | Item/Occurrence/Canvasの参照・投影Loss観測 | 表示OccurrenceをSource Identityとみなすこと |
| SEI | 根拠と独立Expected、推論と判断の区別 | 推論結果によるOwner採否の自動代替 |
| KON Dataspace | 将来の保存・版・権限・公開境界とのAdapter | KONをK2必須Host/Registryにすること |
| TAI/外部Harness | 実行Receipt・外部受領の独立検証 | 内部CLI成功からH2/H3完全配送を推定 |

## 証拠状態・更新手順

K1上流Reference限定PASS、SHO G1bのK1R実Reader限定PASS、TEI #3479のSource-only drift PRマージ、SHO #29の選択範囲変換PRマージを**別々の成果**として扱う。**K2-A〜E、正式F0独立Oracle、Owner受理、正式Writer、H2/H3は本計画では未検証**。

本計画の更新時には、対象main SHA、Provider/Consumer実Call Site、各AttemptのSource/Manifest/Lock、CI URL、失敗ログ、実際の効果範囲を記録する。CIのqueued/in_progress/cancelledをsuccessとしない。PRをマージする際には旧#3329の計画を二重正本にせず、必要に応じて親計画へ本書をリンクする。
