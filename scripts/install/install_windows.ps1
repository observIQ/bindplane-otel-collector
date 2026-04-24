# Copyright  observIQ, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

<#
.SYNOPSIS
    Installs or uninstalls the observIQ Distro for OpenTelemetry Collector on Windows.

.DESCRIPTION
    Downloads and installs the appropriate MSI (amd64 or arm64) for the current
    machine architecture. Accepts all standard MSI installer properties.
    Pass -Uninstall to remove an existing installation.

.PARAMETER Version
    The version to install (e.g. "1.94.0"). Omit or pass "latest" to install the latest release.

.PARAMETER OpAMPEndpoint
    OpAMP server endpoint URL (e.g. "wss://app.bindplane.com/v1/opamp").

.PARAMETER OpAMPSecretKey
    Secret key for OpAMP authentication.

.PARAMETER OpAMPLabels
    Comma-separated key=value labels for OpAMP (e.g. "configuration=windows,env=prod").

.PARAMETER EnableManagement
    Set to "1" to enable managed mode via OpAMP. Default is "0".

.PARAMETER InstallDir
    Custom installation directory. Defaults to the MSI default.

.PARAMETER Clean
    Set to "1" to remove existing configuration on install. Default is "0".

.PARAMETER Quiet
    Run the installer silently instead of showing the installer UI.

.PARAMETER Uninstall
    Uninstall the agent instead of installing it.

.PARAMETER MsiUrl
    Override the full MSI download URL. If set, Version and arch detection are ignored.

.PARAMETER MsiFile
    Path to a local MSI file to install. Skips all download and version resolution steps.

.PARAMETER SkipSignatureCheck
    Skip MSI Authenticode signature verification.

.EXAMPLE
    .\install_windows.ps1 -Version "1.94.0" -EnableManagement "1" `
        -OpAMPEndpoint "<your_endpoint>" `
        -OpAMPSecretKey "<secret-key>"

.EXAMPLE
    .\install_windows.ps1 -Version "1.94.0" -Quiet

.EXAMPLE
    .\install_windows.ps1 -Uninstall
#>

