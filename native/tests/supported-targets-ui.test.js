'use strict';
const {test}=require('node:test');
const assert=require('node:assert/strict');
const picker=require('../frontend/dist/printer-picker.js');
const {loadMyPrinters,availabilityAll,catalog}=require('./my-printers-harness.js');

const builtIn={u1:'snapmaker-u1',c5:'flashforge-creator-5',cc2:'elegoo-centauri-carbon-2',qidi:'qidi-x-plus-5'};
const orcaId='orca-anker-make-m5-0123456789';
const orca={id:orcaId,title:'Anker Make M5',vendor:'Anker',model:'Make M5',nozzle:'0.4',physicalTools:1};
const legacyIds=new Set(Object.values(builtIn));
const nonLegacy=catalog.models.find(model=>!model.legacyTarget&&!picker.isHidden(model));
const tick=()=>new Promise(resolve=>setImmediate(resolve));
const ids=(items)=>items.map(item=>item.id);
const plain=(value)=>JSON.parse(JSON.stringify(value));

test('built-in models are listed only when their native Availability is true',()=>{
  const all=plain(picker.supportedModels(catalog.models,availabilityAll(),[],false));
  assert.deepEqual(ids(all).sort(),Object.values(builtIn).sort());
  const some=plain(picker.supportedModels(catalog.models,availabilityAll({c5:false,qidi:false}),[],false));
  assert.deepEqual(ids(some).sort(),[builtIn.u1,builtIn.cc2].sort());
  const none=plain(picker.supportedModels(catalog.models,availabilityAll({u1:false,c5:false,cc2:false,qidi:false}),[],false));
  assert.deepEqual(none,[]);
});

test('availability must be a real boolean true, not a truthy string or a missing entry',()=>{
  const strings=[{target:'u1',available:'true'},{target:'c5',available:1},{target:'cc2'}];
  assert.deepEqual(picker.supportedModels(catalog.models,strings,[],false),[]);
  assert.deepEqual(picker.supportedModels(catalog.models,null,[],false),[]);
  assert.deepEqual(picker.supportedModels(catalog.models,[null,{available:true}],[],false),[]);
});

test('catalogue-only models never become destinations, even when every adapter is available',()=>{
  const all=picker.supportedModels(catalog.models,availabilityAll(),[],true);
  assert.ok(nonLegacy,'fixture needs a catalogue-only model');
  assert.equal(all.some(item=>item.id===nonLegacy.id),false);
  for(const item of all){
    if(item.legacyTarget)assert.ok(legacyIds.has(item.id),item.id);
  }
});

test('each supported item has exactly the documented fields and no status, tag or path data',()=>{
  const [first]=picker.supportedModels(catalog.models,availabilityAll(),[orca],true);
  assert.deepEqual(Object.keys(first).sort(),['aliases','id','legacyTarget','manufacturer','model','nozzle','releaseDate','slicer']);
  for(const item of picker.supportedModels(catalog.models,availabilityAll(),[orca],true)){
    assert.deepEqual(Object.keys(item).sort(),['aliases','id','legacyTarget','manufacturer','model','nozzle','releaseDate','slicer'],item.id);
    assert.ok(Array.isArray(item.aliases));
    for(const value of [item.manufacturer,item.model,...item.aliases]){
      assert.equal(typeof value,'string');
      assert.ok(value.length<=80,item.id);
    }
  }
});

test('installed Orca targets outside the 603-model catalogue are listed by backend id',()=>{
  const items=plain(picker.supportedModels(catalog.models,[],[orca],false));
  assert.deepEqual(items,[{id:orcaId,manufacturer:'Anker',model:'Make M5',aliases:[],legacyTarget:null,nozzle:'0.4',slicer:null,releaseDate:null}]);
  assert.equal(catalog.models.some(model=>model.id===orcaId),false);
});

test('vendor repeated in the installed title is not shown twice',()=>{
  const [item]=picker.supportedModels(catalog.models,[],[{
    id:'orca-snapmaker-u1-custom-a1b2c3d4e5',title:'Snapmaker U1',vendor:'Snapmaker',model:'U1',
  }],false);
  assert.equal(item.manufacturer,'Snapmaker');
  assert.equal(item.model,'U1');
  assert.equal(picker.title(item),'Snapmaker U1');
});

