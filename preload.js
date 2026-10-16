const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('bridgeApi', {
  pick3mfFile: () => ipcRenderer.invoke('pick-3mf-file'),
  inspectDroppedFile: (filePath) => ipcRenderer.invoke('inspect-dropped-file', filePath),
  convert3mf: (payload) => ipcRenderer.invoke('convert-3mf', payload),
  openOutputFolder: () => ipcRenderer.invoke('open-output-folder')
});