# TEI横断：契約保存・Claim単位の資格化と証拠の非昇格（設計候補）

- **状態**: 評価待ち（未採用）。要求・API・Core型・共通Runner・新Gateの決定ではない。新しいコード実行・実Provider/実DB・Studio GUI・Owner業務受理は **NOT RUN**。
- **静的照合基準**: tei-platform main `8c3d1a5582cf3e9a40b52b3a2ebaa041739b4fd2`（2026-11-12）。既存Issue・設計文書・PRの静的読取。改版後は再照合する。
- **目的**: 構造、業務意味、能力、生成、Build、実行、配送、Studioの間で、局所PASSを別のClaimへ無根拠に転用しない。反例と責務を横断的に接続する。
- **正本**: [現行アーキテクチャ](../../docs/architecture.md)、[現行実装状況](../../docs/status.md)、[ADR-0043](../adr/0043-target-native-artifacts-without-tei-host.md)、[ADR-0048](../adr/0048-shared-application-api-and-transport-adapters.md)、[重要Issueの計画](../workstreams/TEI_PRODUCT_CRITICAL_PATH_2026-10.md)。本書は正本の置換ではない。
- **既存領域設計**: [図表・Pipeline](TOPOLOGY_COMPOSITE_VIEWS_AND_PIPELINE_2026-10.md)、[Trait履行](TRAIT_ATOMIC_COMPOSITION_AND_FULFILLMENT_2026-11.md)、[ETL/Batch](ETL_BATCH_FUNCTION_COMPOSITION_DESIGN_2026-11.md)、[セルフビルド再入](SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)。これらの試験・実装Ownerを移さない。

## 1. 採用を提案する最小原則（新規基盤機構ではない）

1. **Meaning / Contract first**: 意味・要求操作・制約はCanonicalな意味の受理境界へ、実現方式・公開方式・Runtime・物理成果物は別のBindingへ置く。TEI固有Host、中央HTTP、ETL専用DSL、第二Graph/Resolver/Scenario SSOTは必要条件にしない。
2. **Evidence before promotion**: Sourceに記述がある、Parserが読んだ、型が合う、Providerが選ばれた、Runnerが成功した、Artifact byteが一致した、Consumerへ到達した、実Effectが確認された、Ownerが業務意味を受理した、は別々の主張である。一つから隣の成立を推測しない。
3. **Scope-relative qualification**: 一つの検証結果は対象Subject・Scope・問い・前提・Reader/Consumer・Source/Artifact版・実行Attempt・Policy時点に相対化する。Claimの対象を変えたら必要な根拠を再照合する。
4. **No invisible completeness**: 「図にないなら無い」は問われた範囲の全入力・Reader/実消費・関係/役割・未解決・開示条件が閉じた場合だけの限定結論。見えていない処理、未観測のWriter、未取得のExternal/Private情報は無いことにしない。
5. **Minimal meta-generation**: 共通部品の合成で解ける間は専用Generator-of-Generatorsや万能Pipeline Engineを追加しない。異なる実Target間で実証済みの重複・変更負担が残ったとき、再利用可能なTransformの最小生成を検討する。

四層（Authoring / Canonical Semantic / Binding・Realization / Implementation・Execution）は現行構造を維持する。横断の評価表は新しい第五層、認証主体、Claim Registry、永続的な製品Schemaではない。

## 2. Claim間の引継ぎ規則（必要条件・十分条件ではない）

一件のClaimを照合するための**既存Evidenceの一時的な比較列**を、少なくとも以下に分ける。名前を永続API・Core enumに固定しない。

