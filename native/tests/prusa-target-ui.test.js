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

test('the PrusaSlicer row is probed through the backend and shown only when probed and selected',()=>{
  assert.match(app,/async function refreshPrusaEngine\(\)/);
  assert.match(app,/prusaAvailable=Boolean\(\(await api\(\)\?\.PrusaEngineStatus\?\.\(\)\)\?\.available\);/);
  assert.match(app,/catch\{prusaAvailable=false;\}/);
  assert.match(app,/await refreshPrusaEngine\(\);/);
  assert.match(app,/const visible=prusaAvailable&&isSelectedDestination\(prusaTargetId\);/);
  assert.match(app,/setRowVisible\(prusaRow,box,visible\);/);
});

test('select-all never silently includes the PrusaSlicer row',()=>{
  const selectAll=app.slice(app.indexOf('function selectAll()'),app.indexOf('function registerHandlers'));
  // Built-in rows carry data-target-row; the PrusaSlicer row does not.
  assert.match(selectAll,/#targetList \[data-target-row\] input\[type=checkbox\]/);
  assert.doesNotMatch(selectAll.replace(/\/\/.*$/gm,''),/prusa/i);
});

test('select-all label counts only the visible built-in rows, never the PrusaSlicer row',()=>{
  const render=app.slice(app.indexOf('function renderTargets()'),app.indexOf('// Publishes the destination inputs'));
  assert.match(render,/const shown=builtIns\.filter\(\(entry\)=>builtInVisible\(entry\.target\)\);/);
  assert.match(render,/const checkedShown=shown\.filter\(\(entry\)=>entry\.checkbox\.checked\)\.length;/);
  assert.match(render,/el\('selectAllBtn'\)\.textContent=checkedShown===shown\.length\?t\('deselectAll'\):t\('selectAll'\);/);
  assert.doesNotMatch(render,/prusaChecked/);
  assert.match(render,/box\.disabled=busy\|\|!visible;/);
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
