# E10：アプリケーション構成の採用判断 — 二つのConsumerで残る選択と検証順序

- **状態**：評価待ちの設計・検証引継ぎ（2026-10-10）。新しいComposition DSL／Core kind／独立Lock／Runtime Resolverの採用ではない。
- **管理**：[Issue #1016](https://github.com/hat47x/tei-platform/issues/1016)・[Draft PR #1051](https://github.com/hat47x/tei-platform/pull/1051)。
- **前提**：E1〜E4は異種変更の選択的再検査、E5〜E7aは限定した型付きSource定義の追跡、E8はSUI Native Consumerの実装境界、E9はContract Query入力属性の直接Source参照を扱う。いずれも**採用済みの業務意味・実Provider互換・Runtime権限・全依存閉包の証明ではない**。
- **対象要求**：RQ-003／RQ-004／RQ-005／RQ-008／RQ-013／RQ-015。複数の正本と不要な構成記述を増やさず、実案件の再利用と安全な更新を判定する。

## この段階で立証したこと、立証していないこと

「UI構成キットを組み合わせられた」ことと「アプリケーション全体を構成できた」ことは違う。E2のUI見出し変更は案件UI所有の局所候補になり得るが、同じE2のDB接続先変更は、登録済みインスタンスを選んだだけではProvider・秘密情報・移行・アクセス権限を確認できない。E5〜E9で型付きSource参照を増やしても、その差は消えない。

一方、`order_entry`では既存Application・Environment・Plugin/Bundle・Source Indexが相当数の情報を持つ。**構成Profileはこれらを複写する場所ではなく、既存の正本から導出できない採用・接続の判断が実在した場合だけ、その出典に戻れるように記録する候補**とする。SUIはStage 0で部分的なSemantic参照しかTEIにないため、TEIのApplicationが正式に存在することを前提にしない。

## 構成判断を四つの独立した検査へ戻す

| 問い | 判定主体（既存の責務） | 現在のReferenceが返せること | 返してはならないこと |
| --- | --- | --- | --- |
| **何を意味するか** | Application／Domain Accepted Source、Semantic・Business検査 | E5/E9の一部の型付き定義参照・出典 | 未確認の業務規則の採用、業務KeyやRelation意味の推定 |
| **何を使い、どう接続するか** | Capability/Provider Contract、Target/Framework、既存Plugin/Build Lock | 候補の識別・選択不足、E2/E3の局所差分 | Pluginのロード順・同名Versionからの暗黙選定、独立した第二Lock |
| **どう利用・変更できるか** | UI構成キット／Native Domain／Project-owned Source | UI表示変更・Project Overlayの候補、Native Commit境界の出典 | UIの見た目を根拠にAction／Authorizationを変更する |
| **いま実行してよいか** | Runtime Permission・Resource Grant・実Provider・Release／Migration／観測 | 未実行・未確認の項目を列挙する | Profile採用＝権限取得＝デプロイ成功という一括成功判定 |

これらは互いに独立した判定である。例えば`E7=limited-unchanged`であってもDB切替のProvider適合やSUI保存のCASを合格扱いにしない。構成の最終結果を単一の`ready`へ早期圧縮せず、**どの検証が不足し、誰が所有するか**を保持する。

## Consumer A：受注明細Webの二つの異種変更

| ケース | 既存Sourceの正本 | 構成として残る選択 | 合格に必要な別検査 | 失敗時 |
| --- | --- | --- | --- | --- |
| A1. 案件UIの見出し変更 | `reference/ui-customization/web-react/project-extension.mjs` | 既存UI Kit/Project Layerの対応と更新基点 | E2/E3局所差分、実ReactDOM・a11y・Keyboard・IME・ブラウザ動作、実Go ActionのDenial/Validation結果が不変 | UIの局所候補を`unresolved`。業務・DBの正本へ自動書込みしない |
| A2. `order_entry.sales`の開発DB接続先変更 | `reference/workspace/environments/development/environment.tei`、Application DB要求 | 明示された既存Instanceへの接続変更候補 | Application DB Contract、Provider適合・実認証、Secret/Network Grant、Migration/Transaction・Rollback、actual Runtime observation | `proposed-unqualified`のまま再検証。A1が成功してもDB変更は認可しない |
| A3. ApplicationのValidation／Contract Query変更 | `reference/workspace/applications/order-entry/`、`semantic/`、`databases/` | 変更の実際の利用先とSource出典 | E5→E7の限定直接依存、[型付きRef/Impact候補 #1120](https://github.com/hat47x/tei-platform/pull/1139)の接続、既存Semantic／Driver／Conformance | 直接参照不変でも未追跡の推移辺があれば`fullClosure=false`を保持 |

**重要な負例**：UIだけの変更に伴い、E7のIndexへ関係のない別DomainのSource更新が入っても、その理由だけでUI候補を全面停止しない。ただしE7が限定範囲の不変を返したとしても、Action・実装Provider・Runtime Policyまで不変と推論しない。作用範囲の判断根拠が無ければ`unresolved`である。

## Consumer B：SUI Native Canvas

[SUIの実ソース監査（E8）](E8_SUI_NATIVE_CONSUMER_EVIDENCE.md)で、`CardView`→`CanvasShell`→`App`のNative preview／commit分離が確認された。**この既存Native動作をTEI UI Kitの標準Formへ再実装しない。**

アプリケーション構成で未決定なのは次の接続選択に限定する。

1. Native Canvas／DocumentV1の正本をSUIに置くことと、TEI側へ渡す**確定した一回のIntent**の契約・Adapter。
2. 局所編集Commitと保存要求／ETag-CAS／Tenant Session／実行時Permissionを別々に検証すること。
3. Native Sourceが別Repositoryに存在するとき、**TEI Workspace Indexだけで全Source変更が網羅されると主張しない**。対象Git blob／Source Refと権限付き取得境界を明示すること。
4. 採用対象のUI KitとFramework、外部認知機構やProvider、配布Targetについて、既存SUIの資産を優先し、必要な部分だけTEI契約へ接続すること。

この段階でSUIは正式なTEI第二Applicationとして動作していない。したがって、SUIと`order_entry`を**同一の完成済みComposition Profileで実行した**と判定してはならない。

## 最小限の追加情報は何か

採用プロファイルをどうしても表現する場合、候補となる残余項目は次に限る。

| 残余の選択 | 既存ソースから導出しにくい理由 | 正本へ戻す参照 |
| --- | --- | --- |
| Applicationに採用するArchitecture方針 | `web`だけでオンライン／オフライン／Document Editorを確定できない | ApplicationとArchitecture候補の契約・必要Port |
| Target／FrameworkとUI Kitの組 | Web/React ReferenceがWorkspaceの正式Pluginとして採用されたわけではない | Kit identity＋exact版＋Source/Artifact指紋、Target適合 |
| Project-owned拡張とUpstreamの関係 | 案件のOverlayが標準Kit変更に何を期待するかが独立の判断 | Project Source・親資産の指紋・公開Extension Point |
| Native AssetとApplication Actionの境界 | SUIの独自Domainから実TEI Runtimeへの結線は未実証 | Native Source／Action Contract／Adapterの所有者 |
| 構成選択のReview履歴 | Sourceの存在・versionだけでは誰が選択したかわからない | Review Decision／Evidenceの正本 |

**Profileに置かない**：Data・関数従属性・Business Key・Validationのコピー、Database Instanceの第二設定、Secret値、Permission Grant、Provider適合の自己宣言、Accepted業務ルール、構築・実行済みとする真偽値。これらを保存し始めたら、Core/既存Lock/Environmentの二重管理を疑う。

## 対応する検証の所有者

- **Scope/Impact**：E3/E4/E6/E7の局所Source変更検査。全依存解析の拡張は[#1120 / PR #1139](https://github.com/hat47x/tei-platform/pull/1139)など既存の型付き参照解析へ委譲する。Composition専用の第二依存グラフを作らない。
- **ParserとSource Indexの同一bytes**：[#1017 / PR #1111](https://github.com/hat47x/tei-platform/pull/1111)の責務。E7 Envelopeのラベルだけでは出典の真正性を証明しない。
- **ValidationとBusiness Procedureの差**：[#915 / PR #1040](https://github.com/hat47x/tei-platform/pull/1040)の責務。UI/DBの変更影響から業務的正当性を推測しない。
- **Native SUI Action・保存・権限**：[#993](https://github.com/hat47x/tei-platform/issues/993)とSUIの既存正本の責務。E8のSource観察は実接続の代替にならない。
- **Web要求・認証境界**：[#1043](https://github.com/hat47x/tei-platform/issues/1043)など既存Web構成の担当。KitのRenderer置換でRuntime Authorizationを変更しない。

## 統合と次のGate（2026-10-10時点）

- Draft [PR #1051](https://github.com/hat47x/tei-platform/pull/1051)は、`work/issue-1016-composition-e2`にE2〜E9の試作・Goテスト候補を含む。**Go E5テストとtrusted全体Repository Healthのexact-head PASSは未取得**。合成のNode/V8試験成功を統合資格としない。
- 2026-10-10の比較でこのlaneは`main`に対して大きく遅れていた（照合時ahead 85 / behind 486、以後変わり得る）。無造作なrebase／force update／早期main統合はしない。まず並行PR #1111／#1139の所有境界と現在のAPI差を突き合わせ、必要なら安全な通常mergeを**担当laneの調整を経て**選ぶ。
- 合格が必要な実行順序：`go test ./reference/application-composition`、同ディレクトリのNodeテスト、実`reference/workspace`でのE5 CLI→E7比較、実SUI側のNative/保存/Permission試験、最後にexact-headの**`tools/repository-health-check.sh`全項目PASS＋実行前後clean＋current main照合**。
- **Go testやfull healthが未実行なら、PRはDraftのまま保持**する。PRの`mergeable=true`は十分条件ではない。正式なProfile Schema、Capability Kind、独立Lock、一般化された継承機構の採用ADRは、その実証が終わるまで作らない。

## 受入判断（今は保留）

**採用する条件**：二Consumerで、既存資産の選択・接続に要する重複記述、手動変更、生成AIへの追加質問、Version追従時の再作業が、独立Profileなしの場合より実測で減ること。上流更新・欠落・権限不足・別Targetでは偽の成功が増えないこと。

**不要とする条件**：既存Application／Environment／Plugin Lock／UI Kitの参照だけで必要な構成判断を記録でき、追加Profileが既存定義を複写するだけになること。この場合は新しいSchemaを作らず、開発支援のPrompt／read-only Projectionの改善へ収束させる。

測る値：案件ごとの新規定義行数／変更ファイル数／独立した正本の増加件数（原則0）／手動接続・再検証回数／AI追加質問数／Source変更の誤失効・見落とし件数／実Target E2E結果。こうした**反証可能な実測**なしに「構成機構が必要」「完成」と判定しない。
