'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const frontend = path.join(__dirname, '..', 'frontend', 'dist');
const html = fs.readFileSync(path.join(frontend, 'index.html'), 'utf8');
const source = fs.readFileSync(path.join(frontend, 'app.js'), 'utf8');

function element(id) {
  return {
    id, textContent: '', title: '', className: '', dataset: {}, value: '',
    hidden: false, disabled: false, checked: false, open: false,
    children: [], handlers: {},
    classList: { toggle() {}, add() {}, remove() {} },
    addEventListener(type, fn) { this.handlers[type] = fn; },
    setAttribute() {},
    removeAttribute(name) { if (name === 'data-status') delete this.dataset.status; },
    replaceChildren(...children) { this.children = children; },
    append(...children) { this.children.push(...children); },
    appendChild(child) { this.children.push(child); },
    querySelector() { return element('nested'); },
    showModal() { this.open = true; },
    close() { this.open = false; },
    getBoundingClientRect() { return {width: 650, height: 500}; }
  };
}

function setupUI() {
  for (const id of ['checkUpdatesBtn', 'updateNotice', 'updateStatus', 'openUpdateBtn']) {
    assert.match(html, new RegExp('id="' + id + '"'), 'update controls must remain in native UI');
  }
  const ids = [
    'fileList', 'queueCount', 'resultList', 'reportCounts', 'convertBtn',
    'convertCount', 'activityText', 'activityDetail', 'activityIcon',
    'openFolderBtn', 'settingsDialog', 'clearQueueBtn', 'addFilesBtn',
    'addFolderBtn', 'selectAllBtn', 'settingsBtn', 'resultsPanel',
    'clearResultsBtn', 'installedVersion', 'checkUpdatesBtn',
    'updateNotice', 'updateStatus', 'openUpdateBtn', 'dismissUpdateBtn',
    'heBtn', 'enBtn', 'settingsTabs', 'settingsNozzle', 'settingsSlicer',
    'settingsMaterial', 'settingsFilament', 'settingsStatus',
    'closeSettingsBtn', 'cancelSettingsBtn', 'saveSettingsBtn',
    'dropzone', 'targetList'
  ];
  const elements = Object.fromEntries(ids.map((id) => [id, element(id)]));
  const targets = ['u1', 'c5', 'cc2', 'qidi'];
  const rows = Object.fromEntries(targets.map((target) => {
    const checkbox = element(target + '-checkbox');
    checkbox.value = target;
    checkbox.checked = true;
    const summary = element(target + '-summary');
    const row = element(target + '-row');
    row.querySelector = (selector) => selector === 'input' ? checkbox : summary;
    return [target, {checkbox, summary, row, gear: element(target + '-gear')}];
  }));
  const sent = [];
  const app = {
    GetSettings: async () => ({
      version: 1, window: {width: 1100, height: 750},
      printers: Object.fromEntries(targets.map(target => [target, {
        nozzle: 'auto', slicer: 'auto', filaments: {}
      }]))
    }),
    Availability: async () => targets.map(target => ({
      target, name: target, available: true, slicer: 'installed'
    })),
    GetVersion: async () => ({
      version: '0.4.0-beta.3', commit: 'abc123', platform: 'windows-amd64'
    }),
    CheckForUpdates: async () => ({
      status: 'update_available', latestVersion: '0.4.0-beta.4',
      releaseTag: 'v0.4.0-beta.4', assetName: '3MF-Bridge-Windows-native.zip'
    }),
    OpenUpdatePage: async (kind, tag) => {sent.push({kind, tag});}
  };
  const document = {
    getElementById(id) {
      assert.ok(elements[id], 'unexpected ID read by JS: ' + id);
      return elements[id];
    },
    querySelector(selector) {
      const target = selector.match(/data-(?:target-row|settings-target)="([^"]+)"/)?.[1];
      if (!target) return null;
      assert.ok(rows[target]);
      return selector.startsWith('[data-settings-target') ? rows[target].gear : rows[target].row;
    },
    querySelectorAll(selector) {
      if (selector.startsWith('#targetList input')) {
        return targets.map(target => rows[target].checkbox)
          .filter(x => x.checked && !x.disabled);
      }
      return [];
    },
    createElement: () => element('new'),
    createDocumentFragment: () => element('fragment'),
    documentElement: {lang: 'en', dir: 'ltr'},
    body: {dir: 'ltr'},
    addEventListener() {}
  };
  const win = {
    go: {main: {App: app}},
    addEventListener() {},
    runtime: {OnFileDrop() {}, EventsOn() {}}
  };
  const context = {
    document, navigator: {language: 'en-US'}, window: win,
    localStorage: {getItem: () => null, setItem() {}},
    setTimeout, clearTimeout,
    console: {warn() {}, error() {}},
    structuredClone
  };
  vm.runInNewContext(source, context, {filename: 'native-app.js'});
  return {elements, rows, app, sent, win};
}

