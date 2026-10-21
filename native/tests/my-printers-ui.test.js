'use strict';
const {test}=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const path=require('node:path');
const picker=require('../frontend/dist/printer-picker.js');
const catalog=JSON.parse(fs.readFileSync(path.resolve(__dirname,'../frontend/dist/printer-catalog.json'),'utf8'));

test('snapshot is complete yet only 4 target adapters, no media/profiles embedded',()=>{
  assert.equal(catalog.modelCount,603);
  assert.equal(catalog.models.length,603);
  assert.equal(catalog.legacyAdapterCount,4);
  assert.equal(catalog.thirdPartyProfilePayloadsIncluded,false);
  assert.ok(catalog.models.some(x=>x.manufacturer==='Bambu Lab'));
  assert.ok(catalog.models.every(x=>x.imagePath==null && x.logoPath==null));
  assert.equal(catalog.models.filter(x=>x.legacyTarget).length,4);
  assert.equal(picker.search(catalog.models,'',[],100).total,
               603-catalog.models.filter(picker.isHidden).length);
});

test('Bambu printer destination stays hidden in normal list and even when searched',()=>{
  for(const word of ['Bambu','P1S','X1C','Bambu Lab']){
    assert.ok(picker.search(catalog.models,word).models.every(x=>x.manufacturer!=='Bambu Lab'));
  }
  assert.equal(picker.search(catalog.models,'Bambu Lab').total,0);
  assert.ok(catalog.models.some(x=>x.manufacturer==='Bambu Lab'));
});

test('search works for brand, model and legacy aliases, without inventing unverified dates',()=>{
  const u1=picker.search(catalog.models,'snapmaker u1').models;
  assert.ok(u1.some(x=>x.id==='snapmaker-u1'));
  const c5=picker.search(catalog.models,'Creator 5').models;
  assert.ok(c5.some(x=>x.id==='flashforge-creator-5'));
  const cc2=picker.search(catalog.models,'ELEGOO CC2').models;
  assert.ok(cc2.some(x=>x.id==='elegoo-centauri-carbon-2'));
  // A few officially evidenced launches are dated; unknown models stay unknown.
  assert.ok(catalog.models.some(x=>x.releaseDate!==null));
  for(const id of ['snapmaker-u1','flashforge-creator-5','elegoo-centauri-carbon-2']){
    assert.equal(catalog.models.find(x=>x.id===id).releaseDate,null,id);
  }
  assert.equal(picker.search(catalog.models,'nonexistent printer 999999').total,0);
});

