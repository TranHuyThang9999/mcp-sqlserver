param(
    [switch]$Unattended,
    [string]$Version = "latest",
    [string]$LocalExe = "",
    [string]$CodexConfigPath = ""
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

$repo = "TranHuyThang9999/mcp-sqlserver"
$appName = "mcp-sqlserver"

if (-not [string]::IsNullOrWhiteSpace($LocalExe)) {
    if (-not (Test-Path $LocalExe)) {
        Write-Err "LocalExe not found: $LocalExe"
        exit 1
    }
    $exe = (Get-Item -LiteralPath $LocalExe).FullName
    Write-Success "Using local executable: $exe`n" $GREEN
} else {
    if ($Version -eq "latest") {
        $releaseUrl = "https://api.github.com/repos/$repo/releases/latest"
        $response = Invoke-RestMethod $releaseUrl -UseBasicParsing
        $Version = $response.tag_name
    } elseif (-not $Version.StartsWith("v")) {
        $Version = "v$Version"
    }

    Write-Prompt "Version: $Version`n" $CYAN

    $downloadUrl = "https://github.com/$repo/releases/download/$Version/mcp-sqlserver-windows-amd64.zip"
    $tempZip = Join-Path $env:TEMP "mcp-sqlserver-$Version.zip"
    $installRoot = Join-Path $env:LOCALAPPDATA $appName
    $installDir = Join-Path $installRoot "releases\$Version"

    Write-Prompt "Downloading..." $YELLOW
    try {
        Invoke-WebRequest -Uri $downloadUrl -OutFile $tempZip -UseBasicParsing
    } catch {
        Write-Err "Download failed: $_"
        exit 1
    }

    Write-Prompt "Extracting..." $YELLOW
    New-Item -Path $installDir -ItemType Directory -Force | Out-Null
    Expand-Archive -Path $tempZip -DestinationPath $installDir -Force
    Remove-Item $tempZip -Force

    $expectedExe = Join-Path $installDir "mcp-sqlserver-windows-amd64\mcp-sqlserver.exe"
    if (Test-Path $expectedExe) {
        $exe = $expectedExe
    } else {
        $exe = Get-ChildItem -Path $installDir -Recurse -Filter "mcp-sqlserver.exe" -File -ErrorAction SilentlyContinue |
            Sort-Object LastWriteTime -Descending |
            Select-Object -First 1 -ExpandProperty FullName
    }
    if (-not $exe -or -not (Test-Path $exe)) {
        Write-Err "Install failed: could not locate mcp-sqlserver.exe under $installDir"
        exit 1
    }

    Write-Success "Installed: $exe`n" $GREEN
}

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
    if ([string]::IsNullOrWhiteSpace($CodexConfigPath)) {
        $path = "$HOME\.codex\config.toml"
    } else {
        $path = $CodexConfigPath
    }
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
