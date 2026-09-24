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

# Redact secrets from a staged copy in place; originals untouched. Best effort.
# Key list from resource params marked sensitive:true (value/endpoint/DSN
# excluded from whole-value redaction).
function Redact-File {
    param([string]$Path)
    if (-not (Test-Path $Path)) { return }
    $text = Get-Content -Raw -Path $Path
    if ($null -eq $text) { return }
    $keys = 'password|passwd|secret|token|key|creds|credential|honeycomb|authorization|bearer|passphrase|client_id'
    # Block scalar: drop the indented body.
    $blockRe = '(?im)^(?<pre>(?<ind>[ \t]*)[A-Za-z0-9_.-]*(?:' + $keys + ')[A-Za-z0-9_.-]*)[ \t]*:[ \t]*[|>][-+]?\d?[ \t]*\r?\n(?:\k<ind>[ \t]+\S.*(?:\r?\n|$)|[ \t]*\r?\n)*'
    $text = [regex]::Replace($text, $blockRe, ('${pre}: "[REDACTED]"' + "`n"))
    # Require non-empty value so a bare "key:" opener is kept.
    $keyRe = '(?im)^([ \t]*-?[ \t]*[A-Za-z0-9_.-]*(?:' + $keys + ')[A-Za-z0-9_.-]*[ \t]*:[ \t]*)\S.*$'
    $text = [regex]::Replace($text, $keyRe, '$1"[REDACTED]"')
    $text = [regex]::Replace($text, '([A-Za-z][A-Za-z0-9+.-]*://[^:/@\s]+):[^@/\s]+@', '$1:[REDACTED]@')
    $text = [regex]::Replace($text, '([Bb]earer[ \t]+)[A-Za-z0-9._~+/=-]+', '$1[REDACTED]')
    $text = [regex]::Replace($text, 'AKIA[0-9A-Z]{16}', '[REDACTED-AWS-ACCESS-KEY]')
    $text = [regex]::Replace($text, 'eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+', '[REDACTED-JWT]')
    # Mid-line key:value in log/DSN text; value token only (keeps trailing fields).
    $midRe = '(?i)([A-Za-z0-9_.-]*(?:password|passwd|pwd|secret|token|api[_-]?key|access[_-]?key|private[_-]?key|passphrase|authorization|bearer)[A-Za-z0-9_.-]*[ \t]*[:=][ \t]*)[^\s;,"]+'
    $text = [regex]::Replace($text, $midRe, '$1[REDACTED]')
    $text = [regex]::Replace($text, '(?s)-----BEGIN[A-Z ]*PRIVATE KEY-----.*?-----END[A-Z ]*PRIVATE KEY-----', '[REDACTED-PRIVATE-KEY]')
    Set-Content -Path $Path -Value $text -NoNewline
}

# When dot-sourced (e.g. by tests), stop here so only the function loads.
if ($MyInvocation.InvocationName -eq '.') { return }

# Detect the installed collector version by its uninstall registry key
# (dir-independent). Prompt when both or neither are installed, since the
# stopped one may be the one under investigation.
$v1_reg = "Registry::HKEY_LOCAL_MACHINE\Software\Microsoft\Windows\CurrentVersion\Uninstall\observIQ Distro for OpenTelemetry Collector"
$v2_reg = "Registry::HKEY_LOCAL_MACHINE\Software\Microsoft\Windows\CurrentVersion\Uninstall\BindPlane Distro for OpenTelemetry Collector (BDOT)"
$v1_installed = Test-Path $v1_reg
$v2_installed = Test-Path $v2_reg

if ($v1_installed -and $v2_installed) {
    $choice = Read-Host -Prompt "Which collector version to collect? (1 = v1, 2 = v2) "
    $collector_version = if ($choice -eq "2") { "v2" } else { "v1" }
} elseif ($v2_installed) {
    $collector_version = "v2"
} elseif ($v1_installed) {
    $collector_version = "v1"
} else {
    $choice = Read-Host -Prompt "No collector found in the registry. Which version? (1 = v1, 2 = v2) "
    $collector_version = if ($choice -eq "2") { "v2" } else { "v1" }
}

if ($collector_version -eq "v2") {
    $collector_service = "bindplane-otel-collector"
    $registry_path = $v2_reg
} else {
    $collector_service = "observiq-otel-collector"
    $registry_path = $v1_reg
}

