'use strict';
// Native MakerWorld onboarding. The extension stores intercept mode in
// browser-local storage; the app never claims to read/modify browser settings.
(() => {
  const el = id => document.getElementById(id);
  const dialog=el('integrationsDialog');
  if(!dialog)return;
  const translations={
    he:{
      integrationTitle:'חיבור ל־MakerWorld',
      integrationHint:'פתיחת פרויקטים מ־MakerWorld ישירות ב־SliceBridge, בלי להחליף את Bambu Studio.',
      extensionExplanation:'התוסף פועל רק באתר MakerWorld. הוא לא קורא סיסמאות או Cookies, ולא משנה קישורי Bambu במערכת.',
      interceptModesTitle:'בחר את ברירת המחדל בחלון התוסף בדפדפן',
      interceptOff:'כבוי (ברירת המחדל)',
      interceptOffDetail:'Bambu Studio ממשיך להיפתח כרגיל.',
      interceptAsk:'שאל בכל פעם',
      interceptAskDetail:'בחירה בין SliceBridge, ‏Bambu Studio או ביטול בכל פתיחה.',
      interceptAlways:'תמיד ב־SliceBridge',
      interceptAlwaysDetail:'קישורים נתמכים מ־MakerWorld מועברים אוטומטית ל־SliceBridge.',
      modeScopeNote:'מצב היירוט נשמר בתוסף של כל דפדפן בנפרד. כדי לשנות אותו, לחץ על סמל התוסף בדפדפן. התוכנה אינה יכולה לזהות אם התוסף מותקן.',
      browserLinkUnknown:'בודק אם Windows/macOS מוכן לפתיחת קישורים…',
      browserLinkReady:'SliceBridge מוגדר לקבל קישורי דפדפן במחשב זה.',
      browserLinkMissing:'נדרש אישור חד־פעמי להפעלת קישורי SliceBridge.',
      browserLinkError:'לא ניתן לבדוק את שיוך הקישור.',
      openExtensionGuide:'מדריך התקנה לתוסף',
      enableBrowserLinks:'הפעל קישורי דפדפן',
      browserLinkMac:'היישום כולל תמיכה בקישורים ב־macOS. ייתכן שהמערכת תבקש אישור בפתיחה הראשונה.',
      linkNeedsApp:'יש לפתוח את SliceBridge עצמו כדי להפעיל את הקישור. אין שינוי לשיוך הקיים של Bambu.',
      authHint:'אם MakerWorld מבקש התחברות או בדיקת אבטחה, השתמש בהורדת 3MF הרגילה באתר.'
    },
    en:{
      integrationTitle:'MakerWorld Setup',
      integrationHint:'Open MakerWorld projects in SliceBridge without replacing Bambu Studio.',
      extensionExplanation:'The extension runs only on MakerWorld. It does not read cookies or passwords or change Bambu system links.',
      interceptModesTitle:'Choose the default in the extension popup',
      interceptOff:'Off (default)',
      interceptOffDetail:'Bambu Studio opens normally.',
      interceptAsk:'Ask every time',
      interceptAskDetail:'Choose SliceBridge, Bambu Studio or Cancel on each link.',
      interceptAlways:'Always open in SliceBridge',
      interceptAlwaysDetail:'Supported MakerWorld links are routed to SliceBridge automatically.',
      modeScopeNote:'The interception mode is stored separately in each browser extension. Click the extension icon in your browser to change it. The app cannot reliably detect whether the extension is installed.',
      browserLinkUnknown:'Checking OS browser-link association…',
      browserLinkReady:'SliceBridge is registered to receive browser links on this computer.',
      browserLinkMissing:'One-time approval required to enable SliceBridge browser links.',
      browserLinkError:'Could not check browser link association.',
      openExtensionGuide:'Extension setup guide',
      enableBrowserLinks:'Enable browser links',
      browserLinkMac:'The macOS app bundle supports SliceBridge links. macOS may ask for confirmation at first launch.',
      linkNeedsApp:'Approve the SliceBridge link handler. Bambu associations stay unchanged.',
      authHint:'If MakerWorld requires sign-in or verification, use its normal 3MF download.'
    }
  };
  const t = key => translations[document.documentElement.lang==='he'?'he':'en'][key]||key;
  const api=()=>window.go?.main?.App||null;
  function translate(){
    dialog.querySelectorAll('[data-i18n]').forEach(node=>{node.textContent=t(node.dataset.i18n);});
    dialog.querySelectorAll('[data-i18n-title]').forEach(node=>node.title=t(node.dataset.i18nTitle));
    dialog.dir=document.documentElement.dir;
  }
  async function refreshStatus(){
    const status=el('browserLinkStatus');
    const button=el('enableLinksBtn');
    button.disabled=true;
    status.textContent=t('browserLinkUnknown');
    for(let i=0;i<40&&!api();i++)await new Promise(resolve=>setTimeout(resolve,70));
    if(!api()?.BrowserLinkStatus){
      status.textContent=t('browserLinkError');return;
    }
    try{
      const state=await api().BrowserLinkStatus();
      if(state.platform==='darwin'){
        status.textContent=t('browserLinkMac');
        button.hidden=true;
      }else{
        status.textContent=t(state.registered?'browserLinkReady':'browserLinkMissing');
        button.hidden=!!state.registered;
        button.disabled=!!state.registered;
      }
    }catch(e){status.textContent=t('browserLinkError')+' '+(e.message||String(e));}
  }
  el('integrationsBtn').addEventListener('click',()=>{
    translate();dialog.showModal();refreshStatus();
  });
  el('closeIntegrationsBtn').addEventListener('click',()=>dialog.close());
  dialog.addEventListener('click',event=>{
    if(event.target===dialog)dialog.close();
  });
  el('openExtensionGuideBtn').addEventListener('click',async ()=>{
    const status=el('integrationStatus');
    if(!api()?.OpenBrowserSetup){status.textContent=t('browserLinkError');return;}
    try{await api().OpenBrowserSetup();status.textContent='';}
    catch(err){status.textContent=err.message||String(err);}
  });
  el('enableLinksBtn').addEventListener('click',async ()=>{
    const status=el('integrationStatus');
    if(!api()?.EnableBrowserLink){status.textContent=t('browserLinkError');return;}
    try{
      await api().EnableBrowserLink();
      status.textContent='';
      await refreshStatus();
    }catch(err){status.textContent=err.message||String(err);}
  });
  for(const lang of ['he','en']){
    el(lang+'Btn').addEventListener('click',translate);
  }
  translate();
})();
