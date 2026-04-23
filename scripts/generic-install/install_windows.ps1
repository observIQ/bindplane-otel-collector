# Copyright observIQ, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

<#
.SYNOPSIS
    Installs or uninstalls an OTel Distro Builder distribution on Windows.

.DESCRIPTION
    Downloads the appropriate MSI (amd64 or arm64) for the current machine
    architecture and hands it to msiexec. The MSI is resolved from a GitHub
    releases page unless overridden. Pass -Uninstall to run msiexec /x against
    the same MSI; the MSI must still be resolvable via -Url/-Version, -MsiUrl,
    or -MsiFile, and must match the installed version.

.PARAMETER Distribution
    Name of the distribution (e.g. "mydistribution"). Used to construct the MSI
    filename and the install/uninstall log filename.

.PARAMETER Url
    GitHub repository URL (e.g. "https://github.com/org/repo"). Required unless
    -MsiUrl or -MsiFile is provided.

.PARAMETER Version
    The version to install (e.g. "1.2.3"). Omit or pass "latest" to install the
    latest release.

.PARAMETER Endpoint
    OpAMP server endpoint URL (e.g. "wss://app.bindplane.com/v1/opamp").
    When provided, managed mode is enabled automatically.

.PARAMETER SecretKey
    Secret key for OpAMP authentication.

.PARAMETER Labels
    Comma-separated key=value labels for OpAMP (e.g. "env=prod,region=us-west").

.PARAMETER InstallDir
    Custom installation directory. Defaults to the MSI default.

.PARAMETER MsiUrl
    Override the full MSI download URL. If set, -Version and arch detection are
    ignored.

.PARAMETER MsiFile
    Path to a local MSI file. Skips all download and version resolution steps.

.PARAMETER Interactive
    Show the installer UI instead of running silently.

.PARAMETER Uninstall
    Uninstall the distribution instead of installing it. Resolves the MSI the
    same way as install (-Url/-Version, -MsiUrl, or -MsiFile) and runs
    msiexec /x against it.

.PARAMETER SkipSignatureCheck
    Skip MSI Authenticode signature verification.

