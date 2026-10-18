# 3MF Bridge — Multi-printer beta

ממיר מקומי לפרויקטי Bambu Studio / MakerWorld מסוג `.3mf`: בחירת קובץ, סימון מדפסות יעד, ויצירת פלטים ליד קובץ המקור.

**כל ארבעת היעדים פועלים במקביל:** Snapmaker U1, Flashforge Creator 5, ELEGOO Centauri Carbon 2 (CC2) ו-QIDI Plus5 (Orca: X-Plus 5). אפשר לסמן כמה יעדים ולבצע המרה אחת; המנוע יוצר `U1-`, `C5-`, `CC2-`, `QIDI-` עם שמות ללא דריסה. המרה שנכשלת ביעד אחד לא מוחקת פלטים של יעדים אחרים.

## מצב תאימות (18.10.2026)

| מדפסת | סלייסר יעד | פרופיל 0.4 מ״מ | בדיקות |
| --- | --- | --- | --- |
| Snapmaker U1 | Snapmaker Orca | משולב במנוע | עבר פתיחה ו-Slicing לפי דיווח משתמש, כולל שמירת הגדרות וחומרים |
| Flashforge Creator 5 | Flash Studio Desktop | נקרא מפרופילי היצרן המקומיים | המרה וולידציה של 3MF סינתטי חד־חומרי ורב־חומרי עברו; פתיחה בסלייסר אינה מאומתת |
| ELEGOO CC2 | ElegooSlicer | נקרא מפרופילי היצרן המקומיים | המרה וולידציה של 3MF סינתטי חד־חומרי ורב־חומרי עברו; פתיחה בסלייסר אינה מאומתת |
| QIDI Plus5 | OrcaSlicer, X 5 Series | נקרא מפרופילי היצרן המקומיים | המרה וולידציה של 3MF סינתטי חד־חומרי ורב־חומרי עברו; פתיחה בסלייסר אינה מאומתת |

**חשוב:** כל יעד שאינו U1 הוא *beta לא מאומת בממשק הסלייסר*. לפני הדפסה פתח את הפרויקט בסלייסר שלו, השווה גאומטריה, צבעים, חומרים, טמפרטורות, תמיכות והגדרות, ובצע Slice מחדש. הצלחת המרת ZIP או הימצאות פרופיל היצרן אינה הוכחה להדפסה פיזית תקינה.

## דרישות

ל־U1 אין תלות בהתקנת סלייסר לצורך יצירת הפלט. שלושת היעדים הנוספים דורשים *סלייסרים מותקנים עם פרופילי מדפסת מקוריים* במחשב שמבצע את ההמרה. כך המנוע משתמש בפועל ב-machine / process / filament native presets, כולל ה-G-code הייעודי לכל מכונה, ולא מחלק פרופיל Bambu ששמו שונה.

