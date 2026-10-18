'use strict';

const messages = Object.freeze({
  he: {
    title: 'המרת קובצי Bambu 3MF',
    subtitle: 'בחר או גרור פרויקט 3MF, סמן מדפסות יעד, וקבל פרויקטים מותאמים בתיקיית המקור.',
    dropTitle: 'גרור לכאן קובץ 3MF',
    dropHelp: 'או לחץ על הכפתור לבחירה ידנית.',
    choose: 'בחירת קובץ',
    sourceTitle: 'קובץ מקור',
    targetsTitle: 'מדפסות יעד',
    statusTitle: 'סטטוס',
    convert: 'המרת הקובץ',
    openFolder: 'פתיחת תיקיית פלט',
    hint: 'ההמרה מקומית בלבד. יש לבדוק את הדיזה, החומר וה־Slice בסלייסר המתאים לפני הדפסה.',
    unnamed: 'עדיין לא נבחר קובץ.',
    name: 'שם', folder: 'מיקום', size: 'גודל',
    idle: 'ממתין לבחירת קובץ ומדפסת יעד אחת לפחות.',
    selected: 'נבחר קובץ',
    selectTarget: 'בחר מדפסת יעד אחת לפחות והפעל המרה.',
    noTargets: 'לא נבחרה מדפסת יעד.',
    noSource: 'בחר קובץ מקור תחילה.',
    checking: 'מאתר פרופילי סלייסר מקומיים...',
    available: 'פרופיל זמין', unavailable: 'פרופיל לא זמין',
    converting: 'מבצע המרות במקביל...',
    finished: 'נוצרו {n} פרויקטים בתיקיית המקור.',
    output: 'נוצר קובץ', error: 'ההמרה נכשלה', noOutputs: 'לא נוצר אף קובץ.',
    slicerHint: 'יש לפתוח כל קובץ בסלייסר המתאים, לבדוק חומרים ודיזה ולבצע Slice מחדש.',
    folderError: 'לא ניתן לפתוח את התיקייה',
    bridgeError: 'חיבור למעטפת המקומית נכשל',
    presumed: 'המקור לא ציין דיזה: נעשה שימוש בברירת מחדל משוערת של 0.4 מ״מ.'
  },
  en: {
    title: 'Convert Bambu 3MF projects',
    subtitle: 'Select or drop a 3MF project, choose one or more printers, and create converted projects next to the original.',
    dropTitle: 'Drop a 3MF project here',
    dropHelp: 'Or choose a file using the native file picker.',
    choose: 'Choose file',
    sourceTitle: 'Source project',
    targetsTitle: 'Target printers',
    statusTitle: 'Status',
    convert: 'Convert project',
    openFolder: 'Open output folder',
    hint: 'Conversion is completely local. Check nozzle, material and slicing in the target slicer before printing.',
    unnamed: 'No project selected yet.',
    name: 'Name', folder: 'Location', size: 'Size',
    idle: 'Waiting for a project and at least one target printer.',
    selected: 'Selected file',
    selectTarget: 'Select at least one target printer and start conversion.',
    noTargets: 'Select at least one target printer.',
    noSource: 'Choose a source project first.',
    checking: 'Checking local slicer profiles...',
    available: 'profile available', unavailable: 'profile unavailable',
    converting: 'Converting selected printers in parallel...',
    finished: 'Created {n} projects in the source folder.',
    output: 'Created', error: 'Conversion failed', noOutputs: 'No output was created.',
    slicerHint: 'Open every output in its target slicer, check materials and nozzle, and slice before printing.',
    folderError: 'Could not open the folder',
    bridgeError: 'Native bridge is not available',
    presumed: 'Source did not specify nozzle diameter: assuming 0.4 mm.'
  }
});

