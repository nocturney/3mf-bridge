'use strict';
const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

function setupUI() {
  const html = fs.readFileSync(path.join(__dirname, '..', 'frontend', 'dist', 'index.html'), 'utf8');
  assert.match(html, /id="checkUpdatesBtn"/);
  assert.match(html, /id="updateStatus"/);
  assert.match(html, /id="openUpdateBtn"/);
  const ids = [
    'pickFileBtn', 'convertBtn', 'openFolderBtn', 'fileMeta', 'statusList',
    'dropzone', 'heBtn', 'enBtn', 'installedVersion', 'checkUpdatesBtn',
    'updateStatus', 'openUpdateBtn'
  ];
  const elements = {};
  for (const id of ids) {
    elements[id] = {
      id, textContent: '', dataset: {}, hidden: false, disabled: false,
      handlers: {},
      addEventListener(type, fn) { this.handlers[type] = fn; },
      setAttribute() {}, removeAttribute(name) { if (name === 'data-status') delete this.dataset.status; },
      classList: {toggle() {}},
      replaceChildren() {}, appendChild() {},
      getBoundingClientRect() {return {width: 650, height: 500};}
    };
  }
  const sent = [];
  const app = {
    Availability: async () => [],
    GetVersion: async () => ({version: '0.4.0-beta.1', commit: 'abc123', platform: 'windows-amd64'}),
    CheckForUpdates: async () => ({
      status: 'update_available', latestVersion: '0.4.0-beta.2',
      releaseTag: 'v0.4.0-beta.2', assetName: '3MF-Bridge-Windows-native.zip'
    }),
    OpenUpdatePage: async (kind, tag) => {sent.push({kind, tag});}
  };
  const document = {
    getElementById: (id) => elements[id],
    querySelectorAll: () => [],
    querySelector: () => null,
    createElement: () => ({textContent: '', className: ''}),
    documentElement: {setAttribute() {}},
    body: {setAttribute() {}},
    addEventListener() {}
  };
  const context = {
    document, navigator: {language: 'en-US'},
    localStorage: {getItem: () => null, setItem() {}},
    screen: {availWidth: 1200, availHeight: 900},
    window: {go: {main: {App: app}}},
    requestAnimationFrame() {},
    setTimeout
  };
  vm.runInNewContext(fs.readFileSync(
    path.join(__dirname, '..', 'frontend', 'dist', 'app.js'), 'utf8'),
    context, {filename:'native-app.js'});
  return {elements, app, sent};
}
test('update button compares version, offers the validated release, and translates status', async () => {
  const {elements, sent} = setupUI();
  await new Promise((resolve) => setImmediate(resolve));
  assert.equal(elements.installedVersion.textContent, 'v0.4.0-beta.1');
  await elements.checkUpdatesBtn.handlers.click();
  assert.equal(elements.updateStatus.dataset.status, 'update_available');
  assert.match(elements.updateStatus.textContent, /0.4.0-beta.2/);
  assert.equal(elements.openUpdateBtn.hidden, false);
  await elements.openUpdateBtn.handlers.click();
  assert.deepEqual(sent, [{kind: 'release', tag: 'v0.4.0-beta.2'}]);
  elements.heBtn.handlers.click();
  assert.match(elements.updateStatus.textContent, /גרסה חדשה/);
});
test('private repository without published release offers beta-builds link', async () => {
  const {elements, app, sent} = setupUI();
  await new Promise((resolve) => setImmediate(resolve));
  app.CheckForUpdates = async () => ({status: 'no_releases'});
  await elements.checkUpdatesBtn.handlers.click();
  assert.equal(elements.updateStatus.dataset.status, 'no_releases');
  assert.match(elements.updateStatus.textContent, /No GitHub Releases/);
  await elements.openUpdateBtn.handlers.click();
  assert.deepEqual(sent, [{kind: 'builds', tag: ''}]);
});
test('auth required and offline statuses never offer an untrusted install', async () => {
  const {elements, app, sent} = setupUI();
  await new Promise((resolve) => setImmediate(resolve));
  app.CheckForUpdates = async () => ({status: 'authentication_required'});
  await elements.checkUpdatesBtn.handlers.click();
  assert.match(elements.updateStatus.textContent, /gh auth login/);
  await elements.openUpdateBtn.handlers.click();
  assert.deepEqual(sent, [{kind: 'releases', tag: ''}]);
  app.CheckForUpdates = async () => {throw new Error('offline')};
  await elements.checkUpdatesBtn.handlers.click();
  assert.equal(elements.updateStatus.dataset.status, 'error');
  assert.match(elements.updateStatus.textContent, /offline/);
  assert.equal(elements.openUpdateBtn.hidden, true);
});
