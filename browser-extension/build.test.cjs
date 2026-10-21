'use strict';
const test=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const path=require('node:path');
const {packageBrowser}=require('./build.cjs');
const repo=path.resolve(__dirname,'..');
const icons=['16','32','48','128'];

for(const browser of ['chromium','firefox']) {
  test(browser+' extension staging has all self-contained resources',()=>{
    const result=packageBrowser(browser);
    const target=result.path;
    const manifest=JSON.parse(fs.readFileSync(path.join(target,'manifest.json'),'utf8'));
    assert.ok(result.embedded.includes(path.join('native','frontend','dist','extension-packages')));
    assert.deepEqual(fs.readdirSync(result.embedded).sort(),fs.readdirSync(target).sort());
    assert.equal(manifest.manifest_version,3);
    assert.deepEqual(manifest.permissions,['storage']);
    assert.equal((manifest.content_scripts||[]).length,1);
    // Require an exact host allowlist, not an unsafe URL substring match.
    assert.deepEqual([...manifest.content_scripts[0].matches].sort(),[
      'https://makerworld.com/*',
      'https://*.makerworld.com/*'
    ].sort());
    assert.equal(manifest.action.default_popup,'popup.html');
    // A staged browser package must carry the exact reviewed shared content script.
    assert.deepEqual(fs.readFileSync(path.join(target,'content.js')),
      fs.readFileSync(path.join(__dirname,'shared','content.js')));
    assert.deepEqual(fs.readFileSync(path.join(result.embedded,'content.js')),
      fs.readFileSync(path.join(__dirname,'shared','content.js')));
    for(const name of ['content.js','popup.html','popup.css','popup.js',...icons.map(n=>'icon-'+n+'.png')]){
      assert.ok(fs.statSync(path.join(target,name)).size>0,name);
    }
  });
}
test('browser opt-in remains disabled by default',()=>{
  const popup=fs.readFileSync(path.join(__dirname,'shared','popup.js'),'utf8');
  const content=fs.readFileSync(path.join(__dirname,'shared','content.js'),'utf8');
  assert.match(popup,/makerworldMode:\s*'off'/);
  assert.match(content,/let mode = 'off'/);
});
test('approved native artwork has no missing links',()=>{
  const ui=path.join(repo,'native','frontend','dist');
  const html=fs.readFileSync(path.join(ui,'index.html'),'utf8');
  const css=fs.readFileSync(path.join(ui,'app.css'),'utf8');
  assert.match(html,/src="\.\/brand-icon\.png"/);
  assert.match(css,/url\('\.\/slicebridge-banner\.png'\)/);
  assert.ok(fs.statSync(path.join(ui,'brand-icon.png')).size>0);
  assert.ok(fs.statSync(path.join(ui,'slicebridge-banner.png')).size>0);
});
