/* Pure, offline-first destination lookup. No image downloads, DOM injection or IO.
 * supportedModels() is the only destination list the My Printers UI may show:
 * built-in models whose adapter is available, installed Orca targets the backend
 * resolved completely, and the PrusaSlicer project item when its engine is ready.
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
  // Preferred mainstream vendors with declared slicer discovery/integration.
  // Detection alone does not claim a working conversion adapter. All other
  // resolved vendors remain selectable after the divider.
  const primaryMakers=new Set(['snapmaker','flashforge','elegoo','qidi',
                              'prusa','prusa research','prusaslicer',
                              'creality','anycubic','sovol','flsun']);
  // Verified PUBLIC launch dates only, from manufacturer announcements.
  // Unknown releases remain null. Extend this as dated announcements are sourced.
  // QIDI: https://www.globenewswire.com/news-release/2026/08/05/3339368/0/en/QIDI-Plus5-New-Large-3D-Printer-Just-Got-Bigger-Smarter-and-More-Affordable.html
  // Flashforge: https://asia.flashforge.com/products/flashforge-creator-5-pro
  // Snapmaker: https://press.snapmaker.com/ (global retail launch)
  // ELEGOO: https://www.prnewswire.co.uk/news-releases/elegoo-to-launch-multicolor-3d-printer-centauri-carbon-2-combo-on-january-26-302664343.html
  // Prusa: https://blog.prusa3d.com/prusa-core-one-january-update-its-just-around-the-corner_108917/
  const verifiedLaunches=Object.freeze({
    'qidi|plus5':'2026-08-05',
    'qidi|x-plus 5':'2026-08-05',
    'flashforge|creator 5 pro':'2026-04-20',
    'snapmaker|u1':'2026-04-10',
    'elegoo|centauri carbon 2 combo':'2026-01-26',
    'prusa|core one':'2025-01',
    'prusa research|core one':'2025-01',
  });
  function normalized(value){return clean(value).toLocaleLowerCase();}
  function releaseDate(item){
    const exact=typeof item?.releaseDate==='string'?item.releaseDate:'';
    if(/^\d{4}-\d{2}(?:-\d{2})?$/.test(exact))return exact;
    return verifiedLaunches[normalized(item?.manufacturer)+'|'+normalized(item?.model)]||null;
  }
  function manufacturerGroups(items){
    const names=[...new Set((items||[]).map(item=>clean(item?.manufacturer)).filter(Boolean))];
    const sort=(a,b)=>a.localeCompare(b,'en',{sensitivity:'base'});
    return {
      preferred:names.filter(name=>primaryMakers.has(normalized(name))).sort(sort),
      others:names.filter(name=>!primaryMakers.has(normalized(name))).sort(sort),
    };
  }
  function clean(value,max=80){
    return typeof value==='string'?value.replace(/\s+/g,' ').trim().slice(0,max):'';
  }
  function isBambuName(text){return /\bbambu\b/i.test(text)||/^bbl$/i.test(text);}
  function isHidden(model){return (model?.manufacturer||'').toLowerCase()==='bambu lab';}
  function title(model){return ((model?.manufacturer||'')+' '+(model?.model||'')).trim();}
  function dynamicItem(target){
    const vendor=clean(target.vendor)||'Orca';
    const name=clean(target.title)||clean(target.model);
    if(!name||isBambuName(vendor)||isBambuName(name))return null;
    // "Snapmaker U1" under vendor "Snapmaker" reads as "Snapmaker U1", not "Snapmaker Snapmaker U1".
    const lead=vendor.toLocaleLowerCase()+' ';
    const model=name.toLocaleLowerCase().startsWith(lead)?name.slice(lead.length).trim():name;
    const item={id:target.id,manufacturer:vendor,model:model||name,aliases:[],legacyTarget:null};
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
      add({
        id:model.id,
        manufacturer:clean(model.manufacturer),
        model:clean(model.model),
        aliases:(Array.isArray(model.aliases)?model.aliases:[])
          .filter(alias=>typeof alias==='string').map(alias=>clean(alias)).filter(Boolean),
        legacyTarget:model.legacyTarget,
        releaseDate:releaseDate(model),
      });
    }
    for(const target of Array.isArray(installedTargets)?installedTargets:[]){
      if(!target||typeof target.id!=='string'||!orcaTargetPattern.test(target.id))continue;
      add(dynamicItem(target));
    }
    if(prusaAvailable===true){
      add({id:prusaTargetId,manufacturer:'PrusaSlicer',model:'Generic project',aliases:[],legacyTarget:null,releaseDate:null});
    }
    return items;
  }
  function search(items,query='',selectedIds=[],maxResults=limit){
    const input=String(query||'').trim().toLocaleLowerCase();
    const selectedOrder=new Map((selectedIds||[]).map((id,index)=>[id,index]));
    const matches=(items||[]).filter((item)=>{
      if(isHidden(item))return false;
      if(!input)return true;
      return [item.manufacturer,item.model,...(item.aliases||[])]
        .join(' ').toLocaleLowerCase().includes(input);
    });
    matches.sort((left,right)=>{
      // Explicit user selection always leads, in the saved choice order.
      const lpos=selectedOrder.get(left.id),rpos=selectedOrder.get(right.id);
      if(lpos!==undefined||rpos!==undefined){
        if(lpos===undefined)return 1;
        if(rpos===undefined)return -1;
        return lpos-rpos;
      }
      // Newest publicly verified launch first. A missing date is not a
      // reason to invent a year from a model number or profile filename.
      const ldate=releaseDate(left),rdate=releaseDate(right);
      if(ldate!==rdate){
        if(!ldate)return 1;
        if(!rdate)return -1;
        return rdate.localeCompare(ldate);
      }
      // Unknown-date fallback favors the integrated families instead of
      // surfacing alphabetically early, obscure vendors after My Printers.
      const lp=primaryMakers.has(normalized(left.manufacturer));
      const rp=primaryMakers.has(normalized(right.manufacturer));
      if(lp!==rp)return lp?-1:1;
      const byMaker=String(left.manufacturer).localeCompare(String(right.manufacturer),'en',{sensitivity:'base'});
      if(byMaker)return byMaker;
      const byModel=String(left.model).localeCompare(String(right.model),'en',{numeric:true,sensitivity:'base'});
      return byModel||String(left.id).localeCompare(String(right.id),'en');
    });
    const count=Math.max(1,Math.min(100,Number(maxResults)||limit));
    return {total:matches.length,models:matches.slice(0,count)};
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
  return {supportedModels,search,isHidden,title,releaseDate,manufacturerGroups,
          selectedTargetIds,legacyTargetsFor,maxSelected};
});
