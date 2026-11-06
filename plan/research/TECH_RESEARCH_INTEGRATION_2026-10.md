# TEIの技術調査を設計へ反映する方針 — 2026年10月

状態：**現行の技術調査の反映計画**  
最終更新：**2026-11-06**

## 1. 位置づけ

この文書は、2026年9月末から10月初頭までの技術調査を、TEIの製品価値、要求、既存の課題へ反映するための現行計画です。

外部の研究、軽量な認知方式、大規模言語モデル、Wasm、デスクトップ向け開発基盤などを、そのままTEIの仕様の根拠にはしません。製品の中心となる価値について不確実性を減らすために、設計上の境界、比較する観点、参照事例による検証課題を取り入れます。

順序は維持します。

~~~text
調査・実装から得られた根拠
        ↓
製品価値との照合
        ↓
既存の要求・課題への割当て
        ↓
小さな参照事例で検証
        ↓
TEI自身で得た検証根拠
        ↓
設計判断記録・要求・マイルストーンを更新
~~~

研究を読んだだけで、検証根拠の達成段階を引き上げません。

## 2. 製品価値との対応

直近の調査結果は、新しい製品価値を追加するよりも、既存の価値を具体化するために利用します。

| 製品が提供する価値 | 今回の反映方針 |
| --- | --- |
| 業務の意味を一つの正本で扱う | AIや認知処理の出力は、提案・候補・派生情報として扱う。意味を決定する権限を自動的には与えない |
| 人が実質的に確認できるようにする | 全体の再生成よりも、最小限の意味変更、その影響、未解決事項、出所を明示する。決定的な検査で確認対象を絞る |
| 技術選択の自由を保つ | Wails、WebView、Wasm、ネイティブ実行、大規模言語モデルの実行機構を、ホスト・接続部品・実行条件に分けて業務上の意味から切り離す |
| AIを必要な部分へ活用する | 正確な照合や決定的な処理で解ける問題はAIに委ねず、未解決の場合に限って軽量認知、ローカルのモデル、外部のモデルへ段階的に進む |
| 再利用と作業の再開を容易にする | 候補を作った根拠、受理した判断、契約・実装・設定・方針の版を、再現可能な検証根拠へ結び付ける |
| 説明責任を保つ | 提案、判断、実行権限、命令の受付、観測結果を別々に追跡する |
| 一人でも維持できるようにする | 提供元やモデルごとの専用機構を増やさず、候補の提示・適格性の確認・受理を共通の契約で扱う |

## 3. 直近の調査結果から定める設計仮説

### 3.1 認知方式は段階昇格する

知識探索とAI支援の作業領域では、次の順序を仮説として検証します。

~~~text
M0  明示された定義と正確な構造
      ↓ 未解決の場合だけ
M1  決定的な推論・関係図・規則
      ↓ 未解決の場合だけ
M2  軽量な連想・疎な表現・統計による候補生成
      ↓ 未解決の場合だけ
M3  用途を限定したローカルの認知モデル
      ↓ 未解決の場合だけ
M4  ローカルの小規模・大規模言語モデル
      ↓ 未解決の場合だけ
M5  外部の大規模言語モデル
~~~

M2～M5は候補の生成、順位付け、新規性の検出、判断の保留などに使えます。ただし、意味上の適用可能性、受理、書込み権限、実行時の権限を決める主体にはしません。

上位の方式へ進めた理由と、下位の方式では解決できなかった事項を検証根拠に残します。

### 3.2 情報の確かさと取得方法を分ける

「何が分かっているか」と「どの方式で情報を得たか」を、単一の確信度にまとめません。

候補として次を直交させます。

~~~text
情報の確かさに関する状態
  観測済み・推論によるもの・確認済み・判断を保留

情報を得た方式
  決定的な処理・統計的な処理・提供実装への依存
~~~

正式な名称や本番用のスキーマは、参照事例で検証した後に判断します。大切なのは、モデルの確信度や類似度の点数をそのまま判断権限へ結び付けないことです。

### 3.3 軽量な認知処理の役割を限定する

Jev系、FlyHashなどの疎な連想方式、埋め込み表現、LightGBMなどを、言語モデルに代わる万能の推論機構とはみなしません。次の限られた用途で比較します。

- 候補の絞り込み
- 候補の順位付け
- 新規性や不一致の検出
- 既知の誤検出の抑制
- 上位方式への移行や判断保留の手掛かり
- 局所的な処理先の振分け

その結果は必ず現在の情報源、意味の正本、検証根拠と照合し、決定的な方法で適格性を確認します。

### 3.4 ローカルの言語モデル実行機構は任意の提供実装とする

OllamaやLM Studioなど既存のローカル実行機構は、利用できる場合に再利用を検討します。ただし、TEIの中核に必須の依存先とはしません。

モデルのデータ、実行機構、版、設定は業務上の意味から分けます。再現性が必要な検証では、利用した提供実装、モデル、版、設定を記録します。

モデルのデータや実行機構がなくても、意味の正本全体と、決定的な方法による検証・受理の仕組みが成立する構造を保ちます。

### 3.5 実行物の可搬性とアプリケーションの実行先を区別する

Wasmは、Webなどの画面や操作を提供する実行先そのものではありません。

~~~text
正本の定義と意味上の操作
        ↓
型を明示した決定的な実現方法
        ↓
ネイティブ実行・Wasm・外部プロセスなど
        ↓
実行先・ホストの接続部品
        ↓
実行・観測
~~~

Hikeのような極小のWasm向け実装は、小さく持ち運べる実行物を検証するための比較対象とします。ただし、TEIのGo製中核を置き換える前提にはしません。Go、TinyGo、Hikeに似た言語の一部分などを標準として固定せず、実行物の形式、ABI、ホストが提供する機能、資源の条件を比較できるようにします。

### 3.6 ホストは業務上の意味を決めない

Wails、WebView、ブラウザー、デスクトップアプリケーションの起動状態、画面の復旧、要求の取消しなどは、ホストと接続部品の責務です。

こうした失敗を業務上の結果と取り違えず、ホスト側の観測・診断結果として保持します。編集環境のWailsホストと、生成したデスクトップアプリケーションの実行先も区別します。

### 3.7 実行できるかどうかは契約と現在の権限で判断する

AI、規則、人のどの方針から始まった場合でも、外部に影響を及ぼす呼出しは、検証済みの同じ意味上の操作を通して扱います。

再現や監査が必要な参照事例では、少なくとも次を個別に識別できるか検証します。

- 意味上の操作と契約の版
- 実装の版と照合値
- 設定と対応付けの版
- 現在の方針と権限の記録
- 資源の特定結果
- 実行結果
- 観測結果と検証根拠

過去の受理結果やモデルの記憶だけを、将来の実行権限の根拠にはしません。

## 4. 既存の課題への割当て

### ISSUE-000047 — 既存知識を優先する意味の探索

最優先では、読み取り専用の探索に段階的な認知処理を適用します。

1. 正確な識別子、情報源、関係図の近傍を基準にする
2. 決定的な推論によって候補を増やす
3. 軽量な連想方式や統計方式を候補の生成手段として比較する
4. ローカルや外部のモデルは、下位の方式で未解決事項が残る場合だけ使う
5. 最後に、適用可能・不適合・未確定のいずれかを、決定的な方法で確認する

評価では上位候補の的中率だけでなく、次の点を分けて確認します。

- 役立つ候補を発見できたか
- 誤検出や不適合な候補があるか
- 未確定事項を判断保留できるか
- 追加の確認に必要な負担
- 古い版や出所の追跡に関する失敗
- モデルや提供元への依存
- 応答時間、実行物の大きさ、実行時の依存先

### ISSUE-000027 — AIによる変更提案と共通の受理境界

変更提案の生成方法と判断権限を分けたまま、先ほどの各段階で作られた候補を受け取れるようにします。

参照事例では、変更提案から少なくとも次の情報へたどれることを確かめます。

- 候補の出所と生成方式の種類
- `baseModelRevision`
- 変更対象を正確に示す識別情報
- 最小限の意味変更
- 根拠となる情報源と検証結果
- 未解決事項と判断保留
- 利用したモデルや提供元の情報
- 受理の前後に行った決定的な検証と準拠性の確認

確信度やモデルの評価点だけから、受理という判断を自動的には導きません。

### ISSUE-000026 — 二つ目の実行先と実行物の可搬性の検証

二つ目の実行先を選ぶ際には、利用できる機能だけでなく、同じ意味上の操作を異なる実行物やホストで動かした場合の違いを観測します。

二つ目の実行先をWasmに限定しません。選定時には、次の点も比較します。

- ネイティブの実行物と可搬性のある実行物を交換できるか
- ABIとホストが提供する機能の違い
- 起動、復旧、資源の条件
- 意味上の結果と観測結果の一致
- 実行先に固有の事情を意味の正本へ逆流させないこと

### ISSUE-000053 — AIを用いた作業の再開と運用上の知識

