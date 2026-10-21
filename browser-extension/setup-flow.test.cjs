'use strict';
const test=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const vm=require('node:vm');
const path=require('node:path');

const source=fs.readFileSync(path.join(__dirname,'shared','content.js'),'utf8');

function simulate(hash, hostname='makerworld.com') {
  const elements=[];
  const writes=[];
  const storageReads=[];
  const historyUpdates=[];
  const extensions={makerworldMode:'off'};
  const window={};
  window.top=window;
  const body={append(child){child.isConnected=true;},dir:'ltr'};
  function createElement(tag){
    const node={
      tag,children:[],textContent:'',hidden:false,disabled:false,isConnected:false,
      style:{},handlers:{},className:'',id:'',dir:'',
      append(...children){this.children.push(...children);},
      setAttribute(name,value){this[name]=value;},
      addEventListener(name,fn){this.handlers[name]=fn;},
      focus(){node.focused=true;},
      remove(){node.isConnected=false;},
      attachShadow(){
        const shadow={children:[],activeElement:null,append(...children){this.children.push(...children);}};
        node.shadow=shadow;
        return shadow;
      }
    };
    elements.push(node);
    return node;
  }
  const document={
    body,documentElement:{lang:'en'},readyState:'complete',activeElement:null,
    createElement,addEventListener(){}
  };
  const storage={
    get:async (...args)=>{ storageReads.push(args.length); return {...extensions}; },
    async set(value){writes.push(value);Object.assign(extensions,value)}
  };
  const context={
    window,document,location:{hostname,hash,pathname:'/en',search:''},
    navigator:{language:'en-US'},history:{state:null,replaceState(...args){historyUpdates.push(args)}},
    browser:{storage:{local:storage,onChanged:{addListener(){}}}},
    URL,console
  };
  vm.runInNewContext(source,context,{filename:'content.js'});
  return {
    writes,historyUpdates,elements,extensions,storageReads,
    button(text){return elements.find(x=>x.tag==='button'&&x.textContent===text)}
  };
}

test('setup link never changes mode until explicit confirmation',async()=>{
  const app=simulate('#slicebridge-setup=ask');
  assert.equal(app.extensions.makerworldMode,'off');
  assert.deepEqual(app.writes,[]);
  assert.equal(app.historyUpdates.length,1);
  const confirm=app.button('Confirm and save in this browser');
  assert.ok(confirm);
  await confirm.handlers.click();
  assert.equal(app.extensions.makerworldMode,'ask');
  assert.equal(app.writes.length,1);
  assert.equal(confirm.hidden,true);
  assert.ok(app.elements.some(x=>x.id==='status'&&/Saved/.test(x.textContent)));
});
test('cancel never changes an extension preference',async()=>{
  const app=simulate('#slicebridge-setup=always');
  await app.button('Cancel').handlers.click();
  assert.equal(app.extensions.makerworldMode,'off');
  assert.deepEqual(app.writes,[]);
});
test('untrusted mode and hosts never present a setup prompt',()=>{
  for(const [hash,host] of [
    ['#slicebridge-setup=evil','makerworld.com'],
    ['#slicebridge-setup=ask','makerworld.com.evil.test'],
    ['#other=ask','makerworld.com']
  ]){
    const app=simulate(hash,host);
    assert.deepEqual(app.writes,[]);
    assert.equal(app.button('Confirm and save in this browser'),undefined);
  }
});

test('Firefox promise-only storage read never passes an unsupported callback',async()=>{
  const app=simulate('');
  await Promise.resolve();
  assert.deepEqual(app.storageReads,[1]);
});
