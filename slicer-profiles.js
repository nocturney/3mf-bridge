'use strict';
const path = require('node:path');
const fs = require('node:fs/promises');

// Exact installed hardware variants; 0.2 is *not* interchangeable with
// Flashforge Creator 5's 0.25 nozzle.
const PROFILES = Object.freeze({
  u1: {
    vendor: 'Snapmaker',
    slicer: 'Snapmaker Orca',
    model: 'Snapmaker U1',
    name: 'Snapmaker U1 (0.4 nozzle)',
    process: '0.20mm Standard @Snapmaker U1 (0.4 nozzle)',
    rootParts: ['Snapmaker_Orca', 'resources', 'profiles'],
    nozzles: ['0.2', '0.4', '0.6', '0.8'],
    processes: {
      '0.2': '0.12mm Standard @Snapmaker U1 (0.2 nozzle)',
      '0.4': '0.20mm Standard @Snapmaker U1 (0.4 nozzle)',
      '0.6': '0.30mm Standard @Snapmaker U1 (0.6 nozzle)',
      '0.8': '0.40mm Standard @Snapmaker U1 (0.8 nozzle)'
    }
  },
  c5: {
    vendor: 'Flashforge',
    slicer: 'Flash Studio Desktop',
    model: 'Flashforge Creator 5',
    name: 'Flashforge Creator 5 0.4 nozzle',
    process: '0.20mm Standard @FF C5',
    rootParts: ['Flashforge', 'Flash Studio Desktop', 'resources', 'profiles'],
    nozzles: ['0.25', '0.4', '0.6', '0.8'],
    processes: {
      '0.25': '0.12mm Standard @FF C5 0.25 nozzle',
      '0.4': '0.20mm Standard @FF C5',
      '0.6': '0.30mm Standard @FF C5 0.6 nozzle',
      '0.8': '0.32mm Standard @FF C5 0.8 nozzle'
    }
  },
  cc2: {
    vendor: 'Elegoo',
    slicer: 'ElegooSlicer',
    model: 'Elegoo Centauri Carbon 2',
    name: 'Elegoo Centauri Carbon 2 0.4 nozzle',
    process: '0.20mm Standard @Elegoo CC2 0.4 nozzle',
    rootParts: ['ElegooSlicer', 'resources', 'profiles'],
    nozzles: ['0.2', '0.4', '0.6', '0.8'],
    processes: {
      '0.2': '0.10mm Standard @Elegoo CC2 0.2 nozzle',
      '0.4': '0.20mm Standard @Elegoo CC2 0.4 nozzle',
      '0.6': '0.30mm Standard @Elegoo CC2 0.6 nozzle',
      '0.8': '0.40mm Standard @Elegoo CC2 0.8 nozzle'
    }
  },
  qidi: {
    vendor: 'X 5 Series',
    slicer: 'OrcaSlicer',
    model: 'X-Plus 5',
    name: 'X-Plus 5 0.4 nozzle',
    process: '0.20mm Standard @X-Plus 5',
    rootParts: ['OrcaSlicer', 'resources', 'profiles'],
    nozzles: ['0.2', '0.4', '0.6', '0.8'],
    processes: {
      '0.2': '0.10mm Standard @X-Plus 5 0.2 nozzle',
      '0.4': '0.20mm Standard @X-Plus 5',
      '0.6': '0.30mm Standard @X-Plus 5 0.6 nozzle',
      '0.8': '0.40mm Standard @X-Plus 5 0.8 nozzle'
    }
  }
});

