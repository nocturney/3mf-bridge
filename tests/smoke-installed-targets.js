'use strict';

// Run on a Windows host with Flash Studio, ElegooSlicer and OrcaSlicer installed.
const path = require('node:path');
const fs = require('node:fs/promises');
const os = require('node:os');
const assert = require('node:assert/strict');
const { execFile } = require('node:child_process');
const { promisify } = require('node:util');
const run = promisify(execFile);
const { loadTargetBundle } = require('../slicer-profiles');

async function oneTarget(target, root, fixtureName) {
  const dir = path.join(root, target + '-' + fixtureName.replace('.3mf', ''));
  await fs.mkdir(dir);
  const source = path.join(dir, 'bambu.3mf');
  const output = path.join(dir, target + '-bambu.3mf');
  const bundlePath = path.join(dir, 'native-vendor-profile.json');
  const bundle = await loadTargetBundle(target);
  await fs.writeFile(bundlePath, JSON.stringify(bundle), { mode: 0o600 });
  await fs.copyFile(path.join(__dirname, 'fixtures', fixtureName), source);

  const exe = path.join(__dirname, '..', 'bin', 'bridge-core.exe');
  let stdout;
  try {
    ({ stdout } = await run(exe, [
      '--source', source, '--output', output,
      '--target', target, '--profile', bundlePath
    ], { windowsHide: true, timeout: 120000, maxBuffer: 256*1024 }));
  } catch (error) {
    throw new Error(error.stdout || error.stderr || error.message);
  }
  const result = JSON.parse(stdout);
  assert.equal(result.ok, true, JSON.stringify(result));
  assert.equal(result.target, target);
  assert.equal(result.geometry_triangles, 4);
  assert.equal(result.geometry_vertices, 4);
  assert.ok(result.removed_slice_cache_entries >= 3);
  const [oldBytes, newBytes] = await Promise.all([fs.readFile(source), fs.readFile(output)]);
  assert.ok(!oldBytes.equals(newBytes), 'source should be transformed, not copied');
  console.log('VENDOR_CONVERSION_PASS', target, fixtureName, JSON.stringify({
    machine: bundle.profile.printer_model_id,
    modelNozzle: bundle.nozzle,
    profileRoot: bundle.root,
    nativeSettings: Object.keys(bundle.profile.project_settings).length,
    materials: Object.keys(bundle.material_settings),
    triangles: result.geometry_triangles,
    removedCache: result.removed_slice_cache_entries,
    bytes: newBytes.byteLength,
    sha256: require('node:crypto').createHash('sha256').update(newBytes).digest('hex').slice(0,16)
  }));
}

(async () => {
  const root = await fs.mkdtemp(path.join(os.tmpdir(), '3mf-bridge-vendor-'));
  try {
    const tasks = ['c5', 'cc2', 'qidi'].flatMap((target) => [
      { target, fixtureName: 'bambu-one-color.3mf' },
      { target, fixtureName: 'bambu-mixed-three.3mf' }
    ]);
    const results = await Promise.allSettled(tasks.map(({ target, fixtureName }) =>
      oneTarget(target, root, fixtureName)));
    for (let i = 0; i < tasks.length; i++) {
      if (results[i].status !== 'fulfilled') {
        console.error('VENDOR_CONVERSION_FAIL', tasks[i], String(results[i].reason));
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
