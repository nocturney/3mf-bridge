'use strict';
const {test}=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const path=require('node:path');
const vm=require('node:vm');
const {catalog,loadMyPrinters,availabilityAll,picker,createDocument,createWindow}=require('./my-printers-harness.js');

const approved=['Anker','Anycubic','Bambu Lab','Creality','ELEGOO','Flashforge',
  'FLSun','Prusa','Prusa Research','QIDI','Snapmaker','Sovol','Voron'];
const visibleMakers=['Anker','Anycubic','Creality','ELEGOO','Flashforge','FLSun',
  'Prusa Research','QIDI','Snapmaker','Sovol','Voron'];
const available=availabilityAll();

function makerOptions(h) {
  return h.dom('printerManufacturerFilter').children.map(el=>el.value).filter(Boolean);
}
function cardValues(h) {
  return h.dom('catalogModelList').children.flatMap(card=>{
    const main=card.children.find(child=>child.tagName==='INPUT');
    const variants=card.children.find(child=>child.classList.contains('catalog-variant-list'));
    return [main?.value,...(variants?.children.map(row=>row.children[0]?.value)||[])].filter(Boolean);
  });
}
function cardById(h,id) {
  return h.dom('catalogModelList').children.find(card=>
    cardValuesFor(card).includes(id));
}
function cardValuesFor(card) {
  const variants=card.children.find(child=>child.classList.contains('catalog-variant-list'));
  return [card.children[0]?.value,...(variants?.children.map(row=>row.children[0]?.value)||[])];
}
async function setup(saved=[],installed=[],extra={}) {
  let savedRequest=null;
  const h=loadMyPrinters({
    GetMyPrinters:()=>({selectedIds:[...saved],onboardingPending:false}),
    SaveMyPrinters:(ids)=>{
      savedRequest=[...ids];
      return {selectedIds:ids,onboardingPending:false};
    },
  });
  h.api.setSupportedTargets({available,installedTargets:installed,prusaAvailable:true,...extra});
  await h.api.init();
  h.dom('myPrintersBtn').fire('click');
  return {h,lastSave:()=>savedRequest};
}
const tick=()=>new Promise(resolve=>setImmediate(resolve));

test('exactly 13 approved source names, 11 destination display names and all 603 original records',async()=>{
  assert.equal(catalog.models.length,603);
  assert.equal(catalog.models.filter(m=>approved.includes(m.manufacturer)).length,263);
  assert.equal(catalog.models.filter(m=>m.manufacturer==='Bambu Lab').length,14);
  assert.equal(catalog.models.filter(m=>m.manufacturer==='Prusa'||m.manufacturer==='Prusa Research').length,44);
  const {h}=await setup();
  assert.deepEqual(makerOptions(h),visibleMakers);
  assert.equal(h.dom('catalogModelList').children.length,241);
  assert.equal(h.dom('showOtherManufacturersBtn').getAttribute('aria-pressed'),'false');
  const catalogueIds=new Set(catalog.models.filter(m=>m.manufacturer!=='Bambu Lab').map(m=>m.id));
  const inView=cardValues(h).filter(id=>catalogueIds.has(id));
  // Some cards repeat their default ID as the top-level checkbox and
  // an individually selectable variant, so count distinct persisted IDs.
  assert.equal(new Set(inView).size,249);
  assert.equal(inView.some(id=>id.startsWith('prusa-project')),false);
});

