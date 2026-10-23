'use strict';
// Per-printer PrusaSlicer targets discovered by App.InstalledPrusaTargets. Pure picker rules,
// My Printers artwork and selection, and the main list, all run against the real scripts.
const {test}=require('node:test');
const assert=require('node:assert/strict');
const path=require('node:path');
const picker=require('../frontend/dist/printer-picker.js');
const {availabilityAll,catalog,dist,FakeEvent,createDocument,createWindow,loadMyPrinters,recordingApp,runScripts}=require('./my-printers-harness.js');

const builtIn={u1:'snapmaker-u1',c5:'flashforge-creator-5',cc2:'elegoo-centauri-carbon-2',qidi:'qidi-x-plus-5'};
const targets=['u1','c5','cc2','qidi'];
const plain=(value)=>JSON.parse(JSON.stringify(value));
const ids=(items)=>items.map(item=>item.id);
const tick=()=>new Promise(resolve=>setImmediate(resolve));
async function settle(){for(let i=0;i<30;i++)await tick();}

const prusaMk4={id:'prusa-prusa-research-mk4-a1b2c3d4e5',title:'Prusa Research MK4 0.4 nozzle',vendor:'Prusa Research',
  model:'MK4',nozzle:'0.4',slicer:'PrusaSlicer',ready:true};
const prusaMk4Brass={id:'prusa-prusa-research-mk4-f1e2d3c4b5',title:'Prusa Research MK4 0.6 nozzle',vendor:'Prusa Research',
  model:'MK4',nozzle:'0.6',slicer:'PrusaSlicer',ready:true};
const orcaMk4={id:'orca-prusa-research-mk4-b2c3d4e5f6',title:'Prusa Research MK4 0.4 nozzle',vendor:'Prusa Research',
  model:'MK4',nozzle:'0.4',slicer:'OrcaSlicer',physicalTools:1};
const prusaK1={id:'prusa-creality-k1-c3d4e5f6a7',title:'Creality K1 0.4 nozzle',vendor:'Creality',model:'K1',
  nozzle:'0.4',slicer:'PrusaSlicer',ready:true};
const orcaK1={id:'orca-creality-k1-d4e5f6a7b8',title:'Creality K1 0.4 nozzle',vendor:'Creality',model:'K1',
  nozzle:'0.4',slicer:'OrcaSlicer',physicalTools:1};
const onAll=availabilityAll();

// ---- picker: physical grouping, separate readiness and the 589/582/603 baseline ----

test('the catalogue baseline holds: 603 models, 589 non-Bambu IDs, 582 physical cards, no dynamic rows',()=>{
  assert.equal(catalog.models.length,603);
  assert.equal(catalog.models.filter(model=>model.manufacturer!=='Bambu Lab').length,589);
  const listed=picker.catalogItems(catalog.models,onAll,[],true,[]);
  assert.equal(listed.length,590);
  assert.equal(picker.groupDestinations(listed).length,582);
});

test('a ready Prusa target that matches a catalogue model folds into that card and adds no card',()=>{
  const listed=picker.catalogItems(catalog.models,onAll,[],true,[prusaMk4]);
  assert.equal(listed.length,591);
  const groups=picker.groupDestinations(listed);
  assert.equal(groups.length,582);
  const card=groups.find(group=>group.ids.includes(prusaMk4.id));
  assert.ok(card,'Prusa MK4 target must join a catalogue card');
  assert.ok(card.ids.includes('prusa-research-mk4'),'catalogue ID stays in the card');
});

test('mixed Prusa and Orca variants of one physical model share a card with independent IDs',()=>{
  const items=picker.supportedModels(catalog.models,onAll,[orcaMk4],true,[prusaMk4,prusaMk4Brass]);
  const card=picker.groupDestinations(items).find(group=>group.ids.includes(prusaMk4.id));
  assert.ok(card);
  assert.deepEqual(card.ids.filter(id=>id===prusaMk4.id||id===prusaMk4Brass.id||id===orcaMk4.id).sort(),
    [prusaMk4.id,prusaMk4Brass.id,orcaMk4.id].sort());
  assert.equal(new Set(card.ids).size,card.ids.length);
  // Every variant keeps its own record; none is merged or renamed.
  for(const variant of card.variants)assert.ok(variant.id.length>0);
  const byId=Object.fromEntries(card.variants.map(v=>[v.id,v]));
  assert.equal(byId[prusaMk4.id].slicer,'PrusaSlicer');
  assert.equal(byId[orcaMk4.id].slicer,'OrcaSlicer');
  assert.equal(byId[prusaMk4Brass.id].nozzle,'0.6');
});