test('an installed slicer or machine without a resolved backend id is never a destination',()=>{
  const unresolved=[
    {title:'Creality K1',vendor:'Creality',model:'K1'},
    {id:'',title:'Creality K1',vendor:'Creality',model:'K1'},
    {id:'creality-k1',title:'Creality K1',vendor:'Creality',model:'K1'},
    null,
    'orca-anker-make-m5-0123456789',
  ];
  assert.deepEqual(picker.supportedModels(catalog.models,[],unresolved,false),[]);
});

test('unsafe or malformed backend target ids are rejected',()=>{
  const bad=[
    'orca-x',
    'ORCA-Anker-Make-M5-0123456789',
    'orca-../../etc-0123456789',
    '/Users/chris/Library/orca-anker-0123456789',
    'orca-anker-make-m5-0123456789;rm',
    'orca-anker-make-m5-0123456789 ',
    'orca-anker-make-m5-0123456789\n<img src=x>',
    'orca-anker-make-m5-012345678G',
    'orca-anker-make-m5-01234567890',
  ];
  const items=picker.supportedModels(catalog.models,[],bad.map(id=>({id,title:'X',vendor:'Anker'})),false);
  assert.deepEqual(items,[]);
});

test('duplicate backend ids produce one destination',()=>{
  const items=picker.supportedModels(catalog.models,[],[orca,{...orca,title:'Again'}],false);
  assert.deepEqual(ids(items),[orcaId]);
  const doubled=picker.supportedModels(catalog.models,availabilityAll(),[],false);
  assert.equal(new Set(ids(doubled)).size,doubled.length);
});

test('Bambu is never a destination, whether built in, by vendor code or by title',()=>{
  const bambu=[
    {id:'orca-bbl-x1c-aaaaaaaaaa',title:'Bambu Lab X1 Carbon',vendor:'BBL',model:'X1C'},
    {id:'orca-bbl-p1s-bbbbbbbbbb',title:'P1S',vendor:'BBL',model:'P1S'},
    {id:'orca-other-bambu-cccccccccc',title:'Bambu Lab A1',vendor:'Bambu Lab',model:'A1'},
  ];
  const everything=picker.supportedModels(catalog.models,availabilityAll(),[...bambu,orca],true);
  assert.equal(everything.some(item=>/bambu|^bbl$/i.test(item.manufacturer)),false);
  assert.equal(everything.some(item=>/bambu/i.test(item.model)),false);
  assert.equal(everything.some(item=>item.id.startsWith('orca-bbl')),false);
  assert.equal(picker.search(everything,'bambu').total,0);
  assert.equal(catalog.models.some(model=>model.manufacturer==='Bambu Lab'),true);
});

test('PrusaSlicer project is listed only when the engine probe is available',()=>{
  const on=picker.supportedModels(catalog.models,[],[],true);
  // PrusaSlicer is a generic export target, not a physical maker.
  assert.deepEqual(plain(on),[{id:'prusa-project',manufacturer:'Generic',model:'PrusaSlicer project',aliases:['Generic project'],legacyTarget:null,nozzle:null,slicer:null,releaseDate:null}]);
  for(const value of [false,undefined,'true',1,null]){
    assert.deepEqual(picker.supportedModels(catalog.models,[],[],value),[],String(value));
  }
});

test('search finds dynamic and PrusaSlicer destinations, not unsupported models',()=>{
  const items=picker.supportedModels(catalog.models,availabilityAll({qidi:false}),[orca],true);
  assert.ok(picker.search(items,'anker').models.some(item=>item.id===orcaId));
  assert.ok(picker.search(items,'generic project').models.some(item=>item.id==='prusa-project'));
  assert.equal(picker.search(items,'qidi plus5').total,0);
  assert.equal(picker.search(items,'nonexistent printer 999999').total,0);
});