| 比較列 | 判定したいこと | 典型的な欠落 |
| --- | --- | --- |
| Subject / Scope / Question | 何の、どの範囲の、どの問いへの結果か | 同名異Scope、Query述語未対応、Source無ID |
| Required closure | その問いへの必要入力・関係・Role・時点が列挙済みか | 暗黙入力、Graph非表示辺、外部Writer |
| Source / Meaning revision | 実読込byte・Source Index・必要な受理済み意味が一致するか | Source位置とSemantic IDの混同、推定の正式昇格 |
| Mapping / Trait / Provider | 必須の構造・値・時間・操作保証を保存または履行したか | 非単射Projection、未履行Required Trait |
| Realization / Artifact / Consumer | 選択Targetと実Build消費・Artifact一式・受領byteが結ばれたか | declared-only、Proxy child、部分公開、Reader未受領 |
| Effect / Authority / Acceptance | 実副作用の独立観測・現在のGrant・Owner受理が対象Claimに適合するか | HTTP成功＝DB確定、ACK＝業務完了、古いGrant |

**引継ぎできる場合**: 送り側と受け側で対象・版・Scopeを対応付け、必要なFacet/Contractが写像で保存され、当該Claimに必要な実Consumer・独立Expected・現在の許可を既存Ownerが検査した範囲だけ。条件の列挙自体はPASSの証拠ではない。

**引継ぎできない場合**: 欠落・損失・不明・非開示・未観測をゼロやPASSに丸めない。読み取り成功は編集逆写像の許可ではない。Provider conformanceは実Targetの権限ではない。実行時の古い資格は同digestに戻っても自動復活しない。逆にLayoutだけの変更は、意味に影響しないことを確かめられた範囲で不要なMeaning再受理を発生させない。

区別すべき二種類の証拠は、**不変な履歴事実**（特定の版・Attemptで何が起こったか）と、**現在の使用資格**（Grant、Disclosure、Provider選択、読取先、Owner判断等）である。履歴PASSを消す必要はないが、現時点の使用資格へそのまま再利用できない。

## 3. 他チャットの成果を結ぶ最小反証

下表は**既存Issueへ渡す比較観点**であり、独立Expected/Scenarioの第二正本や新しい製品試験IDではない。実試験結果を主張しない。

