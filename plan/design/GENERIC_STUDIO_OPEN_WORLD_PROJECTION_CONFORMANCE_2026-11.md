# TEI Studio標準UIの開放的拡張性：標準データ構造／意味Plugin／汎用Projectionの適合計画（2026-11-02）

## 1. 横断判断と今回の差分

Owner方針は **Script/DSL-first、標準データ構造からTEI Studio共通UIへ投影、ノードや専用画面は任意**。[#2619 Everything is Data](https://github.com/hat47x/tei-platform/issues/2619)／[#2620 Role](https://github.com/hat47x/tei-platform/issues/2620)／[#2621 Markdown authoring](https://github.com/hat47x/tei-platform/issues/2621)／[#2622 Data Operations](https://github.com/hat47x/tei-platform/issues/2622)、[#2617 Script-first](https://github.com/hat47x/tei-platform/pull/2617)、[#2625 Syntax Sugar](https://github.com/hat47x/tei-platform/pull/2625)を設計の上流とする。

今回の不足は**一般のRecord/Sequence/Value/Refの表現と、それを解釈するPluginが追加・未対応・未受理となった場合のStudio共通画面の退避動作**を、実装差分数と編集資格で検査すること。#2617は一つのScript/Control記述とその別Domainへのdescriptor交換、#2625は構文からplain-data候補への正規化を局所検証済み。ここでは**両者を製品内で統合したと誤認しないため**に、意味未解決を扱う構造的な共通表示と、Role拡張による変化部分を切り分ける。

「無限の拡張性」は任意の対象を無条件に解釈できるという保証ではなく、**Domain/Operator/Scriptの新語彙が増えても、Studio Coreの画面種別・固有分岐を増やすことが拡張の必須条件にならない**という開放的な製品アーキテクチャ目標として検証する。

## 2. 合成すべき責務（第二正本を作らない）

| 境界 | 担当／責務 | 今回の検査対象 |
| --- | --- | --- |
| Source／表層 | #2621／#988：Markdown表・簡潔DSL・式・Graphviz・native sourceを版・発生箇所付きで保持 | Source byteと意味候補は別。例示／引用は実行宣言にならない |
| 標準Data／Role | #2620：plain Record/Sequence/Reference/Expressionの意味役割を既存Resolver経由で固定 | `role: trusted-script`という値を受理済みRole／Grantに昇格させない |
| Domain Plugin | #988／#951：Operator意味、型・参照・依存・診断、Source span | 新Roleの追加／未対応にStudio専用画面を要求しない |
| Studioの共通UI | #1124／#1824：text、汎用表・ツリー・Property・参照、Diff/Preview、原文Navigate | 表示に必要な最小構造と、意味注釈の有無を分離 |
| Context／Claim | #2506／#2516／#2582：Scope、存在・NULL・未取得、Freshness、Consumer開示 | 未取得／非開示を自動falseや不存在にしない |
| Action／Target | #2598／#1174／#1129：Intent、実Request、Authority、Effect、Receipt | UIの`visible/enabled`、read-only inspect、Source候補Patchから実Command権限を導かない |

**設計上の最小合成**：

```text
Source A/B/C (Markdown / DSL / JSON / Graphviz ...)
        │  surface-specific parse + retained occurrence
        ▼
Plain standard Data candidate  ── qualifying Role/Operator/Meaning binding
        │                                        │
        ├── Generic Studio inspect/read-only ───┤ optional semantic annotation
        │                                        │
        └── Proven invertible Source mapping ───┘ candidate patch → existing Acceptance
                                                 (not automatic execution grant)
```

Source、plain candidate、Accepted Meaning、実Load Artifact、実Effectは別々の同一性／資格を持つ。Sourceの文字列そのものがRoleであると推定する設計にはしない。

## 3. 共通UIの能力は段階的に増える

**最低限の通常表示**は、任意の公開済みData Sliceに対する `path / structural type / disclosed value`、参照・診断、SourceへのNavigateとする。既存のData/Function/Business固有画面やノード図が利用可能なら追加Projectionとして組み合わせる。

- **未対応Role**：取得が許されたplain-dataだけ表示。意味Annotationはunresolvedとして保持。未知の形を新しい実行Capability・Action・Secretとして自動解釈しない。
- **Role Plugin追加後**：同じData Slice／同じ汎用Rendererを維持したまま、適格な意味Annotationと診断・依存参照が増える。新PluginによるStudioコード変更ゼロを初期比較指標とする。
- **Source editable**：Source occurrence、単一逆写像、Revision、差分、対象Typeが確定した箇所だけ**候補Patch**を提示し、既存Preview/Acceptanceに渡す。
- **Source derived/ambiguous**：描画とNavigateは継続し、原文への安全なPatchが得られるまではread-onlyとする。繰返し出現する同一式・Macro展開・集計Viewの逆写像は別扱い。
- **実行/Disclosure**：表示・活性のScriptが指定するUIの状態と、現在Hostの権限を分離する。Dataの一部が開示不可ならConsumerに非公開情報の値・存在・件数・診断を漏らさない。汎用RendererはProviderから物理Readを行ったかを証明しない。

Studioに固有UIを追加する場合は、**専門的な操作価値・入力能力・実測の利用改善**がGeneric UIでは達しないことを先に比較する。専用UIが追加された後もSource／Meaningの正本とAcceptanceは共有する。

## 4. 新しい限定Fixtureと実証可能な範囲

[合成JS Fixture](../fixtures/studio-open-world/test_open_world_presentation.js)（Nodeから実行可能なIIFE）を追加。Domain固有の画面分岐なしに、以下を比較する。

| 試験 | 保証を検査する観点 | 実施状態 |
| --- | --- | --- |
| OW-01〜04 | 単一の共通表示にpath/type/value、boolean false／number 0／明示NULLが残る | 限定合成 PASS |
| OW-05〜07 | UI条件、シミュレーション運動、書籍章の**三つの異質構造**を同じRendererで表示。未知のRoleはplain-dataへ退避、Plugin追加で意味Annotationのみ増える | 限定合成 PASS |
| OW-08〜09 | `role: trusted-script`記述、未受理Roleは実Actionや受理済み意味にならない | 限定合成 PASS |
| OW-10 | 許可済み公開Sliceが同じ二世界の表示は、非開示値だけの違いから変化しない | 限定合成 PASS |
| OW-11 | Graphのnode/edgeもRecord/Sequenceで表現して通常表示できる。専用Graph UIは任意 | 限定合成 PASS |
| OW-12〜15 | Source-aware Patch候補は正しい単一Occurrence/Revisionでのみ作る。古い版、派生、多重出現はreadonly | 限定合成 PASS |
| OW-16 | `enabled=false`等のUI表示状態はHost Capability/Effect権限でない | 限定合成 PASS |

**実行結果**：このFixtureの全ソース文字列を、GitHub書込み前に同じV8 JSコードのまま評価し、**16 assertions PASS**。GitHubファイルへUTF-8で保存し、再読取によるblob照合を行う。Node実プロセス・GitHub Actions・本物のStudio／Parser／標準Data Canonical／Plugin Inspector／Policy／実Targetはここでは**NOT RUN**。特にOW-10は与えた仮のDisclosure済みProjectionに対する有限比較であり、実Providerの物理ReadやTiming/サイズ/ログを含む完全非干渉の保証ではない。

## 5. 既存Programでの後続受入順序

1. **E0（#2619/#2620/#2621）**：同じplain-dataのRecord/Sequenceを、2種類以上の構文／Source occurrenceから生成し、受理MeaningとSource identityを区別する。#2625の限定結果を比較元とする。
2. **E1（#988/#951）**：既存のPlugin resolution/PinからRole/Operator意味Annotation・型/依存・Source spanを供給する。未知Roleのgeneric表示・diagnosticを比較し、第二Resolverは作らない。
3. **E2（#1124/#1824）**：実`tei inspect`／標準Dataから、同じStudio Core UIで**業務UI条件と別Domain**の二つを表示。Plugin差し替え前後のStudio固有コード変更量、意味情報の範囲、Source編集往復を観測する。
4. **E3（#1180/#1163）**：Headlessで対象Scope、Meaning/Plugin Revision、Plugin未解決、ValueのNULL/missing/0、断片化したContext、Source編集の拒否を既存Scenarioへ接続する。互換Targetでの実評価とUI描画は別資格。
5. **E4（#1824/#2391/#1000）**：Studio Web/Desktop／実生成UIで、標準UIの同じ表示と元Sourceの編集・再評価を観測。Computer Use/Test向け外部制御は既存#2391を使用し、Production実行物の物理除去は別に資格確認する。

**受入指標（提案）**：新Domain Pluginを2件追加した場合、(a) Studio Coreのドメイン別UI変更ファイル数=0、(b) generic text/property/diagnosticの表示可能率、(c) Sourceへの正しい逆写像率と正しくreadonlyに留めた率、(d) 未開示情報を推測する差分0、(e) source/meaning/artifact/実行証拠の誤同一視0を測る。数値0は期待する**製品試験の合格基準候補**であり、現時点で測定済みの製品指標ではない。

**既存Critical Path**：#1119のP0-a（実Effect/Receipt）・P0-b（二Function/Owner-write）およびG0〜G4／#267 Owner意味受理の優先順は変えない。新Host、第二Acceptance Source、汎用Execution Engine、独立表示用Core固定enumをこの計画から導入しない。
