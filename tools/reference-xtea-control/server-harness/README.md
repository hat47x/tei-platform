# XeadServer live harness

`xtea_server.py`のHTTP契約を、模擬サーブレットではなく元の`DBMethod`クラスで確認するための最小ハーネスです。

- 元の`XeadServer/WEB-INF/src/xeadServer/*.java`をコンパイルします。`Service.java`はJDK 11で削除された`com.sun.org.apache.xml.internal.serialize.OutputFormat(Document)`に依存するため除外します。
- 埋め込みJetty 9.4で`/xteaServer/DBMethod`だけを配備します。
- DBは使い捨てのH2ファイルDBです。`H2Init`が`T (ID, NAME)`だけを作ります。
- `DBMethod`は`.xeaf`の`System`要素から`DatabaseName`などを読むため、JNDIやTomcatは不要です。

## 前提

- `XeadServer`のクローン
- `xteadrv.jar`、`h2-1.4.187.jar`などを含むX-TEA Driverのランタイムディレクトリ
- JDK 11とPowerShell

## ビルド

```powershell
.\server-harness\build.ps1 `
  -XteaSource C:\path\to\XeadServer `
  -RuntimeHome C:\path\to\X-TEA\Driver
```

`build\server-classpath.txt`に実行用classpathができます。

## 試験

```powershell
$env:XTEA_SERVER_SOURCE = "C:\path\to\XeadServer"
$env:XTEA_RUNTIME_HOME = "C:\path\to\X-TEA\Driver"
python tests\test_live_server.py
```

試験は一時H2 DBを作り、`SELECT` / `SELECT COUNT` / `INSERT` / `UPDATE` / `DELETE` / `COMMIT` / `ROLLBACK` / `IP`を元の`DBMethod`へHTTPで送ります。

## 境界

- `Service`サーブレットは未配備・未試験です。
- H2 1.4.187の合成テーブルだけを使い、Skeletonや実業務DBには接続しません。
- XeadServerのWAR配備、JNDI DataSource、Tomcat、DB製品別ドライバ、認可・本番運用は対象外です。
