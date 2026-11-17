# 手順3で加える変更

参照用の作業領域の写しへ、次の三つを加える。

1. `tei.workspace.yaml`の末尾へ、設計文書の言語を足す。

   ```yaml
   languages:
     - tei.authoring/v1
   ```

2. `unit-price.tei.md`を`semantic/sales/`へ置く。
3. `applications/order-entry/application.tei`の`uses validation`の並びへ、一行足す。

   ```text
   uses validation sales.order_line.unit_price_positive
   ```

触るファイルは三つである。通常の開発と比べるときは、この数を使う。