function resolveVariant(target, sourceNozzle = '0.4') {
  // Owner-approved non-interruptive substitution: Creator 5 uses 0.25mm,
  // whereas Bambu small-nozzle profiles normally use 0.2mm.
  const nozzle = target === 'c5' && sourceNozzle === '0.2' ? '0.25' : sourceNozzle;
  const spec = PROFILES[target];
  if (!spec) throw new Error('Unknown printer target: ' + target);
  if (!spec.nozzles.includes(nozzle)) {
    throw new Error(spec.model + ' has no exact ' + nozzle + ' mm nozzle profile; supported: ' +
      spec.nozzles.join(', ') + ' mm. No automatic nozzle substitution.');
  }
  let name;
  let materials;
  if (target === 'u1') {
    name = 'Snapmaker U1 (' + nozzle + ' nozzle)';
    materials = {
      PLA: 'Generic PLA @U1 ' + nozzle + ' nozzle',
      PETG: 'Generic PETG @U1 ' + nozzle + ' nozzle',
      ABS: 'Generic ABS @U1 ' + nozzle + ' nozzle',
      ASA: 'Generic ASA @U1 ' + nozzle + ' nozzle'
    };
    if (nozzle === '0.4') {
      // Retain the proven U1 0.4 builtin instead of using this native path.
      materials = { PLA: 'Snapmaker PLA Basic @U1' };
    }
    if (nozzle === '0.6' || nozzle === '0.8') {
      materials.TPU = 'Generic TPU @U1 ' + nozzle + ' nozzle';
    }
  } else if (target === 'c5') {
    name = 'Flashforge Creator 5 ' + nozzle + ' nozzle';
    const suffix = nozzle === '0.25' ? ' 0.25 nozzle' : '';
    materials = {
      PLA: 'Flashforge PLA Basic @FF C5' + suffix,
      PETG: 'Flashforge PETG Basic @FF C5' + suffix,
      ABS: 'Flashforge ABS Basic @FF C5' + suffix,
      ASA: 'Flashforge ASA Basic @FF C5' + suffix
    };
  } else if (target === 'cc2') {
    name = 'Elegoo Centauri Carbon 2 ' + nozzle + ' nozzle';
    materials = {
      PLA: 'Elegoo PLA @ECC2',
      PETG: 'Elegoo PETG @ECC2',
      ABS: 'Elegoo ABS @ECC2',
      ASA: 'Elegoo ASA @ECC2',
      TPU: 'Elegoo TPU 95A @ECC2'
    };
  } else {
    name = 'X-Plus 5 ' + nozzle + ' nozzle';
    materials = {
      PLA: 'Generic PLA @Qidi X-Plus 5 ' + nozzle + ' nozzle',
      PETG: 'Generic PETG @Qidi X-Plus 5 ' + nozzle + ' nozzle',
      ABS: 'Generic ABS @Qidi X-Plus 5 ' + nozzle + ' nozzle'
    };
    if (nozzle !== '0.2') {
      materials.TPU = 'Generic TPU 95A @Qidi X-Plus 5 ' + nozzle + ' nozzle';
    }
  }
  return { name, process: spec.processes[nozzle], materials, nozzle, sourceNozzle };
}

const OMIT = new Set([
  'type', 'name', 'from', 'inherits', 'setting_id', 'instantiation',
  'renamed_from', 'compatible_printers', 'compatible_printers_condition',
  'compatible_prints', 'compatible_prints_condition',
  'upward_compatible_machine'
]);

function defaultRoot(target, env = process.env) {
  const targetKey = String(target).toUpperCase();
  if (env['BRIDGE_SLICER_PROFILE_' + targetKey]) {
    return env['BRIDGE_SLICER_PROFILE_' + targetKey];
  }
  const home = env.ProgramFiles || env.PROGRAMFILES || 'C:\\Program Files';
  return path.join(home, ...PROFILES[target].rootParts);
}