test('selected destinations keep saved order, drop stale IDs and cap at 32',()=>{
  const items=picker.supportedModels(catalog.models,availabilityAll(),[orca],true);
  const stale=[nonLegacy.id,'orca-gone-0123456789','../../x',7,null,orcaId,builtIn.u1,builtIn.u1];
  assert.deepEqual(picker.selectedTargetIds(items,stale),[orcaId,builtIn.u1]);
  const many=Array.from({length:40},(_,i)=>({
    id:'orca-test-model-'+String(i).padStart(2,'0')+'-0123456789',title:'Model '+i,vendor:'Test',model:'M'+i,
  }));
  const big=picker.supportedModels(catalog.models,[],many,false);
  assert.equal(picker.selectedTargetIds(big,big.map(item=>item.id)).length,picker.maxSelected);
  assert.equal(picker.maxSelected,32);
});

test('legacy keys come only from selected built-in destinations',()=>{
  const items=picker.supportedModels(catalog.models,availabilityAll(),[orca],true);
  const picked=[orcaId,builtIn.c5,'prusa-project',nonLegacy.id,builtIn.u1];
  assert.deepEqual(picker.legacyTargetsFor(items,picked).sort(),['c5','u1']);
});

test('My Printers lists only supported destinations, with stale saved IDs hidden and never sent back',async()=>{
  const saved=[nonLegacy.id,'orca-gone-0123456789',builtIn.u1];
  const h=loadMyPrinters({
    GetMyPrinters:()=>({selectedIds:saved,onboardingPending:false}),
    SaveMyPrinters:(requested)=>({selectedIds:requested,onboardingPending:false}),
  });
  h.api.setSupportedTargets({available:availabilityAll({c5:false,cc2:false,qidi:false}),installedTargets:[],prusaAvailable:false});
  await h.api.init();
  assert.deepEqual(h.plain(h.api.selectedTargetIds()),[builtIn.u1]);
  assert.equal(h.api.isSelectedTarget('u1'),true);
  assert.equal(h.api.isSelectedTarget(nonLegacy.id),false);
  assert.equal(h.api.isSelectedTarget('orca-gone-0123456789'),false);

  h.dom('myPrintersBtn').fire('click');
  assert.equal(h.dom('myPrintersDialog').open,true);
  assert.equal(h.dom('selectedPrinterSummary').children.length,1);
  h.dom('saveMyPrintersBtn').fire('click');
  await tick();
  const save=h.calls.filter(call=>call==='SaveMyPrinters');
  assert.equal(save.length,1);
  assert.equal(h.recorded.events.includes('slicebridge:my-printers-changed'),true);
  assert.deepEqual(h.plain(h.api.selectedTargetIds()),[builtIn.u1]);
});

test('save sends only the visible selection, including dynamic and PrusaSlicer destinations',async()=>{
  let sent=null;
  const h=loadMyPrinters({
    GetMyPrinters:()=>({selectedIds:[],onboardingPending:false}),
    SaveMyPrinters:(requested)=>{sent=requested;return {selectedIds:requested,onboardingPending:false};},
  });
  h.api.setSupportedTargets({available:availabilityAll({c5:false,cc2:false,qidi:false}),installedTargets:[orca],prusaAvailable:true});
  await h.api.init();
  h.dom('myPrintersBtn').fire('click');
  h.dom('printerManufacturerFilter').value='Anker';
  h.dom('printerManufacturerFilter').fire('change');
  const cards=h.dom('catalogModelList').children;
  assert.equal(cards.length,1);
  assert.equal(cards[0].children[0].value,orcaId);
  cards[0].children[0].checked=true;
  cards[0].children[0].fire('change');
  h.dom('printerManufacturerFilter').value='';
  h.dom('printerManufacturerFilter').fire('change');
  h.dom('printerCatalogSearch').value='prusa';
  h.dom('printerCatalogSearch').fire('input');
  const prusa=h.dom('catalogModelList').children.find(card=>card.children[0].value==='prusa-project');
  prusa.children[0].checked=true;
  prusa.children[0].fire('change');
  h.dom('saveMyPrintersBtn').fire('click');
  await tick();
  assert.deepEqual(h.plain(sent),[orcaId,'prusa-project']);
  assert.equal(h.api.isSelectedTarget(orcaId),true);
  assert.equal(h.api.isSelectedTarget('prusa-project'),true);
  assert.equal(h.api.isSelectedTarget('u1'),false);
});

