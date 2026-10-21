'use strict';
const {app,BrowserWindow}=require('electron');
const fs=require('node:fs/promises');
const path=require('node:path');
const assert=require('node:assert/strict');
const sleep=(ms)=>new Promise(resolve=>setTimeout(resolve,ms));

async function inspect(win){
  return win.webContents.executeJavaScript(`(() => {
    const e=(q)=>document.querySelector(q);
    const rect=(q)=>{const r=e(q).getBoundingClientRect();return {
      x:r.x,y:r.y,width:r.width,height:r.height,bottom:r.bottom
    };};
    return {
      language:document.documentElement.lang,dir:document.documentElement.dir,
      viewport:{w:innerWidth,h:innerHeight},
      documentScroll:document.documentElement.scrollHeight-innerHeight,
      bodyScroll:document.body.scrollHeight-innerHeight,
      header:rect('.topbar'),footer:rect('.actionbar'),
      workspace:rect('.workspace'),queue:rect('.queue-list'),
      results:rect('.results-scroll'),
      queueScroll:e('.queue-list').scrollHeight-e('.queue-list').clientHeight,
      resultsScroll:e('.results-scroll').scrollHeight-e('.results-scroll').clientHeight,
      queueCount:e('#queueCount').textContent,
      modal:e('#settingsDialog').open,
      modalHeight:e('#settingsDialog').getBoundingClientRect().height,
      conversionDisabled:e('#convertBtn').disabled,
      reportCards:document.querySelectorAll('.file-report').length
    };
  })()`);
}
async function main(){
  await app.whenReady();
  const win=new BrowserWindow({
    width:1100,height:750,show:false,backgroundColor:'#f0f2f4',
    webPreferences:{preload:path.join(__dirname,'layout-preload.cjs'),contextIsolation:true,nodeIntegration:false}
  });
  const failures=[];
  win.webContents.on('console-message',(_event,details)=>{
    if(details.level==='error')failures.push(details.message);
  });
  await win.loadFile(path.join(__dirname,'..','frontend','dist','index.html'));
  await sleep(250);
  let layout=await inspect(win);
  assert.equal(layout.documentScroll,0,'document scroll at desktop size');
  assert.equal(layout.bodyScroll,0,'body scroll at desktop size');
  assert.ok(layout.footer.bottom<=layout.viewport.h+1 &&layout.footer.bottom>=layout.viewport.h-1,
    'fixed actionbar must align to bottom of the window');
  assert.ok(layout.workspace.height>250,'workspace must have useful height');
  assert.ok(layout.queue.height>40,'file list needs independent scroll container');

  await win.webContents.executeJavaScript("document.getElementById('addFilesBtn').click()");
  await sleep(200);
  layout=await inspect(win);
  assert.equal(layout.queueCount,'32','multi-select should append all source files');
  assert.ok(layout.queueScroll>100,'file list should scroll internally');
  assert.equal(layout.documentScroll,0,'adding files must not scroll document');

  const before=win.getContentSize();
  await win.webContents.executeJavaScript("document.getElementById('enBtn').click()");
  await sleep(150);
  const after=win.getContentSize();
  assert.deepEqual(after,before,'changing language must not resize the OS window');
  layout=await inspect(win);
  assert.equal(layout.dir,'ltr','English must change reading direction');

  await win.webContents.executeJavaScript("document.getElementById('settingsBtn').click()");
  await sleep(150);
  layout=await inspect(win);
  assert.equal(layout.modal,true,'printer settings modal must open');
  assert.ok(layout.modalHeight<layout.viewport.h-20,'printer settings must fit screen');
  await win.webContents.executeJavaScript("document.getElementById('closeSettingsBtn').click()");
  layout=await inspect(win);
  assert.equal(layout.modal,false,'printer settings modal must close');
  await win.webContents.executeJavaScript("document.getElementById('convertBtn').click()");
  await sleep(600);
  layout=await inspect(win);
  assert.equal(layout.reportCards,32,'render one report per source file');
  assert.equal(layout.conversionDisabled,false,'completed conversion must preserve checked target printers');
  assert.ok(layout.resultsScroll>200,'results should scroll internally');
  assert.equal(layout.documentScroll,0,'rendering many reports must not scroll window');
  await fs.mkdir(path.join(__dirname,'..','..','dist'),{recursive:true});
  const image=await win.webContents.capturePage();
  await fs.writeFile(path.join(__dirname,'..','..','dist','native-workflow-desktop.png'),image.toPNG());

  win.setContentSize(780,580);
  await sleep(180);
  layout=await inspect(win);
  assert.equal(layout.documentScroll,0,'body stays fixed in compact window');
  assert.ok(layout.footer.bottom<=layout.viewport.h+1&&layout.footer.bottom>=layout.viewport.h-1,
    'footer must remain attached to bottom in compact window');
  assert.ok(layout.workspace.height>100,'compact workspace remains usable');
  const image2=await win.webContents.capturePage();
  await fs.writeFile(path.join(__dirname,'..','..','dist','native-workflow-compact.png'),image2.toPNG());
  assert.deepEqual(failures,[],'no JS errors from frontend');
  console.log('NATIVE_UX_LAYOUT_PASS',JSON.stringify(layout));
  win.destroy();app.quit();
}
main().catch(e=>{console.error('NATIVE_UX_LAYOUT_FAIL',e);app.exit(1)});
