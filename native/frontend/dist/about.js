'use strict';

// Lightweight, offline About dialog. The installed application's embedded
// license/credit documents are authoritative; no GitHub fetch is needed.
(() => {
  const el = id => document.getElementById(id);
  const dialog = el('aboutDialog');
  if (!dialog) return;

  const translations = {
    he: {
      open:'אודות', title:'אודות SliceBridge',
      subtitle:'פרויקט אישי בקוד פתוח לחיבור בין סלייסרים.',
      version:'גרסה מותקנת', license:'רישיון וקוד פתוח',
      licenseSummary:'SliceBridge מופץ תחת GPL-3.0-only. הרישיון המלא מצורף לתוכנה.',
      credits:'תודות וקרדיטים',
      creditBl:'bl2std — מנוע המרה, GPLv3.',
      creditWails:'Wails — ממשק שולחני מקורי, MIT.',
      creditVendor:'פרופילי מדפסות נטענים מהסלייסרים שמותקנים אצל המשתמש; הם לא מצורפים לתוכנה.',
      fullLicense:'הצגת הרישיון המלא',
      fullCredits:'כל הקרדיטים וההודעות',
      hideDocument:'הסתר מסמך',
      source:'קוד המקור', releases:'גרסאות להורדה',
      offline:'פרטי הרישוי והקרדיטים זמינים גם ללא חיבור לאינטרנט. בדיקת עדכונים וקישורים ל־GitHub דורשים חיבור.',
      close:'סגירה', loading:'טוען מסמך מקומי…',
      unavailable:'המידע אינו זמין כרגע.', linkError:'לא ניתן לפתוח את הקישור.',
    },
    en: {
      open:'About', title:'About SliceBridge',
      subtitle:'A small, open-source project connecting slicers.',
      version:'Installed version', license:'License & open source',
      licenseSummary:'SliceBridge is licensed under GPL-3.0-only. The full license is bundled with the app.',
      credits:'Credits',
      creditBl:'bl2std — conversion engine, GPLv3.',
      creditWails:'Wails — native desktop framework, MIT.',
      creditVendor:'Printer presets are read from your installed slicers, not shipped with the app.',
      fullLicense:'Read the full license',
      fullCredits:'All credits & notices',
      hideDocument:'Hide document',
      source:'View source', releases:'Download builds',
      offline:'License and credit information works offline. Update checks and GitHub links need internet.',
      close:'Close', loading:'Reading bundled document…',
      unavailable:'This information is not available.', linkError:'Could not open the link.',
    }
  };

  const api = () => window.go?.main?.App || null;
  const lang = () => document.documentElement.lang === 'he' ? 'he' : 'en';
  const t = key => translations[lang()][key] || key;
  const doc = el('aboutLegalText');
  const legalStatus = el('aboutLegalStatus');
  const linkStatus = el('aboutLinkStatus');
  let shownDocument = null;

  function render() {
    dialog.dir = lang() === 'he' ? 'rtl' : 'ltr';
    el('aboutBtnLabel').textContent = t('open');
    el('aboutBtn').setAttribute('aria-label', t('open'));
    el('aboutBtn').title = t('open');
    dialog.querySelectorAll('[data-about-i18n]').forEach(node => {
      node.textContent = t(node.dataset.aboutI18n);
    });
    for (const [kind, id] of [['license', 'viewLicenseBtn'], ['credits', 'viewCreditsBtn']]) {
      el(id).textContent = shownDocument === kind ? t('hideDocument') : t(kind === 'license' ? 'fullLicense' : 'fullCredits');
    }
  }

  function close() {
    dialog.close();
  }

  async function showDocument(kind) {
    if (shownDocument === kind) {
      shownDocument = null;
      doc.textContent = '';
      doc.hidden = true;
      legalStatus.textContent = '';
      render();
      return;
    }
    shownDocument = kind;
    doc.hidden = false;
    doc.textContent = t('loading');
    legalStatus.textContent = '';
    render();
    try {
      if (!api()?.GetAboutLegalText) throw new Error('Unavailable');
      const content = await api().GetAboutLegalText(kind);
      if (shownDocument === kind) {
        doc.textContent = typeof content === 'string' && content.trim() ? content : t('unavailable');
      }
    } catch {
      if (shownDocument === kind) {
        doc.textContent = '';
        doc.hidden = true;
        legalStatus.textContent = t('unavailable');
      }
    }
  }

  async function openPage(kind) {
    linkStatus.textContent = '';
    try {
      if (!api()?.OpenAboutPage) throw new Error('Unavailable');
      await api().OpenAboutPage(kind);
    } catch {
      linkStatus.textContent = t('linkError');
    }
  }

  el('aboutBtn').addEventListener('click', () => {
    render();
    if (!dialog.open) dialog.showModal();
  });
  el('closeAboutBtn').addEventListener('click', close);
  el('doneAboutBtn').addEventListener('click', close);
  el('viewLicenseBtn').addEventListener('click', () => showDocument('license'));
  el('viewCreditsBtn').addEventListener('click', () => showDocument('credits'));
  el('aboutSourceBtn').addEventListener('click', () => openPage('source'));
  el('aboutReleasesBtn').addEventListener('click', () => openPage('releases'));
  dialog.addEventListener('close', () => {
    shownDocument = null;
    doc.textContent = '';
    doc.hidden = true;
    legalStatus.textContent = '';
    linkStatus.textContent = '';
    render();
  });
  for (const code of ['he', 'en']) {
    el(code + 'Btn').addEventListener('click', render);
  }
  render();
})();
