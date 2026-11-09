# reference-delivery-bundle（#2020）

解決済みのDelivery Profileとplugin descriptorを1つのzipへまとめるReferenceです。SIerが顧客先へplugin構成を持ち運ぶ、SaaSの配備単位を固定する、といった用途を想定します。配布・インストールは行いません。bundleの完全性はdetached signatureで確認できます。

```text
go run ./tools/reference-delivery-bundle/cmd/tei-delivery-bundle \
  pack -profile PROFILE -catalog PLUGIN_DIR -out bundle.zip

go run ./tools/reference-delivery-bundle/cmd/tei-delivery-bundle \
  verify -bundle bundle.zip
```

## detached signature

bundleのexact bytesに対するed25519のdetached signatureを作れます。鍵は利用者が用意し、TEIは鍵を保存・配布しません。

```text
go run ./tools/reference-delivery-bundle/cmd/tei-delivery-bundle \
  keygen -key private.pem -public public.pem

go run ./tools/reference-delivery-bundle/cmd/tei-delivery-bundle \
  sign -bundle bundle.zip -key private.pem -signature bundle.sig

go run ./tools/reference-delivery-bundle/cmd/tei-delivery-bundle \
  verify-signature -bundle bundle.zip -signature bundle.sig -public public.pem
```

`verify-signature`は、bundleの1 byteでも変われば失敗します。公開鍵の配布経路と信頼の確立は利用者の責任です。

## 内容

zipは次のentryを持ちます。

- `manifest.json`: パッケージ全体のファイル一覧とdigest（`reference-export-package`の形式）
- `files/bundle.json`: profile、plugin一覧、catalog digest、lock digest
- `files/<id>-<version>.json`: 選択したplugin descriptor

`catalogDigest`は、pluginのid・version・descriptor digestから計算します。`lockDigest`はDelivery Profileのlockと同じ形式のdigestです。

## 検証

- profileをcatalogで解決し、必須能力・依存・競合を検証する。
- 選択したplugin descriptorをzipへ含める。
- `verify`でパッケージのmanifestと、各plugin descriptorのdigestを確認する。
- 同じplugin idの重複を拒否する。

## インストール

署名を検証したbundleから、plugin descriptorをローカルcatalogへ配置できます。

```text
go run ./tools/reference-delivery-bundle/cmd/tei-delivery-bundle \
  install -bundle bundle.zip -signature bundle.sig -public public.pem \
  -catalog ./plugin-catalog
```

- bundleの署名とパッケージを検証してから書き込む。
- 書き込むのはplugin descriptorだけ。zip内の任意pathは展開しない。
- 同じdigestの既存fileはskipし、異なるdigestの既存fileは上書きせず失敗する。
- 既存fileとの衝突は、書き込み前に全件確認する。
- 書き込み中に失敗した場合、この呼び出しで作ったfileを削除する。削除にも失敗した場合は残ることがある。
- 書き込み後にcatalogを読み直し、重複や不正なdescriptorがないことを確認する。
- 複数fileの書き込みは完全なトランザクションではない。preflightとbest-effort rollbackで部分適用を抑える。

新規catalogを一括で作る場合は`-fresh`を使います。既存catalogを更新するのではなく、staging directoryへ全descriptorを書き、catalogとして検証してから一度のRenameで公開します。

```text
go run ./tools/reference-delivery-bundle/cmd/tei-delivery-bundle \
  install -fresh -bundle bundle.zip -signature bundle.sig -public public.pem \
  -catalog ./new-plugin-catalog
```

- 既存のcatalog directoryがある場合は拒否する。
- 失敗時はtarget directoryを残さない。
- 既存catalogとのmergeは`-fresh`なしの`install`を使う。preflightとbest-effort rollbackを持つが、完全なトランザクションではない。

## 配備状態とロールバック

`activate`は検証済みcatalogをactiveとしてstate fileへ記録し、直前のactiveをpreviousとして保持します。以前のactiveはhistoryへ積みます。

```text
go run ./tools/reference-delivery-bundle/cmd/tei-delivery-bundle \
  activate -catalog ./new-plugin-catalog -state ./deployment-state.json

go run ./tools/reference-delivery-bundle/cmd/tei-delivery-bundle \
  rollback -state ./deployment-state.json

go run ./tools/reference-delivery-bundle/cmd/tei-delivery-bundle \
  status -state ./deployment-state.json
```

- `activate`はcatalogを読み直し、空・不正・digest不一致を拒否する。historyは最大20世代を保持し、古い世代から捨てる。
- `rollback`はhistoryの最新catalogを検証してactiveへ戻す。historyが空の場合は、history導入前のstateと互換のためにpreviousを使う。
- `rollback`を繰り返すと、historyを新しい順に1段階ずつ戻る。
- `status`はactive、previous、historyのcatalogが現在も検証できるかを確認する。historyのcatalogが変更・削除されている場合は失敗する。
- stateはtemp fileへの書き込みとRenameで更新する。
- ロールバックはstateの入れ替えであり、サービスプロセスの起動・停止は行わない。