test('equivalent 0.4 mm Prusa and Orca profiles are ordered without manufacturer/conversion preference',()=>{
  const items=picker.supportedModels(catalog.models,onAll,[orcaMk4],true,[prusaMk4]);
  const card=picker.groupDestinations(items).find(group=>group.ids.includes(prusaMk4.id));
  assert.deepEqual(card.ids,[orcaMk4.id,prusaMk4.id]);
  assert.equal(card.id,orcaMk4.id); // Alphabetical slicer name, never conversion rating.
});

test('equal-nozzle Orca and Prusa entries sort alphabetically, and both stay selectable',()=>{
  const items=picker.supportedModels(catalog.models,[],[orcaK1],true,[prusaK1]);
  const card=picker.groupDestinations(items).find(group=>group.ids.includes(orcaK1.id));
  // No tier: equal nozzle means alphabetic slicer name for this card.
  assert.deepEqual(card.ids,[orcaK1.id,prusaK1.id]);
  assert.equal(card.id,orcaK1.id);
  assert.deepEqual(card.variants.map(v=>v.slicer),['OrcaSlicer','PrusaSlicer']);
});

test('readiness gating: unready and engine-less Prusa targets stay listed but never convert',()=>{
  const unready={...prusaK1,id:'prusa-creality-k1-0a0b0c0d0e',ready:false};
  const listed=picker.catalogItems(catalog.models,[],[],false,[unready,{...prusaK1,id:'prusa-creality-k1-1a1b1c1d1e'}]);
  const row=listed.find(item=>item.id===unready.id);
  assert.ok(row,'unready target keeps catalogue membership');
  assert.equal(row.ready,false);
  // The engine probe is required too: a ready target without it is not a destination.
  const noEngine=listed.find(item=>item.id==='prusa-creality-k1-1a1b1c1d1e');
  assert.equal(noEngine.ready,false);
  assert.deepEqual(picker.supportedModels(catalog.models,[],[],false,[unready,prusaK1]),[]);
  // The generic PrusaSlicer project is ready here too; only the backend-ready K1 joins it.
  assert.deepEqual(ids(picker.supportedModels(catalog.models,[],[],true,[unready,prusaK1])),[prusaK1.id,'prusa-project']);
});

test('ready must be the literal true; strings, numbers and missing flags never make a destination',()=>{
  const weak=[
    {...prusaK1,id:'prusa-creality-k1-2a2b2c2d2e',ready:'true'},
    {...prusaK1,id:'prusa-creality-k1-3a3b3c3d3e',ready:1},
    {...prusaK1,id:'prusa-creality-k1-4a4b4c4d4e',ready:undefined},
  ];
  const converts=ids(picker.supportedModels(catalog.models,[],[],true,weak)).filter(id=>id!=='prusa-project');
  assert.deepEqual(converts,[]);
});

test('malformed or non-Prusa IDs are rejected and never listed as Prusa destinations',()=>{
  const bad=[
    'prusa-x',
    'PRUSA-creality-k1-a1b2c3d4e5',
    'prusa-../../etc-a1b2c3d4e5',
    'prusa-creality-k1-a1b2c3d4e5 ',
    'prusa-creality-k1-a1b2c3d4e',
    'prusa-creality-k1-A1B2C3D4E5',
    'orca-creality-k1-a1b2c3d4e5',
  ];
  const listed=picker.catalogItems(catalog.models,[],[],true,bad.map(id=>({...prusaK1,id,ready:true})));
  assert.equal(listed.some(item=>bad.includes(item.id)),false);
  assert.equal(listed.some(item=>item.id==='orca-creality-k1-a1b2c3d4e5'),false);
});

