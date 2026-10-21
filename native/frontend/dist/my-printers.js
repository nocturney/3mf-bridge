/* SliceBridge My Printers: offline snapshot + explicit, separate local selection.
 * Only supported destinations are listed (see SliceBridgePicker.supportedModels).
 * No other conversion target is activated by browsing this catalogue.
 * No third-party image download or cross-origin source profile fetch.
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
    noSupported:'עדיין אין יעדים זמינים במחשב זה.',nothingSelected:'עדיין לא נבחרו מדפסות.'
  },
  en:{
    title:'My printers',welcome:'Choose printers',welcomeBody:'No printers are selected yet. Choose a manufacturer or search for a model to begin.',
    body:'Add or remove printers without changing your existing nozzle, slicer or filament preferences.',
    search:'Search manufacturer or model',save:'Save printers',cancel:'Not now',selected:'{n} printers selected',
    none:'No models matched',
    showing:'Showing {shown} of {total} models · narrow the search to find more',
    limit:'You can select up to 32 printers.',error:'Could not load or save printer selection.',
    manufacturer:'Manufacturer',chooseManufacturer:'Choose manufacturer…',searchByModel:'Or search a model',startHint:'Choose a manufacturer or search to see models.',
    noSupported:'No destinations are available on this computer yet.',nothingSelected:'No printers selected yet.'
  }
};
const changedEvent='slicebridge:my-printers-changed';
let catalog=null,selected=null,draft=null,firstRun=false,loaded=false,saving=false;
// Destination inputs, pushed in by the app shell through setSupportedTargets().
let availability=[],installed=[],prusaAvailable=false,supported=[];

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

// Artwork stays entirely local. Only the model ID is sent to Wails; the Go
// backend reads a bounded PNG inside an installed slicer profiles directory.
// Only built-in models have artwork; installed Orca and PrusaSlicer items use the fallback avatar.
const coverCache=new Map(),coverQueued=new Set(),coverJobs=[];
let coverRunning=0;
const coverObserver=typeof IntersectionObserver==='function'
  ?new IntersectionObserver(entries=>{
    for(const item of entries){
      if(!item.isIntersecting)continue;
      coverObserver.unobserve(item.target);
      queueCover(item.target.dataset.artId);
    }
  },{root:$('catalogModelList'),rootMargin:'160px'}):null;
function paintCover(tile,uri){
  if(!uri||!uri.startsWith('data:image/png;base64,')||!tile)return;
  if(tile.querySelector('img.printer-cover-photo'))return;
  const image=node('img','printer-cover-photo');
  image.src=uri;image.alt='';image.decoding='async';image.loading='lazy';
  image.onerror=()=>{image.remove();tile.classList.remove('has-cover');};
  tile.append(image);tile.classList.add('has-cover');
}
function refreshVisibleCovers(id,uri){
  document.querySelectorAll('.printer-avatar[data-art-id], .target-printer-icon[data-art-id]').forEach((slot)=>{
    if(slot.dataset.artId===id&&!slot.querySelector('img'))paintCover(slot,uri);
  });
}
function pumpCovers(){
  while(coverRunning<4&&coverJobs.length){
    const id=coverJobs.shift();coverRunning++;
    Promise.resolve().then(()=>api()?.PrinterArtwork?.(id)).then((result)=>{
      const uri=result?.found&&result?.provenance==='local_installed_slicer'&&
        typeof result.dataUri==='string'&&result.dataUri.length<800000&&
        result.dataUri.startsWith('data:image/png;base64,')?result.dataUri:null;
      coverCache.set(id,uri);
      refreshVisibleCovers(id,uri);
    }).catch(()=>coverCache.set(id,null)).finally(()=>{
      coverRunning--;coverQueued.delete(id);pumpCovers();
    });
  }
}
function queueCover(id){
  if(!id||coverCache.has(id)||coverQueued.has(id)||!api()?.PrinterArtwork)return;
  coverQueued.add(id);coverJobs.push(id);pumpCovers();
}
function observeCover(tile,id){
  tile.dataset.artId=id;
  if(coverCache.has(id)){paintCover(tile,coverCache.get(id));return;}
  if(coverObserver)coverObserver.observe(tile);
  else queueCover(id); // older embedded WebViews
}

// Target cards share the existing vetted, local-only artwork loader.
function loadTargetArtwork(){
  if(!loaded)return;
  const chosen=new Set(selectedTargetIds());
  for(const model of supported){
    if(!model.legacyTarget||!chosen.has(model.id))continue;
    const tile=document.querySelector('[data-target-row="'+model.legacyTarget+'"] .target-printer-icon');
    if(!tile)continue;
    tile.dataset.artId=model.id;
    if(coverCache.has(model.id))paintCover(tile,coverCache.get(model.id));
    else queueCover(model.id);
  }
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
    const name=node('strong','',picker.title(entry));
    pill.append(name);selectionList.append(pill);
  }
  const term=$('printerCatalogSearch').value.trim();
  const manufacturer=$('printerManufacturerFilter').value;
  const hasFilter=Boolean(term||manufacturer);
  const showList=hasFilter||supported.length>0;
  const source=manufacturer?supported.filter(m=>m.manufacturer===manufacturer):supported;
  const matches=showList?picker.search(source,term,choice,80):{models:[],total:0};
  const list=$('catalogModelList');
  $('printerCatalogSearch').disabled=empty;
  $('printerManufacturerFilter').disabled=empty;
  $('myPrintersDialog').classList.toggle('has-results',showList);
  $('catalogPickerHint').textContent=empty?t('noSupported'):t('startHint');
  $('catalogPickerHint').hidden=showList;
  // No result counts or unsupported-model classifications unless the list is truncated.
  $('catalogSearchCount').hidden=matches.total<=matches.models.length;
  if(coverObserver)coverObserver.disconnect();
  list.replaceChildren();
  list.hidden=!showList;
  for(const model of matches.models){
    const row=node('label','catalog-model-card');
    if(draft.has(model.id))row.classList.add('is-selected');
    const input=node('input');
    input.type='checkbox';input.value=model.id;input.checked=draft.has(model.id);
    input.disabled=saving||(!input.checked&&draft.size>=picker.maxSelected);
    input.setAttribute('aria-label',picker.title(model));
    input.addEventListener('change',()=>{
      if(input.checked && draft.size>=picker.maxSelected){input.checked=false;$('myPrintersStatus').textContent=t('limit');return;}
      if(input.checked)draft.add(model.id);else draft.delete(model.id);
      $('myPrintersStatus').textContent='';
      render();
    });
    const graphic=node('span','printer-avatar');
    graphic.setAttribute('aria-hidden','true');
    graphic.append(node('span','avatar-device','▣'),node('span','avatar-initials',
      String(model.manufacturer||'?').slice(0,2).toLocaleUpperCase()));
    // Local artwork is looked up for recognised built-in models only.
    if(model.legacyTarget)observeCover(graphic,model.id);
    // No third-party images are bundled or remotely fetched.
    const copy=node('span','catalog-model-copy');
    copy.append(node('strong','',model.model),node('small','',model.manufacturer));
    // Show month/year only where an independent launch source is known.
    if(model.releaseDate){
      const label=document.documentElement.lang==='en'?'Launched: ':'השקה: ';
      copy.append(node('small','model-launch-date',label+model.releaseDate.slice(0,7)));
    }
    row.append(input,graphic,copy);list.append(row);
  }
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
  isReady:()=>loaded,
};
})();
