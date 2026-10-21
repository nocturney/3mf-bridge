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
  assert.equal([...preferred,...others].some(name=>/prusaslicer|generic/i.test(name)),false);
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
  // The full catalogue's makers: mainstream first, one divider, then everyone else A-Z.
  const makers=values.slice(1),divider=makers.indexOf('');
  assert.equal(values[0],'');
  assert.deepEqual(makers.slice(0,divider),['Anycubic','Creality','ELEGOO','Flashforge','FLSun','Prusa Research','QIDI','Snapmaker','Sovol']);
  assert.equal(makers.filter(value=>value==='').length,1);
  assert.deepEqual(options.filter(([,disabled])=>disabled).map(([value])=>value),['']);
  const others=makers.slice(divider+1);
  assert.ok(others.includes('Afinia')&&others.includes('Zortrax'));
  assert.deepEqual(others,[...others].sort((a,b)=>a.localeCompare(b,'en',{sensitivity:'base'})));
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
