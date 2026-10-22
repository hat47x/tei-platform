# XeadServer live harness

`xtea_server.py`のHTTP契約を、模擬サーブレットではなく元の`DBMethod` / `Service`クラスで確認するための最小ハーネスです。

- 元の`XeadServer/WEB-INF/src/xeadServer/*.java`をコンパイルします。
- 埋め込みJetty 9.4で`/xteaServer/DBMethod`と`/xteaServer/Service`を配備します。
- DBMethodは使い捨てのH2ファイルDBを使います。`H2Init`が`T (ID, NAME)`だけを作ります。
- Serviceは`driver-fixture.ps1`で作ったSkeletonの複製を使い、`AF200J`などの参照系Functionを実行します。
- `DBMethod`と`Service`は`.xeaf`の`System`要素から接続情報を読むため、JNDIやTomcatは不要です。

### ビルド時にだけ適用する互換差分

上流ファイルは変更しません。`build.ps1`が`build\server-compat-src`へコピーし、次の2点だけを置換してコンパイルします。

1. `Service.java`の`com.sun.org.apache.xml.internal.serialize`を、X-TEA同梱のXerces `org.apache.xml.serialize`へ置換する。JDK 11で削除された`OutputFormat(Document)`を避けるためです。
2. `Session.java`の`userID.matches("[0-9a-zA-Z]+")`を`user.matches("[0-9a-zA-Z]+")`へ置換する。上流`7947b52`では、`userID = user`の代入より前に`userID`を検査するため、初回認証が必ず失敗します。

2は認証条件の挙動変更です。したがって、このハーネスのService成功は「認証バグを明示的に補った状態で、元のService本体とFunction実行経路が動く」ことの証拠であり、上流Serviceの認証が正しいことの証拠ではありません。

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
$env:XTEA_SKELETON_HOME = "C:\path\to\X-TEA\Driver\Skeleton"
python tests\test_live_server.py
python tests\test_live_service.py
python tests\test_tomcat_war.py
```

`test_live_server.py`は一時H2 DBを作り、`SELECT` / `SELECT COUNT` / `INSERT` / `UPDATE` / `DELETE` / `COMMIT` / `ROLLBACK` / `IP`を元の`DBMethod`へHTTPで送ります。

`test_live_service.py`はSkeletonの一時複製を作り、`Service`へ`FUNCTION=AF200J`、`REQUEST_TYPE=ListSections`を送ります。2件の部門データがJSONで返ることを確認します。

`test_tomcat_war.py`は`build-war.ps1`で`WEB-INF/web.xml`付きのWARディレクトリを作り、埋め込みTomcat 9.0へ配備します。`DBMethod`の状態取得と`Service`の`AF200J`をHTTPで確認します。WARは生成物であり、リポジトリへコミットしません。

## 境界

- Serviceの認証条件は、上記のとおりビルド時に明示的に補っています。認証の正しさ、実ユーザー、認可は未確認です。
- H2 1.4.187の合成テーブルとSkeletonの複製だけを使い、実業務DBには接続しません。
- 埋め込みTomcatへのWAR配備は確認しますが、外部Tomcat、JNDI DataSource、DB製品別ドライバ、本番運用は対象外です。