async function settle() {
  await new Promise(resolve => setImmediate(resolve));
  await new Promise(resolve => setImmediate(resolve));
}

test('check updates offers native-trusted release page and translates into Hebrew', async () => {
  const {elements, sent} = setupUI();
  await settle();
  assert.equal(elements.installedVersion.textContent, 'v0.4.0-beta.3');
  await elements.checkUpdatesBtn.handlers.click();
  assert.equal(elements.updateStatus.dataset.status, 'update_available');
  assert.match(elements.updateStatus.textContent, /0.4.0-beta.4/);
  assert.equal(elements.updateNotice.hidden, false);
  assert.equal(elements.openUpdateBtn.hidden, false);
  await elements.openUpdateBtn.handlers.click();
  assert.deepEqual(sent, [{kind: 'release', tag: 'v0.4.0-beta.4'}]);
  elements.heBtn.handlers.click();
  assert.equal(elements.updateStatus.dataset.status, 'update_available');
  assert.match(elements.updateStatus.textContent, /גרסה חדשה/);
  assert.equal(elements.updateNotice.hidden, false);
});

test('private repository without published releases offers beta artifacts page', async () => {
  const {elements, app, sent} = setupUI();
  await settle();
  app.CheckForUpdates = async () => ({status: 'no_releases'});
  await elements.checkUpdatesBtn.handlers.click();
  assert.equal(elements.updateStatus.dataset.status, 'no_releases');
  assert.match(elements.updateStatus.textContent, /No GitHub Releases/);
  await elements.openUpdateBtn.handlers.click();
  assert.deepEqual(sent, [{kind: 'builds', tag: ''}]);
});

test('authentication and offline errors do not automatically download or install', async () => {
  const {elements, app, sent} = setupUI();
  await settle();
  app.CheckForUpdates = async () => ({status: 'authentication_required'});
  await elements.checkUpdatesBtn.handlers.click();
  assert.match(elements.updateStatus.textContent, /GitHub CLI/);
  await elements.openUpdateBtn.handlers.click();
  assert.deepEqual(sent, [{kind: 'releases', tag: ''}]);

  app.CheckForUpdates = async () => {throw new Error('offline')};
  await elements.checkUpdatesBtn.handlers.click();
  assert.equal(elements.updateStatus.dataset.status, 'error');
  assert.match(elements.updateStatus.textContent, /offline/);
  assert.equal(elements.openUpdateBtn.hidden, true);

  elements.dismissUpdateBtn.handlers.click();
  assert.equal(elements.updateNotice.hidden, true);
  assert.equal(sent.length, 1);
});

test('invalid preferences require explicit recovery and preserve backup location', async () => {
  const {elements, app, win} = setupUI();
  await settle();
  app.GetSettings = async () => { throw new Error('invalid settings JSON'); };
  let recovered = 0;
  win.confirm = () => false;
  app.RecoverInvalidSettings = async () => {
    recovered++;
    return {settings: {printers: {}}, backupPath: '/backup/settings.invalid.bak'};
  };
  await elements.settingsBtn.handlers.click();
  assert.equal(recovered, 0, 'declining confirmation must not recover');
  assert.equal(elements.settingsDialog.open, false);

  win.confirm = () => true;
  await elements.settingsBtn.handlers.click();
  assert.equal(recovered, 1);
  assert.equal(elements.settingsDialog.open, true);
  assert.match(elements.activityDetail.textContent, /backup\/settings\.invalid\.bak/);
});
