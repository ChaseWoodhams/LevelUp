const fs=require('fs');
// Per-pixel choice of plan cut: the LOWEST cut that still clears the local reachable floor
// by HEADROOM. Low enough to slice the roof off, high enough to keep the floor and let a
// Spartan stand on it.
//
// TWO THINGS THIS GETS WRONG IF DONE NAIVELY:
//
// Dilation radius. Taking the highest floor within 6 m (the radius the off-arena cull uses)
// raised the bar over the whole map -- one walkway pulled every street beside it up to a
// 6 m cut, which is no cut at all. It has to be LOCAL: 2 m protects a walkway's own deck
// and its handrail without leaking that height sideways into the street below.
//
// Sampling. The field is on a 1 m grid and the render is 24 px/m, so picking the cut from
// the nearest cell staircases every boundary into 24 px blocks. The grid is smoothed and
// sampled BILINEARLY, which turns those blocks into a smooth contour.
// Must be the --cuts list given to render_slices.py: Streets unless MAP_CUTS="2.0,2.5,..." says otherwise.
const CUTS=(process.env.MAP_CUTS||'2.0,2.5,3.0,3.5,4.0,4.5,5.0,5.5,6.0,6.5,7.0,9.5').split(',').map(Number);
const HEADROOM=2.0, DILATE=1, FILL=10, BLUR=3;
// Frame of the map being rendered: Streets unless MAP_BOUNDS="minX,minY,maxX,maxY" says otherwise.
const [minX,minY,maxX,maxY]=(process.env.MAP_BOUNDS||'-24.32224,-23.018236,27.407486,29.866623').split(',').map(Number);
const r={minX,maxX,minY,maxY};
const W=parseInt(process.argv[4]||"1242",10), H=parseInt(process.argv[5]||"1269",10);

const raw=new Map();
for(const line of fs.readFileSync(process.argv[2],'utf8').trim().split(/\r?\n/)){
  const [a,b,z]=line.split(','); raw.set(a+','+b,parseFloat(z));
}
const dil=new Map();
for(const [k,z] of raw){
  const [a,b]=k.split(',').map(Number);
  for(let dx=-DILATE;dx<=DILATE;dx++) for(let dy=-DILATE;dy<=DILATE;dy++){
    if(dx*dx+dy*dy>DILATE*DILATE) continue;
    const kk=(a+dx)+','+(b+dy); const c=dil.get(kk);
    if(c===undefined||z>c) dil.set(kk,z);
  }
}
// dense grid over the frame, so sampling is array maths rather than map lookups
const GX0=Math.floor(r.minX)-FILL, GY0=Math.floor(r.minY)-FILL;
const GW=Math.ceil(r.maxX)-GX0+FILL+1, GH=Math.ceil(r.maxY)-GY0+FILL+1;
let g=new Float64Array(GW*GH).fill(NaN);
for(const [k,z] of dil){
  const [a,b]=k.split(',').map(Number);
  const ix=a-GX0, iy=b-GY0;
  if(ix>=0&&ix<GW&&iy>=0&&iy<GH) g[iy*GW+ix]=z;
}
// gaps take the nearest known floor (the LOWEST of the ring, so a covered alley beside a
// tall walkway is still cut open) rather than defaulting to the top cut
const g0=Float64Array.from(g);
for(let iy=0;iy<GH;iy++) for(let ix=0;ix<GW;ix++){
  if(!Number.isNaN(g0[iy*GW+ix])) continue;
  let best;
  for(let rad=1;rad<=FILL&&best===undefined;rad++){
    for(let dx=-rad;dx<=rad;dx++) for(let dy=-rad;dy<=rad;dy++){
      if(Math.max(Math.abs(dx),Math.abs(dy))!==rad) continue;
      const jx=ix+dx, jy=iy+dy;
      if(jx<0||jx>=GW||jy<0||jy>=GH) continue;
      const v=g0[jy*GW+jx];
      if(!Number.isNaN(v)&&(best===undefined||v<best)) best=v;
    }
  }
  if(best!==undefined) g[iy*GW+ix]=best;
}
// box blur, so the cut boundary is a smooth contour instead of a cell edge
for(let p=0;p<BLUR;p++){
  const s=Float64Array.from(g);
  for(let iy=0;iy<GH;iy++) for(let ix=0;ix<GW;ix++){
    let sum=0,n=0;
    for(let dy=-1;dy<=1;dy++) for(let dx=-1;dx<=1;dx++){
      const jx=ix+dx, jy=iy+dy;
      if(jx<0||jx>=GW||jy<0||jy>=GH) continue;
      const v=s[jy*GW+jx];
      if(!Number.isNaN(v)){ sum+=v; n++; }
    }
    if(n) g[iy*GW+ix]=sum/n;
  }
}
const sample=(wx,wy)=>{
  const fx=wx-GX0, fy=wy-GY0;
  const ix=Math.floor(fx), iy=Math.floor(fy);
  if(ix<0||ix+1>=GW||iy<0||iy+1>=GH) return NaN;
  const tx=fx-ix, ty=fy-iy;
  const v=(a,b)=>g[b*GW+a];
  const a=v(ix,iy), b=v(ix+1,iy), c=v(ix,iy+1), d=v(ix+1,iy+1);
  if([a,b,c,d].some(Number.isNaN)) return NaN;
  return a*(1-tx)*(1-ty)+b*tx*(1-ty)+c*(1-tx)*ty+d*tx*ty;
};
const pick=f=>{
  const want=f+HEADROOM;
  for(let i=0;i<CUTS.length;i++) if(CUTS[i]>=want) return i;
  return CUTS.length-1;
};
const sx=(r.maxX-r.minX)/W, sy=(r.maxY-r.minY)/H;
const rows=[]; const used=new Array(CUTS.length).fill(0); let unknown=0;
for(let py=0;py<H;py++){
  const row=new Array(W); const wy=r.maxY-(py+0.5)*sy;
  for(let px=0;px<W;px++){
    const f=sample(r.minX+(px+0.5)*sx, wy);
    let i;
    if(Number.isNaN(f)){ i=CUTS.length-1; unknown++; } else i=pick(f);
    used[i]++; row[px]=i.toString(36);
  }
  rows.push(row.join(''));
}
fs.writeFileSync(process.argv[3], W+' '+H+'\n'+rows.join('\n'));
console.log('slice map '+W+'x'+H+'  unresolved px='+unknown+' ('+(100*unknown/(W*H)).toFixed(1)+'%)');
CUTS.forEach((c,i)=>{ if(used[i]) console.log('  cut '+c.toFixed(1)+'m  px='+used[i]+' ('+(100*used[i]/(W*H)).toFixed(1)+'%)'); });
