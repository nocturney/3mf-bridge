const { app, BrowserWindow, ipcMain, dialog, shell } = require('electron');
const path = require('path');
const fs = require('fs/promises');
const { execFile } = require('node:child_process');
const { promisify } = require('node:util');
const runExecutable = promisify(execFile);

let lastOutputFolder = null;

const TARGETS = {
  u1: 'U1',
  c5: 'C5',
  cc2: 'CC2',
  qidi: 'QIDI'
};

function createWindow() {
  const win = new BrowserWindow({
    width: 1180,
    height: 820,
    minWidth: 980,
    minHeight: 700,
    backgroundColor: '#f7f6f2',
    autoHideMenuBar: true,
    webPreferences: {
      contextIsolation: true,
      nodeIntegration: false,
      preload: path.join(__dirname, 'preload.js')
    }
  });

  win.loadFile(path.join(__dirname, 'index.html'));
}

function formatBytes(bytes) {
  if (!Number.isFinite(bytes)) return '';
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
}

async function ensureUniquePath(dir, fileName) {
  const parsed = path.parse(fileName);
  let candidate = path.join(dir, fileName);
  let counter = 2;

  while (true) {
    try {
      await fs.access(candidate);
      candidate = path.join(dir, `${parsed.name}-${counter}${parsed.ext}`);
      counter += 1;
    } catch {
      return candidate;
    }
  }
}

function parseEngineReply(stdout) {
  try {
    const result = JSON.parse(String(stdout || '').trim());
    return result && typeof result === 'object' ? result : null;
  } catch {
    return null;
  }
}

async function convertWithVerifiedEngine(filePath, outputPath) {
  const executable = app.isPackaged
    ? path.join(process.resourcesPath, 'bin', 'bridge-core.exe')
    : path.join(__dirname, 'bin', 'bridge-core.exe');

  try {
    await fs.access(executable);
  } catch {
    return {
      ok: false,
      code: 'ENGINE_NOT_BUILT',
      error: 'מנוע ההמרה לא הותקן. יש לבנות את bridge-core לפני שימוש.'
    };
  }

  try {
    const { stdout } = await runExecutable(executable, [
      '--source', filePath, '--output', outputPath, '--target', 'u1'
    ], {
      windowsHide: true,
      timeout: 180000,
      maxBuffer: 256 * 1024
    });
    return parseEngineReply(stdout) || {
      ok: false, code: 'ENGINE_NO_REPORT', error: 'לא התקבל דוח תקין ממנוע ההמרה.'
    };
  } catch (err) {
    return parseEngineReply(err.stdout) || {
      ok: false,
      code: 'ENGINE_FAILED',
      error: 'מנוע ההמרה נכשל. הקובץ המקורי לא הוחלף.'
    };
  }
}

app.whenReady().then(() => {
  createWindow();

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) createWindow();
  });
});

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') app.quit();
});

ipcMain.handle('pick-3mf-file', async () => {
  const result = await dialog.showOpenDialog({
    title: 'בחר קובץ 3MF',
    properties: ['openFile'],
    filters: [
      { name: '3MF Files', extensions: ['3mf'] },
      { name: 'All Files', extensions: ['*'] }
    ]
  });

  if (result.canceled || !result.filePaths.length) {
    return { canceled: true };
  }

  const filePath = result.filePaths[0];
  const stats = await fs.stat(filePath);

  return {
    canceled: false,
    file: {
      path: filePath,
      name: path.basename(filePath),
      folder: path.dirname(filePath),
      size: stats.size,
      sizeLabel: formatBytes(stats.size)
    }
  };
});

ipcMain.handle('inspect-dropped-file', async (_event, droppedPath) => {
  if (!droppedPath) {
    return { ok: false, error: 'לא התקבל נתיב קובץ.' };
  }

  const ext = path.extname(droppedPath).toLowerCase();
  if (ext !== '.3mf') {
    return { ok: false, error: 'יש לבחור קובץ עם סיומת 3MF בלבד.' };
  }

  const stats = await fs.stat(droppedPath);

  return {
    ok: true,
    file: {
      path: droppedPath,
      name: path.basename(droppedPath),
      folder: path.dirname(droppedPath),
      size: stats.size,
      sizeLabel: formatBytes(stats.size)
    }
  };
});

ipcMain.handle('convert-3mf', async (_event, payload) => {
  const { filePath, targets } = payload || {};

  if (!filePath) {
    return { ok: false, error: 'לא נבחר קובץ מקור.' };
  }

  if (!Array.isArray(targets) || targets.length === 0) {
    return { ok: false, error: 'לא נבחרה אף מדפסת יעד.' };
  }

  const ext = path.extname(filePath).toLowerCase();
  if (ext !== '.3mf') {
    return { ok: false, error: 'קובץ המקור חייב להיות 3MF.' };
  }

  const sourceDir = path.dirname(filePath);
  const baseName = path.basename(filePath);

  // Never present a copied Bambu file as a converted project.
  if (targets.length !== 1 || targets[0] !== 'u1') {
    return {
      ok: false,
      code: 'TARGET_NOT_VALIDATED',
      error: 'כרגע קיימת המרה מאומתת ניסיונית ל-Snapmaker U1 בלבד. שאר היעדים עדיין בבדיקות.'
    };
  }
  let sourceStat;
  try {
    sourceStat = await fs.stat(filePath);
  } catch {
    return { ok: false, error: 'קובץ המקור לא נמצא.' };
  }
  if (!sourceStat.isFile() || sourceStat.size > 250 * 1024 * 1024) {
    return { ok: false, error: 'יש לבחור קובץ 3MF רגיל בגודל עד 250MB.' };
  }

  for (let attempt = 1; attempt <= 100; attempt += 1) {
    const baseOutputName = `${TARGETS.u1}-${baseName}`;
    const outputName = attempt === 1
      ? baseOutputName
      : `${TARGETS.u1}-${path.parse(baseName).name}-${attempt}${path.extname(baseName)}`;
    const outputPath = path.join(sourceDir, outputName);
    const conversion = await convertWithVerifiedEngine(filePath, outputPath);
    if (conversion.code === 'OUTPUT_EXISTS') continue;
    if (!conversion.ok) {
      return {
        ok: false,
        code: conversion.code || 'CONVERSION_FAILED',
        error: conversion.error || 'המרת הקובץ נכשלה.'
      };
    }
    lastOutputFolder = sourceDir;
    return {
      ok: true,
      folder: sourceDir,
      results: [{
        target: 'u1',
        prefix: 'U1',
        outputPath,
        outputName,
        report: conversion
      }]
    };
  }
  return { ok: false, error: 'קיימים יותר מדי קובצי פלט עם אותו שם.' };
});

ipcMain.handle('open-output-folder', async () => {
  if (!lastOutputFolder) {
    return { ok: false, error: 'עדיין אין תיקיית פלט לפתיחה.' };
  }

  const error = await shell.openPath(lastOutputFolder);
  return error ? { ok: false, error } : { ok: true };
});