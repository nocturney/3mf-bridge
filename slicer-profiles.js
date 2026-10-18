'use strict';

const path = require('node:path');
const fs = require('node:fs/promises');

const PROFILES = Object.freeze({
  c5: {
    vendor: 'Flashforge',
    slicer: 'Flash Studio Desktop',
    model: 'Flashforge Creator 5',
    name: 'Flashforge Creator 5 0.4 nozzle',
    process: '0.20mm Standard @FF C5',
    rootParts: ['Flashforge', 'Flash Studio Desktop', 'resources', 'profiles'],
    materials: {
      PLA: 'Flashforge PLA Basic @FF C5',
      PETG: 'Flashforge PETG Basic @FF C5',
      ABS: 'Flashforge ABS Basic @FF C5',
      ASA: 'Flashforge ASA Basic @FF C5'
    }
  },
  cc2: {
    vendor: 'Elegoo',
    slicer: 'ElegooSlicer',
    model: 'Elegoo Centauri Carbon 2',
    name: 'Elegoo Centauri Carbon 2 0.4 nozzle',
    process: '0.20mm Standard @Elegoo CC2 0.4 nozzle',
    rootParts: ['ElegooSlicer', 'resources', 'profiles'],
    materials: {
      PLA: 'Elegoo PLA @ECC2',
      PETG: 'Elegoo PETG @ECC2',
      ABS: 'Elegoo ABS @ECC2',
      ASA: 'Elegoo ASA @ECC2',
      TPU: 'Elegoo TPU 95A @ECC2'
    }
  },
  qidi: {
    vendor: 'X 5 Series',
    slicer: 'OrcaSlicer',
    model: 'X-Plus 5',
    name: 'X-Plus 5 0.4 nozzle',
    process: '0.20mm Standard @X-Plus 5',
    rootParts: ['OrcaSlicer', 'resources', 'profiles'],
    materials: {
      PLA: 'Generic PLA @Qidi X-Plus 5 0.4 nozzle',
      PETG: 'Generic PETG @Qidi X-Plus 5 0.4 nozzle',
      ABS: 'Generic ABS @Qidi X-Plus 5 0.4 nozzle',
      TPU: 'Generic TPU 95A @Qidi X-Plus 5 0.4 nozzle'
    }
  }
});

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
  // Prioritize the machine-specific filename over a shared inherited alias.
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

async function loadTargetBundle(target, options = {}) {
  const spec = PROFILES[target];
  if (!spec) throw new Error('Unknown target: ' + target);
  const root = options.root || defaultRoot(target, options.env);
  const index = await buildIndex(root, spec);
  const machine = flattenPreset(index, 'machine', spec.name);
  const process = flattenPreset(index, 'process', spec.process);
  if (machine.printer_model !== spec.model) {
    throw new Error('Installed machine profile identity mismatch: ' +
      String(machine.printer_model) + ', expected ' + spec.model);
  }
  const materials = {};
  const names = {};
  for (const [type, presetName] of Object.entries(spec.materials)) {
    const settings = flattenPreset(index, 'filament', presetName);
    materials[type] = settings;
    names[type] = presetName;
  }
  // The vendor's own machine/process presets own all firmware commands and
  // motion limits. The target filament preset owns heating, flow and cooling.
  const initial = { ...machine, ...process, ...materials.PLA };
  initial.printer_model = spec.model;
  initial.printer_settings_id = spec.name;
  initial.print_settings_id = spec.process;
  initial.filament_settings_id = [names.PLA];
  initial.filament_type = ['PLA'];
  initial.filament_colour = ['#FFFFFF'];

  return {
    target,
    slicer: spec.slicer,
    root,
    nozzle: '0.4',
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

module.exports = { PROFILES, defaultRoot, buildIndex, flattenPreset, loadTargetBundle };
