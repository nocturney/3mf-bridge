# Printer release-date sources (internal)

Internal log for `releaseDate` values in `native/frontend/dist/printer-catalog.json`.
Not for the public README. Dates are only recorded when a manufacturer press
release, manufacturer-hosted launch post, or dated wire copy of one states them.
Dates are never inferred from preset file timestamps, model numbering, or
retailer listings.

Meaning of `releaseDate`: the date the manufacturer publicly launched or
announced the model in a primary release or launch post. Pre-order openings,
retail or shipping dates are not used, because the source for those is often
a trade article or retailer. Where only a pre-order date is documented, the
value stays `null`.

Recorded 2026-10-21 by a scratch-clone pass. Prusa MINI, MK4S and XL were
added the same day from the Prusa blog posts they link to. Schema: `YYYY-MM-DD`,
day precision, `null` where evidence is absent.

## Dated entries

| Model id | Model | releaseDate | Precision | Evidence | Source |
| --- | --- | --- | --- | --- | --- |
| `prusa-prusa-core-one` | Prusa CORE One | 2024-11-19 | day | Prusa press release, dateline "FRANKFURT, GERMANY – November 19, 2024", announced at Formnext | https://cdn.prusa3d.com/article/meet-the-prusa-core-one-josef-prusa-announces-a-new-3d-printer-as-the-foundation-of-a-new-product-line_239457/ |
| `prusa-prusa-mk4` | Prusa MK4 | 2023-03-29 | day | Prusa Research press release, dateline "PRAGUE, CZECH REPUBLIC, March 29, 2023", names "Original Prusa MK4" | https://cdn.prusa3d.com/article/prusa-research-introduces-a-new-3d-printer-the-mk4-promises-fast-printing-and-high-quality-printing-for-everyone-_234221 |
| `creality-creality-k1` | Creality K1 | 2023-04-09 | day | Creality press release, dateline "Shenzhen, China, April 9, 2023", K1 series unveiled at the 9th anniversary event. The page timestamp reads Apr 9, 2023 00:32 UTC. Trade coverage says April 10 (event timing); the primary dateline is used | https://www.creality.com/blog/creality-unveils-revolutionary-k1-series-and-celebrates-milestone-9th-anniversary |
| `creality-creality-k1-max` | Creality K1 Max | 2023-04-09 | day | Same Creality K1-series release (K1 and K1 Max named as launched) | same as above |
| `flashforge-adventurer-5m` | Adventurer 5M | 2023-09-01 | day | Flashforge PR Newswire release, dateline "HANGZHOU, China, Sept. 1, 2023", names the standard Adventurer 5M as a sister model at $399 | https://www.aap.com.au/aapreleases/cision20230901ae92939/ |
| `flashforge-adventurer-5m-pro` | Adventurer 5M Pro | 2023-09-01 | day | Same Flashforge release; Pro is the main subject, sales from Sept 5, 2023 | same as above |
| `prusa-prusa-mini` | Prusa MINI | 2019-10-12 | day | Prusa blog post by Josef Průša, "I'm excited to announce our brand new Original Prusa MINI". Byline date printed as "October 12. 2019" | https://blog.prusa3d.com/original-prusa-mini-is-here-smart-and-compact-3d-printer_30887/ |
| `prusa-prusa-mk4s` | Prusa MK4S | 2024-08-12 | day | Prusa blog post "The Original Prusa MK4S is Here!", byline date printed as "August 12. 2024". Opens "in the middle of a hot Summer here in Prague" | https://blog.prusa3d.com/the-original-prusa-mk4s-is-here_100605/ |
| `prusa-prusa-xl` | Prusa XL | 2021-11-18 | day | Prusa blog post "First look at the Original Prusa XL", byline date printed as "November 18. 2021". Announces the XL as "our upcoming large-scale CoreXY 3D printer" | https://blog.prusa3d.com/original-prusa-xl-first-look_58312/ |

## Checked but left null

- **Prusa MK4 (Prusa Research profile group, `prusa-research-mk4`)**: the same press release is the only source, and it does not establish that the profile group is the plain MK4. The mapping was assumed, so no date is recorded. The MK4 Input Shaper and MMU3 variants also stay null.
- **ELEGOO Centauri Carbon (`elegoo-centauri-carbon`)**: the only dated source is the Feb 17, 2025 pre-order opening. The first announcement release (Feb 6, 2025) is not a launch date, and the release text was summarised rather than read verbatim. A pre-order date is not a release date, so nothing is recorded.
- **Snapmaker J1 (`snapmaker-j1`)**: the Nov 8, 2022 dateline in the wire copy marks the opening of pre-orders, not a launch date. Snapmaker's own domain was not fetched. Nothing is recorded.
- **QIDI Q1 Pro**: only trade coverage found (Gizchina dated Mar 20, 2024, 3DPrinting.com Mar 21, 2024). No QIDI primary source. Conflicting.
- **Snapmaker U1**: global retail date April 10, 2026 from trade press. The all3dp article returned HTTP 403, and no Snapmaker primary source was found. A launch date was not recorded.
- **Sovol SV08**: no official source found in search.
- **Anycubic Kobra 3**: only All3DP coverage (April 2024 launch, V2 in May 2025). No Anycubic or wire source.
- **Anycubic Kobra S1 Combo**: trade coverage (Jan 2, 2025 pre-sale). The Combo is not a catalogue entry, and the standard Kobra S1 was not confirmed.
- **Elegoo Centauri Carbon 2 Combo**: ELEGOO PR Newswire release, Jan 19, 2026, launch Jan 26, 2026. The mapping between "Combo" and the catalogue entry `Centauri Carbon 2` is unclear, so nothing was recorded.
- **Flashforge AD5X**: Flashforge blog says "November 15" with no year in the snippet. Retailer listings say 14.1.2025. Conflicting, so not recorded.
- **Creality K2 Plus**: sources conflict (April 2024 unveiling, July 31, 2024 release from one trade article). No primary date.
- **Prusa MK3S+ and Prusa MINI+ (`prusa-research-i3-mk3s-mk3s`, `prusa-research-mini-mini`)**: the Prusa blog post "November 2020 Update" (https://blog.prusa3d.com/november2020_40870/, byline date printed as "November 24. 2020") launches both the MK3S+ and MINI+. The catalogue has no separate MK3S+ or MINI+ identity. Those Prusa Research profile groups are "i3 MK3S && MK3S+" and "MINI && MINI+", which also cover the base MK3S and MINI (released earlier). Recording the date there would misdate the base printers, so it stays null. It would be recorded once a single-identity entry exists for each "+" model.
- **Creality Ender-3 V3**: only comparison-table and retailer dates (early 2024). No Creality announcement found.

## Catalog pipeline note

The source of truth is `catalog/data/makers/*.json` (`release_date` and
`release_date_evidence`), which `catalog/tools/export_ui.py` exports into the
dist JSON. The dist is regenerated from the maker files, never edited by hand.
`catalog/tools/build.py` carries cited dates over from the committed maker files
on rebuild (`carry_release_evidence`), because upstream profile repositories do
not supply them. `catalog/tools/check.py` requires `release_date_evidence` for
any non-null date. The evidence field holds the source URL, never free text.
