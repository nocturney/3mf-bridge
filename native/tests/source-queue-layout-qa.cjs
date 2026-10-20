'use strict';
// Manual rendered-layout QA in a real Chromium engine. No Wails RPC, printer or
// slicer actions are invoked; the file selection is a local synthetic fixture.
const fs=require('node:fs');
const os=require('node:os');
const path=require('node:path');
const {spawnSync}=require('node:child_process');
const assert=require('node:assert/strict');
const {pathToFileURL}=require('node:url');
const src=path.resolve(__dirname,'../frontend/dist');
const chrome=process.env.CHROME_BIN || (process.platform==='darwin'
  ?'/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge'
  :process.platform==='win32'
    ?'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe'
    :'/usr/bin/google-chrome');
if(!fs.existsSync(chrome))throw Error('Chromium browser not available at '+chrome);

const original=fs.readFileSync(path.join(src,'index.html'),'utf8');
const fixtureScript=[
  "(function(){",
  "  const query=new URL(location.href).searchParams;",
  "  const count=Number(query.get('files')||'0');",
  "  const lang=query.get('lang')||'he';",
  "  document.documentElement.lang=lang;",
  "  document.documentElement.dir=lang==='he'?'rtl':'ltr';",
  "  document.body.dir=document.documentElement.dir;",
  "  const panel=document.querySelector('.queue-panel');",
  "  const list=document.getElementById('fileList');",
  "  const heading=document.getElementById('queuedFilesLabel');",
  "  panel.classList.toggle('has-files',count>0);",
  "  heading.hidden=count===0;",
  "  list.hidden=count===0;",
  "  document.getElementById('queueCount').textContent=String(count);",
  "  document.getElementById('queueFilesShownCount').textContent=String(count);",
  "  for(let i=0;i<count;i++){",
  "    const item=document.createElement('div');",
  "    item.className='queue-item';",
  "    item.setAttribute('role','listitem');",
  "    const badge=document.createElement('span');",
  "    badge.className='file-badge';badge.textContent='3MF';",
  "    const info=document.createElement('div');",
  "    info.className='queue-file-details';",
  "    const name=document.createElement('strong');",
  "    name.textContent='11-Box+Kettle+large+complex-project-variant-'+(i+1)+'.3mf';",
  "    const small=document.createElement('small');",
  "    small.textContent='C:/models/downloaded · 5.8 MB';",
  "    info.append(name,small);",
  "    const remove=document.createElement('button');",
  "    remove.className='remove-file-btn';remove.textContent='×';",
  "    item.append(badge,info,remove);list.append(item);",
  "  }",
  "  function rect(q){const r=document.querySelector(q).getBoundingClientRect();",
  "    return {x:r.x,y:r.y,width:r.width,height:r.height,bottom:r.bottom};}",
  "  function report(){",
  "    const add=document.getElementById('addFilesBtn');",
  "    add.scrollIntoView({block:'center'});",
  "    const a=add.getBoundingClientRect();",
  "    const hit=document.elementFromPoint(a.x+a.width/2,a.y+a.height/2);",
  "    const out={",
  "      viewport:{width:innerWidth,height:innerHeight},",
  "      language:document.documentElement.lang,",
  "      files:count,",
  "      panel:rect('.queue-panel'),",
  "      targets:rect('.targets-panel'),",
  "      banner:rect('.brand-banner'),",
  "      bannerText:rect('.brand-banner-content'),",
  "      list:rect('.queue-list'),",
  "      heading:rect('.queue-files-heading'),",
  "      first:count?rect('.queue-item'):null,",
  "      footnote:rect('#queueFootnote'),",
  "      workspace:rect('.workspace'),",
  "      results:rect('.results-panel'),",
  "      panelDisplay:getComputedStyle(panel).display,",
  "      listOverflow:getComputedStyle(list).overflowY,",
  "      scrollableFiles:list.scrollHeight-list.clientHeight,",
  "      listClientHeight:list.clientHeight,",
  "      scrollableWorkspace:document.querySelector('.workspace').scrollHeight-document.querySelector('.workspace').clientHeight,",
  "      addButtonClickable:add.contains(hit),",
  "      docOverflow:document.documentElement.scrollHeight-innerHeight",
  "    };",
  "    document.documentElement.setAttribute('data-layout-qa',encodeURIComponent(JSON.stringify(out)));",
  "  }",
  "  setTimeout(report,180);",
  "})();"
].join('\n');
const fixture=original
 .replace(/<script defer src="\.\/[^"]+"[^>]*><\/script>/g,'')
 .replace('</body>','<script>'+fixtureScript+'</script></body>');
