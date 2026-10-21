param(
    [Parameter(Mandatory = $true)][string]$XteaSource,
    [Parameter(Mandatory = $true)][string]$RuntimeHome
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$toolRoot = Split-Path -Parent $root
$deps = Join-Path $toolRoot "build\deps"
$classes = Join-Path $toolRoot "build\server-classes"
$harnessClasses = Join-Path $toolRoot "build\server-harness-classes"
New-Item -ItemType Directory -Force $deps, $classes, $harnessClasses | Out-Null

$maven = "https://repo1.maven.org/maven2"
$jars = @(
    "org/eclipse/jetty/jetty-server/9.4.54.v20240208/jetty-server-9.4.54.v20240208.jar",
    "org/eclipse/jetty/jetty-servlet/9.4.54.v20240208/jetty-servlet-9.4.54.v20240208.jar",
    "org/eclipse/jetty/jetty-security/9.4.54.v20240208/jetty-security-9.4.54.v20240208.jar",
    "org/eclipse/jetty/jetty-util/9.4.54.v20240208/jetty-util-9.4.54.v20240208.jar",
    "org/eclipse/jetty/jetty-http/9.4.54.v20240208/jetty-http-9.4.54.v20240208.jar",
    "org/eclipse/jetty/jetty-io/9.4.54.v20240208/jetty-io-9.4.54.v20240208.jar",
    "javax/servlet/javax.servlet-api/3.1.0/javax.servlet-api-3.1.0.jar"
)
foreach ($jar in $jars) {
    $name = Split-Path $jar -Leaf
    $target = Join-Path $deps $name
    if (-not (Test-Path $target)) {
        Invoke-WebRequest -UseBasicParsing -Uri "$maven/$jar" -OutFile $target
    }
}

$servletApi = Join-Path $deps "javax.servlet-api-3.1.0.jar"
$sourceDir = Join-Path $XteaSource "WEB-INF\src\xeadServer"
$sources = Get-ChildItem $sourceDir -Filter *.java |
    Where-Object { $_.Name -ne "Service.java" } |
    ForEach-Object { $_.FullName }
if (-not $sources) {
    throw "XeadServer sources not found under $sourceDir"
}

$compileClasspath = "$servletApi;$RuntimeHome\*"
& javac -encoding windows-31j -source 8 -target 8 -cp $compileClasspath -d $classes $sources
if ($LASTEXITCODE -ne 0) { throw "javac failed for XeadServer" }

$jetty = (Get-ChildItem $deps -Filter "*.jar" | ForEach-Object { $_.FullName }) -join ";"
$harnessClasspath = "$servletApi;$jetty;$RuntimeHome\*"
& javac -encoding UTF-8 -source 8 -target 8 -cp "$classes;$harnessClasspath" -d $harnessClasses `
    (Join-Path $root "XteaServerHarness.java"), (Join-Path $root "H2Init.java")
if ($LASTEXITCODE -ne 0) { throw "javac failed for the server harness" }
$runClasspath = "$classes;$harnessClasses;$jetty;$RuntimeHome\*"
Set-Content -Encoding ascii (Join-Path $toolRoot "build\server-classpath.txt") $runClasspath
Write-Output "classes: $classes"
Write-Output "harness: $harnessClasses"
Write-Output "classpath written to build\server-classpath.txt"
