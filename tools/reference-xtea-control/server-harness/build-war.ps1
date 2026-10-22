param(
    [Parameter(Mandatory = $true)][string]$XteaSource,
    [Parameter(Mandatory = $true)][string]$RuntimeHome,
    [Parameter(Mandatory = $true)][string]$Xeaf,
    [string]$Target = "build\tomcat-war"
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$toolRoot = Split-Path -Parent $root
$deps = Join-Path $toolRoot "build\deps"
New-Item -ItemType Directory -Force $deps | Out-Null

$tomcatJar = Join-Path $deps "tomcat-embed-core-9.0.89.jar"
if (-not (Test-Path $tomcatJar)) {
    Invoke-WebRequest -UseBasicParsing `
        -Uri "https://repo1.maven.org/maven2/org/apache/tomcat/embed/tomcat-embed-core/9.0.89/tomcat-embed-core-9.0.89.jar" `
        -OutFile $tomcatJar
}
$annotationsJar = Join-Path $deps "tomcat-annotations-api-9.0.89.jar"
if (-not (Test-Path $annotationsJar)) {
    Invoke-WebRequest -UseBasicParsing `
        -Uri "https://repo1.maven.org/maven2/org/apache/tomcat/tomcat-annotations-api/9.0.89/tomcat-annotations-api-9.0.89.jar" `
        -OutFile $annotationsJar
}

# Reuse the compatibility build (Session/Service patches included).
& (Join-Path $root "build.ps1") -XteaSource $XteaSource -RuntimeHome $RuntimeHome
if ($LASTEXITCODE -ne 0) { throw "server compatibility build failed" }

$targetPath = if ([System.IO.Path]::IsPathRooted($Target)) { $Target } else { Join-Path $toolRoot $Target }
if (Test-Path $targetPath) { Remove-Item -Recurse -Force $targetPath }
$webInf = Join-Path $targetPath "WEB-INF"
$classes = Join-Path $webInf "classes"
$lib = Join-Path $webInf "lib"
New-Item -ItemType Directory -Force $classes, $lib | Out-Null

Copy-Item (Join-Path $toolRoot "build\server-classes\*") $classes -Recurse
Copy-Item (Join-Path $RuntimeHome "*.jar") $lib

$escapedXeaf = [System.Security.SecurityElement]::Escape((Resolve-Path $Xeaf).Path)
$webXml = @"
<?xml version="1.0" encoding="UTF-8"?>
<web-app xmlns="http://xmlns.jcp.org/xml/ns/javaee" version="3.1">
  <servlet>
    <servlet-name>DBMethod</servlet-name>
    <servlet-class>xeadServer.DBMethod</servlet-class>
    <init-param><param-name>SystemDefinition</param-name><param-value>$escapedXeaf</param-value></init-param>
    <init-param><param-name>DataSource</param-name><param-value>SalesDs</param-value></init-param>
    <init-param><param-name>SubDB</param-name><param-value></param-value></init-param>
  </servlet>
  <servlet-mapping><servlet-name>DBMethod</servlet-name><url-pattern>/DBMethod</url-pattern></servlet-mapping>
  <servlet>
    <servlet-name>Service</servlet-name>
    <servlet-class>xeadServer.Service</servlet-class>
    <init-param><param-name>SystemDefinition</param-name><param-value>$escapedXeaf</param-value></init-param>
    <init-param><param-name>User</param-name><param-value>00000</param-value></init-param>
    <init-param><param-name>Password</param-name><param-value>0000000000</param-value></init-param>
  </servlet>
  <servlet-mapping><servlet-name>Service</servlet-name><url-pattern>/Service</url-pattern></servlet-mapping>
</web-app>
"@
[System.IO.File]::WriteAllText((Join-Path $webInf "web.xml"), $webXml, (New-Object System.Text.UTF8Encoding($false)))

$runnerClasses = Join-Path $toolRoot "build\tomcat-runner"
New-Item -ItemType Directory -Force $runnerClasses | Out-Null
& javac -encoding UTF-8 -source 8 -target 8 -cp $tomcatJar -d $runnerClasses (Join-Path $root "TomcatRunner.java")
if ($LASTEXITCODE -ne 0) { throw "javac failed for TomcatRunner" }

$runnerClasspath = "$runnerClasses;$tomcatJar;$annotationsJar"
Set-Content -Encoding ascii (Join-Path $toolRoot "build\tomcat-classpath.txt") $runnerClasspath
Write-Output "webapp: $targetPath"
Write-Output "runner: $runnerClasses"
Write-Output "classpath written to build\tomcat-classpath.txt"
