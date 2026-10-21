'use strict';
const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const base = path.resolve(__dirname, '..');
const catalog = JSON.parse(fs.readFileSync(path.join(base, 'frontend/dist/printer-catalog.json'), 'utf8'));
const doc = fs.readFileSync(path.join(base, '..', 'docs/printer-release-date-sources.md'), 'utf8');
const DATE = /^(\d{4})-(\d{2})-(\d{2})$/;

test('every releaseDate is null or a real calendar date not in the future', () => {
 const today = new Date().toISOString().slice(0, 10);
 for (const m of catalog.models) {
  if (m.releaseDate === null) continue;
  assert.equal(typeof m.releaseDate, 'string', m.id);
  const match = DATE.exec(m.releaseDate);
  assert.ok(match, `${m.id} releaseDate must be YYYY-MM-DD, got ${m.releaseDate}`);
  const [y, mo, d] = [Number(match[1]), Number(match[2]), Number(match[3])];
  const utc = new Date(Date.UTC(y, mo - 1, d));
  assert.equal(utc.getUTCFullYear(), y, m.id);
  assert.equal(utc.getUTCMonth(), mo - 1, `${m.id} invalid month/day`);
  assert.equal(utc.getUTCDate(), d, `${m.id} invalid day`);
  assert.ok(m.releaseDate <= today, `${m.id} release date is in the future`);
 }
});

test('dated entries are exactly the ones cited in the internal source log', () => {
 const dated = catalog.models.filter((m) => m.releaseDate !== null).map((m) => m.id).sort();
 assert.ok(dated.length > 0, 'expected at least one trusted release date');
 for (const id of dated) {
  assert.ok(doc.includes('`' + id + '`'), `${id} has a releaseDate but no source entry in docs/printer-release-date-sources.md`);
 }
});

test('model list stays sorted by manufacturer, model, then id', () => {
 const key = (m) => [m.manufacturer.toLowerCase(), m.model.toLowerCase(), m.id];
 for (let i = 1; i < catalog.models.length; i++) {
  const a = key(catalog.models[i - 1]);
  const b = key(catalog.models[i]);
  const cmp = a[0] !== b[0] ? a[0].localeCompare(b[0]) : a[1] !== b[1] ? a[1].localeCompare(b[1]) : a[2].localeCompare(b[2]);
  assert.ok(cmp <= 0, `out of order at ${catalog.models[i].id}`);
 }
});

test('release dates are not inferred for models without evidence', () => {
 const byId = Object.fromEntries(catalog.models.map((m) => [m.id, m]));
 for (const id of ['qidi-x-max-3', 'sovol-sv08', 'snapmaker-u1', 'anycubic-kobra-3']) {
  if (byId[id]) assert.equal(byId[id].releaseDate, null, `${id} must stay null without primary evidence`);
 }
});
