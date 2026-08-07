<#
.SYNOPSIS
    生成 DeepStudent 多分辨率应用图标 (cmd/installer/icon.ico)

.DESCRIPTION
    使用 System.Drawing 程序化绘制一个深蓝渐变底 + 白色 "D" 字样的图标，
    然后合成 16/32/48/64/128/256 共 6 个分辨率的 .ico 文件。

    原理：每个分辨率渲染为 PNG 字节 → 写入 ICO 容器。
    适用环境：Windows PowerShell 5.1+ 或 PowerShell 7+。

.PARAMETER OutputPath
    输出 .ico 路径，默认相对脚本位置：cmd/installer/icon.ico

.EXAMPLE
    PS> .\build-icon.ps1
    PS> .\build-icon.ps1 -OutputPath C:\temp\deepstudent.ico
#>
[CmdletBinding()]
param(
    [string]$OutputPath
)

$ErrorActionPreference = 'Stop'

if (-not $OutputPath) {
    $scriptDir = $null
    if ($PSScriptRoot) { $scriptDir = $PSScriptRoot }
    elseif ($PSCommandPath) { $scriptDir = Split-Path -Parent $PSCommandPath }
    elseif ($MyInvocation.MyCommand.Path) { $scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path }
    if (-not $scriptDir) { $scriptDir = (Get-Location).Path }
    $OutputPath = Join-Path $scriptDir 'icon.ico'
}

# 需要的 GDI+ 程序集
Add-Type -AssemblyName System.Drawing

# ---------- 调色板 ----------
$BgTop    = [System.Drawing.Color]::FromArgb(255,  18,  44,  89)   # 深蓝
$BgBottom = [System.Drawing.Color]::FromArgb(255,  37,  99, 184)   # 略亮
$Accent   = [System.Drawing.Color]::FromArgb(255,  90, 169, 255)   # 高光
$Letter   = [System.Drawing.Color]::White

# ---------- 调试图形：画一个 256x256 母版 ----------
function New-MasterBitmap {
    param([int]$Size)

    $bmp = New-Object System.Drawing.Bitmap $Size, $Size
    $g    = [System.Drawing.Graphics]::FromImage($bmp)
    $g.SmoothingMode     = 'AntiAlias'
    $g.InterpolationMode = 'HighQualityBicubic'
    $g.PixelOffsetMode   = 'HighQuality'
    $g.TextRenderingHint = 'AntiAliasGridFit'

    # ---- 1. 圆角矩形渐变背景 ----
    $rect    = New-Object System.Drawing.Rectangle 0, 0, $Size, $Size
    $corner  = [int]($Size * 0.18)
    $path    = New-Object System.Drawing.Drawing2D.GraphicsPath
    $path.AddArc($rect.X,                 $rect.Y,                  $corner, $corner, 180, 90)
    $path.AddArc($rect.Right - $corner,   $rect.Y,                  $corner, $corner, 270, 90)
    $path.AddArc($rect.Right - $corner,   $rect.Bottom - $corner,   $corner, $corner,   0, 90)
    $path.AddArc($rect.X,                 $rect.Bottom - $corner,   $corner, $corner,  90, 90)
    $path.CloseFigure()

    $lg = New-Object System.Drawing.Drawing2D.LinearGradientBrush(
        (New-Object System.Drawing.PointF 0, 0),
        (New-Object System.Drawing.PointF $Size, $Size),
        $BgTop, $BgBottom)
    $g.FillPath($lg, $path)

    # ---- 2. 顶部高光带 ----
    $hlRect = New-Object System.Drawing.Rectangle 0, 0, $Size, ([int]($Size * 0.45))
    $hlBrush = New-Object System.Drawing.Drawing2D.LinearGradientBrush(
        $hlRect,
        ([System.Drawing.Color]::FromArgb(60,  $Accent)),
        ([System.Drawing.Color]::FromArgb(0,   $Accent)),
        90)
    $g.FillRectangle($hlBrush, $hlRect)

    # ---- 3. 居中 "D" 字 ----
    $fontSize = [int]($Size * 0.62)
    $font     = New-Object System.Drawing.Font 'Segoe UI', $fontSize, ([System.Drawing.FontStyle]::Bold), ([System.Drawing.GraphicsUnit]::Pixel)
    $sf       = New-Object System.Drawing.StringFormat
    $sf.Alignment     = 'Center'
    $sf.LineAlignment = 'Center'

    # 文字阴影
    $shadow = New-Object System.Drawing.SolidBrush ([System.Drawing.Color]::FromArgb(120, 0, 0, 0))
    $shadowRect = New-Object System.Drawing.RectangleF (
        ([single]($Size * 0.05)),
        ([single]($Size * 0.10)),
        ([single]$Size),
        ([single]$Size))
    $g.DrawString('D', $font, $shadow, $shadowRect, $sf)

    # 文字本体
    $brush = New-Object System.Drawing.SolidBrush $Letter
    $textRect = New-Object System.Drawing.RectangleF (
        ([single](-$Size * 0.02)),
        ([single](-$Size * 0.02)),
        ([single]$Size),
        ([single]$Size))
    $g.DrawString('D', $font, $brush, $textRect, $sf)

    # 释放资源
    $brush.Dispose(); $shadow.Dispose()
    $font.Dispose()
    $lg.Dispose(); $hlBrush.Dispose()
    $path.Dispose()
    $g.Dispose()

    return $bmp
}

