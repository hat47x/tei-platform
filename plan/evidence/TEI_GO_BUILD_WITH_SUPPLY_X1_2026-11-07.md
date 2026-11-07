# 萓帷ｵｦ縺九ｉ縺ｮ螳檻uild縺ｨ縲ヾource髢牙桁雉・ｼ縺ｮ鬟溘＞驕輔＞・・1・・
[#2831](https://github.com/hat47x/tei-platform/issues/2831) 縺ｮSB-B莠句燕遒ｺ隱阪→縺励※縲・*螳歛rtifact縺悟・縺ｦ繧４ource髢牙桁縺ｮ雉・ｼ縺ｯ謌千ｫ九＠縺ｪ縺・*縺ｨ縺・≧鬟溘＞驕輔＞繧偵∝ｮ滄圀縺ｫ襍ｰ繧峨○縺ｦ險倬鹸縺励∪縺吶つｧ35.2縺ｮ雋萓宜1縺ｫ蟇ｾ蠢懊＠縺ｾ縺吶・
**縺薙ｌ縺ｯBuild縺ｮ螳溯｡瑚ｨ倬鹸縺ｧ縺ゅ▲縺ｦ縲∥rtifact縺ｮ蜿礼炊縺ｧ縺ｯ縺ゅｊ縺ｾ縺帙ｓ縲・* 莠悟屓Build縺ｮ蜀咲樟諤ｧ繧ゅ∽ｾ帷ｵｦ蜈・・Trust繧ゅ・哩蛹・婿驥昴・謗｡逕ｨ繧ょ性縺ｿ縺ｾ縺帙ｓ縲・4縺ｯ`UNQUALIFIED`縺ｮ縺ｾ縺ｾ縺ｧ縺吶・
## 蜑肴署

[SB-0a](TEI_GO_MODULE_CLOSURE_PREFLIGHT_2026-11-07.md)縺ｧ縲～cmd/tei`縺ｯ`GOPROXY=off`縺ｮ髫秘屬隕ｳ貂ｬ縺ｧ縺ｯ`missing-module`縺ｧ縺励◆縲・SB-0b](https://github.com/hat47x/tei-platform/pull/3142)縺ｧ縲∝ｮ滄∈謚・2莉ｶ縺形go.sum`縺ｮ`h1`縺ｨ荳閾ｴ縺励∽ｾ帷ｵｦ縺縺代〒繧ｪ繝輔Λ繧､繝ｳ蜀崎ｧ｣豎ｺ縺ｧ縺阪ｋ縺薙→繧堤｢ｺ隱阪＠縺ｾ縺励◆縲・
谿九▲縺ｦ縺・◆縺ｮ縺ｯ縲・*蜀崎ｧ｣豎ｺ縺ｧ縺阪ｋ**縲阪→縲・*雉・ｼ縺ゅｋ髢牙桁縺梧・遶九☆繧・*縲阪・髢薙・谿ｵ蟾ｮ縺ｧ縺吶よ悽菴懈･ｭ縺ｯ縺昴・谿ｵ蟾ｮ繧貞ｮ檻uild縺ｧ貂ｬ繧翫∪縺励◆縲・
## 荳峨▽繧貞挨谺・〒霑斐☆

霑ｽ蜉縺励◆`tools/reference-go-build-with-supply`縺ｯ縲∝酔縺伜ｯｾ雎｡縺ｫ縺､縺・※谺｡繧貞挨縲・↓霑斐＠縺ｾ縺吶・
| 谺・| 諢丞袖 |
| --- | --- |
| `buildSucceeded` / `artifactDigest` | 髫秘屬迺ｰ蠅・〒Build縺檎ｵゆｺ・＠縲∥rtifact縺後〒縺阪◆ |
| `sourceClosureState` | 蜷後§adapter縺ｮ`ResolveSourceClosure`縺悟酔縺伜ｯｾ雎｡繧蛋qualified`縺ｨ縺吶ｋ縺義rejected`縺ｨ縺吶ｋ縺・|
| `artifactProducedDespiteUnqualifiedClosure` | 荳贋ｺ後▽縺碁｣溘＞驕輔▲縺ｦ縺・ｋ縺薙→縺ｮ譏守､ｺ |

荳峨▽逶ｮ繧堤峡遶九・逵溷⊃蛟､縺ｫ縺励◆縺ｮ縺ｯ縲・*縲径rtifact縺後≠繧九阪ｒ縲慶losure縺瑚ｳ・ｼ繧呈戟縺､縲阪→隱ｭ縺ｿ譖ｿ縺医↑縺・*縺溘ａ縺ｧ縺吶・
迺ｰ蠅・・`BuildPackage`縺ｨ蜷後§deny-by-default・・CGO_ENABLED=0`縲～GOENV=off`縲～GOEXPERIMENT=`縲～GOFLAGS=`縲～GOSUMDB=off`縲～GOTOOLCHAIN=local`縲～GOVCS=*:off`縲～GOWORK=off`・峨↓縲～GOPROXY=file://<萓帷ｵｦ>`縺ｨ隧ｦ陦後＃縺ｨ縺ｮ`GOCACHE`/`GOMODCACHE`/`GOPATH`/`HOME`/`TMP*`繧帝㍾縺ｭ縺ｾ縺吶・*Host cache縺ｯ邯呎価縺励∪縺帙ｓ縲・*Go螳溯｡後ヵ繧｡繧､繝ｫ縺ｯ荳蠎ｦ縺縺題ｧ｣豎ｺ縺励※digest繧定ｨ倬鹸縺励∝ｾ後°繧臼ATH縺悟､峨ｏ縺｣縺ｦ繧りｨ倬鹸縺悟ｮ滉ｽ薙ｒ謖・☆繧医≧縺ｫ縺励※縺・∪縺吶・
## 隕ｳ貂ｬ邨先棡

- 迺ｰ蠅・ WSL2 Ubuntu 22.04縲；o `1.27.3`・・inux/amd64・・- 蟇ｾ雎｡checkout: `388412958`・・origin/main`・・- 萓帷ｵｦ: `/home/hat47x/go/pkg/mod/cache/download`・・o閾ｪ霄ｫ縺ｮdownload cache蠖｢蠑擾ｼ・- 蟇ｾ雎｡: `./cmd/tei`

| 谺・| 蛟､ |
| --- | --- |
| `buildSucceeded` | `true`・・xit 0・・|
| `artifactDigest` | `sha256:a512ffba1e87beb63779d6b592bb110b5d1e49e5ccc07fb9c006d9d62fc74efd` |
| `sourceClosureState` | `rejected` |
| `artifactProducedDespiteUnqualifiedClosure` | `true` |
| `goExecutableDigest` | `sha256:5f5eb6d0b2fb6aa0bb5d5f8c5f1c944e0de955c925f0111af92a9d8f79680b5b` |

closure縺ｮ諡貞凄逅・罰縺ｯ`module lookup disabled by GOPROXY=off`縺ｧ縺励◆縲ＡResolveSourceClosure`閾ｪ霄ｫ縺形GOPROXY=off`縺ｮ髫秘屬迥ｶ諷九〒襍ｰ繧九◆繧√∽ｾ帷ｵｦ縺ｯ貂｡縺｣縺ｦ縺・∪縺帙ｓ縲・*萓帷ｵｦ繧呈ｸ｡縺励※繧ＡResolveSourceClosure`縺ｯ`moduleDir != root`縺ｧ螟夜ΚModule譬ｹ繧呈拠蜷ｦ縺吶ｋ**縺ｮ縺ｧ縲√←縺｡繧峨・蛛懈ｭ｢逅・罰縺ｧ繧りｳ・ｼ縺ｯ謌千ｫ九＠縺ｾ縺帙ｓ縲ら炊逕ｱ縺ｮ蠖｢縺ｯ螟峨ｏ縺｣縺ｦ繧Ａrejected`縺ｧ縺ゅｊ邯壹￠繧九％縺ｨ繧定ｩｦ鬨薙′蝗ｺ螳壹＠縺ｦ縺・∪縺吶・
## 隗｣驥・
**繧上°縺｣縺溘％縺ｨ**・・
- `cmd/tei`縺ｯ縲・囈髮｢迺ｰ蠅・→`go.sum`荳閾ｴ縺ｮ萓帷ｵｦ縺縺代〒**螳滄圀縺ｫ繝薙Ν繝峨〒縺阪ｋ**縲４B-B縺ｮ縲悟ｮ歛rtifact繧剃ｽ懊ｋ縲榊・縺ｮ蜑肴署縺ｯ貅縺溘○繧九・- 蜷後§蟇ｾ雎｡縺ｮSource髢牙桁縺ｯ縲∝酔縺和dapter縺ｧ縺ｯ**雉・ｼ繧定ｿ斐＆縺ｪ縺・*縲９1・・rtifact謌仙粥縺ｨqualified closure縺ｮ豺ｷ蜷鯉ｼ峨・迴ｾ螳溘・蜊ｱ髯ｺ縺ｧ縺ゅｊ縲∵ｬ・ｒ蛻・￠繧句ｿ・ｦ√′縺ゅｋ縲・
**荳ｻ蠑ｵ縺励↑縺・％縺ｨ**・・
- 縺薙・artifact digest縺ｯ蜿礼炊貂医∩縺ｧ繧る・蛯吝庄閭ｽ縺ｧ繧ゅ≠繧翫∪縺帙ｓ縲ゆｺ悟屓Build縺ｮ蜀咲樟諤ｧ縺ｯ譛ｪ遒ｺ隱阪〒縺吶・- 縲慶losure縺梧拠蜷ｦ縺輔ｌ縺溘°繧餌rtifact縺檎┌蜉ｹ縲阪〒繧ゅ径rtifact縺後≠繧九°繧営losure縺梧・遶九阪〒繧ゅ≠繧翫∪縺帙ｓ縲ゆｺ後▽縺ｯ迢ｬ遶九〒縺吶・- 萓帷ｵｦ蜈・・Trust繝ｻ鄂ｲ蜷阪・逋ｺ陦瑚・・譛ｪ讀懆ｨｼ縺ｧ縺吶・- Linux amd64縺ｧ縺ｮ1蝗槭・隕ｳ貂ｬ縺ｧ縺吶８indows縺ｧ繧ょ酔縺倩ｩｦ鬨薙′騾壹ｋ縺薙→繧貞挨騾皮｢ｺ隱阪＠縺ｦ縺・∪縺吶′縲∥rtifact digest縺ｮOS髢謎ｸ閾ｴ縺ｯ荳ｻ蠑ｵ縺励∪縺帙ｓ縲・
## 讀懆ｨｼ

- `gofmt -l tools/reference-go-build-with-supply`・壼ｷｮ蛻・↑縺・- `go vet ./tools/reference-go-build-with-supply`・啀ASS
- `go test ./tools/reference-go-build-with-supply -count=1`・啀ASS・・莉ｶ縲９1縲∬・蟾ｱ螳檎ｵ仙ｯｾ雎｡縺ｮ豁｣萓九∫腸蠅・′Host cache繧堤ｶ呎価縺励↑縺・％縺ｨ縲’ile URL縺ｮvolume菫晄戟・・- `go test ./tools/repository-plan-check -count=1`・啀ASS

**NOT RUN**・壻ｺ悟屓Build縺ｮ蜀咲樟諤ｧ縲∽ｺ悟屓縺ｮ迢ｬ遶玖ｩｦ陦後〒縺ｮdigest荳閾ｴ縲…losure譁ｹ驥昴・螟画峩・・#1090](https://github.com/hat47x/tei-platform/issues/1090)縺ｮ謇譛芽・愛譁ｭ・峨ヾtage豈碑ｼ・ｼ・B-C・峨，I縲》rusted full checkout縲・
## 谺｡縺ｮ蛻､譁ｭ轤ｹ

`cmd/tei`繧担B-B縺ｮ蟇ｾ雎｡縺ｫ縺吶ｋ縺ｪ繧峨・*Source髢牙桁縺ｮ螟夜ΚModule譁ｹ驥・*繧偵←縺・桶縺・°繧貞・縺ｫ豎ｺ繧√ｋ蠢・ｦ√′縺ゅｊ縺ｾ縺吶よ悽菴懈･ｭ縺ｯ縺昴・譁ｹ驥昴ｒ螟峨∴縺ｦ縺・∪縺帙ｓ縲ょ､峨∴繧九°縺ｩ縺・°縺ｯ[#1090](https://github.com/hat47x/tei-platform/issues/1090)縺ｮ謇譛芽・・蛻､譁ｭ縺ｧ縲～Rejected`縺ｮ縺ｾ縺ｾ騾ｲ繧√ｋ驕ｸ謚櫁い・・rtifact縺ｨclosure繧貞挨Claim縺ｨ縺励※謇ｱ縺・ｶ壹￠繧具ｼ峨ｂ谿九ｊ縺ｾ縺吶・
## 蜿ら・

- 螳溯｣・ [`tools/reference-go-build-with-supply`](../../tools/reference-go-build-with-supply/README.md)
- 蜑肴ｮｵ: [Go驕ｸ謚朞odule髢牙桁縺ｮ莠句燕隕ｳ貂ｬ](TEI_GO_MODULE_CLOSURE_PREFLIGHT_2026-11-07.md)・・B-0a・峨ー繧ｪ繝輔Λ繧､繝ｳGo Module萓帷ｵｦ縺ｮ讀懆ｨｼ](https://github.com/hat47x/tei-platform/pull/3142)・・B-0b・・- 險育判: [`plan/workstreams/TEI_PRODUCT_CRITICAL_PATH_2026-10.md`](../workstreams/TEI_PRODUCT_CRITICAL_PATH_2026-10.md) ﾂｧ35.2