const updateTranslations = Object.freeze({
  he: {
    checkUpdates: 'בדיקת עדכונים',
    checkingUpdates: 'בודק עדכונים ב-GitHub…',
    updateAvailable: 'קיימת גרסה חדשה: {v}',
    latestInstalled: 'הגרסה המותקנת מעודכנת.',
    noPublishedReleases: 'עדיין לא פורסמו GitHub Releases. אפשר לבדוק בניות Beta.',
    platformUnavailable: 'קיימת גרסה {v}, אך אין הורדה מתאימה למחשב זה.',
    unsupportedPlatform: 'עדיין אין חבילת עדכון למערכת זו.',
    authRequired: 'המאגר פרטי. יש להתחבר ל-GitHub CLI באמצעות gh auth login.' ,
    updateError: 'בדיקת עדכונים נכשלה',
    openRelease: 'פתיחת גרסה ב-GitHub',
    openReleases: 'צפייה בגרסאות',
    openPreviewBuilds: 'צפייה בבניות Beta'
  },
  en: {
    checkUpdates: 'Check for updates',
    checkingUpdates: 'Checking GitHub Releases…',
    updateAvailable: 'New version available: {v}',
    latestInstalled: 'You already have the latest published version.',
    noPublishedReleases: 'No GitHub Releases are published yet. You can view beta builds.',
    platformUnavailable: 'Version {v} exists, but has no download for this computer.',
    unsupportedPlatform: 'No update package is available for this platform yet.',
    authRequired: 'This repository is private. Sign in with GitHub CLI using gh auth login.',
    updateError: 'Could not check for updates',
    openRelease: 'Open release on GitHub',
    openReleases: 'View GitHub releases',
    openPreviewBuilds: 'View beta builds'
  }
});

const targetNames = Object.freeze({
  u1: 'Snapmaker U1',
  c5: 'Flashforge Creator 5',
  cc2: 'ELEGOO CC2',
  qidi: 'QIDI Plus5'
});
const fileMeta = document.getElementById('fileMeta');
const statusList = document.getElementById('statusList');
const pickBtn = document.getElementById('pickFileBtn');
const convertBtn = document.getElementById('convertBtn');
const outputBtn = document.getElementById('openFolderBtn');
const dropzone = document.getElementById('dropzone');

let language = /^he\b/i.test(navigator.language || '') ? 'he' : 'en';
try {
  const stored = localStorage.getItem('3mfBridgeLanguage');
  if (stored === 'he' || stored === 'en') language = stored;
} catch { /* private native webviews may disallow localStorage */ }
let selectedFile = null;
let busy = false;
let available = [];
let installedVersion = null;
let updateResult = null;
let updateError = '';
let updateChecking = false;

function tr(name, attrs = {}) {
  let value = messages[language][name] || updateTranslations[language][name] || name;
  for (const [key, val] of Object.entries(attrs)) {
    value = value.replaceAll('{' + key + '}', String(val));
  }
  return value;
}
function setStatus(items) {
  statusList.replaceChildren();
  for (const item of items) {
    const li = document.createElement('li');
    li.className = 'status-item ' + (item.type || 'muted');
    li.textContent = item.text;
    statusList.appendChild(li);
  }
}
function api() {
  return window.go?.main?.App || null;
}
function renderUpdateStatus() {
  const checkButton = document.getElementById('checkUpdatesBtn');
  const versionLabel = document.getElementById('installedVersion');
  const statusLabel = document.getElementById('updateStatus');
  const openButton = document.getElementById('openUpdateBtn');
  versionLabel.textContent = installedVersion ? 'v' + installedVersion.version : '3MF Bridge';
  if (installedVersion?.commit) {
    versionLabel.title = installedVersion.commit;
  }
  checkButton.disabled = updateChecking;
  openButton.hidden = true;
  openButton.dataset.kind = '';
  openButton.dataset.tag = '';
  statusLabel.removeAttribute('data-status');
  if (updateChecking) {
    statusLabel.textContent = tr('checkingUpdates');
    return;
  }
  if (updateError) {
    statusLabel.textContent = tr('updateError') + ': ' + updateError;
    statusLabel.dataset.status = 'error';
    return;
  }
  if (!updateResult) {
    statusLabel.textContent = '';
    return;
  }
  const result = updateResult;
  statusLabel.dataset.status = result.status;
  switch (result.status) {
    case 'update_available':
      statusLabel.textContent = tr('updateAvailable', {v: result.latestVersion});
      openButton.hidden = false;
      openButton.textContent = tr('openRelease');
      openButton.dataset.kind = 'release';
      openButton.dataset.tag = result.releaseTag;
      break;
    case 'up_to_date':
      statusLabel.textContent = tr('latestInstalled');
      break;
    case 'no_releases':
      statusLabel.textContent = tr('noPublishedReleases');
      openButton.hidden = false;
      openButton.textContent = tr('openPreviewBuilds');
      openButton.dataset.kind = 'builds';
      break;
    case 'platform_unavailable':
      statusLabel.textContent = tr('platformUnavailable', {v: result.latestVersion});
      openButton.hidden = false;
      openButton.textContent = tr('openReleases');
      openButton.dataset.kind = 'releases';
      break;
    case 'authentication_required':
      statusLabel.textContent = tr('authRequired');
      openButton.hidden = false;
      openButton.textContent = tr('openReleases');
      openButton.dataset.kind = 'releases';
      break;
    default:
      statusLabel.textContent = tr('unsupportedPlatform');
  }
}
async function checkForUpdates() {
  if (updateChecking || !api()?.CheckForUpdates) return;
  updateChecking = true;
  updateError = '';
  updateResult = null;
  renderUpdateStatus();
  try {
    updateResult = await api().CheckForUpdates();
  } catch (error) {
    updateError = error?.message || String(error);
  } finally {
    updateChecking = false;
    renderUpdateStatus();
  }
}
async function openUpdatePage() {
  const button = document.getElementById('openUpdateBtn');
  const kind = button.dataset.kind;
  if (!kind || !api()?.OpenUpdatePage) return;
  try {
    await api().OpenUpdatePage(kind, button.dataset.tag || '');
  } catch (error) {
    updateError = error?.message || String(error);
    renderUpdateStatus();
  }
}

