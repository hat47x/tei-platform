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

## 10. 2026-11-06：複合図表による概念の非混線性と実Sourceへの検証移転（研究・評価計画案）

- **位置付け**：既存の技術調査計画の追補。ここでは新しい製品要求、Core型、標準API、受入Gate、Studio画面・Pipeline Runtimeを採用しない。統合済みの[トポロジー・複合図表設計](../design/TOPOLOGY_COMPOSITE_VIEWS_AND_PIPELINE_2026-10.md)と、別担当の[read-only合成Reference Draft #2881](https://github.com/hat47x/tei-platform/pull/2881)の**評価を実Sourceへ移す手順**と、Complect削減の比較方法だけを扱う。
- **確認時点**：main `2d0b09b13b16c7f2d7183dbc995d3c62208c79ef`。#2881のexact head `0c3befe7bded8b4cfc47154312254485c5e09161`に対する[専用GitHub Actions #42155639997](https://github.com/hat47x/tei-platform/actions/runs/42155639997)は`completed / success`、合成試験はPR記録上Node 24.21.0で18/18 PASS。ただしその試験の`subjectsComplete`や`readerPinned`は**fixtureの宣言値**で、実Source readerの独立証明ではない。実Studio、実TEI Self-build、実MFT/ETL、Owner Meaning/Effectの完成を示すものではない。

### 10.1 研究仮説を製品要求と区別する

**H1（正確性）**：図表・表・関連グラフの組合せは、特定の問いに必要な構造・関係role・Source版・未解決事項を示すことで、単一の図表または個別コード読解より、誤った「存在しない」の断定を減らせる可能性がある。

**H2（Complect削減）**：Source意味、Role/Operator、表示配置、Target/Plugin、実行Effectの独立性を維持した編集・解析は、これらを一体として扱う実装より、無関係な変更波及と再検証範囲を減らせる可能性がある。

**H3（共通化）**：Studioの三視点、TEI全体セルフビルド、ETL/MFTで共通利用できるのは**型付き対応と問い相対の被覆評価**であり、同じRunner/認可/受理モデルではない。少なくとも二つの異なるConsumerで、独立した業務・技術上の意味を失わずに共通部品を再利用できるか検証する。

いずれも**仮説であり、優位性は未測定**。表示がきれいになった、node数が減ったという理由だけでは支持しない。図表表現不能が確認された際は、ただちに新Core型を足すのではなく、既存構造の合成不足、Mappingの不足、Domain意味の不足、Rendererの不足のどれかを分類する。

### 10.2 合成Referenceから実Sourceへ移す際の独立期待値

単一の合成fixtureが「Source集合は閉じた」と宣言し、その同じ宣言をProjectionと期待値に再利用すると、実際のSource/Plugin欠落を見逃す。従って次段階では、**対象を限定した実Readerによる独立列挙**と**表示から独立した期待値**の両方を要求する。

| 領域・問い | 独立の列挙／期待値の候補 | 合成Projectionで検査する対象 | Claimを止める条件 |
| --- | --- | --- | --- |
| Studio: このRelationやSource上の対象はどこにあるか | #1124の実CLI/Catalog、Source Indexのscope・revision・span。M0→M1の実Consumerは別Owner | 同一qualified keyへのGraph/Table/関連表の対応・role被覆 | 異scope、欠落関係、旧DOM/旧Source、未開示対象。Inspect/Source locatorを推測しない |
| TEI Self-build: 宣言した工程だけが実行・消費されたか | #2831が所有するactual build attemptのStep・消費したSource/toolchain・各nested module・Artifact digest | 宣言Graphとの役割付き対応表、未宣言工程/入力の検出 | Traceが取れない外部Hook、source closure不足、seed trust不明。「実際に何もしていない」と断定しない |
| ETL/MFT: どのMember/転送/取込みが成立したか | #2542/#2604/#1174の実Reader、公開世代・実配送ACK/Receipt・Import Unitと独立結果 | Dataset→Transform→ArtifactSet→Transfer→Ingestionの別関係と世代差を表示 | 一部Member旧世代、ACK喪失、opaque Provider。logical planから受領/Business Effectを推定しない |

独立Expectedを作ることは新しいScenario正本を設けることではない。[#1180](https://github.com/hat47x/tei-platform/issues/1180)等の既存Ownerによる受理済みOracle/Scenarioを使い、取得対象とAttempt、Target、Artifact/Source版を固定する。Sourceに未記述の要素を探す際は、検索結果0件を自動的にclosed-worldな不在へ昇格しない。未取得・未開示の内部構造は公開画面の個数や診断からも漏らさず、[#2516](https://github.com/hat47x/tei-platform/issues/2516)の開示判断へ返す。

### 10.3 実験と観測の最小構成

実験候補は、(a) 単一図表または既存コード・CLIの読解、(b) Graph＋Table＋関連表＋被覆/unknown表示、の**同一Source・同一制約**を比較する。順序効果を避けるため課題順を入れ替える。人間と生成AIで同じ問題・期待値を使うが、AIへの情報供給量やインターフェースは記録し、完全に同等だと仮定しない。

| 最小課題 | 正解の型 | 主な測定 |
| --- | --- | --- |
| 別scopeの同名対象とRelation IDを選択 | 対象の正確なqualified key、曖昧なら拒否 | 誤Inspect/誤意味推定、正しいSourceへの到達操作数 |
| Graph上に線がないが、Readerは一部未解決 | `unknown`。完全性がある限定scopeのみ不在を許可 | 偽の不在断定率、Unknownの過剰/過少 |
| Layoutだけを変更した後の差分確認 | 意味/Build/Authorityの変更0、viewのみ差分 | 無関係な変更・再検証ファイル/工程数 |
| Role、選択Plugin、Source locatorを個別変更 | それぞれ必要な失効面だけ再評価 | 過剰失効と見逃し、追加の文脈探索量 |
| 外部ビルド工程または転送ACKが未観測 | 宣言と実現の一致は未証明、結果は保留 | 漏れた実行・未観測Effectの誤断定、追跡に要する手順 |

評価は、正答率だけでなく、**不正確な確信を持った回答（False Absence）**、誤ったSource/Ownerへの遷移、必要な情報への到達手数、変更で触れた無関係要素、実Source再読込・確認回数、AIの入力token・修正反復と、未解決を正しく維持した割合を記録する。速度・token削減のためにUnknownを勝手にAbsentへ変えたケースは成功ではない。

同時に、**表現・編集の複雑さが別の場所へ移動しただけ**かを調べる。関連表の手動保守、図表専用の第二Source、隠れたHook、特殊なDSL、無関係Plugin追加による過剰Build invalidationは、機能数が増えてもComplect削減の反証になる。必要なドメイン間結合は失敗として数えない。

### 10.4 Gateと所有境界を増やさない順序

1. **基線**：#2881の合成ReferenceとCIを限定的な「投影内整合」証拠として採用する。旧Sourceを含む新しいテストハーネスをここで作らない。
2. **初回移転**：[#1124](https://github.com/hat47x/tei-platform/issues/1124)のM0実CLI→同一Workbench Consumerの経路を先行し、実Source/Consumer対応の安全性を既存Ownerが測る。#2881の図表を実Workbenchへ先に強制投入しない。
3. **並行負例**：[#2831](https://github.com/hat47x/tei-platform/issues/2831)の宣言vs実Build消費・[#2542](https://github.com/hat47x/tei-platform/issues/2542)のArtifact一式公開・配送/取込を独立観測する。P0をStudio UIや全ETL Builder完成のHard dependencyにしない。
4. **比較と縮小**：二Consumer以上の**同種の構造不足**が検証された時点で、[#2620](https://github.com/hat47x/tei-platform/issues/2620)などが標準表現の最小拡張を検討する。認知負荷やエラー率の改善が支持されない場合は、共通新Core/APIでなく既存Pluginの限定投影へ戻す。

**結論の資格**：本節は研究・検証計画への反映であり、Referenceの実Source接続、結果の比較、認知負荷削減、Owner採用を実施したという主張ではない。現在の評価実験は**NOT RUN**。新しいCore schema、第二Source/Resolver、TEI専用Host、汎用Pipeline Runtime、製品共通の実行Authorityを採用しない。

## 11. 2026-11-06：Relationの所属、開示の非干渉、最適化後の追跡を一つの研究手順へ接続

**位置付け**：第10節の研究仮説を、別担当Draftから得られた新しい反例へ照合する**検証手順案（未採用）**。本節は#2881のReference実装、#2972の三視点公開Slice、#2516のDisclosure、#2933の最適化、#1124のStudio Consumerの実装や受入を引き取らない。確認main `3572dcedfa50c3de2903dd4d8ca4ad45ccfa1417`。

### 11.1 Relationの「所属」と「参加」を分離して追跡する

[合成read-only Reference Draft #2881](https://github.com/hat47x/tei-platform/pull/2881)では、Relationのscopeを参加端点Subjectのscopeで代用すると、cross-scope relationの存在や不在を誤って判定することが明らかになった。合成Relationを`declaring scope + relation ID`で識別し、Relation node／role edge／Relation tableの型・role・端点を照合する変更が加わった。[dedicated CI #42156918489](https://github.com/hat47x/tei-platform/actions/runs/42156918489)は **exact head `1dde0d8e5688ea5a36ff5abbb7056601195d678a`・completed/success**。当該PRの記録によればNode 24.21.0で**合成26/26件PASS**。実TEI Reader/ModelRefの製品Scope ABIが存在する証拠ではない。

研究上の含意：単一の「線」の表示だけでは、関係の**宣言主体／所属scope／参加端点とその各scope／関係のrole／有効な版**を混同し得る。これはComplectの具体例である。基本トポロジーに専用のビルド/業務Relation型を増やす前に、既存の型付き参照・n項Relation合成と、関係表の**独立した出典**がこれらを再構成できるか確かめる。

最初の**実Source接続時**は、同じrelation IDを持つ別scopeのRelation、別scopeへの端点、未取得のRelation宣言元を用意し、(a)Source Readerの実列挙、(b)Graph/Table投影、(c)独立Expectedを比較する。Relationの所属scopeが観測できなければ、端点scopeや表示位置から補完せず`unknown`とし、限定的不在を主張しない。製品Role/Relationの正式形式は[#2620](https://github.com/hat47x/tei-platform/issues/2620)/[#1120](https://github.com/hat47x/tei-platform/issues/1120)へ返す。

### 11.2 公開Sliceの観測同値は「不存在証明」と別の性質

[三要素公開Slice Draft #2972](https://github.com/hat47x/tei-platform/pull/2972)では、**事前に公開資格が与えられた同一Slice**からData/Function/Businessのread-only表現を導出し、非公開部分が異なっても公開結果を同一に保つ負例が提案された。PR記録上、ローカルNode 22.16.0で**合成8/8件PASS**。これは実Policyが正しいSliceを作ること、DOM/ARIA/Tool JSON・通信・時刻差などが非干渉であることを証明していない。

構造の問い合わせに対する`unknown`や`withheld`は、内部Sourceに未公開のRelationがあるために分岐した場合、それ自体が非公開情報の存在を示し得る。**内部向けの被覆診断**と**公開Consumerへ出す表示・返答**を同一としない。公開範囲が同一である二世界 W0/W1 に対する、観測可能な出力の同値を独立に評価する。

~~~text
For Consumer C authorized to see the same public slice in W0 and W1:
  publicInputs(C, W0) = publicInputs(C, W1)
  -> compare publicOutputs(C, W0) and publicOutputs(C, W1)
       across graph/table, coverage label, count, layout,
       navigation, diagnostics, export/tool responses
  [timing / transport / real Policy remain independent tests]
~~~

この式は製品上の無条件非干渉保証ではなく**反証可能な試験条件**である。公開Sliceに同じレコードがあることから全Sourceの不在を示す`absent`を発行せず、公開が許された問いの範囲でのみ回答する。公表可能なStatus・内訳・エラー内容は[#2516](https://github.com/hat47x/tei-platform/issues/2516)の現行Policy/Disclosureが決める。Sourceの真正な列挙と独立Oracleの区別は第10節に従う。

### 11.3 最適化後の簡潔な図表にも、説明できる来歴を要求する

[Plugin実現・最適化Draft #2933](https://github.com/hat47x/tei-platform/pull/2933)が扱うFusion、Pushdown、Batch、Cache、並列化は、**論理的な入力／出力関係と物理的な実行・観測の境界**を変え得る。論理工程A/Bを一つの物理実行へFusionしても、利用者がA/Bの存在と依存、条件、失敗・Unknown、実行されなかった工程を調べられるとは限らない。したがって最適化した実効図の「ノードが少ない」ことをComplect削減の成功としない。

実証候補は、(1) 宣言構造の関係とRole、(2) 選択Provider／最適化の根拠と制約、(3) 実際のAttempt・消費Artifact・Step trace、(4)公開可能なViewへ投影できる対応を、**同一固定入力に対して別々に検査**することである。物理1回の実行から論理2工程の独立Effectや成功を推定しない。最適化で中間成果物が消えた場合も、生成・省略の理由とどこまで観測可能かを示し、実観測不在を無副作用の証明へ変えない。Target・Provider・性能・Effectの採否は#2933/#1121/#1174へ返し、新Optimizerや第二実行Graphを作らない。

### 11.4 追加の比較実験と停止線

| 変更を加える対象 | 同一条件で期待する観測 | 失敗と判断すべき反例 |
| --- | --- | --- |
| Relation宣言scopeのみ／端点scopeのみを独立に変える | 所属と参加を別々に追跡し、Relation表・Graphの対応が変化する | 端点のscopeでRelation自身の所属を推定、別scopeの同IDを合流 |
| Private要素・Relationだけを増減し、公開Sliceは固定 | Policy済みSliceを入力とした公開Graph・表・件数・レイアウト・診断は同等 | 非公開要素数や内部Unknownの理由が公開UI/Toolへ漏れる |
| Role/Source locatorのみを変更する | 意味・Source Index・view epoch/利用資格を各依存に従って再検査 | Meaning同一だから旧Source spanや旧公開Grantをそのまま再使用 |
| Pluginを未選択で追加、または物理Fusionを有効化 | 選択済み構成は非干渉、Fusion時は論理構造との追跡が残る | 単なるNode縮約を意味保存/全実行完了と断定 |
| 一式ArtifactのMember版だけを変更 | Consumer読取世代と宣言された論理関係の差を分ける | Graph/Manifestの形だけを見て全Member同一世代・配送完了とする |

研究で優先するのは**偽の不存在、非公開情報漏洩、誤ったSource選択、無関係な変更波及**という可観測な失敗率である。認知負荷の測定では、公開Sliceが異なる比較を混ぜず、正答率・操作数・文脈探索量・修正回数を記録する。合成試験のPASSや単一図面の簡潔さが、より高い成功率を示すとは限らない。

**結論の資格**：ここで新たに実施したのは既存Draft・Issue・専用CIの静的照合と研究計画追補のみ。本節の実Source Reader、実Policy W0/W1、複合UI/DOM、論理→物理最適化の実Run、ETL／セルフビルド、生成AIと人間の比較は **NOT RUN / NOT PROVEN**。新しいCore ABI、標準ProjectionWitness API、第二Source・Policy・Graph・Host、正式受入Gateを追加しない。

## 12. 2026-11-06：設計Intent・実装Realization・検証Assessmentを混線させない複合図表（横断研究案）

- **状態**：調査・評価手順の提案。確認main `f275843b2bd35117f3bbb8d6090a59678d3c33ed`。新Core型／汎用Diff Engine／単一State Enum／Studio書込み・認可・実行権限の採用ではない。既存`10/11の「問い相対の完全性」「Relation所有scope」「公開非干渉」「実現後来歴」を、設計意図と実実装の齟齬検査に適用する。
- **他レーンのOwner**：[メタ分類 #2495](https://github.com/hat47x/tei-platform/issues/2495)、[Facetの演算束縛・公理 #2544](https://github.com/hat47x/tei-platform/issues/2544)、[ER/Key/FD #1126](https://github.com/hat47x/tei-platform/issues/1126)、[Role解釈 #2620](https://github.com/hat47x/tei-platform/issues/2620)、[認識的Annotation #958](https://github.com/hat47x/tei-platform/issues/958)、[Context/公開 #2516](https://github.com/hat47x/tei-platform/issues/2516)、[Studio Consumer #1124](https://github.com/hat47x/tei-platform/issues/1124)。[Draft #2496](https://github.com/hat47x/tei-platform/pull/2496)のメタ構造、[Draft #2778](https://github.com/hat47x/tei-platform/pull/2778)のSQL/DDL静的事実・実DB/Owner受理、[Draft #2881](https://github.com/hat47x/tei-platform/pull/2881)の複合被覆は別担当。各Schema・Corpus・判別票やOwner判断をこの文書へ複製しない。

### 12.1 三つの投影と、比較を成立させるための束縛

| 観点（概念名） | 何を表すか | 何を証明しないか |
| --- | --- | --- |
| **Intent Projection** | 宣言された／Ownerに受理された意味・許容値域・関係・必要制約とそのsource/版・受理資格 | 宣言だけで形式的な矛盾なし／実装済み／実行済みとすること |
| **Realization Projection** | 実Source、解決されたPlugin/Target、DDL/コード、実Artifact等から**実Readerが観測した**構造とその対象版 | 一部File/Targetの読取から全件存在・不存在、業務意味の正しさ、現在のAuthority |
| **Assessment Projection** | IntentとRealizationの**対応付け、差異・未解決・反例・証拠の範囲**を、関係表・差分図として導出 | 独立した第三の意味正本、未観測領域への自動否定、診断だけからのSource自動変更 |

同じqualified subjectでも、Intentの`accepted meaning revision`、Realizationの`actual source/index/toolchain/artifact revision`、Assessmentの`query scope/as-of/Policy・Consumer revision`は一律の同一digestに縮約できない。共通Graphのノードや表示座標をIdentity/Authorityへ昇格させない。Relationについても、宣言scopeと参加端点scope、関係種別／Role、導出と観測を分け、各図表の対応は受理済みSourceと独立Readerにまで追跡する。

**違いの分類**は、証拠が揃った範囲の一致候補／証明済み不一致、意図そのものの矛盾・未決、実現未観測、Source/Consumer stale、現在の公開不可を少なくとも概念上区別する。ただし列挙値・色・通知仕様をここで製品APIへ固定しない。検証不能な実態を「齟齬あり」と断定したり、Owner未受理の提案を正式Intentへ昇格させたりしない。

### 12.2 MECE・閉じた値域・強属性候補が要求する追加の前提

**Enum型の閉じた列挙と、設計上のMECEな分類、ビジネス上の状態遷移は別の主張**である。あるSourceで`Draft / Approved / Rejected`の3値が宣言されても、全ての保存済み行や外部Providerがその集合に閉じること、互いに排他的な制約の成立、全遷移の完備性は自動的には証明されない。

| Claim | Intent側で必要な宣言／証拠 | Realization側で必要な独立観測／停止線 |
| --- | --- | --- |
| 値域の閉包 | 対象Universe、バージョン、可能値、その他／不明の扱い | 実Schema CHECK、Parser/Codec、Legacy値、外部入力などの対象Closure。未読込SourceはUnknown |
| MECEの排他・網羅 | 分類述語が重ならない根拠、分類対象集合の全範囲 | 重複該当／未該当の独立負例。有限fixtureで反例なし≠一般的な公理証明 |
| 業務状態の遷移 | 状態集合と遷移規則、拒否条件、Owner受理 | `switch`やSQL CHECKだけから遷移の意味・Effectを逆推定しない |
| 「強属性」候補 | 原典に即した定義、対象粒度、制約とOwner判断が必要 | 形式的な意味が未確定のままDBのNOT NULL、Key、表示強調から強属性を自動推定しない |
| Facet・代数的法則 | 適用するCarrier、具体的なOperation binding、scope、法則のClaim | 同じFacet名、図上の同じ形、有限試験PASSから実装演算の法則成立を推定しない |

ここでは**強属性の原典上の正式定義を確認できていない**。独自に意味を発明して新Facet/Constraintとして採用せず、[#2495](https://github.com/hat47x/tei-platform/issues/2495)/[#1126](https://github.com/hat47x/tei-platform/issues/1126)の検討待ちにする。

### 12.3 Complectを検出する比較画面：一枚の差分図ではなく複合視点

例：業務上のCustomer Key、物理DBのPrimary Key、UIで行を選ぶRow Keyは**異なる役割**を持ち、値が一致する実装もあり得る。その「同じ値」を一つのKey概念に束ねると、DBの索引再編によって業務IdentityやUI選択が意図せず変わる可能性を見落とす。IntentのKey/FD図、実DB Schema表、両者の型付き対応関係と出典、差分・不明・変更影響の関連グラフを**同一版束縛**で提供する方向を試す。

類似の反例：Data RoleとBusiness Roleが同一Sourceにあって一方だけ未解決、同一値域上で異なる演算束縛を持つ複数Facet、宣言Graphの二工程を物理Targetが一工程へFusionした場合、Relationの宣言scopeが端点scopeと異なる場合。このような**意味の分割点**に対して、「この変更はどの範囲へ波及すべきか」を先に宣言し、Source・投影・実Artifact/Effectで独立に照合する。

図表間リンクには、(a)同一主体の複数Occurrence、(b)別主体の意味Relation、(c)Intent→Realizationの追跡、(d)View間Navigation-onlyを区別する。差分表の行をSource自体として編集することは認めず、可能な場合のみ既存[#267](https://github.com/hat47x/tei-platform/issues/267)のtyped candidate→Preview→Owner受理へ送る。可逆性・現在資格がない図はread-only。

### 12.4 次の小さい反証と成功／撤回判定

既存Ownerの受入へ**テスト候補を受け渡す研究手順**であり、新Fixture IDや別の実行Gateは作らない。

| 独立変更・反例 | 合格へ必要な区別 | 禁止する誤判定 |
| --- | --- | --- |
| IntentがClosed 3値なのに、実Sourceの選択済みDecoderに第4値がある | 宣言と観測対象のscope/版を揃えて違反候補を示す。実Decoder未読ならUnknown | enum宣言だけで実Target全体が完全と主張 |
| 同じDataに異なるFacet/演算束縛を追加 | Carrier同一と演算/法則の束縛を分け、使用箇所単位で判定 | 同じFacet名だけで法則／変換が成立と扱う |
| DB PKのみ変更し、Business KeyとUI Row Keyは不変 | 対応表の役割・Source/Target再確認、無関係なMeaning不変 | 物理Key変更を業務Identity変更と同一視 |
| Public Sliceを変えずPrivate Relationだけを追加 | 公開された図・表・件数・診断・navigationが不変か | 内部AssessmentのUnknown内訳で存在漏洩 |
| 最適化で二つの論理Operationを一つの物理実行へFusion | 論理関係と実行Traceを別投影し、観測できないEffectを未証明とする | 物理Nodeの減少から論理Operationの不存在を推定 |

研究上の評価指標は、第10節の偽不存在・誤選択・読解操作数・AI token/再確認回数に加え、**意図的な矛盾の検出率、未知を適切に保留した割合、独立変更の非意図的波及、必要な比較関係を理解する時間**とする。同じ対象・固定Source/Expectedを使い、単一図・コード読解との対照を記録する。

**撤回条件**：新しい三重Graph SSOT、分類の固定Schema、意味推論の巨大Engine、隠れた手動対応表を要し、局所変更・理解・検証の総コストを増やすなら、標準Coreへの昇格を中止して領域Plugin/限定Viewへ戻す。構造上の不足が独立した複数領域で反復する場合に限り、[#2495](https://github.com/hat47x/tei-platform/issues/2495)/[#2620](https://github.com/hat47x/tei-platform/issues/2620)で最小Primitive/Role契約の必要性を検討する。

**証拠・未保証**：本節は現行IssueとDraftの静的照合を基にした研究案。実Intent↔Reader↔Artifact↔Consumer結線、SQL/実DB、形式証明、運用中Policy、Owner承認、複合Studio画面、人/AIの認知負荷比較は **NOT RUN / NOT PROVEN**。TEI独自共通Hostの必須化、第二Meaning SSOT、第二Resolver、第二Policy/Authorityは提案しない。

## 13. 2026-11-07：複合Projectionの出典資格・非単射性・Complectの変化行列（研究計画への反映）

- **状態・範囲**：本節は既存`10〜12の横断研究を、異なる**実コードの反例**から評価可能な課題へ絞るための計画追補。照合main `680646b8629ce7af5b9bb0ba7db3529151b63a4c`。新Core型・汎用Projection ABI・Reverse Patch Engine・Diff/Policy Engine・正式受入Gateを採用しない。
- **既存Owner**：[TEIの写像忠実性 #2497](https://github.com/hat47x/tei-platform/issues/2497)、[複合read-only Reference Draft #2881](https://github.com/hat47x/tei-platform/pull/2881)、[Studio #1124](https://github.com/hat47x/tei-platform/issues/1124)、[Role #2620](https://github.com/hat47x/tei-platform/issues/2620)、[現在のDisclosure #2516](https://github.com/hat47x/tei-platform/issues/2516)、[Source Acceptance #267](https://github.com/hat47x/tei-platform/issues/267)、[TEA三視点設計](../design/THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md)、[Plugin最適化設計](../design/PLUGIN_REALIZATION_AND_OPTIMIZATION_REQUIREMENTS_2026-11.md)。[SUI #3396](https://github.com/hat47x/sui-sensemaking/pull/3396)は**別Productの独立Consumer証拠**であり、TEI製品API採用の根拠に自動転用しない。

### 13.1 Identityの起源と、表示上のScopeを分離する

[Draft #2881](https://github.com/hat47x/tei-platform/pull/2881)では、既存`tei trace` CLIを二つのcaller指定scopeで実行すると、Scope以外のWorld/Source・直接Edgeが同じままProjection Keyだけ異なり得ることを観測した。`derivedbridge.FromIndex`は投影Keyにcaller指定scopeを付与するため、**出力に`scope`という文字列があることは、Source自身がそのscopeでRelation/Subjectを宣言した証拠にならない**。同Draftの最終exact head `79c16e057d9cd97cd579d8b8fc432f26671beacd`について[専用CI #42207187724](https://github.com/hat47x/tei-platform/actions/runs/42207187724)は`completed/success`。PRには実Go1.26.10 CLI二回→Node24.21.0比較PASS、合成31/31、CLI境界7/7と記録。ただし真のRelation ID・全Role・Source/Provider closure、実Workbench/Policyは未証明。

評価上は少なくとも次を混同しない：**Source/Ownerが宣言した主体scope**、**Relationのdeclaring scope**、**参加端点のscope**、**CLI/Projectionが受け取るcaller選択scope**、**表示occurrenceのview ID/epoch**、**ConsumerのDisclosure scope**。同じ文字列でも、その生成元・時点・解釈契約が異なる。CLIのscopeを変えたときに出力Keyが変わるなら、その変化をSource上のidentity変更と診断しない。

**最初の実証課題**：同じ実Sourceをcaller scope A/Bで投影し、変化するProjection Keyと不変なSource/Worldを独立に比較。Source側の真正な宣言scopeが得られないなら、対応表には`caller-supplied`等の出典区分を持つ**評価用注記**を残し、Source-owned Relation Key、唯一locator、bounded absence、現在の書込資格を勝手に発行しない。正式な出典enumやSourceIndex schemaの新設は#1120/#2497/#1124へ戻す。

### 13.2 同じ図表から元の構造は一般には復元できない

[SUI Draft #3396](https://github.com/hat47x/sui-sensemaking/pull/3396)の**実Product `buildContextProjection()`**に対するテストでは、Cardの`meta.source`だけを変えた二世界、Island所属とCanvas座標を変えた二世界が、**公開出力全体で同じ**になり得る。一方、公開本文やレビュー済みIsland題名の変更では出力が変化する正対照もある。[専用CI #42345896949](https://github.com/hat47x/sui-sensemaking/actions/runs/42345896949)はexact head `a7893f9c4114d140dc19c7ff2f565693c5b1b29b`、`completed/success`、PR記録では該当3/3件PASS。SUIの実逆Patch拒否、OKF/SHO Exporter、TEI共通Renderer/Readerの実装を示すものではない。

これはProjection `f : Source → View` が**非単射**になり得る具体的証拠であり、`f(S1)=f(S2)` から `S1=S2` は導けない。従って図表のbyte/digest一致、表示ノード数の一致、Graph/Table同形は、それだけではSource同一・Relation/Source anchor一致・意味上の同値性・安全な逆Writeを意味しない。

同じDiagram/Textへ集約された複数Sourceの**編集逆写像**は、対象範囲と正本Ownerを固定したうえで、一意な編集対象と許容されるSource変換が**別の信頼可能な証拠**で定まる場合のみ候補化する。非可逆な集計/折畳み/引用は既定read-only。正本に対する変更は既存#267でtyped candidate→Preview→明示受理とし、図の出力からSourceを捏造しない。これは#2497の写像保存・逆変換Ownerへ渡す**横断受入圧力**であり、独自Lens/Writer/APIの採用ではない。

### 13.3 複合図表の「結合可能性」と「不存在」を分ける

図Aと表Bが同じ`qualified key`を表示しても、そのkeyがcaller由来の表示用Scopeを含む場合、**両Viewが同じSource-owned対象を指す保証にはならない**。さらにSource revision/Source Index/Role pin/選択Provider/Public Sliceの資格が異なる場合、同一表示名や同一画面HashでJoinしてはいけない。

- **Cross-view correspondence**：同一Source anchorを保持できる範囲と、caller/display由来の補助対応を区別する。後者はNavigation Candidateであり、Sourceの確定Identityではない。
- **Mapping loss**：投影時に元のRelation role、所属、出典、順序、Card/Islandの情報が落ちていれば、その情報に関する問いは`unknown`。図表を複数合成しても捨てた情報は自動復元されない。
- **Visibility**：非公開情報は内部の診断には必要でも、その存在や件数を公開View/Graph/AI Toolへ漏らさない。W0/W1の観測同値と、内部Reader完全性は別。
- **Absence**：Source-ownedな限定された対象・関係、必要なPredicateとSource/Reader closureが証明できない`tei trace`のdirect edgesや公開Sliceから、Source上の関係不存在を主張しない。空表示は無いことの証明ではない。

これにより「図に無ければ無い」を拒否するだけでなく、**どの問いについて、どの証拠を追加すれば限定的不在・同一性・編集可能性を判定できるか**を説明できる設計が目標になる。

### 13.4 Complectを発見する直交変更行列：単純化の実測方法

単純さの評価を、ノード数や抽象の数ではなく、**意図的に一つの概念軸を変えた場合、どの証拠/投影/実現が適切に変化・失効するか**で測る。独立ExpectedはDomain/Ownerの意味規則から先に記述し、単一のProjectionに期待値を自己申告させない。

| 入力側の単一変化 | 必然的な変化または再資格化 | 不自然な波及・誤推論の例 |
| --- | --- | --- |
| View座標のみ変更 | Layout revision、必要なView描画 | Meaning、Source identity、Build Intent、Effectまで改変 |
| Caller側の`trace -scope`値だけ変更 | Projection Key/selector束縛 | Sourceの宣言scopeやRelation identityが変わったと断言 |
| Source位置のみ移動（意味値同一） | Source Index・jump/引用の現在資格 | 意味自体を変更と断言、逆に旧locatorを無条件再利用 |
| Provider選択またはFusionのみ変更 | Target realization・実工程/Trace・必要なArtifact再資格化 | Logical Role/Relationを暗黙削除、物理1回を論理全工程成功と宣言 |
| 非公開の構造のみ変更、Public Slice同一 | 内部Assessment、必要なPolicy再評価 | 公開Node数・診断・Tool JSONにPrivate存在の差分 |
| 元構造のIsland所属等を変更、公開文書同一 | Source構造、正本の出典・必要なReview | 出力Hash一致で元の構造・編集対象の同一性を確定 |

比較では、(a)**意味・構造の値が変化した**、(b)**値は不変だが再資格化が必要**、(c)**閲覧資格が変化した**、(d)**未観測のため不明**を別列にする。期待された依存影響をComplect失敗とみなさず、独立なはずの部分への波及を測る。期待した変化の欠落も失敗（依存の見落とし）として数える。二値の「変更有無」だけで判断しない。

人間・生成AIで、誤ったSource選択、偽の不存在/同一性の宣言、必要なSourceへ到達する操作、未知の保持、余計な影響修正、Context Token/再確認・試行数を測る。表示項目を減らすだけで正答が悪化すれば、Complect解消とは認めない。`#2497`のStructure-preserving Mapping、`#2018`の生産性比較を既存Ownerとする。

### 13.5 優先順・撤回条件

1. **P0 実Source由来の照合**：#2881の実`tei trace`二scope・直接Edgeの証拠を、#1124/#2497の既存Reader/Consumer境界へ受け渡す。Scopeの起源、Source owned/not-proven、direct edgeと独立Relationを分ける。新たなProduct Source Readerは本研究で作らない。
2. **P1 非単射の横断比較**：SUI #3396の実Projection反例とTEI #2881のcaller Scope変化を、同一の「失われた次元・失効する操作」評価票へ並べる。SUI/SHOの機能・Policy・逆Patchは各Ownerの実証に残す。
3. **P2 変更局所性を実測**：同じ業務変更に対し、コード/単一図と複合図表の独立Expectedに基づく変化行列を比較し、意味・Locator・Public Slice・Artifact/Effectのどの値/資格が変わったかを記録。
4. **P3 最小構造拡張の是非**：異質な二Consumer以上で**同一の構造的不足**が反復し、既存の型付き関係/Projection合成では正しく表せず、かつ認知負荷と変更費用の改善が確認できた場合のみ、#2495/#2497/#2620の既存Ownerが標準Contractの拡充を検討。

**撤回条件**：Source-owned Identityの証明なしに表示スコープを正本化する、非単射の逆写像を勝手に書込へつなぐ、別々のGraph SSOTを増殖させる、非公開の内部Unknownを表示して漏洩させる、メタ生成の隠れ依存で調査・変更コストが増える場合は、共通API導入を撤回してread-only/Domain Pluginへ限定する。

**実証資格**：#2881の実CLI+Node限定CIとSUI #3396の実Product projectionの独立限定CIは確認した。**両者の同一仕様下での統合試験、TEIの正式Relation Sourceアンカー/全Role closure、実Workbench・Policy、Source編集・Owner受理、実Build/ETL、認知負荷の比較は NOT RUN / NOT PROVEN**。ドキュメントの成熟と実製品の資格を混同しない。

## 14. 2026-11-08：複合図表を跨ぐ関係の被覆と、サブシステム分割の反証可能な境界（研究計画）

- **状態**：研究・受入候補。基線main `c2b8062b7737816e33f85f80ac5f3956f5494f65`。既存`10〜13のQuery相対完全性・Intent/Realization/Assessment・非単射性・Complectの変化局所性を、**図表の境界にまたがる関係**に適用する。新しいSubsystem/Hypergraph Core型、専用Partitioner、第二Graph/Owner/Policy、Studio書込みUIの採用ではない。
- **責務の境界**：[サブシステム分割 #3250](https://github.com/hat47x/tei-platform/issues/3250)／[その設計Draft #3255](https://github.com/hat47x/tei-platform/pull/3255)はCRUD結合度・責務・分割候補を所有。[複合図表の合成Reference #2881](https://github.com/hat47x/tei-platform/pull/2881)はread-only Projection内被覆と実CLI直接Edgeの資格を所有。[標準構造の写像 #2497](https://github.com/hat47x/tei-platform/issues/2497)、[Studio #1124](https://github.com/hat47x/tei-platform/issues/1124)、[Source受理 #267](https://github.com/hat47x/tei-platform/issues/267)はそれぞれ既存Owner。本節は**図表間の関係欠落を検出する評価手順だけ**を提案し、これらのコードや既存Fixture IDを変更しない。

### 14.1 「図表への所属」「論理的な分割」「Effectの責任主体」は別の関係

- **View occurrence**：同じqualified subjectは複数の図・表・関連表に現れてよい。Viewへの所属は多対多であり、同一図形や表示IDを意味正本として固定しない。
- **Subsystem allocation candidate**：業務上の所属や責務を割り振る分割提案。未割当・Shared Read・外部System・例外を強制的に唯一の所属へ潰さない。候補の変更は、Source上のCRUD/Relationや業務Effectを黙って書き換えない。
- **Relation declaration**：Relation自身の宣言scopeと、参加する各roleのsubject scopeは異なり得る。境界越え関係を端点scopeから別Relation IDとして再生成しない。
- **Write／Execution Authority**：論理的なSubsystemの所属と、現時点のCommand/Provider/Transaction/Effect Ownerは独立する。越境Edgeを見つけても書込みを許可した証拠ではない。

結果として、複合図表を定義する規則と、分割の評価規則と、実行/受理資格は**互いに異なるProjection/Claim**であり、一つの`owner`フィールドへ束ねない。

### 14.2 Boundary relation tableは「切断辺の一覧」ではなく役割付きの照合表

評価用に固定したSource snapshot・scope・query `Q` と、受理済みでない場合は**候補であることを明示した**分割 `P` を考える。Sourceにある型付きRelation `r` の参加roleと端点が異なるSubsystem候補に属するなら、そのRelationをboundary candidateとして扱う。これは数学上のグラフ分割の正式なCore算法ではなく、**多項関係を二項Edgeへ潰さない**監査の考え方である。

~~~text
Boundary audit candidate for bounded Q and partition proposal P
  Relation: declaring scope + relation identity + type + source revision
  Participants: ordered (role, qualified subject, proposed allocation)
  View occurrences: (view id, local id, projection revision)
  Origin: Source-declared / derived candidate / caller-supplied / unknown
  Coverage: relevant Source and roles covered?  omitted?  opaque?
  Semantics: read / direct CUD / effective CUD / execution dependence / unknown
  Qualification: accepted meaning? source reader evidence? current disclosure?
~~~

この項目は**研究用の記入欄であり製品API案ではない**。同じRelationが複数図に出るときはRelationを複製せず、各表示Occurrenceを同じ根拠付きRelationへ参照させる。cross-view Navigation-only、業務上のRelation、Buildのdata dependency、Effectの実観測は必ず別の意味を持つ。

**例（仮想モデル）**：受注Function `F` とOrder Data `O` はSales候補、Customer Data `C` はCRM候補。Integration scopeで宣言された三項Relation `R` が`(caller:F, input:O, reference:C)`を持つ。Sales図とCRM図と関係表を組み合わせるとき、`R` の宣言scopeはIntegrationのまま、各参加roleとView occurrenceを結ぶ。`R` を「Sales→CRMの読み取り1件」に縮約してRoleやSourceを消さない。さらに`F`を別Subsystemへ表示上だけ移しても、既存の業務RelationをSourceから削除しない。

### 14.3 相互に独立した二方向の照合と、不在Claimの停止線

| 検査方向 | 確かめること | 偽の合格例 |
| --- | --- | --- |
| **Source→Boundary tableの被覆** | QのSource/Relation/Roleを独立列挙し、異所属関係が対応表・関連グラフに欠落なく表現されたか | 表から生成した同じ表を「全Sourceの期待値」と再利用 |
| **Boundary table→Sourceの帰属** | 各行が正しいRelation ID・宣言scope・role/端点・Source版へ追跡できるか | 表示Keyやcallerの`trace -scope`をSourceの宣言scopeへ昇格 |
| **View→Bounded relationの対応** | 各図に表示された端点・非表示要素・関連表行が同一の有効snapshotに属するか | 図AはR10、表BはR11なのに同一時点の依存と表示 |
| **Change→Impactの検査** | Pのみの変更なら内外分類と表示だけを再評価し、業務Relation自体は不変か | 越境CUDを減らすために元のCUD意味を隠して削除 |
| **Current public Consumer** | 許可済みSliceと現在の開示資格で、境界本数/未解決理由/外部参照が表示可能か | Private関係の存在を公開図の件数や診断から漏洩 |

Source側のRelation全件列挙や`P`の所属が不明な場合、**境界の既知件数=0** と **越境Relationが存在しない** を同一視しない。動的SQL/Trigger/外部Provider/未選択Pluginなど、今回のQueryが閉じていない箇所には`unknown`を残す。図表と表が互いに一致することは、独立Source Reader、Current Policy、実行時のEffect閉包を証明しない。

### 14.4 Complect削減の比較実験：内部の見通しと境界の説明責任を両立する

比較対象は (A)巨大な一枚の図、(B)図を分割しただけで関係表なし、(C)同じSourceの複数図＋根拠付き関連表・関連グラフ。**同じ対象Snapshotと独立Expected、同じ公開資格**で比較し、図の見栄えだけで優劣を付けない。

| 反例・対照 | 守るべき性質 | 新しく測る失敗 |
| --- | --- | --- |
| 表示を二つのSubsystem図へ分割し、三項Relationは図間を横断 | 役割/Relation所有scope/参加端点を保持し、どの図からも追跡できる | 欠落したBoundary role、間違った所属・Relation ID |
| Subsystem候補Pだけを変更し、Source/Function/CRUDは同一 | 必要なcut分類だけ変化し、Source意味・Effectを勝手に変えない | 不要な変更波及、誤った`CUD=0`断定 |
| Sourceに未知のTriggerがあり、既知の越境Writeは0 | `known 0 + unresolved`として表現し、実Effectの不存在を宣言しない | 偽の不存在、実装・監視されない依存 |
| 異なる内部Worldを同じ公開Diagramへ投影 | 公開資格内の情報だけで判断し、元構造や自動逆編集を捏造しない | 非単射なProjectionの誤逆写像、非公開情報漏洩 |
| Build／ETLの論理工程を物理的に集約 | 論理間Relation/データ依存と物理Attempt/Receiptを別に保持 | 「表示から消えた工程＝実行なし」の誤断定 |

人間／AIの評価では、**境界をまたぐ変更影響を発見するまでの時間、必要参照数、欠落したRole、偽の不存在・越境ゼロ断定、誤Source選択、不要なSource変更／再確認回数**を測る。元Sourceの情報取得・関係表生成・レビュー・CIの費用も合算する。単一図より判断が遅くなり誤りが増える場合は、共通複合図表の抽象化を撤回またはread-only限定へ縮小する。実験の採点と生産性評価は[#2018](https://github.com/hat47x/tei-platform/issues/2018)へ受け渡す。

**実証順**：P0=既存#2881 CLIで取得できる`direct-edge-candidates`と、Sourceから未取得の独立Relation/Effectを区別して評価票を作成。P1=#3255のSubsystem候補に同一Source・Role対応表を適用し、known 0とunknownの負例を独立Expectedで確認。P2=必要に応じ#1124のGraph/Table ConsumerでSnapshot/Disclosureの整合を評価。P3=二Consumer以上で同種の構造的不足が反復した場合のみ#2495/#2497/#2620のOwnerが標準契約の最小追加を検討。**#3255の分割最適化、#2881のReference実装、#1132のCUD checker、#267の受理を代替・ブロックしない**。

**証拠と停止線**：本節は既存Owner/PR/CIとSourceを使った**設計・評価手順の追補のみ**。Boundary table生成の製品実装、独立Source全文closure、実Studio UI、実CRUD/Transaction/ETL/Self-build、Owner受理、人/AIの比較試験は **NOT RUN / NOT PROVEN**。新共通Host、Policy Engine、Schemaや受理Gatewayを設けない。

## 15. 2026-11-08：複合図表のEvidenceを別Claimへ昇格させないための横断評価手順

- **状態**：本節は`10〜14の独立Reader・Bounded Query・境界Relation監査に、2026-11-08時点の現行mainと別担当Referenceの進展を接続した**研究・受入候補**。固定参照main `768d8ef3b3396a2ce5ccc196f63673683ab177ce`。新しい製品`Proof Registry`、Core Claim Enum、汎用Query DSL、第二のSource/Graph/Authority/Runnerを追加・採用しない。
- **担当と証拠の位置**：[現行main由来の実CLI Trace Draft #3321](https://github.com/hat47x/tei-platform/pull/3321)は実`tei trace`→別言語read-only Consumerの出典・SourceNavigation整合をOwnerとして実装。[複合Graph/Table Draft #2881](https://github.com/hat47x/tei-platform/pull/2881)はsynthetic多項Relation／限定不在候補のOwner。[Role/Type接合Draft #3320](https://github.com/hat47x/tei-platform/pull/3320)は既存Validatorの型照合。[分割Draft #3255](https://github.com/hat47x/tei-platform/pull/3255)は固定Owner下の越境更新下界。[移行／再実現Draft #3252](https://github.com/hat47x/tei-platform/pull/3252)は別Target/Consumerの再資格化。この研究節はそれらを再実装せず、**そのEvidenceから別の結論を導いてよい条件**だけを扱う。

### 15.1 Claimは一列の成熟度ではなく、問いに相対的な必要条件の組合せ

実`tei trace`が二本のdirect edgeを返し、SourceNavigation内の`Trace.subject`、`Navigation.Key`、`SourceIndex.Reference subject`が一致した場合、**そのCLI Packetで特定された直接参照候補が観測できる**という限定Claimを支える。しかし、それだけで独立Relationの宣言ID／declaring scope／多項role／全Source closure／関係が存在しない証拠にはならない。

同様にRole語彙の認識（V0）、既存TypeRefとの局所適合（V1）、実WorkspaceでのBinderの使用（V2）、異なるTarget/実Writerが意味・制約を保持すること（V3）は、それぞれ**別の証拠**。V1を得たからV3が成立するという線形な資格階段を仮定しない。実Targetの資格、Source Closure、Current Disclosure、Owner Acceptanceは互いに独立した軸である。

| 問い／利用したいClaim | 先行する限定証拠と実際の上限 | 昇格に追加で必要な独立証拠 |
| --- | --- | --- |
| **Q1 このSubjectに直接参照候補があるか** | #3321の実CLI Packet、Subject/SourceNavigation/WorkspaceIdentityの整合。観測されたdirect edgeの資格 | 実Readerの対象・scope起源・同一Attempt/版。現在のGrantとPublic Sliceが必要なら別検証 |
| **Q2 複数図の間に特定Roleを持つRelationがあるか** | Q1の`Edge.Relation`はdirect-reference labelであり、独立した意味Relationと同一ではない | 受理済みRelation ID／宣言scope／多項role／端点のSource anchor、同一版での図・表・関連表の対応 |
| **Q3 越境Direct CUDが0か／分割変更で消せるか** | #3255の条件付き下界は固定Owner groupと既知direct CUD、独立列挙等の前提でのみ評価可能 | 対象Function/Data/Owner/Relationの閉包と正本Source、P候補の所属、unknownの計数、業務責務と制約。分割のみと業務修復を分離 |
| **Q4 表示されない処理や効果は本当に無いか** | `tei trace`の0 edge、合成`subjectsComplete`、Buildの宣言Graphの図だけでは不足 | 対象相対のRequired/Selected/Consumed/Observed、実Process/Artifact/Provider/Importerの独立Traceと不透明領域の扱い。実Effect・無副作用は別Owner |
| **Q5 この表示を現在編集・配布してよいか** | 意味／図形／出力digestが一致しても、非単射ProjectionではSource逆写像不可の場合がある | 一意なSource anchor、編集逆変換の資格、現在のSource/Grant/Disclosure、typed Candidate→Preview→#267のAcceptance。配布・実Effectは独立Gate |

上記Q1〜Q5は**研究用の問いの番号**であって、実行するProduct Test IDでも、完全性の全順序でもない。あるClaimは別のClaimより強いとは限らず、必要条件が違う。たとえばSourceの宣言scopeを確証してもCurrent Grantを得たことにならず、GrantがあってもSource/Roleの全件列挙は得られない。

### 15.2 複合図表に表示するEvidence Joinの成立条件

異なる図表や試験を合成するときの説明用チェック欄（**新製品Schemaではない**）：

~~~text
Requested claim:
  exact subject/relation and role query; bounded scope; target consumer
Source bindings:
  source-owned vs caller/view-derived identity; revision; source-index version
Independent evidence:
  owning reader/validator/runner; attempt; actual inputs and omissions
Mapping:
  query predicates preserved; lost roles/occurrences/ordering explicitly unknown
Applicability:
  current accepted meaning, policy/disclosure, target/provider/lock as required
Conclusion:
  supported only for the bounded claim / contradicted / unknown / withheld
~~~

**証拠の単純な加算は禁止**。実CLIのdirect edgeと合成Graph/Tableのrole付きRelationは、片方に存在しないSource-owned Relation IDが他方へ伝播するわけではない。二つの成功したCIが異なるSource/Attempt/Policyを検証しているなら、同じ現在の結論へJoinできない。公開`withheld`は内部Unknownの理由を明かすための表示欄ではない。

**版の戻り（ABA）**：Sourceの内容やProjection digestがA→B→Aで元に戻っても、以前のReview/Grant/実行Attempt/現在のSource Index・Source Locator資格を自動復活させない。過去に正しかったClaimは履歴として保存し、現在の利用資格は対象Ownerの契約で再確認する。新しいglobal epoch／Proof Storeを本節で導入しない。

### 15.3 現行の限定CIと、未成立の統合証拠

- #3321 exact head `4dfdaf75145059859593d5f320ce43025e9736b0`の[専用Actions #42385227799](https://github.com/hat47x/tei-platform/actions/runs/42385227799)は**completed/success**。別担当PRの記録ではNode 24.21.0合成14/14と実Go CLI二scope→Node Consumer比較PASS。ただし`WorkspaceIdentity`はManifestのロード時観測であり、Source/Plugin Closure・Owner受理・実Studio UIには昇格しない。
- #3320 exact head `ec575dc9e7e8cc3e0663ed4c563b986a3a0a4d9b`の[Repository Health #42393505853](https://github.com/hat47x/tei-platform/actions/runs/42393505853)は照合時**completed/failure**、[依存検査 #42393505949](https://github.com/hat47x/tei-platform/actions/runs/42393505949)は**completed/success**。異なるチェックの結果を一つのPASSに丸めず、失敗原因・新head追随・Targetでの意味保存は別Ownerが再確認する。
- #3255の条件付き越境更新下界や[#3252の移行Readiness](https://github.com/hat47x/tei-platform/pull/3252)は**設計／評価票**であり、実Source閉包や実Target/Effectの資格ではない。[#3319](https://github.com/hat47x/tei-platform/pull/3319)が統合した境界関係監査は研究手順であり、実Runner/Relation Readerの実装ではない。

### 15.4 次の最小「誤ったClaim昇格」の反証

| 組合せる出力と変更 | 独立Expectedと正しい結論 | 誤った昇格 |
| --- | --- | --- |
| #3321で直接edgeが2件観測され、独立多項RelationのReaderがない | `direct reference candidates present`。Relation ID/role/declaring scopeは未証明 | 二本のedgeを業務上の完全な二項Relationと呼ぶ |
| #3320の`std.decimal`適合が確認できるが外部Writerが不明 | Role/TypeRef局所適合は履歴証拠。Targetの精度・NULL・送出値域はunknown | 全Targetの意味保存とMECE・閉域性を合格 |
| 固定Owner下の既知越境Direct CUD下界が正、Trigger/動的SQLが未取得 | 分割だけで既知境界を0にできない条件を示す。実Effect総数はunknown | CUD下界から全Write数/最適Partition/業務Ownerを自動決定 |
| 同じ公開図へ異なるPrivate WorldやSource構造が写る | Public viewは同等。逆WriteのSource資格は別照合 | 表示digest一致でSource同一、Private要素不存在を確定 |
| 同Source内容A→B→A、PolicyまたはSource Index改版 | historicalの比較は可、現在のlocator/Grant/Owner受理は再資格化 | 同じHashで過去の利用/編集・実行資格を再生 |

**測定対象**：false-qualified（証拠より強い断定）、false-absence、適切なunknown/withheld、異scope・旧版の誤Join、不要な再検証、必要な再資格化の見逃し、独立Expectedに到達する時間・操作数を記録する。見た目を短くするためにClaimの条件を省略した場合はComplect改善とは認めない。正式な比較研究は[#2018](https://github.com/hat47x/tei-platform/issues/2018)へ渡す。

**次の実証順**：既存#3321の実CLIのQ1だけを固定 → #2881の合成Q2は実Relation Readerが得られるまで別表示 → #3255の固定Owner小例Q3へ同一Source/独立Expectedを適用 → #2831の実Build Traceと#2542のMFT実ReceiptのQ4は各Ownerへ残す → #1124/#2497/#267でSource/ビュー/AcceptanceのQ5へ進む。これらを一つの巨大E2E試験のHard Dependencyにしない。

**成果資格**：今回追加したのは**既存Source・Issue/PR・CIの静的照合と研究計画のみ**。新Go/Node/GUI、真正なRelation Reader、Subsystem分割の実試験、Build/ETL/実Provider、Policy/Ownerの現在受理、人/AIの比較は **NOT RUN / NOT PROVEN**。既存Ownerの成果を上書きせず、複数試験の成功を製品E2EのPASSへ合算しない。
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

**最小次実験**：同じTenant A/同じQuery原典で、Providerの実Read→Read Slice→Completionを一度だけ接続し、Q00/Q05/Q06/Q07の結果を独立Expectedに照合。業務Read/Operation/Host・Disclosureの一改訂一致が不足すればG2はUNQUALIFIED。この#3175時点の実Provider ReadはNOT RUN。後続#3238で隔離PostgreSQLの限定実Readが成立したことは§10.42に記す。Product Function→Provider→ConsumerとOwner意味受理はなおNOT RUN。

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

### 10.42 隔離PostgreSQLの実Readと非出力Predicate：Reference限定の新しい成立点（2026-11-07）

前§10.39のDraft #3175は実ReadPlanと合成RowsのCharacterizationであった。後続[Draft #3238](https://github.com/hat47x/tei-platform/pull/3238)はこれを最新mainへ移植し、固定HEAD d3b67d5809478fad828052ce01bdafe576b2f934 の[隔離PostgreSQL CI #42351796330](https://github.com/hat47x/tei-platform/actions/runs/42351796330)と[Repository Health #42351796383](https://github.com/hat47x/tei-platform/actions/runs/42351796383)が**ともにcompleted/success**。既存の検証専用DB guardを満たして実Readし、要求SubtestのSKIPを禁止するCIで、Tenant Aの独立Fixture L={3,9}からTOP2=[[3],[9]]、TOP1=[[3]]を取得。TOP1のResult Sliceはpartial→局所unverified、Callerが根拠なくCoverageCompleteを自己申告すると局所holdsになり得るというEvaluator境界も保持する。

さらに、Consumerへ投影しない ordered_at をSQL WHEREで実評価する正例を追加。閾値2026-10-02を満たすID9だけが返ることを独立Expectedで検出した。**出力列がないことと、実OperatorがPredicateを評価しなかったことは同義ではない**。逆に出力IDの差から隠した時刻・状態の閾値充足を推知できるため、**Predicate executionの許可**と**結果行の存在・件数・診断のDisclosure許可**を別の要求として扱う。Consumer向け列を落とすだけで秘密非干渉が成立したとはしない。

| 検証段階 | 今回の資格 | 未証明 |
| --- | --- | --- |
| 固定SQL Fixture、実ReadPlan、実PostgreSQL Reader、局所Completion | 選択したReference経路の実Rowsと独立Expected、TOP-N/非出力Predicateの対照がCI PASS | 任意Scope/任意Queryの全件Coverage、別Tenantの真正権限 |
| Function→Query選択→Provider Result Slice | まだ製品の同一Attemptで接続していない | 実Function、必要Role/Dependency Closure、Owner Meaning |
| Consumer→Disclosure→必要ならOperation | Read Resultを別用途に使う候補のみ | RLS/非干渉の二世界、現在Host Grant、UI/CLI/MCP受領、Effect |

したがって**P0を前進**させる。Referenceの実DB試験を「まだ実Readしていない」と言い続けず、次は[#2586](https://github.com/hat47x/tei-platform/issues/2586) Product Functionから実Providerへ選択・実消費した一経路を同一Source/Attemptで束縛し、#2506のResult Slice、#2514の独立Expected、#1163のConsumer/Revisionへ接続する。#3238はDraft・main未統合で、G2/E2/Product Completionは依然UNQUALIFIED。

### 10.43 CLI-first設計Dogfoodとテスト能力T0：観測できた不足をT1で狙う

[main #3264](https://github.com/hat47x/tei-platform/pull/3264)の限定Issue→Design Workflowは、実TEI Binaryを同一checkoutで起動し9種類のread-only CLI JSON、Product Skillの**静的14件**、Required SourceのreadByLoaderと、対象欠落・異版・未読込の合成負例を照合。[専用CI #42357451983](https://github.com/hat47x/tei-platform/actions/runs/42357451983)はcompleted/success。ClaimはPartialであり、Skillが外部Harnessに実Loadされたこと、全Required Source/Role/Ruleを実Parseしたこと、MCP同型受領、業務意味受理を意味しない。

[main #3266](https://github.com/hat47x/tei-platform/pull/3266)は既存Scenarioから実CLIの版固定**6 Step**を実行し、**実CLI負例4＋独立義務負例2**を識別した。[専用CI #42359069285](https://github.com/hat47x/tei-platform/actions/runs/42359069285)はcompleted/success。局所Go statement Coverageは**C0相当10.1%（選択対象のみ）**、C1は**未計測**。証拠Artifactは7日保持の限定設定で、永続証拠Storeや全機能Coverageではない。[main #3268](https://github.com/hat47x/tei-platform/pull/3268)のCI path trigger拡張は検出力と「テストが発火する入力変更集合」を別に扱う。これらのPR専用Workflow成功を、別のRepository Health失敗・中止と同じ成功として合算しない。

**T0→T1の差**：[Draft #3273](https://github.com/hat47x/tei-platform/pull/3273)は、Issue/Design Contextから未カバーRule（行番号>0）を見付け、**まだ実行していない行番号0の新しい負例**をAIが候補化→既存Scenario/固定Independent Expected/既知Mutantで検証するTG-01〜06を計画。T0のpinned Scenarioを再実行するだけでは、新要求・新Ruleに対するC1 Branch/業務CoverageやTest Generationの能力とは言えない。#3262（必要Context/Source閉包）、#1180（Scenario SSOT）、#2514（独立Expected/判別力）、#1163（Evidence）、#2018（手間・コスト）へ責務を委譲。新Generator・Runner・Oracle Storeはここでは採用せず、**T1新Case実実行、C1、実Provider業務E2EはNOT RUN**。

### 10.44 Product要求の衝突：Verified BOMとData Source Workbenchを同じRQ-IDにしない

**横断統合Blockerを実ファイルで確認**。現在mainの requirements/TEI_PRODUCT_REQUIREMENTS.md blob 4893dac4 は、[main #3276](https://github.com/hat47x/tei-platform/pull/3276)の「**RQ-035 一般開発者が異種データソースのデータと構造を管理できること**」を一件だけ持つ。ところが[Draft #3269](https://github.com/hat47x/tei-platform/pull/3269) HEAD 895bdd61 の同じファイルは、別の意味で「**RQ-035 検証済みPlugin構成BOMと複数Version Matrix**」を使用。これは同一IDの別意味への置換であり、ID件数チェックだけでは防げない。すでに[当該DraftのOwner連絡](https://github.com/hat47x/tei-platform/pull/3269#issuecomment-6706614977)があり、研究PRから重複コメント・コード変更はしない。

**統合前停止条件**：Workbench側RQ-035を現mainの既存正本として保持し、BOM要件は未使用の**RQ-036候補**など別IDへ、#3267 Ownerの判断で付け替える。BOM設計・Requirements索引・関連Trace/Tests/Planへの参照を**一括で**変更し、mainの RQ-ID→Title/Meaning fingerprint と候補PRを対照。明示Migrationがない既存IDの意味差替えは拒否。別のRequirements Registryを増設しない。

BOM計画の方向性は採用候補として維持する。#3269のread-only Referenceはexact Lockに実在する組合せだけを候補にし、Pairwiseで**供給された候補のPair**を覆えても、供給されていないVersion Tupleが不可能・実合格とは言えない。candidateUniverseCompleteness=unknown、qualified=falseを維持。Lock解決、Source Closure、選択Artifact、実Target実行、独立Expected、Tenant/Grant、実Provider/Owner資格を一件ずつ付す。BOMとSBOMの目的を区別し、無制約2因子Coverageを全Version互換保証へ昇格しない。[Issue #3271](https://github.com/hat47x/tei-platform/issues/3271)のCache/Snapshot再利用もBOMのexact Artifact/Source時点に適用する別の資格。#3269はDraft・正式BOM発行/実Resolver/Provider試験はNOT RUN。

### 10.45 Plugin-first Tooling／Graph・HTTP／汎用Data Sourceを最小縦切りへ還元する

[main #3277](https://github.com/hat47x/tei-platform/pull/3277)は「Core以外はPlugin」の製品方向と、**現実はそうなっていない**静的Fit/Gapを固定した。Coreは一部しか分離されず、CLI Dispatch/Command広告は手動一覧、Studio Wails Bridgeは固定5操作、Composition/Service Carrierの製品Plugin消費は未接続。D0依存棚卸し→**D1 read-only InspectorまたはModel Catalogの1件を実Plugin Contract→Resolver→CLI/Studio Consumerへ配線**→実Application Pluginと同じBuild/Test/Invocationで繰り返す。公開を緩くする方針と、Selection/Capabilitiesの明示、Provider別作用、Host Grant、配備Targetは別問題であり、新TEI Host/第二Resolver/専用Plugin Graph型は不要。

[main #3275](https://github.com/hat47x/tei-platform/pull/3275)のHTTP Response Decision Graphは、ServerのRouting・Auth・Method・Content Negotiation・業務Outcome/Effect/ReceiptとClient Reactionのノード/エッジを**標準Graph/表**に投影して検証する設計案。HTTP 202を業務完了、通信応答喪失を副作用無しとせず、RFC 9457 Problem DetailsとClientのRetry/Review/Readbackを別Contractへ。実HTTP/Native Web/Studio Handler/Graph LoaderはNOT RUN。REST/OpenAPIのみをTEI実行の正本にしない。

[Draft #3263](https://github.com/hat47x/tei-platform/pull/3263)の三視点Diagram・CRUD Cross-referenceは、Data/Function/BusinessそれぞれのSource/Meaning IDとRenderer layoutを分離し、CRUD宣言、Business Data Flow、実Provider Effectを別列へ。SHOの独立Markdown図解は別Consumer。図面が見えること、特定ノードを編集できること、反映先Sourceが一意に決まること、実DB保存権限があることはすべて異なる。M-01〜10は既存#2514/#1180へ引き渡す設計Candidateであり実Go/Studio/SHO E2EはNOT RUN。

[main #3272](https://github.com/hat47x/tei-platform/pull/3272)・[#3276](https://github.com/hat47x/tei-platform/pull/3276)のData Source Workbenchの職掌は、異種RDB/NoSQL/Log/Cloudの**データ閲覧・更新、開発者向けSchema CRUD/Version/Plan**まで。DBサーバ運用・Backup・権限システム管理等のDBA業務は標準外。既存Database Provider SPIやworkbench-modelはまだSchema/Query中心であり、実Row→Function Data Context→Presentation→Effectは未接続。MigrationのAppliedUnknown→RepoApplied偽昇格疑いは既存#2892へ。Trait/拡張型は、明示した場合に実装要求・適合検査・実Provider能力が**具体的に変わる**ことを優先し、単なるHintを実能力と誤認しない。時点を正本にするSnapshotはTemporal/ConsistencyとDerivedの再計算特性を別Facetとして扱い、Cache hit≠Current Source/Disclosure資格。W0静的棚卸しを製品W5達成としない。

### 10.46 実行時の選択版・Workspace同一性・Harness再開の偽資格を残さない

[Open #3247](https://github.com/hat47x/tei-platform/pull/3247)はCLIのread-only応答にWorkspace Name/Manifest locator+digestを付ける。これはSourceの読み元を説明する**同一性の候補**であって、同名Namespaceに対する唯一性、World Meaning、現在Grant/採用、Readerの完全性の証明ではない。main未統合のためProduct Skillが常にこの欄を受領できるとも仮定しない。

[main #3254](https://github.com/hat47x/tei-platform/pull/3254)の一因子Source変更で、worldFingerprint/modelRevisionは変わっても導出Operationの5件のBasisはすべて不変。現Product Operationに版比較入力・実drift guardが無いため、**Basis一致は現在Worldの有効なAction/Grantを保証しない**。Operation Offerから実Requestへ移す時点でSource/Pattern/Provider/Consumerの要求改訂を再取得・比較する用途別契約を既存#1174/#2600へ返す。新Guard/Engineは作らない。

[Open #3237](https://github.com/hat47x/tei-platform/pull/3237)はDSH再開でSession Logの既送区間は再送されないが、**Model messagesの会話履歴から以前の合成Canaryが再送される**ことをローカルMockで確認。過去のRaw開示と今のDisclosure資格を混同せず、再開前のGrant/Source/Skill/Task Packet失効をContextに再適用する。過去の開示を取り消せたと主張しない。外部実Provider/Egress/OS Read隔離、H2/H3正式比較はNOT RUN。

### 10.47 既存計画の統合健全性と、直近の一試行受入順

**研究PRの統合状態に留意**：この[Draft #3022](https://github.com/hat47x/tei-platform/pull/3022)はGitHub上でmergeable=false。現mainの同じ研究ファイルには[main #3253](https://github.com/hat47x/tei-platform/pull/3253)により既に§13のSource Identity/非単射/Complect比較が増えている一方、#3022 Branchは別の§10.1〜10.47を保持しており§11〜13を含まない。**文書ブランチに記録したこと≠現mainの採用計画**。Merge前にはmainの§10〜13の原文を失わず、重複節を統合・番号再整理してReadbackと差分試験を行う必要がある。このPRを無条件Auto Mergeしない。現時点では追加のOwnerファイルをこの古いBranchから上書きしない。

**クリティカルパス（#1119の既存Gateを維持）**：
P0-A＝#3262の設計Contextを実Source Loader/Scope/Unknown付きで読み、一つの受注Functionを確定。
P0-B＝#3238で成立した**実PG Reader限定正例**を#2586のProduct Function/Query Selectionへつなぎ、Full/TOP1/P非出力のActual/Expectedを同一AttemptでConsumerへ渡す。
P0-C＝Product Skillの実CLI受領＋T0既存Scenario判別と、T1の**新しい未カバーRule負例**の独立生成・採否を分けて検収。
P1＝同じFunctionを表/Graph/SHO/Studioの二Consumerへ、Identity/Loss/Disclosure/Update Targetの非昇格を確認して投影。
P2＝型付きAction→現在Grant→Provider Effect→Receipt→Resource独立再Read→Client Reaction。HTTP GraphやBOM/Plugin/Cacheの局所Referenceを一連の実Effect完了へ誤合算しない。
P3＝H2/H3の実OS読取り隔離・Session再開後開示・Network Egressの拒否を確認して、独立Expected Holdoutつき比較へ。
G4＝#2831のSource Closure/二回Build/Stage2自動再入を別票で検収。

各段に **実Source/Plugin/Provider/Target/Model Revision、選択Input、独立Expected、実Observer、許可Scope、未観測/非公開、具体的CI exact HEAD** を付す。正式Owner Meaning受理、Product一改訂Provider→Consumer、実SHO/Studio/HTTP統合、H2/H3適格比較、G4 Self-Buildは現在未成立。新Host/第二Registry/万能Test Generator/Attesterを増設しない。

## 15. 2026-11-08：他レーンの実装証拠を「三視点モデル・実Read・AIテスト」の独立P0へ引き継ぐ

**性格**：2026-11-07後半のTEI／SHO／SUI／TAI・Plugin／Trait議論と、現行GitHubの実Go/CLI/PG試験を比較した研究上の受入順。前§10〜14のRelation所属、非単射、Complect、サブシステム境界を保持する。旧[Draft研究PR #3022](https://github.com/hat47x/tei-platform/pull/3022)の未統合の長大な履歴をそのまま再Mergeせず、主張が変化した成熟差分だけを現mainへ追加する。**本節は計画・研究引継ぎであり、新しいTEI Core型、Host、Runner、Registry、Accepted Meaningを採用するものではない。**

### 15.1 独立に着手できるP0-M・P0-R・P0-Tと、それらを結ぶP0-C

| 検証Lane・既存Owner | 確認できた有限の成立点 | 次の受入に不足する辺 |
| --- | --- | --- |
| **P0-M 三視点Modeler** #1124/#1125/#1126／[Draft #3263](https://github.com/hat47x/tei-platform/pull/3263) | Data/Function/Business Graph、Function×Data CRUD、同Semantic Subjectの二Canvas共有、Edge occurrenceとSource逆参照を問う合成Fixture・設計 | 実TEI Reader/Parser/Accepted SourceからのGraph・CRUD実導出、Edge/Cell→原Source対応、独立Expectedと既知Mutant。実Studio/SHO Consumerはその後 |
| **P0-R Function実Read** #2586/#2506／[Draft #3238](https://github.com/hat47x/tei-platform/pull/3238) | 隔離実PostgreSQLでFull/Top-1、非出力PredicateのSQL実評価を有限の独立Expectedで照合。固定HEADの実PG CI・Healthは成功 | Product Function→選択Query/SQL原bytes＋bind引数→実Provider→Result Slice/Context→Consumerの**同一Attempt**。Reference PG局所PASSをProduct E2へ転用しない |
| **P0-T AIテスト能力** #3262/#2514/#1180／[main #3304](https://github.com/hat47x/tei-platform/pull/3304) | 実CLI9種・Design Context合成16負例・既存Pinned 6 Step。既知Ruleから追加した行番号0の一時7番目のStepを実CLIで判別 | T1-A一例の成功と、AIが未知Rule/欠陥を盲検で発見するT1-B、C1/NFR/要求Coverageは別。独立Expectedを生成Implementationから導かない |

**P0-Cは統合資格**：P0-Mのsales.orderの意味・Relation IDと、P0-Rの実Rowset・Scope、P0-TのTest対象が、Source/Role/Workspace/Revision/Result grain/Read epochに一致することを独立Consumerで照合。DiagramのCRUD Rは実Provider ReadでもEffectでもない。同一Subjectの二Canvas Occurrenceは同一業務Entityの別表示であり、新しいDB行でもない。P0-Mを実Provider完成までブロックせず、P0-Rを実Diagram完成までブロックしない。既存#1119 Gateを維持。

### 15.2 Data/Presentation/Bindingを分離したまま多Canvas・Itemを検収する

他チャットのSUI検討では、大型Canvasは分割しても関係を維持し、**同一Card/Itemを複数Canvasに出現させられる**。Tree/Outline、KJの空間配置、表・図のProjectionを独立に扱う。未整理の生情報もItemへ暫定格納でき、Task/Action/Eventの意味付与は固定排他的EnumよりRole/Trait候補として検討する。独立した時点・主体・履歴を持つ出来事なら別Semantic Subjectが必要になり得るが、これをTEIの新Core Item型の正式採用としない。

[Draft #3263](https://github.com/hat47x/tei-platform/pull/3263)はEntity9/Relation7/Function4/Business Action3・Canvas2を含む合成候補。二Canvasの別Occurrence、同端点別IDの多重Edge、Tree化で非表示になる複親、Chart集計点と元明細行、編集によるSource変更と業務Value変更を別々に反証する。Identity/Scope/Non-injective loss/Current Disclosureは#2495/#2497/#2507/#2516のOwnerに置く。AI草案Expectedの自己整合を独立業務正答としない。

[Draft #3303](https://github.com/hat47x/tei-platform/pull/3303)は三視点・表現型への受入計画を現mainから再構成する案であり、実Product Diagram/CRUD・Studio/SHO保存・Owner受理を保証しない。図の一つのCardを他Canvasに複製配置することを、意味正本・Relation・Historyの複製としない。Figure drag/Selection/Tab/Seekは原則Presentation状態であり、Owner Action・Provider Effect・Source保存は別経路。

### 15.3 Completionと非出力Predicateの実Read証拠をConsumer資格へつなぐ

[Draft #3238](https://github.com/hat47x/tei-platform/pull/3238)の[実PG CI #42351796330](https://github.com/hat47x/tei-platform/actions/runs/42351796330)と[Health #42351796383](https://github.com/hat47x/tei-platform/actions/runs/42351796383)はcompleted/success。Tenant Aの独立期待集合L={3,9}に対するFull/Top-1の観測と、Consumerに返さないordered_atをSQL WHEREで実評価しID9のみを返す正例が成立。**TOP1の1行は全件でなく、同件数でも重複R=[3,3]は必要9を含まない。** 非出力Predicateを実Providerが評価できたことと、結果行の存在・件数から秘密を推知されてもよいというDisclosure許可は別。

次の同一Attempt Packetで固定する欄は、Function選択・Source/Workspace identity・SQL原bytes/Parameter・実Provider/Query execution・Role/Relation path・Result Slice/Row ID/重複/粒度・Coverage basis・Predicateの実評価箇所・Independent Expected・Source/Policy/Read epoch・現在Consumer Disclosure。CallerのCoverageComplete/Holds自己申告だけでCompleteへ昇格しない。実Providerが非出力列を評価した証拠がある場合には、Consumerへの列開示が無いことだけで過剰拒否もしない。業務Writeは別Laneで、Commit/Receipt lookup/resource-observed/Versionと実更新対象全件性、Owner Meaningを分ける。

### 15.4 T0からT1-Aへの実前進と、T1-Bの未実証

[main #3284](https://github.com/hat47x/tei-platform/pull/3284)／[main #3296](https://github.com/hat47x/tei-platform/pull/3296)は、実CLI応答9種を読みDesign Contextを評価し、Required Source欠落、片側Root、途中切断、JSON重複memberなど**合成16 Negative**を識別した。正常入力でもContextはpartial、qualifiedTestGeneration=false。[main #3299](https://github.com/hat47x/tei-platform/pull/3299)では実SourceとPlugin Lockを改変すると既存Pinned Scenarioを拒否したが、A→B→Aで元bytesへ戻ったことを中間変更なし／現Grant有効とは推論しない。

[main #3304](https://github.com/hat47x/tei-platform/pull/3304)の専用[実CI #42378033965](https://github.com/hat47x/tei-platform/actions/runs/42378033965)は、既知の正数Ruleに基づく行番号0の新しい一時Scenario Stepで positive Ruleのみnot-satisfied、Quantity/Uniquenessはsatisfiedと判別し、偽Expected・偽Fixture・古いPinの3負例を拒否。これは**T1-Aの限定実行**。専用CIのProgram HEADとPR最終HEADのHealthを別証拠にする。Scenario正本に採用したことでも、AIが未知Rule/欠陥を盲検発見したT1-Bでもない。C1/要件/NFR Coverageと実Provider/E2EはNOT RUN。次の非同種Ruleの一因子対照、独立Expected、生成時Source/Role/Plugin Pinを#2514/#1180へ返す。

### 15.5 Plugin-first／Trait／BOM／Workbenchは一つの実Provider呼出しを先に証明する

[main #3277](https://github.com/hat47x/tei-platform/pull/3277)のCore以外Plugin方針は、現CLI・Studioが既にPlugin化済みという事実ではない。[main #3300](https://github.com/hat47x/tei-platform/pull/3300)の静的実装監査では、旧Service Carrierはapplication-runtime以外のScopeを拒否し、InspectorはCLIとStudioから直接operationsを呼ぶ。最短D1はHost-owned Authoring Adapter→既存Contract/Resolver→**実選択Providerを実Invoke**→CLIとStudioの両ConsumerでSource/Selection/Revision/Resultを照合。「Bを選択してAを実呼出し」する同出力Negativeを検出する。新共通TEI Host・第二Resolverを導入しない。

[main #3298](https://github.com/hat47x/tei-platform/pull/3298)のAtomic Trait契約は宣言、適用条件、手動実装義務、Plugin選択、実Link/Invoke、Conformance、現在Grantを区別。Hintでは未実装のConditionalWriteを履行済みとしない。read-only Inspector D1とWrite Traitの未履行検証は並行。BOMの候補Version Pair coverageは、未供給Tupleの不存在・実互換/Provider成功ではない。現要求**RQ-035はData Source Workbench**、BOMは独立要求IDのOwner判断（RQ-036候補）に戻し、同一IDの意味差替えを拒否する。[main #3287](https://github.com/hat47x/tei-platform/pull/3287)のJSONL/PG Fixtureの試験PASSも実Plugin/DB Reader/生成Go Applicationの適合を証明しない。

### 15.6 CLI／外部Harness／Buildの実観測から導く資格の上限

[main #3307](https://github.com/hat47x/tei-platform/pull/3307)はCLI Discoveryの閉じた引数集合を実受理値と照合。[Open #3311](https://github.com/hat47x/tei-platform/pull/3311)はWorkspace Name/Manifestと名前付き出力を追加するが、WorkspaceのSource全閉包、認可、Meaningは証明しない。[Draft #3302](https://github.com/hat47x/tei-platform/pull/3302)のview -executeはread-only契約と実ファイル書込みを分離する提案段階。Harness SkillのToolを「読める」ことと実Invocation Variant/Effect/Disclosureを別判定にする。

DSH read-only/workspace-writeが作業Dir外を読めることは実証済み（main #3228）。[Open #3309](https://github.com/hat47x/tei-platform/pull/3309)の合成Canaryはstdoutと**複数連結zstd frameで圧縮された永続Session Log**に残る。最初のframeだけを読んで機密漏洩なしと判定しない。別OS Identity/実Read ACL、Network Egress拒否、実Provider送出は未実証で、H2/H3の比較適格は引き続き0/4。#2515、TAI #168/#232が既存Owner。

[main #3193](https://github.com/hat47x/tei-platform/pull/3193)の2回Build byte一致と[main #3156](https://github.com/hat47x/tei-platform/pull/3156)のSource Closure=rejectedは両立。[Open #3313](https://github.com/hat47x/tei-platform/pull/3313)は外部Moduleを明示供給したときだけ選択Sourceを受け入れるopt-in Reference。既定repository-onlyの拒否は維持し、go.sum toolchain依存検査・独立byte照合・供給元Trust/署名・Stage2実TEI Buildを分ける。G4はUNQUALIFIED。#2831/#1090 Ownerを迂回しない。

### 15.7 次の小さい受入と旧Draftの扱い

**並行順**：P0-Mは実TEI Sourceから三視点Graph/CRUDを導出してSourceへ逆参照する。P0-Rは一Functionから実PG Query→Data Context→read-only Consumerを同一Attemptで渡す。P0-Tは非同種Ruleで追加Scenario候補→独立Expected→実CLIを再度判別する。P0-Cは3者のSource/Role/Revision/Scope/Grain一致を別のReaderで照合する。P1は標準表現型の二ConsumerでIdentity/Loss/Disclosure/Selection同期を実測。P2は必要な場合だけ型付きAction→現在Grant→実Effect→Receipt→独立Resource Read→Client Reaction。H2/H3隔離とG4は別経路で資格を得る。

この第15章は現行mainの第11〜14章を保持した追加計画である。衝突中の旧[Draft #3022](https://github.com/hat47x/tei-platform/pull/3022)の§10.1〜10.47は経緯・未実証Caseの参考とし、mainへ機械的に重複Mergeしない。必要な唯一の成熟差分だけを既存Issue/PR/研究計画の新Revisionに移す。現行Evidenceと異なる旧状態を再採用しない。独立Expectedは#2514、Scenarioの正本は#1180、実Consume/AcceptanceのEvidenceは#1163、既存Gateは#1119。実Product三視点/PG/Provider/Studio/SHO E2E、未知Test自律生成、外部H2/H3資格、G4 Self-Build、正式Owner意味受理は**NOT RUN / NOT PROVEN**。

### 15.8 必要Ruleの未取得・失効とTest Generation資格（main #3354に一本化）

並行していた[Draft #3324](https://github.com/hat47x/tei-platform/pull/3324)／[Draft #3326](https://github.com/hat47x/tei-platform/pull/3326)の成熟差分は、[main #3354](https://github.com/hat47x/tei-platform/pull/3354)へ統合済み。固定HEAD 0ab2b99fの[専用Dogfood CI #42391707962](https://github.com/hat47x/tei-platform/actions/runs/42391707962)と[Repository Health #42392009764](https://github.com/hat47x/tei-platform/actions/runs/42392009764)はcompleted/success。重複旧Draftをそのまま二重Mergeしない。

**実証の上限**：既存9＋全Indexモードsource-index＋Focused inspect＝**実Read-only CLI 11応答**。独立にRequiredと定めたValidation3 Rule、Source Loader Read、Catalog ID、Definition Role/Path、Canonical Rule・attribute、関連Relation Parent、Inspect ReferenceとFingerprintを照合。限定Consumer Controlsは**32件**（従来16＋独立Required Rule系8＋Focused Rule/Relation/Index系8）。同一専用CIで既存Pinned Scenario6 Stepと既知T1-A一時Stepも実行。これらは**Consumerの合成Tool Reply改変に対する局所判別**で、32個のCode Mutant kill、全Function/Business/MeaningのClosed World、T1-B未知Rule自律発見、C1/NFRではない。

source-indexのsubjectモードを定義全件Indexと誤認した初回失敗は**試験側の取得モード誤り**であり、Rule実不存在の証拠ではない。11応答は逐次実行で原子的World Snapshot未保証。実Readerが見たbytes／Mode／Required母集団／Role・Source span／Result Shape／完全配送／Parser・実Rule Consumed／現在Disclosureを別Claimとし、部分ReferenceではboundedReference=partial、qualifiedTestGeneration=falseを維持。#3262/#2514/#1180/#3223へ引き継ぐ。

### 15.9 実行後の記録障害はClean RejectedでなくEffectUnknown

[Draft #3328](https://github.com/hat47x/tei-platform/pull/3328)では、Executor起動後のRecorder失敗、Lifecycle Observe／MarkUnknown失敗をCOMMAND_RECORD_FAILED／COMMAND_LIFECYCLE_RECORD_FAILED + EffectUnknown=trueとして返し、事前認可拒否だけは確定Rejectedに保つ。既存AttemptがあるStageDispatchedでMayReplay=trueと返していた矛盾も修正した。固定HEAD 1b8ff4e6でRepository Health／専用Dogfood／Self-Build Workflowはcompleted/success。だが**Draft・main未統合**で、実PG障害注入・Commit ACK喪失・別Host永続Fence・実Provider ReadbackはNOT RUN。

**最小独立対照**：A=事前Grant拒否／Executor未起動、B=Executor起動後Recorder失敗、C=Executor完了後Observe失敗、D=既UnknownのMarkUnknown失敗、E=応答喪失後にReceiptあり／Resource現在値異なる。Aだけ確定拒否、B〜Eは証拠が不足するならEffectUnknownを保持し、無条件ReplayやEffect=0を推測しない。Receipt Lookup成功とResource postimage一致も別で、実Version/Scope・現在Grant・現在Consumer Disclosureを検査する。既存Command #1174/#2564、Script回復 #2502、HTTP Decision Graph #3274、Evidence #1163がOwner。第二Transaction Coordinator/Receipt Registryを作らず、P0-Rのread-only Product Readへ書込障害試験をHard dependency化しない。

### 15.10 Discovery・選択・現在版・実Invokeを分ける

[main #3325](https://github.com/hat47x/tei-platform/pull/3325)のOperation Drift参照はWorldFingerprintとModelRevisionをCaller期待値と比較し、current/stale/uncheckedを区別。Basisは属性差異でも変わらず、現在版の代替ではない。参照ツールのcurrentは実Command前のCASや現在Grantの証明ではない。[main #3333](https://github.com/hat47x/tei-platform/pull/3333)はCLI DiscoveryのviewにwriteFlagsを広告し、operation-logとlifecycle-stateが実Named Fileへ書くことをGo実Processで確認した。従来effect=read-onlyは正本を変えない意味に限られ、**OS上の無書込保証ではない**。Flag/Target/実Invocation VariantをHostで再照合する正式契約は未決。[Open #3311](https://github.com/hat47x/tei-platform/pull/3311)のWorkspace name/Manifest digestはOriginの説明であって、全Required Source/実消費/Current Authorityの証明ではない。

Pluginの[main #3337](https://github.com/hat47x/tei-platform/pull/3337)は、宣言Capability12件のうち6件が環境Bindingから未選択とした初版Reference。database-instanceの技術名postgresとCapability ID database.postgresを単純文字列比較しない制限があった。[main #3350](https://github.com/hat47x/tei-platform/pull/3350)で既存Validation.SupportsDatabaseProviderを再使用し、database-provider2件を対応技術により選択済みと分類、同一Referenceの未選択は**4件**へ変わった。固定HEADのHealthはcompleted/success。6→4は**解決規則が異なる観測**であり旧Evidenceは保持する。選択済みでも実Provider Load/Invoke・Host Grantは未実証。

**D1の具体的受入**：read-only Inspectorで選択したPlugin Bと実InvokeしたProvider BのSource/Scope/Contract/Artifact/Version/Resultを、CLIとStudioの二Consumerで一致させる。Selected B→Actual Aでも同じ出力になる負例を独立Expectedで拒否する。#951/#1251/#3278/#3267がOwner。[Draft #3327](https://github.com/hat47x/tei-platform/pull/3327)のA→B→A遅着Inspect応答拒否はUI採用制御の限定試験であり、Host処理・副作用の取消、World CASではない。

### 15.11 Stageの起動主体とH2/H3保存面の観測を分離

[main #3336](https://github.com/hat47x/tei-platform/pull/3336)の参照Stage Driverは別Go実行プロセスを起動し、invoker executable digest／invoked executable digest・argv・PID・exit、子の報告Artifact digestと実Disk Readbackを別に記録。Windows/WSLの局所Go試験7件とRepository Healthは成功。**l3Claimed=false**で、Stage1生成TEI自身をinvokerへ据えた実験ではない。Build二回Digest一致、外部Module供給、Source Closure Qualified、Stage2 Product TEI起動、Owner受理は独立資格。既定ResolveSourceClosureはなおrejected、[Open #3313](https://github.com/hat47x/tei-platform/pull/3313)の外部Module opt-inは製品既定経路未接続。#2831/#1090のL3/G4はUNQUALIFIED。

[Open #3309](https://github.com/hat47x/tei-platform/pull/3309)のDSH合成Canaryはstdoutだけでなく永続Session Logにも存在。Logは**9個の連結zstd Frame**で、先頭だけ展開すると179 bytes／Canary0、全Frame展開では40,136 bytes／Canary1。部分展開による観測0件を「漏洩無し」に昇格しない。cache/storage/attachment限定走査の一致0も全Sink・外部Provider送出・削除・保持期限の証明ではない。read-only/workspace-writeの実OS外部読取拒否、Network Egress、Tool/Skill/独立Expected Holdoutは未実証。H2/H3比較適格0/4、#2515／TAI #168/#232がOwner。機密情報でなく合成Canaryのみを継続。

**次のP0順を更新**：#3354でTT-56のReader/Requiredは一本化済み。P0-Tは不足するFunction/BusinessのRequiredと実受領を圧力試験し、#2586で一Functionの実PG ReadをData Context/Consumerへ同一Attempt接続→P0-R、#3263の実Source→Graph/CRUD Source逆参照→P0-M、#1251のCLI/Studio実Plugin D1、#1174の実DB障害注入を各Ownerで並行させる。G4とH2/H3は別経路。Required/Discovered/Loaded/Parsed/Actually Consumed/Current Grant/Independent Expectedを別列で保持し、#1119既存Gateを変更しない。新Core Host・第二Evidence Store・中央Attester・汎用Test Generatorを導入しない。正式Owner Meaning Acceptance、Product一改訂E2E、L3、H2/H3実隔離はNOT RUN。

### 15.12 CLIのOutput Shape／部分テスト目録を、必要情報の欠落と混同しない

[Open #3356](https://github.com/hat47x/tei-platform/pull/3356)の実CLI Dogfood第2回では、Workspace同一性・Lock再入力・Help・書込みflag等が前進した一方、(1) viewのSemantic FPはWorld/IndexとScopeが異なる、(2)管理対象MarkdownがLoaderで意味読込されない、(3)JSON result／diagnosticsのShapeがCommand別、を確認。MCP Server入口は現実装に無く**MCP経路はNOT RUN**。CLI成功から別HarnessのMCP同型受領を推測しない。

[Open #3366](https://github.com/hat47x/tei-platform/pull/3366)はviewのJSONへWorld/SourceIndex FPを追加する候補。Semantic Uses FPは異なる対象範囲なので値一致を要求しない。[Draft #3370](https://github.com/hat47x/tei-platform/pull/3370)はrequirementsのresultが**Array**であることをDiscovery resultShape=arrayと実CLI試験へ載せる。広告の無いShapeはobjectでなく**unspecified**。CIは確認時点一部成功・全Health実行中。成功Envelope／内側Result型／FP範囲を分け、全Commandの破壊的統一を先行しない。

[Draft #3365](https://github.com/hat47x/tei-platform/pull/3365)のTT-105/106は、部分Scenario Receipt目録で実観測済Stepの局所成果を保持するが、未収録Stepを自動でuntestedConfirmedへ分類しない。終端欠落・偽終端・Source SHA不一致・重複Receipt・未検証Receiptの負例を持つ。固定HEADの専用Dogfood／Health CIはcompleted/successだが、実Harness途中切断や全Product Coverageは未実証。独立Expectedを保持し、局所CoveredとRequirement Universe完全性を別列にする。#1180/#2514/#3262がOwner。

### 15.13 Plugin選択／原子的Trait Required／Inspector実Invocationを接続する

[main #3363](https://github.com/hat47x/tei-platform/pull/3363)のSelection Matrixは既存Capability.Supports*による環境Binding6形式を一覧し、公開合成Workspaceで**24 Binding中 selected22／missing2／ambiguous0**を観測。missing2はeditor.database_workbench（development/production）で、現Product Validationは環境Featureのうちsql_analysisだけをCapability照合する。したがって**Reference missing＝実製品Validation失敗ではない**。ここでValidatorに新規規則を導入するかは#951/Plugin Owner判断。

[main #3368](https://github.com/hat47x/tei-platform/pull/3368)はPlugin package選択とCapability選択の不一致、同一Pluginの能力別選択、同一DB技術への複数Provider候補という3負例を追加。TraitのDeclared／Required／Selected／Resolved／Invoked／Verifiedを分離し、Required集合がpartial/unknownなら不足0件を充足率100%にしない。[main #3377](https://github.com/hat47x/tei-platform/pull/3377)はTrait適用先Field/Row/Function/Plugin/Representationの差、未Parsed Markdown、非開示Required、Manual CAS欠落を設計反例に追加したが、TC-21〜26実ProviderテストはNOT RUN。Traitヒントを実装履行にしない。

[Draft #3371](https://github.com/hat47x/tei-platform/pull/3371)はStudio Host Factoryに型付きInspectionServiceを追加し、Factory指定時の旧operations直接呼出しへのfallbackや不正応答を拒否する限定Port実装。CIの部分成功は確認できるが固定HEADの全Healthは確認時点実行中で、**CLI/Studioで同じ選択済みPlugin Providerを実InvokeするD1はまだ未完成**。最短試験はA/B二Providerを用意し、B選択時に実InvocationがBであることをCLI→Studio両方で証明し、B選択→A実呼出し（Result同値）を失格化。#1251/#951/#3278/#3267がOwner。新Resolver/Hostを作らない。

### 15.14 DB Reverseで観測した制約を三視点の業務意味へ自動昇格しない

[Draft #3372](https://github.com/hat47x/tei-platform/pull/3372)は物理Catalog Reader→WorkbenchにFK/UNIQUEの複合列、別Schema同名表、観測時刻とFPを保持する設計・実Product Go経路のMock Catalog試験候補。現HEADのCIは確認時点pending。Physical ObservationのFK/UNIQUEは、業務Relation/Business Key/CRUDの**構造候補**にすぎず、Owner Accepted Meaning・Snapshot完全性・実PG現在Read・Host Grantではない。

評価は Physical Source Observation→Structural Candidate→Owner Accepted Data/Function/Business Meaning→Graph/CRUD Source逆参照→Product Function/Provider実Read を別Laneで進める。物理制約が無いことから業務Relation不存在を断定せず、DB reverse完了を三視点P0-Mの必須Blocking前提にしない。#3270/#2495/#2497/#1124/#2586に引継ぎ、第二DB Schema Engineや図面正本は作らない。

**現時点の並行優先順**：P0-Tはmain #3354の11 CLI/32限定Controlsから、必要Function/Business情報と実Consumer受領、次に盲検T1-Bへ。P0-D1は選択Provider→実Invocation→CLI/Studio二Consumerを証明。P0-RはReference PGの成功を実Product Function/Query/Provider/Consumerの同一Attemptへ。P0-Mは実TEI Source→三視点Graph/CRUDのread-only導出、Workbench Reverseは候補Inputとして並行。H2/H3隔離・G4は独立Owner。#1119 Gateを変えず、新Host/Runner/Registry/Attesterを増やさない。正式Owner Meaning、実Product E2E、L3/G4、H2/H3比較はNOT RUN。
