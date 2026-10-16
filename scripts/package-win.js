// Windows portable packaging; the native engine lives outside app.asar.
const path = require('node:path');
const fs = require('node:fs/promises');
const { packager } = require('@electron/packager');

async function main() {
  const root = path.resolve(__dirname, '..');
  const exe = path.join(root, 'bin', 'bridge-core.exe');
  await fs.access(exe);
  const paths = await packager({
    dir: root,
    out: path.join(root, 'dist'),
    name: '3MF Bridge',
    platform: 'win32',
    arch: 'x64',
    asar: true,
    overwrite: true,
    prune: true,
    extraResource: [path.join(root, 'bin')],
    ignore: [
      /^\/\.git($|\/)/,
      /^\/\.github($|\/)/,
      /^\/third_party($|\/)/,
      /^\/engine($|\/)/,
      /^\/tests($|\/)/,
      /^\/scripts($|\/)/,
      /^\/bin($|\/)/
    ]
  });
  for (const p of paths) console.log('PACKAGED ' + p);
}
main().catch((e) => {
  console.error(e.message);
  process.exitCode = 1;
});
