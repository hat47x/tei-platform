# ADR-0038: 認知機能を提案ポートとして扱い、受理・権限・実行から分離する

- 状態: 提案中
- 日付: 2026-10-19
- 関連ADR: ADR-0022 / ADR-0030 / ADR-0035
- 関連Issue: #2021 / #267 / #958 / #1045 / #876 / #843
- 関連実装: [`tools/reference-cognition-port/`](../../tools/reference-cognition-port/README.md)

## 背景

TEIの設計支援は、生成AIの提案と意味の受理を分けることで成立している。しかし、軽量モデルを端末内・閉域で使うと、次の失敗が起きやすい。

- モデルの出力を、そのまま受理済みの意味や実行許可として扱う。
- モデル名・版・入力文脈・設定が失われ、後から提案の根拠へ戻れない。
- 能力外・観測不足・command失敗を、空の提案や成功へ潰す。
- モデルを交換したとき、同じ入力と同じ期待値で比較できない。
- 根拠として存在しない参照を、提案が自己申告する。

クラウドの大規模モデルでも同じ問題はあるが、端末内・閉域では、モデルの版・資源・network依存が利用者ごとに異なるため、境界を明示する必要が高い。

## 判断

### 提案だけを返す

認知機能は、候補・分類・診断・要約を**提案**として返す。受理、権限付与、実行、本番配備を決めない。提案の採用は、既存のShared Acceptance / 変更受理の境界で行う。

### 状態を潰さない

少なくとも次を区別する。

- `proposed`: 候補がある。受理ではない。
- `unsupported`: 能力・種別が対象外。
- `unknown`: 判断に必要な観測が足りない。
- `failed`: 実行失敗、timeout、出力不正、上限超過。
- `blocked`: offline要求に対してnetwork依存のproviderを選んだ。

空の提案を成功として返さない。`unsupported`・`unknown`・`failed`・`blocked`は、候補なしの`proposed`ではない。

### 文脈と能力を先に限定する

providerは、提案できる種別（capability）、入力文脈の上限、出力の上限、timeoutを宣言する。requestは、提案に使ってよい受理済み参照・観測・制約へcontextを限定する。providerが宣言していない能力を要求した場合は、推測で代替しない。

### 出所を提案の前後で固定する

receiptへ、request・context・provider descriptor・settingsのdigest、providerのid・model・version・execution・network、入力fingerprint、出力digestを残す。同じcontextを別providerで再利用できるようにし、providerの違いをdigestで区別する。

### 根拠の捏造を拒否する

提案の各候補は`basis`を持ち、`observation:<id>`、`reference:<id>`、`constraint:<index>`のうち、requestのcontextに実在するものだけを参照できる。存在しない根拠を持つ候補は拒否する。

### 提案に受理の語彙を持たせない

提案schemaに`accepted`・`applied`・`authority`などを置かない。未知fieldとして拒否し、提案が自分自身を受理済みにできないようにする。receiptも常に`grantsAuthority: false`、`accepted: false`、`applied: false`を持つ。

### offlineは宣言として扱い、sandboxと混同しない

`offline` profileは、`network: none`を宣言したproviderだけを受け付ける。ただし、これはprovider descriptorの宣言であり、networkの強制やsandboxではない。外部コマンドは信頼済みのものだけを使い、秘密情報の検査は別の境界とする。

### Shared Acceptanceを複製しない

認知ポートは提案のhandoffまでを扱い、受理・差分・影響・適用の正本を作らない。候補の採否、権限、実行は既存の受理境界へ戻す。

## 代替案

- **モデル出力をそのまま変更案として適用する**: 提案と受理・権限を混同するため採用しない。
- **成功した提案だけを残し、失敗・不明を捨てる**: 能力限界と観測不足を隠すため採用しない。
- **providerごとに別のcontext形式を持つ**: 同じ入力でモデルを比較できず、差し替えの意味が失われるため採用しない。
- **提案の根拠を自由記述にする**: 実在しない参照を根拠として受理しかねないため採用しない。
- **offlineをnetwork sandboxとして主張する**: 宣言と強制を混同するため採用しない。

## 影響

- 設計支援の候補は、providerの能力・文脈・出所・限界と組で記録される。
- 軽量モデルと大規模モデルを、同じrequest・context・期待値fixtureで比較できる。
- 失敗・不明を保持するため、UI・CLI・運用側に状態ごとの表示が必要になる。
- モデルの品質・正しさは、提案が成功したことでは証明されない。評価は#876 / #843の範囲へ戻す。
- 受理・適用の実装はShared Acceptance側に残り、二重の正本を作らない。

## 未決

- 軽量モデルの配布・版管理・量子化・資源測定をどこで扱うか。
- `basis`の参照形式を、既存のSemantic Reference / Evidenceの形式へどう寄せるか。
- 提案の評価集合・期待値fixtureを、誰がどの単位で維持するか。
- network宣言を、OS・sandbox・監査でどこまで強制するか。
- モデル交換時の提案差分を、利用者へどう提示するか。

## 検証

#2021の参照実装で、次を確認する。

- 決定的proposerが提案を返し、権限・受理・適用を主張しない。
- 同じcontextを別providerで再利用し、providerの違いをdigestで区別できる。
- offline要求がnetwork providerを`blocked`にする。
- 未対応能力を`unsupported`、観測不足を`unknown`として保持する。
- 外部コマンドの成功・失敗を区別する。
- 提案への`accepted`注入、存在しないbasis、context上限超過を拒否する。

この検証は、実軽量モデルの品質、実Workspaceの受理、Shared Acceptance、本番動作を証明しない。効果が確認できない場合は、Cognition Portの抽象を参照実装の範囲へ戻す。
