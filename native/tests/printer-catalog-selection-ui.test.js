'use strict';
// Full catalogue selection: every non-Bambu catalogue ID is selectable, saved membership never
// implies conversion readiness, one card per physical model, and saved-first / maker / date order.
const {test}=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const path=require('node:path');
const picker=require('../frontend/dist/printer-picker.js');
const {loadMyPrinters,availabilityAll,catalog}=require('./my-printers-harness.js');

const dist=path.resolve(__dirname,'../frontend/dist');
const tick=()=>new Promise(resolve=>setImmediate(resolve));
const plain=(value)=>JSON.parse(JSON.stringify(value));
const nonBambu=catalog.models.filter(model=>!picker.isHidden(model));
const cardIds=(h)=>h.dom('catalogModelList').children.map(card=>card.children[0].value);
const cardFor=(h,id)=>h.dom('catalogModelList').children.find(card=>card.children[0].value===id);
const orcaPro=(nozzle,hex,slicer)=>({
  id:'orca-flashforge-creator-5-pro-'+nozzle.replace('.','')+'-'+hex,vendor:'Flashforge',model:'Creator 5 Pro',
  title:'Flashforge Creator 5 Pro '+nozzle+' nozzle',nozzle,physicalTools:1,slicer,
});
const proVariants=[orcaPro('0.25','1111111111','OrcaSlicer'),orcaPro('0.4','2222222222','OrcaSlicer'),
                   orcaPro('0.4','5555555555','Flash Studio')];

test('all 589 catalogue IDs remain recognized while duplicate spellings share one physical card',()=>{
  assert.equal(nonBambu.length,589);
  const groups=picker.groupDestinations(picker.catalogItems(catalog.models,[],[],false));
  const reachable=groups.flatMap(group=>group.ids).filter(id=>nonBambu.some(model=>model.id===id));
  assert.equal(reachable.length,589);
  assert.equal(new Set(reachable).size,589);
  // Existing saved IDs remain recognized even when aliases collapse to one model card.
  const creality=groups.find(group=>group.model==='CR-10 S Pro V2'||group.model==='CR-10S Pro V2');
  assert.equal(creality.ids.length,2);
});

test('curated catalogue opens first; the complete underlying catalogue is one toggle away',async()=>{
  const h=loadMyPrinters({GetMyPrinters:()=>({selectedIds:[],onboardingPending:false})});
  h.api.setSupportedTargets({available:[],installedTargets:[],prusaAvailable:false});
  await h.api.init();
  h.dom('myPrintersBtn').fire('click');
  const expected=picker.groupDestinations(picker.catalogItems(catalog.models,[],[],false));
  assert.equal(h.dom('catalogModelList').children.length,241);
  assert.equal(h.dom('showOtherManufacturersBtn').getAttribute('aria-pressed'),'false');
  h.dom('showOtherManufacturersBtn').fire('click');
  assert.equal(h.dom('catalogModelList').children.length,expected.length);
  assert.equal(h.dom('showOtherManufacturersBtn').getAttribute('aria-pressed'),'true');
  assert.equal(h.dom('catalogSearchCount').hidden,true);
  // Spelling aliases share ONE physical checkbox, rather than two unrelated
  // nozzle or conversion choices. Their exact stored identities still exist.
  h.dom('printerCatalogSearch').value='CR-10';
  h.dom('printerCatalogSearch').fire('input');
  const group=expected.find(group=>group.ids.includes('creality-creality-cr-10-s-pro-v2'));
  const card=cardFor(h,group.id);
  assert.ok(card);
  assert.equal(card.querySelectorAll('input').length,1);
  assert.equal(card.querySelectorAll('.catalog-variant-list').length,0);
  assert.deepEqual(group.ids.slice().sort(),
    ['creality-creality-cr-10-s-pro-v2','creality-creality-cr-10s-pro-v2']);
});

test('an unknown catalogue model is selected in the dialog but is never a conversion destination',async()=>{
  let sent=null;
  const afinia='afinia-afinia-h-1-hs';
  const h=loadMyPrinters({
    GetMyPrinters:()=>({selectedIds:[afinia,'snapmaker-u1'],onboardingPending:false}),
    SaveMyPrinters:(requested)=>{sent=requested;return {selectedIds:requested,onboardingPending:false};},
  });
  h.api.setSupportedTargets({available:availabilityAll(),installedTargets:[],prusaAvailable:false});
  await h.api.init();
  assert.deepEqual(plain(h.api.selectedTargetIds()),['snapmaker-u1']);
  assert.equal(h.api.isSelectedTarget(afinia),false);
  h.dom('myPrintersBtn').fire('click');
  assert.equal(cardFor(h,afinia).children[0].checked,true);
  assert.equal(h.dom('selectedPrinterSummary').children.length,2);
  // Availability disappears: the catalogue pick is kept, and conversion still has no target for it.
  h.api.setSupportedTargets({available:[],installedTargets:[],prusaAvailable:false});
  assert.equal(cardFor(h,afinia).children[0].checked,true);
  h.dom('saveMyPrintersBtn').fire('click');
  await tick();
  assert.deepEqual(plain(sent),[afinia,'snapmaker-u1']);
  assert.deepEqual(plain(h.api.selectedTargetIds()),[]);
  assert.deepEqual(plain(h.api.legacyTargets()),[]);
});

