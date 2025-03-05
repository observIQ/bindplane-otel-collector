// Copyright  Bindplane, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package os handles grabbing OS info from the system
package os

import (
	"fmt"
	"net"
	"runtime"
	"strings"

	"github.com/shirou/gopsutil/v3/host"
)

const unknownValue = "unknown"

// hostInfo is a singleton for collecting host info
var hostInfo *host.InfoStat

// Hostname retrieves the hostname of the machine
func Hostname() (string, error) {
	info, err := getHostInfo()
	if err != nil {
		return unknownValue, err
	}

	return info.Hostname, nil
}

// Name returns the name of the os.
func Name() (string, error) {
	info, err := getHostInfo()
	if err != nil {
		return unknownValue, err
	}
	return parseName(info, runtime.GOOS), nil
}

// getHostInfo sets hostInfo singleton if not set else returns it
func getHostInfo() (*host.InfoStat, error) {
	if hostInfo != nil {
		return hostInfo, nil
	}

	var err error
	hostInfo, err = host.Info()
	return hostInfo, err
}

// parseName parses the os name from the supplied info.
func parseName(info *host.InfoStat, os string) string {
	if info == nil {
		return unknownValue
	}

	switch os {
	case "darwin":
		return fmt.Sprintf("macOS %s", parseDarwinVersion(info.PlatformVersion))
	case "linux":
		return fmt.Sprintf("%s %s", formatLinuxName(info.Platform), info.PlatformVersion)
	default:
		return info.Platform
	}
}

// parseDarwinVersion parses a darwin version.
func parseDarwinVersion(platformVersion string) string {
	parts := strings.Split(platformVersion, ".")
	if len(parts) < 2 {
		return "unknown version"
	}
	return fmt.Sprintf("%s.%s", parts[0], parts[1])
}

// formatLinuxName formats a linux distribution name.
func formatLinuxName(platform string) string {
	switch strings.ToLower(platform) {
	case "centos":
		return "CentOS"
	case "rhel":
		return "RedHat Enterprise Linux"
	case "ubuntu":
		return "Ubuntu"
	case "suse", "sles":
		return "SLES Enterprise Linux"
	case "coreos":
		return "CoreOS"
	case "linuxmint":
		return "Linux Mint"
	default:
		return strings.Title(platform)
	}
}

// MACAddress returns the MAC address for the host.
func MACAddress() string {
	return findMACAddress(net.Interfaces)
}

// findMACAddress does its best to find the MAC address for the local network interface.
func findMACAddress(interfaces func() ([]net.Interface, error)) string {
	iFaces, err := interfaces()
	if err != nil {
		return unknownValue
	}

	for _, iFace := range iFaces {
		if iFace.HardwareAddr.String() != "" {
			addrs, _ := iFace.Addrs()
			for _, addr := range addrs {
				address := addr.String()
				if strings.Contains(address, "/") {
					address = address[:strings.Index(address, "/")]
				}
				ipAddress := net.ParseIP(address)
				if isValidV4Address(ipAddress) {
					return iFace.HardwareAddr.String()
				}
			}
		}
	}

	return unknownValue
}

// isValidV4Address checks that the IP address is not nil, loopback, or unspecified. It also checks that it is IPv4
func isValidV4Address(address net.IP) bool {
	return address != nil && !address.IsLoopback() && !address.IsUnspecified() && address.To4() != nil
}
