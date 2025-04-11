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

package matcher_test

import (
	"math/rand/v2"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/observiq/bindplane-otel-collector/processor/regexmatchprocessor/internal/matcher"
)

func BenchmarkMatch(b *testing.B) {
	// Extract sample keys from the map
	sampleKeys := make([]string, 0, len(testExamples))
	for k := range testExamples {
		sampleKeys = append(sampleKeys, k)
	}

	// Shuffle the keys to randomize benchmark execution order
	rand.Shuffle(len(sampleKeys), func(i, j int) {
		sampleKeys[i], sampleKeys[j] = sampleKeys[j], sampleKeys[i]
	})

	// --- Naive Benchmark ---
	b.Run("naive", func(b *testing.B) {
		b.ResetTimer() // Start timing after setup
		for i := 0; i < b.N; i++ {
			for _, sample := range sampleKeys {
				expectedName := testExamples[sample]
				actualMatchName := "default"
				// Perform naive matching
				for _, r := range testRegexes {
					if r.Regex.MatchString(sample) {
						actualMatchName = r.Name
						break // First match wins
					}
				}
				// Inline validation (comment out for pure speed measurement)
				require.Equal(b, expectedName, actualMatchName, "Input: %s", sample)
			}
		}
		b.StopTimer()
	})

	// Now let's evaluate the optimized approach
	b.Run("optimized", func(b *testing.B) {
		// Setup matcher outside the timer
		matcher, err := matcher.New(testRegexes, "default")
		require.NoError(b, err)

		b.ResetTimer() // Start timing after setup
		for i := 0; i < b.N; i++ {
			for _, sample := range sampleKeys {
				expectedName := testExamples[sample]
				actualMatchName := matcher.Match(sample)

				// Inline validation (comment out for pure speed measurement)
				require.Equal(b, expectedName, actualMatchName, "Input: %s", sample)
			}
		}
		b.StopTimer()
	})
}

// testRegexes defines a set of known regexes in order of precedence
var testRegexes = []matcher.NamedRegex{
	{
		Name:  "esxi_syslog",
		Regex: regexp.MustCompile(`^(?:\d+\s)?<\d+>\d{1,4}-\d{1,2}-\d{1,2}T\d{1,2}:\d{1,2}:\d{1,2}(?:\.\d+)?(?:[+-][\d:]{4,5})?Z?\s+[^\s]+\s+(?:[^\s:]+?\s[^\s:]+?\s[^\s:]+?\s(?:[^\s\[]+|\[[^\[\]]+\])\s+)?(?:[^\s\[]+(?:\[\d+\])?:\s+)?.+$`),
	},
	{
		Name:  "rfc3164_tz_ip",
		Regex: regexp.MustCompile(`^<\d+>\d{4}\s+\w+\s+\d{1,2}\s+\d{1,2}:\d{1,2}:\d{1,2}\s+TZ[+-]\d+\s+[^\s]+\s+\d+\.\d+\.\d+\.\d+\s+[^\s:]+:\s+.*`),
	},
	{
		Name:  "rfc3164_tz_yr",
		Regex: regexp.MustCompile(`^<\d+>\w+\s+\d{1,2}\s+\d{1,2}:\d{1,2}:\d{1,2}\s+\w{3}\s+\d{4}\s+[^\s]+\s+[^\s:]+:\s+.*`),
	},
	{
		Name:  "cisco_asa_formatted_syslog_rfc_3164_with_year",
		Regex: regexp.MustCompile(`^(?:<\d+>)?\w+\s{1,2}\d{1,2}(?:\s+\d{4})?\s+\d{1,2}:\d{1,2}:\d{1,2}:?\s+[^\s:]+:?\s+(?:[^\s\[]+(?:\[\d+\])?:\s+)?.*`),
	},
	{
		Name:  "access_point_logs_1",
		Regex: regexp.MustCompile(`^(?:<\d+>)(?:\d+:?\s+?\d+:)?\s+\w+\s{1,2}\d{1,2}(?:\s+\d{4})?\s+\d{1,2}:\d{1,2}:\d{1,2}(?:\.\d+)?:?\s+.*`),
	},
	{
		Name:  "access_point_logs_2",
		Regex: regexp.MustCompile(`^(?:<\d+>)[^\s:]+:?\s+?(?:[^\[]+:)?\s+\w+\s{1,2}\d{1,2}(?:\s+\d{4})?\s+\d{1,2}:\d{1,2}:\d{1,2}(?:\.\d+)?:?\s+.*`),
	},
	{
		Name:  "rfc3164",
		Regex: regexp.MustCompile(`^<\d+>\w+\s+\d{1,2}\s+\d{1,2}:\d{1,2}:\d{1,2}\s+[^\s]+\s+[^\s:]*?:?\s*.*`),
	},
	{
		Name:  "rfc5424",
		Regex: regexp.MustCompile(`^(\d+\s)?<\d+>[^\s]+\s+\d{1,4}-\d{1,2}-\d{1,2}T\d{1,2}:\d{1,2}:\d{1,2}(\.\d+)?([+-][\d:]{4,5})?Z?\s+[^\s]+\s+[^\s:]+?:?\s*.*`),
	},
	{
		Name:  "rfc5424_with_optional_octet_frame_unix_timestamp",
		Regex: regexp.MustCompile(`^(?:\d+\s)?<\d+>[^\s]+\s+\d+\.\d+\s+[^\s]+\s+(?:[^\s:]+?\s[^\s:]+?\s[^\s:]+?\s(?:[^\s\[]+|\[[^\[\]]+\])\s+)?(?:[^\s\[]+(?:\[\d+\])?:\s+)?.+$`),
	},
}