test('an older backend that omits the Prusa list, or sends a non-array, changes nothing',()=>{
  const base=picker.catalogItems(catalog.models,onAll,[orcaK1],true);
  assert.equal(base.length,picker.catalogItems(catalog.models,onAll,[orcaK1],true,undefined).length);
  assert.deepEqual(ids(picker.catalogItems(catalog.models,onAll,[orcaK1],true,{targets:[prusaK1]})),ids(base));
  assert.deepEqual(ids(picker.catalogItems(catalog.models,onAll,[orcaK1],true,null)),ids(base));
  // 589 non-Bambu catalogue IDs, one installed Orca target, and the generic PrusaSlicer project.
  assert.equal(base.length,591);
});

test('saved Prusa IDs keep their place in the selection; stale ones are dropped without rewriting',()=>{
  const items=picker.catalogItems(catalog.models,[],[],true,[prusaK1,prusaMk4]);
  const saved=[prusaK1.id,'prusa-gone-0a0b0c0d0e',prusaMk4.id,builtIn.u1];
  // Only ready items convert; catalogue membership (allSavedIds) keeps any listed ID.
  assert.deepEqual(picker.selectedTargetIds(items,saved),[prusaK1.id,prusaMk4.id]);
  // Catalogue membership keeps any listed saved ID, including a catalogue-only built-in model.
  assert.deepEqual(picker.allSavedIds(items,saved),[prusaK1.id,prusaMk4.id,builtIn.u1]);
  assert.deepEqual(saved,[prusaK1.id,'prusa-gone-0a0b0c0d0e',prusaMk4.id,builtIn.u1],'saved list is not mutated');
});

test('Prusa targets never change the Orca or built-in ID shapes, and Bambu stays out',()=>{
  const bambu={id:'prusa-bambu-lab-x1-a1b2c3d4e5',title:'Bambu Lab X1',vendor:'Bambu Lab',model:'X1',nozzle:'0.4',
    slicer:'PrusaSlicer',ready:true};
  const items=picker.supportedModels(catalog.models,onAll,[],true,[bambu,prusaK1]);
  assert.equal(items.some(item=>/bambu/i.test(item.manufacturer)||item.id===bambu.id),false);
  assert.equal(items.find(item=>item.id===prusaK1.id).slicer,'PrusaSlicer');
});

// ---- My Printers: artwork stays local to Orca, saves exact IDs, offers unready targets as membership ----

test('artwork lookups for dynamic PrusaSlicer targets never reach the installed-target artwork method',async()=>{
  const artworkCalls=[];
  const h=loadMyPrinters({
    GetMyPrinters:()=>({selectedIds:[],onboardingPending:false}),
    PrinterArtwork:()=>({found:false}),
    PrinterArtworkForInstalledTarget:(id)=>{artworkCalls.push(id);return {found:false};},
  });
  h.api.setSupportedTargets({available:onAll,installedTargets:[orcaK1],installedPrusaTargets:[prusaK1],prusaAvailable:true});
  await h.api.init();
  h.dom('myPrintersBtn').fire('click');
  h.dom('printerCatalogSearch').value='K1';
  h.dom('printerCatalogSearch').fire('input');
  await settle();
  assert.ok(artworkCalls.includes(orcaK1.id),'Orca keeps its local installed-target lookup');
  assert.equal(artworkCalls.some(id=>id.startsWith('prusa-')),false);
});

test('My Printers saves the exact prusa- ID and never sends an unready target',async()=>{
  let sent=null;
  const unready={...prusaMk4Brass,ready:false};
  const h=loadMyPrinters({
    GetMyPrinters:()=>({selectedIds:[],onboardingPending:false}),
    SaveMyPrinters:(requested)=>{sent=requested;return {selectedIds:requested,onboardingPending:false};},
  });
  h.api.setSupportedTargets({available:onAll,installedTargets:[],installedPrusaTargets:[prusaMk4,unready],prusaAvailable:true});
  await h.api.init();
  h.dom('myPrintersBtn').fire('click');
  h.dom('printerCatalogSearch').value='MK4';
  h.dom('printerCatalogSearch').fire('input');
  const card=h.dom('catalogModelList').children.find(c=>c.children[0].value===prusaMk4.id);
  card.children[0].checked=true;card.children[0].fire('change');
  h.dom('saveMyPrintersBtn').fire('click');
  await tick();
  assert.deepEqual([...sent],[prusaMk4.id]);
  assert.equal(h.api.isSelectedTarget(prusaMk4.id),true);
  assert.equal(h.api.isSelectedTarget(unready.id),false);
});