## ヘルスチェック

active deploymentに対して、利用者が用意したtrustedなコマンドを実行し、終了コードを記録します。

```text
go run ./tools/reference-delivery-bundle/cmd/tei-delivery-bundle \
  health -state ./deployment-state.json \
  -command 'your-service-health-check' -timeout 30s
```

- `TEI_CATALOG`、`TEI_ACTIVE_DIGEST`、`TEI_STATE`を環境変数で渡す。
- 終了コード0を`healthy=true`、それ以外を`healthy=false`とする。
- timeoutを超えた場合は`healthy=false`、`error=timeout`とする。
- 出力はdigestだけを記録し、本文はreceiptへ含めない。
- stateのactive/previousが現在も検証できることを前提にする。

## 起動と停止

trustedなサービスコマンドをバックグラウンドで起動し、PIDを記録します。

```text
go run ./tools/reference-delivery-bundle/cmd/tei-delivery-bundle \
  start -state ./deployment-state.json \
  -command 'your-service-command' -log ./service.log

go run ./tools/reference-delivery-bundle/cmd/tei-delivery-bundle \
  stop -state ./deployment-state.json

go run ./tools/reference-delivery-bundle/cmd/tei-delivery-bundle \
  stop -graceful -timeout 10s -state ./deployment-state.json

go run ./tools/reference-delivery-bundle/cmd/tei-delivery-bundle \
  restart -state ./deployment-state.json \
  -command 'your-service-command' -timeout 10s

go run ./tools/reference-delivery-bundle/cmd/tei-delivery-bundle \
  reconcile -state ./deployment-state.json \
  -command 'your-service-command' -health 'your-health-check' -timeout 10s

go run ./tools/reference-delivery-bundle/cmd/tei-delivery-bundle \
  supervise -state ./deployment-state.json \
  -command 'your-service-command' -health 'your-health-check' \
  -interval 30s -iterations 0
```

- `start`は`bash -c`でコマンドを起動し、PID・catalog・digest・開始時刻・ログpathを`<state>.pid`へ記録する。loginシェル（`bash -lc`）は使わないため、`~/.profile`等の読込は起動に含まれない。コマンドの解決は親プロセスの環境（`PATH`を含む）に依存する。外部のcron・timerから呼ぶ場合は、`PATH`が狭いことがあるため、コマンドを絶対pathで指定するか、実行元の環境で`PATH`を揃えること。
- `TEI_CATALOG`、`TEI_ACTIVE_DIGEST`、`TEI_STATE`を環境変数で渡す。
- `stop`は記録したPIDへ`Kill`を送り、PID fileを削除する。
- `stop -graceful`は`kill -TERM`を送り、timeoutまで終了を待ってから必要なら`Kill`する。アプリケーションのshutdown完了は検証しない。
- `restart`はgraceful stopの後に同じコマンドを起動する。起動準備・healthの完了は待たない。
- `reconcile`は1回だけ監督する。未起動ならstart、healthyならnoop、unhealthyならrestartし、stale PIDは除去する。外部のcronやtimerから呼ぶ。
- `supervise`は`reconcile`をintervalごとに繰り返す。`-iterations 0`は上限なし、Ctrl-Cで終了する。各回のreceiptをJSON Linesで出力する。
- 既にPID fileがある場合、二重起動として拒否する。`reconcile`はループせず、`supervise`はバックオフ・leader electionを行わない。
- このReferenceはLinux/WSLの`bash`と`kill`を前提とする。

## 守っている境界

- bundleは選択済みの構成を固定するだけで、pluginの取得・信頼・権限を付与しない。
- signatureはbundleの完全性だけを示す。署名者・公開鍵の信頼、配備経路の真正性は扱わない。
- `install`はローカルcatalogへの配置だけを行う。`activate` / `rollback`はstate fileの更新だけを行い、サービスを起動・停止しない。
- `health`は利用者が指定したtrustedなコマンドの終了コードを記録するだけで、サービスを起動・監視しない。
- `start` / `stop` / `restart` / `reconcile` / `supervise`はtrustedなコマンドの起動とPIDへのシグナルだけを行う。`stop -graceful`はSIGTERMを送るが、アプリケーションのshutdown完了、バックオフ、leader election、PID再利用の防止は行わない。
- catalogにないpluginを暗黙に追加しない。

## 未実装

- 鍵管理、信頼の起点、署名者の認証。
- 配布、バックオフ付きの継続監視、leader election、アプリケーションのshutdown完了検証、トランザクション。
- 部分bundle、差分更新。
- 実SaaS・実顧客先での配備確認。
