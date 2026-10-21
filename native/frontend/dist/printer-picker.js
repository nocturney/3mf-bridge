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
    return {id:target.id,manufacturer:vendor,model:model||name,aliases:[],legacyTarget:null};
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
      });
    }
    for(const target of Array.isArray(installedTargets)?installedTargets:[]){
      if(!target||typeof target.id!=='string'||!orcaTargetPattern.test(target.id))continue;
      add(dynamicItem(target));
    }
    if(prusaAvailable===true){
      add({id:prusaTargetId,manufacturer:'PrusaSlicer',model:'Generic project',aliases:[],legacyTarget:null});
    }
    return items;
  }
  function search(items,query='',selectedIds=[],maxResults=limit){
    const input=String(query||'').trim().toLocaleLowerCase();
    const selected=new Set(selectedIds||[]);
    const matches=(items||[]).filter((item)=>{
      if(isHidden(item))return false;
      if(!input)return true;
      return [item.manufacturer,item.model,...(item.aliases||[])]
        .join(' ').toLocaleLowerCase().includes(input);
    });
    matches.sort((left,right)=>{
      const delta=Number(selected.has(right.id))-Number(selected.has(left.id));
      if(delta)return delta;
      const lr=left.legacyTarget==null?10:legacyOrder[left.legacyTarget]??10;
      const rr=right.legacyTarget==null?10:legacyOrder[right.legacyTarget]??10;
      if(lr!==rr)return lr-rr;
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
  return {supportedModels,search,isHidden,title,selectedTargetIds,legacyTargetsFor,maxSelected};
});
