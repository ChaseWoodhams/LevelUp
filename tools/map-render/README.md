# map-render — top-down map images for the study viewer

Produces the PNGs under `apps/study/public/maps/`, straight from the game's own geometry.

**The point of this pipeline is that its output needs no calibration.** The camera is framed
on the map's sbsp AABB, which is the same coordinate frame the replay documents live in, so
a world point maps to a pixel by linear interpolation of the rectangle recorded in
`apps/study/src/features/viewer/mapImages.config.ts`. Nothing is fitted by hand, so nothing
can be fitted wrong. The predecessor asset was callout art registered against two
Forge-extracted flag positions — a two-point fit, which cannot tell a rotation from its
mirror, and had the rotation backwards until a player said which room they spawned in.

Verified for Streets: **175,484 player positions across four archived matches, 100.00% of
them landing on drawn geometry.**

## Language exception

`render_slices.py` is Python. This is not a breach of the repo's Go/TS-only rule so much as
the one place it cannot apply: Blender embeds CPython and exposes `bpy` to nothing else.
`composite.ps1` is PowerShell for a narrower reason — it needs a PNG codec, and
`System.Drawing` is the only one available here without adding a dependency to a tool that
runs a handful of times per map. Neither ships with the app; this whole directory is offline
asset generation.

## Prerequisites

- Blender 5.x with the [`ekur`](https://github.com/Surasia/ekur) addon enabled, and the map
  imported to a `.blend` (kept outside this repo — the Streets scene is 181 MB).
- Node (for the two exporters and the coverage check), PowerShell (for the compositor).

## Pipeline

```bash
# 1. Where players can be, and how high they get there, from archived replay documents.
node export_play_cells.js   playcells.txt    # 1 m occupancy cells: "x,y"
node export_play_ceiling.js playceil.txt     # highest feet per cell: "x,y,maxz"

# 2. Render one image per candidate plan cut. One Blender load, N renders out.
blender -b streets_scene.blend --python render_slices.py -- \
  --outdir slices --ppm 32 --cells playcells.txt --palette slate \
  --cuts 2.0,2.5,3.0,3.5,4.0,4.5,5.0,5.5,6.0,6.5,7.0,9.5

# 3. Choose a cut per pixel, from the local reachable floor.
node slice_map.js playceil.txt slicemap.txt 1655 1692

# 4. Composite, and write a mask to check against (pwsh or Windows PowerShell 5.1's powershell.exe).
pwsh -File composite.ps1 -SliceDir slices -SliceMap slicemap.txt \
     -Out sgh_streets.png -MaskOut mask.txt

# 5. The check that decides whether the asset is usable at all.
node check_coverage.js mask.txt              # must be 100.00%
```

Both exporters and the coverage check read replay documents from
`data/cache/replays/halo_infinite/`. The commands above are Streets, which is every tool's
default. Another map takes its frame, matches and cut stack from the environment and flags,
never from edits to the scripts:

```bash
export MAP_BOUNDS=minX,minY,maxX,maxY   # the map's entry in map_quant_bounds.json
export MAP_MATCHES=id,id,...            # archived replays on that map (short ids)
export MAP_CUTS=3.5,4.5,...             # the same list as --cuts below
blender -b <map>_scene.blend --python render_slices.py -- --bounds "$MAP_BOUNDS" \
  --cuts "$MAP_CUTS" --playtop <above the highest floor> --ramphi <highest floor> ...
node slice_map.js playceil.txt slicemap.txt <round(width*ppm)> <round(height*ppm)>
```

`--playtop` culls any object that starts above it (default 9 m, fine for Streets): on a
multi-level map it must clear the highest reachable floor, which `export_play_ceiling.js`
prints as `max`, or whole upper floors are deleted. The scene itself comes from ekur's
"Import Level" operator on `<data folder>/levels/<map>.json`, saved to a `.blend` outside
the repository.

## Why a stack of cuts and not one image

A plain top-down render hides every street under its canopy. Three fixes were tried:

1. **Cull roof objects** — measured, and it does almost nothing. A building arrives from ekur
   as ONE mesh spanning floor to roof, so its bounding box starts at ground level and every
   "is this a roof" test says no. There is no object to delete.
2. **One low cut plane** — the camera near plane does cut per pixel, but Streets stacks
   walkways over streets and its reachable floors span 0 to 5 m. Any single height either
   leaves roofs on or decapitates the upper level.
3. **A stack, chosen per pixel** — what is here. Each pixel takes the lowest cut that still
   clears the local reachable floor by `HEADROOM`, which is low enough to slice the roof off
   and high enough to keep the floor and let a Spartan stand on it.

Two tuning mistakes worth not repeating, both recorded in `slice_map.js`:

- The reachable-height field must be **local**. Dilating it by the 6 m used for the off-arena
  cull let one walkway pull every street beside it up to a 6 m cut, and 96% of pixels landed
  on the top three cuts — no cut at all.
- An empty pixel must **stay empty**. Stepping up the stack until geometry appears puts every
  roof straight back, because a roofed pixel is empty precisely because the cut worked.

The known artefact: pale wedges where one large building spans two cut regions and is sliced
in one but not the other. Closing that means cutting geometry per object rather than per
pixel, which is a much larger job.

## What gets culled, and what does not

`render_slices.py` removes four kinds of non-map before rendering — see its header for the
measurements behind each. Briefly: exact duplicates (same mesh at the same transform, up to
six deep, 23.7% of objects), ekur's `Master Geometries` source-mesh pile (buried at the
origin, 49 changed pixels out of 1.58M), geometry above the play volume, and geometry whose
footprint never comes near a walked cell.

Flat sub-2cm planes are **not** culled, though they look like decals: Halo builds real walls
and floors out of thin brushes, and dropping them tore the faces off buildings.
