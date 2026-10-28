param(
    [ValidateSet("modeler", "editor", "driver")]
    [string]$App = "modeler",
    [Parameter(Mandatory = $true)]
    [Alias("Home")]
    [string]$XteaHome,
    [string]$Model = "",
    [string]$TokenFile = "",
    [int]$Port = 0,
    [string]$User = "",
    [string]$Password = "",
    [switch]$Foreground
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$jar = Join-Path $root "build\xtea-control-agent.jar"
if (-not (Test-Path $jar)) {
    throw "agent jar not found; run build.ps1 (or build.cmd) first"
}
if (-not (Test-Path $XteaHome)) {
    throw "X-TEA home not found: $XteaHome"
}

$appJar = switch ($App) {
    "modeler" { "xteamdl.jar" }
    "editor"  { "xteaedt.jar" }
    "driver"  { "xteadrv.jar" }
}
$appJarPath = Join-Path $XteaHome $appJar
if (-not (Test-Path $appJarPath)) {
    throw "$appJar not found under $XteaHome"
}

if (-not $TokenFile) {
    $TokenFile = Join-Path $root ("build\{0}-endpoint.json" -f $App)
}
New-Item -ItemType Directory -Force (Split-Path -Parent $TokenFile) | Out-Null
Remove-Item $TokenFile -ErrorAction SilentlyContinue

$arguments = @(
    "-javaagent:$jar=port=$Port,tokenFile=$TokenFile",
    "-Duser.language=ja",
    "-Xmx512m",
    "-jar",
    $appJarPath
)
if ($Model) {
    $arguments += $Model
}
if ($App -eq "driver") {
    if ($User) { $arguments += $User }
    if ($Password) { $arguments += $Password }
}

if ($Foreground) {
    & java @arguments
    exit $LASTEXITCODE
}

$process = Start-Process java -ArgumentList $arguments -PassThru `
    -RedirectStandardOutput (Join-Path $root "build\$App-stdout.log") `
    -RedirectStandardError (Join-Path $root "build\$App-stderr.log")
for ($i = 0; $i -lt 300 -and -not (Test-Path $TokenFile); $i++) {
    if ($process.HasExited) {
        $stderr = Join-Path $root "build\$App-stderr.log"
        if (Test-Path $stderr) { Get-Content $stderr }
        throw "X-TEA exited before the control endpoint was written (exit $($process.ExitCode))"
    }
    Start-Sleep -Milliseconds 100
}
if (-not (Test-Path $TokenFile)) {
    throw "control endpoint was not written within 30 seconds"
}
Write-Output "pid: $($process.Id)"
Get-Content $TokenFile
Write-Output "next: python xtea_control.py --endpoint `"$TokenFile`" windows"

