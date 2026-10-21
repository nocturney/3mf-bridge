'use strict';
const {test}=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const path=require('node:path');
const vm=require('node:vm');

const native=path.resolve(__dirname,'..');
const html=fs.readFileSync(path.join(native,'frontend/dist/index.html'),'utf8');
const script=fs.readFileSync(path.join(native,'frontend/dist/integrations.js'),'utf8');

const settle=()=>new Promise(resolve=>setImmediate(resolve));

function harness(storeURL='', developerMode=true) {
  const calls=[];
  const elements=new Map();
  function node(name) {
    if(elements.has(name))return elements.get(name);
    const classes=new Set();
    const el={
      name, hidden:false, disabled:false, open:false, textContent:'',
      value:name==='browserTargetSelect'?'edge':'', dataset:{},
      classList:{
        add:(...s)=>s.forEach(x=>classes.add(x)),
        remove:(...s)=>s.forEach(x=>classes.delete(x)),
        toggle:(s,value)=>{if(value)classes.add(s);else classes.delete(s);}
      },
      listeners:{},children:[],
      parentElement:{classList:{
        add(){},remove(){},toggle(){}
      }},
      addEventListener(type,fn){this.listeners[type]=fn;},
      querySelectorAll(selector) {
        if(selector==='input[name="makerworldMode"]')return [node('mode-off'),node('mode-ask'),node('mode-always')];
        if(selector==='.integration-instructions li')return [node('instruction1'),node('instruction2'),node('instruction3')];
        if(selector==='[data-i18n]')return [];
        return [];
      },
      replaceChildren(...children){this.children=children;},
      append(child){this.children.push(child);},
      showModal(){this.open=true;},
      close(){this.open=false;}
    };
    elements.set(name,el);
    return el;
  }
  const api={
    GetDeveloperMode:async()=>developerMode,
    BrowserLinkStatus:async()=>({platform:'windows',registered:true}),
    BrowserExtensionStoreURL:async(browser)=>{calls.push(['listing',browser]);return storeURL;},
    PrepareBrowserExtension:async(browser)=>{calls.push(['prepare',browser]);return {folder:'/tmp/SliceBridge/BrowserExtensions/chromium'};},
    OpenBrowserExtensionsPage:async(browser)=>{calls.push(['manager',browser]);},
    OpenBrowserExtensionStore:async(browser)=>{calls.push(['store',browser]);},
    OpenMakerWorldMode:async(browser,mode)=>{calls.push(['mode',browser,mode]);}
  };
  const context={
    document:{
      documentElement:{lang:'he',dir:'rtl'},
      getElementById:node,
      createElement:node,
      createTextNode:text=>({textContent:text}),
      addEventListener(){}
    },
    window:{go:{main:{App:api}}},
    setTimeout:setTimeout,
    URL
  };
  vm.runInNewContext(script,context,{filename:'integrations.js'});
  return {elements,calls,node};
}

test('install wizard has one CTA and keeps technical steps folded',()=>{
  for(const id of ['installExtensionBtn','extensionStoreStatus','betaInstallDetails',
    'prepareExtensionBtn','openExtensionsBtn','browserTargetSelect']){
    assert.equal((html.match(new RegExp('id="'+id+'"','g'))||[]).length,1);
  }
  const nested=html.match(/<details id="betaInstallDetails"[\s\S]*?<\/details>/);
  assert.ok(nested);
  assert.match(nested[0],/Load unpacked/);
  assert.doesNotMatch(html,/<details id="betaInstallDetails"[^>]*\bopen\b/);
  assert.doesNotMatch(script,/fetch\s*\(|XMLHttpRequest|window\.open/);
});

test('without published store, one guided click prepares the beta and opens selected browser',async()=>{
  const h=harness();
  await settle();
  h.node('integrationsBtn').listeners.click();
  await new Promise(resolve=>setImmediate(resolve));
  assert.match(h.node('extensionStoreStatus').textContent,/לא פורסם בחנות/);
  assert.equal(h.node('installExtensionBtn').textContent,'התחל התקנת בטא מודרכת');
  assert.equal(h.node('betaInstallDetails').open,false);
  await h.node('installExtensionBtn').listeners.click();
  assert.equal(h.node('betaInstallDetails').open,true);
  assert.ok(h.calls.some(c=>c[0]==='prepare'&&c[1]==='edge'));
  assert.ok(h.calls.some(c=>c[0]==='manager'&&c[1]==='edge'));
  assert.ok(!h.calls.some(c=>c[0]==='store'));
  assert.match(h.node('extensionStoreStatus').textContent,/נפתחו/);
});

test('verified store link launches store only, with no silent sideload',async()=>{
  const h=harness('https://microsoftedge.microsoft.com/addons/detail/example/abcdefghijklmnopabcdefghijklmnop');
  await settle();
  h.node('integrationsBtn').listeners.click();
  await new Promise(resolve=>setImmediate(resolve));
  assert.equal(h.node('installExtensionBtn').textContent,'התקן מחנות הדפדפן');
  await h.node('installExtensionBtn').listeners.click();
  assert.ok(h.calls.some(c=>c[0]==='store'&&c[1]==='edge'));
  assert.ok(!h.calls.some(c=>c[0]==='prepare'));
  assert.equal(h.node('betaInstallDetails').open,false);
});

test('Firefox selection shows temporary beta limitation and safe browser path',async()=>{
  const h=harness();
  await settle();
  h.node('integrationsBtn').listeners.click();
  h.node('browserTargetSelect').value='firefox';
  h.node('browserTargetSelect').listeners.change();
  await new Promise(resolve=>setImmediate(resolve));
  assert.equal(h.node('browserSettingsAddress').textContent,'about:debugging#/runtime/this-firefox');
  assert.equal(h.node('firefoxInstallWarning').hidden,false);
  await h.node('installExtensionBtn').listeners.click();
  assert.ok(h.calls.some(c=>c[0]==='prepare'&&c[1]==='firefox'));
  assert.ok(h.calls.some(c=>c[0]==='manager'&&c[1]==='firefox'));
});

test('an off-domain HTTPS link cannot enable storefront installation',async()=>{
  const h=harness('https://bad.example/detail/fake');
  await settle();
  h.node('integrationsBtn').listeners.click();
  await new Promise(resolve=>setImmediate(resolve));
  assert.equal(h.node('installExtensionBtn').textContent,'התחל התקנת בטא מודרכת');
  await h.node('installExtensionBtn').listeners.click();
  assert.ok(h.calls.some(c=>c[0]==='prepare'));
  assert.ok(!h.calls.some(c=>c[0]==='store'));
});
