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
  const makers=picker.manufacturerNames(items);
  assert.equal(makers.filter(name=>/^elegoo$/i.test(name)).length,1);
  assert.equal(makers.filter(name=>/^qidi$/i.test(name)).length,1);
  assert.ok(makers.includes('ELEGOO'));
  assert.ok(makers.includes('QIDI'));
  assert.equal(new Set(makers.map(name=>name.toLowerCase())).size,makers.length);
  // PrusaSlicer is a generic export target, never a physical maker.
  assert.equal(makers.some(name=>/prusaslicer|generic/i.test(name)),false);
  assert.equal(items.find(item=>item.id==='prusa-project').manufacturer,'Generic');
  assert.equal(picker.makerLabel('Prusa'),'Prusa Research');
  assert.equal(picker.makerLabel(' prusa research '),'Prusa Research');
  assert.equal(picker.makerKey('Creality'),picker.makerKey('  creality '));
});

test('manufacturer dropdown is one alphabetic list, no preferred conversion vendors or separators',async()=>{
  const h=loadMyPrinters({GetMyPrinters:()=>({selectedIds:[],onboardingPending:false})});
  h.api.setSupportedTargets({available:availabilityAll(),installedTargets:[...proVariants,
    {id:'orca-elegoo-centauri-carbon-9999999999',vendor:'Elegoo',model:'Centauri Carbon',title:'Elegoo Centauri Carbon 0.4 nozzle',nozzle:'0.4'},
    {id:'orca-zortrax-m200-0000000001',vendor:'Zortrax',model:'M200',title:'Zortrax M200 0.4 nozzle',nozzle:'0.4'}],prusaAvailable:true});
  await h.api.init();
  const options=h.dom('printerManufacturerFilter').children.map(option=>[option.value,option.disabled===true]);
  const values=options.map(([value])=>value);
  const makers=values.slice(1);
  assert.equal(values[0],'');
  assert.equal(makers.filter(value=>value==='').length,0);
  assert.equal(options.filter(([,disabled])=>disabled).length,0);
  assert.deepEqual(makers,[...makers].sort((a,b)=>a.localeCompare(b,'en',{sensitivity:'base'})));
  assert.equal(makers.includes('Afinia'),false);
  assert.equal(makers.includes('Zortrax'),false);
  assert.equal(makers.length,11); // Prusa is one maker; Bambu remains source-only.
  assert.equal(values.filter(value=>value==='ELEGOO').length,1);
  h.dom('myPrintersBtn').fire('click');
  h.dom('showOtherManufacturersBtn').fire('click');
  const all=h.dom('printerManufacturerFilter').children.map(option=>option.value);
  assert.ok(all.includes('Afinia')&&all.includes('Zortrax'));
  assert.equal(all.includes('Bambu Lab'),false);
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

test('the 5 Pro card stays simple; nozzle and slicer selection is in a separate settings dialog',async()=>{
  let sent=null;
  const h=loadWithGlobals({
    GetMyPrinters:()=>({selectedIds:[oldIds.c5,oldIds.pro],onboardingPending:false}),
    SaveMyPrinters:(requested)=>{sent=requested;return {selectedIds:requested,onboardingPending:false};},
  });
  h.api.setSupportedTargets({available:availabilityAll(),installedTargets:proVariants,prusaAvailable:false});
  await h.api.init();
  h.dom('myPrintersBtn').fire('click');
  h.dom('printerCatalogSearch').value='Creator 5 Pro';
  h.dom('printerCatalogSearch').fire('input');
  const card=cardFor(h,'Creator 5 Pro');
  assert.equal(card.querySelectorAll('.catalog-variant-list').length,0);
  assert.equal(card.querySelectorAll('input').length,1);
  assert.equal(card.children[0].checked,true);
  assert.equal(h.dom('myPrintersSelectionCount').textContent,'2 printers selected');
  const button=card.children.find(x=>x.classList.contains('catalog-model-profile-button'));
  assert.ok(button,'separate settings action should exist');
  button.fire('click');
  assert.equal(h.dom('printerProfileDialog').open,true);
  let options=h.dom('printerProfileOptions').children.map(child=>child.children[0]);
  assert.equal(options.length,5); // only five real installed profiles are offered as new choices.
  assert.equal(options.find(option=>option.value===oldIds.pro).checked,true);
  // Closing without applying does not mutate the draft or the persisted selection.
  h.dom('cancelPrinterProfileBtn').fire('click');
  assert.equal(h.dom('printerProfileDialog').open,false);
  button.fire('click');
  options=h.dom('printerProfileOptions').children.map(child=>child.children[0]);
  options.find(option=>option.value===oldIds.pro).checked=false;
  options.find(option=>option.value===proVariants[0].id).checked=true;
  h.dom('applyPrinterProfileBtn').fire('click');
  assert.equal(h.dom('printerProfileDialog').open,false);
  assert.equal(h.dom('myPrintersSelectionCount').textContent,'2 printers selected');
  assert.equal(cardFor(h,'Creator 5 Pro').children[0].checked,true);
  h.dom('saveMyPrintersBtn').fire('click');
  await flush();
  assert.deepEqual(h.plain(sent),[oldIds.c5,proVariants[0].id]);
});

test('card text never shows a release date, in English or Hebrew',async()=>{
  const h=loadWithGlobals({GetMyPrinters:()=>({selectedIds:[],onboardingPending:false})});
  h.api.setSupportedTargets({available:availabilityAll(),installedTargets:proVariants,prusaAvailable:true});
  await h.api.init();
  h.dom('myPrintersBtn').fire('click');
  const text=textOf(h.dom('catalogModelList'));
  assert.ok(text.includes('Creator 5 Pro'));
  // Model names may carry a year ("Ender-5 Pro (2019)"); release dates may not appear as text.
  assert.doesNotMatch(text,/\b\d{4}-\d{2}(?:-\d{2})?\b/);
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

// Installed covers are asked for by target ID, and each one is only ever attached to its own model.
const installedCover=(dataUri)=>({found:true,provenance:'local_installed_slicer',dataUri});
const proCover='data:image/png;base64,AAAA';
const c5Cover='data:image/png;base64,BBBB';
const avatarOf=(h,model)=>cardFor(h,model).children[1].children[0];

test('installed cover lookups run once per slicer, not once per nozzle',async()=>{
  const asked=[];
  const h=loadWithGlobals({
    GetMyPrinters:()=>({selectedIds:[],onboardingPending:false}),
    PrinterArtworkForInstalledTarget:(id)=>{
      asked.push(id);
      // The second slicer has a cover; four nozzle profiles in the first slicer do not.
      return id===proVariants[4].id?installedCover(proCover):{found:false};
    },
  });
  h.api.setSupportedTargets({available:availabilityAll(),installedTargets:proVariants,prusaAvailable:false});
  await h.api.init();
  h.dom('myPrintersBtn').fire('click');
  h.dom('printerCatalogSearch').value='Creator 5 Pro';
  h.dom('printerCatalogSearch').fire('input');
  await flush();
  // The variants are inspected in physical-nozzle order. One request per
  // slicer is sufficient, without ranking manufacturers or converter tiers.
  assert.deepEqual(asked,[proVariants[0].id,proVariants[4].id]);
  assert.equal(avatarOf(h,'Creator 5 Pro').querySelector('img.printer-cover-photo').src,proCover);
});

test('catalogue cover fast path skips installed Orca target rescans',async()=>{
  const asked=[],catalogAsked=[];
  const h=loadWithGlobals({
    GetMyPrinters:()=>({selectedIds:[],onboardingPending:false}),
    PrinterArtworkForInstalledTarget:(id)=>{asked.push(id);return {found:false};},
    PrinterArtwork:(id)=>{
      catalogAsked.push(id);
      return id==='flashforge-creator-5-pro'?installedCover(proCover):{found:false};
    },
  });
  h.api.setSupportedTargets({available:availabilityAll(),installedTargets:proVariants,prusaAvailable:false});
  await h.api.init();
  h.dom('myPrintersBtn').fire('click');
  h.dom('printerCatalogSearch').value='Creator 5 Pro';
  h.dom('printerCatalogSearch').fire('input');
  await flush();
  assert.deepEqual(asked,[]);
  assert.ok(catalogAsked.includes('flashforge-creator-5-pro'));
  assert.equal(avatarOf(h,'Creator 5 Pro').querySelector('img.printer-cover-photo').src,proCover);
});

test('installed covers attach to their own model; the exact catalogue cover is the only fallback',async()=>{
  const asked=[],catalogAsked=[];
  const h=loadWithGlobals({
    GetMyPrinters:()=>({selectedIds:[],onboardingPending:false}),
    PrinterArtworkForInstalledTarget:(id)=>{asked.push(id);return installedCover(proCover);},
    PrinterArtwork:(id)=>{
      catalogAsked.push(id);
      return id==='flashforge-creator-5'?installedCover(c5Cover):{found:false};
    },
  });
  h.api.setSupportedTargets({available:availabilityAll(),installedTargets:proVariants,prusaAvailable:false});
  await h.api.init();
  h.dom('myPrintersBtn').fire('click');
  h.dom('printerCatalogSearch').value='Creator 5';
  h.dom('printerCatalogSearch').fire('input');
  await flush();
  // The Pro cover never reaches the Creator 5 card, and the Creator 5 card gets its own catalogue cover.
  assert.equal(avatarOf(h,'Creator 5 Pro').querySelector('img.printer-cover-photo').src,proCover);
  assert.equal(avatarOf(h,'Creator 5').querySelector('img.printer-cover-photo').src,c5Cover);
  assert.ok(asked.every(id=>/^orca-/.test(id)));
  assert.ok(catalogAsked.includes('flashforge-creator-5'));
  assert.ok(catalogAsked.includes('flashforge-creator-5-pro'));
});

test('a refused or failed artwork request leaves an honest unavailable card and makes no network read',async()=>{
  const fetches=[];
  const fetch=async(url)=>{fetches.push(url);throw Error('offline');};
  const h=loadWithGlobals({
    GetMyPrinters:()=>({selectedIds:[],onboardingPending:false}),
    PrinterArtworkForInstalledTarget:()=>{throw Error('installed printer profile is not ready');},
    PrinterArtwork:()=>({found:true,provenance:'remote',dataUri:'https://example.test/cover.png'}),
  },{fetch});
  h.api.setSupportedTargets({available:availabilityAll(),installedTargets:proVariants,prusaAvailable:false});
  await h.api.init();
  h.dom('myPrintersBtn').fire('click');
  h.dom('printerCatalogSearch').value='Creator 5';
  h.dom('printerCatalogSearch').fire('input');
  await flush();
  assert.deepEqual(fetches,[]);
  for(const model of ['Creator 5','Creator 5 Pro']){
    assert.equal(avatarOf(h,model).classList.contains('is-unavailable'),true,model);
    assert.equal(avatarOf(h,model).querySelector('img.printer-cover-photo'),null,model);
  }
});

// My Printers is a physical inventory. A conversion engine's ranking and
// readiness never sort manufacturers, variants or the default card checkbox.
const installedTarget=(vendor,model,slicer,nozzle,hex,ready)=>({
  id:'orca-'+(vendor+'-'+model).toLowerCase().replace(/[^a-z0-9]+/g,'-').replace(/^-|-$/g,'')+'-'+hex,
  vendor,model,title:vendor+' '+model+' '+nozzle+' nozzle',nozzle,slicer,physicalTools:1,ready,
});
const defaultCard=(installed,model)=>picker.groupDestinations(picker.supportedModels([],[],installed,false))
  .find(group=>group.model===model);

test('the printer card defaults to a real 0.4 mm nozzle, not a preferred conversion engine',()=>{
  const orca=installedTarget('Flashforge','Creator 5 Pro','OrcaSlicer','0.4','1111111111');
  const flash=installedTarget('Flashforge','Creator 5 Pro','Flash Studio Desktop','0.6','2222222222');
  for(const installed of [[orca,flash],[flash,orca]]){
    const card=defaultCard(installed,'Creator 5 Pro');
    assert.equal(card.id,orca.id);
    assert.deepEqual(card.ids.slice().sort(),[orca.id,flash.id].sort());
  }
});

test('manufacturer-owned profiles never outrank the same physical nozzle from other valid slicers',()=>{
  const cases=[
    [['QIDI','X-Plus 5','OrcaSlicer'],['QIDI','X-Plus 5','QIDIStudio']],
    [['Snapmaker','U1','OrcaSlicer'],['Snapmaker','U1','Snapmaker Orca']],
    [['ELEGOO','Centauri Carbon 2','OrcaSlicer'],['ELEGOO','Centauri Carbon 2','ElegooSlicer']],
    [['Prusa Research','CORE One','OrcaSlicer'],['Prusa Research','CORE One','PrusaSlicer']],
  ];
  for(const [[vendor,model,slicerA],[, ,slicerB]] of cases){
    const other=installedTarget(vendor,model,slicerA,'0.4','3333333333');
    const owned=installedTarget(vendor,model,slicerB,'0.6','4444444444');
    for(const variants of [[other,owned],[owned,other]]){
      assert.equal(defaultCard(variants,model).id,other.id);
    }
  }
});

test('no vendor, process, or converter preference changes the default when no 0.4 variant exists',()=>{
  const qidi=installedTarget('QIDI','X-Plus 5','QIDIStudio','0.8','aaaaaaaaaa');
  const orca=installedTarget('QIDI','X-Plus 5','OrcaSlicer','0.25','bbbbbbbbbb');
  assert.equal(defaultCard([qidi,orca],'X-Plus 5').id,orca.id);
  assert.equal(defaultCard([orca,qidi],'X-Plus 5').id,orca.id);
});

test('ready/not-ready status never affects physical grouping or nozzle ordering',()=>{
  const variants=[
    {id:'prusa-acme-mini-aaaaaaaaaa',manufacturer:'Acme',model:'Mini',nozzle:'0.4',slicer:'PrusaSlicer',ready:false},
    {id:'orca-acme-mini-bbbbbbbbbb',manufacturer:'Acme',model:'Mini',nozzle:'0.6',slicer:'OrcaSlicer',ready:true},
  ];
  const initial=picker.groupDestinations(variants)[0];
  const swapped=picker.groupDestinations(variants.map(v=>({...v,ready:!v.ready})))[0];
  assert.deepEqual(initial.ids,swapped.ids);
  assert.equal(initial.id,swapped.id);
  assert.equal(initial.id,variants[0].id);
});

test('all nozzle variants remain configurable but their technical details never clutter the model card',async()=>{
  const sample=installedTarget('Flashforge','Creator 5 Pro','OrcaSlicer','0.4','eeeeeeeeee');
  const h=loadWithGlobals({GetMyPrinters:()=>({selectedIds:[],onboardingPending:false})});
  h.api.setSupportedTargets({available:[],installedTargets:[sample,...proVariants],prusaAvailable:false,
    installedPrusaTargets:[{id:'prusa-flashforge-creator-5-pro-ffffffffff',vendor:'Flashforge',
      model:'Creator 5 Pro',title:'Flashforge Creator 5 Pro 0.6 nozzle',nozzle:'0.6',ready:false}]});
  await h.api.init();
  h.dom('myPrintersBtn').fire('click');
  h.dom('printerCatalogSearch').value='Creator 5 Pro';
  h.dom('printerCatalogSearch').fire('input');
  const card=cardFor(h,'Creator 5 Pro');
  const labels=textOf(card);
  assert.doesNotMatch(labels,/PrusaSlicer|OrcaSlicer|Flash Studio|0\.4 mm|0\.25 mm|Tier|דירוג/i);
  assert.equal(card.querySelectorAll('input').length,1);
  card.children.find(child=>child.classList.contains('catalog-model-profile-button')).fire('click');
  const optionLabels=textOf(h.dom('printerProfileOptions'));
  assert.match(optionLabels,/PrusaSlicer/);
  assert.match(optionLabels,/OrcaSlicer/);
  assert.equal(h.dom('printerProfileDialog').open,true);
});

test('unrelated manufacturer fork presets never appear as ready options even from an older backend',()=>{
  const ankerBySnap=installedTarget('Anker','M5C','Snapmaker Orca','0.4','1111111111');
  const ankerByElegoo=installedTarget('Anker','M5C','ElegooSlicer','0.4','2222222222');
  const ankerByOrca=installedTarget('Anker','M5C','OrcaSlicer','0.4','3333333333');
  const flashByElegoo=installedTarget('Flashforge','Creator 5 Pro','ElegooSlicer','0.6','4444444444');
  const flashByFlash=installedTarget('Flashforge','Creator 5 Pro','Flash Studio Desktop','0.25','5555555555');
  const items=picker.supportedModels([],[],[ankerBySnap,ankerByElegoo,ankerByOrca,
                                             flashByElegoo,flashByFlash],false);
  assert.deepEqual(items.map(v=>v.id).sort(),[ankerByOrca.id,flashByFlash.id].sort());
});

test('Anker M5C / AnkerMake M5C alias belongs to one card, never copied nozzle settings',()=>{
  const ids=['anker-anker-m5c','anker-ankermake-m5c'];
  const items=picker.catalogItems(catalog.models,[],[],false);
  const group=picker.groupDestinations(items).find(g=>g.manufacturer==='Anker'&&g.model==='M5C');
  assert.ok(group);
  assert.ok(ids.every(id=>group.ids.includes(id)));
  assert.equal(group.variants.filter(v=>v.nozzle||v.slicer).length,0);
  assert.deepEqual(picker.allSavedIds(items,[...ids]),ids);
});

test('saved 5 printer identities survive installed-profile availability and converter changes',async()=>{
  const saved=[...fiveSaved];
  const h=loadWithGlobals({
    GetMyPrinters:()=>({selectedIds:saved,onboardingPending:false}),
    SaveMyPrinters:(requested)=>({selectedIds:requested,onboardingPending:false}),
  });
  h.api.setSupportedTargets({available:availabilityAll(),installedTargets:proVariants,prusaAvailable:false});
  await h.api.init();
  h.dom('myPrintersBtn').fire('click');
  assert.equal(h.dom('myPrintersSelectionCount').textContent,'5 printers selected');
  h.api.setSupportedTargets({available:[],installedTargets:proVariants.map(v=>({...v,slicer:'OrcaSlicer'})),
    prusaAvailable:false});
  h.dom('saveMyPrintersBtn').fire('click');
  await flush();
  assert.deepEqual(h.plain(saved),fiveSaved);
});

test('no conversion ranking implementation remains in the physical printer picker',()=>{
  const pickerScript=fs.readFileSync(path.join(dist,'printer-picker.js'),'utf8');
  const dialogScript=fs.readFileSync(path.join(dist,'my-printers.js'),'utf8');
  assert.doesNotMatch(pickerScript,/sourceTier|primaryMakers|manufacturerSlicers|best source tier/);
  assert.doesNotMatch(dialogScript,/manufacturerGroups|selectedAdapterTargets|מתאם מובנה|builtInVariant/);
});


test('temporarily unchecking and rechecking Creator 5 Pro restores the saved exact 0.25 profile',async()=>{
  const profile=proVariants[0];
  const saved=[oldIds.cc2,oldIds.c5,oldIds.qidi,oldIds.u1,profile.id];
  let request;
  const h=loadWithGlobals({
    GetMyPrinters:()=>({selectedIds:saved,onboardingPending:false}),
    SaveMyPrinters:ids=>{request=[...ids];return {selectedIds:ids,onboardingPending:false};},
  });
  h.api.setSupportedTargets({available:availabilityAll(),installedTargets:proVariants,prusaAvailable:false});
  await h.api.init();
  h.dom('myPrintersBtn').fire('click');
  h.dom('printerCatalogSearch').value='Creator 5 Pro';
  h.dom('printerCatalogSearch').fire('input');
  const before=cardFor(h,'Creator 5 Pro').children[0];
  assert.equal(before.checked,true);
  before.checked=false;before.fire('change');
  assert.equal(h.dom('myPrintersSelectionCount').textContent,'4 printers selected');
  const restore=cardFor(h,'Creator 5 Pro').children[0];
  restore.checked=true;restore.fire('change');
  assert.equal(h.dom('myPrintersSelectionCount').textContent,'5 printers selected');
  h.dom('saveMyPrintersBtn').fire('click');
  await flush();
  assert.deepEqual(h.plain(request),saved);
});

test('two preexisting spelling-alias IDs show one physical printer without erasing either saved ID',async()=>{
  const saved=['anker-anker-m5c','anker-ankermake-m5c'];
  let request;
  const h=loadWithGlobals({
    GetMyPrinters:()=>({selectedIds:saved,onboardingPending:false}),
    SaveMyPrinters:ids=>{request=[...ids];return {selectedIds:ids,onboardingPending:false};},
  });
  h.api.setSupportedTargets({available:[],installedTargets:[],prusaAvailable:false});
  await h.api.init();
  h.dom('myPrintersBtn').fire('click');
  h.dom('printerCatalogSearch').value='M5C';
  h.dom('printerCatalogSearch').fire('input');
  const m5c=cardFor(h,'M5C');
  assert.ok(m5c);
  assert.equal(m5c.querySelectorAll('input').length,1);
  assert.equal(m5c.querySelectorAll('.catalog-variant-list').length,0);
  assert.equal(m5c.children[0].checked,true);
  assert.equal(h.dom('myPrintersSelectionCount').textContent,'1 printers selected');
  assert.equal(h.dom('selectedPrinterSummary').children.length,1);
  h.dom('saveMyPrintersBtn').fire('click');
  await flush();
  assert.deepEqual(h.plain(request),saved);
});
