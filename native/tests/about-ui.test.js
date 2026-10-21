'use strict';
const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const base = path.resolve(__dirname, '..');
const html = fs.readFileSync(path.join(base, 'frontend/dist/index.html'), 'utf8');
const source = fs.readFileSync(path.join(base, 'frontend/dist/about.js'), 'utf8');
const css = fs.readFileSync(path.join(base, 'frontend/dist/app.css'), 'utf8');

function harness({developerMode = false, setFails = false} = {}) {
  const elements = new Map();
  function node(id) {
    if (elements.has(id)) return elements.get(id);
    const el = {
      id, textContent: '', hidden: false, open: false, dir: '', title: '',
      attributes: {}, dataset: {}, handlers: {},
      setAttribute(name, value) { this.attributes[name] = value; },
      addEventListener(event, fn) { this.handlers[event] = fn; },
      showModal() { this.open = true; },
      close() { this.open = false; this.handlers.close?.(); }
    };
    elements.set(id, el);
    return el;
  }
  const ids = [
    'aboutBtn', 'aboutBtnLabel', 'aboutDialog', 'closeAboutBtn',
    'doneAboutBtn', 'viewLicenseBtn', 'viewCreditsBtn',
    'aboutLegalText', 'aboutLegalStatus', 'aboutLinkStatus',
    'aboutSourceBtn', 'aboutReleasesBtn', 'heBtn', 'enBtn',
    'aboutTitle', 'aboutVersionHeading', 'developerModeBtn',
    'developerModeState', 'developerModeStatus'
  ];
  for (const id of ids) node(id);
  for (const [id, kind] of [
    ['aboutTitle', 'title'], ['aboutVersionHeading', 'version'],
    ['doneAboutBtn', 'close'], ['aboutSourceBtn', 'source'],
    ['aboutReleasesBtn', 'releases']
  ]) node(id).dataset.aboutI18n = kind;
  node('aboutDialog').querySelectorAll = selector => selector === '[data-about-i18n]'
    ? ids.map(node).filter(e => e.dataset.aboutI18n) : [];
  const visited = [], opened = [], dispatched = [];
  let nativeDeveloperMode = developerMode;
  const api = {
    GetAboutLegalText: async kind => {
      visited.push(kind);
      return kind === 'license'
        ? 'GNU GENERAL PUBLIC LICENSE\nVersion 3'
        : '# Third-party notices\nbl2std\nWails v2';
    },
    OpenAboutPage: async kind => { opened.push(kind); },
    GetDeveloperMode: async () => nativeDeveloperMode,
    SetDeveloperMode: async enabled => {
      if (setFails) throw new Error('denied');
      nativeDeveloperMode = enabled;
      return enabled;
    }
  };
  const document = {
    getElementById: node,
    documentElement: {lang: 'en'},
    dispatchEvent: event => { dispatched.push(event); }
  };
  class CustomEvent { constructor(type, init) { this.type = type; this.detail = init?.detail; } }
  vm.runInNewContext(source, {document, CustomEvent, window: {go: {main: {App: api}}}},
    {filename: 'about.js'});
  const click = id => node(id).handlers.click();
  return {node, document, visited, opened, dispatched, click};
}

test('About owns the single installed version and manual update controls', () => {
  const dialog = html.slice(html.indexOf('<dialog id="aboutDialog"'));
  const footerStart = html.indexOf('<footer class="actionbar">');
  const footer = html.slice(footerStart, html.indexOf('</footer>', footerStart));
  assert.ok(footerStart > 0);
  for (const id of ['installedVersion', 'checkUpdatesBtn', 'updateNotice', 'updateStatus', 'openUpdateBtn']) {
    assert.match(dialog, new RegExp('id="' + id + '"'));
    assert.equal((html.match(new RegExp('id="' + id + '"', 'g')) || []).length, 1);
    assert.doesNotMatch(footer, new RegExp('id="' + id + '"'));
  }
  assert.match(html, /id="aboutBtn"[^>]*aria-haspopup="dialog"/);
  assert.match(dialog, /id="viewLicenseBtn"/);
  assert.match(dialog, /id="viewCreditsBtn"/);
  assert.match(dialog, /id="aboutSourceBtn"/);
  assert.match(dialog, /id="aboutReleasesBtn"/);
  assert.match(css, /\.about-dialog/);
  assert.match(html, /<script defer src="\.\/about\.js"><\/script>/);
});

