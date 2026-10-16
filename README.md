# 3MF Bridge

כלי Desktop מקומי להמרת פרויקטי Bambu Studio / MakerWorld מסוג `.3mf` למדפסות יעד.
ה־UI נשאר קטן: בחירת קובץ, בחירת יעד, המרה ושמירת התוצר ליד המקור.

## מצב מימוש מאומת — U1 Beta בלבד

| יעד | מצב |
| --- | --- |
| Snapmaker U1 (0.4mm baseline) | מנוע המרה משולב; עבר בדיקות סינתטיות |
| Flashforge Creator 5 | חסום עד שיהיה פרופיל ובדיקות תאימות |
| ELEGOO CC2 | חסום עד שיהיה פרופיל ובדיקות תאימות |
| QIDI Plus5 | חסום עד שיהיה פרופיל ובדיקות תאימות |

חשוב: **קובץ שנוצר אינו מאושר להדפסה אוטומטית**. יש לפתוח אותו ב־Snapmaker Orca, לוודא שהמכונה, הדיזה, החומרים, הטמפרטורות, התמיכות והמיקומים נכונים, ולבצע Slicing מחדש לפני הדפסה.

## מה מנוע ה־U1 מבצע

- משתמש בממיר [bl2std](https://github.com/BryanSmee/bl_to_std) (Go, GPLv3), ב־commit נעול: `4252f5f0a880463fd2e6aeed3125ec3653de60ee`.
- ממפה פילמנטים ו־paint data במודל, מחליף הגדרות מכונה ומייצר פרויקט יעד.
- מעביר רק תת־קבוצה שמרנית של פרמטרי Print Intent כשהם נתמכים בפרופיל היעד; אינו מעתיק G-code או הגדרות חומרה של Bambu.
- מסיר קובצי sliced G-code ומטמון plate ישן מתוך 3MF ומעדכן הפניות בקובצי OPC.
- בודק קיום גאומטריה, מספרי משולשים/קודקודים, מזהה U1 והיעדר מטמון G-code לפני שמירת הפלט.
- שומר בשם `U1-<source>.3mf`, ובמקרה התנגשות `U1-<source>-2.3mf` וכו', ללא דריסה.
- עובד מקומית בלבד. אין העלאה לענן, Analytics או קריאות API בזמן ההמרה.

המנוע **מסרב להמיר** קובצי sliced-only, ארכיונים פגומים, קבצים גדולים מדי, מעל ארבעה חריצי פילמנט והגדרות חומר שאינן מוכרות ל־U1. המטרה היא להתריע במקום לבצע מיפוי שעלול לשנות צבעים או הגדרות הדפסה ללא ידיעת המשתמש.

## הפעלה מקומית (Windows)

דרישות פיתוח: Node.js 22+, npm, Go 1.24.7+.

```powershell
git submodule update --init
.\scripts\build-engine.ps1
npm ci
npm test
npm start
```

אם Go מותקן בנתיב נפרד, אפשר להעביר:
```powershell
.\scripts\build-engine.ps1 -GoExe 'D:\Velvet\Toolchains\go-1.24.7\go\bin\go.exe'
```

בדיקת מנוע עצמאית:
```powershell
.\bin\bridge-core.exe --source 'C:\path\model.3mf' --output 'C:\path\U1-model.3mf' --target u1
```

ב־GitHub Actions, לאחר push לענף הפיתוח, נבדקות בדיקות Go ו־IPC, נוצרת גרסה ניידת ל־Windows ונשמר ZIP תחת Artifacts. אין לבנות מתקין כבד בסביבת עבודה עם מעט מקום בדיסק.

## בדיקות

```powershell
cd engine
go test ./...
cd ..
npm test
```

`tests/make_fixture.py` יוצר קובץ מקור סינתטי (`tests/fixtures/bambu-one-color.3mf`); הוא **אינו תחליף** לפרויקט MakerWorld אמיתי, בדיקת פתיחה ב־Snapmaker Orca או ניסיון הדפסה. נדרשים בהמשך לפחות: מודל MakerWorld רב־צלחות, מודל painted multi-color, מודל עם modifiers ופרופילים שונים, וכן מקרי קצה לכל ארבעת היעדים.

## רישוי וייחוס

המנוע `bl2std` משולב כ־Git submodule, ברישיון GPL-3.0. ייחוס וקוד מקור נשמרים ב־`third_party/bl2std`; ראה `THIRD_PARTY_NOTICES.md` ו־`LICENSE`.

לא שולב קוד מתוך `MkWorld2Snap` או `thadius83/bambu-to-snapmaker-u1` הכפופים ל־PolyForm Noncommercial. רישיון GPLv3 מאפשר עקרונית שימוש מסחרי אבל מחייב עמידה בתנאי הפצה וקוד מקור כשמפיצים מוצר נגזר. לפני הפצה חיצונית יש לבדוק את המכלול והרישיונות.

## מה חסר

1. בדיקת End-to-End על קובצי MakerWorld אמיתיים ו־Snapmaker Orca.
2. בדיקת טיפוס דיזה, שימור modifiers והגדרות per-object בפרויקטים מורכבים.
3. פרופילי יעד אמיתיים, בדיקות וסלייסרים עבור Creator 5, CC2, QIDI.
4. חתימת קובץ ההפעלה, אימות מול אנטי־וירוס, מנגנון עדכון ומתקין אם נדרש.

לא מתבצעת המרה מדומה עבור יעדים שלא אומתו.
