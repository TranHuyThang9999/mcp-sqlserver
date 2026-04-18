param(
    [string]$ServerName = "sqlserver"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Read-Default {
    param(
        [string]$Prompt,
        [string]$Default = "",
        [switch]$Secret
    )

    if ($Secret) {
        $secure = Read-Host "$Prompt" -AsSecureString
        $bstr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($secure)
        try {
            return [Runtime.InteropServices.Marshal]::PtrToStringBSTR($bstr)
        }
        finally {
            [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($bstr)
        }
    }

    if ($Default -ne "") {
        $value = Read-Host "$Prompt [$Default]"
        if ([string]::IsNullOrWhiteSpace($value)) {
            return $Default
        }
        return $value.Trim()
    }

    return (Read-Host $Prompt).Trim()
}

function Ensure-Parent {
    param([string]$Path)

    $parent = Split-Path -Parent $Path
    if (-not [string]::IsNullOrWhiteSpace($parent)) {
        New-Item -ItemType Directory -Force -Path $parent | Out-Null
    }
}

function ConvertTo-JsonText {
    param($Value)

    return ($Value | ConvertTo-Json -Depth 20)
}

function ConvertTo-Hashtable {
    param($InputObject)

    if ($null -eq $InputObject) {
        return $null
    }

    if ($InputObject -is [System.Collections.IDictionary]) {
        $hash = [ordered]@{}
        foreach ($key in $InputObject.Keys) {
            $hash[$key] = ConvertTo-Hashtable $InputObject[$key]
        }
        return $hash
    }

    if ($InputObject -is [System.Collections.IEnumerable] -and $InputObject -isnot [string]) {
        $items = @()
        foreach ($item in $InputObject) {
            $items += ConvertTo-Hashtable $item
        }
        return $items
    }

    if ($InputObject.PSObject.Properties.Count -gt 0 -and $InputObject.GetType().Name -eq "PSCustomObject") {
        $hash = [ordered]@{}
        foreach ($property in $InputObject.PSObject.Properties) {
            $hash[$property.Name] = ConvertTo-Hashtable $property.Value
        }
        return $hash
    }

    return $InputObject
}

function Read-JsonObject {
    param([string]$Path)

    if (-not (Test-Path $Path)) {
        return [ordered]@{}
    }

    $content = Get-Content -Raw -Path $Path
    if ([string]::IsNullOrWhiteSpace($content)) {
        return [ordered]@{}
    }

    return ConvertTo-Hashtable ($content | ConvertFrom-Json)
}

function Write-JsonMcpConfig {
    param(
        [string]$Path,
        [hashtable]$ServerConfig
    )

    Ensure-Parent $Path
    $json = Read-JsonObject $Path

    if (-not $json.ContainsKey("mcpServers") -or $null -eq $json["mcpServers"]) {
        $json["mcpServers"] = [ordered]@{}
    }

    $json["mcpServers"][$ServerName] = $ServerConfig
    ConvertTo-JsonText $json | Set-Content -Encoding UTF8 -Path $Path
}

function Escape-TomlString {
    param([string]$Value)

    return $Value.Replace("\", "\\").Replace('"', '\"')
}

function Write-CodexConfig {
    param(
        [string]$Path,
        [hashtable]$Env,
        [string]$Command
    )

    Ensure-Parent $Path

    $begin = "# BEGIN mcp-sqlserver managed block"
    $end = "# END mcp-sqlserver managed block"
    $existing = ""
    if (Test-Path $Path) {
        $existing = Get-Content -Raw -Path $Path
        $existing = [regex]::Replace($existing, "(?s)\r?\n?# BEGIN mcp-sqlserver managed block.*?# END mcp-sqlserver managed block\r?\n?", "`r`n")
    }

    $lines = New-Object System.Collections.Generic.List[string]
    $lines.Add($begin)
    $lines.Add("[mcp_servers.$ServerName]")
    $lines.Add("command = ""$(Escape-TomlString $Command)""")
    $lines.Add("startup_timeout_sec = 10")
    $lines.Add("tool_timeout_sec = 60")
    $lines.Add("")
    $lines.Add("[mcp_servers.$ServerName.env]")
    foreach ($key in ($Env.Keys | Sort-Object)) {
        $lines.Add("$key = ""$(Escape-TomlString ([string]$Env[$key]))""")
    }
    $lines.Add($end)

    $newBlock = [string]::Join("`r`n", $lines)
    $newContent = ($existing.TrimEnd() + "`r`n`r`n" + $newBlock + "`r`n").TrimStart()
    Set-Content -Encoding UTF8 -Path $Path -Value $newContent
}

function New-ServerConfig {
    param(
        [hashtable]$Env,
        [string]$Command,
        [switch]$Trust
    )

    $config = [ordered]@{
        command = $Command
        env = $Env
    }

    if ($Trust) {
        $config["trust"] = $true
        $config["timeout"] = 30000
    }

    return $config
}

$root = Split-Path -Parent $PSScriptRoot
$binary = Join-Path $root "mcp-sqlserver.exe"
if (-not (Test-Path $binary)) {
    $fallback = Join-Path $root "mcp-sqlserver-test.exe"
    if (Test-Path $fallback) {
        $binary = $fallback
    }
    else {
        throw "Cannot find mcp-sqlserver.exe beside this installer. Download the release zip that includes the Windows binary."
    }
}

Write-Host ""
Write-Host "MCP SQL Server installer"
Write-Host "Binary: $binary"
Write-Host ""

$hostName = Read-Default "SQL Server host" "localhost"
$port = Read-Default "SQL Server port" "1433"
$user = Read-Default "SQL Server user" "sa"
$password = Read-Default "SQL Server password" -Secret
$database = "master"
$encrypt = "disable"
$trustCert = "true"
$maxRows = "500"
$allowSchema = "false"
$allowDangerous = "false"
$allowProcedures = "true"

$envVars = [ordered]@{
    SQL_SERVER_HOST = $hostName
    SQL_SERVER_PORT = $port
    SQL_SERVER_USER = $user
    SQL_SERVER_PASSWORD = $password
    SQL_SERVER_DATABASE = $database
    SQL_SERVER_ENCRYPT = $encrypt
    SQL_SERVER_TRUST_CERT = $trustCert
    MCP_SQLSERVER_MAX_ROWS = $maxRows
    MCP_SQLSERVER_ALLOW_SCHEMA_CHANGES = $allowSchema
    MCP_SQLSERVER_ALLOW_DANGEROUS_SQL = $allowDangerous
    MCP_SQLSERVER_ALLOW_PROCEDURE_CALLS = $allowProcedures
}

Write-Host ""
Write-Host "Choose clients to configure:"
Write-Host "1. Codex"
Write-Host "2. Claude Desktop"
Write-Host "3. Gemini CLI"
Write-Host "4. Cursor / VS Code style MCP JSON"
Write-Host "5. All"
$choice = Read-Default "Selection" "5"

$configured = New-Object System.Collections.Generic.List[string]

if ($choice -in @("1", "5")) {
    $path = Join-Path $HOME ".codex\config.toml"
    Write-CodexConfig -Path $path -Env $envVars -Command $binary
    $configured.Add("Codex: $path")
}

if ($choice -in @("2", "5")) {
    $path = Join-Path $env:APPDATA "Claude\claude_desktop_config.json"
    Write-JsonMcpConfig -Path $path -ServerConfig (New-ServerConfig -Env $envVars -Command $binary)
    $configured.Add("Claude Desktop: $path")
}

if ($choice -in @("3", "5")) {
    $path = Join-Path $HOME ".gemini\settings.json"
    Write-JsonMcpConfig -Path $path -ServerConfig (New-ServerConfig -Env $envVars -Command $binary -Trust)
    $configured.Add("Gemini CLI: $path")
}

if ($choice -in @("4", "5")) {
    $path = Join-Path $HOME ".cursor\mcp.json"
    Write-JsonMcpConfig -Path $path -ServerConfig (New-ServerConfig -Env $envVars -Command $binary)
    $configured.Add("Cursor / VS Code style: $path")
}

Write-Host ""
Write-Host "Configured:"
foreach ($item in $configured) {
    Write-Host " - $item"
}

Write-Host ""
Write-Host "Restart the selected client app, then ask it to run the SQL Server MCP health_check tool."