test('About opens in English, switches to Hebrew and closes normally', () => {
  const h = harness();
  assert.equal(h.node('aboutBtnLabel').textContent, 'About');
  h.click('aboutBtn');
  assert.equal(h.node('aboutDialog').open, true);
  assert.equal(h.node('aboutTitle').textContent, 'About SliceBridge');
  h.document.documentElement.lang = 'he';
  h.click('heBtn');
  assert.equal(h.node('aboutBtnLabel').textContent, 'אודות');
  assert.equal(h.node('aboutVersionHeading').textContent, 'גרסה מותקנת');
  assert.equal(h.node('aboutDialog').dir, 'rtl');
  h.click('doneAboutBtn');
  assert.equal(h.node('aboutDialog').open, false);
});

test('license and credits load locally, without network, and can be dismissed', async () => {
  const h = harness();
  h.click('aboutBtn');
  await h.click('viewLicenseBtn');
  assert.deepEqual(h.visited, ['license']);
  assert.match(h.node('aboutLegalText').textContent, /GNU GENERAL PUBLIC LICENSE/);
  assert.equal(h.node('aboutLegalText').hidden, false);
  await h.click('viewCreditsBtn');
  assert.deepEqual(h.visited, ['license', 'credits']);
  assert.match(h.node('aboutLegalText').textContent, /bl2std/);
  assert.match(h.node('aboutLegalText').textContent, /Wails/);
  await h.click('viewCreditsBtn');
  assert.equal(h.node('aboutLegalText').hidden, true);
  assert.equal(h.node('aboutLegalText').textContent, '');
  assert.doesNotMatch(source, /\bfetch\s*\(|\bXMLHttpRequest\b|\bwindow\.open\s*\(/);
});

test('GitHub actions delegate to allowlisted native methods only', async () => {
  const h = harness();
  await h.click('aboutSourceBtn');
  await h.click('aboutReleasesBtn');
  assert.deepEqual(h.opened, ['source', 'releases']);
  assert.deepEqual(h.visited, []);
});

test('Developer mode is a compact separate About card with a pressed toggle and advanced-only warning', () => {
  const card = html.slice(html.indexOf('<section class="about-card" aria-labelledby="developerModeHeading"'));
  const section = card.slice(0, card.indexOf('</section>'));
  assert.match(section, /<button id="developerModeBtn" type="button" class="compact-btn secondary" aria-pressed="false">/);
  assert.match(section, /data-about-i18n="devWarning"/);
  assert.match(section, /id="developerModeStatus"/);
  assert.equal((html.match(/id="developerModeBtn"/g) || []).length, 1);
  assert.ok(html.indexOf('id="developerModeBtn"') < html.indexOf('id="aboutLicenseHeading"'));
  for (const key of ['devTitle', 'devWarning', 'devButton', 'devOn', 'devOff', 'devError']) {
    assert.match(source, new RegExp(key + ':'));
  }
});

test('the bundled GPL and notices are exact copies of project files', () => {
  const license = fs.readFileSync(path.join(base, '..', 'LICENSE'));
  const credits = fs.readFileSync(path.join(base, '..', 'docs', 'legal', 'THIRD_PARTY_NOTICES.md'));
  assert.deepEqual(fs.readFileSync(path.join(base, 'frontend/dist/legal/LICENSE.txt')), license);
  assert.deepEqual(fs.readFileSync(path.join(base, 'frontend/dist/legal/THIRD_PARTY_NOTICES.md')), credits);
  assert.match(credits.toString('utf8'), /Wails v2/);
  assert.match(credits.toString('utf8'), /bl2std/);
});