- **C5:** `C:\Program Files\Flashforge\Flash Studio Desktop\resources\profiles\` — פרופיל Flashforge Creator 5 0.4 nozzle.
- **CC2:** `C:\Program Files\ElegooSlicer\resources\profiles\` — פרופיל Elegoo Centauri Carbon 2 0.4 nozzle.
- **QIDI:** `C:\Program Files\OrcaSlicer\resources\profiles\` — פרופיל X 5 Series / X-Plus 5 0.4 nozzle.

בתצורה שאינה נתיב ברירת המחדל ניתן להגדיר `BRIDGE_SLICER_PROFILE_C5`, `BRIDGE_SLICER_PROFILE_CC2` או `BRIDGE_SLICER_PROFILE_QIDI` לנתיב תיקיית `resources/profiles` המתאימה. הממשק מאתר את הפרופילים בזמן הפתיחה; יעד חסר מופיע כבוי עם הסבר. אין צורך להעלות קבצים לאינטרנט או להגדיר API.

ממופה עד ארבעה חריצי חומר בלי איחוד צבעים אוטומטי. סוגי חומר מקומיים נתמכים: C5 — PLA/PETG/ABS/ASA; CC2 — PLA/PETG/ABS/ASA/TPU; QIDI — PLA/PETG/ABS/TPU. סוג לא מוכר או חומר עם נתון טמפרטורה/זרימה חסר נחסם במקום שיוחלף ב־PLA בשקט. עבור CC2 נדרשת מערכת CANVAS, ול־QIDI נדרשת QIDI Box כדי לבצע בפועל הדפסה רב־צבעית.

## מה הממיר מבצע

- משתמש ב־[bl2std](https://github.com/BryanSmee/bl_to_std) (Go, GPLv3), ב־commit נעול `4252f5f`, עם שכבת adapters ו-QA משלנו.
- מחליף את זהות המכונה ואת בסיס הגדרות היצרן; ממפה חומרי מקור לחריצים, משמר גאומטריה וצביעת משולשים וכן תת־קבוצה שמרנית של כוונות הדפסה משותפות, כגון layer height, walls, brim, supports.
- מעתיק הגדרות טמפרטורה, זרימה וקירור ממקורות היצרן של *סוג החומר המתאים לכל חריץ*; אין העתקה של G-code מכונה מ־Bambu. פרמטרי יצרן לא תואמים אינם מועתקים.
- מסיר מטמון G-code/plate sliced ישן, בודק שהמשולשים והקודקודים קיימים לאחר ההמרה, שהתוצאה משויכת למדפסת הנכונה, ושאף קובץ פלט קודם לא נדרס.
- לכל יעד תהליך עצמאי: תקלה ב־C5 לא מבטלת תוצאה ל־CC2/‏QIDI/‏U1.
- אין שירות ענן, אין מעקב Analytics ואין תקשורת עם מדפסות.

**מגבלות:** לא נבדקה עדיין תמיכה בקובצי MakerWorld מורכבים עם modifiers מיוחדים, multi-plate, גאומטריה שאינה נכנסת למשטח היעד או מעל ארבעה חריצי חומר. לא הוכח שכל מפתח הגדרה נשמר אחד־לאחד; הגדרות חומרה ו-G-code נלקחות בכוונה מהסלייסר היעד. בדיקה של Slice ממשי ומדפסות תידרש אחרי smoke ראשון של שלושת הסלייסרים.

## פיתוח ובדיקות Windows

נדרשים Node.js 24+, npm ו־Go 1.24.7+ לבניית הקוד. התקנת הכלים דורשת אינטרנט; *המרת קובצי 3MF עצמה מקומית בלבד*.

```powershell
git submodule update --init
.\scripts\build-engine.ps1
npm ci
npm test
npm run test:desktop
npm start
```

במחשב Chris מותקנת Go כתיקייה מקומית:
```powershell
.\scripts\build-engine.ps1 -GoExe 'D:\Velvet\Toolchains\go-1.24.7\go\bin\go.exe'
```

בדיקה נוספת *רק במחשב שבו שלושת הסלייסרים מותקנים*:
```powershell
python .\tests\make-mixed-fixture.py
npm run test:installed
```

הבדיקה הזו מפעילה שלוש המרות **במקביל** בשתי סדרות סינתטיות (חומר אחד ו־PLA/PETG/ABS) ומוודאת שישה 3MF חדשים שאינם זהים למקור. לבניית portable zip:

```powershell
npm run package:win
```

GitHub Actions מריץ בדיקות Go, יחידה ואינטגרציה, בדיקת פתיחת חלון Electron ומעלה Windows Portable ZIP. בפריסה למחשב נוסף יש להתקין את סלייסרי היצרן להמרות C5/CC2/QIDI.

## רישוי

קוד ההמרה bl2std נכלל כ־Git submodule תחת GPLv3; כללי רישוי המכלול מתועדים ב־`LICENSE` וב־`THIRD_PARTY_NOTICES.md`. קובצי פרופיל הסלייסר של Flashforge, ELEGOO ו־QIDI **אינם נכללים בריפו או במתקין**: הם נקראים מתוך ההתקנות המקומיות שברשות המשתמש.
