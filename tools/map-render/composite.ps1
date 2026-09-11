# Composite a stack of plan-cut renders into one map, choosing the cut PER PIXEL.
#
# Reads the slice map written by slice_map.js -- one base-36 digit per pixel, naming which
# slice that pixel takes -- and copies that slice's RGBA straight across. No blending: two
# slices of the same scene differ only in what the near plane removed, so a blend would
# ghost a roof over the floor it was supposed to reveal.
#
# A pixel whose chosen slice is EMPTY stays empty. Stepping up the stack to find geometry
# was tried and is wrong: a roofed pixel is empty precisely because the cut worked, so the
# fallback pulls every roof straight back in.
param(
  [Parameter(Mandatory=$true)][string]$SliceDir,
  [Parameter(Mandatory=$true)][string]$SliceMap,
  [Parameter(Mandatory=$true)][string]$Out,
  [string]$MaskOut = ''
)
$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Drawing

$map = [System.IO.File]::ReadAllLines($SliceMap)
$hdr = $map[0].Split(' '); $W = [int]$hdr[0]; $H = [int]$hdr[1]
$files = Get-ChildItem -Path $SliceDir -Filter 'slice_*.png' | Sort-Object Name
if ($files.Count -eq 0) { throw "no slice_*.png under $SliceDir" }

$bufs = @(); $stride = 0
foreach ($f in $files) {
  $bmp = New-Object System.Drawing.Bitmap($f.FullName)
  if ($bmp.Width -ne $W -or $bmp.Height -ne $H) {
    throw "$($f.Name) is $($bmp.Width)x$($bmp.Height) but the slice map is ${W}x${H}"
  }
  $rect = New-Object System.Drawing.Rectangle(0, 0, $W, $H)
  $d = $bmp.LockBits($rect, [System.Drawing.Imaging.ImageLockMode]::ReadOnly,
                     [System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
  $stride = $d.Stride
  $b = New-Object byte[] ($d.Stride * $H)
  [System.Runtime.InteropServices.Marshal]::Copy($d.Scan0, $b, 0, $b.Length)
  $bmp.UnlockBits($d); $bmp.Dispose()
  $bufs += , $b
}

# NOT `$out`: PowerShell variable names ignore case, so `$out` IS the `[string]$Out` parameter,
# and assigning a byte array to it silently converts the bytes to one string. The first index
# into it then fails with "Unable to index into an object of type System.String".
$pixels = New-Object byte[] ($stride * $H)
for ($y = 0; $y -lt $H; $y++) {
  $row = $map[$y + 1]; $o = $y * $stride
  for ($x = 0; $x -lt $W; $x++) {
    $c = $row[$x]
    $i = if ($c -ge [char]'0' -and $c -le [char]'9') { [int]$c - 48 } else { [int]$c - 87 }
    if ($i -ge $bufs.Count) { $i = $bufs.Count - 1 }
    $src = $bufs[$i]; $k = $o + $x * 4
    $pixels[$k] = $src[$k]; $pixels[$k+1] = $src[$k+1]; $pixels[$k+2] = $src[$k+2]; $pixels[$k+3] = $src[$k+3]
  }
}

$res = New-Object System.Drawing.Bitmap($W, $H, [System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
$rect = New-Object System.Drawing.Rectangle(0, 0, $W, $H)
$d = $res.LockBits($rect, [System.Drawing.Imaging.ImageLockMode]::WriteOnly,
                   [System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
[System.Runtime.InteropServices.Marshal]::Copy($pixels, 0, $d.Scan0, $pixels.Length)
$res.UnlockBits($d)
$res.Save($Out, [System.Drawing.Imaging.ImageFormat]::Png)
$res.Dispose()
Write-Output "$Out  ${W}x${H}  from $($bufs.Count) slices"

# The coverage mask check_coverage.js reads: one character per pixel, opaque or not.
if ($MaskOut -ne '') {
  $sb = New-Object System.Text.StringBuilder
  [void]$sb.AppendLine("$W $H")
  for ($y = 0; $y -lt $H; $y++) {
    $rowc = New-Object char[] $W; $o = $y * $stride
    for ($x = 0; $x -lt $W; $x++) {
      $rowc[$x] = if ($pixels[$o + $x * 4 + 3] -gt 8) { '#' } else { '.' }
    }
    [void]$sb.AppendLine(-join $rowc)
  }
  [System.IO.File]::WriteAllText($MaskOut, $sb.ToString())
  Write-Output "$MaskOut"
}
