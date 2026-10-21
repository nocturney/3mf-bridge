/* Pure, offline-first model lookup. No image downloads, DOM injection or IO. */
(function(root,factory){
  const api=factory();
  if(typeof module==='object' && module.exports)module.exports=api;
  else root.SliceBridgePicker=api;
})(typeof window==='object'?window:globalThis,function(){
  'use strict';
  const order={u1:0,c5:1,cc2:2,qidi:3};
  const limit=80;
  function isHidden(model){return (model?.manufacturer||'').toLowerCase()==='bambu lab';}
  function title(model){return ((model?.manufacturer||'')+' '+(model?.model||'')).trim();}
  function search(models,query='',selectedIds=[],maxResults=limit){
    const input=String(query||'').trim().toLocaleLowerCase();
    const selected=new Set(selectedIds||[]);
    const matches=(models||[]).filter((model)=>{
      if(isHidden(model))return false; // kept in snapshots; Bambu is SOURCE only for now
      if(!input)return true;
      return [model.manufacturer,model.model,...(model.aliases||[])]
        .join(' ').toLocaleLowerCase().includes(input);
    });
    matches.sort((left,right)=>{
      const delta=Number(selected.has(right.id))-Number(selected.has(left.id));
      if(delta)return delta;
      const lr=left.legacyTarget==null?10:order[left.legacyTarget]??10;
      const rr=right.legacyTarget==null?10:order[right.legacyTarget]??10;
      if(lr!==rr)return lr-rr;
      const byMaker=String(left.manufacturer).localeCompare(String(right.manufacturer),'en',{sensitivity:'base'});
      if(byMaker)return byMaker;
      const byModel=String(left.model).localeCompare(String(right.model),'en',{numeric:true,sensitivity:'base'});
      return byModel||String(left.id).localeCompare(String(right.id),'en');
    });
    const count=Math.max(1,Math.min(100,Number(maxResults)||limit));
    return {total:matches.length,models:matches.slice(0,count)};
  }
  function selectedAdapterTargets(models,ids){
    const lookup=new Map((models||[]).map(m=>[m.id,m]));
    return (ids||[]).map(id=>lookup.get(id)).filter(
      model=>model && !isHidden(model) && model.legacyTarget &&
             Object.prototype.hasOwnProperty.call(order,model.legacyTarget)
    ).map(model=>model.legacyTarget);
  }
  function statuses(models,selectedIds,availability){
    const catalog=new Map((models||[]).map(model=>[model.id,model]));
    const avail=new Map((availability||[]).map(item=>[item.target,item]));
    return (selectedIds||[]).map(id=>{
      const model=catalog.get(id);
      if(!model || isHidden(model))return null;
      if(!model.legacyTarget)return {model,kind:'catalog_only',canConvert:false};
      const local=avail.get(model.legacyTarget);
      return {model,kind:local?.available?'ready':'missing_local_profile',
              canConvert:!!local?.available,error:local?.error||''};
    }).filter(Boolean);
  }
  return {search,isHidden,title,selectedAdapterTargets,statuses};
});
