const {contextBridge}=require('electron');
const printerList=['u1','c5','cc2','qidi'];
const names={u1:'Snapmaker U1',c5:'Flashforge Creator 5',cc2:'ELEGOO CC2',qidi:'QIDI Plus5'};
let saved={version:1,printers:Object.fromEntries(printerList.map(t=>[t,{nozzle:'auto',slicer:'auto',filaments:{}}])),window:{width:1100,height:750}};
let pickCount=0;
const mockFiles=Array.from({length:32},(_,i)=>({path:'C:\\models\\project-'+String(i+1).padStart(2,'0')+'.3mf',name:'project-'+String(i+1).padStart(2,'0')+'.3mf',folder:'C:\\models',size:1048576,sizeLabel:'1.0 MB'}));
const app={
  GetSettings:async()=>saved,
  SavePrinterSettings:async(target,prefs)=>{saved.printers[target]=prefs;return saved;},
  Availability:async()=>printerList.map(target=>({target,name:names[target],available:true,slicer:'Installed slicer'})),
  GetVersion:async()=>({version:'0.4.0-beta.3',platform:'windows-amd64'}),
  SelectFiles:async()=>{pickCount++;return pickCount===1?mockFiles:[{path:'C:\\models\\new-project.3mf',name:'new-project.3mf',folder:'C:\\models',size:500,sizeLabel:'500 B'}];},
  SelectFolder:async()=>mockFiles,
  DetectMakerWorldOrigins:async(paths)=>Object.fromEntries(paths.filter(p=>p.includes('project-01.3mf')).map(p=>[p,{kind:'makerworld',confidence:'high',originUrl:'https://makerworld.com/en/models/123',evidence:'download_referrer'}])),
  PrinterOptions:async(target,nozzle,slicer,material)=>({
    nozzles:['0.2','0.4','0.6','0.8'],
    slicers:[{id:target==='qidi'?'orca-slicer':'system',name:'Installed vendor slicer',installed:true}],
    filaments:['Generic '+material+' @'+names[target]+' '+nozzle+' nozzle']
  }),
  ConvertBatch:async({filePaths,targets})=>({
    filesDone:filePaths.length,successes:filePaths.length*targets.length,failures:0,
    files:filePaths.map(path=>({
      source:{path,name:path.split('\\').pop(),nozzle:'0.4',plates:2,objects:3,instances:3,
        filaments:[{id:1,type:'PLA',color:'#B81C28'},{id:2,type:'PETG',color:'#1B92AE'}]},
      results:targets.map(target=>({
        target,ok:true,outputName:target.toUpperCase()+'-'+path.split('\\').pop(),
        outputPath:'C:\\models\\'+target.toUpperCase()+'-'+path.split('\\').pop(),
        report:{target_nozzle:'0.4',source_nozzle:'0.4',source_filament_slots:2,
          geometry_triangles:26700,removed_slice_cache_entries:8,
          preserved_print_keys:['layer_height','support_material','infill_pattern'],
          native_filament_overrides:{}}
      })),
      failures:[]
    }))
  }),
  OpenInSlicer:async()=>{},
  OpenOutputFolder:async()=>{},
  RememberWindowSize:async()=>{},
  CheckForUpdates:async()=>({status:'up_to_date'}),
  OpenUpdatePage:async()=>{},
  BrowserLinkStatus:async()=>({platform:'windows',registered:false}),
  EnableBrowserLink:async()=>{},
  OpenBrowserSetup:async()=>{}
};
contextBridge.exposeInMainWorld('go',{main:{App:app}});
contextBridge.exposeInMainWorld('runtime',{OnFileDrop:()=>{},EventsOn:()=>{}});