// ---- main list: a shell running the real app.js, with and without the backend method ----

function startApp(config={}){
  const document=createDocument();
  const list=document.getElementById('targetList');
  const rows={};
  for(const key of targets){
    const row=document.createElement('label');
    row.className='target-option';row.dataset.targetRow=key;row.hidden=true;
    const input=document.createElement('input');input.setAttribute('type','checkbox');input.value=key;
    const summary=document.createElement('small');summary.dataset.printerSummary=key;
    const label=document.createElement('span');label.className='printer-label';label.append(summary);
    row.append(input,label);list.append(row);rows[key]=row;
  }
  const prusa=document.createElement('label');
  prusa.className='target-option';prusa.dataset.prusaTarget='';prusa.hidden=true;
  const prusaInput=document.createElement('input');prusaInput.setAttribute('type','checkbox');
  prusaInput.value='prusa-project';prusa.append(prusaInput);list.append(prusa);
  const {wailsApp,calls}=recordingApp({
    GetSettings:()=>({version:1,window:{width:1100,height:750},
      printers:Object.fromEntries(targets.map(target=>[target,{nozzle:'auto',slicer:'auto',filaments:{}}]))}),
    GetVersion:()=>({version:'0.4.0',platform:'darwin'}),
    Availability:()=>availabilityAll(),
    InstalledOrcaTargets:()=>({targets:[],rejected:[]}),
    PrusaEngineStatus:()=>({available:true,engine:'prusa-slicer'}),
    ...config,
  });
  const window=createWindow(wailsApp);
  const context={
    window,document,Event:FakeEvent,navigator:{language:'en-US'},
    localStorage:{getItem:()=>null,setItem(){}},setTimeout,clearTimeout,console,
  };
  runScripts([
    path.join(dist,'printer-picker.js'),
    path.join(dist,'app.js'),
    path.join(dist,'my-printers.js'),
  ],context);
  const rowFor=(id)=>document.querySelectorAll('[data-installed-target="'+id+'"]')[0];
  return {
    window,document,calls,rows,rowFor,
    dom:(id)=>document.getElementById(id),
    card:(id)=>document.getElementById('catalogModelList').children.find(c=>c.children[0].value===id)?.children[0],
  };
}
async function ready(app){
  for(let attempt=0;attempt<100&&!app.window.SliceBridgeMyPrinters?.isReady();attempt++)
    await new Promise(resolve=>setTimeout(resolve,2));
  assert.equal(app.window.SliceBridgeMyPrinters?.isReady(),true,'My Printers did not initialise');
  await settle();
}
const visible=(row)=>!row.hidden;
const prusaRows=(app)=>app.document.querySelectorAll('[data-installed-target]')
  .filter(row=>row.dataset.installedTarget.startsWith('prusa-'));

test('an older binary without InstalledPrusaTargets starts cleanly with no Prusa rows and working Orca rows',async()=>{
  const app=startApp({
    InstalledOrcaTargets:()=>({targets:[orcaK1],rejected:[]}),
    GetMyPrinters:()=>({selectedIds:[builtIn.u1,orcaK1.id],onboardingPending:false}),
  });
  assert.equal(app.calls.includes('InstalledPrusaTargets'),false);
  await ready(app);
  assert.equal(app.rowFor(orcaK1.id)?.hidden,false);
  assert.equal(prusaRows(app).length,0);
});

test('a throwing InstalledPrusaTargets call degrades to no Prusa rows without an error banner',async()=>{
  const app=startApp({
    InstalledPrusaTargets:()=>{throw Error('engine probe failed');},
    GetMyPrinters:()=>({selectedIds:[builtIn.u1],onboardingPending:false}),
  });
  await ready(app);
  assert.equal(prusaRows(app).length,0);
  assert.equal(visible(app.rows.u1),true);
  assert.notEqual(app.dom('activityDetail').textContent,'engine probe failed');
});