async function buildIndex(root, spec) {
  const files = [];
  const dirs = [
    path.join(root, spec.vendor),
    ...(spec.vendor === 'Flashforge' ? [path.join(root, 'OrcaFilamentLibrary')] : [])
  ];
  async function walk(dir) {
    for (const child of await fs.readdir(dir, { withFileTypes: true })) {
      const file = path.join(dir, child.name);
      if (child.isDirectory()) await walk(file);
      else if (child.isFile() && child.name.endsWith('.json')) files.push(file);
    }
  }
  for (const dir of dirs) await walk(dir);
  const index = new Map();
  for (const file of files) {
    const data = JSON.parse(await fs.readFile(file, 'utf8'));
    if (!data || typeof data !== 'object' || Array.isArray(data)) continue;
    const type = path.basename(path.dirname(file)) === 'machine'
      ? 'machine'
      : file.includes(path.sep + 'process' + path.sep) ? 'process'
      : file.includes(path.sep + 'filament' + path.sep) ? 'filament' : null;
    const actualType = type || (file.includes(path.sep + 'machine' + path.sep) ? 'machine' : null);
    if (!actualType) continue;
    const stem = path.basename(file, '.json');
    for (const name of new Set([stem, data.name].filter(Boolean))) {
      const key = actualType + ':' + name;
      if (!index.has(key)) index.set(key, []);
      index.get(key).push({ file, data });
    }
  }
  return index;
}

function flattenPreset(index, type, presetName, stack = new Set()) {
  const key = type + ':' + presetName;
  const matches = index.get(key);
  if (!matches || matches.length === 0) {
    throw new Error('Missing installed ' + type + ' preset: ' + presetName);
  }
  const current = matches.find((candidate) =>
    path.basename(candidate.file, '.json') === presetName
  ) || matches[0];
  if (stack.has(current.file)) throw new Error('Profile inheritance cycle: ' + presetName);
  stack.add(current.file);
  let result = {};
  if (current.data.inherits) {
    result = flattenPreset(index, type, current.data.inherits, stack);
  }
  for (const [key, value] of Object.entries(current.data)) {
    if (!OMIT.has(key)) result[key] = value;
  }
  stack.delete(current.file);
  return result;
}

function checkNozzleValues(values, nozzle, target) {
  if (!Array.isArray(values) || values.length === 0 ||
    values.some((v) => !Number.isFinite(Number(v)) || Math.abs(Number(v) - Number(nozzle)) > 1e-6)) {
    throw new Error('Installed ' + target + ' machine nozzle diameter does not match ' + nozzle + ' mm');
  }
}

async function loadTargetBundle(target, options = {}) {
  const spec = PROFILES[target];
  const sourceNozzle = String(options.nozzle || '0.4');
  const variant = resolveVariant(target, sourceNozzle);
  const nozzle = variant.nozzle;
  const root = options.root || defaultRoot(target, options.env);
  const index = await buildIndex(root, spec);
  const machine = flattenPreset(index, 'machine', variant.name);
  const process = flattenPreset(index, 'process', variant.process);
  if (machine.printer_model !== spec.model) {
    throw new Error('Installed machine profile identity mismatch: ' +
      String(machine.printer_model) + ', expected ' + spec.model);
  }
  checkNozzleValues(machine.nozzle_diameter, nozzle, target);
  const materials = {};
  const names = {};
  for (const [type, presetName] of Object.entries(variant.materials)) {
    const settings = flattenPreset(index, 'filament', presetName);
    materials[type] = settings;
    names[type] = presetName;
  }
  const initial = { ...machine, ...process, ...materials.PLA };
  // Only the machine owns nozzle geometry, machine limits and firmware.
  initial.nozzle_diameter = machine.nozzle_diameter;
  for (const key of Object.keys(machine)) {
    if (key.startsWith('machine_') || key === 'change_filament_gcode' ||
      key === 'layer_change_gcode' || key === 'before_layer_change_gcode') {
      initial[key] = machine[key];
    }
  }
  initial.printer_model = spec.model;
  initial.printer_settings_id = variant.name;
  initial.print_settings_id = variant.process;
  initial.filament_settings_id = [names.PLA];
  initial.filament_type = ['PLA'];
  initial.filament_colour = ['#FFFFFF'];
  return {
    target,
    slicer: spec.slicer,
    root,
    nozzle,
    profile: {
      name: target,
      display_name: spec.model,
      printer_model_id: spec.model,
      filament_slots: 4,
      filament_profiles: names,
      default_filament_profile: names.PLA,
      project_settings: initial
    },
    material_settings: materials
  };
}

module.exports = { PROFILES, defaultRoot, resolveVariant, buildIndex, flattenPreset, loadTargetBundle };
