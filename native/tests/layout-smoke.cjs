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
      banner:rect('.brand-banner'),progressBar:rect('.conversion-progress'),
      workspace:rect('.workspace'),queue:rect('.queue-list'),
      results:rect('.results-scroll'),
      queueScroll:e('.queue-list').scrollHeight-e('.queue-list').clientHeight,
      resultsScroll:e('.results-scroll').scrollHeight-e('.results-scroll').clientHeight,
      queueCount:e('#queueCount').textContent,
      modal:e('#settingsDialog').open,
      modalHeight:e('#settingsDialog').getBoundingClientRect().height,
      conversionDisabled:e('#convertBtn').disabled,
      integrationsModal:e('#integrationsDialog').open,
      integrationsStatus:e('#browserLinkStatus').textContent,
      reportCards:document.querySelectorAll('.file-report').length,
      clearResultsDisabled:e('#clearResultsBtn').disabled,
      brandImageLoaded:e('.brand-mark').naturalWidth>0,
      metaInFooter:!!e('footer .app-meta #installedVersion')&&!!e('footer .app-meta #checkUpdatesBtn'),
      hasJsonExport:!!e('#exportReportBtn'),
      integrationLabel:e('#integrationsBtn').textContent.trim(),
      integrationButton:rect('#integrationsBtn'),
      progressPercent:e('#progressPercent').textContent,
      progressFill:rect('#progressFill'),progressTrack:rect('#progressTrack'),
      makerworldBadges:document.querySelectorAll('.makerworld-badge').length
    };
  })()`);
}
async function main(){
  await app.whenReady();
  const win=new BrowserWindow({
    width:1100,height:750,show:false,backgroundColor:'#F5FAF7',
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
  assert.ok(Math.abs(layout.footer.bottom-layout.banner.y)<2,'actionbar must sit above fixed banner');
  assert.ok(Math.abs(layout.banner.bottom-layout.progressBar.y)<2,'progress must sit directly below banner');
  assert.ok(Math.abs(layout.progressBar.bottom-layout.viewport.h)<2,'progress must align with bottom of window');
  assert.equal(layout.brandImageLoaded,true,'new SliceBridge icon must render');
  assert.equal(layout.metaInFooter,true,'version and update button share bottom bar');
  assert.equal(layout.hasJsonExport,false,'old JSON export must be removed');
  assert.equal(layout.integrationLabel,'MakerWorld Setup','integration button should have a readable title');
  assert.ok(layout.integrationButton.x>=0 && layout.integrationButton.x+layout.integrationButton.width<=layout.viewport.w+1,
    'English integration button must fit the top bar');
  await win.webContents.executeJavaScript("document.getElementById('heBtn').click()");
  await sleep(80);
  layout=await inspect(win);
  assert.equal(layout.dir,'rtl','Hebrew language must use RTL');
  assert.equal(layout.integrationLabel,'חיבור ל־MakerWorld','Hebrew integration label must be human readable');
  assert.ok(layout.integrationButton.x>=0 && layout.integrationButton.x+layout.integrationButton.width<=layout.viewport.w+1,
    'Hebrew integration button must fit the top bar');
  await win.webContents.executeJavaScript("document.getElementById('enBtn').click()");
  await sleep(80);
  layout=await inspect(win);
  assert.ok(layout.workspace.height>250,'workspace must have useful height');
  assert.ok(layout.queue.height>40,'file list needs independent scroll container');
  await win.webContents.executeJavaScript("document.getElementById('integrationsBtn').click()");
  await sleep(150);
  layout=await inspect(win);
  assert.equal(layout.integrationsModal,true,'MakerWorld onboarding modal must open');
  assert.match(layout.integrationsStatus,/approval required/i,'consent requirement is visible');
  await win.webContents.executeJavaScript("document.getElementById('closeIntegrationsBtn').click()");
  layout=await inspect(win);
  assert.equal(layout.integrationsModal,false,'MakerWorld modal must close');

  await win.webContents.executeJavaScript("document.getElementById('addFilesBtn').click()");
  await sleep(200);
  layout=await inspect(win);
  assert.equal(layout.queueCount,'32','multi-select should append all source files');
  assert.equal(layout.makerworldBadges,1,'verified MakerWorld origin is marked in the queue');
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

  // Manual clearing must preserve the source queue and disable stale open actions.
  await win.webContents.executeJavaScript("document.getElementById('clearResultsBtn').click()");
  layout=await inspect(win);
  assert.equal(layout.reportCards,0,'manual clearing empties results');
  assert.equal(layout.queueCount,'32','manual clearing preserves the file queue');
  assert.equal(layout.clearResultsDisabled,true,'clear button is disabled when empty');
  await win.webContents.executeJavaScript("document.getElementById('convertBtn').click()");
  await sleep(500);
  layout=await inspect(win);
  assert.equal(layout.reportCards,32,'conversion can run again after clearing');
  // Adding a truly new file resets old reports automatically; not the queue.
  await win.webContents.executeJavaScript("document.getElementById('addFilesBtn').click()");
  await sleep(250);
  layout=await inspect(win);
  assert.equal(layout.queueCount,'33','new source joins the queue');
  assert.equal(layout.reportCards,0,'new source must clear stale previous results');
  assert.equal(layout.clearResultsDisabled,true,'no stale clear-results button');

  win.setContentSize(780,580);
  await sleep(180);
  layout=await inspect(win);
  assert.equal(layout.documentScroll,0,'body stays fixed in compact window');
  assert.ok(Math.abs(layout.footer.bottom-layout.banner.y)<2,'compact actionbar must be above banner');
  assert.ok(Math.abs(layout.banner.bottom-layout.progressBar.y)<2,'compact banner must be above progress');
  assert.ok(Math.abs(layout.progressBar.bottom-layout.viewport.h)<2,'compact progress must remain at window bottom');
  assert.ok(layout.workspace.height>100,'compact workspace remains usable');
  const image2=await win.webContents.capturePage();
  await fs.writeFile(path.join(__dirname,'..','..','dist','native-workflow-compact.png'),image2.toPNG());
  assert.deepEqual(failures,[],'no JS errors from frontend');
  console.log('NATIVE_UX_LAYOUT_PASS',JSON.stringify(layout));
  win.destroy();app.quit();
}
main().catch(e=>{console.error('NATIVE_UX_LAYOUT_FAIL',e);app.exit(1)});
