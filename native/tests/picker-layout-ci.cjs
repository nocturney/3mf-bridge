'use strict';
// Rendered-geometry QA for the My Printers dialog in a real Chromium or Edge renderer.
// No npm packages: the browser runs headless with --dump-dom and reports measurements
// through a data attribute on <html>. The page is the real frontend index.html with its
// #myPrintersDialog markup, app.css, printer-picker.js and my-printers.js, plus a synthetic
// Wails App that returns the committed catalogue, five saved IDs and installed Creator 5 Pro
// nozzle/slicer variants. No Wails RPC, printer or slicer action is invoked.
// Run from the repo root or native/:  node native/tests/picker-layout-ci.cjs
// CHROME_BIN overrides the browser. PICKER_LAYOUT_CI_SCREENSHOTS=<dir> also writes one PNG per layout.
// Exits 1 on any failed check, when no Chromium/Edge is found, or when a launched browser produces
// no render marker. Nothing is reported as passed.
const fs=require('node:fs');
const os=require('node:os');
const path=require('node:path');
const {spawnSync}=require('node:child_process');
const assert=require('node:assert/strict');
const {pathToFileURL}=require('node:url');

const dist=path.resolve(__dirname,'../frontend/dist');
const indexHtml=fs.readFileSync(path.join(dist,'index.html'),'utf8');
const catalog=JSON.parse(fs.readFileSync(path.join(dist,'printer-catalog.json'),'utf8'));
// Non-Bambu catalogue IDs plus the generic PrusaSlicer item: 590 listed IDs, 582 physical cards.
const catalogIds=catalog.models.filter(model=>model.manufacturer.toLowerCase()!=='bambu lab').map(model=>model.id);
assert.equal(catalogIds.length,589,'catalogue should hold 589 non-Bambu IDs');

// Edge first, then Chrome, on Windows and macOS. Linux keeps Chrome/Chromium as a fallback.
function browserCandidates(){
  if(process.env.CHROME_BIN)return [process.env.CHROME_BIN];
  if(process.platform==='win32')return [
    'C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe',
    'C:\\Program Files\\Microsoft\\Edge\\Application\\msedge.exe',
    'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe',
    'C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe'
  ];
  if(process.platform==='darwin')return [
    '/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge',
    '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'
  ];
  return ['/usr/bin/microsoft-edge','/usr/bin/google-chrome','/usr/bin/chromium'];
}
const browsers=[...new Set(browserCandidates())].filter(candidate=>fs.existsSync(candidate));
if(!browsers.length){
  console.error('UNTESTED: no Chromium or Edge binary found. Tried: '+browserCandidates().join(' | '));
  process.exit(1);
}
console.log('CHROMIUM_CI_CANDIDATES',browsers.map(browser=>path.basename(browser)).join(' | '));

// Synthetic offline data. Five saved IDs per scenario; installed variants are the Creator 5 Pro
// nozzle/slicer targets an Orca-based install would expose.
const availability=['u1','c5','cc2','qidi'].map(target=>({target,available:true}));
const installedTargets=[
  ['0.25','1111111111','Flash Studio Desktop'],['0.4','2222222222','OrcaSlicer'],['0.6','3333333333','OrcaSlicer'],
  ['0.8','4444444444','OrcaSlicer'],['0.4','5555555555','Flash Studio'],
  ['0.25','6666666666','OrcaSlicer'],
].map(([nozzle,hex,slicer])=>({
  id:'orca-flashforge-creator-5-pro-'+nozzle.replace('.','')+'-'+hex,vendor:'Flashforge',model:'Creator 5 Pro',
  title:'Flashforge Creator 5 Pro '+nozzle+' nozzle',nozzle,slicer,
}));
// The user's five physical printers, including the exact 0.25mm Pro profile.
const savedWithInstalled=['elegoo-centauri-carbon-2','flashforge-creator-5',
  'qidi-x-plus-5','snapmaker-u1','orca-flashforge-creator-5-pro-025-1111111111'];
const savedCatalogueOnly=['flashforge-creator-5','afinia-afinia-h-1-hs','sovol-sv06-plus',
  'qidi-x-plus-5','elegoo-centauri-carbon-2'];