# ---------- 母版 → PNG 字节 ----------
function ConvertTo-PngBytes {
    param([System.Drawing.Bitmap]$Bitmap)

    $ms = New-Object System.IO.MemoryStream
    $Bitmap.Save($ms, [System.Drawing.Imaging.ImageFormat]::Png)
    return $ms.ToArray()
}

# ---------- 写 ICO 文件 ----------
function Write-IcoFile {
    param(
        [string]$Path,
        [hashtable]$Images   # key = size, value = byte[]
    )

    $fs = [System.IO.File]::Create($Path)
    try {
        $bw = New-Object System.IO.BinaryWriter $fs

        # ICONDIR
        $bw.Write([uint16]0)                # reserved
        $bw.Write([uint16]1)                # type = ICO
        $bw.Write([uint16]$Images.Count)    # count

        # 入口大小
        $entrySize   = 16
        $headerSize  = 6 + ($Images.Count * $entrySize)
        $offset      = $headerSize

        # ICONDIRENTRY[]
        foreach ($size in ($Images.Keys | Sort-Object)) {
            $bytes = $Images[$size]
            $dim   = [byte](if ($size -ge 256) { 0 } else { $size })
            $bw.Write([byte]$dim)            # width
            $bw.Write([byte]$dim)            # height
            $bw.Write([byte]0)               # colors
            $bw.Write([byte]0)               # reserved
            $bw.Write([uint16]1)             # planes
            $bw.Write([uint16]32)            # bit count
            $bw.Write([uint32]$bytes.Length) # size
            $bw.Write([uint32]$offset)       # offset
            $offset += $bytes.Length
        }

        # 图像数据
        foreach ($size in ($Images.Keys | Sort-Object)) {
            $bw.Write($Images[$size])
        }
    }
    finally {
        $fs.Close()
    }
}

# ---------- 主流程 ----------
$sizes = 16, 32, 48, 64, 128, 256
$images = @{}

# 先画 256 母版，再缩放得到其余分辨率，确保 16x16 不会丢笔锋
$master = New-MasterBitmap -Size 256
try {
    foreach ($s in $sizes) {
        if ($s -eq 256) {
            $bmp = $master
        } else {
            $bmp = New-Object System.Drawing.Bitmap $s, $s
            $g   = [System.Drawing.Graphics]::FromImage($bmp)
            $g.InterpolationMode = 'HighQualityBicubic'
            $g.SmoothingMode     = 'HighQuality'
            $g.PixelOffsetMode   = 'HighQuality'
            $g.DrawImage($master, 0, 0, $s, $s)
            $g.Dispose()
        }
        $images[$s] = ConvertTo-PngBytes -Bitmap $bmp
        if ($s -ne 256) { $bmp.Dispose() }
    }
}
finally {
    $master.Dispose()
}

$outDir = Split-Path -Parent $OutputPath
if ($outDir -and -not (Test-Path $outDir)) {
    New-Item -ItemType Directory -Path $outDir -Force | Out-Null
}
Write-IcoFile -Path $OutputPath -Images $images

$total = (Get-Item $OutputPath).Length
Write-Host "[build-icon] wrote $OutputPath ($([math]::Round($total/1KB,1)) KB, $(@($images.Keys).Count) resolutions)"
