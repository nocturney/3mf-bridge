'use strict';
// Regression tests for My Printers cards: canonical makers, one card per physical
// model with its variants, the five original selections, model-specific artwork,
// and no release-date text on cards.
const {test}=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const path=require('node:path');
const picker=require('../frontend/dist/printer-picker.js');
const {loadMyPrinters,availabilityAll,catalog,FakeElement,createDocument,createWindow,
       recordingApp,runScripts}=require('./my-printers-harness.js');

const dist=path.resolve(__dirname,'../frontend/dist');
const script=fs.readFileSync(path.join(dist,'my-printers.js'),'utf8');
const tick=()=>new Promise(resolve=>setTimeout(resolve,0));
const flush=async()=>{for(let i=0;i<6;i++)await tick();};

// Installed Orca targets, as the backend lists them: nozzle variants and a second slicer.
const pro=(nozzle,hex,slicer)=>({
  id:'orca-flashforge-creator-5-pro-'+nozzle.replace('.','')+'-'+hex,
  vendor:'Flashforge',model:'Creator 5 Pro',title:'Flashforge Creator 5 Pro '+nozzle+' nozzle',
  nozzle,physicalTools:1,slicer,
});
const proVariants=[
  pro('0.25','1111111111','OrcaSlicer'),
  pro('0.4','2222222222','OrcaSlicer'),
  pro('0.6','3333333333','OrcaSlicer'),
  pro('0.8','4444444444','OrcaSlicer'),
  pro('0.4','5555555555','Flash Studio'),
];
const oldIds={
  cc2:'elegoo-centauri-carbon-2',c5:'flashforge-creator-5',qidi:'qidi-x-plus-5',u1:'snapmaker-u1',
  pro:'orca-flashforge-creator-5-pro-04-2222222222',
};
const fiveSaved=[oldIds.cc2,oldIds.c5,oldIds.qidi,oldIds.u1,oldIds.pro];

// Collects all visible text under a fake DOM element, in order.
function textOf(el){
  if(typeof el==='string')return el;
  return [el.textContent||'',...el.children.map(textOf)].join(' ');
}
function cards(h){return h.dom('catalogModelList').children;}
function cardFor(h,model){return cards(h).find(card=>card.children[1]?.children?.[1]?.children?.[0]?.textContent===model);}
function modelOf(card){return card.children[1].children[1].children[0].textContent;}
function variantBoxes(card){
  const list=card.children.find(child=>child.classList.contains('catalog-variant-list'));
  return list?list.children.map(option=>option.children[0]):[];
}

// Same as loadMyPrinters, plus extra globals (fetch) and a PrinterArtwork stub.
function loadWithGlobals(app={},extra={}){
  const document=createDocument();
  const {wailsApp,calls}=recordingApp(app);
  const window=createWindow(wailsApp);
  window.SliceBridgePicker=picker;
  runScripts([path.join(dist,'my-printers.js')],{window,document,Event:class{constructor(type){this.type=type;}},
                                                  setTimeout,clearTimeout,...extra});
  return {api:window.SliceBridgeMyPrinters,dom:(id)=>document.getElementById(id),calls,
          plain:(value)=>JSON.parse(JSON.stringify(value))};
}

test('maker spellings collapse to one canonical maker and one dropdown entry',()=>{
  const installed=[
    pro('0.4','6666666666','OrcaSlicer'),
    {...pro('0.4','7777777777','OrcaSlicer'),id:'orca-elegoo-centauri-carbon-7777777777',vendor:'Elegoo',model:'Centauri Carbon',title:'Elegoo Centauri Carbon 0.4 nozzle'},
    {id:'orca-qidi-x-max-3-8888888888',vendor:'Qidi',model:'X-Max 3',title:'Qidi X-Max 3 0.4 nozzle',nozzle:'0.4'},
  ];
  const items=picker.supportedModels(catalog.models,availabilityAll(),installed,true);
  const {preferred,others}=picker.manufacturerGroups(items);
  assert.equal(preferred.filter(name=>/^elegoo$/i.test(name)).length,1);
  assert.equal(preferred.filter(name=>/^qidi$/i.test(name)).length,1);
  assert.ok(preferred.includes('ELEGOO'));
  assert.ok(preferred.includes('QIDI'));
  assert.equal(new Set([...preferred,...others].map(name=>name.toLowerCase())).size,
               preferred.length+others.length);
  // PrusaSlicer is a generic export target, never a physical maker.
  assert.equal([...preferred,...others].some(name=>/prusaslicer/i.test(name)),false);
  assert.equal(items.find(item=>item.id==='prusa-project').manufacturer,'Generic');
  assert.equal(picker.makerLabel('Prusa'),'Prusa Research');
  assert.equal(picker.makerLabel(' prusa research '),'Prusa Research');
  assert.equal(picker.makerKey('Creality'),picker.makerKey('  creality '));
});

