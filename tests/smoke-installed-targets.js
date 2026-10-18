'use strict';

// Windows-only integration: relies on the four actual slicer profile installs.
const path = require('node:path');
const fs = require('node:fs/promises');
const os = require('node:os');
const assert = require('node:assert/strict');
const { execFile } = require('node:child_process');
const { promisify } = require('node:util');
const run = promisify(execFile);
const { loadTargetBundle } = require('../slicer-profiles');

const exe = path.join(__dirname, '..', 'bin', 'bridge-core.exe');

async function execute(...args) {
  try {
    const { stdout } = await run(exe, args, {
      windowsHide: true, timeout: 120000, maxBuffer: 256*1024
    });
    return JSON.parse(stdout);
  } catch (error) {
    throw new Error(error.stdout || error.stderr || error.message);
  }
}

async function oneTarget(target, root, fixtureName) {
  const dir = path.join(root, target + '-' + fixtureName.replace('.3mf', ''));
  await fs.mkdir(dir);
  const source = path.join(dir, 'bambu.3mf');
  const output = path.join(dir, target + '-bambu.3mf');
  const bundlePath = path.join(dir, 'native-vendor-profile.json');
  await fs.copyFile(path.join(__dirname, 'fixtures', fixtureName), source);
  const nozzle = await execute('--inspect-nozzle', '--source', source);
  assert.equal(nozzle.ok, true);
  const bundle = await loadTargetBundle(target, { nozzle: nozzle.source_nozzle });
  await fs.writeFile(bundlePath, JSON.stringify(bundle), { mode: 0o600 });

  const args = [
    '--source', source, '--output', output,
    '--target', target
  ];
  if (target !== 'u1' || bundle.nozzle !== '0.4') {
    args.push('--profile', bundlePath);
  }
  const result = await execute(...args);
  assert.equal(result.ok, true, JSON.stringify(result));
  assert.equal(result.target, target);
  assert.equal(result.source_nozzle, nozzle.source_nozzle);
  assert.equal(result.target_nozzle, bundle.nozzle);
  assert.equal(result.nozzle_remapped,
    nozzle.source_nozzle !== bundle.nozzle);
  assert.equal(result.geometry_triangles, 4);
  assert.equal(result.geometry_vertices, 4);
  assert.ok(result.removed_slice_cache_entries >= 3);
  const [oldBytes, newBytes] = await Promise.all([
    fs.readFile(source), fs.readFile(output)
  ]);
  assert.ok(!oldBytes.equals(newBytes), 'source should be transformed, not copied');
  if (fixtureName.includes('overheight')) {
    assert.ok(!result.preserved_print_keys.includes('layer_height'),
      'invalid Bambu layer height must not override target nozzle range');
  }
  if (fixtureName.startsWith('bambu-nozzle-')) {
    const actual = await execute('--inspect-nozzle', '--source', output);
    assert.equal(actual.source_nozzle, bundle.nozzle,
      'output nozzle settings do not match target slicer');
  }
  console.log('VENDOR_CONVERSION_PASS', target, fixtureName, JSON.stringify({
    model: bundle.profile.printer_model_id,
    sourceNozzle: nozzle.source_nozzle,
    targetNozzle: bundle.nozzle,
    nativeSettings: Object.keys(bundle.profile.project_settings).length,
    triangleCount: result.geometry_triangles,
    bytes: newBytes.byteLength
  }));
}

(async () => {
  const root = await fs.mkdtemp(path.join(os.tmpdir(), '3mf-bridge-nozzle-'));
  try {
    const fixtures = [
      'bambu-one-color.3mf',
      'bambu-mixed-three.3mf',
      ...['0.2','0.4','0.6','0.8'].map(
        (n) => 'bambu-nozzle-' + n + '.3mf'),
      ...['0.2','0.6','0.8'].map(
        (n) => 'bambu-mixed-nozzle-' + n + '.3mf'),
      'bambu-nozzle-0.2-overheight.3mf'
    ];
    // Each target runs independently and concurrently with the other three,
    // limiting unnecessary repetitive loading of vendor JSON on slow PCs.
    const targets = ['u1','c5','cc2','qidi'];
    const results = await Promise.allSettled(targets.map(async (target) => {
      for (const fixtureName of fixtures) {
        await oneTarget(target, root, fixtureName);
      }
    }));
    for (let i = 0; i < targets.length; i++) {
      if (results[i].status !== 'fulfilled') {
        console.error('VENDOR_CONVERSION_FAIL', targets[i],
          String(results[i].reason));
        process.exitCode = 1;
      }
    }
  } finally {
    await fs.rm(root, { recursive: true, force: true });
  }
})().catch((err) => {
  console.error(err.stack);
  process.exitCode = 1;
});
