'use strict';
const {test}=require('node:test');
const assert=require('node:assert/strict');
const picker=require('../frontend/dist/printer-picker.js');
const {availabilityAll,loadMyPrinters}=require('./my-printers-harness.js');
const wait=()=>new Promise(resolve=>setImmediate(resolve));

function profile(id,vendor,model,nozzle,slicer,ready=true){
  return {id,vendor,manufacturer:vendor,model,title:vendor+' '+model+' '+nozzle+' nozzle',
    nozzle,slicer,ready,physicalTools:1};
}

test('same-nozzle default uses a verified installed manufacturer app across supported brands',()=>{
  const brands=[
    ['QIDI','X-Smart 5','QIDIStudio'],
    ['Flashforge','Creator 5 Pro','Flash Studio Desktop'],
    ['Snapmaker','Artisan','Snapmaker Orca'],
    ['ELEGOO','Neptune 5','ElegooSlicer'],
    ['Prusa Research','MK4','PrusaSlicer'],
  ];
  for(const [maker,model,brandApp] of brands){
    const slug=(maker+'-'+model).toLowerCase().replace(/[^a-z0-9]+/g,'-');
    const fallback=profile('orca-'+slug+'-aaaaaaaaaa',maker,model,'0.4','OrcaSlicer');
    const vendor=profile((brandApp==='PrusaSlicer'?'prusa-':'orca-')+slug+'-bbbbbbbbbb',maker,model,'0.4',brandApp);
    for(const order of [[fallback,vendor],[vendor,fallback]]){
      const group=picker.groupDestinations(order)[0];
      assert.equal(group.id,vendor.id,maker+': validated manufacturer profile must win');
      assert.deepEqual(group.ids.slice().sort(),[fallback.id,vendor.id].sort());
    }
  }
});

test('manufacturer profile on different nozzle cannot silently change the physical nozzle',()=>{
  const vendor=profile('orca-qidi-smart5-bbbbbbbbbb','QIDI','X-Smart 5','0.25','QIDIStudio');
  const fallback=profile('orca-qidi-smart5-aaaaaaaaaa','QIDI','X-Smart 5','0.4','OrcaSlicer');
  assert.equal(picker.groupDestinations([vendor,fallback])[0].id,fallback.id);
  assert.equal(picker.preferredSlicerVariant([vendor,fallback],'QIDI','0.4').id,fallback.id);
  assert.equal(picker.preferredSlicerVariant([vendor,fallback],'QIDI','0.6'),null);
});

test('incomplete manufacturer profile never beats a validated generic installed fallback',()=>{
  const broken=profile('orca-qidi-smart5-bbbbbbbbbb','QIDI','X-Smart 5','0.4','QIDIStudio',false);
  const fallback=profile('orca-qidi-smart5-aaaaaaaaaa','QIDI','X-Smart 5','0.4','OrcaSlicer',true);
  const group=picker.groupDestinations([broken,fallback])[0];
  assert.equal(group.id,fallback.id);
  assert.deepEqual(group.ids.slice().sort(),[broken.id,fallback.id].sort());
});

test('QIDIStudio X 5 Series is one QIDI physical printer with the original exact IDs',()=>{
  const id='orca-x5series-smart5-aaaaaaaaaa';
  const vendor={id,vendor:'X 5 Series',title:'QIDI X-Smart 5 0.4 nozzle',
    model:'X-Smart 5',nozzle:'0.4',slicer:'QIDIStudio',physicalTools:1};
  const generic={id:'orca-qidi-smart5-bbbbbbbbbb',vendor:'QIDI',
    title:'QIDI X-Smart 5 0.4 nozzle',model:'X-Smart 5',
    nozzle:'0.4',slicer:'OrcaSlicer',physicalTools:1};
  const items=picker.supportedModels([],[],[vendor,generic],false);
  const groups=picker.groupDestinations(items);
  assert.equal(groups.length,1);
  assert.equal(groups[0].manufacturer,'QIDI');
  assert.equal(groups[0].model,'X-Smart 5');
  assert.equal(groups[0].id,id);
  assert.equal(picker.makerLabel('X 5 Series'),'QIDI');
  assert.deepEqual(groups[0].ids.slice().sort(),[id,generic.id].sort());
  assert.deepEqual(picker.manufacturerNames(items),['QIDI']);
});

test('manual Orca override for one selected printer remains saved unchanged',async()=>{
  const vendor=profile('orca-flashforge-creator-5-pro-aaaaaaaaaa',
    'Flashforge','Creator 5 Pro','0.4','Flash Studio Desktop');
  const manual=profile('orca-flashforge-creator-5-pro-bbbbbbbbbb',
    'Flashforge','Creator 5 Pro','0.4','OrcaSlicer');
  const selected=['elegoo-centauri-carbon-2','flashforge-creator-5',
    'qidi-x-plus-5','snapmaker-u1',manual.id];
  let persisted=null;
  const h=loadMyPrinters({
    GetMyPrinters:()=>({selectedIds:selected,onboardingPending:false}),
    SaveMyPrinters:ids=>{persisted=[...ids];return {selectedIds:ids,onboardingPending:false};}
  });
  h.api.setSupportedTargets({available:availabilityAll(),installedTargets:[vendor,manual],prusaAvailable:false});
  await h.api.init();
  h.dom('myPrintersBtn').fire('click');
  assert.equal(h.dom('myPrintersSelectionCount').textContent,'5 printers selected');
  const card=h.dom('catalogModelList').children.find(card=>
    card.dataset.printerKey==='flashforge|creator5pro');
  assert.ok(card?.children[0].checked);
  assert.equal(card.querySelectorAll('.catalog-model-profile-button').length,0);
  const pill=h.dom('selectedPrinterSummary').children.find(row=>
    row.children[0]?.textContent==='Flashforge Creator 5 Pro');
  const button=pill.children.find(child=>child.classList.contains('selected-printer-settings'));
  assert.ok(button);
  button.fire('click');
  const radios=h.dom('printerProfileOptions').children.map(el=>el.children[0]);
  assert.equal(radios.length,2);
  assert.equal(radios.find(r=>r.value===manual.id).checked,true);
  h.dom('cancelPrinterProfileBtn').fire('click');
  h.dom('saveMyPrintersBtn').fire('click');
  await wait();
  assert.deepEqual(persisted,selected);
});

test('only one compatible slicer means no extra button on a selected-printer chip',async()=>{
  const installed=profile('orca-qidi-smart5-aaaaaaaaaa','QIDI','X-Smart 5','0.4','QIDIStudio');
  const otherNozzle=profile('orca-qidi-smart5-bbbbbbbbbb','QIDI','X-Smart 5','0.25','OrcaSlicer');
  const h=loadMyPrinters({GetMyPrinters:()=>({selectedIds:[installed.id],onboardingPending:false})});
  h.api.setSupportedTargets({available:[],installedTargets:[installed,otherNozzle],prusaAvailable:false});
  await h.api.init();
  h.dom('myPrintersBtn').fire('click');
  const pill=h.dom('selectedPrinterSummary').children[0];
  assert.equal(pill.children.some(child=>child.classList.contains('selected-printer-settings')),false);
});
