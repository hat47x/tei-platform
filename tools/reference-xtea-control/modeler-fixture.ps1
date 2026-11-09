param(
    [Parameter(Mandatory = $true)][string]$ModelHome,
    [string]$Source = "MfgOrder.xead",
    [string]$Target = "build\modeler-fixture"
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$targetPath = if ([System.IO.Path]::IsPathRooted($Target)) { $Target } else { Join-Path $root $Target }
if (Test-Path $targetPath) {
    Remove-Item -Recurse -Force $targetPath
}
New-Item -ItemType Directory -Force $targetPath | Out-Null
$sourcePath = Join-Path $ModelHome $Source
if (-not (Test-Path $sourcePath)) {
    throw "model not found: $sourcePath"
}
$model = Join-Path $targetPath $Source
Copy-Item $sourcePath $model
Write-Output "fixture: $targetPath"
Write-Output "model:   $model"
Write-Output "This model is disposable. The save scenario changes its system name."
