'use strict';
const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs/promises');
const os = require('node:os');
const path = require('node:path');
const { PROFILES, resolveVariant, loadTargetBundle } = require('../slicer-profiles');

async function writeJson(root, type, name, object) {
  const destination = path.join(root, type, name + '.json');
  await fs.mkdir(path.dirname(destination), { recursive: true });
  await fs.writeFile(destination, JSON.stringify(object));
}

async function fakeProfileRoot(target) {
  const root = await fs.mkdtemp(path.join(os.tmpdir(), '3mf-slicer-profile-'));
  const spec = PROFILES[target];
  const vendor = path.join(root, spec.vendor);
  const library = path.join(root, 'OrcaFilamentLibrary');
  await fs.mkdir(library, { recursive: true });

  await writeJson(vendor, 'machine', 'machine_common', {
    name: 'machine_common', type: 'machine', machine_start_gcode: 'G28',
    machine_end_gcode: 'M104 S0', nozzle_diameter: ['0.4'],
    printer_settings_id: 'base system'
  });
  await writeJson(vendor, 'machine', spec.name, {
    type: 'machine', name: spec.name, inherits: 'machine_common',
    printer_model: spec.model, printer_settings_id: spec.name
  });
  await writeJson(vendor, 'process', 'process_common', {
    type: 'process', name: 'process_common', layer_height: '0.20',
    wall_loops: '2', support_threshold_angle: '45'
  });
  await writeJson(vendor, 'process', spec.process, {
    type: 'process', name: spec.process,
    inherits: 'process_common', layer_height: '0.18'
  });
  await writeJson(vendor, 'filament', 'filament_base', {
    type: 'filament', name: 'filament_base',
    nozzle_temperature: ['210'], filament_density: ['1.24']
  });
  for (const [type, name] of Object.entries(resolveVariant(target, '0.4').materials)) {
    await writeJson(vendor, 'filament', name, {
      type: 'filament', name, inherits: 'filament_base',
      filament_type: [type], filament_settings_id: [name]
    });
  }
  return root;
}

test('three vendor-derived profiles resolve concurrently and carry machine/process/filament inheritance', async () => {
  const targets = ['c5', 'cc2', 'qidi'];
  const roots = await Promise.all(targets.map(fakeProfileRoot));
  try {
    const data = await Promise.all(targets.map((target, i) =>
      loadTargetBundle(target, { root: roots[i] })));
    for (let i = 0; i < targets.length; i++) {
      const { profile, material_settings, nozzle } = data[i];
      assert.equal(profile.name, targets[i]);
      assert.equal(profile.printer_model_id, PROFILES[targets[i]].model);
      assert.equal(profile.project_settings.machine_start_gcode, 'G28');
      assert.equal(profile.project_settings.layer_height, '0.18');
      assert.equal(profile.project_settings.printer_settings_id, PROFILES[targets[i]].name);
      assert.equal(profile.filament_slots, 4);
      assert.equal(nozzle, '0.4');
      assert.ok(Object.keys(material_settings).length >= 4);
      assert.deepEqual(material_settings.PETG.nozzle_temperature, ['210']);
    }
  } finally {
    await Promise.all(roots.map((root) => fs.rm(root, { recursive: true, force: true })));
  }
});

test('missing, broken, cyclic or spoofed vendor profiles fail closed', async () => {
  await assert.rejects(
    loadTargetBundle('qidi', { root: path.join(os.tmpdir(), 'missing-vendor-profiles-xxx') }));
  const root = await fakeProfileRoot('c5');
  try {
    const machine = path.join(root, 'Flashforge', 'machine', PROFILES.c5.name + '.json');
    let cfg = JSON.parse(await fs.readFile(machine, 'utf8'));
    cfg.printer_model = 'DIFFERENT MODEL';
    await fs.writeFile(machine, JSON.stringify(cfg));
    await assert.rejects(loadTargetBundle('c5', { root }), /identity mismatch/);

    cfg.printer_model = PROFILES.c5.model;
    cfg.inherits = PROFILES.c5.name;
    await fs.writeFile(machine, JSON.stringify(cfg));
    await assert.rejects(loadTargetBundle('c5', { root }), /inheritance cycle/);
  } finally {
    await fs.rm(root, { recursive: true, force: true });
  }
});

test('nozzle selection uses exact installed variants, with owner-approved C5 0.2 -> 0.25', () => {
  for (const target of ['u1', 'c5', 'cc2', 'qidi']) {
    for (const nozzle of ['0.2', '0.4', '0.6', '0.8']) {
      const variant = resolveVariant(target, nozzle);
      const expected = target === 'c5' && nozzle === '0.2' ? '0.25' : nozzle;
      assert.equal(variant.nozzle, expected);
      assert.ok(variant.name.includes(expected));
      assert.ok(variant.process);
      assert.ok(variant.materials.PLA);
      if (target !== 'u1' || nozzle !== '0.4') assert.ok(variant.materials.PETG);
    }
  }
  assert.throws(() => resolveVariant('qidi', '0.25'), /no exact/);
  assert.throws(() => resolveVariant('u1', '0.25'), /no exact/);
  assert.equal(resolveVariant('c5', '0.25').nozzle, '0.25');
});