const markerDir=fs.mkdtempSync(path.join(src,'.source-qa-'));
const browserProfile=fs.mkdtempSync(path.join(os.tmpdir(),'slicebridge-chrome-'));
const htmlPath=path.join(markerDir,'fixture.html');
fs.writeFileSync(htmlPath,fixture,'utf8');
// Resolve the CSS and icon from the actual frontend dist; do not copy them.
const testHTML=fs.readFileSync(htmlPath,'utf8')
 .replace('href="./app.css"','href="../app.css"')
 .replace('src="./brand-icon.png"','src="../brand-icon.png"');
fs.writeFileSync(htmlPath,testHTML,'utf8');
const trials=process.env.QUEUE_LAYOUT_SINGLE?[[1280,900,Number(process.env.QUEUE_LAYOUT_SINGLE),'he']]:[
  [1100,750,1,'he'],
  [1100,750,2,'he'],
  [1100,750,15,'he'],
  [1280,900,1,'he'],
  [1280,900,2,'en'],
  [1280,900,12,'en'],
  [780,580,1,'he'],
  [780,580,8,'en'],
  [746,620,2,'he'],
  [820,620,2,'he'],
  [960,620,2,'en'],
  [900,620,2,'he'],
  [900,620,2,'en'],
  [720,500,2,'he'],
  [720,500,2,'en'],
  [1100,750,0,'he']
];
try{
  for(const [w,h,files,lang] of trials){
    const url=pathToFileURL(htmlPath).href+'?files='+files+'&lang='+lang;
    const args=[
      '--headless=new','--disable-gpu','--disable-extensions',
      '--no-first-run','--no-default-browser-check','--no-sandbox',
      '--user-data-dir='+browserProfile,
      '--window-size='+w+','+h,
      '--virtual-time-budget=2500','--dump-dom',url
    ];
    const p=spawnSync(chrome,args,{encoding:'utf8',timeout:25000,maxBuffer:8*1024*1024});
    if(p.status!==0)throw Error('Chrome failed: '+p.status+' '+p.stderr.slice(-1500));
    const m=p.stdout.match(/data-layout-qa="([^"]+)"/);
    if(!m)throw Error('No computed-geometry report: '+p.stdout.slice(-1000)+' '+p.stderr.slice(-1000));
    const q=JSON.parse(decodeURIComponent(m[1]));
    assert.equal(q.language,lang);
    assert.ok(q.bannerText.y>=q.banner.y-2 && q.bannerText.bottom<=q.banner.bottom+2,
      'fixed banner text is clipped in this viewport: '+JSON.stringify(q));
    // Headless Edge macOS reserves ~130 px for its virtual browser chrome.
    assert.ok(q.viewport.width>=w-20,'renderer ignored requested test width');
    // Headless Chrome reserves 151px for its virtual frame on Windows.
    assert.ok(q.viewport.height>=h-160,'renderer ignored requested test height');
    assert.ok(q.panel.width>320,'source panel width too narrow');
    assert.ok(q.addButtonClickable,
      'Add files button is intercepted by another panel at '+w+'x'+h+': '+JSON.stringify(q));
    if(files){
      assert.ok(q.panel.height>=178 && q.panel.height<=355,
        'source panel not content-sized: '+JSON.stringify(q));
      if(files<=2)assert.ok(q.panel.height<300,'few files should not consume most workspace');
      assert.ok(q.listClientHeight>=50,'file list clipped: '+JSON.stringify(q));
      assert.ok(q.first.y>=q.list.y-2,'first row above scroll frame');
      assert.ok(q.first.bottom<=q.list.bottom+2,'first row below scroll frame');
      assert.ok(q.footnote.bottom<=q.panel.bottom+2,'queue footnote outside panel');
      assert.equal(q.listOverflow,'auto');
      if(files>=8)assert.ok(q.scrollableFiles>100,'long file list fails to scroll');
      if(files<=2 && q.viewport.width>=1000 && q.viewport.height>=720){
        assert.ok(q.targets.bottom<=q.workspace.bottom+2,
          'target-printer panel hidden below workspace with few source files: '+JSON.stringify(q));
        assert.ok(q.scrollableWorkspace<=2,
          'unexpected workspace scrolling for few files in normal desktop window: '+JSON.stringify(q));
      }
    }else{
      assert.ok(q.panel.height<300,'empty queue should stay compact');
    }
    assert.ok(q.docOverflow<=1,'document itself scrolled (should scroll workspace)');
    if(h<=580&&files)assert.ok(q.scrollableWorkspace>100,'compact view missing workspace scroll');
    console.log('QUEUE_RENDER_PASS',w+'x'+h,'actual='+q.viewport.width+'x'+q.viewport.height,lang,'files='+files,
      'panel='+Math.round(q.panel.height),'targetsBottom='+Math.round(q.targets.bottom),'workspaceBottom='+Math.round(q.workspace.bottom),
      'list='+q.listClientHeight,'listScroll='+q.scrollableFiles,
      'workspaceScroll='+q.scrollableWorkspace);
  }
}finally{
 fs.rmSync(markerDir,{recursive:true,force:true});
 fs.rmSync(browserProfile,{recursive:true,force:true});
}