function renderFile() {
  if (!selectedFile) {
    fileMeta.textContent = tr('unnamed');
    return;
  }
  fileMeta.textContent = [
    tr('name') + ': ' + selectedFile.name,
    tr('folder') + ': ' + selectedFile.folder,
    tr('size') + ': ' + selectedFile.sizeLabel
  ].join('\n');
}
function selectLanguage(lang) {
  language = lang;
  document.documentElement.setAttribute('lang', lang);
  document.documentElement.setAttribute('dir', lang === 'he' ? 'rtl' : 'ltr');
  document.body.setAttribute('dir', lang === 'he' ? 'rtl' : 'ltr');
  try { localStorage.setItem('3mfBridgeLanguage', lang); } catch {}
  document.querySelectorAll('[data-i18n]').forEach((el) => {
    el.textContent = tr(el.dataset.i18n);
  });
  document.getElementById('heBtn').classList.toggle('active', lang === 'he');
  document.getElementById('enBtn').classList.toggle('active', lang === 'en');
  document.getElementById('heBtn').setAttribute('aria-pressed', lang === 'he');
  document.getElementById('enBtn').setAttribute('aria-pressed', lang === 'en');
  renderFile();
  renderTargetAvailability();
  setStatus([{ text: selectedFile ? tr('selectTarget') : tr('idle') }]);
  renderUpdateStatus();
  fitWindow();
}
function fitWindow() {
  requestAnimationFrame(() => requestAnimationFrame(() => {
    const bounds = document.querySelector('.shell').getBoundingClientRect();
    const desiredW = Math.max(640, Math.ceil(bounds.width + 64));
    const desiredH = Math.max(460, Math.ceil(bounds.height + 96));
    const maxW = Math.max(640, screen.availWidth - 28);
    const maxH = Math.max(460, screen.availHeight - 34);
    if (api()?.FitWindow) {
      api().FitWindow(Math.min(desiredW, maxW), Math.min(desiredH, maxH))
        .catch(() => {}); // window can already be closing
    }
  }));
}
function renderTargetAvailability() {
  for (const state of available) {
    const box = document.querySelector('input[value="' + state.target + '"]');
    const text = document.querySelector('[data-target="' + state.target + '"]');
    if (!box || !text) continue;
    box.disabled = !state.available;
    if (!state.available) box.checked = false;
    text.textContent = targetNames[state.target] +
      (state.available ? '' : ' — ' + tr('unavailable'));
    text.title = state.available ? state.slicer : (state.error || tr('unavailable'));
  }
}
async function refresh() {
  if (!api()) return;
  setStatus([{ text: tr('checking') }]);
  try {
    available = await api().Availability();
    renderTargetAvailability();
    setStatus([{ text: tr('idle') }]);
    fitWindow();
  } catch (error) {
    setStatus([{ type: 'error', text: tr('bridgeError') + ': ' + error.message }]);
  }
}
function showSelectedFile(info) {
  selectedFile = info;
  outputBtn.disabled = true;
  renderFile();
  setStatus([
    { type: 'success', text: tr('selected') + ': ' + info.name },
    { text: tr('selectTarget') }
  ]);
}
async function chooseFile() {
  if (busy || !api()) return;
  try {
    const pick = await api().SelectFile(language);
    if (!pick.canceled && pick.file) showSelectedFile(pick.file);
  } catch (e) {
    setStatus([{ type: 'error', text: e.message }]);
  }
}
async function processDrop(paths) {
  if (busy || !Array.isArray(paths) || paths.length === 0 || !api()) return;
  try {
    showSelectedFile(await api().CheckFile(paths[0]));
  } catch (e) {
    setStatus([{ type: 'error', text: e.message }]);
  }
}
async function convert() {
  if (busy || !api()) return;
  if (!selectedFile) {
    setStatus([{ type: 'warning', text: tr('noSource') }]); return;
  }
  const targets = Array.from(document.querySelectorAll('.target-grid input:checked'))
    .map((el) => el.value);
  if (!targets.length) {
    setStatus([{ type: 'warning', text: tr('noTargets') }]); return;
  }
  busy = true;
  pickBtn.disabled = convertBtn.disabled = true;
  outputBtn.disabled = true;
  setStatus([{ text: tr('converting') }]);
  try {
    const result = await api().Convert({filePath: selectedFile.path, targets});
    const lines = [];
    if (result.results?.length) {
      lines.push({type: 'success', text: tr('finished', {n: result.results.length})});
      for (const item of result.results) {
        // Output reports target nozzle, including the approved silent
        // Creator 5 .2 -> .25 conversion, without an interrupting dialog.
        lines.push({
          type: 'success',
          text: tr('output') + ': ' + item.outputName +
            (item.report?.target_nozzle ? ' (' + item.report.target_nozzle + ' mm)' : '')
        });
      }
      lines.push({type: 'warning', text: tr('slicerHint')});
      outputBtn.disabled = false;
    }
    if (result.sourceNozzleAssumed) lines.push({type: 'warning', text: tr('presumed')});
    for (const failure of result.failures || []) {
      lines.push({
        type: 'error',
        text: targetNames[failure.target] + ': ' + failure.error +
          (failure.code ? ' (' + failure.code + ')' : '')
      });
    }
    if (!lines.length) lines.push({
      type: 'error', text: result.error || tr('noOutputs')
    });
    setStatus(lines);
  } catch (error) {
    setStatus([{type: 'error', text: tr('error') + ': ' + error.message}]);
  } finally {
    busy = false;
    pickBtn.disabled = convertBtn.disabled = false;
  }
}
async function openFolder() {
  if (!api()) return;
  try { await api().OpenOutputFolder(); }
  catch (e) { setStatus([{ type: 'error', text: tr('folderError') + ': ' + e.message }]); }
}