// Layouts cover both curated-default and user-expanded full-catalogue views,
// desktop/compact windows and English LTR / Hebrew RTL; no network or device writes.
const scenarios=[
  {name:'en-ltr-1100x750',lang:'en',dir:'ltr',width:1100,height:750,filter:'',installed:true},
  {name:'en-ltr-1100x750-expanded',lang:'en',dir:'ltr',width:1100,height:750,filter:'',installed:true,expanded:true},
  {name:'en-ltr-1100x750-creator5pro',lang:'en',dir:'ltr',width:1100,height:750,filter:'Creator 5 Pro',installed:true},
  {name:'he-rtl-1100x750',lang:'he',dir:'rtl',width:1100,height:750,filter:'',installed:true},
  {name:'he-rtl-1100x750-creator5pro',lang:'he',dir:'rtl',width:1100,height:750,filter:'Creator 5 Pro',installed:true},
  {name:'en-ltr-900x650-creator5pro',lang:'en',dir:'ltr',width:900,height:650,filter:'Creator 5 Pro',installed:true},
  {name:'he-rtl-900x650',lang:'he',dir:'rtl',width:900,height:650,filter:'',installed:true},
  {name:'he-rtl-900x650-expanded',lang:'he',dir:'rtl',width:900,height:650,filter:'',installed:true,expanded:true},
  {name:'he-rtl-900x650-creator5pro',lang:'he',dir:'rtl',width:900,height:650,filter:'Creator 5 Pro',installed:true},
  {name:'en-ltr-1100x750-catalogue-mapping',lang:'en',dir:'ltr',width:1100,height:750,filter:'',installed:false},
  {name:'en-ltr-1100x750-catalogue-all-ids',lang:'en',dir:'ltr',width:1100,height:750,filter:'',installed:false,expanded:true},
];