test('default search cannot expose other makers; show more restores them and is reversible',async()=>{
  const {h}=await setup();
  const search=h.dom('printerCatalogSearch');
  search.value='Afinia';search.fire('input');
  assert.equal(cardValues(h).some(id=>id.startsWith('afinia-')),false);
  h.dom('showOtherManufacturersBtn').fire('click');
  assert.equal(h.dom('showOtherManufacturersBtn').getAttribute('aria-pressed'),'true');
  assert.ok(makerOptions(h).includes('Afinia'));
  assert.ok(cardValues(h).includes('afinia-afinia-h-1-hs'));
  assert.equal(h.dom('showOtherManufacturersBtn').textContent,'Show selected manufacturers only');
  h.dom('showOtherManufacturersBtn').fire('click');
  assert.equal(h.dom('showOtherManufacturersBtn').getAttribute('aria-pressed'),'false');
  assert.equal(makerOptions(h).includes('Afinia'),false);
  assert.equal(cardValues(h).some(id=>id.startsWith('afinia-')),false);
  search.value='Creator 5 Pro';search.fire('input');
  assert.ok(cardValues(h).includes('flashforge-creator-5-pro'));
});

test('full original catalogue is accessible on demand: 589 IDs + generic export, never Bambu targets',async()=>{
  const {h}=await setup();
  h.dom('showOtherManufacturersBtn').fire('click');
  assert.equal(h.dom('catalogModelList').children.length,582);
  const visible=new Set(cardValues(h));
  const expected=catalog.models.filter(m=>m.manufacturer!=='Bambu Lab').map(m=>m.id);
  assert.equal(expected.length,589);
  assert.ok(expected.every(id=>visible.has(id)));
  assert.ok(visible.has('prusa-project'));
  assert.equal(visible.size,590);
  assert.ok(catalog.models.filter(m=>m.manufacturer==='Bambu Lab').every(m=>!visible.has(m.id)));
  assert.equal(h.dom('catalogSearchCount').hidden,true);
});

test('a saved printer from a hidden maker stays checked, searchable, and is saved unchanged',async()=>{
  const saved=['afinia-afinia-h-1-hs','snapmaker-u1'];
  const {h,lastSave}=await setup(saved);
  assert.equal(h.dom('myPrintersSelectionCount').textContent,'2 printers selected');
  assert.ok(makerOptions(h).includes('Afinia'),'selected hidden maker must be discoverable');
  const entry=cardById(h,saved[0]);
  assert.ok(entry,'hidden-but-selected catalogue entry is visible in the compact view');
  assert.ok(entry.children.some(child=>child.tagName==='INPUT'&&child.checked));
  assert.equal(h.dom('selectedPrinterSummary').children.length,2);
  assert.equal(cardValues(h).includes('artillery-artillery-sidewinder-x1'),false);
  h.dom('saveMyPrintersBtn').fire('click');
  await tick();
  assert.deepEqual(lastSave(),saved);
  assert.deepEqual(h.plain(h.api.selectedTargetIds()),['snapmaker-u1']);
});

test('five actual selected models and one dynamic Creator 5 Pro profile survive expand, collapse and save',async()=>{
  const pro={id:'orca-flashforge-creator-5-pro-aaaabbbbcc',vendor:'Flashforge',
    model:'Creator 5 Pro',title:'Flashforge Creator 5 Pro 0.25 nozzle',
    nozzle:'0.25',slicer:'Flash Studio Desktop'};
  const saved=['elegoo-centauri-carbon-2','flashforge-creator-5','qidi-x-plus-5',
    'snapmaker-u1',pro.id];
  const {h,lastSave}=await setup(saved,[pro],{prusaAvailable:false});
  assert.equal(h.dom('myPrintersSelectionCount').textContent,'5 printers selected');
  assert.equal(h.dom('selectedPrinterSummary').children.length,5);
  assert.ok(saved.every(id=>cardValues(h).includes(id)));
  h.dom('showOtherManufacturersBtn').fire('click');
  h.dom('showOtherManufacturersBtn').fire('click');
  assert.ok(saved.every(id=>cardValues(h).includes(id)));
  assert.equal(h.dom('myPrintersSelectionCount').textContent,'5 printers selected');
  assert.deepEqual(h.plain(h.api.selectedTargetIds()),saved);
  h.dom('saveMyPrintersBtn').fire('click');
  await tick();
  assert.deepEqual(lastSave(),saved);
  assert.deepEqual(h.plain(h.api.selectedTargetIds()),saved);
});

