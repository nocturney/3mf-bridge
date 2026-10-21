// Assemble self-contained unpacked browser extensions from shared source files.
// Leaves signed/store distributions to the platform-specific release process.
'use strict';
const fs = require('node:fs');
const path = require('node:path');

const root = path.resolve(__dirname, '..');
const shared = path.join(__dirname, 'shared');
const outRoot = path.join(root, 'dist', 'extensions');
const sharedFiles = [
  'content.js', 'popup.html', 'popup.css', 'popup.js',
  'icon-16.png', 'icon-32.png', 'icon-48.png', 'icon-128.png'
];

function assertSafeResource(name) {
  if (typeof name !== 'string' || path.basename(name) !== name || name.startsWith('.')) {
    throw new Error('Unsafe extension resource: ' + String(name));
  }
}

function packageBrowser(browser) {
  if (!['chromium', 'firefox'].includes(browser)) throw new Error('Unknown browser');
  const manifestSource = path.join(__dirname, browser, 'manifest.json');
  const manifest = JSON.parse(fs.readFileSync(manifestSource, 'utf8'));
  if (manifest.manifest_version !== 3) throw new Error('Expected Manifest V3');
  const referenced = new Set();
  for (const script of manifest.content_scripts || []) {
    for (const file of script.js || []) referenced.add(file);
    for (const file of script.css || []) referenced.add(file);
  }
  for (const file of Object.values(manifest.icons || {})) referenced.add(file);
  for (const file of Object.values(manifest.action?.default_icon || {})) referenced.add(file);
  if (manifest.action?.default_popup) referenced.add(manifest.action.default_popup);
  for (const name of referenced) {
    assertSafeResource(name);
    if (!sharedFiles.includes(name)) throw new Error('Unrecognized manifest resource: ' + name);
  }
  const target = path.join(outRoot, browser);
  const embedded = path.join(root,'native','frontend','dist','extension-packages',browser);
  for (const directory of [target, embedded]) {
    fs.rmSync(directory, {recursive:true,force:true});
    fs.mkdirSync(directory, {recursive:true});
    fs.copyFileSync(manifestSource, path.join(directory, 'manifest.json'));
    for (const name of sharedFiles) {
      const source = path.join(shared, name);
      if (!fs.existsSync(source)) throw new Error('Missing shared resource: ' + name);
      fs.copyFileSync(source, path.join(directory, name));
    }
    for (const name of referenced) {
      if (!fs.existsSync(path.join(directory, name))) throw new Error('Missing packaged resource: ' + name);
    }
  }
  return {browser, path:target, embedded, resources:sharedFiles.length};
}

if (require.main === module) {
  for (const browser of ['chromium', 'firefox']) {
    const result=packageBrowser(browser);
    console.log(result.browser + ': ' + result.path + ' (' + result.resources + ' assets)');
  }
}
module.exports={packageBrowser};
