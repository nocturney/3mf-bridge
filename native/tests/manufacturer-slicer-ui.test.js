'use strict';
const {test}=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const path=require('node:path');
const vm=require('node:vm');
const picker=require('../frontend/dist/printer-picker.js');
const catalog=JSON.parse(fs.readFileSync(path.resolve(__dirname,'../frontend/dist/printer-catalog.json'),'utf8'));
const scriptPath=path.resolve(__dirname,'../frontend/dist/my-printers.js');
const script=fs.readFileSync(scriptPath,'utf8');
const css=fs.readFileSync(path.resolve(__dirname,'../frontend/dist/app.css'),'utf8');

// Backend contract: App.ManufacturerSlicers() -> [{id,name,manufacturers,installed,evidence?,sourceOnly}].
const slicerFixture=[
  {id:'prusaslicer',name:'PrusaSlicer',manufacturers:['Prusa','Prusa Research'],installed:true,
   evidence:'/Users/someone/Library/Application Support/PrusaSlicer',sourceOnly:false},
  {id:'creality-print',name:'Creality Print',manufacturers:['Creality'],installed:true,sourceOnly:false},
  {id:'anycubic-slicer-next',name:'Anycubic Slicer Next',manufacturers:['Anycubic'],installed:true,sourceOnly:false},
];
let helperModule=null;
// JSON round-trip: values built inside the vm context have another realm's prototypes.
const pureHelper=(slicers,maker)=>JSON.parse(JSON.stringify(
  (helperModule??=loadModule()).slicersForManufacturer(slicers,maker)));

// Inert stand-in for DOM nodes. init() only assigns properties and appends, so
// this checks the module's logic without needing a browser.
function inertElement(){
  const target={dataset:{},classList:{add(){},remove(){},toggle(){},contains(){return false}}};
  return new Proxy(target,{
    get(obj,key){
      if(key in obj)return obj[key];
      if(typeof key==='symbol'||key==='then')return undefined;
      return ()=>inertElement();
    },
    set(obj,key,value){obj[key]=value;return true;},
  });
}
function loadModule(app){
  const window={SliceBridgePicker:picker,dispatchEvent(){},go:{main:{App:app}}};
  const document={documentElement:{lang:'en'},getElementById:()=>inertElement(),
                  querySelector:()=>null,createElement:()=>inertElement()};
  vm.runInNewContext(script,{window,document,setTimeout,clearTimeout,Event:class{constructor(type){this.type=type}}},
                     {filename:scriptPath});
  return window.SliceBridgeMyPrinters;
}
function fakeApp(extra={}){
  const calls=[];
  return {
    calls,
    PrinterCatalog(){calls.push('PrinterCatalog');return catalog;},
    GetMyPrinters(){calls.push('GetMyPrinters');return {selectedIds:[],onboardingPending:false};},
    ...extra,
  };
}

test('installed PrusaSlicer, Creality Print and Anycubic Slicer Next attach to their exact makers',()=>{
  assert.deepEqual(pureHelper(slicerFixture,'Prusa Research').map(x=>x.name),['PrusaSlicer']);
  assert.deepEqual(pureHelper(slicerFixture,'Prusa').map(x=>x.name),['PrusaSlicer']);
  assert.deepEqual(pureHelper(slicerFixture,'Creality').map(x=>x.name),['Creality Print']);
  assert.deepEqual(pureHelper(slicerFixture,'Anycubic').map(x=>x.name),['Anycubic Slicer Next']);
  assert.deepEqual(pureHelper(slicerFixture,'Anker'),[]);
});

test('no false negatives: whitespace and case differences in the backend list still match',()=>{
  const variant=[{id:'creality-print',name:'Creality Print',manufacturers:['  creality '],installed:true,sourceOnly:false}];
  assert.deepEqual(pureHelper(variant,'Creality').map(x=>x.id),['creality-print']);
  assert.deepEqual(pureHelper(variant,'  CREALITY').map(x=>x.id),['creality-print']);
  const twoForOneMaker=[
    {id:'a',name:'First',manufacturers:['Creality'],installed:true},
    {id:'b',name:'Second',manufacturers:['Creality'],installed:true},
    {id:'a',name:'First again',manufacturers:['Creality'],installed:true},
  ];
  assert.deepEqual(pureHelper(twoForOneMaker,'Creality'),[{id:'a',name:'First'},{id:'b',name:'Second'}]);
});

