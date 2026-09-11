const fs=require('fs');
const lines=fs.readFileSync(process.argv[2],'utf8').split(/\r?\n/);
const [W,H]=lines[0].split(' ').map(Number);
const mask=lines.slice(1,1+H);
const D='C:/Users/Wolfie/Documents/HALO/LevelUp/data/cache/replays/halo_infinite/';
// Frame and matches of the map being checked: Streets unless MAP_BOUNDS="minX,minY,maxX,maxY"
// and MAP_MATCHES="id,id" say otherwise.
const [minX,minY,maxX,maxY]=(process.env.MAP_BOUNDS||'-24.32224,-23.018236,27.407486,29.866623').split(',').map(Number);
const r={minX,maxX,minY,maxY};
const MATCHES=(process.env.MAP_MATCHES||'0e97be38,36e80b83,e01d80a1,c85e424e').split(',');
const sx=W/(r.maxX-r.minX), sy=H/(r.maxY-r.minY);
let on=0,off=0;
for(const f of MATCHES){
  const doc=JSON.parse(fs.readFileSync(D+f+'.json','utf8'));
  for(const t of doc.tracks) for(const p of t.points){
    const px=Math.round((p.x-r.minX)*sx), py=Math.round((r.maxY-p.y)*sy);
    if(px<0||py<0||px>=W||py>=H){off++;continue;}
    const row=mask[py];
    (row && row[px]==='#') ? on++ : off++;
  }
}
const tot=on+off;
console.log('player positions:', tot);
console.log('  ON rendered geometry :', on, '('+(100*on/tot).toFixed(2)+'%)');
console.log('  off / empty          :', off, '('+(100*off/tot).toFixed(2)+'%)');
console.log('\ncalibration used: NONE — the frame is the game\'s own sbsp AABB');
