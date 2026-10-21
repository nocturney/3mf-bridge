![SliceBridge approved banner — Different slicers. One simple workflow.](assets/slicebridge-readme-banner-rounded.svg)

# SliceBridge

**Different slicers. One simple workflow.**

[English](#english) · [עברית](#hebrew) · [Downloads](https://github.com/nocturney/slicebridge/releases)

<a id="english"></a>
## English

**SliceBridge** helps move editable 3MF projects between printers and slicers without rebuilding the model from scratch. Start with a compatible Bambu Studio / Orca-style project (including projects downloaded from MakerWorld), choose an available destination, and continue working in your own slicer. The conversion happens locally and leaves the source file untouched.

This is a small open-source project put together from existing tools, with AI assistance, to make everyday multi-printer work easier.

### Printer and slicer support

**SliceBridge is not limited to a fixed list of four printer models.** It uses different conversion paths depending on the destination and the profiles installed on your computer:

- **Installed Orca-family printer profiles:** SliceBridge discovers machine presets in supported local Orca-family slicer installations and offers additional printer targets when it can resolve a compatible **machine + process + filament** combination. These targets are derived from the profiles found on your computer, not from a hardcoded four-printer list. The actual available models and materials vary by installation.
- **PrusaSlicer projects:** The bundled [Panda2Prusa](https://github.com/CupsOhJoe/Panda2Prusa) engine converts compatible projects to an editable **generic PrusaSlicer 3MF**. Choose the precise printer, print and filament presets in PrusaSlicer before slicing; they are not assigned automatically by SliceBridge.
- **Dedicated printer integrations:** Existing, independently implemented paths remain available for **Snapmaker U1, Flashforge Creator 5, ELEGOO Centauri Carbon 2 and QIDI Plus5 / X-Plus 5**. These are the original integrations, **not the full extent of the application's target support**.

Supported local Orca-family profile sources currently include **OrcaSlicer, QIDIStudio, ElegooSlicer, Flash Studio and Snapmaker Orca**. Having a particular slicer installed is not enough by itself: a usable printer target must pass the profile-resolution and conversion checks.

**Discover more printers:** *My Printers* includes an offline reference catalogue of **603 models from 84 manufacturers**. SliceBridge also detects **22 known desktop slicer applications**, including software from Creality, Anycubic, Prusa, QIDI, ELEGOO, Flashforge, Snapmaker and Sovol. This helps identify your installed tools and find printer models. Catalogue entries and detected slicer applications are not, by themselves, conversion adapters. The conversion target list reflects the adapters and compatible local profiles actually available.

### What it does

- Converts **editable 3MF projects**, rather than turning a model into pre-sliced, printer-specific G-code.
- Keeps source geometry and project structure, including objects/plates where applicable; validates converted output rather than merely changing the file extension.
- Preserves source color/material slots and painting or object assignments through supported conversion paths, without silently combining separate source filaments. If a project has more virtual colors than the printer has physical tools, **you** decide the mapping in the destination slicer.
- Uses real compatible machine, nozzle, process and filament settings for printer-specific paths; refuses unsupported or inconsistent combinations instead of silently choosing arbitrary settings.
- Adds multiple files or folders (up to **200 source 3MF projects**), selects multiple destinations, and provides individual success/error results with conversion details.
- Keeps generated projects separate from source files, avoids overwriting existing outputs, and removes stale sliced machine-code caches from converted projects.
- Runs conversion locally on **Windows and macOS**, without a paid API or a cloud conversion service.

### Getting started

1. Download a development build from [GitHub Releases](https://github.com/nocturney/slicebridge/releases): Windows ZIP or macOS Apple Silicon ZIP. Extract the **entire archive**; the Prusa conversion engine is bundled with the app.
2. Add one or more compatible `.3mf` files, drop files into the app, or select a folder.
3. Explore your printers in *My Printers*, then select the **available conversion targets** in the main window. Additional Orca-family targets appear when compatible installed profiles resolve successfully. Selecting a catalogue-only entry does not create a conversion target.
4. Convert, open the resulting project in its destination slicer, and review printer, nozzle, filament and physical tool assignments before slicing or printing.

The interface is available in **English and Hebrew (RTL)**. No printer is controlled and no print starts automatically.

### MakerWorld connection

Ordinary MakerWorld 3MF downloads can be imported as files without a browser extension. The optional MakerWorld browser integration currently uses an experimental, **manually installed unpacked extension** and is **not listed in an official browser store**.

For that reason, the MakerWorld setup button is **greyed out and non-clickable by default**. Its tooltip points to **About → Developer Mode**, where experienced users can explicitly enable the manual setup wizard. Turning Developer Mode off locks it again. The integration does not silently install an extension or replace Bambu Studio as the system default. See the [advanced installation guide](browser-extension/INSTALLATION.md).

### Compatibility and verification

**Our support rule:** A conversion path is supported when it is implemented and passes relevant software conversion tests. A physical test print is useful additional validation, **not a prerequisite**.

Compatibility still depends on the source project, destination format and actual local presets. The printer catalogue is a discovery resource, not a promise that every listed printer has a tested adapter; the 22 slicer detections do not mean 22 interchangeable conversion engines. **Bambu Lab remains a source, not a destination.** Generic PrusaSlicer output requires final printer/material preset selection in PrusaSlicer. Always review generated projects in the destination slicer before printing.

<a id="hebrew"></a>
<div dir="rtl" align="right" lang="he">
<h2>עברית</h2>

<p><strong><span dir="ltr">SliceBridge</span></strong> מאפשר להעביר פרויקטי <span dir="ltr">3MF</span> ניתנים לעריכה בין מדפסות וסלייסרים, בלי לבנות את הפרויקט מחדש. מתחילים בפרויקט תואם של <span dir="ltr">Bambu Studio</span> או ממשפחת <span dir="ltr">Orca</span> (כולל פרויקטים שהורדו מ־<span dir="ltr">MakerWorld</span>), בוחרים יעד זמין וממשיכים לעבוד בסלייסר הרגיל. ההמרה מתבצעת מקומית, וקובץ המקור נשאר ללא שינוי.</p>

<p>זהו פרויקט קטן בקוד פתוח, שהרכבתי בעזרת פתרונות קיימים וכלי בינה מלאכותית כדי להקל על עבודה יומיומית עם מדפסות שונות.</p>

<h3>תמיכה במדפסות ובסלייסרים</h3>

<p><strong>SliceBridge אינו מוגבל לרשימה סגורה של ארבע מדפסות.</strong> קיימים כמה מסלולי המרה, בהתאם לסלייסר ולפרופילים שמותקנים במחשב:</p>
<ul>
<li><strong>פרופילים מותקנים ממשפחת <span dir="ltr">Orca</span>:</strong> התוכנה מאתרת פרופילי מכונה בסלייסרים מקומיים נתמכים ומציעה מדפסות יעד נוספות כאשר היא מצליחה לפתור סט תואם של <strong>מכונה + תהליך הדפסה + פילמנט</strong>. רשימת הדגמים הזמינים נגזרת מהפרופילים שבמחשב, ולא מרשימה קשיחה של ארבעה דגמים. הזמינות תלויה בהתקנה המקומית.</li>
<li><strong>פרויקטים של <span dir="ltr">PrusaSlicer</span>:</strong> מנוע <a href="https://github.com/CupsOhJoe/Panda2Prusa" dir="ltr">Panda2Prusa</a> המצורף יוצר פרויקט <span dir="ltr">3MF</span> כללי וניתן לעריכה עבור <span dir="ltr">PrusaSlicer</span>. לאחר הפתיחה יש לבחור בסלייסר את פרופילי המדפסת, ההדפסה והחומר המתאימים; <span dir="ltr">SliceBridge</span> אינו משייך אותם אוטומטית.</li>
<li><strong>התאמות ייעודיות:</strong> מסלולי ההמרה הקיימים ל־<span dir="ltr">Snapmaker U1</span>, <span dir="ltr">Flashforge Creator 5</span>, <span dir="ltr">ELEGOO Centauri Carbon 2</span> ול־<span dir="ltr">QIDI Plus5 / X-Plus 5</span> ממשיכים לפעול. אלה ההתאמות המקוריות, <strong>ולא מלוא התמיכה של התוכנה</strong>.</li>
</ul>

<p>מקורות פרופילים מקומיים נתמכים ממשפחת <span dir="ltr">Orca</span> כוללים כיום את <span dir="ltr">OrcaSlicer</span>, <span dir="ltr">QIDIStudio</span>, <span dir="ltr">ElegooSlicer</span>, <span dir="ltr">Flash Studio</span> ו־<span dir="ltr">Snapmaker Orca</span>. עצם התקנת הסלייסר אינה מספיקה: כדי להציע מדפסת כיעד המרה, הפרופילים חייבים לעבור בדיקות התאמה.</p>

<p><strong>גילוי מדפסות נוספות:</strong> במסך ״המדפסות שלי״ יש קטלוג מקומי של <strong>603 דגמים מ־84 יצרנים</strong>. התוכנה יודעת גם לזהות התקנה של <strong>22 יישומי סלייסר</strong>, ובהם תוכנות של <span dir="ltr">Creality</span>, <span dir="ltr">Anycubic</span>, <span dir="ltr">Prusa</span>, <span dir="ltr">QIDI</span>, <span dir="ltr">ELEGOO</span>, <span dir="ltr">Flashforge</span>, <span dir="ltr">Snapmaker</span> ו־<span dir="ltr">Sovol</span>. הקטלוג וזיהוי התוכנות מסייעים למצוא דגמים ותוכנות מותקנות, אך אינם יוצרים לבדם מנוע המרה. יעדי ההמרה הזמינים נקבעים לפי מימוש ההמרה והפרופילים התואמים בפועל.</p>

<h3>מה אפשר לעשות?</h3>
<ul>
<li>להמיר פרויקטי <span dir="ltr">3MF</span> שנשארים <strong>ניתנים לעריכה</strong>, במקום להפיק ישירות <span dir="ltr">G-code</span> שכבר נפרס למדפסת מסוימת.</li>
<li>לשמר מבנה וגאומטריה של הפרויקט, כולל אובייקטים ומשטחי הדפסה כשהפורמט מאפשר זאת, ולבדוק את תוצר ההמרה.</li>
<li>לשמר ערוצי צבע וחומר נפרדים ושיוכי צביעה או אובייקטים במסלולים הנתמכים, בלי לאחד פילמנטים באופן אוטומטי. כשיש יותר צבעים וירטואליים מכלים פיזיים, <strong>המשתמש מחליט</strong> כיצד למפות אותם בסלייסר היעד.</li>
<li>להשתמש בפרופילי מכונה, דיזה, תהליך וחומר תואמים; שילוב לא תקין או חסר נחסם במקום להמציא הגדרות.</li>
<li>להוסיף כמה קבצים או תיקיות (עד <strong>200 פרויקטי <span dir="ltr">3MF</span></strong>), לבחור כמה יעדי המרה ולקבל דוח נפרד על הצלחות, שגיאות ופרטי ההמרה.</li>
<li>להשאיר את קובצי המקור ללא שינוי, לא לדרוס קובצי פלט קיימים ולהסיר נתוני פריסה ישנים שאינם מתאימים למדפסת החדשה.</li>
<li>לבצע את ההמרה מקומית ב־<span dir="ltr">Windows</span> וב־<span dir="ltr">macOS</span>, ללא שירות המרה בענן או <span dir="ltr">API</span> בתשלום.</li>
</ul>

<h3>איך מתחילים?</h3>
<ol>
<li>מורידים גרסת פיתוח מ־<a href="https://github.com/nocturney/slicebridge/releases">GitHub Releases</a> עבור <span dir="ltr">Windows</span> או <span dir="ltr">macOS (Apple Silicon)</span>. מחלצים את <strong>כל</strong> קובצי הארכיון; מנוע ההמרה ל־<span dir="ltr">PrusaSlicer</span> כבר מצורף לתוכנה.</li>
<li>מוסיפים קובצי <span dir="ltr">.3mf</span> תואמים, גוררים קבצים לחלון או בוחרים תיקייה שלמה.</li>
<li>אפשר לעיין במדפסות דרך ״המדפסות שלי״, ואז לבחור <strong>יעדי המרה זמינים</strong> במסך הראשי. יעדים נוספים ממשפחת <span dir="ltr">Orca</span> מופיעים כאשר נמצאו פרופילים מקומיים תואמים. בחירת דגם שמופיע בקטלוג בלבד אינה מפעילה עבורו המרה.</li>
<li>ממירים, פותחים את התוצר בסלייסר היעד ובודקים את המדפסת, הדיזה, החומר ושיוכי הכלים הפיזיים לפני פריסה או הדפסה.</li>
</ol>

<p>הממשק זמין <strong>בעברית (מימין לשמאל) ובאנגלית</strong>. התוכנה אינה מפעילה מדפסת ואינה מתחילה הדפסה אוטומטית.</p>

<h3>חיבור ל־<span dir="ltr">MakerWorld</span></h3>

<p>קובצי <span dir="ltr">3MF</span> שהורדו מ־<span dir="ltr">MakerWorld</span> ניתנים לפתיחה רגילה גם בלי תוסף לדפדפן. החיבור האופציונלי לדפדפן מבוסס כרגע על <strong>תוסף ניסיוני שמותקן ידנית מתיקייה</strong>, והוא <strong>עדיין לא פורסם בחנות תוספים רשמית</strong>.</p>

<p>לכן כפתור ההגדרה של <span dir="ltr">MakerWorld</span> <strong>אפור ולא לחיץ כברירת מחדל</strong>. הטולטיפ מפנה אל <strong>אודות ← מצב מפתחים (<span dir="ltr">Developer Mode</span>)</strong>, שם משתמשים מנוסים יכולים לאפשר במפורש את אשף ההתקנה הידנית. כיבוי מצב המפתחים נועל אותו מחדש. התוסף אינו מותקן בשקט, וההגדרה אינה משנה את ברירת המחדל של <span dir="ltr">Bambu Studio</span>. ראו <a href="browser-extension/INSTALLATION.md">מדריך התקנה למשתמשים מתקדמים</a>.</p>

<h3>תאימות ואימות</h3>

<p><strong>מבחינתנו, המרה נחשבת נתמכת כשהיא מומשה ועברה בדיקות תוכנה רלוונטיות.</strong> אין צורך בהדפסה פיזית כדי להגדיר אותה כנתמכת; הדפסת ניסיון היא אימות נוסף.</p>

<p>התאימות עדיין תלויה בפרויקט המקור, בפורמט היעד ובפרופילים המקומיים. 603 הדגמים הם קטלוג לגילוי, ולא הבטחה שלכל אחד מהם יש כבר מנוע המרה; גם זיהוי 22 סלייסרים אינו אומר שקיימים 22 מנועי המרה הדדיים. <strong>מדפסות <span dir="ltr">Bambu Lab</span> הן כיום מקור, לא יעד.</strong> ב־<span dir="ltr">PrusaSlicer</span> יש לבחור ידנית את פרופילי המדפסת והחומר. לפני הדפסה יש לבדוק את הפרויקט בסלייסר היעד.</p>
</div>

---

**Development downloads / הורדות גרסאות פיתוח:** [GitHub Releases](https://github.com/nocturney/slicebridge/releases) · Experimental releases, not stable installers; Windows/macOS may warn about unsigned builds.

**Source & licensing / קוד ורישיונות:** [Source code](https://github.com/nocturney/slicebridge/tree/main) · [GPL-3.0-only](LICENSE) for covered project code · [License scope](LICENSING.md) · [Third-party notices](THIRD_PARTY_NOTICES.md). Vendor machine/filament presets are read from installed slicers and are not distributed as a universal profile pack.

**Project documentation / תיעוד:** [Changelog](CHANGELOG.md) · [Source version](VERSION) · [Printer catalogue & validation](catalog/README.md) · [MakerWorld advanced setup](browser-extension/INSTALLATION.md).