document.getElementById('checkUpdatesBtn').addEventListener('click', checkForUpdates);
document.getElementById('openUpdateBtn').addEventListener('click', openUpdatePage);
pickBtn.addEventListener('click', chooseFile);
convertBtn.addEventListener('click', convert);
outputBtn.addEventListener('click', openFolder);
document.getElementById('heBtn').addEventListener('click', () => selectLanguage('he'));
document.getElementById('enBtn').addEventListener('click', () => selectLanguage('en'));
dropzone.addEventListener('keydown', (event) => {
  if (event.key === 'Enter' || event.key === ' ') { event.preventDefault(); chooseFile(); }
});
// Prevent navigation for native file drops.
document.addEventListener('dragover', (event) => event.preventDefault());
document.addEventListener('drop', (event) => event.preventDefault());

selectLanguage(language);
(async () => {
  for (let tries = 0; tries < 60 && !api(); tries++) {
    await new Promise((resolve) => setTimeout(resolve, 100));
  }
  if (!api()) {
    setStatus([{ type: 'error', text: tr('bridgeError') }]);
    convertBtn.disabled = true;
    return;
  }
  if (window.runtime?.OnFileDrop) {
    window.runtime.OnFileDrop((_x, _y, paths) => processDrop(paths), true);
  }
  try {
    installedVersion = await api().GetVersion();
    renderUpdateStatus();
  } catch (error) {
    installedVersion = {version: 'unknown'};
    renderUpdateStatus();
  }
  await refresh();
})();
