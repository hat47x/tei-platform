# reference-cognition-port（#2021）

軽量な認知・推論を、TEIの設計支援へ接続するための**提案ポート**の参照です。端末内・閉域・低資源の環境でも候補を出せるようにしつつ、提案を意味の受理・権限・実行から分離します。

この実装はモデル品質を証明しません。決定的な参照proposerと外部コマンド境界を使い、**提案の出所・入力文脈・能力限界・失敗状態が失われないこと**を確認します。

```text
go run ./tools/reference-cognition-port/cmd/tei-cognition-propose \
  -request tools/reference-cognition-port/testdata/request.json \
  -provider tools/reference-cognition-port/testdata/providers/local-rules-a.json
```

## 境界

`Propose`は提案receiptを返すだけで、次を行いません。

- 提案を受理済み意味へ昇格する。
- 権限・実行許可・本番配備を決める。
- 提案を画面やWorkspaceへ適用する。
- providerの自己申告を、正しさや能力の証拠にする。

receiptは常に`grantsAuthority: false`、`accepted: false`、`applied: false`を持ちます。候補を採用する場合は、既存のShared Acceptance / 変更受理の境界へ渡す必要があります。

## 状態

- `proposed`: 候補がある。受理ではない。
- `unsupported`: providerが要求された能力を宣言していない、またはその種別を実装していない。
- `unknown`: 判断に必要な観測が足りない。空の提案へ潰さない。
- `failed`: command失敗、timeout、出力不正、上限超過など。
- `blocked`: offline要求に対して、networkを必要とするproviderを選んだ。

`unsupported`・`unknown`・`failed`・`blocked`は成功として扱いません。CLIは`proposed`以外で終了コード1を返します。

## request

```json
{
  "schema": "tei.reference.cognition-request/v0",
  "id": "request-validation-1",
  "kind": "validation-candidate",
  "profile": "offline",
  "context": {
    "workspaceRevision": "synthetic-context-v0",
    "acceptedReferences": ["sales.order_line.quantity"],
    "observations": [
      {
        "id": "obs-1",
        "statement": "quantity may be negative",
        "reference": "sales.order_line.quantity"
      }
    ],
    "constraints": ["no network"]
  }
}
```

- `profile`は`offline`または`connected`です。`offline`は`network: none`を宣言したproviderだけを受け付けます。
- `context`は、提案に使ってよい受理済み参照・観測・制約へ限定します。
- `context`の実byteは上限`maxContextBytes`で拒否できます。
- この参照の`workspaceRevision`は合成値であり、実Workspaceの受理済みrevisionではありません。

## provider

```json
{
  "schema": "tei.reference.cognition-provider/v0",
  "id": "local-rules-a",
  "model": "deterministic-rules",
  "version": "1",
  "execution": "local",
  "network": "none",
  "capabilities": ["validation-candidate", "diagnosis"],
  "limits": {
    "maxContextBytes": 65536,
    "maxOutputBytes": 65536,
    "timeoutSeconds": 10
  },
  "settings": {"strategy": "first"}
}
```

- `execution`: `local`（決定的参照proposer）または`external`（信頼済みコマンド）。
- `network`: `none`または`required`。**宣言**であり、sandboxによる強制ではありません。外部コマンドは信頼済みのものだけを指定し、秘密情報を出力しないでください。
- `capabilities`: 提案できる種別。要求された種別がなければ`unsupported`。
- `limits`: 文脈・出力のbyte上限とtimeout。
- `settings.strategy`: `first`または`all`（localのみ）。同じ観測から最初の候補だけ、または全候補を返します。

## 外部コマンド

```text
go run ./tools/reference-cognition-port/cmd/tei-cognition-propose \
  -request tools/reference-cognition-port/testdata/request.json \
  -provider tools/reference-cognition-port/testdata/providers/external-local.json \
  -command 'your-local-proposer "$TEI_COGNITION_CONTEXT" > "$TEI_COGNITION_OUTPUT"'
```

コマンドへ`TEI_COGNITION_CONTEXT`、`TEI_COGNITION_OUTPUT`、`TEI_COGNITION_PROVIDER`、`TEI_COGNITION_MODEL`、`TEI_COGNITION_VERSION`を渡します。コマンドは`TEI_COGNITION_OUTPUT`へ提案JSONを書きます。特定ベンダーのAPI・認証・課金・モデル版管理はTEI内に埋め込みません。

## 提案の検査

- 提案schemaとstateを検査する。
- `proposed`は1件以上の候補を要求する。`unsupported`・`unknown`・`failed`は候補を持てない。
- 候補の`kind`はrequestの`kind`と一致する。
- 候補は`basis`を必須とし、`observation:<id>`、`reference:<id>`、`constraint:<index>`のうち、**実際にrequestのcontextに存在するもの**だけを参照できる。根拠の捏造を拒否する。
- `accepted`などの未知fieldを持つ提案は拒否する。提案が自分自身を受理済みにできない。
- `maxOutputBytes`を超える出力、timeout、command失敗を`failed`として返す。

## 検証

`go test ./tools/reference-cognition-port/... -count=1`で、次を確認します。

- 決定的proposerが提案を返し、権限・受理・適用を主張しない。
- 同じcontextを別providerで再利用でき、providerの違いをdigestで区別できる。
- offline要求がnetwork providerを`blocked`にする。
- 未対応能力を`unsupported`、観測不足を`unknown`として保持する。
- 外部コマンドの成功・失敗を区別する。
- 提案への`accepted`注入、存在しないbasis、context上限超過を拒否する。

## 未実装・限界

- 実軽量モデルの実行・評価。`local-rules-a`/`b`は決定的な参照proposerであり、モデルではありません。
- 実Workspaceの受理済みrevision・意味定義との結合。現在のcontextは合成です。
- Shared Acceptanceへのhandoff、変更案の生成、差分・影響の計算。
- 端末内モデルの配布・版管理・量子化・資源測定。
- network宣言の強制、外部コマンドのsandbox、秘密情報の検査。
- モデル差し替え時の品質比較。同じfixtureを再利用できる境界だけを確認します。
