// Rebuild deterministic SliceBridge icons from the vector master.
// Run: npm i --no-save sharp && node native/branding/build-icons.cjs
const fs = require('node:fs');
const path = require('node:path');
const crypto = require('node:crypto');

const base = path.resolve(__dirname, '..');
const sharp = require(process.env.SHARP_MODULE || 'sharp');
const master = path.join(__dirname, 'slicebridge-symbol.svg');
const bytes = fs.readFileSync(master);
const sizes = [16, 24, 32, 48, 64, 128, 256];
const iconsetSizes = [
  ['icp4',16], ['icp5',32], ['icp6',64], ['ic07',128],
  ['ic08',256], ['ic09',512], ['ic10',1024]
];
async function png(size) {
  return sharp(bytes, { density: 256 }).resize(size, size, {
    fit:'contain',kernel:'lanczos3'
  }).png({compressionLevel:9}).toBuffer();
}
function ico(pngs) {
  const header=Buffer.alloc(6 + pngs.length*16);
  header.writeUInt16LE(0,0);header.writeUInt16LE(1,2);
  header.writeUInt16LE(pngs.length,4);
  let offset=header.length;
  for (let i=0;i<pngs.length;i++) {
    const size=sizes[i], entry=6+i*16;
    header.writeUInt8(size===256?0:size,entry);
    header.writeUInt8(size===256?0:size,entry+1);
    header.writeUInt8(0,entry+2);header.writeUInt8(0,entry+3);
    header.writeUInt16LE(1,entry+4);
    header.writeUInt16LE(32,entry+6);
    header.writeUInt32LE(pngs[i].length,entry+8);
    header.writeUInt32LE(offset,entry+12);
    offset+=pngs[i].length;
  }
  return Buffer.concat([header,...pngs]);
}
function icns(entries) {
  const blocks=entries.map(([tag, bytes])=>{
    const header=Buffer.alloc(8);
    header.write(tag,0,4,'ascii');
    header.writeUInt32BE(bytes.length+8,4);
    return Buffer.concat([header, bytes]);
  });
  const main=Buffer.alloc(8);
  main.write('icns',0,4,'ascii');
  main.writeUInt32BE(8+blocks.reduce((sum,b)=>sum+b.length,0),4);
  return Buffer.concat([main,...blocks]);
}
(async ()=>{
  const commonSizes = await Promise.all(sizes.map(png));
  const iconset=await Promise.all(iconsetSizes.map(async ([t,size])=>[t,await png(size)]));
  const apppng=iconset.find(([_,b])=>b)[1]; // overwritten below with explicit 1024
  const highRes=await png(1024);
  const files = [
    [path.join(base,'build','appicon.png'), highRes],
    [path.join(base,'build','windows','icon.ico'), ico(commonSizes)],
    [path.join(base,'build','darwin','icon.icns'), icns(iconset)],
    [path.join(__dirname,'slicebridge-preview.png'),await png(512)],
    [path.join(base,'frontend','dist','brand-icon.svg'),bytes]
  ];
  for(const [name,data] of files){
    fs.mkdirSync(path.dirname(name),{recursive:true});
    fs.writeFileSync(name,data);
    console.log(path.relative(base,name),data.length,crypto.createHash('sha256').update(data).digest('hex').slice(0,14));
  }
})().catch(e=>{console.error(e);process.exit(1);});
