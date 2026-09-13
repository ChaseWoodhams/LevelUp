const fs=require('fs');
const D='C:/Users/Wolfie/Documents/HALO/LevelUp/data/cache/replays/halo_infinite/';
// Per 1 m cell: the HIGHEST player feet ever recorded there. That is the top of the
// reachable stack at that spot -- on a multi-level map it is the upper walkway, not the
// ground. Anything sitting well above it is a roof, whatever floor it happens to cover.
// Matches on the map being rendered: Streets unless MAP_MATCHES="id,id" says otherwise.
const MATCHES=(process.env.MAP_MATCHES||'0e97be38,36e80b83,e01d80a1,c85e424e').split(',');
// Same density filter as export_play_cells.js (MAP_MIN_POINTS), so a glitch sample cannot raise
// a cell's ceiling: on Recharge, unfiltered, one lone sample put the reachable floor at 244 m.
const MIN=parseInt(process.env.MAP_MIN_POINTS||'1',10);
const hi=new Map(), count=new Map();
for(const f of MATCHES){
  const doc=JSON.parse(fs.readFileSync(D+f+'.json','utf8'));
  for(const t of doc.tracks) for(const p of t.points){
    const k=Math.round(p.x)+','+Math.round(p.y);
    count.set(k,(count.get(k)||0)+1);
    if(typeof p.z!=='number' || p.z < -20) continue;   // -50 outliers are fall-through
    const c=hi.get(k);
    if(c===undefined || p.z>c) hi.set(k,p.z);
  }
}
const kept=[...hi].filter(([k])=>count.get(k)>=MIN);
const out=kept.map(([k,z])=>k+','+z.toFixed(2));
fs.writeFileSync(process.argv[2],out.join('\n'));
const zs=kept.map(([,z])=>z).sort((a,b)=>a-b);
console.log('cells:',out.length,'ceil-z min',zs[0].toFixed(2),'p50',zs[zs.length>>1].toFixed(2),'max',zs[zs.length-1].toFixed(2));