// Runs inside the page after my-printers.js. Opens the real dialog and reports its geometry.
function pageMain(config){
  const root=document.documentElement;
  const fixture=window.__fixture;
  root.lang=config.lang;root.dir=config.dir;document.body.dir=config.dir;
  const api=window.SliceBridgeMyPrinters;
  const fail=(error)=>root.setAttribute('data-picker-layout-ci-error',encodeURIComponent(String(error&&error.stack||error)));
  const J=(r)=>JSON.stringify(Object.fromEntries(Object.entries(r).map(([k,v])=>[k,Math.round(v)])));
  const text=(el)=>(el?.textContent||'').replace(/\s+/g,' ').trim();
  const rectOf=(el)=>{const r=el.getBoundingClientRect();return {x:r.left,y:r.top,right:r.right,bottom:r.bottom,w:r.width,h:r.height};};
  const shown=(r)=>r.w>0&&r.h>0;
  const inside=(r,outer)=>r.x>=outer.x-1&&r.y>=outer.y-1&&r.right<=outer.right+1&&r.bottom<=outer.bottom+1;
  const overlap=(a,b)=>Math.max(0,Math.min(a.right,b.right)-Math.max(a.x,b.x))*
    Math.max(0,Math.min(a.bottom,b.bottom)-Math.max(a.y,b.y));
  const describe=(el)=>el?el.tagName.toLowerCase()+' class="'+(el.getAttribute('class')||'')+'" "'+text(el).slice(0,30)+'"':'none';
  function measure(){
    const issues=[];
    const dialog=document.getElementById('myPrintersDialog');
    const frame=dialog.querySelector('.dialog-frame');
    const scroll=dialog.querySelector('.dialog-scroll');
    const list=document.getElementById('catalogModelList');
    const summary=document.getElementById('selectedPrinterSummary');
    const fr=rectOf(frame),sr=rectOf(summary),lr=rectOf(list),scrollArea=rectOf(scroll);
    if(!dialog.open)issues.push('dialog did not open');
    if(fr.x<-0.5||fr.y<-0.5||fr.right>innerWidth+0.5||fr.bottom>innerHeight+0.5)
      issues.push('dialog frame outside viewport: '+J(fr)+' viewport '+innerWidth+'x'+innerHeight);
    if(frame.scrollHeight>frame.clientHeight+1)
      issues.push('dialog frame clips content vertically: scroll '+frame.scrollHeight+' client '+frame.clientHeight);
    if(frame.scrollWidth>frame.clientWidth+1)
      issues.push('dialog frame clips content sideways: scroll '+frame.scrollWidth+' client '+frame.clientWidth);
    // Stacked blocks in the dialog must never overlap each other.
    const blocks=[['selection',document.getElementById('selectedPrinterOverview')],
      ['filter',dialog.querySelector('.printer-filter-row')],['hint',document.getElementById('catalogPickerHint')],
      ['count',document.getElementById('catalogSearchCount')],['list',list]]
      .map(([name,el])=>[name,rectOf(el)]).filter(([,r])=>shown(r));
    for(let i=0;i<blocks.length;i++)for(let j=i+1;j<blocks.length;j++){
      if(overlap(blocks[i][1],blocks[j][1])>2)
        issues.push('blocks overlap: '+blocks[i][0]+' '+J(blocks[i][1])+' and '+blocks[j][0]+' '+J(blocks[j][1]));
    }
    if(summary.scrollHeight>summary.clientHeight+1)
      issues.push('selection summary scrolls, hiding pills: scroll '+summary.scrollHeight+' client '+summary.clientHeight);
    const pills=[...summary.querySelectorAll('.selected-printer-pill')];
    for(const pill of pills){
      const pr=rectOf(pill);
      // At short heights the summary intentionally scrolls horizontally. A
      // pill outside the viewport in X is fine; clipping its NAME or settings
      // button vertically (or inside its own bounds) is never acceptable.
      if(pr.y<sr.y-1||pr.bottom>sr.bottom+1)
        issues.push('selected pill clipped vertically: "'+text(pill)+'" '+J(pr)+' summary '+J(sr));
      if(pill.scrollWidth>pill.clientWidth+1||pill.scrollHeight>pill.clientHeight+1)
        issues.push('selected pill content clipped: "'+text(pill)+'"');
      const name=pill.querySelector('strong'),action=pill.querySelector('.selected-printer-settings');
      if(name&&name.scrollWidth>name.clientWidth+1)
        issues.push('selected printer name truncated: "'+text(name)+'"');
      if(action&&!inside(rectOf(action),pr))
        issues.push('slicer action outside selected printer pill: '+J(rectOf(action)));
    }
    // The part of a box that is visible: inside the list, the dialog scroll area and the viewport.
    const visiblePart=(r)=>{
      const l=rectOf(list),s=rectOf(scroll);
      const part={x:Math.max(r.x,l.x,s.x,0),y:Math.max(r.y,l.y,s.y,0),
        right:Math.min(r.right,l.right,s.right,innerWidth),bottom:Math.min(r.bottom,l.bottom,s.bottom,innerHeight)};
      return part.right-part.x>1&&part.bottom-part.y>1?part:null;
    };
    const centre=(r)=>[(r.x+r.right)/2,(r.y+r.bottom)/2];
    const scope=document.getElementById('showOtherManufacturersBtn');
    const makers=[...document.getElementById('printerManufacturerFilter').options]
      .map(option=>option.value).filter(Boolean);
    const cards=[...list.querySelectorAll('.catalog-model-card')];
    const cardReport=[];
    let variantTotal=0,hitTests=0;
    for(const card of cards){
      const cr=rectOf(card),name=text(card.querySelector('.catalog-model-copy strong'));
      const variants=[...card.querySelectorAll('.catalog-variant')];
      variantTotal+=variants.length;
      cardReport.push({name,key:card.dataset.printerKey,variants:variants.map(text),
        hasProfileSettings:Boolean(card.querySelector('.catalog-model-profile-button')),
        values:[...card.querySelectorAll('input[type="checkbox"]')].map(input=>input.value)});
      if(cr.x<lr.x-1||cr.right>lr.right+1)
        issues.push('card outside model grid width: '+name+' card '+J(cr)+' grid '+J(lr));
      if(card.scrollHeight>card.clientHeight+1||card.scrollWidth>card.clientWidth+1)
        issues.push('card clips its content: '+name+' scroll '+card.scrollWidth+'x'+card.scrollHeight+
          ' client '+card.clientWidth+'x'+card.clientHeight);
      for(const el of card.querySelectorAll('*')){
        const r=rectOf(el);
        if(shown(r)&&!inside(r,cr))
          issues.push('card child outside its card: '+name+' <'+el.tagName.toLowerCase()+'> "'+text(el).slice(0,40)+
            '" '+J(r)+' card '+J(cr));
      }
      for(const variant of variants){
        const span=variant.querySelector('span');
        if(span&&span.scrollWidth>span.clientWidth+1)
          issues.push('variant label truncated: "'+text(variant)+'" scroll '+span.scrollWidth+' client '+span.clientWidth);
      }
    }
    // A list squeezed below one card's height shows only a strip of it; the rest needs a nested scroll.
    const firstCard=cards[0]?rectOf(cards[0]):null;
    if(firstCard&&lr.h<firstCard.h-1)
      issues.push('model list squeezed below one card height: list '+J(lr)+' first card '+J(firstCard));
    // Hit tests: scroll each card into the dialog scroll area first, then probe the centre of the
    // visible part of the card and of each variant. The top-most element there must belong to it.
    for(const card of cards){
      card.scrollIntoView({block:'nearest'});
      const name=text(card.querySelector('.catalog-model-copy strong'));
      const cardPart=visiblePart(rectOf(card));
      if(cardPart){
        const [x,y]=centre(cardPart),hit=document.elementFromPoint(x,y);
        hitTests++;
        if(!card.contains(hit))
          issues.push('card covered by another element: "'+name+'" visible '+J(cardPart)+' hit '+describe(hit));
      }
      for(const variant of card.querySelectorAll('.catalog-variant')){
        const part=visiblePart(rectOf(variant));
        if(!part)continue;
        const [x,y]=centre(part),hit=document.elementFromPoint(x,y);
        hitTests++;
        if(!variant.contains(hit))
          issues.push('variant covered by another element: "'+text(variant)+'" visible '+J(part)+' hit '+describe(hit));
      }
    }
    // The user sees ONE checkbox per physical printer; saved IDs may be
    // nozzle/profile IDs different from the card's canonical default ID.
    const picker=window.SliceBridgePicker;
    const knownGroups=picker.groupDestinations(
      picker.catalogItems(fixture.catalog.models,fixture.availability,fixture.installed,true));
    const saved=config.saved.map(id=>{
      const group=knownGroups.find(group=>group.ids.includes(id));
      const card=cards.find(card=>card.dataset.printerKey===group?.key);
      const input=card?.querySelector('input[type="checkbox"]');
      return {id,present:Boolean(card),checked:Boolean(input?.checked)};
    });
    return {issues,viewport:{width:innerWidth,height:innerHeight},lang:root.lang,dir:root.dir,
      frame:fr,scrollArea,list:lr,firstCard,
      cardCount:cards.length,variantCount:variantTotal,hitTests,
      pills:pills.map(text),selectionCount:text(document.getElementById('myPrintersSelectionCount')),
      scopeExpanded:scope.getAttribute('aria-pressed')==='true',scopeLabel:text(scope),
      makers,searchCount:text(document.getElementById('catalogSearchCount')),saved,cards:cardReport};
  }
  if(!api||typeof api.init!=='function')return fail('SliceBridgeMyPrinters did not load');
  api.setSupportedTargets({available:fixture.availability,installedTargets:fixture.installed,prusaAvailable:true});
  api.init().then(async()=>{
    const button=document.getElementById('myPrintersBtn');
    if(!api.isReady()||button.disabled)throw Error('My Printers did not finish init');
    button.click();
    const dialog=document.getElementById('myPrintersDialog');
    if(!dialog.open)throw Error('My Printers dialog did not open');
    if(config.expanded)document.getElementById('showOtherManufacturersBtn').click();
    if(config.filter){
      const search=document.getElementById('printerCatalogSearch');
      search.value=config.filter;search.dispatchEvent(new Event('input'));
    }
    await document.fonts.ready;
    const report=measure();
    // Hardware catalogue has NO settings action. Slicer alternatives are
    // configured for an already selected model in the selected-printer summary.
    if(config.installed&&config.filter==='Creator 5 Pro'){
      const creator=[...document.querySelectorAll('.catalog-model-card')]
        .find(card=>text(card.querySelector('.catalog-model-copy strong'))==='Creator 5 Pro');
      if(creator?.querySelector('.catalog-model-profile-button'))
        throw Error('Technical settings action leaked onto a catalogue card');
      const pill=[...document.querySelectorAll('.selected-printer-pill')]
        .find(row=>text(row).includes('Creator 5 Pro'));
      const button=pill?.querySelector('.selected-printer-settings');
      if(!button)throw Error('Selected Creator 5 Pro lacks a slicer preference in the selected summary');
      button.click();
      const profile=document.getElementById('printerProfileDialog');
      const profileFrame=profile.querySelector('.dialog-frame');
      const profileRect=rectOf(profileFrame);
      report.profile={open:profile.open,frame:profileRect,
        contentClipped:profileFrame.scrollHeight>profileFrame.clientHeight+1||
          profileFrame.scrollWidth>profileFrame.clientWidth+1,
        choices:[...document.querySelectorAll('#printerProfileOptions input[type="radio"]')]
          .map(input=>({id:input.value,checked:input.checked}))};
      document.getElementById('cancelPrinterProfileBtn').click();
    }
    root.setAttribute('data-picker-layout-ci',encodeURIComponent(JSON.stringify(report)));
  }).catch(fail);
}