.EXAMPLE
    .\install_windows.ps1 -Distribution "mydistribution" `
        -Url "https://github.com/org/repo" `
        -Endpoint "wss://app.bindplane.com/v1/opamp" `
        -SecretKey "my-secret"

.EXAMPLE
    .\install_windows.ps1 -Distribution "mydistribution" `
        -Url "https://github.com/org/repo" `
        -Version "1.2.3"

.EXAMPLE
    .\install_windows.ps1 -Distribution "mydistribution" `
        -Url "https://github.com/org/repo" `
        -Version "1.2.3" -Uninstall
#>

[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$Distribution,

    [Parameter(Mandatory = $false)]
    [string]$Url,

    [Parameter(Mandatory = $false)]
    [string]$Version,

    [Parameter(Mandatory = $false)]
    [string]$Endpoint,

    [Parameter(Mandatory = $false)]
    [string]$SecretKey,

    [Parameter(Mandatory = $false)]
    [string]$Labels,

    [Parameter(Mandatory = $false)]
    [string]$InstallDir,

    [Parameter(Mandatory = $false)]
    [string]$MsiUrl,

    [Parameter(Mandatory = $false)]
    [string]$MsiFile,

    [Parameter(Mandatory = $false)]
    [switch]$Interactive,

    [Parameter(Mandatory = $false)]
    [switch]$Uninstall,

    [Parameter(Mandatory = $false)]
    [switch]$SkipSignatureCheck
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# ---- Helpers -----------------------------------------------------------------

function Write-Info {
    param([string]$Message)
    Write-Host "[INFO]  $Message"
}

function Write-Warn {
    param([string]$Message)
    Write-Host "[WARN]  $Message" -ForegroundColor Yellow
}

function Fail {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
    exit 1
}

# ---- Privilege check ---------------------------------------------------------

function Assert-Administrator {
    $principal = [Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()
    if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
        Fail "This script must be run as Administrator. Re-run from an elevated PowerShell prompt."
    }
}

# ---- Architecture detection --------------------------------------------------

function Get-Arch {
    # PROCESSOR_ARCHITEW6432 is set when running a 32-bit process on a 64-bit OS (WOW64).
    # Fall back to PROCESSOR_ARCHITECTURE when not in WOW64.
    $arch = if ($env:PROCESSOR_ARCHITEW6432) { $env:PROCESSOR_ARCHITEW6432 } else { $env:PROCESSOR_ARCHITECTURE }
    switch ($arch) {
        "AMD64" { return "amd64" }
        "ARM64" { return "arm64" }
        default { Fail "Unsupported architecture: $arch. Only x64 (amd64) and ARM64 are supported." }
    }
}

# ---- Version resolution ------------------------------------------------------

function Get-LatestVersion {
    param([string]$RepositoryUrl)
    # GitHub redirects /releases/latest to /releases/tag/v{version}.
    # We follow the redirect with AllowAutoRedirect = $false to extract the version.
    try {
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
        $req = [System.Net.HttpWebRequest]::Create("$RepositoryUrl/releases/latest")
        $req.AllowAutoRedirect = $false
        $req.Method = "HEAD"
        $resp = $req.GetResponse()
        try {
            $location = $resp.Headers["Location"]
        }
        finally {
            $resp.Close()
        }
        if (-not $location) {
            Fail "No redirect received from $RepositoryUrl/releases/latest; cannot determine latest version."
        }
        return ($location -split '/')[-1].TrimStart('v')
    }
    catch {
        Fail "Failed to retrieve latest version from ${RepositoryUrl}/releases/latest: $_"
    }
}

# ---- Download ----------------------------------------------------------------

function Get-Msi {
    param(
        [string]$Url,
        [string]$Destination
    )

    Write-Info "Downloading MSI from $Url"
    try {
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
        $wc = New-Object System.Net.WebClient
        $wc.DownloadFile($Url, $Destination)
    }
    catch {
        Fail "Failed to download MSI: $_"
    }
    Write-Info "Download complete: $Destination"
}

# ---- Build msiexec argument list ---------------------------------------------

function Build-MsiexecArgs {
    param(
        [string]$MsiPath,
        [string]$LogPath
    )

    $msiArgs = @("/i", "`"$MsiPath`"", "/l*v", "`"$LogPath`"")

    if (-not $Interactive) {
        $msiArgs += "/quiet"
    }

    # When an endpoint is provided, enable managed mode automatically.
    if ($Endpoint) {
        $msiArgs += "ENABLEMANAGEMENT=`"1`""
        $msiArgs += "OPAMPENDPOINT=`"$Endpoint`""
    }

    if ($SecretKey) {
        $msiArgs += "OPAMPSECRETKEY=`"$SecretKey`""
    }

    if ($Labels) {
        $msiArgs += "OPAMPLABELS=`"$Labels`""
    }

    if ($InstallDir) {
        $msiArgs += "INSTALLDIR=`"$InstallDir`""
    }

    return $msiArgs
}

# ---- Main --------------------------------------------------------------------

function Main {
    Assert-Administrator

    if (-not $MsiFile -and -not $MsiUrl -and -not $Url) {
        Fail "Either -Url, -MsiUrl, or -MsiFile must be provided."
    }

    $tmpDir = [System.IO.Path]::GetTempPath()
    $action = if ($Uninstall) { "uninstall" } else { "install" }
    $logPath = Join-Path $tmpDir "$Distribution-$action.log"
    $cleanupMsi = $false

    if ($MsiFile) {
        if (-not (Test-Path -Path $MsiFile -PathType Leaf)) {
            Fail "MSI file not found: $MsiFile"
        }
        $msiPath = $MsiFile
        Write-Info "Using local MSI: $msiPath"
    }
    else {
        if ($MsiUrl) {
            $resolvedUrl = $MsiUrl
            $msiFileName = Split-Path $MsiUrl -Leaf
        }
        else {
            $arch = Get-Arch
            if (-not $Version -or $Version -eq "latest") {
                $resolvedVersion = Get-LatestVersion -RepositoryUrl $Url
                Write-Info "Latest version: $resolvedVersion"
            }
            else {
                $resolvedVersion = $Version.TrimStart('v')
            }
            if ($arch -eq "arm64") {
                $msiFileName = "${Distribution}-arm64.msi"
            }
            else {
                $msiFileName = "${Distribution}.msi"
            }
            $resolvedUrl = "$($Url.TrimEnd('/'))/releases/download/v${resolvedVersion}/${msiFileName}"
        }

        $msiPath = Join-Path $tmpDir $msiFileName
        $cleanupMsi = $true
        Get-Msi -Url $resolvedUrl -Destination $msiPath
    }

    if (-not $SkipSignatureCheck) {
        $sig = Get-AuthenticodeSignature -FilePath $msiPath
        if ($sig.Status -ne 'Valid') {
            Fail "MSI signature verification failed: $($sig.StatusMessage)"
        }
        Write-Info "MSI signature verification successful."
    }

    if ($Uninstall) {
        $msiArgs = @("/x", "`"$msiPath`"", "/l*v", "`"$logPath`"")
        if (-not $Interactive) {
            $msiArgs += "/quiet"
        }
    }
    else {
        $msiArgs = Build-MsiexecArgs -MsiPath $msiPath -LogPath $logPath
    }

    Write-Info "Running: msiexec $($msiArgs -join ' ')"

    $proc = Start-Process -FilePath "msiexec.exe" -ArgumentList $msiArgs -Wait -PassThru
    $exitCode = $proc.ExitCode

    if ($cleanupMsi) {
        try {
            Remove-Item -Path $msiPath -Force
        }
        catch {
            Write-Warn "Failed to remove temporary MSI: $_"
        }
    }

    if ($Uninstall) {
        switch ($exitCode) {
            0    { Write-Info "Uninstallation completed successfully." }
            1605 { Fail "Product is not installed (msiexec 1605). See the log for details: $logPath" }
            3010 { Write-Warn "Uninstallation succeeded. A reboot is required to complete removal." }
            default { Fail "msiexec exited with code $exitCode. See the log for details: $logPath" }
        }
    }
    else {
        switch ($exitCode) {
            0    { Write-Info "Installation completed successfully." }
            1603 { Fail "Installation failed. See the install log for details: $logPath" }
            1638 { Write-Info "$Distribution is already installed at this version. No changes made." }
            3010 { Write-Warn "Installation succeeded. A reboot is required to complete setup." }
            default { Fail "msiexec exited with code $exitCode. See the install log for details: $logPath" }
        }
    }
}

Main
