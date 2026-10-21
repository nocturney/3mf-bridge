'use strict';
const {test}=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const path=require('node:path');
const vm=require('node:vm');

const native=path.resolve(__dirname,'..');
const html=fs.readFileSync(path.join(native,'frontend/dist/index.html'),'utf8');
const css=fs.readFileSync(path.join(native,'frontend/dist/app.css'),'utf8');
const about=fs.readFileSync(path.join(native,'frontend/dist/about.js'),'utf8');
const integrations=fs.readFileSync(path.join(native,'frontend/dist/integrations.js'),'utf8');
const wait=ms=>new Promise(resolve=>setTimeout(resolve,ms));
const settle=()=>new Promise(resolve=>setImmediate(resolve));

const LOCKED_HE='חיבור ל־MakerWorld נעול. פתח אודות ← מצב מפתחים כדי לפתוח אותו.';
const LOCKED_EN='MakerWorld setup is locked. Open About → Developer mode to unlock it.';

// One shared fake document runs about.js and integrations.js exactly as index.html loads them.
// options.bridge=false hides window.go; get/set select the native failure mode under test.
function harness({bridge=true,developerMode=false,get='ok',set='ok'}={}) {
  const elements=new Map();
  const docListeners={};
  const nativeState={developerMode,calls:[]};
  function node(id) {
    if(elements.has(id))return elements.get(id);
    const classes=new Set();
    const listeners={};
    const el={
      id, hidden:false, disabled:false, open:false, textContent:'', title:'', tabIndex:0,
      value:id==='browserTargetSelect'?'edge':'', dir:'', dataset:{}, attributes:{},
      children:[], listeners,
      classList:{
        add:(...s)=>s.forEach(x=>classes.add(x)),
        remove:(...s)=>s.forEach(x=>classes.delete(x)),
        toggle:(s,value)=>{if(value)classes.add(s);else classes.delete(s);},
        contains:s=>classes.has(s)
      },
      parentElement:{classList:{add(){},remove(){},toggle(){}}},
      setAttribute(name,value){this.attributes[name]=String(value);},
      getAttribute(name){return this.attributes[name];},
      addEventListener(type,fn){(listeners[type]||=[]).push(fn);},
      querySelectorAll(selector){
        if(selector==='input[name="makerworldMode"]')return [node('mode-off'),node('mode-ask'),node('mode-always')];
        if(selector==='.integration-instructions li')return [node('instruction1'),node('instruction2'),node('instruction3')];
        if(selector==='[data-about-i18n]')return aboutI18n;
        return [];
      },
      replaceChildren(...c){this.children=c;},
      append(c){this.children.push(c);},
      showModal(){this.open=true;},
      close(){this.open=false;}
    };
    elements.set(id,el);
    return el;
  }
  // index.html ships the MakerWorld button disabled; the native state can only unlock it.
  node('integrationsBtn').disabled=true;
  // The i18n nodes mirror every data-about-i18n attribute that index.html really declares.
  const aboutI18n=[...html.matchAll(/data-about-i18n="(\w+)"/g)].map(m=>({textContent:'',dataset:{aboutI18n:m[1]}}));
  const api={
    GetDeveloperMode:()=>{
      nativeState.calls.push(['get']);
      if(get==='pending')return new Promise(()=>{});
      if(get==='reject')return Promise.reject(new Error('bridge down'));
      return Promise.resolve(nativeState.developerMode);
    },
    SetDeveloperMode:enabled=>{
      nativeState.calls.push(['set',enabled]);
      if(set==='reject')return Promise.reject(new Error('denied'));
      if(set==='nonboolean')return Promise.resolve(undefined);
      nativeState.developerMode=enabled;
      return Promise.resolve(enabled);
    },
    GetAboutLegalText:async()=>'',
    OpenAboutPage:async()=>{},
    BrowserLinkStatus:async()=>({platform:'windows',registered:true}),
    BrowserExtensionStoreURL:async()=>'',
    OpenMakerWorldMode:async()=>{},
    OpenBrowserExtensionsPage:async()=>{},
    PrepareBrowserExtension:async()=>({folder:'/tmp/SliceBridge/BrowserExtensions/edge'})
  };
  const win=bridge?{go:{main:{App:api}}}:{};
  const document={
    documentElement:{lang:'en',dir:'ltr'},
    getElementById:node,
    createElement:tag=>({tagName:tag,textContent:'',dir:''}),
    createTextNode:text=>({textContent:text}),
    addEventListener:(type,fn)=>{(docListeners[type]||=[]).push(fn);},
    dispatchEvent:event=>{(docListeners[event.type]||[]).forEach(fn=>fn(event));}
  };
  class CustomEvent{constructor(type,init){this.type=type;this.detail=init?.detail;}}
  const context={document,window:win,CustomEvent,URL,setTimeout:(fn,ms)=>setTimeout(fn,Math.min(ms,5))};
  vm.runInNewContext(integrations,context,{filename:'integrations.js'});
  vm.runInNewContext(about,context,{filename:'about.js'});
  const fire=(id,type='click')=>Promise.all((node(id).listeners[type]||[]).map(fn=>fn({target:node(id)})));
  const setLang=lang=>{
    document.documentElement.lang=lang;
    document.documentElement.dir=lang==='he'?'rtl':'ltr';
    return fire(lang+'Btn');
  };
  const i18n=key=>aboutI18n.find(n=>n.dataset.aboutI18n===key).textContent;
  return {node,fire,setLang,i18n,nativeState,win,api};
}

