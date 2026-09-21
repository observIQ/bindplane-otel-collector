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

# Redact secrets from a staged bundle file in place (originals untouched).
# Pass 1: YAML "<sensitive-key>: value" -> [REDACTED]. Pass 2: secret-shaped
# values anywhere (URL creds, Bearer, AWS key, JWT, PEM), which covers logs.
# Line-based, so multi-line YAML block scalars are not covered except PEM.
function Redact-File {
    param([string]$Path)
    if (-not (Test-Path $Path)) { return }
    $text = Get-Content -Raw -Path $Path
    if ($null -eq $text) { return }
    $keyRe = '(?im)^([ \t]*-?[ \t]*[A-Za-z0-9_.-]*(password|passwd|secret|token|key|creds|honeycomb|api[_-]?key|access[_-]?key|private[_-]?key|encryption[_-]?key|credential|passphrase|authorization|bearer|connection[_-]?string|account[_-]?key)[A-Za-z0-9_.-]*[ \t]*:[ \t]*).*$'
    $text = [regex]::Replace($text, $keyRe, '$1"[REDACTED]"')
    $text = [regex]::Replace($text, '([A-Za-z][A-Za-z0-9+.-]*://[^:/@\s]+):[^@/\s]+@', '$1:[REDACTED]@')
    $text = [regex]::Replace($text, '([Bb]earer[ \t]+)[A-Za-z0-9._~+/=-]+', '$1[REDACTED]')
    $text = [regex]::Replace($text, 'AKIA[0-9A-Z]{16}', '[REDACTED-AWS-ACCESS-KEY]')
    $text = [regex]::Replace($text, 'eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+', '[REDACTED-JWT]')
    # Mid-line "<sensitive-key>: value" in log text, redacted to end of line.
    $midRe = '(?i)([A-Za-z0-9_.-]*(password|passwd|secret|token|api[_-]?key|access[_-]?key|private[_-]?key|passphrase|authorization|bearer)[A-Za-z0-9_.-]*[ \t]*[:=][ \t]*).*'
    $text = [regex]::Replace($text, $midRe, '$1"[REDACTED]"')
    $text = [regex]::Replace($text, '(?s)-----BEGIN[A-Z ]*PRIVATE KEY-----.*?-----END[A-Z ]*PRIVATE KEY-----', '[REDACTED-PRIVATE-KEY]')
    Set-Content -Path $Path -Value $text -NoNewline
}

# When dot-sourced (e.g. by tests), stop here so only the function loads.
if ($MyInvocation.InvocationName -eq '.') { return }

# Define the default directory for logs
$registry_path = "Registry::HKEY_LOCAL_MACHINE\Software\Microsoft\Windows\CurrentVersion\Uninstall\observIQ Distro for OpenTelemetry Collector"

if (Test-Path $registry_path) {
    $collector_dir = (Get-ItemProperty -Path $registry_path -Name "InstallLocation").InstallLocation
} else {
    $collector_dir = "C:/Program Files/observIQ OpenTelemetry Collector"
    Write-Host "observIQ OpenTelemetry Collector directory not found in the registry. Trying default location: $collector_dir"
}

# Check if the directory exists
if (!(Test-Path $collector_dir)) {
    Write-Host "Directory $collector_dir does not exist."
    $collector_dir = Read-Host -Prompt "Please enter the directory for the observIQ OpenTelemetry Collector installation"
    if (!(Test-Path $collector_dir)) {
        Write-Host "Directory $collector_dir does not exist."
        exit
    }
}

# Create a timestamp for output directory name
$datestamp = Get-Date -Format "yyyy_MM_dd_HH_mm_ss"

# Create a new directory to hold the copied files
$output_dir = "Support_Bundle_$datestamp"
New-Item -ItemType Directory -Force -Path $output_dir

# Grab the collector VERSION.txt file
if (Test-Path "$collector_dir/VERSION.txt") {
    Write-Host "Adding $collector_dir/VERSION.txt"
    Copy-Item "$collector_dir/VERSION.txt" -Destination "$output_dir/" -Force
}

