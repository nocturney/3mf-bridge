# SliceBridge — bilingual marketing kit / ערכת מרקטינג דו־לשונית

This kit documents **pre-release** communications. It is not a download/release announcement.

## Brand and asset rules
- Product: **SliceBridge** (one word); English line: **Bridge your prints.**; Hebrew: **מחברים בין סלייסרים.**
- Palette: Deep Forest `#193F38`, Forest `#2F7F73`, Mint `#5BBFA8`, Light Mint `#A9E5D0`, Mint Foam `#DFF6EE`.
- **Approved artwork is authoritative.** Do not recreate the production icon: [original icon](../docs/assets/approved-icon.png) and [original banner](../docs/assets/approved-wave.png). New graphics are supporting typography/illustrations, not replacement logos.
- For display, round the banner's **four visible corners** with CSS `border-radius: 16px; overflow: hidden` on its image/frame; preserve the original PNG file.
- Source files below are self-contained SVG. Export to PNG for networks that do not allow SVG uploads; keep text legible and do not cover the development disclaimer.

## Shareable visuals

| Asset | Format | English | עברית |
| --- | --- | --- | --- |
| Project approach | 1080×1350 (4:5) | [Approach EN](../docs/assets/marketing/approach-en.svg) | [גישה HE](../docs/assets/marketing/approach-he.svg) |
| Three-step workflow | 1200×675 (16:9) | [Workflow EN](../docs/assets/marketing/workflow-en.svg) | [תהליך HE](../docs/assets/marketing/workflow-he.svg) |
| Community message | 1200×675 (16:9) | [Community EN](../docs/assets/marketing/community-en.svg) | [קהילה HE](../docs/assets/marketing/community-he.svg) |
| Approved icon | original PNG | [Icon](../docs/assets/approved-icon.png) | [אייקון](../docs/assets/approved-icon.png) |
| Approved banner | original PNG | [Banner](../docs/assets/approved-wave.png) | [באנר](../docs/assets/approved-wave.png) |
| Existing OG share preview | 1280×640 PNG | [Open Graph](../docs/assets/social-preview.png) | [תצוגת שיתוף](../docs/assets/social-preview.png) |

**On-site gallery:** [Press & media page](../docs/press.html) after publishing on GitHub Pages. Editable, ready-to-copy posts: [COPY.md](COPY.md).

## Accurate origin and attribution
- Present SliceBridge as an **AI-assisted adaptation/assembly of existing tools and solutions**, not a new technology invented from scratch or an invented origin narrative.
- Preserve separate credit and licensing requirements for upstream components. Marketing copy never substitutes for [third-party notices](../THIRD_PARTY_NOTICES.md).

## Safe messages while release is on hold
- SliceBridge is a local-first **3MF project bridge**, not another slicer.
- The project currently focuses on Bambu Studio 3MF to **supported** target slicers/printers; the feature list and printer catalogue are not claims of blanket conversion support.
- The source 3MF is intended to be preserved. Formal release verification remains pending; always review the result in the destination slicer before printing.
- The project-owned code is declared **GPL-3.0-only** after [PR #12](https://github.com/nocturney/slicebridge/pull/12) merged. Ownership, third-party rights, and binary distribution checks still remain **HOLD**; do not imply a released binary is available.
- **There is no approved public download.** Never use “Download now,” “supports every printer,” “one-click print,” “guaranteed compatibility,” or unverified browser support.
- Do not use vendor logos/product imagery implying partnership or endorsement.

## Recommended posting sequence (not scheduled)
1. Share the **project approach**, with the straightforward explanation of existing solutions + AI assistance in [COPY.md](COPY.md).
2. Share **workflow** with the short explanation of the three steps and current compatibility limits.
3. Share **community card** with an invitation to watch the [GitHub project](https://github.com/nocturney/slicebridge).
4. Only after signed release QA and licensing clearance: update assets/copy with actual downloads and verified supported-printer matrix.

## ALT text / טקסט חלופי
- Approach EN: “SliceBridge: existing 3D-printing solutions adapted with AI assistance into a 3MF workflow tool; still in development.”
- Approach HE: “SliceBridge: שילוב פתרונות קיימים בעזרת בינה מלאכותית להתאמה לצורת עבודה מסוימת; עדיין בפיתוח.”
- Workflow EN: “Three development-stage steps: Bambu Studio 3MF, local conversion with compatible profiles, and review in a supported target slicer.”
- Workflow HE: “שלושה שלבים: קובץ 3MF של Bambu Studio, המרה מקומית בעזרת פרופילים תואמים, ובדיקה בסלייסר יעד נתמך.”
- Community EN: “Different slicers, stronger community; SliceBridge is an in-development bridge, not another slicer.”
- Community HE: “סלייסרים שונים, קהילה חזקה יותר; SliceBridge הוא גשר בין כלים קיימים שנמצא בפיתוח.”

## Quality control
- Check RTL alignment in browser and at the export resolution (especially Latin names and 3MF inside Hebrew).
- Keep rounded corners consistent; prefer 18–30 px (depending on card size).
- Approved icon/banner pixels are unchanged. Decorative cards may evolve, but must not impersonate a final release UI or a confirmed screenshot.
- Re-check `README.md`, `FAQ.md`, compatibility claims, and [distribution gate](../DISTRIBUTION_CHECKLIST.md) before each public announcement.