// Builds the dialog document: the real index.html, minus its scripts, with the real app.css and
// the two real My Printers scripts. Absolute file URLs point into the frontend dist; nothing
// is copied.
function dialogDocument(scenario){
  const fileUrl=(name)=>pathToFileURL(path.join(dist,name)).href;
  const fixture={
    catalog,
    availability,
    installed:scenario.installed?installedTargets:[],
    saved:scenario.installed?savedWithInstalled:savedCatalogueOnly,
  };
  const json=(value)=>JSON.stringify(value).replace(/</g,'\\u003c');
  const driver={lang:scenario.lang,dir:scenario.dir,filter:scenario.filter,
    installed:scenario.installed===true,expanded:scenario.expanded===true,saved:fixture.saved};
  // Only five known, checked-in deferred script includes exist in the real index.
  // Remove those exact tags, rather than attempting to sanitize HTML with a
  // general regex (which cannot safely recognize all closing-script syntax).
  const expectedScripts=['printer-picker.js','app.js','my-printers.js','integrations.js','about.js'];
  let stripped=indexHtml;
  for(const script of expectedScripts){
    const exactTag='<script defer src="./'+script+'"></script>';
    assert.equal(stripped.split(exactTag).length,2,'unexpected source script markup: '+script);
    stripped=stripped.replace(exactTag,'');
  }
  stripped=stripped.replace('href="./app.css"','href="'+fileUrl('app.css')+'"');
  return stripped.replace('</body>',()=>[
    '<style>*,*::before,*::after{transition:none!important;animation:none!important;caret-color:transparent!important}</style>',
    '<script>window.__fixture='+json(fixture)+';'+
      'window.go={main:{App:{'+
        'PrinterCatalog:async()=>window.__fixture.catalog,'+
        'GetMyPrinters:async()=>({selectedIds:window.__fixture.saved,onboardingPending:false}),'+
        'SaveMyPrinters:async(ids)=>({selectedIds:ids,onboardingPending:false}),'+
        'PrinterArtwork:async()=>({found:false}),'+
        'PrinterArtworkForInstalledTarget:async()=>({found:false})'+
      '}}};</script>',
    '<script src="'+fileUrl('printer-picker.js')+'"></script>',
    '<script src="'+fileUrl('my-printers.js')+'"></script>',
    '<script>('+pageMain.toString()+')('+json(driver)+');</script>',
    '</body>'
  ].join('\n'));
}