開発者やエージェントが作業を再開するときには、チャット履歴や提供元が保持する記憶に頼らず、現在受理されている意味と検証根拠から必要な文脈を再構成できるかを検証します。

特に、実行候補について次を区別します。

~~~text
知られている操作
  ≠ 現在も適用可能
  ≠ 現在の実行権限がある
  ≠ 必要な資源が利用可能
  ≠ 実際に呼び出した
  ≠ 完了を観測した
~~~

作業再開時に渡す文脈情報にモデル固有の状態を必須とはせず、契約、実装、設定、方針の版をたどれることを優先します。

## 5. 編集環境での人による確認への影響

RQ-010とRQ-028では、AIが作った自然言語の要約だけを、人の確認に用いる画面にはしません。

情報を集約した表示で、少なくとも次の項目を区別できることを、今後の参照事例で検証する条件に加えます。

- 正本に記述された事実と観測した事実
- 推論によって作られた候補
- 確認済み・受理済みの判断
- 未解決事項・判断保留
- 情報源、検証根拠、影響の詳細への参照
- 提供元やモデルから得た補助情報

人は評価点そのものではなく、意味の差分とその根拠を確認して受理します。

## 6. 実装順

新しい横断的な専用機構や判定段階は追加せず、既存の課題を次の順に詳しく検証します。

1. ISSUE-000047で正確な照合と決定的な処理を基準とし、上位方式へ進める条件を参照事例で定める
2. 軽量認知を候補生成に限定して比較する
3. ISSUE-000027で候補から最小限の意味変更を提案し、共通の受理境界を通す
4. ISSUE-000026で実行物の可搬性とホストの違いを、二つ目の実行先の比較に加える
5. ISSUE-000053で作業再開時の文脈と、実行の出所の追跡を結び付ける
6. 編集環境とRQ-028で、意味の差分、未確定事項、出所を人が理解できるか、実際の検証根拠を得る

この順序は厳密な直列ではありません。1～3と二つ目の実行先の検証は並行できます。ただし、下位の方式で基準となる結果を確認する前に、上位の方式の採用を固定しません。

## 7. 非目標

- Jev、FlyHash、埋め込み表現、LightGBM、特定の言語モデルをTEIの標準的な推論機構にすること
- OllamaやLM Studioを必須の実行機構にすること
- Hike、TinyGo、WasmをTEIの中核の必須実装にすること
- モデルの確信度を意味の検証根拠や受理・権限の判断に使うこと
- 編集環境やエージェントのための第二の正本を作ること
- 提供元ごとに異なる受理規則や利用許可の意味を設けること
- 研究の評価点をTEIの固定的な成果指標として使うこと
- 研究だけを根拠に製品要求の検証段階を引き上げること

## 8. 完了条件

この技術調査の反映作業は、文書を作成しただけでは完了とみなしません。

少なくとも、既存の課題に対応する参照事例で、次のことを説明できる状態を目指します。

- 決定的な方法で解ける問題を上位のモデルへ不必要に送らない
- 軽量な認知処理やモデルが候補生成に役立っても、判断権限は与えない
- 未確定事項や判断保留を成功として扱わない
- 提案を最小限の意味変更として共通の受理境界へ渡す
- モデルや提供元を交換・削除しても意味の正本は変わらない
- 実行物やホストを交換しても意味上の操作を追跡できる
- 現在の権限、資源、観測結果を過去の提案やモデルの記憶と分ける
- 人による確認で、意味、推論、未確定事項、出所を区別できる

検証根拠の達成段階は、各要求の監査において、実際の参照事例が成立したことを確認してから更新します。

## 9. 2026年10月9日の追加調査 — 実用上の価値につなぐ

この追補は**採用前の設計・実証候補**です。外部研究の成績をTEIの達成度に読み替えず、上記の統合方針、既存Requirement、受入境界は変更しません。まず、利用者が「何を安心して変更できるか」「何が本当に動くか」を確かめるため、小さな比較を追加します。

### 9.1 仕様の不足を、実装前に発見できる価値

