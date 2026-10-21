const { app, BrowserWindow, ipcMain, dialog, shell, screen } = require('electron');
const path = require('path');
const fs = require('fs/promises');
const { execFile } = require('node:child_process');
const { promisify } = require('node:util');
const runExecutable = promisify(execFile);
const { fitContentToDisplay } = require('./window-sizing');
const os = require('node:os');
const { PROFILES, loadTargetBundle } = require('./slicer-profiles');

let lastOutputFolder = null;

const TARGETS = {
  u1: 'U1',
  c5: 'C5',
  cc2: 'CC2',
  qidi: 'QIDI'
};

async function fitWindowToInitialContent(win) {
  // Measure after the document and local fonts have loaded. The renderer
  // reports only numeric dimensions, not arbitrary window controls.
  const content = await win.webContents.executeJavaScript(`
    (async () => {
      await document.fonts.ready;
      const shell = document.querySelector('.shell');
      const body = document.body;
      const styles = getComputedStyle(body);
      const horizontalPadding =
        parseFloat(styles.paddingLeft) + parseFloat(styles.paddingRight);
      const verticalPadding =
        parseFloat(styles.paddingTop) + parseFloat(styles.paddingBottom);
      const shellBounds = shell ? shell.getBoundingClientRect() : body.getBoundingClientRect();
      return {
        width: Math.ceil(shellBounds.width + horizontalPadding + 6),
        // Unlike body.scrollHeight, this excludes body's min-height: 100vh.
        height: Math.ceil(shellBounds.height + verticalPadding + 6)
      };
    })()
  `);

  if (win.isDestroyed()) return;
  const windowBounds = win.getBounds();
  const clientBounds = win.getContentBounds();
  const frameWidth = Math.max(0, windowBounds.width - clientBounds.width);
  const frameHeight = Math.max(0, windowBounds.height - clientBounds.height);
  const { workArea } = screen.getDisplayMatching(windowBounds);

  // Leave a small margin for the window frame and any OS-reserved space.
  const fitting = fitContentToDisplay({
    contentWidth: content.width,
    contentHeight: content.height,
    maxClientWidth: Math.max(1, workArea.width - frameWidth - 20),
    maxClientHeight: Math.max(1, workArea.height - frameHeight - 20)
  });
  win.webContents.setZoomFactor(fitting.zoomFactor);
  win.setContentSize(fitting.width, fitting.height);
  win.center();
}