test('integrated makers lead the dropdown, then one divider, then the rest alphabetically',async()=>{
  const h=loadMyPrinters({GetMyPrinters:()=>({selectedIds:[],onboardingPending:false})});
  h.api.setSupportedTargets({available:availabilityAll(),installedTargets:[...proVariants,
    {id:'orca-elegoo-centauri-carbon-9999999999',vendor:'Elegoo',model:'Centauri Carbon',title:'Elegoo Centauri Carbon 0.4 nozzle',nozzle:'0.4'},
    {id:'orca-zortrax-m200-0000000001',vendor:'Zortrax',model:'M200',title:'Zortrax M200 0.4 nozzle',nozzle:'0.4'}],prusaAvailable:true});
  await h.api.init();
  const options=h.dom('printerManufacturerFilter').children.map(option=>[option.value,option.disabled===true]);
  const values=options.map(([value])=>value);
  assert.deepEqual(values,['','ELEGOO','Flashforge','QIDI','Snapmaker','','Generic','Zortrax']);
  assert.deepEqual(options.filter(([,disabled])=>disabled).map(([value])=>value),['']);
  assert.equal(values.filter(value=>value==='ELEGOO').length,1);
});

test('a physical model is one card, however many nozzles and slicers it has',()=>{
  const groups=picker.groupDestinations(picker.supportedModels(catalog.models,[],proVariants,false));
  assert.equal(groups.length,1);
  assert.equal(groups[0].variants.length,5);
  assert.equal(groups[0].model,'Creator 5 Pro');
  // The default ticked target is a 0.4 mm one; choosing it never replaces another variant.
  assert.ok(proVariants.filter(v=>v.nozzle==='0.4').map(v=>v.id).includes(groups[0].id));
  assert.deepEqual(groups[0].ids.slice().sort(),proVariants.map(v=>v.id).sort());
});

test('Creator 5 and Creator 5 Pro, and Centauri Carbon and Centauri Carbon 2, stay distinct cards',()=>{
  const installed=[
    pro('0.4','2222222222','OrcaSlicer'),
    {id:'orca-elegoo-centauri-carbon-aaaaaaaaaa',vendor:'ELEGOO',model:'Centauri Carbon',title:'ELEGOO Centauri Carbon 0.4 nozzle',nozzle:'0.4'},
  ];
  const items=picker.supportedModels(catalog.models,availabilityAll(),installed,false);
  const groups=picker.groupDestinations(items);
  const names=groups.map(group=>group.model);
  assert.ok(names.includes('Creator 5'));
  assert.ok(names.includes('Creator 5 Pro'));
  assert.ok(names.includes('Centauri Carbon'));
  assert.ok(names.includes('Centauri Carbon 2'));
  assert.equal(groups.length,names.length);
  assert.equal(new Set(names).size,names.length);
  const c5=groups.find(group=>group.model==='Creator 5');
  assert.equal(c5.legacyTarget,'c5');
  assert.equal(c5.ids.some(id=>id.includes('creator-5-pro')),false);
});

test('the five original selections survive load, edit and save exactly, in saved order',async()=>{
  let sent=null;
  const h=loadWithGlobals({
    GetMyPrinters:()=>({selectedIds:fiveSaved,onboardingPending:false}),
    SaveMyPrinters:(requested)=>{sent=requested;return {selectedIds:requested,onboardingPending:false};},
  });
  h.api.setSupportedTargets({available:availabilityAll(),installedTargets:proVariants,prusaAvailable:false});
  await h.api.init();
  assert.deepEqual(h.plain(h.api.selectedTargetIds()),fiveSaved);
  for(const id of fiveSaved)assert.equal(h.api.isSelectedTarget(id),true,id);
  h.dom('myPrintersBtn').fire('click');
  h.dom('saveMyPrintersBtn').fire('click');
  await flush();
  assert.deepEqual(h.plain(sent),fiveSaved);
});

