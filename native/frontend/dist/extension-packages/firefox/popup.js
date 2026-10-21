(() => {
  'use strict';
  const ext = globalThis.browser || globalThis.chrome;
  const modes = new Set(['off', 'ask', 'always']);
  const status = document.getElementById('status');
  const radios = Array.from(document.querySelectorAll('input[name="mode"]'));
  function choose(mode) {
    for(const radio of radios) radio.checked = radio.value === mode;
  }
  async function read() {
    try {
      const existing = await ext.storage.local.get({makerworldMode:'off'});
      choose(modes.has(existing.makerworldMode) ? existing.makerworldMode : 'off');
    } catch {
      status.textContent = 'Browser storage is unavailable.';
      for(const radio of radios) radio.disabled = true;
    }
  }
  for(const radio of radios) {
    radio.addEventListener('change', async () => {
      if(!radio.checked) return;
      for(const item of radios) item.disabled = true;
      try {
        await ext.storage.local.set({makerworldMode:radio.value});
        status.textContent = radio.value === 'off' ? 'Off. Bambu Studio remains unchanged.' :
          radio.value === 'ask' ? 'You will be asked for supported MakerWorld links.' :
            'Supported MakerWorld links will use SliceBridge.';
      } catch {
        status.textContent = 'Could not save mode.';
        await read();
      } finally {
        for(const item of radios) item.disabled = false;
      }
    });
  }
  read();
})();
