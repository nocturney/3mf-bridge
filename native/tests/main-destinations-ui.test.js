'use strict';
// Integration: the real printer-picker.js, my-printers.js and app.js run together
// in one vm window. The target list matches index.html: four built-in rows and the
// PrusaSlicer row, all hidden until My Printers and the backend allow them.
const {test}=require('node:test');
const assert=require('node:assert/strict');
const path=require('node:path');
const {availabilityAll,dist,FakeEvent,createDocument,createWindow,recordingApp,runScripts}=require('./my-printers-harness.js');

const orcaId='orca-anker-make-m5-0123456789';
const otherOrcaId='orca-anker-make-m7-1123456789';
const orca={id:orcaId,title:'Anker Make M5',vendor:'Anker',model:'Make M5',nozzle:'0.4',physicalTools:1,materials:['PLA']};
const otherOrca={id:otherOrcaId,title:'Anker Make M7',vendor:'Anker',model:'Make M7',nozzle:'0.4',physicalTools:1,materials:['PLA']};
const builtIn={u1:'snapmaker-u1',c5:'flashforge-creator-5',cc2:'elegoo-centauri-carbon-2',qidi:'qidi-x-plus-5'};
const targets=['u1','c5','cc2','qidi'];

async function settle(){for(let i=0;i<30;i++)await new Promise(resolve=>setImmediate(resolve));}
// The production app starts initializeNative() on a zero-delay timer after
// deferred frontend modules register. Repeated setImmediate calls can finish
// before that timer runs (especially on Windows), so UI tests must wait for the
// actual backend/selection handshake, not an arbitrary number of event ticks.
async function ready(app){
  for(let attempt=0;attempt<100&&!app.window.SliceBridgeMyPrinters?.isReady();attempt++)
    await new Promise(resolve=>setTimeout(resolve,2));
  assert.equal(app.window.SliceBridgeMyPrinters?.isReady(),true,'native My Printers initialization did not finish');
  await settle();
}

// index.html's target list as shipped: built-in rows and the PrusaSlicer row, all hidden.
function seedTargetList(document){
  const list=document.getElementById('targetList');
  const rows={};
  for(const key of targets){
    const row=document.createElement('label');
    row.className='target-option';row.dataset.targetRow=key;row.hidden=true;
    const input=document.createElement('input');input.setAttribute('type','checkbox');input.value=key;
    const summary=document.createElement('small');summary.dataset.printerSummary=key;
    const label=document.createElement('span');label.className='printer-label';label.append(summary);
    row.append(input,label);list.append(row);
    rows[key]=row;
  }
  const prusa=document.createElement('label');
  prusa.className='target-option';prusa.dataset.prusaTarget='';prusa.hidden=true;
  const prusaInput=document.createElement('input');prusaInput.setAttribute('type','checkbox');
  prusaInput.value='prusa-project';
  prusa.append(prusaInput);list.append(prusa);
  rows.prusa=prusa;
  // The file queue sits in a panel; renderQueue toggles that panel's class.
  const panel=document.createElement('section');panel.append(document.getElementById('fileList'));
  return rows;
}

function startApp(config={}){
  const document=createDocument();
  const rows=seedTargetList(document);
  const {wailsApp,calls}=recordingApp({
    GetSettings:()=>({version:1,window:{width:1100,height:750},
      printers:Object.fromEntries(targets.map(target=>[target,{nozzle:'auto',slicer:'auto',filaments:{}}]))}),
    GetVersion:()=>({version:'0.4.0',platform:'darwin'}),
    Availability:()=>availabilityAll(),
    InstalledOrcaTargets:()=>({targets:[],rejected:[]}),
    PrusaEngineStatus:()=>({available:false,engine:'prusa-slicer'}),
    ...config,
  });
  const window=createWindow(wailsApp);
  const context={
    window,document,Event:FakeEvent,
    navigator:{language:'en-US'},
    localStorage:{getItem:()=>null,setItem(){}},
    setTimeout,clearTimeout,console,
  };
  // Same order as index.html, where app.js runs before the deferred my-printers.js.
  runScripts([
    path.join(dist,'printer-picker.js'),
    path.join(dist,'app.js'),
    path.join(dist,'my-printers.js'),
  ],context);
  return {
    window,document,rows,calls,
    dom:(id)=>document.getElementById(id),
    card:(id)=>document.getElementById('catalogModelList').children.find(c=>c.children[0].value===id)?.children[0],
  };
}

