/* Pure, offline-first destination lookup. No image downloads, DOM injection or IO.
 * supportedModels() is the only destination list the My Printers UI may show:
 * built-in models whose adapter is available, installed Orca targets the backend
 * resolved completely, and the PrusaSlicer project item when its engine is ready.
 * groupDestinations() folds those target IDs into one card per physical model;
 * the IDs themselves (nozzle, slicer, built-in adapter) are never merged or replaced.
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
  const limit=80;
  const maxSelected=32;
  // Canonical maker keys (casefolded) of integrated and mainstream makers. Detection
  // alone does not claim a working conversion adapter. All other makers remain
  // selectable after the divider. PrusaSlicer is a generic engine, not a maker.
  const primaryMakers=new Set(['snapmaker','flashforge','elegoo','qidi','prusa research',
                              'creality','anycubic','sovol','flsun']);
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
    const text=clean(name),lead=clean(maker).toLocaleLowerCase('en')+' ';
    return text.toLocaleLowerCase('en').startsWith(lead)?text.slice(lead.length).trim():text;
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
  function manufacturerGroups(items){
    const labels=new Map();
    for(const item of items||[]){
      // A generic PrusaSlicer export is a usable destination, not a manufacturer.
      if(item?.id===prusaTargetId)continue;
      const name=clean(item?.manufacturer);
      if(!name)continue;
      const key=makerKey(name);
      if(!labels.has(key))labels.set(key,makerLabel(name));
    }
    const sort=(a,b)=>a.localeCompare(b,'en',{sensitivity:'base'});
    const keys=[...labels.keys()];
    return {
      preferred:keys.filter(key=>primaryMakers.has(key)).map(key=>labels.get(key)).sort(sort),
      others:keys.filter(key=>!primaryMakers.has(key)).map(key=>labels.get(key)).sort(sort),
    };
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
  function dynamicItem(target){
    const vendor=clean(target.vendor)||'Orca';
    const name=stripNozzle(clean(target.title)||clean(target.model));
    if(!name||isBambuName(vendor)||isBambuName(name))return null;
    // "Snapmaker U1" under vendor "Snapmaker" reads as "Snapmaker U1", not "Snapmaker Snapmaker U1".
    const model=withoutMakerPrefix(name,vendor)||name;
    const item={
      id:target.id,manufacturer:makerLabel(vendor),model,aliases:[],legacyTarget:null,
      nozzle:nozzleOf(target),slicer:clean(target.slicer)||null,
    };
    return {...item,releaseDate:releaseDate(item)};
  }
  function supportedModels(models,availability,installedTargets,prusaAvailable){
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
      if(!Object.prototype.hasOwnProperty.call(legacyOrder,model.legacyTarget))continue;
      if(!usable.has(model.legacyTarget))continue;
      const item={
        id:model.id,
        manufacturer:makerLabel(model.manufacturer),
        model:clean(model.model),
        aliases:(Array.isArray(model.aliases)?model.aliases:[])
          .filter(alias=>typeof alias==='string').map(alias=>clean(alias)).filter(Boolean),
        legacyTarget:model.legacyTarget,
        nozzle:null,
        slicer:null,
      };
      add({...item,releaseDate:releaseDate(item)});
    }
    for(const target of Array.isArray(installedTargets)?installedTargets:[]){
      if(!target||typeof target.id!=='string'||!orcaTargetPattern.test(target.id))continue;
      add(dynamicItem(target));
    }
    if(prusaAvailable===true){
      // Generic export target: not a physical maker, so it is never a dropdown brand.
      add({id:prusaTargetId,manufacturer:'Generic',model:'PrusaSlicer project',aliases:['Generic project'],
           legacyTarget:null,nozzle:null,slicer:null,releaseDate:null});
    }
    return items;
  }
  // One card per physical model. Variants keep their own IDs; nothing is merged or replaced.
  function nozzleNumber(item){
    const value=Number.parseFloat(item?.nozzle);
    return Number.isFinite(value)?value:Number.POSITIVE_INFINITY;
  }
  function variantOrder(left,right){
    if(Boolean(left.legacyTarget)!==Boolean(right.legacyTarget))return left.legacyTarget?-1:1;
    const byNozzle=nozzleNumber(left)-nozzleNumber(right);
    if(byNozzle&&Number.isFinite(byNozzle))return byNozzle;
    const bySlicer=String(left.slicer||'').localeCompare(String(right.slicer||''),'en',{sensitivity:'base'});
    return bySlicer||String(left.id).localeCompare(String(right.id),'en');
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
      // Default when a card is ticked: the built-in adapter, else the 0.4 mm variant, else the first.
      const preferred=variants.find(v=>v.legacyTarget)||variants.find(v=>v.nozzle==='0.4')||variants[0];
      const first=variants[0];
      groups.push({
        key,
        id:preferred.id,
        ids:variants.map(v=>v.id),
        variants,
        manufacturer:first.manufacturer,
        model:first.model,
        aliases:[...new Set(variants.flatMap(v=>v.aliases||[]))],
        legacyTarget:variants.find(v=>v.legacyTarget)?.legacyTarget||null,
        releaseDate:variants.map(v=>v.releaseDate).find(Boolean)||null,
      });
    }
    return groups;
  }
  // Maker-level sort: selected cards in saved order, then dated launches newest first,
  // then unknown dates with built-in adapters, then integrated makers, then alphabetical.
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
    const ldate=releaseDate(left),rdate=releaseDate(right);
    if(ldate!==rdate){
      if(!ldate)return 1;
      if(!rdate)return -1;
      return rdate.localeCompare(ldate);
    }
    const lad=Boolean(left.legacyTarget),rad=Boolean(right.legacyTarget);
    if(lad!==rad)return lad?-1:1;
    const lp=primaryMakers.has(makerKey(left.manufacturer));
    const rp=primaryMakers.has(makerKey(right.manufacturer));
    if(lp!==rp)return lp?-1:1;
    const byMaker=String(left.manufacturer).localeCompare(String(right.manufacturer),'en',{sensitivity:'base'});
    if(byMaker)return byMaker;
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
    const count=Math.max(1,Math.min(100,Number(maxResults)||limit));
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
  // Saved IDs that are no longer supported (stale catalogue entries, an uninstalled
  // Orca target) are dropped here, so they are never shown or sent back on save.
  function selectedTargetIds(items,selectedIds){
    const supported=new Set((items||[]).map(item=>item.id));
    const out=[],seen=new Set();
    for(const id of Array.isArray(selectedIds)?selectedIds:[]){
      if(typeof id!=='string'||!supported.has(id)||seen.has(id))continue;
      seen.add(id);out.push(id);
    }
    return out.slice(0,maxSelected);
  }
  function legacyTargetsFor(items,selectedIds){
    const chosen=new Set(selectedTargetIds(items,selectedIds));
    return (items||[]).filter(item=>chosen.has(item.id)&&item.legacyTarget)
      .map(item=>item.legacyTarget);
  }
  return {supportedModels,search,searchGroups,groupDestinations,isHidden,title,releaseDate,
          manufacturerGroups,physicalKey,lookupKeys,makerKey,makerLabel,variantText,
          selectedTargetIds,legacyTargetsFor,maxSelected};
});