// The dialog document runs in an iframe of exactly the scenario size, so its viewport is the requested
// CSS size. Headless Chrome reserves window chrome (about 143px here), so the host window is
// simply made large enough to hold the largest frame (1100x750) with room to spare.
function hostPage(scenario){
  const json=(value)=>JSON.stringify(value).replace(/</g,'\\u003c');
  return [
    '<!doctype html><html><head><meta charset="utf-8"><title>My Printers layout</title>',
    '<style>html,body{margin:0;padding:0;overflow:hidden;background:#fff}iframe{display:block;border:0}</style>',
    '</head><body><script>(function(){',
    '  const root=document.documentElement;',
    '  const frame=document.createElement("iframe");',
    '  frame.width='+scenario.width+';frame.height='+scenario.height+';',
    '  frame.srcdoc='+json(dialogDocument(scenario))+';',
    '  document.body.append(frame);',
    '  root.setAttribute("data-picker-host-viewport",JSON.stringify({width:innerWidth,height:innerHeight}));',
    '  let polls=0;',
    '  (function poll(){',
    '    const doc=frame.contentDocument&&frame.contentDocument.documentElement;',
    '    const report=doc&&doc.getAttribute("data-picker-layout-ci");',
    '    if(report){root.setAttribute("data-picker-layout-ci",report);return;}',
    '    const error=doc&&doc.getAttribute("data-picker-layout-ci-error");',
    '    if(error){root.setAttribute("data-picker-layout-ci-error",error);return;}',
    '    if(++polls>200){root.setAttribute("data-picker-layout-ci-error",encodeURIComponent("frame produced no report"));return;}',
    '    setTimeout(poll,20);',
    '  })();',
    '})();</script></body></html>'
  ].join('\n');
}

