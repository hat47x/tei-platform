param(
    [string]$OutputDir = "build"
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $root

$output = Join-Path $root $OutputDir
$classes = Join-Path $output "agent-classes"
$fixtureClasses = Join-Path $output "fixture-classes"
$jarPath = Join-Path $output "xtea-control-agent.jar"
New-Item -ItemType Directory -Force $classes, $fixtureClasses | Out-Null
Remove-Item -Recurse -Force $classes, $fixtureClasses -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force $classes, $fixtureClasses | Out-Null

$sources = Get-ChildItem -Recurse -File (Join-Path $root "agent\src\xteaControl\*.java") |
    ForEach-Object { $_.FullName }
& javac --release 8 -encoding UTF-8 -d $classes $sources
if ($LASTEXITCODE -ne 0) { throw "javac failed for the control agent" }

$manifest = Join-Path $output "MANIFEST.MF"
@(
    "Manifest-Version: 1.0"
    "Premain-Class: xteaControl.Agent"
    "Agent-Class: xteaControl.Agent"
    "Can-Redefine-Classes: false"
    "Can-Retransform-Classes: false"
    "Implementation-Title: TEI X-TEA Control Agent"
    "Implementation-Version: 1"
    ""
) -join "`r`n" | Set-Content -Encoding ascii $manifest

& jar cfm $jarPath $manifest -C $classes .
if ($LASTEXITCODE -ne 0) { throw "jar failed for the control agent" }

$fixture = Join-Path $root "tests\fixtures\FakeXteaApp.java"
& javac --release 8 -encoding UTF-8 -d $fixtureClasses $fixture
if ($LASTEXITCODE -ne 0) { throw "javac failed for the test fixture" }

Write-Output "agent:   $jarPath"
Write-Output "fixture: $fixtureClasses"