test('hidden maker with valid installed Orca conversion is never disabled or erased by the visual filter',async()=>{
  const foreign={id:'orca-artillery-sidewinder-0123456789',vendor:'Artillery',
    title:'Artillery Sidewinder 0.4 nozzle',model:'Sidewinder',
    nozzle:'0.4',slicer:'OrcaSlicer'};
  const {h,lastSave}=await setup([foreign.id],[foreign]);
  assert.equal(h.api.isSelectedTarget(foreign.id),true);
  assert.ok(cardValues(h).includes(foreign.id),'saved hidden installed profile stays visible');
  assert.ok(makerOptions(h).includes('Artillery'));
  h.dom('showOtherManufacturersBtn').fire('click');
  h.dom('showOtherManufacturersBtn').fire('click');
  assert.equal(h.api.isSelectedTarget(foreign.id),true);
  h.dom('saveMyPrintersBtn').fire('click');
  await tick();
  assert.deepEqual(lastSave(),[foreign.id]);
});

test('installed QIDIStudio X 5 Series profile is shown under QIDI in the picker only',async()=>{
  const target={id:'orca-x5series-plus5-0123456789',vendor:'X 5 Series',
    title:'X 5 Series Plus5 0.4 nozzle',model:'Plus5',
    nozzle:'0.4',slicer:'QIDIStudio'};
  const {h}=await setup([],[target]);
  assert.deepEqual(makerOptions(h),visibleMakers);
  const filtered=h.dom('printerManufacturerFilter');
  filtered.value='QIDI';filtered.fire('change');
  assert.ok(cardValues(h).includes(target.id));
  assert.ok(makerOptions(h).includes('QIDI'));
  assert.ok(!makerOptions(h).includes('X 5 Series'));
});

test('manufacturer expansion is session-only and language toggle copy is bilingual',async()=>{
  const {h}=await setup();
  h.dom('showOtherManufacturersBtn').fire('click');
  assert.equal(h.dom('showOtherManufacturersBtn').getAttribute('aria-pressed'),'true');
  h.dom('myPrintersDialog').close();
  h.dom('myPrintersBtn').fire('click');
  assert.equal(h.dom('showOtherManufacturersBtn').getAttribute('aria-pressed'),'false');
  assert.equal(makerOptions(h).includes('Afinia'),false);
  h.document.documentElement.lang='he';
  h.api.render();
  assert.equal(h.dom('showOtherManufacturersBtn').textContent,'הצג יצרנים נוספים');
  h.dom('showOtherManufacturersBtn').fire('click');
  assert.equal(h.dom('showOtherManufacturersBtn').textContent,'הצג רק יצרנים נבחרים');
});

test('manufacturer scope cannot modify conversion/readiness or source-only Bambu policy',async()=>{
  const before=picker.supportedModels(catalog.models,available,[],true).map(x=>x.id);
  assert.ok(before.includes('prusa-project'));
  assert.ok(before.includes('snapmaker-u1'));
  assert.equal(before.some(id=>id.startsWith('bambu')),false);
  const {h}=await setup(['snapmaker-u1']);
  const targets=h.plain(h.api.selectedTargetIds());
  h.dom('showOtherManufacturersBtn').fire('click');
  h.dom('showOtherManufacturersBtn').fire('click');
  assert.deepEqual(h.plain(h.api.selectedTargetIds()),targets);
  assert.deepEqual(picker.supportedModels(catalog.models,available,[],true).map(x=>x.id),before);
});


test('optional picker load failure does not crash My Printers script or unrelated app UI',()=>{
  const script=fs.readFileSync(path.resolve(__dirname,'../frontend/dist/my-printers.js'),'utf8');
  const document=createDocument();
  const window=createWindow({});
  // Intentional: no window.SliceBridgePicker, as in a failed optional asset.
  assert.doesNotThrow(()=>vm.runInNewContext(script,{document,window,Event:class{}}));
  assert.equal(window.SliceBridgeMyPrinters.isReady(),false);
});
