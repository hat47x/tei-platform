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
| **TR-11-SPEC：仕様に沿った検証と空虚なPASSの排除** | #2514（独立検証基準／mutation）・#2619（定義と受理）・#1180（Scenario）・#2811（テスト記述） | 同じ受理候補に、満たす例、反例、曖昧／欠損Source、旧Revision、意図的なEvaluator mutantを用意。生成コードやAIの提案とは独立したExpectedで検査。検査対象Sourceが実際に読み込まれたか、条件が空集合で偶然真になっていないか、現ArtifactをConsumerが受領したかを別々に示す。#2514にある一つの局所Mutation killを全規則の検証済みへ一般化しない |
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

- **M4Bench・対象集合**：[原文](https://arxiv.org/html/2610.37459v1)は、8タスクの制御比較でoperational specだけでも報酬が+0.162改善する一方、全体のSkill効果では行Key F1が+0.003にとどまると報告する。異なる指標・母集団の数値を直接比較しない。全チェックのstrict passはwith-skill 25/280、no-skill 13/280。**計算値の一致・対象Key集合の正確さ・実取得範囲・現在資格を別の検証基準にする**。#2506/#1706/#2622/#2514へ接続。
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
| **E3 判別力**：同一Scenario/Source/Expectedでrelation-unique一行Mutantを注入 | 別CI失敗をMutant kill扱い、弱い検証基準を正しいと推論 | #2514の#3017に限定実績。NULL/unknownと不完全scopeは別の未実施試験 |
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

**測定**：値一致率、Key集合のprecision/recall、全必須Obligationのstrict pass、既知Mutant検出、未実施/unknownを正しく保留した件数、Source/実Build/Consumer対応、レビューの追加時間/トークン、保守・変更量。ExpectedをAgentの出力SQLやCandidateと同じ生成経路で作らない。AI Reviewerの同意と独立検証基準の正しさは別。

**実施順・採否**：P0=E1/E2/E3のread-only・既知Mutant再利用、P1=E4/E5/E6の現在Grant・Receipt・後着、P2=E7の実Host/Plugin比較。#1063/#988がDSL・変換、#2514が検証基準、#1180がScenario、#1163が同一Attempt、#1174が実Effect、#1824がUIを所有する。新Verification IR、Agent Runtime、Memory Registry、Plugin Loader、Auth Engine、Proof Storeを研究だけで採用しない。

**証拠の上限**：論文に書かれた効果 ≠ TEIでの実証。Issueへの受入案 ≠ 実装。単一Mutant kill ≠ 全Domain 検証基準適格。実PG Commit/Receipt部分PASS ≠ 同一業務UI/Host/Consumer成功。Draft PR #3022は文書への計画反映であり、テスト実行・Owner受理・main統合ではない。研究計画が#1119の既存Critical Pathや主担当の作業ブランチを変更しない。


### 10.8 追加精読：存在・不在の証拠と検証の有効性

2026年11月6日。§10.5〜10.7と現行 #2514/#1163/#2622/#2600/#2598/#1174/#1824/#2389/#1180 を再照合した。**以下は研究で直接測定した事実と、その事実からTEIへ導く設計・受入候補を区別する。新たなCore、形式検証器、Evidence Registry、Agent専用認可、TEI Hostを提案しない。** 各課題の具体的な試験実装は既存Ownerが担い、この文書が第二Scenario/検証基準正本にならないようにする。

#### 10.8.1 観測の非対称性：positive witness と closed-world absence

[M4Bench arXiv:2610.37459v1](https://arxiv.org/html/2610.37459v1) は、operational specificationが同論文の8タスク比較でvalue-column agreementを改善しても、28タスク全体のkey F1差が+0.003にとどまり、strict passはwith-skill 25/280、no-skill 13/280だったと報告する。これらの評価条件・指標・対象タスクは一致しないため、数値を直接比較・TEI性能へ転用しない。著者らも採択部分集合には目的選定の偏りがあると明記している。

**TEIで追加する判定**：単一の正しいEntity観測は、要求した特定Entityが少なくとも一つ存在したという限定Claimを支持し得る。一方「対象Entityは他に存在しない」「更新対象なし」「未取得の関係なし」「影響を受ける成果物なし」は**否定・全称Claim**であり、対象母集団、現在Scope・Filter、Providerの取得完了、Snapshot／Revision、非公開の影響を考慮しない限り支持されない。権限により見えない集合については「開示可能範囲で0件」と「世界全体に存在しない」を区別。RLS/NULL/empty/page/partial/unavailable/permission-deniedの診断に、秘密の件数やフィールド内容を露出させない。

既存対応：#2506 FC-01/06/11（Function Contextと集合）、#1706（集合完全性）、#2622 DO-01/07・LUX-05（多Provider）、#2387（Query/SQL静的Explain）、#2516（Disclosure）、#2514（検証基準）。SQLのFK・CHECKが静的に宣言されていても実全行整合・履歴の正当性や業務Owner受理を推論しない。

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
| RES-06 Shared-origin Expected | 実装とExpectedを同じ生成AI出力から構成し、同じ誤解に一致してPASS | Source・検証基準の出典、独立Property、既知Mutantの検出力を別観測。#2514/#1180 |
| RES-07 Late UI receipt | 実Providerの過去Receiptは正しいが、新Workspaceで古い応答を表示 | 実Effectの歴史的成功と現ConsumerのSelection epoch・Disclosureを分ける。#1824/#2598/#2516 |

**必要Obligationの分母**：実行した試験件数や見つかったSource数を分母としない。Claimに必要なObligation集合を先に固定する。未取得・未読込・未評価・対象不明はunknown/not-runとして残し、理由のあるnot-applicable以外は分母から勝手に除去しない。Coverage（量）、既知誤実装の検出力（既知誤実装を検出できるか）、Evidence applicability（対象Revision/Attemptへの効力）を独立した列にする。必須の一項目がmissingなら全体Claimの合格を主張しない。実際の四層分類S/M/E/Aは§10.6を再利用し、別の正式スキーマは作らない。

### 10.10 仕様をSkillで届けたか、実際に読ませたかを分離

§10.7のA/B/C/D比較の追加統制。M4Bench [Table 1、§5](https://arxiv.org/html/2610.37459v1) はtask instruction+schema、complete operational spec、targeted skill、decoy skill、raw reference SQLを別条件として比較する。これをTEIに転用する際、以下は異なる条件とみなす。

1. **A**：Task＋Schemaのみ。固定Expectedと評価用Sourceは他条件と同じ。
2. **B**：同じTask＋Schemaに完全な業務操作仕様（Rule、grain、time window、NULL、母集団）を**直接表示**。
3. **C1**：Bと同じ内容をSkill/Plugin経路で**発見可能にしただけ**。実Read未確認の試行をBとの同条件比較に含めない。
4. **C2**：Skill経路で実際に選択・取得・読込した内容とVersion/bytes/長さを確認できた試行。Bとの比較時は同じ仕様内容であることを固定し、配送差、発見可否、消費量、トークン費用と成果を別指標で報告。
5. **対照**：別タスクのDecoy/無関係Skill、またはSource/Toolchain条件が異なるRaw SQLは、仕様内容の効果と包装の効果の推定に混ぜない。

**不採用条件**：包んだSkillを増やしただけで実Read/正しいSource消費・Key集合・strict passが改善せず、コストと第二正本が増えるなら、構造的に明瞭なMarkdown/表・直接仕様供給を優先。逆に必要な仕様の発見と実消費が改善したなら、既存#1063/#988/#2509/#2514/#1180のAuthoring/CLI/Conformance内で適用を検討。論文の測定値をTEIの期待改善率や固定KPIにはしない。

**今回の変更状態**：文献再読・現行Issue照合によるドキュメントDraft更新のみ。RES-01〜07、A/B/C1/C2の比較、新たな同一Attempt製品E2E、二Host/二Provider実測、Ownerの採用・意味受理は NOT RUN。#3017の限定Mutant kill、#2853の実PG限定観測、#2747の参照実行を合算して製品資格にしない。PR #3022はreview用Draftを維持し、main統合／Gate合格を主張しない。


### 10.11 実装済みCompletionClaimへの研究知見接続（2026-11-06の検討、CQ-02は#3051で修正済み）

[M4Bench arXiv:2610.37459v1](https://arxiv.org/html/2610.37459v1) の行集合誤り、[Verification Agents arXiv:2611.02285v1](https://arxiv.org/html/2611.02285v1) のvacuityとwitnessの問題を、[main内のcompletion.go](https://github.com/hat47x/tei-platform/blob/main/internal/semantic/completion.go) と [#2926](https://github.com/hat47x/tei-platform/pull/2926)（統合済み）、[空集合のOwner判断](https://github.com/hat47x/tei-platform/blob/main/plan/research/EMPTY_SET_COMPLETION_SEMANTICS_2026-11.md) へ直接接続する。**新Completion Engine／Proof IRは不要**。

**現行Sourceの限定事実**：EvaluateCompletionは観測TargetResult、呼出側指定Coverage（complete/partial/unestablished）、EmptyJustified、HandoffConfirmed、Conflictingを入力し、holds/fails/unverified/contradictionを返す。Ownerが定義するL（対象集合）の実Provider取得、範囲、時点や行帰属の証明はこの関数では行わない。CoverageCompleteやEmptyJustifiedを引数に設定できること自体は、当該範囲の全件を観測した証明ではない。

以下は当時の静的分析で導いた判別Fixture候補。その後CQ-02は[#3051](https://github.com/hat47x/tei-platform/pull/3051)でmain修正・同HEADのGo/CI成功まで確認された。CQ-01/03〜06の追加実Provider試験・Owner意味受理は未実施。

| ID | 一つだけ変える入力・状況 | 現行の分岐／独立Expectedで検討すること | Owner |
| --- | --- | --- | --- |
| CQ-01 | 完全0件と、ページ0件・RLS制限の0件・Replica遅延を対照 | 誤ってCoverageCompleteを供給すると空のallow条件がholdsとなり得る。対象L、Filter、Scope、Provider、時点とReader完了を独立に確認する | #2506/#2622/#1706 |
| CQ-02 | Targetsが空、CoverageUnestablished、EmptyPolicyDeny | **main #3051で修正済み**：観測0件＋Coverage未確立/部分はunverified、CoverageCompleteで空が確定しEmptyPolicyDenyならfails。対象内の確実な違反は部分取得でもfails。CoverageCompleteの真正性は呼出側に残る | #267/#2514/#1163 |
| CQ-03 | Handoff類型、Targets空、EmptyJustified真、CoverageComplete、HandoffConfirmed偽 | 現行コードは空のrequires-evidence分岐でholdsへ進み、後段のHandoffConfirmed判定を通らない。**対象0件で手作業確認は非適用なのか、それでも必要なのか**を業務手順のOwnerが決める | #1125/#267 |
| CQ-04 | existence類型で、観測1件Holds真だがCoveragePartial、または観測1件Holds偽でCoveragePartial | 現行非空分岐は全対象成立型。前者unverified、後者failsとなる。**存在条件の個別行Pと完了述語全体を単純に同一視しない**。実Pが存在命題そのものなら入力Mappingの正当性を説明する | #267/#2514 |
| CQ-05 | 別Tenant/時点/部分Readerに由来するEmptyJustified真、または矛盾するがConflicting偽 | Bool引数だけで実Source同一性や実矛盾集合は検査されない。Evidenceの発行者・Reader資格・同一Scope/Version/Principalを同一Attemptで追跡する | #1163/#2389/#2516 |
| CQ-06 | 組織Membershipが遡及訂正され、同じ行値でも有効期間と観測時点が変わる | 旧CoverageCompleteと旧holdsを新しいL/現在Grantへ転用しない。業務時間と実Host認可時点を独立に管理する | #2389/#2600/#1174 |

**最小検証順**：まず#3051のCQ-02修正後のfocused Go/CIを基線とし、CQ-01/03〜05は既存#2514の独立期待値を同一固定Fixtureへ割り当て、現実装とOwner期待の差を分類。その次に#2506のFunction Data Context→実Provider Read→Rの被覆証拠→Completion評価→#1163の同一Attempt/Consumerを連結。CQ-06は組織レーンの既存Fixtureを利用し、必要な書込み試行にだけ#2598/#1174の現在Grant/Effect/Receiptを接続する。CompletionHoldsをProvider Commit済み・正当な承認・業務全体の完了と同一視しない。

### 10.12 Meaning受理と実Grantの2×2：研究計画への接合だけ行う

[Draft #3026](https://github.com/hat47x/tei-platform/pull/3026) は、M＝Owner Accepted Meaning、G＝Current Host Grantの直交する2×2比較と、受注／組織／Workflowの同形異義を既に記述している。**その比較表を本計画へ複製せず、各セルの独立Expected・実証範囲を#1180/#2514/#1163と接続することだけを研究レーンの課題とする**。

M未受理・G許可でもMeaningを採用しない。M受理済み・G拒否なら旧Meaningは履歴として保持し、現在Dispatchは拒否。両方満たしてもTarget・Version・Guard集合・Provider Commit Fenceと現在Consumerが未確認なら業務完了ではない。未受理・拒否の二不足は権限ある診断では別々に保持するが、低権限利用者へ非公開情報を露出させない。Concordatの表示可能Tool集合とHostの実Resource Grantを同一視しない（[論文 §5.3](https://arxiv.org/html/2610.37461v2)）。Draft #3026は未統合で、正式意味受理ではない。

### 10.13 Skillの「割付」と「実提供」を区別する評価方式

§10.7/10.10のA/B/C1/C2について、[TEI #2018](https://github.com/hat47x/tei-platform/issues/2018) および [TAI #168](https://github.com/hat47x/tai-pleroma/issues/168) の既存H2/H3評価計画を再利用する。

- **割付ベース（ITT型・記述集計）**：Skillを利用する予定だった試行が未発見・未ロード・Toolなし・実送出不可でも、分母に含めて障害と費用を測る。C1でSkillを発見可能にしただけでは利用成功としない。
- **実提供条件ベース（PP型・限定比較）**：同一Task、Source raw SHA、検証基準、モデル、Tool権限、提出方式を固定し、H3のRoot＋Required Referencesが実ロードされ、両腕のTool呼出しが実証されたものだけを比較。C2の実消費内容・版も記録する。配送の成功率と、正しく届けた仕様の効果を混同しない。
- **Claimの上限**：H2/H3は既存の固定Task 1件・4独立Sessionが計画分母。未実行であり、仮に4 Sessionが成功しても一般的な因果効果やp95は主張しない。回答品質、Key集合、strict pass、false-qualified、unknown保持、トークン／Tool／時間／人的費用を別記録。

新しいBenchmark RuntimeやSkill評価正本は不要。#2018が経済性、#1180がRequired Obligation、#2514が独立検証基準、#2509がAI/CLI境界を担当する。M4Benchの測定効果量をTEI/TAIの期待改善率へ転用しない。

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

**移行検収の主担当（#3030後続訂正済み、§10.19参照）**：[Draft #3030](https://github.com/hat47x/tei-platform/pull/3030) の M0〜M6／TM-01〜14 を引き継ぐ。研究Draft #3022のMDT-01〜08は「研究から導いた判別観点」として残し、別の移行正本・第二Fixture実装を作らない。すでに段階設計と具体的な負例が#3030/#3031/#2621/#2823にあるため、追加すべきなのは**どの実行結果から何を言ってよいか**の資格境界である。

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
4. **学習済み手順の自己証明**：Source内にAIが生成したRule、Expected、Skill説明が共存しても、Required Obligationの母集団を候補の自己申告や「新拡張子で読めた件数」から決めない。独立ExpectedとKnown Mutant、実Loader/Parser/Tool受領を区別。#1180/#2514/#2018。

**限定業務例の接合**：[PR #3024](https://github.com/hat47x/tei-platform/pull/3024)（exact HEAD 705933667d5b081036314a9161280323e08a64bd、[局所CI #42208159977](https://github.com/hat47x/tei-platform/actions/runs/42208159977)成功）は旧 .tei を使って CaseRecord Persistence→Query SQLのSource/Revisionを確認した**E1c相当の限定実測**であり、*.tei.md Migration/Function Data Context/実SQL Provider Readを証明しない。#3030のM4/M5以後に同じ公開Case・同一独立Expectedを新Sourceへ適用し、A2の同等性とA3の実Provider資格を別に合否判定。組織E2・G1/G2を研究移行のHard Dependencyで止めない。

**決定・実装・未実証の区別**：*.tei.mdへの移行方針はOwner決定済み。#3030はDesign Draft、#3031は検出拒否の実Go局所PASSを持つImplementation Draft、#3024は旧Sourceの局所Go/CI PASS。A1〜A4・新形式による実Provider/CLI/Studio/SHOの本格一巡、移行後Owner Meaning Acceptanceは未達。現行#1119 Gateは据え置き、研究PRの追記を製品Gate達成へ転用しない。


### 10.17 検証基準の二方向判別力：正常と拒否を固定する（2026年11月7日：#3028・#3031・#3032・#3037の実証を横断照合）

**研究から実装へ渡す今回の成熟差分**：検証の厳格さを「不正入力を拒否できた件数」だけで測ると、すべてを拒否する壊れた実装も合格する。逆に「正常入力で実行できる」だけでは、未選択・未取得・不正Grant・Scope不完全を見落とす。必要なのは、**同一Source／同一独立期待値から一要因だけ変えた正負対照**と、実行した入口が本当に対象製品経路に到達した証拠である。論文のVerification Agents／M4Benchの原則を、既存#1180 Scenarioと#2514独立検証基準で実証する。新検証エンジンや第二Test SSOTは作らない。

| 対象／既存Owner | 正例で必要な観測 | 負例で必要な観測 | 現況・Claim ceiling |
| --- | --- | --- | --- |
| **DSL Bootstrap**（#988） | 同じParser Pin/Sourceと契約版が合えば、実Callback到達・候補生成が確認できる | 同じSourceでPin／Fingerprint／Manifest版を一要因変え、適切に停止・旧結果を保持 | [Draft #3028](https://github.com/hat47x/tei-platform/pull/3028)はBS-02/03/06に正負対照を追加し、HEAD aa51946a の[Repository Health #42240218716](https://github.com/hat47x/tei-platform/actions/runs/42240218716)成功。実Plugin世代切替・クラッシュ回復・新 *.tei.md Parserは未実証 |
| **TEI Markdown**（#3030/#2823/#2621） | 実選択された新形式の元byte/Span/RoleをReader→Parser→実消費先まで保持 | 不明Parserなら明示拒否、旧 .tei と二重Subject／未読込Source／未対応Roleを成功扱いしない | [Draft #3031](https://github.com/hat47x/tei-platform/pull/3031)は後者の一部だけ局所CI成功。前者の正常受理と旧新意味同等はNOT RUN |
| **Completion**（#267/#2514） | 全件性が独立に裏付けられたL、P成立、業務別EmptyPolicyが整合する | 部分集合、0行未取得、別Scope証拠、Handoff未確認、矛盾を混ぜて誤完了を検出 | main #2926に加え#3051で未観測0件の誤ったfailsを修正し、Focused Go/CI成功。呼出側Coverage/EmptyJustifiedの実Provider由来は未検証。CQ-01/03〜06は提案のみ |
| **Command Authorization**（#1174/#2598） | 当該主体・Action・Resourceの現在Grantで、正しいTargetだけExecutorへ到達 | 付与なし、無効なRule、同形の別Tenant/Resource、Grant失効で意図せずExecしない | [main #3037](https://github.com/hat47x/tei-platform/pull/3037)で参照Authorizerの3試験を記録。許可・拒否・判定不能を分離。ただし実Grant正本とEpochReader、本番Contextは未接続 |
| **Runtime DB decision**（#1174/#1119） | open period＋covering Grantで実更新、読み取りで対象結果を確認 | closed period、観測不能Effect、重複Run、missing Grantで書込み前停止 | [main #3032](https://github.com/hat47x/tei-platform/pull/3032)の[実PostgreSQL Actions #42235336807](https://github.com/hat47x/tei-platform/actions/runs/42235336807)はsix named testsすべて実PASS、SKIP無し。局所Reference判定の証拠であり、署名付きGrant・実運用Provider/全業務E2Eは未達 |

**二つの否定的な誤りを別に測定**：false qualification（不足証拠を合格扱い）と false rejection/false invalidation（適格な対象を不必要に拒否・失効）。テストの総合Greenと判別力は別。業務意味・Current Grant・Source実消費・実Provider・独立Consumerが必要なClaimでは、限定Go/CI成功をそのまま製品資格へ昇格しない。期待値は実装やAgentの予測から生成せず、契約と独立したFixture起点で先に固定する。

### 10.18 現在資格の継承境界：Studio後着と実Host Effectを混同しない（追跡更新）

旧[Draft #3036](https://github.com/hat47x/tei-platform/pull/3036)は歴史的成果を保存して未統合Close。後継の[Draft #3117](https://github.com/hat47x/tei-platform/pull/3117)（exact HEAD b75df7a54a27a540e51c57778fd173213752491f）は当時のmainから同じ5ファイルを移植し、[Node回帰 #42309517633](https://github.com/hat47x/tei-platform/actions/runs/42309517633)／[実Wails Server/Chromium #42309517587](https://github.com/hat47x/tei-platform/actions/runs/42309517587)／[Repository Health #42309517593](https://github.com/hat47x/tei-platform/actions/runs/42309517593)がcompleted/success。実ChromiumのPASS_BOOTSTRAP_LATE_CATALOGはServer初期Bootstrap / の後着応答の限定資格であり、Desktop既定 /workbench/ のCatalog/Inspect/ValidateやCommand/Provider Effectの受入ではない。#3117はDraft・main未統合、旧#3036の歴史的CIを新HEADの証明に流用しない。

**別UI・別資格**：旧応答/旧finallyの競合を実main.js＋Node模擬DOMで試したのはBootstrap側。Windows Desktop初期経路は **/workbench/** であり、上記Desktop smokeは既存read-onlyであってBootstrap Command/Plan/Executeの実WebView2試験ではない。Workbenchには別のcreateResponseScope/workspaceEpochがあるので、#1824/#1124の担当で実Catalog/Inspect/Validate後着とDOM/ARIA・現在Disclosureを照合する。新UI Scope/Hostを重複導入しない。

**Consumer資格 ≠ Effect資格**：古い画面応答を捨てても、実Hostに送出した書込みは取消しにならない。過去OperationのReceipt/Commit成功だけでは現在Workspaceに表示できない。Selection epochは画面応答の採否、OperationID/Idempotency Keyは実Effectの同一性を扱い、互いの代用にしない。#2598/#1174/#2516と同一Attemptで検証する必要は残る。

### 10.19 移行検収の単一正本：旧矛盾は後続の記録で訂正済み

初期の[Draft #3030](https://github.com/hat47x/tei-platform/pull/3030)と[Draft #2667](https://github.com/hat47x/tei-platform/pull/2667)は互いを移行の主担当に指定していた。しかし2026-11-07の更新で**#3030をAuthoring移行仕様・検収の正本（M0〜M6/TM-01〜14）、#2667を横断差分・履歴・Owner受渡し**とする整理が、双方の後続記録および#2621に反映された。過去の逆参照は履歴であり、現時点で新たなOwner決定待ちの矛盾として扱わない。#3022のMDT-01〜08も第三の移行Gateにはせず研究上の判別観点にとどめる。

**未解決なのは責任分担ではなく実証**：現在の#3031ガードはmanaged Root内の未対応 *.tei.md を拒否するが、非管理Raw Draftの保存と必須Sourceの読込みは異なる用途。暗黙のSource除外や旧形式fallbackでvalidateを見かけ上PASSさせず、managed/selected/read/parsed/actual build consumedを明示的に区別する。Markdown形式・Role/Source span・旧新同等性・Studio/SHO Editor/Provider E2EはNOT RUN。試験対象は#3030 TM、#2514独立検証基準の既存Fixtureに戻す。

**用語**：[Issue #3039](https://github.com/hat47x/tei-platform/issues/3039) とmain統合[#3040](https://github.com/hat47x/tei-platform/pull/3040)の規約に従い、テスト判定は「独立検証基準」「独立期待値」「検証基準の判別力」。Oracle DatabaseはRDBMSの名称としてのみ使用し、実装API識別子・過去Receiptは機械的に変更しない。

### 10.20 Provider全件性の偽証に対する独立判別試験（研究からの成熟差分、NOT RUN）

#3051で、未観測の空集合をfailsとする局所問題は解決した。しかし**実Readが部分集合Rなのに呼出側がCoverageCompleteと自己申告すれば、EvaluateCompletionはその入力を前提にholdsを返し得る**。これは評価器のバグと即断せず、Provider/Function Data ContextからCompletionEvidenceへ変換する境界の責務として検査する。M4Benchの対象Key集合評価、Verification Agentsの空虚な成功・独立witnessと接続する具体的な次の一手は、既存#2506/#2514/#1163の同一Fixtureに限定した偽完全性の負例である。

| 同一合成Owner対象L={D1,D2}に対するRead | 渡してはいけないClaim | 独立検証基準と観測 |
| --- | --- | --- |
| R={D1}, Providerのページ終端未確認、D1はPを満たす | CoverageComplete／全件完了 | **unverified**。Lの独立Required定義、Request filter/scope、ページ継続/Reader終了、Row identity・Revisionの一致を別観測 |
| R={D1,D2}, 一つの同一Read cohortで全件を確認、両方P成立 | 取得範囲の裏付けなしに完了宣言 | Providerがcompleteを保証し実観測を照合できた場合だけCompletionHolds候補。単に2件見えたことは完全性を証明しない |
| R={D1}, D1は明白にP違反 | 完了または単なる未取得によるunknown | 該当Target/Predicateの出典が現在有効なら、未取得D2があっても**fails**。既知反例の証拠は維持 |
| R=空、RLS／Tenant／選択違いによる見かけ上0件 | L=空・業務不存在・完了 | **unverified/開示可能範囲で0件**。非開示の実件数を漏らさず現在の権限とScope相対にする |

**先に固定するもの**：Owner対象集合Lの定義・Predicate/EmptyPolicyと版、合成DB Fixtureの真の行ID、想定Query/Scope、独立期待値。**実際に取得するもの**：Selected Source/Parser/Function/Query/Provider実Binding、実Read RequestとResult Slice、ページ終端・Snapshot/Revisionが保証される範囲、受取Consumer。行数やProviderの自己申告からLを再定義しない。全Providerに同一Snapshot機能を強制せず、整合したRead cohortが立証できない場合はunknownを保持する。

**配置**：#2514は独立期待値とfalse complete Mutantの判別力、#2506/#2622はData ContextとRead witness、#1163はSource/Target/Attempt/Consumerの同一性、#267/#1125は業務EmptyPolicyの意味、#2600は版変更時の再資格化。新完成判定器や認可Engineは作らない。最初は**read-only**でProvider取得の同一性を確かめ、書込みは別の既存#1174 Command/PG/Receipt契約で実証する。新試験・本番Provider・業務Ownerの受理はNOT RUN。


### 10.21 「観測0件／差分0件／成功」の証拠資格（2026-11-07；#3054〜#3088の既存main差分）

§10.8/10.11/10.20の「観測0件≠対象集合0件」は、Read・Validation・Effectの各層へ展開可能。**新しい共通Proof型を作るのではなく、Claimに必要な集合と実観測された集合を分ける**。

| Claim | mainに統合された事実 | 禁止する推論と未検証事項 |
| --- | --- | --- |
| Workspaceの必要Sourceを消費 | [#3073](https://github.com/hat47x/tei-platform/pull/3073)は存在しないRootや未読込に見えるファイルを診断。[#3078](https://github.com/hat47x/tei-platform/pull/3078)は管理54件中、Loader実読取53件と未読取README1件を区別し、readByLoader/sourceRootFindingsを追加 | 管理／実Readは別。readByLoader真でもParse/Role/Required closure/Build消費の資格はない。管理外・宣言外のSourceも自動不存在としない。#2823 |
| 宣言は検証済み | [#3069](https://github.com/hat47x/tei-platform/pull/3069)が利用者宣言件数を追加。[#3081](https://github.com/hat47x/tei-platform/pull/3081)は空WorkspaceへWORKSPACE_DECLARES_NO_MODEL_ELEMENTS警告 | 空Workspaceでもok:true/exit0。標準std catalogの件数と利用者宣言を混同せず、Consumerが必要とするSource集合の充足をokだけで推論しない。#2822/#1180 |
| CLIスキーマが広告どおり | [#3058](https://github.com/hat47x/tei-platform/pull/3058)がdiscoveryを追加し、[#3095](https://github.com/hat47x/tei-platform/pull/3095)がenvelopedコマンドの実Process応答schemaVersionと広告を照合し、偽広告の負例を確認 | schemaVersion一致はresultの形や意味の互換性を保証しない。特に#3078はworkspace-files/v0のresultを**配列からobjectへ変更**し、旧Consumer互換性は未決。#2824/#2516 |
| 宣言と適用に差はない | [#3055](https://github.com/hat47x/tei-platform/pull/3055)・[#3057](https://github.com/hat47x/tei-platform/pull/3057)は同種別別対象の適用をDifferencesへ残しCLI/Studioで表示。[#3068](https://github.com/hat47x/tei-platform/pull/3068)は作用を列挙しないExecutorをAppliedUnobserved=true/difference-check: not observedとする | Differencesが空でもAppliedが未観測なら差なしとは言えない。差が存在しても参照Commandの成功とは両立するため、Owner承認済みScopeやProvider Commit成功には自動昇格させない。#2598/#1174 |
| Commitしたので保存された | [#3054](https://github.com/hat47x/tei-platform/pull/3054)はMemoryUnitOfWorkのcommitted=trueとpersisted=false/durability=noneを独立欄へ分離。[#3088](https://github.com/hat47x/tei-platform/pull/3088)はmain版での一時PostgreSQL六試験PASS・SKIP0をPR版とは別記録 | 参照UoWのCommitを永続化証拠に使わない。main #3124はpersistence=not-applicable/not-observed/observedを導入。main #3127/#3131の独立Lookupは同じOperationIDのReceiptの存在を実PGで確かめるが、対象Resourceの現在値を見ない。実Commit/Receiptと行値・現在Grant・業務Meaning・Consumer到達は別。#2998/#1174/#1163 |

**研究から渡す同一Fixtureの反証**：①ManagedだがreadByLoader=false／必要Role未解析、②validate okかつ宣言0件warning、③schemaVersion一致だが旧Consumerはresult配列を期待、④適用先が宣言外だが同種別のDifference、⑤AppliedUnobservedなのに空のDifferencesを差なし扱い、⑥committed=trueかつpersisted=false、⑦実PGの局所試験PASSなのに当該業務Consumerへ届かない。正例と一要因差分の負例を#1180/#2514の既存Independent Expectedに割当て、独立期待値をProducer出力から再生成しない。

**適用範囲**：同じ成功・空・不在・観測なしでも、対象集合がManaged / Loader-read / Parsed / Role-required / Builder-consumed / Applied / Receipt / Consumer-deliveredのどれかを明記する。false qualificationと、未選択の草稿変更等で無関係なArtifactを失効させるfalse invalidationを別に測定する。#2600/#2516へ再利用条件を委譲。

### 10.22 Sourceの物理同一性の負例と固定Studio Host

[Issue #3061](https://github.com/hat47x/tei-platform/issues/3061)では、**固定Studio Hostの特定版 4bc437c7**におけるResolved Physical Root置換で、allowlist済みWorkspace IDからWorkspace外のCatalogが見える負例を報告（[#3059](https://github.com/hat47x/tei-platform/pull/3059)の負例CI）。Configured Aliasだけの置換はPASS、物理Rootの置換はSECURITY FAIL。これは固定Hostの限定実証であり、**現mainの同じ脆弱性を立証したわけではない**。後続の[main統合 #3098](https://github.com/hat47x/tei-platform/pull/3098)は現mainのReadOnly Hostが起動時の実体rootを保持し、各操作前に同一性を再確認する仕様と[#3063](https://github.com/hat47x/tei-platform/pull/3063)の限定試験を記録した。一方、固定PoC版4bc437c7は別の歴史的脆弱性のまま扱い、更新の要否とin-flight fd raceは#3061/#1824のOwner判断。

Source Path文字列、Workspace ID、Source byte digest、実物理rootのidentityを同一証拠としない。許可時と実Readerアクセス時のrealpath/Root identity、symlink/rename・ABA・別Tenant/Scope・現在権限を照合し、矛盾時は読取停止。診断で非公開のSource件数やResource名を漏らさない。#2571/#2823/#1124/#1824/#2516へ独立負例を割当て、第二Path Resolver/Host Authorityを設けない。

### 10.23 次の最小実証：一つの証拠連鎖、二つの別途Gate

**Read-onlyを先行**：旧Source/独立Expectedを固定し、Selected/Required→実Read byte/locator（#3078はこの一部）→Parser/Role/Required closure→Function/Query/実ProviderのRead対象Lと取得集合R/完了性（§10.20）→CLI/Consumerの実受領値・Schema/開示資格を、#1163で同一Revision/Attemptへ束縛する。新しい *.tei.md は#3031の未束縛拒否を出発点とし、**正式Parserの正例がない限り意味Sourceとして合格としない**。#3095のschemaVersion一致と旧ClientのJSON互換性を混同しない。

**Writeは別の縦切り**：Meaning Acceptance、現在Host Grant/対象Target、Required Guard、実PG Effect、OperationID/Receipt Lookup、**別ConnectionによるResource現在値Read**、現在Consumer Disclosureは各々独立Witnessが必要。[main #3124](https://github.com/hat47x/tei-platform/pull/3124)のpersistence三状態、[#3127](https://github.com/hat47x/tei-platform/pull/3127)のReceipt Lookup、[#3131](https://github.com/hat47x/tei-platform/pull/3131)の実PostgreSQL試験は前進だが、Receipt LookupはResourceの現在値を検証しない。#3118の宣言Effect報告、#3055/#3068のApplied/Unobserved、#3133の未配線ChainFileの成果も単一Attemptの実Effect全件性やAuditの真正性を証明しない。#3088のmain版PostgreSQL六件PASSや参照Commandの別CIを合算して一つのOwner/Consumer業務完了にしない。

**Buildは並行**：[merged #3091](https://github.com/hat47x/tei-platform/pull/3091)のModule Preflightにより、隔離GOPROXY=offでcmd/teiにpgx/v5、x/text、modernc.org/sqliteの未供給が観測され、Source closure/Artifact生成以前のSB-0bが必要と判明。G4はUNQUALIFIED。[Draft #3130](https://github.com/hat47x/tei-platform/pull/3130)は選択Moduleが0件・stdlib-onlyをsupported扱いする空虚な成功をunknownへ修正し、exact HEADのRepo Health／Windows Builder CI成功。ただしまだmain未統合で、実Module供給・Go Artifact/Linkの証明ではない。[merged #3094](https://github.com/hat47x/tei-platform/pull/3094)の構成Kernel A/B/C局所パケットは同じ固定checkoutで個別に実行されたが、単一Request/Operationに結合していない。artifacts manifest.complete=false／未解決6も記録されている。個別のPASSを製品一気通貫のPASSへ加算しない。

**状態**：既存main局所実装と負例を照合したDocs-only研究差分。新 *.tei.md Reader/Parser、Provider→Completionの実全件証明、旧固定Studio Hostの修復とin-flight fd race、workspace-files旧Consumer互換、Host/Provider/Owner一改訂E2EはNOT RUN。現mainの限定root identity試験は後続#3098で確認済み（§10.22）。新Core、Runner、Registry、専用TEI Hostを追加せず、#1119の既存Gateを維持する。


### 10.24 OKF v0.2：可搬な知識と実行資格を二段階に分離する（2026-11-07、OKF v0.2一次仕様 §3〜§12の再確認）

**出典**：[GoogleCloudPlatform OKF v0.2 canonical SPEC](https://github.com/GoogleCloudPlatform/open-knowledge-format/blob/main/SPEC.md)。OKFはfrontmatter typeを持つportable Markdown Concept形式であり、storage/serving/実Query/実Attester ABI/Receipt wire/sandboxを提供しない。§10.5のDiscover→Load→Parameterize→Execute→Attest→Gateは実装説明（informative）、§12はRuntime protocol/Attester ABI/cache未規定と明記。新TEI Runtimeや独自中央Attestation Engineを採用済みとはみなさない。

| OKF v0.2の最小仕様・シグナル | TEI/SHOで扱う段階 | 非推論と既存Owner |
| --- | --- | --- |
| 非予約 *.md はparseable YAML frontmatter＋非空type。未知type/keys、任意fields欠落・broken link・index欠落は寛容に扱う | OKFの**形式Conformance**と、TEIのRequired Source/Role/Operator/現在実行資格を別列で出す。SHO Raw Draft保存・OKF閲覧・TEI Meaning Promoteを別Operationへ | Conformance PASSを実TEI Parser、Accepted Meaning、外部Code実行可に昇格しない。#3030/#2621、[SHO #18](https://github.com/hat47x/sho-composer/issues/18) |
| sources[].resourceと任意id、Claim脚注／generated/verified Actor・時刻 | Source IDと足場、Reference/Revision、Source spanとClaimごとの引用を保持。Writer/Readerは未知keysと元bytesの非対象領域を保持 | resourcesはURL以外のScope記述も可能。URLの有無・verifiedのhuman:文字列・usage_countを真正性／独立検証基準／Authorityと同一視しない。#2514、[SEI #820](https://github.com/hat47x/sei-cognition/issues/820) |
| statusはdraft/stable/deprecated、省略時stable。stale_afterはUTC offset付き絶対時刻 | 文書Lifecycle／期限とSource Revision・Operator Pin・Decisionの現在適用性を別軸で追跡 | stableデフォルトでもOwner受理とは言えず、期限内でもSource改訂・撤回・A→B→Aを失効判定から外さない。#2600/#267 |
| index.mdは任意の段階的発見、log.mdは人間向け日時履歴、通常Markdown Linkは無型 | SkillのRoot→必要Conceptだけ取得、必要Referenceの閉包、Source/Consumer実読込を別計数 | index未掲載≠対象不存在、Log≠Decision履歴/Receipt、Link≠型付き関係・Grant。#1180/#2823、SEI #820 |
| Concept IDはbundle pathの末尾.mdを除いた値 | **TEI固有の衝突例**：sales.order.tei.md のOKF Concept IDはsales.order.teiとなり得る。旧sales.order.teiのPath、TEI Semantic Subject ID、Owner Acceptance IDと異なるidentityとして保持する | ファイル名の見かけ上の一致で旧新のMeaning/Decision/Build/Receiptを自動結合しない。#3030 TM、#2600、SHO #18 |
| Attested Computationはruntime、型付きparameters、正規computation、executor.resource/receipt、決定的attester.resourceを区別 | 定義／契約versionと実行したArtifact、Receipt／Attester verdictを同一Attemptで照合 | OKFの形式適合、定義verified、Run attested、Owner Accepted Meaning、現在Host Grant、実Provider Effectを**別判定列**へ。#2514/#1163、TAI #514 |

**導入形態**：OKFをTEI/SHO/SEIの正本へ上書きしない。既存[TEI Draft #3029](https://github.com/hat47x/tei-platform/pull/3029)とSHO #18が選択式の文書/Domain Lint、SEI #820がKnowledge Export/Import境界、TAI #514がSkill/Attestationの横断計画を担当する。新Parser/Resolver/Source Registry/中央Trust Scoreは導入しない。

### 10.25 Attested Computation：まずread-onlyの「正規計算が走ったか」だけを実証する

一次仕様OKF v0.2 §10.3は**Agentは宣言済みtyped parameterの値だけを供給し、正規computationを編集しない**とする。Consumerがbindした実SQL/compiled artifactをReceiptの実行Artifactと照合し、deterministic no-LLM AttesterがReceiptと権威あるJob/Result再読取に基づいて判定する。**Definition verified**は文書／定義のレビュー記録、**run attested**は一試行の計算経路に対する検証で、互いに代替できない。

既存[TAI #514](https://github.com/hat47x/tai-pleroma/issues/514) AC-01〜12／[TEI #2514](https://github.com/hat47x/tei-platform/issues/2514)の独立期待値・既知Mutantを実証の正本とする。新ケース番号を第三の検収正本にしない。最小の同一Fixtureで必要な正負対照は、正規typed bind→実Read→独立Result（正）、AgentによるQuery差替え／型不正、旧JobIDの流用、Receipt捏造、Result一致だが別Operator/Plugin Pin、constant-PASS Attester、部分Rowsetなのにcomplete、期限内定義だが現在Source変更、Run attest成功だがOwner受理撤回（負）である。**Attesterのbytes/実行環境/入力と判定版もPinし、Attester自身の自己申告だけを証拠としない**。

**最低観測束**：selected OKF Concept bytes/Path/Source Revision／正規TEI Function/Query/Role/Operator/Targetのexact selectionとtyped parameter bytes／実Host/Provider実行資格・Request・実際に実行されたSQLまたはcompiled Artifact／実Receipt JobID・Result・Scope/Row completeness・Timestamp／**Receiptの独立Lookupと、Result/Resourceの独立再Readを別Witness**／Attester actual code+verdict／現在Consumerの開示資格。欠落したらunknown/unsupported/not-run、モデルに架空のReceiptを書かせない。OKF Receiptは**bundle外のruntime artifact**であるため、TEI既存#1163 Evidence Packetへ参照する設計にとどめ、OKF文書へ各実行ログを強制保存しない。

**採用順**：最初は実CLIに存在するread-only Source/Inspect/Queryの範囲で**契約読取だけ**を提案し、正式Executor/Attesterが無ければunsupported。実Host Side Effect・DB更新・配備・削除はOKF §10のRun Attestationをそのまま転用せず、#1174のOperationID/Grant/Commit fence/Receipt/独立ReadとOwnerの明示受理が必要。新Attester ABI/Runtime/自動Host Permissionを先に固定しない。実OKF Contract→TEI CLI→Executor→Attesterの一改訂試験はNOT RUN。

### 10.26 外部Harnessの実Tool到達：H2/H3の比較母集団へ混ぜない（2026-11-07）

[TEI Evidence PR #3113](https://github.com/hat47x/tei-platform/pull/3113)／[TAI #168](https://github.com/hat47x/tai-pleroma/issues/168)に、DSH 0.2.2-rc.1から固定TEI read-only実行ファイルを起動し、実Tool request/CLI result/最終回答まで到達したh2-08の**限定実観測**がある。固定Source revision 50ebf8014340fd1906f6b35cfa30173e0ee2fc6c／公開sales.order／固定Participant Task。Tool Call 56、実TEI起動32、CLI Validate warning3。試行9件は参加者Task完了1、短い接続確認2、MALFORMED_RESPONSE5、INVALID_CONFIG1。PRはOpenで、DSHの全原JSONLは保存されておらず、製品側の要約・保存したTool Result以外を独立に再構成できたとは言わない。

**新しい境界**：read-only指定でも実作業Dir外の元Worktreeを読み出せたため、Skill/Expected/Holdout非露出・OS isolationを保証できない。Windows workspace-write sandboxはWin32 5で失敗、Egress拒否も未実証。したがって**技術的 Tool transport到達**と**H2比較適格**を分離。H2/H3四Sessionの適格件数は0/4で、探索9件を既存4つのITT分母へ事後追加しない。

**最短次段**：既存OS/VM/Container機能でH2 Participantに固定Binary・公開Workspace・Taskのみを見せ、Skill Root/References/独立Expected/過去回答/他Sessionへ本当にアクセスできない観測を先に固定。その条件で一つ目の比較適格H2を実行し、その後H3で必要Skill Root+Referenceの実Reader受領を観測してH2→H3→H3→H2四独立Sessionへ進む。[TEI #2605](https://github.com/hat47x/tei-platform/issues/2605)がProduct Skill現物とCLI一致、[TAI #168](https://github.com/hat47x/tai-pleroma/issues/168)が比較条件を所有。旧[TEI #2614](https://github.com/hat47x/tei-platform/issues/2614)はclosed/duplicateなので、新担当をそこへ戻さない。TAI #514のOKF計算Attestationを初回H2/H3のBlocking条件にしない。

### 10.27 SEI知識管理：OKFを交換投影に使い、Truth/Decisionの正本にしない

[SEI #820](https://github.com/hat47x/sei-cognition/issues/820)（open/設計、KO-01〜15すべてNOT RUN）は、OKF v0.2のsources、Actor、status/stale_after、index/log、Attested Computation等をSEI Observation/Hypothesis/Recommendation/Decision/Method/Unknown/Lineageの**選択的Export/Import候補**へ結び付けている。元SEI Logical Ref/Exact Binding/Responsibility/Method Application/未解決残余・Authorityの正本は変えない。Source引用の脚注順が変わっても元idが変わらない、同一起点の転載2件を独立根拠2件と数えない、status省略stableでDecisionのunresolvedを消さない、Actor human:ラベルから本人確認を推論しない等の負例を既存SEI Ownerへ渡す。

TEIとの実接点は、(1) accepted meaningの根拠Original SourceとRevision/Scope、(2) attested計算Resultの現在妥当性と未取得・非開示、(3) TEI/SHO/SUI/TAIへ渡すClaimの説明・未決・現在Context、の**read-only Handoff**。外部OKFの閲覧は可能でも、Native Decision StoreへのImport/上書きにはSEI既存#775のContract・Exact Binding・Owner Reviewが必要。OKF index不掲載／broken linkは形式として許容しても、必要根拠の閉包不明なら用途別Qualificationはunknown。SEI版付きObservationの失効、更新時のA→B→A、Decision撤回をOKF status/stale_afterで代用しない。

**次の実験は既存Ownerの順序で**：TEI #2605/TAI #168の外部実Tool/Skill比較を先行。SEI #820の一Concept/複数Claimのread-only Exportを別Fixtureで照合し、OKF形式適合・Source実Read・SEI問い相対の十分性・Attester verdict・現在Host/Consumerを別列にする。実SEI OKF Import/Export、実Attester、実H3 Skill Load、Owner Meaning受理はNOT RUN。

**本追補の性質**：OKF原典と現GitHub Issue/PRを照合した研究・計画更新。既存#3029/#2514/#2605/TAI#514/SEI#820を参照するものであり、新Plugin実装・第二Evidence Registry・新Attester Runner・共通Knowledge Store・業務Hostは作らない。業務G1/G2、G4 Self-Build、SHO WriterをOKF実証のHard dependencyにしない。


### 10.28 Receiptを読めたことと、業務状態を読めたことを分ける（2026-11-07、実PostgreSQL exact-HEAD CIまで追跡）

mainへ統合された[#3124](https://github.com/hat47x/tei-platform/pull/3124)はCommandのPersistenceを not-applicable（参照UoW／宣言Effectなし）・not-observed（Durable Store想定だが検証不可）・observed（PersistenceObserverが与えられた宣言Effectを確認）の三状態へ分離。ObserverにはExecutorのResultを渡さず、自己申告値の横流しを抑止。一方[#3127](https://github.com/hat47x/tei-platform/pull/3127)のEffectcommit ObserverはRequest/Grant/OperationIDが一致するRunner.LookupでReceiptがreplayedとなれば観測成立とする。

**このobservedの観測対象はReceiptであってResource現在値ではない**。[#3131](https://github.com/hat47x/tei-platform/pull/3131) exact HEAD ed1af3b17db6690bb8c9846c7f5ef36ed1ada02d の[実PostgreSQL CI #42318032659](https://github.com/hat47x/tei-platform/actions/runs/42318032659)はcompleted/success。新試験は実Commit→Receipt Lookupでpersistence=observedとし、Receipt行を除去するとLookup失敗になる一方、対象objects行の現在値が同じであることを確認する。WindowsローカルではDSNなしで新PG試験はSKIPだったため、CIと混同しない。

| 観測段階 | 局所実証／期待できるClaim | そこで止める推論 |
| --- | --- | --- |
| CommandがCommitを返す | UnitOfWork / Provider boundaryの応答 | 永続化、独立観測、Owner受理は未確認 |
| Receipt Lookupが同じOperationIDを再取得 | #3131の特定PostgreSQL経路でのReceipt existence/replay | **対象行が現在も期待値であるとは言えない**。Receipt削除とResource保持を対照 |
| 独立Data ReaderがResourceを再読取 | **main #3155で実装・実PG局所CI成功**。別poolでobjectsのVersion/Valueを読み、Commit payloadに一致した場合だけresource-observedとする | すべてのCompositionでの既定化・Current Grant・RLS/Replica遅延・Owner/Consumer一改訂E2EはNOT RUN。Row 0件を不存在としない |
| Consumerが現在の結果を受領 | Schema/Policy/Disclosureと対象・改訂・Attempt一致 | Business Completion/Meaning Acceptance/現在Grantの自動導出は禁止 |

**研究の最小反例**：一つの合成受注に対してA：Receiptあり・objects現在値がExpectedどおり、B：Receiptあり・後続更新でobjectsが違う、C：Receipt削除・objects値は保持、D：Commit応答喪失でLookup=unknown、E：Receiptあり・別Tenant/別OperationID、F：Providerなしの参照MemoryUoW。A〜Fは独立Expectedを先に固定し、Receipt Lookup successとResource Result正しさを混同して判定するConsumer／Attesterを失格とする。既存#2998/#1174/#1163/#2514/TAI #514に属し、別の共通Persistence Engine/Receipt ABIを作らない。**A〜FのうちResource一致／不一致・Receipt削除はmain #3155と#3131で局所実証に進んだ。全A〜Fの同一Consumer横断試験とOwner受理はNOT RUN**。

[main #3133](https://github.com/hat47x/tei-platform/pull/3133)はAccess判定ログ用ChainFileで途中改変を検出し、後続の[main #3136](https://github.com/hat47x/tei-platform/pull/3136)は限定書出しToolの-access-logへChainFileを接続。[main #3211](https://github.com/hat47x/tei-platform/pull/3211)はHeadPin/PinnedJournalを追加し末尾削除の局所検出も実装したが、当該ToolへのPinnedJournal配線、独立Pin保管、全履歴を書き換えられる者への耐性、判定本文の真正性は未証明。**Record存在／記録連鎖整合／決定の真偽／Hostによる現在認可**は別の検証列とする。

### 10.29 独立検証基準の判別力：第二Mutantが実AI Scenarioに検出された

[NEVER MERGE Draft #3128](https://github.com/hat47x/tei-platform/pull/3128)のHEAD 4ccb8759a2dcdb48766b8fde33cae845c1aa2379 は、[正常基線 #2901](https://github.com/hat47x/tei-platform/pull/2901) HEAD a6eb80cffe16a57f4cb2461ff2946065955a4e50（Health成功）からrelation-uniqueの評価で**最後の複合Key要素を無視する**変更だけを入れたMutant。既存Scenario・Fixture・独立Expectedは不変。前回[#3017](https://github.com/hat47x/tei-platform/pull/3017)の「最終Result誤判定」と異なり、今回は入力処理（Tail Truncation）の欠陥。

[Mutant Health #42317468571](https://github.com/hat47x/tei-platform/actions/runs/42317468571)はcompleted/failure。しかし単にCI赤だっただけでなく、**実ログJob 128233525880の452〜454行**で TestAIAddedScenarioCandidateFiles/ai-candidate-line-number-uniqueness が ai-line-number-duplicate について Canonical Validation ... result=satisfied, expected=not-satisfied を検出。同じGo testでは既存Evaluator回帰（546〜550行）、DSL language-pluginの非同等試験（583〜586行）もFAIL。gofmt／vet／reference workspace validateはPASS。したがって**この固定Caseに対する第二のAI Scenario mutation killは実証済み**。別Workflow「Action Surface actual Go CLI witness」の失敗は lstat ... action_surface_handoff_test.go file missingという別の失敗で、Mutant killの根拠に流用しない。

判別力について言えるのは、正常基線同一Fixture上で二種類の誤実装（#3017 result inversion、#3128 tail truncation）に対して既存AI ScenarioがRedになったことまで。Providerが実際に全Keyを返した保証、非表示/Null/Partial、別Scope/実DB、業務Owner Meaning Acceptanceは未実証。#2514が独立ExpectedとMutant分類のOwner、#1180が必要Scenario集合、#1163が実Source/Target/Provider/Consumer一改訂のEvidence。**新しいMutation Runnerや第二検証基準ストアは作らず、#3128は絶対にmainへmergeしない。**

### 10.30 観測不足のsupported判定と、外部ToolのGrant型の未決

[main #3091](https://github.com/hat47x/tei-platform/pull/3091)のGo Module Preflightでは、標準packageだけ・空のgo list streamで対象Moduleを実観測せずsupportedが返り得る。[Draft #3130](https://github.com/hat47x/tei-platform/pull/3130)がこれをunknownへ変更し、exact HEAD d6ac7dedの[Windows Builder CI #42317891135](https://github.com/hat47x/tei-platform/actions/runs/42317891135)／[Health #42317891168](https://github.com/hat47x/tei-platform/actions/runs/42317891168)はcompleted/success。ただし**main未統合**で、Source閉包・実Artifact/Link/Plugin Invocation・G4 Self-Buildは不成立。空の観測結果は標準package由来の「正常な対象集合ゼロ」か「対象Moduleを調べ損ねた」のか区別する。#2831/#1251の既存受入へ返す。

[main #3121](https://github.com/hat47x/tei-platform/pull/3121)では、参照Toolの環境Grantを追加しようとすると、Provider CapabilityがPlugin Catalogへ解決できず PERMISSION_GRANT_CAPABILITY_NOT_RESOLVED が実発生した。これは**ToolをPluginへ正規化するか、別種のPrincipal/Grantとして定義するかが未決**であり、GrantのValidationを緩めてSource/Capabilityの未解決を通してはいけない。OKF AttesterやDSH/MCP Adapterを接続する際にも、Skill/Tool文書の存在を環境Grantとして推論しない。既存#951/#1174/#1251およびAccess設計Ownerで決定し、Research Draftで新しいHost/Plugin登録方式を先行採用しない。

**優先して実証する順序は不変**：read-only Source/Provider/Consumerの実Read閉包→必要な場合の同一OperationIDでのEffect/Receipt/Data Readback→Meaning/Grant/Attesterの用途別再資格化。H2/H3隔離・G4・SHO Writerを互いにHard Dependencyにしない。


### 10.31 機能に束縛される標準表現型：データ構造とRendererの間に置く契約（2026-11-07、標準データ構造・標準表現型のOwner議論を既存Issueへ還元）

X-TEAのテーブル1:1／1:N→XF単票・明細→遷移の固定対応を、TEIの形式正本へそのまま昇格させない。ユーザーが提起した「テーブルから独立した標準データ構造」と「機能から独立した標準表現型」は別の再利用単位であり、具体的な表示・入力Instanceは**FunctionのData Contextと操作目的に束縛されることで初めて意味を持つ**。型定義やPalette Previewが単独で存在できることと、業務上有効な機能Instanceが単独成立することは別。

**既存の単一Owner**：[Issue #2495](https://github.com/hat47x/tei-platform/issues/2495)が構造・多重Facet、[#2497](https://github.com/hat47x/tei-platform/issues/2497)がProjection忠実度と逆写像、[#2507の2026-11-07成熟追補](https://github.com/hat47x/tei-platform/issues/2507#issuecomment-6701694460)が標準表現型・Function合成、[#2506](https://github.com/hat47x/tei-platform/issues/2506)がData Context/Result Slice、[#997](https://github.com/hat47x/tei-platform/issues/997)が実UI Renderer、[#1127](https://github.com/hat47x/tei-platform/issues/1127)がFunction Patternを所有する。Research #3022に第二の表現型Catalog、Core enum、Renderer、Case SSOTは作らない。

| 層 | 契約・再利用対象 | 独立して存在できない／保証しないもの |
| --- | --- | --- |
| Standard Data Structure | Record、Relation、Bag/Sequence、Hierarchy、Graph、Time/Spatial/N-D等。Entity/Edge occurrence identity、Grain、Cardinality、Order、Null/Missing、Read Scope/Completeness、Unit/時点は複数Facetで表現 | RDB table名・PK型＝業務意味、関係の存在＝実Query取得、全件の観測、編集許可 |
| Standard Representation Type | Panel/Tab/Group等のレイアウト、Text/Single・Multi-select/Toggle/Slider等の値入力、Table/Tree/File-system/Graph/Chart/Timeline/Map等の構造・分析投影。定義・Preview自体は独立可 | Widget選択＝Data Relation、Tab非表示＝Data不存在、Slider範囲＝Business Rule、File-tree表示＝OS実権限 |
| Function Binding | 明示的Data Context/Result Slice＋Field/Role/Relation path/Occurrence・選択集合＋Function Intent＋投影のloss/編集可否を適用 | 表現型だけで業務Function/Action成立、型名一致＝適格、UIで見える＝現在Disclosure許可 |
| Operation/Host/Consumer | ViewChange→typed Intent→既存Operation/Grant/Provider/Receiptと、Web・Desktop・CUIの実Projection | Tab/Sort/ZoomやdragだけでCommit、Offer表示＝Host Grant、受理済みモデル編集＝実業務Data編集 |

**検証の最小差分（試験正本は#2507のP-Form/Layout/TableChart/TreeGraph/Action/Dual-use）**：同じData Context上のPanel↔Tab／Table↔Chart／Tree↔Graphの切替で原Entity/Edge ID・Completeness・Source版・ActionのTargetが不変か、表示で失う情報を明示できるかを既存FP-01/02で判別する。Treeの単親化で隠れたEdgeを消さない。集計Chartの点を元明細1行に暗黙Patchしない。Slider stepと金額/数量Precisionが合わないならLossを宣言、未取得のSelect候補や非表示Tabを不存在としない。Form/Modeler/CUIの操作面が違っても同じBusiness Action IDの現在Grant・EffectをUIコンポーネントで捏造しない。最初から全UI種類の実装をGateにしない。

**状態**：既存Issueへ反映済みの成熟設計候補を研究目線で橋渡し。正式標準表現型schema/DSL、実Renderer/Designer、Studio/SUIの全UI、Owner意味受理はNOT RUN。先行はHeadless Function→Binding→Projectionの説明可能性と二種類の異質Consumer、後続に実UI/Readback。

### 10.32 Pattern宣言はまだ操作面へ実消費されない：Bindingの欠けた一辺を測る

[main #3145](https://github.com/hat47x/tei-platform/pull/3145)の合成CLI観測では、見積り機能のPattern候補を追加してWorld/ModelRevisionが変わっても、tei operationsは8件のderivedのみ・declared 0・gaps 0のまま変化しなかった。[main #3167](https://github.com/hat47x/tei-platform/pull/3167)で受理済みWorld（Pattern5件）とCandidate World（6件）を実比較してもOperationは8件で同一。これは**Function/Patternが宣言されていることと、Operation Surfaceがその宣言を実消費したことは別**という反証である。

照合の手掛かりはData ID、Capability ID、Relation targetの3系統だが、Data IDだけではread/list等のKindが曖昧、効果無宣言のread-only commandには対応Operationが無く、Relation targetはOperation側にない。[Draft #3172](https://github.com/hat47x/tei-platform/pull/3172)はW3のKind対応表と衝突/上書き条件を設計候補として整理しただけで、実配線・採用・意味受理はNOT RUN。読み取りcommandの未結合を勝手に書込Effect付き操作へ補完せず、origin=declared/out-of-scopeの扱いはOwner判断に戻す。

さらに[main #3163](https://github.com/hat47x/tei-platform/pull/3163)の実試験で、modelRevisionは**管理対象Sourceの版**であり、Loaded World版・Function実消費版ではないことを固定した。空Workspaceでも空入力hashのmodelRevision／非空のstd catalog／declarationCounts.total=0は共存する。Functionへの表現BindingとOperationを検証する場合、World Fingerprint／modelRevision／Function Context source／actual consumed Pattern／Operation bindingを別証拠にする。名前・hash・8件Operations・validate okだけで閉包を成立としない。

**次の最小一因子試験**：同じSource/ContextのPatternを1件だけ追加し、(a) World/Source Indexに宣言は実在、(b) 選択Patternが必要なRole/Scopeを満たす、(c) Operation Bindingが実表示されない現行結果、(d) 明示的に結べた候補だけにOperation ID／Action・Revision・Disclosureが引き継がれる将来結果、を独立Expectedと別列で対照。既存#2507/#2586/#2514/#1180がOwner。新Binding Engine、W3の先行採用はしない。

### 10.33 ReceiptとResourceの観測資格がmainで分離実装された（旧§10.28の現況を更新）

[main #3155](https://github.com/hat47x/tei-platform/pull/3155) exact HEAD 19f00c7e3fd89aecbac791002cb081c00052bcfe により、receipt-observedとresource-observedは明示的に別状態となり、persisted=trueは**Resource実再読取によるpayload/Version一致**に限定された。DetailedPersistenceObserver、effectcommit.ResourceReader、postgres.Provider.ReadResource、commandadapter.ResourceReadが追加され、別pool Connectionでobjectsを読む。既存Observerの互換契約は保全。

[実PostgreSQL CI #42328707393](https://github.com/hat47x/tei-platform/actions/runs/42328707393)（postgres-conformance）はcompleted/success。Job #128269964949の実ログ926〜927・991〜992・1056〜1057行で、TestPostgresCommandObservesTheStoredResourceが3回RUN/PASS（SKIP無し）。Receipt確認の[#3131](https://github.com/hat47x/tei-platform/pull/3131)に続き、**実データの現在値確認に限定した一つの入口が成立**した。

ただし、ResourceReadを実Product Compositionへ既定で有効化するか、読取権限・別poolの時点一貫性・Concurrent Update・Replay時の確認方法、Resultの現在DisclosureとOwner Business Completionは未決。resource-observedは**観測時点の特定Resourceの一致**であり、将来も不変・全Action scope完了・Owner Meaning acceptedを含まない。途中更新でReadが不一致となったとき、Commitが起きなかったと逆推論せず、Receipt/Resource両方を独立観測する。既存#2998/#1174/#1163/#2514で再資格化し、第二Receipt Store/Observerは作らない。

### 10.34 Go供給・実Artifact・Source閉包と、外部Harness隔離を分離する

**Build**：[main #3142](https://github.com/hat47x/tei-platform/pull/3142)で、選択された12 Moduleの.mod/.zipをgo.sumのh1と独立照合し、非選択5 Moduleは欠落で不合格にしない限定SB-0bが成立。Host cache非継承、1-byte tamperを検出する実測と[Health #42331834213](https://github.com/hat47x/tei-platform/actions/runs/42331834213)成功。ただし供給元Trustや正式Source閉包のOwner Acceptanceではない。[Open #3156](https://github.com/hat47x/tei-platform/pull/3156)では固定WSL2/Go1.27.3の隔離環境で供給からcmd/tei Buildが実成功しArtifact SHAを記録、一方既存ResolveSourceClosureは同じ対象をrejectedと判定。実Build成功≠Source Closure qualifiedの差が実証された。#3156の当該HEAD Healthは照合時in_progress、main未統合、G4はUNQUALIFIED。同一Stage1 Builder実使用・Stage2再入・二回Bit一致も未実証。#2831/#1090のSource Closure方針をResearch側で勝手に緩めない。

**OSファイル制限**：[main #3152](https://github.com/hat47x/tei-platform/pull/3152)ではaccess.FileSystemの文字列Prefix照合だけだとRoot内junction→外側ファイル操作が通った実負例を踏まえ、Go os.OpenRootによるOS同一操作時の拒否へ切替。外向きリンクと、場合によってはRoot内の正当なリンクも拒否する局所特性・OS差を別記録し、[main #3159](https://github.com/hat47x/tei-platform/pull/3159)へ説明更新。ただしこれは**FileSystem API一経路**の制限で、外部DSHハーネスの全Tool/exec/Web/NetworkやProcessのSandbox成立にはならない。TAI #232/TEI #2515が隔離Owner。

**外部送出**：[main #3153](https://github.com/hat47x/tei-platform/pull/3153)のDSH 0.2.2-rc.1ローカルMockでは、Agentが読んだ合成Markerがモデル入力だけでなくdsh_session_logのtool/result eventに2箇所現れ、Tool Resultの二次送出がある。[main #3161](https://github.com/hat47x/tei-platform/pull/3161)では**既定Profileでもsession log extensionが約45.5KB**のRequestに載り、503返答と200空body STREAM_CLOSEDで水位が進まず再送された。これらは実Mock HTTP観測であり、実外部Providerへの送信・保存・全Egressや通信遮断の実効性はNOT RUN。H2/H3のSkill/Expected Holdout非露出はPath/Tool/Session/Networkを含めて実強制した場合だけ比較適格とする。

**合成禁止の境界**：go.sum一致・実Build artifact、os.OpenRoot拒否、DSH read-only session、OKF verified、Recorder Hash Chainの個別PASSを一つのWorld/Host security Gateに合算しない。現在実Readしたbytes、実接続境界、現在のGrant・Disclosure、実Network送出、Source ClosureとOwner QualificationをClaimごとに保持する。

**次の実証優先**：P0=Function Data Context→標準表現Binding→read-only Operation/Consumerの一改訂実投影（#2507/#2586、未知とlossを保持）。P1=既に局所合格したResource現在値Readを実Function/Action Compositionへ連結（#2998/#1174/#1163）。P2=SB-BのBuild成功とSource閉包不適格が分岐する理由を#1090のOwner Contractへ戻す。P3=既存TAI #168のH2/H3隔離とDSH二次送出を実観測。実GUI/Writer/Provider Full E2E、Self-Build G4、外部通信の安全資格、正式Owner Meaning AcceptanceはNOT RUN。


### 10.35 表現型の媒体時間軸と文書位置の非同一性（#3192を既存#2497/#2507の検証へ限定接続）

[Draft #3192](https://github.com/hat47x/tei-platform/pull/3192)は、X-TEA由来の「データ構造トポロジー↔表現型トポロジー」を音声・動画・文章まで一般化した設計候補。表現型定義はFunctionから独立して再利用できる一方、意味を持つInstanceはFunction Data ContextとRole/Operationへ明示的に配置・束縛する。形式・表示媒体・意味・認可の正本は増やさない。**現#3192 HEAD 951ab46eの[Repository Health #42337006404](https://github.com/hat47x/tei-platform/actions/runs/42337006404)はcompleted/failure（Plan READMEの索引試験）で、正式設計採用やUI実装のPASSとはしない。**

| 区別する軸 | 独立した意味 | 反例・停止条件 |
| --- | --- | --- |
| Domain/event-time | 業務発生・効力期間・Sensor観測対象時刻 | Provider取得時刻／映像再生時刻から業務の効力時点を推測しない |
| Source observation/revision | 原bytes・実Rowset・Scope・Readerの取得版 | Streamを再生できたことを、全Window取得や現在Meaningへ昇格しない |
| Media clock | 音声sample、動画frame、再生時刻、同期基準 | 別時計のMedia/Sensorを同じ時刻軸とせず、同期には写像と誤差を要求 |
| Structural position | Markdown段落・文書offset、Graph layout、画面上の位置 | 文章の順序を時系列、描画座標をRelation、ドラッグを業務保存としない |

**独立Expectedの最小組**：既存[#2507のFunction Pattern反例](https://github.com/hat47x/tei-platform/issues/2507#issuecomment-6701694460)と#2497/#2514の正本に、①JOINによる行増幅とTable/Chart、②DAGの複親をTreeにする損失、③QuantityをSliderのstepで丸める損失、④外部SensorとVideoのclock差、⑤Markdown→TTSの再生時間がSource spanに等しくない例、⑥Sonificationから元の計量値へ暗黙逆写像しない例、⑦Partial集合がTabで非表示、⑧同一GraphのModelerと業務Functionで別Grant、を接続する。保存／変換／新軸生成とRead-only／可逆編集／明示Actionを別列にする。新Media Engine、二重Presentation schema、第二Test SSOTは不要。実Audio/Video再生・同期・Browser・SHO/SUI編集はNOT RUN。

### 10.36 Pattern Crosswalkは動いたがProduct OperationとCLI公開は別

[main #3195](https://github.com/hat47x/tei-platform/pull/3195)のread-only参照Toolは既存Operation Derivationを再利用し、Data ID・Capability ID・Relation targetを用いてPattern宣言とOperationを照合。基準Workspaceで操作8件・Pattern5件のうち**一致4／未結合2／衝突0**、見積CandidateではPattern6件でも一致4／未結合3。同一Dataの名前の類似、ロード順、効果の無いCommandから保存Effectを推測しない。

[main #3204](https://github.com/hat47x/tei-platform/pull/3204)の一要因対照：base=4/2/0、Effect付きCommand=5/2/0、参照先Persistence追加=5/1/0、同一Dataの第二Browse追加=3/2/1（競合時は双方不採用）、EffectあるがPersistenceなし=4/3/0。数字は順に一致／未結合／衝突。両PRの[Health #42337454670](https://github.com/hat47x/tei-platform/actions/runs/42337454670)／[Health #42339293511](https://github.com/hat47x/tei-platform/actions/runs/42339293511)はcompleted/success。これは**W3対応候補の実参照判別力**であり、正式W3採用、Function→Operation製品配線、実DB/Consumer E2Eではない。

[main #3210](https://github.com/hat47x/tei-platform/pull/3210)の実CLI観測で、CLIの受理53コマンドに対しDiscovery広告は選択された19件。operationsを含む34件は意図的に広告対象外、成功時は単段JSON、失敗時はstdout 0 byte＋stderr text。W5の**CLI入口そのもの**だけは存在するが、公開Catalogと共通エンベロープは未完。製品Skillや外部Harnessへ広告された操作として使用可能とは書かない。公開範囲の採否は既存Operation Owner、出力形式は#2824のOwnerに委ねる。研究PRでは第二Operation生成器を作らない。

### 10.37 Buildの再現性・観測Source集合・正式閉包は別資格

[main #3193](https://github.com/hat47x/tei-platform/pull/3193)は固定WSL2/Go 1.27.3/linux-amd64、隔離供給でcmd/teiを**二回独立Build**し、両方exit 0、Artifact digest一致、Attempt ID相異を観測。[Health #42339228583](https://github.com/hat47x/tei-platform/actions/runs/42339228583) completed/success。先行#3142の選択Module供給検証、#3156のBuild成功なのにSource Closure=rejectedという結果と、**いずれも同時に成り立つ**。同じdigestを出しても資格ある閉包、Stage2の実TEI再入、Owner受理は未証明。G4はUNQUALIFIED。

[Open #3217](https://github.com/hat47x/tei-platform/pull/3217)は実選択Sourceをrepo 318件・外部12 Modules／325 filesとして数え、module相対IDと専用observed digestを出す設計。Qualified digestと観測digestを別domainにし、qualificationは常にfalse。[Windows Builder #42341998886](https://github.com/hat47x/tei-platform/actions/runs/42341998886)はexact HEAD c9c86166で**completed/failure**。実試験がWindowsのTemp pathからmalformed external member nameを検出しており、単なるCI infraとは呼ばない。#3217はmain未統合。ただし後続HEAD 0788dd0fの[Windows Builder CI #42345322321](https://github.com/hat47x/tei-platform/actions/runs/42345322321)はcompleted/success（前HEAD c9c86166での失敗から進展）。Source Closure方針とG4は依然不適格。#2831/#1090のSource Closure方針を研究側で変更しない。

合格票では選択Source／実Go Toolchain／Module供給・h1／実Build／Artifact byte／2 Attempt／実選択Source観測／正式Closure Qualification／Stage2で使ったBuilder image／現在Authorityを別列とする。Go buildの2回一致だけでTEI生成物が自分を再ビルドしたとは推定しない。

### 10.38 Access記録の欠落・DSH跨Lane漏洩・長期Runの回復境界

[main #3136](https://github.com/hat47x/tei-platform/pull/3136)では一つのReference書出しToolにChainFileを実接続。さらに[main #3211](https://github.com/hat47x/tei-platform/pull/3211)のHeadPin/PinnedJournalは鎖末尾の削除をpinとの不一致で検出する局所機能で、[Health #42340826630](https://github.com/hat47x/tei-platform/actions/runs/42340826630)成功。後続の[main #3230](https://github.com/hat47x/tei-platform/pull/3230)は当該固定World書出しToolへ-access-headを配線し、logとpinの一致・末尾削除拒否を局所試験とE1関連CIで確認した。ただしpinの独立保全先、記録とpinの同時改竄耐性、全Toolへの普及は未決。鎖整合・記録完全性・記録本文の真実性・実Request帰属・現在Host Authorityを分ける。

[main #3200](https://github.com/hat47x/tei-platform/pull/3200)のDSH Tool Process実測は、read-onlyのPowerShell制約が一部APIを拒否する一方、127.0.0.1 Mockへの通信は成立するという有限観測。[Open #3216](https://github.com/hat47x/tei-platform/pull/3216)ではWindowsの別cwd／別SessionのLane A/Bが、双方とも相手Laneの**合成canaryをread-onlyで読めた**。後続の[main #3228](https://github.com/hat47x/tei-platform/pull/3228)ではworkspace-writeのRoot外書込み拒否とRoot外読取り成功を同じWindows実環境で実測した。一方、別OS User/ACL/Network Namespace隔離による読取り拒否はNOT RUN。これはH2/H3の比較適格0/4を解消せず、Worktree/cwd/CLIのread-only指定はSkill Holdout/Expected非露出を保証しない。#2515とTAI #232/#168へ既存負例として接続する。

**他チャットのOracle比較から抽出する要件候補**：Runの証拠を (1)記録の整合、(2)実Request/Operation帰属、(3)当時／現在のGrant資格、(4)計算定義・Resultの検証、(5)実Provider Effectと独立再読取、(6)業務成果とOwner Acceptanceに分ける。Read-only／拒否／結果unknownも原因・影響・次の行動が説明できるRun履歴を検討し、応答喪失時は独立Lookup/Readbackが無いまま盲目的に再送しない。Skill Manifestに依存・副作用・Owner・Token/Context予算・回復条件、RunbookにResumeのSource/Artifact/Grant再検証を候補として結ぶが、**汎用Scheduler／新Receipt ABI／新Host・自動再送機構の採用はしない**。既存#1174/#1163/#2514、TAI #514のOwnerに渡す。

**直近の実証順**：P0=Function ContextからのPattern Crosswalk・不一致・衝突を独立Expectedで固定し、その後の公開W5/Consumer実受領を別Gate化。P1=媒体時間軸・文章位置のlossをHeadlessで検証。P2=実Resource Readを一Function/一Operationへ束ねる。P3=Builder二回の再現性とClosure資格を#1090で整理。P4=H2/H3のOS identity・Network・Skill/Expected非露出を現物で検証。既存#1119 Gate不変。正式表現型DSL、Media実行、W3/W5本番接続、Stage2 Self-Build、Harness隔離、業務Owner受理はいずれもNOT RUN。


### 10.39 実ReadPlanとCompletionの証拠資格（#3175 CDE-Q05〜07の実Go API Characterizationを採り入れた再資格化）

[Draft #3175](https://github.com/hat47x/tei-platform/pull/3175) exact HEAD defbf7297fb98d444879bb1766c6af95d2a17707 の[Health #42334653630](https://github.com/hat47x/tei-platform/actions/runs/42334653630)はcompleted/success。mainのAuthorizeBrowseRead/scopedPublication/fixtureAuthorizationを使って**実ReadPlan**（top=1, select=orderId）を組んだうえで、合成RowsetをEvaluateCompletionに入力する限定試験が成立。SQL FixtureではTenant AのOrder 3=OPEN・9=READYを固定し、独立ExpectedをEvaluatorの出力から再生成しない。実PostgreSQL/実Provider経由のRowsetではなく、正式なG2/E2/E2Eとも別。

| ケース | 原典集合・Read/判定入力 | 期待する独立判定とAPIの観測 |
| --- | --- | --- |
| 正常対照 Q00 | L={A/3,A/9}、Scope・Predicate・全件性が実証 | 局所holdsは説明できる。ただしCoverage資格をCaller自己申告だけで満たしたとはみなさない |
| **Q05 同件数異集合** | L={A/3,A/9}、R=[A/3,A/3]（重複）、CallerがCoverageCompleteと誤申告 | 重複を異なる二対象と数えず、必要A/9の欠落を検出。現APIは自己申告Completeを信じ局所holdsになり得る。正しくpartialならunverified |
| **Q06 Top-1** | ReadPlan top=1のR=[A/9]を全件として渡す | 読めた1件は全母集団ではない。partial→unverified、誤申告Complete→局所holdsとなり得る |
| **Q07 非出力依存** | status=READYに必要な値をselect=orderIdではConsumerに出さない | 非表示だから常に未評価とはしない。実SQL WHERE/Operatorが非出力Predicateを評価し、その入力・Source・Pin・Scope/時点を証明するならQualification候補。今回の合成Rowset/自己申告Holdsにはその証拠がない |

**導入すべき狭い契約**：ReadPlan・実際に消費したQuery/Operator・Result Slice/row identity・重複・Predicate評価場所・Required Source/Scope・Coverage terminal・Snapshot/Current Grantと、Callerが自称したCoverage/Holdsを**別欄**で照合。ReadPlanに制限があることから、全件取得や不存在を導かない。逆に非公開Predicate列をConsumerに開示しなければ常に未確定、と誤って判定しない。実Providerの認可された非出力依存評価を許すが、その証拠が無ければunknown。新Evidence Engineは設けず、#2506（Result Slice）、#2586（Function/Query実Read）、#2514（Independent Expected）、#1163（同一Attempt）へ渡す。

**最小次実験**：同じTenant A/同じQuery原典で、Providerの実Read→Read Slice→Completionを一度だけ接続し、Q00/Q05/Q06/Q07の結果を独立Expectedに照合。業務Read/Operation/Host・Disclosureの一改訂一致が不足すればG2はUNQUALIFIED。実Readの実行・Owner意味受理はNOT RUN。

### 10.40 複数Representationの同期はSelectionでありMutationではない

前回§10.31/10.35のFunction-bound Instanceについて、[Draft #3222](https://github.com/hat47x/tei-platform/pull/3222)は同じSubjectをTable/Chart、DAGをTree/Graph、SHO文書、Media等で異なるOccurrence/Grainで投影するときの**同期**を追補した。#3222はdocs-only、固定HEAD db42626aの[Health #42342454235](https://github.com/hat47x/tei-platform/actions/runs/42342454235)はcompleted/failure（README索引試験）で、正式契約採用や実Renderer試験を意味しない。既存[Draft #3192](https://github.com/hat47x/tei-platform/pull/3192)は媒体/軸の詳細、[Draft #3225](https://github.com/hat47x/tei-platform/pull/3225)はData/Function/Businessの旧計画§13を現mainへ回収する役割で、[Health #42343050887](https://github.com/hat47x/tei-platform/actions/runs/42343050887)成功。ただし#3225の復元も設計案であり、正式受理ではない。#3179はMaster Plan受入順のOwner。研究PRは第二仕様・重複Fixtureを追加しない。

**相互に置換できないIdentity**：Semantic Subject/Relation edge Occurrence、Function Context Instance/Result Slice、UI Representation Occurrence（表1行・Tree表示Path・Chart集計点・文書span）、Interaction selection/focus/filter、Media clock、Source/World revision、Disclosure Scope/Selection epoch。同期で選択対象を対応付けても、Widget occurrenceを業務Entity IDに昇格せず、Chart集計の選択を一明細更新Targetに変換しない。旧応答A→B→A、非表示Edge、Partial Tab、別Tenantの同名Subject、SHO原文/Modelerとの編集権限の違いを一要因負例に含む。Sort/Tab/SeekをDomain Mutation/Source Saveに自動変換しない。

**優先順**：既存#2507 FP-01/02／#2497 Projection Loss／#2582 Refresh/ABAに同一FunctionのHeadless二表現を1件だけ載せ、Role/Identity/Scope/Loss/Disclosureと「同期だけ」「編集Intent候補」「実Action可能」を別列にする。実Writeは#2500/#1174へ渡す。MV-01〜08の期待値は既存#2514/#1180で管理。第二Selection Engine、独立Coordination Store、標準表現型Kindの早期固定なし。

### 10.41 エージェントに公開するOperation・自律区分・隔離の再判断

**操作種別はまだ一つの共通語彙ではない**。[main #3220](https://github.com/hat47x/tei-platform/pull/3220)の合成観測ではPlugin CapabilityKindは7種の閉じた宣言検査を持つのに、Operation kindは導出規則内の語彙で未知値を検査せず既定rankで扱う。この相違は将来PluginのKindを自動拡張可能とみなす根拠ではない。W3/W5のOwnerが語彙を規則の版で閉じるか、既存#951のCapability Contractへ接続するかを判断するまでは、外部Skillへ新しいOperation kindを安定契約として広告しない。

[Open #3231](https://github.com/hat47x/tei-platform/pull/3231)は、合成Worldでunattendedがread-onlyのlist/readからのみ導かれ、service principal/Grant/Idempotency/業務責任の有無を検査していないことを実観測。human-gatedを生成する経路も未実装。**自律区分ラベル≠現在の実行権限**であり、読み取り専用OperationがunattendedでもDSH/TAIに無審査の実データ閲覧権限を与えない。W5の公開対象/Envelope、W3のPattern Binding、実Host Grant/Disclosureを別Gateへ。

**DSH読取隔離は実測で未達**。[main #3228](https://github.com/hat47x/tei-platform/pull/3228)ではWindows workspace-writeがRoot内書込みPASS、外部への書込みUnauthorizedAccessExceptionで拒否、**外部既存合成ファイルの絶対Path・../相対Path読取りは成功**。先行[Open #3216](https://github.com/hat47x/tei-platform/pull/3216)でも別cwd/Sessionの両Laneが相手のcanaryを読めた。DSH sandbox modeの見かけ、Tool Processの制限、OS read-ACL/identity、Network Egressは別で、Skill Holdout・独立Expected・非公開Sourceにアクセスできない事実は未立証。TAI #168/232、TEI #2515へ渡し、**H2/H3比較適格は依然0/4**。信頼境界の決定に実顧客データを使わず合成canaryだけで検査する。

**Auditの局所進展**：[main #3230](https://github.com/hat47x/tei-platform/pull/3230)の固定World書出しToolは-access-logと-access-headを組で渡すとPinnedJournalを実使用し、途中・末尾削除を検出して書出し前に拒否。[固定HEAD CI #42344253330](https://github.com/hat47x/tei-platform/actions/runs/42344253330)もcompleted/success。だがPinと記録を同時に書き換える者、Pinの独立保全先、記録本文の真正性は未証明であり、TEI全Runnerの署名済みReceiptではない。新Audit Engineをここへ持ち込まない。

**Build資格の追跡**：[main #3156](https://github.com/hat47x/tei-platform/pull/3156)は隔離Buildが成功してもSource Closure=rejectedという事実をmainへ導入。[Open #3217](https://github.com/hat47x/tei-platform/pull/3217)はWindows用Member Identityの負例後、HEAD 0788dd0fの[Windows Builder #42345322321](https://github.com/hat47x/tei-platform/actions/runs/42345322321)はcompleted/success。ただしPRはmain未統合・qualified=false。#1090判断とG4 UNQUALIFIEDは不変。前回§10.37の古いHEADでのWindows failureは履歴として保持し、最新成功により「Windows未達が継続」と誤記しない。

**テスト研究**：[Draft #3223](https://github.com/hat47x/tei-platform/pull/3223)はTosca/GraphWalker/PICT/rapid/Schemathesis/Pact/Testcontainers/Toxiproxy/Playwright等を既存Scenario SSOT・独立Expected・実Provider/Browserへ対応付ける**調査設計**で、[Health #42342553727](https://github.com/hat47x/tei-platform/actions/runs/42342553727)はREADME索引試験によりfailure。ツール導入済み／TEI Test Product完成とはしない。PoCを追加するなら、まず#3175のCoverage偽証と既存#3128 Mutantを検出できる**一因子負例**を現在の#1180/#2514/ #1163へ配置し、第二Runner/検証基準を作らない。

**次の研究実証優先順**：P0=#3175の既存実ReadPlan→実Provider Result Slice→Completion/Claimの同一Attempt、P1=FunctionにBoundな二RepresentationのIdentity/Loss/Selection同期をHeadless Consumerで確認、P2=選択OperationのW3 CrosswalkとW5公開/認可の段差をOwnerの採否後に実測、P3=DSH H2/H3のOS read isolation＋Egress拒否、P4=Builder Source closure／G4を既存#2831へ。一つの成功を他Gateへ横滑りさせない。新Core/Host/Attester/Registryを増設せず、#1119のGateは変更しない。正式Owner Accepted Meaning、DB一気通貫、Media/Renderer、H2/H3比較、G4 Self-BuildはNOT RUN。