| 反例・変更 | 禁止する誤昇格 | 既存Owner |
| --- | --- | --- |
| Sourceは同じ表示名だがScopeが異なる、または元X-TEA要素に安定IDがない | Source位置・一時View IDからPin/逆編集の資格を合成 | [#1124](https://github.com/hat47x/tei-platform/issues/1124)、[#1120](https://github.com/hat47x/tei-platform/issues/1120) |
| Graph→Tree／集計表でRelationやOccurrenceを失う | read-onlyの投影成功から元グラフの可逆編集を許す | [#2497](https://github.com/hat47x/tei-platform/issues/2497)、[#4350](https://github.com/hat47x/tei-platform/issues/4350) |
| 保存Providerの構造的特殊化で原子更新・条件付き書込みが不足 | Pluginの互換名・解決成功をRequired Trait履行と扱う | [#951](https://github.com/hat47x/tei-platform/issues/951)、[#3278](https://github.com/hat47x/tei-platform/issues/3278) |
| Build childがProxy経由で別実行物を呼ぶ、または宣言外のbyteを読む | Artifact digest一致を全actual consumed closure・builder由来の保証と扱う | [#2831](https://github.com/hat47x/tei-platform/issues/2831) |
| ETLの途中Stage完了・File ACK・Dataset片方のみ新世代 | Business Accepted、完全なDataset公開、冪等Commitの証拠と扱う | [#1130](https://github.com/hat47x/tei-platform/issues/1130)、[#2542](https://github.com/hat47x/tei-platform/issues/2542)、[#2604](https://github.com/hat47x/tei-platform/issues/2604) |
| FunctionのCRUD欄が空、または代表Bindingが一つ | 全Writer閉包・越境直接CUDがゼロと断定する | [#3250](https://github.com/hat47x/tei-platform/issues/3250)、[#1127](https://github.com/hat47x/tei-platform/issues/1127) |
| Private Worldを変えPublic Sliceは同一 | 公開のCount・Score・Digest・診断・DOMからPrivateの存在が分かる | [#2516](https://github.com/hat47x/tei-platform/issues/2516) |
| Source/Plugin/GrantがA→B→Aへ戻る | 昔の受理・Current Grant・実Effectの資格が復活する | [#2555](https://github.com/hat47x/tei-platform/issues/2555)、[#1174](https://github.com/hat47x/tei-platform/issues/1174) |
| 旧Legacy観測LとOwner受理Desired Dが意図的に異なり、Target観測TはDと一致 | 旧実装とのbyte/挙動一致と意図的な業務改善を同じ互換性PASSにする | [#1124](https://github.com/hat47x/tei-platform/issues/1124)、[#3394](https://github.com/hat47x/tei-platform/issues/3394) |
| TEI→EKIへのAdmissionは成功、実Worker結果・独立Readerは欠落 | 委譲受理を実Function Outcomeや業務Effectと扱う | [#4384](https://github.com/hat47x/tei-platform/issues/4384)、[#4194](https://github.com/hat47x/tei-platform/issues/4194) |

三要素分析法（データ・機能・業務）の原典に忠実な表現と、TEIの汎用Graph/Role・AI解析は混同しない。Legacy観測・受理済み意図・Target観測は三要素分析法の三視点とも別の軸とする。

## 4. 実装順への受渡し（新しいHard Gateを増やさない）

| 順 | 確認対象 | 既存の受入先と停止線 |
| --- | --- | --- |
| **P0-A** | 既存Read-only Operationの同一Scope・版・失敗意味をCLI/MCP等で比較 | #1158 / ADR-0048。入口が同じJSONを返すだけでは独立Business Oracleにならない |
| **P0-B（並走）** | セルフビルドのDeclared→Selected→Actually Consumed→Actual Outputと実childの観測 | #2831。L2bとG4全体資格、Seed trustを分離 |
| **P0-C（並走）** | 必須Trait→解決済みProvider→実履行と、Facet→表現型のLoss/逆写像 | #3278 / #4350 / #2497。Referenceの合成PASSは実Targetへ昇格しない |
| **P1** | 一つのread-only Batch FunctionのInput ClosureとArtifact/Consumerを接続 | #1130 / #2542 / #2604。Source/Plan/Run/Published/Consumedを別Claim |
| **P1** | Source根拠つきDirect CUDとUnknownの分割案、Effective Effectは独立欄 | #3250 / #4412 / #4413。候補順位やGraph下界を全業務の証明に使わない |
| **P2（P0のHard依存なし）** | StudioのLocate-first read-only Consumerから限定変更Proposalへ | #1124 / #267。非可逆表現・無ID Sourceの編集は拒否し、Owner受理を別にする |

各実装Ownerは既存のScenario SSOT（#1180）、独立Expected（#2514）、Scope/Disclosure（#1120/#2516）を利用する。本書が新しいSchema・Engine・Contract Resolver・Policy・Recorderを所有しない。

## 5. 検証可能な研究仮説と撤回条件

**仮説**: 同じSubject/Scope/Claimを、Source→構造→操作→実現→実Consumerの各境界で明示し、局所証拠を過大昇格させない設計により、人間と生成AIの誤った肯定・誤った不存在判断と変更負担が減少する。

**測定候補**: （a）誤ったPASS・不存在断定の件数、（b）必要Source/Ownerに到達する操作数、（c）変更後に不要となる再資格化数と見落とした必須再資格化数、（d）AIが修正するまでの往復・Token、（e）新抽象の維持コスト。既存単一図/散文と、同一Sourceからの表・Graph・診断を比較する。実験は **NOT RUN**。

**撤回・縮小**: 同一のClaim表を各Ownerが二重管理する、Consumer間で意味がずれる、Scope/Policy漏えいが増える、実験で理解・修正負担が減らない、または既存Contractで十分に表現できるなら、新しい抽象や永続化は導入しない。再利用できる比較表として留める。

**本書の証拠上限**: GitHubの既存Issue/PRと設計・Sourceの静的照合に基づく横断設計だけ。新規Go/Node/Windows/Wails実行、実Build/ETL/DB/EKI Consumer、Current Grant、外部Policy、Ownerの仕様採用は確認していない。