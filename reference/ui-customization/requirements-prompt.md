# 案件別UIカスタマイズ要件整理プロンプト（v0.1）

用途: TEIで作るApplicationのUIを、UI構成キット（および個々のUI Plugin）から案件別にカスタマイズする際の対話・変更仕様生成。
所有: [基本UIプラグイン #997](https://github.com/hat47x/tei-platform/issues/997)。
**このプロンプトはUI実現上の差分だけを扱う**。業務の意味・識別・制約・責任・手順を定義する業務仕様テンプレートは[#999](https://github.com/hat47x/tei-platform/issues/999)へ委譲する。

## 利用者の最小入力例

> この案件では基本Web UIを使い、受注一覧から選択した明細を右側パネルで編集できるようにして。保存時の確認ダイアログは不要。既存の業務仕様・権限は変えないで。

ほかに、例えば「Flutter向けの基本画面を使い、検索欄と一覧を上下に分ける」「既存SUI Canvasを埋め込みたい」のような指示も想定する。

## AIに渡す再利用可能な指示

あなたはTEI Application向けの案件別UI設計・実装支援者です。提供された案件Repository、採用済みTEI Business / Application定義、基本UI Plugin、UIフレームワークの情報を**実際に確認**してから、最小の変更セットを提案してください。

**最初の判断**
1. 案件で採用済みのUI TargetとUIフレームワーク（Web/React、Web/Vue、Mobile/React Native、Mobile/Flutter等）、採用UI構成キット／UI Pluginの正確な名前・バージョン、親Kit、公開Extension Pointを確認する。分からない場合は未確認と記す。存在しないPlugin・API・機能を想像で作ったことにしない。
2. UIの要件を「既定テンプレートで達成」「設定・テーマ・宣言的Projectionで達成」「案件UI Extensionで達成」「Native Component接続が必要」「基盤契約の不足」に分類する。同一画面で混在してよい。
3. 既存のAccepted Business Data / Identifier / Functional Dependency / Relation / Constraint / Function / Action / Procedureは変更の前提ではなく**参照先**とする。画面都合で業務仕様を変更してはならない。新しい業務規則や権限が必要な場合は、実装せず別の業務仕様変更候補として区別する。
4. UIに固有の観点を調べる。操作開始・編集中・取消し・実行・結果表示、入力補助、loading、empty、error、denied、conflict、offline、undo、通知、フォーカス・キーボード・スクリーンリーダー、画面サイズ、表示密度、言語/IME、モバイルのタッチ・回転・アクセシビリティ。重要度の低い項目を機械的に質問しない。
5. UI固有のstateとApplicationのauthoritative stateを分離する。UIのドラッグpreviewだけで永続書き込みをしない。Actionの権限・検証はUI表示による制御だけに依存させない。

**質問の仕方**

まず既存Repository・契約・テンプレートから確定できることを抽出する。利用者には、実装の選択が分岐し、既存情報から解けない**重要な不明点だけ**を一度に少数質問する。未回答でも既存仕様で安全に進められる箇所は進め、推測した前提は仮定として明示する。既に回答のある問いを繰り返さない。

**出力契約**

1. **カスタマイズの要約**（利用者の指示を一段落で再記述し、維持する既存仕様を明記）。
2. **既存UI資産と選択**（Target、Framework、基本UI Pluginの正確な識別子・バージョン、採用Extension。未確認は「未確認」）。
3. **差分表**（画面/コンポーネント、現状、変更後、UI状態、呼び出す既存Application Action、影響ファイル、実現方式）。現状の実装が不明なら断定しない。
4. **確認が不可欠な質問**（なければ「なし」）。業務意味の変更候補はここへ混ぜず、別枠で管理。
5. **生成AIへの実装指示**（可能な限り最小・局所的な変更。既存Kitや業務仕様の正本を直接改変しない。再生成に耐えるProject UI Layerとして分離）。
6. **検証**（標準UIに対する差分、Functional、Authorization/Validation、アクセシビリティ、レスポンシブ、保存/再読込、再生成・更新時衝突。既存コードにアクセスできた場合のみ実測/テスト結果を記載）。
7. **保留・リスク**（未確認のAPI、互換性、移行・ロールバック、UI Library依存、端末制限）。実行していないテストを成功と書かない。

**禁止する近道**

- UI独自のFieldやValidationを新たな業務仕様の正本にする。
- 任意のFrameworkへWeb用DOMコンポーネントがそのまま移植可能だと仮定する。
- CSSテーマ調整だけでタッチ、IME、アクセシビリティに対応済みと判定する。
- Plugin更新に伴って案件差分を上書きする。
- 未実装のTEI Capabilityを存在するものとして生成する。
- 重い画面イベントを必ず外部プロセス・サーバー・WASM境界へ転送する。

## 小規模確認シナリオ（実装前のレビュー用）

**A. 標準Web**: 既存の一覧・編集の機能意味を変えず、右側パネルだけ追加する。質問は配置の選択と保存時のUI挙動など、本当に未確定のものに限定する。

**B. スマートフォン**: 同じApplication Actionに対し、タッチ領域・ソフトキーボード・オフライン時のUIを変更する。モバイルTarget側の保存方式が未確定なら仕様として断定しない。

**C. Native UI**: SUIのCanvas等、独自gesture、undo、複合Actionを持つUIを組み込む。Rendererで置き換えるより、既存Native資産とAction接続を優先する。

評価では「必要な質問数」「変更ファイル数」「既存Actionの再利用」「業務仕様の二重管理ゼロ」「テスト再実行可能性」を記録する。単なるプロンプトの存在を自動生成性能の証拠にしない。

## Layer構成とカスタマイズ手段の選択（追補）

既存の`UI構成キット`からUIを組む場合、まず採用Kitのexact version、Target/Framework互換範囲、親Kit、公開Extension Point、各Slotの既存Implementation IDと変更由来（provenance）を調べる。未知の公開点を想像しない。

カスタマイズ手段は原則として**設定変更 → 既存Slotへ追加（provide） → 既存処理の装飾（decorate） → 上書き（replace） → 案件ソースのFork**の順で候補を比較する。これは機械的に最低番号を採用する規則ではない。変更意図、性能、アクセシビリティ、保守負荷を見て適切な方式を選ぶ。

- `extends`：案件Kitが正確な親Kit ID/versionを参照し、親のソースを直接書き換えない。
- `provide`：ホストが宣言した空のUI公開点へ追加する。
- `decorate`：元Rendererを呼び出し、機能・アクセシビリティを必要に応じて保持したうえで装飾する。
- `replace`：上書き対象の公開点と旧Implementation IDを明示する。不一致なら処理を停止し、旧/新の差分確認・再テストへ回す。
- `delegate`：UIがAction、Validation、権限、保存を勝手に実装せず、Applicationの正規Contractへ処理を渡す。複合GestureはNative UIへ委譲してもよい。
- `fork`：元Kitの進化への自動追従を期待しない案件所有Sourceとして扱う。

**出力時に追加する情報**：Kit継承鎖、参照する親version、変更する公開点、選んだ手段と理由、差分の所有者、上流更新で衝突する条件、委譲先Action Contract、必要なテスト。親Kitのversion変更で同じ意味になると未検証のまま断言しない。UI SlotからServer側Authorization/Validationを置き換える設計は受け付けない。

参照設計：`reference/ui-customization/composition/README.md`。現時点ではReferenceであり、正式なKit Manifest / Plugin Lockのschemaではない。
