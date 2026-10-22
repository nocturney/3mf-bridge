/* Pure, offline-first destination lookup. No image downloads, DOM injection or IO.
 * Two lists, never mixed up:
 * - catalogItems(): every non-Bambu catalogue model plus installed Orca targets, installed
 *   per-printer PrusaSlicer targets and the PrusaSlicer project item. This is what My Printers
 *   lets the user select. Each item has a `ready` flag saying whether it is a working conversion
 *   destination.
 * - supportedModels(): only the ready items (built-in adapters that are available,
 *   installed Orca targets resolved completely, PrusaSlicer targets the backend marked ready
 *   while the engine probe succeeds).
 *   Conversion reads this list only. Catalogue membership never implies readiness.
 * groupDestinations() folds target IDs into one card per physical model; the IDs
 * themselves (nozzle, slicer, built-in adapter, catalogue ID) are never merged or replaced.
 */
(function(root,factory){
  const api=factory();
  if(typeof module==='object' && module.exports)module.exports=api;
  else root.SliceBridgePicker=api;
})(typeof window==='object'?window:globalThis,function(){
  'use strict';
  const legacyOrder={u1:0,c5:1,cc2:2,qidi:3};
  const prusaTargetId='prusa-project';
  // Same shape the Go backend issues (presets.IsOrcaTargetID). Anything else is not a destination.
  const orcaTargetPattern=/^orca-[a-z0-9]+(?:-[a-z0-9]+)*-[0-9a-f]{10}$/;
  // Per-printer PrusaSlicer targets, shaped like Orca IDs. The generic "prusa-project" never matches.
  const prusaTargetPattern=/^prusa-[a-z0-9]+(?:-[a-z0-9]+)*-[0-9a-f]{10}$/;
  const limit=80;
  // Upper bound for one list render. The full non-Bambu catalogue fits, so nothing is cut off.
  const maxRender=1000;
  const maxSelected=32;
  // My Printers is a hardware inventory, not a list of ranked conversion
  // integrations. Every real manufacturer gets the same alphabetical treatment.
  // Explicit, reviewed spelling aliases only. Anything else is compared casefolded.
  const makerAliases=Object.freeze({'prusa':'prusa research'});
  // Brand casing for canonical makers whose published spelling is not plain title case.
  const makerLabels=Object.freeze({'elegoo':'ELEGOO','qidi':'QIDI','flsun':'FLSun','prusa research':'Prusa Research'});
  // Verified PUBLIC launch dates only, from manufacturer announcements.
  // Unknown releases remain null. Extend this as dated announcements are sourced.
  // QIDI: https://www.globenewswire.com/news-release/2026/08/05/3339368/0/en/QIDI-Plus5-New-Large-3D-Printer-Just-Got-Bigger-Smarter-and-More-Affordable.html
  // Flashforge: https://asia.flashforge.com/products/flashforge-creator-5-pro
  // Snapmaker: https://press.snapmaker.com/ (global retail launch)
  // ELEGOO: https://www.prnewswire.co.uk/news-releases/elegoo-to-launch-multicolor-3d-printer-centauri-carbon-2-combo-on-january-26-302664343.html
  // Prusa: https://blog.prusa3d.com/prusa-core-one-january-update-its-just-around-the-corner_108917/
  const verifiedLaunchSources=Object.freeze({
    'qidi|plus5':'2026-08-05',
    'qidi|x-plus 5':'2026-08-05',
    'flashforge|creator 5 pro':'2026-04-20',
    'snapmaker|u1':'2026-04-10',
    'elegoo|centauri carbon 2 combo':'2026-01-26',
    'prusa|core one':'2025-01',
    'prusa research|core one':'2025-01',
  });
  function clean(value,max=80){
    return typeof value==='string'?value.replace(/\s+/g,' ').trim().slice(0,max):'';
  }
  // Maker identity: trim, casefold, then explicit aliases. "ELEGOO", "Elegoo" and "elegoo" are one maker.
  function makerKey(name){
    const key=clean(name).toLocaleLowerCase('en');
    return makerAliases[key]||key;
  }
  function makerLabel(name){
    const key=makerKey(name);
    return makerLabels[key]||clean(name);
  }
  // Model identity ignores case, spaces, hyphens and underscores only. Digits and words still
  // differ, so "Creator 5" and "Creator 5 Pro", or "Centauri Carbon" and "Centauri Carbon 2", stay distinct.
  function modelKey(model){
    return clean(model,160).toLocaleLowerCase('en').replace(/[\s\-_]+/g,'');
  }
  function physicalKey(manufacturer,model){
    return makerKey(manufacturer)+'|'+modelKey(model);
  }
  // "QIDI Plus5" under maker "QIDI" is the model "Plus5". Used for aliases and installed names.
  function withoutMakerPrefix(name,maker){
    const text=clean(name);
    // Orca's Anker records use "Anker M5C", while the catalogue also uses
    // "AnkerMake M5C". They are the same model, not two physical printers.
    // Longest prefix first; match whole words only (never "AnkerM5C").
    const prefixes=makerKey(maker)==='anker'?['AnkerMake','Anker']:[clean(maker)];
    for(const prefix of prefixes){
      const lead=prefix.toLocaleLowerCase('en')+' ';
      if(text.toLocaleLowerCase('en').startsWith(lead))return text.slice(lead.length).trim();
    }
    return text;
  }
  // Every physical key a model answers to: its own name, then its known aliases.
  function lookupKeys(item){
    const keys=[physicalKey(item?.manufacturer,item?.model)];
    for(const alias of Array.isArray(item?.aliases)?item.aliases:[]){
      keys.push(physicalKey(item?.manufacturer,withoutMakerPrefix(alias,item?.manufacturer)));
    }
    return [...new Set(keys)];
  }
  const verifiedLaunches=Object.freeze(Object.fromEntries(Object.entries(verifiedLaunchSources)
    .map(([source,date])=>{const [maker,model]=source.split('|');return [physicalKey(maker,model),date];})));
  function releaseDate(item){
    const exact=typeof item?.releaseDate==='string'?item.releaseDate:'';
    if(/^\d{4}-\d{2}(?:-\d{2})?$/.test(exact))return exact;
    return verifiedLaunches[physicalKey(item?.manufacturer,item?.model)]||null;
  }
  function manufacturerNames(items){
    const labels=new Map();
    for(const item of items||[]){
      // A generic PrusaSlicer export is a destination, not a manufacturer.
      if(item?.id===prusaTargetId)continue;
      const name=clean(item?.manufacturer);
      if(!name)continue;
      const key=makerKey(name);
      if(!labels.has(key))labels.set(key,makerLabel(name));
    }
    // No featured manufacturers, dividers or conversion-capability tiers.
    return [...labels.values()].sort((a,b)=>a.localeCompare(b,'en',{sensitivity:'base'}));
  }
  function isBambuName(text){return /\bbambu\b/i.test(text)||/^bbl$/i.test(text);}
  function isHidden(model){return (model?.manufacturer||'').toLowerCase()==='bambu lab';}
  function title(model){return ((model?.manufacturer||'')+' '+(model?.model||'')).trim();}
  // Installed machine names carry the nozzle in the file stem ("... 0.4 nozzle", "0.4+0.6").
  // A bare trailing number is never stripped, so "Creator 5" keeps its "5".
  function stripNozzle(name){
    return name.replace(/\s+(?:\d+\.\d+(?:\s*\+\s*\d+\.\d+)*\s*(?:mm)?|nozzle)(?:\s+nozzle)?$/i,'').trim();
  }
  function nozzleOf(target){return clean(target.nozzle,12)||null;}
  function variantText(item){
    if(item.legacyTarget)return '';
    return [item.nozzle?item.nozzle+' mm':null,item.slicer||null].filter(Boolean).join(' · ');
  }
  // One installed machine. `slicer` and `ready` are passed in by the caller: Orca targets are
  // always resolved (the backend lists only those), while a PrusaSlicer target is ready only
  // when the backend validated it and the engine probe succeeded.
  function installedItem(target,fallbackVendor,slicer,ready){
    const vendor=clean(target.vendor)||fallbackVendor;
    const name=stripNozzle(clean(target.title)||clean(target.model));
    if(!name||isBambuName(vendor)||isBambuName(name))return null;
    // "Snapmaker U1" under vendor "Snapmaker" reads as "Snapmaker U1", not "Snapmaker Snapmaker U1".
    const model=withoutMakerPrefix(name,vendor)||name;
    const item={
      id:target.id,manufacturer:makerLabel(vendor),model,aliases:[],legacyTarget:null,
      nozzle:nozzleOf(target),slicer,
    };
    return {...item,releaseDate:releaseDate(item),ready};
  }
  // Defense-in-depth for older backends: the vendor forks bundle many other
  // manufacturers' Orca presets. Those are not manufacturer-owned destinations.
  // Unbranded OrcaSlicer (or an explicit custom profile source) still supports
  // cross-manufacturer models after the backend's compatibility validation.
  const forkMaker=Object.freeze({
    'snapmakerorca':'snapmaker',
    'flashstudio':'flashforge',
    'flashstudiodesktop':'flashforge',
    'elegooslicer':'elegoo',
    'qidistudio':'qidi',
  });
  function matchesInstalledFork(target){
    const app=clean(target?.slicer).toLocaleLowerCase('en').replace(/[\s\-_]+/g,'');
    const owner=forkMaker[app];
    if(!owner)return true;
    const vendor=makerKey(target?.vendor);
    return vendor===owner||(app==='qidistudio'&&vendor==='x 5 series');
  }
  function dynamicItem(target){
    if(!matchesInstalledFork(target))return null;
    return installedItem(target,'Orca',clean(target.slicer)||null,true);
  }
  // Only backend-listed targets with the exact shape count; `ready` must be the literal true.
  function prusaDynamicItem(target,engineReady){
    if(!target||typeof target.id!=='string'||!prusaTargetPattern.test(target.id))return null;
    return installedItem(target,'Prusa','PrusaSlicer',target.ready===true&&engineReady===true);
  }
  // Every destination the My Printers list can show. `ready` is true only for a working
  // conversion target: an available built-in adapter, a resolved Orca target, a backend-ready
  // per-printer PrusaSlicer target, or the generic PrusaSlicer project when its engine is ready.
  // Other catalogue models, and unready PrusaSlicer targets, stay selectable but never convert.
  function catalogItems(models,availability,installedTargets,prusaAvailable,installedPrusaTargets){
    const usable=new Set((Array.isArray(availability)?availability:[])
      .filter(item=>item?.available===true&&typeof item.target==='string')
      .map(item=>item.target));
    const items=[],seen=new Set();
    const add=(item)=>{
      if(!item||seen.has(item.id))return;
      seen.add(item.id);items.push(item);
    };
    for(const model of Array.isArray(models)?models:[]){
      if(!model||typeof model.id!=='string'||isHidden(model))continue;
      const legacy=Object.prototype.hasOwnProperty.call(legacyOrder,model.legacyTarget)?model.legacyTarget:null;
      const manufacturer=makerLabel(model.manufacturer);
      const item={
        id:model.id,
        manufacturer,
        // "Creality CR-10 S" under maker "Creality" is the model "CR-10 S", as for installed targets.
        model:withoutMakerPrefix(clean(model.model),manufacturer)||clean(model.model),
        aliases:(Array.isArray(model.aliases)?model.aliases:[])
          .filter(alias=>typeof alias==='string').map(alias=>clean(alias)).filter(Boolean),
        legacyTarget:legacy,
        nozzle:null,
        slicer:null,
        releaseDate:typeof model.releaseDate==='string'?model.releaseDate:null,
        ready:legacy!==null&&usable.has(legacy),
      };
      add({...item,releaseDate:releaseDate(item)});
    }
    for(const target of Array.isArray(installedTargets)?installedTargets:[]){
      if(!target||typeof target.id!=='string'||!orcaTargetPattern.test(target.id))continue;
      add(dynamicItem(target));
    }
    // Missing or malformed list (older backend): no per-printer PrusaSlicer rows at all.
    for(const target of Array.isArray(installedPrusaTargets)?installedPrusaTargets:[]){
      add(prusaDynamicItem(target,prusaAvailable===true));
    }
    // Generic export target: not a physical maker, so it is never a dropdown brand.
    add({id:prusaTargetId,manufacturer:'Generic',model:'PrusaSlicer project',aliases:['Generic project'],
         legacyTarget:null,nozzle:null,slicer:null,releaseDate:null,ready:prusaAvailable===true});
    return items;
  }
  // Conversion destinations only: the ready subset of catalogItems().
  function supportedModels(models,availability,installedTargets,prusaAvailable,installedPrusaTargets){
    return catalogItems(models,availability,installedTargets,prusaAvailable,installedPrusaTargets)
      .filter(item=>item.ready);
  }
  // One card per physical model. Variants keep their own IDs; nothing is merged or replaced.
  function nozzleNumber(item){
    const value=Number.parseFloat(item?.nozzle);
    return Number.isFinite(value)?value:Number.POSITIVE_INFINITY;
  }
  // My Printers sorts real hardware variants by nozzle and the actual slicer
  // name, never by engine readiness, adapter strength or vendor preference.
  // Conversion capabilities belong only to supportedModels() and the batch UI.
  function variantOrder(left,right){
    const byNozzle=nozzleNumber(left)-nozzleNumber(right);
    if(Number.isFinite(byNozzle)&&byNozzle!==0)return byNozzle;
    if(Number.isFinite(nozzleNumber(left))!==Number.isFinite(nozzleNumber(right)))
      return Number.isFinite(nozzleNumber(left))?-1:1;
    const bySlicer=String(left.slicer||'').localeCompare(String(right.slicer||''),'en',{sensitivity:'base'});
    return bySlicer||String(left.id).localeCompare(String(right.id),'en');
  }
  // Newest verified or catalogued date among a card's variants; null when every date is unknown.
  function latestDate(variants){
    return variants.map(v=>v.releaseDate).filter(Boolean).sort().at(-1)||null;
  }
  function groupDestinations(items){
    const byKey=new Map();
    for(const item of Array.isArray(items)?items:[]){
      if(!item||typeof item.id!=='string')continue;
      const key=physicalKey(item.manufacturer,item.model);
      if(!byKey.has(key))byKey.set(key,[]);
      byKey.get(key).push(item);
    }
    const groups=[];
    for(const [key,variants] of byKey){
      variants.sort(variantOrder);
      // Default card selection depends on physical nozzle (0.4 mm if present),
      // or the canonical catalogue entry. Never auto-rank conversion engines.
      // Existing selected IDs are kept verbatim by My Printers, regardless.
      const preferred=variants.find(v=>v.nozzle==='0.4')||
        variants.find(v=>v.nozzle==null&&v.slicer==null)||variants[0];
      const first=variants[0];
      groups.push({
        key,
        id:preferred.id,
        ids:variants.map(v=>v.id),
        variants,
        ready:variants.some(v=>v.ready),
        manufacturer:first.manufacturer,
        model:first.model,
        aliases:[...new Set(variants.flatMap(v=>v.aliases||[]))],
        legacyTarget:variants.find(v=>v.legacyTarget)?.legacyTarget||null,
        releaseDate:latestDate(variants),
      });
    }
    return groups;
  }
  // Card order: saved selections first (saved order), then maker A-Z, then release date newest
  // first within the maker, undated models after dated ones, alphabetical by model.
  function rankOf(entry,order){
    let best=Number.POSITIVE_INFINITY;
    for(const id of entry.ids||[entry.id]){
      const position=order.get(id);
      if(position!==undefined&&position<best)best=position;
    }
    return best;
  }
  function compareEntries(left,right,order){
    const lrank=rankOf(left,order),rrank=rankOf(right,order);
    if(lrank!==rrank){
      if(!Number.isFinite(lrank))return 1;
      if(!Number.isFinite(rrank))return -1;
      return lrank-rrank;
    }
    const byMaker=String(left.manufacturer).localeCompare(String(right.manufacturer),'en',{sensitivity:'base'});
    if(byMaker)return byMaker;
    const ldate=releaseDate(left),rdate=releaseDate(right);
    if(ldate!==rdate){
      if(!ldate)return 1;
      if(!rdate)return -1;
      return rdate.localeCompare(ldate);
    }
    const byModel=String(left.model).localeCompare(String(right.model),'en',{numeric:true,sensitivity:'base'});
    return byModel||String(left.id).localeCompare(String(right.id),'en');
  }
  function matchEntries(entries,query,selectedIds,maxResults){
    const input=String(query||'').trim().toLocaleLowerCase();
    const order=new Map((selectedIds||[]).map((id,index)=>[id,index]));
    const matches=(entries||[]).filter((entry)=>{
      if(isHidden(entry))return false;
      if(!input)return true;
      return [entry.manufacturer,entry.model,...(entry.aliases||[])]
        .join(' ').toLocaleLowerCase().includes(input);
    });
    matches.sort((left,right)=>compareEntries(left,right,order));
    const count=Math.max(1,Math.min(maxRender,Number(maxResults)||limit));
    return {total:matches.length,models:matches.slice(0,count)};
  }
  // Cards: one entry per physical model, with all its target IDs in .ids.
  function searchGroups(groups,query='',selectedIds=[],maxResults=limit){
    return matchEntries(groups,query,selectedIds,maxResults);
  }
  // Item-level search over raw catalogue or target records. Each record is its own entry.
  function search(items,query='',selectedIds=[],maxResults=limit){
    const entries=(items||[]).map(item=>({...item,ids:[item.id],aliases:item.aliases||[]}));
    return matchEntries(entries,query,selectedIds,maxResults);
  }
  // Saved IDs that are not in the given list (a stale catalogue entry, an uninstalled Orca
  // target) are dropped, de-duplicated and capped at maxSelected, keeping saved order.
  function knownIds(known,selectedIds){
    const out=[],seen=new Set();
    for(const id of Array.isArray(selectedIds)?selectedIds:[]){
      if(typeof id!=='string'||!known.has(id)||seen.has(id))continue;
      seen.add(id);out.push(id);
    }
    return out.slice(0,maxSelected);
  }
  // Conversion IDs only: saved IDs that are ready targets in `items` (pass supportedModels()).
  // A saved catalogue-only model is never a conversion destination, even if it is in the list.
  function selectedTargetIds(items,selectedIds){
    const ready=new Set((items||[]).filter(item=>item?.ready===true).map(item=>item.id));
    return knownIds(ready,selectedIds);
  }
  // Every saved ID the My Printers list can show (pass catalogItems()). Used only to seed
  // the selection dialog, so catalogue membership is kept without implying readiness.
  function allSavedIds(items,selectedIds){
    return knownIds(new Set((items||[]).map(item=>item?.id)),selectedIds);
  }
  function legacyTargetsFor(items,selectedIds){
    const chosen=new Set(selectedTargetIds(items,selectedIds));
    return (items||[]).filter(item=>chosen.has(item.id)&&item.legacyTarget)
      .map(item=>item.legacyTarget);
  }
  return {catalogItems,supportedModels,search,searchGroups,groupDestinations,isHidden,title,releaseDate,
          manufacturerNames,physicalKey,lookupKeys,makerKey,makerLabel,variantText,
          selectedTargetIds,allSavedIds,legacyTargetsFor,maxSelected,maxRender};
});
