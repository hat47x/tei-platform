# Script-first + 標準データ構造からの汎用Studio投影：Headless受入案（2026-11-02）

- **状態**：設計・局所合成Reference。正式DSL構文、Plugin ABI、TEI Studio実装、Owner意味受理は未採用／未実証。
- **方針**：Script/DSLは元のSourceに帰属する。型・参照・依存・診断・可視化は、標準データ構造／既存Language・Operator Pluginが提供する意味を用途別に**投影**する。Studioの通常のtext/property/table/tree/diff/previewを優先し、ドメイン追加ごとの特製UIを前提にしない。
- **本稿の責務**：既存計画から**共通UIへ渡す情報と、編集できる範囲／実行できる範囲を安全に区切る小さな検証票**を導くこと。

## 1. 既存の正本・Ownerとの接続

| Owner | 本検証に与えるもの／残す責務 |
| --- | --- |
| [#988](https://github.com/hat47x/tei-platform/issues/988#issuecomment-6573241174) | Language/Operator、Source span、typed dependency、NULL・未確定・Effect意味。Scriptが一次SourceのときはText、構造化Source内の式なら当該Sourceを正本とする |
| [#1124](https://github.com/hat47x/tei-platform/issues/1124#issuecomment-6573251117) | ModelerのSource-aware編集・Preview・Acceptance・Reprojection。Derived nodeから曖昧な逆編集をしない |
| [#1824](https://github.com/hat47x/tei-platform/issues/1824#issuecomment-6573259470) | Wails/Webの共通UI部品、generic inspect、テキスト/表/ツリー/診断・Sourceナビゲーション |
| [#2507](https://github.com/hat47x/tei-platform/issues/2507) / [#2506](https://github.com/hat47x/tei-platform/issues/2506) | Function PatternとContextのSemantic Subject/Grain/Context Instance、必要データ、操作Offer |
| [#2497](https://github.com/hat47x/tei-platform/issues/2497) / [#2516](https://github.com/hat47x/tei-platform/issues/2516) | 読み取りProjectorの意味保存・双方向編集の可逆性／Consumer・目的相対の開示 |
| [#2500](https://github.com/hat47x/tei-platform/issues/2500) / [#2598](https://github.com/hat47x/tei-platform/issues/2598) | View change→型付きIntent候補→実Requestの対象・時点・根拠束縛 |
| [#1174](https://github.com/hat47x/tei-platform/issues/1174) / [#2504](https://github.com/hat47x/tei-platform/issues/2504) | Host現在Authority/Effect FenceとScript実行Targetの隔離 |
| [#1131](https://github.com/hat47x/tei-platform/issues/1131) / [#2514](https://github.com/hat47x/tei-platform/issues/2514) | 独立Scenario/Oracle。描画成功や局所Fixture PASSをMeaning Acceptanceへ昇格させない |

## 2. 小さく、繰り返し適用できる構成

例示（**正式DSL文法ではない**）：

```text
visible(order.discount) = customer.kind == "corporate"
enabled(action.submit) = form.valid && !form.busy
```

1. **Source/Meaning**：Scriptテキスト（構造化Source内部ならその正本）から、Source revision / byte範囲 / Semantic target / Operator binding を既存Pluginが解決。
2. **Typed Inspect**：標準UIはPluginの解決結果（型・参照・依存・単位/時間・エラー・Effect）を、共通の表・プロパティ・ツリー・Source navigationへ投影。未知OperatorをStudioが独自に解釈しない。
3. **Scoped Evaluation**：必要な対象ID、Context Revision、選択epoch、Scope、取得済み/NULL/欠損、実評価時点、Consumer開示を揃え、Pluginへ評価を委ねる。Resultは真／偽／未確定を区別。通常Scriptが純粋でも依存値がunknownなら結論を推測しない。
4. **Presentation**：visibility/enablementなどの条件は描画に反映するが、画面上のtrueは業務Action実行許可・現在Host Grantを意味しない。未確定のvisibilityは表示可否が未決であるため、安全に公開可能な範囲だけ描画する。
5. **Edit Candidate**：Source spanが一意・現在Rev・具体的target一致・元byte一致・可逆編集が証明できる場合のみ、同じSourceへPatch候補を作る。Parser/Validator/Ownerの受理後に正本を書き換える。
6. **Action Intent**：UI操作からの変更候補は既存#2500→#2598→#1174へ送る。UI conditionがtrueでもHost/Effectが許可されるとは限らない。

この6段は**検証観点**であり、新しい固定Runtime・ABI・Authority/Approval・二つ目の意味正本ではない。SourceはどのLanguage/Targetを使うかに応じて変わり、DSLへの全面テキスト化を強制しない。

## 3. 局所合成Reference（実測）

[局所Fixture `verify_generic_script_projection.py`](../fixtures/script-first-generic-projection/verify_generic_script_projection.py) は標準Python `dataclasses`/`unittest`だけを利用。

実行：`python3 -m unittest -v verify_generic_script_projection.py`、`python3 -m py_compile verify_generic_script_projection.py`。

**2026-11-02 隔離コンテナ実測**：Python stdlibの14/14 unittest PASS、py_compile PASS。実行ファイル11220 bytes、SHA-256 `d2641822aadeb7535d1f62bddc69076e617e6e60061038d275a86ce69eaf951d`、Git blob SHA-1 `88ad61e8ab8bc5dc686eced29ca81ade802207fe`。後者はGitHubに保存した同ファイルのBlob SHAと一致した。

| Fixture | 確認した限定の性質 |
| --- | --- |
| SF-01〜05 | 日本語付きUTF-8 Sourceの**byte** spanと正確なPatch候補、stale/digest drift、derived/ambiguous、誤targetの拒否 |
| SF-06 | true/falseな条件値とRendererの見せ方、Host許可の区別 |
| SF-07〜10 | 未提供/開示対象外を同じ利用不可結果へ、NULL・古いepoch・部分Contextをfalseに潰さない |
| SF-11 | 同じ汎用Inspectの形をOrderと仮のBear Domainへ再利用（実2 Pluginではない） |
| SF-12〜13 | 未知のPlugin評価結果と型違反の診断 |
| SF-14 | 同じ文字列がSource内の別位置に出現しても、正確なbyte spanだけを書き換える |

**この試験で実装・検証していないもの**：TEI本体のParser/Language Plugin、AST/Meaning/Source Acceptance、実Studio/Wails/React UI、Node/Diff Renderer、正式TEI Context/Provider、秘密情報への実Policy、Capability/Effect、実ゲーム・実業務Target。Fixtureの`plugin_eval`は外から注入する**模擬関数**で、UIが演算意味を推測する実装ではない。Sourceを書き換える関数が返すのは**候補bytesのみ**。安全な再パース・Ownerの意味受理や書込は行わない。

## 4. 横断で残る本質的なリスク

### A. 編集可能性はSourceとConsumer/Claimに相対

同じSourceから多数のDerived node/Graph/tabledocが作られたとき、可視化できることと可逆編集できることは異なる。Macro展開、生成式、多対一Projection、版間自動対応であれば、その表示からの直接書戻しを保留し原Sourceへ案内する。

### B. UnknownとUnauthorizedは表示情報量まで分けて評価

内部診断の`missing`/`withheld`/`partial`を画面へそのまま出すと、秘密の有無・件数・Source名が漏洩し得る。本Fixtureは **未提供と未開示を同じ`unavailable`にする狭い対照**に過ぎない。実際は#2516/#1124の二世界テストで、Node数、DOM/ARIA、候補有無、診断、タイミング等の公開結果全体を比較する。Function/PluginのInspectorが出す依存先フィールド名自体も開示判定が必要。

### C. Source変更と現在の実行許可を分離

画面のenabled/hidden条件は利用体験を定義するだけ。現在のHostのEffect/Authorization/Transactionは、UIを経由しない直接Invokeでも必ず独立に検査される必要がある。公開Script/Macroは#2502/#2504のLifecycleと現権限を用いる。

### D. Script更新とRuntime世代更新を分離

[Draft #2612](https://github.com/hat47x/tei-platform/pull/2612)のMeaning revision／Definition／Instance／Provider Generation／Dirty Session／Rule発効時点は別軸。Script条件のSourceを編集しただけで、起動中の旧Functionが新条件へ切り替わったことにはならない。Derived Studio ViewのreparseとTarget Generationの切替は独立受入。

## 5. 次の製品接続（Owner間で重複実装せず）

| 段階 | 責務 | 追加の成立条件／対象 |
| --- | --- | --- |
| **P0 独立Reference** | 本Fixture／#988／#1124 | 14件の限定合成結果を読み解き、欠ける情報を既存Semantic/Plugin/Source/Rendererへマッピング |
| **P1 実TEIのinspect受取** | #988/#1124 | 同じ固定Source/Language+Operator版からtyped dependency、Source byte span、診断を取得。模擬descriptorを実取得で置換 |
| **P2 HeadlessのSource往復** | #1124/#2497/#1131 | 受理され得る編集をtyped change→Preview→Acceptance→reparse、派生/ambiguous/staleは拒否。結果のSource/Meaning版を固定 |
| **P3 二Consumer＋新Domain** | #1824/#997/#2507 | CUI/Studio read-only（将来実UI）で二つの異質なDomainを共通Inspect。新Domain追加時のStudio固有画面差分0が成立するか計測 |
| **P4 実UIとEffect** | #1824/#2391/#2598/#1174 | Web/Desktop、画面表示と実Action/Host現在Grantの独立性、二世界開示、旧Rev/epoch、実ターゲット |
| **P5 再利用効果** | #2018/#2493 | 新Plugin追加あたりのCore/Studio専用コード増分、必要Adapter、通過したMeaning/Context/Target Claim、誤成功率 |

P0のFixture PASSをP1〜P4の成功として扱わず、最新mainと各担当の既存Draftを確認して実製品接続の差分を最小化する。

## 6. Owner判断の条件

- Standard UIの表/Tree/Graphは**Derived Presentation**。正本Source/Meaning/Action/Authorityは既存基盤に属する。
- **新しいDomain Pluginが何を解釈できるか**はそのPluginのMeaning/Contractの版に依存。未知Domainでもテキスト・Source位置・型・診断など、証拠があるものは標準UIに表示できる。
- 真偽未確定で誤って操作を可能にしない設計は必要だが、権限の拒否ロジックは既存Hostへ委譲する。
- CUI/SHO/Studioへの開示はConsumerごとに別の資格。何を載せられるかをRaw/Providerの権限と混同しない。
- 低コード構成はScript/標準Dataを基本とし、専用ビジュアル編集の必要性が二つ以上のConsumer/実例で立証されたときに追加する。探索の領域・Script/Plugin数に上限を置かない。

**本計画の変更は設計と小さな合成fixtureのみ。正式なTEI API/DSL Syntax/UI部品やProvider/Host/Owner受理は行っていない。**
