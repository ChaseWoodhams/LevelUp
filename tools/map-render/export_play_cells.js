const fs=require('fs');
const D='C:/Users/Wolfie/Documents/HALO/LevelUp/data/cache/replays/halo_infinite/';
// Matches on the map being rendered: Streets unless MAP_MATCHES="id,id" says otherwise.
const MATCHES=(process.env.MAP_MATCHES||'0e97be38,36e80b83,e01d80a1,c85e424e').split(',');
// A cell counts once it holds MAP_MIN_POINTS samples (default 1, every cell). A decoder glitch
// is a lone sample kilometres away: on Recharge, cells with >= 20 samples keep 99.98 % of all
// positions and cut the extents from 2.5 km back to the 34 m arena.
const MIN=parseInt(process.env.MAP_MIN_POINTS||'1',10);
const count=new Map();
for(const f of MATCHES){
  const doc=JSON.parse(fs.readFileSync(D+f+'.json','utf8'));
  for(const t of doc.tracks) for(const p of t.points){
    // 1 m occupancy cells: the render only needs where players CAN be, not how often
    const k=Math.round(p.x)+','+Math.round(p.y);
    count.set(k,(count.get(k)||0)+1);
  }
}
const out=[...count].filter(([,n])=>n>=MIN).map(([k])=>k);
fs.writeFileSync(process.argv[2],out.join('\n'));
console.log('cells:',out.length);
