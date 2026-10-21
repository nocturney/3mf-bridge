# Printer release-date sources (internal)

Internal log for `releaseDate` values in `native/frontend/dist/printer-catalog.json`.
Not for the public README. Dates are only recorded when a manufacturer press
release, manufacturer-hosted launch post, or dated wire copy of one states them.
Dates are never inferred from preset file timestamps, model numbering, or
retailer listings.

Recorded 2026-10-21 by a scratch-clone pass. Schema: `YYYY-MM-DD`, day
precision, `null` where evidence is absent.

## Dated entries

| Model id | Model | releaseDate | Precision | Evidence | Source |
| --- | --- | --- | --- | --- | --- |
| `prusa-prusa-core-one` | Prusa CORE One | 2024-11-19 | day | Prusa press release, dateline "FRANKFURT, GERMANY – November 19, 2024", announced at Formnext | https://cdn.prusa3d.com/article/meet-the-prusa-core-one-josef-prusa-announces-a-new-3d-printer-as-the-foundation-of-a-new-product-line_239457/ |
| `prusa-prusa-mk4` | Prusa MK4 | 2023-03-29 | day | Prusa Research press release, dateline "PRAGUE, CZECH REPUBLIC, March 29, 2023", names "Original Prusa MK4" | https://cdn.prusa3d.com/article/prusa-research-introduces-a-new-3d-printer-the-mk4-promises-fast-printing-and-high-quality-printing-for-everyone-_234221 |
| `prusa-research-mk4` | MK4 (Prusa Research profile group) | 2023-03-29 | day | Same Prusa press release as above. Assumed to be the same plain MK4 machine; the MK4 Input Shaper and MMU3 variants are separate entries and stay `null` | same as above |
| `creality-creality-k1` | Creality K1 | 2023-04-09 | day | Creality press release, dateline "Shenzhen, China, April 9, 2023", K1 series unveiled at the 9th anniversary event. The page timestamp reads Apr 9, 2023 00:32 UTC. Trade coverage says April 10 (event timing); the primary dateline is used | https://www.creality.com/blog/creality-unveils-revolutionary-k1-series-and-celebrates-milestone-9th-anniversary |
| `creality-creality-k1-max` | Creality K1 Max | 2023-04-09 | day | Same Creality K1-series release (K1 and K1 Max named as launched) | same as above |
| `flashforge-adventurer-5m` | Adventurer 5M | 2023-09-01 | day | Flashforge PR Newswire release, dateline "HANGZHOU, China, Sept. 1, 2023", names the standard Adventurer 5M as a sister model at $399 | https://www.aap.com.au/aapreleases/cision20230901ae92939/ |
| `flashforge-adventurer-5m-pro` | Adventurer 5M Pro | 2023-09-01 | day | Same Flashforge release; Pro is the main subject, sales from Sept 5, 2023 | same as above |
| `elegoo-centauri-carbon` | Centauri Carbon | 2025-02-17 | day | ELEGOO press release tagged on elegoo.com "Pre-order Centauri Carbon Now", dated Feb 17, 2025 (pre-order opening). The first announcement PR, "Elegoo to Open Pre-Orders for Centauri Carbon on February 17", was dated Feb 6, 2025 and is not used. Only the blog-post date was confirmed. The release text was summarised, not read verbatim | https://www.elegoo.com/blogs/news/https-www-prnewswire-com-news-releases-pre-order-centauri-carbon-now-elegoos-first-core-xy-3d-printer-for-reliability-in-creativity-302377993-html |
| `snapmaker-j1` | J1 | 2022-11-08 | day | Snapmaker PR Newswire release (wire copy), dateline "SHENZHEN, China, Nov. 8, 2022", pre-orders open in the official store from Nov 8. This is the pre-order date, not a later retail date. Snapmaker's own domain was not fetched | https://kyodonewsprwire.jp/release/202211099464 |

## Checked but left null

- **QIDI Q1 Pro**: only trade coverage found (Gizchina dated Mar 20, 2024, 3DPrinting.com Mar 21, 2024). No QIDI primary source. Conflicting.
- **Snapmaker U1**: global retail date April 10, 2026 from trade press. The all3dp article returned HTTP 403, and no Snapmaker primary source was found. A launch date was not recorded.
- **Sovol SV08**: no official source found in search.
- **Anycubic Kobra 3**: only All3DP coverage (April 2024 launch, V2 in May 2025). No Anycubic or wire source.
- **Anycubic Kobra S1 Combo**: trade coverage (Jan 2, 2025 pre-sale). The Combo is not a catalogue entry, and the standard Kobra S1 was not confirmed.
- **Elegoo Centauri Carbon 2 Combo**: ELEGOO PR Newswire release, Jan 19, 2026, launch Jan 26, 2026. The mapping between "Combo" and the catalogue entry `Centauri Carbon 2` is unclear, so nothing was recorded.
- **Flashforge AD5X**: Flashforge blog says "November 15" with no year in the snippet. Retailer listings say 14.1.2025. Conflicting, so not recorded.
- **Creality K2 Plus**: sources conflict (April 2024 unveiling, July 31, 2024 release from one trade article). No primary date.
- **Prusa XL**: trade coverage says November 2021 announcement. Prusa's own post was not fetched.
- **Prusa MINI**: Prusa blog date reported as 12 Oct 2019 (per Wikipedia/All3DP). Trade coverage dated 13 Oct. Blog not fetched, so not recorded.
- **Prusa MK4S**: Aug 12 vs Aug 13, 2024 across trade sources. Prusa primary not fetched.
- **Creality Ender-3 V3**: only comparison-table and retailer dates (early 2024). No Creality announcement found.

## Catalog pipeline note

The source of truth is `catalog/data/makers/*.json` (`release_date` and
`release_date_evidence`), which `catalog/tools/export_ui.py` exports into the
dist JSON. The dist was edited directly as requested. The next pipeline rebuild
will overwrite these values unless the same dates go into the maker source files.
`catalog/tools/check.py` requires `release_date_evidence` for any non-null date.
