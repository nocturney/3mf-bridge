/* SliceBridge My Printers: offline snapshot + explicit, separate local selection.
 * Only supported destinations are listed (see SliceBridgePicker.supportedModels).
 * One card per physical model; its nozzle/slicer variants keep their own target IDs.
 * No other conversion target is activated by browsing this catalogue.
 * No network reads for artwork: pictures come only from installed slicers, asked
 * through the backend. No bundled photos, manifests or cross-origin fetches.
 */
(function(){
'use strict';
const $ = (id)=>document.getElementById(id);
const picker=window.SliceBridgePicker;
const translations={
  he:{
    title:'המדפסות שלי',welcome:'בחירת מדפסות',welcomeBody:'לא נבחרה עדיין מדפסת. בחר יצרן או חפש דגם כדי להתחיל.',
    body:'בחר מדפסות לעבודה ולמעקב. הגדרות דיזה, סלייסר ופילמנט קיימות אינן משתנות.',
    search:'חיפוש לפי יצרן או דגם',save:'שמור מדפסות',cancel:'לא עכשיו',selected:'נבחרו {n} מדפסות',
    none:'לא נמצאו דגמים',
    showing:'מציג {shown} מתוך {total} דגמים · צמצם את החיפוש להצגת דגמים נוספים',
    limit:'אפשר לבחור עד 32 מדפסות.',error:'לא ניתן לטעון או לשמור את בחירת המדפסות.',
    manufacturer:'יצרן',chooseManufacturer:'בחר יצרן…',searchByModel:'או חפש דגם',startHint:'בחר יצרן או חפש דגם כדי להציג מדפסות.',
    noSupported:'עדיין אין יעדים זמינים במחשב זה.',nothingSelected:'עדיין לא נבחרו מדפסות.',
    variants:'זרבובית וסלייסר',builtInVariant:'מתאם מובנה',imageUnavailable:'תמונה אינה זמינה'
  },
  en:{
    title:'My printers',welcome:'Choose printers',welcomeBody:'No printers are selected yet. Choose a manufacturer or search for a model to begin.',
    body:'Add or remove printers without changing your existing nozzle, slicer or filament preferences.',
    search:'Search manufacturer or model',save:'Save printers',cancel:'Not now',selected:'{n} printers selected',
    none:'No models matched',
    showing:'Showing {shown} of {total} models · narrow the search to find more',
    limit:'You can select up to 32 printers.',error:'Could not load or save printer selection.',
    manufacturer:'Manufacturer',chooseManufacturer:'Choose manufacturer…',searchByModel:'Or search a model',startHint:'Choose a manufacturer or search to see models.',
    noSupported:'No destinations are available on this computer yet.',nothingSelected:'No printers selected yet.',
    variants:'Nozzle and slicer',builtInVariant:'Built-in target',imageUnavailable:'Image unavailable'
  }
};
const changedEvent='slicebridge:my-printers-changed';
let catalog=null,selected=null,draft=null,firstRun=false,loaded=false,saving=false;
// Destination inputs, pushed in by the app shell through setSupportedTargets().
let availability=[],installed=[],prusaAvailable=false,supported=[],groups=[];
let groupByKey=new Map(),groupByItemId=new Map(),catalogIndex=new Map();

const t=(key,variables={})=>{
  const lang=document.documentElement.lang==='en'?'en':'he';
  let str=translations[lang][key]||key;
  for(const [name,value] of Object.entries(variables))str=str.replaceAll('{'+name+'}',String(value));
  return str;
};
const node=(tag,cls,text)=>{
  const out=document.createElement(tag);
  if(cls)out.className=cls;
  if(text!=null)out.textContent=String(text); // never interpret upstream strings as markup
  return out;
};
const api=()=>window.go?.main?.App;

// Artwork is resolved per physical model, never per nozzle. Order: each installed
// target's own cover (its exact printer_model in its own slicer install), then the
// exact catalogue model's local cover. A card with neither shows "Image unavailable"
// rather than another model's picture. Nothing is bundled or downloaded.
const artCache=new Map(),artQueued=new Set(),artQueue=[],artGroups=new Map();
// Avatars in the dialog and icons in the main list are painted from separate registries,
// so a dialog re-render never strips artwork from destination rows still resolving.
const slotRegistry={dialog:new Map(),target:new Map()}; // scope -> physical key -> Set of slots
const installedTargetPattern=/^orca-[a-z0-9]+(?:-[a-z0-9]+)*-[0-9a-f]{10}$/;
let artRunning=0;
let artObserver=typeof IntersectionObserver==='function'
  ?new IntersectionObserver(entries=>{
    for(const item of entries){
      if(!item.isIntersecting)continue;
      artObserver.unobserve(item.target);
      queueArtwork(artGroups.get(item.target.dataset.artKey));
    }
  },{root:$('catalogModelList'),rootMargin:'160px'}):null;

function registerSlot(scope,slot,group){
  const registry=slotRegistry[scope];
  if(!registry.has(group.key))registry.set(group.key,new Set());
  registry.get(group.key).add(slot);
}
function markUnavailable(slot){
  if(!slot.classList.contains('printer-avatar'))return;
  slot.classList.add('is-unavailable');
  if(!slot.querySelector('.image-unavailable'))slot.append(node('span','image-unavailable',t('imageUnavailable')));
}
function paintSlot(slot,art){
  if(!slot)return;
  if(!art){markUnavailable(slot);return;}
  if(slot.querySelector('img.printer-cover-photo'))return;
  const image=node('img','printer-cover-photo');
  image.src=art.src;image.alt='';image.decoding='async';image.loading='lazy';
  image.onerror=()=>{image.remove();slot.classList.remove('has-cover');markUnavailable(slot);};
  slot.append(image);slot.classList.add('has-cover');
}
// Keys a group answers to: its own physical key plus known catalogue aliases.
function artKeysFor(group){return picker.lookupKeys(group);}
// Only installed Orca-family targets of this physical model, in nozzle order.
function installedTargetIdsFor(group){
  return group.variants
    .filter(variant=>!variant.legacyTarget&&installedTargetPattern.test(variant.id))
    .map(variant=>variant.id);
}
function catalogIdsFor(group){
  const ids=[];
  for(const key of artKeysFor(group))for(const id of catalogIndex.get(key)??[]){
    if(!ids.includes(id))ids.push(id);
  }
  return ids;
}
// Asks the backend for one picture. Only a validated PNG from an installed slicer is accepted.
async function backendArtwork(method,argument){
  const result=await Promise.resolve().then(()=>api()?.[method]?.(argument)).catch(()=>null);
  return result?.found&&result?.provenance==='local_installed_slicer'&&
    typeof result.dataUri==='string'&&result.dataUri.length<800000&&
    result.dataUri.startsWith('data:image/png;base64,')?result.dataUri:null;
}
async function resolveArtwork(group){
  for(const id of installedTargetIdsFor(group)){
    const uri=await backendArtwork('PrinterArtworkForInstalledTarget',id);
    if(uri)return {src:uri};
  }
  for(const id of catalogIdsFor(group)){
    const uri=await backendArtwork('PrinterArtwork',id);
    if(uri)return {src:uri};
  }
  return null;
}
function pumpArtwork(){
  while(artRunning<4&&artQueue.length){
    const group=artQueue.shift();artRunning++;
    resolveArtwork(group).catch(()=>null).then((art)=>{
      artCache.set(group.key,art);
      artQueued.delete(group.key);
      artRunning--;
      for(const registry of Object.values(slotRegistry))
        for(const slot of registry.get(group.key)??[])paintSlot(slot,art);
      pumpArtwork();
    });
  }
}
function queueArtwork(group){
  if(!group||artCache.has(group.key)||artQueued.has(group.key))return;
  artQueued.add(group.key);artQueue.push(group);pumpArtwork();
}
function observeArtwork(slot,group){
  slot.dataset.artKey=group.key;
  artGroups.set(group.key,group);
  registerSlot('dialog',slot,group);
  if(artCache.has(group.key)){paintSlot(slot,artCache.get(group.key));return;}
  if(artObserver)artObserver.observe(slot);
  else queueArtwork(group); // older embedded WebViews
}

function buildCatalogIndex(){
  catalogIndex=new Map();
  for(const model of catalog?.models??[]){
    if(picker.isHidden(model)||!Array.isArray(model.localCoverCandidates)||!model.localCoverCandidates.length)continue;
    for(const key of picker.lookupKeys(model)){
      if(!catalogIndex.has(key))catalogIndex.set(key,[]);
      catalogIndex.get(key).push(model.id);
    }
  }
}

// Target cards share the same artwork resolution as My Printers cards.
function destinationSlots(){
  const slots=[];
  const chosen=new Set(selectedTargetIds());
  for(const item of supported){
    if(!chosen.has(item.id))continue;
    const selector=item.legacyTarget?'[data-target-row="'+item.legacyTarget+'"]':null;
    const row=selector?document.querySelector(selector):null;
    if(row)slots.push([row.querySelector('.target-printer-icon'),item]);
  }
  for(const row of document.querySelectorAll('[data-installed-target]')){
    const item=supported.find(entry=>entry.id===row.dataset.installedTarget);
    if(item&&chosen.has(item.id))slots.push([row.querySelector('.target-printer-icon'),item]);
  }
  return slots;
}
function loadTargetArtwork(){
  if(!loaded)return;
  slotRegistry.target.clear();
  for(const [tile,item] of destinationSlots()){
    const group=groupByItemId.get(item.id);
    if(!tile||!group)continue;
    tile.dataset.artKey=group.key;
    registerSlot('target',tile,group);
    if(artCache.has(group.key))paintSlot(tile,artCache.get(group.key));
    else queueArtwork(group);
  }
}
// Called by the app shell after installed-target rows are rebuilt.
function refreshDestinationArt(){
  if(!loaded)return;
  loadTargetArtwork();
}

// Saved printers that are currently supported and selected, in saved order.
function selectedTargetIds(){
  return loaded?picker.selectedTargetIds(supported,selected.selectedIds):[];
}
// Accepts a destination ID or a built-in key ("u1"), so the app shell can ask by either.
function isSelectedTarget(targetId){
  if(!loaded||typeof targetId!=='string')return false;
  const item=supported.find(entry=>entry.id===targetId||entry.legacyTarget===targetId);
  return Boolean(item)&&selectedTargetIds().includes(item.id);
}
function legacyTargets(){
  return loaded?picker.legacyTargetsFor(supported,selected.selectedIds):[];
}
function syncCopy(){
  $('myPrintersBtn').title=t('title');
  $('myPrintersBtn').setAttribute('aria-label',t('title'));
  $('myPrintersBtn').querySelector('span').textContent=t('title');
  $('myPrintersTitle').textContent=firstRun?t('welcome'):t('title');
  $('myPrintersIntro').textContent=firstRun?t('welcomeBody'):t('body');
  $('printerCatalogSearch').placeholder=t('search');
  $('printerManufacturerLabel').textContent=t('manufacturer');
  $('printerSearchLabel').textContent=t('searchByModel');
  $('printerManufacturerFilter').querySelector('option[value=""]').textContent=t('chooseManufacturer');
  $('saveMyPrintersBtn').textContent=t('save');
  $('cancelMyPrintersBtn').textContent=t('cancel');
}
function refreshManufacturerOptions(){
  const filter=$('printerManufacturerFilter');
  const previous=filter.value;
  filter.replaceChildren();
  const placeholder=node('option','',t('chooseManufacturer'));
  placeholder.value='';filter.append(placeholder);
  const {preferred,others}=picker.manufacturerGroups(supported);
  const makers=[...preferred,...others];
  for(const maker of preferred){
    const option=node('option','',maker);option.value=maker;filter.append(option);
  }
  if(preferred.length&&others.length){
    const divider=node('option','','────────────');
    divider.value='';divider.disabled=true;
    divider.setAttribute('aria-hidden','true');
    filter.append(divider);
  }
  for(const maker of others){
    const option=node('option','',maker);option.value=maker;filter.append(option);
  }
  filter.value=makers.includes(previous)?previous:'';
}
function variantLabel(item){
  return item.legacyTarget?t('builtInVariant'):(picker.variantText(item)||picker.title(item));
}
// A variant checkbox list appears only for physical models with several installed targets.
function variantList(group){
  const list=node('div','catalog-variant-list');
  list.setAttribute('role','group');
  list.setAttribute('aria-label',t('variants'));
  for(const variant of group.variants){
    const option=node('label','catalog-variant');
    const box=node('input');
    box.type='checkbox';box.value=variant.id;box.checked=draft.has(variant.id);
    box.disabled=saving||(!box.checked&&draft.size>=picker.maxSelected);
    box.addEventListener('change',()=>{
      if(box.checked&&draft.size>=picker.maxSelected){box.checked=false;$('myPrintersStatus').textContent=t('limit');return;}
      if(box.checked)draft.add(variant.id);else draft.delete(variant.id);
      $('myPrintersStatus').textContent='';
      render();
    });
    option.append(box,node('span','',variantLabel(variant)));
    list.append(option);
  }
  return list;
}
// One card per physical model. Ticking the card selects its default target; unticking
// removes every target of that model. Individual variants stay independently selectable.
function groupCard(group,index){
  const selectedCount=group.ids.filter(id=>draft.has(id)).length;
  const row=node('div','catalog-model-card');
  if(selectedCount)row.classList.add('is-selected');
  const input=node('input');
  input.type='checkbox';input.id='printer-pick-'+index;
  input.value=group.id;input.checked=selectedCount>0;
  input.disabled=saving||(!input.checked&&draft.size>=picker.maxSelected);
  input.setAttribute('aria-label',picker.title(group));
  input.addEventListener('change',()=>{
    if(input.checked){
      if(draft.size>=picker.maxSelected){input.checked=false;$('myPrintersStatus').textContent=t('limit');return;}
      draft.add(group.id);
    }else{
      for(const id of group.ids)draft.delete(id);
    }
    $('myPrintersStatus').textContent='';
    render();
  });
  const label=node('label','catalog-model-main');
  label.setAttribute('for',input.id);
  const graphic=node('span','printer-avatar');
  graphic.setAttribute('aria-hidden','true');
  graphic.append(node('span','avatar-device','▣'),node('span','avatar-initials',
    String(group.manufacturer||'?').slice(0,2).toLocaleUpperCase()));
  observeArtwork(graphic,group);
  const copy=node('span','catalog-model-copy');
  copy.append(node('strong','',group.model),node('small','',group.manufacturer));
  label.append(graphic,copy);
  row.append(input,label);
  if(group.variants.length>1)row.append(variantList(group));
  return row;
}
function render(){
  syncCopy();
  if(!loaded||!draft)return;
  const choice=[...draft];
  const empty=!supported.length;
  $('myPrintersSelectionCount').textContent=t('selected',{n:choice.length});
  const selectionList=$('selectedPrinterSummary');
  selectionList.replaceChildren();
  const entries=supported.filter(item=>draft.has(item.id));
  $('selectedPrinterSummary').hidden=!entries.length;
  $('selectedPrinterOverview').classList.toggle('is-empty',!entries.length);
  if(!entries.length)selectionList.append(node('span','catalog-muted',t('nothingSelected')));
  for(const entry of entries){
    const pill=node('span','selected-printer-pill');
    // Two variants of one model stay distinguishable in the summary.
    const variant=picker.variantText(entry);
    const name=node('strong','',variant?picker.title(entry)+' · '+variant:picker.title(entry));
    pill.append(name);selectionList.append(pill);
  }
  const term=$('printerCatalogSearch').value.trim();
  const manufacturer=$('printerManufacturerFilter').value;
  const hasFilter=Boolean(term||manufacturer);
  const showList=hasFilter||supported.length>0;
  const source=manufacturer?groups.filter(group=>group.manufacturer===manufacturer):groups;
  const matches=showList?picker.searchGroups(source,term,choice,80):{models:[],total:0};
  const list=$('catalogModelList');
  $('printerCatalogSearch').disabled=empty;
  $('printerManufacturerFilter').disabled=empty;
  $('myPrintersDialog').classList.toggle('has-results',showList);
  $('catalogPickerHint').textContent=empty?t('noSupported'):t('startHint');
  $('catalogPickerHint').hidden=showList;
  // No result counts or unsupported-model classifications unless the list is truncated.
  $('catalogSearchCount').hidden=matches.total<=matches.models.length;
  if(artObserver)artObserver.disconnect();
  slotRegistry.dialog.clear();
  list.replaceChildren();
  list.hidden=!showList;
  matches.models.forEach((group,index)=>list.append(groupCard(group,index)));
  if(showList&&!matches.models.length)list.append(node('p','catalog-empty',t('none')));
  $('catalogSearchCount').textContent=t('showing',{
    shown:matches.models.length,total:matches.total
  });
  $('saveMyPrintersBtn').disabled=saving||empty;
}
// Called by the app shell whenever Availability, installed Orca targets or the
// PrusaSlicer probe change. Rebuilds the supported list; a stale draft is trimmed.
function setSupportedTargets(next={}){
  availability=Array.isArray(next.available)?next.available:[];
  installed=Array.isArray(next.installedTargets)?next.installedTargets:[];
  prusaAvailable=next.prusaAvailable===true;
  recomputeSupported();
  if(draft)draft=new Set([...draft].filter(id=>supported.some(item=>item.id===id)));
  if(loaded){
    refreshManufacturerOptions();
    loadTargetArtwork();
  }
  if($('myPrintersDialog')?.open)render();
  window.dispatchEvent(new Event(changedEvent));
}
function recomputeSupported(){
  supported=picker?picker.supportedModels(catalog?catalog.models:[],availability,installed,prusaAvailable):[];
  groups=picker?picker.groupDestinations(supported):[];
  groupByKey=new Map(groups.map(group=>[group.key,group]));
  groupByItemId=new Map();
  for(const group of groups)for(const variant of group.variants)groupByItemId.set(variant.id,group);
}
function open(onboarding=false){
  if(!loaded||saving)return;
  if($('myPrintersDialog').open)return;
  firstRun=Boolean(onboarding);
  draft=new Set(selectedTargetIds());
  $('printerCatalogSearch').value='';
  $('printerManufacturerFilter').value='';
  $('myPrintersStatus').textContent='';
  render();
  $('myPrintersDialog').showModal();
  $('printerCatalogSearch').focus();
}
async function save(){
  if(!loaded||!draft||saving||!supported.length)return;
  saving=true;
  $('saveMyPrintersBtn').disabled=true;
  $('myPrintersStatus').textContent='';
  try{
    // Only supported, selected IDs are sent. Stale saved IDs are not sent back.
    const requested=[...draft].filter(id=>supported.some(item=>item.id===id));
    const reply=await api().SaveMyPrinters(requested);
    if(!reply||reply.onboardingPending||!Array.isArray(reply.selectedIds)){
      throw Error('Saved printer selection was not confirmed');
    }
    selected=reply;
    loadTargetArtwork();
    $('myPrintersDialog').close();
    window.dispatchEvent(new Event(changedEvent));
  }catch(err){
    $('myPrintersStatus').textContent=t('error')+' '+String(err?.message||err);
  }finally{
    saving=false;
    if($('myPrintersDialog').open)render();
  }
}
async function init(){
  if(!picker||!api()?.PrinterCatalog||!api()?.GetMyPrinters)return;
  try{
    const [list,prefs]=await Promise.all([api().PrinterCatalog(),api().GetMyPrinters()]);
    if(!list||list.kind!=='slicebridge-offline-printer-picker'||
       list.modelCount!==603||!Array.isArray(list.models)||
       !prefs||!Array.isArray(prefs.selectedIds))throw Error('Invalid offline catalog');
    catalog=list;selected=prefs;
    buildCatalogIndex();
    recomputeSupported();
    refreshManufacturerOptions();
    loaded=true;
    loadTargetArtwork();
    $('myPrintersBtn').disabled=false;
    window.dispatchEvent(new Event(changedEvent));
    if(prefs.onboardingPending)open(true);
  }catch(err){
    // Old converter stays usable if the optional catalogue cannot load.
    // Never overwrite malformed customer selection data.
    $('myPrintersBtn').disabled=true;
    $('myPrintersBtn').title=t('error')+' '+String(err?.message||err);
  }
}
$('myPrintersBtn').addEventListener('click',()=>open(false));
$('cancelMyPrintersBtn').addEventListener('click',()=>$('myPrintersDialog').close());
$('closeMyPrintersBtn').addEventListener('click',()=>$('myPrintersDialog').close());
$('saveMyPrintersBtn').addEventListener('click',save);
$('printerCatalogSearch').addEventListener('input',render);
$('printerManufacturerFilter').addEventListener('change',render);
$('myPrintersDialog').addEventListener('close',()=>{
  draft=null;
  firstRun=false;
});
window.SliceBridgeMyPrinters={
  init,setSupportedTargets,selectedTargetIds,isSelectedTarget,legacyTargets,render,
  refreshDestinationArt,
  isReady:()=>loaded,
};
})();
