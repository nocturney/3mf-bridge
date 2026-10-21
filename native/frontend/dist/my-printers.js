/* SliceBridge My Printers: offline snapshot + explicit, separate local selection.
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
    nozzle:'דיזות',none:'לא נמצאו דגמים',
    showing:'מציג {shown} מתוך {total} דגמים · צמצם את החיפוש להצגת דגמים נוספים',
    hidden:'Bambu Lab מוסתרת כרגע כמדפסת יעד. פרויקטי Bambu ו־MakerWorld עדיין נתמכים כמקור.',
    limit:'אפשר לבחור עד 32 מדפסות.',error:'לא ניתן לטעון או לשמור את בחירת המדפסות.',
    manufacturer:'יצרן',chooseManufacturer:'בחר יצרן…',searchByModel:'או חפש דגם',startHint:'בחר יצרן או חפש דגם כדי להציג מדפסות.',nothingSelected:'עדיין לא נבחרו מדפסות.'
  },
  en:{
    title:'My printers',welcome:'Choose printers',welcomeBody:'No printers are selected yet. Choose a manufacturer or search for a model to begin.',
    body:'Add or remove printers without changing your existing nozzle, slicer or filament preferences.',
    search:'Search manufacturer or model',save:'Save printers',cancel:'Not now',selected:'{n} printers selected',
    nozzle:'Nozzles',none:'No models matched',
    showing:'Showing {shown} of {total} models · narrow the search to find more',
    hidden:'Bambu Lab is hidden as a target for now. Bambu and MakerWorld projects remain usable as sources.',
    limit:'You can select up to 32 printers.',error:'Could not load or save printer selection.',
    manufacturer:'Manufacturer',chooseManufacturer:'Choose manufacturer…',searchByModel:'Or search a model',startHint:'Choose a manufacturer or search to see models.',nothingSelected:'No printers selected yet.'
  }
};
let catalog=null,selected=null,draft=null,installed=[],firstRun=false,loaded=false,saving=false;

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
  if(!loaded||!catalog||!selected)return;
  const chosen=new Set(selected.selectedIds||[]);
  for(const model of catalog.models){
    if(!model.legacyTarget||!chosen.has(model.id))continue;
    const tile=document.querySelector('[data-target-row="'+model.legacyTarget+'"] .target-printer-icon');
    if(!tile)continue;
    tile.dataset.artId=model.id;
    if(coverCache.has(model.id))paintCover(tile,coverCache.get(model.id));
    else queueCover(model.id);
  }
}

function legacyTargets(){
  if(!loaded)return null;
  return picker.selectedAdapterTargets(catalog.models,selected.selectedIds);
}
function selectedStatuses(){
  return !loaded?[]:picker.statuses(catalog.models,selected.selectedIds,installed);
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
  $('catalogPickerHint').textContent=t('startHint');
  $('saveMyPrintersBtn').textContent=t('save');
  $('cancelMyPrintersBtn').textContent=t('cancel');
  $('myPrintersHiddenNote').textContent=t('hidden');
}
function render(){
  syncCopy();
  if(!loaded||!draft)return;
  const choice=[...draft];
  $('myPrintersSelectionCount').textContent=t('selected',{n:choice.length});
  const selectionList=$('selectedPrinterSummary');
  selectionList.replaceChildren();
  const entries=picker.statuses(catalog.models,choice,installed);
  $('selectedPrinterSummary').hidden=!entries.length;
  $('selectedPrinterOverview').classList.toggle('is-empty',!entries.length);
  if(!entries.length)selectionList.append(node('span','catalog-muted',t('nothingSelected')));
  for(const entry of entries){
    const pill=node('span','selected-printer-pill');
    const name=node('strong','',picker.title(entry.model));
    pill.append(name);selectionList.append(pill);
  }
  const term=$('printerCatalogSearch').value.trim();
  const manufacturer=$('printerManufacturerFilter').value;
  const hasFilter=Boolean(term||manufacturer);
  const source=manufacturer?catalog.models.filter(m=>m.manufacturer===manufacturer):catalog.models;
  const matches=hasFilter?picker.search(source,term,choice,80):{models:[],total:0};
  const list=$('catalogModelList');
  $('myPrintersDialog').classList.toggle('has-results',hasFilter);
  $('catalogPickerHint').hidden=hasFilter;
  $('catalogSearchCount').hidden=!hasFilter;
  $('myPrintersHiddenNote').hidden=!term.toLocaleLowerCase().includes('bambu');
  if(coverObserver)coverObserver.disconnect();
  list.replaceChildren();
  list.hidden=!hasFilter;
  for(const model of matches.models){
    const row=node('label','catalog-model-card');
    if(draft.has(model.id))row.classList.add('is-selected');
    const input=node('input');
    input.type='checkbox';input.value=model.id;input.checked=draft.has(model.id);
    input.disabled=saving||(!input.checked&&draft.size>=32);
    input.setAttribute('aria-label',picker.title(model));
    input.addEventListener('change',()=>{
      if(input.checked && draft.size>=32){input.checked=false;$('myPrintersStatus').textContent=t('limit');return;}
      if(input.checked)draft.add(model.id);else draft.delete(model.id);
      $('myPrintersStatus').textContent='';
      render();
    });
    const graphic=node('span','printer-avatar');
    graphic.setAttribute('aria-hidden','true');
    graphic.append(node('span','avatar-device','▣'),node('span','avatar-initials',
      String(model.manufacturer||'?').slice(0,2).toLocaleUpperCase()));
    observeCover(graphic,model.id);
    // No third-party images are bundled or remotely fetched.
    const copy=node('span','catalog-model-copy');
    copy.append(node('strong','',model.model),node('small','',model.manufacturer));
    const nozzles=(model.nozzles||[]).join(' / ');
    if(nozzles)copy.append(node('small','nozzle-list',t('nozzle')+': '+nozzles+' mm'));
    row.append(input,graphic,copy);list.append(row);
  }
  if(hasFilter&&!matches.models.length)list.append(node('p','catalog-empty',t('none')));
  $('catalogSearchCount').textContent=t('showing',{
    shown:matches.models.length,total:matches.total
  });
  $('saveMyPrintersBtn').disabled=saving;
}
function setAvailability(next){
  installed=Array.isArray(next)?next:[];
  if(loaded&&$('myPrintersDialog')?.open)render();
}
function open(onboarding=false){
  if(!loaded||saving)return;
  if($('myPrintersDialog').open)return;
  firstRun=Boolean(onboarding);
  draft=new Set(selected.selectedIds);
  $('printerCatalogSearch').value='';
  $('printerManufacturerFilter').value='';
  $('myPrintersStatus').textContent='';
  render();
  $('myPrintersDialog').showModal();
  $('printerCatalogSearch').focus();
}
async function save(){
  if(!loaded||!draft||saving)return;
  saving=true;
  $('saveMyPrintersBtn').disabled=true;
  $('myPrintersStatus').textContent='';
  try{
    const requested=[...draft];
    const reply=await api().SaveMyPrinters(requested);
    if(!reply||reply.onboardingPending||!Array.isArray(reply.selectedIds)){
      throw Error('Saved printer selection was not confirmed');
    }
    selected=reply;
    loadTargetArtwork();
    $('myPrintersDialog').close();
    window.dispatchEvent(new Event('slicebridge:my-printers-changed'));
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
    const filter=$('printerManufacturerFilter');
    filter.replaceChildren();
    const placeholder=node('option','',t('chooseManufacturer'));
    placeholder.value='';filter.append(placeholder);
    const makers=[...new Set(catalog.models.filter(m=>!picker.isHidden(m)).map(m=>m.manufacturer))]
      .sort((a,b)=>a.localeCompare(b,'en',{sensitivity:'base'}));
    for(const maker of makers){const option=node('option','',maker);option.value=maker;filter.append(option);}
    loaded=true;
    loadTargetArtwork();
    $('myPrintersBtn').disabled=false;
    window.dispatchEvent(new Event('slicebridge:my-printers-changed'));
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
  init,setAvailability,legacyTargets,selectedStatuses,render,
  isReady:()=>loaded,
};
})();