test('only backend-ready, selected Prusa targets get a row; the row is checked, keeps its nozzle and slicer text, and converts by exact ID',async()=>{
  const sent=[];
  const unready={...prusaMk4Brass,ready:false};
  const app=startApp({
    InstalledPrusaTargets:()=>({targets:[prusaMk4,unready],rejected:[]}),
    GetMyPrinters:()=>({selectedIds:[prusaMk4.id,unready.id],onboardingPending:false}),
    SelectFiles:()=>[{path:'C:/models/a.3mf',name:'a.3mf',folder:'C:/models',sizeLabel:'1 KB'}],
    ConvertBatch:(request)=>{sent.push(request.targets);return {filesDone:1,successes:0,failures:0,files:[]};},
  });
  await ready(app);
  assert.equal(app.rowFor(unready.id),undefined,'unready target gets no row');
  const row=app.rowFor(prusaMk4.id);
  assert.ok(row,'ready target gets a row');
  assert.equal(visible(row),true);
  assert.equal(row.querySelector('input').checked,true);
  assert.equal(row.querySelector('small'),null,'technical subtitle is hidden from printer tiles');
  assert.match(row.title,/0.4 mm · PrusaSlicer/,'exact profile remains inspectable on hover');
  app.dom('addFilesBtn').fire('click');
  await settle();
  app.dom('convertBtn').fire('click');
  await settle();
  assert.equal(sent.length,1);
  assert.deepEqual([...sent[0]],[prusaMk4.id]);
});

test('a ready Prusa row stays hidden when My Printers does not select it, and nothing is dispatched',async()=>{
  const sent=[];
  const app=startApp({
    InstalledPrusaTargets:()=>({targets:[prusaMk4],rejected:[]}),
    GetMyPrinters:()=>({selectedIds:[builtIn.u1],onboardingPending:false}),
    SelectFiles:()=>[{path:'C:/models/a.3mf',name:'a.3mf',folder:'C:/models',sizeLabel:'1 KB'}],
    ConvertBatch:(request)=>{sent.push(request.targets);return {filesDone:1,successes:0,failures:0,files:[]};},
  });
  await ready(app);
  const row=app.rowFor(prusaMk4.id);
  assert.equal(visible(row),false);
  assert.equal(row.querySelector('input').checked,false);
  app.rows.u1.querySelector('input').checked=true;
  app.dom('addFilesBtn').fire('click');
  await settle();
  app.dom('convertBtn').fire('click');
  await settle();
  assert.deepEqual(sent.map(list=>[...list]),[['u1']]);
});

test('a saved Prusa selection survives a reload with the same ID and is not rewritten on load',async()=>{
  let saves=0;
  const app=startApp({
    InstalledPrusaTargets:()=>({targets:[prusaMk4],rejected:[]}),
    GetMyPrinters:()=>({selectedIds:[prusaMk4.id,builtIn.u1],onboardingPending:false}),
    SaveMyPrinters:()=>{saves++;return {selectedIds:[],onboardingPending:false};},
  });
  await ready(app);
  assert.equal(visible(app.rowFor(prusaMk4.id)),true);
  assert.equal(app.rowFor(prusaMk4.id).querySelector('input').value,prusaMk4.id);
  assert.equal(saves,0);
});

test('Prusa rows and the generic PrusaSlicer row coexist and are told apart by ID',async()=>{
  const app=startApp({
    InstalledPrusaTargets:()=>({targets:[prusaMk4],rejected:[]}),
    GetMyPrinters:()=>({selectedIds:[prusaMk4.id,'prusa-project'],onboardingPending:false}),
  });
  await ready(app);
  assert.equal(visible(app.rowFor(prusaMk4.id)),true);
  assert.equal(visible(app.document.querySelectorAll('[data-prusa-target]')[0]),true);
  assert.equal(app.rows.u1.querySelector('input').value,'u1');
});

test('app.js reads the Prusa list only through the optional method and builds rows only from ready targets',()=>{
  const fs=require('node:fs');
  const app=fs.readFileSync(path.join(dist,'app.js'),'utf8');
  assert.match(app,/if\(!api\(\)\?\.InstalledPrusaTargets\)\{installedPrusaTargets=\[\];return;\}/);
  assert.match(app,/return \[\.\.\.installedTargets,\.\.\.installedPrusaTargets\.filter\(\(target\)=>target\?\.ready===true\)\];/);
  assert.match(app,/installedPrusaTargets\.find\(\(item\)=>item\.id===id\)\?\.title/);
});
