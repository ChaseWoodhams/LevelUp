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
const cx=(r.minX+r.maxX)/2, cy=(r.minY+r.maxY)/2;
const covered=(x,y)=>{
  const px=Math.round((x-r.minX)*sx), py=Math.round((r.maxY-y)*sy);
  if(px<0||py<0||px>=W||py>=H) return false;
  const row=mask[py];
  return !!row && row[px]==='#';
};
// THE CONTROL THAT KEEPS 100 % HONEST. Coverage alone cannot fail on a frame that is mostly
// drawn: Recharge's first render kept a background surface that filled 85 % of the frame, and
// its player positions scored 100 % — mirrored, rotated and shifted positions scored 100 % too.
// A usable image puts players on drawn floor AND loses them when the positions are moved.
const moves={
  'real        ':(x,y)=>[x,y],
  'mirror X    ':(x,y)=>[2*cx-x,y],
  'mirror Y    ':(x,y)=>[x,2*cy-y],
  'rotate 180  ':(x,y)=>[2*cx-x,2*cy-y],
  'shift 3 m   ':(x,y)=>[x+3,y+3],
  'shift 6 m X ':(x,y)=>[x+6,y],
};
const on=Object.fromEntries(Object.keys(moves).map(k=>[k,0]));
let tot=0;
for(const f of MATCHES){
  const doc=JSON.parse(fs.readFileSync(D+f+'.json','utf8'));
  for(const t of doc.tracks) for(const p of t.points){
    tot++;
    for(const [k,mv] of Object.entries(moves)){ const [x,y]=mv(p.x,p.y); if(covered(x,y)) on[k]++; }
  }
}
console.log('player positions:', tot);
for(const [k,n] of Object.entries(on)) console.log('  '+k+': '+n+' ('+(100*n/tot).toFixed(2)+'%)');
console.log('\nframe: x '+r.minX+'..'+r.maxX+', y '+r.minY+'..'+r.maxY+
  ' (world coordinates, from MAP_BOUNDS; nothing fitted). The real row must be ~100 % and every moved row clearly lower.');
