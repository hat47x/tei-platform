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
- 書き込み後にcatalogを読み直し、重複や不正なdescriptorがないことを確認する。
- 複数fileの書き込みはトランザクションではない。途中で失敗した場合、それまでのfileは残る。

## 守っている境界

- bundleは選択済みの構成を固定するだけで、pluginの取得・信頼・権限を付与しない。
- signatureはbundleの完全性だけを示す。署名者・公開鍵の信頼、配布経路の真正性は扱わない。
- `install`はローカルcatalogへの配置だけを行う。サービスへの配備、起動、ロールバックは行わない。
- catalogにないpluginを暗黙に追加しない。

## 未実装

- 鍵管理、信頼の起点、署名者の認証。
- 配布、サービスへのインストール、ロールバック、トランザクション。
- 部分bundle、差分更新。
- 実SaaS・実顧客先での配備確認。
