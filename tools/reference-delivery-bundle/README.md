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

`activate`は検証済みcatalogをactiveとしてstate fileへ記録し、直前のactiveをpreviousとして保持します。

```text
go run ./tools/reference-delivery-bundle/cmd/tei-delivery-bundle \
  activate -catalog ./new-plugin-catalog -state ./deployment-state.json

go run ./tools/reference-delivery-bundle/cmd/tei-delivery-bundle \
  rollback -state ./deployment-state.json

go run ./tools/reference-delivery-bundle/cmd/tei-delivery-bundle \
  status -state ./deployment-state.json
```

- `activate`はcatalogを読み直し、空・不正・digest不一致を拒否する。
- `rollback`はprevious catalogを検証してからactiveとpreviousを入れ替える。previousがない場合は拒否する。
- `status`はactiveとpreviousのcatalogが現在も検証できるかを確認する。
- stateはtemp fileへの書き込みとRenameで更新する。
- ロールバックは1段階の入れ替えであり、サービスプロセスの起動・停止・ヘルスチェックは行わない。

## 守っている境界

- bundleは選択済みの構成を固定するだけで、pluginの取得・信頼・権限を付与しない。
- signatureはbundleの完全性だけを示す。署名者・公開鍵の信頼、配備経路の真正性は扱わない。
- `install`はローカルcatalogへの配置だけを行う。`activate` / `rollback`はstate fileの更新だけを行い、サービスを起動・停止しない。
- catalogにないpluginを暗黙に追加しない。

## 未実装

- 鍵管理、信頼の起点、署名者の認証。
- 配布、サービスへのインストール・起動・停止・ヘルスチェック、多段階ロールバック、トランザクション。
- 部分bundle、差分更新。
- 実SaaS・実顧客先での配備確認。
