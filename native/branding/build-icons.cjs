// Produce all platform icons from the APPROVED Direction 1 presentation crop.
// No tracing, redrawing or interpretation: resized pixels are from the photo.
'use strict';
const fs=require('node:fs');
const path=require('node:path');
const crypto=require('node:crypto');
const sharp=require(process.env.SHARP_MODULE||'sharp');
const native=path.resolve(__dirname,'..');
const repo=path.resolve(native,'..');
const input=path.join(__dirname,'direction1-reference','approved-icon.png');
const tile=fs.readFileSync(input);
const sizes=[16,24,32,48,64,128,256];
const chunks=[['icp4',16],['icp5',32],['icp6',64],['ic07',128],['ic08',256],['ic09',512],['ic10',1024]];
function roundMask(size){
 const round=Math.round(size*.23);
 return Buffer.from('<svg width="'+size+'" height="'+size+'" xmlns="http://www.w3.org/2000/svg">'+
   '<rect x="0" y="0" width="'+size+'" height="'+size+'" rx="'+round+'" ry="'+round+'" fill="#fff"/></svg>');
}
async function png(size){
 const src=await sharp(tile).resize(size,size,{fit:'fill',kernel:'lanczos3'}).ensureAlpha().png().toBuffer();
 // Cut only the four corner pixels outside the approved rounded tile.
 return sharp(src).composite([{input:roundMask(size),blend:'dest-in'}]).png({compressionLevel:9}).toBuffer();
}
function ico(images){
 const h=Buffer.alloc(6+images.length*16);
 h.writeUInt16LE(1,2);h.writeUInt16LE(images.length,4);
 let offset=h.length;
 for(let i=0;i<images.length;i++){
  const size=sizes[i],at=6+i*16;
  h.writeUInt8(size===256?0:size,at);h.writeUInt8(size===256?0:size,at+1);
  h.writeUInt16LE(1,at+4);h.writeUInt16LE(32,at+6);
  h.writeUInt32LE(images[i].length,at+8);h.writeUInt32LE(offset,at+12);
  offset+=images[i].length;
 }
 return Buffer.concat([h,...images]);
}
function icns(images){
 const entries=images.map(([tag,bytes])=>{
  const h=Buffer.alloc(8);h.write(tag,0,4,'ascii');h.writeUInt32BE(bytes.length+8,4);
  return Buffer.concat([h,bytes]);
 });
 const h=Buffer.alloc(8);h.write('icns',0,4,'ascii');
 h.writeUInt32BE(8+entries.reduce((n,x)=>n+x.length,0),4);
 return Buffer.concat([h,...entries]);
}
(async()=>{
 const win=await Promise.all(sizes.map(png));
 const mac=await Promise.all(chunks.map(async ([name,size])=>[name,await png(size)]));
 const files=[
  [path.join(native,'build','appicon.png'),await png(1024)],
  [path.join(native,'build','windows','icon.ico'),ico(win)],
  [path.join(native,'build','darwin','icon.icns'),icns(mac)],
  [path.join(native,'frontend','dist','brand-icon.png'),await png(128)]
 ];
 for(const size of [16,32,48,128]){
  files.push([path.join(repo,'browser-extension','shared','icon-'+size+'.png'),await png(size)]);
 }
 // Take only the wave portion from the approved banner crop. Keep its text in
 // accessible HTML so it is not duplicated by background-image rendering.
 const wave=fs.readFileSync(path.join(__dirname,'direction1-reference','approved-wave.png'));
 const waveMeta=await sharp(wave).metadata();
 const left=Math.floor(waveMeta.width*.46);
 const waveOnly=await sharp(wave).extract({left,top:0,width:waveMeta.width-left,height:waveMeta.height}).png().toBuffer();
 files.push([path.join(native,'frontend','dist','slicebridge-banner.png'),waveOnly]);
 for(const [file,bytes] of files){
  fs.mkdirSync(path.dirname(file),{recursive:true});fs.writeFileSync(file,bytes);
  console.log(path.relative(repo,file),bytes.length,crypto.createHash('sha256').update(bytes).digest('hex').slice(0,13));
 }
})().catch(e=>{console.error(e);process.exit(1);});
