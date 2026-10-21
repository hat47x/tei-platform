param(
    [string]$Target = "upstream",
    [switch]$Latest
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$lock = Get-Content (Join-Path $root "upstream.json") -Raw | ConvertFrom-Json
$targetPath = if ([System.IO.Path]::IsPathRooted($Target)) { $Target } else { Join-Path $root $Target }
New-Item -ItemType Directory -Force $targetPath | Out-Null

foreach ($name in @("XeadModeler", "XeadEditor", "XeadDriver", "XeadServer")) {
    $entry = $lock.repositories.$name
    $destination = Join-Path $targetPath $name
    if (-not (Test-Path (Join-Path $destination ".git"))) {
        Write-Output "clone $name"
        git clone --depth 1 $entry.url $destination
        if ($LASTEXITCODE -ne 0) { throw "git clone failed: $name" }
    } else {
        Write-Output "reuse $name"
    }
    if (-not $Latest) {
        $current = (git -C $destination rev-parse HEAD).Trim()
        if ($current -ne $entry.commit) {
            Write-Warning "$name HEAD is $current; upstream.json pins $($entry.commit). Use -Latest to accept the current master."
        }
    }
    $actual = (git -C $destination rev-parse HEAD).Trim()
    Write-Output ("{0} {1}" -f $name, $actual)
}

Write-Output "source root: $targetPath"