test('only supported destinations are offered: catalogue-only and unavailable adapters have no state',()=>{
  const script=fs.readFileSync(path.resolve(__dirname,'../frontend/dist/my-printers.js'),'utf8');
  const u1=catalog.models.find(x=>x.id==='snapmaker-u1');
  const c5=catalog.models.find(x=>x.id==='flashforge-creator-5');
  const other=catalog.models.find(x=>!x.legacyTarget && !picker.isHidden(x));
  const availability=[{target:'c5',available:false,error:'Flash Studio not installed'},{target:'u1',available:true}];
  const items=picker.supportedModels(catalog.models,availability,[],false);
  assert.deepEqual(items.map(x=>x.id),[u1.id]);
  assert.equal(items.some(x=>x.id===other.id||x.id===c5.id),false);
  assert.deepEqual(picker.legacyTargetsFor(items,[other.id,c5.id,u1.id]),['u1']);
  // The retired status vocabulary never reappears in the UI copy or the helper.
  assert.doesNotMatch(script,/catalog_only|missing_local_profile|selectedStatuses|statuses\(/);
  assert.doesNotMatch(fs.readFileSync(path.resolve(__dirname,'../frontend/dist/printer-picker.js'),'utf8'),/statuses|selectedAdapterTargets/);
});

test('main list has no catalogue-only section and built-in rows start hidden until selected',()=>{
  const doc=fs.readFileSync(path.resolve(__dirname,'../frontend/dist/index.html'),'utf8');
  const app=fs.readFileSync(path.resolve(__dirname,'../frontend/dist/app.js'),'utf8');
  const style=fs.readFileSync(path.resolve(__dirname,'../frontend/dist/app.css'),'utf8');
  assert.doesNotMatch(doc,/catalogOnlyTargets|catalog-only/);
  assert.doesNotMatch(app,/catalogOnly|catalog-only|selectedStatuses|classList\.add\('unavailable'\)/);
  for(const target of ['u1','c5','cc2','qidi']){
    assert.match(doc,new RegExp('<label class="target-option" data-target-row="'+target+'" hidden>'));
  }
  assert.match(app,/function builtInVisible\(target\)/);
  assert.match(app,/data\?\.available===true&&Boolean\(window\.SliceBridgeMyPrinters\?\.isSelectedTarget\(target\)\)/);
  assert.match(style,/\.target-option\[hidden\]\{display:none!important\}/);
});

test('the main list and the picker share one destination set through setSupportedTargets',()=>{
  const app=fs.readFileSync(path.resolve(__dirname,'../frontend/dist/app.js'),'utf8');
  assert.match(app,/function publishSupportedTargets\(\)\{\s*window\.SliceBridgeMyPrinters\?\.setSupportedTargets\(\{available,installedTargets,prusaAvailable\}\);/);
  assert.match(app,/available=Array\.isArray\(reply\)\?reply:\[\];/);
  assert.match(app,/installedTargets=Array\.isArray\(reply\?\.targets\)\?reply\.targets:\[\];/);
  // Handed over once after all probes, and before My Printers reads its inputs in init().
  assert.match(app,/await refreshPrusaEngine\(\);[\s\S]*?publishSupportedTargets\(\);\s*await window\.SliceBridgeMyPrinters\?\.init\(\);/);
  assert.match(app,/prusaAvailable=Boolean\(\(await api\(\)\?\.PrusaEngineStatus\?\.\(\)\)\?\.available\)/);
  assert.match(app,/window\.addEventListener\('slicebridge:my-printers-changed',\(\)=>renderTargets\(\)\)/);
  assert.match(app,/row\.hidden=!isSelectedDestination\(target\.id\)/);
  assert.match(app,/setRowVisible\(entry\.row,entry\.checkbox,isSelectedDestination\(entry\.id\)\)/);
});

test('saved selections are never shown as unsupported and are not sent back when stale',()=>{
  const script=fs.readFileSync(path.resolve(__dirname,'../frontend/dist/my-printers.js'),'utf8');
  assert.match(script,/const requested=\[\.\.\.draft\]\.filter\(id=>supported\.some\(item=>item\.id===id\)\);/);
  assert.match(script,/function selectedTargetIds\(\)\{\s*return loaded\?picker\.selectedTargetIds\(supported,selected\.selectedIds\):\[\];\s*\}/);
  assert.match(script,/if\(!loaded\|\|!draft\|\|saving\|\|!supported\.length\)return;/);
});

test('broad list capped at 80 and selected first; text cannot execute JS',()=>{
  const other=catalog.models.find(x=>!x.legacyTarget && !picker.isHidden(x));
  const view=picker.search(catalog.models,'',[other.id]);
  assert.equal(view.models.length,80);
  assert.equal(view.models[0].id,other.id);
  assert.equal(picker.title({manufacturer:'X<script>',model:'<img src=x>'}),
               'X<script> <img src=x>');
  // UI inserts those fields via textContent only, not innerHTML.
  const js=fs.readFileSync(path.resolve(__dirname,'../frontend/dist/my-printers.js'),'utf8');
  assert.doesNotMatch(js,/innerHTML\s*=/);
  assert.match(js,/textContent|createTextNode|node\(/);
});

test('new picker UI has bilingual names, consent-based selection and no network media',()=>{
  const doc=fs.readFileSync(path.resolve(__dirname,'../frontend/dist/index.html'),'utf8');
  const script=fs.readFileSync(path.resolve(__dirname,'../frontend/dist/my-printers.js'),'utf8');
  for(const id of ['myPrintersBtn','myPrintersDialog','printerCatalogSearch',
                   'printerManufacturerFilter','catalogPickerHint',
                   'catalogModelList','saveMyPrintersBtn','cancelMyPrintersBtn']){
    assert.ok(doc.includes('id="'+id+'"'),id);
  }
  assert.match(doc,/src="\.\/printer-picker\.js"/);
  assert.match(doc,/src="\.\/my-printers\.js"/);
  assert.match(script,/SaveMyPrinters/);
  assert.match(script,/GetMyPrinters/);
  assert.match(script,/PrinterCatalog/);
  assert.doesNotMatch(script,/קטלוג בלבד|מתאם המרה קיים|catalog only|adapter exists/i);
  assert.doesNotMatch(doc,/id="myPrintersScope"/);
  assert.match(script,/pill\.append\(name\);selectionList\.append\(pill\)/);
  assert.doesNotMatch(script,/copy\.append\(meta\)/);
  // No network reads at all: artwork comes from installed slicers via the backend only.
  assert.equal((script.match(/fetch\s*\(/g)||[]).length,0);
  assert.doesNotMatch(script,/printer-image-manifest|printer-images|imageManifest/);
  assert.doesNotMatch(script,/XMLHttpRequest|https?:\/\//);
});


test('first-run displays available destinations immediately, without requiring a filter',()=>{
  const doc=fs.readFileSync(path.resolve(__dirname,'../frontend/dist/index.html'),'utf8');
  const script=fs.readFileSync(path.resolve(__dirname,'../frontend/dist/my-printers.js'),'utf8');
  const backend=fs.readFileSync(path.resolve(__dirname,'../printer_catalog.go'),'utf8');
  const style=fs.readFileSync(path.resolve(__dirname,'../frontend/dist/app.css'),'utf8');
  assert.match(backend,/Version: 1, SelectedIDs: \[\]string\{\}/);
  for(const target of ['u1','c5','cc2','qidi']){
    assert.doesNotMatch(doc,new RegExp('<input type="checkbox" value="'+target+'" checked>'));
  }
  assert.match(doc,/id="printerManufacturerFilter"/);
  assert.match(doc,/id="catalogPickerHint"/);
  assert.match(script,/const hasFilter=Boolean\(term\|\|manufacturer\)/);
  assert.match(script,/const showList=hasFilter\|\|supported\.length>0/);
  assert.match(script,/const matches=showList\?picker\.searchGroups/);
  assert.match(script,/list\.hidden=!showList/);
  assert.match(script,/observeArtwork\(graphic,group\)/);
  assert.match(style,/\.catalog-model-list\[hidden\]/);
});

test('exactly one My Printers button and one Settings button, with responsive icons',()=>{
  const doc=fs.readFileSync(path.resolve(__dirname,'../frontend/dist/index.html'),'utf8');
  const js=fs.readFileSync(path.resolve(__dirname,'../frontend/dist/app.js'),'utf8');
  const pickerJS=fs.readFileSync(path.resolve(__dirname,'../frontend/dist/my-printers.js'),'utf8');
  const css=fs.readFileSync(path.resolve(__dirname,'../frontend/dist/app.css'),'utf8');
  assert.equal((doc.match(/id="myPrintersBtn"/g)||[]).length,1);
  assert.equal((doc.match(/id="settingsBtn"/g)||[]).length,1);
  assert.equal((doc.match(/id="manageTargetsBtn"/g)||[]).length,0);
  assert.equal((doc.match(/data-settings-target="/g)||[]).length,0);
  assert.doesNotMatch(js,/manageTargetsBtn|data-settings-target/);
  assert.doesNotMatch(pickerJS,/manageTargetsBtn/);
  assert.match(css,/@media\(max-width:1160px\)/);
  assert.match(css,/\.topbar \.labeled-btn span\{display:none\}/);
  assert.match(doc,/id="emptyTargetNotice"/);
});


test('queue drop area can scroll instead of cropping in constrained window heights',()=>{
  const css=fs.readFileSync(path.resolve(__dirname,'../frontend/dist/app.css'),'utf8');
  assert.match(css,/\/\* 0\.4\.0-beta\.4\.dev\.2[\s\S]*?responsive UI/);
  const refinement=css.slice(css.indexOf('/* 0.4.0-beta.4.dev.2'));
  assert.match(refinement,/\.workspace\{\s*[\s\S]*?overflow-y:auto;/);
  assert.match(refinement,/\.left-column\{\s*[\s\S]*?grid-template-rows:minmax\(205px,1fr\) auto;/);
  assert.match(refinement,/\.queue-panel\{\s*min-height:205px\}/);
  assert.match(refinement,/@media\(max-height:750px\)\{\s*\.left-column\{grid-template-rows:minmax\(205px,auto\) auto\}/);
  assert.match(refinement,/@media\(max-width:740px\) and \(max-height:750px\)/);
});

test('target printers are accessible two-column selectable cards with a narrow one-column fallback',()=>{
  const doc=fs.readFileSync(path.resolve(__dirname,'../frontend/dist/index.html'),'utf8');
  const css=fs.readFileSync(path.resolve(__dirname,'../frontend/dist/app.css'),'utf8');
  const refinement=css.slice(css.indexOf('/* 0.4.0-beta.4.dev.2'));
  assert.match(refinement,/\.target-list\{\s*display:grid;\s*grid-template-columns:repeat\(2,minmax\(0,1fr\)\)/);
  assert.match(refinement,/\.target-option:has\(input:checked\)/);
  assert.match(refinement,/@media\(max-width:480px\)\{\s*\.target-list\{grid-template-columns:1fr\}/);
  // Four built-in printer cards; the hidden PrusaSlicer project card is extra.
  assert.equal((doc.match(/class="target-option" data-target-row="/g)||[]).length,4);
  assert.equal((doc.match(/class="target-printer-icon" aria-hidden="true"/g)||[]).length,
               (doc.match(/class="target-option"/g)||[]).length);
  for(const key of ['u1','c5','cc2','qidi']){
    assert.match(doc,new RegExp('<input type="checkbox" value="'+key+'"'));
    assert.match(doc,new RegExp('data-printer-summary="'+key+'"'));
  }
});

test('header always shows labels for printers, settings and MakerWorld, even below 1160px',()=>{
  const doc=fs.readFileSync(path.resolve(__dirname,'../frontend/dist/index.html'),'utf8');
  const css=fs.readFileSync(path.resolve(__dirname,'../frontend/dist/app.css'),'utf8');
  const refinement=css.slice(css.indexOf('/* 0.4.0-beta.4.dev.2'));
  for(const id of ['myPrintersBtn','settingsBtn','integrationsBtn']){
    const hit=doc.match(new RegExp('<button id="'+id+'"[^>]*>([\\s\\S]*?)</button>'));
    assert.ok(hit,id);
    assert.match(hit[1],/<span[^>]*>[^<]+<\/span>/);
  }
  assert.match(refinement,/\.topbar \.labeled-btn span\{\s*display:inline!important;/);
  assert.match(refinement,/@media\(max-width:740px\)[\s\S]*?\.topbar\{flex-wrap:wrap;/);
  assert.match(refinement,/@media\(max-width:380px\)[\s\S]*?\.topbar \.labeled-btn span\{font-size:9\.5px\}/);
  assert.equal((doc.match(/id="myPrintersBtn"/g)||[]).length,1);
  assert.equal((doc.match(/id="settingsBtn"/g)||[]).length,1);
});


test('target printer cards reuse locally installed PNG covers with safe fallback',()=>{
  const html=fs.readFileSync(path.resolve(__dirname,'../frontend/dist/index.html'),'utf8');
  const js=fs.readFileSync(path.resolve(__dirname,'../frontend/dist/my-printers.js'),'utf8');
  const css=fs.readFileSync(path.resolve(__dirname,'../frontend/dist/app.css'),'utf8');
  const backend=fs.readFileSync(path.resolve(__dirname,'../printer_artwork.go'),'utf8');
  const expected=[['u1','snapmaker-u1'],['c5','flashforge-creator-5'],
                  ['cc2','elegoo-centauri-carbon-2'],['qidi','qidi-x-plus-5']];
  for(const [target,modelID] of expected){
    const model=catalog.models.find(model=>model.id===modelID);
    assert.ok(model);
    assert.equal(model.legacyTarget,target);
    assert.ok(model.localCoverCandidates.length>0);
    assert.match(html,new RegExp('data-target-row="'+target+'"'));
  }
  assert.equal((html.match(/class="target-printer-icon"/g)||[]).length,
               (html.match(/class="target-option"/g)||[]).length);
  assert.match(js,/function loadTargetArtwork\(\)/);
  assert.match(js,/if\(!chosen\.has\(item\.id\)\)continue;/);
  assert.match(js,/document\.querySelector\(selector\)/);
  assert.match(js,/data-target-row="'\+item\.legacyTarget/);
  assert.match(js,/tile\.dataset\.artKey=group\.key/);
  assert.match(js,/else queueArtwork\(group\)/);
  assert.match(js,/selected=reply;\s*loadTargetArtwork\(\)/);
  assert.match(js,/loaded=true;\s*loadTargetArtwork\(\)/);
  assert.match(js,/result\?\.provenance==='local_installed_slicer'/);
  assert.equal((js.match(/fetch\s*\(/g)||[]).length,0);
  assert.doesNotMatch(js,/XMLHttpRequest|https?:\/\//);
  assert.match(css,/\.target-printer-icon \.printer-cover-photo\{/);
  assert.match(css,/object-fit:contain/);
  assert.match(backend,/func \(a \*App\) PrinterArtwork\(modelID string\)/);
  assert.match(backend,/func \(a \*App\) PrinterArtworkForInstalledTarget\(targetID string\)/);
  assert.match(js,/PrinterArtworkForInstalledTarget/);
});

test('bundled photo and manifest artifacts are gone from the app',()=>{
  const dist=path.resolve(__dirname,'../frontend/dist');
  assert.equal(fs.existsSync(path.join(dist,'printer-image-manifest.json')),false);
  assert.equal(fs.existsSync(path.join(dist,'printer-images')),false);
});

test('brand banner keeps just the short first headline',()=>{
  const html=fs.readFileSync(path.resolve(__dirname,'../frontend/dist/index.html'),'utf8');
  assert.match(html,/DIFFERENT SLICERS\.<br>ONE SIMPLE WORKFLOW\./);
  assert.doesNotMatch(html,new RegExp(['A STRONGER','COMMUNITY'].join(' '),'i'));
});

test('installed Orca profiles join the target list via backend-issued IDs only',()=>{
  const html=fs.readFileSync(path.resolve(__dirname,'../frontend/dist/index.html'),'utf8');
  const js=fs.readFileSync(path.resolve(__dirname,'../frontend/dist/app.js'),'utf8');
  const batch=fs.readFileSync(path.resolve(__dirname,'../batch.go'),'utf8');
  const backend=fs.readFileSync(path.resolve(__dirname,'../presets/orca.go'),'utf8');
  // The four built-in rows stay the only static adapters in the HTML.
  assert.equal((html.match(/data-target-row="/g)||[]).length,4);
  // Installed rows are appended to the same list that conversion reads.
  assert.match(js,/api\(\)\?\.InstalledOrcaTargets/);
  assert.match(js,/installedRows\.push\(\{id:target\.id,row,checkbox\}\)/);
  assert.match(js,/#targetList input\[type=checkbox\]:checked:not\(:disabled\)/);
  assert.match(js,/printerNames\[id\]\|\|installedTargets\.find/);
  // Frontend never sends or receives installed-profile paths.
  assert.doesNotMatch(js,/target\.(path|root|file|executable)\b/);
  assert.match(backend,/func IsOrcaTargetID\(id string\) bool/);
  assert.match(batch,/presets\.IsOrcaTargetID\(target\)/);
  assert.match(batch,/presets\.ResolveOrcaBundle\(target\)/);
});

test('native bootstrap waits for all deferred My Printers modules to register',()=>{
  const html=fs.readFileSync(path.resolve(__dirname,'../frontend/dist/index.html'),'utf8');
  const app=fs.readFileSync(path.resolve(__dirname,'../frontend/dist/app.js'),'utf8');
  assert.ok(html.indexOf('src="./app.js"')<html.indexOf('src="./my-printers.js"'));
  assert.match(app,/setTimeout\(initialiseNative,0\);/);
  assert.doesNotMatch(app,/^initialiseNative\(\);/m);
});