[CmdletBinding()]
param(
    [Parameter(Mandatory = $false)]
    [string]$Version,

    [Parameter(Mandatory = $false)]
    [string]$OpAMPEndpoint,

    [Parameter(Mandatory = $false)]
    [string]$OpAMPSecretKey,

    [Parameter(Mandatory = $false)]
    [string]$OpAMPLabels,

    [Parameter(Mandatory = $false)]
    [ValidateSet("0", "1")]
    [string]$EnableManagement = "0",

    [Parameter(Mandatory = $false)]
    [string]$InstallDir,

    [Parameter(Mandatory = $false)]
    [ValidateSet("0", "1")]
    [string]$Clean = "0",

    [Parameter(Mandatory = $false)]
    [string]$MsiUrl,

    [Parameter(Mandatory = $false)]
    [string]$MsiFile,

    [Parameter(Mandatory = $false)]
    [switch]$Quiet,

    [Parameter(Mandatory = $false)]
    [switch]$Uninstall,

    [Parameter(Mandatory = $false)]
    [switch]$SkipSignatureCheck
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# ---- Constants ---------------------------------------------------------------

$DOWNLOAD_BASE = "https://bdot.bindplane.com"
$MSI_NAME_AMD64 = "observiq-otel-collector.msi"
$MSI_NAME_ARM64 = "observiq-otel-collector-arm64.msi"
$PRODUCT_DISPLAY_NAME = "observIQ Distro for OpenTelemetry Collector"

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

function Get-MsiName {
    # PROCESSOR_ARCHITEW6432 is set when running a 32-bit process on a 64-bit OS (WOW64).
    # Fall back to PROCESSOR_ARCHITECTURE when not in WOW64.
    $arch = if ($env:PROCESSOR_ARCHITEW6432) { $env:PROCESSOR_ARCHITEW6432 } else { $env:PROCESSOR_ARCHITECTURE }
    switch ($arch) {
        "AMD64" { return $MSI_NAME_AMD64 }
        "ARM64" { return $MSI_NAME_ARM64 }
        default { Fail "Unsupported architecture: $arch. Only x64 (amd64) and ARM64 are supported." }
    }
}

# ---- Version resolution ------------------------------------------------------

function Get-LatestVersion {
    try {
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
        $wc = New-Object System.Net.WebClient
        $version = $wc.DownloadString("https://bdot.bindplane.com/latest")
        return $version.Trim()
    }
    catch {
        Fail "Failed to retrieve latest version from https://bdot.bindplane.com/latest: $_"
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

    # /i  = install
    $msiArgs = @("/i", "`"$MsiPath`"")

    $msiArgs += "/l*v", "`"$LogPath`""

    if ($Quiet) {
        $msiArgs += "/quiet"
    }

    if ($EnableManagement -eq "1") {
        $msiArgs += "ENABLEMANAGEMENT=`"1`""
    }

    if ($OpAMPEndpoint) {
        $msiArgs += "OPAMPENDPOINT=`"$OpAMPEndpoint`""
    }

    if ($OpAMPSecretKey) {
        $msiArgs += "OPAMPSECRETKEY=`"$OpAMPSecretKey`""
    }

    if ($OpAMPLabels) {
        $msiArgs += "OPAMPLABELS=`"$OpAMPLabels`""
    }

    if ($InstallDir) {
        $msiArgs += "INSTALLDIR=`"$InstallDir`""
    }

    if ($Clean -eq "1") {
        $msiArgs += "CLEAN=`"1`""
    }

    return $msiArgs
}

# ---- Uninstall ---------------------------------------------------------------

function Get-ProductCode {
    $product = Get-CimInstance -ClassName Win32_Product `
        -Filter "Name = '$PRODUCT_DISPLAY_NAME'" `
        -ErrorAction SilentlyContinue |
        Select-Object -First 1
    if ($product) {
        return $product.IdentifyingNumber
    }
    return $null
}

function Invoke-Uninstall {
    Write-Info "Searching for installed '$PRODUCT_DISPLAY_NAME'..."

    $productCode = Get-ProductCode
    if (-not $productCode) {
        Fail "'$PRODUCT_DISPLAY_NAME' is not installed."
    }

    Write-Info "Found product code: $productCode"

    $msiArgs = @("/x", $productCode)
    if ($Quiet) {
        $msiArgs += "/quiet"
    }

    Write-Info "Running: msiexec $($msiArgs -join ' ')"

    $proc = Start-Process -FilePath "msiexec.exe" -ArgumentList $msiArgs -Wait -PassThru
    $exitCode = $proc.ExitCode

    switch ($exitCode) {
        0    { Write-Info "Uninstallation completed successfully." }
        3010 { Write-Warn "Uninstallation succeeded. A reboot is required to complete removal." }
        default { Fail "msiexec exited with code $exitCode. See Windows Event Log for details." }
    }
}

# ---- Main --------------------------------------------------------------------

function Main {
    Assert-Administrator

    if ($Uninstall) {
        Invoke-Uninstall
        return
    }

    $tmpDir = [System.IO.Path]::GetTempPath()
    $logPath = Join-Path $tmpDir "observiq-otel-collector-install.log"
    $cleanupMsi = $false

    if ($MsiFile) {
        # Use the locally provided MSI file directly
        $msiPath = $MsiFile
        if (-not (Test-Path -Path $msiPath -PathType Leaf)) {
            Fail "MSI file not found: $msiPath"
        }
        Write-Info "Using local MSI: $msiPath"
    }
    else {
        # Resolve the MSI URL
        if ($MsiUrl) {
            $resolvedUrl = $MsiUrl
            $msiFileName = Split-Path $MsiUrl -Leaf
        }
        else {
            $msiFileName = Get-MsiName
            if (-not $Version -or $Version -eq "latest") {
                $resolvedVersion = Get-LatestVersion
                if (-not $resolvedVersion) {
                    Fail "Could not determine latest version to install."
                }
                Write-Info "Latest version: $resolvedVersion"
                $resolvedUrl = "$DOWNLOAD_BASE/v$($resolvedVersion.TrimStart('v'))/$msiFileName"
            }
            else {
                $resolvedUrl = "$DOWNLOAD_BASE/v$($Version.TrimStart('v'))/$msiFileName"
            }
        }

        $msiPath = Join-Path $tmpDir $msiFileName
        $cleanupMsi = $true

        Get-Msi -Url $resolvedUrl -Destination $msiPath
    }

    if ($SkipSignatureCheck) {
        Write-Warn "Authenticode signature verification is being bypassed with the '-SkipSignatureCheck' flag."
        Write-Warn "This disables a critical security check and should only be used if your organization policies permit it."
    }
    else {
        $sig = Get-AuthenticodeSignature -FilePath $msiPath
        if ($sig.Status -ne 'Valid') {
            $sigMessage = "MSI signature verification failed: $($sig.StatusMessage)"
            if ($Quiet) {
                Fail "Failed to verify MSI signature: $($sig.StatusMessage). Use '-SkipSignatureCheck' to skip verification."
            }
            Write-Warn $sigMessage
            Write-Warn "This may indicate: the signing certificate is not trusted on this machine, the MSI has been tampered with, the signature or certificate has expired or been revoked, or the certificate chain could not be verified (e.g., network issues reaching CRL/OCSP)."
            Write-Warn "Continuing is NOT RECOMMENDED."
            $response = Read-Host "Continue with unverified MSI anyway? [y/N]"
            if ($response -notmatch '^[yY]') {
                Fail "Aborted by user."
            }
            Write-Warn "Continuing with unverified MSI at user's request."
        }
        else {
            Write-Info "MSI signature verification successful"
        }
    }

    $msiArgs = Build-MsiexecArgs -MsiPath $msiPath -LogPath $logPath

    Write-Info "Running: msiexec $($msiArgs -join ' ')"

    $proc = Start-Process -FilePath "msiexec.exe" -ArgumentList $msiArgs -Wait -PassThru
    $exitCode = $proc.ExitCode

    # Clean up downloaded MSI (not user-provided files)
    if ($cleanupMsi) {
        try {
            Remove-Item -Path $msiPath -Force
        } catch {
            Write-Warn "Failed to remove MSI: $_"
        }
    }

    switch ($exitCode) {
        0    { Write-Info "Installation completed successfully." }
        1603 { Fail "Installation failed. If a newer version is already installed, use -Uninstall first. See the install log for details: $logPath" }
        1638 { Write-Info "Another version of $PRODUCT_DISPLAY_NAME is already installed. No changes made." }
        3010 { Write-Warn "Installation succeeded. A reboot is required to complete the setup." }
        default { Fail "msiexec exited with code $exitCode. See the install log for details: $logPath" }
    }
}

Main
