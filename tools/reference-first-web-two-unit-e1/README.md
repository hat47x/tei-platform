# 第一Web Target：BrowserとLinux/APIの二成果物（E1局所実証）

Issue [#1182](https://github.com/hat47x/tei-platform/issues/1182)の最初の**物理成果物**による検証です。同じ`order_entry`、固定されたHandover `B/R`、Target build `T=実checkout HEAD`を引数とし、次の二つを別の場所に生成します。

```text
artifact-root/
  browser/index.html    ← 実第一Web Targetの /custom-submit から生成済みHTMLを採取
  api/order-entry       ← Go: -tags=tei_order_only のLinux/amd64実バイナリ
  measurement.json      ← Source/B/R/T、成果物digest/bytes、未検証範囲
  smoke-observation.json ← 任意のHTTP疎通診断（smoke.shを実行した場合）
```

Browser HTMLは実際の`workspace.Load → Driver → Projection → template`から派生し、当Reference内でCanonical意味や認可Policyを再定義しません。Goバイナリは第一Web Target既存の業務Runner、Validation、Legacy Authentication、Authorization、Business Result/Execution Error/Observationを使用します。生成したHTMLをサーバー側と物理的に分けても、BrowserへGoのDB ProviderやWorkspace Loaderのソースを配布する必要はありません。ただしブラウザ成果物の静的点検だけでは任意のsecret不存在や外部依存全体の安全性を証明しません。

## 手順

信頼できるclean checkoutのRepositoryルートで実行します。既存の受理済みbaselineとして記録されている`R`を明示します（**引数による指定だけではAcceptance Authorityの証明にはなりません**）。

```sh
destination="/tmp/tei-two-unit-$(git rev-parse --short HEAD)"
bash tools/reference-first-web-two-unit-e1/build.sh \
  "$destination" B e5231839a5dcc62cce1181a2731fcb7c70d16736

# build済みの二成果物を、実際のGoプロセス＋検証専用Node Hostで接続
bash tools/reference-first-web-two-unit-e1/smoke.sh "$destination"
```

`browser-host.mjs`は成果物ではなく、**別々の実体を同じloopback Originで接続する診断専用のHost**です。外部公開も配布もしてはいけません。静的HTMLはGET `/`と`/custom-submit`でそのまま返し、`GET /manifest.json`、`GET /scenario/read`、`POST /scenario/submit`、`POST /custom-submit`のみを厳密なloopback APIへ中継します。未知の経路は404、Origin不一致は403、接続不能なAPIには502を返し、業務成功を作りません。Browserから送られたHost/Role/Sessionヘッダーを信用して業務認可を与えません。Content-Security-Policyは外部スクリプトと他Originへのform送信を禁止します。

`smoke.sh`は実ファイルのSHA-256照合、Browser静的GETと元ファイルのbytes一致、API manifestの同一性、フォーム経由の合成業務結果とAPI直接実行結果のstatus/body一致、Origin不一致の拒否、API停止時にもBrowserが残ることとAPI未到達が成功にならないことを検査します。実ブラウザ（Playwright等）の操作、実DB Provider、実IdP/本番認証・TLS、CSRF総合対策、cookie/session、分散配置、Visual Fidelityは**検証しません**。ブラウザ画面が実在することとBrowser Functional Evidenceを混同しないでください。

### 証拠の高さ

- `measurement.json`: 二つの**実際に生成した**物理成果物のbytes/SHA-256、同一checkoutのSource `T`、呼出し側指定のHandover `B/R`、未検証の領域を記録。正式なBuild Artifact Lockや署名ではない。
- `smoke-observation.json`: 合成Reference Credentialをruntime inputとしてHTTPにのみ使用し、レスポンスstatus/bodyの一致を検査。資格情報自体とレスポンスのrawデータは証跡に保存しない。HTTP応答が一致しても本番Authorizeが成立したことにはならない。
- 入力に使用した`reference/workspace`と開発用認証Fixtureは、Linux/API側が実行時に参照します。この実証はCompiler/Workspace/Providerが**API実行ファイルから完全に除外された**ことを保証するものではありません。
- Buildとsmokeは同一Source版の実checkoutで実行する。Targetの`T`と受理済み`R`は別々に保存し、Gate C当時の`T1`と混同しない。
- 保守する二成果物は **Browser HTMLファイル**と**Linux/API実行ファイル**。診断Host、合成Fixture、測定JSONは配布用成果物として数えない。

## 残る受入条件

本ReferenceはE1の**物理二成果物と合成フォームHTTP経路の局所実証**に限定します。同一Accepted Applicationでの全Scenario（認証拒否・認可拒否・Validation・Unknownを含む）とGate Cとの差分、実Browser replay/Visual、Provider/Authorization/Secretの負例、正式なBuilder/Lockへの接続は#1182で継続します。第二Target/Hostの#259を完了にしません。
