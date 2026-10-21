const { contextBridge, ipcRenderer, webUtils } = require('electron');

contextBridge.exposeInMainWorld('bridgeApi', {
  pick3mfFile: () => ipcRenderer.invoke('pick-3mf-file'),
  getDroppedFilePath: (file) => webUtils.getPathForFile(file),
  inspectDroppedFile: (filePath) => ipcRenderer.invoke('inspect-dropped-file', filePath),
  getTargetAvailability: () => ipcRenderer.invoke('get-target-availability'),
  convert3mf: (payload) => ipcRenderer.invoke('convert-3mf', payload),
  openOutputFolder: () => ipcRenderer.invoke('open-output-folder')
});