# Install dir: registry InstallLocation, else the default (unchanged across
# versions), else prompt.
if (Test-Path $registry_path) {
    $collector_dir = (Get-ItemProperty -Path $registry_path -Name "InstallLocation").InstallLocation
} else {
    $collector_dir = "C:/Program Files/observIQ OpenTelemetry Collector"
    Write-Host "Collector directory not found in the registry. Trying default location: $collector_dir"
}

# Check if the directory exists
if (!(Test-Path $collector_dir)) {
    Write-Host "Directory $collector_dir does not exist."
    $collector_dir = Read-Host -Prompt "Please enter the collector installation directory"
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
if ($collector_version -eq "v2") {
    # v2 logs: supervisor.log in the install dir plus the supervisor_storage logs.
    $v2_logs = @()
    if (Test-Path "$collector_dir/supervisor.log") { $v2_logs += Get-Item "$collector_dir/supervisor.log" }
    if (Test-Path "$collector_dir/supervisor_storage") {
        $v2_logs += Get-ChildItem "$collector_dir/supervisor_storage" -Filter *.log -File -ErrorAction SilentlyContinue
    }
    if ($v2_logs.Count -eq 0) {
        Write-Host "No logs found for the v2 collector in $collector_dir"
    } elseif ($response -eq "n") {
        $v2_logs | ForEach-Object { Copy-Item $_.FullName -Destination "$output_dir/" -Force }
    } else {
        $recent = $v2_logs | Sort-Object LastWriteTime | Select-Object -Last 1
        Write-Host "Adding $($recent.FullName)"
        Copy-Item $recent.FullName -Destination "$output_dir/" -Force
    }
} else {
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
}

# Redact copied logs before bundling.
Get-ChildItem -Path $output_dir -File |
    Where-Object { $_.Name -match '\.(log|err)(\.\d+)?$' } |
    ForEach-Object { Redact-File $_.FullName }

# Collector Config. Artifacts differ by version: v1 ships config.yaml +
# manager.yaml, v2 ships supervisor.yaml + supervisor_storage/effective.yaml.
$response = Read-Host -Prompt "Do you want to include the collector config and manager files (Y or n)? "

if ($response -ne "n") {
    if ($collector_version -eq "v2") {
        $config_files = @("$collector_dir/supervisor.yaml", "$collector_dir/supervisor_storage/effective.yaml")
    } else {
        $config_files = @("$collector_dir/config.yaml", "$collector_dir/manager.yaml")
    }
    foreach ($config_file in $config_files) {
        if (Test-Path $config_file) {
            $name = Split-Path $config_file -Leaf
            Write-Host "Adding $config_file (redacted)"
            Copy-Item $config_file -Destination "$output_dir/$name" -Force
            Redact-File "$output_dir/$name"
        }
    }
}

# Capture system info
Get-ComputerInfo | Out-File "$output_dir/systeminfo.txt"

# Live CPU, memory, and disk stats. Always collected (cheap, non-sensitive).
$statsFile = "$output_dir/system_stats.txt"
"=== cpu load (%) ===" | Out-File $statsFile
try {
    (Get-CimInstance Win32_Processor -ErrorAction Stop |
        Measure-Object -Property LoadPercentage -Average).Average | Out-File -Append $statsFile
} catch { "CPU load unavailable: $($_.Exception.Message)" | Out-File -Append $statsFile }
"=== memory (KB) ===" | Out-File -Append $statsFile
try {
    Get-CimInstance Win32_OperatingSystem -ErrorAction Stop |
        Select-Object TotalVisibleMemorySize, FreePhysicalMemory |
        Format-List | Out-File -Append $statsFile
} catch { "Memory stats unavailable: $($_.Exception.Message)" | Out-File -Append $statsFile }
"=== disk ===" | Out-File -Append $statsFile
try {
    Get-Volume -ErrorAction Stop |
        Select-Object DriveLetter, FileSystemLabel,
            @{n='SizeGB';e={[math]::Round($_.Size/1GB,2)}},
            @{n='FreeGB';e={[math]::Round($_.SizeRemaining/1GB,2)}} |
        Format-Table -AutoSize | Out-File -Append $statsFile
} catch { "Disk stats unavailable: $($_.Exception.Message)" | Out-File -Append $statsFile }

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

# Open file handles. Windows has no nofile-style cap, so exhaustion is bounded
# by paged/nonpaged pool; collect the counts and, best effort, the handle list.
$response = Read-Host -Prompt "Collect open file handles? (Y or n)? "

if ($response -ne "n") {
    $svc = Get-CimInstance Win32_Service -Filter "Name='$collector_service'" -ErrorAction SilentlyContinue
    $collectorPid = if ($svc -and $svc.ProcessId) { $svc.ProcessId } else { (Get-Process -Name $collector_service -ErrorAction SilentlyContinue).Id }

    if ($collectorPid) {
        Get-Process -Id $collectorPid -ErrorAction SilentlyContinue |
            Select-Object Id, ProcessName, Handles |
            Format-List | Out-File "$output_dir/handle_count.txt"
    } else {
        "Collector process not found." | Out-File "$output_dir/handle_count.txt"
    }

    # System-wide handle count from the process list (no perf counters needed).
    $sysHandles = (Get-Process -ErrorAction SilentlyContinue | Measure-Object Handles -Sum).Sum
    "System-wide handle count: $sysHandles" | Out-File "$output_dir/system_handles.txt"
    # Paged/nonpaged pool bytes, best effort (needs the perf counter subsystem).
    try {
        Get-Counter '\Memory\Pool Paged Bytes', '\Memory\Pool Nonpaged Bytes' -ErrorAction Stop |
            ForEach-Object { $_.CounterSamples } |
            Select-Object Path, CookedValue |
            Format-List | Out-File -Append "$output_dir/system_handles.txt"
    } catch {
        "Paged/nonpaged pool bytes unavailable (Get-Counter: $($_.Exception.Message))" |
            Out-File -Append "$output_dir/system_handles.txt"
    }
    "Windows has no configurable file-handle limit; exhaustion is bounded by paged/nonpaged pool." |
        Out-File -Append "$output_dir/system_handles.txt"

    $handleExe = Get-Command handle64.exe, handle.exe -ErrorAction SilentlyContinue | Select-Object -First 1
    if (-not $handleExe) {
        $dl = Read-Host -Prompt "handle.exe not found. Download it from Sysinternals? (Y or n)? "
        if ($dl -ne "n") {
            try {
                $zip = Join-Path $env:TEMP "Handle.zip"
                $dest = Join-Path $env:TEMP "Handle"
                Invoke-WebRequest -Uri "https://download.sysinternals.com/files/Handle.zip" -OutFile $zip -UseBasicParsing -TimeoutSec 300 -ErrorAction Stop
                Expand-Archive -Path $zip -DestinationPath $dest -Force
                $handleExe = Get-Command (Join-Path $dest "handle64.exe"), (Join-Path $dest "handle.exe") -ErrorAction SilentlyContinue | Select-Object -First 1
            } catch {
                Write-Host "handle.exe download failed. Download it manually from https://learn.microsoft.com/sysinternals/downloads/handle and re-run this script."
            }
        }
    }
    if ($handleExe) {
        # Verify the Authenticode signature before executing a fetched binary on a
        # production host. Sysinternals tools are Microsoft-signed.
        $sig = Get-AuthenticodeSignature $handleExe.Source
        if ($sig.Status -ne 'Valid' -or $sig.SignerCertificate.Subject -notmatch 'Microsoft') {
            Write-Host "Skipping handle.exe: Authenticode signature not valid or not Microsoft-signed (status: $($sig.Status))."
        } elseif ($collectorPid) {
            & $handleExe.Source -accepteula -p $collectorPid 2>&1 | Out-File "$output_dir/open_handles.txt"
        } else {
            & $handleExe.Source -accepteula 2>&1 | Out-File "$output_dir/open_handles.txt"
        }
    }
}

# Compress the files into a zip archive
$zip_filename = "$output_dir.zip"
Compress-Archive -Path "$output_dir/*" -DestinationPath $zip_filename -Force

# Remove the original output directory
Remove-Item -Path $output_dir -Force -Recurse

# Note the location of the copied files
Write-Host "Files and system info have been copied to $(Resolve-Path $zip_filename)."