test('an empty supported result shows one honest line and no unsupported lists',()=>{
  const h=loadMyPrinters({GetMyPrinters:()=>({selectedIds:[builtIn.u1],onboardingPending:false})});
  h.api.setSupportedTargets({available:availabilityAll({u1:false,c5:false,cc2:false,qidi:false}),installedTargets:[],prusaAvailable:false});
  return h.api.init().then(()=>{
    h.dom('myPrintersBtn').fire('click');
    assert.equal(h.dom('catalogPickerHint').textContent,'No destinations are available on this computer yet.');
    assert.equal(h.dom('catalogModelList').children.length,0);
    assert.equal(h.dom('saveMyPrintersBtn').disabled,true);
    assert.equal(h.dom('printerCatalogSearch').disabled,true);
    assert.equal(h.dom('printerManufacturerFilter').disabled,true);
    assert.equal(h.dom('printerManufacturerFilter').children.length,1);
    assert.equal(h.api.isReady(),true);
  });
});

test('manufacturer dropdown offers only makers of supported destinations',async()=>{
  const h=loadMyPrinters({});
  h.api.setSupportedTargets({available:availabilityAll({c5:false,cc2:false,qidi:false}),installedTargets:[orca],prusaAvailable:false});
  await h.api.init();
  const options=h.dom('printerManufacturerFilter').children;
  assert.deepEqual(options.map(option=>option.value),['','Snapmaker','','Anker']);
  assert.equal(options[2].disabled,true,'separator must not be a selectable maker');
  assert.match(options[2].textContent,/─/);
});

test('losing availability trims an open draft and removes the destination everywhere',async()=>{
  const h=loadMyPrinters({GetMyPrinters:()=>({selectedIds:[builtIn.u1,orcaId],onboardingPending:false})});
  h.api.setSupportedTargets({available:availabilityAll(),installedTargets:[orca],prusaAvailable:false});
  await h.api.init();
  assert.equal(h.api.isSelectedTarget('u1'),true);
  h.dom('myPrintersBtn').fire('click');
  h.api.setSupportedTargets({available:availabilityAll({u1:false}),installedTargets:[orca],prusaAvailable:false});
  assert.equal(h.api.isSelectedTarget('u1'),false);
  assert.equal(h.api.isSelectedTarget(orcaId),true);
  assert.deepEqual(h.plain(h.api.selectedTargetIds()),[orcaId]);
  assert.equal(h.recorded.events.filter(type=>type==='slicebridge:my-printers-changed').length>=2,true);
});

test('the 32-printer limit holds in the dialog, not only in the saved file',async()=>{
  const many=Array.from({length:33},(_,i)=>({
    id:'orca-test-model-'+String(i).padStart(2,'0')+'-0123456789',title:'Model '+i,vendor:'Test',model:'M'+i,
  }));
  const h=loadMyPrinters({GetMyPrinters:()=>({selectedIds:[],onboardingPending:false})});
  h.api.setSupportedTargets({available:[],installedTargets:many,prusaAvailable:false});
  await h.api.init();
  h.dom('myPrintersBtn').fire('click');
  h.dom('printerCatalogSearch').value='test';
  h.dom('printerCatalogSearch').fire('input');
  const card=(id)=>h.dom('catalogModelList').children.find(c=>c.children[0].value===id).children[0];
  for(const item of many.slice(0,32)){
    card(item.id).checked=true;
    card(item.id).fire('change');
  }
  assert.equal(h.dom('myPrintersSelectionCount').textContent,'32 printers selected');
  card(many[32].id).checked=true;
  card(many[32].id).fire('change');
  assert.equal(card(many[32].id).checked,false);
  assert.equal(h.dom('myPrintersStatus').textContent,'You can select up to 32 printers.');
  assert.equal(h.dom('myPrintersSelectionCount').textContent,'32 printers selected');
});