test('no false positives: substrings, uninstalled, malformed and generic entries attach to nothing',()=>{
  assert.deepEqual(pureHelper(slicerFixture,'Creality Labs'),[]);
  assert.deepEqual(pureHelper(slicerFixture,'Crea'),[]);
  assert.deepEqual(pureHelper([{...slicerFixture[1],installed:false}],'Creality'),[]);
  assert.deepEqual(pureHelper([{...slicerFixture[1],installed:'true'}],'Creality'),[]);
  assert.deepEqual(pureHelper([{...slicerFixture[1],manufacturers:'Creality'}],'Creality'),[]);
  assert.deepEqual(pureHelper([{...slicerFixture[1],name:'   '}],'Creality'),[]);
  assert.deepEqual(pureHelper(null,'Creality'),[]);
  assert.deepEqual(pureHelper(slicerFixture,''),[]);
});

test('generic OrcaSlicer install without a verified maker list attaches to no catalogue printer',()=>{
  const generic=[
    {id:'orcaslicer',name:'OrcaSlicer',manufacturers:[],installed:true,sourceOnly:false},
    {id:'orcaslicer-wild',name:'OrcaSlicer',manufacturers:['*'],installed:true,sourceOnly:false},
  ];
  const makers=[...new Set(catalog.models.map(model=>model.manufacturer))];
  for(const maker of makers)assert.deepEqual(pureHelper(generic,maker),[],maker);
});

test('Bambu Lab destination stays hidden even when Bambu Studio is detected as a source',()=>{
  const bambu=[{id:'bambu-studio',name:'Bambu Studio',manufacturers:['Bambu Lab'],installed:true,sourceOnly:true}];
  assert.deepEqual(pureHelper(bambu,'Bambu Lab').map(x=>x.name),['Bambu Studio']);
  // The picker never builds Bambu cards, so the detected source cannot surface a line.
  assert.equal(picker.search(catalog.models,'Bambu Lab').total,0);
  assert.equal(picker.search(catalog.models,'').models.some(x=>x.manufacturer==='Bambu Lab'),false);
});

test('helper output is only id and name; evidence and local paths never leave the module',()=>{
  const out=pureHelper(slicerFixture,'Prusa');
  assert.deepEqual(out,[{id:'prusaslicer',name:'PrusaSlicer'}]);
  assert.equal(JSON.stringify(out).includes('/Users/'),false);
  const long=[{id:'x',name:'N'.repeat(200),manufacturers:['Creality'],installed:true}];
  assert.equal(pureHelper(long,'Creality')[0].name.length,80);
  assert.doesNotMatch(script,/\.evidence|\.path\b|\.root\b|\.executable\b/);
});

