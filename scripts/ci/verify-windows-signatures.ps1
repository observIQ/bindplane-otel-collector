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
Verifies the Authenticode signatures on the released Windows artifacts.

.DESCRIPTION
Checks that every shipped Windows executable carries a valid signature, not
just the MSI wrapper. Signing the MSI says nothing about its payload, and the
console upgrade path downloads the zip and never touches the MSI, so the
executables inside both containers are what a customer allow-lists by
publisher (BPOP-5780).

For each Windows zip under -Root it verifies observiq-otel-collector.exe and
updater.exe. For each MSI it verifies the MSI itself, then performs an
administrative install to extract the payload and verifies the same two
executables as installed.

Exits non-zero if any artifact fails, so a release cannot be promoted out of
prerelease with unsigned binaries in it.

.PARAMETER Root
Directory holding the downloaded release artifacts.

.EXAMPLE
./verify-windows-signatures.ps1 -Root ./artifacts
#>
param(
  [Parameter(Mandatory = $true)]
  [string]$Root
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$script:failures = @()
$script:checked = 0

# Assert one file carries a valid Authenticode signature. Records a failure
# rather than throwing, so one run reports every bad artifact at once.
function Test-Signature {
  param(
    [string]$Path,
    [string]$Context
  )

  $script:checked++
  $name = "$Context :: $(Split-Path $Path -Leaf)"

  if (-not (Test-Path $Path)) {
    $script:failures += "$name -- file not found"
    Write-Host "  FAIL $name -- file not found"
    return
  }

  $sig = Get-AuthenticodeSignature -LiteralPath $Path

  if ($sig.Status -ne 'Valid') {
    $script:failures += "$name -- $($sig.Status): $($sig.StatusMessage)"
    Write-Host "  FAIL $name -- $($sig.Status): $($sig.StatusMessage)"
    return
  }

  Write-Host "  OK   $name"
  Write-Host "         signer:     $($sig.SignerCertificate.Subject)"
  Write-Host "         thumbprint: $($sig.SignerCertificate.Thumbprint)"
  Write-Host "         expires:    $($sig.SignerCertificate.NotAfter)"
  if ($null -eq $sig.TimeStamperCertificate) {
    Write-Host "         timestamp:  NONE REPORTED"
  }
  else {
    Write-Host "         timestamp:  $($sig.TimeStamperCertificate.Subject)"
  }
}

# The executables that must be signed, wherever they are shipped.
$expectedExes = @('observiq-otel-collector.exe', 'updater.exe')

$rootPath = (Resolve-Path -LiteralPath $Root).Path
$work = Join-Path ([System.IO.Path]::GetTempPath()) "verify-windows-$(Get-Random)"
New-Item -ItemType Directory -Path $work -Force | Out-Null

Write-Host "Verifying Windows artifacts under $rootPath"
Write-Host ""

# --- Windows zips: what the console upgrade path actually downloads ---
$zips = @(Get-ChildItem -Path $rootPath -Recurse -Filter '*-windows-*.zip')
if ($zips.Count -eq 0) {
  $script:failures += 'no Windows zip archives found to verify'
  Write-Host 'FAIL no Windows zip archives found to verify'
}
foreach ($zip in $zips) {
  Write-Host "zip $($zip.Name)"
  $dest = Join-Path $work "zip-$($zip.BaseName)"
  Expand-Archive -LiteralPath $zip.FullName -DestinationPath $dest -Force
  foreach ($exe in $expectedExes) {
    $found = @(Get-ChildItem -Path $dest -Recurse -Filter $exe)
    if ($found.Count -eq 0) {
      $script:failures += "$($zip.Name) :: $exe -- not present in archive"
      Write-Host "  FAIL $($zip.Name) :: $exe -- not present in archive"
      continue
    }
    foreach ($f in $found) {
      Test-Signature -Path $f.FullName -Context $zip.Name
    }
  }
  Write-Host ""
}

# --- MSIs: the wrapper, then its payload as installed ---
$msis = @(Get-ChildItem -Path $rootPath -Recurse -Filter '*.msi')
if ($msis.Count -eq 0) {
  $script:failures += 'no MSI installers found to verify'
  Write-Host 'FAIL no MSI installers found to verify'
}
foreach ($msi in $msis) {
  Write-Host "msi $($msi.Name)"
  Test-Signature -Path $msi.FullName -Context $msi.Name

  # An administrative install extracts the payload without installing the
  # service, which is how the "after install" bytes get verified. It only
  # works for a package built for this machine's architecture, so decide from
  # the package itself rather than from a guessed msiexec exit code: an
  # unexpected failure must stay a failure. The arm64 payload is still covered
  # by the arm64 zip check above.
  if ($msi.Name -match '-arm64\.msi$') {
    Write-Host "  SKIP $($msi.Name) -- arm64 package cannot be extracted on this x64 runner; payload covered by the zip check"
    Write-Host ""
    continue
  }

  $dest = Join-Path $work "msi-$($msi.BaseName)"
  New-Item -ItemType Directory -Path $dest -Force | Out-Null
  $proc = Start-Process -FilePath 'msiexec.exe' `
    -ArgumentList '/a', "`"$($msi.FullName)`"", '/qn', "TARGETDIR=`"$dest`"" `
    -Wait -PassThru
  if ($proc.ExitCode -ne 0) {
    $script:failures += "$($msi.Name) -- administrative install failed with exit code $($proc.ExitCode)"
    Write-Host "  FAIL $($msi.Name) -- administrative install failed with exit code $($proc.ExitCode)"
    Write-Host ""
    continue
  }

  foreach ($exe in $expectedExes) {
    $found = @(Get-ChildItem -Path $dest -Recurse -Filter $exe)
    if ($found.Count -eq 0) {
      $script:failures += "$($msi.Name) :: $exe -- not found in extracted payload"
      Write-Host "  FAIL $($msi.Name) :: $exe -- not found in extracted payload"
      continue
    }
    foreach ($f in $found) {
      Test-Signature -Path $f.FullName -Context "$($msi.Name) (installed)"
    }
  }
  Write-Host ""
}

Remove-Item -LiteralPath $work -Recurse -Force -ErrorAction SilentlyContinue

Write-Host "Checked $script:checked artifact(s)."
if ($script:failures.Count -gt 0) {
  Write-Host ""
  Write-Host "$($script:failures.Count) signature check(s) failed:"
  foreach ($f in $script:failures) {
    Write-Host "  - $f"
  }
  exit 1
}

Write-Host 'All Windows signature checks passed.'