test('integrated manufacturers appear ahead of alphabetical others, with one disabled separator',()=>{
  const items=[
    {manufacturer:'Zortrax'},{manufacturer:'QIDI'},{manufacturer:'Anycubic'},
    {manufacturer:'Flashforge'},{manufacturer:'Snapmaker'},{manufacturer:'Creality'},
    {manufacturer:'Prusa'},{manufacturer:'ELEGOO'},{manufacturer:'Elegoo'},{manufacturer:'Qidi'},
    {manufacturer:'Generic'}
  ];
  const grouped=picker.manufacturerGroups(items);
  assert.deepEqual(grouped.preferred,['Anycubic','Creality','ELEGOO','Flashforge','Prusa Research','QIDI','Snapmaker']);
  assert.deepEqual(grouped.others,['Generic','Zortrax']);
  assert.deepEqual(picker.manufacturerGroups([]),{preferred:[],others:[]});
});

test('unselected printers sort by verified launch date, selected first in saved order',()=>{
  const targets=[
    {id:'orca-zzzzz-unknown-1111111111',vendor:'Zzzzz',model:'Unknown One',title:'Unknown One'},
    {id:'orca-qidi-plus5-2222222222',vendor:'QIDI',model:'Plus5',title:'QIDI Plus5'},
    {id:'orca-flashforge-creator-5-pro-3333333333',vendor:'Flashforge',model:'Creator 5 Pro',title:'Flashforge Creator 5 Pro'},
    {id:'orca-snapmaker-u1-4444444444',vendor:'Snapmaker',model:'U1',title:'Snapmaker U1'},
    {id:'orca-anycubic-old-5555555555',vendor:'Anycubic',model:'Unknown Two',title:'Unknown Two'},
  ];
  const items=picker.supportedModels(catalog.models,[],targets,false);
  const order=picker.search(items,'').models.map(item=>item.id);
  assert.deepEqual(order,[
    'orca-qidi-plus5-2222222222','orca-flashforge-creator-5-pro-3333333333',
    'orca-snapmaker-u1-4444444444','orca-anycubic-old-5555555555',
    'orca-zzzzz-unknown-1111111111'
  ]);
  assert.equal(items.find(x=>x.id==='orca-qidi-plus5-2222222222').releaseDate,'2026-08-05');
  assert.equal(items.find(x=>x.id==='orca-zzzzz-unknown-1111111111').releaseDate,null);
  const selected=['orca-zzzzz-unknown-1111111111','orca-snapmaker-u1-4444444444'];
  assert.deepEqual(picker.search(items,'',selected).models.map(item=>item.id).slice(0,2),selected);
});

test('unknown launch dates stay unknown and popular brands lead undated fallbacks',()=>{
  const list=[
    {id:'z',manufacturer:'Acme',model:'Mystery M300',aliases:[]},
    {id:'s',manufacturer:'Snapmaker',model:'Unknown New X',aliases:[]},
    {id:'p',manufacturer:'Prusa',model:'No Public Date',aliases:[]},
  ];
  assert.equal(picker.releaseDate(list[0]),null);
  assert.deepEqual(picker.search(list,'').models.map(x=>x.id),['p','s','z']);
  assert.equal(picker.releaseDate({manufacturer:'QIDI',model:'Plus5',releaseDate:'2025-12-01'}),'2025-12-01');
  assert.equal(picker.releaseDate({manufacturer:'QIDI',model:'Plus5',releaseDate:'yesterday'}),'2026-08-05');
});

test('four installed Creator 5 Pro nozzle variants share one card with distinct variant labels',()=>{
  const variants=['0.25','0.4','0.6','0.8'].map((nozzle,i)=>({
    id:'orca-flashforge-creator-5-pro-'+i+'-'+String(i).repeat(10),
    vendor:'Flashforge',model:'Creator 5 Pro',
    title:'Flashforge Creator 5 Pro '+nozzle+' nozzle',nozzle,physicalTools:4,
  }));
  const items=picker.supportedModels(catalog.models,[],variants,false);
  assert.equal(items.length,4);
  assert.equal(new Set(items.map(picker.variantText)).size,4);
  const [group]=picker.groupDestinations(items);
  assert.equal(picker.groupDestinations(items).length,1);
  assert.equal(group.model,'Creator 5 Pro');
  assert.deepEqual(group.ids,variants.map(v=>v.id));
  assert.deepEqual(group.variants.map(v=>v.nozzle),['0.25','0.4','0.6','0.8']);
  // The default ticked target is the 0.4 mm variant; every variant keeps its own ID.
  assert.equal(group.id,variants[1].id);
});
