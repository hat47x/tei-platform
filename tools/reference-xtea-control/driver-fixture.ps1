param(
    [Parameter(Mandatory = $true)][string]$SkeletonHome,
    [string]$Target = "build\driver-fixture"
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$targetPath = if ([System.IO.Path]::IsPathRooted($Target)) { $Target } else { Join-Path $root $Target }
if (Test-Path $targetPath) {
    Remove-Item -Recurse -Force $targetPath
}
Copy-Item -Recurse $SkeletonHome $targetPath
Remove-Item (Join-Path $targetPath "db\db.lck") -Force -ErrorAction SilentlyContinue
Remove-Item (Join-Path $targetPath "derby.log") -Force -ErrorAction SilentlyContinue

$dbPath = (Join-Path $targetPath "db").Replace("\", "/")
$xeaf = Join-Path $targetPath "Skeleton.xeaf"
$text = [System.IO.File]::ReadAllText($xeaf)
$old = "jdbc:derby://localhost/&lt;CURRENT>\db"
$new = "jdbc:derby:$dbPath"
if (-not $text.Contains($old)) {
    throw "DatabaseName pattern not found in $xeaf"
}
$text = $text.Replace($old, $new)
[System.IO.File]::WriteAllText($xeaf, $text, (New-Object System.Text.UTF8Encoding($false)))

Write-Output "fixture: $targetPath"
Write-Output "xeaf:    $xeaf"
Write-Output "db:      $dbPath"
Write-Output "This database is disposable. The scenario mutates one department name."
