'use strict';
// MakerWorld setup wizard. Browser extension modes are never silently changed
// by this native app: only a confirmation in the extension can persist them.
(() => {
  const el = id => document.getElementById(id);
  const dialog = el('integrationsDialog');
  if (!dialog) return;
  const translateMap = {
    he: {
      integrationTitle:'חיבור ל־MakerWorld',
      integrationHint:'שלושה שלבים לפתיחת פרויקטים ב־SliceBridge, בלי לשנות את Bambu Studio.',
      stepOneTitle:'אפשר פתיחת קישורים במחשב',
      stepOneHint:'כך הדפדפן יוכל להעביר קובץ 3MF אל SliceBridge. פעולה חד־פעמית.',
      browserLinkUnknown:'בודק אם קישורי SliceBridge פועלים במחשב…',
      browserLinkReady:'מוכן: Windows יודע לפתוח קישורי SliceBridge.',
      browserLinkMissing:'נדרשת פעולה: לחץ על הכפתור כדי לאפשר ל־Windows לפתוח קישורים ב־SliceBridge.',
      browserLinkMac:'macOS: התמיכה בקישורים כלולה באפליקציה. ודא שפתחת את SliceBridge לפחות פעם אחת.',
      browserLinkError:'לא ניתן לבדוק את מצב הקישורים. אפשר לנסות שוב.',
      enableBrowserLinks:'אפשר קישורי SliceBridge',
      stepTwoTitle:'התקן את התוסף בדפדפן',
      stepTwoHint:'התקנה חד־פעמית לכל דפדפן. הקבצים מוכנים כאן במחשב — בלי GitHub.',
      browserLabel:'דפדפן',
      prepareExtension:'פתח תיקיית תוסף',
      installInstructionOne:'פתח בדפדפן את הכתובת:',
      installInstructionTwo:'הפעל Developer mode ואז לחץ Load unpacked (טען תוסף לא ארוז).',
      installInstructionThree:'בחר את תיקיית התוסף שנפתחה בלחיצה על הכפתור למעלה.',
      installInstructionTwoFirefox:'בחר This Firefox ולחץ Load Temporary Add-on.',
      installInstructionThreeFirefox:'בחר את הקובץ manifest.json מתוך תיקיית התוסף שנפתחה.',
      firefoxInstallWarning:'ב־Firefox זו התקנת בדיקה זמנית; יש לטעון מחדש אחרי סגירת הדפדפן.',
      folderPrepared:'התיקייה מוכנה ונפתחה. אם לא הופיעה, הנתיב הוא:',
      folderError:'לא הצלחנו להכין את קובצי התוסף: ',
      stepThreeTitle:'בחר מה יקרה בלחיצה על MakerWorld',
      stepThreeHint:'בחר אפשרות ואז אשר אותה בדפדפן שבחרת בשלב 2. הבחירה כאן לבדה אינה משנה את התוסף.',
      interceptOff:'כבוי — Bambu Studio כרגיל',
      interceptOffDetail:'SliceBridge לא יירט קישורים ממייקרוורלד.',
      interceptAsk:'שאל בכל פעם',
      interceptAskDetail:'בחר בין SliceBridge ל־Bambu Studio בכל פתיחת פרויקט.',
      interceptAlways:'תמיד דרך SliceBridge',
      interceptAlwaysDetail:'קישורי MakerWorld נתמכים יועברו ל־SliceBridge.',
      confirmInBrowser:'פתח בדפדפן ואשר בחירה',
      selectModeFirst:'קודם בחר אחת משלוש האפשרויות.',
      modeLaunched:'הדפדפן נפתח. אשר את הבחירה בחלון של תוסף SliceBridge. עד לאישור, המצב הקודם נשאר בתוקף.',
      modeNoExtension:'אם לא מופיע חלון אישור: בדוק שהתוסף מותקן ופעיל בדפדפן שנפתח.',
      modeError:'לא הצלחנו לפתוח את הדפדפן: ',
      modeScopeNote:'חשוב: ההגדרה נשמרת בכל דפדפן בנפרד, ורק לקישורים נתמכים באתר MakerWorld. ברירת המחדל של Bambu Studio נשארת ללא שינוי.',
      done:'סיום'
    },
    en: {
      integrationTitle:'MakerWorld Setup',
      integrationHint:'Three steps to open MakerWorld projects in SliceBridge without replacing Bambu Studio.',
      stepOneTitle:'Enable app links',
      stepOneHint:'Allow the browser to hand off 3MF files to SliceBridge. One-time setup.',
      browserLinkUnknown:'Checking whether SliceBridge links work on this computer…',
      browserLinkReady:'Ready: Windows can open SliceBridge links.',
      browserLinkMissing:'Action needed: click the button to let Windows open SliceBridge links.',
      browserLinkMac:'macOS: app-link support is included. Make sure you have launched SliceBridge at least once.',
      browserLinkError:'Could not verify browser-link setup. Try again.',
      enableBrowserLinks:'Enable SliceBridge links',
      stepTwoTitle:'Install the browser extension',
      stepTwoHint:'One-time setup per browser. The files are built into the app — no GitHub required.',
      browserLabel:'Browser',
      prepareExtension:'Open extension folder',
      installInstructionOne:'Open this address in your browser:',
      installInstructionTwo:'Turn on Developer mode and select Load unpacked.',
      installInstructionThree:'Select the extension folder that the button above opened.',
      installInstructionTwoFirefox:'Choose This Firefox and click Load Temporary Add-on.',
      installInstructionThreeFirefox:'Select manifest.json from the opened extension folder.',
      firefoxInstallWarning:'Firefox temporary add-ons disappear on browser restart; reload the test extension.',
      folderPrepared:'The extension folder is ready and opened. If it did not appear, the path is:',
      folderError:'Could not prepare the extension files: ',
      stepThreeTitle:'Choose how MakerWorld opens',
      stepThreeHint:'Select a mode, then confirm it in the browser chosen in step 2. This app does not silently change extension settings.',
      interceptOff:'Off — use Bambu Studio',
      interceptOffDetail:'SliceBridge will not intercept MakerWorld links.',
      interceptAsk:'Ask every time',
      interceptAskDetail:'Choose SliceBridge or Bambu Studio each time.',
      interceptAlways:'Always use SliceBridge',
      interceptAlwaysDetail:'Supported MakerWorld links will open in SliceBridge.',
      confirmInBrowser:'Open browser to confirm choice',
      selectModeFirst:'Select one of the three options first.',
      modeLaunched:'Your browser was opened. Confirm this choice in the SliceBridge extension prompt. The previous mode remains until you approve.',
      modeNoExtension:'No confirmation prompt? Check that the extension is installed and enabled in the browser that opened.',
      modeError:'Could not open the browser: ',
      modeScopeNote:'Important: each browser stores its own choice, and only supported MakerWorld links are affected. Bambu Studio remains your system default.',
      done:'Done'
    }
  };
  const t = key => (translateMap[document.documentElement.lang==='he'?'he':'en'][key] || key);
  const api = () => window.go?.main?.App || null;
  const browserAddresses = Object.freeze({
    edge:'edge://extensions',
    chrome:'chrome://extensions',
    brave:'brave://extensions',
    firefox:'about:debugging#/runtime/this-firefox'
  });
  let folderResult = null;
  let ready = false;
  const prepare = el('prepareExtensionBtn');
  const enable = el('enableLinksBtn');
  const confirm = el('applyBrowserModeBtn');
  const browser = el('browserTargetSelect');
  const modes = Array.from(dialog.querySelectorAll('input[name="makerworldMode"]'));

  function setMessage(target, text, error=false) {
    target.textContent = text;
    target.classList.toggle('error', Boolean(error));
  }
  function updateBrowserGuide() {
    const type = browser.value;
    el('browserSettingsAddress').textContent = browserAddresses[type] || browserAddresses.edge;
    el('firefoxInstallWarning').hidden = type !== 'firefox';
    const firefox = type === 'firefox';
    const steps = dialog.querySelectorAll('.integration-instructions li');
    steps[1].textContent = t(firefox?'installInstructionTwoFirefox':'installInstructionTwo');
    steps[2].textContent = t(firefox?'installInstructionThreeFirefox':'installInstructionThree');
  }
  function translate() {
    dialog.querySelectorAll('[data-i18n]').forEach(node => {node.textContent=t(node.dataset.i18n);});
    dialog.dir = document.documentElement.dir;
    updateBrowserGuide();
    if (folderResult) showPreparedFolder(folderResult.folder);
  }
  function showPreparedFolder(folder) {
    const status = el('extensionFolderStatus');
    status.replaceChildren(document.createTextNode(t('folderPrepared')+' '));
    const code = document.createElement('code');
    code.dir='ltr'; code.textContent=folder;
    status.append(code);
    status.classList.remove('error');
  }
  async function refreshStatus() {
    const status = el('browserLinkStatus');
    const stateClass = status.parentElement.classList;
    enable.hidden = true;
    enable.disabled = true;
    status.textContent = t('browserLinkUnknown');
    stateClass.remove('ready','missing','error');
    for (let i=0;i<30&&!api();i++) await new Promise(resolve=>setTimeout(resolve,70));
    if (!api()?.BrowserLinkStatus) {
      status.textContent = t('browserLinkError');
      stateClass.add('error');
      return;
    }
    try {
      const state = await api().BrowserLinkStatus();
      if (state.platform === 'darwin') {
        status.textContent = t('browserLinkMac');
        stateClass.add('ready');
      } else if (state.platform === 'windows') {
        ready = Boolean(state.registered);
        status.textContent = t(ready?'browserLinkReady':'browserLinkMissing');
        stateClass.add(ready?'ready':'missing');
        enable.hidden = ready;
        enable.disabled = ready;
      } else {
        status.textContent=t('browserLinkError');
        stateClass.add('error');
      }
    } catch {
      status.textContent=t('browserLinkError');
      stateClass.add('error');
    }
  }
  el('integrationsBtn').addEventListener('click',()=>{
    translate();
    dialog.showModal();
    refreshStatus();
  });
  el('closeIntegrationsBtn').addEventListener('click',()=>dialog.close());
  el('doneIntegrationsBtn').addEventListener('click',()=>dialog.close());
  dialog.addEventListener('click',event=>{if(event.target===dialog)dialog.close();});
  enable.addEventListener('click',async()=>{
    enable.disabled=true;
    const status=el('integrationStatus');
    if(!api()?.EnableBrowserLink){setMessage(status,t('browserLinkError'),true);return;}
    try {
      await api().EnableBrowserLink();
      setMessage(status,'');
      await refreshStatus();
    } catch(e) {
      setMessage(status,t('browserLinkError'),true);
      enable.disabled=false;
    }
  });
  browser.addEventListener('change',()=>{
    folderResult=null;
    setMessage(el('extensionFolderStatus'),'');
    updateBrowserGuide();
  });
  prepare.addEventListener('click',async()=>{
    prepare.disabled=true;
    const status=el('extensionFolderStatus');
    if(!api()?.PrepareBrowserExtension){
      setMessage(status,t('folderError')+t('browserLinkError'),true);
    } else {
      try {
        const result=await api().PrepareBrowserExtension(browser.value);
        folderResult=result;
        showPreparedFolder(result.folder);
      }catch(e){
        setMessage(status,t('folderError')+(e?.message||String(e)),true);
      }
    }
    prepare.disabled=false;
  });
  for(const radio of modes){
    radio.addEventListener('change',()=>{
      confirm.disabled=!modes.some(r=>r.checked);
      setMessage(el('modeStatus'),'');
    });
  }
  confirm.addEventListener('click',async()=>{
    const mode=modes.find(r=>r.checked)?.value;
    if(!mode){setMessage(el('modeStatus'),t('selectModeFirst'),true);return;}
    confirm.disabled=true;
    try{
      if(!api()?.OpenMakerWorldMode)throw new Error(t('browserLinkError'));
      await api().OpenMakerWorldMode(browser.value,mode);
      setMessage(el('modeStatus'),t('modeLaunched')+' '+t('modeNoExtension'));
    }catch(e){
      setMessage(el('modeStatus'),t('modeError')+(e?.message||String(e)),true);
    }finally{
      confirm.disabled=false;
    }
  });
  for(const lang of ['he','en']) el(lang+'Btn').addEventListener('click',translate);
  translate();
})();
