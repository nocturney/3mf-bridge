const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs/promises');
const path = require('node:path');
const os = require('node:os');
const vm = require('node:vm');
const Module = require('node:module');

const handlers = new Map();
const fakeElectron = {
  app: {
    isPackaged: false,
    whenReady: () => new Promise(() => {}),
    on: () => {}
  },
  BrowserWindow: class {},
  ipcMain: { handle: (name, callback) => handlers.set(name, callback) },
  dialog: { showOpenDialog: async () => ({ canceled: true, filePaths: [] }) },
  shell: { openPath: async () => '' }
};
const originalLoad = Module._load;
try {
  Module._load = function(request, parent, isMain) {
    if (request === 'electron') return fakeElectron;
    return originalLoad.call(this, request, parent, isMain);
  };
  require('../main.js');
} finally {
  Module._load = originalLoad;
}

const fixture = path.join(__dirname, 'fixtures', 'bambu-one-color.3mf');

test('renderer script parses and copy-only conversion is removed', async () => {
  const html = await fs.readFile(path.join(__dirname, '..', 'index.html'), 'utf8');
  const script = html.match(/<script>([\s\S]*?)<\/script>/);
  assert.ok(script, 'renderer script exists');
  assert.doesNotThrow(() => new vm.Script(script[1]));
  const main = await fs.readFile(path.join(__dirname, '..', 'main.js'), 'utf8');
  assert.ok(!main.includes('fs.copyFile(filePath, outputPath)'), 'must not fake conversion');
});

test('IPC converts U1, preserves source, creates collision-safe outputs', async () => {
  const dir = await fs.mkdtemp(path.join(os.tmpdir(), '3mf-bridge-ipc-'));
  try {
    const input = path.join(dir, 'model.3mf');
    await fs.copyFile(fixture, input);
    const before = await fs.readFile(input);
    const convert = handlers.get('convert-3mf');
    assert.equal(typeof convert, 'function');

    const first = await convert(null, { filePath: input, targets: ['u1'] });
    assert.equal(first.ok, true, JSON.stringify(first));
    assert.equal(first.results[0].outputName, 'U1-model.3mf');
    assert.equal(first.results[0].report.engine, 'bl2std');
    assert.ok(first.results[0].report.removed_slice_cache_entries > 0);
    const converted = await fs.readFile(first.results[0].outputPath);
    assert.ok(!converted.equals(before), 'converted 3MF must not be byte-identical');
    assert.ok((await fs.readFile(input)).equals(before), 'original must never change');

    const second = await convert(null, { filePath: input, targets: ['u1'] });
    assert.equal(second.ok, true, JSON.stringify(second));
    assert.equal(second.results[0].outputName, 'U1-model-2.3mf');
    assert.ok((await fs.readFile(first.results[0].outputPath)).equals(converted));
  } finally {
    await fs.rm(dir, { recursive: true, force: true });
  }
});

test('IPC blocks unverified printer targets before writing any output', async () => {
  const dir = await fs.mkdtemp(path.join(os.tmpdir(), '3mf-bridge-unsupported-'));
  try {
    const input = path.join(dir, 'model.3mf');
    await fs.copyFile(fixture, input);
    for (const targets of [['c5'], ['cc2'], ['qidi'], ['u1', 'c5']]) {
      const result = await handlers.get('convert-3mf')(null, { filePath: input, targets });
      assert.equal(result.ok, false, JSON.stringify(result));
      assert.equal(result.code, 'TARGET_NOT_VALIDATED');
    }
    assert.deepEqual(await fs.readdir(dir), ['model.3mf']);
  } finally {
    await fs.rm(dir, { recursive: true, force: true });
  }
});

test('IPC refuses invalid .3mf archives without false success', async () => {
  const dir = await fs.mkdtemp(path.join(os.tmpdir(), '3mf-bridge-invalid-'));
  try {
    const input = path.join(dir, 'broken.3mf');
    await fs.writeFile(input, 'not zip data');
    const result = await handlers.get('convert-3mf')(null, { filePath: input, targets: ['u1'] });
    assert.equal(result.ok, false);
    assert.equal(result.code, 'SOURCE_INVALID');
    assert.deepEqual(await fs.readdir(dir), ['broken.3mf']);
  } finally {
    await fs.rm(dir, { recursive: true, force: true });
  }
});
