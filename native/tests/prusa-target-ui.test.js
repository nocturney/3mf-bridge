'use strict';
const {test}=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const path=require('node:path');
const vm=require('node:vm');

const dist=path.resolve(__dirname,'../frontend/dist');
const html=fs.readFileSync(path.join(dist,'index.html'),'utf8');
const app=fs.readFileSync(path.join(dist,'app.js'),'utf8');
const prusaGo=fs.readFileSync(path.resolve(__dirname,'../prusa_engine.go'),'utf8');

function translationsFromSource(){
  const start=app.indexOf('const translations = ');
  const end=app.indexOf('\nconst printerNames');
  assert.ok(start>=0&&end>start,'translations literal must be present');
  const literal=app.slice(start+'const translations = '.length,end).trim().replace(/;$/,'');
  return vm.runInNewContext('('+literal+')');
}

test('PrusaSlicer row sits in #targetList, starts hidden and carries the fixed value',()=>{
  const listStart=html.indexOf('id="targetList"');
  const emptyNotice=html.indexOf('id="emptyTargetNotice"');
  const rowStart=html.indexOf('data-prusa-target');
  assert.ok(listStart>0&&rowStart>listStart&&rowStart<emptyNotice,'row must live inside the target list');
  assert.equal(html.match(/data-prusa-target/g).length,1);
  const row=html.slice(html.lastIndexOf('<label',rowStart),html.indexOf('</label>',rowStart)+'</label>'.length);
  assert.match(row,/\shidden[\s>]/,'row must be hidden until the sidecar probe succeeds');
  assert.match(row,/<input type="checkbox" value="prusa-project">/);
});

test('PrusaSlicer row carries no readiness, approval or catalogue-only wording',()=>{
  const rowStart=html.indexOf('data-prusa-target');
  const row=html.slice(rowStart,html.indexOf('</label>',rowStart));
  assert.doesNotMatch(row,/ready|approv|catalog|readiness|verified|compatib/i);
  assert.doesNotMatch(html,/data-target-row="prusa/,'the generic target must not reuse built-in row selectors');
});

test('Go backend, JS and HTML agree on the fixed target identifier',()=>{
  assert.match(prusaGo,/prusaTargetID\s*=\s*"prusa-project"/);
  assert.match(app,/const prusaTargetId='prusa-project';/);
  assert.match(html,/value="prusa-project"/);
});

test('new PrusaSlicer strings exist in Hebrew and English and never overclaim',()=>{
  const dictionary=translationsFromSource();
  const keys=['prusaProjectName','prusaProjectSummary','prusaGenericReview'];
  for(const lang of ['he','en']){
    for(const key of keys){
      assert.ok(dictionary[lang][key]&&dictionary[lang][key].length>0,`${lang}.${key} missing`);
    }
  }
  assert.equal(dictionary.en.prusaProjectName,'PrusaSlicer project');
  for(const key of keys){
    assert.doesNotMatch(dictionary.en[key],/ready|approved|verified|guarantee|validated/i,`en.${key} overclaims`);
  }
  assert.match(dictionary.he.prusaProjectName,/PrusaSlicer/);
});

test('translation dictionaries keep matching key sets',()=>{
  const dictionary=translationsFromSource();
  assert.deepEqual(Object.keys(dictionary.he).sort(),Object.keys(dictionary.en).sort());
});

test('the PrusaSlicer checkbox is probed through the backend and hidden on failure',()=>{
  assert.match(app,/async function refreshPrusaEngine\(\)/);
  assert.match(app,/api\(\)\.PrusaEngineStatus\(\)/);
  assert.match(app,/row\.hidden=!available;/);
  assert.match(app,/catch\{available=false;\}/);
  assert.match(app,/await refreshPrusaEngine\(\);/);
});

test('select-all never silently includes the PrusaSlicer row',()=>{
  const selectAll=app.slice(app.indexOf('function selectAll()'),app.indexOf('function registerHandlers'));
  assert.match(selectAll,/\[data-prusa-target\]/);
});

test('select-all label counts the PrusaSlicer checkbox as a separate row',()=>{
  const render=app.slice(app.indexOf('function renderTargets()'),app.indexOf('async function refreshAvailability'));
  assert.match(render,/const prusaChecked=prusaCheckbox\(\)\?\.checked\?1:0;/);
  assert.match(render,/installedRows\.filter\(\(entry\)=>entry\.checkbox\.checked\)\.length\+prusaChecked/);
  assert.match(render,/if\(prusaCheckbox\(\)\)prusaCheckbox\(\)\.disabled=busy;/);
});

test('results name the target and surface the generic-printer review note',()=>{
  assert.match(app,/if\(id===prusaTargetId\)return t\('prusaProjectName'\);/);
  assert.match(app,/if\(report\.printer_profile_generic\)\{/);
  assert.match(app,/t\('prusaGenericReview'\)/);
});

test('the target list still renders the four built-in rows unchanged in order',()=>{
  assert.deepEqual([...app.matchAll(/const targetOrder = \[([^\]]*)\];/g)].map((m)=>m[1]),["'u1','c5','cc2','qidi'"]);
  for(const target of ['u1','c5','cc2','qidi']){
    assert.match(html,new RegExp(`data-target-row="${target}"`));
  }
});
