'use strict';
// Deterministic geometry QA for the My Printers dialog, in installed Chrome via puppeteer-core.
// Loads the real app.css, the real #myPrintersDialog markup from index.html and the real
// printer-picker.js / my-printers.js, with a fixed catalogue and a stubbed Wails App.
// Checks, at 1100x750 and 900x650, in English (LTR) and Hebrew (RTL):
//   - the dialog stays inside the viewport and its frame does not overflow;
//   - the filter row never overlaps the model list, and the list never overflows sideways;
//   - every card contains its own content (no card overflows its box);
//   - every nozzle/slicer label sits inside its card and its text is not truncated;
//   - every visible label is the top-most element at its centre (nothing covers it).
// Run (outside the repo, puppeteer-core is not a project dependency):
//   NODE_PATH=/path/to/node_modules node native/tests/picker-layout-qa.cjs
// CHROME_PATH overrides the browser. Otherwise the newest Puppeteer-cached chrome-headless-shell is
// used, then Google Chrome.
// Exits 1 on any failed check. Screenshots and the fixture are written to a new temp directory.
const fs=require('node:fs');
const os=require('node:os');
const path=require('node:path');
const assert=require('node:assert/strict');

let puppeteer;
try{puppeteer=require('puppeteer-core');}
catch{
  console.error('puppeteer-core is not resolvable. Install it outside the repo and set NODE_PATH.');
  process.exit(1);
}

const dist=path.resolve(__dirname,'../frontend/dist');
const html=fs.readFileSync(path.join(dist,'index.html'),'utf8');
const catalog=JSON.parse(fs.readFileSync(path.join(dist,'printer-catalog.json'),'utf8'));
const dialog=html.slice(html.indexOf('<dialog id="myPrintersDialog"'),html.indexOf('</dialog>',html.indexOf('<dialog id="myPrintersDialog"'))+'</dialog>'.length);
assert.ok(dialog.includes('id="catalogModelList"'),'dialog markup extracted');

// Installed Orca targets for one physical model with five nozzle/slicer variants, plus a second model.
const orcaInstalled=[
  ['0.25','1111111111','OrcaSlicer'],['0.4','2222222222','OrcaSlicer'],['0.6','3333333333','OrcaSlicer'],
  ['0.8','4444444444','OrcaSlicer'],['0.4','5555555555','Flash Studio'],
].map(([nozzle,hex,slicer])=>({
  id:'orca-flashforge-creator-5-pro-'+nozzle.replace('.','')+'-'+hex,vendor:'Flashforge',model:'Creator 5 Pro',
  title:'Flashforge Creator 5 Pro '+nozzle+' nozzle',nozzle,physicalTools:1,slicer,
}));
const saved=['flashforge-creator-5','orca-flashforge-creator-5-pro-04-2222222222','afinia-afinia-h-1-hs'];
const availability=['u1','c5','cc2','qidi'].map(target=>({target,available:true,name:target,slicer:'Installed slicer'}));

const page_script=`
window.__fixture={
  catalog:${JSON.stringify(catalog)},
  saved:${JSON.stringify(saved)},
  availability:${JSON.stringify(availability)},
  installed:${JSON.stringify(orcaInstalled)}
};
window.go={main:{App:{
  PrinterCatalog:async()=>window.__fixture.catalog,
  GetMyPrinters:async()=>({selectedIds:window.__fixture.saved,onboardingPending:false}),
  SaveMyPrinters:async(ids)=>({selectedIds:ids,onboardingPending:false}),
  PrinterArtwork:async()=>({found:false}),
  PrinterArtworkForInstalledTarget:async()=>({found:false}),
}}};
`;

function fixtureHtml(lang,dir){
  return `<!doctype html><html lang="${lang}" dir="${dir}"><head><meta charset="utf-8">
<link rel="stylesheet" href="${path.join(dist,'app.css')}">
<style>*,*::before,*::after{transition:none!important;animation:none!important;caret-color:transparent!important}</style>
</head><body>
<button id="myPrintersBtn" type="button"><span></span></button>
${dialog}
<script>${page_script}</script>
<script src="${path.join(dist,'printer-picker.js')}"></script>
<script src="${path.join(dist,'my-printers.js')}"></script>
<script>
window.__ready=(async()=>{
  const api=window.SliceBridgeMyPrinters;
  api.setSupportedTargets({available:window.__fixture.availability,installedTargets:window.__fixture.installed,prusaAvailable:true});
  await api.init();
})();
</script></body></html>`;
}