test('a physical model and its installed nozzle targets share one card with a 0.4 mm default',async()=>{
  const h=loadMyPrinters({GetMyPrinters:()=>({selectedIds:[],onboardingPending:false})});
  h.api.setSupportedTargets({available:[],installedTargets:proVariants,prusaAvailable:false});
  await h.api.init();
  const groups=picker.groupDestinations(picker.catalogItems(catalog.models,[],proVariants,false));
  const pro=groups.filter(group=>group.model==='Creator 5 Pro');
  assert.equal(pro.length,1);
  assert.ok(pro[0].ids.includes('flashforge-creator-5-pro'));
  assert.equal(pro[0].ready,true);
  assert.ok(proVariants.filter(v=>v.nozzle==='0.4').map(v=>v.id).includes(pro[0].id),'default is based on physical nozzle');
  h.dom('myPrintersBtn').fire('click');
  assert.equal(cardIds(h).filter(id=>id==='flashforge-creator-5-pro').length,0,'catalogue ID is not a separate card');
  h.dom('printerCatalogSearch').value='Creator 5 Pro';
  h.dom('printerCatalogSearch').fire('input');
  const card=cardFor(h,pro[0].id);
  assert.equal(card.querySelectorAll('input').length,1);
  assert.equal(card.querySelectorAll('.catalog-variant-list').length,0);
  assert.equal(card.querySelectorAll('.catalog-model-profile-button').length,0,
    'unselected hardware cards never show a slicer/nozzle settings button');
  card.children[0].checked=true;card.children[0].fire('change');
  const pill=h.dom('selectedPrinterSummary').children.find(x=>x.children.some(c=>c.textContent==='Flashforge Creator 5 Pro'));
  assert.ok(pill,'selected printer has a summary chip');
  const button=pill.children.find(child=>child.classList.contains('selected-printer-settings'));
  assert.ok(button,'slicer override exists only for an actively selected printer with alternatives');
  button.fire('click');
  const options=h.dom('printerProfileOptions').children;
  assert.equal(options.length,2); // automatic Flash Studio + validated OrcaSlicer at same nozzle
  assert.ok(options.every(option=>option.children[0].value.startsWith('orca-')));
});

test('saved selections come first in saved order, then makers A-Z, newest release first within a maker',()=>{
  const groups=picker.groupDestinations(picker.catalogItems(catalog.models,availabilityAll(),[],true));
  const saved=['snapmaker-u1','afinia-afinia-h-1-hs'];
  const order=picker.searchGroups(groups,'',saved,picker.maxRender).models;
  assert.deepEqual(order.slice(0,2).map(group=>group.id),saved);
  const rest=order.slice(2);
  const makers=rest.map(group=>String(group.manufacturer));
  assert.deepEqual(makers,[...makers].sort((a,b)=>a.localeCompare(b,'en',{sensitivity:'base'})));
  // Within Prusa Research: dated launches newest first, then undated models alphabetically.
  const prusa=rest.filter(group=>group.manufacturer==='Prusa Research');
  const dated=prusa.filter(group=>group.releaseDate);
  assert.deepEqual(dated.map(group=>group.releaseDate),[...dated.map(group=>group.releaseDate)].sort().reverse());
  const firstUndated=prusa.findIndex(group=>!group.releaseDate);
  assert.ok(firstUndated>0);
  assert.ok(prusa.slice(firstUndated).every(group=>!group.releaseDate));
  const undated=prusa.slice(firstUndated).map(group=>group.model);
  assert.deepEqual(undated,[...undated].sort((a,b)=>a.localeCompare(b,'en',{numeric:true,sensitivity:'base'})));
  // No invented dates: a model without a catalogued or verified release has no date at all.
  assert.equal(groups.find(group=>group.manufacturer==='Afinia').releaseDate,null);
});

test('dates are never shown and no category or support-tier wording appears in the picker',()=>{
  const script=fs.readFileSync(path.join(dist,'my-printers.js'),'utf8');
  const html=fs.readFileSync(path.join(dist,'index.html'),'utf8');
  const dialog=html.slice(html.indexOf('<dialog id="myPrintersDialog"'),html.indexOf('id="settingsDialog"'));
  assert.doesNotMatch(script,/tier|categor|קטגור|שכבה/i);
  assert.doesNotMatch(dialog,/tier|categor|קטגור|שכבה/i);
  assert.doesNotMatch(script,/releaseDate|model-launch-date/);
});

test('the picker stylesheet has no fixed card or list heights that could hide variant rows',()=>{
  const css=fs.readFileSync(path.join(dist,'app.css'),'utf8');
  const pickerRules=css.slice(css.indexOf('/* My Printers: RTL/LTR-safe picker'),css.indexOf('/* Navigation and first-use picker refinement'));
  assert.doesNotMatch(pickerRules,/catalog-model-card\{[^}]*min-height:\s*\d+px/);
  assert.doesNotMatch(pickerRules,/catalog-model-list\{[^}]*max-height:\s*\d+px/);
  assert.match(pickerRules,/\.catalog-model-card\{display:grid;/);
  assert.match(pickerRules,/\.catalog-variant-list\{grid-column:1\/-1;/);
});