const visible=(row)=>!row.hidden;
const checked=(row)=>row.querySelector('input').checked;

test('built-in, installed and PrusaSlicer rows appear only when My Printers selects a supported destination',async()=>{
  const app=startApp({
    Availability:()=>availabilityAll({c5:false}),
    InstalledOrcaTargets:()=>({targets:[orca,otherOrca,orca],rejected:[]}),
    PrusaEngineStatus:()=>({available:true,engine:'prusa-slicer'}),
    GetMyPrinters:()=>({selectedIds:[builtIn.u1,builtIn.c5,orcaId,'prusa-project','orca-stale-0123456789'],onboardingPending:false}),
  });
  await ready(app);
  const {rows}=app;
  assert.equal(visible(rows.u1),true);
  assert.equal(checked(rows.u1),true);
  // Saved but unavailable: no row at all.
  assert.equal(visible(rows.c5),false);
  assert.equal(rows.c5.querySelector('input').checked,false);
  assert.equal(visible(rows.cc2),false);
  assert.equal(visible(rows.qidi),false);
  // Installed Orca: only the selected target appears, once, checked.
  const orcaRows=app.document.querySelectorAll('[data-installed-target="'+orcaId+'"]');
  assert.equal(orcaRows.length,1);
  assert.equal(visible(orcaRows[0]),true);
  assert.equal(checked(orcaRows[0]),true);
  const otherRows=app.document.querySelectorAll('[data-installed-target="'+otherOrcaId+'"]');
  assert.equal(visible(otherRows[0]),false);
  assert.equal(checked(otherRows[0]),false);
  // PrusaSlicer row: probed and selected, so shown and checked.
  assert.equal(visible(rows.prusa),true);
  assert.equal(checked(rows.prusa),true);
  assert.equal(app.dom('selectAllBtn').hidden,false);
  assert.equal(app.dom('emptyTargetNotice').hidden,true);
  assert.equal(app.document.querySelectorAll('.unavailable').length,0);
});

test('a PrusaSlicer row stays hidden without a probe, even when the saved selection names it',async()=>{
  const app=startApp({
    GetMyPrinters:()=>({selectedIds:['prusa-project'],onboardingPending:false}),
    PrusaEngineStatus:()=>({available:false,engine:'prusa-slicer'}),
  });
  await ready(app);
  assert.equal(visible(app.rows.prusa),false);
  assert.equal(checked(app.rows.prusa),false);
});

test('saving in My Printers hides and unchecks deselected rows and keeps the others as they were',async()=>{
  const app=startApp({
    InstalledOrcaTargets:()=>({targets:[orca],rejected:[]}),
    PrusaEngineStatus:()=>({available:true,engine:'prusa-slicer'}),
    GetMyPrinters:()=>({selectedIds:[builtIn.u1,orcaId,'prusa-project'],onboardingPending:false}),
    SaveMyPrinters:(requested)=>({selectedIds:requested,onboardingPending:false}),
  });
  await ready(app);
  // The user unchecks the Orca row on the main list; that choice is kept until the next save.
  const orcaRow=app.document.querySelectorAll('[data-installed-target="'+orcaId+'"]')[0];
  orcaRow.querySelector('input').checked=false;
  orcaRow.querySelector('input').fire('change');
  app.dom('myPrintersBtn').fire('click');
  app.dom('printerCatalogSearch').value='snapmaker';
  app.dom('printerCatalogSearch').fire('input');
  app.card(builtIn.u1).checked=false;
  app.card(builtIn.u1).fire('change');
  app.dom('saveMyPrintersBtn').fire('click');
  await settle();
  assert.equal(visible(app.rows.u1),false);
  assert.equal(checked(app.rows.u1),false);
  assert.equal(visible(orcaRow),true);
  assert.equal(checked(orcaRow),false);
  assert.equal(visible(app.rows.prusa),true);
  assert.equal(checked(app.rows.prusa),true);
  assert.equal(app.dom('selectAllBtn').hidden,false,'remaining Orca and Prusa rows can be selected as a group');
});

