param(
    [switch]$SkipLive,
    [switch]$IncludeXeadServer
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
    Invoke-Test "model CLI" @("tests\test_model_cli.py")

    if ($IncludeXeadServer) {
        Invoke-Test "server contract (reference)" @("tests\test_server_client.py")
        if (-not $SkipLive) {
            if ($env:XTEA_SERVER_SOURCE -and $env:XTEA_RUNTIME_HOME) {
                Invoke-Test "live DBMethod (reference)" @("tests\test_live_server.py")
            } else {
                Write-Output ""
                Write-Output "SKIP live DBMethod: XTEA_SERVER_SOURCE / XTEA_RUNTIME_HOME are not set"
                $results += [pscustomobject]@{ Test = "live DBMethod"; Result = "SKIP" }
            }
            if ($env:XTEA_SERVER_SOURCE -and $env:XTEA_RUNTIME_HOME -and $env:XTEA_SKELETON_HOME) {
                Invoke-Test "live Service (reference)" @("tests\test_live_service.py")
                Invoke-Test "Tomcat WAR (reference)" @("tests\test_tomcat_war.py")
                if ($env:TOMCAT_HOME -and $env:TOMCAT_URL) {
                    Invoke-Test "external Tomcat (reference)" @("tests\test_external_tomcat.py")
                } else {
                    Write-Output ""
                    Write-Output "SKIP external Tomcat: TOMCAT_HOME / TOMCAT_URL are not set"
                    $results += [pscustomobject]@{ Test = "external Tomcat"; Result = "SKIP" }
                }
            } else {
                Write-Output ""
                Write-Output "SKIP live Service / Tomcat WAR: XTEA_SKELETON_HOME is not set"
                $results += [pscustomobject]@{ Test = "live Service"; Result = "SKIP" }
                $results += [pscustomobject]@{ Test = "Tomcat WAR"; Result = "SKIP" }
            }
        }
    } else {
        Write-Output ""
        Write-Output "SKIP XeadServer: out of scope (use -IncludeXeadServer for the legacy reference tests)"
        $results += [pscustomobject]@{ Test = "XeadServer (reference)"; Result = "SKIP" }
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