test('MakerWorld is locked before native state arrives and a click cannot open the wizard', async()=>{
  const h=harness({get:'pending',developerMode:true});
  await settle();
  assert.equal(h.node('integrationsBtn').disabled,true);
  assert.equal(h.node('integrationsGate').tabIndex,0);
  assert.equal(h.node('integrationsGate').title,LOCKED_EN);
  await h.fire('integrationsBtn');
  assert.equal(h.node('integrationsDialog').open,false);
});

test('a locked MakerWorld keeps its tooltip on the focusable wrapper, not the disabled button',async()=>{
  const h=harness();
  await settle();
  assert.equal(h.node('integrationsBtn').disabled,true);
  assert.equal(h.node('integrationsGate').title,LOCKED_EN);
  assert.equal(h.node('integrationsGate').tabIndex,0);
  assert.equal(h.node('integrationsDialog').open,false);
  // Clicking the wrapper is a no-op too: only the button opens the wizard.
  await h.fire('integrationsGate');
  assert.equal(h.node('integrationsDialog').open,false);
});

test('enabling Developer Mode in About unlocks MakerWorld and the wizard opens from it',async()=>{
  const h=harness();
  await settle();
  await h.fire('developerModeBtn');
  assert.deepEqual(h.nativeState.calls.filter(c=>c[0]==='set'),[['set',true]]);
  assert.equal(h.node('developerModeBtn').attributes['aria-pressed'],'true');
  assert.equal(h.node('developerModeState').textContent,'On');
  assert.equal(h.node('developerModeBtn').disabled,false);
  assert.equal(h.node('integrationsBtn').disabled,false);
  assert.equal(h.node('integrationsGate').title,'');
  assert.equal(h.node('integrationsGate').tabIndex,-1);
  await h.fire('integrationsBtn');
  assert.equal(h.node('integrationsDialog').open,true);
});

test('disabling Developer Mode closes an open wizard and locks MakerWorld again',async()=>{
  const h=harness({developerMode:true});
  await settle();
  await h.fire('integrationsBtn');
  assert.equal(h.node('integrationsDialog').open,true);
  await h.fire('developerModeBtn');
  assert.deepEqual(h.nativeState.calls.filter(c=>c[0]==='set'),[['set',false]]);
  assert.equal(h.node('integrationsDialog').open,false);
  assert.equal(h.node('integrationsBtn').disabled,true);
  assert.equal(h.node('integrationsGate').title,LOCKED_EN);
  assert.equal(h.node('developerModeBtn').attributes['aria-pressed'],'false');
  await h.fire('integrationsBtn');
  assert.equal(h.node('integrationsDialog').open,false);
});

