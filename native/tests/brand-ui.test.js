'use strict';
const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const base=path.resolve(__dirname,'..');
const read=(f)=>fs.readFileSync(path.join(base,f),'utf8');

test('approved SliceBridge name and Mint/Forest identity are embedded',()=>{
 const page=read('frontend/dist/index.html');
 const manifest=JSON.parse(read('wails.json'));
 assert.equal(manifest.name,'SliceBridge');
 assert.equal(manifest.outputfilename,'SliceBridge');
 assert.equal(manifest.info.productName,'SliceBridge');
 const css=read('frontend/dist/app.css');
 const icon=read('frontend/dist/brand-icon.svg');
 assert.match(page,/<title>SliceBridge<\/title>/);
 assert.match(page,/class="brand-mark" src="\.\/brand-icon\.svg"/);
 assert.match(page,/<strong>Slice<span class="brand-accent">Bridge<\/span><\/strong>/);
 for(const hex of ['DFF6EE','A9E5D0','5BBFA8','2F7F73','193F38']){
  assert.ok((css+icon).includes('#'+hex),'missing approved shade '+hex);
 }
 assert.match(icon,/id="arch"/);
 assert.doesNotMatch(page,/class="brand-mark"[^>]*>3<\/div>/);
});

test('version and check button are colocated in the bottom action bar',()=>{
 const page=read('frontend/dist/index.html');
 const footer=page.slice(page.indexOf('<footer class="actionbar">'));
 const header=page.slice(0,page.indexOf('<footer class="actionbar">'));
 assert.match(footer,/class="app-meta"/);
 assert.match(footer,/id="installedVersion"/);
 assert.match(footer,/id="checkUpdatesBtn"/);
 assert.doesNotMatch(header,/id="checkUpdatesBtn"/);
 assert.doesNotMatch(header,/id="installedVersion"/);
 assert.equal((page.match(/id="checkUpdatesBtn"/g)||[]).length,1);
 assert.equal((page.match(/id="settingsBtn"/g)||[]).length,1);
});

test('results clear is manual and automatic, while JSON export is removed',()=>{
 const page=read('frontend/dist/index.html');
 const app=read('frontend/dist/app.js');
 assert.match(page,/id="clearResultsBtn"/);
 assert.doesNotMatch(page,/id="exportReportBtn"|id="reportToolbar"/);
 assert.doesNotMatch(app,/ExportLastReport|exportReportBtn|reportToolbar/);
 assert.match(app,/function clearResults\(/);
 assert.match(app,/if\(next.length===files.length\)return;/);
 assert.match(app,/files=next;\s*memoryStatus=null;\s*clearResults\(\);/);
 assert.match(app,/el\('clearResultsBtn'\)\.addEventListener/);
 for(const lang of ['clearResults:\'נקה תוצאות\'','clearResults:\'Clear results\'']){
  assert.ok(app.includes(lang),'missing translation '+lang);
 }
});
