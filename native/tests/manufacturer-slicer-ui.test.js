'use strict';
// An installed slicer is not conversion support. My Printers no longer asks which
// installed slicers list a maker, and no card carries an "Installed slicer" line.
// A destination exists only when a backend adapter, an installed Orca target or the
// PrusaSlicer probe says so.
const {test}=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const path=require('node:path');
const picker=require('../frontend/dist/printer-picker.js');
const {loadMyPrinters,availabilityAll,catalog}=require('./my-printers-harness.js');
const tick=()=>new Promise(resolve=>setImmediate(resolve));

const dist=path.resolve(__dirname,'../frontend/dist');
const script=fs.readFileSync(path.join(dist,'my-printers.js'),'utf8');
const app=fs.readFileSync(path.join(dist,'app.js'),'utf8');
const css=fs.readFileSync(path.join(dist,'app.css'),'utf8');

// Backend shape of App.ManufacturerSlicers(); kept only to prove it is ignored.
const installedSlicers=[
  {id:'prusaslicer',name:'PrusaSlicer',manufacturers:['Prusa','Prusa Research'],installed:true,
   evidence:'/Users/someone/Library/Application Support/PrusaSlicer',sourceOnly:false},
  {id:'creality-print',name:'Creality Print',manufacturers:['Creality'],installed:true,sourceOnly:false},
];

test('the picker no longer calls ManufacturerSlicers or carries its helpers',()=>{
  assert.doesNotMatch(script,/ManufacturerSlicers/);
  assert.doesNotMatch(script,/slicersForManufacturer|loadManufacturerSlicers|SLICER_LOOKUP_MS|manufacturerSlicers/);
  assert.doesNotMatch(app,/ManufacturerSlicers/);
});

test('no installed-slicer context line remains in either language or in styling',()=>{
  assert.doesNotMatch(script,/slicerInstalled|slicersInstalled|Installed slicers?:|installed-slicer-line/);
  assert.doesNotMatch(script,/סלייסר מותקן|סלייסרים מותקנים/);
  assert.doesNotMatch(css,/installed-slicer-line/);
  assert.doesNotMatch(script,/installed-slicer/);
});

test('an installed slicer alone is not conversion support: nothing is ready, and the list is the catalogue',async()=>{
  const h=loadMyPrinters({ManufacturerSlicers:()=>installedSlicers});
  h.api.setSupportedTargets({available:[],installedTargets:[],prusaAvailable:false});
  await h.api.init();
  assert.equal(h.api.isReady(),true);
  assert.equal(h.calls.includes('ManufacturerSlicers'),false);
  assert.deepEqual(h.plain(h.api.selectedTargetIds()),[]);
  h.dom('myPrintersBtn').fire('click');
  // Catalogue models stay selectable; the slicer adds no maker or card of its own.
  const {preferred,others}=picker.manufacturerGroups(picker.catalogItems(catalog.models,[],[],false));
  const makers=h.dom('printerManufacturerFilter').children.map(option=>option.value).filter(Boolean);
  assert.deepEqual(makers,[...preferred,'',...others].filter(Boolean));
  assert.equal(h.dom('catalogModelList').children.length,picker.groupDestinations(picker.catalogItems(catalog.models,[],[],false)).length);
});

test('an installed slicer never makes a catalogue model a conversion target',async()=>{
  let sent=null;
  const h=loadMyPrinters({
    ManufacturerSlicers:()=>installedSlicers,
    SaveMyPrinters:(requested)=>{sent=requested;return {selectedIds:requested,onboardingPending:false};},
  });
  h.api.setSupportedTargets({available:availabilityAll({u1:true,c5:false,cc2:false,qidi:false}),installedTargets:[],prusaAvailable:false});
  await h.api.init();
  const creality=catalog.models.find(model=>model.manufacturer==='Creality');
  h.dom('myPrintersBtn').fire('click');
  h.dom('printerCatalogSearch').value=creality.model;
  h.dom('printerCatalogSearch').fire('input');
  const card=h.dom('catalogModelList').children.find(entry=>entry.children[0].value===creality.id);
  card.children[0].checked=true;
  card.children[0].fire('change');
  h.dom('saveMyPrintersBtn').fire('click');
  await tick();
  assert.deepEqual(h.plain(sent),[creality.id]);
  assert.equal(h.api.isSelectedTarget(creality.id),false);
  assert.deepEqual(h.plain(h.api.selectedTargetIds()),[]);
  assert.deepEqual(h.plain(h.api.legacyTargets()),[]);
});

test('a backend slicer list cannot create a destination the availability check did not admit',()=>{
  // The helper takes no slicer input at all; only adapter, Orca and Prusa signals count.
  assert.deepEqual(picker.supportedModels(catalog.models,[],[],false),[]);
  assert.equal(picker.supportedModels(catalog.models,availabilityAll({u1:false,c5:false,cc2:false,qidi:false}),[],false).length,0);
});
