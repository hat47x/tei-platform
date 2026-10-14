# TEI Studio：静的合成の独立黒箱検査（#1824 / #7）

PoC Bの**実装者ではない側**から、通常ビルドとTEI宣言による静的合成の挙動を照合するための限定的な検査器です。Python標準ライブラリだけを使用し、TEIのResolver、Go Service、UI Registry、製品用宣言・Lock形式を新設しません。PoCの受入条件と横断の記録方法は[#1824](https://github.com/hat47x/tei-platform/issues/1824)および#8担当のEvidence手順を参照してください。本READMEは新Gate・新runbookではありません。

## 実行

```bash
python3 -m unittest -v tools/reference-studio-composition-evidence-7/test_probe.py
python3 tools/reference-studio-composition-evidence-7/probe.py --spec /path/to/trusted-fixtures/probe-spec.json
```

`--spec`には**信頼できる開発者が用意したローカルJSONファイル**のみを指定してください。そのディレクトリを作業ディレクトリとして、`argv`の外部コマンドをshellを介さず実行します。指定コマンド自体は任意実行権限を持つため、外部から受け取った無審査のspecを実行しないでください。

specは**この検査器専用の入力形式**であり、#1が実装する合成宣言やTEIのschemaではありません。実装担当から実行手順が提示された後、下記の各`argv`へ**本物のコマンドとfixtureの組**を指定します。stdoutは単一のJSON文書とし、JSON Pointer（`/result`など）には、そのコマンドが実際に返す位置を設定します。架空の本番APIやエラーコードを前提としません。

```json
{
  "timeoutSeconds": 30,
  "comparison": {
    "regular": {"argv": ["<実際の通常ビルド実行コマンド>", "..."], "resultPointer": "/..."},
    "composed": {"argv": ["<実際の合成ビルド実行コマンド>", "..."], "resultPointer": "/..."}
  },
  "negative": {
    "unknown": {"argv": ["<未知部品の入力を与える実コマンド>", "..."], "errorPointer": "/...", "expectedCode": "<実際の失敗分類>"},
    "duplicate": {"argv": ["<重複ID>", "..."], "errorPointer": "/...", "expectedCode": "<実際の失敗分類>"},
    "missing_dependency": {"argv": ["<依存欠落>", "..."], "errorPointer": "/...", "expectedCode": "<実際の失敗分類>"},
    "cycle": {"argv": ["<循環>", "..."], "errorPointer": "/...", "expectedCode": "<実際の失敗分類>"},
    "incompatible_version": {"argv": ["<バージョン不適合>", "..."], "errorPointer": "/...", "expectedCode": "<実際の失敗分類>"}
  },
  "provenance": {
    "lock": {"path": "実lockファイル", "sha256": "<独立に固定した64桁SHA256>", "composedDigestPointer": "/..."},
    "asset": {"path": "実assetファイル", "sha256": "<独立に固定した64桁SHA256>", "composedDigestPointer": "/..."},
    "build": {"path": "実build成果物", "sha256": "<独立に固定した64桁SHA256>", "composedDigestPointer": "/..."}
  }
}
```

この例の角括弧付き文字列は**説明上の未解決箇所であり実行可能なfixtureではありません**。#1の宣言形式、失敗分類、来歴の出力位置が決まる前に埋めて受入根拠へ流用しないでください。assetやbuildが複数物理ファイルの場合、現状の検査器では各種類につき代表ファイル1件しか検査しません。複数成果物の完全性は**未検証**と記録します。lockとbuildの意味、署名・信頼・現在権限の真正性もこのバイト照合だけでは確認できません。

## 判定内容

- **通常対照と合成**：指定JSON Pointer以下の実操作結果を、オブジェクトキー順に依存せず意味比較し、各操作を2回実行して選択結果の再現性を検査します。バイト一致を機能同等性の代わりにしません。
- **負例5種類**：未知部品、重複、依存欠落、循環、不適合バージョンごとに非ゼロ終了と、期待する機械可読の失敗分類の一致を検査します。
- **来歴3種類**：実際に生成されたlock/asset/buildのバイト列をSHA-256で照合し、合成結果JSONの指定ポインタがそのdigestを参照するか検査します。検査用の固定digestを出力結果から自動採用しません。
- **安全側の区別**：`PASS`=全項目の観測成功（exit 0）、`FAIL`=結果が期待に反する（exit 1）、`BLOCKED`=必須fixture/コマンド等が不足（exit 2）。実行されなかったPoCをPASSとしません。出力には生のコマンドstdout/stderrを含めず、必要最小限のハッシュと失敗分類を残します。

## 初回の検証記録

- 照合対象main：`128429482e02d96afe0123ee7da023256094dfca`（2026-10-14時点）
- 自己試験：`test_probe.py`の**10件が成功**（2026-10-14、Python 3.13.5、隔離した一時ディレクトリに置いた模擬コマンドを使い3組に分割実行）。外部コマンドの受理/拒否検出能力の限定的な試験です。
- GitHubのブランチ上に置いた同一ファイルのCI/Go実行、PoC Bの実builder、2 Go Service + 2 UI部品、実Host、実Web/Desktop、lock/asset/buildを含む**実物の合成照合は未実施**。#1の実装口を待つ項目は`BLOCKED`です。
- 現時点では実際の合成fixtureとコマンドが未提供のため、本検査器で製品のPoC BをPASSと判断してはいけません。#1の合成Contractが確定した後に差分を確認し、必要な最小限の拡張・負例fixtureを別途追加します。