// testExamples maps input strings to the expected name of the *first* matching regex in testRegexes
var testExamples = map[string]string{

	// esxi_syslog
	`<182>2024-07-25T18:54:02.265Z esxi-a vmkernel: cpu21:2097238)ScsiDeviceIO: 4124: Cmd(0x45b9309b77c8) 0x42, CmdSN 0xd82373 from world 20761373 to dev "t10.VMDRAID_Intel_Raid_1_VolumeVolume000000001" failed H:0xc D:0x0 P:0x0`: "esxi_syslog",
	`<165>2024-03-15T09:12:45.123Z esxi-b vmkernel: cpu5:12345)VMKernel: 1234: Network interface vmnic0 link up`:                                                                                                                     "esxi_syslog",
	`<190>2024-01-01T00:00:00Z esxi-c vmkernel: cpu0:1)Storage: 5678: Storage path vmhba0:C0:T0:L0 is now active`:                                                                                                                    "esxi_syslog",
	`<178>2024-12-31T23:59:59.999Z esxi-d vmkernel: cpu10:9876)VMKernel: 4321: Virtual machine 1234 powered on`:                                                                                                                      "esxi_syslog",
	`<174>2024-06-30T15:30:00.500Z esxi-e vmkernel: cpu15:5432)ScsiDeviceIO: 8765: Cmd(0x123456789abc) 0x43, CmdSN 0x987654 from world 123456 to dev "t10.VMDRAID_Intel_Raid_2_VolumeVolume000000002" completed successfully`:        "esxi_syslog",

	// rfc3164_tz_ip
	`<0>1990 Oct 22 10:52:01 TZ-6 scapegoat.dmz.example.org 10.1.2.3 sched[0]: That's All Folks!`:          "rfc3164_tz_ip",
	`<1>2024 Mar 15 14:30:45 TZ+5 server1.example.com 192.168.1.1 sshd[1234]: Connection closed by user`:   "rfc3164_tz_ip",
	`<2>2023 Dec 31 23:59:59 TZ-8 server2.example.com 172.16.0.1 kernel[5678]: System reboot requested`:    "rfc3164_tz_ip",
	`<3>2022 Jun 30 12:00:00 TZ+0 server3.example.com 10.0.0.1 cron[9012]: Job completed successfully`:     "rfc3164_tz_ip",
	`<4>2021 Jan 01 00:00:01 TZ+3 server4.example.com 192.168.0.1 nginx[3456]: New connection established`: "rfc3164_tz_ip",

	// rfc3164_tz_yr
	`<165>Aug 24 05:34:00 CST 1987 mymachine myproc[10]: %% It's time to make the do-nuts.  %%  Ingredients: Mix=OK, Jelly=OK # Devices: Mixer=OK, Jelly_Injector=OK, Frier=OK # Transport: Conveyer1=OK, Conveyer2=OK # %%`: "rfc3164_tz_yr",
	`<166>Jan 01 00:00:00 UTC 2024 server1 process[123]: System initialization complete`: "rfc3164_tz_yr",
	`<167>Mar 15 12:30:45 EST 2023 server2 daemon[456]: Service started successfully`:    "rfc3164_tz_yr",
	`<168>Dec 31 23:59:59 PST 2022 server3 app[789]: Application shutdown initiated`:     "rfc3164_tz_yr",
	`<169>Jun 30 15:45:30 GMT 2021 server4 service[012]: Maintenance window started`:     "rfc3164_tz_yr",

	// cisco_asa_formatted_syslog_rfc_3164_with_year
	`<167>May 20 2024 15:08:29: %ASA-7-777777: Built local-host inside:10.10.0.20`:                     "cisco_asa_formatted_syslog_rfc_3164_with_year",
	`<168>Jan 15 2023 09:30:45: %ASA-6-123456: TCP connection established for outside:192.168.1.1/443`: "cisco_asa_formatted_syslog_rfc_3164_with_year",
	`<169>Mar 31 2022 14:22:33: %ASA-5-234567: VPN session terminated for user:admin`:                  "cisco_asa_formatted_syslog_rfc_3164_with_year",
	`<170>Dec 25 2021 00:00:01: %ASA-4-345678: Denied ICMP type=8 from outside:10.0.0.1`:               "cisco_asa_formatted_syslog_rfc_3164_with_year",
	`<171>Jul 04 2020 12:00:00: %ASA-3-456789: Failed to establish VPN connection for user:guest`:      "cisco_asa_formatted_syslog_rfc_3164_with_year",

	// access_point_logs_1
	`<190>363041: 363044: Nov 7 14:38:47.482 SST: %SEC-xx-xxxxxxx: list SNMP denied xx.xx.xx.xx xxx packets`: "access_point_logs_1",
	`<191>123456: 123459: Jan 1 00:00:01.001 AP1: %WLAN-1-234567: Client 00:11:22:33:44:55 associated`:       "access_point_logs_1",
	`<192>234567: 234570: Mar 15 12:30:45.500 AP2: %SEC-2-345678: Authentication failed for user:admin`:      "access_point_logs_1",
	`<193>345678: 345681: Jun 30 15:45:30.750 AP3: %WLAN-3-456789: Client 66:77:88:99:AA:BB disconnected`:    "access_point_logs_1",
	`<194>456789: 456792: Dec 31 23:59:59.999 AP4: %SEC-4-567890: RADIUS server 10.0.0.1 unreachable`:        "access_point_logs_1",

	// access_point_logs_2
	`<132>7777-ABX1340-01: *spamApTask3: May 20 15:09:15.005: %LOG-4-Q_IND: [SA]capwap_ac_sm.c:2053 Ignoring discovery request received on a wrong VLAN (6) on interface (13) from AP 8D-C6-86-9D-08-3C`: "access_point_logs_2",
	`<133>8888-CDX2450-02: *wlanTask1: Jan 15 09:30:45.123: %LOG-3-Q_IND: [SA]wlan.c:1234 Client 00:11:22:33:44:55 associated to SSID:Guest`:                                                             "access_point_logs_2",
	`<134>9999-EFX3560-03: *authTask2: Mar 31 14:22:33.456: %LOG-2-Q_IND: [SA]auth.c:5678 Authentication failed for user:admin`:                                                                          "access_point_logs_2",
	`<135>0000-GHX4670-04: *radioTask4: Dec 25 00:00:01.789: %LOG-5-Q_IND: [SA]radio.c:9012 Channel changed to 6 due to interference`:                                                                    "access_point_logs_2",
	`<136>1111-IJX5780-05: *sysTask5: Jul 04 12:00:00.012: %LOG-1-Q_IND: [SA]sys.c:3456 System reboot initiated by admin`:                                                                                "access_point_logs_2",

	// rfc3164
	`<34>Oct 11 22:14:15 my:machine su:\'su root\' failed for lonvick on /dev/pts/8`: "rfc3164",
	`<35>Jan 01 00:00:01 serv:er1 sshd:Accepted password for root from 192.168.1.1`:  "rfc3164",
	`<36>Mar 15 12:30:45 serv:er2 kernel:CPU0: Core temperature above threshold`:     "rfc3164",
	`<37>Jun 30 15:45:30 serv:er3 cron:Job completed successfully`:                   "rfc3164",
	`<38>Dec 31 23:59:59 serv:er4 nginx:New connection from 10.0.0.1`:                "rfc3164",

	// rfc5424
	`<85>1 2023-04-05T16:20:56Z loki.example.com su - ID47 - BOM\'su root\' failed for lonvick on /dev/pts/8757 <14>1 2022-10-19T15:15:32-07:00 b-fw-1-b.ruckus.edu - - - - 1,2022/10/19 15:15:24,012501002053,TRAFFIC,end,2561,2022/10/19 15:15:24,10.13.63.5,138.23.62.34,0.0.0.0,0.0.0.0,Allow Internal Zones to Net Services,,,dns-base,vsys1,iecn,inside,ae4.1032,ae4.1030,RUCKUS-Log-Forwarding-Template,2022/10/19 15:15:24,4127477,1,50251,53,0,0,0x19,udp,allow,212,82,130,2,2022/10/19 15:14:54,0,any,,7088216349921087004,0x8000000000000000,10.0.0.0-10.255.255.255,United States,,1,1,aged-out,389,12,0,0,vsys1,b-fw-1-b,from-policy,,,0,,0,,N/A,0,0,0,0,79a1a95e-34c3-4819-b826-27391a206529,0,0,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,2022-10-19T15:15:32.492-07:00,,,infrastructure,networking,network-protocol,3,\"used-by-malware,has-known-vulnerability,pervasive-use\",dns,dns-base,no,no,0`: "rfc5424",
	`<86>1 2024-01-01T00:00:01.123Z server1.example.com sshd - ID48 - Accepted password for root <15>1 2023-12-31T23:59:59-08:00 fw-1.example.com - - - - 1,2023/12/31 23:59:59,012501002054,TRAFFIC,end,2562,2023/12/31 23:59:59,192.168.1.1,10.0.0.1,0.0.0.0,0.0.0.0,Allow Internal Zones to Net Services,,,ssh-base,vsys1,iecn,inside,ae4.1033,ae4.1031,SSH-Log-Forwarding-Template,2023/12/31 23:59:59,4127478,1,50252,22,0,0,0x20,tcp,allow,213,83,131,3,2023/12/31 23:59:29,0,any,,7088216349921087005,0x8000000000000001,192.168.0.0-192.168.255.255,United States,,1,1,aged-out,390,13,0,0,vsys1,fw-1,from-policy,,,0,,0,,N/A,0,0,0,0,79a1a95e-34c3-4819-b826-27391a206530,0,0,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,2023-12-31T23:59:59.493-08:00,,,infrastructure,networking,network-protocol,3,\"used-by-malware,has-known-vulnerability,pervasive-use\",ssh,ssh-base,no,no,0`:                         "rfc5424",
	`<87>1 2023-03-15T12:30:45.456Z server2.example.com kernel - ID49 - CPU temperature warning <16>1 2023-03-15T12:30:45-05:00 fw-2.example.com - - - - 1,2023/03/15 12:30:45,012501002055,TRAFFIC,end,2563,2023/03/15 12:30:45,172.16.0.1,10.0.0.2,0.0.0.0,0.0.0.0,Allow Internal Zones to Net Services,,,http-base,vsys1,iecn,inside,ae4.1034,ae4.1032,HTTP-Log-Forwarding-Template,2023/03/15 12:30:45,4127479,1,50253,80,0,0,0x21,tcp,allow,214,84,132,4,2023/03/15 12:30:15,0,any,,7088216349921087006,0x8000000000000002,172.16.0.0-172.31.255.255,United States,,1,1,aged-out,391,14,0,0,vsys1,fw-2,from-policy,,,0,,0,,N/A,0,0,0,0,79a1a95e-34c3-4819-b826-27391a206531,0,0,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,2023-03-15T12:30:45.494-05:00,,,infrastructure,networking,network-protocol,3,\"used-by-malware,has-known-vulnerability,pervasive-use\",http,http-base,no,no,0`:                         "rfc5424",
	`<88>1 2022-06-30T15:45:30.789Z server3.example.com cron - ID50 - Job completed <17>1 2022-06-30T15:45:30-04:00 fw-3.example.com - - - - 1,2022/06/30 15:45:30,012501002056,TRAFFIC,end,2564,2022/06/30 15:45:30,10.0.0.3,192.168.2.1,0.0.0.0,0.0.0.0,Allow Internal Zones to Net Services,,,smtp-base,vsys1,iecn,inside,ae4.1035,ae4.1033,SMTP-Log-Forwarding-Template,2022/06/30 15:45:30,4127480,1,50254,25,0,0,0x22,tcp,allow,215,85,133,5,2022/06/30 15:45:00,0,any,,7088216349921087007,0x8000000000000003,10.0.0.0-10.255.255.255,United States,,1,1,aged-out,392,15,0,0,vsys1,fw-3,from-policy,,,0,,0,,N/A,0,0,0,0,79a1a95e-34c3-4819-b826-27391a206532,0,0,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,2022-06-30T15:45:30.495-04:00,,,infrastructure,networking,network-protocol,3,\"used-by-malware,has-known-vulnerability,pervasive-use\",smtp,smtp-base,no,no,0`:                                      "rfc5424",
	`<89>1 2021-12-31T23:59:59.999Z server4.example.com nginx - ID51 - New connection established <18>1 2021-12-31T23:59:59-06:00 fw-4.example.com - - - - 1,2021/12/31 23:59:59,012501002057,TRAFFIC,end,2565,2021/12/31 23:59:59,192.168.3.1,10.0.0.4,0.0.0.0,0.0.0.0,Allow Internal Zones to Net Services,,,https-base,vsys1,iecn,inside,ae4.1036,ae4.1034,HTTPS-Log-Forwarding-Template,2021/12/31 23:59:59,4127481,1,50255,443,0,0,0x23,tcp,allow,216,86,134,6,2021/12/31 23:59:29,0,any,,7088216349921087008,0x8000000000000004,192.168.0.0-192.168.255.255,United States,,1,1,aged-out,393,16,0,0,vsys1,fw-4,from-policy,,,0,,0,,N/A,0,0,0,0,79a1a95e-34c3-4819-b826-27391a206533,0,0,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,2021-12-31T23:59:59.496-06:00,,,infrastructure,networking,network-protocol,3,\"used-by-malware,has-known-vulnerability,pervasive-use\",https,https-base,no,no,0`:               "rfc5424",

	// rfc5424_with_optional_octet_frame_unix_timestamp
	`<134>1 1729780712.350113344 SOMEPLACE_SOMEWHERE vpn_firewall src=xx.xx.xx.xx dst=xx.xx.xx.xx mac=00:00:00:00:0000:00 protocol=xxx sport=xxx dport=xxx pattern: allow (dst xx.xx.xx.xx/xx || dst xx.xx.xx.xx/xx)`: "rfc5424_with_optional_octet_frame_unix_timestamp",
	`<135>1 1729780712.341701835 SOMEPLACE_SOMEWHERE vpn_firewall src=10.0.0.1 dst=192.168.1.1 mac=00:11:22:33:44:55 protocol=tcp sport=443 dport=8080 pattern: allow all`:                                            "rfc5424_with_optional_octet_frame_unix_timestamp",
	`<136>1 1729780711.444469190 SOMEPLACE_SOMEWHERE urls src=172.16.0.1:80 dst=10.0.0.2:443 mac=00:22:33:44:55:66 agent='Mozilla/5.0' request: GET http://example.com`:                                               "rfc5424_with_optional_octet_frame_unix_timestamp",
	`<137>1 1729780721.993317388 ANOTHERPLACE_ELSEWHERE ip_flow_end src=192.168.0.1 dst=10.0.0.3 protocol=udp sport=53 dport=53 translated_src_ip=172.16.0.2 translated_port=1024`:                                    "rfc5424_with_optional_octet_frame_unix_timestamp",
	`<138>1 1729780722.123456789 ANOTHERPLACE_ELSEWHERE ip_flow_end src=10.0.0.4 dst=192.168.0.2 protocol=tcp sport=22 dport=22 translated_src_ip=172.16.0.3 translated_port=2048`:                                    "rfc5424_with_optional_octet_frame_unix_timestamp",

	// default
	`This input does not match any specific pattern`: "default",
}
