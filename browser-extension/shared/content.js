/* SliceBridge MakerWorld browser integration, limited to MakerWorld pages.
   Default: Off. Explicit modes: off / ask / always. */
(() => {
  'use strict';
  if (window.top !== window) return;
  const hostname = location.hostname.toLowerCase().replace(/\.$/, '');
  if (hostname !== 'makerworld.com' && !hostname.endsWith('.makerworld.com')) return;
  const ext = globalThis.browser || globalThis.chrome;
  const modes = new Set(['off', 'ask', 'always']);
  let mode = 'off';
  let popup = null;

  function saveMode(next) {
    if (!modes.has(next)) return;
    mode = next;
    try { ext?.storage?.local?.set({ makerworldMode: next }); } catch {}
  }
  function loadMode() {
    const store = ext?.storage?.local;
    if (!store) return;
    try {
      const maybe = store.get({ makerworldMode: 'off' }, value => {
        if (!maybe?.then) mode = modes.has(value?.makerworldMode) ? value.makerworldMode : 'off';
      });
      if (maybe?.then) maybe.then(value => {
        mode = modes.has(value?.makerworldMode) ? value.makerworldMode : 'off';
      }).catch(() => {});
      ext.storage.onChanged?.addListener((changes, area) => {
        if (area === 'local' && changes.makerworldMode)
          mode = modes.has(changes.makerworldMode.newValue) ? changes.makerworldMode.newValue : 'off';
      });
    } catch { mode = 'off'; }
  }
  function trustedHostname(host) {
    host = host.toLowerCase().replace(/\.$/, '');
    return host === 'makerworld.com' || host.endsWith('.makerworld.com') ||
      host === 'bblmw.com' || host.endsWith('.bblmw.com');
  }
  function remoteFromBambu(raw) {
    if (typeof raw !== 'string' || raw.length > 32768) return null;
    const lower = raw.toLowerCase();
    let candidate = '';
    if (lower.startsWith('bambustudio://')) {
      let outer;
      try { outer = new URL(raw); } catch { return null; }
      if (outer.hostname.toLowerCase() !== 'open') return null;
      candidate = outer.searchParams.get('file') || '';
    } else if (lower.startsWith('bambustudioopen://')) {
      candidate = raw.slice('bambustudioopen://'.length);
    } else return null;
    if (!candidate) return null;
    try {
      if (/^https%3a%2f%2f/i.test(candidate)) candidate = decodeURIComponent(candidate);
      const remote = new URL(candidate);
      if (remote.protocol !== 'https:' || remote.username || remote.password ||
          !trustedHostname(remote.hostname) || (remote.port && remote.port !== '443'))
        return null;
      return remote.href;
    } catch { return null; }
  }
  function launchSliceBridge(remote) {
    // Never replace bambustudio:// or leak browser credentials.
    location.href = 'slicebridge://import?url=' + encodeURIComponent(remote);
  }
  function offerChoice(remote, original) {
    if (popup) return;
    const he = /^he\b/i.test(document.documentElement.lang || navigator.language || '');
    const copy = he ? {
      heading: 'לאן לפתוח את הפרויקט?',
      info: 'SliceBridge ינסה להוריד את קובץ ה־3MF דרך הקישור שהאתר מספק. Bambu Studio יישאר זמין.',
      once: 'SliceBridge הפעם',always: 'תמיד ב־SliceBridge',
      bambu: 'Bambu Studio',cancel: 'ביטול'
    } : {
      heading: 'Open this project with…',
      info: 'SliceBridge will try to import the 3MF using MakerWorld’s link. Bambu Studio remains available.',
      once: 'SliceBridge this time',always: 'Always use SliceBridge',
      bambu: 'Bambu Studio',cancel: 'Cancel'
    };
    const previousFocus = document.activeElement;
    const wrap = document.createElement('div');
    wrap.style.cssText = 'position:fixed;inset:0;z-index:2147483647;display:grid;place-items:center;background:rgba(6,34,28,.44);font-family:system-ui,Arial,sans-serif;';
    const shadow = wrap.attachShadow({ mode: 'closed' });
    const style = document.createElement('style');
    style.textContent = `
    *{box-sizing:border-box}section{width:min(92vw,420px);background:#F6FEFA;color:#193F38;
      border:1px solid #A7F3D0;border-radius:16px;padding:22px;box-shadow:0 18px 58px #0006}
    h2{font-size:19px;margin:0 0 9px}p{font-size:13px;line-height:1.55;margin:0 0 18px;color:#43645A}
    nav{display:grid;gap:8px}button{min-height:40px;border:1px solid #A9D7C6;border-radius:9px;
      font-weight:640;background:white;color:#193F38;font-size:13px;cursor:pointer}
    button.primary{background:#166E63;color:white;border-color:#166E63}
    button:focus-visible{outline:3px solid #2CC4A7}
    `;
    const section = document.createElement('section');
    section.dir = he ? 'rtl' : 'ltr';
    section.setAttribute('role', 'dialog');
    section.setAttribute('aria-modal', 'true');
    const title = document.createElement('h2');
    title.id = 'slicebridge-choice-heading';
    title.textContent = copy.heading;
    section.setAttribute('aria-labelledby', title.id);
    const detail = document.createElement('p'); detail.textContent = copy.info;
    const nav = document.createElement('nav');
    const close = () => {
      wrap.remove(); popup = null;
      if (previousFocus?.isConnected && typeof previousFocus.focus === 'function') previousFocus.focus();
    };
    const add = (label,callback,primary=false) => {
      const b=document.createElement('button');
      b.type='button';b.textContent=label;
      if(primary) b.className='primary';
      b.addEventListener('click',()=>{close();callback();});
      nav.append(b);
      return b;
    };
    const first = add(copy.once,()=>launchSliceBridge(remote),true);
    add(copy.always,()=>{saveMode('always');launchSliceBridge(remote);});
    add(copy.bambu,()=>{location.href=original;});
    add(copy.cancel,()=>{});
    section.append(title,detail,nav);shadow.append(style,section);
    wrap.addEventListener('keydown', e => {
      if(e.key==='Escape'){e.preventDefault();close();}
      if(e.key==='Tab'){
        const buttons = Array.from(nav.querySelectorAll('button:not([disabled])'));
        const active = shadow.activeElement;
        if (!buttons.length) {e.preventDefault();return;}
        if (e.shiftKey && (active===buttons[0] || !buttons.includes(active))) {
          e.preventDefault(); buttons[buttons.length-1].focus();
        } else if (!e.shiftKey && (active===buttons[buttons.length-1] || !buttons.includes(active))) {
          e.preventDefault(); buttons[0].focus();
        }
      }
    });
    wrap.addEventListener('mousedown',e=>{if(e.target===wrap)close();});
    (document.body || document.documentElement).append(wrap);
    popup=wrap;
    first.focus();
  }
  loadMode();
  // Works with ordinary anchor links and many dynamic anchors. If MakerWorld
  // uses direct JS protocol navigation without an anchor, let Bambu handle it.
  document.addEventListener('click', event => {
    if(mode==='off' || event.defaultPrevented || event.button>0 ||
      event.ctrlKey||event.metaKey||event.shiftKey||event.altKey) return;
    const anchor = event.target?.closest?.('a[href]');
    if(!anchor) return;
    const original=anchor.getAttribute('href') || '';
    const remote=remoteFromBambu(original);
    if(!remote) return;
    event.preventDefault();
    event.stopImmediatePropagation();
    if(mode==='always') launchSliceBridge(remote);
    else offerChoice(remote,original);
  },true);
})();
