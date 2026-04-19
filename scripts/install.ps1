param(
    [switch]$Unattended,
    [string]$Version = "latest"
)

$ErrorActionPreference = "Stop"

function Write-Prompt($msg, $color = "Cyan") { Write-Host $msg -ForegroundColor $color }
function Write-Input($msg) { Write-Host $msg -ForegroundColor "Yellow" }
function Write-Success($msg) { Write-Host $msg -ForegroundColor "Green" }
function Write-Err($msg) { Write-Host $msg -ForegroundColor "Red" }

function ConvertTo-TomlString($value) {
    $escaped = [string]$value
    $escaped = $escaped.Replace('\', '\\').Replace('"', '\"')
    return '"' + $escaped + '"'
}

function Set-TomlBlock($path, $block) {
    $normalizedBlock = $block.Trim()
    if (Test-Path $path) {
        $existing = Get-Content $path -Raw
        $pattern = '(?m)^\[mcp_servers\.sqlserver\]\r?\n(?:(?!^\[).*\r?\n)*(?:^\[mcp_servers\.sqlserver\.env\]\r?\n(?:(?!^\[).*\r?\n)*)?'
        if ([regex]::IsMatch($existing, $pattern)) {
            $updated = [regex]::Replace($existing, $pattern, $normalizedBlock + "`r`n")
        } else {
            $updated = $existing.TrimEnd() + "`r`n`r`n" + $normalizedBlock + "`r`n"
        }
    } else {
        $updated = $normalizedBlock + "`r`n"
    }
    Set-Content -Path $path -Value $updated -Encoding UTF8
}

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
    MCP_SQLSERVER_ALLOW_SCHEMA_CHANGES = "false"
    MCP_SQLSERVER_ALLOW_DANGEROUS_SQL = "false"
    MCP_SQLSERVER_ALLOW_PROCEDURE_CALLS = "true"
}

function Set-CodexConfig($env, $cmd) {
    $path = "$HOME\.codex\config.toml"
    $dir = Split-Path $path -Parent
    if (-not (Test-Path $dir)) { New-Item $dir -ItemType Directory -Force | Out-Null }

    $cmdToml = ConvertTo-TomlString $cmd
    $hostToml = ConvertTo-TomlString $env.SQL_SERVER_HOST
    $portToml = ConvertTo-TomlString $env.SQL_SERVER_PORT
    $userToml = ConvertTo-TomlString $env.SQL_SERVER_USER
    $passToml = ConvertTo-TomlString $env.SQL_SERVER_PASSWORD
    $dbToml = ConvertTo-TomlString $env.SQL_SERVER_DATABASE
    $encryptToml = ConvertTo-TomlString $env.SQL_SERVER_ENCRYPT
    $trustCertToml = ConvertTo-TomlString $env.SQL_SERVER_TRUST_CERT
    $maxRowsToml = ConvertTo-TomlString $env.MCP_SQLSERVER_MAX_ROWS
    $allowSchemaToml = ConvertTo-TomlString $env.MCP_SQLSERVER_ALLOW_SCHEMA_CHANGES
    $allowDangerousToml = ConvertTo-TomlString $env.MCP_SQLSERVER_ALLOW_DANGEROUS_SQL
    $allowProceduresToml = ConvertTo-TomlString $env.MCP_SQLSERVER_ALLOW_PROCEDURE_CALLS

    $block = @"
[mcp_servers.sqlserver]
command = $cmdToml
startup_timeout_sec = 10
tool_timeout_sec = 60

[mcp_servers.sqlserver.env]
SQL_SERVER_HOST = $hostToml
SQL_SERVER_PORT = $portToml
SQL_SERVER_USER = $userToml
SQL_SERVER_PASSWORD = $passToml
SQL_SERVER_DATABASE = $dbToml
SQL_SERVER_ENCRYPT = $encryptToml
SQL_SERVER_TRUST_CERT = $trustCertToml
MCP_SQLSERVER_MAX_ROWS = $maxRowsToml
MCP_SQLSERVER_ALLOW_SCHEMA_CHANGES = $allowSchemaToml
MCP_SQLSERVER_ALLOW_DANGEROUS_SQL = $allowDangerousToml
MCP_SQLSERVER_ALLOW_PROCEDURE_CALLS = $allowProceduresToml
"@
    Set-TomlBlock $path $block
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

switch ($choice) {
    "1" {
        Set-CodexConfig $envVars $exe
        Set-CursorConfig $envVars $exe
        Set-ClaudeConfig $envVars $exe
    }
    "2" { Set-CodexConfig $envVars $exe }
    "3" { Set-CursorConfig $envVars $exe }
    "4" { Set-ClaudeConfig $envVars $exe }
}

Write-Prompt "`n=== DONE ===`n" $GREEN
Write-Host "Restart your client and use health_check tool."
