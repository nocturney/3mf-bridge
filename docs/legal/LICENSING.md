# SliceBridge open-source licensing

**Project-owned source code: GNU General Public License, version 3 only (SPDX: `GPL-3.0-only`).** The complete license text is in [LICENSE](../../LICENSE). The public project's former Electron prototype and its npm root manifest/lockfile use the same identifier. The ongoing Go/Wails development branch previously used GPL-3.0-only as well.

The GPL gives recipients the rights to use, inspect, modify, and redistribute covered code under its conditions. It is **not** a promise of warranty, printer compatibility, or ready-to-install binaries.

## What is — and is not — licensed here

- **Original SliceBridge code and documentation:** offered under GPL-3.0-only **to the extent the respective contributors own or are entitled to license their contributions**. Copyright remains with the respective owners/contributors. The license text does not change ownership.
- **Upstream third-party code:** stays under its own existing license and copyright, including the pinned `BryanSmee/bl_to_std` GPLv3 component used on the development branch. See [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md). Neither the SliceBridge name nor this `LICENSE` grants rights to another author's work.
- **Printer/process/filament presets and other vendor data:** not redistributed by this repository under the intended model. A user-installed profile stays subject to its original terms, even when SliceBridge reads it locally. A public Git source is **not** a blanket redistribution grant.
- **Product identity and graphics:** the approved SliceBridge name, icon and banner identify the project. Copyright permission for project-owned artwork, to the extent controlled by contributors, does **not** by itself grant trademark rights or permission to falsely claim official affiliation/endorsement. [Trademark guidance](TRADEMARKS.md) applies.
- **Other dependencies:** their own accompanying license terms and notices remain applicable. For distributed builds, preserve the relevant notices and GPL corresponding source; see [simple release notes](../release/DISTRIBUTION_CHECKLIST.md).

## Prior snapshots

The early Electron prototype on the public default branch previously carried an `ISC` declaration in `package.json`, despite lacking a root license file. The switch to GPL-3.0-only describes the licensing intent for the updated project; **do not assume that previously valid permissions granted for older snapshots can be revoked retroactively**, and do not assert that unknown third-party contributions have been relicensed.

## Brand art clarification

The approved brand images are included to present and build the official SliceBridge project. These images were approved for the official project presentation. Modified or third-party builds should avoid implying official endorsement; third-party artwork and marks retain their own rights.

## Contributor policy

Code contributions accepted into SliceBridge are intended to be distributed under **GPL-3.0-only**. By submitting, contributors should have the right to grant those rights, preserve existing notices, and identify imported third-party files and separate licenses. No CLA or assignment of ownership is implied by this document.

## Sharing binaries

GPLv3 allows community distribution, including experimental builds. When distributing GPL-covered binaries, provide the matching corresponding source and required upstream copyright/license notices; don't bundle proprietary vendor presets without permission. Explain tested platforms and limitations.

**Signing and Apple notarization are optional**, not GPL requirements. Unsigned Windows builds may display publisher warnings; unnotarized macOS builds may be blocked by Gatekeeper and need explicit user action. See the [simple community release guide](../release/DISTRIBUTION_CHECKLIST.md).

The existing `v0.4.0-beta.3` draft is an old development preview, not a release based on the current app code.

### עברית
<div dir="rtl" align="right" lang="he">
<p><strong>קוד המקור המקורי של SliceBridge מופץ ברישיון GPL-3.0-only.</strong> הרישיון מעניק זכויות שימוש, עיון, שינוי והפצה בכפוף לתנאיו. הזכויות בקוד ובקבצים של צדדים שלישיים, בפרופילים של היצרנים ובסימני המסחר נבחנות בנפרד. פרסום הקוד אינו אישור אוטומטי להפצה של תוכנה מוכנה או של נכסים שלא נבדקו.</p>
<p>הקובץ <a href="../../LICENSE">LICENSE</a> מכיל את נוסח הרישיון המלא; למסמכי ייחוס נוספים ראו <a href="THIRD_PARTY_NOTICES.md">צדדים שלישיים</a> ואת <a href="../release/DISTRIBUTION_CHECKLIST.md">דרישות ההפצה</a>.</p>
</div>
