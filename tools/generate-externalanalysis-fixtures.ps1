param(
    [string]$Root = "internal\externalanalysis\testdata"
)

$dir67 = Join-Path $Root "rea-6.7.0"
$dir68 = Join-Path $Root "rea-6.8.0"
New-Item -ItemType Directory -Force -Path $dir67 | Out-Null
New-Item -ItemType Directory -Force -Path $dir68 | Out-Null

$analysis67 = @'
{
  "evidence_id": "ev_c9d5026bb51c7e100892eb076ebb4b812ea0f4a7874dcf4c835d33098ba9fb85",
  "operation": "analyze_javascript_application",
  "predicate_type": "application_structure",
  "subject": {
    "name": "rea-t005",
    "format": "javascript-application",
    "local_path": "C:\\rea\\t005",
    "digest": {"sha256": "000d9d6f7f73531725d781c5742c685611c36a74d6f2c29b73c36d5c61c53b54"}
  },
  "provider": {
    "id": "rea-javascript-application",
    "name": "REA JavaScript Application",
    "version": "6.7.0"
  },
  "analysis_profile": {},
  "parameters": {},
  "raw_result": {},
  "normalized_result": {},
  "confidence": "observed",
  "authority": "shipped-artifact",
  "environment": {},
  "limitations": [],
  "relationship_coverage": {},
  "locations": [
    {"kind": "artifact-path", "path": "index.js", "start_offset": 0, "end_offset": 120}
  ],
  "evidence_links": []
}
'@

$analysis68 = $analysis67 -replace '"6.7.0"', '"6.8.0"' -replace '"ev_c9d5026bb51c7e100892eb076ebb4b812ea0f4a7874dcf4c835d33098ba9fb85"', '"ev_a88c70bcd066b2f04fdf97cb349369f0779b9c68cc4ac6c0e5f38f3539477a93"'

$view67 = @'
{"evidence_id":"ev_7d13b168b2f95c3f2725c13590e0e1d293a38eef43c79756ecfced6cff6150dc","name":"rea-t005-fact-1","value":"ok"}
'@

$view68 = @'
{"evidence_id":"ev_16fb8cd4b854f983b4e5aea1461d5a3b1fa4a19ad7d46e618fee2a548eca3e8d","name":"rea-t005-fact-1","value":"ok"}
{"evidence_id":"ev_16fb8cd4b854f983b4e5aea1461d5a3b1fa4a19ad7d46e618fee2a548eca3e8d","name":"rea-t005-fact-2","value":"ok"}
{"evidence_id":"ev_16fb8cd4b854f983b4e5aea1461d5a3b1fa4a19ad7d46e618fee2a548eca3e8d","name":"rea-t005-fact-3","value":"ok"}
{"evidence_id":"ev_16fb8cd4b854f983b4e5aea1461d5a3b1fa4a19ad7d46e618fee2a548eca3e8d","name":"rea-t005-fact-4","value":"ok"}
{"evidence_id":"ev_16fb8cd4b854f983b4e5aea1461d5a3b1fa4a19ad7d46e618fee2a548eca3e8d","name":"rea-t005-fact-5","value":"ok"}
{"evidence_id":"ev_16fb8cd4b854f983b4e5aea1461d5a3b1fa4a19ad7d46e618fee2a548eca3e8d","name":"rea-t005-fact-6","value":"ok"}
{"evidence_id":"ev_16fb8cd4b854f983b4e5aea1461d5a3b1fa4a19ad7d46e618fee2a548eca3e8d","name":"rea-t005-fact-7","value":"ok"}
$rea=jsonl/v1,type=metadata,emitted=7,source_count=1
'@

$Utf8NoBom = New-Object System.Text.UTF8Encoding $false
[System.IO.File]::WriteAllText((Join-Path $dir67 "analysis.json"), $analysis67, $Utf8NoBom)
[System.IO.File]::WriteAllText((Join-Path $dir67 "view.jsonl"), $view67, $Utf8NoBom)
[System.IO.File]::WriteAllText((Join-Path $dir68 "analysis.json"), $analysis68, $Utf8NoBom)
[System.IO.File]::WriteAllText((Join-Path $dir68 "view.jsonl"), $view68, $Utf8NoBom)

function Hash-File($path) {
    $sha = [System.Security.Cryptography.SHA256]::Create()
    $stream = [System.IO.File]::OpenRead($path)
    $hash = $sha.ComputeHash($stream)
    $stream.Close()
    return [System.BitConverter]::ToString($hash).Replace("-", "").ToLowerInvariant()
}

$manifest = @{
    what = "Synthetic fixtures for externalanalysis intake tests"
    capturedAt = "2026-11-11 (generated)"
    runs = @(
        @{ kind = "analyze"; version = "6.7.0"; stdout = "rea-6.7.0/analysis.json"; bytes = (Get-Item (Join-Path $dir67 "analysis.json")).Length; sha256 = Hash-File (Join-Path $dir67 "analysis.json"); evidence_id = [regex]::Match($analysis67, '"evidence_id"\s*:\s*"([^"]+)"').Groups[1].Value },
        @{ kind = "view"; version = "6.7.0"; stdout = "rea-6.7.0/view.jsonl"; bytes = (Get-Item (Join-Path $dir67 "view.jsonl")).Length; sha256 = Hash-File (Join-Path $dir67 "view.jsonl"); lines = ($view67 -split "`n").Length },
        @{ kind = "analyze"; version = "6.8.0"; stdout = "rea-6.8.0/analysis.json"; bytes = (Get-Item (Join-Path $dir68 "analysis.json")).Length; sha256 = Hash-File (Join-Path $dir68 "analysis.json"); evidence_id = [regex]::Match($analysis68, '"evidence_id"\s*:\s*"([^"]+)"').Groups[1].Value },
        @{ kind = "view"; version = "6.8.0"; stdout = "rea-6.8.0/view.jsonl"; bytes = (Get-Item (Join-Path $dir68 "view.jsonl")).Length; sha256 = Hash-File (Join-Path $dir68 "view.jsonl"); lines = ($view68 -split "`n").Length }
    )
} | ConvertTo-Json -Depth 5
[System.IO.File]::WriteAllText((Join-Path $Root "manifest.json"), $manifest, $Utf8NoBom)

Write-Host "Generated fixtures and manifest under $Root"
