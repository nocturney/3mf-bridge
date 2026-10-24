/* SliceBridge My Printers: offline snapshot + explicit, separate local selection.
 * A curated manufacturer list is the default view; "show all" reveals the full
 * non-Bambu catalogue. Existing saved IDs are ALWAYS included in the view.
 * Selecting a catalogue model saves membership only; conversion reads just the ready
 * destinations (SliceBridgePicker.supportedModels). One card per physical model; its
 * nozzle/slicer variants keep their own target IDs.
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
    manufacturer:'יצרן',chooseManufacturer:'בחר יצרן…',searchByModel:'או חפש דגם',
    noSupported:'עדיין אין יעדים זמינים במחשב זה.',nothingSelected:'עדיין לא נבחרו מדפסות.',
    variants:'דגם, דיזה וסלייסר',nozzleUnspecified:'ללא שיוך דיזה',imageUnavailable:'תמונה אינה זמינה',
    profileSettings:'סלייסר',profileHint:'המערכת תעדיף סלייסר יצרן מותקן עם פרופיל תקין. אפשר לבחור סלייסר חלופי לאותה מדפסת ודיזה; השינוי נשמר רק בלחיצה על ״שמור מדפסות״.',
    profileApply:'שמור העדפה',profileCancel:'ביטול',profileChoose:'יש לבחור סלייסר.',
    slicerAutomatic:'ברירת מחדל — {slicer}',
    showAllMakers:'הצג יצרנים נוספים',showCuratedMakers:'הצג רק יצרנים נבחרים'
  },
  en:{
    title:'My printers',welcome:'Choose printers',welcomeBody:'No printers are selected yet. Choose a manufacturer or search for a model to begin.',
    body:'Add or remove printers without changing your existing nozzle, slicer or filament preferences.',
    search:'Search manufacturer or model',save:'Save printers',cancel:'Not now',selected:'{n} printers selected',
    none:'No models matched',
    showing:'Showing {shown} of {total} models · narrow the search to find more',
    limit:'You can select up to 32 printers.',error:'Could not load or save printer selection.',
    manufacturer:'Manufacturer',chooseManufacturer:'Choose manufacturer…',searchByModel:'Or search a model',
    noSupported:'No destinations are available on this computer yet.',nothingSelected:'No printers selected yet.',
    variants:'Model, nozzle and slicer',nozzleUnspecified:'No nozzle specified',imageUnavailable:'Image unavailable',
    profileSettings:'Slicer',profileHint:'A compatible installed manufacturer slicer is preferred automatically. Override only with a verified slicer for this same printer and nozzle; save using Save printers.',
    profileApply:'Apply preference',profileCancel:'Cancel',profileChoose:'Choose a slicer.',
    slicerAutomatic:'Default — {slicer}',
    showAllMakers:'Show more manufacturers',showCuratedMakers:'Show selected manufacturers only'
  }
};
const changedEvent='slicebridge:my-printers-changed';
let catalog=null,selected=null,draft=null,firstRun=false,loaded=false,saving=false;
let showOtherManufacturers=false;
let activeProfileGroupKey=null;
// UI ONLY. Bambu Lab is retained as a Bambu source maker, but remains excluded
// from destination picking by SliceBridgePicker.isHidden (unchanged).
// Prusa and Prusa Research are one displayed maker; all underlying IDs remain.
const curatedManufacturers=Object.freeze([
  'Anker','Anycubic','Bambu Lab','Creality','ELEGOO','Flashforge','FLSun',
  'Prusa','Prusa Research','QIDI','Snapmaker','Sovol','Voron'
]);
// Avoid dereferencing the optional picker module during script load: a
// missing frontend resource must not crash the app's unrelated conversion UI.
const curatedMakerKeys=new Set(curatedManufacturers.map(name=>
  name==='Prusa'?'prusa research':name.toLocaleLowerCase('en')));
// QIDIStudio's installed "X 5 Series" vendor is a QIDI family only for UI
// grouping; this does not rewrite its ID, source profile or conversion binding.
function displayMakerKey(name){
  const key=picker.makerKey(name);
  return key==='x 5 series'?'qidi':key;
}
function visibleGroup(group){
  return showOtherManufacturers||curatedMakerKeys.has(displayMakerKey(group.manufacturer))||
    Boolean(draft)&&group.ids.some(id=>draft.has(id));
}
// Destination inputs, pushed in by the app shell through setSupportedTargets().
// `listed` is every selectable item (catalogue, installed Orca, PrusaSlicer, per-printer PrusaSlicer);
// `supported` is the ready subset that conversion may use.
let availability=[],installed=[],installedPrusa=[],prusaAvailable=false,listed=[],supported=[],groups=[];
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

// Artwork is resolved per physical model, never per nozzle. Catalogue IDs
// are always tried: some models have no Orca cover candidates but DO have
// exact PrusaSlicer vendor thumbnails resolved by the native backend.
// Only misses resolve an installed target once per slicer, not once per nozzle.
// Missing images show "Image unavailable"; nothing is bundled or downloaded.
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
// A physical printer shares a cover across nozzle diameters in one slicer.
// Resolve one representative per slicer, avoiding repeated full preset scans.
function installedTargetIdsFor(group){
  const seenSlicers=new Set(), ids=[];
  for(const variant of group.variants){
    if(variant.legacyTarget||!installedTargetPattern.test(variant.id))continue;
    const slicer=String(variant.slicer||variant.id).trim().toLocaleLowerCase('en');
    if(seenSlicers.has(slicer))continue;
    seenSlicers.add(slicer);ids.push(variant.id);
  }
  return ids;
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
  // Fast path: catalogue exact names read the same installed slicer resources.
  for(const id of catalogIdsFor(group)){
    const uri=await backendArtwork('PrinterArtwork',id);
    if(uri)return {src:uri};
  }
  // Newly discovered dynamic Orca models may not have catalogue cover hints.
  for(const id of installedTargetIdsFor(group)){
    const uri=await backendArtwork('PrinterArtworkForInstalledTarget',id);
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
  // Group cards use picker.catalogItems(), which removes a repeated vendor
  // prefix from names such as "Anycubic i3 Mega" and "Creality CR-10 S".
  // Index those SAME canonical items, not raw catalogue rows, otherwise
  // their physical keys never agree and the picture RPC is never called.
  // Include models without Orca cover hints: the backend can resolve their
  // exact PrusaSlicer vendor thumbnail instead.
  const canonical=picker.catalogItems(catalog?.models??[],[],[],false,[]);
  for(const model of canonical){
    if(model.id==='prusa-project'||picker.isHidden(model))continue;
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
  const scope=$('showOtherManufacturersBtn');
  scope.textContent=t(showOtherManufacturers?'showCuratedMakers':'showAllMakers');
  scope.setAttribute('aria-pressed',String(showOtherManufacturers));
  $('saveMyPrintersBtn').textContent=t('save');
  $('cancelMyPrintersBtn').textContent=t('cancel');
  $('printerProfileHint').textContent=t('profileHint');
  $('applyPrinterProfileBtn').textContent=t('profileApply');
  $('cancelPrinterProfileBtn').textContent=t('profileCancel');
}
function refreshManufacturerOptions(){
  const filter=$('printerManufacturerFilter');
  const previous=filter.value;
  filter.replaceChildren();
  const placeholder=node('option','',t('chooseManufacturer'));
  placeholder.value='';filter.append(placeholder);
  const makers=picker.manufacturerNames(listed
    .filter(item=>item.id!=='prusa-project'&&
      (showOtherManufacturers||curatedMakerKeys.has(displayMakerKey(item.manufacturer))||
       Boolean(draft)&&draft.has(item.id)))
    .map(item=>displayMakerKey(item.manufacturer)==='qidi'
      ?{...item,manufacturer:'QIDI'}:item));
  for(const maker of makers){
    const option=node('option','',maker);option.value=maker;filter.append(option);
  }
  filter.value=makers.includes(previous)?previous:'';
}
// Show physical nozzle/slicer details without any conversion-capability labels.
function variantLabel(item){
  if(item.legacyTarget)return t('nozzleUnspecified');
  // Spelling aliases are model identities, not conversion/nozzle presets.
  if(!item.nozzle&&!item.slicer)return item.catalogueLabel||picker.title(item);
  return picker.variantText(item)||picker.title(item);
}
// Keep the catalogue uncluttered. The selected-printer settings offer only
// verified slicer alternatives for the SAME physical nozzle. Other profiles
// stay in the backend but are not presented as hardware/nozzle choices.
function eligibleSlicerChoices(group){
  const active=group.variants.find(item=>draft?.has(item.id));
  const defaultItem=group.variants.find(item=>item.id===group.id);
  // No profile dialogue is allowed to change the hardware nozzle. For
  // built-ins the independent printer-settings panel handles that hardware
  // preference; this selector is ONLY for equivalent installed slicers.
  const nozzle=active?.nozzle||defaultItem?.nozzle||null;
  const choices=group.variants.filter(item=>
    item.ready===true&&item.slicer&&item.nozzle===nozzle);
  const bySlicer=new Map();
  for(const item of choices){
    const key=String(item.slicer).toLocaleLowerCase('en').replace(/[\s_-]+/g,'');
    const existing=bySlicer.get(key);
    if(!existing||item.id===active?.id)bySlicer.set(key,item);
  }
  return [...bySlicer.values()];
}
function profileConfigurable(group){
  return eligibleSlicerChoices(group).length>1;
}
function openProfileSettings(group){
  if(saving||!draft)return;
  activeProfileGroupKey=group.key;
  $('printerProfileTitle').textContent=picker.title(group);
  $('printerProfileHint').textContent=t('profileHint');
  $('printerProfileStatus').textContent='';
  const options=$('printerProfileOptions');
  options.replaceChildren();
  const active=group.variants.find(item=>draft.has(item.id));
  const choices=eligibleSlicerChoices(group);
  const currentNozzle=active?.nozzle||choices[0]?.nozzle;
  const auto=picker.preferredSlicerVariant(choices,group.manufacturer,currentNozzle);
  if(!auto)return;
  // The default button selects this exact, already validated app/profile ID;
  // don't misrepresent that persisted ID as a future dynamic "auto" binding.
  // Neither default nor override can infer a different physical nozzle.
  const automatic=node('label','printer-profile-option');
  const defaultRadio=node('input');
  defaultRadio.type='radio';defaultRadio.name='printerProfileChoice';
  defaultRadio.value=auto.id;defaultRadio.checked=active?.id===auto.id;
  automatic.append(defaultRadio,node('span','',t('slicerAutomatic',{slicer:auto.slicer})));
  options.append(automatic);
  for(const variant of choices){
    if(variant.id===auto.id)continue;
    const option=node('label','printer-profile-option');
    const radio=node('input');
    radio.type='radio';radio.name='printerProfileChoice';
    radio.value=variant.id;radio.checked=active?.id===variant.id;
    radio.disabled=saving;
    option.append(radio,node('span','',variant.slicer));
    options.append(option);
  }
  $('printerProfileDialog').showModal();
}
function applyProfileSettings(){
  if(saving||!draft||!activeProfileGroupKey)return;
  const group=groupByKey.get(activeProfileGroupKey);
  if(!group)return;
  const next=[...$('printerProfileOptions').querySelectorAll('input')]
    .find(input=>input.checked);
  if(!next){
    $('printerProfileStatus').textContent=t('profileChoose');
    return;
  }
  const removed=group.ids.filter(id=>draft.has(id)).length;
  if(draft.size-removed>=picker.maxSelected){
    $('printerProfileStatus').textContent=t('limit');
    return;
  }
  // Changing a printer profile replaces only that printer's target ID(s).
  // No slicer profiles, materials, other selected printers or settings are edited.
  for(const id of group.ids)draft.delete(id);
  draft.add(next.value);
  $('printerProfileDialog').close();
  $('myPrintersStatus').textContent='';
  refreshManufacturerOptions();
  render();
}
// One card per physical model. Ticking the card selects its default target; unticking
// removes every target of that model. Individual variants stay independently selectable.
function groupCard(group,index){
  const selectedCount=group.ids.filter(id=>draft.has(id)).length;
  const row=node('div','catalog-model-card');
  row.dataset.printerKey=group.key;
  if(selectedCount)row.classList.add('is-selected');
  const input=node('input');
  input.type='checkbox';input.id='printer-pick-'+index;
  input.value=group.id;input.checked=selectedCount>0;
  input.disabled=saving||(!input.checked&&draft.size>=picker.maxSelected);
  input.setAttribute('aria-label',picker.title(group));
  input.addEventListener('change',()=>{
    if(input.checked){
      if(draft.size>=picker.maxSelected){input.checked=false;$('myPrintersStatus').textContent=t('limit');return;}
      // Re-select the user's original exact profile if they temporarily
      // unchecked this model; never downgrade a saved 0.25 mm to a default.
      const prior=group.ids.find(id=>selected?.selectedIds?.includes(id));
      draft.add(prior||group.id);
    }else{
      for(const id of group.ids)draft.delete(id);
    }
    $('myPrintersStatus').textContent='';
    refreshManufacturerOptions();
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
  // The catalogue is hardware only. Slicer preferences belong with the
  // selected printers above, not on every unselected catalogue model card.
  return row;
}
function render(){
  syncCopy();
  if(!loaded||!draft)return;
  const choice=[...draft];
  const empty=!groups.length;
  // Count PHYSICAL printers and show simple model names, not profiles.
  const entries=[],seenGroups=new Set();
  for(const id of choice){
    const group=groupByItemId.get(id);
    if(!group||seenGroups.has(group.key))continue;
    seenGroups.add(group.key);
    entries.push(group);
  }
  $('myPrintersSelectionCount').textContent=t('selected',{n:entries.length});
  const selectionList=$('selectedPrinterSummary');
  selectionList.replaceChildren();
  $('selectedPrinterSummary').hidden=!entries.length;
  $('selectedPrinterOverview').classList.toggle('is-empty',!entries.length);
  if(!entries.length)selectionList.append(node('span','catalog-muted',t('nothingSelected')));
  for(const group of entries){
    const pill=node('span','selected-printer-pill');
    pill.append(node('strong','',picker.title(group)));
    if(profileConfigurable(group)){
      const settings=node('button','selected-printer-settings',t('profileSettings'));
      settings.type='button';
      settings.setAttribute('aria-label',t('profileSettings')+': '+picker.title(group));
      settings.addEventListener('click',()=>openProfileSettings(group));
      pill.append(settings);
    }
    selectionList.append(pill);
  }
  const term=$('printerCatalogSearch').value.trim();
  const manufacturer=$('printerManufacturerFilter').value;
  // Curated physical manufacturers by default. Selected IDs from other makers
  // stay available and checked; "show more" merely lifts this visual filter.
  const showList=!empty;
  const visible=groups.filter(visibleGroup);
  const source=manufacturer?visible.filter(group=>displayMakerKey(group.manufacturer)===displayMakerKey(manufacturer)):visible;
  const matches=showList?picker.searchGroups(source,term,choice,picker.maxRender):{models:[],total:0};
  const list=$('catalogModelList');
  $('printerCatalogSearch').disabled=empty;
  $('printerManufacturerFilter').disabled=empty;
  $('myPrintersDialog').classList.toggle('has-results',showList);
  $('catalogPickerHint').textContent=t('noSupported');
  $('catalogPickerHint').hidden=showList;
  // Result counts only appear when a filter hides part of the catalogue.
  $('catalogSearchCount').hidden=matches.total<=matches.models.length;
  if(artObserver)artObserver.disconnect();
  slotRegistry.dialog.clear();
  list.replaceChildren();
  list.hidden=!showList;
  // A Chromium/WebView IntersectionObserver can miss newly replaced cards
  // after successive search/manufacturer filters. For an explicit, bounded
  // result set, resolve its exact local images eagerly. The full 603-model
  // catalogue remains lazy, so browsing never starts hundreds of file reads.
  const eagerFilteredArtwork=Boolean(term||manufacturer)&&matches.models.length<=96;
  matches.models.forEach((group,index)=>{
    list.append(groupCard(group,index));
    if(eagerFilteredArtwork)queueArtwork(group);
  });
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
  installedPrusa=Array.isArray(next.installedPrusaTargets)?next.installedPrusaTargets:[];
  prusaAvailable=next.prusaAvailable===true;
  recomputeSupported();
  if(draft)draft=new Set([...draft].filter(id=>listed.some(item=>item.id===id)));
  if(loaded){
    refreshManufacturerOptions();
    loadTargetArtwork();
  }
  if($('myPrintersDialog')?.open)render();
  window.dispatchEvent(new Event(changedEvent));
}
function recomputeSupported(){
  listed=picker?picker.catalogItems(catalog?catalog.models:[],availability,installed,prusaAvailable,installedPrusa):[];
  supported=listed.filter(item=>item.ready);
  groups=picker?picker.groupDestinations(listed):[];
  groupByKey=new Map(groups.map(group=>[group.key,group]));
  groupByItemId=new Map();
  for(const group of groups)for(const variant of group.variants)groupByItemId.set(variant.id,group);
}
function open(onboarding=false){
  if(!loaded||saving)return;
  if($('myPrintersDialog').open)return;
  firstRun=Boolean(onboarding);
  // Every saved ID stays visible and selected here, even when it is not a ready target now.
  draft=new Set(picker.allSavedIds(listed,selected.selectedIds));
  showOtherManufacturers=false;
  refreshManufacturerOptions();
  $('printerCatalogSearch').value='';
  $('printerManufacturerFilter').value='';
  $('myPrintersStatus').textContent='';
  render();
  $('myPrintersDialog').showModal();
  $('printerCatalogSearch').focus();
}
async function save(){
  if(!loaded||!draft||saving||!listed.length)return;
  saving=true;
  $('saveMyPrintersBtn').disabled=true;
  $('myPrintersStatus').textContent='';
  try{
    // Only listed, selected IDs are sent. Stale saved IDs are not sent back.
    const requested=[...draft].filter(id=>listed.some(item=>item.id===id));
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
$('showOtherManufacturersBtn').addEventListener('click',()=>{
  showOtherManufacturers=!showOtherManufacturers;
  refreshManufacturerOptions();
  render();
});
$('cancelPrinterProfileBtn').addEventListener('click',()=>$('printerProfileDialog').close());
$('closePrinterProfileBtn').addEventListener('click',()=>$('printerProfileDialog').close());
$('applyPrinterProfileBtn').addEventListener('click',applyProfileSettings);
$('printerProfileDialog').addEventListener('close',()=>{activeProfileGroupKey=null;});
$('myPrintersDialog').addEventListener('close',()=>{
  if($('printerProfileDialog').open)$('printerProfileDialog').close();
  draft=null;
  firstRun=false;
});
window.SliceBridgeMyPrinters={
  init,setSupportedTargets,selectedTargetIds,isSelectedTarget,legacyTargets,render,
  refreshDestinationArt,
  isReady:()=>loaded,
};
})();
