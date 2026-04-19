param(
    [switch]$Unattended,
    [string]$Version = "latest"
)

$ErrorActionPreference = "Stop"

function Write-Prompt($msg, $color = "Cyan") { Write-Host $msg -ForegroundColor $color }
function Write-Input($msg) { Write-Host $msg -ForegroundColor "Yellow" }
function Write-Success($msg) { Write-Host $msg -ForegroundColor "Green" }
function Write-Err($msg) { Write-Host $msg -ForegroundColor "Red" }

$GREEN = "Green"
$YELLOW = "Yellow"
$CYAN = "Cyan"
$MAGENTA = "Magenta"

Write-Prompt "`n=== MCP SQL Server Installer ===`n" $GREEN

$releasePath = "https://github.com/TranHuyThang9999/mcp-sqlserver/releases/$Version/download"
$appName = "mcp-sqlserver"

if ($Version -eq "latest") {
    $releaseUrl = "https://api.github.com/repos/TranHuyThang9999/mcp-sqlserver/releases/latest"
    $response = Invoke-RestMethod $releaseUrl -UseBasicParsing
    $Version = "v" + $response.tag_name -replace "v", ""
    Write-Prompt "Version: $Version`n" $CYAN
}

$downloadUrl = "$releasePath/mcp-sqlserver-windows-amd64.zip"
$tempZip = "$env:TEMP\mcp-sqlserver.zip"
$installDir = "$env:LOCALAPPDATA\$appName"

Write-Prompt "Downloading..." $YELLOW
try {
    Invoke-WebRequest -Uri $downloadUrl -OutFile $tempZip -UseBasicParsing
} catch {
    Write-Err "Download failed: $_"
    exit 1
}

Write-Prompt "Extracting..." $YELLOW
Expand-Archive -Path $tempZip -DestinationPath $installDir -Force
Remove-Item $tempZip -Force

$exe = Join-Path $installDir "mcp-sqlserver.exe"
if (-not (Test-Path $exe)) {
    $exe = Get-ChildItem $installDir -Filter "*.exe" | Select-Object -First 1 -ExpandProperty FullName
}

Write-Success "Installed: $exe`n" $GREEN

Write-Input "`n>>> SQL SERVER CONFIGURATION`n" $MAGENTA
$sqlHost = Read-Host "Host     [localhost]"
if ([string]::IsNullOrWhiteSpace($sqlHost)) { $sqlHost = "localhost" }

$port = Read-Host "Port     [1433]"
if ([string]::IsNullOrWhiteSpace($port)) { $port = "1433" }

$user = Read-Host "User    [sa]"
if ([string]::IsNullOrWhiteSpace($user)) { $user = "sa" }

$sec = Read-Host "Password" -AsSecureString
$bstr = [System.Runtime.InteropServices.Marshal]::SecureStringToBSTR($sec)
$pass = [System.Runtime.InteropServices.Marshal]::PtrToStringBSTR($bstr)
[System.Runtime.InteropServices.Marshal]::ZeroFreeBSTR($bstr)

$db = Read-Host "Database [master]"
if ([string]::IsNullOrWhiteSpace($db)) { $db = "master" }

Write-Prompt "`n>>> CONFIGURING CLIENTS`n" $MAGENTA
Write-Host "1. All clients"
Write-Host "2. Codex only"
Write-Host "3. Cursor/VS Code only"
Write-Host "4. Claude Desktop only"
$choice = Read-Host "Selection [1]"

$envVars = @{
    SQL_SERVER_HOST = $sqlHost
    SQL_SERVER_PORT = $port
    SQL_SERVER_USER = $user
    SQL_SERVER_PASSWORD = $pass
    SQL_SERVER_DATABASE = $db
    SQL_SERVER_ENCRYPT = "disable"
    SQL_SERVER_TRUST_CERT = "true"
    MCP_SQLSERVER_MAX_ROWS = "500"
}

function Set-CodexConfig($env, $cmd) {
    $path = "$HOME\.codex\config.toml"
    $dir = Split-Path $path -Parent
    if (-not (Test-Path $dir)) { New-Item $dir -ItemType Directory -Force | Out-Null }

    $block = @"
[mcp_servers.sqlserver]
command = "$cmd"
startup_timeout_sec = 10
tool_timeout_sec = 60

[mcp_servers.sqlserver.env]
SQL_SERVER_HOST = "$($env.SQL_SERVER_HOST)"
SQL_SERVER_PORT = "$($env.SQL_SERVER_PORT)"
SQL_SERVER_USER = "$($env.SQL_SERVER_USER)"
SQL_SERVER_PASSWORD = "$($env.SQL_SERVER_PASSWORD)"
SQL_SERVER_DATABASE = "$($env.SQL_SERVER_DATABASE)"

"@
    Add-Content -Path $path -Value $block -Encoding UTF8
    Write-Success "Codex: $path"
}

function Set-CursorConfig($env, $cmd) {
    $path = "$HOME\.cursor\mcp.json"
    $dir = Split-Path $path -Parent
    if (-not (Test-Path $dir)) { New-Item $dir -ItemType Directory -Force | Out-Null }

    $config = @{
        mcpServers = @{
            sqlserver = @{
                command = $cmd
                env = $env
            }
        }
    }
    $config | ConvertTo-Json -Depth 10 | Set-Content -Path $path -Encoding UTF8
    Write-Success "Cursor: $path"
}

function Set-ClaudeConfig($env, $cmd) {
    $path = "$env:APPDATA\Claude\claude_desktop_config.json"
    $dir = Split-Path $path -Parent
    if (-not (Test-Path $dir)) { New-Item $dir -ItemType Directory -Force | Out-Null }

    if (Test-Path $path) {
        $existing = Get-Content $path -Raw | ConvertFrom-Json
        $mcp = $existing.mcpServers
    } else {
        $mcp = @{}
    }
    $mcp | Add-Member -Name "sqlserver" -Value @{
        command = $cmd
        env = $env
    } -Force
    @{ mcpServers = $mcp } | ConvertTo-Json -Depth 10 | Set-Content -Path $path -Encoding UTF8
    Write-Success "Claude: $path"
}

$exeEscaped = $exe -replace '\\', '\\'

switch ($choice) {
    "1" {
        Set-CodexConfig $envVars $exeEscaped
        Set-CursorConfig $envVars $exeEscaped
        Set-ClaudeConfig $envVars $exeEscaped
    }
    "2" { Set-CodexConfig $envVars $exeEscaped }
    "3" { Set-CursorConfig $envVars $exeEscaped }
    "4" { Set-ClaudeConfig $envVars $exeEscaped }
}

Write-Prompt "`n=== DONE ===`n" $GREEN
Write-Host "Restart your client and use health_check tool."