test('ticking a model card selects only its default target; unticking removes that model only',async()=>{
  const h=loadWithGlobals({GetMyPrinters:()=>({selectedIds:[oldIds.c5],onboardingPending:false})});
  h.api.setSupportedTargets({available:availabilityAll(),installedTargets:proVariants,prusaAvailable:false});
  await h.api.init();
  h.dom('myPrintersBtn').fire('click');
  h.dom('printerCatalogSearch').value='Creator 5';
  h.dom('printerCatalogSearch').fire('input');
  const proCard=cardFor(h,'Creator 5 Pro');
  assert.equal(proCard.children[0].checked,false);
  proCard.children[0].checked=true;
  proCard.children[0].fire('change');
  assert.equal(h.dom('myPrintersSelectionCount').textContent,'2 printers selected');
  assert.equal(cardFor(h,'Creator 5 Pro').children[0].checked,true);
  assert.equal(cardFor(h,'Creator 5').children[0].checked,true);
  // Unticking the Pro card removes its five targets and leaves the built-in Creator 5 alone.
  cardFor(h,'Creator 5 Pro').children[0].checked=false;
  cardFor(h,'Creator 5 Pro').children[0].fire('change');
  assert.equal(h.dom('myPrintersSelectionCount').textContent,'1 printers selected');
  assert.equal(cardFor(h,'Creator 5').children[0].checked,true);
});

test('variant checkboxes keep other nozzles selected and never swap one for another',async()=>{
  let sent=null;
  const h=loadWithGlobals({
    GetMyPrinters:()=>({selectedIds:[oldIds.pro],onboardingPending:false}),
    SaveMyPrinters:(requested)=>{sent=requested;return {selectedIds:requested,onboardingPending:false};},
  });
  h.api.setSupportedTargets({available:[],installedTargets:proVariants,prusaAvailable:false});
  await h.api.init();
  h.dom('myPrintersBtn').fire('click');
  h.dom('printerCatalogSearch').value='Creator 5 Pro';
  h.dom('printerCatalogSearch').fire('input');
  const card=cardFor(h,'Creator 5 Pro');
  const boxes=variantBoxes(card);
  assert.equal(boxes.length,5);
  assert.equal(boxes.filter(box=>box.checked).length,1);
  assert.equal(boxes.find(box=>box.value===oldIds.pro).checked,true);
  const [quarter]=boxes.filter(box=>box.value===proVariants[0].id);
  quarter.checked=true;
  quarter.fire('change');
  const again=variantBoxes(cardFor(h,'Creator 5 Pro'));
  assert.equal(again.find(box=>box.value===oldIds.pro).checked,true);
  assert.equal(again.find(box=>box.value===proVariants[0].id).checked,true);
  h.dom('saveMyPrintersBtn').fire('click');
  await flush();
  assert.deepEqual(h.plain(sent),[oldIds.pro,proVariants[0].id]);
});

test('card text never shows a release date, in English or Hebrew',async()=>{
  const h=loadWithGlobals({GetMyPrinters:()=>({selectedIds:[],onboardingPending:false})});
  h.api.setSupportedTargets({available:availabilityAll(),installedTargets:proVariants,prusaAvailable:true});
  await h.api.init();
  h.dom('myPrintersBtn').fire('click');
  const text=textOf(h.dom('catalogModelList'));
  assert.ok(text.includes('Creator 5 Pro'));
  assert.doesNotMatch(text,/\b(?:19|20)\d{2}\b/);
  assert.doesNotMatch(text,/Launched|השקה/);
  assert.doesNotMatch(script,/model-launch-date|Launched:|השקה:/);
  assert.equal(h.dom('catalogModelList').querySelectorAll('.model-launch-date').length,0);
});

