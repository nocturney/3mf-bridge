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

test('an installed slicer alone is not conversion support: no card, maker or message',async()=>{
  const h=loadMyPrinters({ManufacturerSlicers:()=>installedSlicers});
  h.api.setSupportedTargets({available:[],installedTargets:[],prusaAvailable:false});
  await h.api.init();
  assert.equal(h.api.isReady(),true);
  assert.equal(h.calls.includes('ManufacturerSlicers'),false);
  h.dom('myPrintersBtn').fire('click');
  assert.equal(h.dom('catalogModelList').children.length,0);
  assert.deepEqual(h.dom('printerManufacturerFilter').children.map(option=>option.value),['']);
  assert.equal(h.dom('catalogPickerHint').textContent,'No destinations are available on this computer yet.');
});

test('an installed slicer does not add makers that have no available adapter',async()=>{
  const h=loadMyPrinters({ManufacturerSlicers:()=>installedSlicers});
  h.api.setSupportedTargets({available:availabilityAll({u1:true,c5:false,cc2:false,qidi:false}),installedTargets:[],prusaAvailable:false});
  await h.api.init();
  const makers=h.dom('printerManufacturerFilter').children.map(option=>option.value);
  assert.deepEqual(makers,['','Snapmaker']);
  assert.equal(makers.includes('Creality'),false);
  assert.equal(makers.includes('Prusa'),false);
});

test('a backend slicer list cannot create a destination the availability check did not admit',()=>{
  // The helper takes no slicer input at all; only adapter, Orca and Prusa signals count.
  assert.deepEqual(picker.supportedModels(catalog.models,[],[],false),[]);
  assert.equal(picker.supportedModels(catalog.models,availabilityAll({u1:false,c5:false,cc2:false,qidi:false}),[],false).length,0);
});
