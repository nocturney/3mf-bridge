const { app, BrowserWindow, ipcMain, dialog, shell } = require('electron');
const path = require('path');
const fs = require('fs/promises');

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

  const results = [];

  for (const target of targets) {
    const prefix = TARGETS[target];
    if (!prefix) continue;

    const outputName = `${prefix}-${baseName}`;
    const outputPath = await ensureUniquePath(sourceDir, outputName);

    await fs.copyFile(filePath, outputPath);

    results.push({
      target,
      prefix,
      outputPath,
      outputName: path.basename(outputPath)
    });
  }

  lastOutputFolder = sourceDir;

  return {
    ok: true,
    folder: sourceDir,
    results
  };
});

ipcMain.handle('open-output-folder', async () => {
  if (!lastOutputFolder) {
    return { ok: false, error: 'עדיין אין תיקיית פלט לפתיחה.' };
  }

  await shell.openPath(lastOutputFolder);
  return { ok: true };
});