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

# Define the default directory for logs
$registry_path = "Registry::HKEY_LOCAL_MACHINE\Software\Microsoft\Windows\CurrentVersion\Uninstall\observIQ Distro for OpenTelemetry Collector"

if (Test-Path $registry_path) {
    $collector_dir = (Get-ItemProperty -Path $registry_path -Name "InstallLocation").InstallLocation
} else {
    $collector_dir = "C:\Program Files\observIQ OpenTelemetry Collector"
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

# Determine whether to copy only the most recent log
$response = Read-Host -Prompt "Do you want to include only the most recent logs (Y or n)?  "
if ($response -eq "n") {
    Copy-Item "$collector_dir\log\*" -Destination "$output_dir\" -Force
} else {
    if (Test-Path "$collector_dir\log\observiq_collector.err") {
        Write-Host "Adding $collector_dir\log\observiq_collector.err"
        Copy-Item "$collector_dir\log\observiq_collector.err" -Destination "$output_dir\" -Force
    }
    Write-Host "Adding $collector_dir\log\collector.log"
    Copy-Item "$collector_dir\log\collector.log" -Destination "$output_dir\" -Force
}

# Collector Config
$response = Read-Host -Prompt "Do you want to include the collector config (Y or n)? "

if ($response -ne "n") {
    Write-Host "Adding $collector_dir\config.yaml"
    Copy-Item "$collector_dir\config.yaml" -Destination "$output_dir\" -Force
}

# Capture system info
Get-ComputerInfo | Out-File "$output_dir\systeminfo.txt"

# Capture profiles
$response = Read-Host -Prompt "Collect go pprof profiles? (Y or n)? "

if ($response -ne "n") {
    $pprof_port = Read-Host -Prompt "Enter the pprof port (default 13133): "
    if ([string]::IsNullOrWhiteSpace($pprof_port)) {
        $pprof_port = 13133
    }

    $profiles = @("profile", "block", "goroutine", "heap", "mutex", "threadcreate", "trace")

    foreach ($profile in $profiles) {
        $url = "http://localhost:$pprof_port/debug/pprof/$profile?seconds=30"
        $output_file = "$output_dir\$profile.txt"
        Write-Host "Collecting $profile profile from $url"
        try {
            Invoke-WebRequest -SkipCertificateCheck -Uri $url -OutFile $output_file -UseBasicParsing -ErrorAction Stop
        } catch {
            Write-Host "Failed to collect $profile profile: $_"
        }
    }
}

# Compress the files into a zip archive
$zip_filename = "$output_dir.zip"
Compress-Archive -Path "$output_dir\*" -DestinationPath $zip_filename -Force

# Remove the original output directory
Remove-Item -Path $output_dir -Force -Recurse

# Note the location of the copied files
Write-Host "Files and system info have been copied to $(Resolve-Path $zip_filename)."