# Determine whether to copy only the most recent log
$response = Read-Host -Prompt "Do you want to include only the most recent logs (Y or n)?  "
if ($response -eq "n") {
    Copy-Item "$collector_dir/log/*" -Destination "$output_dir/" -Force
} else {
    if (Test-Path "$collector_dir/log/observiq_collector.err") {
        Write-Host "Adding $collector_dir/log/observiq_collector.err"
        Copy-Item "$collector_dir/log/observiq_collector.err" -Destination "$output_dir/" -Force
    }
    if (Test-Path "$collector_dir/log/observiq_collector.err.1") {
        Write-Host "Adding $collector_dir/log/observiq_collector.err.1"
        Copy-Item "$collector_dir/log/observiq_collector.err.1" -Destination "$output_dir/" -Force
    }
    Write-Host "Adding $collector_dir/log/collector.log"
    Copy-Item "$collector_dir/log/collector.log" -Destination "$output_dir/" -Force
}

# Redact any logs copied into the output directory before they are bundled.
Get-ChildItem -Path $output_dir -File |
    Where-Object { $_.Name -match '\.(log|err)(\.\d+)?$' } |
    ForEach-Object { Redact-File $_.FullName }

# Collector Config
$response = Read-Host -Prompt "Do you want to include the collector config (Y or n)? "

if ($response -ne "n") {
    # Stage a copy and redact it before bundling so the on-disk originals are
    # never modified.
    if (Test-Path "$collector_dir/config.yaml") {
        Write-Host "Adding $collector_dir/config.yaml (redacted)"
        Copy-Item "$collector_dir/config.yaml" -Destination "$output_dir/" -Force
        Redact-File "$output_dir/config.yaml"
    }
    if (Test-Path "$collector_dir/manager.yaml") {
        Write-Host "Adding $collector_dir/manager.yaml (redacted)"
        Copy-Item "$collector_dir/manager.yaml" -Destination "$output_dir/" -Force
        Redact-File "$output_dir/manager.yaml"
    }
}

# Capture system info
Get-ComputerInfo | Out-File "$output_dir/systeminfo.txt"

# Capture profiles
$response = Read-Host -Prompt "Collect go pprof profiles [requires PowerShell 6.0.0 or greater]? (Y or n)? "

if ($response -ne "n") {
    if ($PSVersionTable.PSVersion.Major -lt 6) {
        Write-Host "PowerShell 6.0.0 or greater is required to collect pprof profiles. Aborting pprof collection."
        exit
    }
    $pprof_port = Read-Host -Prompt "Enter the pprof port (default 1777): "
    if ([string]::IsNullOrWhiteSpace($pprof_port)) {
        $pprof_port = 1777
    }

    $profiles = @("profile", "block", "goroutine", "heap", "mutex", "threadcreate", "trace")

    # Launch every profile request concurrently as a background job. profile
    # (CPU) and trace share the same 30s window so go tool trace can attribute
    # CPU stacks to trace spans.
    $jobs = @()
    foreach ($profile in $profiles) {
        $url = "http://localhost:$pprof_port/debug/pprof/$($profile)?seconds=30"
        switch ($profile) {
            "profile" { $output_file = "$output_dir/cpu.pprof" }
            "trace"   { $output_file = "$output_dir/trace.out" }
            default   { $output_file = "$output_dir/$profile.pprof" }
        }
        Write-Host "Collecting $profile profile from $url"
        $jobs += Start-Job -Name $profile -ScriptBlock {
            param($url, $output_file)
            Invoke-WebRequest -SkipCertificateCheck -Uri $url -OutFile $output_file -UseBasicParsing -ErrorAction Stop
        } -ArgumentList $url, $output_file
    }

    # Wait for all jobs, then report each failure by name.
    $jobs | Wait-Job | Out-Null
    foreach ($job in $jobs) {
        if ($job.State -eq "Failed") {
            $reason = $job.ChildJobs[0].JobStateInfo.Reason.Message
            Write-Host "Failed to collect $($job.Name) profile: $reason"
        }
        Receive-Job -Job $job -ErrorAction SilentlyContinue | Out-Null
        Remove-Job -Job $job
    }
}

# Compress the files into a zip archive
$zip_filename = "$output_dir.zip"
Compress-Archive -Path "$output_dir/*" -DestinationPath $zip_filename -Force

# Remove the original output directory
Remove-Item -Path $output_dir -Force -Recurse

# Note the location of the copied files
Write-Host "Files and system info have been copied to $(Resolve-Path $zip_filename)."