test('a destination newly selected in My Printers appears checked after save',async()=>{
  const app=startApp({
    GetMyPrinters:()=>({selectedIds:[],onboardingPending:false}),
    SaveMyPrinters:(requested)=>({selectedIds:requested,onboardingPending:false}),
  });
  await ready(app);
  assert.equal(visible(app.rows.cc2),false);
  assert.equal(app.dom('emptyTargetNotice').hidden,false);
  app.dom('myPrintersBtn').fire('click');
  app.dom('printerCatalogSearch').value='elegoo';
  app.dom('printerCatalogSearch').fire('input');
  app.card(builtIn.cc2).checked=true;
  app.card(builtIn.cc2).fire('change');
  app.dom('saveMyPrintersBtn').fire('click');
  await settle();
  assert.equal(visible(app.rows.cc2),true);
  assert.equal(checked(app.rows.cc2),true);
  assert.equal(app.dom('emptyTargetNotice').hidden,true);
});

test('select-all and conversion honour only visible, enabled rows',async()=>{
  const sent=[];
  const app=startApp({
    Availability:()=>availabilityAll({cc2:false,qidi:false}),
    InstalledOrcaTargets:()=>({targets:[orca],rejected:[]}),
    PrusaEngineStatus:()=>({available:false,engine:'prusa-slicer'}),
    GetMyPrinters:()=>({selectedIds:[builtIn.u1,builtIn.c5,orcaId],onboardingPending:false}),
    SelectFiles:()=>[{path:'C:/models/a.3mf',name:'a.3mf',folder:'C:/models',sizeLabel:'1 KB'}],
    ConvertBatch:(request)=>{sent.push(request.targets);return {filesDone:1,successes:0,failures:0,files:[]};},
  });
  await ready(app);
  // Select all now includes every visible installed profile as well as built-ins.
  const orcaRow=app.document.querySelectorAll('[data-installed-target="'+orcaId+'"]')[0];
  assert.equal(checked(orcaRow),true);
  app.dom('selectAllBtn').fire('click');
  assert.equal(checked(app.rows.u1),false);
  assert.equal(checked(app.rows.c5),false);
  assert.equal(checked(orcaRow),false);
  app.dom('selectAllBtn').fire('click');
  assert.equal(checked(app.rows.u1),true);
  assert.equal(checked(app.rows.c5),true);
  assert.equal(checked(orcaRow),true);
  // A checked box on a hidden row, set after the last render, must still not convert.
  app.rows.cc2.querySelector('input').checked=true;
  app.rows.prusa.querySelector('input').checked=true;

  app.dom('addFilesBtn').fire('click');
  await settle();
  app.dom('convertBtn').fire('click');
  await settle();
  assert.equal(sent.length,1);
  assert.deepEqual([...sent[0]].sort(),['c5','u1',orcaId].sort());
});

test('readiness handshake waits for a delayed native response',async()=>{
  const app=startApp({
    GetMyPrinters:()=>new Promise(resolve=>setTimeout(
      ()=>resolve({selectedIds:[builtIn.c5],onboardingPending:false}),45)),
  });
  assert.equal(app.window.SliceBridgeMyPrinters?.isReady(),false);
  await ready(app);
  assert.equal(visible(app.rows.c5),true);
  assert.equal(checked(app.rows.c5),true);
});

test('with no selection the main list says so and conversion stays disabled',async()=>{
  const app=startApp({GetMyPrinters:()=>({selectedIds:[],onboardingPending:false})});
  await ready(app);
  assert.equal(app.dom('emptyTargetNotice').hidden,false);
  assert.equal(app.dom('convertBtn').disabled,true);
  assert.equal(app.dom('selectAllBtn').hidden,true);
});


