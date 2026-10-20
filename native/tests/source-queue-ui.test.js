'use strict';

const {test}=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const path=require('node:path');
const root=path.resolve(__dirname,'..');
const read=p=>fs.readFileSync(path.join(root,p),'utf8');

test('source-file list receives a proper independent panel after selecting files',()=>{
  const html=read('frontend/dist/index.html');
  const js=read('frontend/dist/app.js');
  const css=read('frontend/dist/app.css');
  for(const id of ['fileList','queueCount','queuedFilesLabel','queueFilesShownCount',
                   'dropzone','queueFootnote']){
    assert.equal((html.match(new RegExp('id="'+id+'"','g'))||[]).length,1);
  }
  assert.match(html,/id="queuedFilesLabel" hidden/);
  assert.match(html,/id="fileList" role="list"/);
  assert.match(js,/dom\.fileList\.parentElement\.classList\.toggle\('has-files',hasFiles\)/);
  assert.match(js,/el\('queuedFilesLabel'\)\.hidden=!hasFiles/);
  assert.match(js,/el\('queueFilesShownCount'\)\.textContent=hasFiles\?String\(files\.length\):''/);
  assert.match(js,/dom\.fileList\.setAttribute\('aria-label',t\('queuedFiles'\)\)/);
  assert.match(js,/queuedFiles:'קבצים שנבחרו'/);
  assert.match(js,/queuedFiles:'Selected source files'/);

  const current=css.slice(css.indexOf('/* dev.5 —'));
  assert.ok(current.startsWith('/* dev.5 —'),'responsive source-queue rules missing');
  assert.match(current,/\.left-column\{\s*display:flex;\s*flex-direction:column;/);
  assert.match(current,/\.queue-panel\.has-files\{\s*height:clamp\(390px,49dvh,490px\)/);
  assert.match(current,/\.queue-panel\.has-files \.queue-list\{min-height:145px\}/);
  assert.match(current,/\.queue-panel \.queue-list\{[\s\S]*?overflow-y:auto;/);
  assert.match(current,/\.queue-files-heading\[hidden\]\{display:none!important\}/);
  assert.match(current,/@media\(max-width:960px\)/);
  assert.match(current,/@media\(max-height:700px\)/);
});

test('empty queue remains compact and does not leave a visible scrollbar',()=>{
  const css=read('frontend/dist/app.css');
  const js=read('frontend/dist/app.js');
  const now=css.slice(css.indexOf('/* dev.5 —'));
  assert.match(now,/\.queue-panel\{\s*flex:none;\s*height:224px;/);
  assert.match(now,/\.queue-panel \.queue-list\[hidden\]\{display:none!important\}/);
  assert.match(js,/dom\.fileList\.hidden=!hasFiles/);
});

test('source file names display useful details and retain removal affordance',()=>{
  const css=read('frontend/dist/app.css');
  const app=read('frontend/dist/app.js');
  const now=css.slice(css.indexOf('/* dev.5 —'));
  assert.match(now,/\.queue-file-details strong\{[\s\S]*?-webkit-line-clamp:2/);
  assert.match(app,/const name=node\('strong','',file\.name\);name\.title=file\.path/);
  assert.match(app,/const location=node\('small','',file\.folder\+' · '\+file\.sizeLabel\)/);
  assert.match(app,/remove\.title=language==='he'\?'הסר מהתור'/);
});