// Runs inside the page. Returns every geometry issue it can see.
function measure(){
  const rect=(el)=>el.getBoundingClientRect();
  const issues=[];
  const frame=document.querySelector('#myPrintersDialog .dialog-frame');
  const list=document.getElementById('catalogModelList');
  const filter=document.querySelector('#myPrintersDialog .printer-filter-row');
  const overview=document.getElementById('selectedPrinterOverview');
  const fr=rect(frame),lr=rect(list);
  if(fr.top<-0.5||fr.left<-0.5||fr.bottom>innerHeight+0.5||fr.right>innerWidth+0.5)
    issues.push('dialog outside viewport');
  if(frame.scrollHeight>frame.clientHeight+1)issues.push('dialog frame content overflows its box');
  if(frame.scrollWidth>frame.clientWidth+1)issues.push('dialog frame overflows sideways');
  if(filter&&rect(filter).bottom>lr.top+0.5)issues.push('filter row overlaps model list');
  if(overview&&filter&&rect(overview).bottom>rect(filter).top+0.5)issues.push('selection overview overlaps filter row');
  if(list.scrollWidth>list.clientWidth+1)issues.push('model list scrolls sideways');
  const cards=[...list.querySelectorAll('.catalog-model-card')];
  const labels=[...list.querySelectorAll('.catalog-variant')];
  for(const card of cards){
    const cr=rect(card);
    if(card.scrollHeight>card.clientHeight+1)issues.push('card content overflows its box: '+card.textContent.trim().slice(0,50));
    if(cr.left<lr.left-0.5||cr.right>lr.right+0.5)issues.push('card outside list width: '+card.textContent.trim().slice(0,40));
  }
  for(const label of labels){
    const lb=rect(label),cr=rect(label.closest('.catalog-model-card'));
    const text=label.textContent.trim();
    if(lb.left<cr.left-0.5||lb.right>cr.right+0.5||lb.top<cr.top-0.5||lb.bottom>cr.bottom+0.5)
      issues.push('variant outside its card: '+text);
    if(label.scrollWidth>label.clientWidth+1)issues.push('variant text truncated: '+text);
    const cy=(lb.top+lb.bottom)/2,cx=(lb.left+lb.right)/2;
    if(cy<lr.top||cy>lr.bottom)continue;
    const hit=document.elementFromPoint(cx,cy);
    if(!hit||!label.contains(hit))issues.push('variant covered by another element: '+text+' ('+(hit?hit.className||hit.tagName:'none')+')');
  }
  for(const pill of document.querySelectorAll('#selectedPrinterSummary .selected-printer-pill')){
    const p=rect(pill),o=rect(document.getElementById('selectedPrinterSummary'));
    if(p.bottom>o.bottom+1||p.right>o.right+1||p.left<o.left-1)issues.push('selected pill clipped: '+pill.textContent.trim().slice(0,40));
  }
  return {issues,cards:cards.length,labels:labels.length,
          viewport:{w:innerWidth,h:innerHeight},dir:document.documentElement.dir,
          listClientHeight:list.clientHeight,frameHeight:Math.round(fr.height)};
}

async function scenario(browser,outDir,{name,width,height,lang,dir,filter}){
  const page=await browser.newPage();
  await page.setViewport({width,height,deviceScaleFactor:1});
  const file=path.join(outDir,name+'.html');
  fs.writeFileSync(file,fixtureHtml(lang,dir));
  await page.goto('file://'+file,{waitUntil:'load'});
  await page.evaluate(()=>window.__ready);
  await page.click('#myPrintersBtn');
  if(filter){
    await page.evaluate((value)=>{
      const input=document.getElementById('printerCatalogSearch');
      input.value=value;input.dispatchEvent(new Event('input'));
    },filter);
  }
  await page.evaluate(()=>document.fonts.ready);
  const result=await page.evaluate(measure);
  await page.screenshot({path:path.join(outDir,name+'.png')});
  await page.close();
  return result;
}

function findChrome(){
  const cache=path.join(os.homedir(),'.cache/puppeteer/chrome-headless-shell');
  const builds=fs.existsSync(cache)?fs.readdirSync(cache).sort().reverse():[];
  for(const build of builds){
    const candidate=path.join(cache,build,'chrome-headless-shell-mac-arm64','chrome-headless-shell');
    if(fs.existsSync(candidate))return candidate;
  }
  return '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
}

(async()=>{
  const chrome=process.env.CHROME_PATH||findChrome();
  if(!fs.existsSync(chrome)){console.error('Chrome not found at '+chrome+'; set CHROME_PATH');process.exit(1);}
  const outDir=fs.mkdtempSync(path.join(os.tmpdir(),'picker-layout-qa-'));
  const browser=await puppeteer.launch({executablePath:chrome,headless:true,args:['--no-first-run','--no-default-browser-check']});
  const scenarios=[
    {name:'ltr-1100x750-all',width:1100,height:750,lang:'en',dir:'ltr'},
    {name:'ltr-1100x750-flashforge',width:1100,height:750,lang:'en',dir:'ltr',filter:'Creator 5'},
    {name:'ltr-900x650-all',width:900,height:650,lang:'en',dir:'ltr'},
    {name:'ltr-900x650-flashforge',width:900,height:650,lang:'en',dir:'ltr',filter:'Creator 5'},
    {name:'rtl-1100x750-all',width:1100,height:750,lang:'he',dir:'rtl'},
    {name:'rtl-1100x750-flashforge',width:1100,height:750,lang:'he',dir:'rtl',filter:'Creator 5'},
    {name:'rtl-900x650-all',width:900,height:650,lang:'he',dir:'rtl'},
    {name:'rtl-900x650-flashforge',width:900,height:650,lang:'he',dir:'rtl',filter:'Creator 5'},
  ];
  let failed=0;
  try{
    for(const sc of scenarios){
      const r=await scenario(browser,outDir,sc);
      const ok=r.issues.length===0&&r.cards>0;
      if(!ok)failed++;
      console.log(`${ok?'PASS':'FAIL'} ${sc.name}: ${r.cards} cards, ${r.labels} variant labels, frame ${r.frameHeight}px`);
      for(const issue of r.issues)console.log('   - '+issue);
    }
  } finally{
    await browser.close();
  }
  console.log('screenshots: '+outDir);
  if(failed){console.log(failed+' scenario(s) failed');process.exit(1);}
  console.log('all '+scenarios.length+' layout scenarios passed');
})().catch((err)=>{console.error(err);process.exit(1);});
