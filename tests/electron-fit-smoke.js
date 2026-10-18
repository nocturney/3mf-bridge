// Run with Electron, not Node: electron tests/electron-fit-smoke.js
// Intercepts show() to test Chromium measurements without opening a window.
const { app, BrowserWindow, screen } = require('electron');
app.disableHardwareAcceleration();

let observed = false;
const watchdog = setTimeout(() => {
  if (observed) return;
  console.error('ELECTRON_WINDOW_FIT_TIMEOUT');
  process.exit(2);
}, 12000);

BrowserWindow.prototype.show = function () {
  if (observed) return;
  observed = true;
  clearTimeout(watchdog);
  const win = this;
  Promise.resolve().then(async () => {
    const metrics = await win.webContents.executeJavaScript(`
      (() => {
        const root = document.documentElement;
        const shell = document.querySelector('.shell');
        const content = document.querySelector('.content');
        const targets = document.querySelector('#targetsPanel').getBoundingClientRect();
        const status = document.querySelector('#statusPanel').getBoundingClientRect();
        return {
          width: window.innerWidth,
          height: window.innerHeight,
          scrollWidth: root.scrollWidth,
          scrollHeight: root.scrollHeight,
          shellBottom: shell.getBoundingClientRect().bottom,
          gridColumns: getComputedStyle(content).gridTemplateColumns,
          targetRowTop: targets.top,
          statusRowTop: status.top
        };
      })()
    `);
    const bounds = win.getBounds();
    const workingArea = screen.getDisplayMatching(bounds).workArea;
    const zoom = win.webContents.getZoomFactor();
    const windowOnScreen = bounds.x >= workingArea.x && bounds.y >= workingArea.y &&
      bounds.x + bounds.width <= workingArea.x + workingArea.width &&
      bounds.y + bounds.height <= workingArea.y + workingArea.height;
    const fits = windowOnScreen &&
      metrics.scrollWidth <= metrics.width + 2 &&
      metrics.scrollHeight <= metrics.height + 2 &&
      metrics.shellBottom <= metrics.height + 2;
    const result = { fits, zoom, bounds, workingArea, metrics };
    process.stdout.write('ELECTRON_FIT_SMOKE ' + JSON.stringify(result) + '\n');
    if (!fits) process.exitCode = 1;
  }).catch((err) => {
    console.error('ELECTRON_FIT_SMOKE_ERROR ' + err.stack);
    process.exitCode = 1;
  }).finally(() => {
    app.quit();
  });
};

require('../main.js');
