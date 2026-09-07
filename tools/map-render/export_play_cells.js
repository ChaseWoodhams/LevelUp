const fs=require('fs');
const D='C:/Users/Wolfie/Documents/HALO/LevelUp/data/cache/replays/halo_infinite/';
const out=new Set();
for(const f of ['0e97be38','36e80b83','e01d80a1','c85e424e']){
  const doc=JSON.parse(fs.readFileSync(D+f+'.json','utf8'));
  for(const t of doc.tracks) for(const p of t.points){
    // 1 m occupancy cells: the render only needs where players CAN be, not how often
    out.add(Math.round(p.x)+','+Math.round(p.y));
  }
}
fs.writeFileSync(process.argv[2],[...out].join('\n'));
console.log('cells:',out.size);
