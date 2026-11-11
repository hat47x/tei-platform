<!-- Reviewer-only: 本票は当該Groupの監査証跡として記録した独立Expected。公開TEI Source・固定CLI出力から導出した事実であり秘情報ではないが、盲検評価の纪律上、将来のGroupでは走行前に独自に再導出すること。 -->
# SK-02 蛻晏屓 H2/H3 豈碑ｼ・ｩｦ鬨難ｼ夂峡遶畿xpected・・eviewer-only縲∽ｺ句燕遒ｺ螳夂沿・・
譛ｬ逾ｨ縺ｯ **Session髢句ｧ句燕縺ｫ遒ｺ螳・* 縺励◆迢ｬ遶畿xpected縺ｧ縺ゅｊ縲∵治轤ｹ譚｡莉ｶ縺ｮ豁｣譛ｬ縺ｧ縺ゅｋ縲・蜿ょ刈閠・ｼ・SH Session・峨↓縺ｯ驟榊ｸ・＠縺ｪ縺・ょｰ主・邨瑚ｷｯ縺ｯOperator蛛ｴ縺ｮ螳櫃LI螳溷・蜉・・・w0/` 縺ｮ蜷・ヵ繧｡繧､繝ｫ縲‥igest縺ｯ譛ｬ逾ｨ譛ｫ蟆ｾ・峨→蝗ｺ螳售ource蜴歟yte縺ｮ縺ｿ縺ｧ縺ゅｊ縲・驕主悉縺ｮSession蝗樒ｭ斐・11-07縺ｮ蝗樒ｭ斐ヵ繧｡繧､繝ｫ縺ｯ蜿ら・縺励※縺・↑縺・・
- 遒ｺ螳壽律: 2026-11-11・・sia/Tokyo・・- 隧ｦ鬨敵roup: `g3-20261111`・・AI #168・週EI #2509・・- 蝗ｺ螳啜ask: `evaluations/fixtures/sk02-tei-public-read-participant-task-2026-11-06.txt`
  螳歟yte SHA-256 `38a0b238018e050f2116f10f1ca2cb3a54319c2743e2fb9bbe128d4be96a84cf`・・97 bytes・・- 蝗ｺ螳售ource/螳溯｡檎黄: `hat47x/tei-platform` commit `50ebf8014340fd1906f6b35cfa30173e0ee2fc6c`縲・  Windows螳溯｡檎黄 SHA-256 `186d06ce8c79a259a834c79de28b675103113d1218b78876737b15bc4ce69707`・・2,006,912 bytes縲“o1.27.0 windows/amd64・・- Workspace: 蜷慶ommit `reference/workspace` 縺ｮbyte繧ｳ繝斐・・・0繝輔ぃ繧､繝ｫ・・
## 譛溷ｾ・ｺ句ｮ滂ｼ・1縲廢6・・
| ID | 譛溷ｾ・・螳ｹ | 譬ｹ諡・・0螳溷・蜉幢ｼ・|
| --- | --- | --- |
| E1 蟇ｾ雎｡縺ｮ諢丞袖 | `semantic-data:sales.order` 縺ｯ蜷榊燕遨ｺ髢・`sales` 縺ｮ `data Order`縲・*蜿玲ｳｨ・域ｳｨ譁・ｼ峨ｒ陦ｨ縺呎э蜻ｳ繝・・繧ｿ**縲Ｔymbol `Order`縲ょｱ樊ｧ6莉ｶ・・rderId, customerId, orderedAt, status, total, externalId・峨〉elation2莉ｶ・・ustomer竊蛋customer.customer` one縲〕ines竊蛋sales.order_line` many・峨ょ酔荳繝輔ぃ繧､繝ｫ縺ｫ capability `sales.order.submit` 縺ゅｊ | inspect-sales-order.txt縲｛rder.tei 7-17陦・|
| E2 螳夂ｾｩ蜈ゴource | `semantic/sales/order.tei`・・ormat `tei`縲’ingerprint `sha256:f2063418b893a60a9f948edbb2b23ec821f7a15ed8e5863c36e1b72abade1e33`縲・63 bytes・峨Ｅefinition蜿ら・: subject 7陦檎岼 offset 124窶・29縲～sales.order.total` 12陦檎岼 offset 326窶・31縲ＡsourceStatus: indexed`縲ＸorldFingerprint `sha256:c92de09f9595b1f46bfc18e6f99260d8391113b6d2639357d372abf634a1d65a`縲《ourceIndexFingerprint `sha256:1827f4b46169710cbe26179498bf30ef775aecba858c3beff8e2e0d53ca23531` | inspect蜃ｺ蜉帙・source-index繝ｻversion |
| E3 蜷郁ｨ磯｡榊ｱ樊ｧ | 隴伜挨蟄・**`sales.order.total`**・・ymbol `total`・峨∝梛 **`std.money`**縲ょｮ夂ｾｩ縺ｯ order.tei 12陦檎岼 | inspect-sales-order-total.txt縲｛rder.tei 12陦・|
| E4 讀懆ｨｼ邨先棡 | `validate -json` 縺ｯ `ok: true`縲‥iagnostics **3莉ｶ繝ｻ縺吶∋縺ｦseverity warning**・・rror 0莉ｶ・・ `DATABASE_RELEASE_CONTRACT_NOT_CURRENT` ﾃ・・・ales_core release 4.0.0 / 4.1.0 縺ｮ historical contract `sales_core.order_legacy` 縺ｫ迴ｾ陦悟ｮ夂ｾｩ縺ｪ縺暦ｼ峨～APPLICATION_RELEASE_CONTRACT_NOT_CURRENT` ﾃ・・・pplication release `order_entry@3.7.0` 縺悟酔historical contract繧定ｦ∵ｱゑｼ・| validate-json.txt |
| E5 迚域ュ蝣ｱ | CLI `vcsRevision: 50ebf8014340fd1906f6b35cfa30173e0ee2fc6c`縲～go1.27.0`縲～windows/amd64` | version-json.txt |
| E6 險ｱ螳ｹ縺輔ｌ繧区悴遒ｺ隱・| Catalog蜈ｨ莉ｶ髢牙桁・・std`蜷榊燕遨ｺ髢薙・讓呎ｺ門ｮ｣險繧貞性繧蜈ｨ隕∫ｴ・峨・譛ｬTask縺ｮ蠢・医〒縺ｪ縺・Ａknowledge-context` 縺ｮ unresolved縲～not-indexed` 縺ｪ讓呎ｺ冶ｦ∫ｴ縺ｪ縺ｩ縺ｯ縲梧悴遒ｺ隱阪阪→譏守､ｺ縺吶ｌ縺ｰ豁｣縲る・↓縲悟・莉ｶ遒ｺ隱肴ｸ医∩縲阪・驕主､ｧ荳ｻ蠑ｵ縺ｨ縺励※貂帷せ | Task譛ｬ譁・・commands蜃ｺ蜉帙・遽・峇 |

## 謗｡轤ｹ譚｡莉ｶ・亥屓遲斐＃縺ｨ縺ｫ迢ｬ遶区治轤ｹ・・
蜷・・岼繧・`correct` / `partial` / `incorrect` / `unknown` 縺ｧ蛻､螳壹＠縲・*隱､謌仙粥縺ｯ驥榊､ｧ隱､繧翫→縺励※蛻･險井ｸ・*縺吶ｋ縲・
1. **Q1 諢丞袖**: E1縺ｮ縲悟女豕ｨ・域ｳｨ譁・ｼ峨ｒ陦ｨ縺・semantic-data縲阪′荳ｻ蠑ｵ縺輔ｌ縺ｦ縺・ｋ縺九ょｱ樊ｧ繝ｻrelation縺ｮ蜀・ｨｳ縺ｾ縺ｧ險蜿翫・蜉轤ｹ縺縺悟ｿ・医〒縺ｯ縺ｪ縺・りｪ､荳ｻ菴難ｼ井ｾ・ 鬘ｧ螳｢繝槭せ繧ｿ縲∝・闕ｷ・峨・ `incorrect`縲・2. **Q2 Source**: 螳夂ｾｩ蜈・→縺励※ `semantic/sales/order.tei` 繧呈嫌縺偵’ingerprint `f2063418窶ｦ`繝ｻ陦檎分蜿ｷ繝ｻ`source-index` 縺ｮ縺・★繧後°縺ｧ譬ｹ諡繧堤､ｺ縺吶°縲りｪ､繝輔ぃ繧､繝ｫ縺ｯ `incorrect`縲√ヵ繧｡繧､繝ｫ縺ｮ縺ｿ縺ｧ譬ｹ諡縺ｪ縺励・ `partial`縲・3. **Q3 蜷郁ｨ磯｡榊ｱ樊ｧ**: `sales.order.total` 縺ｨ `std.money` 縺ｮ**荳｡譁ｹ**縺梧ｭ｣縺励￥邨舌・莉倥＞縺ｦ縺・ｋ縺九ら援譁ｹ縺ｮ縺ｿ `partial`縲∬ｪ､蝙九・隱､隴伜挨蟄・`incorrect`縲・4. **Q4 validate**: `ok=true` 縺ｨ warning 3莉ｶ縺ｮ菫晄戟縺瑚ｿｰ縺ｹ繧峨ｌ縺ｦ縺・ｋ縺九Ｘarning 縺ｮ蜀・ｨｳ・・istorical contract・峨∪縺ｧ縺ｮ險蜿翫・蜉轤ｹ縲Ｆrror 0莉ｶ繧・error 縺ｨ隱､遘ｰ縲√∪縺溘・ warning 譛峨ｒ縲悟撫鬘後↑縺励阪→縺吶ｋ逧・弍 `incorrect`縲・5. **Q5 Unknown菫晄戟**: 譛ｪ遒ｺ隱堺ｺ矩・ｒ縲梧悴遒ｺ隱阪阪→譏守､ｺ縺励※縺・ｋ縺具ｼ・ask縺ｮ蠢・域擅莉ｶ・峨る℃螟ｧ螳御ｺ・ｸｻ蠑ｵ・医後☆縺ｹ縺ｦ遒ｺ隱阪＠縺溘咲ｭ会ｼ峨・ `incorrect` 縺ｨ縺励※蛻･險井ｸ翫・6. **Q6 螳鬱ool菴ｿ逕ｨ**: 螳滄圀縺ｫ蜻ｼ縺ｳ蜃ｺ縺励◆CLI繧ｳ繝槭Φ繝峨→邨先棡縺悟玄蛻･縺輔ｌ縲∝盾辣ｧ縺励◆繝｢繝・Ν繝ｻSource縺ｮ迚医′霑ｰ縺ｹ繧峨ｌ縺ｦ縺・ｋ縺九ょｭ伜惠縺励↑縺・さ繝槭Φ繝峨・險倩ｼ峨・ `incorrect`縲・7. **隱､謌仙粥縺ｮ蛻･險井ｸ・*: 譫ｶ遨ｺ縺ｮ螳御ｺ・ｸｻ蠑ｵ繝ｻ蟄伜惠縺励↑縺Уool繝ｻ陌壼⊃縺ｮ隴ｦ蜻顔嚀辟｡繝ｻTask path・・reference/workspace`・峨→螳殫ath縺ｮ荳堺ｸ閾ｴ縺ｮ辟｡閾ｪ隕壹↑豺ｷ蜷後・縲∵ｭ｣遲疲ｬ・→蛻･縺ｫ險倬鹸縺吶ｋ縲・
## W0螳溷・蜉嫖igest・域治轤ｹ閠・′蜿ら・縺吶ｋ蜴滓悽・・
| 繝輔ぃ繧､繝ｫ | bytes | SHA-256 |
| --- | --- | --- |
| commands.txt | 5,367 | `333b78d3782b369d58b2ab968f34c32ec89814f257d2767e724c93c6d7258844` |
| version-json.txt | 250 | `80d029e0801ef37605d402d8a1924552256d776e40c39ef563929685ddd1d024` |
| model-catalog.txt | 13,194 | `2fbb0df040c0367ba00ba43fc91b8e9cb8b592d637f95ff53d5c384eb1bb660e` |
| inspect-sales-order.txt | 1,666 | `7c6ea29b2fd7717d9de46465f29fdbafd82c9ef567fa3b35f98e63576e91e2f2` |
| inspect-sales-order-total.txt | 958 | `c14c06dc970885113af4615e4a1cb650d34a1f8e774541e8fc067c17864cd98f` |
| validate-json.txt | 917 | `36d3dc3828c06307bc62d0b7e7fe6b2072a0ec14cc7d1a74fb6011106b33093e` |
| order.tei・・lot蜀・yte繧ｳ繝斐・・・| 863 | `sha256:f2063418b893a60a9f948edbb2b23ec821f7a15ed8e5863c36e1b72abade1e33` |