const browserArgs=[
  '--headless=new','--disable-gpu','--disable-extensions',
  '--disable-background-networking','--disable-background-mode',
  '--disable-component-update','--disable-sync',
  '--no-first-run','--no-default-browser-check','--no-sandbox',
  '--window-size=1200,1000',
  '--virtual-time-budget=2500','--dump-dom'
];

// Tries each candidate until one renders a marker. Launch errors and missing markers fall
// through to the next binary; a rendered geometry report is never retried for a different result.
function renderScenario(scenario,htmlPath,profileRoot){
  const url=pathToFileURL(htmlPath).href;
  const failures=[];
  for(const [index,browser] of browsers.entries()){
    const profile=path.join(profileRoot,'profile-'+index+'-'+scenario.name);
    const result=spawnSync(browser,[...browserArgs,'--user-data-dir='+profile,url],{
      encoding:'utf8',timeout:60000,maxBuffer:64*1024*1024
    });
    const stdout=result.stdout||'';
    // Only the host page's <html> start tag is matched; the frame's own markup is not in the dump.
    const match=stdout.match(/<html\b[^>]*\sdata-picker-layout-ci="([^"]+)"/);
    const errorMatch=stdout.match(/<html\b[^>]*\sdata-picker-layout-ci-error="([^"]+)"/);
    const hostMatch=stdout.match(/<html\b[^>]*\sdata-picker-host-viewport="([^"]+)"/);
    if(match){
      try{
        const report=JSON.parse(decodeURIComponent(match[1]));
        report.host=hostMatch?JSON.parse(hostMatch[1].replace(/&quot;/g,'"')):null;
        return {browser,report};
      }
      catch(error){failures.push({browser,problem:'invalid geometry report: '+error.message});continue;}
    }
    failures.push({
      browser:path.basename(browser),status:result.status,signal:result.signal,
      problem:errorMatch?'page error: '+decodeURIComponent(errorMatch[1]).slice(0,600):'no render marker',
      error:result.error?result.error.message:null,
      stderr:(result.stderr||'').slice(-600),
    });
  }
  return {browser:null,failures};
}

