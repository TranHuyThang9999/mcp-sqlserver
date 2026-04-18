param(
    [string]$Repo = "TranHuyThang9999/mcp-sqlserver",
    [string]$ServerName = "sqlserver",
    [string]$Version = "latest"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Get-GitHubRelease {
    param(
        [string]$Repository,
        [string]$ReleaseVersion
    )

    $headers = @{
        "User-Agent" = "mcp-sqlserver-installer"
        "Accept" = "application/vnd.github+json"
    }

    if ($ReleaseVersion -eq "latest") {
        $uri = "https://api.github.com/repos/$Repository/releases/latest"
    }
    else {
        $uri = "https://api.github.com/repos/$Repository/releases/tags/$ReleaseVersion"
    }

    try {
        return Invoke-RestMethod -Uri $uri -Headers $headers
    }
    catch {
        throw "Cannot find GitHub release '$ReleaseVersion' for $Repository. Publish a release first, or pass -Version vX.Y.Z."
    }
}

function Get-ReleaseAsset {
    param(
        $Release,
        [string]$AssetName
    )

    $asset = $Release.assets | Where-Object { $_.name -eq $AssetName } | Select-Object -First 1
    if ($null -eq $asset) {
        $available = ($Release.assets | ForEach-Object { $_.name }) -join ", "
        throw "Release '$($Release.tag_name)' does not contain '$AssetName'. Available assets: $available"
    }

    return $asset
}

$assetName = "mcp-sqlserver-windows-amd64.zip"
$release = Get-GitHubRelease -Repository $Repo -ReleaseVersion $Version
$asset = Get-ReleaseAsset -Release $release -AssetName $assetName

$localAppData = $env:LOCALAPPDATA
if ([string]::IsNullOrWhiteSpace($localAppData)) {
    $localAppData = Join-Path $HOME "AppData\Local"
}

$installRoot = Join-Path $localAppData "mcp-sqlserver"
$releaseDir = Join-Path (Join-Path $installRoot "releases") $release.tag_name
$workDir = Join-Path $installRoot "downloads"
$zipPath = Join-Path $workDir $assetName

New-Item -ItemType Directory -Force -Path $workDir | Out-Null
New-Item -ItemType Directory -Force -Path $releaseDir | Out-Null

Write-Host ""
Write-Host "Downloading MCP SQL Server $($release.tag_name)"
Write-Host "Repository: $Repo"
Write-Host "Asset: $assetName"
Write-Host "Install directory: $releaseDir"
Write-Host ""

Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $zipPath -Headers @{
    "User-Agent" = "mcp-sqlserver-installer"
}

Expand-Archive -Path $zipPath -DestinationPath $releaseDir -Force

$installer = Get-ChildItem -Path $releaseDir -Recurse -Filter "install-windows.ps1" | Select-Object -First 1
if ($null -eq $installer) {
    throw "Downloaded package does not contain install-windows.ps1"
}

Write-Host ""
Write-Host "Running installer from $($installer.FullName)"
Write-Host ""

& $installer.FullName -ServerName $ServerName
