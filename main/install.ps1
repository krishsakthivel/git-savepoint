$ErrorActionPreference = "Stop"

$repo = "krishsakthivel/git-savepoint"
$asset = "git-savepoint-windows-amd64.exe"
$url = "https://github.com/$repo/releases/latest/download/$asset"

$tmp = Join-Path $env:TEMP "git-savepoint-install.exe"

Write-Host "downloading $asset..."
Invoke-WebRequest -Uri $url -OutFile $tmp -UseBasicParsing


Unblock-File -Path $tmp -ErrorAction SilentlyContinue

Write-Host "installing..."
& $tmp install

Remove-Item $tmp -ErrorAction SilentlyContinue

Write-Host ""
Write-Host "done. open a new terminal and run: git-savepoint"