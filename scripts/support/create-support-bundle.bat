:: Copyright  Bindplane, Inc.
::
:: Licensed under the Apache License, Version 2.0 (the "License");
:: you may not use this file except in compliance with the License.
:: You may obtain a copy of the License at
::
::      http://www.apache.org/licenses/LICENSE-2.0
::
:: Unless required by applicable law or agreed to in writing, software
:: distributed under the License is distributed on an "AS IS" BASIS,
:: WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
:: See the License for the specific language governing permissions and
:: limitations under the License.
@echo off
setlocal enabledelayedexpansion

:: Default directory for logs
FOR /F "tokens=2*" %%A IN ('reg query "HKLM\Software\Microsoft\Windows\CurrentVersion\Uninstall\observIQ Distro for OpenTelemetry Collector" /v "InstallLocation"') DO set "collector_dir=%%~B"

:: Check if directory exists, and ask for it if not
if not exist "%collector_dir%" (
    echo observIQ OpenTelemetry Collector directory not found in the registry.
    call :get_dir
)

echo Using directory %collector_dir%
:: Format the date and time for filename
set "datestamp=%DATE:~-4%_%DATE:~4,2%_%DATE:~7,2%"
set "timestamp=%TIME:~0,2%_%TIME:~3,2%_%TIME:~6,2%"
:: replace leading space with zero for hours < 10
set "timestamp=%timestamp: =0%"  

:: Create a new directory to hold the copied files
set "output_dir=Support_Bundle_%datestamp%_%timestamp%"
echo Creating directory %output_dir%

mkdir "%output_dir%"

:: Determine whether to copy only the most recent log
set /p response="Do you want to include only the most recent logs? [Y/n] "
if /I "%response%"=="n" (
    xcopy /Y "%collector_dir%\log\*" "%output_dir%\"
) else (
    if exist "%collector_dir%\log\observiq_collector.err" (
        xcopy /Y "%collector_dir%\log\observiq_collector.err" "%output_dir%\"
    )
    xcopy /Y "%collector_dir%\log\collector.log" "%output_dir%\"
)

:: Add the configuration file
set /p response="Do you want to include the collector config file? [Y/n] "
if /I not "%response%"=="n" (
    echo Copying %collector_dir%\config.yaml
    xcopy /Y "%collector_dir%config.yaml" "%output_dir%\"
)

:: Capture system info
systeminfo > "%output_dir%\systeminfo.txt"

:: Note the location of the copied files
echo Files and system info have been copied to directory %CD%\%output_dir%.

:: End of main script
goto :eof

:: Subroutine to get the directory from the user
:get_dir
set /p collector_dir=Please enter the directory for the observIQ OpenTelemetry Collector installation:
if not exist "%collector_dir%" (
    echo Directory %collector_dir% does not exist.
    exit /b
)
goto :eof