test('real five-printer setup: Creator 5 Pro 0.25 is checked, Select all toggles all five, no technical subtitles',async()=>{
  const proId='orca-flashforge-creator-5-pro-025-9988776655';
  const pro={id:proId,vendor:'Flashforge',model:'Creator 5 Pro',
    title:'Flashforge Creator 5 Pro 0.25 nozzle',nozzle:'0.25',
    slicer:'Flash Studio Desktop',physicalTools:1,materials:['PLA']};
  const saved=[builtIn.cc2,builtIn.c5,builtIn.qidi,builtIn.u1,proId];
  const sent=[];
  const app=startApp({
    InstalledOrcaTargets:()=>({targets:[pro],rejected:[]}),
    GetMyPrinters:()=>({selectedIds:saved,onboardingPending:false}),
    SaveMyPrinters:ids=>({selectedIds:ids,onboardingPending:false}),
    SelectFiles:()=>[{path:'C:/models/a.3mf',name:'a.3mf',folder:'C:/models',sizeLabel:'1 KB'}],
    ConvertBatch:request=>{sent.push(request.targets);return {filesDone:1,successes:0,failures:0,files:[]};},
  });
  await ready(app);
  const rows=[...targets.map(t=>app.rows[t]),...app.document.querySelectorAll('[data-installed-target="'+proId+'"]')];
  assert.equal(rows.length,5);
  assert.ok(rows.every(row=>visible(row)&&checked(row)), 'all five saved targets must start checked');
  const installed=rows[4];
  assert.equal(installed.querySelector('strong').textContent,'Flashforge Creator 5 Pro');
  assert.equal(installed.querySelector('small'),null);
  assert.match(installed.title,/0.25 mm · Flash Studio Desktop/);
  assert.equal(app.dom('selectAllBtn').hidden,false);
  app.dom('selectAllBtn').fire('click');
  assert.ok(rows.every(row=>!checked(row)),'Select all clears all FIVE checked rows');
  app.dom('selectAllBtn').fire('click');
  assert.ok(rows.every(row=>checked(row)),'Select all restores all FIVE including Creator 5 Pro');
  app.dom('addFilesBtn').fire('click');
  await settle();
  app.dom('convertBtn').fire('click');
  await settle();
  assert.equal(sent.length,1);
  assert.deepEqual([...sent[0]].sort(),['cc2','c5','qidi','u1',proId].sort());
});

test('four-slot single-tool conversion warns outside collapsed details instead of claiming print-ready success',async()=>{
  const source={path:'C:/qa/four-colours.3mf',name:'four-colours.3mf',
    folder:'C:/qa',sizeLabel:'1 KB',nozzle:'0.4',objects:1,instances:1,plates:1,
    filaments:[
      {type:'PLA',color:'#FF0000'},
      {type:'PLA',color:'#00FF00'},
      {type:'PLA',color:'#0000FF'},
      {type:'PLA',color:'#FFFFFF'},
    ]};
  async function perform(mappingRequired){
    const app=startApp({
      InstalledOrcaTargets:()=>({targets:[orca],rejected:[]}),
      GetMyPrinters:()=>({selectedIds:[orcaId],onboardingPending:false}),
      SelectFiles:()=>[source],
      ConvertBatch:()=>({
        filesDone:1,successes:1,failures:0,error:'',
        files:[{source,failures:[],results:[{
          target:orcaId,ok:true,outputName:'four-converted.3mf',outputPath:'C:/qa/four-converted.3mf',
          report:{source_filament_slots:4,target_filament_slots:4,virtual_filament_slots:4,
            physical_toolheads:1,target_nozzle:'0.4',user_slot_mapping_required:mappingRequired,
            geometry_triangles:12,removed_slice_cache_entries:0},
        }]}],
      }),
    });
    await ready(app);
    app.dom('addFilesBtn').fire('click');await settle();
    app.dom('convertBtn').fire('click');await settle();
    return app;
  }
  const review=await perform(true);
  const row=review.dom('resultList').querySelector('.result-row');
  assert.ok(row,'conversion result must be visible');
  const pill=row.querySelector('.status-pill');
  assert.equal(pill.classList.contains('review'),true);
  assert.equal(pill.textContent,'Review mapping');
  assert.equal(pill.classList.contains('good'),false,
    'cannot claim ordinary green success when physical tool mapping needs review');
  const warning=row.querySelector('.material-mapping-warning');
  assert.ok(warning,'manual material mapping caution must be visible on result card');
  assert.equal(warning.closest('details'),null,'warning cannot live in collapsed details');
  assert.equal(warning.hidden,false);
  assert.match(warning.textContent,/All 4 source color\/material/);
  const detail=row.querySelector('.report-details');
  assert.ok(detail,'diagnostic details remain accessible');
  assert.equal(detail.open,false,'even closed diagnostic panel must not hide warning');

  const ordinary=await perform(false);
  const ordinaryRow=ordinary.dom('resultList').querySelector('.result-row');
  assert.equal(ordinaryRow.querySelector('.status-pill').classList.contains('good'),true);
  assert.equal(ordinaryRow.querySelector('.material-mapping-warning'),null,
    'ordinary one-to-one conversion should not get a false material warning');
});