test('a card with no bundled or installed artwork says so, and never borrows another model',async()=>{
  const h=loadWithGlobals({GetMyPrinters:()=>({selectedIds:[],onboardingPending:false})});
  h.api.setSupportedTargets({available:availabilityAll(),installedTargets:proVariants,prusaAvailable:false});
  await h.api.init();
  h.dom('myPrintersBtn').fire('click');
  h.dom('printerCatalogSearch').value='Creator 5';
  h.dom('printerCatalogSearch').fire('input');
  await flush();
  for(const model of ['Creator 5','Creator 5 Pro']){
    const avatar=cardFor(h,model).children[1].children[0];
    assert.equal(avatar.classList.contains('is-unavailable'),true,model);
    assert.equal(avatar.querySelector('img.printer-cover-photo'),null,model);
  }
});

test('bundled manifest images attach to the exact physical model only',async()=>{
  const manifest={
    'Flashforge|Creator 5 Pro':'printer-images/flashforge-creator-5-pro.png',
    'Flashforge|Creator 5':'printer-images/flashforge-creator-5.png',
    'Flashforge|Creator 5 Pro Max':'printer-images/not-a-model.png',
    'Flashforge|Bad':'../escape.png',
    'Flashforge|Scheme':'javascript:alert(1).png',
  };
  const fetch=async()=>({ok:true,json:async()=>manifest});
  const h=loadWithGlobals({GetMyPrinters:()=>({selectedIds:[],onboardingPending:false})},{fetch});
  h.api.setSupportedTargets({available:availabilityAll(),installedTargets:proVariants,prusaAvailable:false});
  await h.api.init();
  await flush();
  h.dom('myPrintersBtn').fire('click');
  h.dom('printerCatalogSearch').value='Creator 5';
  h.dom('printerCatalogSearch').fire('input');
  await flush();
  const proImg=cardFor(h,'Creator 5 Pro').children[1].children[0].querySelector('img.printer-cover-photo');
  assert.ok(proImg,'Creator 5 Pro card should show its own bundled image');
  assert.equal(proImg.src,'./printer-images/flashforge-creator-5-pro.png');
  // Each card gets only the image its own key maps to.
  assert.equal(cardFor(h,'Creator 5 Pro').children[1].children[0].querySelector('img.printer-cover-photo').src,
               './printer-images/flashforge-creator-5-pro.png');
  assert.equal(cardFor(h,'Creator 5').children[1].children[0].querySelector('img.printer-cover-photo').src,
               './printer-images/flashforge-creator-5.png');
  assert.equal(h.dom('catalogModelList').children.length,2);
});

test('the bundled manifest is optional: a missing or malformed file leaves the UI working',async()=>{
  const missing=loadWithGlobals({GetMyPrinters:()=>({selectedIds:[],onboardingPending:false})},
    {fetch:async()=>{throw Error('offline');}});
  missing.api.setSupportedTargets({available:availabilityAll(),installedTargets:[],prusaAvailable:false});
  await missing.api.init();
  assert.equal(missing.api.isReady(),true);
  const notJson=loadWithGlobals({GetMyPrinters:()=>({selectedIds:[],onboardingPending:false})},
    {fetch:async()=>({ok:true,json:async()=>{throw SyntaxError('<html>');}})});
  notJson.api.setSupportedTargets({available:availabilityAll(),installedTargets:[],prusaAvailable:false});
  await notJson.api.init();
  assert.equal(notJson.api.isReady(),true);
});

test('the installed PNG cover of a catalogue model is used only for that model',async()=>{
  const requested=[];
  const PrinterArtwork=(id)=>{
    requested.push(id);
    return id==='flashforge-creator-5-pro'
      ?{found:true,provenance:'local_installed_slicer',dataUri:'data:image/png;base64,AAAA'}
      :{found:false};
  };
  const h=loadWithGlobals({GetMyPrinters:()=>({selectedIds:[],onboardingPending:false}),PrinterArtwork});
  h.api.setSupportedTargets({available:availabilityAll(),installedTargets:proVariants,prusaAvailable:false});
  await h.api.init();
  h.dom('myPrintersBtn').fire('click');
  h.dom('printerCatalogSearch').value='Creator 5';
  h.dom('printerCatalogSearch').fire('input');
  await flush();
  const proImg=cardFor(h,'Creator 5 Pro').children[1].children[0].querySelector('img.printer-cover-photo');
  assert.equal(proImg.src,'data:image/png;base64,AAAA');
  assert.equal(cardFor(h,'Creator 5').children[1].children[0].querySelector('img.printer-cover-photo'),null);
  assert.ok(requested.includes('flashforge-creator-5-pro'));
});