function checkScenario(scenario,report){
  const problems=[...report.issues];
  // The frame is sized exactly, so its viewport must match the scenario; only the host window is
  // allowed to be larger than the frame.
  if(report.viewport.width!==scenario.width||report.viewport.height!==scenario.height)
    problems.push('dialog viewport is '+report.viewport.width+'x'+report.viewport.height+', expected '+
      scenario.width+'x'+scenario.height);
  if(!report.host||report.host.width<scenario.width||report.host.height<scenario.height)
    problems.push('host window too small to hold the '+scenario.width+'x'+scenario.height+' frame: '+JSON.stringify(report.host));
  if(report.lang!==scenario.lang||report.dir!==scenario.dir)
    problems.push('document language/direction '+report.lang+'/'+report.dir+' expected '+scenario.lang+'/'+scenario.dir);
  if(report.pills.length!==5)problems.push('selection summary shows '+report.pills.length+' pills, expected 5');
  if(report.pills.some(pill=>/\bmm\b|OrcaSlicer|Flash Studio|QIDIStudio|physical tools/i.test(pill)))
    problems.push('technical profiles leaked into the selected printer summary: '+JSON.stringify(report.pills));
  if(report.variantCount!==0)problems.push('technical variant rows leaked into the catalogue');
  if(!/\b5\b/.test(report.selectionCount))problems.push('selection count does not read 5: "'+report.selectionCount+'"');
  if(report.hitTests<1)problems.push('no visible element was hit-tested: scrollArea '+J(report.scrollArea)+
    ' list '+J(report.list)+' firstCard '+J(report.firstCard||{}));
  const creator=report.cards.find(card=>card.name==='Creator 5 Pro');
  if(scenario.installed){
    if(creator?.variants.length!==0||creator?.values.length!==1)
      problems.push('Creator 5 Pro shows technical sub-options in its model tile');
    if(creator?.hasProfileSettings)
      problems.push('Unselected hardware catalogue card contains a slicer-settings button');
    if(scenario.filter){
      if(!report.profile?.open||report.profile.choices.length!==2)
        problems.push('Selected Creator 5 Pro slicer settings must contain auto and same-nozzle Orca only: '+JSON.stringify(report.profile));
      const exact=report.profile?.choices.find(choice=>choice.id==='orca-flashforge-creator-5-pro-025-1111111111');
      if(!exact?.checked)problems.push('the saved Creator 5 Pro 0.25 mm profile was not preserved: '+JSON.stringify(report.profile));
      if(report.profile?.frame){
        const f=report.profile.frame;
        if(f.x<0||f.y<0||f.right>scenario.width||f.bottom>scenario.height||report.profile.contentClipped)
          problems.push('separate profile dialog clipped or outside viewport: '+JSON.stringify(report.profile));
      }
    }
  }
  // The curated default is 241 cards; catalogue-only scenarios have an
  // extra saved Afinia printer retained even though it is outside the shortlist.
  const expectedCount=scenario.expanded?582:(scenario.installed?241:242);
  if(!scenario.filter&&report.cardCount!==expectedCount)
    problems.push('expected '+expectedCount+' model cards, measured '+report.cardCount);
  if(report.scopeExpanded!==Boolean(scenario.expanded))
    problems.push('scope toggle state differs from scenario');
  if(report.makers.includes('Bambu Lab'))
    problems.push('Bambu destination appeared in maker dropdown');
  if(!scenario.expanded&&!scenario.installed&&!report.makers.includes('Afinia'))
    problems.push('saved hidden maker Afinia was not retained in dropdown');
  if(!scenario.expanded&&scenario.installed&&report.makers.includes('Afinia'))
    problems.push('unselected hidden Afinia maker appeared by default');
  if(!scenario.expanded&&report.makers.includes('Artillery'))
    problems.push('unselected hidden maker was shown in curated dropdown');
  if(scenario.filter&&report.cardCount<1)problems.push('Creator 5 Pro filter shows no card');
  for(const saved of report.saved){
    if(!scenario.filter){
      if(!saved.present||!saved.checked)problems.push('saved ID not selected in the dialog: '+saved.id+' '+JSON.stringify(saved));
    }else if(saved.present&&!saved.checked){
      problems.push('visible saved ID not selected under filter: '+saved.id);
    }
  }
  if(!scenario.filter&&report.saved.filter(saved=>saved.present&&saved.checked).length!==5)
    problems.push('expected 5 preserved saved IDs selected, measured '+
      report.saved.filter(saved=>saved.present&&saved.checked).length);
  if(scenario.filter&&scenario.installed&&!report.saved.some(saved=>saved.id.startsWith('orca-')&&saved.present))
    problems.push('installed Creator 5 Pro variant not shown under filter');
  if(!scenario.installed&&scenario.expanded){
    const keys=report.cards.map(card=>card.key);
    if(new Set(keys).size!==582)problems.push('expanded list must contain 582 distinct physical model identities');
    if(report.cards.some(card=>card.values.length!==1))
      problems.push('expanded list contains model cards with multiple selection boxes');
    const picker=require('../frontend/dist/printer-picker.js');
    const expected=picker.groupDestinations(picker.catalogItems(catalog.models,availability,[],true));
    if(expected.length!==582||expected.flatMap(g=>g.ids).filter(id=>catalogIds.includes(id)).length!==589)
      problems.push('underlying 589 non-Bambu source IDs changed');
  }
  return problems;
}

