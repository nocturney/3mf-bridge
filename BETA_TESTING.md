# SliceBridge — closed beta testing guide

**Status: planning only. No public download or production-ready release is available.** Interested testers can follow the repository; participating in a future limited beta requires explicit access to a vetted build after its release gate is passed.

## English

### What to report

Test a **supported printer/slicer combination only**, with the correct nozzle and material physically installed. A successful conversion alone does not establish a safe print.

1. Record the OS, tested app build/commit, source slicer, target slicer, printer, nozzle and material.
2. Start with a small synthetic/non-sensitive 3MF test project. Keep a backup of the original.
3. Record whether conversion completed, the destination slicer opened it, preview/slicing succeeded and original inputs stayed unchanged.
4. **Do not initiate a physical print** until normal slicer safety checks have been performed and the printer's actual nozzle, temperature limits, material and build volume are validated. A print is optional, not assumed.
5. Report the result through [Beta compatibility report](https://github.com/nocturney/slicebridge/issues/new?template=beta_test.yml). Attach redacted logs/screenshots only if they contain no personal, customer or proprietary information.
6. For reproducible failures open a [bug report](https://github.com/nocturney/slicebridge/issues/new?template=bug_report.yml); for vulnerabilities follow [private security reporting](SECURITY.md).

### Testing status definitions

- **Converted**: an output was generated, but not necessarily usable.
- **Opened**: the target slicer loaded it.
- **Sliced**: the target slicer generated a valid-looking toolpath preview.
- **Physically printed**: actual hardware was run under controlled and documented conditions.

Never equate a model's appearance in a verified **profile registry** with conversion or physical print validation.

### Privacy, licensing and reproducibility

Do not post sensitive project files, personally identifiable information, tokens, proprietary slicer payloads or customer data publicly. Share synthetic fixtures instead. Record input provenance and permissions. Tester results are community reports and may not represent independent certification.

## עברית

<div dir="rtl" align="right" lang="he">
<p><strong>זהו מסמך הכנה בלבד, ולא הזמנה להורדה ציבורית.</strong> בדיקת בטא תתבצע רק לאחר אישור גרסה ותוך שימוש במדפסות, דיזות וחומרים ידועים ומתאימים.</p>
<ol dir="rtl">
<li>מתעדים מערכת הפעלה, גרסת תוכנה, סלייסר מקור ויעד, מדפסת, דיזה וחומר.</li>
<li>מתחילים בקובץ <span dir="ltr">3MF</span> סינתטי ללא מידע של לקוחות, תוך גיבוי המקור.</li>
<li>בודקים בנפרד: האם נוצר קובץ, האם נפתח בסלייסר היעד והאם בוצע <span dir="ltr">Slice</span> תקין.</li>
<li>לא מדפיסים בפועל ללא אימות כל הגדרות הבטיחות והחומרה. הדפסה פיזית אינה חובה לצורך דיווח.</li>
<li>מדווחים דרך <a href="https://github.com/nocturney/slicebridge/issues/new?template=beta_test.yml">טופס בדיקת הבטא</a>, בלי לפרסם קבצים פרטיים, סיסמאות או פרופילי יצרן.</li>
</ol>
<p>קיום מדפסת בקטלוג אינו מוכיח שהמרה או הדפסה פיזית נבדקו.</p>
</div>