test('a bridge that is missing or fails keeps MakerWorld locked and reports the error in About',async()=>{
  const missing=harness({bridge:false});
  await settle();
  assert.equal(missing.node('integrationsBtn').disabled,true);
  await missing.fire('developerModeBtn');
  assert.equal(missing.node('developerModeBtn').attributes['aria-pressed'],'false');
  assert.equal(missing.node('developerModeState').textContent,'Off');
  assert.equal(missing.node('developerModeStatus').textContent,'Could not change developer mode right now.');
  assert.equal(missing.node('integrationsBtn').disabled,true);
  await missing.fire('integrationsBtn');
  assert.equal(missing.node('integrationsDialog').open,false);

  const readFails=harness({get:'reject',developerMode:true});
  await settle();
  assert.equal(readFails.node('integrationsBtn').disabled,true);
  await readFails.fire('integrationsBtn');
  assert.equal(readFails.node('integrationsDialog').open,false);

  for (const set of ['reject','nonboolean']) {
    const h=harness({set});
    await settle();
    await h.fire('developerModeBtn');
    assert.equal(h.node('developerModeBtn').attributes['aria-pressed'],'false',set);
    assert.equal(h.node('developerModeStatus').textContent,'Could not change developer mode right now.',set);
    assert.equal(h.node('developerModeBtn').disabled,false,set);
    assert.equal(h.node('integrationsBtn').disabled,true,set);
    await h.fire('integrationsBtn');
    assert.equal(h.node('integrationsDialog').open,false,set);
  }
});

test('a bridge that becomes ready after load is picked up without a click',async()=>{
  const h=harness({bridge:false,developerMode:true});
  await settle();
  assert.equal(h.node('integrationsBtn').disabled,true);
  await wait(20);
  h.win.go={main:{App:h.api}};
  await wait(120);
  assert.equal(h.node('integrationsBtn').disabled,false);
});

test('MakerWorld tooltip and Developer Mode copy switch between Hebrew and English',async()=>{
  const h=harness();
  await settle();
  await h.setLang('he');
  assert.equal(h.node('integrationsGate').title,LOCKED_HE);
  assert.equal(h.node('developerModeState').textContent,'כבוי');
  assert.equal(h.i18n('devTitle'),'מצב מפתחים');
  assert.match(h.i18n('devWarning'),/משתמשים מתקדמים בלבד/);
  await h.setLang('en');
  assert.equal(h.node('integrationsGate').title,LOCKED_EN);
  assert.equal(h.node('developerModeState').textContent,'Off');
  assert.equal(h.i18n('devTitle'),'Developer mode');
  assert.match(h.i18n('devWarning'),/advanced users only/);
  await h.fire('developerModeBtn');
  assert.equal(h.node('developerModeState').textContent,'On');
  await h.setLang('he');
  assert.equal(h.node('developerModeState').textContent,'פעיל');
  assert.equal(h.node('integrationsGate').title,'');
});

test('index.html locks the button with disabled, wraps it in a focusable tooltip, and only the disabled button ignores pointer events',()=>{
  const gate=html.match(/<span id="integrationsGate"[^>]*>\s*<button id="integrationsBtn"[^>]*>/);
  assert.ok(gate);
  assert.match(gate[0],/class="topbar-gate"[^>]*tabindex="0"/);
  assert.match(gate[0],/title="חיבור ל־MakerWorld נעול\. פתח אודות ← מצב מפתחים כדי לפתוח אותו\."/);
  assert.match(gate[0],/<button id="integrationsBtn"[^>]*\sdisabled>/);
  assert.equal((html.match(/id="integrationsBtn"/g)||[]).length,1);
  assert.match(css,/#integrationsBtn:disabled\s*\{[^}]*pointer-events:none/);
  assert.doesNotMatch(css.match(/\.topbar-gate\s*\{[^}]*\}/)[0],/pointer-events/);
  assert.equal(html.match(/integrationsGate"[^>]*title="([^"]+)"/)[1],LOCKED_HE);
  assert.match(integrations,/makerworldLocked:'חיבור ל־MakerWorld נעול\. פתח אודות ← מצב מפתחים כדי לפתוח אותו\.'/);
  assert.match(integrations,/makerworldLocked:'MakerWorld setup is locked\. Open About → Developer mode to unlock it\.'/);
});

test('native state is the only source of truth: no browser storage and no network fetch',()=>{
  for (const source of [about,integrations]) {
    assert.doesNotMatch(source,/localStorage|sessionStorage|indexedDB/);
    assert.doesNotMatch(source,/\bfetch\s*\(|XMLHttpRequest/);
  }
  assert.match(about,/GetDeveloperMode/);
  assert.match(about,/SetDeveloperMode\(requested\)/);
  assert.match(integrations,/GetDeveloperMode/);
  assert.doesNotMatch(integrations,/SetDeveloperMode/);
});