test('slicer names are rendered as text only, never as markup',()=>{
  const hostile=[{id:'h',name:'<img src=x onerror=alert(1)>',manufacturers:['Creality'],installed:true}];
  // Returned as plain data; the card builder only ever passes it to textContent.
  assert.equal(pureHelper(hostile,'Creality')[0].name,'<img src=x onerror=alert(1)>');
  assert.doesNotMatch(script,/innerHTML|insertAdjacentHTML|outerHTML|document\.write|\beval\s*\(|new Function/);
  assert.match(script,/copy\.append\(node\('small','installed-slicer-line',/);
  assert.match(script,/if\(slicerNames\.length\)copy\.append/);
});

test('card line is one quiet context line, uses the exact-maker helper, and shows nothing otherwise',()=>{
  assert.match(script,/const slicerNames=slicersForManufacturer\(manufacturerSlicers,model\.manufacturer\)\.map\(s=>s\.name\);/);
  assert.match(script,/t\(slicerNames\.length>1\?'slicersInstalled':'slicerInstalled',\{names:slicerNames\.join\(', '\)\}\)/);
  assert.match(css,/\.catalog-model-copy \.installed-slicer-line\{color:#526B61;font-size:10px;overflow-wrap:anywhere;unicode-bidi:plaintext\}/);
  // No badges, tags, status chips or new pills in the picker or the selected summary.
  assert.doesNotMatch(script,/badge|chip|certif|approved|convertib|auto-?config|preset\b/i);
  assert.doesNotMatch(script,/['"][\w -]*\btag\b[\w -]*['"]/i);
});

test('new copy is natural bilingual text without approval or convertibility claims',()=>{
  const en=script.match(/slicerInstalled:'Installed slicer: \{names\}',slicersInstalled:'Installed slicers: \{names\}'/);
  assert.ok(en);
  const he=script.match(/slicerInstalled:'([^']+)',slicersInstalled:'([^']+)'/);
  assert.ok(he);
  assert.ok(he[1].includes('{names}')&&he[1].includes('מותקן'));
  assert.ok(he[2].includes('{names}')&&he[2].includes('מותקנים'));
  assert.doesNotMatch(en[0],/approved|convertible|certified|compatible|supported|preset|verified|ready/i);
});

test('discovery is optional: missing, failing or malformed ManufacturerSlicers never block My Printers',async()=>{
  const cases=[
    ['method absent',fakeApp()],
    ['method throws synchronously',fakeApp({ManufacturerSlicers(){throw Error('boom');}})],
    ['method rejects',fakeApp({ManufacturerSlicers(){return Promise.reject(Error('offline'));}})],
    ['non-array reply',fakeApp({ManufacturerSlicers(){return {installed:true};}})],
    ['null reply',fakeApp({ManufacturerSlicers(){return null;}})],
    ['valid reply',fakeApp({ManufacturerSlicers(){return slicerFixture;}})],
  ];
  for(const [label,app] of cases){
    const mod=loadModule(app);
    await mod.init();
    assert.equal(mod.isReady(),true,label);
    assert.deepEqual(app.calls.filter(c=>c==='PrinterCatalog'||c==='GetMyPrinters'),
                     ['PrinterCatalog','GetMyPrinters'],label);
  }
});

test('a hanging ManufacturerSlicers call is dropped so the existing picker still loads',async()=>{
  const app=fakeApp({ManufacturerSlicers(){return new Promise(()=>{});}});
  const mod=loadModule(app);
  const started=Date.now();
  await mod.init();
  assert.equal(mod.isReady(),true);
  assert.ok(Date.now()-started<5000);
});

test('slicer discovery never feeds convertibility: 603 stays and catalogue-only printers stay non-convertible',()=>{
  assert.equal(catalog.models.length,603);
  const everyInstalled=catalog.models.flatMap(model=>pureHelper(slicerFixture,model.manufacturer).map(()=>model.id));
  assert.ok(everyInstalled.length>0);
  const catalogOnly=catalog.models.find(model=>!model.legacyTarget&&!picker.isHidden(model));
  const result=picker.statuses(catalog.models,[catalogOnly.id],[]);
  assert.equal(result[0].kind,'catalog_only');
  assert.equal(result[0].canConvert,false);
  assert.equal(picker.selectedAdapterTargets(catalog.models,[catalogOnly.id]).length,0);
  // The render path feeds only the picker's own availability list into statuses.
  assert.match(script,/picker\.statuses\(catalog\.models,choice,installed\)/);
  assert.doesNotMatch(script,/statuses\([^)]*manufacturerSlicers/);
});

test('init loads slicer discovery in the same Promise.all as the existing calls',()=>{
  assert.match(script,/Promise\.all\(\[\s*api\(\)\.PrinterCatalog\(\),api\(\)\.GetMyPrinters\(\),loadManufacturerSlicers\(\)\s*\]\)/);
  assert.match(script,/catalog=list;selected=prefs;manufacturerSlicers=slicers;/);
  assert.match(script,/Array\.isArray\(reply\)\?reply:\[\]/);
  assert.match(script,/\.catch\(\(\)=>\[\]\)/);
  assert.doesNotMatch(script,/fetch\s*\(|XMLHttpRequest|https?:\/\//);
});
