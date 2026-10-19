![SliceBridge approved banner — Different slicers. A stronger community.](assets/slicebridge-approved-banner.png)

# SliceBridge

**Bridge your prints. · מחברים בין סלייסרים.**

**One project. More printers. Fewer walls.**  
פרויקט אחד. יותר מדפסות. פחות מחסומים.

[English](#english) · [עברית](#hebrew)

---

<a id="english"></a>
## English

### Different slicers. A stronger community.

A great 3D model should not be limited by the slicer in which it was created.

Yet anyone with more than one printer knows the friction: a project arrives as a Bambu Studio 3MF, but the next machine belongs to another ecosystem. The geometry may be portable, while printer profiles, nozzle sizes, filaments and slicing settings are not. Moving between them can mean rebuilding a project, losing useful choices or taking risks with the wrong machine configuration.

**SliceBridge exists to make that handoff simpler, more transparent and more open.**

It is a lightweight desktop bridge—not another slicer. For supported combinations, you bring in a Bambu Studio 3MF project, choose the printers you want to use, and generate a separate 3MF project for each target. The conversion engine works with the target slicer's genuine machine, process and filament profiles when available, while preserving the source project and compatible design/print intent. You then open the results in the slicers you already trust and check them before printing.

No need to choose one printer brand for your entire workshop. No need to change how everyone else shares a model.

### The workflow we are building

1. **Bring your project:** choose or drop a Bambu Studio 3MF file.
2. **Choose your destinations:** select one or more supported target printers and their actual nozzle/material configuration.
3. **Bridge intelligently:** use verified target machine and material presets—not a simple rename of a Bambu profile.
4. **Keep every option:** generate independent, non-overwriting files and a result for each selected printer.
5. **Finish in your slicer:** review, slice and print using the target slicer's normal workflow.

### Why it matters

- **Local-first by design.** Conversion does not require uploading models to a cloud service.
- **Printer-aware rather than one-size-fits-all.** Compatibility, nozzle and filament constraints matter; unsupported combinations should fail clearly rather than produce misleading output.
- **Preserve the original.** A failed target should not erase the source project or successful outputs for other printers.
- **Respect existing tools.** SliceBridge is not a slicer, printer controller, cloud print farm or a promise of universal 3MF compatibility.
- **Built for a mixed-printer world.** One desktop workflow can serve a workshop with multiple manufacturers.

### From today's targets to tomorrow's bridge

The conversion work has focused on **Snapmaker U1**, **Flashforge Creator 5**, **ELEGOO Centauri Carbon 2** and **QIDI Plus5**. These paths have received earlier conversion/slicing checks, but not every 3MF feature, hardware variant or physical print has been independently validated.

The next steps extend the idea without pretending that every printer is already supported:

- **Printer catalogue:** a versioned, validated index with manufacturer/model search and printer selection at first launch or later in Settings. Prefer verified **manufacturer-native profiles**; use **OrcaSlicer**, then **PrusaSlicer**, as compatibility-checked sources or fallbacks when appropriate. A listed printer is not automatically a working conversion target. See [catalogue proposal #7](https://github.com/nocturney/3mf-bridge/issues/7).
- **Optional MakerWorld handoff:** a browser-assisted route from MakerWorld to SliceBridge, with explicit **Off / Ask / Always** choices (Off by default), without replacing Bambu Studio's system file or protocol associations. Chromium/Firefox work is in progress; Safari requires separate work. See [draft PR #6](https://github.com/nocturney/3mf-bridge/pull/6).
- **Native-feeling Windows and macOS app:** the current development direction uses **Go + Wails**, WebView2 on Windows and WebKit on macOS, with English LTR and Hebrew RTL. Electron is an earlier prototype, not the intended final runtime.

### Project status — October 2026

**In development; not a general-availability release.** This default branch currently contains an early Electron UI shell, not the complete converter. The more advanced conversion and native-app work lives in development branches; the newest MakerWorld/browser and banner integration is in [draft PR #6](https://github.com/nocturney/3mf-bridge/pull/6), which still requires end-to-end browser testing, security review and distribution/signing work. The proposed broad printer catalogue is not implemented yet.

Always verify the actual printer, installed nozzle, material and slicing preview before printing. Source code and development documentation are available in [the active development branch](https://github.com/nocturney/3mf-bridge/tree/feature/browser-compat-and-icon).

---

<a id="hebrew"></a>
## עברית

### סלייסרים שונים. קהילה אחת חזקה יותר.

מודל תלת־ממדי מוצלח לא אמור להיות מוגבל לתוכנת הסלייסר שבה הוא נוצר.

אבל מי שעובד עם מדפסות ממספר יצרנים מכיר את הבעיה: פרויקט מגיע כקובץ 3MF של Bambu Studio, ואילו המדפסת שאליה רוצים להעביר אותו פועלת בסביבה אחרת. הגאומטריה יכולה לעבור, אבל פרופילי המדפסת, קוטר הדיזה, החומרים והגדרות ההדפסה אינם בהכרח תואמים. המעבר בין המערכות דורש לפעמים בנייה מחדש של הפרויקט, ויתור על הגדרות חשובות או הסתכנות בפרופיל מכונה לא מתאים.

**כאן נכנס SliceBridge: גשר קטן בין עולמות הדפסה שונים.**

זו תוכנת מחשב קלה, לא סלייסר נוסף. עבור שילובים שנתמכים ואומתו, מכניסים פרויקט 3MF של Bambu Studio, בוחרים מדפסת יעד אחת או יותר ומקבלים קובץ 3MF נפרד לכל מדפסת. מנוע ההמרה משתמש, כשניתן, בפרופילי המכונה, התהליך והפילמנט האמיתיים של סלייסר היעד, תוך שמירה על קובץ המקור ועל מאפייני הפרויקט שניתן להעביר בבטחה. לבסוף פותחים את התוצאה בסלייסר המוכר, בודקים ומבצעים Slice.

המטרה אינה להחליף את הסלייסרים, אלא **לאפשר להם לעבוד יחד** — גם בסטודיו שבו לכל מדפסת יצרן ותוכנה משלה.

### איך זה אמור לעבוד

1. **מכניסים פרויקט:** בוחרים קובץ 3MF של Bambu Studio או גוררים אותו לחלון.
2. **בוחרים יעדים:** מסמנים מדפסת אחת או כמה מדפסות, עם הדיזה והחומר שבאמת מותקנים.
3. **ממירים עם מודעות לחומרה:** משתמשים בפרופילי יעד תקינים, ולא רק משנים את שם המדפסת בקובץ.
4. **שומרים על גמישות:** יוצרים פלט נפרד לכל יעד, בלי לדרוס את המקור או את התוצאות הקודמות.
5. **ממשיכים בכלי המוכר:** פותחים בסלייסר היעד, בודקים את ההגדרות, מבצעים Slice ורק אז מדפיסים.

### העקרונות שלנו

- **מקומי קודם כול.** המרת הקבצים אינה מחייבת העלאה לענן.
- **אמינות לפני הבטחות.** התאמה בין מכונה, דיזה וחומר חשובה יותר מתווית כללית של ״תמיכה בכל המדפסות״.
- **המקור נשאר המקור.** תקלה בהמרה למדפסת אחת לא אמורה למחוק קובץ קיים או לפגוע בתוצאות מוצלחות של יעדים אחרים.
- **עובדים עם הכלים הקיימים.** SliceBridge אינו סלייסר, מערכת לשליטה במדפסות או שירות הדפסה בענן.
- **חופש בחירה.** פרויקט אחד יכול להגיע לכמה מערכות הדפסה, כאשר קיימת עבורן תמיכה אמיתית.

### לאן ממשיכים

מנוע ההמרה התמקד עד כה בארבע מדפסות: **Snapmaker U1**, **Flashforge Creator 5**, **ELEGOO Centauri Carbon 2** ו־**QIDI Plus5**. התקבלו תוצאות בדיקה חיוביות במסלולי המרה וחיתוך קודמים, אבל לא נבדקו באופן בלתי תלוי כל אפשרויות ה־3MF, כל שילובי החומרה או הדפסות פיזיות.

חזון ההמשך כולל:

- **קטלוג מדפסות מתעדכן:** חיפוש לפי יצרן ודגם, בחירה בהפעלה הראשונה והוספת מדפסות בהגדרות. ההעדפה היא לפרופיל **מקורי ומאומת של היצרן**, ורק אחריו **OrcaSlicer** ו־**PrusaSlicer** בהתאם לתאימות בפועל. מדפסת שנמצאת בקטלוג אינה בהכרח נתמכת להמרה. [אפיון הקטלוג — #7](https://github.com/nocturney/3mf-bridge/issues/7).
- **מעבר אופציונלי מ־MakerWorld:** סיוע דרך תוסף דפדפן עם בחירה מפורשת בין **כבוי / לשאול בכל פעם / תמיד להשתמש ב־SliceBridge**. ברירת המחדל כבויה; לא מחליפים את שיוכי Bambu Studio במערכת ההפעלה. התמיכה בדפדפני Chromium ו־Firefox נמצאת בפיתוח; Safari מצריך עבודה נפרדת. [PR #6 — טיוטת פיתוח](https://github.com/nocturney/3mf-bridge/pull/6).
- **יישום טבעי ל־Windows ול־macOS:** כיוון הפיתוח הנוכחי מבוסס **Go + Wails**, עם WebView2 ו־WebKit, וממשק עברי RTL ואנגלי LTR. מעטפת Electron היא אב־טיפוס מוקדם, לא סביבת הריצה הסופית המתוכננת.

### מצב הפרויקט — אוקטובר 2026

**הפרויקט בפיתוח ועדיין אינו מוכן להפצה כללית.** ענף ברירת המחדל מכיל בשלב זה ממשק Electron ראשוני בלבד, לא את מנוע ההמרה המלא. הקוד המתקדם ומעטפת Wails נמצאים בענפי הפיתוח. שילוב התוסף, הבאנר וזרימת MakerWorld נמצא ב־[PR #6](https://github.com/nocturney/3mf-bridge/pull/6), שעדיין במצב טיוטה וזקוק לבדיקות מקצה לקצה, בדיקות אבטחה והשלמת הכנות להפצה. קטלוג המדפסות הרחב עדיין בתכנון.

לפני כל הדפסה יש לוודא את המדפסת, קוטר הדיזה הפיזי, החומר ותצוגת ה־Slice. קוד המקור והתיעוד הטכני נמצאים ב־[ענף הפיתוח הפעיל](https://github.com/nocturney/3mf-bridge/tree/feature/browser-compat-and-icon).

---

<sub>SliceBridge visual identity: approved Mint / Forest artwork, preserved from the original project design. The image above is the original approved banner crop, not a newly generated substitute.</sub>
