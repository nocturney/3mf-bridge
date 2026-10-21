'use strict';
// Runs native/frontend/dist scripts inside node's vm with a small fake DOM.
// Only what the scripts touch is modelled: ids, classes, attributes, properties,
// text, children, listeners and simple selectors. No layout, no HTML parsing.
const vm=require('node:vm');
const fs=require('node:fs');
const path=require('node:path');
const picker=require('../frontend/dist/printer-picker.js');

const dist=path.resolve(__dirname,'../frontend/dist');
const scriptPath=path.join(dist,'my-printers.js');
const catalog=JSON.parse(fs.readFileSync(path.join(dist,'printer-catalog.json'),'utf8'));

function camel(name){return name.replace(/-([a-z])/g,(_,c)=>c.toUpperCase());}

class FakeElement{
  constructor(tag='div',id=''){
    this.tagName=String(tag).toUpperCase();
    this.id=id;
    this.children=[];
    this.parent=null;
    this.listeners={};
    this.attributes={};
    this.dataset={};
    this.classes=new Set();
    this.style={};
    this.value='';
    this.checked=false;
    this.disabled=false;
    this.hidden=false;
    this.open=false;
    this.textContent='';
    this.placeholder='';
  }
  get parentElement(){return this.parent;}
  get className(){return [...this.classes].join(' ');}
  set className(value){this.classes=new Set(String(value).split(/\s+/).filter(Boolean));}
  get classList(){
    const classes=this.classes;
    return {
      add:(...names)=>names.forEach((name)=>classes.add(name)),
      remove:(...names)=>names.forEach((name)=>classes.delete(name)),
      toggle:(name,force)=>{
        const on=force===undefined?!classes.has(name):Boolean(force);
        if(on)classes.add(name);else classes.delete(name);
        return on;
      },
      contains:(name)=>classes.has(name),
    };
  }
  addEventListener(type,fn){(this.listeners[type]??=[]).push(fn);}
  fire(type){for(const fn of this.listeners[type]??[])fn({type,target:this,currentTarget:this});}
  append(...nodes){
    for(const child of nodes){
      if(child==null||typeof child==='string')continue;
      child.parent=this;this.children.push(child);
    }
  }
  replaceChildren(...nodes){
    for(const child of this.children)child.parent=null;
    this.children=[];
    this.append(...nodes);
  }
  remove(){
    if(!this.parent)return;
    this.parent.children=this.parent.children.filter((child)=>child!==this);
    this.parent=null;
  }
  setAttribute(name,value){this.attributes[name]=String(value);}
  getAttribute(name){return name in this.attributes?this.attributes[name]:null;}
  removeAttribute(name){delete this.attributes[name];}
  focus(){}
  showModal(){this.open=true;}
  close(){if(!this.open)return;this.open=false;this.fire('close');}
  querySelector(selector){return selectAll(this,selector)[0]??null;}
  querySelectorAll(selector){return selectAll(this,selector);}
  closest(selector){
    const chain=parseChain(selector);
    for(let node=this;node;node=node.parent){
      if(matchesChain(node,chain))return node;
    }
    return null;
  }
}

