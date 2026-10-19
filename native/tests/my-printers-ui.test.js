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

test('search works for brand, model and legacy aliases, without inventing release dates',()=>{
  const u1=picker.search(catalog.models,'snapmaker u1').models;
  assert.ok(u1.some(x=>x.id==='snapmaker-u1'));
  const c5=picker.search(catalog.models,'Creator 5').models;
  assert.ok(c5.some(x=>x.id==='flashforge-creator-5'));
  const cc2=picker.search(catalog.models,'ELEGOO CC2').models;
  assert.ok(cc2.some(x=>x.id==='elegoo-centauri-carbon-2'));
  assert.ok(catalog.models.every(x=>x.releaseDate===null));
  assert.equal(picker.search(catalog.models,'nonexistent printer 999999').total,0);
});

test('selected legacy adapters and unrelated index-only printers stay separate',()=>{
  const u1=catalog.models.find(x=>x.id==='snapmaker-u1');
  const c5=catalog.models.find(x=>x.id==='flashforge-creator-5');
  const other=catalog.models.find(x=>!x.legacyTarget && !picker.isHidden(x));
  const chosen=[other.id,c5.id,u1.id];
  assert.deepEqual(picker.selectedAdapterTargets(catalog.models,chosen),['c5','u1']);
  const result=picker.statuses(catalog.models,chosen,[
    {target:'c5',available:false,error:'Flash Studio not installed'},
    {target:'u1',available:true}
  ]);
  assert.deepEqual(result.map(x=>x.kind),['catalog_only','missing_local_profile','ready']);
  assert.deepEqual(result.map(x=>x.canConvert),[false,false,true]);
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
                   'catalogModelList','saveMyPrintersBtn','cancelMyPrintersBtn']){
    assert.ok(doc.includes('id="'+id+'"'),id);
  }
  assert.match(doc,/src="\.\/printer-picker\.js"/);
  assert.match(doc,/src="\.\/my-printers\.js"/);
  assert.match(script,/SaveMyPrinters/);
  assert.match(script,/GetMyPrinters/);
  assert.match(script,/PrinterCatalog/);
  assert.match(script,/קטלוג בלבד/);
  assert.match(script,/catalog only/i);
  assert.doesNotMatch(script,/fetch\s*\(|XMLHttpRequest|https?:\/\//);
});
