// Copyright  observIQ, Inc.
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

package osinfo

import (
	"errors"
	"net"
	"runtime"
	"testing"

	"github.com/shirou/gopsutil/v3/host"
	"github.com/stretchr/testify/require"
)

func TestName(t *testing.T) {
	switch runtime.GOOS {
	case "darwin":
		name, err := Name()
		require.NoError(t, err)
		require.Contains(t, name, "macOS")
	default:
		t.Skip()
	}
}

func TestParseName(t *testing.T) {
	testCases := []struct {
		name     string
		info     *host.InfoStat
		os       string
		expected string
	}{
		{
			name:     "Nil info",
			info:     nil,
			os:       "darwin",
			expected: "unknown",
		},
		{
			name: "Darwin bad version",
			info: &host.InfoStat{
				PlatformVersion: "no version",
			},
			os:       "darwin",
			expected: "macOS unknown version",
		},
		{
			name: "Darwin valid version",
			info: &host.InfoStat{
				PlatformVersion: "11.1.0",
			},
			os:       "darwin",
			expected: "macOS 11.1",
		},
		{
			name: "windows",
			info: &host.InfoStat{
				Platform: "windows",
			},
			os:       "windows",
			expected: "windows",
		},
		{
			name: "centos",
			info: &host.InfoStat{
				Platform:        "centos",
				PlatformVersion: "0.0",
			},
			os:       "linux",
			expected: "CentOS 0.0",
		},
		{
			name: "rhel",
			info: &host.InfoStat{
				Platform:        "rhel",
				PlatformVersion: "0.0",
			},
			os:       "linux",
			expected: "RedHat Enterprise Linux 0.0",
		},
		{
			name: "ubuntu",
			info: &host.InfoStat{
				Platform:        "ubuntu",
				PlatformVersion: "0.0",
			},
			os:       "linux",
			expected: "Ubuntu 0.0",
		},
		{
			name: "suse",
			info: &host.InfoStat{
				Platform:        "suse",
				PlatformVersion: "0.0",
			},
			os:       "linux",
			expected: "SLES Enterprise Linux 0.0",
		},
		{
			name: "sles",
			info: &host.InfoStat{
				Platform:        "sles",
				PlatformVersion: "0.0",
			},
			os:       "linux",
			expected: "SLES Enterprise Linux 0.0",
		},
		{
			name: "coreos",
			info: &host.InfoStat{
				Platform:        "coreos",
				PlatformVersion: "0.0",
			},
			os:       "linux",
			expected: "CoreOS 0.0",
		},
		{
			name: "linuxmint",
			info: &host.InfoStat{
				Platform:        "linuxmint",
				PlatformVersion: "0.0",
			},
			os:       "linux",
			expected: "Linux Mint 0.0",
		},
		{
			name: "unknown linux",
			info: &host.InfoStat{
				Platform:        "unknown",
				PlatformVersion: "0.0",
			},
			os:       "linux",
			expected: "Unknown 0.0",
		},
	}

	for _, tc := range testCases {
		name := parseName(tc.info, tc.os)
		require.Equal(t, tc.expected, name)
	}
}

func TestIsValidV4Address(t *testing.T) {
	testCases := []struct {
		name     string
		address  net.IP
		expected bool
	}{
		{
			name:     "valid",
			address:  net.IPv4(1, 1, 1, 1),
			expected: true,
		},
		{
			name:     "invalid",
			address:  net.IPv6loopback,
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			valid := isValidV4Address(tc.address)
			require.Equal(t, tc.expected, valid)
		})
	}
}

func TestMACAddress(t *testing.T) {
	mac := MACAddress()
	require.NotEqual(t, "unknown", mac)
}

func TestFindMACAddress(t *testing.T) {
	testCases := []struct {
		name       string
		interfaces func() ([]net.Interface, error)
		expected   string
	}{
		{
			name: "Failed to get interfaces",
			interfaces: func() ([]net.Interface, error) {
				return nil, errors.New("failure")
			},
			expected: "unknown",
		},
		{
			name: "No interfaces",
			interfaces: func() ([]net.Interface, error) {
				return []net.Interface{}, nil
			},
			expected: "unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mac := findMACAddress(tc.interfaces)
			require.Equal(t, tc.expected, mac)
		})
	}
}