[SpecBench（2026-05）](https://arxiv.org/abs/2605.30314)は、コードの修正能力とは別に、仕様案に残る**欠落・曖昧さ・不整合・誤った前提**の検出能力を評価しています。既存の設計議論とコードの文脈があっても、仕様上の不足を確実に見つけられるわけではありません。

[RADIANT（2026-07）](https://arxiv.org/abs/2607.16708)は、要求から複数のモデルを生成し、要素単位の追跡関係と変更影響の計算を接続する研究です。構文上の妥当性と意味の正しさが同じではない点も重要です。形式検証の方式そのものをTEIに取り込むのではなく、異なる設計視点を通した**反証可能な変更影響**に注目します。

TEIの設計へ反映する観点：

- RQ-026のデータ、機能・操作、業務手順の相互参照から、要求の欠落、矛盾、不明点を実装前に示す。
- RQ-028では、生成したコードの一覧よりも、影響を受ける規則、操作、データ、利用者の権限、未解決事項を人が確認できるように示す。
- 既存の#982、#998、ISSUE-000075、#915、#267へ割り当てる。要求、業務上の意味、受理済みの情報源を正本とし、AIが発見した問題は確認が必要な候補として扱う。

**REF-SPEC-01（候補）：** `order_entry`にある顧客の参照規則を題材に、欠落（必要な参照先がない）、曖昧さ（更新できるか分からない）、不整合（読み取り専用なのに更新が要求される）、古い前提（基準版が変更された）の四つを用意します。同じ変更案について、(a)通常の情報源の探索と、(b)TEIによる依存関係・差分・未解決事項の表示を比較します。重要な不足の見落とし、誤警報、人が開いた情報源の数、説明できる影響範囲、確認時間を別々に測ります。#915の既存の参照事例にない事実や権限は推測で補いません。

### 9.2 見た目ではなく「操作が働く」価値

[VISTA（2026-09-29改訂）](https://arxiv.org/abs/2605.26144)は、画面を描画できたことと、必要な部品を見つけて実際に操作できることを別々に測ります。TEIでも既存Gate Cの視覚評価を、機能や人による承認の代替にはできません。

TEIの設計へ反映する観点：

- RQ-013、RQ-010、RQ-021の評価では、画面の部品が存在すること、操作が正しい業務上の行為につながること、拒否すべき操作が拒否されること、現実の状態を観測できることを別々に検証する。
- #997、#1000、#1001、#259、ISSUE-000046の画面・実行先に関する検証へ戻す。画面の見た目が一致しても、業務上の結果や観測結果が一致しなければ成功とは判断しない。

**REF-INTERACTION-01（候補）：** 最初のWeb向け実行先で、顧客の参照、入力の検証、許可された更新、許可されない更新要求のうち、既存の参照事例で実行できる一連の操作を選ぶ。各操作について、画面の部品→イベント→意味上の操作→実行・検証環境の結果→観測結果の対応を記録する。キーボード操作とエラー通知は画面固有の評価として加える。見た目の評価、人による操作の確認、決定的な方法で確認した操作結果は別々に記録し、人による検証を行っていない場合は「未実施」と明記する。

### 9.3 小さな実行物を、標準機能のプラグイン化と接続する価値

[WASI 0.3（2026-06-11公開、0.3.1は同年08-11）](https://github.com/bytecodealliance/wasi.dev/blob/main/docs/releases/wasi-p3.md)では、部品を組み合わせる仕組みに、非同期処理のfutureやstreamなどが導入された。一方、実行基盤と各言語の開発ツールが同じ速度で対応するとは限らない。WASI 0.3が標準化されたからといって、TEIのGo製ソースをそのまますべてのホストで動かせるわけではない。

[Wails v3（2026-08時点でBeta、2026-10時点でもBeta）](https://v3.wails.io/status/)は、GoとWebViewを組み合わせるデスクトップ向けの候補として、ホストの起動と終了、複数ウィンドウ、対応付けを調べる材料になる。ただし、編集環境のホスト、生成した業務アプリケーションの実行先、プラグインの実行基盤はそれぞれ別の責務を持つ。

[TinyGo](https://github.com/tinygo-org/tinygo)はGoのソースをWasmやWASI向けにコンパイルする候補です。[Hike](https://github.com/kanryu/hike-lang)はGoに似た構文を持つ**別の言語**です。一般的なGoのソースをそのままHikeでコンパイルできるとは想定しません。小さな実行物を比較する際も、この違いを明記します。

TEIの設計へ反映する観点：

- #259の二つ目の実行先の比較に、実行物の形式、ホストが提供する機能、ABIと呼出規約、同期と非同期、権限と資源、起動と復旧、実行物の大きさ、可搬性を加える。
- #943、#973、#976、#1016では、標準機能を含むプラグインの意味上の契約、依存関係の解決、ビルド成果物の固定情報を先に保つ。Wasm形式やコンパイラ名をプラグインの識別子に混ぜない。
- Wasmの採用だけで二つ目のアプリケーション実行先が成立したとは判断しない。Webやスマートフォンの画面の実装とも区別する。

**REF-PORTABLE-01（候補）：** 外部に影響を及ぼさない小さな意味上の変換を、同じ入力と期待する出力で定義し、利用できるGoのネイティブ実行やTinyGoによるWasmなどで比較する。Hikeは別言語のソースへの移植費用も記録できる場合に限って候補に加える。生成物の照合値、依存先、ホストが提供する機能、ABI、失敗・取消し、CPUとメモリ、初回起動時間、結果の一致を記録する。WASI 0.2と0.3は開発ツールとホストの対応状況を確認して別の条件で扱い、一方の成功をもう一方へ一般化しない。実行ホストの許可、資源、リースは既存の権限の境界に従って扱い、契約を読み込んだだけで実行権限を与えない。

### 9.4 優先度と着手条件

| 順位 | まず確かめたい利用者価値 | 既存の受け皿 | 続行・見送りの判断 |
| --- | --- | --- | --- |
| 1 | 小さい業務変更で、見落としや矛盾を人が判断できる | ISSUE-000075 / #982 / #998 / #267 | REF-SPEC-01を既存Sourceと受理境界で示せるか。広い仕様生成や新規基盤が必須なら見送る |
| 2 | 画面の各操作が業務上の結果につながる | #997 / #1000 / #259 | REF-INTERACTION-01で機能・視覚・観測を別々に測れるか |
| 3 | 実行環境を替えても、同じ意味と権限条件を保てる | #259 / #943 / #973 / #976 | REF-PORTABLE-01で実際のホストとABIを記録し、既存の権限機構を再利用できるか |

この順番によって、既存の課題の着手承認、統合手順、実装の優先順位を変更することはありません。ISSUE-000075の候補Bでは、2026-10-08に業務規則の変更が受理されています。ただし、Web画面まで含めた一周の実証や全明細への規則適用を完了したわけではありません。REF-SPEC-01は、この受理をやり直すための作業ではありません。受理済みの意味と別の設計視点に残る違いを、読み取り専用で検査する候補です。

**完了の扱い：** この文書の更新は、研究を計画へ反映した段階にとどまります。参照事例の実装、比較実験、利用者による確認、性能測定、適合性の判定は実施していません。どの候補も、基準版、情報源、実行環境、期待する結果、実測値、反例を記録した段階で初めて、対応する要求の検証根拠を更新するか検討します。

### 9.5 REF-SPEC-01の最初の実施手順（静的確認済み／コマンド実行未確認）

最初の段階では新しい仕様生成器を作りません。既存の受注の参照事例と業務手順の検査器を使い、**確認できる問題と、まだ判断できない問題**を分けます。2026-10-09にリポジトリのソースを読み取り、次の境界を確認しました。以下のコマンドについては、実行結果も所要時間も記録していません。

| 確認対象 | ソースに記述されている事実 | 実行時に確かめること |
| --- | --- | --- |
| 明示された規則 | 受注の業務手順BP-05には、数量が正であることと行番号が一意であることを検証する、二つの規則への参照がある | 現在の検査器がどの参照を解決し、どれを未解決として返すか |
| 反する更新要求 | 顧客参照のBP-02を編集しようとする検証では、読み取り専用のデータソースに書込みも求めている | 想定した利用条件の衝突だけを検出し、更新権限を推測で作らないか |
| 三つの視点の違い | 2026-10-08に受理した行番号の正値規則は、アプリケーション側で三つ目の検証規則となった。一方、業務手順BP-05は二つの規則への参照を保持している | 意図的に独立させているのか、変更の影響を見落としたのか、人が判断できるか。検査器が必ず不整合と判定するとは仮定しない |
| 古い基準版の提案 | 候補Bとして保存した提案は、受理前の`baseModelRevision`を指す | 現行版へ確認なく適用しないか。過去の確認結果を現在の受理の根拠に流用しないか |
| ソースで見つからない事項 | ソースの検索で記述が見つからないだけでは、禁止されていることや存在しないことは証明できない | 必要な参照先の契約違反と、単に見つけられていない状態を区別できるか |

読み取り専用で確認するためのコマンド：

    go run ./tools/reference-business-procedure-check

    go run ./tools/reference-business-procedure-check -probe reference/business-system-design/order-entry-customer-edit-conflict.json -expect-diagnostic BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT

既存の説明は[業務システム設計の参照事例](../../reference/business-system-design/README.md)、受理の履歴はISSUE-000075を参照します。過去の提案・事前確認・確認記録のファイルは、**当時の基準版についての根拠**であり、現在の全体モデルでも受理できるという証明ではありません。

最初に作る読取り結果では、(1) 正確な情報源・意味上の識別子・衝突の根拠から**判断できた事項**、(2) 記述の不在・古い前提・業務手順固有の意図など**人の判断が残る事項**、(3) 実際の提供実装・画面・未実行の実行検証など**根拠が届かない事項**、(4) 変更の提案、保留、追加調査から**人が選ぶ事項**を分けます。

**停止条件：** 顧客の編集に関する衝突を既存の検査器で再現できない場合や、異なる視点の違いを示すために受理済みの業務定義を無断で書き換える必要がある場合は、実装を増やさず、原因を既存の課題へ戻します。成功を記録する際には、正確な対象版、実行環境、コマンド、診断、実行前後にソースが変わっていないことを記録します。今回のソースの静的な確認だけでは成功とは判断しません。

## 10. 2026年11月6日：技術短報の論文精読から既存計画への接続

**状態：文献調査による受入条件の提案。TEIでの新規テスト、実Host／DB効果、正式な意味受理は NOT RUN。**
10月までの調査はこのまま履歴として維持し、本節は11月上旬の追加知見だけを記録する。研究で報告された効果量をTEI自身の達成値に転用しない。既存のIssue、独立Expected、Source/Artifactの版、Hostの現在Authorityに接続し、新しいProof Engine、Agent Runtime、Plugin Resolver、共通Host、受理正本は作らない。

### 10.1 文献と読み取りの境界

| 文献（公開／改訂） | 外部研究から確認できる観点 | TEIでは採用を先取りしないもの |
| --- | --- | --- |
| [Modern Software Engineering with Verification Agents](https://arxiv.org/abs/2611.02285)（2026-11-01） | 仕様を先に固定し、生成コード・改変・統合結果を独立に検査する。充足可能性のwitness、空虚なPASS、mutation、意味的なmerge競合を区別 | 同論文のVIR、verified lowering、semantic merge/fleet構想は提案であり、全体系の実稼働実証ではない。TEIへの新しいIR／証明カーネル導入を決めない |
| [Beyond Source Compatibility: WebAssembly as a Stable ABI for ARINC 653 Applications](https://arxiv.org/abs/2611.00891)（2026-10-31） | 同一Wasm成果物を異なるRuntime／CPU条件で利用する際、Guest/Host ABI、整数幅、構造体配置、import、TLSの明示が不可欠 | 実証は限定されたRuntime／OS構成。異なるARINC OSでの全面可搬性、安全認証、TEI一般Plugin ABIとしてのWIT採用を証明しない |
| [ServerBench: How can your coding agents design specialized LLM serving systems?](https://arxiv.org/abs/2611.06841)（2026-11-04） | 変更後の実サービスについて、新鮮な入力、異なる負荷・状態・並列度、実応答／停止を再観測する。局所ベンチ成功と本番相当の安定性は別 | LLM推論サーバ特有のスループット・金額指標や個別最適化をTEIの標準KPIにしない |
| [M4Bench](https://arxiv.org/abs/2610.37459)（2026-10-27） | 完全な操作仕様が生成・照合の正確さを大きく左右する。値の一致だけでなく、全件の包含／除外、粒度、期間、集計の意味を独立に検査 | 特定のEHR操作例やSkill包装の性能を、TEIの業務定義が正しいことの証拠にしない |
| [Concordat: Typed Sessions for LLM Agents](https://arxiv.org/abs/2610.37461)（2026-10-27、10-30改訂） | 利用可能な操作の順序、確認段階、意図した対象を型付きセッションで制限する発想。曖昧な承認文（単なる yes）では実対象まで確定できない | Agentに表示された許可ツール集合や会話上の承認を、現在のHost Grant／作用対象の認可に置き換えない |
| [Rethinking Verification in Agent Harnesses](https://arxiv.org/abs/2610.34577)（2026-10-25） | 実装者とは独立した検査役が、要件と変更差分から適合性を吟味する方式。検査の配置と費用対効果を比較 | 論文内のモデル・ベンチマーク別改善幅をTEIの性能期待値や採択基準へ直接転用しない |
| [From Interaction to Capability: Task Advancement as State and Supervision for GUI Agents](https://arxiv.org/abs/2610.37482)（2026-10-27） | 作業全体が失敗しても検証済みの局所前進を条件付きで再利用し、開始条件・到達状態・検査方法を持つ | 過去のGUI成功履歴やAgent記憶を、最新Sourceの受理・現在Grant・DB効果の証拠としない |

### 10.2 重複しない受入条件の移管候補

| 短報起点の反証課題 | 既存Owner／実施先 | 最小Fixture・確認する負例／独立Evidence |
| --- | --- | --- |
| **TR-11-SPEC：仕様に沿った検証と空虚なPASSの排除** | #2514（独立Oracle／mutation）・#2619（定義と受理）・#1180（Scenario）・#2811（テスト記述） | 同じ受理候補に、満たす例、反例、曖昧／欠損Source、旧Revision、意図的なEvaluator mutantを用意。生成コードやAIの提案とは独立したExpectedで検査。検査対象Sourceが実際に読み込まれたか、条件が空集合で偶然真になっていないか、現ArtifactをConsumerが受領したかを別々に示す。#2514にある一つの局所Mutation killを全規則の検証済みへ一般化しない |
| **TR-11-ABI：意味契約／バイナリ形式／Host Authorityの三分離** | #951（論理Capability）・#1861（実Wasm Component／二Host）・#2390（外部Host Consumer）・#943（実Build） | 既存見積もり23ベクトルと独立Expectedを再利用し、同一契約に対する実成果物、byte layout、import、Host別bindingを記録。実行可能な二Hostで失敗／拒否も比較し、Contract一致でもABI誤配置、異Host import不足、現在Grant不足を検出。現時点で実Componentによる二Host実行と正式WIT採択は NOT RUN |
| **TR-11-STATE：短い成功を実サービス全体の成功としない** | #2514／#1180（テスト対象）・#1129（Lifecycle）・#1174（実Effect）・#1824（Studio） | 再実行／通信断／重複要求／部分確定／後着UI応答／新しい独立入力を共通Attempt・Source・対象に束縛。変更後のexact artifactで再走し、Process exit、表示、参照Executor、Commit、別Readerの観測を分ける。短い合成実行のPASSから本番の完遂を推論しない |
| **TR-11-GRAIN：データ変換の完全性と粒度の検証** | #2622（Data Operations）・#2506（明示Data Context）・#2514（独立Expected） | 同一意味のRDB実現とAPI／導出候補を、時間範囲、母集団、業務Key、重複、NULL／未取得、順序、集約Grainで照合。数値一致でも取りこぼした行がある例と、partial取得をcompleteと誤表示する例を含める。Model Relation存在だけでは取得範囲を広げない |
| **TR-11-ACTION：AIの操作順序と実行権限の間を検査** | #2598（Offer→Requestの実Target）・#1174（Host権限／Effect）・#1824（UI操作） | 会話上の yes が別Requestや別Tenantに流用される例、現在Grant撤回、Target alias変更、操作を省略した場合を反例化。Agent側の型付き状態は助言・診断に使えても、実Provider／外部Hostが現時点の権限・対象・Effect scopeを強制する |
| **TR-11-REUSE：検証済み局所進捗だけを再利用** | #1824（Studio）・#2509（AI／CLI境界）・#2600（再資格化） | 再利用単位にSource/Revision、入力・前提、操作、既知Expected、実際の観測、対象Scopeを関連付ける。前回の部分成功は次回の実行可能性でもOwner受理でもなく、Source drift／Grant drift／出力先変更なら再資格化する。Agent専用永続状態正本を設けない |

### 10.3 順序・判定・停止線

1. **P0：既存定義とFixtureだけで反証可能性を上げる。** TR-11-SPEC、GRAIN、ACTIONについてread-only Source位置と独立Expectedの有無を調査。既存の#2514 mutant、#2622 DO-01〜08、#2598／#1174の既存負例を再利用し、新規試験名や第二Runnerをむやみに増やさない。
2. **P1：実際に消費された対象と成果物を固定する。** Source宣言→解決→実読込→Build選択→Artifact→Consumer／Request→実Effectのうち、各Claimに必要な部分だけを明示。失敗・未実行・unsupportedを別状態として残し、validate成功を実消費成功と取り違えない。
3. **P2：HostとProviderの実差を測る。** TR-11-ABIは既存#1861の実Component一件を優先し、同一契約を異なる二Hostで比較。TR-11-STATE／GRAINは実DB/API／独立Readerが用意できるConsumerに限って検証する。生成した合成Evidenceを実Host適合へ昇格しない。
4. **P3：採否は費用と失敗検出力で判断する。** Verification Agentを常設すべきか、WIT/Componentを公開ABIにすべきか、履歴再利用を製品機能にすべきかは、追加する抽象化・ツールチェーン・遅延・保守費用に対する反例の検出実績と具体Consumerの必要性で比較する。

**判断境界：** 文献で裏付けられた観点／TEIへ類推した設計仮説／Issueへの受入候補／実際に検証した結果／Ownerによる業務意味受理は互いに別。既存の#1119 Critical PathとA0〜A4、現行Authority、およびmainへの取込み条件を本節だけで改定しない。今回の追記は文書差分であり、上記TR-11-*の実行ログや正式採用は存在しない。

### 10.4 Wailsの版更新（旧記述の時点性を保持）

[Wails v3.1.0公式リリース](https://wails.io/blog/wails-v3-1-released/)は**2026-11-01**公開。前節9.3の「10月時点でもBeta」は当時の記述であり、11月6日時点の現状を表さない。#1824では3.1.0の正確な版を固定した比較候補とし、CGO不要経路は実験的機能として別に評価する。Windows／macOS／Linux、native依存、Web/Desktop接続、ビルドと再現性、現行Studioとの差分を実測する前に標準更新やアプリ生成Targetの採択を宣言しない。


### 10.5 他チャットと最新Issueを踏まえた精読差分（11月6日・追補）

**位置付け：外部研究の実証、TEIでの設計仮説、実装済み証拠、Ownerの判断は互いに別。以下の追加Fixture／比較実験は NOT RUN。** §10.1〜10.4を繰り返すのではなく、既存Ownerの受入をより判別可能にする。

- **M4Bench・対象集合**：[原文](https://arxiv.org/html/2610.37459v1)は、8タスクの制御比較でoperational specだけでも報酬が+0.162改善する一方、全体のSkill効果では行Key F1が+0.003にとどまると報告する。異なる指標・母集団の数値を直接比較しない。全チェックのstrict passはwith-skill 25/280、no-skill 13/280。**計算値の一致・対象Key集合の正確さ・実取得範囲・現在資格を別のOracleにする**。#2506/#1706/#2622/#2514へ接続。
- **Concordat・同意の束縛**：[原文](https://arxiv.org/html/2610.37461v2)の二状態プロトコルは「yes/yep/yeah」を同意として扱い、著者自身が「yes, but not that one」も通り得ることを限界と明記。**提示されたToolの許可・ユーザーが確認した正確なOperation/Target/Version・Hostの現在Grant/Effect Fenceを分離する**。単なる「yes」や同じPlugin所属を実操作対象の許可にしない。#2598/#1174/#1824/#2389へ接続。
- **Verification Agents・提案と実測の区別**：[原文](https://arxiv.org/html/2611.02285v1)のSpec-first、vacuity witness、mutation、semantic mergeは有用。ただしVIR／Shared Synthesis Fleetの例示的PASSや将来構成を、実製品での全面実証と読み替えない。#2514の#2901対照→#3017のrelation-unique一行Mutant killは、合成Fixture・特定Evaluatorで独立に確認できた**限定実測**である。次の反証は空集合、必要Source未読込、NULL/unknown、scope不完全に焦点を当てる。
- **Rethinking Verification・別Reviewerの費用対効果**：[原文](https://arxiv.org/html/2610.34577v1)では独立Criticが、実験された完了ゲートより良い精度／追加費用のバランスを示した。**別AI Reviewerが賛成したことは独立Expectedではない**。#2514/#1180/#2667で既知Mutantの検出・非検出、誤報、追加トークン／時間を並記し、常設の複数Critic導入は実測後に判断する。
- **ServerBench・状態依存と実経路**：[原文](https://arxiv.org/html/2611.06841v1)はsession依存と並列branchを保持するstateful replayを使う。TEIは#1163の**同一Attempt**でSource、実Build、Guard、Request、Provider、Receipt、独立Read、Consumerを必要範囲で結合する。別PRの限定PASSを足し合わせて一つの成功した業務実行としない。LLM serving固有の性能指標は流用しない。
- **Task Advancement Memory・再利用時点**：[原文](https://arxiv.org/abs/2610.37482)の局所検証済み作業進捗は#1824/#2600においてSource Revision、選択epoch、Provider、Grantと再照合してから使用。過去のGUI操作／Agent記憶を現在の承認・実効果に昇格しない。
- **Wasm ABI・不要なPlugin化を避ける**：[原文](https://arxiv.org/abs/2611.00891)のABI/shimの実証は限定されたOS/Runtimeに対するもの。#1861のWIT候補をTEI契約の正本とせず、#2390の外部Consumerと#1251のBuild実消費を対照する。[Draft #3021](https://github.com/hat47x/tei-platform/pull/3021)の「Pluginなし直接Go計算の23ベクトル」はD0対照候補であり、実Plugin、二Host、現在Grantの証拠ではない。D0で要求を満たせるならPluginを追加しないことも正常な結論。

### 10.6 共通Business Caseの反証表：単一縦切り、独立Expected、Claimの上限

先行Consumerは order_entry の受注参照→確認→保存Intent→単一Providerの効果観測とする。#1125のBusiness ProcedureとWorkflow Instance、#2389のMembership/Occupancy／業務責任とHost Grantを**データ関係だけから推論しない**。read-onlyでSource→Function Data Context→実Read→Consumerを資格化してから書込みへ進む。全試験を一度に作るのではなく既存FixtureとEvidenceを再利用する。

| 対照（固定する独立Expectedの候補） | 検出する誤成功 | Owner／まだ証明していないこと |
| --- | --- | --- |
| **E1 集合・粒度**：親H、子D1=2件、D2=3件。JOIN 6行は各Entityの件数ではない | 集計値一致だけで対象行・関係が正しいと判断、JOIN増幅 | #2506 FC-01/#2622 DO-01/#2387。実Readerの母集団まで未検証 |
| **E2 取得資格**：D2先頭1件のみ、完全な0件、未取得、NULL、非開示、別Snapshotを区別 | 0件と未知・部分集合を混同し、全件／不存在／業務完了を推論 | #1706/#2622/#2516。Provider取得範囲とConsumer開示は別 |
| **E3 判別力**：同一Scenario/Source/Expectedでrelation-unique一行Mutantを注入 | 別CI失敗をMutant kill扱い、弱いOracleを正しいと推論 | #2514の#3017に限定実績。NULL/unknownと不完全scopeは別の未実施試験 |
| **E4 確認・権限**：受注Aの更新を確認して実PayloadをBへ差替え、Tenant/Grant失効 | 会話上のyes・画面Offer・組織所属だけで実Commit許可 | #2598/#1174/#2389。対象と現在Grantの実束縛は未完 |
| **E5 Commit不明**：ACK喪失後同OperationIDのReceiptと別接続Readを照合し、新Key盲再送を拒否 | timeout＝未Commit、既存Receipt＝最新Consumer成功とみなす | #1174/#2564。main #2853の実PostgreSQL限定PASSと同一Source/GUI E2Eは別 |
| **E6 三つの版ずれ**：Source A→B→A、DB実Value A→B→A、Workspace選択A→B後の遅延A応答 | 同bytes／同値／旧Resultを現在の実行・表示資格へ復活 | #2600/#1163/#1824。各Revision/Version/epochを独立に検証 |
| **E7 Plugin粒度**：直接Go／Plugin／Wasm候補を同じ23見積もりベクトルで比較 | 未選択Pluginの混入・欠けたnested module・型だけ合致 | #3021/#1251 PB-01/07/10/#1861。二Host ABI・実Buildは未資格 |

この表は新しいScenario/Conformance正本ではなく、既存#1180/#2514/#1163へ**どのRequired Evidenceが揃ったか**を記録する比較用資料。照合を次の4つの観測に分ける。

1. **S 静的事実**：DDL/SQL等で宣言されたPK・FK・CHECK、Parser/Selector/Bindingの構造。PK列の宣言順は表示SortKeyではない。FK宣言は実データ整合や業務Relation受理を証明しない。
2. **M 意味候補**：Constraint、関係、状態遷移、Business ProcedureとActor、受理すべき意味。AIや構造一致はOwner意味受理の代わりではない。
3. **E 実行観測**：実際のSource読込、Build入力、Provider Read、Commit/Receipt、別Reader、現在Consumer。main #2853の実PG部分検証、#2747のReferenceExecutor等は個別実測であり、同一Attemptの製品完了とは限らない。
4. **A Owner判断**：意味・対象・業務完了・採用の明示判断。S/M/Eのどの部分からも自動昇格させない。

### 10.7 研究知見からの比較実験と採否の条件

**M4Benchに対応する最小A/B/C比較**：同一の業務要求・実Source・固定Expectedを与えて、A=Task＋Schemaのみ、B=十分なoperational spec（Markdown表、Rule、粒度、NULL、対象集合、時点）、C=Bと同じ内容をSkill/Plugin配送形式で与える。別途D=必要な場合だけ既存Native SQL/Generatorを入力に加える。Dは情報量が異なるためBとCの「包装の効果」とは混同しない。

**測定**：値一致率、Key集合のprecision/recall、全必須Obligationのstrict pass、既知Mutant検出、未実施/unknownを正しく保留した件数、Source/実Build/Consumer対応、レビューの追加時間/トークン、保守・変更量。ExpectedをAgentの出力SQLやCandidateと同じ生成経路で作らない。AI Reviewerの同意と独立Oracleの正しさは別。

**実施順・採否**：P0=E1/E2/E3のread-only・既知Mutant再利用、P1=E4/E5/E6の現在Grant・Receipt・後着、P2=E7の実Host/Plugin比較。#1063/#988がDSL・変換、#2514がOracle、#1180がScenario、#1163が同一Attempt、#1174が実Effect、#1824がUIを所有する。新Verification IR、Agent Runtime、Memory Registry、Plugin Loader、Auth Engine、Proof Storeを研究だけで採用しない。

**証拠の上限**：論文に書かれた効果 ≠ TEIでの実証。Issueへの受入案 ≠ 実装。単一Mutant kill ≠ 全Domain Oracle適格。実PG Commit/Receipt部分PASS ≠ 同一業務UI/Host/Consumer成功。Draft PR #3022は文書への計画反映であり、テスト実行・Owner受理・main統合ではない。研究計画が#1119の既存Critical Pathや主担当の作業ブランチを変更しない。


### 10.8 追加精読：存在・不在の証拠と検証の有効性

2026年11月6日。§10.5〜10.7と現行 #2514/#1163/#2622/#2600/#2598/#1174/#1824/#2389/#1180 を再照合した。**以下は研究で直接測定した事実と、その事実からTEIへ導く設計・受入候補を区別する。新たなCore、形式検証器、Evidence Registry、Agent専用認可、TEI Hostを提案しない。** 各課題の具体的な試験実装は既存Ownerが担い、この文書が第二Scenario/Oracle正本にならないようにする。

#### 10.8.1 観測の非対称性：positive witness と closed-world absence

[M4Bench arXiv:2610.37459v1](https://arxiv.org/html/2610.37459v1) は、operational specificationが同論文の8タスク比較でvalue-column agreementを改善しても、28タスク全体のkey F1差が+0.003にとどまり、strict passはwith-skill 25/280、no-skill 13/280だったと報告する。これらの評価条件・指標・対象タスクは一致しないため、数値を直接比較・TEI性能へ転用しない。著者らも採択部分集合には目的選定の偏りがあると明記している。

**TEIで追加する判定**：単一の正しいEntity観測は、要求した特定Entityが少なくとも一つ存在したという限定Claimを支持し得る。一方「対象Entityは他に存在しない」「更新対象なし」「未取得の関係なし」「影響を受ける成果物なし」は**否定・全称Claim**であり、対象母集団、現在Scope・Filter、Providerの取得完了、Snapshot／Revision、非公開の影響を考慮しない限り支持されない。権限により見えない集合については「開示可能範囲で0件」と「世界全体に存在しない」を区別。RLS/NULL/empty/page/partial/unavailable/permission-deniedの診断に、秘密の件数やフィールド内容を露出させない。

既存対応：#2506 FC-01/06/11（Function Contextと集合）、#1706（集合完全性）、#2622 DO-01/07・LUX-05（多Provider）、#2387（Query/SQL静的Explain）、#2516（Disclosure）、#2514（Oracle）。SQLのFK・CHECKが静的に宣言されていても実全行整合・履歴の正当性や業務Owner受理を推論しない。

#### 10.8.2 過去のPASSの有効性：実施済みと現在適用可能を分離

[ServerBench arXiv:2611.06841v1 §10.3](https://arxiv.org/html/2611.06841v1) は、検証済み内容が後続の関連変更後にも適用可能か、submitted treeで検証したか、過去にしか実施していないか、必要なcoverageが無いかを区別する。これはそのベンチマークの観測手法であり、TEIの安全性を既に証明するものではない。

**TEIで追加する判定**：PASSを単なる時刻付きBooleanにしない。既存Evidence上で Source/Role/Query/Operator/Generator/Toolchain/Build closure/Host/Provider/Runtime configuration/Consumer のうち **ClaimのRequired Dependencies**を特定し、検証された実際の入力と、現在の選択・実消費を比較する。
- 一致するが今の実行Authority／Disclosureが違う：Artifactの再利用はあり得ても現在の実行資格・出力開示は再評価。
- 同一bytesへSourceがA→B→Aと戻った：現Revisionと受理判断、途中の失効・依存変更を別評価。
- 未使用Pluginの追加：実Selectionと物理Build closureが不変なら無関係なArtifactまで一律失効させない。
- 参照されたSource／SQL assetが実際は未取得・未読込：キャッシュHIT、旧Receipt、validate PASSだけで現在の検証済みへ昇格させない。
- 観測済みConsumerの旧画面応答：過去Attemptの実Effectは保存しても、現Workspace/Selection epochの画面表示資格は別判断。

既存対応：#2600（再利用/否定依存/失効）、#1163（同一Source/Target/Attempt Evidence）、#1180（必要Obligation）、#1824（後着UI）、#1251（実Bundle閉包）。新たなProof StoreやValidatorを設けない。

#### 10.8.3 正しい結果でも不正な手順は見逃される

[Concordat arXiv:2610.37461v2 §5.3](https://arxiv.org/html/2610.37461v2) は、会話上の単純な yes 判定と違反検出が同一定義を共有しており、ゼロ違反もその定義に依存すること、yes but not that one でも通ってしまうことを限界として明記している。したがって「型付きTool集合を適用すればユーザー同意を保証する」という一般化は避ける。

**TEIで追加する判定**：Outcome correctness、Business Procedure/Workflowの順序・責任、明示Confirmationの対象Version、Current Host Grant、Provider Effectの正当性を独立に判定。二法人兼務者が一方のTenantで承認後に別Tenantへ切替、代理権失効後に旧Confirmを再使用、対象Aの確認からPayload対象Bへ差替え、古いUI Promiseが新Workspaceに届く反例を既存#1125/#2389/#2598/#1174/#1824へ割り当てる。単なる会話履歴・Agent-visible Tools・Membership・成功したDB応答を、正当な業務承認の代用にしない。新たな共通Workflow Engineや認可Engineの導入は要らない。

### 10.9 Existing Scenarioを流用する追加の独立反証（提案／NOT RUN）

| ID／Expectedの固定 | PASSと誤認させる条件 | Required Evidence／担当 |
| --- | --- | --- |
| RES-01 Positive witness | Parent Hの子を1行実取得し「Hの全子が1行」と報告 | その1行のSource/identityは正でもcompleteとは不明。#2506/#1706 |
| RES-02 False absence | 取得先頭ページ0件・RLS範囲0件・API partialを「全件0」「変更不要」とする | 取得母集団、Filter、Scope、完了、時点の証拠が欠ければunknown。秘匿件数を返さない。#2622/#2516 |
| RES-03 Unconsumed test | SourceのRule一覧は完全だがLoaderが一部を読まず、実行した検査だけPASS | Required Obligationの母集団、実Source/Build消費、判別力のある独立Expected。#1180/#2514/#1163 |
| RES-04 Post-change PASS | 旧Provider向け試験がPASS後、同Query IDのDB別SQL／Generator／Grantを変更 | 旧PASSは歴史的証拠として保持、現在適用資格はunknown/再試験。#2600/#2862/#951 |
| RES-05 Outcome-protocol collision | 受注Bの保存が正しい値で成功、しかし確認したのは受注Aまたは別Tenant | Outcome PASSでも操作の対象・承認・権限はFAIL/unknown。#2598/#1174/#2389 |
| RES-06 Shared-origin Oracle | 実装とExpectedを同じ生成AI出力から構成し、同じ誤解に一致してPASS | Source・Oracle provenance、独立Property、既知Mutantの検出力を別観測。#2514/#1180 |
| RES-07 Late UI receipt | 実Providerの過去Receiptは正しいが、新Workspaceで古い応答を表示 | 実Effectの歴史的成功と現ConsumerのSelection epoch・Disclosureを分ける。#1824/#2598/#2516 |

**必要Obligationの分母**：実行した試験件数や見つかったSource数を分母としない。Claimに必要なObligation集合を先に固定する。未取得・未読込・未評価・対象不明はunknown/not-runとして残し、理由のあるnot-applicable以外は分母から勝手に除去しない。Coverage（量）、Oracle discrimination（既知誤実装を検出できるか）、Evidence applicability（対象Revision/Attemptへの効力）を独立した列にする。必須の一項目がmissingなら全体Claimの合格を主張しない。実際の四層分類S/M/E/Aは§10.6を再利用し、別の正式スキーマは作らない。

### 10.10 仕様をSkillで届けたか、実際に読ませたかを分離

§10.7のA/B/C/D比較の追加統制。M4Bench [Table 1、§5](https://arxiv.org/html/2610.37459v1) はtask instruction+schema、complete operational spec、targeted skill、decoy skill、raw reference SQLを別条件として比較する。これをTEIに転用する際、以下は異なる条件とみなす。

1. **A**：Task＋Schemaのみ。固定Expectedと評価用Sourceは他条件と同じ。
2. **B**：同じTask＋Schemaに完全な業務操作仕様（Rule、grain、time window、NULL、母集団）を**直接表示**。
3. **C1**：Bと同じ内容をSkill/Plugin経路で**発見可能にしただけ**。実Read未確認の試行をBとの同条件比較に含めない。
4. **C2**：Skill経路で実際に選択・取得・読込した内容とVersion/bytes/長さを確認できた試行。Bとの比較時は同じ仕様内容であることを固定し、配送差、発見可否、消費量、トークン費用と成果を別指標で報告。
5. **対照**：別タスクのDecoy/無関係Skill、またはSource/Toolchain条件が異なるRaw SQLは、仕様内容の効果と包装の効果の推定に混ぜない。

**不採用条件**：包んだSkillを増やしただけで実Read/正しいSource消費・Key集合・strict passが改善せず、コストと第二正本が増えるなら、構造的に明瞭なMarkdown/表・直接仕様供給を優先。逆に必要な仕様の発見と実消費が改善したなら、既存#1063/#988/#2509/#2514/#1180のAuthoring/CLI/Conformance内で適用を検討。論文の測定値をTEIの期待改善率や固定KPIにはしない。

**今回の変更状態**：文献再読・現行Issue照合によるドキュメントDraft更新のみ。RES-01〜07、A/B/C1/C2の比較、新たな同一Attempt製品E2E、二Host/二Provider実測、Ownerの採用・意味受理は NOT RUN。#3017の限定Mutant kill、#2853の実PG限定観測、#2747の参照実行を合算して製品資格にしない。PR #3022はreview用Draftを維持し、main統合／Gate合格を主張しない。


### 10.11 実装済みCompletionClaimへの研究知見接続（2026-11-06：現行mainの静的Source確認、実試験は未実施）

[M4Bench arXiv:2610.37459v1](https://arxiv.org/html/2610.37459v1) の行集合誤り、[Verification Agents arXiv:2611.02285v1](https://arxiv.org/html/2611.02285v1) のvacuityとwitnessの問題を、[main内のcompletion.go](https://github.com/hat47x/tei-platform/blob/main/internal/semantic/completion.go) と [#2926](https://github.com/hat47x/tei-platform/pull/2926)（統合済み）、[空集合のOwner判断](https://github.com/hat47x/tei-platform/blob/main/plan/research/EMPTY_SET_COMPLETION_SEMANTICS_2026-11.md) へ直接接続する。**新Completion Engine／Proof IRは不要**。

**現行Sourceの限定事実**：EvaluateCompletionは観測TargetResult、呼出側指定Coverage（complete/partial/unestablished）、EmptyJustified、HandoffConfirmed、Conflictingを入力し、holds/fails/unverified/contradictionを返す。Ownerが定義するL（対象集合）の実Provider取得、範囲、時点や行帰属の証明はこの関数では行わない。CoverageCompleteやEmptyJustifiedを引数に設定できること自体は、当該範囲の全件を観測した証明ではない。

以下は**コード分岐の静的読解から導いた判別Fixture候補**であり、新たなGoテストやOwner意味受理は未実施。現行動作を挙げた行も「既に不具合確定」とはしない。

| ID | 一つだけ変える入力・状況 | 現行の分岐／独立Expectedで検討すること | Owner |
| --- | --- | --- | --- |
| CQ-01 | 完全0件と、ページ0件・RLS制限の0件・Replica遅延を対照 | 誤ってCoverageCompleteを供給すると空のallow条件がholdsとなり得る。対象L、Filter、Scope、Provider、時点とReader完了を独立に確認する | #2506/#2622/#1706 |
| CQ-02 | Targetsが空、CoverageUnestablished、EmptyPolicyDeny | 現行コードの空分岐はCoverageを確認せずfailsを返す。**実Lが空と判明していない状況でも確定的失敗と言ってよいか**をOwnerに照会。実行を許可しないことと、業務の不成立を証明することは別 | #267/#2514 |
| CQ-03 | Handoff類型、Targets空、EmptyJustified真、CoverageComplete、HandoffConfirmed偽 | 現行コードは空のrequires-evidence分岐でholdsへ進み、後段のHandoffConfirmed判定を通らない。**対象0件で手作業確認は非適用なのか、それでも必要なのか**を業務手順のOwnerが決める | #1125/#267 |
| CQ-04 | existence類型で、観測1件Holds真だがCoveragePartial、または観測1件Holds偽でCoveragePartial | 現行非空分岐は全対象成立型。前者unverified、後者failsとなる。**存在条件の個別行Pと完了述語全体を単純に同一視しない**。実Pが存在命題そのものなら入力Mappingの正当性を説明する | #267/#2514 |
| CQ-05 | 別Tenant/時点/部分Readerに由来するEmptyJustified真、または矛盾するがConflicting偽 | Bool引数だけで実Source同一性や実矛盾集合は検査されない。Evidenceの発行者・Reader資格・同一Scope/Version/Principalを同一Attemptで追跡する | #1163/#2389/#2516 |
| CQ-06 | 組織Membershipが遡及訂正され、同じ行値でも有効期間と観測時点が変わる | 旧CoverageCompleteと旧holdsを新しいL/現在Grantへ転用しない。業務時間と実Host認可時点を独立に管理する | #2389/#2600/#1174 |

**最小検証順**：まずCQ-01〜05を既存#2926試験と#2514独立Expectedに対して同一の固定Fixtureで確認し、現実装とOwner期待の差を分類。その次に#2506のFunction Data Context→実Provider Read→Rの被覆証拠→Completion評価→#1163の同一Attempt/Consumerを連結。CQ-06は組織レーンの既存Fixtureを利用し、必要な書込み試行にだけ#2598/#1174の現在Grant/Effect/Receiptを接続する。CompletionHoldsをProvider Commit済み・正当な承認・業務全体の完了と同一視しない。

### 10.12 Meaning受理と実Grantの2×2：研究計画への接合だけ行う

[Draft #3026](https://github.com/hat47x/tei-platform/pull/3026) は、M＝Owner Accepted Meaning、G＝Current Host Grantの直交する2×2比較と、受注／組織／Workflowの同形異義を既に記述している。**その比較表を本計画へ複製せず、各セルの独立Expected・実証範囲を#1180/#2514/#1163と接続することだけを研究レーンの課題とする**。

M未受理・G許可でもMeaningを採用しない。M受理済み・G拒否なら旧Meaningは履歴として保持し、現在Dispatchは拒否。両方満たしてもTarget・Version・Guard集合・Provider Commit Fenceと現在Consumerが未確認なら業務完了ではない。未受理・拒否の二不足は権限ある診断では別々に保持するが、低権限利用者へ非公開情報を露出させない。Concordatの表示可能Tool集合とHostの実Resource Grantを同一視しない（[論文 §5.3](https://arxiv.org/html/2610.37461v2)）。Draft #3026は未統合で、正式意味受理ではない。

### 10.13 Skillの「割付」と「実提供」を区別する評価方式

§10.7/10.10のA/B/C1/C2について、[TEI #2018](https://github.com/hat47x/tei-platform/issues/2018) および [TAI #168](https://github.com/hat47x/tai-pleroma/issues/168) の既存H2/H3評価計画を再利用する。

- **割付ベース（ITT型・記述集計）**：Skillを利用する予定だった試行が未発見・未ロード・Toolなし・実送出不可でも、分母に含めて障害と費用を測る。C1でSkillを発見可能にしただけでは利用成功としない。
- **実提供条件ベース（PP型・限定比較）**：同一Task、Source raw SHA、Oracle、モデル、Tool権限、提出方式を固定し、H3のRoot＋Required Referencesが実ロードされ、両腕のTool呼出しが実証されたものだけを比較。C2の実消費内容・版も記録する。配送の成功率と、正しく届けた仕様の効果を混同しない。
- **Claimの上限**：H2/H3は既存の固定Task 1件・4独立Sessionが計画分母。未実行であり、仮に4 Sessionが成功しても一般的な因果効果やp95は主張しない。回答品質、Key集合、strict pass、false-qualified、unknown保持、トークン／Tool／時間／人的費用を別記録。

新しいBenchmark RuntimeやSkill評価正本は不要。#2018が経済性、#1180がRequired Obligation、#2514が独立Oracle、#2509がAI/CLI境界を担当する。M4Benchの測定効果量をTEI/TAIの期待改善率へ転用しない。

**今回の追補状態**：研究と既存main Source/Issue/Draft照合によるDocs-only反証設計。CQ-01〜06、Provider→Completion評価の実接続、H2/H3四Session、Owner正式受理はNOT RUN。既存の限定Go test・PG試験を単一の製品E2Eと合算せず、#1119のCritical Path／現在のGate／既存Host/Plugin方針を維持する。


### 10.14 11月6日Owner方針：TEI Authoringを *.tei.md へ統一する

**優先関係**：11月6日のOwnerの明示判断は、従来の .tei を原則廃止してTEI対象文書を *.tei.md に寄せるというもの。現行[#2621](https://github.com/hat47x/tei-platform/issues/2621)末尾の「.tei.mdを標準化しない／.md＋OKFと独立 .tei を比較する」は**その前に記録された設計候補**であり、以後の移行先を拘束しない。一方、この判断だけから現CLIのParser/Loader/Generator/Studio/SHOが新拡張子に対応済みとも、既存Sourceを機械的に安全移行できるとも推測しない。関連する旧Issueや [.tei向けEmptyPolicy文書](https://github.com/hat47x/tei-platform/blob/main/plan/research/EMPTY_SET_COMPLETION_SEMANTICS_2026-11.md) の記録は当時の実装事実として残し、移行完了時に正しく更新する。

**分ける三つの契約**：
1. **物理Authoring意図**：末尾が *.tei.md なら「TEIで扱うことを意図したMarkdown」と機械判別しやすい。通常 *.md は一般の文章として残す。拡張子は編集対象の発見・選択の手掛かりであり、正式なRole、Language Plugin、Accepted Meaning、実行Grantの資格ではない。
2. **文書内容と解釈**：元のMarkdown本文・表・引用・図・コードフェンス・YAML frontmatterを基本的に保持し、選ばれたRole/Profile/Source rangeを実Parserが読んだかを独立に確認。OKF v0.2のConcept Fileは末尾 .md と必須typeを持ち、tagsは任意。*.tei.md はファイル名上の意図表示とOKFの任意メタデータを併用可能だが、*.tei.mdであるだけで「OKF検証済み」と呼ばない。独自TEIフィールド・Role構文は#2621/#2620/#988で検証するまで未固定。
3. **Meaning/Build/Effect資格**：宣言されたSource→Resolver/Language Plugin→Loader実読込→Parse/Role occurrence→Required依存照合→Owner受理→選択Build/実Consumer→現在Host Grant/Provider Effectは別。*.tei.mdへのRenameやタグ追加で一段も自動昇格させない。未読込Sourceを「Roleが存在しない」や「Ruleが0件なのでPASS」とみなさない（#2822/#2823/#2514/#1163）。

**移行の負例と独立Expected**（全て提案／NOT RUN）：

| ID | 入力／反例 | Expectedと担当 |
| --- | --- | --- |
| **MDT-01** | ordinary.md に prose・引用中のtei/Role語・Markdown表 | 明示SelectionなしにTEI実行対象へ昇格しない。通常編集の原文bytes保持。#2621/#1124 |
| **MDT-02** | schema.tei.md に明示Role/Frontmatter、別のREADME.mdと複数Code Fence | 指定されたoccurrenceだけ解釈、未選択範囲は未消費。frontmatterのtype/tagsだけでMeaning受理や実行許可を付けない。#2621/#988/#2620 |
| **MDT-03** | 旧 .tei が独自文法でMarkdown互換ではない | 単純RenameしてParsing成功と主張しない。必要なら明示的変換・Preview・変換損失表を提示し、原Source保持・相対参照再解決・独立ExpectedでMeaningを照合。#2621/#1920/#988 |
| **MDT-04** | パス拡張子が変わるがbyteが同じ、またはSource A→B→A | File locator/Source occurrence/依存版/Build選択・受理範囲を再確認。byte一致だけで旧Acceptance/Artifact/Executionを再資格化しない。#2600/#1163 |
| **MDT-05** | 非対応Plugin、未知Role、0件のTEI Source、文書中のRequired Rootが未読込 | 宣言Inventory・Loader Read・Parsed/Anchored・Required fulfilledを別記録し、空集合やUnknownを正常完了としない。#2822/#2823/#2620/#2514 |
| **MDT-06** | Editorのno-op、局所表セル編集、CRLF/BOM/UTF-8、外部競合 | 既存Source Fidelityを流用し、通常保存の非対象bytes保持、変更Preview、再読取/再Parse、競合保留を確認。全Markdown自動整形なし。#2621/#1124/#2497 |
| **MDT-07** | 独立Expected表と実装コードを同じ文書へ併記 | テストと生成物の配置が同じでもExpectedを実装由来で再生成せず、Source版・実消費・Mutant判別力を分ける。#1180/#2514 |
| **MDT-08** | .tei.md文書をSHOへ取り込み、同名異内容や移動後に再開 | Source Identity/Revision/actual realpathを保持し、別Sourceへ自動再結合・再受理しない。既存SHO/MMI C-01/T2を再利用。#2621とSHO担当 |

**作業順**：P0は#2621の移行先・拒否条件を明記し、現mainのSource発見・Loader・Parser・ConsumerのFit/Gapをread-onlyで測る。P1で旧 .tei と新 *.tei.md の同一公開Fixtureを選び、形式変換の必要性・再読取・Source差分・独立Expectedを確認。P2でCLI/Studio/SHO/生成Targetの実Reader資格、既存Build/Source選択への影響、旧Sourceの移行期限や互換性をOwnerが判断。新拡張子決定は採用、移行の全機械的実装・旧 .tei の具体的な廃止期限・Frontmatter標準Schemaは未採用と区別する。

### 10.15 証拠の自己申告を評価器の証明と取り違えない：Completion／Role／Skillの共通停止線

既存mainのCompletion評価（#2926）はCoverageCompleteやEmptyJustifiedを**呼出側入力として受け取る**。#2620/#2823のRole観測もLoaderが読んだこととRequired Rootの充足は別。#2018のSkill比較でも「選択可能」「Root読込」「必要Refs実読込」「Tool呼出し」の証拠は異なる。三つを束ねる最小不変条件は、**どのClaimにも必要な対象集合とRequired Obligationを評価対象そのものの自己申告だけから確定させない**こと。

| Claim | 自己申告や局所結果だけでは不足する点 | 独立に確認すべき境界／Owner |
| --- | --- | --- |
| 全対象のCompletion | Targets全件・CoverageComplete・EmptyJustifiedの値 | OwnerのLと実Read集合Rの同一Scope/時点/Principal、非開示対象、否定依存。#267/#2506/#1163 |
| Role/Sourceの完全性 | 読み込んだファイル内のRole一覧、Required=[]、validate成功 | 外部Required Rootと実Loader/Parser/Version・失効、閉包全体。#2822/#2823/#2620/#988 |
| Skillの有用性 | Manifest/Tool一覧・Skillに同じ仕様が保存済み | 同一Taskと独立Expectedを固定し、実Load/実Tool/Consumerが受け取ったbytes・不達率と品質を分離。#2018/#2509 |
| 本番Effectの完了 | Preview、承認文、Reference executorのcommitted、過去Receipt | 実Target/現在Grant/Provider Commit/Operation identity/別Reader/現在Consumer。#2598/#1174/#1824 |

**Claim別に必要なEvidenceを決める**ための共通点検票であり、新しい横断Proof Engine／Registry／Test Runnerは作らない。OwnerのSource/Meaning受理とHostの現在Grantは常に独立。追加MDT-01〜08／各境界の同一Attempt実証はNOT RUN。今回のResearch Draftは計画だけで、既存の実PG局所PASS・局所Mutation kill・Parser局所CIを製品E2Eへ合算しない。


### 10.16 一時拒否ガードの実測と、TEI Markdownを本当に解釈する次のGate（2026-11-06、他Lane最新実装との再整合）

**実証済みの進展とClaim ceiling**：[PR #3031](https://github.com/hat47x/tei-platform/pull/3031)（Draft、exact HEAD 63953ccb5c12ead26ec1268e7c0afdba4eff66ae）は、現Loaderで未対応の *.tei.md がmanaged sourceに現れる場合に「黙って読まずvalidate成功」とする代わりに TEI_MD_PARSER_UNBOUND で停止する暫定処理を実装。直接確認した[専用CI #42217379093](https://github.com/hat47x/tei-platform/actions/runs/42217379093)は completed/success。該当HEADに対するRepository Health・Workspace/Studio・Wails smoke・self-build・vulnerability等のWorkflowも completed/success。**これは特定HEADでのfail-closedガードと既存回帰の証拠であって、新 *.tei.md のParser適合／正式Source読込／Meaning受理の証拠ではない**。PRはmain未統合。

**移行計画の正本**：[Draft #3030](https://github.com/hat47x/tei-platform/pull/3030) の M0〜M6／TM-01〜14 を引き継ぐ。研究Draft #3022のMDT-01〜08は「研究から導いた判別観点」として残し、別の移行正本・第二Fixture実装を作らない。すでに段階設計と具体的な負例が#3030/#3031/#2621/#2823にあるため、追加すべきなのは**どの実行結果から何を言ってよいか**の資格境界である。

| 段階 | 最低限の観測 | 現状と不足する資格 |
| --- | --- | --- |
| A0 非対応の認識 | 指定sourceをmanaged inventoryで捕捉し、Parser未選択なら診断＋停止。旧 .tei と普通の .md の非干渉 | PR #3031の限定Go/Workflow PASS。ただし別root・Manifest未登録・Role Required Root欠落を全件捕捉したとは言わない |
| A1 実Parser消費 | Authoring原bytes/複合拡張子認識→selected/pinned Language→Markdown中の明示定義領域→元Span・Parser Role occurrence | 未実装／NOT RUN。タグや拡張子だけでLanguageを自己選択・自己受理させない（#988 Bootstrap K0） |
| A2 意味・移行同等性 | legacy .tei と *.tei.md の対応するSubject/Relation/Rule、NULL、粒度、明示Expected、消失項目、重複Subjectを比較 | 未実証。旧Sourceを勝手に削除せず、固定GateとAccepted Meaningの所在を独立確認（#2621/#1920/#2514） |
| A3 実消費と集合同等性 | 実Source→Function/Query→実Provider Readの集合・Scope・時点・Completeness→独立Consumer | 未実証。Parsed/ValidだけでTargetSet L、CoverageComplete、業務不存在を主張しない（#2506/#2586/#1163/#2926） |
| A4 効果と現在権限 | 実Request/OperationID、Current Grant/Host Effect、Receipt/独立Read、Consumer開示・Owner受理 | 未実証。Reader/Parser移行成功を現在GrantやBusinessVerifiedへ昇格させない（#2598/#1174/#2516） |

**優先すべき差分反証**（既存TM/CQ/RES/Issueを参照、追加Runnerなし）：
1. **移行による空虚な完了**：legacy .tei で定義したTargetSet Lの一部が *.tei.md へ移ったが、Loader未消費またはRole/Query未束縛のまま。観測集合R=空／部分で旧Completion/Ruleがholdsとなる誤成功を止める。まずA0で明示拒否、A1後もL定義・Required Source閉包・実Provider Scopeを独立に検査。#2514/#2506/#2926。
2. **二つのSource正本と誤Fallback**：旧 .tei と新 *.tei.md が同じSubjectを示す、または新Parser/Query Assetが未解決。旧Sourceへ黙ってfallbackして「新形式の受入PASS」としない。対象の現Selection・実Reader bytes・同じExpectedとの照合で意味・技術失敗を分離。#3030 TM-03/04/09、#2600。
3. **旧成功の過剰失効／誤復活**：未選択の新 *.tei.md Draft追加だけで無関係な旧Artifactまで全面失効させない。他方、実Selection/Build inputが旧→新へ切り替わった時は同hashでもSource occurrence/Role/Operator版/実Consumerの資格を再評価。#2600／#1163。
4. **学習済み手順の自己証明**：Source内にAIが生成したRule、Expected、Skill説明が共存しても、Required Oblicationの母集団を候補の自己申告や「新拡張子で読めた件数」から決めない。独立ExpectedとKnown Mutant、実Loader/Parser/Tool受領を区別。#1180/#2514/#2018。

**限定業務例の接合**：[PR #3024](https://github.com/hat47x/tei-platform/pull/3024)（exact HEAD 705933667d5b081036314a9161280323e08a64bd、[局所CI #42208159977](https://github.com/hat47x/tei-platform/actions/runs/42208159977)成功）は旧 .tei を使って CaseRecord Persistence→Query SQLのSource/Revisionを確認した**E1c相当の限定実測**であり、*.tei.md Migration/Function Data Context/実SQL Provider Readを証明しない。#3030のM4/M5以後に同じ公開Case・同一独立Expectedを新Sourceへ適用し、A2の同等性とA3の実Provider資格を別に合否判定。組織E2・G1/G2を研究移行のHard Dependencyで止めない。

**決定・実装・未実証の区別**：*.tei.mdへの移行方針はOwner決定済み。#3030はDesign Draft、#3031は検出拒否の実Go局所PASSを持つImplementation Draft、#3024は旧Sourceの局所Go/CI PASS。A1〜A4・新形式による実Provider/CLI/Studio/SHOの本格一巡、移行後Owner Meaning Acceptanceは未達。現行#1119 Gateは据え置き、研究PRの追記を製品Gate達成へ転用しない。