const workDir=fs.mkdtempSync(path.join(os.tmpdir(),'slicebridge-picker-ci-'));
const profileRoot=fs.mkdtempSync(path.join(os.tmpdir(),'slicebridge-chromium-'));
let failed=0;
const successfulRenderers=new Set();
try{
  for(const scenario of scenarios){
    const htmlPath=path.join(workDir,scenario.name+'.html');
    fs.writeFileSync(htmlPath,hostPage(scenario),'utf8');
    const result=renderScenario(scenario,htmlPath,profileRoot);
    if(!result.report){
      failed++;
      console.log('FAIL '+scenario.name+': no real browser rendered the dialog');
      for(const failure of result.failures)console.log('   - '+JSON.stringify(failure));
      continue;
    }
    successfulRenderers.add(path.basename(result.browser));
    const problems=checkScenario(scenario,result.report);
    const q=result.report;
    // Optional visual evidence: one PNG per layout. Off in CI unless the directory is given.
    if(process.env.PICKER_LAYOUT_CI_SCREENSHOTS){
      const dir=path.resolve(process.env.PICKER_LAYOUT_CI_SCREENSHOTS);
      fs.mkdirSync(dir,{recursive:true});
      spawnSync(result.browser,[...browserArgs.filter(arg=>arg!=='--dump-dom'),
        '--user-data-dir='+path.join(profileRoot,'shot-'+scenario.name),
        '--screenshot='+path.join(dir,scenario.name+'.png'),pathToFileURL(htmlPath).href],
        {encoding:'utf8',timeout:60000});
    }
    if(problems.length){
      failed++;
      console.log('FAIL '+scenario.name+' ['+path.basename(result.browser)+']: '+problems.length+' problem(s)');
      for(const problem of problems)console.log('   - '+problem);
      console.log('   measured: viewport='+q.viewport.width+'x'+q.viewport.height+' frame='+J(q.frame)+
        ' list='+J(q.list)+' firstCard='+J(q.firstCard||{})+
        ' cards='+q.cardCount+' variants='+q.variantCount+' hitTests='+q.hitTests+' pills='+q.pills.length);
    }else{
      console.log('PASS '+scenario.name+' ['+path.basename(result.browser)+']: viewport='+q.viewport.width+'x'+
        q.viewport.height+' cards='+q.cardCount+' variants='+q.variantCount+' hitTests='+q.hitTests+
        ' pills='+q.pills.length+' frame='+J(q.frame)+' list='+J(q.list)+' firstCard='+J(q.firstCard||{}));
    }
  }
}finally{
  fs.rmSync(workDir,{recursive:true,force:true});
  fs.rmSync(profileRoot,{recursive:true,force:true,maxRetries:5,retryDelay:100});
}
function J(r){return JSON.stringify(Object.fromEntries(Object.entries(r).map(([k,v])=>[k,Math.round(v)])));}
if(failed){
  console.log(failed+' of '+scenarios.length+' My Printers layout scenario(s) failed');
  process.exit(1);
}
console.log('all '+scenarios.length+' My Printers layout scenarios passed in '+[...successfulRenderers].join(', '));
