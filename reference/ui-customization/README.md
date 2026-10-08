# 基本UIプラグインと案件別カスタマイズ — Reference方針

Status: design / prompt reference（Web/Reactの最小StarterはReference実装済み。ただし実ReactDOM / 正式Plugin化は未了）
管理Issue: [#997 基本UIプラグイン](https://github.com/hat47x/tei-platform/issues/997)
独立した別課題: [#998 TEI内部の業務仕様テンプレート](https://github.com/hat47x/tei-platform/issues/998)

## 厳密に分ける二層

**基本UI Plugin**は「あるプラットフォームやフレームワークでApplication UIを作り、TEIへどう繋ぐか」を提供する**技術的な出発点**である。UIホスト、画面遷移、汎用表示・入力部品、Action Invocation、Permission/Validation結果表示、busy/error状態、アクセシビリティ、テーマ・多言語・拡張点を扱う。**個別案件が差し替え・改修することを前提**とする。

**業務仕様テンプレート**は「受注・承認等の業務をどの意味・制約・手順で設計するか」という**TEI内の再利用可能な設計素材**である。Dataの識別子、関数従属性、Relation、Business Function/Action、Actor・Procedure・例外を扱い、案件へ適用した後のAccepted業務仕様と区別する。RendererやUI Frameworkの実装を所有しない。

結合するのは、**案件のAccepted業務仕様から導出したApplication/Action Contract**である。UI Pluginがテンプレート業務規則を複製したり、UI操作に基づいて業務上の正本を書き換えたりしない。

## プラットフォームとフレームワークの二軸

「Web」「スマートフォン」は提供先Target群、「React」「Vue」「React Native」「Flutter」等はUI実装Framework（またはToolkit）であり、一つの列挙型へ無理に畳まない。

| Target群 | 基本Plugin候補 | 固有論点 | 初期位置づけ |
| --- | --- | --- | --- |
| Webブラウザ | React、Vueなど | DOM、routing、responsive、a11y、SSR、Web専用API | **Web/React 1本で縦切り** |
| スマートフォン | React Native、Flutterなど | touch、software keyboard、OS権限、offline、端末戻る操作、App配布 | **Mobile 1本で縦切り**（採用FrameworkはEvidenceで選ぶ） |
| デスクトップ | Wails/WebView、ネイティブtoolkitなど | window/process boundary、local files、OS API、packaging | WebViewは既存Web Plugin再利用範囲から検証 |
| その他Target | kiosk、組み込み、専用業務端末など | 入力装置、表示性能、accessibility | 実Consumer要求が出たら追加 |

一つのTarget上に複数Framework候補があっても、v1ですべての組合せを完成させることは要求しない。必要になったFrameworkをPlugin候補として評価する。React DOMコンポーネントをReact NativeのHostへそのまま持ち込めるとは扱わない。FlutterがDart/Flutter側で実装されることも、TEIのGo/JavaScript Runtimeと混同しない。

### 共通契約とPlugin実装

```text
TEI内の業務仕様テンプレート (#998)
    → 案件Accepted業務仕様
    → Application / Action Contract（Framework非依存）
    → Target / Framework別の基本UI Plugin (#997)
       ├─ 標準画面・標準部品
       ├─ 案件UI拡張・差し替え
       └─ 案件固有のNative UI
```

共通契約が保証すべき対象はAction入力、結果、busy/denied/validation/errorの意味、権限・Revision・保存の境界であり、**すべてのpointer eventを同じ構造へ変換することではない**。SUIのような高頻度・複合操作はNative状態に留め、適切なApplication Actionへ接続する。

Pluginとしては、少なくとも論理的identity、version、TargetとFrameworkのcompatible range、Renderer、Extension API、依存・CSS隔離、ロールバック可能性を区別する。これらは**契約候補**であり、現行TEIがすべて実装済みとの主張ではない。

## AIによる案件別UI要件整理

再利用可能な指示は[requirements-prompt.md](requirements-prompt.md)を使用する。基本UI Pluginを参照し、利用者の短い指示からUI差分・確認点・実装箇所・テスト計画を導く。

- 既に分かる条件を繰り返し質問しない。
- **UIの変更**と、別の承認を要する**業務仕様変更候補**を分離する。
- プロジェクト固有のUI変更は、原則Plugin本体への無条件上書きではなくExtensionに保持する。
- 生成AIによるコード変更、検証、差分確認は実際に行ったものだけEvidenceとする。

## 受入試験案

1. Web/Reactの基本FormまたはListを一つ用意する。案件固有の「右側編集パネル」等を短い指示で追加し、既存Action/Validationの意味を保つ。
2. スマホ用の一Frameworkで、同じApplication Actionを別レイアウト・touchとソフトキーボード対応で実現する（Web UIのコード移植を必須としない）。
3. Native Canvasを使うSUIについて、基本Rendererへ押し込まず同じApplication境界に接続できるか確認する（[#993](https://github.com/hat47x/tei-platform/issues/993)）。
4. Plugin更新後に案件固有の変更が失われず、互換性違反と競合を診断できるか確認する。
5. Frameworkごとに差分行数・変更ファイル・AIへの追加質問数・手動同期数・Action再利用数・bundle/起動/入力遅延を記録する。

**実装済みReferenceの入口**: [web-react/README.md](web-react/README.md)（PR #1005）。既存First Web Target JSON Command契約を消費するReact Starterと案件専用サイドパネルを追加し、loopback HTTP＋React API構造スタブで9件成功した。ただし本物のReactDOM上での描画やGo ServerとのE2Eは未実証であり、Web/Reactの正式Plugin本体を完成したとはしない。

**業務仕様テンプレートの独立Reference**: [../business-spec-templates/README.md](../business-spec-templates/README.md)（PR #1009）。三視点を既存Source参照として扱い、採用提案とAcceptanceを混同しない。UI Starterの内部に業務仕様テンプレートを格納しない。

**重要**: Smartphone Plugin、Web/React正式Plugin packageの実装・ブラウザ実行検証・Framework別互換性評価は引き続き未了。基本UI Pluginそのものの普遍性やSIerの工数削減は、実際の二つ目以降の案件で検証してから判断する。
