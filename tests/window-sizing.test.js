const test = require('node:test');
const assert = require('node:assert/strict');
const { fitContentToDisplay, MIN_ZOOM } = require('../desktop/electron/window-sizing');

test('ordinary monitor opens at exactly the CSS content dimensions', () => {
  const fit = fitContentToDisplay({
    contentWidth: 1028,
    contentHeight: 650,
    maxClientWidth: 1800,
    maxClientHeight: 960
  });
  assert.equal(fit.zoomFactor, 1);
  assert.equal(fit.width, 1028);
  assert.equal(fit.height, 650);
  assert.equal(fit.fitsWithoutScrolling, true);
});

test('limited display scales layout to fit its usable work area', () => {
  const fit = fitContentToDisplay({
    contentWidth: 1030,
    contentHeight: 850,
    maxClientWidth: 990,
    maxClientHeight: 700
  });
  assert.ok(fit.zoomFactor < 1);
  assert.ok(fit.zoomFactor >= MIN_ZOOM);
  assert.ok(fit.width <= 990);
  assert.ok(fit.height <= 700);
  assert.equal(fit.fitsWithoutScrolling, true);
});

test('very small monitor preserves readable zoom and reports scroll needed', () => {
  const fit = fitContentToDisplay({
    contentWidth: 1200,
    contentHeight: 1550,
    maxClientWidth: 970,
    maxClientHeight: 670
  });
  assert.equal(fit.zoomFactor, MIN_ZOOM);
  assert.equal(fit.height, 670);
  assert.equal(fit.fitsWithoutScrolling, false);
});

test('small content does not create an unusably tiny window', () => {
  const fit = fitContentToDisplay({
    contentWidth: 400,
    contentHeight: 200,
    maxClientWidth: 1200,
    maxClientHeight: 850
  });
  assert.equal(fit.zoomFactor, 1);
  assert.equal(fit.width, 720);
  assert.equal(fit.height, 520);
});

test('invalid sizes fail without sizing an Electron window', () => {
  assert.throws(() => fitContentToDisplay({
    contentWidth: 0,
    contentHeight: 700,
    maxClientWidth: 1024,
    maxClientHeight: 768
  }), /positive finite/);
  assert.throws(() => fitContentToDisplay({
    contentWidth: NaN,
    contentHeight: 700,
    maxClientWidth: 1024,
    maxClientHeight: 768
  }), /positive finite/);
});

test('desktop layout uses side-by-side panels and leaves mobile layout intact', async () => {
  const fs = require('node:fs/promises');
  const path = require('node:path');
  const html = await fs.readFile(path.join(__dirname, '..', 'desktop', 'electron', 'index.html'), 'utf8');
  assert.match(html, /@media \(min-width: 840px\)/);
  assert.match(html, /grid-template-columns: minmax\(0, 1fr\) minmax\(0, 1fr\)/);
  assert.match(html, /@media \(max-width: 720px\)/);
  for (const id of ['sourcePanel', 'targetsPanel', 'statusPanel', 'mainActions', 'footerHint']) {
    assert.ok(html.includes('id="' + id + '"'), id);
  }
});