function createWindow() {
  const win = new BrowserWindow({
    width: 1050,
    height: 740,
    minWidth: 620,
    minHeight: 400,
    show: false,
    backgroundColor: '#f7f6f2',
    autoHideMenuBar: true,
    webPreferences: {
      contextIsolation: true,
      nodeIntegration: false,
      preload: path.join(__dirname, 'preload.js')
    }
  });

  win.loadFile(path.join(__dirname, 'index.html'))
    .then(async () => {
      // Let the three installed-slicer checks update their labels first.
      await win.webContents.executeJavaScript('window.bridgeLayoutReady');
      if (!win.isDestroyed()) await fitWindowToInitialContent(win);
    })
    .catch((error) => console.warn('Window size auto-fit skipped:', error.message))
    .finally(() => {
      if (!win.isDestroyed()) win.show();
    });
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

async function inspectSourceNozzle(filePath) {
  const executable = app.isPackaged
    ? path.join(process.resourcesPath, 'bin', 'bridge-core.exe')
    : path.join(__dirname, 'bin', 'bridge-core.exe');
  try {
    const { stdout } = await runExecutable(executable, [
      '--inspect-nozzle', '--source', filePath
    ], { windowsHide: true, timeout: 30000, maxBuffer: 256 * 1024 });
    const parsed = parseEngineReply(stdout);
    return parsed && parsed.ok ? parsed
      : { ok: false, code: 'SOURCE_NOZZLE_INVALID', error: 'לא זוהה קוטר הדיזה במקור.' };
  } catch (err) {
    return parseEngineReply(err.stdout) || {
      ok: false, code: 'SOURCE_NOZZLE_INVALID',
      error: err.message || 'בדיקת קוטר הדיזה במקור נכשלה.'
    };
  }
}

async function convertWithVerifiedEngine(filePath, outputPath, target, sourceNozzle = '0.4') {
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

  let scratch;
  try {
    const args = ['--source', filePath, '--output', outputPath, '--target', target];
    if (target !== 'u1' || sourceNozzle !== '0.4') {
      // Read the matching local nozzle, process and filament presets.
      // The C5 resolver maps source 0.2mm to installed 0.25mm, as approved.
      const bundle = await loadTargetBundle(target, { nozzle: sourceNozzle });
      scratch = await fs.mkdtemp(path.join(os.tmpdir(), '3mf-bridge-profile-'));
      const profileFile = path.join(scratch, 'vendor-preset.json');
      await fs.writeFile(profileFile, JSON.stringify(bundle), { mode: 0o600 });
      args.push('--profile', profileFile);
    }
    const { stdout } = await runExecutable(executable, args, {
      windowsHide: true, timeout: 180000, maxBuffer: 256 * 1024
    });
    return parseEngineReply(stdout) || {
      ok: false, code: 'ENGINE_NO_REPORT', error: 'לא התקבל דוח תקין ממנוע ההמרה.'
    };
  } catch (err) {
    return parseEngineReply(err.stdout) || {
      ok: false, code: 'ENGINE_FAILED',
      error: err.message || 'מנוע ההמרה נכשל. קובץ המקור לא הוחלף.'
    };
  } finally {
    if (scratch) await fs.rm(scratch, { recursive: true, force: true });
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

ipcMain.handle('get-target-availability', async () => {
  const checks = await Promise.all(Object.keys(PROFILES).filter((target) => target !== 'u1').map(async (target) => {
    try {
      const bundle = await loadTargetBundle(target);
      return { target, available: true, slicer: bundle.slicer, model: bundle.profile.printer_model_id };
    } catch (err) {
      return { target, available: false, error: err.message };
    }
  }));
  return [
    { target: 'u1', available: true, slicer: 'Snapmaker Orca', model: 'Snapmaker U1' },
    ...checks
  ];
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

  if (targets.some((target) => !TARGETS[target]) ||
      new Set(targets).size !== targets.length) {
    return { ok: false, code: 'TARGET_INVALID', error: 'רשימת יעדים לא תקינה.' };
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

  const nozzleInfo = await inspectSourceNozzle(filePath);
  if (!nozzleInfo.ok) {
    const code = nozzleInfo.code || 'SOURCE_NOZZLE_INVALID';
    return {
      ok: false, folder: sourceDir, results: [],
      failures: targets.map((target) => ({
        ok: false, target, code,
        error: nozzleInfo.error || 'בדיקת קוטר הדיזה בקובץ המקור נכשלה.'
      })),
      error: nozzleInfo.error
    };
  }

  // Independent conversions start concurrently; one unsupported filament or
  // vendor profile must not discard successful outputs for other printers.
  const converted = await Promise.all(targets.map(async (target) => {
    const prefix = TARGETS[target];
    for (let attempt = 1; attempt <= 100; attempt += 1) {
      const outputName = attempt === 1
        ? `${prefix}-${baseName}`
        : `${prefix}-${path.parse(baseName).name}-${attempt}${path.extname(baseName)}`;
      const outputPath = path.join(sourceDir, outputName);
      try {
        await fs.access(outputPath);
        continue;
      } catch { /* use atomic no-overwrite publishing in the Go engine */ }
      const conversion = await convertWithVerifiedEngine(
        filePath, outputPath, target, nozzleInfo.source_nozzle
      );
      if (conversion.code === 'OUTPUT_EXISTS') continue;
      if (!conversion.ok) {
        return {
          ok: false, target, code: conversion.code || 'CONVERSION_FAILED',
          error: conversion.error || 'המרת הקובץ נכשלה.'
        };
      }
      return { ok: true, target, prefix, outputPath, outputName, report: conversion };
    }
    return { ok: false, target, code: 'OUTPUT_COLLISIONS', error: 'קיימים יותר מדי קבצים עם שם זהה.' };
  }));

  const results = converted.filter((entry) => entry.ok);
  const failures = converted.filter((entry) => !entry.ok);
  if (results.length > 0) lastOutputFolder = sourceDir;
  return {
    ok: results.length > 0,
    folder: sourceDir,
    sourceNozzle: nozzleInfo.source_nozzle,
    sourceNozzleAssumed: nozzleInfo.source_nozzle_assumed,
    results,
    failures,
    error: results.length === 0 ? 'כל ההמרות נכשלו. ראו פירוט לפי מדפסת.' : null
  };
});

ipcMain.handle('open-output-folder', async () => {
  if (!lastOutputFolder) {
    return { ok: false, error: 'עדיין אין תיקיית פלט לפתיחה.' };
  }

  const error = await shell.openPath(lastOutputFolder);
  return error ? { ok: false, error } : { ok: true };
});