// Compound selectors: tag, #id, .class, [attr], [attr=value], [attr="value"],
// :checked, :disabled and :not(:pseudo), joined by spaces (descendant).
const tokenPattern=/^(?:([a-zA-Z][\w-]*)|#([\w-]+)|\.([\w-]+)|\[([\w-]+)(?:=(?:"([^"]*)"|([^\]]*)))?\]|:not\(:(\w+)\)|:(\w+))/;
function parseCompound(text){
  const out={tag:null,id:null,classes:[],attrs:[],pseudos:[]};
  let rest=text;
  while(rest){
    const match=rest.match(tokenPattern);
    if(!match)throw Error('unsupported selector: '+text);
    rest=rest.slice(match[0].length);
    if(match[1])out.tag=match[1].toUpperCase();
    else if(match[2])out.id=match[2];
    else if(match[3])out.classes.push(match[3]);
    else if(match[4])out.attrs.push({name:match[4],value:match[5]??match[6]});
    else if(match[7])out.pseudos.push({name:match[7],negate:true});
    else out.pseudos.push({name:match[8],negate:false});
  }
  return out;
}
function parseChain(selector){return String(selector).trim().split(/\s+/).map(parseCompound);}
function attrValue(el,name){
  if(name==='value')return el.value;
  if(name.startsWith('data-'))return el.dataset[camel(name.slice(5))]??null;
  if(name in el&&typeof el[name]!=='function'){
    const value=el[name];
    if(typeof value==='boolean')return value?'':null;
    return value==null?null:String(value);
  }
  return el.getAttribute(name);
}
function pseudoMatches(el,name){
  if(name==='checked')return Boolean(el.checked);
  if(name==='disabled')return Boolean(el.disabled);
  throw Error('unsupported pseudo-class: '+name);
}
function matchesCompound(el,compound){
  if(!el||typeof el!=='object'||!el.tagName)return false;
  if(compound.tag&&el.tagName!==compound.tag)return false;
  if(compound.id&&el.id!==compound.id)return false;
  if(compound.classes.some((name)=>!el.classes.has(name)))return false;
  if(!compound.attrs.every(({name,value})=>{
    const actual=attrValue(el,name);
    if(actual==null)return false;
    return value===undefined||String(actual)===value;
  }))return false;
  return compound.pseudos.every(({name,negate})=>pseudoMatches(el,name)!==negate);
}
function matchesChain(el,chain){
  if(!matchesCompound(el,chain[chain.length-1]))return false;
  let index=chain.length-2;
  for(let node=el.parent;node&&index>=0;node=node.parent){
    if(matchesCompound(node,chain[index]))index--;
  }
  return index<0;
}
function selectAll(root,selector){
  const chain=parseChain(selector);
  const out=[];
  const walk=(node)=>{
    for(const child of node.children){
      if(typeof child==='string')continue;
      if(matchesChain(child,chain))out.push(child);
      walk(child);
    }
  };
  walk(root);
  return out;
}

// Every id a script asks for exists, as it does on the real page. Elements are
// attached under one root so document-level selectors can find them. The two
// elements that hold children in index.html get the same structure here.
function createDocument(){
  const root=new FakeElement('html');
  const byId=new Map();
  const seeded={
    myPrintersBtn:(el)=>el.append(new FakeElement('span')),
  };
  const getElementById=(id)=>{
    if(!byId.has(id)){
      const el=new FakeElement('div',id);
      byId.set(id,el);
      root.append(el);
      seeded[id]?.(el);
    }
    return byId.get(id);
  };
  return {
    documentElement:{lang:'en'},
    body:{dir:'ltr'},
    root,
    getElementById,
    createElement:(tag)=>new FakeElement(tag),
    createDocumentFragment:()=>new FakeElement('fragment'),
    querySelector:(selector)=>selectAll(root,selector)[0]??null,
    querySelectorAll:(selector)=>selectAll(root,selector),
    addEventListener(){},
  };
}

// A window that really delivers the events scripts dispatch to each other.
function createWindow(app){
  const listeners={};
  return {
    go:{main:{App:app}},
    addEventListener(type,fn){(listeners[type]??=[]).push(fn);},
    dispatchEvent(event){for(const fn of listeners[event.type]??[])fn(event);return true;},
  };
}
class FakeEvent{constructor(type){this.type=type;}}

// Runs the given scripts, in order, in one vm context.
function runScripts(files,context){
  for(const file of files){
    vm.runInNewContext(fs.readFileSync(file,'utf8'),context,{filename:file});
  }
}

// Wails-style App stub. Every call is recorded in order, including test stubs.
function recordingApp(app={}){
  const calls=[];
  const defaults={
    PrinterCatalog:()=>catalog,
    GetMyPrinters:()=>({selectedIds:[],onboardingPending:false}),
  };
  const merged={...defaults,...app};
  const wailsApp={};
  for(const [key,fn] of Object.entries(merged)){
    wailsApp[key]=typeof fn==='function'?(...args)=>{calls.push(key);return fn(...args);}:fn;
  }
  return {wailsApp,calls};
}

// Loads the real My Printers module against the given fake Wails App object.
// Returns the module API plus the fake DOM, recorded window events and app calls.
function loadMyPrinters(app={}){
  const document=createDocument();
  const {wailsApp,calls}=recordingApp(app);
  const window=createWindow(wailsApp);
  window.SliceBridgePicker=picker;
  const events=[];
  const recordEvents=window.dispatchEvent;
  window.dispatchEvent=(event)=>{events.push(event.type);return recordEvents(event);};
  runScripts([scriptPath],{window,document,Event:FakeEvent,setTimeout,clearTimeout});
  return {
    api:window.SliceBridgeMyPrinters,
    document,
    dom:(id)=>document.getElementById(id),
    recorded:{calls,events},
    calls,
    plain:(value)=>JSON.parse(JSON.stringify(value)),
  };
}

// Built-in adapter list as App.Availability() reports it.
function availabilityAll(overrides={}){
  return ['u1','c5','cc2','qidi'].map((target)=>({
    target,name:target,slicer:'Installed slicer',available:overrides[target]??true,
  }));
}

module.exports={
  loadMyPrinters,availabilityAll,catalog,dist,
  FakeElement,createDocument,createWindow,FakeEvent,recordingApp,runScripts,
  picker,
};
