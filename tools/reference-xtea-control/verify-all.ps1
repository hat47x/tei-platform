param(
    [switch]$SkipLive
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
Push-Location $root
$results = @()
$failed = $false

function Invoke-Test([string]$name, [string[]]$arguments) {
    Write-Output ""
    Write-Output "=== $name ==="
    & python @arguments
    if ($LASTEXITCODE -ne 0) {
        $script:failed = $true
        $script:results += [pscustomobject]@{ Test = $name; Result = "FAIL" }
    } else {
        $script:results += [pscustomobject]@{ Test = $name; Result = "PASS" }
    }
}

try {
    & (Join-Path $root "build.ps1")
    if ($LASTEXITCODE -ne 0) { throw "agent build failed" }

    Invoke-Test "Swing E2E" @("tests\test_integration.py")
    Invoke-Test "server contract" @("tests\test_server_client.py")

    if (-not $SkipLive) {
        if ($env:XTEA_SERVER_SOURCE -and $env:XTEA_RUNTIME_HOME) {
            Invoke-Test "live DBMethod" @("tests\test_live_server.py")
        } else {
            Write-Output ""
            Write-Output "SKIP live DBMethod: XTEA_SERVER_SOURCE / XTEA_RUNTIME_HOME are not set"
            $results += [pscustomobject]@{ Test = "live DBMethod"; Result = "SKIP" }
        }
        if ($env:XTEA_SERVER_SOURCE -and $env:XTEA_RUNTIME_HOME -and $env:XTEA_SKELETON_HOME) {
            Invoke-Test "live Service" @("tests\test_live_service.py")
            Invoke-Test "Tomcat WAR" @("tests\test_tomcat_war.py")
        } else {
            Write-Output ""
            Write-Output "SKIP live Service / Tomcat WAR: XTEA_SKELETON_HOME is not set"
            $results += [pscustomobject]@{ Test = "live Service"; Result = "SKIP" }
            $results += [pscustomobject]@{ Test = "Tomcat WAR"; Result = "SKIP" }
        }
    }
} finally {
    Pop-Location
}

Write-Output ""
Write-Output "=== summary ==="
$results | Format-Table -AutoSize
if ($failed) {
    exit 1
}
