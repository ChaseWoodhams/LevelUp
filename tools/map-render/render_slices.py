# Render the map as a STACK OF PLAN CUTS, one per height in --cuts.
#
# WHY A STACK. A roof hides the street under it, and no per-object cull can fix that: a
# building arrives as ONE mesh spanning floor to roof, so its bbox starts at ground level
# and every "is this a roof" test says no. The only thing that separates roof from floor is
# the camera near plane, which cuts per PIXEL rather than per object.
#
# One cut height cannot serve the whole map either -- Streets stacks walkways over streets,
# and the reachable floor ranges 0 to 5 m. So render every candidate cut here and let the
# compositor pick, per pixel, the one just above the local reachable floor. Colour is
# assigned from each object's own base height and is identical in every slice, so the
# composite is seamless in colour and differs only in what got clipped away.
import bpy, sys
from collections import defaultdict
from mathutils import Vector
def arg(n,d): return sys.argv[sys.argv.index(n)+1] if n in sys.argv else d

MIN_X, MAX_X = -24.32224, 27.407486
MIN_Y, MAX_Y = -23.018236, 29.866623
PPM      = float(arg('--ppm','24'))
PLAY_TOP = float(arg('--playtop','9.0'))
RAMP_LO  = float(arg('--ramplo','-1.0'))
RAMP_HI  = float(arg('--ramphi','6.0'))
REACH    = float(arg('--reach','6.0'))
CELLS    = arg('--cells','')
OUTDIR   = arg('--outdir','')
PALETTE  = arg('--palette','slate')
CUTS     = [float(c) for c in arg('--cuts','1.2,1.9,2.6,3.3,4.0,4.7,5.4,6.1,7.0,9.5').split(',')]
CAM_Z    = 600.0

def lin(c):
    return c/12.92 if c <= 0.04045 else ((c+0.055)/1.055)**2.4
def hexc(s):
    return tuple(int(s[i:i+2],16)/255.0 for i in (0,2,4))

PALETTES = {
  'slate':     (hexc('0d1117'), hexc('c3cddc'), hexc('05070c')),
  'blueprint': (hexc('79828f'), hexc('f6f8fa'), hexc('161b25')),
  'amber':     (hexc('17110c'), hexc('e6cd9c'), hexc('080604')),
}

w,h = MAX_X-MIN_X, MAX_Y-MIN_Y
cx,cy = (MIN_X+MAX_X)/2, (MIN_Y+MAX_Y)/2
scene = bpy.context.scene
def drop(o): bpy.data.objects.remove(o, do_unlink=True)
start=len([o for o in bpy.data.objects if o.type=='MESH'])

# masters: the importer's source-mesh pile, buried at the origin
n_master=0
mc=bpy.data.collections.get('Master Geometries')
if mc:
    for o in list(mc.objects): drop(o); n_master+=1

# duplicates: same mesh data at the same world matrix, up to 6 deep
seen=defaultdict(list)
for o in [o for o in bpy.data.objects if o.type=='MESH']:
    seen[(o.data.name, tuple(round(v,4) for r in o.matrix_world for v in r))].append(o)
n_dup=0
for k,v in seen.items():
    for o in v[1:]: drop(o); n_dup+=1

cells=set()
if CELLS:
    for line in open(CELLS):
        line=line.strip()
        if line:
            a,b=line.split(','); cells.add((int(a),int(b)))
r=int(REACH); near=set()
for (a,b) in cells:
    for dx in range(-r,r+1):
        for dy in range(-r,r+1):
            if dx*dx+dy*dy <= r*r: near.add((a+dx,b+dy))

def touches(pts):
    if not near: return True
    x0=int(min(p.x for p in pts)); x1=int(max(p.x for p in pts))+1
    y0=int(min(p.y for p in pts)); y1=int(max(p.y for p in pts))+1
    if (x1-x0)*(y1-y0) > 40000: return True
    for a in range(x0,x1+1):
        for b in range(y0,y1+1):
            if (a,b) in near: return True
    return False

base=[]; n_high=0; n_far=0
for o in [o for o in bpy.data.objects if o.type=='MESH']:
    pts=[o.matrix_world @ Vector(v) for v in o.bound_box]
    zmin=min(p.z for p in pts)
    if zmin >= PLAY_TOP: drop(o); n_high+=1; continue
    if not touches(pts): drop(o); n_far+=1; continue
    base.append((o, zmin))
print('CULL start=%d masters=%d dups=%d abovePlay=%d offArena=%d kept=%d' % (
    start, n_master, n_dup, n_high, n_far, len(base)), flush=True)

lo_c, hi_c, out_c = PALETTES[PALETTE]
span=max(RAMP_HI-RAMP_LO,1e-6)
for o,z in base:
    t=min(max((z-RAMP_LO)/span,0.0),1.0)
    o.color=tuple(lin(lo_c[i]+(hi_c[i]-lo_c[i])*t) for i in (0,1,2))+(1.0,)

for o in [o for o in bpy.data.objects if o.type=='CAMERA']: drop(o)
cd=bpy.data.cameras.new("TopDown"); cd.type='ORTHO'; cd.ortho_scale=max(w,h)
cd.clip_end=CAM_Z+200.0
cam=bpy.data.objects.new("TopDown",cd); scene.collection.objects.link(cam)
cam.location=(cx,cy,CAM_Z); cam.rotation_euler=(0,0,0); scene.camera=cam

scene.render.resolution_x=round(w*PPM); scene.render.resolution_y=round(h*PPM)
scene.render.resolution_percentage=100
scene.render.film_transparent=True
scene.render.engine='BLENDER_WORKBENCH'
scene.display.render_aa='16'
scene.view_settings.view_transform='Standard'
scene.view_settings.look='None'
scene.render.image_settings.file_format='PNG'
scene.render.image_settings.color_mode='RGBA'

sh=scene.display.shading
sh.light='FLAT'; sh.color_type='OBJECT'; sh.show_shadows=False
sh.show_cavity=True; sh.cavity_type='BOTH'
sh.curvature_ridge_factor=0.6; sh.curvature_valley_factor=1.2
sh.cavity_ridge_factor=0.5; sh.cavity_valley_factor=1.4
sh.show_object_outline=True
sh.object_outline_color=tuple(lin(c) for c in out_c)

for i,cut in enumerate(CUTS):
    cd.clip_start=CAM_Z-cut
    scene.render.filepath='%s/slice_%02d.png' % (OUTDIR, i)
    print('SLICE %02d cut=%.1fm res=%dx%d' % (i,cut,scene.render.resolution_x,scene.render.resolution_y), flush=True)
    bpy.ops.render.render(write_still=True)
print('SLICES_DONE %s' % ','.join('%.1f'%c for c in CUTS), flush=True)
