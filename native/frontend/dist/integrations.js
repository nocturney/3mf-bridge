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
      stepTwoHint:'בחר דפדפן — SliceBridge יציג את דרך ההתקנה הזמינה.',
      guidedBetaInstall:'התחל התקנת בטא מודרכת',
      installFromStore:'התקן מחנות הדפדפן',
      storeChecking:'בודק אפשרות התקנה רשמית…',
      storeReady:'התקנה רשמית זמינה. הדפדפן יבקש את אישורך.',
      storeNotPublished:'התוסף עדיין לא פורסם בחנות. כרגע זמינה התקנת בטא מודרכת בלבד.',
      betaInstallGuide:'הוראות התקנת בטא',
      betaInstallInfo:'התוסף כלול באפליקציה. הדפדפן יבקש ממך לאשר את הטעינה; אין התקנה שקטה.',
      openExtensionsPage:'פתח את מסך התוספים',
      betaOpened:'תיקיית התוסף ומסך התוספים נפתחו. אשר מצב מפתחים ובחר ״טען תוסף לא ארוז״.',
      betaOpenedFirefox:'התיקייה ודף התוספים נפתחו. בחר Load Temporary Add-on ואז manifest.json. זו התקנה זמנית.',
      storeOpened:'חנות התוספים נפתחה בדפדפן שבחרת. לחץ ״התקן״ בחנות לאישור.',
      browserLabel:'דפדפן',
      prepareExtension:'פתח תיקיית תוסף',
      installInstructionOne:'מסך התוספים בדפדפן:',
      installInstructionTwo:'הפעל Developer mode ואז לחץ Load unpacked (טען תוסף לא ארוז).',
      installInstructionThree:'בחר את תיקיית SliceBridge שנפתחה לך במחשב.',
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
      stepTwoHint:'Choose your browser and SliceBridge will show the available install route.',
      guidedBetaInstall:'Start guided beta install',
      installFromStore:'Install from browser store',
      storeChecking:'Checking official store availability…',
      storeReady:'Official store install is available. Your browser will ask for approval.',
      storeNotPublished:'The extension is not yet listed in a browser store. Guided beta installation is available.',
      betaInstallGuide:'Beta installation steps',
      betaInstallInfo:'The beta extension is included in the app. Your browser must approve the install; nothing is silently installed.',
      openExtensionsPage:'Open browser extensions page',
      betaOpened:'The extension folder and browser extensions page are open. Enable Developer mode and choose Load unpacked.',
      betaOpenedFirefox:'The folder and Firefox add-on page are open. Choose Load Temporary Add-on, then select manifest.json. This install is temporary.',
      storeOpened:'The browser store is open. Choose Install there to confirm.',
      browserLabel:'Browser',
      prepareExtension:'Open extension folder',
      installInstructionOne:'Browser extensions page:',
      installInstructionTwo:'Turn on Developer mode and select Load unpacked.',
      installInstructionThree:'Select the SliceBridge folder opened in File Explorer or Finder.',
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
  let storeLink = null;
  let storeRequest = 0;
  const install = el('installExtensionBtn');
  const betaDetails = el('betaInstallDetails');
  const openExtensions = el('openExtensionsBtn');
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
  function updateStoreStatus() {
    install.textContent=t(storeLink?'installFromStore':'guidedBetaInstall');
    setMessage(el('extensionStoreStatus'),t(storeLink?'storeReady':'storeNotPublished'));
  }
  // Both UI and native code require an exact approved store host and path.
  // The native code owns the canonical published URL and opens the browser.
  function isOfficialStoreLink(browserType, raw) {
    if(typeof raw!=='string')return false;
    try {
      const parsed=new URL(raw);
      const store=browserType==='brave'?'chrome':browserType;
      const host={edge:'microsoftedge.microsoft.com',
        chrome:'chromewebstore.google.com',firefox:'addons.mozilla.org'}[store];
      const prefix={edge:'/addons/detail/',
        chrome:'/detail/',firefox:'/en-US/firefox/addon/'}[store];
      return parsed.protocol==='https:' && parsed.host===host &&
        parsed.pathname.startsWith(prefix) && !parsed.search && !parsed.hash &&
        !parsed.username && !parsed.password;
    }catch{return false;}
  }
  // Blank means "not published": never send users to store search results.
  async function refreshStore() {
    const request=++storeRequest;
    const selected=browser.value;
    storeLink=null;
    install.disabled=true;
    setMessage(el('extensionStoreStatus'),t('storeChecking'));
    let candidate='';
    try {
      if(api()?.BrowserExtensionStoreURL)candidate=await api().BrowserExtensionStoreURL(selected);
    } catch { candidate=''; }
    if(request!==storeRequest || browser.value!==selected)return;
    storeLink=isOfficialStoreLink(selected,candidate)?candidate:null;
    install.disabled=false;
    updateStoreStatus();
  }
  function translate() {
    dialog.querySelectorAll('[data-i18n]').forEach(node => {node.textContent=t(node.dataset.i18n);});
    dialog.dir = document.documentElement.dir;
    updateBrowserGuide();
    updateStoreStatus();
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
    refreshStore();
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
    betaDetails.open=false;
    setMessage(el('extensionFolderStatus'),'');
    updateBrowserGuide();
    refreshStore();
  });
  async function prepareFiles(){
    if(!api()?.PrepareBrowserExtension)throw new Error(t('browserLinkError'));
    const result=await api().PrepareBrowserExtension(browser.value);
    if(!result||typeof result.folder!=='string'||!result.folder){
      throw new Error(t('folderError'));
    }
    folderResult=result;
    showPreparedFolder(result.folder);
  }
  async function openExtensionManager(){
    if(!api()?.OpenBrowserExtensionsPage)throw new Error(t('browserLinkError'));
    await api().OpenBrowserExtensionsPage(browser.value);
  }
  // Store = one explicit click on a verified listing. No store yet = one
  // explicit click to stage files and open the selected browser's add-on page.
  // The browser still requires approval; we never silently install an add-on.
  install.addEventListener('click',async()=>{
    install.disabled=true;
    browser.disabled=true;
    try {
      if(storeLink){
        if(!api()?.OpenBrowserExtensionStore)throw new Error(t('browserLinkError'));
        await api().OpenBrowserExtensionStore(browser.value);
        setMessage(el('extensionStoreStatus'),t('storeOpened'));
      }else{
        betaDetails.open=true;
        prepare.disabled=true;
        openExtensions.disabled=true;
        await prepareFiles();
        try {
          await openExtensionManager();
          setMessage(el('extensionStoreStatus'),t(browser.value==='firefox'?'betaOpenedFirefox':'betaOpened'));
        }catch(e){
          setMessage(el('extensionStoreStatus'),t('modeError')+(e?.message||String(e)),true);
        }
      }
    }catch(e){
      setMessage(el('extensionStoreStatus'),t('folderError')+(e?.message||String(e)),true);
    }finally{
      install.disabled=false;
      browser.disabled=false;
      prepare.disabled=false;
      openExtensions.disabled=false;
    }
  });
  prepare.addEventListener('click',async()=>{
    prepare.disabled=true;
    try {
      await prepareFiles();
    }catch(e){
      setMessage(el('extensionFolderStatus'),t('folderError')+(e?.message||String(e)),true);
    }finally{ prepare.disabled=false; }
  });
  openExtensions.addEventListener('click',async()=>{
    openExtensions.disabled=true;
    try{
      await openExtensionManager();
    }catch(e){
      setMessage(el('extensionFolderStatus'),t('modeError')+(e?.message||String(e)),true);
    }finally{ openExtensions.disabled=false; }
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
